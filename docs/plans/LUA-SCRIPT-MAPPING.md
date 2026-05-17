# Lua Script Deployment — Vnum Mapping

**Date:** 2026-05-16
**Source of truth:** `src/spec_assign.c` (C ASSIGNMOB calls) + `test_scripts/mob/archive/` (Lua scripts)
**Status:** Planning

## How Script Assignment Works

1. Mob files (`lib/world/mob/<zone>.mob`) have an optional `Script:` E-spec field
2. Format: `Script: <filename> <trigger_flags>`
3. Engine loads from `lib/world/scripts/mob/<filename>`
4. Trigger flags are a bitmask (C source values, `src/structs.h`):
   - bribe=2, greet=4, ongive=8, sound=16, death=32
   - onpulse_all=64, onpulse_pc=128, fight=256, oncmd=512

> ⚠️ **DP-166:** The Go engine has shifted bit values (bribe=1, greet=2, etc.).
> Must fix DP-166 before deploying scripts. All flag values below use C source values.
> After DP-166 fix, Go and C values will match.

## Current State

Only 4 mobs have Script: fields:
- Zone 122: `Script: 122/healer.lua 24` (sound+ongive)
- Zone 144: `Script: 144/hisc.lua 512` (oncmd)
- Zone 212: `Script: 212/blacksmith.lua 24` (sound+ongive)
- Zone 212: `Script: 212/highpriest.lua 24` (sound+ongive)

## C Spec_procs → Lua Script Mapping

The original C codebase had ASSIGNMOB calls in `spec_assign.c`. Lua scripts replaced some of these. The mapping below shows which mobs should get which Lua scripts.

### Combat AI Scripts (generic — many mobs share these)

#### fighter.lua (melee combat AI)
Mobs from ASSIGNMOB(*, fighter):
```
12111, 12850, 14407, 20002, 20011, 20019, 20020, 20036, 20042,
4914, 5200, 7901, 7902
```
Trigger: `fight` (flag 1)
Script path: `mob/fighter.lua`

#### magic_user.lua (spellcaster AI)
Mobs from ASSIGNMOB(*, magic_user):
```
11005, 11006, 11007, 11008, 11016, 11024, 11029, 11030,
11706, 12848, 1307, 1315, 14219, 14221, 14306, 14314,
14406, 14435, 15804, 15807, 18601, 18603, 18604, 19412,
20003, 20008, 20009, 20010, 20014, 20023, 20025, 20026,
20030, 2108, 2716, 2720, 2732, 2733, 2734, 3103, 3113,
3118, 4101, 4202, 4704, 4916, 4919, 5503, 5507, 7023,
7969, 8081, 9903, 9905, 9911
```
Trigger: `fight` (flag 1)
Script path: `mob/magic_user.lua`

#### cleric.lua (healing/cursing AI)
Mobs from ASSIGNMOB(*, cleric):
```
11023, 11024, 11038, 11039, 12877, 1305, 14202, 14206,
14220, 14309, 20005, 20018, 20029, 20035, 20041, 21221,
4102, 7970
```
Trigger: `fight` (flag 1)
Script path: `mob/cleric.lua`

#### sorcery.lua (advanced magic AI)
Level-based split from magic_user mobs. Mobs level 30+ get sorcery.lua:
```
1315 (37), 20026 (35), 2720 (35), 7023 (35), 11706 (34), 14406 (34),
2716 (34), 2734 (34), 11016 (33), 1307 (33), 2733 (33), 7969 (33),
9905 (33), 11005 (32), 11006 (32), 11007 (32), 11008 (32), 20008 (32),
2732 (32), 14435 (31), 15804 (31), 20023 (31), 11030 (30), 4704 (30),
4916 (30), 4919 (30)
```
Trigger: `fight` (flag 1)
Script path: `mob/sorcery.lua`

### Functional Scripts (replace specific C spec_procs)

| Lua Script | Replaces C spec_proc | Mobs | Trigger flags |
|------------|---------------------|------|---------------|
| `no_move.lua` | no_move_east/west/north/south | 12876, 14410, 16308, 14421, 2106 | oncmd (512) |
| `cityguard.lua` | cityguard | 18215, 21200-21203, 21227, 21228, 2747 | fight (256), onpulse_pc (128) |
| `take_jail.lua` | take_to_jail | 8001, 8002, 8020, 8027, 8059 | oncmd (512) |
| `breed_killer.lua` | breed_killer | 7900, 7910 | fight (256) |
| `dracula.lua` | dracula | 14110, 7903 | fight (256), oncmd (512) |
| `dragon_breath.lua` | dragon_breath | 11000, 11001, 11002, 20027, 4209, 4705 | fight (256) |
| `mindflayer.lua` | mindflayer | 14414 | fight (256), oncmd (512) |
| `brain_eater.lua` | brain_eater | 14420, 14432 | onpulse_all (64) |
| `eq_thief.lua` | eq_thief | 12118, 14225, 7979 | onpulse_pc (128) |
| `beholder.lua` | beholder | 11023 | fight (256), oncmd (512), onpulse_pc (128) |
| `medusa.lua` | medusa | 14101, 14102 | fight (256) |
| `werewolf.lua` | werewolf | 5510 | fight (256) |
| `conjured.lua` | conjured | 81-86 | fight (256) |
| `rescuer.lua` | rescuer | 15808, 7909 | fight (256) |
| `backstabber.lua` | backstabber | 9151 | fight (256) |
| `teleporter.lua` | teleporter | 14411 | oncmd (512) |
| `teleport_vict.lua` | teleport_victim | 14405 | fight (256) |
| `never_die.lua` | never_die | 14401, 19119 | onpulse_all (64) |
| `no_get.lua` | no_get | 14416, 14430 | oncmd (512) |
| `troll.lua` | troll | 10029, 14100, 14311, 14312, 19900, 19901 | fight (256) |
| `snake.lua` | snake | 14103, 14127, 14415 | fight (256) |
| `paladin.lua` | paladin | 71, 7915 | fight (256) |
| `hisc.lua` | hisc (commented out in C) | 14412 | oncmd (512) |
| `identifier.lua` | identifier | 8087 | oncmd (512) |
| `jailguard.lua` | jailguard | 8088 | oncmd (512) |
| `mercenary.lua` | thief (partial) | 12127, 18218, 20004, 21242, 3119 | fight (256) |
| `strike.lua` | — | TBD | fight (256) |

### Lua-Only Scripts (no C spec_proc equivalent)

These scripts provide behavior that was never in C:

| Script | Purpose | Trigger | Needs vnum assignment |
|--------|---------|---------|----------------------|
| `assembler.lua` | Crafting base pattern | ongive (8) | Any crafting mob |
| `blacksmith.lua` | Weapon/armor assembly | sound (16), ongive (8) | 212 blacksmith area |
| `highpriest.lua` | Spell training services | sound (16), ongive (8) | 212 high priest |
| `healer.lua` | Healing services | sound (16), ongive (8) | 122 healer area |
| `banker.lua` | Starting gold via bonds | sound (16), ongive (8) | TBD newbie area |
| `clerk.lua` | Starting gear | sound (16), ongive (8) | TBD newbie area |
| `creation.lua` | Character creation quiz | oncmd (512) | TBD newbie area |
| `guard_captain.lua` | City guard captain | fight (256), oncmd (512) | TBD city |
| `farmer_wheat.lua` | Wheat farming chain | ongive (8) | TBD crafting zone |
| `miller.lua` | Wheat → flour | ongive (8) | TBD crafting zone |
| `baker_flour.lua` | Flour → dough | ongive (8) | TBD crafting zone |
| `baker_dough.lua` | Dough → bread | ongive (8) | TBD crafting zone |
| `bane.lua` | Daemon ceremony P1 | fight (256), death (32), greet (4), onpulse_pc (128) | TBD quest zone |
| `valoran.lua` | Daemon ceremony P2 | fight (256), death (32), greet (4), onpulse_pc (128) | TBD quest zone |
| `shopkeeper.lua` | Enhanced shop behavior | ongive (8), sound (16) | TBD shops |
| `enchanter.lua` | Weapon enchanting | sound (16), ongive (8) | TBD |
| `crystal_forger.lua` | Crystal armor exchange | oncmd (512) | TBD |
| `dragon_forger.lua` | Dragon scale armor exchange | oncmd (512) | TBD |
| `donation.lua` | Donation room cleanup | onpulse_all (64) | TBD donation rooms |
| `beggar.lua` | Ambient begging | sound (16) | TBD cities |
| `citizen.lua` | Ambient citizen | sound (16), bribe (2) | TBD cities |
| `carpenter.lua` | Ambient carpenter | sound (16) | TBD |
| `bearcub.lua` | Wanders to mama | onpulse_all (64) | TBD forest |
| `mount.lua` | Mount behavior | TBD | TBD |
| `stable.lua` | Stable behavior | oncmd (512) | TBD cities |
| `petitioner.lua` | Petition behavior | TBD | TBD |
| `hermit.lua` | Hermit behavior | TBD | TBD |
| `minstrel.lua` | Minstrel behavior | sound (16) | TBD |
| `singingdrunk.lua` | Drunk singing | sound (16) | TBD taverns |
| `towncrier.lua` | Town crier | sound (16) | TBD cities |
| `mime.lua` | Mime behavior | TBD | TBD |
| `porcupine.lua` | Porcupine combat | fight (256) | TBD |
| `phoenix.lua` | Phoenix behavior | TBD | TBD |
| `griffin.lua` | Griffin behavior | fight (256) | TBD |
| `ettin.lua` | Ettin combat | fight (256) | TBD |
| `troll.lua` | Troll combat (already mapped above) | — | — |
| `sandstorm.lua` | Environmental hazard | onpulse_all (64) | TBD desert zones |
| `weatherworker.lua` | Weather behavior | TBD | TBD |
| `golem_from_crate.lua` | Golem creation | oncmd (512) | TBD |
| `golem_miner.lua` | Golem mining | TBD | TBD |
| `golem_to_crate.lua` | Golem storage | oncmd (512) | TBD |
| `memory_moss.lua` | Memory moss | onpulse_all (64) | TBD |
| `fire_ant.lua` | Fire ant combat | fight (256) | TBD |
| `fire_ant_larva.lua` | Fire ant larva | TBD | TBD |
| `remove_curse.lua` | Curse removal | ongive (8) | TBD |
| `paralyse.lua` | Paralysis behavior | TBD | TBD |
| `triflower.lua` | Tri-flower | TBD | TBD |
| `tyr.lua` | Tyr behavior | TBD | TBD |
| `keep_sorcerer.lua` | Keep sorcerer | fight (256) | TBD |
| `seiji.lua` | Seiji behavior | TBD | TBD |
| `ki_kuroda.lua` | Aki Kuroda | TBD | TBD |
| `shop_give.lua` | Shop give behavior | ongive (8) | TBD |
| `merchant_inn.lua` | Inn merchant | sound (16), ongive (8) | TBD |
| `merchant_walk.lua` | Walking merchant | TBD | TBD |
| `head_shrinker.lua` | Head shrinker | TBD | TBD |
| `thornslinger.lua` | Thornslinger combat | fight (256) | TBD |
| `bradle.lua` | Bradle combat | fight (256) | TBD |
| `caerroil.lua` | Caerroil combat | fight (256) | TBD |
| `cuchi.lua` | Cuchi behavior | oncmd (512) | 18306 |
| `souleater.lua` | Soul eater | TBD | TBD |
| `aversin.lua` | Aversin behavior | TBD | TBD |
| `prisoner.lua` | Prisoner behavior | TBD | TBD |
| `zealot.lua` | Zealot behavior | TBD | TBD |
| `quanlo.lua` | Quan Lo | TBD | 19405 |
| `warg.lua` | Warg combat | fight (256) | TBD |
| `neckbreak.lua` | Neck break | TBD | TBD |
| `autodraw.lua` | Lottery (SHELVE) | — | Skip |

## Deployment Strategy

### Phase 1: Core Engine
1. Deploy `globals.lua` → `lib/world/scripts/globals.lua`
2. Deploy `mob/no_move.lua` → `lib/world/scripts/mob/no_move.lua`
3. Deploy `mob/assembler.lua` → `lib/world/scripts/mob/assembler.lua`

### Phase 2: Combat AI Matrices
1. Deploy `mob/fighter.lua`, `mob/magic_user.lua`, `mob/cleric.lua`, `mob/sorcery.lua`
2. Update mob world files for all ASSIGNMOB mobs listed above
3. Set trigger flags appropriately (fight=1 for combat AI)

### Phase 3: Functional Scripts
Deploy scripts that replace C spec_procs, update corresponding mob world files.

### Phase 4: Lua-Only Scripts
Deploy scripts without C equivalents. Vnums assigned per table above.

### Phase 5: Remaining RESTORE Scripts (92)
High-value quest/mob scripts — each needs vnum lookup and trigger assignment.

### Phase 6: Ambient/Candidate Scripts (41)
Low-priority flavor scripts.

## Resolved Vnum Assignments

### Key Lua-Only Script Mobs

| Script | Mob Vnums | Zone | Level |
|--------|-----------|------|-------|
| `banker.lua` | 8007 | 80 | — |
| `clerk.lua` | 18210, 18228 | 182 | 34 |
| `guard_captain.lua` | 21202 | 212 | 31 |
| `cityguard.lua` | 18215, 21200, 21201, 21203, 21227, 21228, 2747 | various | 11-31 |
| `creation.lua` | TBD — likely oncmd trigger on a guild/reception mob |
| `take_jail.lua` | 8001, 8002, 8020, 8027, 8059 | 80 | 16-31 |
| `breed_killer.lua` | 7900, 7910 | 79 | 30-31 |
| `blacksmith.lua` | 212 area crafting mobs | 212 | — |
| `highpriest.lua` | 21221 (high priest) | 212 | 30 |
| `healer.lua` | 122 area mobs | 122 | — |

### Sorcery vs Magic_user Split

Threshold: **level 30**. Mobs level 30+ get `sorcery.lua`, below get `magic_user.lua`.
- 26 mobs → sorcery.lua
- 29 mobs → magic_user.lua

## Remaining Open Questions

1. **creation.lua** — Need to identify which mob runs the character creation quiz (likely a reception/guild mob in newbie zone)
2. **Crafting chain vnums** — farmer_wheat, miller, baker_flour, baker_dough need zone assignments
3. **Trigger flag verification** — Confirm flag values against Go engine constants (sound=8, ongive=16, oncmd=512, fight=1, onpulse_all=4, onpulse_pc=2, death=32, greet=64, bribe=128)
4. **Script path format** — Confirm engine uses `mob/<script_name>` (not `mob/archive/<script_name>`)
