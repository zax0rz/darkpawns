# Special-procedure scout — 2026-08-27

This is the first depth inventory for the mob special-procedure family. The C
oracle remains authoritative; this note records the source census and the Go
dispatch seam without treating a registry smoke test as behavioral proof.

## C source census

The three `src/spec_procs*.c` files contain 113 `SPECIAL` definitions:

| file | definitions |
| --- | ---: |
| `spec_procs.c` | 41 |
| `spec_procs2.c` | 43 |
| `spec_procs3.c` | 29 |
| **total** | **113** |

The lexical trigger census is overlapping (one procedure can have several
trigger classes):

| trigger class | count | classification rule |
| --- | ---: | --- |
| fight | 45 | `FIGHTING`, `POS_FIGHTING`, or `GET_MOB_WAIT` guard |
| greet/entry | 26 | explicit `enter`, `char_to_room`, or `perform_move` path |
| percent/random | 34 | `number()` draw in the procedure |
| timer/clock | 4 | `GET_MOB_WAIT`, `time_info`, or mount rent-time logic |
| named command/direction | 57 | `CMD_IS`, `IS_MOVE`, `switch(cmd)`, or `strcmp(cmd, ...)` |
| pulse/no-command gate | 72 | `!cmd`, `cmd ||`, or equivalent autonomous-tick gate |

There is no separately named `greet` callback in these three files; the
greet/entry count covers the explicit arrival/entry mechanics. The source
also contains two pulse-only helper bodies with no command, fight, random, or
clock token (`elemental_room`, `conjured`, `elements_minion`, and `never_die`
are the four no-trigger-token bodies after comments/string stripping; their
behavior is still pulse or state driven).

The source assignment inventory has 233 active `ASSIGNMOB` calls, 228 unique
mob VNums, and 66 unique final mob-procedure names. Five VNums are assigned
twice in C; the later assignment wins at boot:

| VNum | earlier | final | source lines |
| ---: | --- | --- | --- |
| 8014 | `guild_guard` | `guild` | `spec_assign.c:271,280` |
| 11023 | `cleric` | `cleric` | `spec_assign.c:321,334` |
| 11024 | `magic_user` | `cleric` | `spec_assign.c:322,335` |
| 11029 | `magic_user` | `magic_user` | `spec_assign.c:323,336` |
| 11030 | `magic_user` | `magic_user` | `spec_assign.c:324,337` |

The complete definition enumeration is:

- `spec_procs.c` (41): `guild`, `dump`, `snake`, `summoner`, `thief`,
  `magic_user`, `fighter`, `paladin`, `guild_guard`, `puff`, `fido`,
  `janitor`, `cityguard`, `mayor`, `dragon_breath`, `citizen`, `cuchi`,
  `mini_thief`, `black_undead_knight`, `red_undead_knight`, `mickey`,
  `mallory`, `cleric`, `conductor`, `brass_dragon`, `outofjailguard`,
  `jailguard`, `dracula`, `pet_shops`, `enter_circle`, `elevator`,
  `elemental_room`, `pray_for_items`, `fearface`, `start_room`,
  `newbie_zone_entrance`, `suck_in`, `oro_quarters_room`, `oro_study_room`,
  `bank`, `horn`.
- `spec_procs2.c` (43): `normal_checker`, `ninelives`, `whirlpool`, `couch`,
  `stableboy`, `tipster`, `rescuer`, `pissedalchemist`, `remorter`,
  `assassin`, `tattoo1`, `tattoo2`, `tattoo3`, `eviltrade`, `identifier`,
  `tattoo4`, `evillead`, `little_boy`, `ira`, `take_to_jail`, `jail`,
  `medusa`, `eq_thief`, `portal_room`, `breed_killer`, `carrion`, `bat_room`,
  `bat`, `no_move_east`, `key_seller`, `castle_guard_east`, `mindflayer`,
  `backstabber`, `teleporter`, `no_move_west`, `no_move_north`, `never_die`,
  `no_move_south`, `chosen_guard`, `castle_guard_down`, `castle_guard_up`,
  `castle_guard_north`, `wall_guard_ns`.
- `spec_procs3.c` (29): `clerk`, `butler`, `brain_eater`, `teleport_victim`,
  `con_seller`, `no_move_down`, `troll`, `quan_lo`, `alien_elevator`,
  `werewolf`, `field_object`, `portal_to_temple`, `turn_undead`, `itoh`,
  `mirror`, `prostitute`, `roach`, `mortician`, `conjured`, `hisc`,
  `recruiter`, `elements_master_column`, `elements_platforms`,
  `elements_load_cylinders`, `elements_galeru_column`,
  `elements_galeru_alive`, `elements_minion`, `elements_guardian`,
  `fly_exit_up`.

## Go dispatch inventory

`getMobVNumSpec` in `pkg/game/mobact.go:103-110` resolves
`MobSpecAssign[vnum]` through `SpecRegistry`. Autonomous mobile activity uses
the same lookup at `pkg/game/mobact.go:190-194`; player command dispatch uses
the equivalent `GetMobSpec` lookup in `pkg/session/commands.go:618-629`.

The Go inventory has 228 `MobSpecAssign` entries and 66 unique mob-procedure
names. All 66 assigned names are present in `SpecRegistry`; the registry has
121 total entries across mob, object, and room procedures. The C-final
assignment comparison and a unit guard cover the two mapping drifts found by
this scout (`8014 → guild`, `11024 → cleric`); `src/` was not edited.

## Live depth proof

`docs/fidelity/depth/spec-procs.tsv` and
`cmd/dp-oracle-diff/scenarios/spec-proc-movement.txt` prove the two simplest
deterministic command arms:

- mob `2106` → `no_move_west`
- mob `14410` → `no_move_east`

The RED vehicle on clean `main` showed Go's invented “heavy object” bytes
against C's act pair. The GREEN vehicle now matches C for seeds 1, 2, 3, 5,
and 8, including the registered-VNum dispatch. The remaining fight/greet,
percent, and timer-driven procedures require their own vehicles; no deep
engine backlog row was opened in this scout.
