# Port Fidelity Audit: Module 14 (`ban.c`)

This audit examines the port fidelity between the legacy C source file `src/ban.c` and its Go counterparts in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/ban.c` (313 lines)
- **Functions**: `load_banned` (loads IP bans from file), `isbanned` (checks IP against bans), `write_ban_list`, `do_ban` (bans a site), `do_unban` (unbans a site), `Valid_Name` (verifies duplicate and invalid/offensive names), `Read_Invalid_List` (loads invalid name substrings from `XNAME_FILE`).

### Go Port Files
- **Command Wrappers & Bridge**:
  - `pkg/game/merge_bridge.go` (Declares the global `banManager` singleton, defines package-level wrappers `LoadBanned`, `ReadInvalidList`, `AddBan`, `RemoveBan`, `IsBanned`, and the stubbed `ValidName`)
- **Game Logic**:
  - `pkg/game/bans.go` (Defines `BanEntry`, `BanManager` struct, and methods `LoadBanned`, `WriteBanList`, `IsBanned`, `AddBan`, `RemoveBan`, `ReadInvalidList`, and the correct `ValidName` profanity check)
- **Active Gates**:
  - `pkg/telnet/listener.go` (TCP Telnet listener that checks `IsBanned` at connect boundary)
  - `pkg/session/session_login.go` (WebSocket connection path; completely bypasses site bans, and calls the stubbed `ValidName`)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Critical Gate Bypass: WebSocket Site/IP Bans Ignored
- **Source Context**: `pkg/session/session_login.go#L19-L295` (`handleLogin`), `pkg/session/manager.go`
- **Fidelity Bug**: IP-based site bans are completely ignored when players connect via the modern WebSocket web client. The session manager upgraded and managed sessions but **never** checks the `BanManager.IsBanned` database. Only the raw Telnet TCP port executes this gate. As a result, any banned player can completely bypass their IP ban simply by loading the web client (WebSocket path).

### 2. Blunt Hammer Gate on Telnet (Destroying `BAN_NEW` and `BAN_SELECT` Logic)
- **Source Context**: `pkg/telnet/listener.go#L106-L118`
- **Logic Gap**: Legacy C supports three types of IP bans:
  - `BAN_ALL` (full disconnect).
  - `BAN_NEW` (only blocks new character creation, returning players can still connect).
  - `BAN_SELECT` (only allows registered, approved characters from that IP).
- **Fidelity Bug**: In `listener.go`, the Telnet gate instantly closes any connection where `banLevel > 0`:
  ```go
  if banLevel := manager.GetBanManager().IsBanned(remoteIP); banLevel > 0 {
      _ = conn.Close()
  ```
  This immediately locks out returning players or approved characters if a site is banned with `BAN_NEW` or `BAN_SELECT`, rendering the selective ban levels functionally identical to a blunt `BAN_ALL` ban.

### 3. Invalid Name List (Profanity Filter) is a Mock/Stub (Dead Security Check)
- **Source Context**: `pkg/game/merge_bridge.go#L103-L109` (`ValidName`)
- **Fidelity Bug**: In `merge_bridge.go`, the global package-level function `ValidName(name string)` is a stub that only checks name length:
  ```go
  func ValidName(name string) bool {
      if len(name) < 2 || len(name) > 20 {
          slog.Warn("Invalid name length", "name", name)
          return false
      }
      return true
  }
  ```
  It **completely ignores** the `banManager.ValidName(name)` function implemented in `pkg/game/bans.go`. Because the session login layer (`session_login.go#L144`) calls this stubbed global function, the MUD's invalid/offensive name filter is completely dead, allowing players to create characters with highly offensive or profanity-laced names in production.

### 4. Non-Existent and Mismapped File Paths
- **Source Context**: `pkg/game/merge_bridge.go#L20-L24`
- **Fidelity Bug**:
  - `banFilePath` points to `"data/banned"` and `invalidFilePath` points to `"data/invalid"`.
  - The folder `./data/` does not exist in the authoritative workspace, meaning `LoadBanned` and `ReadInvalidList` will fail to find these files on boot.
  - Furthermore, in legacy C, the invalid names file is located at `lib/text/xnames`. The Go port looks in `data/invalid`, missing the existing authoritative assets folder entirely.

---

## 3. Go Improvements Over C

### 1. Memory Safety
- **Fidelity Improvement**: Legacy C managed bans via a singly-linked list (`struct ban_list_element *next`) utilizing manual `CREATE`/`FREE` and nested `REMOVE_FROM_LIST` pointer manipulation macros. Go uses a clean, safe slice `[]BanEntry` inside the `BanManager` struct, eliminating dangling pointer and memory leak vulnerabilities.

### 2. Stack Overflow Immunity
- **Fidelity Improvement**: In C, `write_ban_list` called the recursive helper `_write_one_node` to write the list in reverse order, presenting potential stack-overflow vulnerabilities if the ban list grew extremely large. Go replaces this with a simple reverse loop (`for i := len(bm.bans) - 1; i >= 0; i--`) which executes in $O(1)$ stack space.

### 3. Object-Oriented Encapsulation
- **Fidelity Improvement**: Legacy C relied on global variables `ban_list` and `invalid_list`. The Go port encapsulates this state in a clean, testable `BanManager` struct, enabling clean dependency injection.

---

## 4. Concurrency & Thread Safety

- **Read/Write Races on Ban Records**:
  - `BanManager` methods like `AddBan` and `RemoveBan` modify the `bans` slice.
  - Concurrently, connection listeners (WebSocket and Telnet threads) call `IsBanned` to check incoming IPs.
  - Because Go slices are not thread-safe for concurrent read/write operations, a read-write mutex (`sync.RWMutex`) must be added to `BanManager` to protect slice lookups and mutations.

---

## 5. Summary of Recommended Fixes

1. **Unify Ban Gates for Telnet and WebSockets**:
   Enforce the IP ban check at the connection boundary of both Telnet (`listener.go`) and WebSockets (`manager.go#HandleWebSocket`) using a unified session-manager gate.
2. **Correct selective Ban Levels**:
   Update connection handshakes to only disconnect immediately if `banLevel == BanAll`. For `BanNew`, allow the connection but reject new character creation. For `BanSelect`, disconnect only if the logging-in character name is not present in an approved list.
3. **Restore `banManager` check in `ValidName`**:
   Fix the stubbed package-level `ValidName` in `pkg/game/merge_bridge.go` to delegate directly to `banManager.ValidName(name)` so that profanities and restricted names are blocked.
4. **Fix File Paths**:
   Point `banFilePath` to `"lib/etc/banned"` and `invalidFilePath` to `"lib/text/xnames"` to match standard CircleMUD directory structures and load existing game files.
5. **Thread-Safe Mutexing**:
   Add a `sync.RWMutex` to the `BanManager` struct to prevent data races between concurrent players connecting and admins adding/removing bans.
