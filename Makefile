ifneq ($(wildcard .env),)
include .env
export
else
$(warning WARNING: .env file not found! Using .env.example)
include .env.example
export
endif

# On Windows, Git Bash (MSYS) runs the recipes and rewrites env values that look like POSIX paths
# when it starts a native program: UPLOADS_PUBLIC_PATH=/uploads would reach the app as
# "C:/Program Files/Git/uploads". Exclude it from that conversion; elsewhere this does nothing.
export MSYS2_ENV_CONV_EXCL = UPLOADS_PUBLIC_PATH

BASE_STACK = docker compose -f docker-compose.yml

# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

compose-up: ### Run docker compose (without backend and reverse proxy)
	$(BASE_STACK) up --build -d db && docker compose logs -f
.PHONY: compose-up

compose-up-all: ### Run docker compose (with backend and reverse proxy)
	$(BASE_STACK) up --build -d
.PHONY: compose-up-all

compose-down: ### Down docker compose
	$(BASE_STACK) down --remove-orphans
.PHONY: compose-down

swag-v1: ### swag init
	swag init --parseDependency -g internal/controller/restapi/router.go
.PHONY: swag-v1

deps: ### deps tidy + verify
	go mod tidy && go mod verify
.PHONY: deps

deps-audit: ### check dependencies vulnerabilities
	govulncheck ./...
.PHONY: deps-audit

fix-diff: ### Show code changes by `go fix`
	go fix -diff ./...
.PHONY: fix-diff

format: ### Run code formatter
	go fix ./...
	gofumpt -l -w .
	gci write . --skip-generated -s standard -s default
.PHONY: format

# Port of the admin web in `make dev`; the API keeps HTTP_PORT from .env.
WEB_PORT ?= 4200

dev: web/node_modules ### Run Postgres, the API (with migrations) and the admin web on :4200 together; SKIP_DB=1 to use your own Postgres
ifneq ($(SKIP_DB),1)
	$(BASE_STACK) up -d db
endif
	npm --prefix web exec -- concurrently --kill-others --names api,web --prefix-colors blue,green \
		"go run -tags migrate ./cmd/app" \
		"npm --prefix web start -- --port $(WEB_PORT)"
.PHONY: dev

# Set here rather than as `VAR=x cmd`: on Windows concurrently runs the commands through cmd.exe.
dev: export CGO_ENABLED = 0
dev: export API_URL = http://localhost:$(HTTP_PORT)

web/node_modules: web/package-lock.json
	npm --prefix web ci
	@touch web/node_modules

run: deps swag-v1 ### swag run for API v1
	go mod download && \
	CGO_ENABLED=0 go run -tags migrate ./cmd/app
.PHONY: run

docker-rm-volume: ### remove the Postgres docker volume
	docker volume rm go-clean-template_db_data
.PHONY: docker-rm-volume

linter-golangci: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

linter-hadolint: ### check by hadolint linter
	git ls-files --exclude='Dockerfile*' --ignored | xargs hadolint
.PHONY: linter-hadolint

linter-dotenv: ### check by dotenv linter
	dotenv-linter
.PHONY: linter-dotenv

test: ### run test
	go test -v -race -covermode atomic -coverprofile=coverage.txt ./internal/... ./pkg/...
.PHONY: test

mock: ### run mockgen
	mockgen -source ./internal/repo/contracts.go -package usecase_test > ./internal/usecase/mocks_repo_test.go
	mockgen -source ./internal/usecase/contracts.go -package usecase_test > ./internal/usecase/mocks_usecase_test.go
.PHONY: mock

migrate-create:  ### create new migration
	migrate create -ext sql -dir migrations '$(word 2,$(MAKECMDGOALS))'
.PHONY: migrate-create

migrate-up: ### migration up
	migrate -path migrations -database '$(PG_URL)?sslmode=disable' up
.PHONY: migrate-up

bin-deps: ### install tools
	go install tool
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate
.PHONY: bin-deps

pre-commit: deps swag-v1 mock format linter-golangci test ### run pre-commit
.PHONY: pre-commit
