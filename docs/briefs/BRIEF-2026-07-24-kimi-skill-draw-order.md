# BRIEF (kimi k3) — DP-1212: fix R3b skill draw-ORDER (defer improvement) + StartCombat-on-success

**Owner:** kimi-k3 (first brief — read the two docs below before starting).
**Gate:** pipeline-level draw-ORDER unit tests now; Claude runs the paired-trace
oracle gate on all successful-path scenarios after. CI green.
**Read first (required):** **`GLM.md`** (the brief-execution operating manual —
workflow, gates, git hygiene, the pitfalls; it governs your behavior) and
**`docs/fidelity/RULEBOOK.md`** (esp. **R3a/R3b/R3d/R5a/R5e**). The C source is the
final authority (R5e) — verify every claim below against it before copying.
**Full root-cause report (your primary reference):**
`docs/reports/dp-1212-rng-draw-order-root-cause-2026-07-24.md` — read it in full;
it has the paired-trace proof, exact call paths, and the acceptance gate.

**Git:** branch off **`glm/god-advancelevel-draws`** (commit `304b4d3d`, PR #469 —
NOT `main`) as `kimi/skill-draw-order`. This bundles the (correct, proven) #469 God
fix with your ordering fix so they land together — the acceptance gate requires it.
Before starting: `git fetch origin && git log --oneline -1 glm/god-advancelevel-draws`
to confirm you're on the right base. Edit → commit → push → open a PR. Do NOT merge.
Sized M (contract change + 4 handlers + tests).

---

## The bug (R3b — operation order is behavior)

On a **successful** skill roll, C draws the fight-message dice **before** the
skill-improvement draw; Go draws them **reversed**. Proven by paired C/Go trace on
`combat-kick-opener` (report §3): first divergence at draw 4100 —

```
        C oracle                     Go port
4099    number(1,101)=44 (roll)      number(1,101)=44        ✓ aligned (thanks to #469)
4100    skill_message dice(1,2)=1    improveSkill number(1,200)=40   ✗ REVERSED
4101    improveSkill (1,200)=194     skill_message dice(1,2)=2
```

The reversed `dice(1,2)` selects a different player-facing message variant (R1).

**Why:** C's `damage()`/`hit()` calls `skill_message()` (which draws
`dice(1,number_of_attacks)`) *before* returning to the command handler, which then
runs `improve_skill()`. Go computes and calls `improveSkill()` **eagerly inside
`Do*`**, then **defers** the `SkillMessage` draw to `sendSkillResult()`
(`pkg/command/skill_commands.go:~1522`). So Go's order is improve→message; C's is
message→improve. Verify: `pkg/game/skill_combat.go` `DoKick` calls
`improveSkill(ch, SkillKick)` right before its success `return`.

This is a **class (R5c)** — the same eager-improve/deferred-message shape is in
`DoBackstab`, `DoKick`, `DoTrip`, `DoHeadbutt` (all in `skill_combat.go`). **This PR
fixes those four.** (Bash is excluded: it still hardcodes messages instead of
`skill_message`, so its message draw can't be ordered until a separate reroute PR
lands. Do not touch `DoBash` here.)

> ⚠️ **The existing headbutt draw-order test is WRONG.** `headbutt_skillmsg_test.go`
> (around line 249) documents and asserts the current Go order
> (`improve → dice(1,N)`). It was built from the implementation, so it passes while
> encoding the bug (R5a — "a test built from the same logic as the impl is a
> re-skin; it passes with the bug"). You must **reverse** what it asserts to the C
> order. Do not treat its current green as evidence of correctness.

## Second bug in scope — StartCombat omitted on success (enrollment gap)

All five skills' **successful** `SkillResult` literals omit `StartCombat: true`.
`sendSkillResult` enrolls combat via `DoSpellDamage` only when `Damage>0`, else via
`StartCombat` only when `Damage<=0`. A level-1 kick success deals
`GET_LEVEL>>1 == 0` damage → neither path fires → **empty combat pulses** (C's
`damage(...,0,...)` still enrolls both combatants). Fix: add `StartCombat: true` to
the success literals for the **four in-scope skills** (backstab/kick/trip/headbutt).
It's safe — the sender only calls `StartCombat` when `Damage<=0`, so positive-damage
paths won't double-enroll.

## The fix — defer improvement to the sender so it runs in C order

**Do NOT** fix this by swapping raw calls or burning dummy draws — the *side
effects and their order* must match C (R3b/R1/R4). Represent deferred improvement
on the result and let the sender run the C sequence.

1. **Add a field to `SkillResult`** (`pkg/game/` — where `SkillResult` is defined):
   ```go
   // DeferredImprove lists the skills to run improveSkill() on AFTER the
   // skill_message/damage step, matching C's order (skill_message draws its
   // dice inside damage()/hit(), THEN improve_skill runs). Ordered; repeat an
   // entry for a skill C improves twice (headbutt). DP-1212 / R3b.
   DeferredImprove []string
   ```
2. **In `Do{Backstab,Kick,Trip,Headbutt}` success paths:** REMOVE the eager
   `improveSkill(ch, ...)` call; instead set `DeferredImprove: []string{Skill<X>}`
   on the returned `SkillResult`. **Headbutt** calls `improveSkill` twice in C
   (`new_cmds.c`) → `DeferredImprove: []string{SkillHeadbutt, SkillHeadbutt}`.
   **Backstab** improves on BOTH its to-hit-miss and hit sub-paths (verify against
   `act.offensive.c` do_backstab + `hit()`); set `DeferredImprove` on whichever
   sub-paths C improves, keeping C's order (to-hit/damage/message THEN improve).
3. **In `sendSkillResult` (`skill_commands.go`):** after the existing
   `SkillMessage` (and `DoSpellDamage`/`StartCombat`) block — i.e. after the
   message dice is drawn and damage/enrollment is applied — iterate
   `result.DeferredImprove` and call the **real** `improveSkill(ch, name)` for each,
   in order. This makes the runtime order message-dice → improve-draw, matching C.
   - Use the real `improveSkill` (it may emit player-facing "You are getting
     better…" output and draws `number(1,200)` [+ `number(1,3)`]) — do NOT stub or
     dummy-draw (R1/R4).
   - Confirm `DoSpellDamage` does not itself draw RNG between the message and the
     improvement for these skills (their damage is a fixed formula); if it does,
     that ordering must still match C — trace it.

## Draw-order per skill (verify each against C before coding — R5e)
| skill | C success order | improves |
|---|---|---|
| kick | `damage`→skill_message dice, then improve | 1 |
| trip | `damage`→skill_message dice, then improve | 1 |
| headbutt | `damage`→skill_message dice, then improve ×2 | 2 |
| backstab | `hit`/`damage`→skill_message dice, then improve | 1 (on the C-improved sub-path) |

## Tests
- **Reverse** the headbutt draw-order test to assert C order
  (skill_message `dice(1,N)` → `number(1,200)` improve → `number(1,3)` → second
  `number(1,200)` improve, per C do_headbutt — verify the exact sequence).
- **Add pipeline-level draw-order tests** for each in-scope skill that drive the
  **real `sendSkillResult` path** (not `Do*` and `SkillMessage` separately — that's
  what hid the bug). Assert the **next shared-stream value after the full
  operation** (roll → message dice → improve draws), and that combat is enrolled.
- Existing tests asserting the old (improve-first) order must be updated to C order,
  not deleted — and note in the PR which you changed and why (surface it; don't
  silently "fix" — R5e).

## Oracle gate (Claude, after merge — informational)
I author **successful-path** fixtures for all four skills (the 75% openers can land
on the failure branch — backstab's traced roll was 84 → failure — so I force the
success path) and gate on: `combat-kick-opener` byte-green with kick roll 44 on
both, message-dice-before-improve on both, combat present in the pulses, and a
matching `(range,result)` paired trace through the opener + immediate combat, for
all four skills. Until then your pipeline draw-order unit tests are the gate.

## Guardrails (new executor — read GLM.md; these are load-bearing)
- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/` — read-only C ground truth.
- Verify every C claim above by reading the C function yourself (R5e). The report is
  strong but the source is law.
- Run ALL gates (GLM.md §gates): `make fmt`, `go build ./...`, `go vet ./...`,
  `go test ./... -race`, `golangci-lint run ./...`, `make reachability`.
- Don't stage `.zcode/`, generated reachability reports,
  `website/static/map/world-sphere.json`, or `docs/reports/reek/*`.
- Do NOT touch `DoBash` or bash tests (out of scope — separate reroute PR).

## Deliverable
`SkillResult.DeferredImprove` + the four success paths deferring improvement +
`sendSkillResult` running message→improve in C order + `StartCombat:true` on the
four success literals + the reversed/added draw-order tests, on a branch off #469.
Claude authors the successful-path fixtures and runs the paired-trace gate; the
class is done only when all four successful-path scenarios are byte-green with
matching draw traces (R5a — roll values + draw order, not just normalized bytes).
