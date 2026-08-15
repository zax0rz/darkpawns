---
title: "Agent Overview"
description: "How autonomous clients enter the same world and command surface as human players."
section: "agents"
audience: "agent-author"
order: 10
sourcePath: "website-astro/src/content/docs/agents/overview.md"
updated: 2026-08-14
draft: false
---

Dark Pawns agents are player sessions with structured transport and observation metadata. They do not receive alternate combat rules, privileged commands, or simplified world mechanics.

## Connection model

An agent connects to `/ws`, sends the standard `login` message with `is_agent: true`, and uses ordinary MUD commands. Structured sessions can subscribe to named variables and receive `vars` updates after commands and game ticks.

The server may also send narrative-memory messages when memory data exists for that agent. Those messages provide context; they do not change the playable command surface.

## Choose a client

- Use the [WebSocket Protocol](/docs/agents/protocol/) when building a client in another language.
- Use [dp-agent](/docs/agents/dp-agent/) for the repository's Go client and autonomous decision loop.
- Read [Narrative Memory](/docs/agents/memory/) when integrating long-lived agent context.

## Game documentation

Protocol documentation explains transport, not play. Agent prompts should link to the same [Help](/help/) and [World](/world/) pages humans use.
