# Claude Code Batch — Run 3: Wired Dead Systems

## Issues
- DP-426: Six heartbeat callbacks never wired (URGENT)
- DP-427: StartAITicker never called (URGENT)
- DP-422: Board system never initialized (URGENT)
- DP-423: WriteMagic never checked (HIGH)
- DP-415: PerformAlias never called (HIGH)
- DP-416: ReadAliases never called on login (HIGH)

## Task: Wire heartbeat callbacks (DP-426)

**File:** `cmd/server/main.go:161-175`

C source (`src/comm.c heartbeat`) fires all system ticks. Go's `main.go` only registers `OnPointUpdate`. Five callbacks are nil:
- `OnEventProcess` — spell delays, combat timer
- `OnWeatherAndTime` — weather cycles, time progression
- `OnAffectUpdate` — affect tick-down
- `OnZoneUpdate` — zone resets
- `OnMobileActivity` — mob AI ticks
- `OnCheckIdlePasswords` — idle timeout
- `OnHuntItems` — item hunt
- `OnFlushPlayerFile` — auto-save

**Fix:** Look at how `OnPointUpdate` is wired in `main.go`. Wire the remaining callbacks the same way. Each callback has a corresponding function in the game layer — find them by searching for the function names that match each callback's purpose.

Key files to check:
- `pkg/game/weather.go` — weather/time updates
- `pkg/game/affects.go` or `limits_condition.go` — affect tick-down
- `pkg/game/world.go` — zone updates
- `pkg/game/ai.go` — mobile activity

**C source:** `src/comm.c — heartbeat()`

## Task: Wire StartAITicker (DP-427)

**File:** `pkg/game/ai.go:181-224` + `cmd/server/main.go`

`StartAITicker()` exists and handles mob AI + starts event queue. Never called.

**Fix:** Call `world.StartAITicker()` in `main.go` after world initialization. Check the function signature and what it needs (likely just the World instance).

**C source:** `src/comm.c — heartbeat()` mobile activity section

## Task: Initialize board system (DP-422)

**File:** `pkg/game/world.go:96-97` + `pkg/game/boards.go:549`

`NewWorld` initializes `Boards` as nil. `GetOrInitBoards()` is never called.

**Fix:** Call `GetOrInitBoards()` during world initialization in `main.go`, or make `Boards` lazy-initialized (call `GetOrInitBoards()` on first access). Check if `GetOrInitBoards` needs any arguments.

**C source:** `src/boards.c — init_boards()`

## Task: Wire WriteMagic for board writes (DP-423)

**File:** `pkg/game/boards.go:565-571` + `pkg/session/`

`genBoard` sets `ch.WriteMagic = magic` but the session layer never checks it. Player input isn't captured into the board buffer.

**Fix:** In the session command processing loop (likely `pkg/session/session_command.go` or `commands.go`), add a check before normal command dispatch:
```
if player.WriteMagic != nil {
    // Route input to board write buffer
    // When write is complete, clear WriteMagic
}
```

Check how other editors (e.g. description editor, alias editor) are wired in the session layer — there should be a pattern for `WriteMagic` or similar flags.

**C source:** `src/boards.c — write_message()` + `src/comm.c` editor input handling

## Task: Wire alias expansion (DP-415)

**File:** `pkg/session/commands.go:325` (`ExecuteCommand`)

`PerformAlias` in `pkg/game/aliases.go:193` is never called. Players define aliases but typing them yields "Unknown command."

**Fix:** At the top of `ExecuteCommand`, before command matching, call `game.PerformAlias` to expand any alias. The function likely takes the player and the command string, and returns the expanded command.

**C source:** `src/alias.c — perform_alias()`

## Task: Load aliases on login (DP-416)

**File:** `pkg/session/session_login.go:173-175`

`RecordToPlayer` loads the player struct but never calls `ReadAliases`. Saved aliases are lost on relog.

**Fix:** After player struct is loaded in `RecordToPlayer`, call `game.ReadAliases(player)` to populate the aliases slice from the saved file.

**C source:** `src/alias.c — read_aliases()`

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
