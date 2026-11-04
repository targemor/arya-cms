# go-clean-template

Go clean-architecture service: a small Strapi-style headless CMS. Domains `user`, `contenttype`,
`content` and `upload` (Media Library), exposed over REST/Fiber from one shared use-case layer, plus an
Angular admin frontend in `web/`.
Module path: `github.com/lucimux/arya`. The Go version is declared in `go.mod`.

## Commands — drive everything through the Makefile

The `Makefile` loads `.env` (falling back to `.env.example`) and exports it into every target, and
several targets carry flags that matter. Running the underlying tool directly means running it with
different settings than CI. Use the target, not the tool.

| Task | Use this | Not this |
| --- | --- | --- |
| Install tool binaries (`swag`, `mockgen`, `migrate`, linters) | `make bin-deps` | `go install ...` |
| Unit tests | `make test` | `go test ./...` — the target adds `-race -covermode atomic` and scopes to `./internal/... ./pkg/...` |
| Lint | `make linter-golangci` | `golangci-lint run` |
| Format | `make format` | `gofmt` — the target runs `go fix`, `gofumpt`, and `gci` with the repo's import grouping |
| Regenerate mocks | `make mock` | `mockgen ...` |
| Regenerate Swagger | `make swag-v1` | `swag init` — the target passes `--parseDependency -g internal/controller/restapi/router.go` |
| Tidy / verify modules | `make deps` | `go mod tidy` |
| Vulnerability scan | `make deps-audit` | `govulncheck ./...` |
| Start Postgres | `make compose-up` | `docker compose up` |
| Start the whole stack including the app | `make compose-up-all` | `docker compose up` |
| Tear down | `make compose-down` | `docker compose down` |
| Run the app locally | `make run` | `go run ./cmd/app` — the target regenerates docs and builds with `-tags migrate` |
| Run API + admin web for development | `make dev` | starting them by hand — the target starts the db, sets `API_URL` for the web proxy and stops both on Ctrl+C |
| Create a migration | `make migrate-create <name>` | `migrate create ...` |
| Apply migrations | `make migrate-up` | `migrate -path ... up` |
| Full check before pushing | `make pre-commit` | running the steps by hand |

`make help` lists every target.

Traps in these targets:

- **`make migrate-create <name>` prints an error after it succeeds.** The target reads the name via
  `$(word 2,$(MAKECMDGOALS))`, so `make` then tries to build `<name>` as a target and reports
  `No rule to make target`. The migration files are already created; ignore that line.
- **`make run` and `make pre-commit` depend on `swag-v1`**, so they need `swag` on `PATH`. Run
  `make bin-deps` first.
- **`make deps` is slow** (`go mod verify` checks every tool dependency); expect minutes.

Never claim a change is done without `make format`, `make linter-golangci` and `make test` passing.

## Dependency rule

```
cmd/app → internal/app → internal/controller/restapi  ─┐
                       → internal/repo/*              ─┤→ internal/usecase (interfaces)
                                                       └→ internal/entity
```

- `internal/entity` — domain types and sentinel errors. Imports nothing from this module.
- `internal/usecase/contracts.go` — the interfaces controllers call. `internal/usecase/<domain>/`
  implements them.
- `internal/repo/contracts.go` — the interfaces use cases call. Implemented by
  `internal/repo/persistent/<domain>/` (Postgres), `internal/repo/filestore/<domain>/` (files on
  disk, e.g. content-type JSON Schemas) and `internal/repo/storage/<provider>/` (Media Library bytes).
- `internal/controller/restapi/v1/` — handlers, with `request/` and `response/` DTOs.
- `pkg/` — infrastructure with no domain knowledge (http server, logger, jwt, postgres, tracing,
  fswatcher). Must not import `internal/`.

Inner layers never import outer ones. Wiring happens exactly once, in `internal/app/app.go`
(`initUseCases`, `initServers`).

## Where code goes

Decide by asking what the code knows about:

| The code knows about… | It belongs in | Shape |
| --- | --- | --- |
| nothing but the domain | `internal/entity/<name>.go` | struct + methods |
| a domain rule that spans repositories | `internal/usecase/<domain>/<domain>.go` | method on `UseCase` |
| SQL, a table, a driver error code | `internal/repo/persistent/<domain>/<domain>.go` | method on `Repo` |
| files on disk holding domain data | `internal/repo/filestore/<domain>/` | method on `Repo` |
| where uploaded bytes live (disk, S3, …) | `internal/repo/storage/<provider>/` | a `repo.StorageProvider` |
| an outbound HTTP API | `internal/repo/webapi/<name>.go` (create it) | method on the webapi struct |
| an HTTP status, a request/response shape | `internal/controller/restapi/v1/<domain>.go` | handler |
| a server, pool, client or middleware with no domain knowledge | `pkg/<name>/` | reusable package |

Layout rules:

- One file per domain per layer, named after the domain (`user.go`), plus a `tracing.go` in every
  `usecase/<domain>/` and `repo/*/<domain>/` package.
- Package name matches the directory name.
- Constructors are always `New`, return the *interface* from `contracts.go`, and take their
  dependencies as arguments. No package-level state — `gochecknoglobals` and `gochecknoinits` are on.
- Unexported package constants are prefixed with an underscore (`_defaultEntityCap`,
  `_tracerName`). `mnd` rejects bare numeric literals, so add a constant rather than inlining one.

## The handler contract

Every handler does the same six things in the same order. Copy the shape from a neighbouring handler.

1. **Get the caller.** `ctx.Locals("userID").(string)`. A failure here is an auth error, not a 500.
2. **Decode into a DTO** from `restapi/v1/request/`. Never decode into an `entity` type, and never
   pass a `request.*` type into a use case.
3. **Validate the DTO** with the injected `*validator.Validate` (`r.v.Struct(req)`). Structural
   validation (required, min, max, oneof) lives here. Domain validation — anything that needs to
   know the rules, like a status transition — lives on the entity or the use case and must not be
   duplicated in the controller.
4. **Call the use case** with `ctx.UserContext()` (**not** `ctx.Context()`, which drops the trace).
   `context.Context` is the first parameter of every method that crosses a layer.
5. **Map the error.** `errors.Is` against the `entity.Err*` sentinels → an HTTP status, via a
   per-domain `[]errorStatus` table passed to `r.mappedError` (see `content.go`, `upload.go`). Log the
   wrapped error, return a generic message to the caller.
6. **Encode the response.** Return an `entity` type directly only when its JSON shape is already
   the API shape — `entity.User` is safe this way because `PasswordHash` is tagged `json:"-"`.
   Anything with a wrapper, a projection or a different field set gets a type in
   `restapi/v1/response/` (`response.Token`, `response.ContentEntries`).

## Conventions that will bite you

**Traced decorators.** Every repo and use-case constructor returns the *interface*, wrapped:
`func New(...) usecase.User { return newTraced(&UseCase{...}) }`. The wrapper lives in
`tracing.go` next to each implementation. Adding a method to a contract means implementing it in
three places: the interface, the struct, and the `traced*` decorator. Skipping the decorator is a
compile error that reads as an unrelated interface mismatch.

**Mocks are generated, not written.** `internal/usecase/mocks_*_test.go` come from `go:generate`
directives in the two `contracts.go` files. After any contract change run `make mock` before
running tests.

**Error style.** Domain failures are sentinel errors in `internal/entity/errors.go`
(`ErrUserNotFound`, `ErrInvalidEntry`, `ErrUploadTooLarge`, …). Every other error is wrapped
with the layer path: `fmt.Errorf("UserUseCase - Register - uc.repo.Store: %w", err)`. The `err113`
linter rejects `errors.New`/`fmt.Errorf` for new dynamic errors — add a sentinel to `entity`
instead. Controllers are the only place that translates a sentinel into a status.

**Config is environment-only.** `config/config.go` parses env vars with `caarlos0/env`. A new
setting needs the struct field, an entry in `.env.example`, and — if the container needs it — the
`environment` block in `docker-compose.yml`. Fields tagged `required` make the app fail at
startup.

**User-defined identifiers reach SQL.** Content-type table and attribute names come from users.
They are validated on `entity.ContentType` (lowercase snake_case) *and* quoted with
`pgx.Identifier{...}.Sanitize()` wherever they are written into SQL; values always go in as
parameters. Keep both layers when touching `repo/persistent/content` or `contenttype`.

**The linter is strict and non-obvious.** `.golangci.yml` enables ~45 linters. The ones that
reject otherwise-fine code: `wsl_v5` and `nlreturn` (blank line required before `return` and
around blocks), `funlen` (65 lines / 40 statements), `gocyclo` (10), `gocognit` (15), `mnd`,
`gochecknoglobals`, `exhaustive` (switches over enum types need every case or a `default`), `dupl`
(threshold 100 — copy-pasting a handler trips it), `gocritic` `hugeParam` (pass structs over 80
bytes, like `entity.UploadFile` or `entity.Filter`, by pointer).

## Generated files — do not hand-edit

- `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` ← `make swag-v1`, driven by the
  annotation comments above REST handlers and on `NewRouter` in `internal/controller/restapi/router.go`.
- `internal/usecase/mocks_*_test.go` ← `make mock`.

## Keep the README in step

`README.md` documents the API and the layout. Update it whenever a change touches what it documents:

| Change | What to update |
| --- | --- |
| New or changed endpoint | the domain table under `Domains` |
| New domain | the `Overview` bullet list, `Content` index, and a new `Domains` subsection |
| New or renamed env var | the config table under `Observability` if tracing-related; otherwise the `config` section |
| New server, port or service URL | the `Quick start` service list |
| New `make` target used in the getting-started flow | the `Quick start` code blocks |
| Renamed or moved package under `internal/` or `pkg/` | the matching `Project structure` subsection |

Frontend changes go in `web/README.md`.

## Adding a use case to an existing domain

1. Add the method to `internal/usecase/contracts.go`.
2. Implement it in `internal/usecase/<domain>/<domain>.go` and mirror it in the same package's
   `tracing.go`.
3. If it needs new persistence, repeat the same three steps in `internal/repo/contracts.go`, the
   repo package, and that package's `tracing.go`.
4. `make mock`, then extend `internal/usecase/<domain>_test.go` (table-driven, gomock, `t.Parallel()` —
   `paralleltest` enforces it).
5. Expose it: handler in `internal/controller/restapi/v1/<domain>.go` + route in that package's
   `router.go` + swagger annotations, then `make swag-v1`.

## Adding a whole domain

Work outward from the entity. Every step has a counterpart in the existing domains — read one
before writing.

1. **Entity** — `internal/entity/<domain>.go` with the type and its invariants; new sentinel errors
   go in `internal/entity/errors.go`.
2. **Migration** — `make migrate-create create_<table>`, then fill both the `.up.sql` and the
   `.down.sql`. A missing down-migration is a review failure.
3. **Repo contract** — add the interface to `internal/repo/contracts.go`.
4. **Repo implementation** — `internal/repo/persistent/<domain>/<domain>.go` (`package <domain>`,
   Squirrel queries, driver errors mapped to sentinels) plus `tracing.go` next to it, and a `New`
   returning the interface wrapped in `newTraced`.
5. **Use-case contract** — add the interface to `internal/usecase/contracts.go`.
6. **Use-case implementation** — `internal/usecase/<domain>/<domain>.go` plus its `tracing.go`,
   same `New` pattern.
7. **`make mock`**, then `internal/usecase/<domain>_test.go` against the generated mocks.
8. **Wiring** — add a field to `useCases` in `internal/app/app.go`, construct repo and use case in
   `initUseCases`, and thread the use case through `initServers` into the router.
9. **REST** — DTOs in `restapi/v1/request/` and `response/`, handlers in `restapi/v1/<domain>.go`,
   registration in `restapi/v1/router.go`, and the use case added to the `V1` struct in
   `restapi/v1/controller.go`.
10. **Docs** — swagger annotations on the handlers, `make swag-v1`, and the README updates listed above.

## API versioning

Routes are grouped under a version package (`restapi/v1/`). A v2 means a new sibling package with
the same layout, registered alongside v1 in `restapi/router.go` with an `app.Group("/v2")`. v1 stays
untouched. `README.md` shows the worked example.

## Database

`migrations/` holds golang-migrate pairs (`<timestamp>_<name>.up.sql` / `.down.sql`). Create with
`make migrate-create <name>`, apply with `make migrate-up`. The app applies them itself at startup
only when built with the `migrate` build tag (`internal/app/migrate.go`) — that is what `make run`,
`make dev` and the Dockerfile do. Queries are built with Squirrel via the embedded `*postgres.Postgres`;
there is no ORM. Content tables (`content_<tableName>`) are not migrations: their DDL comes from the
content-type definitions at startup and whenever a file in `SCHEMAS_DIR` changes.

## Tests

Unit tests only: `internal/...` and `pkg/...`, run by `make test` with `-race` (needs a C compiler;
on a machine without one, run the same command inside a `golang` Docker image). Use-case tests use
the generated gomock mocks, are table-driven, and call `t.Parallel()` (`paralleltest` enforces it).
Nothing exercises Postgres automatically; check repository changes against a real database.

## Frontend (`web/`)

Angular 22 (standalone, zoneless, signals) + NG-ZORRO 22, tested with Vitest.

- Commands: `npm --prefix web test -- --watch=false`, `npm --prefix web run build`; `make dev` runs it
  against the API. `ng serve` proxies `/v1` and `/uploads` to `API_URL` (`web/proxy.conf.js`).
- All API calls go through `HttpClient`, so `authInterceptor` adds the JWT and signs out on 401 —
  including `nz-upload`, which uses `HttpClient` internally. Don't set auth headers by hand.
- NG-ZORRO 22 renamed things older examples use: `nz-input-group` is gone (`nz-input-wrapper`,
  `nz-input-password`), `NzDropDownModule` → `NzDropdownModule`, `NzToolTipModule` →
  `NzTooltipModule`. Check `node_modules/ng-zorro-antd/types/*.d.ts` when unsure.
- To show a form's errors on submit use `showFormErrors()` (`src/app/shared/forms.ts`):
  `markAllAsDirty()` alone does not repaint `nz-form-control`.
- Binding `nzSortFn` on a `th` turns sorting on whatever the value; leave it off columns that can't sort.
