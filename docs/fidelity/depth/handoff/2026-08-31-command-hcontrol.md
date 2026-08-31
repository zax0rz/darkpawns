# Depth handoff — 2026-08-31 — `hcontrol`

## Frontier and queue position

- Started from clean `main` at `4791932e8` after the merged `handbook`
  slice, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-headbutt.md`.
- The starting frontier was 2,092 total, with 2,032 proven/delegated, 16
  blocked, and 44 excluded. The hcontrol manifest adds 25 proven cases,
  producing 2,117 total, 2,057 proven/delegated, 16 blocked, and 44 excluded;
  actionable completion is 2,057/2,073 (99.2%).
- `hcontrol` is registered at `src/interpreter.c:491` and reaches
  `do_hcontrol` in `src/house.c:579-598`. The five subcommands were audited
  in C source order: `build`, `destroy`, `pay`, `show`, and `key`. An exact
  interpreter-table sweep leaves `hiccup` at `src/interpreter.c:492` as the
  next unmanifested command family; the next session must return to clean
  `main`, pull, rerun the frontier check, reread this handoff, and begin
  `hiccup`.

## C call path and branch inventory

The C registration and handler were traced before changing Go:

- The row is `POS_DEAD`, level `LVL_GRGOD` (38), and dispatches to
  `do_hcontrol`. `half_chop()` separates the subcommand and remainder;
  `is_abbrev()` selects the first matching `build`, `destroy`, `pay`, `show`,
  or `key` branch, otherwise the handler emits `HCONTROL_FORMAT`.
- `build` covers the max-house gate, C `atoi()` vnum parsing, duplicate-house
  gate, direction lookup, exit existence, reverse two-way-door check, owner
  lookup, persistent room flags/control record, and the success message.
- `key` covers usage, missing house, C `atoi()` key parsing, object lookup,
  record mutation, and success. `pay` covers usage, missing house, timestamp
  mutation, and success. `destroy` covers usage, missing house, room-flag and
  control-record deletion, and success. `show` covers the empty table and
  defined-record listing, including C `asctime()` day-prefix bytes.
- The C source uses an overlapping `sprintf(buf, "%s...", buf, ...)` while
  appending the first listing row. The deployed oracle emits the row without
  the nominal two-line header; the Go listing follows those observed bytes.
- No descriptorless handler branch is reachable from this registered command
  surface, so no additional exclusion was invented.

## Confirmed divergences and fix

The complete vehicle was run against pre-fix `main` before the feature branch.
The first build branch deadlocked the Go telnet command: hcontrol held
`World.mu` while `Player.SendMessage` tried to acquire its read lock, so the
first divergent command emitted no Go bytes and prevented later commands from
running. After moving player replies and persistence outside the world writer
lock, the vehicle exposed the remaining byte/state divergences:

1. Go invented `Invalid house vnum.` / `Invalid key vnum.` branches instead of
   following C `atoi()` zero behavior.
2. The Go server had no live house player lookup wired, so valid owner/list
   branches returned `Player lookup not available.` and could not mutate the
   house record.
3. Go formatted house dates as `Jan 2 2006` and emitted the nominal listing
   header, while the C oracle emits the local `Mon Aug 31`-style prefix and,
   for the observed compiled C path, only the row.

Only these confirmed divergences were fixed. `RegisterHousePlayerLookup()` is
now wired to a snapshot of online world players, C-compatible numeric parsing
is local to hcontrol, list dates use the C day-prefix shape, and all stateful
hcontrol sends occur after releasing `World.mu`. No source or C-oracle file was
edited.

## Coverage proof

- Added `hcontrol-depth.txt` with the Implementor entry, usage and unknown
  subcommand gates, empty/defined/paid/cleared listings, build validation and
  lifecycle, key validation and success, payment, and destruction paths.
  The frozen-clock warmup includes `~dpclock pulse 20` so queued command timing
  cannot hide the registered path.
- Added focused `TestHcontrolRegistrationUsesCEntryGate` and 25 manifest rows
  in `docs/fidelity/depth/hcontrol.tsv`. The live scenario matched the C
  oracle at seeds `1, 2, 3, 5, 8`; one run used `--show-oracle` to verify the
  intended C blocks executed.

This follows R1/R2/R3/R4/R5e and R5c: C bytes and command registration remain
authoritative, C numeric and timestamp behavior is preserved, the shared
lookup/lock path is verified, no player-facing branches are invented, and the
full hcontrol behavior class was rechecked after the first failure.

## Changes, gates, and integration

- PR #933 (`glm/depth-hcontrol`, final feature commit `e79bf8889`) passed
  hosted `test`, `lint`, and `security` checks; the release-only build/deploy
  jobs were skipped as expected. It was merged only after every reported
  check was green; merged `main` is `3c61c2ebf`.
- Local gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  `gofumpt -l .`, and `git diff --check`.

The next session must begin from clean `main`, pull, run `make fidelity-depth`,
reread this handoff, and continue the interpreter-table sweep with `hiccup` at
`src/interpreter.c:492`.
