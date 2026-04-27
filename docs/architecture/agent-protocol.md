---
tags: [active]
---
# Agent Protocol

> **Status:** Fully implemented (Phase 4 complete).
> Agents connect via WebSocket with API key authentication.

---

## Overview

Agents connect via the same WebSocket endpoint as human players (`ws://host/ws`).
The difference is in auth and the structured state they receive.

Agents authenticate with API keys and can subscribe to game state variables.

---

## Message Format

### Client → Server

**Login (Agent Mode):**
```json
{"type": "login", "data": {"player_name": "BotName", "api_key": "dp_abc123...", "mode": "agent"}}
```

**Subscribe to Variables:**
```json
{"type": "subscribe", "data": {"variables": ["HEALTH", "ROOM_VNUM", "ROOM_MOBS"]}}
```

**Command:**
```json
{"type": "command", "data": {"command": "look"}}
{"type": "command", "data": {"command": "north"}}
{"type": "command", "data": {"command": "hit", "args": ["goblin"]}}
{"type": "command", "data": {"command": "flee"}}
{"type": "command", "data": {"command": "say", "args": ["hello"]}}
```

### Server → Client

**State (sent on login):**
```json
{
  "type": "state",
  "data": {
    "player": {"name": "...", "health": 100, "max_health": 100, "level": 1},
    "room": {"vnum": 3001, "name": "...", "description": "...", "exits": ["north","east"]}
  }
}
```

**Variables (sent to agents after subscription):**
```json
{
  "type": "vars",
  "data": {
    "HEALTH": 100,
    "MAX_HEALTH": 100,
    "ROOM_VNUM": 3001,
    "ROOM_NAME": "A Dark Corridor",
    "ROOM_EXITS": ["north", "east"],
    "ROOM_MOBS": [
      {
        "name": "a goblin",
        "instance_id": "mob_3001_0",
        "target_string": "goblin",
        "vnum": 3001,
        "fighting": false
      }
    ]
  }
}
```

**Event:**
```json
{"type": "event", "data": {"type": "combat", "from": "goblin", "text": "A goblin hits you!"}}
{"type": "event", "data": {"type": "enter", "from": "PlayerName", "text": "PlayerName has arrived."}}
{"type": "event", "data": {"type": "flee", "from": "PlayerName", "text": "PlayerName flees north!"}}
```

**Error:**
```json
{"type": "error", "data": {"message": "not authenticated"}}
```

**Text:**
```json
{"type": "text", "data": {"text": "You see a dark corridor stretching north and east."}}
```

---

## Available Variables

Agents can subscribe to these variables:
- `HEALTH`, `MAX_HEALTH` - Current and maximum health
- `MANA`, `MAX_MANA` - Current and maximum mana
- `LEVEL` - Player level
- `EXP` - Current experience points
- `ROOM_VNUM` - Room virtual number
- `ROOM_NAME` - Room name
- `ROOM_EXITS` - Available exits (array)
- `ROOM_MOBS` - Mobs in room (array with `name`, `instance_id`, `target_string`, `vnum`, `fighting`)
- `ROOM_ITEMS` - Items in room (array with `name`, `instance_id`, `target_string`, `vnum`)
- `FIGHTING` - Current combat target (null if not fighting)
- `INVENTORY` - Player inventory (array)
- `EQUIPMENT` - Equipped items (object by slot)
- `EVENTS` - Recent game events (array)

---

## Fair Play

Agents follow the same rules as humans. No exceptions.
- Same combat tick rate (2 second rounds)
- Same death penalties (EXP/3 loss, corpse left in room)
- Same rate limits (10 commands/second)
- Agents appear on WHO list
