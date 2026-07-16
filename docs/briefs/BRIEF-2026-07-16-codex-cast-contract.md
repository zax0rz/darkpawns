# BRIEF (codex) — faithful `cast` target/eligibility/gating contract (DP-1117, O39)

**Owner:** codex (frontier). **Gate:** Claude establishes the oracle RED and runs red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** This is a meaty one — one focused PR (or a small stacked pair: parse/gate first, mana/roll second).

## Why this is the frontier task
`do_cast` (`src/spell_parser.c:916-1113`) is the whole player-facing spell entry contract: quoted-name parsing, class/skill eligibility, peaceful gating, target resolution across six scopes driven by per-spell `TAR_*` flags, self-cast rules, and — critically — **mana is checked *before* the success roll and deducted *after* it** (half on failure, full on success). Go's `cast_cmds.go` (`pkg/session/cast_cmds.go:153-275`) shortcuts almost all of this: unquoted first word, `SpellMap`-only lookup, defaults no-target to **self**, resolves only an in-room char, and **pre-deducts mana before** `spells.Cast`. The good news: the per-spell target-flag table **already exists** in Go (`pkg/spells/affect_spells.go` etc. via `setupSpellInfo(..., TarCharRoom|TarFightVict)`), matching C's `spello()`. The job is to route `do_cast` through it.

## Read-only source of truth
C: `~/.openclaw/workspace/darkpawns-c-oracle/src/spell_parser.c` — `do_cast` (916-1113), `mag_manacost` (257), the `spello()` table (~1280-1370), and `src/spells.h` `TAR_*` (294-304). **Never edit the oracle tree.**
Go: `pkg/session/cast_cmds.go`, `pkg/spells/` (`SpellInfo`, `TarCharRoom` etc., `setupSpellInfo`, `Cast`, `CallMagic`), `pkg/game/class_spells.go`.

## The C contract — exact order, messages are first-class
For the ordinary `cast` (non-psionic/non-mystic) path. **Reproduce this order precisely; every string is fidelity.**

1. NPC → return. (Psionic "will" wording is a separate class — **out of scope**; keep the `cast`/spell wording. Leave psionic/mystic message variants for a follow-up unless trivially parallel.)
2. **Parse `'...'`** via `strtok(argument, "'")`:
   - no opening `'` (empty arg) → `"Cast what where?\r\n"`
   - opening but no closing `'` → `"Spell names must be enclosed in the magick symbols: '\r\n"`
   - `t` = everything after the closing `'` = the target string.
3. `find_skill_num(s)` → spellnum. `< 1 || > MAX_SPELLS` → `"Cast what?!?\r\n"`
4. `GET_LEVEL(ch) < min_level[class]` → `"You do not know that spell!\r\n"`
5. `GET_SKILL(ch, spellnum) == 0` → `"You are unfamiliar with that spell.\r\n"`
6. `ROOM_PEACEFUL && SINFO.violent` → `"This room just has such a peaceful, easy feeling..\r\n"`
7. **Target resolution** by `SINFO.targets` (the Go `SpellInfo.Targets` flags):
   - `TAR_IGNORE` → target found (no lookup).
   - **target string non-empty** → try in this exact order, first hit wins: `TAR_CHAR_ROOM` (`get_char_room_vis`) → `TAR_CHAR_WORLD` (`get_char_vis`) → `TAR_OBJ_INV` → `TAR_OBJ_EQUIP` → `TAR_OBJ_ROOM` → `TAR_OBJ_WORLD`.
   - **target string empty** → `TAR_FIGHT_SELF` (self, if fighting) → `TAR_FIGHT_VICT` (opponent, if fighting) → `TAR_CHAR_ROOM && !violent` → **default to self** → else `"Upon who should the spell be cast?\r\n"` (or `"...what..."` if the spell targets objects: `TAR_OBJ_ROOM|TAR_OBJ_INV|TAR_OBJ_WORLD`).
8. `target && tch==ch && violent` → `"You shouldn't cast that on yourself -- could be bad for your health!\r\n"`
9. `!target` (fell through) → `send OK` + `say_spell(...)` + `"Cannot find the target of your spell!\r\n"` (edge case; preserve it).
10. **Mana CHECK (do not deduct):** `mana = mag_manacost(ch, spellnum)`; if `mana>0 && GET_MANA<mana && level<IMMORT` → `"You haven't the energy to cast that spell!\r\n"` and return. **No mana spent.**
11. `weight_add` from carry ratio (spell_parser.c:1080-1096 — port the exact ladder: `CAN_CARRY_W/IS_CARRYING_W`; ≥4→0, 3→5, 2→7, 1→10, −1→0; clamp `MAX(0,...)`; immort → −20).
12. **The success roll — this is the ONLY RNG draw in the whole path:** `if (number(0, 101+weight_add) > GET_SKILL(ch, spellnum))` → **failure**:
    - `WAIT_STATE(ch, PULSE_VIOLENCE)`,
    - `skill_message(0,...)` or `"You lost your concentration!\r\n"`,
    - `if (mana>0) GET_MANA -= (mana >> 1)` (**half** mana),
    - `if (violent && tch && IS_NPC(tch) && !FIGHTING(tch)) hit(tch, ch, ...)`.
    - **else success:** `cast_spell(...)` → on 1: `WAIT_STATE(PULSE_VIOLENCE)` + `GET_MANA -= mana` (**full**).

### DRAW-PARITY LAW (do not get this wrong)
Every gating failure in steps 2-10 draws **zero** PRNG and applies **zero** mana/wait. The single `number(0, 101+weight_add)` at step 12 is the **only** draw, and it happens **after** all gating. The current Go code's pre-deduct + early self-target changes both the mana math and (once effects roll) the stream. Match C: gate first with no draws, then one roll.

## The Go fix (shape, not prescription)
Rebuild `cast_cmds.go`'s handler as a faithful `do_cast`: quoted parse → `find_skill_num` equivalent → eligibility (class min-level, skill%>0) → peaceful gate → target resolution driven by `SpellInfo.Targets` (reuse the existing flags + add the world/obj/equip lookups) → self-cast gate → mana check (no deduct) → weight_add → single success roll → `spells.Cast`/`CallMagic` on success, half-mana + optional retaliation on failure. Keep `spells.Cast`/`CallMagic` as the effect layer; the fix is the *front half* (parse/gate/target/mana ordering) around it.

## Oracle RED (Claude establishes + gates)
Fresh L1 **mage** (class `M`). Structural, RNG-free probes (tier-1) that gate on messages only:
- `cast` → `Cast what where?`
- `cast flame arrow` (unquoted) → `Spell names must be enclosed in the magick symbols: '`
- `cast 'flame arrow'` (no target, violent, not fighting) → `Upon who should the spell be cast?`  ← Go currently defaults to self
- `cast 'bogusspell'` → `Cast what?!?`
- `cast 'cure light'` (mage doesn't know it — cleric spell) → `You do not know that spell!`
- (self-cast) `cast 'flame arrow' <ownname>` → `You shouldn't cast that on yourself...`
These draw no RNG, so they're clean red→green. (The actual success/failure roll is RNG and excluded from the tier-1 gate per DP-1117.) Claude writes/owns the scenario; note expected-greens in the PR.

## Out of scope
- Psionic/mystic "will" wording + classes (separate).
- Spell **effects** (`CallMagic`/`cast_spell` internals) — already scaffolded; don't rewrite them, just call them at the right point with the right target/mana ordering.
- `mag_manacost` formula changes if it already matches C (verify it does; cite if you touch it).

## Tests you own (golden/table, deterministic)
- Table-test the parse/gate branches → exact message + "no RNG drawn, no mana spent" for each early return (use a `dprng` draw-count check or the seam pattern from #374/#375).
- Target-resolution order: with a spell flagged `TAR_CHAR_ROOM|TAR_CHAR_WORLD`, a room match wins over a world match; empty-target non-violent defaults to self; empty-target violent → "Upon who".
- Mana: failure roll spends `mana>>1`, success spends `mana`, insufficient-mana gate spends 0. Use `dprng.ResetStream` to force the roll outcome.

## PR hygiene
- Commits end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- PR body ends with: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`
