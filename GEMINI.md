# Go Backend Master Class (eats)

## Project Overview
This project is a Go-based backend application (named `eats`) structured as a modular monolith. It follows domain-driven design principles and clean architecture (ports and adapters), with a clear separation of concerns into adapters (database), HTTP API layer, and application core (`app`).

**Key Technologies:**
- **Language:** Go 1.25+
- **HTTP Framework:** [Echo v4](https://echo.labstack.com/)
- **Database:** PostgreSQL accessed via [pgx](https://github.com/jackc/pgx)
- **Database Queries:** [SQLC](https://sqlc.dev/) for generating type-safe Go code from SQL.
- **API Design:** OpenAPI specifications with code generation via [oapi-codegen](https://github.com/deepmap/oapi-codegen).
- **Task Runner:** [Task](https://taskfile.dev/) (`Taskfile.yml`) for executing project scripts.
- **Testing:** Standard `go test`, Testify, and Go-Cmp for assertions, with dedicated targets for unit, integration, and component tests.
- **Logging:** Structured logging using Go's standard `log/slog` and `ThreeDotsLabs/humanslog`.

## Directory Overview
The workspace uses Go Workspaces (`go.work`) targeting a single module located in the `./project` directory.

- `project/backend/cmd/main.go`: The main entry point of the application.
- `project/backend/common/`: Shared utilities and cross-cutting concerns (logging, errors, HTTP middlewares, UUIDs).
- `project/backend/orders/`: A dedicated bounded context/module for the "Orders" domain. It contains its own:
  - `adapters/db/`: SQLC queries, database models, and repository implementations.
  - `api/http/`: OpenAPI specifications (`openapi.yaml`) and generated Echo handlers.
  - `app/`: Core business logic and service definitions.
  - `module.go`: The wiring for the module's dependencies.
- `project/backend/tests/`: End-to-end component tests for the application.
- `project/docker-compose.yaml`: Infrastructure setup (likely containing PostgreSQL).
- `project/Taskfile.yml`: Project tasks (running tests, generating code, linting, etc.).

## Building and Running

The project relies heavily on `Taskfile.yml` for execution and automation. Ensure you have `task` installed. Most commands are expected to be run within the `project/` directory or at the root if configured appropriately.

**Start Infrastructure:**
```bash
task up
```
*(Runs `docker compose up` to start dependencies like the database)*

**Run the Application:**
You need the `POSTGRES_URL` environment variable configured.
```bash
POSTGRES_URL="postgresql://user:password@localhost:5432/eats" go run ./project/backend/cmd/main.go
```

**Testing:**
```bash
task test            # Runs all tests (unit, integration, component)
task test-unit       # Runs only unit tests
task test-integration# Runs integration tests (requires DB)
task test-component  # Runs component tests
```

**Code Generation:**
```bash
task gen
```
*(Triggers `go generate ./...` to update SQLC queries, OpenAPI handlers, and mocks, followed by formatting).*

## Development Conventions
- **Code Generation:** Never manually edit files that end in `.gen.go` or `.sql.go`. Modify the source files (e.g., `openapi.yaml` or `.sql` files) and run `task gen`.
- **Formatting & Linting:** Code is formatted using `gofumpt`. Before committing, always run:
  - `task fmt` to format code.
  - `task lint` to run `golangci-lint`.
  - `task tidy` to clean up `go.mod`.
- **Error Handling:** Centralized error handling types appear to be defined in `backend/common/errors.go` and mapped to HTTP responses in `errors_echo.go`.
- **Dependency Injection:** Dependencies are constructed and injected in `main.go` and `module.go` files rather than relying on global state or init functions.
