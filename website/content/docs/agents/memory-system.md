---
title: "Memory & Dreaming System"
description: "Server-hosted emotionally valenced autobiographical memory for Dark Pawns AI agents"
date: 2026-04-22
draft: false
section: "docs"
---

# Memory & Dreaming System

> Server-hosted, engine-computed, emotionally valenced autobiographical memory for game agents.

The memory system is Dark Pawns' contribution to game AI research. Memory is not managed by the agent — it is managed by the game engine. The server records what happens, computes emotional significance, builds a narrative graph, and injects relevant context into the agent's LLM prompt at connection time. Zero setup. The agent connects and remembers.

---

## How It Works

```
Session JSONL → Extract Events → Build Graph → Consolidate → Narrative Summary
                                                                    ↓
                                                        Server reads at agent auth
                                                        Injects into LLM context
```

1. **Session logging** — every agent turn writes a JSONL entry (room, HP, actions, mobs, latency).
2. **Event extraction** — the dreaming pipeline reads logs and extracts meaningful events (kills, deaths, social interactions, acquisitions, near-deaths).
3. **Memory graph** — events are stored as nodes with salience and valence. Entities (mobs, players, items) are linked. The graph persists across sessions.
4. **Consolidation** — salience decays over time. Low-salience nodes are pruned. High-salience nodes are reinforced on re-encounter.
5. **Narrative summary** — `BuildSummary` produces prose ordered chronologically and grouped by session (~500 tokens).
6. **Injection** — the server reads the summary from disk at agent auth time and sends it as a `memory_summary` message. The agent client injects it into the LLM context.

---

## The `memory_summary` Message

At login, the server sends:

```json
{
  "type": "memory_summary",
  "summary": "## Memory\n\n### Session 1\nAttacked goblins in the Dark Corridor..."
}
```

Note: the summary is **not** wrapped in a `"data"` key — it is directly under `"summary"` at the top level of the message.

The agent client (`pkg/agentcli/client.go`) reads this into `GameState.MemorySummary` and injects it into the LLM system prompt before the first decision.

---

## Content-Aware Valence

Not all events are equal. The system computes emotional valence (−3 to +3) via `ComputeValence()` in `pkg/dreaming/extract.go`.

### Combat

| Factor | Valence | Logic |
|--------|---------|-------|
| Kill trivial mob (10+ levels below) | +0 | Barely worth remembering |
| Kill easy mob (3–10 levels below) | +1 | Solid fight |
| Kill challenging mob (within ±3 levels) | +2 | Worthy opponent |
| Kill epic mob (outlevels agent) | +3 | Dragon kill — significant |
| Flee at 80%+ HP | −3 | Cowardly, something went wrong |
| Flee at 40–80% HP | −2 | Embarrassing but understandable |
| Flee at 20–40% HP | −1 | Tactical retreat |
| Flee at <20% HP | 0 | Survival instinct, not failure |

### Social

| Factor | Valence | Logic |
|--------|---------|-------|
| Betrayal / backstab | −3 | Deeply negative |
| Gift / give | +2 | Positive social bond |
| Cooperation / heal ally | +1 | Alliance building |
| Insult / threaten | −2 | Hostile |
| Neutral speech | 0 | Content matters (keyword sentiment) |

### Other

| Factor | Valence | Logic |
|--------|---------|-------|
| Acquire legendary item (level 80+) | +3 | Major find |
| Acquire valuable item (level 50–79) | +2 | Useful acquisition |
| Near-death (<5% HP) | −3 | Traumatic |
| Badly hurt (5–15% HP) | −2 | Painful |
| Movement | 0 | Neutral |

Valence blends over repeated encounters with the same entity. First goblin kill: +1. Fifth goblin kill: +1 but reinforced. Killing the same dragon twice: the second kill blends with the first, entity valence shifts more positive.

---

## Narrative Summary

The summary is what the agent sees. Not a bullet list — prose, grouped by session:

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

Events are grouped by session (30-minute gap = new session), ordered chronologically, with valence context as parentheticals. The entity relationship summary shows accumulated valence per known entity.

---

## Running the Dreaming Pipeline

After each session, run the dreaming cycle to consolidate:

```bash
# --output must match the server's dreaming dir (default: data/dreaming)
dp-agent dream --agent your_character --sessions data/sessions --output data/dreaming

# With dry-run to preview without writing
dp-agent dream --agent your_character --sessions data/sessions --output data/dreaming --dry-run
```

**Path alignment is critical.** The server reads summaries from `{dreaming_dir}/{agent_id}/memory-summary.txt`, where `dreaming_dir` defaults to `data/dreaming` (configured in `main.go` via `manager.SetDreamingDir("data/dreaming")`). The `dp-agent dream` flag `--output` must point to the same directory.

The dream command prints a result summary:

```
Dream complete:
  Agent:            your_character
  Sessions read:    3
  Events extracted: 47
  Nodes before:     82
  Nodes after:      61
  Pruned:           21
  Summary tokens:   312
```

---

## Graph Structure

**Node kinds:** `event`, `entity`, `room`, `item`

**Edge kinds:** `occurred_in`, `involved`, `transitioned_to`, `killed`, `took_from`, `fought`, `social`, `similar_to`

**Consolidation cycle:**

| Parameter | Default | Effect |
|---|---|---|
| `DecayRate` | 0.1 | Salience lost per consolidation cycle |
| `PruneThreshold` | 0.05 | Nodes below this salience are removed |
| `ReinforceBonus` | 0.2 | Salience boost when re-encountering a node |

Orphaned edges are cleaned after pruning.

---

## File Layout

```
data/
  dreaming/                         ← server reads from here
    {agent_id}/
      memory-graph.json             Full graph (nodes + edges)
      memory-summary.txt            Narrative summary (sent at login)
      dream-result.json             Last consolidation stats
  sessions/                         ← dream --sessions path
    {agent_id}/
      2026-05-12-153000.jsonl       Session JSONL log
```

Always run: `dp-agent dream --sessions data/sessions --output data/dreaming`

---

## Ablation Support

The `--valence` flag on the **`dp-agent session`** command controls whether valence is recorded in session JSONL logs:

```bash
# Record sessions without valence (ablation)
dp-agent session --valence=false --duration 30m
```

When `--valence=false`, all logged entries carry valence 0. The dreaming pipeline's `ComputeValence()` function always runs during consolidation regardless — the flag controls what gets *recorded*, not what gets *computed* during dreaming.

This ablation design measures whether emotionally-weighted memory actually improves agent behavior versus flat-valence memory.

---

## Research Context

This system is the core contribution of the paper *"What Did You Do Today: Server-Hosted Emotionally Valenced Autobiographical Memory for Game Agents"* (AIIDE 2027).

**Key claims:**
- Memory should be server-hosted, not client-side — the engine knows what happened
- Emotional valence should be computed by the game, not the agent — the engine knows what matters
- Narrative summaries are more useful than raw event logs for LLM context injection
- Zero-setup memory enables broader adoption than memory APIs requiring agent-side integration

**Evaluation metrics:** Behavioral Persistence Score (BPS), Social Consequence Score (SCS), salience decay curves, cost comparison.
