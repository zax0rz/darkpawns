# Dashboard and logs verification

B+C extends foundation PR #2195. Data audit: [METRICS-AUDIT.md](METRICS-AUDIT.md).
Operator documentation: [Dashboard and logs](../website-astro/src/content/docs/server/dashboard-and-logs.md).

## Checks

- `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`: pass individually.
- `go test -race ./pkg/telemetry ./pkg/admin -run 'TestHistory|TestDashboardEndpoint|TestLogbook|TestObservability'`: pass.
- `make deadcode-ratchet`: pass, 52 baselined, 0 new.
- Admin `npm run build` and `npm run lint`: pass. Vite reports the existing large-chunk warning; operational Recharts now ships in the dashboard bundle.
- World asset generation completed. `make site-check generate-caddy-redirects`: pass after installing the locked site dependencies. No deployment performed.
- New Huma endpoint tests exercise recorded totals/gaps, absent recording, invalid ranges/dates, authentication/role enforcement, filtered raw/CSV content, escaped commas, spreadsheet formulas, and actual-caller mudlog visibility.
- Recorder tests exercise counters, sample/peak aggregation, retention, restart continuity, actual executable changes, shutdown persistence and visible write failures.

## Fidelity

Observer hooks are atomic increments only. No player output, random draw,
command decision or player save format changes (R1/R3/R4). The online gauge now
samples world characters rather than all transport sessions.

Targeted census, clean 3/3:
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-27/sol-admin-observer-hooks-verified/`

Scenarios: `character-creation-name-retry` (creation acceptance),
`entry-password-retry-success` (returning authentication),
`raw-kill-protection-backfire` (existing death observer). The retained manifest
freezes the dirty implementation over `4fb40c8f0`. An earlier registration was
refused because schema generation changed the checkout during snapshot; it was
not a game comparison. No final full/combined census; Claude runs the batch gate.

## Browser evidence

`admin-ui/scripts/observability-smoke.mjs` uses the opt-in `adminpreview` test
server and a disposable SQLite world. Its history is explicitly synthetic.
Production charts have no synthetic fallback. The script resets the fixture
router before navigation to avoid the production 5 requests/second limiter
making a rapid multi-page test look signed out. The limiter remains unchanged.

Retained evidence:
`/home/zach/Archives/darkpawns/admin-ui/2026-10-27/dashboard-logs/`

Desktop: dashboard, seven-day history, activity/deaths, release, historical
errors, server log, filtered errors, mudlog, empty filter and request failure.
Mobile: dashboard, activity, server/mudlog logs, including lower-page captures.
First-boot captures and real Chrome screenshots additionally verify unknown
history and a connected level-31 filtered SSE view. Downloads verify severity
plus a UTC time filter. The original Operations tail remains available.

The finish reviewer scored all four material fixes resolved: adjacent coverage,
matching historical drill-downs, unavailable-vs-empty states, and product/design
documentation. Its final disposition was **ship** at that fix-list scope. The
mechanical detector reported no findings. The documenter preserved system tokens.

## Operational limits

History retains seven days, persists once a minute plus context shutdown, and
can lose the latest minute on abrupt termination. Counts are assigned to the
10-second observation interval ending at a sample. Hour/day aggregation can
hide gaps within partially sampled intervals, which coverage explicitly states.
Deaths include players and mobs; login counts include authenticated reconnects.

Log downloads cover the filtered retained ring, not an arbitrary on-disk file.
CSV neutralizes potential spreadsheet formulas; raw text stays unchanged.
The same actual-caller ceiling applies to mudlog list, live stream and exports;
negative-level file-only events and privileged catalog keys are excluded.
Native EventSource cannot send the mandatory CSRF header, so the console reads
the existing SSE endpoint through fetch with that header and cancellation/retry.

Last update means a changed running executable was observed at startup. First
observation cannot establish a previous deployment time and remains unknown.
