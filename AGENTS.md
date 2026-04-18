# Vitals Agent Guidelines

## Repository Overview

Vitals is a lightweight, mobile-friendly web app for tracking daily weight and water intake. Built with Go, PostgreSQL, and vanilla JS. Follows **hexagonal (ports & adapters) architecture**.

## Project Structure

```
cmd/vitals/        ← entry point, wires everything together
internal/
  domain/          ← core business logic & interfaces (no external deps)
  adapters/        ← implementations (HTTP handlers, DB repositories)
web/               ← frontend (HTML templates, CSS, vanilla JS)
docs/              ← API, auth, database, architecture docs
scripts/           ← utility scripts
```

## Common Commands

```bash
make build         # compile binary
make run           # build and run
make test          # run tests with race detector
make lint          # run golangci-lint
make dev           # run with live-reload friendly flags
```

## Architecture Guidelines

- **Domain layer** (`internal/domain/`) must have zero external dependencies — no DB imports, no HTTP libs.
- **Adapters** implement domain interfaces; business logic never leaks into adapters.
- Add tests alongside implementation files (`_test.go` in the same package).
- Use `make lint` before committing — the CI pipeline runs the same linter.

## Development Notes

- Database: PostgreSQL. Schema lives in `docs/database.md` and migration scripts under `scripts/`.
- Authentication: see `docs/authentication.md` for session/token patterns used.
- API reference: `docs/api.md`.
