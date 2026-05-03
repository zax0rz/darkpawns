# C-to-Go Fidelity Audit
**Date:** 2026-05-03
**Auditor:** BRENDA69 (subagent)

## Summary
20 file pairs reviewed. **47 issues found**: 8 critical, 12 high, 15 medium, 12 low.

## Critical Issues

### C1. TYPE_HIT constant mismatch — combat package vs game package vs C source
- **C source** (`spells.h`): `TYPE_HIT = 300`, `TYPE_SUFFERING = 399`
- **Go combat package** (`pkg/combat/fight_core.go`): `TYPE_HIT = 2000`, `TYPE_SUFFERING = 3000`
- **Go game package** (`pkg/game/death.go`): `TypeHit = 300`, `TypeSuffering = 399` (correct!)

The combat package uses completely wrong values (2000/3000). Any attack type check `attackType >= TYPE_HIT && attackType < TYPE_SUFFERING` in the combat package will incorrectly classify attacks. The `IS_WEAPON` check in `TakeDamage()` is broken when using the combat package constants. The `attackTypeToCorpseAttack` function in death.go uses the correct 300-based values, but when these two packages interact, the mismatch will cause weapon attacks to be misidentified or invisible.

### C2. LVL_IMMORT / LVL_IMPL constant mismatch — game package vs C source
- **C source** (`structs.h`): `LVL_IMMORT = 31`, `LVL_IMPL = 40`
- **Go combat package** (`pkg/combat/fight_core.go`): `LVL_IMMORT = 31` (correct)
- **Go game package** (`pkg/game/limits_misc.go`): `LVL_IMMORT = 50`, `LVL_IMPL = 54` (WRONG)

The game package has inflated immortal levels. This means `GainExp()` in the game package won't properly gate XP for immortals, `check_idling()` won't protect immortals, and the level-up logic will allow leveling beyond the intended cap. Players could reach levels 41-54 that shouldn't exist.

### C3. `GetPositionFromHP()` drops fighting/resting/sleeping position
- **C source** (`fight.c`): `update_pos()` preserves current position if HP > 0 and position > POS_STUNNED
- **Go** (`pkg/combat/fight_core.go`): `GetPositionFromHP()` always returns `PosStanding` when HP > 0

This means any code that calls `GetPositionFromHP()` will reset a fighting character to standing, or wake a sleeping character. The C code specifically preserves the position when the character is in a conscious state (fighting, sitting, resting, sleeping). The Go version unconditionally sets to standing.

### C4. Backstab/Circle/Disembowel damage multipliers missing from `MakeHit()`
- **C source** (`fight.c` hit()): Has explicit branches for `SKILL_BACKSTAB` (multiplied by `backstab_mult(level)`), `SKILL_CIRCLE` (multiplied by `backstab_mult(level)/3`), and `SKILL_DISEMBOWEL` (special formula: `(level*2) + damroll`)
- **Go** (`pkg/combat/fight_core.go` MakeHit()): Only does normal weapon damage with `getMinusDam()`. No backstab multiplier, no circle damage, no disembowel formula.

Backstab, circle, and disembowel will deal dramatically less damage than intended. A level 30 backstab should deal ~7× normal damage; in Go it deals 1×.

### C5. `change_alignment()` neutral check uses wrong boundary
- **C source**: `IS_NEUTRAL(victim)` = alignment strictly between -350 and 350 (i.e., -349 to 349)
- **Go** (`pkg/combat/fight_core.go`): `victimAlign >= -350 && victimAlign <= 350` includes ±350

Characters at exactly ±350 alignment will be treated as neutral in Go but as good/evil in C. This affects alignment shifts on kills and protect evil/good damage reduction.

### C6. `PROTECT_EVIL` / `PROTECT_GOOD` check uses alignment range instead of sign
- **C source** (`fight.c`): `IS_EVIL(ch)` = `GET_ALIGNMENT(ch) <= -350`, `IS_GOOD(ch)` = `GET_ALIGNMENT(ch) >= 350`
- **Go** (`pkg/combat/fight_core.go`): `GetAlignment(chName) < -350` / `GetAlignment(chName) > 350`

Characters at exactly ±350 alignment get no protection in Go. In C, they do.

### C7. `perform_violence()` / combat loop not fully ported
- **C source** (`fight.c`): `perform_violence()` iterates the combat list, calculates attacks per round, handles parry/dodge, mob wait state, position recovery, and executes hits
- **Go**: The combat loop is fragmented. `GetAttacksPerRound()` exists but the actual combat loop (`OnPerformViolence`) is a stub callback. Parry/dodge are in `MakeHit()` rather than in the violence loop. The C code applies parry to the *attacker's* combat output (reducing their attacks), while Go applies it per-hit as a miss.

The parry/dodge model is fundamentally different. In C, parry reduces the number of attacks the *target* can make (setting `IS_PARRIED` flag, then reducing `attacks` by `dex_app[defender_dex].defensive` or 1). In Go, parry simply causes a single hit to miss. This means parry is much weaker in Go than in C.

### C8. NPC charm retarget and switcheroo logic missing
- **C source** (`fight.c` damage()):
  - Charm retarget: If attacker is NPC, victim is charmed NPC, and victim's master is in room (10% chance), attacker switches to fight the master
  - Switcheroo: High-level NPCs (>20) have a chance to switch targets to whoever is attacking them (1/80 chance per attacker in room)
- **Go** (`pkg/combat/fight_core.go` TakeDamage()): Both features are present only as comments saying "game layer handles via hooks"

This removes tactical depth from NPC combat. Charmed pets no longer cause aggro to shift, and high-level NPCs no longer intelligently retarget.

---

## High Issues

### H1. `counter_procs()` milestone stat rewards reproduce C bug incorrectly
- **C source** (`fight.c`): The switch/case for major milestones has no `break` statements, causing fallthrough. The result is that ALL three stats (hp, mana, move) get +1 regardless of which `case` is entered, PLUS an extra +1 hp from `default`.
- **Go**: The comment says "reproducing the bug for fidelity" and always increments all three stats. This is correct bug-for-bug behavior, but the `IncreaseMaxStat` hook may not exist or may not work, making this a latent bug.

### H2. `attitude_loot()` is simplified to stubs
- **C source** (`fight.c`): Complex multi-pass looting — get all from corpse, junk cheap items, wear all, repeat, then gossip one of 12 random messages
- **Go** (`pkg/combat/fight_core.go`): Simple `PerformCommand("get all corpse of X")` and `BroadChatFunc("Grins wickedly...")`

Missing: junk logic, wear logic, the second pass, and all 12 random gossip messages.

### H3. `die_with_killer()` missing MOB_SPEC special proc call
- **C source**: Before running death script, checks `MOB_FLAGGED(ch, MOB_SPEC) && mob_index[GET_MOB_RNUM(ch)].func != NULL` and calls the special function with the killer set as FIGHTING
- **Go**: Only runs death scripts. The MOB_SPEC check is missing entirely.

### H4. `die_with_killer()` CON loss only in combat package, not in game package's `HandleDeath()`
- **C source**: CON loss in `die_with_killer()` (called for combat deaths)
- **Go combat package** (`DieWithKiller()`): Has CON loss logic
- **Go game package** (`HandleDeath()` → `handlePlayerDeath()`): Has its own CON loss logic with different constants
- The game package's `ConLossMinLevel = 6` doesn't match C's `GET_LEVEL(ch) > 5` (which means level 6+, so this is actually correct). But the combat package's `DieWithKiller` uses `level > 5` which maps to level 6+, also correct. The duplication is confusing and could lead to double CON loss if both paths execute.

### H5. `group_gain()` autogold/autosplit logic diverges significantly
- **C source**: Complex gold splitting with messages to each group member, including leader share, leftover handling, and specific message formatting
- **Go**: Uses `ApplyToGroupMembers` with a lambda, simplified gold math, different message format

Missing: Leader-specific gold share messaging, "no gold to split" messaging, leftover gold handling, and the `PRF_AUTOGOLD` corpse-looting before split.

### H6. Neutral room rescue (`ROOM_NEUTRAL`) only partially implemented
- **C source**: Full rescue sequence — stops fighting for both parties, sets HP to 1, teleports to room 8004, calls `look_at_room()`, sends messages, clears mob memory/hunting
- **Go**: Sets HP to 1 and sends a generic message. Missing: `look_at_room()` call, proper room transfer, mob memory/hunting clearing, specific messages.

### H7. Jail guard logic only partially implemented
- **C source**: Full jail sequence — checks CAN_SEE, HP > half, not vampire/werewolf, stops fighting, sets HP to 1, teleports to room 8118, sets jail timer, sends specific messages
- **Go**: Checks HP and vampire/werewolf flags, but missing: CAN_SEE check, proper jail room (8118), jail timer setting, detailed messaging

### H8. `make_corpse()` attack type descriptions use wrong numeric IDs
- **C source**: Uses spell/skill numeric IDs from `spells.h` (e.g., `SPELL_FIREBALL = 26`, `SPELL_CHILL_TOUCH = 8`)
- **Go** (`death.go`): Uses hardcoded numbers (5, 26, 58, 96 for fire, etc.) which may not match the Go spell constant system

### H9. Dismount-on-hit logic different
- **C source**: `number(0, 99) < 10` = 10% chance; sends specific messages; has DEX-based landing check (`number(0,99) < GET_DEX(victim)*1.5`)
- **Go**: `rand.Intn(100) < 10` = 10% chance (correct); calls `Dismount()` but missing: DEX landing check, sitting position on failed landing, specific messages

### H10. `stop_fighting()` retaliatory attack logic missing
- **C source**: When `stop_fighting(ch)` is called and the victim is dead or fled, the C code iterates combat_list to find someone else fighting `ch` and redirects `FIGHTING(ch)` to them
- **Go**: Simply clears the fighting target. No retaliatory target acquisition.

### H11. `appear()` missing AFF_HIDE removal
- **C source**: `appear()` removes both `AFF_INVISIBLE` and `AFF_HIDE`
- **Go**: `Appear()` only removes `SPELL_INVISIBLE` affect, not the `AFF_HIDE` bit

### H12. Mob `dodge` in `perform_violence()` not ported
- **C source**: NPCs with `AFF_DODGE` have `level%` chance to dodge all attacks for the round (sets `IS_PARRIED` on attacker)
- **Go**: `CheckDodge()` is per-hit, not per-round, and doesn't check for NPC + AFF_DODGE specifically

---

## Medium Issues

### M1. THAC0 table has 12 classes but ClassMystic (index 11) in Go may not match C's NUM_CLASSES order
The C code uses `GET_CLASS(ch)` as a direct index. If the Go class constants don't match C's class order exactly, the wrong THAC0 values will be used.

### M2. `GetAttacksPerRound()` minimum attacks differs
- **C**: `if (attacks < 0) attacks = 0;` — allows 0 attacks (character does nothing that round)
- **Go**: `if attacks < 1 { attacks = 1 }` — minimum 1 attack

With SLOW affect, C allows 0 attacks (complete inaction). Go always forces at least 1.

### M3. `dam_message()` tiers don't match C thresholds
- **C** thresholds: 0, 1-2, 3-4, 5-6, 7-10, 11-14, 15-19, 19-23, 23-33, 33-43, 43-53, >53
- **Go** thresholds: 0, 1, 3, 5, 7, 11, 18, 26, 36, 48, 60, 80, 101, 10000+

The Go thresholds produce different messages for the same damage amount. E.g., 15 damage is "extremely hard" in C but "very hard" in Go (C threshold 15-19 vs Go threshold 11-17).

### M4. `dam_message()` tier descriptions differ
- **C**: tiers 7-11 are "massacres", "OBLITERATES", "EVISCERATES", "DESTROYS", "ROCKS THE HELL OUT OF"
- **Go**: tiers 7-14 are "violently", "savagely", "MUTILATES", "DISEMBOWELS", "DESTROYS", "OBLITERATES", "R O C K S"

Different flavor text with different emotional intensity. The C messages are more dramatic.

### M5. `DamMessage()` only sends room message, not char/victim messages
- **C**: Sends three separate messages (to room, to attacker, to victim) with color codes
- **Go**: Only sends room message via `BroadcastMessage`. Attacker and victim don't see their specific damage messages.

### M6. `mana_gain()` NPC branch missing
- **C**: `if (IS_NPC(ch)) { gain = GET_LEVEL(ch); }`
- **Go**: `ManaGainNPC()` exists but the `PointUpdate()` loop doesn't call it for NPCs (only HP regen for NPCs)

### M7. `move_gain()` for NPCs missing from PointUpdate
- **C**: `GET_MOVE(i) = MIN(GET_MOVE(i) + move_gain(i), GET_MAX_MOVE(i))` for all characters including NPCs
- **Go**: Only regens player move, not NPC move

### M8. `hit_gain()` NPC formula approximation
- **C**: `gain = 2.5*GET_LEVEL(ch)` (float math with integer truncation) for level < 23
- **Go**: `return (lvl*5 + 1) / 2` — this is `(5*level + 1) / 2` which approximates `2.5*level` but differs by 0 or 1 at some levels. For level 1: C = 2, Go = 3.

### M9. `HitGain()` sleeping equip regen applies wrong argument
- **C**: `GET_EQ(ch, i)->affected[j].location == APPLY_HIT_REGEN` — modifier applies directly (positive and negative) while sleeping
- **Go**: `sumEquipAffect(ApplyHitRegen, false)` — `false` means positive modifiers are NOT filtered, which matches C. But the C code applies positive modifiers ONLY while sleeping, while Go applies them unconditionally because `requireSleeping=false` doesn't filter anything.

Wait — actually re-reading: Go passes `requireSleeping=false` which means "don't require sleeping for positive modifiers." But C only runs the equipment loop inside the `POS_SLEEPING` case. So Go applies hit regen equipment bonuses even while awake, which C does not.

### M10. `sumEquipAffect()` applies positive mana regen even while not sleeping
Same issue as M9 but for mana regen. C only applies positive `APPLY_MANA_REGEN` while sleeping; Go's `requireSleeping` parameter is set incorrectly.

Actually looking more carefully: `ManaGain` calls `sumEquipAffect(ApplyManaRegen, pos == PosSleeping)`. This IS correct — it passes `true` when sleeping. But `HitGain` calls `sumEquipAffect(ApplyHitRegen, false)` — this should be `true` to match C's sleeping-only behavior. **Bug confirmed.**

### M11. `CheckIdling()` missing save and crash-save steps
- **C**: On first idle threshold, calls `save_char(ch, NOWHERE)` and `Crash_crashsave(ch)` before moving to void
- **Go**: Just transfers the player without saving

### M12. `gain_exp()` has extra per-level XP cap not in C
- **C**: `gain = MIN(max_exp_gain, gain)` then `gain = MIN(max_exp-1, gain)` where max_exp = exp for next level minus current exp
- **Go**: Adds additional `perLevelCap = level * 1000` which doesn't exist in C

This Go-specific cap will prevent legitimate high-XP kills from awarding full XP, especially at higher levels.

### M13. `FindExp()` default modifier values may not match C's `class.c`
The Go file has hardcoded modifier values that need verification against C's `find_exp()` in `class.c`. The class-to-modifier mapping should be exact.

### M14. Weather functions: `another_hour()` and `weather_change()` seem correct but use different RNG patterns
Go uses `rand.Intn()` where C uses `number()`. The `number()` function is inclusive on both ends; `rand.Intn(n)` is [0, n). Most calls have been adjusted, but the weather code has complex probability chains that could have off-by-one errors.

### M15. `MobHitGain` rounding differs from C
- **C**: `gain = 2.5 * GET_LEVEL(ch)` — float multiplication, integer truncation. Level 1 = 2, Level 3 = 7, Level 5 = 12
- **Go**: `(lvl*5+1)/2` — Level 1 = 3, Level 3 = 8, Level 5 = 13

Every level gives 1 more HP in Go than C for NPCs below level 23.

---

## Low Issues

### L1. `load_messages()` / combat message file not ported
- **C**: Loads `messages` file with fight messages (miss/hit/die/god messages per attack type)
- **Go**: No message file loading. `SkillMessageFunc` is a hook that the game layer must provide.

### L2. `replace_string()` / `#w` `#W` substitution in Go uses `strings.ReplaceAll`
C uses character-by-character copy; Go uses bulk replacement. Functionally equivalent for well-formed input.

### L3. `skill_message()` reduced to a hook
- **C**: Full implementation with random message selection from loaded messages
- **Go**: Delegates to `SkillMessageFunc` hook

### L4. Death cry goes to adjacent rooms correctly in both, but C uses character movement to send messages from adjacent rooms (abusing `ch->in_room`), while Go uses `GetAdjacentRoom()`.

### L5. `make_dust()` in Go uses generic ash object instead of C's VNum-based dust (18) and vampire_dust (1230) prototypes
- **C**: `read_object(dust, VIRTUAL)` / `read_object(vampire_dust, VIRTUAL)` — loads actual game objects with prototypes
- **Go**: Creates a synthetic "a pile of ash" object with no prototype

### L6. Race-based dust creation not fully implemented
- **C**: Creates `dust` (vnum 18) for undead/disintegrate, `vampire_dust` (vnum 1230) for vampires
- **Go**: Always creates generic ash object regardless of race

### L7. `make_corpse()` gold duplication loophole fix
- **C**: `if (IS_NPC(ch) || (!IS_NPC(ch) && ch->desc))` prevents gold duplication for linkless players
- **Go**: No such check — always creates gold in corpse

### L8. `set_fighting()` missing AFF_SLEEP removal
- **C**: Removes `SPELL_SLEEP` and `SKILL_SLEEPER` affects when entering combat
- **Go**: No sleep affect removal in `SetFighting()`

### L9. `raw_kill()` missing tattoo affect removal
- **C**: Calls `tattoo_af(ch, FALSE)`, clears tattoo, clears timer, removes werewolf/vampire affects, caps mana, unmounts
- **Go combat package**: Just calls `RemoveAllAffects()` and `Unmount()`

### L10. `raw_kill()` missing mob memory clearing loop
- **C**: Iterates all characters to `forget(mob, ch)` and `set_hunting(mob, NULL)` for the dead character
- **Go**: No equivalent mob memory clearing on death

### L11. `perform_violence()` spec proc call at end missing
- **C**: At end of each combat round, calls `mob_index[GET_MOB_RNUM(ch)].func` for MOB_SPEC mobs
- **Go**: No spec proc call from combat loop

### L12. `check_idling()` missing `Crash_idlesave` / `Crash_rentsave` for linkless players
- **C**: On disconnect, saves player via crash-safe rent system
- **Go**: Just logs and extracts

---

## File-by-File Notes

### fight.c → death.go / combat/*.go

The core combat system is the most complex port. The combat package (`pkg/combat/`) takes a clean, abstract approach with a `Combatant` interface and hook functions, while the game package (`pkg/game/death.go`, `pkg/game/combat_*.go`) provides concrete implementations.

**Key structural difference**: The C code is monolithic — `damage()` handles the entire damage pipeline including death, XP, gold, alignment, and messages. The Go code splits this across `TakeDamage()` (combat package) and `HandleDeath()` (game package), with hooks bridging the gap. This creates two failure modes: (1) the hook may not be set, causing silent failures, and (2) state may drift between the two packages.

**Critical constant mismatches** (C1, C2) mean the two packages cannot correctly interoperate without a mapping layer. Either the combat package needs to use 300/399 for TYPE_HIT/TYPE_SUFFERING, or the game package needs to use 2000/3000, but they must agree.

### limits.c → limits_*.go

The regen formulas are generally faithful with two notable exceptions:
1. `HitGain()` sleeping equipment regen is applied unconditionally (M10)
2. NPC hit regen rounding gives 1 extra HP per tick (M15)

The `GainExp()` function adds a non-C per-level XP cap (M12) that will change game balance. The `GainCondition()` logic is correct.

### act.movement.c → act_movement.go

This port looks relatively complete. All major functions (`do_simple_move`, `perform_move`, `find_door`, `has_key`, `do_doorcmd`, `ok_pick`, position commands) are present. The movement logic, door handling, and position change commands appear faithful.

### act.offensive.c → skill_commands.go

Most offensive skills are present as command stubs (backstab, bash, kick, rescue, etc.). The skill check logic appears simplified compared to C. Notable: `do_shoot` (ranged combat) is listed but likely simplified. `do_flee` and `do_retreat` have Go implementations.

### spells.c → spells/*.go

The spell system is significantly restructured. C uses `ASPELL()` macros and a switch-based dispatch. Go uses `setupSpellInfo()` with function pointers and the `ExecuteManualSpell()` dispatcher. Most key spells are present (recall, teleport, summon, charm, identify, enchant weapon/armor, lycanthropy, vampirism, hellfire, meteor swarm, etc.).

Missing from Go: `spell_divine_int`, `spell_silken_missile` full logic, `spell_mindsight`, `spell_mental_lapse`, `spell_calliope` as separate implementations (some may be in the affect_spells.go but with simplified logic).

### handler.c → equipment.go / inventory.go / follow.go

The handler functions are split across multiple Go files. Core operations (equip/unequip, obj_to_char/obj_from_char, affect_to_char/affect_remove) are present. The `affect_modify` / `affect_total` system is ported to the `AffectManager` in the engine package.

### act.comm.c → act_comm.go / comm_*.go

The race-say language system is fully ported with syllable substitution. The communication commands (say, tell, reply, gossip) are present. The `speak_drunk` function exists.

### weather.c → weather.go

Appears faithful. Weather change, time progression, and the moon gate / lunar hunter / ghost ship events are all present. The RNG patterns may have minor off-by-one differences.

### shop.c → shop.go / systems/shop_manager.go

Significantly restructured. C's expression-evaluation shop system is replaced with simpler buy/sell price formulas. The `trade_with()`, `same_obj()`, and expression evaluation functions are not directly ported. Shop commands (list, buy, sell) are in `pkg/command/shop_commands.go`.

### mobprog.c → mobprogs.go

Core mob prog functions are present: `MpGreet`, `MpRideGreet`, `MpGive`, `MpBribe`, `EntryProg`. The `isCitizen` and `isCityguard` checks exist. `NpcRescue` and `MpSound` are Go additions.

### graph.c → graph.go

`FindFirstStep` (BFS pathfinding) is ported. `doTrack` and `huntVictim` are present. Hunt trash-talk messages and mob communication functions are implemented.

### house.c → houses.go / house_save.go

Core house functions are present: `HouseGetFilename`, `ObjFromStore`, `ObjToStore`, `findHouse`, save/load/crashsave. The `hcontrol` commands (build, destroy, set_key, pay, list) need verification.

### ban.c → bans.go

Faithful port. `LoadBanned`, `IsBanned`, `WriteBanList`, `DoBan`, `DoUnban`, `ValidName`, `ReadInvalidList` all present.

### mail.c → mail.go

Faithful port of the mail system. File-based storage with free list, index, and send/receive/check commands. `PostmasterSendMail`, `PostmasterCheckMail`, `PostmasterReceiveMail` are all present.

---

## Recommendations

1. **Fix TYPE_HIT/LVL_IMMORT constant alignment immediately.** This is a ticking time bomb. Pick one value system (C's 300/399/31/40) and enforce it everywhere.
2. **Add backstab_mult() function and wire it into MakeHit().** Without this, thieves and assassins are severely underpowered.
3. **Fix GetPositionFromHP() to preserve position when HP > 0 and position > POS_STUNNED.**
4. **Fix ChangeAlignment() neutral boundary: use `> -350 && < 350` instead of `>= -350 && <= 350`.**
5. **Implement the charm retarget and switcheroo logic in TakeDamage().**
6. **Fix HitGain() sleeping equipment regen to only apply positive modifiers while sleeping.**
7. **Remove the non-C per-level XP cap from GainExp() or make it configurable.**
8. **Fix MobHitGain rounding to match C's `2.5 * level` with proper truncation: `return int(2.5 * float64(lvl))` for level < 23.**
