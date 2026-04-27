# Dark Pawns Moderation System

## Overview

The Dark Pawns moderation system provides tools for managing player behavior, handling abuse reports, and maintaining a positive gaming environment. The system includes both in-game commands and administrative tools.

## Architecture

### Components

1. **Moderation Manager** (`pkg/moderation/`) - Core moderation logic
2. **Admin Commands** (`pkg/command/admin_commands.go`) - In-game admin commands
3. **Database Schema** - PostgreSQL tables for persistence
4. **Admin Web Interface** (Optional) - Web-based moderation dashboard
5. **Docker Compose Stack** - Standalone moderation services

### Data Flow

```
Player Report → Abuse Report → Admin Review → Action Taken → Audit Log
     ↓              ↓              ↓              ↓              ↓
  Chat Logs → Word Filter → Spam Detection → Penalties → Player Notes
```

## Installation

### Option 1: Integrated with Main Server

1. Ensure PostgreSQL is running
2. Update the main server to initialize moderation tables:

```go
// In main.go
import "github.com/zax0rz/darkpawns/pkg/moderation"

// After creating database connection
modManager := moderation.NewManager(database)
adminCommands := command.NewAdminCommands(manager, modManager)
adminCommands.RegisterCommands()
```

### Option 2: Standalone Moderation Stack

```bash
# Start the moderation stack
docker-compose -f docker-compose.moderation.yml up -d

# The stack includes:
# - PostgreSQL database on port 5433
# - Admin web interface on port 3000 (placeholder)
# - Moderation API on port 8081
# - Redis cache on port 6379
```

## Usage

### Player Commands

#### Report Abuse
```
report <player> <type> [description]

Types:
  harassment - Unwanted behavior, bullying
  spam       - Excessive messages, advertising
  cheating   - Exploits, hacks, unfair advantage
  hate_speech - Discriminatory language
  exploit    - Game bugs, glitches
  other      - Anything else

Examples:
  report bob harassment "Following me and insulting"
  report alice spam "Advertising gold selling"
  report charlie cheating "Using speed hack"
```

### Admin Commands

#### Warning
```
warn <player> <reason>
```
Sends a warning to the player. Example: `warn bob "Stop harassing other players"`

#### Mute
```
mute <player> <duration> [reason]
```
Temporarily mutes a player. Duration examples: `5m`, `1h`, `1d`. Example: `mute alice 30m "Spamming chat"`

#### Kick
```
kick <player> <reason>
```
Disconnects a player from the game. Example: `kick charlie "Repeated harassment after warning"`

#### Ban
```
ban <player> <duration> [reason]
```
Bans a player. Use `permanent` for indefinite ban. Example: `ban bob permanent "Using cheats"`

#### Investigate
```
investigate <player>
```
Shows player information, online status, location, and penalty history.

#### List Reports
```
reports
```
Shows pending abuse reports that need review.

#### List Penalties
```
penalties
```
Shows active player penalties (mutes, bans, warnings).

#### Word Filter Management
```
filter add <pattern> [regex] [action]
filter remove <id>
filter list
```
Manages filtered words. Actions: `censor`, `warn`, `block`, `log`.

#### Spam Configuration
```
spamconfig <threshold> [window] [action]
```
Configures spam detection. Example: `spamconfig 15 10s block`

### Admin Web Interface (Planned)

The web interface would provide:
- Dashboard with statistics
- Report review queue
- Player search and investigation
- Chat log viewer
- Real-time monitoring
- Bulk operations

## Database Schema

### Key Tables

1. **abuse_reports** - Player-submitted reports
2. **admin_log** - Audit trail of all admin actions
3. **player_penalties** - Active mutes, bans, warnings
4. **word_filters** - Filtered words and actions
5. **chat_logs** - Message history for investigation
6. **player_notes** - Moderator notes on players

### Retention Policies

- Chat logs: 30 days
- Abuse reports: 90 days after resolution
- Admin logs: 1 year
- Player penalties: Until expiration + 30 days

## Automated Moderation

### Word Filtering

The system automatically checks messages against word filters:
- **Censor**: Replaces matched text with `****`
- **Warn**: Sends warning to player, logs incident
- **Block**: Prevents message from being sent
- **Log**: Records incident for review

### Spam Detection

Detects excessive messaging:
- Default: 10 messages per minute
- Configurable threshold and window
- Actions: warn, mute, or block

### Penalty Enforcement

- **Mutes**: Prevent chatting for duration
- **Bans**: Prevent login for duration
- **Warnings**: Tracked for pattern detection
- **Automatic escalation**: Repeated offenses trigger stronger penalties

## Integration Points

### With Game Systems

1. **Chat System**: Intercepts and filters messages
2. **Login System**: Checks for active bans
3. **Command System**: Processes admin commands
4. **Combat System**: Can mute combat spam
5. **Scripting System**: Lua hooks for custom moderation

### With External Systems

1. **Discord Webhooks**: Report notifications
2. **Email Alerts**: For critical incidents
3. **API Endpoints**: For external tools
4. **Metrics Export**: To monitoring systems

## Best Practices

### For Moderators

1. **Document everything**: Always include reasons for actions
2. **Be consistent**: Apply rules equally to all players
3. **Escalate gradually**: Warning → Mute → Kick → Ban
4. **Investigate first**: Use `investigate` before taking action
5. **Communicate clearly**: Explain rules and consequences

### For Players

1. **Use report system**: Don't engage with rule-breakers
2. **Provide details**: Include context in reports
3. **Respect decisions**: Appeal through proper channels
4. **Know the rules**: Read server rules and guidelines

## Development

### Adding New Features

1. Extend `moderation.Manager` for new functionality
2. Add commands to `AdminCommands`
3. Update database schema if needed
4. Add tests in `pkg/moderation/*_test.go`

### Testing

```bash
# Run moderation tests
go test ./pkg/moderation/...

# Test with specific flags
go test -v ./pkg/moderation -run TestWordFilter
```

### API Documentation

See `pkg/moderation/types.go` for data structures and `pkg/moderation/manager.go` for API methods.

## Troubleshooting

### Common Issues

1. **Commands not working**: Check admin permissions in `isAdmin()` function
2. **Database errors**: Verify PostgreSQL connection and schema
3. **Word filters not applying**: Check pattern matching and regex flags
4. **Spam detection too sensitive**: Adjust `spamconfig` settings

### Logs

- Moderation actions: Check server logs for `ADMIN:` prefix
- Database errors: PostgreSQL error logs
- System errors: Application logs

## Security Considerations

1. **Admin authentication**: Secure admin account creation
2. **Audit trails**: All actions are logged and immutable
3. **Data privacy**: Chat logs should be access-controlled
4. **API security**: Use API keys for external integrations
5. **Input validation**: Sanitize all user inputs

## Future Enhancements

1. **Machine learning**: AI-based toxicity detection
2. **Reputation system**: Player trust scores
3. **Appeal system**: Formal ban appeals
4. **Temporary admin**: Limited-time moderator roles
5. **Cross-server bans**: Shared ban lists across instances
6. **Real-time alerts**: Push notifications for critical reports