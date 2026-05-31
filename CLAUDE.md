# Squares API Contribution Guide

This guide provides context for coding agents working in this repository. Squares API is the backend for a real-time football squares pool: a Go + Gin + GORM service backed by PostgreSQL, fronted by OIDC auth, with NATS for pub/sub fan-out of WebSocket events. It ships as a Docker image and is deployed via the sibling `charts/squares-api` Helm chart.

## Directory overview

- `cmd/main.go` – process entrypoint. Loads env, builds the bootstrap, starts the server.
- `internal/` – application source (private, not intended for import by other modules).
  - `bootstrap/` – wires the app: HTTP server, GORM DB, NATS, OIDC verifier, validators, Prometheus metrics.
  - `config/` – env loading, typed config structs, DB/NATS/OIDC init, and schema migrations (`migrate.go` + embedded `migrations/*.sql`, run at startup).
  - `errs/` – sentinel errors (`ErrContestNotFound`, `ErrInsufficientRole`, etc.) used to map service errors → HTTP status codes in handlers.
  - `handler/` – Gin HTTP handlers, one file per resource (`contest_handler.go`, `participant_handler.go`, …). Each handler defines its own interface so it can be stubbed in tests.
  - `metrics/` – Prometheus collectors (counters, histograms). Each file (`http.go`, `auth.go`, `business.go`, `nats.go`, `request_size.go`, `ws.go`) defines its metrics and registers them via a package `init()`. `bootstrap/metrics.go` only starts the Prometheus HTTP scrape server.
  - `middleware/` – Gin middleware (`auth.go` for OIDC JWT, `cors.go`, `logger.go`, `prometheus.go`, `request_size.go`).
  - `model/` – GORM entities (`contest.go`, `square.go`, `participant.go`, …), request/response DTOs, swagger types, context keys.
  - `repository/` – GORM data access. One file per aggregate root.
  - `routes/` – Gin route registration grouped by resource.
  - `service/` – business logic; orchestrates repositories, NATS, validation. Defines its own interfaces consumed by handlers.
  - `templates/` – HTML email templates.
  - `util/` – cross-cutting helpers (logger from context, error helpers, capitalization).
- `test/` – Testcontainers-driven integration tests. Spins up Postgres + NATS in Docker.
- `docs/` – swag-generated OpenAPI docs (`docs.go`, `swagger.json`, `swagger.yaml`).
- `Dockerfile` – production image. Single-stage on `alpine`: CI builds the binary with `make build` (cross-compiled to `linux/arm64`), uploads it as an artifact, and the Docker build copies that prebuilt binary into the image.
- `make nats` – local helper to port-forward a NATS server for development.
- `.env`, `.env.test` – local secrets (gitignored). `.env.example` and `.env.test.example` are committed templates.

## Tooling

- Language: **Go 1.26.x** (see [go.mod](go.mod)).
- **The [Makefile](Makefile) is the canonical way to run tasks — prefer `make <target>` over raw `go`/tool commands.** Run `make help` to list targets. CI (the shared `workflows/ci-go.yml`) invokes the same targets, so the Makefile is the single source of truth for build/test/lint/coverage flow; the raw commands noted below are just what each target runs. The lint step is the one exception CI runs via the `golangci-lint-action` (for reliable install/caching), against the same `.golangci.yml`.
- Run/build: `make run` / `make build`. Build is overridable for cross-compiles: `make build OUT=… MAIN=… BUILD_FLAGS=… LDFLAGS=…` with `GOOS`/`GOARCH`/`CGO_ENABLED` env (this is how CI produces the linux/arm64 binary).
- Lint: `make lint` (runs **golangci-lint** with [.golangci.yml](.golangci.yml)). Enabled linters include `errcheck`, `govet` (with `shadow`), `staticcheck`, `gosec`, `errorlint`, `gocritic`, `prealloc`, `unparam`. Also `make vet` for `go vet`, `make verify` for `go mod verify`.
- Tests: stdlib `go test` + **testify** (`assert`, `require`). `make test` for unit (no Docker), `make test-integration` for the testcontainers suite, `make test-all` for everything. Pass `RACE=-race` to enable the race detector (CI does).
- Coverage: enforced via [.testcoverage.yml](.testcoverage.yml) (`profile: coverage.out`, `total: 80`). `make cover` runs the unit tests with a coverage profile and the `go-test-coverage` gate. The gate measures `handler`, `service`, `util`, `middleware`, `bootstrap`, `config`, and `repository`; `mocks` (generated), `routes`, `metrics`, `model`, `errs`, `templates`, `cmd`, `docs` are excluded (generated/trivial). Interfaces are mocked with **mockery** (`make mocks` → `internal/mocks`, config in [.mockery.yaml](.mockery.yaml)); repositories use **go-sqlmock** unit tests (no cgo). One `_test.go` per source file — no `testutil`/`_internal` test files.
- Integration tests in `test/` use **testcontainers-go** to launch Postgres + NATS. They require a working Docker daemon — on Windows, rootless Docker is **not** supported by testcontainers, so use Docker Desktop or run just unit tests with `make test`.
- Swagger docs: regenerate with `make swag` after changing handler annotations.
- Dependency upgrades: `make deps` (`go get -u -t ./... && go mod tidy`); `make tidy` for tidy alone.
- Migrations: schema is managed by **golang-migrate** SQL files in `internal/config/migrations/`, embedded via `go:embed` and applied on startup in `config/setupPrimary` (advisory-locked, safe across replicas). Create a new pair with `make migrate-create NAME=add_foo`; `make migrate-up`/`migrate-down` (set `DATABASE_URL`) for manual ops. The baseline `000001_init` uses `IF NOT EXISTS` so it cleanly versions a pre-existing AutoMigrate'd database. Models in `internal/model` no longer drive schema — change a model **and** add a migration.

## Architecture

The codebase follows a strict **handler → service → repository** layering:

- **Handlers** (`internal/handler/`) parse requests, call exactly one service method, and translate domain errors (from `internal/errs/`) into HTTP status codes via a `switch errors.Is(err, …)` block. They never touch the DB directly.
- **Services** (`internal/service/`) own business logic, validation, and cross-aggregate orchestration. They depend on repository interfaces and may publish NATS events. Each service exposes its own interface in the same file as the implementation.
- **Repositories** (`internal/repository/`) wrap GORM. They take and return models, take a `context.Context`, and never know about HTTP. Each repository exposes an interface for testability.
- **Bootstrap** (`internal/bootstrap/server.go`) is the composition root — the only place where concrete handler/service/repository structs are constructed and wired together. Routes are registered there via the helpers in `internal/routes/`.

## Code style

- Use lowercase, single-package files; one logical resource per file (`contest_service.go`, not `services.go`).
- Define dependencies as **interfaces**. Service interfaces are declared alongside their implementation in `internal/service/<resource>_service.go` and consumed by handlers (e.g. handler structs hold a `service.ContestService`). Repository interfaces are declared in `internal/repository/<resource>_repository.go` and consumed by services. Keeping each interface next to its implementation keeps the production wiring obvious; mocks are generated from the interfaces with **mockery** into `internal/mocks` (`make mocks`).
- Constructors are `NewXxx(...)` and return the interface type.
- Use `context.Context` as the **first parameter** of any function that crosses a layer or talks to the DB/NATS.
- Logging: pull a `*slog.Logger` from context with `util.LoggerFromContext(ctx)` (or `util.LoggerFromGinContext(c)` in handlers). Never call `slog.Default()` directly inside services/repositories.
- Errors: return sentinel errors from `internal/errs` from services so handlers can map them. Wrap lower-level errors with `fmt.Errorf("...: %w", err)` only when adding context. Repositories typically pass GORM errors through unchanged; services translate `gorm.ErrRecordNotFound` to a domain error.
- Avoid comments unless the code is genuinely non-obvious. Prefer expressive names.
- Never use `interface{}` / `any` for domain data — define a struct.
- `prealloc`, `gocritic`, and `unparam` are enabled, so allocate slices with capacity hints and remove unused parameters.

## Routes & handlers

- Route groups are registered in `internal/routes/*_routes.go` and called from `internal/bootstrap/server.go`.
- New endpoint workflow:
  1. Add the method to the relevant handler interface and implement it. Include `swag` annotations (`@Summary`, `@Tags`, `@Param`, `@Success`, `@Router`, `@Security BearerAuth`).
  2. If the endpoint requires new business logic, add a method to the corresponding service interface + implementation.
  3. If new persistence is needed, add a repository method + interface.
  4. Register the route in `internal/routes/<resource>_routes.go`.
  5. Run `make mocks` to regenerate the mockery mock for the changed interface.
  6. Add handler-level test cases (success, validation error, service error) following the existing patterns.
  7. Regenerate swagger docs.

## Authentication & context

- Protected routes are gated by `middleware.AuthMiddleware()` (in `internal/middleware/auth.go`), which validates the OIDC JWT and stores claims under context keys defined in `internal/model/key.go` (e.g. `model.UserKey`, `model.ClaimsKey`).
- Read identity in handlers with `c.GetString(model.UserKey)` and pass it down as a plain `string` argument — services should not depend on `*gin.Context`.

## Real-time / NATS

- WebSocket connections are upgraded in `internal/handler/ws_handler.go` and managed by `internal/service/ws_service.go`.
- Cross-instance broadcasting goes through NATS subjects defined in `internal/service/nats_service.go`. When publishing a domain event from a service, also update the WS service consumer if a new subject is introduced.

## Testing

- One `_test.go` per source file (no `testutil`/`_internal` test files). Tests of unexported helpers use white-box `package <pkg>`; tests that need mockery mocks use black-box `package <pkg>_test` (the `internal/mocks` package imports the real packages, so white-box would create an import cycle).
- Interface dependencies are mocked with **mockery** (`make mocks` regenerates `internal/mocks`; config in `.mockery.yaml`). Use the testify expecter API, e.g. `m := mocks.NewContestService(t); m.EXPECT().CreateContest(...).Return(...)`. The `NatsService` mock is stubbed permissively (publishes are fire-and-forget goroutines). Handler tests use `gin.New()` / `authenticatedMiddleware(user)` helpers in `contest_handler_test.go`.
- Repositories have **go-sqlmock** unit tests (pure Go, no cgo) plus end-to-end coverage from the integration tests in `test/` (real Postgres + NATS via testcontainers, needs Docker). sqlmock tests set `mock.MatchExpectationsInOrder(false)` since GORM preloads run in non-deterministic order.
- After changing an interface, run `make mocks` to regenerate its mock.
- Run `make test` (or `go test ./...`) and `make lint` before committing. The 80% coverage threshold is enforced in CI via `make cover`.

## Deployment

- The production image is built from the [Dockerfile](Dockerfile) (single-stage; CI pre-builds the binary and the Docker build only copies it in) and pushed via CI. Runtime config comes from environment variables (see `.env.example`).
- Helm chart lives in the sibling `charts/squares-api/` workspace folder. Don't change the chart from this repo unless explicitly asked — coordinate via that workspace.

## Commit conventions

Use conventional commits. Common types and scopes for this repo:

- Types: `feat`, `fix`, `refactor`, `chore`, `ci`, `docs`, `test`, `style`.
- Scopes (optional): `handler`, `service`, `repository`, `routes`, `middleware`, `model`, `bootstrap`, `ws`, `nats`, `auth`, `tests`, `build`, `deploy`.

Example: `feat(handler): add search query param to /contests/me`.

Always run `go test ./...` and `golangci-lint run` before committing.
