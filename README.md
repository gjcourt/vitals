# Vitals

A mobile-friendly Go web app for tracking daily weight and water intake.
Single binary, single tenant, no JS build step.

## Quickstart

```bash
go run ./cmd/vitals
```

Open http://localhost:8080. On first run there are no users yet, so you're
sent to `/signup` to create the one account the deployment will use; after
that, everyone signs in at `/login`. With no further configuration, data is
kept in memory and lost on restart.

For durable storage, point it at a SQLite file:

```bash
SQLITE_PATH=vitals.db go run ./cmd/vitals
```

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

## API

All routes below are under `/api` and, except for `/health` and `/auth/*`,
require an authenticated session.

- `GET /api/health`
- `POST /api/auth/login`, `POST /api/auth/logout`
- `POST /api/auth/setup` — creates the first (and only) user; fails once a user exists
- `GET /api/auth/config` — reports whether SSO is enabled
- `GET /api/auth/oidc/login`, `GET /api/auth/oidc/callback` — SSO flow, 404 unless SSO is configured
- `GET /api/weight/today`, `PUT /api/weight/today` — body: `{ "value": 75.4, "unit": "kg" }`
- `GET /api/weight/recent?limit=14`
- `POST /api/weight/undo-last`
- `GET /api/water/today`
- `POST /api/water/event` — body: `{ "deltaLiters": 0.25 }`
- `GET /api/water/recent?limit=20`
- `POST /api/water/undo-last`
- `GET /api/charts/daily?days=90&unit=lb`

Weight and water are stored as append-only events; "today's value" and totals
are derived from them, and undo removes the most recent event.

## Architecture

Hexagonal (ports & adapters):

```
cmd/vitals/            entry point: reads env, picks a store, wires services + HTTP server
internal/
  domain/                  entities and repository interfaces (stdlib only)
  app/                     application services: validation and business logic
  adapter/
    http/                  driving adapter: routes, handlers, auth middleware
    memory/                driven adapter: in-memory store (dev default)
    sqlite/                driven adapter: SQLite store (durable storage)
web/                       static frontend: HTML, CSS, vanilla JS
```

Storage is pluggable behind the `domain` repository interfaces, so the same
`app` services run unchanged against either store; `main.go` picks one based
on `SQLITE_PATH`. See [`docs/README.md`](docs/README.md) for the full
documentation set, including the detailed architecture reference.

## Development

```bash
make build   # compile ./vitals
make test    # go test -race ./...
make lint    # golangci-lint
make all     # clean + lint + test + build
```

CI additionally runs `go-arch-lint` to enforce the dependency rule that
`domain` never imports `app` or `adapter`, and `app` never imports `adapter`.

## Container image

```bash
docker build -t vitals .
docker run -p 8080:8080 -e SQLITE_PATH=/data/vitals.db -v vitals-data:/data vitals
```

`.github/workflows/image.yml` builds and pushes multi-arch images
(`linux/amd64`, `linux/arm64`) to `ghcr.io/gjcourt/vitals` on every push to
`master`.
