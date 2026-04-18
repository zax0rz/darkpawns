# Dark Pawns Architecture

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Clients                               │
├──────────────┬──────────────┬───────────────────────────────┤
│   Web Client │   Telnet     │         Agent/Bot             │
│   (React)    │   (GMCP)     │      (WebSocket/JSON)         │
└──────┬───────┴──────┬───────┴───────────────┬───────────────┘
       │              │                       │
       └──────────────┴──────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │    Go Game Server       │
                    │  (gorilla/websocket)    │
                    ├─────────────────────────┤
                    │  • Command Interpreter  │
                    │  • Combat Engine        │
                    │  • Lua Script Engine    │
                    │  • World State Manager  │
                    └────────────┬────────────┘
                                 │
            ┌────────────────────┼────────────────────┐
            │                    │                    │
   ┌────────▼────────┐  ┌────────▼────────┐  ┌───────▼────────┐
   │   PostgreSQL    │  │     Redis       │  │   Lua Scripts  │
   │  (Persistent    │  │  (Sessions,     │  │   (World       │
   │   Game State)   │  │   Cache, Pub/Sub)│  │   Behaviors)   │
   └─────────────────┘  └─────────────────┘  └────────────────┘
```

## Components

### Game Server (Go)

The core server handles:
- **Networking:** WebSocket for modern clients, Telnet for classic
- **Command Interpreter:** Parses player input, dispatches to handlers
- **Combat Engine:** Real-time combat resolution
- **Lua Engine:** Executes world scripts via gopher-lua
- **State Management:** Tracks rooms, mobs, items, players

### Database (PostgreSQL)

Stores persistent data:
- Player accounts and characters
- Item instances and ownership
- Mob spawn state
- World configuration

Uses JSONB for flexible schema where needed (character stats, room state).

### Cache (Redis)

Handles ephemeral data:
- Active sessions
- Online player lists
- Room occupancy
- Rate limiting counters
- Pub/sub for real-time events

### Lua Scripts

The original Dark Pawns Lua scripts drive:
- Room behaviors (traps, environmental effects)
- Mob AI (special attacks, reactions)
- Object interactions (magic items, quests)

## Agent Protocol

Agents connect via WebSocket and communicate with structured JSON:

```
Client                     Server
  │    auth_apikey {key}     │
  │ ───────────────────────> │
  │                          │
  │    auth_result {success} │
  │ <─────────────────────── │
  │                          │
  │    action {move}         │
  │ ───────────────────────> │
  │                          │
  │    state {room, self}    │
  │ <─────────────────────── │
  │                          │
  │    event {combat}        │
  │ <─────────────────────── │
```

See [Agent Protocol](agent-protocol.md) for full specification.

## Dual Interface

| Feature | Human Interface | Agent Interface |
|---------|-----------------|-----------------|
| Transport | WebSocket / Telnet | WebSocket |
| Format | Rich text + VT100 | JSON |
| State | Parsed from text | Structured objects |
| Actions | Natural language | Typed commands |
| Perception | Description text | Full state snapshot |

Both interfaces share the same underlying game state and rules.
