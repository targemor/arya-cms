# CMS admin

Angular + [NG-ZORRO](https://ng.ant.design) admin panel for the Go API in the repository root.

## Run

From the repository root, `make dev` starts Postgres, the API and this admin together (see the root README). To run
them separately:

```sh
# Go API on :8080 (from the repository root)
make compose-up
make run

# Admin on http://localhost:4200, proxying /v1 to the API
cd web
npm install
npm start
```

`ng serve` forwards `/v1` and `/uploads` to `http://localhost:8080` ([proxy.conf.js](proxy.conf.js)), so the browser
talks to one origin and the API needs no CORS. Point it at another backend with `API_URL`:

```sh
API_URL=http://localhost:18080 npm start
```

In production, serve the built files (`npm run build` → `dist/admin/browser`) behind the same reverse proxy as the API,
with unknown paths falling back to `index.html`.

## Test

```sh
npm test -- --watch=false
```

## Structure

| Path | What it holds |
| --- | --- |
| `src/app/core/auth/` | `AuthService` (login, JWT in `localStorage`), `authInterceptor` (adds `Authorization`, signs out on 401), `authGuard` / `guestGuard` |
| `src/app/layout/` | Admin shell: `nz-layout`, collapsible `nz-sider` menu, user menu |
| `src/app/pages/login/` | Login page, reactive form with `nz-form` |
| `src/app/pages/content-type-builder/` | Defines content types: reactive form with a `FormArray` of fields (text, number, boolean, media), client-side checks matching the API, live preview of the stored JSON Schema |
| `src/app/pages/content-manager/` | Entries of one content type (`/content-manager/:tableName`): `nz-table` with columns from the schema, server-side pagination and sort sent as Strapi query params, create/edit modal with a form generated from the schema |
| `src/app/pages/media-library/` | Media Library: drag-and-drop `nz-upload` to `POST /v1/upload` plus the gallery |
| `src/app/shared/media-gallery/` | File grid (`nz-row` / `nz-col` / `nz-card`) with pagination; delete mode for the library, select mode for the picker |
| `src/app/shared/media-picker/` | Form control for media fields: value is the file id; opens the gallery in a `nz-modal` to choose a file |
| `src/app/core/content/` | `ContentTypesService` (shared signal state behind the builder, the sider submenu and the content manager) and `ContentService` for entries |
| `src/app/core/media/` | `MediaService` for `/v1/upload/files` and the `UploadFile` type |

The JWT expiry (`exp`) is only read to decide whether to show the login page; the API verifies the token on every
request.
