# BRIEF (glm) — bash reroute: the full catch-up (skill_message + defer improve + StartCombat)

**Owner:** glm-5.2 (you did the kick/trip/headbutt reroutes — this is the 5th and
last, applying all three now-established patterns to bash at once).
**Gate:** byte/draw-exact unit tests now; Claude runs the bash-opener oracle gate
after (skill-resolution layer only — see the gate note). CI green.
**Git:** branch off `main` as `glm/bash-skillmsg`. `git fetch origin && rebase onto
origin/main` first (main now has #469/#470/#473 — the God-draw, draw-order, and
enrollment fixes bash must build on). Confirm `merge-base HEAD origin/main ==
origin/main` tip. Edit → commit → push → PR. Do NOT merge. Sized M.
**Finding:** DP-1207/DP-1211 class (skill_message reroute), applied to **bash** —
the only combat skill still hardcoding its messages. Its *outcome* was already
fixed by #469's draw alignment; this is purely the message/improve/enrollment
catch-up.
**Cite:** C `src/act.offensive.c:419-497` (do_bash; outcome at :483-497), `damage()`
→skill_message + set_fighting; Go `pkg/game/skill_combat.go` (`DoBash`, currently
hardcoded + eager improve), `pkg/game/death.go:710` (`SkillBashNum = 132`, already
defined + in the corpse switch); reference PRs #464 (kick reroute), #467
(trip/headbutt), #470 (`DeferredImprove`), #473 (`StartCombat`-on-success). Rules
**R1/R3a/R3b/R4/R5e**.

---

## The C truth (verify yourself — R5e)
`do_bash` outcome (`act.offensive.c:483-497`):
```c
if (percent > prob) { damage(ch, vict, 0, SKILL_BASH); GET_POS(ch)=POS_SITTING; WAIT_STATE(ch, PULSE_VIOLENCE); }
else if (damage(ch, vict, (GET_LEVEL(ch)/2)+1, SKILL_BASH)) {   /* hit AND damage landed */
    if (!subcmd) improve_skill(ch, SKILL_BASH);
    GET_POS(vict)=POS_SITTING; WAIT_STATE(vict, PULSE_VIOLENCE*2);
}
if (!IS_NPC(ch)) WAIT_STATE(ch, PULSE_VIOLENCE*2);   /* both branches → ch wait = 2 */
```
- `damage()` routes the message through **skill_message** (set **132**, drawing
  `dice(1,N)`) and calls `set_fighting` (mutual enroll) — hit AND miss.
- Draw order on a hit: `number(1,101)` [the percent roll] → `dice(1,N)` [skill_message]
  → `number(1,200)` [improve_skill]. Same shape as kick/trip/headbutt.
- Message set 132 exists (`lib/misc/messages`: "Your bash at $N sends $M sprawling!").

## The fix — apply the three established patterns to `DoBash`
Mirror `DoKick`/`DoTrip` post-#470/#473. In `DoBash` (`skill_combat.go`):

1. **Reroute messages (like #467):** delete the hardcoded `MessageToCh/Vict/Room`
   on BOTH the miss and hit returns and the now-unused `chPronouns/victPronouns`;
   set `SkillMsgType: SkillBashNum` (132) on both.
2. **Defer improvement (like #470):** remove the eager `improveSkill(ch, SkillBash)`
   on the hit path; add `DeferredImprove: []string{SkillBash}` to the hit result so
   the improve draw runs AFTER the skill_message dice (via `sendSkillResult`) —
   matching C's `damage()`→…→`improve_skill` order. (Bash improves only on hit, and
   only `!subcmd`; the player path is subcmd 0, so always on a hit.)
3. **Enroll combat (like #473):** set `StartCombat: true` on BOTH the miss and hit
   results — C's `damage(…,0,…)` (miss) and `damage(…,dam,…)` (hit) both
   `set_fighting`. (Miss deals 0 → enrolls via the `Damage<=0` path; hit deals
   positive → enrolls via #473's surviving-hit path.)

### Preserve exactly as-is (do NOT change — not part of the reroute)
- The formula: `percent = ((5-(AC/10))*2)+Number(1,101)`, `prob=GetSkill(bash)`, the
  move-cost (`SpendMove(10)`), the MOB_NOBASH / immortal-caster overrides.
- `dam = (GetLevel()/2)+1` (matches C).
- The position/wait effect fields: **miss** keeps `SelfStumble: true` (C
  `GET_POS(ch)=POS_SITTING`) + `WaitCh: 2`; **hit** keeps `TargetFalls: true` (C
  `GET_POS(vict)=POS_SITTING`) + `WaitCh: 2` + `WaitTarget: 2`
  (C `WAIT_STATE(vict, PULSE_VIOLENCE*2)`).
- ⚠️ **Verify `StunTarget`:** the current hit result also sets `StunTarget: true`,
  but C only sets `POS_SITTING` (sitting, not stunned). If `StunTarget` drives the
  victim below POS_SITTING it diverges from C — **flag it in the PR as a possible
  separate finding; do not fix it here** (out of the reroute's scope). Just don't
  *add* stun; leave the field as the current code has it.
- The `else if (damage(...))` guard: C only knocks down / improves / waits the
  victim if the damage actually landed. If the current Go hit path already applies
  TargetFalls/improve unconditionally, leave that behavior as today (the fixture
  always lands 1 damage); note the guard in the PR for a later pass.

## Draw parity (R3 — must hold)
Hit path consumes, in order: `Number(1,101)` (already drawn in DoBash) →
`dice(1,N)` (skill_message, in sendSkillResult) → `number(1,200)` (deferred
improve, in sendSkillResult). Assert via a pipeline draw-order test through the
real `sendSkillResult` (not `DoBash`+`SkillMessage` separately). Confirm
`StartCombat` is draw-free (it is — verified in #473, but re-confirm).

## Tests (mirror `kick_skillmsg_test.go` / the #470/#473 tests)
- **miss:** `SkillMsgType==SkillBashNum`, `StartCombat`, `SelfStumble`, empty
  `MessageToCh/Vict/Room`; `messages.Variants(SkillBashNum)` resolves.
- **hit:** `SkillMsgType==SkillBashNum`, `Damage==(level/2)+1`, `StartCombat`,
  `TargetFalls`, `DeferredImprove==[SkillBash]`, empty hardcoded messages.
- **draw order:** pipeline test — roll → skill_message dice → improve, next-stream
  value asserted after the full op.
- **DP-1206 gate regression:** `GetSkill(bash)==0` still returns the bare
  martial-arts message unchanged.

## Oracle gate (Claude, after merge — informational; read this)
Bash hit at L1 knocks the victim down (positive damage → sitting), so
`combat-bash-opener`'s post-hit pulses will hit the **same melee-round layer as
trip/headbutt (DP-1215)** — it will NOT go fully green. That's expected. I gate on
bash's **skill-resolution** correctness: the `bash trainee` **probe block**
byte-matches C (faithful skill_message variant), and kick/backstab/combat-death/
unlearned-bash stay green. Full bash-opener green is blocked on DP-1215, same as
trip/headbutt — closing the skill layer is the deliverable here.

## Guardrails
- **Never** edit `src/`, `darkpawns-c-oracle/`, `lib/` — read-only.
- All gates (GLM.md §gates): `make fmt`, build, vet, `test ./... -race`,
  `golangci-lint run`, `make reachability`.
- Don't stage `.zcode/`, generated reachability reports,
  `website/static/map/world-sphere.json`, `docs/reports/reek/*`.
- Do NOT change the DP-1212 draw ORDER for other skills or the formula/effects.

## Deliverable
`DoBash` rerouted through `skill_message` (`SkillBashNum`), improvement deferred,
`StartCombat` on both branches, hardcoded strings + pronoun vars deleted,
formula/effects/waits preserved, `StunTarget` flagged (not fixed), draw order
verified, mirror tests. Closes the combat-skill reroute class (all 5). Claude
gates the skill-resolution layer; the melee rounds are DP-1215.
