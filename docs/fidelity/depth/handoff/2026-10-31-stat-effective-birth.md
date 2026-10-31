# Handoff — `stat <player>` reads the affected birth (2026-10-31)

**Status: fixed and proven; one oracle scenario is blocked by a second, unrelated
divergence.**

## The fix

`stat <player>` printed the raw `p.Birth` for both the `Created:` stamp and
`Age [%d]` (`pkg/session/wiz_stats.go:279-280`). C prints `k->player.time.birth`
for both, and `affect_modify` has already shifted that field in place for
`APPLY_AGE`:

- `src/handler.c:160-161` — `case APPLY_AGE: ch->player.time.birth -= (mod * SECS_PER_MUD_YEAR);`
- `src/act.wizard.c:754` — `asctime(localtime(&(k->player.time.birth)))` (the Created stamp)
- `src/act.wizard.c:764` — `age(k).year`, i.e. `mud_time_passed(time(0), ch->player.time.birth)` (`src/utils.c:333`)

Both lines now read `p.EffectiveBirth()` (`pkg/game/player_physical.go:78`), which
is the same `Birth - birthShift` arithmetic `GetAgeParts` already used for
`score` (`pkg/game/player_stats.go:202-208`). `EffectiveBirth` had no callers
before this change.

**Save stays on the raw field, and that is parity** (`pkg/game/save.go:158` writes
`p.Birth`). C's `char_to_store` (`src/db.c:2531`) strips the character before it
copies anything: it unequips every slot (`src/db.c:2537-2544`, and `unequip_char`
removes those objects' affects) and clears the tattoo affects
(`tattoo_af(ch, FALSE)`, `src/db.c:2587`), so `st->birth = ch->player.time.birth`
(`src/db.c:2589`) copies the **raw** timestamp. The port's raw save matches.

## Proof

`pkg/session/stat_effective_birth_test.go` —
`TestStatPlayerAgeAndCreatedUseTheAffectedBirth` puts an `APPLY_AGE` item on a
character and asserts both stat lines. On `origin/main` it fails:

```
stat age line = "...Age [718]...", want "Played [0d 0h], Age [723]" (raw birth would give 718)
```

and it passes on the fix. The `Created:` stamp moves with it (2006 → 2001 in the
failing output).

## The oracle scenario

`cmd/dp-oracle-diff/scenarios/stat-effective-birth-depth.txt` sets up a God and a
magic-user peer, then stats the peer before, during and after wearing the robes:
red at `970e4ad1b` (C prints `Played [0d 0h], Age [27]`, the port `Age [17]`) and
CLEAN with the fix (`stat-age-rebased`, 3/3 with `wizard-stat-color-depth` and
`score-state-depth`).

An earlier version of the scenario wore ring 8077 instead. That run was red on a
divergence unrelated to this fix, in the same report:

```
-Racial Hatreds: Troll None None None None
+Racial Hatreds: None None None None None
```

The port hardcodes that line — `pkg/session/wiz_stats.go:357` sends the literal
`"None None None None None"` — and never reads `Player.RaceHates`
(`pkg/game/player.go:100-102`, filled from worn items at
`pkg/game/effective_attributes.go:65`). C builds it from the target's own slots:

```c
for (i = 0; i < 5; i++)
  mob_race[i] = mob_race_name(GET_RACE_HATE(k, i));   /* src/act.wizard.c:891-894 */
```

### v0.1.9 item: equipment APPLY_RACE_HATE is not shown in player stat

Repro: a character wearing 8077, "a ring of holy fire", whose `A 25 6` is
APPLY_RACE_HATE for race 6 (Troll, `src/structs.h:524`). C prints
`Racial Hatreds: Troll None None None None`; the port prints `None` five times.
The port has both the field and the data — the line simply ignores them, so this
is a rendering gap rather than a missing port. **Not fixed here.** It is in
neither `expected_divergence_pins.tsv` nor `expected_divergences.tsv`, so it cannot
be pinned as EXPECTED either.

The scenario below avoids it by wearing the robes, which carry no race hate.

## Scenario fixtures worth remembering

- **The robes at 19404 are wearable — by a mage or a cleric.** Their `A 9 10` is
  APPLY_AGE +10 (`src/structs.h:508`). The `72925184` in the extra-flags slot sets
  anti-class bits 14, 15, 19, 20, 22 and 26 — thief, warrior, ninja, paladin,
  assassin, ranger (`src/structs.h:482-493`) — and nothing else, so mage (12) and
  cleric (13) are both accepted. An earlier version of this note claimed
  `invalid_class` (`src/handler.c:714-717`) refused them for every character; that
  refusal was real but class-specific, because the peer was a warrior.
- The ring **8077** is `A 12 5` (APPLY_MANA) plus `A 25 6` (APPLY_RACE_HATE,
  Troll). It has **no age effect at all**, so a run against it can only ever show
  the Racial Hatreds divergence — which an earlier version of this note misread as
  an age test.
- Object-affect **location 9 is APPLY_AGE** (`pkg/engine/affect.go:195`,
  `src/structs.h:508`); 12 is APPLY_MANA (`:511`) and 25 is APPLY_RACE_HATE
  (`:524`). The `A` lines follow the values line, so read the whole record.
- `spawn-obj <vnum> <max> <room> <zone>` appends an `O 0 <vnum> <max> <room>`
  reset to **`<zone>.zon`** (`cmd/dp-oracle-diff/main.go:1387-1412`): the zone
  argument is the reset's zone, and a pre-existing reset for the same vnum
  satisfies a low `max`, so the object never appears. Use a large `max`.
- `stat`'s `Created:` stamp is normalised to `<WALL_CLOCK>`, so the Age line is the
  only observable of an APPLY_AGE shift in a scenario.
- Character creation answers are **sex, race, class** (`src/interpreter.c:2012`,
  `:2035`, `:2081`): `N F H M K Y <ENTER> 1` is a female Human magic-user from
  Kir Drax'in. Class letters are C/T/W/M/i (`pkg/game/class_tables.go:50`).

```text
# stat-effective-birth-depth.txt
[fixture]
empty-players
quiet-mobs
spawn-obj 19404 99 8162 194

[setup:oracle]        # God, then the peer, in both engines
Ageconfig / Y / oraclepass / oraclepass / N / M / H / W / K / Y / <ENTER> / 1
[setup:oracle:peer]   # female Human magic-user — the robes are mage-wearable
Agepeer   / Y / oraclepass / oraclepass / N / F / H / M / K / Y / <ENTER> / 1
[setup:port] / [setup:port:peer]   # same, lowercase y

[warmup]
goto 8162
teleport Agepeer 8162

[probe]
stat player Agepeer
force Agepeer get robes
force Agepeer wear robes
stat player Agepeer
force Agepeer remove robes
stat player Agepeer
```
