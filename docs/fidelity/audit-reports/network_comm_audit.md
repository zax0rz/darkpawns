# Port Fidelity Audit: Module 19 (`comm.c` / Network & Game Loop)

This audit examines the port fidelity between the legacy C source file `src/comm.c` (the core networking, socket connection pump, and game heartbeat loop) and its Go counterparts in `cmd/server/`, `pkg/session/`, `pkg/telnet/`, and `pkg/engine/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/comm.c` (2,637 lines)
- **Functions**: `dp_main` (legacy main), `init_game`, `init_socket`, `get_max_players`, `game_loop` (main MUD loop), `room_activity`, `object_activity`, `heartbeat` (tick dispatcher), `timediff`, `timeadd`, `new_descriptor`, `write_to_descriptor`, `process_input`, `process_output`, `close_socket`, `flush_queues`, `nonblock`.

### Go Port Files
Go distributes this low-level socket management and synchronous tick cycle across multi-threaded asynchronous goroutines and packages:
- `cmd/server/main.go` (Main server orchestrator, HTTP and WebSockets routing, manual wiring)
- `pkg/engine/gameloop.go` (Game tick orchestrator with 100ms ticker, pulse modulo dispatchers)
- `pkg/telnet/listener.go` (Raw TCP Telnet server socket listener, MSSP and GMCP negotiations)
- `pkg/session/manager.go` (WebSocket connection management, player registry, hook wiring)
- `pkg/session/session_manager.go` (Session lifecycles, global broadcasts: `SendToAll`/`SendToOutdoor`)
- `pkg/session/session_pump.go` (Read/write goroutines per connection, JSON wrapping, ANSI stripping)
- `pkg/session/session_login.go` (Login credentials validation, guest bypass, rate limits)
- `pkg/session/char_creation.go` (Nanny state machine for character selection)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Hollow Game Loop Heartbeat (Weather, Events, Room/Object Scripts are DEAD)
- **Source Context**: `cmd/server/main.go#L162-L172` (`NewGameLoop`)
- **Fidelity Bug**: The Go game loop orchestrator `pkg/engine/gameloop.go` supports all legacy heartbeat callbacks (`OnZoneUpdate`, `OnRoomActivity`, `OnObjectActivity`, `OnWeatherAndTime`, `OnAffectUpdate`, `OnEventProcess`, etc.). However, `main.go` **only registers `OnPointUpdate`**:
  ```go
  gameLoop := engine.NewGameLoop(engine.GameLoopCallbacks{
      OnPointUpdate: func() {
          gameWorld.PointUpdate()
      },
      OnPerformViolence: func() {
          // Combat engine handles its own 2s tick via CombatEngine.Start()
      },
      OnMobileActivity: func() {
          // Future: mob AI wandering, speech triggers
      },
  })
  ```
- **Impact**:
  - **Frozen Weather and Time**: Mud hours and weather changes (`OnWeatherAndTime`) never tick; time remains permanently frozen.
  - **Dead Priority Event Queue**: Scheduled priority game events (`OnEventProcess`) are never processed; micro-second event queues are completely inactive.
  - **Frozen Status Affects**: Buffs and debuffs on characters (`OnAffectUpdate`) never expire.
  - **Disabled Underwater Drowning**: Drowning in underwater rooms and damage from hot water sectors (`OnRoomActivity`) never execute.
  - **Disabled Room and Object Scripts**: Room and object script on-pulse triggers (`ROOM_SCRIPT`/`OBJ_SCRIPT` with `ONPULSE` flags) never run.

### 2. WebSocket Site Ban Bypass
- **Source Context**: `pkg/session/manager.go#L397-L433` (`HandleWebSocket`)
- **Fidelity Bug**: In legacy C, site/IP bans (`isbanned`) are checked at the socket level inside `new_descriptor`, instantly dropping blacklisted connections.
  The Go telnet socket listener in `telnet/listener.go` replicates this check. However, **the WebSocket handler completely bypasses site bans**, allowing blacklisted IP ranges to freely connect and interact via the Web client interface.
- **Impact**: Server bans are easily bypassed by using the web interface instead of a direct telnet client, posing a severe security boundary exploit.

### 3. Missing MCCP (MUD Client Compression Protocol) in Telnet
- **Source Context**: `pkg/telnet/listener.go`
- **Fidelity Bug**: Legacy C implements MCCP (zlib-based stream compression) to minimize network bandwidth usage during massive output dumps. The Go telnet server completely omits MCCP support (it only supports plain TCP writing).
- **Impact**: Telnet players on mobile connections or high-latency channels experience significantly higher bandwidth costs and potential buffer congestion during massive room views or combat dumps.

### 4. Race Option Name Drift (Kender vs Halfling)
- **Source Context**: `pkg/session/char_creation.go#L367` (`getRaceOptions`)
- **Fidelity Bug**: The character creation nanny state lists race ID 3 as `"Halfling"` instead of `"Kender"`.
- **Impact**: This creates a lore inconsistency with the internal `RaceKender = 3` constant (`pkg/game/character.go`) and legacy C checks.

### 5. Hollow Wizard Commands: Freeze, Thaw, and Newbie
- **Source Context**: `pkg/session/wiz_system.go#L276-L285` (`cmdWizutil`), `pkg/session/wiz_system.go#L357-L377` (`cmdNewbie`)
- **Fidelity Bug**: 
  - **Freeze & Thaw**: The wizard utility sub-commands `freeze` and `thaw` print messages to the wizard and target session (e.g. `"You feel frozen!"`), but they **completely omit setting or clearing the target player's `PlrFrozen` state**.
  - **Newbie**: The `newbie` command logs the action and prints `"Newbied."`, but **does not actually spawn or give any starting items** (tunic, bread, skin, club) to the target player, leaving the command entirely non-functional.
- **Impact**: Wizards cannot freeze rule-breaking players or gift newbie packages, rendering these utility commands completely decorative.

### 6. Total Absence of Frozen Player Constraints (`PlrFrozen`)
- **Source Context**: `pkg/session/` (command interpreter) and `pkg/game/` (nanny entry)
- **Fidelity Bug**: In legacy C, players flagged as `PLR_FROZEN` are heavily restricted:
  - The command interpreter blocks them from executing any standard command, printing `"You are totally frozen!"`.
  - The login nanny forces them to spawn inside the isolated frozen start room `1202`.
  In Go, although the `PlrFrozen` constant exists, the command interpreter never checks it, nor does character loading force their room location to `1202`.
- **Impact**: Even if a wizard could successfully toggle the `PlrFrozen` flag, frozen players could still log in, wander, and execute all combat and social commands with zero restrictions.

---

## 3. Go Improvements Over C

### 1. Concurrent Goroutines per Connection
- **Fidelity Improvement**: Legacy C relied on a single-threaded synchronous `select()` loop that was vulnerable to freezing if a single client suffered socket write blocks. Go replaces this with native multi-threaded goroutines (`readPump`/`writePump`) per player session, allowing concurrent I/O scaling across multi-core processors.

### 2. Safe Cryptographic Password Hashing
- **Fidelity Improvement**: Legacy C utilized weak DES crypt functions for password storage in pfiles. Go upgrades this to industry-standard, secure `bcrypt` hashing for database-backed authentication.

### 3. Native Telnet Protocol Improvements (MSSP & GMCP)
- **Fidelity Improvement**: The Go telnet listener supports native MSSP (MUD Server Status Protocol) for discovery crawling and GMCP (Generic MUD Communication Protocol) to stream structured JSON player data to modern clients.

---

## 4. Concurrency & Thread Safety

- **Session Takeover Concurrency Lock**:
  - In `pkg/session/manager.go#Register`, during a link-dead session takeover, the manager must forcibly lock `m.mu` but then sleep to wait for a 5-second takeover probe. 
  - To avoid deadlocking during this wait window (and allowing other players to continue accessing the manager), the manager unlocks `m.mu` inside a loop, allowing other goroutines to register, before re-acquiring the lock:
    ```go
    for time.Now().Before(oldSess.takeOverAt) {
        m.mu.Unlock()
        time.Sleep(200 * time.Millisecond)
        m.mu.Lock()
        ...
    }
    ```
  - This avoids blocking the entire MUD server during agent-takeover handshakes.

---

## 5. Summary of Recommended Fixes

1. **Wire Hollow Heartbeat Callbacks**:
   In `cmd/server/main.go`, properly populate the callbacks inside `NewGameLoop` to call their game-world counterparts:
   ```go
   OnWeatherAndTime: func() { gameWorld.WeatherAndTime() },
   OnAffectUpdate:   func() { gameWorld.AffectUpdate() },
   OnRoomActivity:   func() { gameWorld.RoomActivity() },
   OnObjectActivity: func() { gameWorld.ObjectActivity() },
   OnEventProcess:   func() { gameWorld.EventProcess() },
   ```
2. **Apply Site Bans to WebSocket Handler**:
   In `pkg/session/manager.go#HandleWebSocket`, invoke the ban manager check prior to creating the session:
   ```go
   if banLevel := m.GetBanManager().IsBanned(ip); banLevel > 0 {
       _ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "your IP is banned"))
       _ = conn.Close()
       return
   }
   ```
3. **Synchronize Nanny Screen Names**:
   Change `"Halfling"` back to `"Kender"` in the character creation race options to preserve thematic fidelity.
4. **Implement Missing Wizard Command Actions**:
   - Update `cmdWizutil` to actually set and clear the `PlrFrozen` bit on the target player's flags, and write the change to the database.
   - Update `cmdNewbie` to instantiate and equip starter items (tunic, skin, bread, club) on the target character using the world spawning system.
5. **Enforce `PlrFrozen` Restrictions**:
   - Update the session command router (`HandleMessage`/`handleCommand`) to block players with `PlrFrozen` enabled from performing any action.
   - Add a check at login to force frozen players to spawn in room `1202`.
