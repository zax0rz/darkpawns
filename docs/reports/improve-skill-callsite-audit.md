# `improve_skill` call-site audit (DP-1168, part 1)

This report compares each audited Go `improveSkill(...)` call site in the port with the matching C `do_*` function in the oracle, focusing on **control-flow placement** and **RNG draw order**. The C reference is `~/.openclaw/workspace/darkpawns-c-oracle/src/`.

`improveSkill` (`pkg/game/combat_helpers.go:41`) ports C `improve_skill` (`src/act.other.c:1704`). Its internal contract is:

1. `if IS_NPC: return`
2. `percent = GET_SKILL(ch, skill)`
3. `if number(1,200) > GET_WIS+GET_INT: return` — **drawn on every PC call, before the percent bounds check**
4. `if percent >= 97 || percent <= 0: return`
5. `newpercent = number(1,3)` — drawn only past the gate
6. `SET_SKILL(+newpercent)`; "improves" message only on `+3`

The audit below lists, for each call site, the RNG draws that precede the `improve_skill`/`improveSkill` call **within the same skill handler** and the branch that reaches it.

---

### backstab

- **C:** `src/act.offensive.c:229` — inside the `else` branch of `if (AWAKE(vict) && (percent > prob))` (i.e., the skill-roll success path). Immediately after `hit(ch, vict, SKILL_BACKSTAB)`.
  - Preceding draws in `do_backstab`:
    1. `number(1, 101)` at line 221 (skill success roll).
    2. `number(50, 100)` at line 222 **only when `subcmd` is true** (prob calculation).
    3. `hit()` at line 228 performs its own internal d20/THAC0 draws before `improve_skill` is reached.

- **Go:** `pkg/game/skill_combat.go:102` and `:122` — two call sites instead of one.
  - Line 102 is inside `if !combat.CalculateHitChance(...)` (the hit-check miss path) after the skill-roll success path.
  - Line 122 is in the hit path after damage calculation.
  - Preceding draws in `DoBackstab`:
    1. `dprng.Number(1, 101)` at line 78 (skill success roll).
    2. `combat.CalculateHitChance(...)` at line 101 performs its own internal d20/THAC0 draws before either `improveSkill` call is reached.

- **Verdict: DIVERGENCE**
  - C calls `improve_skill` **once**, unconditionally after `hit()` in the skill-roll success branch.
  - Go splits this into **two call sites** (miss-after-hit-check and hit). Each execution path consumes exactly one `improveSkill` call, so the per-path RNG draw count inside `improveSkill` matches C, but the call-site structure and control-flow placement differ.

---

### bash

- **C:** `src/act.offensive.c:492` — inside `else if (damage(ch, vict, ...))`, gated by `if (!subcmd)`.
  - Preceding draws in `do_bash`:
    1. `number(1, 101)` at line 475 (skill success roll).
    2. `number(50, 100)` at line 476 **only when `subcmd` is true** (prob calculation).
    3. `damage()` at line 489 performs its own internal draws before `improve_skill` is reached.

- **Go:** `pkg/game/skill_combat.go:205` — in the success branch (`percent <= prob`), after damage calculation. `DoBash` has no `subcmd` parameter.
  - Preceding draws in `DoBash`:
    1. `dprng.Number(1, 101)` at line 171 (skill success roll).

- **Verdict: DIVERGENCE**
  - Go always calls `improveSkill` on a successful bash; C gates the call with `if (!subcmd)`. The success branch placement and the main skill-roll draw match, but the subcmd guard is missing in the port.

---

### kick

- **C:** `src/act.offensive.c:631` — inside the `else` branch of `if (percent > prob)` (success path), immediately after `damage(ch, vict, GET_LEVEL(ch) >> 1, SKILL_KICK)`.
  - Preceding draws in `do_kick`:
    1. `number(1, 101)` at line 622 (skill success roll).

- **Go:** `pkg/game/skill_combat.go:259` — inside the `else` branch of `if (percent > prob)` (success path), before building the success `SkillResult`.
  - Preceding draws in `DoKick`:
    1. `dprng.Number(1, 101)` at line 240 (skill success roll).

- **Verdict: MATCH**
  - Same branch, same single preceding draw, no subcmd involvement in either version.

---

### trip

- **C:** `src/new_cmds.c:808` — inside `else if (damage(ch, victim, ...))`, gated by `if (!subcmd)`.
  - Preceding draws in `do_trip`:
    1. `number(1, 121)` at line 788 (skill success roll).
    2. `damage()` at line 805 performs its own internal draws before `improve_skill` is reached.

- **Go:** `pkg/game/skill_combat.go:340` — in the success branch (`percent <= prob`), after damage calculation. `DoTrip` has no `subcmd` parameter.
  - Preceding draws in `DoTrip`:
    1. `dprng.Number(1, 121)` at line 305 (skill success roll).

- **Verdict: DIVERGENCE**
  - Same branch and main draw match, but Go lacks the `if (!subcmd)` guard that C has around `improve_skill`.

---

### headbutt

- **C:** `src/new_cmds.c:450` and `:457` — both inside the `else` branch of `if (percent > ...)` (success path), after `damage(ch, victim, GET_LEVEL(ch), SKILL_HEADBUTT)`.
  - Line 450 is unconditional within the success branch.
  - Line 457 is additionally gated by `if (!subcmd)`.
  - Preceding draws in `do_headbutt`:
    1. `number(1, 121)` at line 422 (skill success roll).
    2. `number(50, 100)` at line 437 **only when `subcmd` is true** (prob calculation).

- **Go:** `pkg/game/skill_combat.go:430` — in the success branch (`percent <= skillLevel`), after damage calculation. `DoHeadbutt` has no `subcmd` parameter.
  - Preceding draws in `DoHeadbutt`:
    1. `dprng.Number(1, 121)` at line 388 (skill success roll).

- **Verdict: DIVERGENCE**
  - C calls `improve_skill` **twice** in the non-subcmd success path (lines 450 and 457) and once in the subcmd success path. Go calls `improveSkill` exactly once on success. Additionally, Go has no subcmd handling.

---

### rescue

- **C:** `src/act.offensive.c:567` — inside the `else` branch of `if (percent > prob)` (success path), after the success messages and before `stop_fighting`/`set_fighting`.
  - Preceding draws in `do_rescue`:
    1. `number(1, 101)` at line 553 (skill success roll).

- **Go:** `pkg/game/skill_combat.go:523` — inside the `else` branch of `if (percent > prob)` (success path), before `combatEngine.StopCombat`/`StartCombat`.
  - Preceding draws in `DoRescue`:
    1. `dprng.Number(1, 101)` at line 506 (skill success roll).

- **Verdict: MATCH (with note)**
  - The call-site branch and draw order match. C uses `prob = subcmd ? 100 : GET_SKILL(ch, SKILL_RESCUE)`; Go has no subcmd parameter and always uses `ch.GetSkill(SkillRescue)`. This changes the success probability but does not change where `improveSkill` sits in control flow.

---

### circle

- **C:** `src/new_cmds.c:2464` — inside the `else` branch of `if (AWAKE(vict) && (percent > prob))` (skill-roll success path), immediately after `hit(ch, vict, SKILL_CIRCLE)`.
  - Preceding draws in `do_circle`:
    1. `number(1, 101)` at line 2449 (skill success roll).
    2. `hit()` at line 2463 performs its own internal draws before `improve_skill` is reached.

- **Go:** `pkg/game/skill_combat.go:739` — inside the `else` branch of `if (target.GetPosition() > combat.PosSleeping && percent > prob)` (skill-roll success path), after damage calculation.
  - Preceding draws in `DoCircle`:
    1. `dprng.Number(1, 101)` at line 701 (skill success roll).

- **Verdict: MATCH**
  - Same branch and same single preceding skill-roll draw. Both call `improve_skill`/`improveSkill` only in the skill-roll success path.

---

### charge

- **C:** `src/new_cmds.c:952` — inside the `else` branch of `if (percent > prob)` (success path), after `damage(...)`, gated by `if (!subcmd)`.
  - Preceding draws in `do_charge`:
    1. `number(1, 101)` at line 926 (skill success roll).

- **Go:** `pkg/game/skill_combat.go:807` — inside the `else` branch of `if (percent > prob)` (success path), after damage calculation. `DoCharge` has no `subcmd` parameter.
  - Preceding draws in `DoCharge`:
    1. `combat.GetRoller().Number(1, 101)` at line 777 (skill success roll).

- **Verdict: DIVERGENCE**
  - Same branch and the equivalent preceding draw match, but Go lacks the `if (!subcmd)` guard that C has around `improve_skill`.

---

### berserk

- **C:** `src/new_cmds.c:2169` — due to a dangling-else bug, `improve_skill(ch, SKILL_BERSERK)` is **not actually inside the `else` branch** (no braces), so it runs unconditionally after the success/failure messages.
  - Preceding draws in `do_berserk`:
    1. `number(1, 101)` at line 2147 (skill success roll).
    2. `number(50, 100)` at line 2159 **only when `subcmd` is true** (prob calculation).

- **Go:** `pkg/game/skill_berserk_kuji.go:105` — placed unconditionally after the `failed` check, deliberately preserving the C dangling-else behavior.
  - Preceding draws in `DoBerserk`:
    1. `dprng.Number(1, 101)` at line 87 (skill success roll).

- **Verdict: MATCH**
  - Go explicitly ports the dangling-else quirk: `improveSkill` runs on every use, not just on success. The preceding skill-roll draw matches.

---

## Summary

| Skill | Verdict | Notes |
|---|---|---|
| backstab | DIVERGENCE | Go splits the single C call into two sites (hit-check miss and hit paths). Per-path draw count matches, but control-flow placement differs. |
| bash | DIVERGENCE | Go missing `if (!subcmd)` guard around `improveSkill`. |
| kick | MATCH | Same branch, same single preceding draw. |
| trip | DIVERGENCE | Go missing `if (!subcmd)` guard around `improveSkill`. |
| headbutt | DIVERGENCE | C calls `improve_skill` twice in non-subcmd success path; Go calls once. No subcmd handling in Go. |
| rescue | MATCH* | Same branch and draw order; *prob differs because Go lacks subcmd (prob=100) path. |
| circle | MATCH | Same branch, same single preceding draw. |
| charge | DIVERGENCE | Go missing `if (!subcmd)` guard around `improveSkill`. |
| berserk | MATCH | Go faithfully preserves C's dangling-else unconditional call. |

**Overall:** 5/9 audited sites diverge from the C reference in the placement or gating of the `improve_skill` call. The most common pattern is the missing `subcmd` guard in the Go port for bash, trip, and charge. Headbutt additionally differs in call count. Backstab differs structurally by splitting one C call into two Go sites.

No game logic was changed in producing this report.

---

## Reconciliation — Claude verification (DP-1168, part 2)

The part-1 audit correctly identified the structural differences, but the gate
step is: **does the difference change player-observable behavior / the seeded draw
stream?** Verifying each against C reduces the 4 reported divergences to **1 real
fidelity bug**.

### The `subcmd` divergences are no-ops for players — bash, trip, charge: NOT real
`subcmd` is only ever non-zero when a **mob spec-proc** invokes the skill
(`spec_procs.c:524/528/553/555` pass `subcmd=1`); every command-table registration
passes `subcmd=0` (`interpreter.c:347/378/783/…`). And `improve_skill` returns
immediately on `IS_NPC` (`act.other.c:1710`). So **anyone who can actually improve a
skill is always `subcmd==0`**, making C's `if(!subcmd)` guard always-true for them.
The port's unconditional `improveSkill` on player success is therefore **behaviorally
identical to C**. (The `number(50,100)` prob draw that C makes "only when subcmd" also
never fires for a player, matching the port.) Trip is never even called with
`subcmd=1` anywhere. **Verdict: MATCH for player fidelity.**

### backstab — NOT real (draw-equivalent)
Go's two call sites are **mutually exclusive**: the skill-roll miss path
(`skill_combat.go:82`) calls neither; on skill-roll success exactly one of `:102`
(to-hit miss) or `:122` (to-hit hit) fires — one `improveSkill` per execution, same as
C's single call after `hit()`. Draw order also matches: C does `hit()` (to-hit d20,
then damage dice on a hit) → `improve_skill`; Go does `CalculateHitChance` (to-hit) →
[damage dice on hit] → `improveSkill`. **Verdict: MATCH.**

### headbutt — REAL (fixed)
C calls `improve_skill` **twice** on a player's success: `new_cmds.c:450`
(unconditional) and `:457` (`if(!subcmd)`, always true for players), separated only
by a non-drawing position update. The port called it once, **dropping a `number(1,200)`
gate draw and desyncing the seeded stream on every successful headbutt.** Fixed in
`skill_combat.go` (two consecutive `improveSkill` calls). Proven by
`TestDoHeadbutt_ImprovesSkillTwice` (golden, red→green: success yields two independent
`number(1,3)` increments; ResetStream-seeded, deterministic).

### Reconciled summary
| Skill | Part-1 verdict | Verified verdict | Action |
|---|---|---|---|
| backstab | DIVERGENCE | **MATCH** (draw-equivalent) | none |
| bash | DIVERGENCE | **MATCH** (subcmd no-op for players) | none |
| kick | MATCH | MATCH | none |
| trip | DIVERGENCE | **MATCH** (subcmd no-op for players) | none |
| headbutt | DIVERGENCE | **DIVERGENCE** (missing 2nd call) | **fixed + golden test** |
| rescue | MATCH* | MATCH | none |
| circle | MATCH | MATCH | none |
| charge | DIVERGENCE | **MATCH** (subcmd no-op for players) | none |
| berserk | MATCH | MATCH | none |

**Net: 1 real fidelity bug (headbutt), fixed.** The audit's value was flagging the
structural differences; the gate's value was determining which are observable.

### Oracle note
`improve_skill` fires only mid-combat on a skill the caster owns. The only
improve-wired skill a newbie can use is **backstab** (thief, L1) — and it's a MATCH,
so there's nothing to prove red→green there. Headbutt is a **level-15 warrior skill**;
the C oracle has no auto-implementor and `advance`/`set` need an already-god char, so a
level-15 char can't be scripted into a fresh oracle boot. The headbutt fix is therefore
gated by the deterministic CMWC-stream golden test (same standard accepted for the
slice-3a core), not a live telnet differential.
