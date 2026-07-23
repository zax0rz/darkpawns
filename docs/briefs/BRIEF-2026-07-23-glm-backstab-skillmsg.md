# BRIEF (glm) — route backstab through the skill_message path (DP-1203)

**Owner:** glm-5.2. **Gate:** Claude runs the differential oracle red→green on
`combat-backstab-opener` (+ `combat-death` stays green) + reviews unit tests; CI green.
**Git:** branch off `main` as `glm/backstab-skillmsg`, commit, push, open a PR. Do NOT
merge. Sized to one PR (S/M).
**Closes:** DP-1203. **Related:** DP-1033 (backstab multiplier/to-hit, merged),
DP-1201/DP-1202 (what made this gate-able).
**Cite:** `src/act.offensive.c:220-231` (do_backstab), `src/fight.c:1023-1092`
(skill_message), `src/fight.c:1534` (damage() dispatch), `lib/misc/messages`
(the `131` Backstab set); rules **R1**, **R3**, **R4** (`docs/fidelity/RULEBOOK.md`).

> The combat *engine* and the skill_message infrastructure are already faithful
> and green (that's why `combat-death` passes). This bug is narrow: `DoBackstab`
> **bypasses** that infrastructure with hardcoded strings. You are rerouting it,
> not building new machinery.

## The C truth (do_backstab miss, the gated path)

```c
percent = number(1, 101);                 // DRAW 1
prob    = GET_SKILL(ch, SKILL_BACKSTAB);
if (AWAKE(vict) && (percent > prob))
    damage(ch, vict, 0, SKILL_BACKSTAB);  // miss → damage(0, SKILL_BACKSTAB)
else { hit(ch, vict, SKILL_BACKSTAB); improve_skill(...); }
WAIT_STATE(ch, PULSE_VIOLENCE);
```

`damage(0, SKILL_BACKSTAB)` → (attacktype is not a weapon) → `skill_message(0, ch,
vict, SKILL_BACKSTAB)` which does:

```c
nr = dice(1, fight_messages[i].number_of_attacks);   // DRAW 2 (one draw, even if N==1)
```

then, because `dam == 0`, emits the selected set's **miss_msg** trio. For the fresh
low-skill thief, `percent > prob` almost always, so the gated transcript is the
**miss** branch: **two draws — number(1,101) then dice(1,N)** — and the message
`$N quickly avoids your backstab and you nearly cut your own finger!`
(`lib/misc/messages` set **131**, line 230 → "A guard trainee quickly avoids your
backstab and you nearly cut your own finger!").

## The Go bug

`pkg/game/skill_combat.go` `DoBackstab` returns a `SkillResult` with **hardcoded**
`MessageToCh/Vict/Room` strings on every branch (miss :87-89, to-hit-miss :105-107,
hit :127-129). `sendSkillResult` (`pkg/command/skill_commands.go:1508`) just prints
those strings and routes damage via `DoSpellDamage`. **Nothing ever calls the
skill_message path** (`cbSkillMessage` → `pkg/combat/skill_messages.go:624`, which
DOES draw `Dice(1, len(variants))` exactly like C). So the miss:
- emits an **invented** line ("You try to backstab $N, but $E notices you!" — R4;
  its "notices you" wording wrongly borrows the *MOB_AWARE* branch), and
- **skips DRAW 2**, desyncing the seeded stream (R3).

**Also a latent number bug you must fix:** `combat.SKILL_BACKSTAB = 100`
(`fight_core.go:60`) is a Go-internal enum **unrelated to the messages file**,
which keys the Backstab set by C's number **131** (`lib/misc/messages:226`; Go's
loader `fight_messages.go:109` keys by that file number). `100` is currently used
nowhere but comments. When you route through the skill_message path you MUST pass
attacktype **131** (C's `SKILL_BACKSTAB` / the messages-file key), or the lookup
misses and emits nothing. Confirm how the (green) weapon path obtains its
messages-file-matching attacktype numbers and follow that; do **not** feed the
`100` enum into `cbSkillMessage`.

## The fix

Route backstab's combat message + damage through the **same path C uses**, so the
message comes from `lib/misc/messages` and the `dice(1,N)` draw happens, in order.

- **Draw order is law (R3):** `number(1,101)` FIRST (keep the existing
  `dprng.Number(1,101)`), THEN the skill_message `dice(1,N)` draw. No draws in
  between on the miss branch.
- **Miss branch** (`percent > prob`, awake): emit via the skill_message path with
  `dam=0, attacktype=131` (the C `damage(ch,vict,0,SKILL_BACKSTAB)` equivalent).
  That single call both draws `dice(1,N)` AND emits the miss_msg trio. Then start
  combat (C's `damage()` sets fighting) and `WAIT_STATE` = 1 round (`WaitCh:1` —
  already correct post-DP-1201). **Delete** the hardcoded miss strings.
- **Success branch** (`hit(ch,vict,SKILL_BACKSTAB)`): keep the DP-1033 to-hit roll,
  but on a landed hit route the damage through the skill_message path with
  `attacktype=131` so the **hit_msg** comes from the file (not "Your deadly
  backstab strikes deep!"), and on a to-hit miss route `dam=0` through it too.
  (This branch is NOT exercised by the low-skill gate fixture — verify by
  inspection; Claude may add a forced-success fixture later. Still fix it — R1.)
- **Do not double-emit or double-apply.** `cbSkillMessage` emits all three
  (char/vict/room) itself, so the routed branches must carry **no** `MessageTo*`.
  Damage/death (corpse, XP — DP-942) must still flow through the existing
  `DoSpellDamage` pipeline for `dam>0`; don't lose it, and don't let both
  `DoSpellDamage` and a direct `TakeDamage` apply HP twice.

You choose the cleanest structure. Two viable shapes (pick one, justify in the PR):
1. **Extend the SkillResult applier:** add a field (e.g. `SkillMsgType int` +
   `SkillMsgDam int`) so `sendSkillResult` calls `cbSkillMessage(dam, ch, vict,
   131, room)` in place of the hardcoded `MessageToCh`, keeping `DoSpellDamage`
   for `dam>0`. Lowest blast radius; keeps the SkillResult pattern intact.
2. **Route inside DoBackstab** via the combat engine's `TakeDamage`/damage entry
   (`fight_core.go:309`) directly, returning a message-less SkillResult. More
   faithful to C's call shape but must not double-apply damage.

Reference the WORKING weapon path for the exact wiring: `fight_core.go:434-441`
(`cbSkillMessage(dam,…)` for non-weapon attacktypes) and `skill_messages.go:618-631`
(the `Dice(1,N)` draw). The messages file is already loaded at boot (weapon
skill_messages are green), so the `131` set is available — you're only pointing
backstab at it.

## Tests (real verification — the oracle can gate the miss, unit tests pin the draws)

In `pkg/game` / `pkg/command` (wherever DoBackstab is testable with a seeded roller):
- **draw count + order (miss):** a backstab that misses consumes exactly TWO
  shared draws — `number(1,101)` then the skill_message `dice(1,N)` — in that
  order (assert via a fake/seeded roller index, mirroring the cast_cmds draw tests).
- **message source:** the miss emits the `lib/misc/messages` set-131 miss text, not
  the old invented string; assert no path emits "notices you"/"strikes deep".
- **no double-apply:** a hitting backstab applies HP damage once (not twice).

## Oracle gate (Claude authors/runs)

- **`combat-backstab-opener` RED→GREEN** — the committed anchor
  (`cmd/dp-oracle-diff/scenarios/combat-backstab-opener.txt`): fresh L1 thief
  backstabs non-aware #16303 as the opener; miss message + downstream rounds must
  match C byte-for-byte once the draw is restored.
- **`combat-death` stays GREEN** — you did not perturb the weapon path.

## R5c — first of a class (note in PR, do NOT fix here)

`DoBash`, `DoKick`, `DoDisarm`, … in `skill_combat.go` almost certainly hardcode
their messages the same way (bypassing skill_message). Scope THIS PR to backstab
(the gated one); list the siblings you spot in the PR so we can sweep them next
(each gets its own opener fixture + gate).

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — reference
  only; the messages file is C-authored data.
- `make reachability` zero regressions; `go test -race`; **run `golangci-lint`**;
  `gofumpt -w` every file you touch (worktree pushes bypass the hook).
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

Backstab rerouted through the skill_message path with attacktype **131**, correct
draw count/order, hardcoded strings deleted, death pipeline intact, unit tests +
the R5c sibling list. Claude reconciles + runs the oracle gate.
