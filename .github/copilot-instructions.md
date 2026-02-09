# Biometrics — Copilot Instructions

## Project Overview

Go web application for tracking daily weight and water intake. PostgreSQL
backend, vanilla JS frontend. Module path: `biometrics`.

## Language & Tooling Rules

- This is a **Go project**. All source code must be Go.
- Scripts (build helpers, codegen, migrations) must be **shell (bash/zsh)** or
  **Go**. **Never use Python** or other scripting languages.
- Use `go test`, `go build`, `go vet`, and `go fmt` for all validation.
- Use the **`gh` CLI** for all GitHub interactions (creating PRs, checking CI,
  merging).

## Architecture — Hexagonal (Ports & Adapters)

The codebase follows hexagonal architecture. Respect the dependency rule:
**dependencies always point inward**.

```
cmd/biometrics/          ← entry point, wires everything together
internal/
  domain/                ← core: entities + port interfaces (ZERO external deps)
  app/                   ← application services (depend ONLY on domain ports)
  adapter/
    postgres/            ← driven adapter: implements domain repository ports
    http/                ← driving adapter: HTTP handlers calling app services
web/                     ← static frontend assets (HTML/CSS/JS)
```

### Layer rules

| Layer | May import | Must NOT import |
|---|---|---|
| `domain` | stdlib only | `app`, `adapter`, any DB/HTTP library |
| `app` | `domain` | `adapter`, any DB/HTTP library |
| `adapter/postgres` | `domain`, `database/sql`, `github.com/lib/pq` | `app`, `adapter/http` |
| `adapter/http` | `domain`, `app`, `net/http` | `adapter/postgres` |
| `cmd/biometrics` | everything (wiring) | — |

### Key conventions

- **Domain entities** live in `internal/domain/` — these are the canonical types
  used across all layers.
- **Port interfaces** (repositories) are defined in `internal/domain/` alongside
  the entities they operate on.
- **Application services** in `internal/app/` contain all business logic and
  validation. They accept domain port interfaces via constructor injection.
- **Adapters** implement or consume domain ports. The postgres adapter
  implements repository interfaces. The HTTP adapter calls app services.
- When adding a new feature, start from the domain (entity + port), then service,
  then adapter(s).

## Git Workflow & Branch Strategy

**All changes must be on branches — never commit directly to `master`.**

### Branch naming

Use descriptive prefixed branch names:
- `feat/<short-description>` — new features
- `fix/<short-description>` — bug fixes
- `refactor/<short-description>` — structural changes
- `test/<short-description>` — adding/improving tests
- `docs/<short-description>` — documentation only

### PR workflow (use `gh` CLI)

1. Create a branch: `git checkout -b <type>/<description>`
2. Make focused commits with clear messages.
3. Push and open a PR: `gh pr create --title "..." --body "..."`
4. Verify CI passes: `gh pr checks`
5. After review/approval: `gh pr merge --squash`

### PR size constraint

**PRs must bias towards 500 lines of code or less.**

- Break large changes into a series of small, focused, independently-shippable
  PRs.
- Each PR must compile (`go build ./...`) and pass all tests
  (`go test ./...`) on its own.
- If a refactor requires multiple PRs, stack them sequentially — each building
  on the previous merged PR.
- Prefer many small PRs over one big one. It's fine to have 5 PRs for a single
  feature if they are each focused and reviewable.

### Commit messages

Follow conventional commits:
- `feat: add water intake chart endpoint`
- `fix: correct weight unit conversion rounding`
- `refactor: extract weight repository interface to domain layer`
- `test: add service-layer tests for water undo`
- `docs: update README with hexagonal architecture overview`

## Testing Requirements

**Every PR that changes Go code must include or update tests.**

### Test strategy by layer

| Layer | Test type | Dependencies | Location |
|---|---|---|---|
| `domain` | Unit tests | None (pure logic) | `internal/domain/*_test.go` |
| `app` | Unit tests with mocks | Mock implementations of domain ports | `internal/app/*_test.go` |
| `adapter/http` | HTTP integration tests | `httptest` + stub services or mock repos | `internal/adapter/http/*_test.go` |
| `adapter/postgres` | Integration tests | Real DB (skip in CI if unavailable) | `internal/adapter/postgres/*_test.go` |

### Testing rules

- Use the standard `testing` package. No external test frameworks.
- Use **table-driven tests** where there are multiple cases.
- Mock repository interfaces using simple struct-with-function-fields pattern
  (no codegen mocking libraries needed).
- Test happy paths, error paths, and edge cases.
- Run the full suite before every PR: `go test ./...`
- Aim for meaningful coverage — focus on behaviour and error paths, not a
  coverage percentage.

### Example mock pattern

```go
type mockWeightRepo struct {
    addFn    func(ctx context.Context, v float64, u string, t time.Time) (int64, error)
    latestFn func(ctx context.Context, day string) (*domain.WeightEntry, error)
    // ... other methods
}

func (m *mockWeightRepo) AddWeightEvent(ctx context.Context, v float64, u string, t time.Time) (int64, error) {
    if m.addFn != nil {
        return m.addFn(ctx, v, u, t)
    }
    return 0, nil
}
```

## Code Style

- Run `gofmt` / `goimports` on all Go files.
- Keep packages small and focused.
- Prefer returning `error` over panicking.
- Use `context.Context` as the first parameter in all repository and service
  methods.
- Exported types and functions require doc comments.
- Keep handlers thin — validation and business logic belong in app services.

## Build & Run

```bash
# Build
go build ./...

# Test
go test ./...

# Run locally
DATABASE_URL="postgres://user:pass@localhost:5432/biometrics?sslmode=disable" \
  go run ./cmd/biometrics

# Docker
docker build -t biometrics .
docker run -e DATABASE_URL="..." -p 8080:8080 biometrics
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string |
| `ADDR` | `:8080` | Listen address |
| `WEB_DIR` | `web` | Path to static frontend assets |
