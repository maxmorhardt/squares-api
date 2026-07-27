# AGENTS

## Project Context

- Project: Squares API (`github.com/maxmorhardt/squares-api`)
- Language: Go (`go 1.26`)
- Purpose: Backend for a real-time football squares pool. Gin HTTP API over PostgreSQL via GORM, OIDC-authenticated, with NATS pub/sub fanning WebSocket events across replicas.
- Target environments: Linux containers (`amd64`/`arm64`) on Kubernetes, multiple replicas behind a single Envoy Gateway.
- Related repos: `charts` (the `squares-api` Helm chart), `k8s` (the Argo CD Application that deploys it), `squares` (the frontend), `workflows` (shared CI).

## Repository Layout

- `cmd/main.go` - process entrypoint. Loads env, builds the bootstrap, starts the server.
- `internal/bootstrap/` - composition root. The only place concrete handlers, services, and repositories are constructed and wired.
- `internal/config/` - env loading, typed config, DB/NATS/OIDC init, and schema migrations (`migrate.go` plus embedded `migrations/*.sql`).
- `internal/errs/` - sentinel errors handlers map to HTTP status codes.
- `internal/handler/` - Gin handlers, one file per resource. Each defines its own interface so it can be stubbed.
- `internal/metrics/` - Prometheus collectors, registered via package `init()`.
- `internal/middleware/` - auth, CORS, logging, Prometheus, request size.
- `internal/model/` - GORM entities, request/response DTOs, swagger types, context keys.
- `internal/repository/` - GORM data access, one file per aggregate root.
- `internal/routes/` - route registration grouped by resource.
- `internal/service/` - business logic, validation, cross-aggregate orchestration, NATS publishes.
- `internal/util/` - logger-from-context, error helpers, capitalization.
- `test/` - testcontainers integration suite (real Postgres and NATS, needs Docker).
- `docs/` - swag-generated OpenAPI output.

## Core Principles

1. Strict layering
   - Handler to service to repository, in one direction only.
   - Handlers parse requests, call exactly one service method, and map domain errors to status codes. They never touch the DB.
   - Repositories take and return models, take a `context.Context`, and know nothing about HTTP.
2. Interfaces at every boundary
   - Declare each interface alongside its implementation so production wiring stays obvious.
   - Constructors are `NewXxx(...)` and return the interface type.
   - Mocks are generated from those interfaces with mockery, never hand-written.
3. Explicit, typed errors
   - Services return sentinel errors from `internal/errs` so handlers can map them.
   - Services translate `gorm.ErrRecordNotFound` into a domain error; repositories pass GORM errors through unchanged.
   - Wrap with `fmt.Errorf("...: %w", err)` only when adding real context.
4. Context is a first-class parameter
   - `context.Context` is the first parameter of any function crossing a layer or touching the DB or NATS.
   - Pull loggers from context with `util.LoggerFromContext(ctx)`. Never call `slog.Default()` in a service or repository.
5. Migrations own the schema
   - Models no longer drive schema. Changing a model means adding a migration too.
   - Migrations are embedded and applied at startup under an advisory lock, so they must be safe across concurrent replicas.
6. Typed domain data
   - Never use `interface{}` or `any` for domain data. Define a struct.
7. Identity stays a plain value
   - Handlers read identity with `c.GetString(model.UserKey)` and pass it down as a `string`. Services never depend on `*gin.Context`.

## Agent Instructions

- Make the smallest safe change that solves the requested problem, and keep it inside existing package boundaries.
- **The Makefile is the canonical task runner.** Prefer `make <target>` over raw `go` or tool invocations. CI runs the same targets, so the Makefile is the single source of truth.
- One logical resource per file, lowercase names (`contest_service.go`, not `services.go`).
- After changing any interface, run `make mocks` to regenerate `internal/mocks`.
- After changing handler annotations, run `make swag` to regenerate `docs/`.
- `prealloc`, `gocritic`, and `unparam` are enabled, so give slices capacity hints and drop unused parameters.
- Avoid comments unless the code is genuinely non-obvious. Prefer expressive names.
- When adding a NATS subject, update the WS service consumer in the same change or the event is published into the void.
- Do not change the Helm chart from this repo. Coordinate through the `charts` workspace.

## New Endpoint Checklist

1. Add the method to the handler interface and implement it, with swag annotations (`@Summary`, `@Tags`, `@Param`, `@Success`, `@Router`, `@Security BearerAuth`).
2. Add the service method and interface entry if new business logic is needed.
3. Add the repository method and interface entry if new persistence is needed.
4. Add a migration if the schema changes.
5. Register the route in `internal/routes/<resource>_routes.go`.
6. Run `make mocks`.
7. Add handler tests for success, validation error, and service error.
8. Run `make swag`.

## Testing Guidance

- One `_test.go` per source file. No `testutil` or `_internal` test files.
- White-box (`package <pkg>`) for unexported helpers; black-box (`package <pkg>_test`) when the test needs mockery mocks, since `internal/mocks` imports the real packages and would otherwise cycle.
- Use the testify expecter API: `m := mocks.NewContestService(t); m.EXPECT().CreateContest(...).Return(...)`.
- Repositories are tested with go-sqlmock (pure Go, no cgo). Set `mock.MatchExpectationsInOrder(false)`, because GORM preloads run in non-deterministic order.
- Run targeted package tests first, then `make test` for the full unit run. `make test-integration` needs a working Docker daemon; on Windows use Docker Desktop, since testcontainers does not support rootless Docker.
- Coverage is gated at **80%** by `.testcoverage.yml` via `make cover`, measuring `handler`, `service`, `util`, `middleware`, `bootstrap`, `config`, and `repository`. Do not lower the threshold to get a build green.
- Always run `make test` and `make lint` before committing.

## Dependency Checklist

Before adding a new dependency, verify:

- Can this be done with the stdlib or an existing internal package?
- What is the transitive dependency and maintenance impact?
- Does it pull in cgo? Repository tests are deliberately cgo-free.
- Does it duplicate something Gin, GORM, or the NATS client already provides?
- Is the trade-off recorded in the commit rationale?

## Commit Tags

Conventional commits, enforced on PR titles and consumed by release-please. The type determines the release, so it is a functional choice, not a stylistic one.

- `feat`: New user-facing capability. Cuts a minor release.
  - Use for a new endpoint, a new NATS event, or new behavior on an existing route.
- `fix`: Corrects wrong behavior, a regression, a crash, or a security issue. Cuts a patch release.
  - Use for failure-path and edge-case corrections. Renovate security PRs land as `fix(deps)` for this reason.
- `refactor`: Restructuring with no behavior change.
- `chore`: Maintenance that is not user-facing, including routine dependency bumps.
- `ci`: Workflow, build, or release automation changes.
- `docs`: Documentation only.
- `test`: Test-only additions or maintenance.
- `style`: Formatting only.

Optional scopes: `handler`, `service`, `repository`, `routes`, `middleware`, `model`, `bootstrap`, `ws`, `nats`, `auth`, `tests`, `build`, `deploy`.

Example commit subjects:

- `feat(handler): add search query param to /contests/me`
- `fix(service): exclude deleted contests from stats queries`
- `refactor(repository): collapse duplicate preload chains`
- `ci: run integration tests with the race detector`

## Non-Goals for Routine Changes

- Large refactors without a clear user-facing benefit.
- Bypassing a layer, for example querying the DB from a handler.
- New dependencies when the stdlib or an existing internal package is sufficient.
- Schema changes made by editing a model without a matching migration.
- Lowering the coverage threshold or disabling a linter to get a build green.
- Editing the Helm chart or the Argo CD Application from this repo.
