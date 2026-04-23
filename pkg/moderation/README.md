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
- **Database Integration**: PostgreSQL persistence for all moderation data

## Quick Start

### Integration with Main Server

```go
import "github.com/zax0rz/darkpawns/pkg/moderation"

// Initialize moderation system
db, _ := sql.Open("postgres", "your-connection-string")
modManager := moderation.NewManager(db)

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

The package automatically creates the following tables:

1. `abuse_reports` - Player-submitted reports
2. `admin_log` - Audit trail of admin actions
3. `player_penalties` - Active player restrictions
4. `word_filters` - Filtered words and phrases

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
func NewManager(db *sql.DB) *Manager
```
Creates a new moderation manager with optional database connection.

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
    MessagesPerMinute: 10,    // Threshold for spam
    DuplicateWindow:   5 * time.Second, // Window for duplicate detection
    Action:           FilterActionWarn, // Action when spam detected
}
```

## Examples

### Basic Integration

```go
// Setup
db, _ := sql.Open("postgres", "postgres://user:pass@localhost/db")
mod := moderation.NewManager(db)

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

## Dependencies

- PostgreSQL (optional, for persistence)
- Standard Go libraries only

## License

Part of the Dark Pawns project. See main project LICENSE for details.