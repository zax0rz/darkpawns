# Phase 2c Task 2 Implementation Report

## Task: Fix world correctness issues found in QA audit

**Repo:** `/home/zach/.openclaw/workspace/darkpawns-phase1/`
**Original C source:** `/home/zach/.openclaw/workspace/darkpawns/src/`

## Changes Implemented

### H1: Implement advance_level()

**Files Changed:**
1. `pkg/game/level.go` - **NEW FILE**
   - Added `conApp` table (from `constants.c:1124-1150`)
   - Added `wisApp` table (from `constants.c:1152-1178`)
   - Implemented `AdvanceLevel()` method (from `class.c:600-720`)
   - Calculates HP/mana/move gains when leveling up
   - Includes CON-based HP bonus and WIS-based practice bonus

2. `pkg/game/player.go`
   - Updated `NewCharacter()` to call `AdvanceLevel()` at level 1
   - This ensures starting characters get proper HP: `10 + con_app[con].hitp + random_range`

**Source Citations:**
- `class.c:600-720` - `advance_level()` function
- `constants.c:1124-1150` - `con_app[]` table
- `constants.c:1152-1178` - `wis_app[]` table
- `class.c:538` - `do_start()` calls `advance_level()` for level 1

### H2: Mob equipped items → corpse

**Files Changed:**
1. `pkg/game/death.go`
   - Updated `handleMobDeath()` to transfer both inventory AND equipment to corpse
   - Added loop to collect all items from `deadMob.Equipment` map

**Source Citations:**
- `fight.c:make_corpse()` lines ~383-410
- Transfers BOTH inventory items AND all equipped slots into corpse container

### H5: Fix sentinel flag + aggression

**Files Changed:**
1. `pkg/game/ai.go`
   - Fixed `runMobAI()` to only skip movement for sentinel mobs, not aggression checks
   - Sentinel mobs can still attack if aggressive

**Source Citations:**
- `mobact.c:110-132` - `MOB_SENTINEL` only prevents movement, does NOT prevent aggression checks

### H6: Enforce MOB_STAY_ZONE

**Files Changed:**
1. `pkg/game/ai.go`
   - Updated `wanderMob()` to check for "stay_zone" action flag
   - Added zone check: if mob has `MOB_STAY_ZONE`, only pick exits where `destination_room.Zone == current_room.Zone`

**Source Citations:**
- `mobact.c:127` - when choosing a direction to wander, skip exits that lead to a different zone

### H7: Check ROOM_DEATH and ROOM_NOMOB before mob movement

**Files Changed:**
1. `pkg/parser/wld.go`
   - Added `parseRoomFlags()` function to parse room flag bitmask
   - Updated room parsing to extract flags from `.wld` file format

2. `pkg/game/ai.go`
   - Updated `wanderMob()` to check `ROOM_DEATH` and `ROOM_NOMOB` flags before allowing movement

**Source Citations:**
- `structs.h` - `ROOM_DEATH = 1` (bit 1), `ROOM_NOMOB = 2` (bit 2)
- `mobact.c` - before moving a mob to a room, checks `!ROOM_DEATH` and `!ROOM_NOMOB`

### M2/M3: Fix equipment slot mapping

**Files Changed:**
1. `pkg/game/equipment.go`
   - Added `SlotShield` constant
   - Updated `getWearFlags()` function:
     - `ITEM_WEAR_TAKE` (bit 0) = item can be picked up, NOT an equip slot (removed `SlotHold` mapping)
     - `ITEM_WEAR_SHIELD` (bit 9) now maps to `SlotShield`, not `SlotHold`

**Source Citations:**
- `structs.h:446-462` - `ITEM_WEAR_*` constants
- `structs.h:391-405` - equipment positions

### M4: Dual equipment slots

**Files Changed:**
1. `pkg/game/equipment.go`
   - Added dual slot constants: `SlotFingerR`, `SlotFingerL`, `SlotNeck1`, `SlotNeck2`, `SlotWristR`, `SlotWristL`
   - Updated `String()` and `ParseEquipmentSlot()` methods
   - Updated `getWearFlags()` to map `ITEM_WEAR_FINGER` to both finger slots, `ITEM_WEAR_NECK` to both neck slots, `ITEM_WEAR_WRIST` to both wrist slots
   - Updated `Equip()` method to handle dual slots: prefer right/first slot, use left/second if already occupied

**Source Citations:**
- `structs.h:391-405` - players have dual slots:
  - `WEAR_FINGER_R` and `WEAR_FINGER_L` (two ring slots)
  - `WEAR_NECK_1` and `WEAR_NECK_2` (two neck slots)
  - `WEAR_WRIST_R` and `WEAR_WRIST_L` (two wrist slots)

### M7: Fix inventory capacity formula

**Files Changed:**
1. `pkg/game/inventory.go`
   - Updated `SetCapacity()` to accept DEX and level instead of just strength
   - New formula: `5 + (dex / 2) + (level / 2)`

2. `pkg/game/player.go`
   - Updated `NewPlayer()` to call `SetCapacity(10, 1)` as default
   - Updated `NewCharacter()` to call `SetCapacity(p.Stats.Dex, p.Level)`

**Source Citations:**
- `utils.h:448-449`: `CAN_CARRY_N(ch) = 5 + (GET_DEX(ch) >> 1) + (GET_LEVEL(ch) >> 1)`

## Deviations from Original

1. **Weight tracking not implemented**: The original also has `CAN_CARRY_W(ch)` for weight capacity based on strength. This is flagged as TODO for Phase 3.

2. **Practice points not implemented**: The `AdvanceLevel()` function calculates practice bonuses from WIS but doesn't actually add them to player yet. This needs player practice tracking system.

3. **Move points not implemented**: The `AdvanceLevel()` function calculates move gains but doesn't add them to player yet. Move points system needs to be implemented.

4. **Immortal level handling**: The `AdvanceLevel()` function doesn't handle immortal levels (LVL_IMMORT) yet.

5. **Zone resets partially implemented**: Zone reset system exists but needs more work for periodic resets based on zone lifespan.

## Build Status

Build successful: `go build ./...` passes with no errors.

## Files Changed Summary

1. **New Files:**
   - `pkg/game/level.go`

2. **Modified Files:**
   - `pkg/game/player.go`
   - `pkg/game/death.go`
   - `pkg/game/ai.go`
   - `pkg/game/equipment.go`
   - `pkg/game/inventory.go`
   - `pkg/parser/wld.go`

## Next Steps for Phase 3

1. Implement weight capacity tracking (`CAN_CARRY_W`) using `str_app` table
2. Add practice points system to track and use practice points
3. Implement move points system
4. Complete zone reset implementation with proper timing and zone lifespan
5. Add immortal level handling
6. Implement proper resurrection mechanics (currently uses modern respawn)