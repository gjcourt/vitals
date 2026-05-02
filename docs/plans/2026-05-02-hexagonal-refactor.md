---
title: "Hexagonal architecture migration"
status: "In progress"
created: "2026-05-02"
updated: "2026-05-02"
updated_by: "george"
tags: ["architecture", "hex", "refactor"]
---

# Hexagonal architecture migration

## Current layout

```
internal/
  domain/     — entity types (Water, Weight, Auth); repository interfaces
                (WaterRepository, WeightRepository, etc.) embedded here
  app/        — WaterService, WeightService, ChartsService, AuthService
  adapter/    — singular; http, memory, postgres sub-packages
```

The app layer is in good shape. The main gaps are: no formal `ports/` package
(interfaces are in `domain/`), the `adapter/` directory is singular, and the
HTTP adapter imports `app/` directly (no inbound port interface).

## Migration steps

1. **Extract outbound ports to `internal/ports/outbound/`** — move
   `WaterRepository`, `WeightRepository`, `AuthRepository`, etc. from `domain/`
   to `ports/outbound/`. Keep entity types in `domain/`. Update imports. One PR.

2. **Define inbound port interfaces in `internal/ports/inbound/`** — create
   interfaces matching the public API of each app service. One PR.

3. **Thread inbound ports through the HTTP adapter** — update `adapter/http` to
   depend on `ports/inbound/` interfaces rather than concrete `app/` types.
   This unblocks the `adapters-no-app` depguard rule. One PR.

4. **Rename `adapter/` → `adapters/`** (plural). One PR.

5. **Add function-field fakes** — add fakes for each outbound port, wire into
   `ServerDeps`. Update app-layer tests to use `testdoubles.NewServerDeps()`.

6. **Activate remaining depguard rules** — add `adapters-no-app` once step 3
   is done.

## Depguard notes

Bootstrap rules active: `domain-no-adapters`, `app-no-adapters`.

Pending rule (blocked): `adapters-no-app` — `adapter/http` currently imports
`vitals/internal/app` directly. Unblocked after step 3.
