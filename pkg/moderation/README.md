# Dark Pawns Moderation Package

## Overview

The `moderation` package provides comprehensive tools for managing player behavior, handling abuse reports, and maintaining game integrity in Dark Pawns.

## Features

- **Abuse Reporting System**: Players can report other players for various violations
- **Admin Commands**: In-game commands for warnings, mutes, kicks, and bans
- **Audit Logging**: Complete trail of all moderation actions
- **Word Filtering**: Automatic detection and handling of inappropriate content
- **Spam Detection**: Rate limiting and duplicate message detection
- **Player Penalties**: Temporary and permanent restrictions
- **Database Integration**: persistence for all moderation data, on PostgreSQL or embedded SQLite

## Quick Start

### Integration with Main Server

```go
import "github.com/zax0rz/darkpawns/pkg/moderation"

// Initialize moderation system. The dialect travels with the handle: these
// tables are created and queried through the connection pkg/db opened, so they
// are translated by the same dialect value.
database, _ := db.New("sqlite:///data/darkpawns.db") // or a postgres:// DSN
modManager := moderation.NewManager(database.SQLDB(), database.Dialect())

// Check messages for filtered content
filteredMsg, action, shouldBlock := modManager.CheckMessage(playerName, message)
if shouldBlock {
    // Block the message
    return
}

// Record message for spam detection
modManager.RecordMessage(playerName)
```

### Database Schema

The package automatically creates the following tables, on either backend:

1. `abuse_reports` - Player-submitted reports
2. `admin_log` - Audit trail of admin actions
3. `player_penalties` - Active player restrictions
4. `word_filters` - Filtered words and phrases

On SQLite the schema is translated by `pkg/db`'s dialect support: `SERIAL`
becomes `INTEGER PRIMARY KEY AUTOINCREMENT`, `ADD COLUMN IF NOT EXISTS` becomes a
`pragma_table_info` guard, and `NOW()` in a query body is computed in Go and
bound, because SQLite has no such function.

## API Reference

### Types

#### AbuseReport
```go
type AbuseReport struct {
    ID          int
    Reporter    string
    Target      string
    ReportType  ReportType  // harassment, spam, cheating, etc.
    Description string
    RoomVNum    int
    Timestamp   time.Time
    Status      ReportStatus // pending, reviewed, resolved, dismissed
    ReviewedBy  string
    ReviewedAt  *time.Time
    Resolution  string
}
```

#### AdminLogEntry
```go
type AdminLogEntry struct {
    ID        int
    Admin     string
    Action    AdminAction // warn, mute, kick, ban, investigate
    Target    string
    Reason    string
    Duration  *time.Duration
    Timestamp time.Time
    IPAddress string
}
```

#### PlayerPenalty
```go
type PlayerPenalty struct {
    PlayerName  string
    PenaltyType AdminAction
    IssuedAt    time.Time
    ExpiresAt   *time.Time // nil for permanent
    Reason      string
    IssuedBy    string
}
```

### Manager Methods

#### NewManager
```go
func NewManager(conn *sql.DB, dialect db.Dialect) *Manager
```
Creates a new moderation manager. `conn` is the game store's connection and
`dialect` is the SQL flavour it was opened for; a nil `conn` gives the
memory-only manager and the dialect is then never consulted.

#### CheckMessage
```go
func (m *Manager) CheckMessage(playerName, message string) (string, FilterAction, bool)
```
Checks a message for filtered words and spam. Returns:
- Filtered message (with censored content if applicable)
- Action taken (censor, warn, block, log)
- Whether to block the message entirely

#### RecordMessage
```go
func (m *Manager) RecordMessage(playerName string)
```
Records a message for spam detection. Call this after a message passes filtering.

## Configuration

### Word Filters

Word filters can be configured via database or in-memory:

```go
filter := WordFilterEntry{
    Pattern: "badword",
    IsRegex: false,
    Action:  FilterActionCensor, // or warn, block, log
    CreatedBy: "admin",
    CreatedAt: time.Now(),
}
```

### Spam Detection

Configure via `SpamDetectionConfig`:

```go
config := SpamDetectionConfig{
    MessagesPerMinute: 10,               // Threshold for spam
    Action:            FilterActionWarn, // Action when spam detected
}
```

## Examples

### Basic Integration

```go
// Setup
database, _ := db.New("postgres://user:pass@localhost/db")
mod := moderation.NewManager(database.SQLDB(), database.Dialect())

// In your message handler
func handleChatMessage(playerName, message string) (string, error) {
    // Check for filtered content
    filtered, action, block := mod.CheckMessage(playerName, message)
    if block {
        return "", fmt.Errorf("message blocked")
    }
    
    // Record for spam detection
    mod.RecordMessage(playerName)
    
    // Log if action was taken
    if action != moderation.FilterActionLog {
        log.Printf("Filter action %s on message from %s", action, playerName)
    }
    
    return filtered, nil
}
```

### Admin Action Logging

```go
// Log an admin action (simplified example)
func logAdminAction(admin, target, reason string) {
    entry := moderation.AdminLogEntry{
        Admin:     admin,
        Action:    moderation.ActionWarn,
        Target:    target,
        Reason:    reason,
        Timestamp: time.Now(),
    }
    // Save to database or in-memory log
}
```

## Testing

Run the test suite:

```bash
go test ./pkg/moderation/...
```

Tests cover:
- Word filtering and censorship
- Regex pattern matching
- Spam detection logic
- Message checking workflow
- The four tables against a real database: SQLite always, PostgreSQL when
  `DATABASE_URL` points at a disposable database (`backend_test.go`)

## Dependencies

- `github.com/zax0rz/darkpawns/pkg/db` for the connection and its dialect
- Standard Go libraries only, plus the `pkg/db` drivers (lib/pq, modernc.org/sqlite)

## License

Part of the Dark Pawns project. See main project LICENSE for details.