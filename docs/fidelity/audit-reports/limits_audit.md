# Port Fidelity Audit: Module 31 (`limits.c`)

This audit examines the port fidelity between the legacy C source file `src/limits.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/limits.c` (687 lines)
- **Functions**: 
  - `mana_gain`, `hit_gain`, `move_gain`: Compute hourly regeneration rates for character pools (Mana, Hit, and Move).
  - `gain_exp` & `gain_exp_regardless`: Adjust player experience, cap single-kill gains, and advance levels.
  - `gain_condition`: Updates hunger, thirst, and sobriety states.
  - `check_idling`: Enforces idle void pulls and force-rent socket disconnects.
  - `point_update`: Master periodic tick function updating all characters (conditions, regen, poison/cutthroat damage, memory clearing) and decaying global objects (corpses, summons, dust, puddles).

### Go Port Files
- **Regeneration Logic**:
  - [limits_gain.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/limits_gain.go): Implements `ManaGain`, `HitGain`, and `MoveGain` formulas.
- **Periodic Update Loop**:
  - [limits_condition.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/limits_condition.go): Implements `PointUpdate` tick function and `decayObjectsInRoom`.
- **Experience & Idle Checks**:
  - [limits_exp.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/limits_exp.go): Implements `GainExp`, `GainExpRegardless`, and `CheckAutowiz`.
  - [limits_misc.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/limits_misc.go): Implements `CheckIdling`, `sumEquipAffect`, and `isFighting` helper.

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Backwards Equipment Mana Regen Logic (Critical Bug)
- **Source Context**: `pkg/game/limits_misc.go#L67` (`sumEquipAffect`), `pkg/game/limits_gain.go#L35` (`ManaGain`).
- **Fidelity Bug**: Legacy C specifies that positive equipment mana regen modifiers *only* apply if the player is sleeping, while negative modifiers apply always.
  Go's helper is defined as:
  ```go
  func (p *Player) sumEquipAffect(location int, requireSleeping bool) int {
      ...
      if requireSleeping && af.Modifier > 0 {
          continue // skips positive modifiers when requireSleeping is true
      }
      ...
  }
  ```
  In `ManaGain`, Go calls:
  ```go
  gain += p.sumEquipAffect(ApplyManaRegen, pos == PosSleeping)
  ```
  If a player is sleeping (`pos == PosSleeping` is `true`), `requireSleeping` is passed as `true`, so positive regen modifiers are **skipped (ignored)**.
  If a player is awake (`pos == PosSleeping` is `false`), `requireSleeping` is passed as `false`, so positive modifiers are **included**.
- **Impact**: The logic is completely backwards. Awake characters receive positive mana regeneration from equipment, whereas sleeping characters are locked out of their positive gear bonuses. The caller must pass `pos != PosSleeping` to correctly skip modifiers only when the player is awake.

### 2. Severe Room-Locked Object Decay (Behavior & Memory Leak Gap)
- **Source Context**: `src/limits.c#L530` (C objects loop), `pkg/game/limits_condition.go#L278` (Go `PointUpdate` NPC block).
- **Fidelity Bug**: In legacy C, `point_update` loops over the global `object_list` containing every active object in the MUD. It decays and extracts corpses, summoned circles, dust piles, puddles, and field objects regardless of where they reside (on the floor, in a container, or carried by players).
  In Go, `decayObjectsInRoom` is **only** called inside the NPC (mob instance) loop of `PointUpdate`.
- **Impact**: 
  - **Undeleting Room Floods**: If a room contains no active mobs, any items on the floor (like corpses left after a player kill, puddles, or dust) will **never decay**. Empty zones will accumulate trash indefinitely, degrading server memory and world presentation.
  - **No Inventory Decay**: Items carried in player inventories or stored inside player bags (such as decomposing corpses) are completely ignored and will never rot, allowing infinite corpse carrying.

### 3. Custom XP Capping (Behavior Discrepancy)
- **Source Context**: `pkg/game/limits_exp.go#L150` (`GainExp`).
- **Fidelity Bug**: Go introduces a dynamic per-level cap: `perLevelCap := p.Level * 1000`. This limits single-kill XP to a factor of the player's level, preventing low-level characters from power-leveling on disproportionate group kills.
- **Impact**: While this is a highly beneficial design improvement for leveling balance, it deviates from the stock CircleMUD C code which only applies a static flat cap (`max_exp_gain`).

---

## 3. Go Improvements Over C

### 1. Autowiz Process Isolation
- **Go Enhancement**: C's `check_autowiz` spawned a shell sub-process running `nice ../bin/autowiz ...` via `system()`. This posed serious command injection risks and could block server threads. Go cleanly replaces this with simple structural logging (`slog.Info`), letting modern admin tools consume MUD events safely in a decoupled manner.

### 2. Active Jail Release
- **Go Enhancement**: While legacy C merely decremented `GET_JAIL_TIMER`, Go's periodic tick proactively checks for expiration and automatically teleports players out of jail back to the starting city (`MortalStartRoom`), giving players a much cleaner experience without relying on manual admin check-ups.

### 3. Concurrency Thread-Safety
- **Go Enhancement**: Unlike C's unsafe shared structures, Go's HMV pool updates read stats safely using `RLock()` and update fields under explicit `Lock()` protection on the `Player` struct, avoiding concurrent write panic potentials in high-concurrency environments.

---

## 4. Concurrency & Thread Safety

- **World snapshot iteration**:
  - `PointUpdate` safely takes a read lock on the world's players and mobs arrays, copies them to local slices, releases the lock, and processes tick states. This keeps game-loop blocking time extremely brief and thread-safe.
- **Individual player locks**:
  - Pool updates strictly isolate reads and writes on player attributes via standard locking practices (`p.mu.Lock()` / `p.mu.Unlock()`), preventing dirty reads in concurrent API socket checks.

---

## 5. Summary of Recommended Fixes

1. **Fix backwards equipment check in `ManaGain`**:
   Update `pkg/game/limits_gain.go#L35` to pass `pos != PosSleeping` as the second argument:
   ```go
   gain += p.sumEquipAffect(ApplyManaRegen, pos != PosSleeping)
   ```
2. **Implement Global Object Decay Loop**:
   Relocate `decayObjectsInRoom` out of the localized NPC loop in `pkg/game/limits_condition.go`. Update `PointUpdate` to fetch all active objects in the world and decrement/decay them globally (including carried items) to match C's global garbage collection.
