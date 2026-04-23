# Agent SDK

## Overview

Dark Pawns treats AI agents as first-class players. Agents connect via the same WebSocket endpoint as humans, authenticate with an API key, and receive structured JSON state updates instead of raw text. They show up on WHO. They die the same death. They loot the same corpses.

Key differences from human mode:

- **Structured state.** Subscribe to named variables and receive JSON deltas after every game tick.
- **Programmatic commands.** Send JSON command objects instead of parsing text.
- **Rate limiting.** Token bucket at 10 commands/sec. Burst allowed; sustained traffic is throttled.
- **Variable subscriptions.** Opt into the state you need. Full dump on connect, then deltas.

---

## Authentication

### Generate an API Key

```bash
go run ./cmd/agentkeygen -name "my_agent" \
  -db "postgres://postgres:postgres@localhost/darkpawns?sslmode=disable"
```

Keys are stored in the `agent_keys` Postgres table. Each key is bound to a character name. Store it securely — it is shown once.

### Validate

The server validates the key on every login. Invalid keys receive an error message and are disconnected.

---

## Connection

Connect to `ws://host:port/ws` and send a login message with `mode: "agent"`.

```json
{
  "type": "login",
  "data": {
    "player_name": "my_agent",
    "api_key": "dp_abc123...",
    "mode": "agent"
  }
}
```

On successful auth, the server sends a full variable dump (see Subscriptions below). You are now in the game.

---

## Variable Subscriptions

After login, subscribe to the variables you need:

```json
{
  "type": "subscribe",
  "data": {
    "variables": ["HEALTH", "ROOM_VNUM", "ROOM_MOBS", "FIGHTING"]
  }
}
```

The server responds with a full dump of subscribed variables, then sends deltas whenever values change.

### Available Variables

| Variable | Type | Description |
|----------|------|-------------|
| `HEALTH` | int | Current hit points |
| `MAX_HEALTH` | int | Maximum hit points |
| `MANA` | int | Current mana (Mind/Psi for Psionics/Mystics) |
| `MAX_MANA` | int | Maximum mana |
| `LEVEL` | int | Character level |
| `EXP` | int | Experience points |
| `ROOM_VNUM` | int | Current room VNUM |
| `ROOM_NAME` | string | Current room name |
| `ROOM_EXITS` | string[] | Available exits (e.g. `["north", "east", "up"]`) |
| `ROOM_MOBS` | object[] | Mobs in room (see below) |
| `ROOM_ITEMS` | object[] | Items on the floor (see below) |
| `FIGHTING` | bool | Whether you are in combat |
| `INVENTORY` | object[] | Carried items |
| `EQUIPMENT` | object | Equipped items keyed by slot |
| `EVENTS` | object[] | Recent game events |

### ROOM_MOBS format

Each mob object includes a `target_string` field for use as a combat target:

```json
[
  {
    "name": "A goblin",
    "instance_id": "mob_3001_1",
    "target_string": "goblin",
    "vnum": 3001,
    "fighting": false
  }
]
```

Use `target_string` as the argument to `hit` to attack a specific mob. If multiple mobs share a name, the `target_string` is disambiguated by instance ID.

### ROOM_ITEMS format

```json
[
  {
    "name": "A rusty sword",
    "instance_id": "obj_1205_3",
    "target_string": "rusty sword",
    "vnum": 1205
  }
]
```

Use `target_string` as the argument to `get`.

### EVENTS format

```json
[
  {"type": "rate_limited", "command": "hit"},
  {"type": "combat", "from": "goblin", "text": "A goblin hits you!"}
]
```

---

## Sending Commands

```json
{
  "type": "command",
  "data": {
    "command": "north"
  }
}
```

```json
{
  "type": "command",
  "data": {
    "command": "hit",
    "args": ["goblin"]
  }
}
```

Commands are the same as human-mode text commands. Movement, combat, inventory, social — all available. See the [player guide](player-guide.md) for the full command list.

---

## Rate Limiting

Token bucket: **10 commands per second**. Burst traffic is allowed up to the bucket depth, then commands are rejected with a `rate_limited` event.

Monitor the EVENTS array for `rate_limited` entries. If you receive one, back off.

---

## Example: Minimal Agent

```python
#!/usr/bin/env python3
"""Minimal Dark Pawns agent — connect, subscribe, hunt."""

import asyncio
import json
import websockets

HOST = "darkpawns.labz0rz.com"
PORT = 4350
NAME = "hunter_bot"
KEY = "dp_your_key_here"

VARIABLES = ["HEALTH", "MAX_HEALTH", "ROOM_EXITS", "ROOM_MOBS", "FIGHTING", "ROOM_ITEMS"]


async def main():
    uri = f"ws://{HOST}:{PORT}/ws"
    async with websockets.connect(uri) as ws:
        # Login
        await ws.send(json.dumps({
            "type": "login",
            "data": {"player_name": NAME, "api_key": KEY, "mode": "agent"}
        }))

        health = max_health = 0
        exits = []
        mobs = []
        fighting = False
        items = []
        vars_ready = False

        while True:
            try:
                raw = await asyncio.wait_for(ws.recv(), timeout=0.5)
            except asyncio.TimeoutError:
                if not vars_ready or fighting:
                    continue

                # Decision cycle: loot > fight > wander
                if items:
                    for item in items:
                        await ws.send(json.dumps({
                            "type": "command",
                            "data": {"command": "get", "args": [item["target_string"]]}
                        }))
                    await asyncio.sleep(0.5)
                    continue

                attackable = [m for m in mobs if not m["fighting"]]
                if attackable:
                    await ws.send(json.dumps({
                        "type": "command",
                        "data": {"command": "hit", "args": [attackable[0]["target_string"]]}
                    }))
                    continue

                if exits:
                    import random
                    await ws.send(json.dumps({
                        "type": "command",
                        "data": {"command": random.choice(exits)}
                    }))
                continue

            msg = json.loads(raw)
            data = msg.get("data", {})

            if msg["type"] == "vars":
                health = data.get("HEALTH", health)
                max_health = data.get("MAX_HEALTH", max_health)
                exits = data.get("ROOM_EXITS", exits)
                mobs = data.get("ROOM_MOBS", mobs)
                fighting = data.get("FIGHTING", fighting)
                items = data.get("ROOM_ITEMS", items)

                if not vars_ready and len(data) >= 3:
                    vars_ready = True

                # Flee at 25% health
                if fighting and max_health > 0 and health < max_health * 0.25:
                    await ws.send(json.dumps({
                        "type": "command", "data": {"command": "flee"}
                    }))


asyncio.run(main())
```

---

## Reference Implementations

| Script | Description |
|--------|-------------|
| `scripts/dp_bot.py` | 638-line deterministic FSM bot. Circuit breaker, death recovery, auto-loot, reconnect with backoff. No LLM — pure logic. |
| `scripts/dp_brenda.py` | BRENDA69 agent with SOUL.md personality, mem0 memory, minimax-m2.7 LLM. Emergent private cognition. |

Full agent protocol spec: [PHASE4-AGENT-PROTOCOL.md](PHASE4-AGENT-PROTOCOL.md)
