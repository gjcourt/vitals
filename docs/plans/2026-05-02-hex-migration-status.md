---
title: "Hex migration status"
status: "Complete"
created: "2026-05-02"
updated: "2026-05-02"
updated_by: "george"
tags: ["architecture", "hex", "tracking"]
---

# Hex migration status

## Depguard rules

| Rule | Status | Notes |
|---|---|---|
| `domain-no-other-internal` | Active ✓ | Domain is clean |
| `ports-no-impl` | Active ✓ | Ports only import domain |
| `app-no-adapters` | Active ✓ | App layer depends only on ports |
| `adapters-no-app` | Active ✓ | Adapters depend on inbound ports, not app directly |

## Migration checklist

- [x] Step 1 — extract repository interfaces → `internal/ports/outbound/`
- [x] Step 2 — define `ports/inbound/` interfaces for each app service
- [x] Step 3 — thread inbound ports through `adapters/http` (unblocked `adapters-no-app`)
- [x] Step 4 — rename `adapter/` → `adapters/`
- [x] Step 5 — add fakes to `testdoubles/`, wire `ServerDeps`
- [x] Step 6 — activate `adapters-no-app` depguard rule
