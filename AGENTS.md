# AGENTS.md

> Vitals is a mobile-friendly Go web app for tracking daily weight and water intake. — https://github.com/gjcourt/vitals

## Commands

| Command | Use |
|---------|-----|
| `make build` | Compile binary to `./vitals` |
| `make run` | Build + run |
| `make dev` | Run with live-reload friendly flags |
| `make test` | Run tests with race detector |
| `make lint` | golangci-lint |
| `make clean` | Remove build artifacts |
| `make all` | clean + lint + test + build |

Single test: `go test ./internal/app -run TestEntry -v`
Pre-push: `make all`

## Architecture

Hexagonal architecture (ports & adapters). Entry point: `cmd/vitals/main.go`.

- `internal/domain/` — entity types and core business rules (entries, users).
- `internal/app/` — application orchestration; calls into domain and storage adapters.
- `internal/adapter/http/` — driving adapter (HTTP server, handlers, templates).
- `internal/adapter/memory/` — in-memory storage adapter.
- `internal/adapter/postgres/` — PostgreSQL storage adapter.
- `web/` — frontend (HTML templates, CSS, vanilla JS).

Today there is no explicit `internal/ports/` package; storage adapters implement domain interfaces directly. See `docs/architecture/` for the overview.

## Conventions

- **Domain has zero external deps** — no DB imports, no HTTP libs in `internal/domain/`.
- **Adapters implement domain interfaces** — business logic never leaks into adapters.
- **Test files co-located** with implementation (`_test.go` in the same package).
- **Conventional Commits** for every commit (`feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`, `ci:`).
- **Branch names** follow `<type>/<description>`.

## Invariants

- `internal/domain/` must not import any third-party packages outside stdlib.
- `internal/app/` must not import `internal/adapter/` directly — it depends on interfaces.
- The compiled binary lives at `./vitals`; never committed.

## What NOT to Do

- Do not import `database/sql` or HTTP types from `internal/domain/`.
- Do not import `internal/adapter/` from `internal/app/` — depend on the interface, not the implementation.
- Do not skip `make lint` and `make test` before committing.
- Do not commit credentials or local DB dumps.

## Domain

Personal health-tracking web app. Users log a daily weight in kg and water intake in mL; the UI surfaces recent entries and basic trends. Single-tenant per deployment; mobile-first layout.

## Cross-service dependencies

| Service | Interface | Purpose |
|---|---|---|
| PostgreSQL | `internal/adapter/postgres` | Production entry storage |
| In-memory | `internal/adapter/memory` | Default / ephemeral storage for dev |

Deployed in the homelab cluster; image-tag bumps must be coordinated with the corresponding manifests under `../homelab/`.

## Container image

`.github/workflows/image.yml` builds and pushes the image to GHCR on every push to `master` (plus manual `workflow_dispatch`). It authenticates with the built-in `GITHUB_TOKEN` — no PAT — and builds multi-arch (`linux/amd64,linux/arm64`). This replaces the old manual `make image` flow.

**Image:** `ghcr.io/gjcourt/vitals`

Each build publishes three tags:

| Tag | Mutability | Use |
|---|---|---|
| `YYYY-MM-DD` | mutable — a later same-day build overwrites it | build date (UTC) |
| `YYYY-MM-DD-<sha7>` | **immutable & unique** | **the tag to pin in deployments** |
| `latest` | mutable — always the newest build | convenience |

**Deploying:** after a push to `master`, read the exact published tag from the `image.yml` run (or `gh api user/packages/container/vitals/versions`), then pin the `YYYY-MM-DD-<sha7>` tag in `homelab/apps/base/vitals/deployment.yaml`.

**First-build gotcha:** if a `GITHUB_TOKEN` push ever 403s, the GHCR package exists but is unlinked (created by an old manual PAT push) — delete it (`gh api --method DELETE user/packages/container/vitals`, needs the `delete:packages` scope) so the next run recreates it auto-linked, then re-run.

## Quality gate before push

1. `make lint`
2. `make test`
3. `make build`

Or `make all`, which runs the lot.

## Documentation

`docs/` taxonomy: `architecture/` · `design/` · `operations/` · `plans/` · `reference/` · `research/`. See each folder's `README.md` for scope. Index: `docs/README.md`.

Full architecture reference (layers, dependency diagram, request flow, ports & adapters map, boundary guard): [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md).

## Observability

Logs to stderr in slog text format. No metrics endpoint today; cluster-level pod status is the source of health signal.

When you learn a new convention or invariant in this repo, update this file.
