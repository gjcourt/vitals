---
title: "Replace PostgreSQL with SQLite"
status: "In progress"
created: "2026-09-13"
updated: "2026-09-13"
updated_by: "george"
---

# Replace PostgreSQL with SQLite

## Why

The production database is **empty**. Measured 2026-09-12 against
`vitals-db-production-cnpg-v1-5`:

```text
public schema tables   0
database size          7475 kB   (an empty Postgres baseline)
app connections        0         (only postgres->postgres, not the app)
app pod                Running 73 days, serving /api/health and nothing else
provisioned            3 x CNPG replicas, 3 x 1Gi iSCSI PVCs
```

Three PostgreSQL replicas and three iSCSI volumes are carrying **zero rows**
for a single-user body-metrics app.

That is not just wasted capacity. Those three volumes are in the blast radius of
the cluster's most common failure — read-only iSCSI remounts — and have been
recovered by hand repeatedly, most recently on 2026-09-09. **Storage that holds
nothing is still storage that breaks.**

## This is a revert, not a migration

The app **was** SQLite. `modernc.org/sqlite` (pure Go, no CGO) was the original
implementation in commit `c988243`; it moved to PostgreSQL on 2026-02-20 in
`aee0d14`. The old adapter is in git history as a working reference for exactly
the pieces that need rewriting — placeholders, timestamp handling, and pool size.

## What makes this easy

- **No ORM.** `database/sql` + `github.com/lib/pq`, hand-written SQL.
- **No PostgreSQL-only features anywhere.** Zero occurrences of `jsonb`, arrays,
  `uuid`, CTEs, window functions, full-text search, `LISTEN`/`NOTIFY`, advisory
  locks, stored procedures or triggers.
- **5 tables**, 4–6 columns each.
- **Zero tests touch the Postgres adapter**, so nothing has to be rewritten.
  (That is also a coverage gap — see below.)
- **`replicas: 1`**, so SQLite's single-writer model costs nothing.
- **No data to migrate.** The database is empty.

## What actually needs changing

| Construct | Count | Change |
| :--- | ---: | :--- |
| `$N` placeholders | 36 | `?` |
| `BIGSERIAL` | 3 | `INTEGER PRIMARY KEY AUTOINCREMENT` |
| `TIMESTAMPTZ` | 5 | **`DATETIME`, not `TEXT`** — see below |
| `RETURNING` | 3 | none — SQLite supports it since 3.35 |
| `ADD COLUMN IF NOT EXISTS` | — | guard via `PRAGMA table_info` |

## ⚠️ The one real trap, found during implementation

The plan originally said to store timestamps as `TEXT`. **That is wrong and it
fails at runtime, not at compile time.**

`modernc.org/sqlite` decides whether to return a `time.Time` or a raw `string`
based on the **declared column type**. With `TEXT`, every scan into a
`time.Time` fails:

```text
sql: Scan error on column index 2, name "created_at":
unsupported Scan, storing driver.Value type string into type *time.Time
```

Measured 2026-09-13 against the driver:

| Declared type | Round-trips to `time.Time`? |
| :--- | :--- |
| `TEXT` | ❌ scan error |
| `DATETIME` | ✅ |
| `TIMESTAMP` | ✅ |

SQLite stores the value as text either way — only the affinity differs — so this
costs nothing and avoids editing every scan site in the three repo files.

**This was caught by a test written specifically because it was the likeliest
place for the port to go quietly wrong.** Without it the app would have compiled,
started, passed a health check, and failed on the first real read.

## Steps

1. **New `internal/adapter/sqlite/`** implementing the same domain interfaces as
   the Postgres adapter. Hexagonal layout means `internal/domain/` and
   `internal/app/` are untouched.
2. **Rewrite `migrate()`** — 13 DDL statements — for SQLite dialect. Add
   `PRAGMA journal_mode = WAL` and `PRAGMA foreign_keys = ON`, both of which the
   original SQLite code set and Postgres did not need.
3. **Pool size `SetMaxOpenConns(1)`.** SQLite is single-writer; the pre-migration
   code already did this.
4. **`cmd/vitals/main.go`** — select the adapter on `SQLITE_PATH` instead of
   `POSTGRES_URL`.
5. **Write adapter tests.** None exist for Postgres either, so this is net-new
   coverage rather than a port. Use a tempfile database.
6. **Remove the Postgres adapter and `lib/pq`.**
7. **Homelab side** (separate repo, separate PR): add a PVC and writable mount,
   retire the CNPG `Cluster` / `ScheduledBackup` / `ObjectStore`.

## Decision: remove PostgreSQL rather than keep both

Keeping a second storage adapter means maintaining a code path nobody exercises.
The empty database is evidence of which one is actually in use. `internal/adapter/memory/`
remains for dev and tests, so there is still a second implementation keeping the
interfaces honest.

## ⚠️ Risks

**Backup parity regresses, and this is the real one.** Production currently has
CNPG WAL archiving and a `ScheduledBackup` to S3 via the Barman Cloud plugin.
A SQLite file has none of that. It is easy to complete this migration and
silently end up with **no backups at all**.

Mitigation is required before this is called done — `litestream` for continuous
replication, or a `VACUUM INTO` cron writing to the same object store. Academic
while the database is empty; not academic the first time a weight is logged.

**`readOnlyRootFilesystem: true`** is set on the container. SQLite needs a
writable path, so this needs an explicit writable volume mount rather than
relaxing the flag.

**A PVC is still a PVC.** This trades three iSCSI volumes for one. It reduces the
blast radius; it does not leave it.

## ⚠️ Stale documentation found

`docs/reference/2026-05-02-database.md` describes UUID primary keys. The code
uses `BIGINT`. That doc has been wrong since at least the Postgres migration and
should be corrected regardless of the outcome here.

## Done when

1. `make all` passes (lint, test, build).
2. The SQLite adapter has tests; the Postgres adapter and `lib/pq` are gone.
3. The app runs against a SQLite file in the cluster and serves real reads/writes
   — verified by logging a weight and reading it back, not by the pod being
   `Running`.
4. A backup mechanism exists and has been restored from once.
5. The CNPG cluster and its three PVCs are deleted.
