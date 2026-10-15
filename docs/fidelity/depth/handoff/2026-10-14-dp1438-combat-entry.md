# DP-1438 — combat entry is C's hit() → damage(), not "enrol, then swing"

Date: 2026-10-14 · Branch: `glm/dp-1438-combat-entry` · Rules: R1, R3b, R3d, R5c, R5e, R5g, R5h

## The finding (both halves)

C's first swing goes `hit()` → `damage()`:

- `hit()` rolls `diceroll = number(1, 20)` (`src/fight.c:1815`) and never calls
  `set_fighting`.
- `damage()` refuses before anything is enrolled: the peaceful-room arm
  `!IS_OUTLAW(victim) && FIGHTING(victim) != ch` (`src/fight.c:1335-1342`), the
  newbie/level gates (`:1344-1357`), and shopkeeper protection (`:1359-1367`).
- Only a damage that isn't refused reaches `set_fighting(ch, victim)` at
  `src/fight.c:1408` (unstunned, not already fighting — the jail guard ran
  before it, the charm/switcheroo draws after it) and `set_fighting(victim, ch)`
  at `:1445`.

Go had two shapes of the same bug:

1. **Commands skipped the roll.** `CmdHit` and `cmdAssist` preflighted
   `DamageRefused` and returned before the initial attack, so C's
   `number(1,20)` was never drawn — the seed-1 first mismatches behind the
   corpus-2 DRAW rows (streams 4106, 4087, 4081, 4152; reports retained under
   `~/Archives/darkpawns/oracle-runs/2026-10-14/luna-living-world-dp1364-corpus2-diagnostics/`).
2. **Mobiles enrolled first.** Callers used eager `StartCombat`, so by the time
   the pulse reached the gate, `FIGHTING(victim) == ch` already held and the
   peaceful exemption applied — in room 8162 (flags 28) the courier dies in Go
   and lives in C. This half's call sites convert with the mobact train (below).

## The fix

`CombatEngine.PerformDeferredInitialAttack` (pkg/combat/engine.go) runs one
`performOneHit` with both enrolments deferred to the damage point:

1. consumes `hit()`'s draws (the existing per-round order — roll, then gate),
2. runs `damage()`'s gates against the pre-enrolment state,
3. enrols the attacker where C `damage()` calls `set_fighting` — for a player
   attacker before the NPC-only redirect call, for an NPC attacker inside it
   (set_fighting is not NPC-gated; the jail/charm/switcheroo redirects are) —
   and enrols the defender at the existing damage-point block.

A refused swing draws, prints C's refusal bytes, and leaves nobody fighting.

### Converted in this train (with C citations)

| Caller | Go site | C cite |
|---|---|---|
| `do_hit` (hit/murder/kill aliases) | `pkg/session/combat_cmds.go` | `act.offensive.c:126-127` |
| `do_assist` | `pkg/session/cmd_combat_basic.go` | `act.offensive.c:95` |
| cutthroat failed arm (skill tail `InitialAttack`) | `pkg/command/skill_commands.go` | `new_cmds.c:652` |
| violent cast fumble retaliation | `pkg/session/cast_cmds.go` | `spell_parser.c:1101-1106` |
| caught-steal retaliation | `pkg/game/act_informative.go` | `act.other.c:528-529` (via `kender_steal`'s `do_steal` subcmd delegation, `spec_procs2.c:626`) |

### Audit: leave-alones (R5c, verified against C this session)

| Caller | Go site | C cite | Why left |
|---|---|---|---|
| `do_rescue` success arm | `pkg/game/skill_combat.go:582` | `act.offensive.c:575-576` | pure `set_fighting`, no `hit()` |
| backstabber special | `pkg/game/spec_fighter.go` | `fight.c:1400-1408` (in-file citation) | draws happen in the caller; gate then post-gate enrolment already match C |
| shoot retaliation | `pkg/command/shoot.go` | `act.offensive.c:955` | already `PerformRangedRetaliation` (deferred) |
| aware-backstab retaliation | `pkg/command/skill_commands.go` | `act.offensive.c:217` | already `PerformUnenrolledInitialAttack` |
| mobile specials via the mob seam | `pkg/game/spec_procs.go:822` | per-special | already `StartCombatFromMob` (defender deferred) |

### Follow-up class (next train, blocked on overlaps)

- **mobact arms** — `pkg/game/mobact.go` (memory/helper/aggr24/aggressive ×6):
  C calls `hit(ch, vict)` at `src/mobact.c:223, 227, 254, 279, 299, 320, 334`.
  Convert to `PerformDeferredInitialAttack` **after** DeepSeek's #1974
  (DP-1435 mobact rewrite) and Luna's #1969 (DP-1364 mobile iteration) merge;
  both were open when this train branched. The engine entry is ready for them.
- **mobprogs (Lua `action`/`hit`)** — `pkg/game/mobprogs.go:535,544`; C
  `mobprog.c:85, 225, 427`.
- **mob-rescue special** — `pkg/game/spec_procs2.go:592` enrols like C's
  `set_fighting` pair (leave-alone), but its ally swing (`mobHit`) should ride
  the deferred entry; C `act.offensive.c:577-578` (`if (subcmd) hit(vict,
  tmp_ch)`).
- **other specials** — `spec_procs3.go:1725`, `spec_procs_missing.go:38`,
  `spec_procs.go:824`'s attacker side enrols eagerly (unobservable at the gate,
  but a refused special swing leaves the attacker enrolled, unlike C
  `fight.c:1318-1368`).
- **hunter path** — C `graph.c:312, 344` `hit(ch, HUNTING(ch))`; no direct Go
  `StartCombat` site found (rides mobact/ai — audit again with that train).
- **spell damage enrolment timing** — `pkg/spells/damage_spells.go:328`; C
  enrols spell damage at `fight.c:1408` after the gates; audit the
  `DoSpellDamage` stub seam against that point.
- **`PerformUnenrolledInitialAttack` refusal arm** — `engine.go:379` enrols the
  attacker even when the damage was refused; C enrols nobody
  (`fight.c:1318-1368`). Its proven aware-backstab cases are green because the
  refusal arm is unreachable there; fold into the deferred entry with the
  mobact train and re-run the backstab corpus.

## Proof

- Unit (real dispatch, dprng stream probes, seeds pinned to natural-1 misses):
  `TestDeferredInitialAttackEnrollmentAndProtections`,
  `TestDeferredInitialAttackKeepsExistingTarget`,
  `TestCmdHitRefusedSwingStillDraws`, `TestCmdAssistRefusedSwingStillDraws`,
  `TestCmdHitAllowedSwingDrawsAndEnrols`,
  `TestCutthroatTailRefusedInPeacefulRoom`,
  `TestCastFumbleRetaliationDrawsAndEnrols`,
  `TestKenderStealCaughtRetaliationDrawsAndEnrols`,
  `TestKenderStealCaughtRetaliationRefusedInPeacefulRoom`.
- R5h reverts verified: restoring the preflight in `CmdHit`/`cmdAssist` fails
  both refused-swing tests (stream unmoved); restoring the eager/plain shapes
  in the cutthroat tail, cast fumble and steal retaliation fails their tests
  (cutthroat enrolled both sides; cast/steal drew nothing).
- Draw parity: `census.sh start --draw-parity --scenarios
  combat-entry-newbie-gate,combat-entry-peaceful-wait,combat-entry-shopkeeper,combat-entry-charm-relations`
  must report no DRAW rows (run recorded with the census section below).
- The four scenarios stay byte-green: the fix changes draw order and enrolment
  only; the refusal bytes were already faithful (DP-1045/DP-1327).
- **Peaceful mobile entry (room 8162)** stays open with the blocked depth row
  `mobact.peaceful-first-hit-enrolment` (DeepSeek's DP-1371 batch handoff,
  riding PR #1974): add the scenario from the withdrawn
  `mobact-aggr24-weaker-mobile-grin` template when the mobact arms convert —
  C shows the grin and a living courier; Go currently kills the courier.

## Manifest rows added/updated

- `combat-entry.tsv`: `hit.first-swing-draw-order`,
  `assist.first-swing-draw-order`,
  `set_fighting.first-swing-enrolment-point` (new);
  `assist.immediate-swing` note corrected (its draw-parity claim held only for
  landed swings).
- `cutthroat.tsv`: `cutthroat.entry-gate-order`.
- `cast.tsv`: `cast.fumble-retaliation-hit`.
- `other.tsv`: `steal.caught-retaliation-hit`.
