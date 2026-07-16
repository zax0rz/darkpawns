# BRIEF (codex) — Faithful level-1 caster spells (mage + cleric)

**Owner:** codex (frontier). **Gate:** Claude runs the differential oracle red→green.
**Branch off `main`.** Sized to one PR (or a small stacked pair).

## Why this chunk
Same "provable at level 1" sweet spot as the thief skills: a fresh L1 mage/cleric can
cast their starting spells, so every fix is oracle-gateable by a live telnet
differential. The L1-castable spells (from `class.c` spell_level assignments) are:

- **mage (CLASS_MAGIC_USER, level 1):** `magic missile` (damage), `infravision` (affect)
- **cleric (CLASS_CLERIC, level 1):** `cure light` (heal)

Start with **magic missile** and **cure light** (RNG damage/heal → the draw-parity
test); `infravision` (a pure affect + message) is a good third.

## The casting pipeline — draw order is LAW
C `cast_spell`/`do_cast` (`~/.openclaw/workspace/darkpawns-c-oracle/src/spell_parser.c`,
read-only). The player-typed cast path, in order:
1. target resolution, position/mana checks (no draw).
2. **cast-success roll:** `if (number(0, 101+weight_add) > GET_SKILL(ch, spellnum))`
   → failure ("You lost your concentration!" etc.) — `spell_parser.c:1098`. **One draw,
   always, on every cast attempt.** `weight_add` comes from carried weight; for a fresh
   L1 caster it's 0 (confirm).
3. `say_spell(ch, spellnum, tch, tobj)` — the incantation broadcast (`spell_parser.c:271`),
   with its garble/known-vs-unknown text. No RNG, but the exact TO_CHAR/TO_ROOM strings
   are first-class fidelity — match them.
4. the spell effect (below).
5. `WAIT_STATE` — match the pulse count.

Any reordering or missing draw here desyncs the seeded stream. Verify whether the port
draws the success roll BEFORE `say_spell` and the effect, exactly as C does.

## The effects
- **magic missile** — `mag_damage` (`src/magic.c:~618-633`): `dam = dice(4,3) + level`
  (plus a reagent bonus if `has_reagents`; a fresh newbie has none → `dice(4,3)+level`).
  4 `number(1,3)` draws for the dice, in order, AFTER the success roll. Then damage is
  applied via the normal damage path (mind its own draws/messages — compare to the
  already-faithful combat `damage()`).
- **cure light** — the heal path (`mag_points`/cleric heal in `magic.c`): find the exact
  `dice(...)` for cure_light and the cap at `GET_MAX_HIT`, plus the "You feel better."
  / already-full messages. Draws the heal dice after the success roll.
- **infravision** — `mag_affects`: applies AFF_INFRAVISION for a level-scaled duration,
  with C's exact "Your eyes glow red." / already-affected messages. No damage dice.

## Current Go state
The port already has spell scaffolding — audit and fix, don't rebuild from scratch:
`pkg/spells/` (`damage_spells.go`, `affect_spells.go`, `say_spell.go`, `spells.go`),
`pkg/session/cast_cmds.go`. Check: (a) the cast-success roll uses
`number(0, 101+weight_add)` in C's position; (b) magic missile is `dice(4,3)+level`
with the draws in C's order; (c) `say_spell` strings match; (d) cure_light dice + cap +
messages; (e) mana cost and WAIT_STATE parity.

## Oracle gate (Claude runs — you don't need DP_ORACLE_BIN)
Provide a scenario sketch per spell in the PR description; Claude authors/normalizes and
runs `dp-oracle-diff`. Shapes:
- **magic missile:** fresh L1 mage, `spawn-mob 16303 1 8105 80`, warmup `n/e/s/e`, probe
  `cast 'magic missile' trainee` repeated. Diff the incantation, hit/damage message, and
  next RNG observation. (Caveat: a mob target means mob-spawn draws precede the cast —
  those are already stream-aligned per the combat-swing scenario, but if you see an
  unexplained divergence, flag it against **DP-1170**, the open steal-from-mob draw
  investigation, rather than assuming your cast code is wrong.)
- **cure light:** fresh L1 cleric; cast on self (`cast 'cure light'` or `cast 'cure light'
  <self>`). At full HP the heal caps but the dice still roll — diff the message + next RNG.
- **infravision:** fresh L1 mage, `cast infravision`, repeated (re-affect message). No mob.

Each must be RED on pre-fix `main` and GREEN after. Claude gates every one.

## Guardrails
- **Never** edit anything under `darkpawns-c-oracle/` — reference only.
- Branch off `main`; keep the PR focused (magic missile + cure light, then infravision, is
  a fine split).
- Draw-count/-order parity is the acceptance bar. When unsure whether C draws in a branch,
  grep the C and match it — the cast-success `number()` and every effect die count.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.
- The class prime-stat char-creation flow already exists; a fresh L1 mage/cleric is created
  the same way the thief scenarios do (creation letter `M` = mage, `C` = cleric at the
  class prompt).

## Deliverable
Faithful `magic missile`, `cure light`, `infravision` (cast pipeline draws + effect
formulas + messages + mana/wait), with unit tests, plus the per-spell oracle scenario
sketches. Claude reconciles + runs the oracle gate.
