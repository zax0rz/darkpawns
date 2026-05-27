# Port Fidelity Audit: Module 40 (`new_cmds2.c`)

This audit examines the port fidelity between the legacy C source file `src/new_cmds2.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/new_cmds2.c` (1,028 lines)
- **Functions & Features**:
  - **Advanced Character Skills**:
    - `do_scrounge`: Wilderness forage skill yielding sector-appropriate food/fish items.
    - `do_first_aid`: First aid skill reviving targets at `<= 0` HP (dying/unconscious state) to exactly `1` HP.
    - `do_disarm`: Combat martial skill disarming an opponent, removing their weapon to inventory.
    - `do_mindlink`: Psionic transfer of HP from caller to victim's mana pool.
    - `do_detect`: Detects secret doors in the room.
    - `do_dig`: Digs in suitable dirt/forest terrains to unearth random loot.
    - `do_turn`: Cleric holy turning of undead, with minor damage, fleeing, or instant disintegration.
    - `do_serpent_kick`: Martial kick with a small chance to spawn a helper/training mob at high levels.
  - **Special Procedures**:
    - `beholder`: Blocks spellcasting in the same room and casts disintegrate, sleep, curse, and disrupt at random.
    - `recharger`: Recharges wands and staves for gold, permanently decreasing maximum charges by 1.
    - `no_get`: Guard procedure attacking players taking room items.
    - `zen_master`: Teleports/recalls attackers away instead of taking damage.
    - `black_horn`: Summoning horn that generates scaled hunting mobs.
  - **Attraction Tickers**:
    - `hunt_items`: Background hourly task setting aggressive mobs to track players wearing specific attractive items (e.g. Skarash spiked armor).

### Go Port Files
- **Martial & Advanced Skills**:
  - [pkg/game/skills2.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/skills2.go): Houses the ports of `DoScrounge`, `DoFirstAid`, `DoDisarm`, `DoMindlink`, `DoDetect`, `DoDig`, and `DoTurn`.
  - [pkg/game/skill_advanced.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/skill_advanced.go): Contains `DoSerpentKick`.
- **Special Procedures**:
  - [pkg/game/spec_procs_missing.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/spec_procs_missing.go): Implements `specBeholder`, `specRecharger`, `specNoGet`, and `specZenMaster`.
- **Attraction Tick Loop**:
  - [pkg/engine/gameloop.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/engine/gameloop.go): Defines `OnHuntItems` callback for hourly updates.

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Completely Inactive Disarm Mechanics (Flavor Text Only)
- **Source Context**: `pkg/game/skills2.go#L182-L185` (`DoDisarm`), `src/new_cmds2.c#L236` (`do_disarm`).
- **Fidelity Bug**: In legacy C, succeeding on a `disarm` attempt unequips the victim's wielded weapon and transfers it to their inventory:
  ```c
  obj_to_char(unequip_char(vict, WEAR_WIELD), vict);
  ```
  While the Go port implements the success percentage roll and prints descriptive disarm messages (e.g. `"$n deftly disarms you!"`), **it completely lacks any logic to un-equip the target's weapon or transfer it to inventory**.
  - **Impact**: Charmed pets or players are "disarmed" in flavor text but retain their wielded weapons and continue to deal full weapon damage in combat.

### 2. Dormant Background Hunt Attraction Ticker (`OnHuntItems` Gap)
- **Source Context**: `pkg/engine/gameloop.go#L75` & `L241` (`OnHuntItems`), `src/new_cmds2.c#L765` (`hunt_items`).
- **Fidelity Bug**: Legacy C executes `hunt_items` every Mud hour to track players wearing high-value attractive gear (Skarash spiked armor, jade/opal items) and direct specific mobs to hunt them across zones.
  While the Go server defines the `OnHuntItems` callback in its main heartbeats (`gameloop.go`), **the callback is never assigned or registered during server bootstrap**. 
  - **Impact**: The entire attractive gear hunting system is entirely dormant in Go; monsters never track players based on their inventory/equipment sets.

### 3. 100x Inflation on Wand/Staff Recharge Cost (Economy Ruined)
- **Source Context**: `pkg/game/spec_procs_missing.go#L93` (`specRecharger`), `src/new_cmds2.c#L460` (`recharger`).
- **Fidelity Bug**: In legacy C, the recharge cost is `spell_level * 100` (e.g. a wand casting level 25 Heal costs 2,500 gold per charge).
  The Go port calculates the recharge cost as:
  ```go
  cost := 1000 * spellLvl * maxCharges
  ```
  - **Impact**: For a level 25 wand with 10 max charges, the cost becomes `1000 * 25 * 10 = 250,000` gold! Recharging in Go is **100 times more expensive than C**, destroying the MUD's magical economy.

### 4. Beholder Anti-Magic Spell-Block Bypassed
- **Source Context**: `pkg/game/spec_procs_missing.go#L118` (`specBeholder`), `src/new_cmds2.c#L334` (`beholder`).
- **Fidelity Bug**: In legacy C, the `beholder` special procedure completely intercepts spell casting commands (`cast` and `recite`), breaking concentration and blocking mages from casting spells.
  The Go port of `specBeholder` completely bypasses this command-block by ignoring all non-combat tick queries:
  ```go
  if cmd != "" {
      return false
  }
  ```
  - **Impact**: Mages can freely cast spells in combat against Beholders, neutralizing the signature anti-magic cone challenge.

### 5. First Aid Omission for Mobs
- **Source Context**: `pkg/game/skills2.go#L123` (`DoFirstAid`), `src/new_cmds2.c#L170` (`do_first_aid`).
- **Fidelity Bug**: C's `do_first_aid` revives any character in the room (PC or NPC). The Go port casts the target as `target.(*Player)`, meaning players cannot use first aid to stabilize dying/unconscious charmed pets or friendly mobs.

---

## 3. Go Improvements Over C

### 1. Robust Spawning and Decay on Digging
- **Go Enhancement**: In C, digging was rigid and limited. Go's `DoDig` implements safe object spawning (`world.SpawnObject`) with structured timer controls (`SetTimer`), ensuring unearthed puddle decay mechanics work seamlessly without manual reallocations.

### 2. Clean Holy Turn Clamping
- **Go Enhancement**: Go's `DoTurn` cleanly parses undead names and clamps holy damage ranges safely, returning descriptive combat notifications with elegant formatting.

---

## 4. Concurrency & Thread Safety

- **Local Object Instances**:
  - Scrounging (`DoScrounge`) and digging (`DoDig`) utilize local item instances added directly to character inventory pools, isolating mutations from parallel world builders.
- **Atomic Gold Deductions**:
  - The `specRecharger` procedure locks the player instance (`ch.mu.Lock()`) during gold checks and deductions, preventing concurrent transaction race conditions.

---

## 5. Summary of Recommended Fixes

1. **Implement Weapon De-equipping in `DoDisarm`**:
   Add equipment de-equipping logic inside `DoDisarm` to un-wield the weapon slot and place it in the target's inventory list upon success.
2. **Assign the `OnHuntItems` Callback**:
   Wire the `hunt_items` attraction scanner to `OnHuntItems` during server boot to restore monster attraction tracking.
3. **Correct Recharging Prices**:
   Adjust the cost calculation in `specRecharger` to match C: `cost := spellLvl * 100` per charge.
4. **Restore Beholder Anti-Magic Block**:
   Update `specBeholder` to intercept `"cast"` and `"recite"` commands and block spell casting inside the room.
