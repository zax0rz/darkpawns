---
title: "Architecture"
description: "A source-oriented map of the server entrypoint and its principal packages."
section: "server"
audience: "developer"
order: 10
sourcePath: "website-astro/src/content/docs/server/architecture.md"
updated: 2026-08-14
draft: false
---

Dark Pawns is a Go server that loads the original Diku-format world, exposes telnet and WebSocket sessions, and runs the game through a shared world and combat engine.

## Startup path

[`cmd/server/main.go`](https://github.com/zax0rz/darkpawns/blob/main/cmd/server/main.go) is the composition root. In broad order it:

1. Parses flags and deterministic clock/RNG settings.
2. Loads the world through `pkg/parser`.
3. Constructs `pkg/game.World`.
4. Opens PostgreSQL persistence.
5. Creates `pkg/session.Manager` and wires combat, scripting, memory, and decision-log callbacks.
6. Registers HTTP, WebSocket, metrics, admin, and telnet surfaces.
7. Starts reset/tick workers and waits for shutdown.

## Core packages

| Package | Responsibility |
| --- | --- |
| `pkg/parser` | Reads rooms, mobs, objects, zones, and shops from the original files. |
| `pkg/game` | Runtime world state, players, mob/object instances, resets, movement, shops, and skills. |
| `pkg/session` | Login, character creation, connections, command dispatch, and agent variables. |
| `pkg/combat` | Shared tick-based combat calculations and combatant interfaces. |
| `pkg/spells` | Spell metadata, saving throws, damage, affects, and spell execution. |
| `pkg/scripting` | Sandboxed Lua triggers using a serialized VM. |
| `pkg/db` | PostgreSQL-backed player, agent, decision, and narrative persistence. |
| `pkg/agentcli` | Reusable structured client, FSM, logging, and daemon support. |
| `pkg/dreaming` | Event extraction, memory graph consolidation, valence, and summaries. |
| `pkg/telnet` | Raw telnet listener translated into the shared session protocol. |

## Behavioral authority

The Go implementation is not the authority for game behavior. The reachable C path in `src/` and the read-only oracle determine player-facing behavior. Architecture changes must preserve that boundary; see [Port Fidelity Workflow](/docs/research/port-fidelity/).
