# WORKER TASK — improve_skill call-site audit (DP-1168, part 1 of 2)

**Type:** READ-AND-REPORT. Produce a markdown findings report. **Do NOT change game
logic.** No oracle access needed. Claude verifies your report and runs the oracle gate.

**Branch off `main`.** Deliverable: `docs/reports/improve-skill-callsite-audit.md`
(commit it; open a PR titled `docs: improve_skill call-site audit (DP-1168)`).

## Background (already landed — do not re-litigate)
`improveSkill` (`pkg/game/combat_helpers.go`) was rewritten to faithfully port C's
`improve_skill` (`src/act.other.c:1704`). Contract:
```
if IS_NPC: return
percent = GET_SKILL(ch, skill)
if number(1,200) > GET_WIS+GET_INT: return   # ALWAYS drawn for a PC, BEFORE the bounds check
if percent >= 97 || percent <= 0: return
newpercent = number(1,3)                      # only past the gate
SET_SKILL(+newpercent); "improves" msg only on +3
```
The C oracle lives at `~/.openclaw/workspace/darkpawns-c-oracle/src/*.c` — **read-only reference**.

## Your job
For each Go `improveSkill(...)` call site, open the matching C `do_*` function and
verify the call sits at the **same point in control flow AND draw order**: same
success/fail branch, after the same sequence of `number(...)` draws, not gated by an
extra/missing condition. C draw order is law — a call placed after a different number
of `number()` draws desyncs the seeded stream.

### Call sites to audit
| Skill | Go (`pkg/game/skill_combat.go`) | C reference |
|---|---|---|
| backstab | lines ~102, ~122 | `act.offensive.c:229` (`do_backstab`) |
| bash | ~205 | `act.offensive.c:492` (`do_bash`) |
| kick | ~259 | `act.offensive.c:631` (`do_kick`) |
| trip | ~340 | `new_cmds.c:808` (`do_trip`) |
| headbutt | ~430 | `new_cmds.c:450`/`457` (`do_headbutt`) |
| rescue | ~523 | `act.offensive.c:567` (`do_rescue`) |
| circle | ~739 | search `SKILL_CIRCLE` in `act.offensive.c` |
| charge | ~807 | search `SKILL_CHARGE` in `new_cmds.c`/`act.offensive.c` |
| berserk | `pkg/game/skill_berserk_kuji.go:101` | `do_berserk` — **note: C has a dangling-else bug where improve_skill runs unconditionally; preserve that quirk, don't "fix" it** |

(Line numbers are approximate — grep `improveSkill(` in the Go files and the `SKILL_*`
constant in C to locate exact spots.)

### Report format — one section per site
```
### <skill>
- C: <file:line> — <the branch/condition improve_skill(ch, SKILL_X) sits under, and
  the number(...) draws that precede it in that function>
- Go: <file:line> — <the branch/condition the improveSkill call sits under, and the
  dprng.Number draws that precede it>
- Verdict: MATCH | DIVERGENCE
- If DIVERGENCE: exactly what differs (wrong branch / extra or missing draw before the
  call / gated differently). Do not propose a code fix — just describe precisely.
```
End with a summary table (skill → MATCH/DIVERGENCE).

## Guardrails
- Never edit anything under `darkpawns-c-oracle/` — it is the reference.
- Don't touch game logic in this PR; it's docs-only.
- If a Go call site's skill isn't implemented in the port, note "NOT WIRED" and move on.
