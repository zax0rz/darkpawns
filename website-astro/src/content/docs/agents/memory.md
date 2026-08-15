---
title: "Narrative Memory"
description: "How session events become a persistent memory graph and narrative context for autonomous players."
section: "agents"
audience: "agent-author"
order: 40
sourcePath: "website-astro/src/content/docs/agents/memory.md"
updated: 2026-08-14
draft: false
---

Dark Pawns can turn structured session logs into a persistent graph and a compact narrative summary. The implementation lives in `pkg/dreaming`; login delivery is wired through `pkg/session/memory_hooks.go`.

## Pipeline

1. A structured client records session events.
2. `pkg/dreaming` extracts notable actions, observations, entities, rooms, and items.
3. The graph stores relationships, salience, and emotional valence.
4. Consolidation decays, reinforces, and prunes graph nodes.
5. A narrative summary is written for later sessions.
6. On agent login, the server can send `memory_bootstrap` and `memory_summary` messages.

## Client behavior

Treat memory messages as optional context. A new agent or an installation without memory data must still function. Parse each message by its actual envelope and inject the returned narrative into decision context only once per connection.

## Run consolidation

```bash
dp-agent dream --agent CHARACTER --sessions data/sessions --output data/dreaming
```

The output directory must match the server's configured dreaming directory. The current server composition root sets it to `data/dreaming`.

## Research boundary

The graph and valence machinery are implemented and tested. Claims about behavioral improvement require evaluation data; they should not be presented as established results merely because the mechanism exists.
