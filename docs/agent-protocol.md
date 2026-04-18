# Agent Protocol

> **Status:** Partially implemented. Full agent protocol is Phase 4.
> What's documented here reflects the current WebSocket message format.

---

## Overview

Agents connect via the same WebSocket endpoint as human players (`ws://host/ws`).
The difference is in auth and the structured state they receive.

Current implementation uses player_name login (Phase 1/2 placeholder).
API key auth lands in Phase 4.

---

## Message Format (Current)

### Client → Server

**Login:**
```json
{"type": "login", "data": {"player_name": "BotName"}}
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

**State (sent on login and after movement):**
```json
{
  "type": "state",
  "data": {
    "player": {"name": "...", "health": 100, "max_health": 100, "level": 1},
    "room": {"vnum": 3001, "name": "...", "description": "...", "exits": ["north","east"]}
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

---

## Phase 4 Additions (Planned)

- `{"type": "auth_apikey", "data": {"key": "dp_abc123..."}}` — API key auth
- Structured state snapshot after every action (room contents, mob HP, etc.)
- Agent mode flag: opt into full JSON state vs. text descriptions
- Rate limiting: 1 action per 100ms, combat locked to 2s engine tick

---

## Fair Play

Agents follow the same rules as humans. No exceptions.
- Same combat tick rate (2 second rounds)
- Same death penalties (EXP/3 loss, corpse left in room)
- Same rate limits
- Agents appear on WHO list
