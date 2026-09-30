<!-- readme-type: service -->
# vitals

Mobile-friendly Go web app for logging daily weight and water intake

Tracking daily weight and water intake usually means a spreadsheet or a
general-purpose fitness app with far more surface area than the job needs.
Vitals is a single-tenant, single-binary Go app that does just those two
things, with a mobile-first UI and no JS build step, meant to be self-hosted
by one deployment's one user account.

**Status:** in daily use on the homelab — deployed to staging and production
since 2026-09-13.

## Quick start

Needs: Go 1.27.

```bash
git clone https://github.com/gjcourt/vitals && cd vitals
go run ./cmd/vitals
```

Open http://localhost:8080. On first run there are no users yet, so you're
sent to `/signup` to create the one account the deployment will use; after
that, everyone signs in at `/login`. With no further configuration, data is
kept in memory and lost on restart; set `SQLITE_PATH` (see Configuration) for
a run that survives a restart.

## Usage

Log today's weight, view recent entries and trend charts, and log water
intake through the mobile-first UI at `/`; the same actions are available as
JSON under `/api` (see the routes in [`docs/reference/`](docs/reference/)).

## Configuration

All configuration is via environment variables; there are no config files or
CLI flags.

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | Listen address |
| `WEB_DIR` | `web` | Path to static frontend assets |
| `SQLITE_PATH` | *(unset)* | Path to a SQLite database file. Unset falls back to an in-memory store (dev default, wiped on restart). |
| `SSO_ISSUER_URL` | *(unset)* | OIDC issuer URL. Setting this enables an SSO login option alongside username/password. |
| `SSO_CLIENT_ID` | *(unset)* | OIDC client ID |
| `SSO_CLIENT_SECRET` | *(unset)* | OIDC client secret |
| `SSO_REDIRECT_URL` | *(unset)* | OIDC redirect URL registered with the identity provider |

Sessions are cookie-based (bcrypt-hashed passwords, random session tokens). A
reverse proxy that sets a `Remote-User` header (e.g. Authelia forward-auth)
is also honored and takes priority over the cookie.

## How it works

Vitals follows hexagonal (ports & adapters) architecture: `internal/domain`
holds entities and repository interfaces with zero external deps,
`internal/app` holds validation and business logic, and
`internal/adapter/http|memory|sqlite` implement the driving HTTP layer and
the two interchangeable stores. `cmd/vitals/main.go` picks a store based on
`SQLITE_PATH`, so the same `app` services run unchanged against either one.
Weight and water are stored as append-only events; "today's value" and
totals are derived from them, and undo removes the most recent event. See
[`docs/README.md`](docs/README.md) for the full documentation set, including
the architecture reference.

## Development

```bash
golangci-lint run ./...
go-arch-lint check
make test
```

`make build` compiles the binary to `./vitals`; `make all` runs clean, lint,
test and build. Conventions for contributors and agents:
[AGENTS.md](AGENTS.md).

## Deployment

Runs on the homelab, staging and production, built and pushed to
`ghcr.io/gjcourt/vitals` by `.github/workflows/image.yml` on every push to
`master`. See [`gjcourt/homelab`](https://github.com/gjcourt/homelab)
`apps/production/vitals/` and `apps/staging/vitals/` for the deployed
manifests.

## License

No licence file yet.
