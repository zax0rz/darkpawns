# Dependency Injection for Command System

## Problem
There was a circular dependency between:
- `pkg/session` → imports `pkg/command` (in `commands.go`)
- `pkg/command` → imports `pkg/session` (in all command files)

## Solution
Implemented dependency injection using an interface:

### 1. SessionInterface (`pkg/command/interface.go`)
Defines the methods that command handlers need from a session:
```go
type SessionInterface interface {
    GetPlayer() *game.Player
    SendMessage(message string) error
    Send(message string)
    MarkDirty(vars ...string)
    GetManager() interface{}
}
```

### 2. Session Implementation (`pkg/session/manager.go`)
Added methods to `Session` struct to implement `SessionInterface`:
```go
func (s *Session) GetPlayer() *game.Player
func (s *Session) SendMessage(message string) error
func (s *Session) Send(message string)
func (s *Session) MarkDirty(vars ...string)
func (s *Session) GetManager() interface{}
```

### 3. Updated Command Handlers
All command functions now accept `SessionInterface` instead of `*session.Session`:
```go
// Before:
func cmdSkills(s *session.Session, args []string) error

// After:
func cmdSkills(s SessionInterface, args []string) error
```

Updated all `s.Player` references to `s.GetPlayer()`.

### 4. Removed Session Import
Removed `github.com/zax0rz/darkpawns/pkg/session` import from all command files.

## Benefits
1. **Broken Circular Dependency**: `pkg/command` no longer imports `pkg/session`
2. **Testability**: Command handlers can be tested with mock implementations
3. **Flexibility**: Different session implementations can be used
4. **Clear Contract**: Explicit interface defines what commands need

## Usage

### Adding New Commands
1. Define function with `SessionInterface` parameter:
   ```go
   func cmdNewCommand(s SessionInterface, args []string) error
   ```
2. Use `s.GetPlayer()` instead of `s.Player`
3. Use `s.SendMessage()` or `s.Send()` for output

### Testing Commands
Create mock implementation:
```go
type MockSession struct {
    player *game.Player
    messages []string
}

func (m *MockSession) GetPlayer() *game.Player { return m.player }
func (m *MockSession) SendMessage(msg string) error {
    m.messages = append(m.messages, msg)
    return nil
}
// ... implement other interface methods
```

## Notes
- The `GetManager()` method returns `interface{}` because admin commands need access to the session manager
- `MarkDirty()` is for agent variable subscriptions
- `Send()` and `SendMessage()` both send messages to the client (different naming conventions in existing code)