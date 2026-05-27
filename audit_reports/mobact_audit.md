# Port Fidelity Audit: Module 36 (`mobact.c`)

This audit examines the port fidelity between the legacy C source file `src/mobact.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/mobact.c` (409 lines)
- **Functions**:
  - `mobile_activity`: The central MUD mobile AI tick loop running periodically for all active NPCs. It manages waking sleepers, hunting, special procedures, scavenging for items, random room wandering (with sector restrictions), sound/pulse triggers, aggressive attack matching (standard and alignment-based), race hate, memory retribution, and aid/assist helper behaviors.
  - `remember`, `forget`, `clearMemory`: Utility functions managing NPC memory retribution structures using a singly-linked list (`memory_rec`).

### Go Port Files
- **Mobile AI Engines**:
  - [pkg/game/mobact.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/mobact.go): Contains the central `MobileActivity` dispatcher, managing aggressive triggers, memory targeting, and helper assists.
  - [pkg/game/ai.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/ai.go): Implements the random wandering logic (`wanderMob`) and manages the AI tick scheduling.
- **Auxiliary Systems**:
  - [pkg/game/graph.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/graph.go): Houses the pathfinding and hunter logic (`huntVictim`), mapping to legacy hunt behaviors.
  - [pkg/game/scripts.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/scripts.go): Defines script trigger checking (`HasScript`) and execution (`RunScript`) wrappers.

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Uncalled Hunting Logic (Forgotten System Integration)
- **Source Context**: `pkg/game/graph.go#L252` (`huntVictim`), `pkg/game/mobact.go#L122` (`mobileActivityForMob`).
- **Fidelity Gap**: In legacy C, hunter-flagged NPCs hunt their target up to two steps per tick:
  ```c
  if ((GET_POS(ch) == POS_STANDING) && (MOB_FLAGGED(ch, MOB_HUNTER)) && (!FIGHTING(ch)))
      hunt_victim(ch);
  if ((GET_POS(ch) == POS_STANDING) && (MOB_FLAGGED(ch, MOB_HUNTER)) && (!FIGHTING(ch)))
      hunt_victim(ch);
  ```
  While the Go port fully implements the highly detailed `huntVictim` pathfinding, door-opening, and evasion sequence in `pkg/game/graph.go`, **it is never called by the mobile activity execution pipeline**. The `hunter` flag is unchecked, and `w.huntVictim` is completely bypassed, disabling mob hunting.

### 2. Missing Sector/Terrain Constraints in Mob Wandering
- **Source Context**: `pkg/game/ai.go#L95` (`wanderMob`).
- **Fidelity Gap**: In legacy C (`src/mobact.c#L130-L138`), before performing a random move, the NPC's movement capabilities are validated against the target room's sector:
  ```c
  if ((SECT(to_room) == SECT_WATER_SWIM || SECT(to_room) == SECT_WATER_NOSWIM) && !CAN_SWIM(ch))
      continue;
  if ((SECT(to_room) == SECT_FLYING) && !IS_FLYING(ch))
      continue;
  ```
  The Go implementation in `wanderMob` completely omits these terrain and movement type checks. Terrestrial mobs will wander into deep water or high altitude environments, breaking world constraints.

### 3. Continuous Wander Bug (Hyper-active Mobs)
- **Source Context**: `pkg/game/ai.go#L95` (`wanderMob`), `pkg/game/mobact.go#L168`.
- **Fidelity Gap**: In C, wandering is probabilistic. The random roll `door = number(0, 18)` ensures that if `door >= 6` (NUM_OF_DIRS), no move is attempted. This results in a ~31.5% chance to move per tick, and a ~68.4% chance to stand still.
  In Go, `wanderMob` is called every tick and immediately moves the mob to a random adjacent exit if one is available. This causes mobs to move **100% of the time on every tick**, making them slide between rooms constantly and causing extreme congestion.

### 4. Missing Script Tick Triggers (`onpulse_all`, `onpulse_pc`, `sound`)
- **Source Context**: `pkg/game/mobact.go#L122` (`mobileActivityForMob`).
- **Fidelity Gap**: Legacy C executes script hooks inside the periodic AI tick loop:
  - `sound` (1-in-16 chance, triggering both `mp_sound` and the Lua "sound" trigger).
  - `onpulse_all` (if room has any occupants).
  - `onpulse_pc` (if room has human players).
  The Go port totally omits these periodic script check triggers inside `MobileActivity`. While `fight` and `death` triggers are safely wired in other packages, critical time-based behaviors on specialized mobs (like Sungod, never_die, and Jailguards) are entirely dormant.

### 5. Missing Race Hate Aggression
- **Source Context**: `pkg/game/mobact.go#L122`.
- **Fidelity Gap**: C MUD contains a custom "Race Hate" feature (`src/mobact.c#L236-L259`) where NPCs detect characters in the room who hate their race, triggering aggressive preemptive combat and speech alerts (`Come to destroy my kin? Die!`). This entire sequence is absent in Go's `MobileActivity`.

### 6. Waking Sleeper'd Mobs Early-Return
- **Source Context**: `pkg/game/mobact.go#L124` & `L145`.
- **Fidelity Gap**: The wake/stand check in Go is placed at line 145:
  ```go
  if ch.GetPosition() < combat.PosSitting {
      ch.SetStatus("standing")
  }
  ```
  However, line 124 performs an early return if the mob is sleeping:
  ```go
  if ch.GetFighting() != "" || ch.GetPosition() <= combat.PosSleeping {
      return
  }
  ```
  As a result, sleeping mobs can never reach the wake-up block. (Note: This matches a legacy C dead-code bug where `!AWAKE(ch)` continued the loop at the very top, but in Go, it should ideally be resolved to allow sleeping mobs to recover).

---

## 3. Go Improvements Over C

### 1. Concurrency Goroutine-Safety
- **Go Enhancement**: C’s mob activity loop operated on the global mutable linked list `character_list`, exposing it to corruption if elements were added or removed mid-iteration. Go solves this by taking a safe snapshot slice of active mobs under a read lock (`w.activeMobs`) and iterating over the local copy.

### 2. Fine-grained Per-Mob Locking
- **Go Enhancement**: To prevent blocking the entire world during a slow script call or network operation, Go utilizes individual RWMutex locks on each `MobInstance` (`ch.mu.Lock()`), keeping the rest of the game loop highly responsive.

---

## 4. Concurrency & Thread Safety

- **Deadlock Mitigation in Wandering**:
  - The `wanderMob` implementation releases the individual mob lock (`mob.mu.Unlock()`) before sending player notification messages to avoid nested I/O deadlock cascades, then re-acquires the lock (`mob.mu.Lock()`) safely prior to returning.
- **Local Room Snapshots**:
  - Wandering looks up adjacent exits and room flags from thread-safe world snapshots (`w.snapshots.Snapshot()`), ensuring concurrent modifications to room flags by builders or spells don't trigger race conditions.

---

## 5. Summary of Recommended Fixes

1. **Wire up `w.huntVictim`**:
   Add the hunter validation and `w.huntVictim(ch)` calls inside `mobileActivityForMob` to enable NPC path tracking.
2. **Apply Sector Checks in Wandering**:
   Add terrain checks in `wanderMob` to prevent land mobs from moving into deep water or high altitudes.
3. **Restore Probabilistic Movement**:
   Introduce a probability roll (e.g. `rand.Intn(19) < 6`) in `wanderMob` to restore natural, slower wandering frequencies.
4. **Implement Missing Script Pulses**:
   Wire `RunScript("onpulse_all")`, `RunScript("onpulse_pc")`, and `RunScript("sound")` into the mob tick execution loop.
