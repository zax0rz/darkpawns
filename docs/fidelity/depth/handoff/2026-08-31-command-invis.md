# Depth-fidelity handoff: `invis`

Date: 2026-08-31

## Queue position and result

This session consumed the next unclaimed interpreter-table command after
`insult`: `invis`, registered at `src/interpreter.c:519` as `do_invis` at
`POS_DEAD` with `LVL_IMMORT`. The special-procedure inventory remains
exhausted, and the single blocked `objmagic.sleep-entry-gates` row remains
deferred; neither was repicked.

The pre-slice frontier was 2,306 total cases, with 2,246 proven/delegated,
16 blocked, and 44 excluded. After adding the 13 `invis` rows, the frontier
is 2,319 total, with 2,259 proven/delegated, 16 blocked, and 44 excluded:
2,259/2,275 actionable cases, or 99.3%.

Feature PR #977 (`fix: depth-prove invis C fidelity`) passed hosted `test`,
`lint`, and `security` checks; build-and-push and deploy were skipped by the
PR workflow. It merged to main as `7e777b678` (feature commit `0bae2da44`).
No merge was performed while a required check was pending.

The next unmanifested command in source registration order is `junk` at
`src/interpreter.c:521`.

## C call path and reachable branches

The authoritative command path is
`src/interpreter.c:519` → `src/act.wizard.c:1663-1688`
(`ACMD(do_invis)`) → `perform_immort_vis` or `perform_immort_invis` at
`src/act.wizard.c:1621-1659`. The shared visible path also reaches
`appear()` in `src/fight.c:108-121`; the prompt exposes
`GET_INVIS_LEV` in `src/comm.c:1062-1065`.

The registered command is rejected before the handler for actors below
`LVL_IMMORT` or below `POS_DEAD`. The C handler's reachable branches are:

- no argument with an old invisibility level above zero: call
  `perform_immort_vis`;
- no argument with an old level of zero: call `perform_immort_invis` at the
  actor's level;
- a first token parsed with C `atoi` above the actor's level: emit
  `You can't go invisible above your own level.` and leave state unchanged;
- a first token parsed below one, including zero, a negative number, or a
  nonnumeric token that `atoi` converts to zero: call `perform_immort_vis`;
- a positive permitted first token: call `perform_immort_invis` at that
  level.

`perform_immort_invis` captures the old level, then sends the gone message to
neighbours whose levels satisfy `target >= old && target < new`, and the
standing-beside-you message to neighbours satisfying
`target < old && target >= new`. These are strict boundary comparisons. It
then stores the new level and sends the actor the exact `Your invisibility
level is %d.` acknowledgement. `perform_immort_vis` has the exact
already-fully-visible early return; otherwise it zeroes the level, calls
`appear`, and sends the fully-visible acknowledgement. `appear` removes the
invisible spell and hide/invisible affect bits, then sends the immortal
strange-presence room line. The C `iN >` prompt state is part of the actor
transcript.

The NPC handler text is not reachable from this command surface because the
interpreter's immortal command gate rejects NPC command execution; it was
recorded during the C path audit rather than invented into the proof vehicle.

## RED diagnosis and confirmed fix

On main-equivalent Go code, the primary oracle vehicle was RED: Go toggled the
`PLR_INVISIBLE` flag and emitted `You are now invisible.` or
`You are now visible.`, while C stored a numeric invisibility level, emitted
the exact level/visible messages, notified threshold-crossing peers, removed
the shared affect state through `appear`, and exposed `iN >` in the prompt.
Go also lacked the C positive-level, above-own-level, zero, negative, and
nonnumeric-`atoi` branches. The exact C output and missing audiences were
confirmed before changing the port (R1/R4/R5e).

The fix adds runtime `InvisLevel` state with typed accessors, ports the C
handler and shared `appear` path, makes visibility honour the C wizinvis
threshold before the immortal shortcut, and routes the session command to
the world handler. The prompt and quit leave-broadcast condition now use the
C invisibility level. The runtime field is intentionally not added to the
save format because that format must remain faithful until a corresponding C
representation is established. The existing Go-only `vis` command remains
outside this C `invis` slice.

No `src/` or `darkpawns-c-oracle/` file was edited. Shared visibility ownership
was audited under R5b/R5c, and no behavior absent from C was added under R4.

## Durable proof

The manifest `docs/fidelity/depth/invis.tsv` contains 13 rows covering the
entry gate, toggle-on/off, accepted numeric levels, above-own-level rejection,
zero/negative/nonnumeric `atoi` behavior, below-threshold crossing
notifications in both directions, exact-level and above-level strict-boundary
cases, the shared `appear` audience, and prompt state.

The durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/invis-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/invis-threshold-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/invis-entry-gate-depth.txt`.

`invis-depth` and `invis-threshold-depth` were each run with
`--show-oracle --seed 1`, then at seeds 2, 3, 5, and 8; every intended C block
was reached and every run reported no normalized divergence. The entry-gate
vehicle is GREEN and confirms the C `Huh?!?` rejection on the main-equivalent
gate. Focused tests cover registration/gate parity, prompt state, level and
threshold audiences, and the immortal visibility ordering. The scenarios use
the registered immortal mob vehicle and disposable peer setup; no C fixture
or checked-in world data was changed.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 deterministic transcript parity across seeds, R4 absence of invented
parse/visibility behavior, and R5e verification of the actual C call path.
The shared `appear`/visibility audit preserves R5b/R5c ownership boundaries.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `test -z "$(gofumpt -l .)"` and `git diff --check`.

After merging, clean main ran `make fidelity-depth` successfully at the
2,319-case frontier. Hosted checks for PR #977 started normally; `lint` and
`security` passed first, followed by `test`, and no workflow retry was needed.

The next session must start from clean main, pull, reconfirm the frontier,
read the depth-testing guide and this newest handoff, then map and prove
`junk` in table order. Continue to leave one dated handoff per session.
