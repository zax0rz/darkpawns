# Claude Code Batch — Run 4: Security & Ban System

## Issues
- DP-418: WebSocket bypasses IP bans (URGENT)
- DP-420: ValidName stub bypasses profanity filter (URGENT)
- DP-419: Telnet treats all ban types as BanAll (HIGH)
- DP-421: Ban/xnames file paths wrong (MEDIUM)

## Task: Add ban check to WebSocket path (DP-418)

**File:** `pkg/session/session_login.go` (WebSocket connection path)

Telnet listener (`pkg/telnet/listener.go:107`) checks `IsBanned(remoteIP)`. WebSocket session manager never calls it.

**Fix:** In the WebSocket connection handler (where the session is first created), add:
```go
if banManager.IsBanned(remoteIP) {
    // Send ban message and close connection
    return
}
```

Check how the ban manager is accessed — it's likely on the World or a global. Also check how `remoteIP` is extracted from the WebSocket request.

**C source:** `src/comm.c — init_connection()` ban check

## Task: Wire ValidName to ban system (DP-420)

**File:** `pkg/game/merge_bridge.go:103-109`

Global `ValidName()` only checks length. `banManager.ValidName(name)` in `bans.go` correctly checks against the invalid names list.

**Fix:** In the global `ValidName()` function, add a call to the ban manager's `ValidName` method. If the ban manager isn't accessible from the global function, you may need to:
1. Make the ban manager a package-level variable, or
2. Pass it as a parameter, or
3. Route the check through the World instance

Check how `ValidName` is called — if it's called during character creation in the session layer, the session layer likely has access to the World/ban manager.

**C source:** `src/boards.c — Valid_Name()` or `src/db.c` invalid name check

## Task: Implement ban level logic (DP-419)

**File:** `pkg/telnet/listener.go:107`

C source has three ban levels:
- `BAN_ALL` (0x01) — block all connections from this IP
- `BAN_NEW` (0x02) — block new character creation, allow existing
- `BAN_SELECT` (0x04) — block all except approved characters

Go checks `banLevel > 0` — any ban triggers disconnect.

**Fix:** Replace the simple `> 0` check with level-specific logic:
- `BAN_ALL` → disconnect immediately
- `BAN_NEW` → allow connection but block character creation (set a flag on the session)
- `BAN_SELECT` → allow connection only if character is on the approved list

Check `pkg/game/bans.go` for the ban level constants and the `IsBanned`/`GetBanLevel` methods.

**C source:** `src/comm.c — ban check` + `src/ban.c`

## Task: Fix ban/xnames file paths (DP-421)

**File:** `pkg/game/merge_bridge.go:20-24`

`banFilePath = "data/banned"`, `invalidFilePath = "data/invalid"`. Neither exists.

C source uses `lib/etc/banned` and `lib/text/xnames`.

**Fix:** Update paths to match the deployed server layout. Check what's actually on the server at `/opt/darkpawns/lib/etc/` and `/opt/darkpawns/lib/text/`. If the files don't exist yet, create empty ones. Update the constants to point to the correct paths.

**C source:** `src/ban.c — ban_file`, `src/db.c — XNAME_FILE`

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
