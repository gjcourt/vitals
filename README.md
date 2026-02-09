# Biometrics

A simple, mobile-friendly web app for tracking daily weight and water intake.
Built with Go, PostgreSQL, and vanilla JS.

## Architecture

The project follows **hexagonal (ports & adapters) architecture**:

```
cmd/biometrics/          ← entry point, wires everything together
internal/
  domain/                ← core: entities + port interfaces (zero external deps)
  app/                   ← application services (business logic + validation)
  adapter/
    postgres/            ← driven adapter: implements domain repository ports
    http/                ← driving adapter: HTTP handlers calling app services
web/                     ← static frontend assets (HTML/CSS/JS)
```

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

Then open http://localhost:8080

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | *(required)* | PostgreSQL connection string |
| `ADDR` | `:8080` | Listen address |
| `WEB_DIR` | `web` | Path to static frontend assets |

## API

- `GET /api/health`
- `GET /api/weight/today`
- `PUT /api/weight/today` — body: `{ "value": 75.4, "unit": "kg" }`
- `GET /api/weight/recent?limit=14`
- `POST /api/weight/undo-last`
- `GET /api/water/today`
- `POST /api/water/event` — body: `{ "deltaLiters": 0.25 }`
- `GET /api/water/recent?limit=20`
- `POST /api/water/undo-last`
- `GET /api/charts/daily?days=90&unit=lb`
