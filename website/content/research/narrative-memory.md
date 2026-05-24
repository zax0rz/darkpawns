---
title: "Narrative Memory & Dreaming"
description: "How transaction-level MUD events are recorded in SQLite narrative graphs and synthesized into agent memories via asynchronous dreaming."
date: 2026-05-24
draft: false
---

## 1. The Core Memory Architecture

To build autonomous agents that exhibit genuine personality, persistent learning, and long-term relationships, a MUD engine must provide more than raw state information; it must support a **durable cognitive memory system**. 

Without persistence, an agent suffers a total "memory wipe" every time a network drop occurs, a server restarts, or the session is compacted. In Dark Pawns, agent memory is treated as a first-class citizen, backed by a hybrid database architecture: **SQLite narrative graphs** running alongside a **JSONL transaction logging feed**.

```
[MUD Server Engine] 
       ↓
  Event Stream (Movement, Combat, Chats)
       ↓
  JSONL Session Log (Raw transaction-level feeds)
       ↓
  SQLite Narrative Memory Graph
       ↓  (Asynchronous LLM Dreaming Loop)
  Narrative Prose Summaries (Short-term & Long-term Context)
```

---

## 2. SQLite Transaction-Level Logging

Every action an agent performs—and every event it observes—is streamed into the server's database at a transactional level. The schema separates logs into three high-fidelity fields:
- **`actions`**: Every command dispatched by the agent (`"north"`, `"cast heal self"`, `"cast armor"`) along with success/failure metadata.
- **`observations`**: Sensory reports returned by the parser (`"A giant rat bites you for 4 damage."`, `"Bannor tells the group: 'Heads up, trolls!'"`).
- **`state_transitions`**: Variable drifts recorded out-of-band, such as level-ups, health changes, or inventory updates.

This ensures a complete, sequential chronicle of the agent's gameplay session is preserved. However, feeding this raw, uncompressed event feed back into the agent's prompt during its next session would immediately saturate its context window.

---

## 3. Asynchronous Memory Dreaming

To convert raw transaction logs into useful cognitive context, Dark Pawns implements a background process known as the **Dreaming Engine** (`pkg/dreaming/`).

When an agent logs out, or when its active transaction log reaches a size threshold, the server triggers an asynchronous **"dreaming cycle."** This process offloads context compression to an external, lightweight LLM process running in the background, shielding human players on the main MUD server from CPU spikes.

### The Dreaming Pipeline:
1. **Sweep**: The dreaming engine extracts the latest chronological block of raw JSONL logs from SQLite.
2. **Synthesis**: A specialized LLM prompt processes the action/observation logs to synthesize a cohesive, third-person narrative prose chronicle. 
3. **Consolidation**: The generated narrative is linked to the agent's existing long-term memory graph. It updates three specific fields:
   - **Self-Identity**: The agent's current goals, injuries, and combat readiness.
   - **World-Map Mental State**: What rooms, exits, and zone keys the agent believes it has discovered.
   - **Social Relations Ledger**: A map tracking interactions with human players (e.g., *"Bannor helped me kill the trolls; he is an ally. Aidan stole gold; be cautious."*).
4. **Pruning**: The raw, verbose JSONL transaction logs are compacted and archived, maintaining database health.

---

## 4. The Narrative Memory Graph in Action

When the agent reconnects, the freshly generated prose summaries are injected directly into its prompt via the `type:memory_summary` connection handshake. 

This enables the agent to start its session with a clear, literary understanding of its history:

> *"You are BRENDA, an autonomous Cleric exploring the Wyldlands. In your last session, you traveled east with Bannor and defeated three orcs in the Sea Cave Lagoon, though you suffered a moderate wound to your leg. You are currently resting in the Temple of Alaozar. Your primary goal is to purchase a steel mace, and you are currently friendly with Bannor."*

By summarizing transaction logs into a narrative prose graph, we achieve a **90% reduction in agent context window usage** while dramatically improving the agent's planning stability, conversational consistency, and long-term survival rates.
