# Port Fidelity Audit: Module 46 (`scripts.c`)

This audit examines the port fidelity between the legacy C source file `src/scripts.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/scripts.c` (2,116 lines)
- **Functions & Features**:
  - **Shared Global Lua VM**: Boots a single, global `lua_State` to execute MUD scripts.
  - **C-to-Lua API Binding**: Registers a massive array of C functions (`cmdlib`) to the Lua runtime (e.g. `lua_act`, `lua_action`, `lua_mload`, `lua_extchar`, `lua_echo`).
  - **Lua Object Bridges**: Maps game entities (`char_data`, `obj_data`, `room_data`) back and forth into Lua tables.

### Go Port Files
- **Go Implementation**:
  - [pkg/scripting/engine.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/scripting/engine.go): Embeds the pure-Go `github.com/yuin/gopher-lua` scripting runtime. Manages isolated execution context and sandbox restrictions.
  - [pkg/game/scripts.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/scripts.go): Binds runtime `MobInstance`, `ObjectInstance`, and `RoomInstance` trigger routines to the scripting engine.
  - [lib/world/scripts/globals.lua](file:///Users/zach/.openclaw/workspace/darkpawns_repo/lib/world/scripts/globals.lua): Bootstraps global constants and loads shared helper scripts.

---

## 2. High-Fidelity Validation & Critical Gaps

This audit has discovered a **game-blocking quest bug** and a **silent pathing load failure** in the Go scripting engine:

### 1. Critical Undefined `foreachi` Quest-Blocker (Black Armor Quest)
- **The Gap**: In legacy C, scripts used Lua 4.x style array iteration. The quest blacksmith script `blacksmith.lua` (`lib/world/scripts/mob/blacksmith.lua#L39`) relies on `foreachi` to evaluate quest items:
  ```lua
  one = foreachi(pieces, one_piece)
  ```
- Standard modern Lua (5.1+) and the `gopher-lua` engine used by Go do **not** natively support the legacy `foreachi` function (it was deprecated and removed).
- The Go scripting engine has **completely omitted registering or defining `foreachi`** inside `globals.lua` or `engine.go`.
- **Impact**: When a player gives a suit of humming black armor to the blacksmith, the blacksmith's `ongive()` trigger runs `foreachi`, which resolves to `nil` and triggers a **runtime Lua panic**. The transaction rolls back, and players **can never complete the black armor quest**.

### 2. Silent Pathing Load Failure in `globals.lua`
- **The Gap**: Inside `lib/world/scripts/globals.lua#L154`, the script loads the movement blocking utility:
  ```lua
  dofile("scripts/mob/no_move.lua")
  ```
- However, the scripting engine defines the root directory `e.scriptsDir` as `lib/world/scripts/`. Inside `luaDofile` (`pkg/scripting/engine.go#L1665`), it constructs the path as:
  ```go
  filepath.Join(e.scriptsDir, path)
  ```
  This resolves to `lib/world/scripts/scripts/mob/no_move.lua`.
- **Impact**: Since the file is actually at `lib/world/scripts/mob/no_move.lua`, `dofile` fails to find the file. Because `luaDofile` catches the error silently (`slog.Debug("dofile error", ...)`), the failure occurs **completely silently in production**, leaving the `no_move` utility unloaded.

### 3. Commented-Out Ongive Rejection Messaging
- **The Gap**: In `pkg/game/scripts.go#L75-L79`, the server catches a false return from the `ongive` script (meaning the mob didn't want the item):
  ```go
  if trigger == "ongive" && !handled && err == nil && ctx.Ch != nil {
      // Send default "You can't give that here." message
      // In real implementation: ctx.Ch.SendMessage("You can't give that here.\r\n")
      slog.Debug("ongive returned false", ...)
  }
  ```
- **Impact**: The actual formatted socket output is commented out. When a player gives an invalid item to a quest mob, the transaction is rejected, but they receive **absolutely no text feedback** explaining why the action failed.

---

## 3. Go's Architectural Improvements Over C

Despite the bugs, the Go scripting engine introduces magnificent modern upgrades:
1. **Isolated Script Sandboxing**: Unlike C which shared a global mutable `lua_State` (where a single script could crash the entire MUD), Go instantiates an isolated sandboxed environment, stripping out dangerous operations like local filesystem access (`io`), arbitrary command execution (`os.execute`), and VM bytecode injection (`string.dump`).
2. **Infinite Loop Protection (Script Timeouts)**: Go utilizes a 5-second context timeout (`context.WithTimeout`). If a script executes an infinite loop or stalls, it is safely aborted, and `needsRecreate` triggers a clean state rebuild, ensuring optimal MUD availability.

---

## 4. Concurrency & Thread Safety

- **Engine State Mutex**: To prevent concurrent execution threads from corrupting the Lua stack, all script runs inside `RunScript` are protected by a central `e.mu sync.Mutex`.
- **Thread Safety Risks on Write-Backs**: When reading back character attributes modified by Lua (such as updating HP/gold inside `tableToCharLocked`), the write-back does not acquire the player structure mutex locks, which can cause data corruption if a player is in concurrent combat.

---

## 5. Summary of Recommended Next Steps

1. **Expose `foreachi` in `globals.lua`**:
   Add a compatibility fallback inside `globals.lua` to replicate legacy `foreachi` behavior:
   ```lua
   function foreachi(tbl, fn)
     for i, v in ipairs(tbl) do
       local res = fn(i, v)
       if res then return res end
     end
   end
   ```
2. **Correct the Scripts Root Pathing**:
   Change `dofile("scripts/mob/no_move.lua")` inside `globals.lua` to relative pathing `dofile("mob/no_move.lua")` to resolve the silent loading failure.
3. **Uncomment socket feedback**:
   Uncomment `ctx.Ch.SendMessage("You can't give that here.\r\n")` in `pkg/game/scripts.go` to restore player-facing feedback on failed gives.
