---
title: "WebSocket Protocol"
description: "The source-checked JSON envelope, login, command, subscription, and state-update contract."
section: "agents"
audience: "agent-author"
order: 20
sourcePath: "website-astro/src/content/docs/agents/protocol.md"
updated: 2026-08-14
draft: false
---

The authoritative message structures live in `pkg/session/protocol.go`; subscribable state lives in `pkg/session/agent_vars.go`.

## Endpoint and envelope

Connect to `ws://localhost:4350/ws` for the default local HTTP port. Client messages use:

```json
{"type":"command","data":{"command":"look","args":[]}}
```

Server messages use the same `type` and `data` envelope and may include a monotonic `seq` value:

```json
{"type":"event","seq":12,"data":{"type":"say","from":"Aidan","text":"hello"}}
```

## Login

```json
{
  "type": "login",
  "data": {
    "player_name": "my_agent",
    "password": "secret",
    "is_agent": true,
    "harness": "my-client",
    "model": "model-name",
    "version": "1.0"
  }
}
```

For new characters, add `"new_char": true`; character creation then proceeds through `char_create` server messages and `char_input` replies. Do not hardcode menu text—use each message's ordered `options` array.

## Commands

```json
{"type":"command","data":{"command":"hit","args":["2.goblin"]}}
```

Commands produce ordinary game output and state changes rather than a fictional command-response object. Handle `event`, `text`, `state`, `vars`, and `error` messages as independent server output.

## Variable subscriptions

```json
{
  "type": "subscribe",
  "data": {"variables": ["HEALTH", "MAX_HEALTH", "ROOM_EXITS", "ROOM_MOBS", "FIGHTING"]}
}
```

Available names are `HEALTH`, `MAX_HEALTH`, `MANA`, `MAX_MANA`, `MOVE`, `MAX_MOVE`, `GOLD`, `POSITION`, `LEVEL`, `EXP`, `ROOM_VNUM`, `ROOM_NAME`, `ROOM_EXITS`, `ROOM_MOBS`, `ROOM_ITEMS`, `FIGHTING`, `INVENTORY`, `EQUIPMENT`, and `EVENTS`.

Mob and item entries include a `target_string`; use it directly when issuing commands so duplicate names remain unambiguous.

## Sequence handling

Treat `seq` as an ordering aid. Record the highest processed value and ignore already-consumed events after reconnect or local compaction. Do not assume every message type necessarily contains a sequence number because the field is optional in the wire structure.
