# Dark Pawns Admin product context

The console serves Dark Pawns operators and world builders. Operators need to
check server health and player activity first, then inspect evidence through
activity details, operations and filtered logs. Builders need visible world
records, permissions, validation and save boundaries.

The dashboard extends the existing console shell and its single Haunted
Paperback light theme. Source-owned copies of shadcn/ui components and Recharts
provide controls and charts within that system.

Operational charts show real recorded observations only. Unknown release/update
times, missing observations, partial coverage, storage failures and retention
limits must remain explicit. Telemetry retains seven days; server and mudlog
exports cover only the filtered retained ring, up to 1,000 entries. Synthetic
fixtures are development evidence and never substitute for operational history.

Observability supports the faithful game without changing its player-facing
bytes, random draws, commands or decisions (R1–R4). Telemetry stores character names and world metadata for operator activity
details, with no hosts, addresses, credentials or chat. Existing authorization and mudlog visibility still govern inspection
and exports.

Confirmed scope: B+C retained telemetry, charts, drilldowns and filtered raw/CSV
exports; health and player activity first, confirmed on 2026-10-27. Data definitions
and implementation limits are recorded in [METRICS-AUDIT.md](METRICS-AUDIT.md)
and the [operator guide](../website-astro/src/content/docs/server/dashboard-and-logs.md).
