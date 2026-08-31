# Depth handoff — 2026-08-31 — `groinrip`

## Frontier and queue position

- Started from clean `main` at `9d0a36212` after the merged `groan` slice,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-groan.md`.
- The starting frontier was 1,968 total, with 1,911 proven/delegated, 16
  blocked, and 41 excluded. The `groinrip` manifest adds 15
  proven/delegated cases and one explicit exclusion. The post-slice frontier
  is 1,984 total, with 1,926 proven/delegated, 16 blocked, and 42 excluded;
  actionable completion is 1,926/1,942 (99.2%).
- The blocked special-procedure row `objmagic.sleep-entry-gates` remains the
  next queue phase only after the special-procedure inventory is exhausted;
  this command slice did not alter it. The source-order command sweep now
  reaches `grope` at `src/interpreter.c:479`, immediately after this slice.

## C call path and branch inventory

`src/interpreter.c:478` registers `groinrip` as `POS_FIGHTING`, minimum level
0, routed to `do_groinrip`. The actual handler in `src/new_cmds.c:2565-2658`
was traced before changing Go:

- The descriptorless, non-subcommand return at `new_cmds.c:2571-2572` is not
  reachable from the registered player-command surface and emits no bytes.
- `ROOM_PEACEFUL` is checked first and emits the violence refusal.
- The learned-skill gate emits its C-specific `\n\r` terminator and returns
  for the normal command subcommand.
- The mounted gate precedes `one_argument`, target lookup, self, shopkeeper,
  immortal-victim, sex, and RNG branches.
- `one_argument` consumes only the first target word. A failed lookup uses the
  existing `FIGHTING(ch)` pointer; with neither target it emits `Groinrip
  who?`. A resolved self target emits the masochism refusal.
- Shopkeepers are refused before victim-level and sex checks. A non-NPC target
  at or above `LEVEL_IMMORT` receives two actor lines and leaves the attacker
  sitting. Non-male victims receive the exact no-groin refusal.
- The normal path consumes `number(1,121)`. A sleeping victim or an attacker
  above `LEVEL_IMMORT` forces percent to zero; success calls
  `damage(ch,victim,GET_LEVEL(ch),SKILL_GROINRIP)`, sends a victim-as-actor
  `TO_ROOM` act with the authored CRLF split, consumes `number(0,10)` for the
  optional vnum-21 puke, then improves the skill. Both success and failure
  end with `WAIT_STATE(ch, PULSE_VIOLENCE*2)`.
- `fight.c:1023-1092` selects the numbered attack-type 174 message after the
  damage state transition. `lib/misc/messages:1237-1249` supplies the die,
  miss, hit, and god message triples; no Go literal combat text was invented.

## Coverage and RED-to-GREEN evidence

The C-first vehicles use authoritative room/mob/player fixtures and never edit
`src/` or `darkpawns-c-oracle/`:

- `groinrip-depth` initially RED on main for self-target output, trailing-word
  parsing, and the post-command combat wait. After the fix it was GREEN at
  seeds 1, 2, 3, 5, and 8.
- `groinrip-outcome-depth` proves the numbered hit message, target-as-actor
  room audience, combat pulse, and wait behavior. It was GREEN at seeds 1, 2,
  3, 5, and 8.
- Separate GREEN vehicles prove the mounted gate, no-skill gate, authored
  shopkeeper gate, non-male target gate, immortal-victim rejection and sitting
  state, sleeping-target forced success, and the no-argument FIGHTING-pointer
  fallback. The fallback required resolving Go's name-backed fighting state
  through the canonical pointer-preserving helper rather than keyword lookup.
- Focused unit tests in `pkg/game/groinrip_depth_test.go` pin set 174,
  level damage, in-damage routing, the two-pulse wait, post-room improvement,
  the room line break, the puke request, draw consumption, and direct gate
  outcomes. `pkg/session/groinrip_test.go` pins the registered C entry gate.

The implementation preserves the shared damage boundary in
`DoGroinripDamage`, adds the C 174 mapping, performs vnum-21 puke placement
after the room act with timer 2, and defers improvement until after that room
act. The shared peaceful/newbie/shopkeeper/follower/death damage matrix stays
delegated to `combat-entry` rather than duplicated.

This follows R1/R2/R3/R4/R5e: player-facing bytes and the command registration
remain authoritative, draw/order and wait state are checked, no C behavior is
invented, and the actual handler call path was verified. The shared damage
delegation follows R5b/R5c so the fix is audited at the common boundary rather
than copied into another skill family.

## Changes, gates, and integration

- Added the C-first scenarios and `docs/fidelity/depth/groinrip.tsv` with 16
  explicit rows.
- Added the Go implementation and focused tests in commit `0bb60b089`.
- Local gates passed on `glm/depth-groinrip`: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`.
- PR #915 (`glm/depth-groinrip`) passed hosted `test`, `lint`, and `security`
  checks and was merged. No workflow retry was required.

The next session must return to clean `main`, pull, rerun `make fidelity-depth`,
reread this handoff, and take the next un-manifested command-table family in
order: `grope` at `src/interpreter.c:479`.
