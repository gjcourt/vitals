# architecture/

How the system is built **today** — the current shape of the code.

**Start here:** [`ARCHITECTURE.md`](ARCHITECTURE.md) — the full ports & adapters
reference: layer diagram, end-to-end request flow, the ports/adapters map, the
inward dependency rule, and the go-arch-lint boundary guard. The dated
[`2026-05-02-overview.md`](2026-05-02-overview.md) is the original high-level
summary.

**Put here:**
- System-overview docs that describe layers, packages, and dependency flow as they are right now.
- Diagrams and prose that explain the present architecture.

**Do not put here:**
- Proposals for future architecture — `design/`.
- Phased migration sequencing — `plans/`.
- API quirks — `reference/`.
- Runbooks — `operations/`.

**Naming convention:** `<yyyy-mm-dd>-<topic>.md`
Examples: `2026-05-02-overview.md`.

**Allowed `status:` values:** `Stable`, `Superseded`.

When the architecture changes materially, supersede the existing doc with a new one and set `superseded_by:` on the old one.
