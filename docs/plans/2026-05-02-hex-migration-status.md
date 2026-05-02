---
title: "Hex migration status"
status: "In progress"
created: "2026-05-02"
updated: "2026-05-02"
updated_by: "george"
tags: ["architecture", "hex", "tracking"]
---

# Hex migration status

## Depguard rules

| Rule | Status | Notes |
|---|---|---|
| `domain-no-adapters` | Active ✓ | Domain is clean |
| `app-no-adapters` | Active ✓ | App layer depends only on domain interfaces |
| `adapters-no-app` | Pending ✗ | `adapter/http` imports `app/` directly — fix in step 3 |
| `adapters-no-cross-import` | Pending ✗ | Low priority; enable after step 4 |

## Migration checklist

- [ ] Step 1 — extract repository interfaces → `internal/ports/outbound/`
- [ ] Step 2 — define `ports/inbound/` interfaces for each app service
- [ ] Step 3 — thread inbound ports through `adapter/http` (unblocks `adapters-no-app`)
- [ ] Step 4 — rename `adapter/` → `adapters/`
- [ ] Step 5 — add fakes to `testdoubles/`, wire `ServerDeps`
- [ ] Step 6 — activate `adapters-no-app` depguard rule
