# Memory System

> Server-hosted, engine-computed, emotionally valenced autobiographical memory for game agents.

The memory system is Dark Pawns' contribution to game AI research. Memory is not managed by the agent — it's managed by the game engine. The server records what happens, computes emotional significance, builds a narrative graph, and injects relevant context into the agent's LLM prompt. Zero setup. The agent connects and remembers.

## How It Works

```
Session JSONL → Extract Events → Build Graph → Consolidate → Narrative Summary
                                                                      ↓
                                                          Server reads at auth
                                                          Injects into LLM context
```

1. **Session logging** — every agent turn writes a JSONL entry (room, HP, actions, mobs).
2. **Event extraction** — the dreaming pipeline reads logs and extracts meaningful events (kills, deaths, social interactions, acquisitions).
3. **Memory graph** — events are stored as nodes with salience and valence. Entities (mobs, players, items) are linked. The graph persists across sessions.
4. **Consolidation** — salience decays over time. Low-salience nodes are pruned. High-salience nodes are reinforced on re-encounter.
5. **Narrative summary** — `BuildSummary` produces prose ordered chronologically and grouped by session. This is the memory the agent sees.
6. **Injection** — the server reads the summary from disk at agent auth time and sends it as a `memory_summary` message. The agent client injects it into the LLM context.

## Content-Aware Valence

Not all events are equal. The system computes emotional valence (-3 to +3) based on context:

### Combat

| Factor | Valence | Logic |
|--------|---------|-------|
| Kill trivial mob (10+ levels below) | +0 | Barely worth remembering |
| Kill easy mob (3-10 levels below) | +1 | Solid fight |
| Kill challenging mob (within ±3 levels) | +2 | Worthy opponent |
| Kill epic mob (outlevels agent) | +3 | Dragon kill — significant |
| Flee at 80%+ HP | -3 | Cowardly, something went wrong |
| Flee at 40-80% HP | -2 | Embarrassing but understandable |
| Flee at 20-40% HP | -1 | Tactical retreat |
| Flee at <20% HP | 0 | Survival instinct, not failure |

### Social

| Factor | Valence | Logic |
|--------|---------|-------|
| Betrayal / backstab | -3 | Deeply negative |
| Gift / give | +2 | Positive social bond |
| Cooperation / heal ally | +1 | Alliance building |
| Insult / threaten | -2 | Hostile |
| Neutral speech | 0 | Content matters (keyword sentiment) |

### Other

| Factor | Valence | Logic |
|--------|---------|-------|
| Acquire legendary item (80+) | +3 | Major find |
| Acquire valuable item (50-79) | +2 | Useful acquisition |
| Near-death (<5% HP) | -3 | Traumatic |
| Badly hurt (5-15% HP) | -2 | Painful |
| Movement | 0 | Neutral |

Valence blends over repeated encounters with the same entity. First kill of a goblin: +1. Fifth goblin kill: still +1 but reinforced. Killing the same dragon twice: the second kill blends with the first, entity valence shifts positive.

## Narrative Summary

The summary is what the agent sees. Not a bullet list — prose.

```
## Memory

### Session 1 — Jan 12 at 3:15 PM – 3:47 PM

Attacked goblins in the Dark Corridor (noteworthy).
Killed a rogue troll in the Mountain Pass (a significant moment).
Low HP (12/50) while fighting a cave bear (a difficult moment).

### Session 2 — Jan 12 at 4:02 PM

Picked up a gleaming sword in the Dragon's Lair (noteworthy).
Said "We should group up" in the Tavern.

### Relationships

Brenda — trusted ally (met 3 times)
Goblin Shaman — dangerous (met 2 times)
```

Events are grouped by session (30-minute gap = new session), ordered chronologically, and include valence context as parentheticals. The entity relationship summary shows accumulated valence per known entity.

## Server-Side Injection

At agent auth time, the server:

1. Reads `data/dreaming/{agent_id}/memory-summary.txt`
2. Sends `{"type": "memory_summary", "summary": "..."}` to the agent
3. The agent client sets `state.MemorySummary` and injects it into the LLM context

The agent does nothing. The memory is there when it connects.

## Running the Dreaming Pipeline

```bash
# After a session, consolidate memories
dp-agent dream --agent-id brenda

# Or from the server directory
go run ./cmd/dp-agent dream --agent-id brenda \
  --input data/logs/sessions/brenda/ \
  --output data/dreaming/brenda/
```

This reads all JSONL logs, extracts events, builds/updates the memory graph, runs consolidation (decay + prune), and writes the narrative summary.

## Graph Structure

**Node kinds:** `event`, `entity`, `room`, `item`

**Edge kinds:** `occurred_in`, `involved`, `transitioned_to`, `killed`, `took_from`, `fought`, `social`, `similar_to`

**Consolidation cycle:**
- Salience decays by `DecayRate` (default 0.1) per cycle
- Nodes below `PruneThreshold` (default 0.05) are removed
- Re-encountered nodes get `ReinforceBonus` (default 0.2) salience boost
- Orphaned edges are cleaned after pruning

## Ablation Support

The `--valence false` flag disables emotional valence computation. All events get valence 0. This is for the ablation experiment — measuring whether valence actually helps agent behavior.

## File Layout

```
data/
  dreaming/
    {agent_id}/
      memory-graph.json      Full graph (nodes + edges)
      memory-summary.txt     Narrative summary (injected into LLM)
      dream-result.json      Last consolidation stats
  logs/
    sessions/
      {agent_id}/
        2026-05-12-153000.jsonl   Session log
```

## Research Context

This system is the core contribution of the AIIDE 2027 paper "What Did You Do Today: Server-Hosted Emotionally Valenced Autobiographical Memory for Game Agents."

Key claims:
- Memory should be server-hosted, not client-side — the engine knows what happened
- Emotional valence should be computed by the game, not the agent — the engine knows what matters
- Narrative summaries are more useful than raw event logs for LLM context
- Zero-setup memory enables broader adoption than memory APIs that require agent-side integration

**Evaluation metrics:** Behavioral Persistence Score (BPS), Social Consequence Score (SCS), salience decay curves, cost comparison.
