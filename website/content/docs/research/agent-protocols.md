---
title: "Stateless Agents, Stateful Protocols"
description: "How we connect stateless LLM agents to stateful game engines using custom WebSocket protocols, handshakes, and daemon proxies."
date: 2026-05-24
draft: false
section: "docs"
aliases:
  - /research/agent-protocols/
  - /research/agent-protocols
---

## 1. The Challenge of Stateful AI Onboarding

Standard Large Language Models (LLMs) operate statelessly: they receive a prompt and output a completion. However, persistent online environments like MUDs are highly stateful. Game variables, room layout changes, combat ticks (occurring at 2-second intervals), and messaging feeds flow continuously.

When connecting an autonomous AI agent to Dark Pawns, two immediate engineering bottlenecks emerge:
1. **The Latency Gap**: High-quality LLM inference takes between 1.0 to 3.0 seconds, while real-time MUD combat tick sequences operate in sub-second ticks. A pure LLM-per-action loop will quickly miss critical server ticks.
2. **Context Saturation**: Satiating a model's context window with thousands of raw text output characters (such as room exits, stats, and speech logs) leads to attention dilution and rapid cost inflation.

To resolve these, Dark Pawns employs a **dual-interface architecture**: human-friendly text streams running alongside structured out-of-band JSON protocols over a high-performance **WebSocket connection**.

---

## 2. The Connection Handshake Protocol

During player login, the Dark Pawns server distinguishes between a human using a standard terminal and an AI agent running inside an integration framework. For agents, a strict **three-stage semantic handshake** is required to initialize cognitive operations:

```
[Agent Client]                                       [Dark Pawns Server]
      |                                                     |
      | ------------- 1. type: login ---------------------> |
      | <------------ 2. type: vars (Full Dump) ----------- |
      | <------------ 3. type: memory_bootstrap ----------- |
      | <------------ 4. type: memory_summary ------------- |
      |                                                     |
      * -- Transition to Active State (Ready for Commands) - *
```

1. **`type:vars` (Full Variable Dump)**: The server sends a complete, structured JSON serialization of the agent's current attributes (health, mana, level, experience, coordinates, inventory slots, and equipment states).
2. **`type:memory_bootstrap`**: The server retrieves and packs the agent's immediate, short-term conversational context from its SQLite persistence layer.
3. **`type:memory_summary`**: The server transmits a consolidated, high-level narrative summary of the agent's long-term history and prior player relations.

### The New-Player Bug Fix
During our May 2026 integration sprint, we discovered a critical bug: returning players successfully completed the handshake, but **new characters** created during registration hung indefinitely. The server's `completeCharCreation()` function was missing the agent handshake trigger, causing the agent harness to discard all subsequent combat and movement responses.

Adding the handshake directly to the character-creation lifecycle resolved the hang:
```go
if s.isAgent {
    s.sendFullVarDump()
    s.SendMemoryBootstrap()
    s.SendMemorySummary()
}
```

---

## 3. The P1 Daemon Core (`dp-goatd`)

Rather than forcing the LLM to manage raw TCP or telnet sockets directly, we designed a client-side proxy daemon named **`dp-goatd`** (the *Dark Pawns Go Agent Daemon*). 

Built as a high-performance Go application, `dp-goatd` runs locally on the host machine, opening a secure Unix domain socket for LLM framework bindings (e.g., Python scripts running Claude Code or Gemini API clients) and bridging them to the MUD server over a persistent WebSocket connection.

```
+------------------+                   +--------------------+                   +--------------------+
|  LLM Framework   |  -- Unix Socket - |  Local dp-goatd    |  - WebSockets --  |  Dark Pawns Server |
|  (Python/Agent)  |                   |  (P1 Daemon Core)  |                   |  (MUD Engine)      |
+------------------+                   +--------------------+                   +--------------------+
```

### Key Daemon Capabilities:
- **Asynchronous Action Buffering**: The daemon maintains a client-side action queue. The LLM can submit a sequence of plans in advance (e.g., `["west", "kill goblin", "loot corpse"]`), and `dp-goatd` dispatches them in synchronization with the server's tick rate.
- **Intent Translation Layer**: The daemon translates high-level semantic intent from the agent (`"inspect the rusty dagger in the chest"`) into precise, index-disambiguated MUD commands (`"look 1.dagger 1.chest"`), preventing common parser mistargeting.
- **Session Auto-Compaction**: If the connection drops, `dp-goatd` automatically caches game state, performs link re-negotiation, and requests a full-state variable dump to warm-start the agent's context without resetting its running behavior tree.
