# operations/

Runbooks, smoke tests, and on-call / day-to-day operating procedures.

**Put here:**
- How to run, deploy, restart, and troubleshoot the service.
- Step-by-step recovery procedures for known failure modes.
- Local development setup and smoke tests.

**Do not put here:**
- Configuration reference / API specs — `reference/`.
- Architecture explanation — `architecture/`.

**Naming convention:** `<yyyy-mm-dd>-<topic>.md`
Examples: `2026-05-02-running-locally.md`, `2026-09-01-postgres-backup-recovery.md`.

**Allowed `status:` values:** `Stable`, `Superseded`.

Stale runbooks are dangerous. When a procedure changes, update the doc in the same PR.
