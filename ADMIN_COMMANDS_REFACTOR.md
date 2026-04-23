# Admin Commands Refactoring

## Summary
Refactored admin commands to break circular dependency with session manager by using interfaces instead of concrete types.

## Changes Made

### 1. Updated `common.CommandSession` Interface
- Added `GetPlayerRoomVNum() int` method to get player's room location
- Added `HasPlayer() bool` method to check if session has a player
- Added `Close()` method to close sessions
- These methods replace direct access to `GetPlayer().FieldName`

### 2. Updated `session.Session` Implementation
- Added `GetPlayerRoomVNum()` method that calls `player.GetRoomVNum()`
- Added `HasPlayer()` method that checks if `player != nil`
- Added `Close()` method that calls the existing `close()` method

### 3. Refactored `admin_commands.go`
- Replaced all `GetPlayer().Name` calls with `GetPlayerName()`
- Replaced all `GetPlayer().GetRoomVNum()` calls with `GetPlayerRoomVNum()`
- Replaced `GetPlayer() == nil` checks with `!HasPlayer()`
- Removed unused imports (`sync`, `game`)
- Added missing `log` import
- Fixed unused variable warnings by using `_ =` for stub implementations
- Updated `cmdInvestigate` to use interface methods instead of direct player field access

### 4. Dependency Injection
- `AdminCommands` struct already uses `common.CommandManager` interface (not concrete `*session.Manager`)
- Constructor `NewAdminCommands` accepts `common.CommandManager` interface
- This allows any type implementing the interface to be injected
- No direct dependency on `session` package

## Benefits

1. **Broken Circular Dependency**: Admin commands no longer depend on concrete `session.Manager` type
2. **Better Testability**: Can use mock implementations of `CommandManager` and `CommandSession`
3. **Cleaner Interface**: `CommandSession` interface now provides all needed methods
4. **Type Safety**: No more type assertions on `GetPlayer()` return value

## Compilation Status
- `admin_commands.go` compiles successfully
- `door_commands.go` and `shop_commands.go` already use interfaces correctly
- The `session` package implements the required interface methods
- Shop system has unrelated compilation errors (interface type assertions)

## Testing
The refactored code maintains the same functionality:
- Admin commands can be registered with any `CommandManager`
- Command handlers receive `CommandSession` interface
- All player information accessed through interface methods
- No breaking changes to existing code

## Notes
- The `cmdInvestigate` function shows "[Requires player interface]" for level and health fields
- To fully implement, would need to add `GetPlayerLevel()` and `GetPlayerHealth()` methods to `CommandSession`
- Or create a `Player` interface in `common` package with these methods