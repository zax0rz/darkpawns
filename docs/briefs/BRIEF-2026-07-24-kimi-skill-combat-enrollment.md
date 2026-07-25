# BRIEF (kimi k3) — DP-1213: positive-damage skill hits must start engine combat + fix downed-mob stand-up

**Owner:** kimi-k3. **Gate:** unit tests now; Claude runs the `combat-trip-opener`
/ `combat-headbutt-opener` oracle gates after (they currently red only on this).
CI green.
**Read first:** `GLM.md` (operating manual — governs you) and
`docs/fidelity/RULEBOOK.md` (esp. **R1/R3a/R3b/R5e**). The C source is law (R5e).
**Git:** `git fetch origin` first. This work needs #469 AND #470 (the DP-1212
draw-order fix) in the base. **Confirm the base has #470:**
`git log --oneline origin/main | grep -c "defer skill improvement"` → if `1`,
branch off `origin/main`; if `0` (#470 not merged yet), branch off
**`origin/kimi/skill-draw-order`** (commit `0b0380db`, which already has #469+#470)
instead. Name the branch `kimi/skill-combat-enrollment`. Confirm your merge-base is
the chosen base's tip (no stale-main drift). Edit → commit → push → PR. Do NOT
merge. Sized S/M.
**Finding:** DP-1213 (root cause instrumented + confirmed — see the Linear issue).
**Cite:** Go `pkg/command/skill_commands.go` (`sendSkillResult`, the
`result.Damage <= 0` gate on `StartCombat`), `pkg/game/damage_stubs.go:75`
(`DoSpellDamage`), `pkg/combat/engine.go:436-461` (`processCombatPair` stand-up),
`:375` (`PerformRound`); C `src/fight.c` (`damage()`→`set_fighting`,
`perform_violence` :1977-1998). Rules R1/R3b.

---

## Bug 1 (primary) — positive-damage skill hits never enroll engine combat

Proven by instrumentation: after a successful **trip**/**headbutt**, the combat
engine's `combatOrder` is **empty every pulse** (`PerformRound` had nothing to
run), so no combat rounds fire — the post-skill pulses are blank while C's fight
proceeds (`A guard trainee scrambles to his feet! …tries to hit you…`).

**Why:** a positive-damage skill success routes through `DoSpellDamage`
(`damage_stubs.go:75`), which calls only `v.SetFighting(attackerName)` — it sets
the victim's `.Fighting` **field** but does NOT enroll either combatant in the
engine's `combatOrder` (the list `PerformRound` iterates), and does not set the
attacker's `.Fighting`. And `sendSkillResult` calls `engine.StartCombat` **only
when `result.Damage <= 0`** — so only kick (L1 = 0 damage) enrolls and goes green;
trip (`(lvl/2)+1`) and headbutt (`GET_LEVEL`) deal positive damage and never start
engine combat. C's `damage()` calls `set_fighting` (mutual, adds both to
`combat_list`) **unconditionally**, regardless of damage.

### Fix — enroll on every successful hit (not just 0-damage)
In `sendSkillResult`, change the `StartCombat` gate so it fires for **all** results
with `StartCombat == true` and a **surviving** target, not only `Damage <= 0`:
- Keep the current order: `SkillMessage` (message dice) → `DoSpellDamage` (apply
  damage, may kill) → **`StartCombat(ch, target)`** → deferred improvement. This
  matches C: `damage()` does skill_message, applies damage, `set_fighting`, then
  the handler improves. Do not reorder relative to the deferred-improve loop.
- **Guard against a killed target:** if `DoSpellDamage` killed the victim (it runs
  the death pipeline at POS_DEAD), do NOT `StartCombat`. Gate on the target still
  being alive/valid (e.g. `target.GetPosition() != combat.PosDead`, or use
  `DoSpellDamage`'s return / re-resolve the target) — mirror C, which doesn't
  enroll a corpse. The L1 trip/headbutt do 1 damage vs a ~20-HP trainee, so the
  live path is what we gate, but the guard must be correct.
- `StartCombat` sets mutual `.Fighting` + `combatOrder` and is idempotent (checks
  for an existing pair), so calling it after `DoSpellDamage` (which already set the
  victim's field) simply completes the mutual enrollment.

### ⚠️ MUST VERIFY (R3a — this is a DP-1212-adjacent stream hazard)
`engine.StartCombat` **must draw NO RNG** (C's `set_fighting` is pure list
manipulation). Read it and confirm it consumes zero `dprng` draws — otherwise
enrolling positive-damage hits would insert a phantom draw between the message and
the improvement and desync the shared stream (the exact class of bug DP-1212 was).
State in the PR that you verified StartCombat is draw-free.

## Bug 2 (secondary) — downed-mob stand-up is gated wrong (surfaces after Bug 1)

Once Bug 1 enrolls the knocked-down mob, `processCombatPair` (`engine.go:443-461`)
still mishandles it. The "scrambles to his feet" stand-up is nested **inside**
`if wc.GetWaitState() > 0 { … ; return }`. So a downed NPC whose wait is **already
0** skips the whole block and falls through to
`if attacker.GetPosition() < PosFighting { StopCombat }` (line ~468) — it **ends
combat instead of standing up**.

C (`fight.c:1977-1998`) treats these as **two separate steps**:
```c
if (GET_MOB_WAIT(ch) > 0) { GET_MOB_WAIT(ch) -= PULSE_VIOLENCE; attacks = 0; }
if ((GET_POS(ch) < POS_FIGHTING) && !GET_MOB_WAIT(ch)) {   /* SEPARATE if */
    GET_POS(ch) = POS_FIGHTING;
    act("$n scrambles to $s feet!", ...);   /* + "You drag yourself to your feet." to ch */
}
/* …then attacks proceed… */
```
### Fix — mirror C's two-step structure
Restructure `processCombatPair`'s NPC block so:
1. If wait > 0: decrement and skip this round's attacks (as today), but do NOT
   `return` past the stand-up unconditionally.
2. **Separately**, if `GetPosition() < PosFighting && waitState == 0`: stand up
   (`SetPosition(PosFighting)` + the existing "scrambles to his feet!" broadcast),
   then continue into attacks — do NOT fall into the `StopCombat` branch.

Keep the exact existing broadcast text/pronoun logic (it's already correct at
`engine.go:456`). Confirm the PC path (`engine.go:1990-1998` analogue, if present)
matches C too, but the NPC path is what trip/headbutt exercise.

## Tests
- **Enrollment:** a unit/engine test proving that after a positive-damage skill hit
  (via the real `sendSkillResult` path, target surviving) BOTH combatants are in
  `combatOrder` and mutually `.Fighting`; and that a **killing** hit does NOT
  enroll a corpse.
- **StartCombat draw-free:** assert (via the `dprng`/`levelNumber`-style counting
  seam) that `StartCombat` advances the shared stream by zero.
- **Stand-up:** an engine test where a POS_SITTING NPC with `waitState == 0` in
  combat **stands up** (`scrambles to his feet`) and attacks, rather than combat
  stopping. Mirror the existing `engine_test.go` sitting-orc tests; there's likely
  one asserting the *current* (buggy) behavior — update it to C behavior and note
  it in the PR (don't silently flip — R5e).

## Oracle gate (Claude, after merge — informational)
I re-run `combat-trip-opener` + `combat-headbutt-opener` → expect GREEN (the downed
trainee scrambles up and fights across the pulses, matching C). kick/backstab/
unlearned-bash/combat-death stay green. Per R5a I check the mob's position/wait per
pulse + roll values, not just normalized bytes.

## Guardrails
- **Never** edit `src/`, `darkpawns-c-oracle/`, `lib/` — read-only.
- All gates (GLM.md §gates): `make fmt`, build, vet, `test ./... -race`,
  `golangci-lint run`, `make reachability`.
- Don't stage `.zcode/`, generated reachability reports,
  `website/static/map/world-sphere.json`, `docs/reports/reek/*`.
- Do NOT change the DP-1212 draw ORDER (message→improve) or `DeferredImprove` —
  only add enrollment after `DoSpellDamage` and fix the stand-up gating.

## Deliverable
`sendSkillResult` enrolls both combatants on every surviving-target success hit
(not just 0-damage), verified draw-free; `processCombatPair` stands a downed
wait-0 NPC up instead of ending combat (mirroring C's two-step
`perform_violence`); tests for both + the corpse guard. Claude greens the trip +
headbutt openers.
