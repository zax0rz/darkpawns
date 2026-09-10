# Port Fidelity Audit: Module 20 (`config.c`)

This audit examines the port fidelity between the legacy C source file `src/config.c` (game operations, constants, and limits) and its Go implementations in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/config.c` (287 lines)
- **Configuration Variables**: 
  - `level_can_shout`, `holler_move_cost`
  - `max_exp_gain`, `max_exp_loss`
  - `max_npc_corpse_time`, `max_pc_corpse_time`
  - `dts_are_dumps`, `OK`, `NOPERSON`, `NOEFFECT`
  - `free_rent`, `max_obj_save`, `min_rent_cost`, `auto_save`, `autosave_time`, `crash_file_timeout`, `rent_file_timeout`
  - `mortal_start_room`, `kiroshi_start_room`, `alaozar_start_room`, `immort_start_room`, `frozen_start_room`
  - `donation_room_1`, `donation_room_2`, `donation_room_3`
  - `DFLT_PORT`, `DFLT_DIR`, `MAX_PLAYERS`, `max_filesize`, `max_bad_pws`, `nameserver_is_slow`
  - `MENU`, `SHORT_GREETINGS`, `GREETINGS`, `WELC_MESSG`, `START_MESSG`
  - `use_autowiz`, `min_wizlist_lev`
  - `ident`, `LOGNAME`

### Go Port Files
Unlike legacy C where all global configurations sat in a single re-compilable `.c` file, the Go port distributes these constants across relevant functional domains:
- `pkg/game/limits.go` (Defines XP gain/loss limits: `maxExpGain`, `maxExpLoss`)
- `pkg/game/objsave.go` (Defines Rent limits: `RentFileTimeout`, `CrashFileTimeout`, `MaxObjSave`, `MinRentCost`)
- `pkg/game/death.go` (Defines `MortalStartRoom` and raw corpse creation)
- `pkg/game/act_comm.go` (Defines communication restrictions: `levelCanShout`, `levelCanGossip`, `hollerMoveCost`)
- `pkg/game/limits_condition.go` (Implements object and corpse decay loops)
- `pkg/game/logging.go` (Defines `LogDeathTrap` stub)
- `pkg/session/char_creation.go` (Controls hometown start rooms: Kir Drax'in, Kir-Oshi, Alaozar)
- `pkg/session/session_login.go` & `pkg/session/manager.go` (Provides modern IP-based rate limiting and brute force lockout trackers)
- `cmd/server/main.go` (Accepts modern command line flags for server `port`, `telnet-port`, `world` path, and `web` path, replacing compiled-in system defaults)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Broken Corpse Container System (Stranded Equipment on Death)
- **Source Context**: `pkg/game/death.go#L698-L752` (`makeCorpse`), `pkg/game/object.go#L112-L117` (`IsContainer`)
- **Fidelity Bug**: Legacy C creates corpse objects as containers (`ITEM_CONTAINER`) and puts player equipment inside on death. 
  In the Go port, `makeCorpse` builds a synthetic corpse object with a `nil` prototype:
  ```go
  corpse := &ObjectInstance{
      Prototype: nil, // synthetic object, no prototype vnum
      VNum:      -1,
      ...
  }
  ```
  However, `ObjectInstance.IsContainer()` determines containment capability strictly from the object's prototype:
  ```go
  func (o *ObjectInstance) IsContainer() bool {
      return o.GetTypeFlag() == 1 // GetTypeFlag returns 0 if Prototype == nil
  }
  ```
  As a result, `corpse.IsContainer()` returns `false`. When `makeCorpse` loops through a dying character's inventory and equipment to transfer them into the corpse, it calls:
  ```go
  _ = w.MoveObjectToContainer(item, corpse)
  ```
  This delegates to `MoveObject` which checks `!container.AddToContainer(obj)`. Because `IsContainer()` returns `false`, `AddToContainer` returns `false`, and `MoveObject` fails, triggering a rollback that re-attaches all items back onto the dead player.
- **Impact**: Players never lose their gear when they die! The equipment is seamlessly rolled back into their inventory and equipped slots during death. The created corpse on the ground is completely empty, breaking core MUD survival mechanics.

### 2. Permanent Ground Clutter (Corpses Last Forever)
- **Source Context**: `pkg/game/limits_condition.go#L315-L320` (`decayObjectsInRoom`), `pkg/game/death.go#L698-L709` (`makeCorpse`)
- **Fidelity Bug**: 
  - **No Timers Set**: `src/config.c` configures `max_npc_corpse_time = 5` and `max_pc_corpse_time = 10` ticks until decay. In Go, `makeCorpse` completely fails to set `corpse.Timer = ...` on the constructed corpse, leaving it at default `0`.
  - **Broken Decay Loop**: The decay checker in `limits_condition.go` checks:
    ```go
    // Corpse decay — ITEM_CONTAINER with val[3] set (corpse flag)
    if obj.IsContainer() && obj.GetValue(3) != 0 {
    ```
    Since the corpse `IsContainer()` returns `false` (due to the `nil` prototype) and `obj.GetValue(3)` returns `0` (since `GetValue` returns `0` for `nil` prototypes), the corpse decay loop is entirely bypassed.
- **Impact**: All corpses created in the MUD stay on the ground forever. The server will gradually accumulate infinite corpse instances over long runtimes, leading to persistent memory leaks and massive performance degradation in busy zones.

### 3. Broken Immortals & Auto-Loot due to Hollow `IsCorpse` Field
- **Source Context**: `pkg/game/death.go#L698-L709` (`makeCorpse`), `pkg/game/spec_procs3.go#L1006` (`specMortician`), `pkg/session/manager.go#L300` (Auto-Loot)
- **Fidelity Bug**: The `ObjectInstance` struct has a typed boolean field `IsCorpse bool`. However, `makeCorpse` never sets this field to `true`, leaving it `false` (it only sets an `"is_corpse": true` key in the legacy `CustomData` map).
- **Impact**: 
  - **Auto-Loot**: The auto-loot loop checks `if item.IsCorpse ...` which evaluates to `false`, breaking auto-loot.
  - **Mortician Quest Panic**: `specMortician` checks:
    ```go
    if obj.IsCorpse && strings.Contains(strings.ToLower(obj.Prototype.Keywords), ...)
    ```
    Because `obj.IsCorpse` is `false`, it skips the branch, preventing players from retrieving their corpses. Furthermore, if `IsCorpse` were true, evaluating `obj.Prototype.Keywords` on a synthetic corpse (where `Prototype` is `nil`) would instantly trigger a **nil pointer panic**, crashing the entire server.

### 4. Experience Loss Cap Typos (10x reduction in penalty)
- **Source Context**: `pkg/game/limits.go#L35` (`maxExpLoss`)
- **Fidelity Bug**: In `src/config.c`, `max_exp_loss` is set to `500000` (500,000 EXP).
  The Go port's `limits.go` defines:
  ```go
  maxExpLoss = 50000
  ```
- **Impact**: Dying carries a 10 times lower experience penalty than originally designed, severely trivializing character deaths.

### 5. Compiled-In Housing & Start Rooms Completely Omitted
- **Source Context**: `src/config.c` (VNUMs 1204, 1202, 8053, 18204)
- **Fidelity Bug**: 
  - **Immortal Start Room** (`1204`) and **Frozen Start Room** (`1202`) are completely un-referenced in the Go codebase. Frozen players are not sent to any special holding cell.
  - **Donation Rooms** (`8053`, `18204`) are unported. Dropping items in Go never routes them to a community chest, and items are left on the ground.

---

## 3. Go Improvements Over C

### 1. Dynamic Config via Startup Flags
- **Fidelity Improvement**: In legacy C, the server port, telnet port, and directories were compiled directly into `config.c`. Go replaces this with modern command-line flags (`-port`, `-telnet-port`, `-world`, `-web`, `-hugo`), permitting seamless containerization and deployment without code rebuilds.

### 2. Standardized Hashed Accounts & IP Lockouts
- **Fidelity Improvement**: Instead of the legacy compiled `max_bad_pws = 3` counter that terminates individual connections, Go integrates a secure, modern account rate-limiter and IP brute-force lockout manager (`loginLimiter` and `loginAttempts`), defending the telnet and websocket boundaries from automated dictionary attacks.

---

## 4. Summary of Configuration Mismatches

The following table summarizes the configuration parameters compiled in `src/config.c` against their hardcoded/defined counterparts in Go:

| Parameter | C Value | Go Value | Status / Location | Impact of Drift |
|---|---|---|---|---|
| `level_can_shout` | `2` | `5` | `act_comm.go#L96` | Low-level characters must grind to level 5 before communicating globally. |
| `holler_move_cost` | `20` | `10` (Unused) | `act_comm.go#L98` | The `holler` command is omitted; global broadcasts cost no move points. |
| `max_exp_gain` | `100000` | `100000` | `limits.go#L34` | Matches legacy. |
| `max_exp_loss` | `500000` | `50000` | `limits.go#L35` | **Critical Typo**: Player death penalty is 10x too low. |
| `max_npc_corpse_time`| `5` | Unset (`0`) | `limits_condition.go` | **Severe Bug**: NPC corpses never decompose. |
| `max_pc_corpse_time` | `10` | Unset (`0`) | `limits_condition.go` | **Severe Bug**: Player corpses never decompose. |
| `dts_are_dumps` | `YES` | Unused | N/A | Items dropped in death traps are not junked. |
| `max_obj_save` | `30` | `60` | `objsave.go#L32` | Max saved items on rent doubled, increasing storage load. |
| `min_rent_cost` | `100` | `10` | `objsave.go#L33` | Base receptionist cost is 10x cheaper. |
| `crash_file_timeout` | `10` | `365` | `objsave.go#L31` | Offline player crash records last 36.5 times longer on disk. |
| `donation_room_1` | `8053` | Unused | N/A | Donation room system completely missing. |
| `immort_start_room` | `1204` | Unused | N/A | Immortals start in mortal rooms instead of wizard offices. |
| `frozen_start_room` | `1202` | Unused | N/A | Frozen cheaters do not spawn in the penalty box. |

---

## 5. Summary of Recommended Fixes

1. **Resolve Corpse Containment & Decay Capabilities**:
   - Refactor `ObjectInstance.IsContainer()` to return `true` if `o.IsCorpse` is `true`, even if `o.Prototype` is `nil`.
   - Update `makeCorpse` to set `corpse.IsCorpse = true` and initialize its values array or override values to properly satisfy `GetValue(3) != 0` checks.
   - Wire in `max_npc_corpse_time = 5` and `max_pc_corpse_time = 10` timers on corpse creation so the decay loop can decrement and successfully extract them.
2. **Secure the Mortician Special Procedure against Nil Panics**:
   - Ensure `specMortician` checks for `obj.IsCorpse` safely without dereferencing a nil `obj.Prototype`. Use `obj.Runtime.Name` or keywords built dynamically in `makeCorpse`.
3. **Correct the Experience Loss Penalty**:
   - Set `maxExpLoss` in `pkg/game/limits.go` to `500000` to restore original death risk.
4. **Clean up Configuration Parameters**:
   - Centralize configuration constants from `limits.go`, `objsave.go`, and `act_comm.go` into a unified `Config` or `Settings` structure to emulate the single-location edit convenience of the legacy `src/config.c`.
