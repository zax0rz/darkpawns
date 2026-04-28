---
title: "WebSocket Protocol"
description: "Complete WebSocket protocol specification for Dark Pawns agents"
date: 2026-04-22
draft: false
section: "docs"
---

# WebSocket Protocol Specification

Dark Pawns uses a WebSocket-based protocol for real-time communication between clients and the game server. The protocol supports both human players (text mode) and AI agents (JSON mode).

## Connection Details

- **URL**: `ws://localhost:4350/ws` (development) or `wss://darkpawns.labz0rz.com/ws` (production)
- **Protocol**: WebSocket (RFC 6455)
- **Message Format**: JSON for agents, plain text for humans
- **Rate Limit**: 10 commands per second per connection

## Message Types

### 1. Login (`type: "login"`)

Required for all connections. Determines whether the client is a human player or an AI agent.

**Request:**
```json
{
  "type": "login",
  "data": {
    "player_name": "brenda69",
    "api_key": "agent-key-123",
    "mode": "agent"
  }
}
```

**Response (success):**
```json
{
  "type": "login_response",
  "data": {
    "success": true,
    "message": "Welcome to Dark Pawns, brenda69!"
  }
}
```

**Response (error):**
```json
{
  "type": "error",
  "data": {
    "code": "AUTH_FAILED",
    "message": "Invalid API key"
  }
}
```

### 2. Command (`type: "command"`)

Send game commands to the server.

**Request:**
```json
{
  "type": "command",
  "data": {
    "command": "look",
    "args": []
  }
}
```

**Response:**
```json
{
  "type": "command_response",
  "data": {
    "success": true,
    "message": "You look around...",
    "output": "The Town Square\nYou are in the bustling town square. Exits: north, east, south, west."
  }
}
```

### 3. Subscription (`type: "subscribe"`)

Agents can subscribe to game state variables to receive automatic updates.

**Request:**
```json
{
  "type": "subscribe",
  "data": {
    "variables": ["HEALTH", "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS"]
  }
}
```

**Response:**
```json
{
  "type": "subscription_response",
  "data": {
    "success": true,
    "subscribed": ["HEALTH", "ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS"]
  }
}
```

### 4. State Update (`type: "state_update"`)

Server pushes state updates to subscribed agents when variables change.

**Message:**
```json
{
  "type": "state_update",
  "data": {
    "HEALTH": 85,
    "MAX_HEALTH": 100,
    "ROOM_VNUM": 3001,
    "ROOM_NAME": "The Town Square",
    "ROOM_EXITS": ["north", "east", "south", "west"],
    "ROOM_MOBS": [
      {
        "vnum": 1001,
        "name": "a town guard",
        "short_desc": "A vigilant town guard stands here.",
        "health": 100
      }
    ]
  }
}
```

## Available State Variables

Agents can subscribe to these variables:

| Variable | Type | Description |
|----------|------|-------------|
| `HEALTH` | integer | Current health points |
| `MAX_HEALTH` | integer | Maximum health points |
| `MANA` | integer | Current mana points |
| `LEVEL` | integer | Character level |
| `ROOM_VNUM` | integer | Current room virtual number |
| `ROOM_NAME` | string | Current room name |
| `ROOM_EXITS` | array | Available exits from current room |
| `ROOM_MOBS` | array | Mobs in current room (with `target_string` for combat) |
| `ROOM_ITEMS` | array | Items in current room |
| `FIGHTING` | object | Current combat target (null if not fighting) |
| `INVENTORY` | array | Items in inventory |
| `EQUIPMENT` | object | Equipped items by slot |
| `EVENTS` | array | Recent game events |

## Rate Limiting

- **Command Rate**: 10 commands per second (token bucket algorithm)
- **Connection Limit**: 5 concurrent connections per IP
- **Message Size**: 16KB maximum per message

Rate limit headers are included in error responses:
```json
{
  "type": "error",
  "data": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded",
    "retry_after": 1.5
  }
}
```

## Error Handling

All error responses follow this format:
```json
{
  "type": "error",
  "data": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {}  // Optional additional details
  }
}
```

Common error codes:
- `AUTH_FAILED`: Invalid API key or authentication failure
- `RATE_LIMITED`: Rate limit exceeded
- `INVALID_COMMAND`: Unknown or malformed command
- `PLAYER_NOT_FOUND`: Player doesn't exist
- `INTERNAL_ERROR`: Server error

## Best Practices for Agents

1. **Always subscribe to state variables** you need rather than polling with commands
2. **Handle rate limiting** with exponential backoff
3. **Maintain connection state** and reconnect on failure
4. **Use the `target_string` field** from `ROOM_MOBS` for combat commands
5. **Implement health-based decision making** (rest when low health)
6. **Log all actions** for debugging and learning

## Example Agent State Machine

```python
# Pseudo-code for agent decision making
def decide_action(state):
    if state.fighting:
        return "hit " + state.fighting.target_string
    elif state.health < state.max_health * 0.3:
        return "rest"
    elif state.room_items:
        return "get " + state.room_items[0].name
    elif state.room_exits:
        return choose_exit(state.room_exits)
    else:
        return "look"
```

## Testing Your Agent

Use the provided test script:
```bash
# Test agent connection
python3 -m scripts.test_agent --name test-agent --key your-api-key

# Test with specific commands
python3 -m scripts.test_agent --command "look;north;east"

# Test rate limiting
python3 -m scripts.test_agent --stress-test
```

## Next Steps

- Read the [Example Agents](/agents/examples/) page for complete implementations
- Check the [API Reference](/api/) for detailed endpoint documentation
- Join the [Discord community](https://discord.gg/darkpawns) for help and discussion
