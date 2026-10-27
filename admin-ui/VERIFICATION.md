# Foundation verification

Base: `dac144ef2` (`origin/main` when this worktree was created).
Scope: plan A of `V017-ADMIN-PLAN.md`, 2026-10-27. Dashboard recording and live
charts remain plan B. The component preview uses explicitly synthetic data.

## Gates

Passed individually: `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, `make deadcode-ratchet`, `make fmt`, admin-ui build,
admin-ui ESLint, `make build-site` (including voice, Markdown, inventory, wordmark,
palette and Astro checks). The admin dependency audit reports zero vulnerabilities.
The design detector returned no findings. The linter emitted a shared-cache
warning referencing a removed unrelated worktree, then reported zero issues.

No oracle census: no game command, player output, draw order, world persistence,
or oracle source was changed (R1/R3/R4). Admin compatibility is covered by the
existing Huma/role/cookie/CSRF/OLC/player/file tests and browser smoke checks.

## Browser evidence

Before/after evidence is retained at:
`~/Archives/darkpawns/admin-ui/2026-10-27/admin-foundation/`.
The manifest records source revisions, captures, validation and SHA-256 hashes.
The source-owned smoke script fails on uncaught page errors, empty working areas
or error-boundary screens. It uses real API handlers with disposable SQLite,
notification configuration, world records and editable files.

Desktop, 1440×1000: Dashboard; Zones; zone detail/editor; Mobs; mob detail/editor;
Objects; object detail/editor; room detail/editor; Shops; shop editor; Players
(with a character opened); Notifications; Operations; Workshop; Builder guide;
Lua scripts; Server text; Terminal; Sign in; 404. All 24 routes passed.
Four narrow captures at 390×844: Dashboard, Players, Notifications, Mobs.
Additional proof: chart component at 1440×1000, dashboard at the connected browser's
2123×790 viewport, reset dialog, account menu, select menu, player dialog and error
toast. Terminal rendering is verified; this fixture does not run a game WebSocket.

Manual Chrome checks: saved a character stat and restored it, switched editor
and online-player tabs, toggled and discarded a notification switch draft, opened
and cancelled the reset confirmation (Cancel received focus), opened the account
and log-line menus, dismissed the player dialog with Escape, and observed the
server's failed-save response as a dismissible error toast. No outbound test
notifications or production actions were performed.

## Compatibility findings

The five supposedly unconverted HandleFunc routes already have Huma operations.
Their unsupported-method fallbacks remain. The generated client revealed these
existing mismatches, addressed in this foundation:

- Liveness used `/admin/health` and expected JSON. `/health` returns `OK` text.
- Finding update accepts either field; its input schema incorrectly required both.
- Arrays can be null, and online item records expose `slot`/`name`, rather than
  the frontend's guessed `wear_location`/`short_desc`.

These are explicit interface corrections; the work does not claim identical UI
behavior where the previous console was broken. Existing game behavior stays
unchanged. Native browser confirmations for unsaved-page guards remain; custom
inspection, reset, kick and reorder dialogs use the shared primitives.

## Decisions for Zach

Review this foundation before stacking B/C/E/F. B should inventory retained
metrics first, then add real histories and useful dashboard drill-downs. No merge,
deploy, or final combined release census was performed here.
