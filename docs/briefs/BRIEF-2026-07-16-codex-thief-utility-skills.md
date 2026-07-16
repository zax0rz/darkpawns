# BRIEF (codex) — Faithful port of the newbie thief utility skills

**Owner:** codex (frontier). **Gate:** Claude runs the differential oracle red→green.
**Branch off `main`.** Scope is intentionally sized to one PR (or a small stacked pair).

## Why this chunk
`improve_skill` and guild learning are now faithful (PRs #356/#359/#361/#365). The next
high-value, **oracle-gateable-at-level-1** target is the thief utility skills a newbie
starts with: `hide`, `sneak`, `steal` (`do_start` grants SNEAK=10, HIDE=5, PEEK=15,
STEAL=15, BACKSTAB=10, PICK_LOCK=10 — class.c:558-563). Unlike the combat skills
(kick/bash/…, which need level+learning to exercise), these can be driven on a fresh
level-1 thief, so **every fix here is provable by a live telnet differential**, not just
a golden test. That's the reason to do these now.

The current Go ports (`pkg/game/skill_stealth.go`) are approximations with explicit
TODOs ("we don't have dex_app_skill table yet") and several fidelity gaps.

## Prerequisite — port `dex_app_skill` (do this first)
C `const struct dex_skill_type dex_app_skill[]` — `constants.c:1060`, 26 rows (dex
0..25), 5 int columns. Field order is the struct in `structs.h` (`struct
dex_skill_type` — confirm exact names; conventionally `{p_pocket, p_locks, traps,
sneak, hide}`). Port it verbatim into `pkg/game` (mirror how `int_app`/`prac_params`
were ported) with an accessor like `dexAppSkill(dex, field)` that clamps dex to
[0,25]. **Do not "simplify" any value.** Add a golden test asserting a few rows
byte-for-byte against the C table.

## The skills — C model + current Go divergences

Read the C directly at `~/.openclaw/workspace/darkpawns-c-oracle/src/act.other.c`
(read-only). Draw order is law: match every `number()` call and its order exactly.

### hide — `do_hide` (act.other.c:247, body ~271-306)
C flow:
1. Always `send_to_char("You attempt to hide yourself.\r\n")` (the `subcmd` kabuki
   variant is ninja-only — ignore for the newbie path).
2. `if (IS_AFFECTED(ch, AFF_HIDE)) REMOVE_BIT_AR(...)` — **clears the bit, does NOT
   return**; it then re-rolls.
3. `percent = number(1,101)`
4. `if (percent > GET_SKILL(HIDE) + dex_app_skill[GET_DEX(ch)].hide) return;` — silent
   failure (no extra message).
5. `SET_BIT_AR(AFF_FLAGS, AFF_HIDE); improve_skill(ch, SKILL_HIDE);`

Go `DoHide` gaps: returns "You step out of the shadows" and bails when already hidden
(C re-rolls); missing the `dex_app_skill[...].hide` term; invented failure/success
messages; **no `improveSkill` call.**

### sneak — `do_sneak` (act.other.c:214)
Mirror-image of hide but with `.sneak` and `AFF_SNEAK`, plus C's exact "Okay, you'll
try to move silently…" wording and a duration/affect application (`affect_to_char`
with a timed sneak affect — port the affect, not just a boolean bit, if C uses one).
Verify whether `do_sneak` calls `improve_skill` (grep it) and match. Go `DoSneak` has
the same missing-dex-bonus and message gaps and the same early-return-on-toggle bug.

### steal — `do_steal` (act.other.c:309)
The big one. Three branches (equipment / inventory-item / coins). Cite exactly:
- `percent = number(1,101)` up front; heavy items add weight, higher-level victims add
  a level delta to `percent` (act.other.c:451-453).
- coins: `gold = (GET_GOLD(vict) * number(1,10)) / 100; gold = MIN(1782, gold);` then
  the `Bingo!`/`gold==1` messages and `improve_skill(ch, SKILL_STEAL)` **only when the
  steal succeeds and gold>1** (act.other.c:507-518).
- failure sets `ohoh` → the victim may aggro (act.other.c:495-503).
- `improve_skill(SKILL_STEAL)` also fires on the successful item paths (441/485).
Go `DoSteal` approximates gold and mob-gold and is missing the improve calls and the
exact message set. Match C's draws and messages precisely.

## The `improve_skill` contract (already faithful — just call it)
`improveSkill(ch, skill)` (`pkg/game/combat_helpers.go`) is a faithful port of
`act.other.c:1704`. Call it at the exact C point (after the success bit is set), once
per C call site. It draws `number(1,200)` on every PC call then `number(1,3)` past the
gate — so **placing it at the wrong point, or omitting it, desyncs the seeded stream.**

## Oracle gate (Claude will run — you don't need DP_ORACLE_BIN)
Provide, per skill, a scenario sketch (fixture + setup + probe) in the PR description;
Claude authors/normalizes and runs `dp-oracle-diff`. Rough shapes:
- hide/sneak: fresh L1 thief, `hide` / `sneak` repeated N times in a room; diff the
  per-command output (messages + the improve "improves" line on +3). Repeatable,
  non-combat — ideal.
- steal: L1 thief + a mob with gold in the room; `steal coins <mob>` repeated; diff.
Each must be RED on the pre-fix port and GREEN after. Claude gates every one.

## Guardrails
- **Never** edit anything under `darkpawns-c-oracle/` (world or src) — reference only.
- Branch off `main`; keep the PR focused (dex table + hide/sneak, then steal, is a fine
  split).
- Draw-count/-order parity is the acceptance bar, not just "looks right." When unsure
  whether C draws in a branch, grep the C and match it — including draws you think are
  "cosmetic" (dam_message-style `number()` calls count).
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable
`dex_app_skill` port + golden test; faithful `DoHide`/`DoSneak`/`DoSteal` (messages,
dex bonuses, improve_skill wiring, toggle semantics) with unit tests; PR to `main` with
the per-skill oracle scenario sketches. Claude reconciles + runs the oracle gate.
