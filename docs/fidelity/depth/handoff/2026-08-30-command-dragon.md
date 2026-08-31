# Depth handoff — 2026-08-30 — `dragon`

## Frontier and queue position

- Started from clean `main` after `git pull --ff-only`, commit `08f4149ee`.
- Read `docs/fidelity/DEPTH_TESTING.md` and the newest handoff,
  `2026-08-30-command-doh.md`.
- `make fidelity-depth` after this slice: 1,603 total; 1,548
  proven/delegated; 14 blocked; 41 excluded; actionable 1,548/1,562 (99.1%).
- The source-order next command after the already-proven `doh` row was
  `dragon`, registered at `src/interpreter.c:422`. This slice is complete;
  the next unclaimed command is `drink` only if its existing manifest still
  leaves a new case, otherwise continue at the next source-order gap.

## C call path and branch inventory

`src/interpreter.c:422` registers `dragon` with `POS_FIGHTING` and
`do_dragon_kick`. The handler is `src/act.offensive.c:636-690`:

1. `GET_SKILL` is checked before parsing or target lookup.
2. `one_argument` parses the first non-fill-word token; a room target is
   selected, then `FIGHTING(ch)` is the fallback, otherwise `Kick who?`.
3. Self-target returns `Aren't we funny today...`; mounted callers return
   `Dismount first!`; fewer than ten move points return
   `You're too exhausted!`; otherwise ten move points are spent.
4. The C percentage is `((5 - (GET_AC(vict) / 10)) << 1) + number(1, 101)`.
   Miss calls `damage(ch, vict, 0, SKILL_DRAGON_KICK)`; hit calls
   `damage(ch, vict, GET_LEVEL(ch)*1.5, SKILL_DRAGON_KICK)`, then
   `improve_skill`. Both arms finish with `WAIT_STATE(ch, PULSE_VIOLENCE+2)`.
5. `fight.c:1314-1718` owns the shared damage gates; `fight.c:1023-1092`
   selects the numbered skill-message record. The C value is 188, and the
   exact record is `lib/misc/messages:1267-1281`.

## RED, confirmed fixes, and GREEN proof

The first clean-main vehicle was RED: Go emitted the wrong target refusal,
checked the skill before the intended C target path because the wizard vehicle
stored `dragon kick` under the wrong Go key, and used invented literal combat
messages instead of `damage()`/`skill_message`. It also failed to enroll the
zero-damage miss in combat. The C-first corrections were:

- parse the Go command through `game.OneArgument` and emit `Kick who?`;
- map the C skillset display name `dragon kick` to the Go key
  `SkillDragonKick` (and add the C-supported `maxhit` wizard field needed to
  isolate the wait vehicle);
- correct the C attack/message number from the erroneous 222 to 188;
- route both result arms through set-188 `SkillMessage`, set `StartCombat`,
  carry `DamageSkill`, defer `improve_skill`, and map dragon damage to attack
  type 188 for the shared death path.

Proof vehicles:

- `cmd/dp-oracle-diff/scenarios/combat-transform-reject.txt` — no-skill gate;
- `cmd/dp-oracle-diff/scenarios/dragon-depth.txt` — no argument, missing
  target, fill-word/self parsing, skill-1 failure, and wait/combat aftermath;
- `cmd/dp-oracle-diff/scenarios/dragon-success-depth.txt` — success damage,
  set-188 message selection, high-HP wait isolation, and combat aftermath.

Both vehicles were GREEN for seeds 1, 2, 3, 5, and 8. Seed 3 was run with
`--show-oracle` and reached the success arm. Focused unit proofs are in
`TestDoDragonKick_WaitStateAlwaysThree`,
`TestDoDragonKick_BlocksInsufficientMove`, and
`TestDoDragonKick_MissDrawOrder`. The durable case inventory is
`docs/fidelity/depth/dragon.tsv` (13 handler cases plus the delegated shared
damage case).

## Gates

On `glm/spec-dragon`, after the oracle matrix:

- `make fidelity-depth` — PASS
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS
- `golangci-lint run ./...` — PASS, 0 issues
- `gofumpt -l .` — clean

This slice follows R1/R2/R3/R4 and R5e: C bytes and command surface remain
authoritative; the shared RNG/message order is proven; no C or oracle files
were edited; and the actual registered handler/callee path was audited before
the Go changes. The R5c shared damage behavior is delegated to the existing
`combat-entry` matrix rather than duplicated.
