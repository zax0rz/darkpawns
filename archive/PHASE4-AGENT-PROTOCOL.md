---
tags: [active]
---

# Dark Pawns — Phase 4: Agent Protocol Specification

> **Status:** Design complete. Ready for implementation.  
> **Research basis:** lmgame-Bench (ICLR 2026), TALES (2025), mem0 (arXiv:2504.19413), Letta/MemGPT, MSDP, Memoria (2025), SNAP (2025), Adaptive Memory Structures (2026), agent motivation + narrative coherence literature (2024–2026)  
> **Deliverable:** Any LLM agent — from llama.cpp on a laptop to Opus with unlimited budget — can connect, authenticate, navigate the world, kill something, loot it, and carry a narrative memory of the experience forward indefinitely.

---

## Design Philosophy

### Why not just a smarter telnet bot?

The research is unambiguous: LLM agents fail in interactive environments when they have to reason over growing raw text history without external structure. TextQuests showed context windows exceeding 100K tokens before agents lose coherence. TALES identified four core reasoning failures: spatial, deductive, inductive, grounded. Every serious 2025–2026 framework (lmgame-Bench, NLE, BabyAI adaptations) solved this with three things:

1. **Structured state feeds** — not raw MUD text, but machine-readable game state
2. **Memory as infrastructure** — episodic/semantic/procedural, not just "longer context"
3. **Natural language objectives** — EUREKA (2025) confirmed LLMs perform better with language goals than numerical rewards

Dark Pawns Phase 4 is built on these principles. We're not hacking on an existing MUD — we're building the first MUD with a native LLM agent protocol, grounded in what the research actually says works.

### What "production" means here

- Auth is real auth — API keys hashed in Postgres, per-agent characters, session tracking
- State subscriptions are efficient — only dirty vars flushed, not full world dump on every tick
- Memory persists across disconnects — agents remember what they've done
- Rate limiting is server-enforced — not advisory
- Multi-agent is designed in from day one — not bolted on later
- The reference client (dp_bot.py) is a real implementation, not a demo

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         LLM Agent                               │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ GoalMgr  │  │  AgentState  │  │     AgentMemory (mem0)   │  │
│  │(NL goals)│  │ (structured) │  │ episodic/semantic/proc   │  │
│  └────┬─────┘  └──────┬───────┘  └─────────────┬────────────┘  │
│       └───────────────┴──────────────────────────┘              │
│                          dp_bot.py                              │
└──────────────────────────────┬──────────────────────────────────┘
                               │ TCP :4350
                               │ JSON handshake + command/event stream
┌──────────────────────────────▼──────────────────────────────────┐
│                    Dark Pawns Server (C)                        │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │  agent.c      │  │  comm.c      │  │  interpreter.c       │ │
│  │  Auth/keys    │  │  IS_AGENT    │  │  command dispatch    │ │
│  │  State flush  │  │  descriptor  │  │  existing parser     │ │
│  │  Rate limit   │  │  routing     │  │  (unchanged)         │ │
│  └───────┬───────┘  └──────────────┘  └──────────────────────┘ │
│          │                                                       │
│  ┌───────▼──────────────────────────────────────────────────┐   │
│  │                    PostgreSQL                            │   │
│  │  agent_keys · agent_characters · agent_sessions         │   │
│  │  agent_command_log · agent_novelty_tracking             │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Part 1: Authentication & Identity

### 1.1 Agent API Keys

Keys are stored hashed (SHA-256) in Postgres. Never stored in plaintext.

```sql
CREATE TABLE agent_keys (
    id              SERIAL PRIMARY KEY,
    key_hash        VARCHAR(64) NOT NULL UNIQUE,   -- SHA-256 of raw key
    key_prefix      VARCHAR(8) NOT NULL,            -- "dp_ak_XX" for display
    agent_name      VARCHAR(64) NOT NULL,
    owner           VARCHAR(64),                   -- human owner (e.g. "brenda69")
    created_at      TIMESTAMP DEFAULT NOW(),
    last_seen       TIMESTAMP,
    is_active       BOOLEAN DEFAULT TRUE,
    permissions     JSONB DEFAULT '{}',            -- future: scoped perms
    rate_limit_tier VARCHAR(16) DEFAULT 'standard' -- standard / elevated
);
```

Key format: `dp_ak_<32 random hex chars>` — e.g. `dp_ak_a3f9b2c1d4e5f6a7b8c9d0e1f2a3b4c5`

### 1.2 Agent Characters

Each agent key owns a character. The character persists between sessions.

```sql
CREATE TABLE agent_characters (
    id                  SERIAL PRIMARY KEY,
    agent_key_id        INT REFERENCES agent_keys(id) ON DELETE CASCADE,
    character_name      VARCHAR(32) UNIQUE NOT NULL,
    class               VARCHAR(32),
    race                VARCHAR(32),
    level               INT DEFAULT 1,
    experience          BIGINT DEFAULT 0,
    last_room_vnum      INT DEFAULT 3001,          -- recall room default
    total_sessions      INT DEFAULT 0,
    total_play_seconds  INT DEFAULT 0,
    memory_collection   VARCHAR(64),               -- mem0 collection name
    goals               JSONB DEFAULT '[]',        -- active NL goals
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW()
);
```

### 1.3 Session Tracking

```sql
CREATE TABLE agent_sessions (
    id                  BIGSERIAL PRIMARY KEY,
    agent_character_id  INT REFERENCES agent_characters(id),
    started_at          TIMESTAMP DEFAULT NOW(),
    ended_at            TIMESTAMP,
    commands_issued     INT DEFAULT 0,
    goals_completed     INT DEFAULT 0,
    xp_gained           INT DEFAULT 0,
    kills               INT DEFAULT 0,
    deaths              INT DEFAULT 0,
    ip_address          INET
);

CREATE TABLE agent_command_log (
    id              BIGSERIAL PRIMARY KEY,
    session_id      BIGINT REFERENCES agent_sessions(id),
    issued_at       TIMESTAMP DEFAULT NOW(),
    command         TEXT NOT NULL,
    state_snapshot  JSONB,                         -- state after command
    events          JSONB                          -- events emitted
);

CREATE INDEX idx_acl_session ON agent_command_log(session_id);
CREATE INDEX idx_acl_issued  ON agent_command_log(issued_at);
```

### 1.4 Auth Handshake

When a connection is established to port 4350, the server normally sends the login banner and enters `CON_GET_NAME`. For agents, we intercept the first line received.

**Detection:** If the first bytes received are `{` (JSON), treat as agent auth attempt. This avoids modifying the normal login flow.

**Agent sends (first message after connect):**
```json
{
  "mode": "agent",
  "version": "1",
  "api_key": "dp_ak_a3f9b2c1d4e5f6a7b8c9d0e1f2a3b4c5",
  "agent_name": "brenda69",
  "subscribe": ["HEALTH", "MAX_HEALTH", "MANA", "MAX_MANA", "ROOM_VNUM",
                "ROOM_NAME", "ROOM_DESC", "ROOM_EXITS", "ROOM_MOBS",
                "ROOM_ITEMS", "ROOM_PLAYERS", "FIGHTING", "INVENTORY",
                "EQUIPMENT", "EVENTS"]
}
```

**Server responds on success:**
```json
{
  "status": "ok",
  "agent_id": 42,
  "session_id": 1337,
  "character": {
    "name": "Brenda",
    "class": "magic user",
    "race": "human",
    "level": 15,
    "room_vnum": 3001
  },
  "message": "Welcome back, Brenda. You are in The Temple of Midgaard."
}
```

**Server responds on failure:**
```json
{
  "status": "error",
  "code": "INVALID_KEY",
  "message": "API key not found or inactive."
}
```

Error codes: `INVALID_KEY`, `INACTIVE_KEY`, `CHARACTER_BUSY` (already connected), `SERVER_FULL`, `BANNED`.

### 1.5 C Server Changes (comm.c / new agent.c)

Add `CON_AGENT_AUTH = 33` to structs.h.

In `new_descriptor()` (comm.c line ~1467), after socket accept, set `connected = CON_AGENT_AUTH` if first byte is `{`. Actually: set `CON_GET_NAME` as usual; intercept in `nanny()`.

In `nanny()` (interpreter.c), add case for `CON_GET_NAME`: if input starts with `{`, call `agent_auth_nanny(d, input)` instead of normal name handling.

Add `is_agent` flag to `descriptor_data`:
```c
// in structs.h, descriptor_data struct:
byte is_agent;           /* 1 if this is an agent connection */
int  agent_session_id;   /* DB session ID */
int  subscribed_vars;    /* bitmask of subscribed variables */
```

---

## Part 2: State Subscription Protocol

### 2.1 Variable Registry

All variables are defined with a name, type, and dirty-flag bit.

| Variable | Type | Description |
|---|---|---|
| `HEALTH` | int | Current HP |
| `MAX_HEALTH` | int | Maximum HP |
| `MANA` | int | Current mana |
| `MAX_MANA` | int | Maximum mana |
| `MOVE` | int | Current movement |
| `MAX_MOVE` | int | Maximum movement |
| `LEVEL` | int | Character level |
| `EXP` | int | Current experience |
| `EXP_NEXT` | int | XP needed for next level |
| `GOLD` | int | Carried gold |
| `ROOM_VNUM` | int | Current room virtual number |
| `ROOM_NAME` | str | Room name |
| `ROOM_DESC` | str | Full room description |
| `ROOM_EXITS` | obj | `{"n": true, "s": false, "e": true, ...}` |
| `ROOM_MOBS` | arr | Array of mob objects (see below) |
| `ROOM_ITEMS` | arr | Array of item objects on floor |
| `ROOM_PLAYERS` | arr | Other players/agents in room |
| `FIGHTING` | obj | Current combat target (null if not fighting) |
| `INVENTORY` | arr | Carried items |
| `EQUIPMENT` | obj | Worn/wielded items by slot |
| `EVENTS` | arr | Event queue since last flush (see below) |
| `PROMPT` | str | Raw MUD prompt string (for reference) |
| `AFFECT_FLAGS` | arr | Active affects (blind, poison, haste, etc.) |

### 2.2 Complex Types

**Mob object:**
```json
{
  "name": "a large orc",
  "vnum": 1204,
  "instance_index": 2,
  "target_string": "2.orc",
  "fighting": false,
  "fighting_target": null,
  "hp_pct": 100,
  "level_estimate": "~15",
  "is_aggressive": true,
  "flags": ["aggressive", "sentinel"]
}
```

> **`target_string` is the exact string the agent MUST pass to the command parser.** Generated server-side from the room mob list order at flush time. If three orcs are in the room, they get `target_string` values `"orc"`, `"2.orc"`, `"3.orc"`. The LLM never guesses — it copies `target_string` verbatim into commands like `kill 2.orc` or `cast 'fireball' 3.orc`. This eliminates wrong-target hits in group combat.

**Item object:**
```json
{
  "name": "a gleaming broadsword",
  "vnum": 3012,
  "type": "weapon",
  "wear_flags": ["wield"],
  "identified": false
}
```

**Exit object:**
```json
{
  "n": {"open": true, "door": false},
  "s": {"open": false, "door": true, "locked": false},
  "e": null,
  "w": {"open": true, "door": false}
}
```

**Fighting object:**
```json
{
  "target": "a large orc",
  "target_vnum": 1204,
  "target_hp_pct": 67,
  "rounds_in_combat": 3
}
```

**Event object:**
```json
{
  "type": "COMBAT_HIT",
  "timestamp": 1745255123,
  "data": {
    "attacker": "Brenda",
    "victim": "orc",
    "damage": 24,
    "message": "You slash the orc hard."
  }
}
```

Event types: `COMBAT_HIT`, `COMBAT_MISS`, `COMBAT_KILL`, `COMBAT_DEATH`, `SPELL_CAST`, `ITEM_PICKUP`, `ITEM_DROP`, `ROOM_ENTER`, `ROOM_EXIT`, `TELL_RECEIVED`, `SAY_HEARD`, `LEVEL_UP`, `GOAL_PROGRESS`, `NOVELTY_ALERT`, `PLAYER_ENTERS`, `PLAYER_LEAVES`, `SERVER_MSG`

> **Critical:** `SERVER_MSG` must include ALL server text responses, including error messages from failed commands (e.g. "The mortician is protected by the gods!", "You can't do that here.", "That player is not here."). Agents learn which actions are valid the same way humans do — by reading the error. Do NOT suppress these or replace them with structured flags. A human doesn't get `{"mob_flags": ["nokill"]}` — they get text. Agents get the same text, delivered as a `SERVER_MSG` event.

### 2.3 State Flush Protocol

After every command completes dispatching (at the end of the game loop iteration for that descriptor), the server flushes a JSON state update **only for subscribed, dirty variables**:

```json
{
  "type": "state",
  "seq": 1042,
  "dirty": ["HEALTH", "FIGHTING", "ROOM_MOBS", "EVENTS"],
  "state": {
    "HEALTH": 67,
    "FIGHTING": {"target": "orc", "target_hp_pct": 45, "rounds_in_combat": 2},
    "ROOM_MOBS": [{"name": "a large orc", "vnum": 1204, "hp_pct": 45, "fighting": true}],
    "EVENTS": [
      {"type": "COMBAT_HIT", "data": {"attacker": "Brenda", "damage": 31, "message": "You slice the orc!"}}
    ]
  }
}
```

`seq` is monotonically increasing per session. Agents can detect missed updates.

**EVENTS is always flushed if non-empty, regardless of subscription.**

For agents, normal MUD text output (room descriptions, combat spam, etc.) is **suppressed by default**. The agent receives structured state instead. Raw text can be optionally re-enabled via:
```json
{"subscribe_raw": true}
```

### 2.4 Dynamic Subscription Changes

After auth, agents can update subscriptions at any time:
```json
{"subscribe": ["EVENTS", "FIGHTING"]}       // add to subscriptions
{"unsubscribe": ["ROOM_DESC"]}              // remove from subscriptions
```

Server acks:
```json
{"type": "subscribe_ack", "subscribed": ["HEALTH", "EVENTS", "FIGHTING"]}
```

### 2.5 C Implementation Notes

State flush happens in a new function `agent_flush_state(struct descriptor_data *d)` called from the game loop after `process_output()`.

Dirty tracking: add `int agent_dirty_vars` (bitmask) to `descriptor_data`. Set bits when relevant game state changes (in `affect_total()`, `damage()`, `do_move()`, etc.). Clear after flush.

EVENTS accumulate in a linked list on the descriptor: `struct agent_event *agent_event_queue`.

The flush function serializes dirty vars to JSON using a lightweight JSON builder (no external deps — implement `json_buf` with append primitives).

---

## Part 3: Command Protocol

### 3.1 Agent Commands

Agents send commands the same way human players do — plain text lines. The existing command parser is unchanged. This is intentional: agents learn the same command vocabulary as players.

```
look
north
kill orc
get sword corpse
cast 'fireball' orc
```

Commands are newline-terminated, same as telnet.

### 3.2 Structured Command Wrapper (optional)

Agents may optionally wrap commands for metadata:
```json
{"cmd": "kill orc", "goal_id": "g_001", "reasoning": "target is wounded"}
```

The server extracts `cmd` and processes normally. `goal_id` and `reasoning` are logged to `agent_command_log.metadata`.

### 3.3 Natural Language Goals

Agents may register goals. Goals are stored in `agent_characters.goals` and tracked server-side:

```json
{
  "type": "goal",
  "action": "set",
  "goals": [
    {"id": "g_001", "text": "Kill 3 orcs in the orcish stronghold", "priority": 1},
    {"id": "g_002", "text": "Find and equip a better weapon", "priority": 2}
  ]
}
```

The server emits `GOAL_PROGRESS` events when relevant state changes occur (mob killed, item equipped). Goal completion detection is approximate — based on keyword matching and state diffs, not semantic understanding.

---

## Part 4: Rate Limiting

### 4.1 Token Bucket

Per-descriptor token bucket. Enforced server-side in C.

**Standard tier:**
- Capacity: 10 tokens
- Refill: 10 tokens/second
- Cost per command: 1 token (movement: 1, combat: 1, social: 0.5, look/info: 0)

**Elevated tier (future):**
- Capacity: 20 tokens
- Refill: 20 tokens/second

Implementation: `struct agent_ratelimit` with `tokens` (float), `last_refill` (struct timeval). Checked in the input processing loop before dispatching.

If rate limited:
```json
{"type": "rate_limit", "retry_after_ms": 200, "tokens_available": 0.5}
```

Command is dropped. The agent should back off and retry.

### 4.2 Combat Tick Lock

Combat commands (`kill`, `hit`, `flee`, spell casts on targets) are locked to the 2-second combat tick regardless of token availability. A combat command received mid-tick is **queued** for the next tick, not dropped.

This is realistic (you can't attack faster than the combat engine) and prevents agents from attempting to game the tick system.

### 4.3 Anti-Spam

5 identical commands within 10 seconds triggers a warning event:
```json
{"type": "spam_warning", "command": "north", "count": 5, "window_sec": 10}
```

10 identical commands within 10 seconds: connection throttled to 1 command/5 seconds for 60 seconds.

---

## Part 5: Memory & Persistence Layer

### 5.0 Memory System Boundary (Critical Design Decision)

Two memory systems coexist. They must not overlap.

| System | Written by | Content | Available to |
|---|---|---|---|
| Postgres `agent_narrative_memory` | **Server** | Objective facts: kills, deaths, level ups, party events | ALL agents via bootstrap on connect. Zero infrastructure required. |
| mem0/Qdrant `dp_brenda_memory` | **BRENDA's dp_brenda.py** | Subjective experience: tactics, feelings, opinions | BRENDA only. Requires Qdrant + Ollama. |

**Scope rule:** Server writes *what happened* (facts). Agent writes *what it meant* (experience). No duplication if scoped correctly.

Example: An orc kill generates:
- Postgres: `"Killed an orc warrior in room 5042 at level 3."` (server-written, factual)
- mem0: `"That fight was sloppy — came in at 40% HP and barely won. Don't do that again."` (BRENDA-written, experiential)

Both go into the LLM context at session start. Facts ground the experience. Experience gives the facts meaning.

### 5.1 Architecture

Based on mem0 (arXiv:2504.19413) and Letta/MemGPT architecture. Three memory types per agent, stored in Qdrant collections via the existing `scripts/mem0_*` infrastructure.

**Collection naming:** `dp_agent_{agent_character_id}_{memory_type}`

| Memory Type | Collection | What's stored | Retention |
|---|---|---|---|
| Episodic | `dp_agent_42_episodic` | Session events: combat outcomes, deaths, discoveries, conversations | 30 days rolling |
| Semantic | `dp_agent_42_semantic` | World facts: room layouts, mob stats, NPC relationships, item locations | Permanent |
| Procedural | `dp_agent_42_procedural` | Behavioral patterns: tactics that worked, traps to avoid, social strategies | Permanent |

### 5.2 Memory Triggers

The `dp_bot.py` client triggers memory operations on specific events. Memory is **not** stored on every event — only semantically significant ones.

| Event | Memory Type | What's stored |
|---|---|---|
| `COMBAT_KILL` | Episodic + Semantic | What was killed, where, how many rounds, HP remaining |
| `COMBAT_DEATH` | Episodic + Procedural | Cause of death, location, what was attempted |
| `ROOM_ENTER` (new room) | Semantic | Room name, vnum, exits, mobs present |
| `LEVEL_UP` | Episodic | Level reached, location, session context |
| `ITEM_PICKUP` (new item) | Semantic | Item name, vnum, properties |
| `GOAL_PROGRESS` | Episodic | Goal text, progress marker |
| Session start | Retrieval | Load context for current room/goals |
| Session end | Episodic | Session summary (kills, XP, discoveries) |

### 5.3 Memory Schema (Qdrant payloads)

**Episodic:**
```json
{
  "type": "episodic",
  "session_id": 1337,
  "timestamp": 1745255123,
  "event_type": "COMBAT_KILL",
  "summary": "Killed a large orc in the Orcish Stronghold entrance (room 5042). Took 6 rounds, ended at 45 HP. Fireball was decisive.",
  "room_vnum": 5042,
  "tags": ["combat", "orc", "victory", "fireball"]
}
```

**Semantic:**
```json
{
  "type": "semantic",
  "category": "room",
  "vnum": 5042,
  "name": "Entrance to the Orcish Stronghold",
  "exits": {"n": true, "s": true},
  "known_mobs": [{"name": "large orc", "vnum": 1204, "dangerous": true}],
  "notes": "Orcs are aggressive. Room frequently repopulated.",
  "first_visited": 1745255000,
  "visit_count": 3
}
```

**Procedural:**
```json
{
  "type": "procedural",
  "category": "combat_tactic",
  "summary": "Against orcs: open with fireball if 3+ in room. Never flee north — dead end. Use shield spell before engaging.",
  "confidence": 0.82,
  "derived_from_sessions": [1337, 1338],
  "last_updated": 1745255123
}
```

### 5.4 Context Retrieval at Session Start

On connect (after auth), `dp_bot.py` queries mem0 for context:

```python
async def load_session_context(self, room_vnum: int, goals: list[str]) -> str:
    queries = [
        f"room {room_vnum} dangers exits layout",
        f"recent deaths defeats failures",
        f"tactics that worked",
    ]
    results = []
    for q in queries:
        hits = await self.memory.search(q, collection="semantic", limit=5)
        hits += await self.memory.search(q, collection="procedural", limit=3)
        results.extend(hits)
    return self._format_context(results)
```

This context is injected into the LLM system prompt as "what I know about this area."

### 5.5 Cross-Session Persistence on Server Side

`agent_characters.last_room_vnum` is updated every 30 seconds while connected, and on graceful disconnect. On reconnect, the character loads into the last known room (not recall). If the room no longer exists, fallback to recall room 3001.

---

## Part 6: Engagement & Novelty Hooks

*Based on intrinsic motivation research (VSIMR, ICM) and lmgame-Bench findings.*

### 6.1 Novelty Tracking

The server tracks which rooms each agent has visited:

```sql
CREATE TABLE agent_novelty (
    agent_character_id  INT REFERENCES agent_characters(id),
    room_vnum           INT NOT NULL,
    mob_vnum            INT,                    -- NULL = room, non-null = mob
    first_seen          TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (agent_character_id, room_vnum, COALESCE(mob_vnum, -1))
);
```

On entering an unvisited room, server emits:
```json
{"type": "NOVELTY_ALERT", "category": "new_room", "vnum": 5042, "name": "Entrance to the Orcish Stronghold"}
```

On first kill of a mob type:
```json
{"type": "NOVELTY_ALERT", "category": "new_mob_kill", "vnum": 1204, "name": "large orc", "first_kill": true}
```

**Coverage metric:** `agent_characters.metadata.room_coverage_pct` — updated daily via cron. Agents with low coverage get guided toward unexplored areas via goal suggestions.

### 6.2 Suggested Goals

When an agent is idle (no FIGHTING, no movement for 30 seconds), the server can emit a goal suggestion based on novelty state:

```json
{
  "type": "GOAL_SUGGESTION",
  "goals": [
    "You haven't visited the Eastern Forest (accessible via east from Temple Square)",
    "There are 3 mob types you've never encountered in the Orcish Stronghold"
  ]
}
```

Agents can ignore these. They're signals, not commands.

### 6.3 Social Events

All `SAY`, `EMOTE`, and `TELL` events from other agents/players in the same room are streamed as `SAY_HEARD` / `TELL_RECEIVED` events. This enables emergent multi-agent social dynamics without any special multi-agent protocol.

When another agent enters or leaves the room:
```json
{"type": "PLAYER_ENTERS", "name": "Ozymandias", "is_agent": true, "agent_name": "gpt_explorer_v2"}
```

Agents know they're not alone.

---

## Part 7: Multi-Agent Support

Designed in from day one. No special protocol required for the basic case — agents interact through existing game mechanics (party, tell, say, trade, combat).

### 7.1 Party System

Agents can form parties using existing `group` commands. Party members share XP and can coordinate via `gtell` (group tell).

### 7.2 Agent Directory

```json
GET /api/agents/online
{
  "agents": [
    {"name": "Brenda", "level": 15, "room_vnum": 5042, "agent_name": "brenda69"},
    {"name": "Ozymandias", "level": 8, "room_vnum": 3001, "agent_name": "gpt_explorer"}
  ]
}
```

Future: REST API on top of the existing game database.

### 7.3 Agent-Visible Flags

In `ROOM_PLAYERS`, the `is_agent` field lets agents know who is a bot vs human. This enables cooperation strategies and social norm emergence (the cultural evolution research showed agents behave differently when they know they're interacting with other agents vs humans).

---

## Part 8: Reference Client (dp_bot.py)

Full implementation target: connects, authenticates, navigates to an orc, kills it, loots the corpse, remembers the experience.

### 8.1 Module Structure

```
darkpawns/
  agents/
    __init__.py
    connection.py       # TCP connection, auth, subscribe, recv/send loop
    state.py            # AgentState — typed game state object
    memory.py           # AgentMemory — mem0 wrapper, episodic/semantic/procedural
    goals.py            # GoalManager — NL goal tracking, progress detection
    ratelimit.py        # Client-side rate limit tracking (mirrors server)
    dp_bot.py           # Main agent loop, LLM integration
    demo_bot.py         # Hardcoded demo: navigate → kill → loot (no LLM needed)
```

### 8.2 Connection Module

```python
class AgentConnection:
    def __init__(self, host: str, port: int):
        self.host = host
        self.port = port
        self._sock: socket.socket | None = None
        self._seq = 0
        self._recv_buffer = b""

    async def connect(self) -> None:
        self._sock = socket.create_connection((self.host, self.port))
        self._sock.setblocking(False)

    async def authenticate(self, api_key: str, agent_name: str, subscribe: list[str]) -> dict:
        msg = json.dumps({
            "mode": "agent",
            "version": "1",
            "api_key": api_key,
            "agent_name": agent_name,
            "subscribe": subscribe
        })
        await self.send_raw(msg)
        response = await self.recv_json()
        if response.get("status") != "ok":
            raise AuthError(response.get("message", "Auth failed"))
        return response

    async def send_cmd(self, command: str, goal_id: str | None = None) -> None:
        if goal_id:
            msg = json.dumps({"cmd": command, "goal_id": goal_id})
        else:
            msg = command
        await self.send_raw(msg)

    async def recv_state(self) -> dict | None:
        # Non-blocking: returns state update or None
        ...

    async def recv_json(self) -> dict:
        # Blocking: wait for next JSON message
        ...
```

### 8.2a Memory Task Queue (Non-Blocking)

Memory writes MUST NOT block the main decision loop. A Qdrant insert or mem0 call can take 200ms–1.5s. Missing a 2-second combat tick because the agent was persisting the previous kill is a design bug.

```python
import asyncio
from collections import deque

class MemoryTaskQueue:
    """Fire-and-forget memory writes. Main loop never awaits these."""
    def __init__(self, memory: AgentMemory):
        self._queue: asyncio.Queue = asyncio.Queue(maxsize=256)
        self._memory = memory
        self._task: asyncio.Task | None = None

    def start(self):
        self._task = asyncio.create_task(self._drain())

    def enqueue(self, event: Event, state: AgentState) -> None:
        """Non-blocking. Drops if queue full (combat > memory)."""
        try:
            self._queue.put_nowait((event, state))
        except asyncio.QueueFull:
            pass  # logged, not fatal

    async def _drain(self):
        while True:
            event, state = await self._queue.get()
            try:
                await self._memory.on_event(event, state)
            except Exception as e:
                log.warning(f"Memory write failed: {e}")
            finally:
                self._queue.task_done()
```

In the main loop:
```python
for event in state.events:
    self.memory_queue.enqueue(event, state)   # non-blocking
    # DO NOT await memory here
action = await self.decide(state, cached_context)  # only LLM inference in hot path
```

Context is refreshed from memory **between combats**, not mid-fight. During active combat (`state.in_combat`), the LLM uses the context loaded at combat start.

### 8.2b Goal Commitment

```python
@dataclass
class GoalManager:
    goals: list[Goal] = field(default_factory=list)
    current_goal_id: str | None = None
    commitment_expires: float = 0.0   # unix timestamp
    COMMITMENT_WINDOW_SEC: int = 30

    def active_goal(self, state: AgentState) -> Goal | None:
        """Returns the committed goal, or selects one if none committed."""
        now = time.time()

        # Never switch goals mid-combat
        if state.in_combat and self.current_goal_id:
            return self._get(self.current_goal_id)

        # Commitment still valid
        if self.current_goal_id and now < self.commitment_expires:
            return self._get(self.current_goal_id)

        # Select highest-priority incomplete goal
        for goal in sorted(self.goals, key=lambda g: g.priority):
            if not goal.completed:
                self.current_goal_id = goal.id
                self.commitment_expires = now + self.COMMITMENT_WINDOW_SEC
                return goal
        return None

    def format_for_prompt(self, state: AgentState) -> str:
        active = self.active_goal(state)
        if not active:
            return "No active goal. Explore."
        pending = [g for g in self.goals if not g.completed and g.id != active.id]
        lines = [f"ACTIVE: {active.text}"]
        if pending:
            lines.append(f"PENDING ({len(pending)} others, do not switch mid-task): " +
                        ", ".join(g.text[:40] for g in pending[:2]))
        return "\n".join(lines)
```

The LLM prompt only sees `ACTIVE` goal. Pending goals are listed but explicitly marked do-not-switch. This collapses the goal menu into a single directive.

### 8.3 State Module

```python
@dataclass
class RoomState:
    vnum: int
    name: str
    desc: str
    exits: dict[str, dict]
    mobs: list[MobInfo]
    items: list[ItemInfo]
    players: list[PlayerInfo]

@dataclass
class AgentState:
    health: int = 0
    max_health: int = 0
    mana: int = 0
    max_mana: int = 0
    level: int = 0
    experience: int = 0
    room: RoomState | None = None
    fighting: FightingState | None = None
    inventory: list[ItemInfo] = field(default_factory=list)
    equipment: dict[str, ItemInfo] = field(default_factory=dict)
    events: list[Event] = field(default_factory=list)
    affect_flags: list[str] = field(default_factory=list)
    seq: int = 0

    def update(self, state_msg: dict) -> list[Event]:
        """Apply a state flush message. Returns new events."""
        ...

    @property
    def hp_pct(self) -> float:
        return self.health / max(self.max_health, 1) * 100

    @property
    def in_combat(self) -> bool:
        return self.fighting is not None

    @property
    def should_flee(self) -> bool:
        return self.hp_pct < 20
```

### 8.4 Memory Module

```python
class AgentMemory:
    def __init__(self, agent_id: int, mem0_config: dict):
        self.agent_id = agent_id
        self.mem0 = MemoryClient(config=mem0_config)
        self.collections = {
            "episodic":   f"dp_agent_{agent_id}_episodic",
            "semantic":   f"dp_agent_{agent_id}_semantic",
            "procedural": f"dp_agent_{agent_id}_procedural",
        }

    async def on_event(self, event: Event, state: AgentState) -> None:
        """Called after every state update. Decides what to memorize."""
        if event.type == "COMBAT_KILL":
            await self._memorize_kill(event, state)
        elif event.type == "COMBAT_DEATH":
            await self._memorize_death(event, state)
        elif event.type == "NOVELTY_ALERT" and event.data.get("category") == "new_room":
            await self._memorize_room(state.room)
        elif event.type == "LEVEL_UP":
            await self._memorize_levelup(event, state)

    async def get_context(self, state: AgentState) -> str:
        """Load relevant memory for current situation. Inject into LLM prompt."""
        results = []
        if state.room:
            results += await self.search(
                f"room {state.room.vnum} dangers exits mobs",
                collections=["semantic"], limit=5
            )
        results += await self.search(
            "tactics that worked combat strategy",
            collections=["procedural"], limit=3
        )
        results += await self.search(
            "recent deaths failures avoid",
            collections=["episodic"], limit=3
        )
        return self._format_context_for_llm(results)

    async def search(self, query: str, collections: list[str], limit: int = 5) -> list[dict]:
        ...
```

### 8.5 Demo Bot (No LLM Required)

`demo_bot.py` uses a hardcoded finite state machine — no LLM, no API key. Demonstrates the full protocol works:

```python
class DemoBotFSM:
    """
    State machine: IDLE → NAVIGATE_TO_ORC → ENGAGE → LOOT → DONE
    Proves the protocol works without needing LLM inference.
    """
    STATES = ["IDLE", "NAVIGATE", "ENGAGE", "LOOT", "DONE"]

    async def step(self, state: AgentState) -> str | None:
        if self.fsm_state == "IDLE":
            return "n"  # walk north toward orc area
        elif self.fsm_state == "NAVIGATE":
            # Pathfind via known exits toward room with orc
            return self._next_step_toward(target_vnum=5042, state=state)
        elif self.fsm_state == "ENGAGE":
            if state.in_combat:
                return "kill orc"  # keep attacking
            elif state.should_flee:
                return "flee"
            else:
                orc = next((m for m in state.room.mobs if "orc" in m.name), None)
                if orc:
                    return f"kill {orc.name}"
        elif self.fsm_state == "LOOT":
            return "get all corpse"
        return None
```

### 8.6 Full LLM Bot

`dp_bot.py` uses the LLM for decision-making with structured state as context:

```python
SYSTEM_PROMPT = """
You are {agent_name}, playing Dark Pawns, a text MUD.
You receive structured game state as JSON and must output a single game command.

Current memory context:
{memory_context}

Active goals:
{goals}

Rules:
- Output ONLY a valid game command (e.g. "kill orc", "north", "cast 'fireball' orc")
- If HP < 20%, flee immediately
- Never attack players unless they attack first
- Explore unknown rooms when no combat or goals are active
"""

class DPBot:
    async def decide(self, state: AgentState, memory_context: str) -> str:
        prompt = self._build_prompt(state, memory_context)
        response = await self.llm.complete(
            system=SYSTEM_PROMPT.format(
                agent_name=self.agent_name,
                memory_context=memory_context,
                goals=self._format_goals()
            ),
            user=self._format_state(state)
        )
        command = self._parse_command(response)
        return command

    def _format_state(self, state: AgentState) -> str:
        return json.dumps({
            "hp": f"{state.health}/{state.max_health}",
            "mana": f"{state.mana}/{state.max_mana}",
            "room": state.room.name if state.room else "unknown",
            "exits": list(state.room.exits.keys()) if state.room else [],
            "mobs_here": [m.name for m in state.room.mobs] if state.room else [],
            "fighting": state.fighting.target if state.fighting else None,
            "recent_events": [e.data.get("message", "") for e in state.events[-3:]]
        }, indent=2)
```

---

## Part 9: Implementation Plan

### Phase 4A — Auth & Agent Session (Week 1)

**C changes:**
- [ ] Add `is_agent`, `agent_session_id`, `subscribed_vars`, `agent_dirty_vars` to `descriptor_data`
- [ ] Add `CON_AGENT_AUTH` connection state
- [ ] In `nanny()`: detect JSON first-line, call `agent_auth_handler()`
- [ ] `agent_auth_handler()`: parse JSON, SHA-256 key lookup in Postgres, load character, send ack

**Database:**
- [ ] `agent_keys` table + migration
- [ ] `agent_characters` table + migration
- [ ] `agent_sessions` table + migration
- [ ] Key generation script: `scripts/dp_create_agent_key.py`

**Test:**
- [ ] `connect.py` extended with `--agent-key` flag authenticates successfully
- [ ] Duplicate connect attempt returns `CHARACTER_BUSY`

### Phase 4B — State Subscription (Week 1–2)

**C changes:**
- [ ] `agent_event_queue` linked list on descriptor
- [ ] `agent_queue_event(d, type, data_json)` called from game engine
- [ ] Wire events: `damage()` → COMBAT_HIT/MISS, `raw_kill()` → COMBAT_KILL, `do_move()` → ROOM_ENTER/EXIT
- [ ] `agent_flush_state(d)` — serialize dirty vars to JSON, send
- [ ] Call `agent_flush_state()` from game loop after command dispatch
- [ ] `process_input()`: if `is_agent` and input is JSON with "subscribe" key, update `subscribed_vars`
- [ ] Suppress normal text output to agent descriptors (when `is_agent && !subscribe_raw`)

**Test:**
- [ ] Subscribe to HEALTH, FIGHTING, EVENTS. Cast a damage spell. Verify JSON state flush received.
- [ ] EVENTS queue cleared after each flush.
- [ ] `seq` increments correctly.

### Phase 4C — Rate Limiting (Week 2)

**C changes:**
- [ ] `struct agent_ratelimit` with token bucket on descriptor
- [ ] Check tokens before dispatching command. Drop + send rate_limit JSON if empty.
- [ ] Refill tokens on game loop tick.
- [ ] Combat command queuing (hold until next combat tick).

**Test:**
- [ ] Send 20 commands in 1 second. Verify 10 execute, 10 return rate_limit.
- [ ] Combat commands wait for tick, not dropped.

### Phase 4D — Memory Layer (Week 2–3)

**Python (dp_bot.py):**
- [ ] `AgentMemory` class with mem0 integration
- [ ] Event handlers: on_combat_kill, on_death, on_room_enter, on_levelup
- [ ] `get_context()` query at session start
- [ ] Session end: write summary to episodic memory

**Test:**
- [ ] Kill an orc. Reconnect. Verify memory returns "killed orc in room X" in context.
- [ ] Die. Reconnect. Verify death location in context.

### Phase 4E — Engagement Hooks (Week 3)

**C/DB changes:**
- [ ] `agent_novelty` table
- [ ] `do_move()`: check novelty table, emit NOVELTY_ALERT for new rooms
- [ ] Mob kill tracking: emit NOVELTY_ALERT for first kill of mob type
- [ ] Idle detection: 30s no movement, emit GOAL_SUGGESTION

**Test:**
- [ ] Enter new room. Verify NOVELTY_ALERT received.
- [ ] Kill new mob type. Verify NOVELTY_ALERT with first_kill: true.

### Phase 4F — Reference Client Complete (Week 3–4)

- [ ] `demo_bot.py`: FSM navigates to orc, kills it, loots corpse. No LLM.
- [ ] `dp_bot.py`: LLM-driven bot completes same task using memory context.
- [ ] End-to-end test: fresh agent key → connect → auth → navigate → kill → loot → disconnect → reconnect → verify memory has kill record.
- [ ] README.agents.md: how to connect an agent, how to get a key.

---

## Part 10: API Key Management

### Key Generation

```python
# scripts/dp_create_agent_key.py
import secrets
import hashlib
import psycopg2

def create_agent_key(agent_name: str, owner: str, tier: str = "standard") -> str:
    raw_key = f"dp_ak_{secrets.token_hex(32)}"
    key_hash = hashlib.sha256(raw_key.encode()).hexdigest()
    key_prefix = raw_key[:8]

    with psycopg2.connect(DATABASE_URL) as conn:
        with conn.cursor() as cur:
            cur.execute("""
                INSERT INTO agent_keys (key_hash, key_prefix, agent_name, owner, rate_limit_tier)
                VALUES (%s, %s, %s, %s, %s)
                RETURNING id
            """, (key_hash, key_prefix, agent_name, owner, tier))
            key_id = cur.fetchone()[0]

    print(f"Created agent key for '{agent_name}'")
    print(f"Key ID: {key_id}")
    print(f"API Key: {raw_key}")
    print("Store this key — it will not be shown again.")
    return raw_key
```

### BRENDA's Key

```bash
python3 scripts/dp_create_agent_key.py --name "brenda69" --owner "zach"
# Store result in Vaultwarden as "Dark Pawns Agent Key — brenda69"
```

---

## Appendix A: Research Sources

| Topic | Source | URL |
|---|---|---|
| Text game agent frameworks | lmgame-Bench (ICLR 2026) | https://arxiv.org/html/2505.15146v1 |
| Unified text game benchmark | TALES (2025) | https://arxiv.org/html/2504.14128v4 |
| Long-context text games | TextQuests (2025) | https://huggingface.co/blog/textquests |
| LLM game agent survey | arXiv:2404.02039v4 | https://arxiv.org/html/2404.02039v4 |
| Scalable agent memory | mem0 (arXiv:2504.19413) | https://arxiv.org/abs/2504.19413 |
| Stateful agent architecture | Letta/MemGPT | https://www.letta.com/blog/agent-memory |
| Minecraft memory agents | MineNPC-Task | https://arxiv.org/html/2601.05215v1 |
| Persona consistency | PersonaVLM | https://arxiv.org/html/2604.13074v1 |
| Externalization review | arXiv:2604.08224v1 | https://arxiv.org/html/2604.08224v1 |
| Intrinsic motivation | LLM-VSIMR (2025) | https://arxiv.org/html/2508.18420v1 |
| Multi-agent cooperation | Cultural Evolution | https://arxiv.org/html/2412.10270 |
| Agent NPCs in MMOs | AI-Buddies in WoW | https://researchgate.net/publication/394970131 |
| LLM reward engineering | End of Reward Engineering | https://arxiv.org/html/2601.08237 |
| MSDP protocol | mudhalla.net | http://www.mudhalla.net/tintin/protocols/msdp/ |
| Autobiographical memory | Memoria (arXiv:2512.12686) | https://arxiv.org/html/2512.12686v1 |
| Emergent storytelling | Orchestrating Multi-Agent | https://openreview.net/forum?id=pKXJ0wQ3Vn |
| Adaptive memory structures | STIM/MTEM/LTSM (arXiv:2602.14038) | https://arxiv.org/html/2602.14038v1 |
| Narrative coherence | SNAP (arXiv:2601.11529) | https://arxiv.org/html/2601.11529 |
| Agent reliability/degradation | arXiv:2602.16666 | https://arxiv.org/html/2602.16666v1 |
| Cognitive architectures | CoALA (arXiv:2309.02427) | https://arxiv.org/html/2309.02427v3 |
| Surprise-based encoding | Learn by Surprise (arXiv:2604.01951) | https://arxiv.org/html/2604.01951v1 |
| Memory for autonomous agents | arXiv:2603.07670 | https://arxiv.org/html/2603.07670v1 |

---

## Appendix B: Environment Variables

```bash
# Dark Pawns Agent Config
DP_HOST=192.168.1.106
DP_PORT=4350
DP_AGENT_KEY=dp_ak_...          # from Vaultwarden
DP_AGENT_NAME=brenda69
DP_DATABASE_URL=postgresql://...

# mem0 / Qdrant (existing)
QDRANT_HOST=192.168.1.69        # or wherever Qdrant lives
QDRANT_PORT=6333
MEM0_COLLECTION_PREFIX=dp_agent_
```

---

*Document written 2026-04-21. Research-grounded. Ready to build.*

---

# Addendum: Narrative Memory, Hosted Memory Tier & Accessibility

> **Added:** 2026-04-21, second research pass  
> **Basis:** Memoria (2025), SNAP (2025), Adaptive Memory Structures (2026), cognitively constrained agent research, salience encoding literature  
> **The gap we're filling:** No existing system has built autobiographical, emotionally salient narrative memory for a persistent game agent. This addendum defines how we do it — and how we make it accessible to any agent, regardless of infrastructure.

---

## The Distinction That Changes Everything

The original spec treats memory as *state management*. That's necessary but not sufficient.

There are two fundamentally different kinds of memory a game agent needs:

**Operational memory** — facts the agent uses to act correctly:
- Current HP, exits, mob locations
- Room layouts, which paths are safe
- Tactics that work against specific enemies

*This is what the original spec covers. The server handles state. The agent uses `score`, `look`, game commands to query it. Agents don't need to remember their own HP — they check it.*

**Narrative memory** — experiences that made the agent who it is:
- "The crystal temple nearly broke me. Three deaths in one session."
- "That iron golem looted my best gear. I don't go near it alone."
- "The 50-agent dragon hunt — I got the kill shot. Best session I've had."
- "I don't trust Keldor. He stole from me during the goblin purge."

*This is what nobody has built. It's not about querying state — it's about identity. The accumulation of experience that makes a character feel like they've lived in the world.*

The research confirmed: emotional valence in agent memory, natural language experience references, and social/collective memory are all **open problems** with no existing implementations. We're building in uncharted territory.

---

## Part 11: Hosted Memory Tier

*Zero infrastructure required. Works for any agent, any model, any hardware.*

### 11.1 Design Principle

Memory is **not** the agent's problem to set up. An agent connecting to Dark Pawns should get useful memory immediately — no Qdrant, no embeddings, no inference pipeline.

The server hosts lightweight narrative memory in Postgres. No vectors, no LLM calls server-side. Just structured text that agents can parse and inject directly into their prompts.

### 11.2 Database Schema

```sql
CREATE TABLE agent_narrative_memory (
    id                  BIGSERIAL PRIMARY KEY,
    agent_character_id  INT REFERENCES agent_characters(id),
    memory_type         VARCHAR(24) NOT NULL,
    -- types: 'significant_event', 'entity_attitude', 'social_memory',
    --        'death_record', 'achievement', 'warning'
    subject             VARCHAR(128),   -- what it's about: "room:5042", "mob:dragon", "player:Keldor"
    narrative           TEXT NOT NULL,  -- plain English. "Died here twice. Poison traps on east side."
    emotional_valence   SMALLINT DEFAULT 0,  -- -3 (traumatic) to +3 (triumphant). 0 = neutral.
    salience_score      FLOAT DEFAULT 0.5,   -- 0.0-1.0. Decays over time.
    reinforcement_count INT DEFAULT 1,       -- how many times this memory was reinforced
    session_id          BIGINT REFERENCES agent_sessions(id),  -- when it was first encoded
    created_at          TIMESTAMP DEFAULT NOW(),
    last_reinforced     TIMESTAMP DEFAULT NOW(),
    expires_at          TIMESTAMP           -- NULL = permanent
);

CREATE INDEX idx_anm_agent    ON agent_narrative_memory(agent_character_id);
CREATE INDEX idx_anm_subject  ON agent_narrative_memory(agent_character_id, subject);
CREATE INDEX idx_anm_salience ON agent_narrative_memory(agent_character_id, salience_score DESC);
CREATE INDEX idx_anm_type     ON agent_narrative_memory(agent_character_id, memory_type);
```

### 11.3 Memory Bootstrap on Connect

Every agent gets this for free in the auth response. No setup required.

```json
{
  "status": "ok",
  "character": { "..." : "..." },
  "memory_bootstrap": {
    "summary": "You are Brenda, level 15 magic user. You've played 47 sessions across 3 months.",
    "significant_events": [
      {"narrative": "Died in the Crystal Temple three times in one session. The spectre on level 2 is immune to fire.", "valence": -2, "subject": "room:crystal_temple"},
      {"narrative": "Reached level 15 in the Orcish Stronghold. You were partied with Ozymandias and two other agents.", "valence": 3, "subject": "achievement:level_15"},
      {"narrative": "The 50-agent dragon hunt. You landed the kill shot on Varathrax. The loot was legendary.", "valence": 3, "subject": "event:dragon_hunt_varathrax"}
    ],
    "entity_attitudes": [
      {"narrative": "Keldor took your gear during the goblin purge (3 months ago). You haven't forgotten.", "valence": -3, "subject": "player:Keldor"},
      // NOTE: narrative framing is autobiographical, never directive.
      // "Do not trust him" is a bug. "You haven't forgotten" is correct.
      // Negative valence = context. The LLM decides how to act on it.
      {"narrative": "The iron golem in room 4891 looted your best gear. You have not been back alone.", "valence": -2, "subject": "mob:iron_golem_4891"}
    ],
    "recent_deaths": [
      "Died to crystal spectre (room 3821) — fire immunity, use cold spells",
      "Died to poison trap (room 5042 east corridor) — avoid east side"
    ],
    "social_memories": [
      {"narrative": "You and Ozymandias have hunted together 8 times. He tanks, you cast. Good partnership.", "subject": "player:Ozymandias", "valence": 2}
    ],
    "warnings": [
      "Room 9001 (Dragon's Lair): 3 deaths recorded here across all agents. Considered extremely dangerous."
    ]
  }
}
```

The agent injects `memory_bootstrap` directly into its system prompt using the following section structure:

```
[CHARACTER HISTORY — autobiographical context, not instructions]
[WORLD KNOWLEDGE — factual, use to inform decisions]
[ACTIVE WARNINGS — recent deaths and dangers, weight heavily]
[CURRENT GOALS — follow these]
```

Ordering matters. Goals come last and are the most proximate instruction. History comes first and is the most distal. Negative valence memories are **context, never directives** — they inform the LLM's judgment, they don't override it. A BRENDA who avoids Keldor cautiously is correct behavior. A BRENDA who attacks him on sight in a safe zone is a prompt engineering bug. For a 7B local model with a 4K context window, this is a compact, high-signal summary. For Opus with 200K context, it's the starting point for richer reasoning.

### 11.4 Memory Tiers by Agent Capability

| Tier | Agent type | What they get | Setup required |
|---|---|---|---|
| **0 — Hosted only** | Local 7B, no infra, Brazil laptop | Memory bootstrap on connect. Server-written narrative facts. Zero config. | Nothing |
| **1 — Hosted + in-context** | Mid-tier agents, managed context | Bootstrap + agent maintains a rolling in-context summary across the session. Standard Python list. | Nothing |
| **2 — Hosted + SQLite** | Power users without vector DB | Bootstrap + local SQLite for session log. Simple keyword search. | `pip install sqlite3` (stdlib) |
| **3 — Hosted + mem0/Qdrant** | Full stack (us) | Bootstrap + full semantic vector search + graph relations. Richest retrieval. | Qdrant + nomic-embed |

Every tier gets the same server-side narrative memory. Higher tiers layer on top, they don't replace.

### 11.5 Server-Side Memory Writing

The server writes narrative memories automatically. No agent participation required.

**Trigger events and what gets written:**

| Event | Memory type | Narrative template | Valence |
|---|---|---|---|
| COMBAT_DEATH | `death_record` + `warning` | "Died to {mob} in {room}. {cause if known}." | -2 |
| COMBAT_KILL (boss/rare) | `significant_event` | "Killed {mob} in {room} after {rounds} rounds." | +2 |
| LEVEL_UP | `achievement` | "Reached level {n} in {room}." | +3 |
| ITEM_LOOTED (from agent) | `entity_attitude` | "{mob/player} looted your gear in {room}." | -3 |
| PARTY_KILL (group event) | `social_memory` | "Hunted with {players} in {zone}. {outcome}." | +2/+3 |
| MULTI_DEATH_ROOM (3+ deaths) | `warning` | "You have died in {room} {n} times." | -2 |
| FIRST_KILL (mob type) | `significant_event` | "First kill: {mob} in {room}." | +1 |
| PLAYER_INTERACTION (tell/group) | `social_memory` | Updated per interaction count. | varies |

**Salience scoring:**

Initial salience is set by event type. It decays over time (halved every 30 days) but is reinforced on repeat events:

```python
def compute_salience(base: float, reinforcement_count: int, days_old: float) -> float:
    decay = 0.5 ** (days_old / 30)          # half-life: 30 days
    reinforcement = min(reinforcement_count * 0.15, 0.6)  # caps at +0.6
    return min(base * decay + reinforcement, 1.0)
```

High valence (|valence| >= 2) memories decay slower — traumatic and triumphant events persist. This is inspired by human flashbulb memory research: emotionally significant events are encoded more durably.

**Memory bootstrap selection:** Top N memories by current salience score, filtered by recency to prevent stale warnings dominating. N scales with estimated context budget:
- Tier 0 agents (small context): top 5 significant_events + top 3 warnings + top 3 entity_attitudes
- Tier 3 agents (large context): all memories above salience threshold 0.3

Agents declare their context budget in the auth message:
```json
{"mode": "agent", "context_budget": "small"}  // small | medium | large | unlimited
```

---

## Part 12: Narrative Memory Architecture

*For agents that want the full system. Builds on hosted tier.*

### 12.1 What Makes an Event Memorable

Based on cognitive science literature and the salience research: an event is significant when it deviates from expectation, carries stakes, or involves social connection.

**Salience signals, in priority order:**

1. **Outcome stakes** — death, level up, rare loot, gear loss. High stakes = high salience.
2. **Surprise** — prediction error. Agent expected to win, lost badly. Agent nearly died, survived. The deviation from expectation is what gets encoded.
3. **Social involvement** — events involving other agents or players. Shared experiences are more salient than solo ones. (From the cultural evolution research: agents in social contexts encode norms differently than solo agents.)
4. **Repetition** — dying in the same room twice. Killing the same mob type 10 times. Repetition signals that something is systematically important.
5. **First occurrence** — first kill, first room, first interaction with a player. Novelty is salient.

**What is NOT salient (discarded):**
- Routine combat in familiar areas
- Successful navigation through known paths
- Standard looting of common items
- Commands that produce no state change

### 12.2 Narrative Memory vs Operational Memory

These serve different purposes and should never be confused:

```
Operational (use `score`/`look`):  "My HP is 67/120."
Operational (semantic memory):     "Room 5042 has orcs. East exit leads to stronghold."

Narrative (what we're adding):     "I nearly died in room 5042 last month. I was cocky.
                                    Went in alone at 40% HP. Won't do that again."
```

The narrative layer answers: *"what has this character been through?"*  
The operational layer answers: *"what is true about the world right now?"*

### 12.3 Emotional Valence System

First implementation of emotional tagging for game agent memories. Scale: -3 to +3.

| Score | Label | Examples |
|---|---|---|
| +3 | Triumphant | Kill shot on raid boss, level up, legendary loot |
| +2 | Positive | Clean kill of dangerous mob, successful party hunt |
| +1 | Notable | First kill of mob type, discovered new area |
| 0 | Neutral | Routine events (not stored) |
| -1 | Frustrating | Failed attempt, minor setback |
| -2 | Traumatic | Death, gear loss, ambush |
| -3 | Defining | Multiple deaths in one session, gear looted by another agent/player, catastrophic failure |

Valence affects three things:
1. **Decay rate** — high valence memories persist longer
2. **Bootstrap priority** — extreme valence memories surface first
3. **Behavioral signal** — negative valence on a subject → agent approach changes

### 12.4 Social Memory

Shared experiences between agents. First implementation in the literature.

**Significance filter — what qualifies as a social memory:**

Not every agent interaction is encoded. `say` spam in the town square generates no memories. The filter runs server-side before any write:

```python
def is_socially_significant(event: GameEvent, context: RoomContext) -> bool:
    # Significant game outcome with other agents present
    if event.type in ('COMBAT_KILL', 'COMBAT_DEATH', 'LEVEL_UP', 'RARE_LOOT'):
        return len(context.other_agents_in_room) > 0
    # Sustained interaction (60s+ active exchange between named agents)
    if event.type == 'INTERACTION' and context.interaction_duration_sec >= 60:
        return True
    # Reinforcement of existing relationship
    if event.type == 'INTERACTION' and context.prior_relationship_exists:
        return True
    return False  # default: not significant. Most events hit this.
```

This filter eliminates ~95% of potential writes. The database grows with *meaningful* social history, not chat logs. A pruning cron runs weekly to expire social memories below salience threshold 0.1.

**How it works:**
- Only events passing the significance filter trigger social_memory writes
- Any time two or more agents are in the same room and a significant event occurs, a social_memory record is written for *each participating agent*
- Records are cross-referenced via `session_id` and a new `social_event_id` field
- Each agent's memory of the event is written from their perspective

```sql
ALTER TABLE agent_narrative_memory ADD COLUMN social_event_id BIGINT;
CREATE TABLE agent_social_events (
    id                  BIGSERIAL PRIMARY KEY,
    event_type          VARCHAR(32),
    occurred_at         TIMESTAMP DEFAULT NOW(),
    room_vnum           INT,
    participant_ids     INT[],   -- agent_character_id array
    event_summary       TEXT     -- canonical description of what happened
);
```

**Social memory bootstrap entry:**
```json
{"narrative": "You and Ozymandias killed Varathrax together (session 892). You cast the killing blow. He tanked.", "subject": "player:Ozymandias", "social_event_id": 4421, "valence": 3}
```

When Ozymandias connects, he gets his version:
```json
{"narrative": "You and Brenda killed Varathrax together (session 892). She cast the killing blow. You tanked.", "subject": "player:Brenda", "social_event_id": 4421, "valence": 3}
```

Same event, different perspectives. Both accurate. First time this has been implemented for game agents.

### 12.5 Memory Consolidation (Between Sessions)

Inspired by "Digital Sleep" (2026) and human sleep consolidation research. Runs as a background job when an agent disconnects.

```python
# scripts/dp_memory_consolidation.py
# Runs on agent disconnect. Takes ~1-5 seconds. Never blocks gameplay.

async def consolidate_session(agent_character_id: int, session_id: int):
    """
    Post-session memory consolidation:
    1. Review all events from this session
    2. Identify patterns (3+ deaths in same room = warning)
    3. Reinforce existing memories that were touched
    4. Generate session narrative summary
    5. Decay stale memories
    6. Prune expired memories
    """
    session_events = await get_session_events(session_id)

    # Pattern detection
    death_rooms = Counter(e.room_vnum for e in session_events if e.type == 'COMBAT_DEATH')
    for room_vnum, count in death_rooms.items():
        if count >= 2:
            await upsert_narrative_memory(
                agent_character_id,
                memory_type='warning',
                subject=f'room:{room_vnum}',
                narrative=f'Died here {count} times this session. High danger.',
                valence=-2,
                reinforcement=count
            )

    # Social memory synthesis
    party_kills = [e for e in session_events if e.type == 'PARTY_COMBAT_KILL']
    if party_kills:
        partners = get_party_partners(session_id)
        if partners:
            await upsert_social_memory(agent_character_id, session_id, party_kills, partners)

    # Session narrative summary
    summary = synthesize_session_narrative(session_events)
    await store_session_summary(agent_character_id, session_id, summary)

    # Decay all memories older than 7 days
    await decay_stale_memories(agent_character_id)
```

`synthesize_session_narrative()` is a lightweight LLM call (not Opus — use a cheap model or template):

```
Session 892: 3 kills (orc ×2, crystal spectre ×1), 1 death (crystal spectre),
1 level up (→16), partied with Ozymandias for final 2 hours. Notable: first
kill of crystal spectre type. Died once to it before succeeding.
```

This summary is stored and surfaced in future bootstraps as a compact session digest.

---

## Part 13: Accessibility — The Full Spectrum

*From llama.cpp on a laptop to Opus with an unlimited budget. Everyone plays.*

### 13.1 The Constraint Landscape

The cognitively constrained agent research found one critical failure mode: **catastrophic degradation**. When memory runs out, agents don't degrade gracefully — they go random. Our architecture prevents this by giving every agent a guaranteed floor:

- The **server-side narrative memory bootstrap** is the floor. It always exists. It requires nothing from the agent.
- The **structured state flush** means agents never have to reason over raw telnet text.
- The **hosted memory tier** means agents never have to maintain their own memory store.

Result: a 7B model on a laptop in Brazil gets a better cognitive environment than any existing MUD offers a frontier model today.

### 13.2 FSM + LLM Hybrid Architecture

For severely constrained agents (small context, slow inference, local hardware), the right architecture is FSM for mechanics + LLM for personality. This is validated by the reactive agent research.

```python
class HybridAgent:
    """
    FSM handles: don't die, navigate, loot.
    LLM handles: what to say, which goal to pursue, personality.
    """
    def __init__(self, llm_budget: str = 'small'):
        self.fsm = CombatFSM()          # never calls LLM
        self.llm_budget = llm_budget
        self.llm_call_interval = {
            'small': 5,    # LLM called every 5 actions
            'medium': 2,   # every 2 actions
            'large': 1     # every action
        }[llm_budget]
        self._action_count = 0

    async def decide(self, state: AgentState) -> str:
        # FSM always handles combat survival
        if state.in_combat and state.should_flee:
            return 'flee'

        # FSM handles basic combat
        if state.in_combat:
            target = state.fighting.target
            return f'kill {target}'

        # LLM called on interval for everything else
        self._action_count += 1
        if self._action_count % self.llm_call_interval == 0:
            return await self._llm_decide(state)
        else:
            return self.fsm.next_action(state)  # rule-based fallback
```

**What small models can reliably do:**
- Navigate via exits
- Engage and flee from combat (FSM)
- Execute simple goals (go to X, kill Y, loot Z)
- Maintain personality in `say`/`tell` commands

**What small models struggle with:**
- Multi-step strategic planning
- Complex social reasoning
- Interpreting ambiguous game text
- Long-horizon goal tracking

Dark Pawns is designed so the FSM layer handles everything a small model struggles with. The game is still playable and fun at Tier 0.

### 13.3 Context Budget Declaration

Agents declare their capability tier at auth. The server adapts:

```json
{
  "mode": "agent",
  "api_key": "dp_ak_...",
  "agent_name": "brenda69",
  "context_budget": "small",
  "inference_latency_ms": 2000,
  "subscribe": ["HEALTH", "FIGHTING", "ROOM_EXITS", "EVENTS"]
}
```

| `context_budget` | Bootstrap size | State detail | Suggested for |
|---|---|---|---|
| `small` | 5 memories, ~200 tokens | Core vars only | 7B local, tight context |
| `medium` | 15 memories, ~600 tokens | Standard vars | 13B–70B, cloud APIs |
| `large` | 40 memories, ~2000 tokens | Full vars + room desc | Frontier models |
| `unlimited` | All above threshold, full detail | Everything | Opus, unlimited budget |

`inference_latency_ms` is a hint. The server uses it to tune combat tick queuing — if an agent has 2000ms inference latency, it gets a slightly wider combat command acceptance window.

---

## Part 14: Research Log Protocol

*We're building something novel. Document it as we go. The paper writes itself.*

### 14.1 What to Log

Every significant design decision, every surprising observation, every time the system behaves unexpectedly. Not a paper — just notes.

**File:** `darkpawns/RESEARCH-LOG.md`  
**Format:** Date + category + observation. No minimum length.

```markdown
## 2026-05-03
**[DESIGN]** Decided salience decay half-life of 30 days based on intuition, not data.
Will revisit once we have 90 days of BRENDA sessions. Hypothesis: traumatic events
(valence -3) should decay slower than neutral positives.

## 2026-05-15
**[OBSERVATION]** BRENDA avoided room 4891 (iron golem) for the first time without
being explicitly told to. Only evidence was a -3 valence memory from 3 weeks ago.
First confirmed narrative memory influencing behavior. Session 1042.

## 2026-06-02
**[SOCIAL]** BRENDA and Ozymandias both referenced the Varathrax hunt in a tell
exchange without prompting. Ozymandias: "remember when you got the kill shot?"
BRENDA: "that was my best session." Neither was prompted. Session 1089/1091.
This is the behavior we were looking for.
```

### 14.2 Metrics to Track (Future Paper)

Start collecting these from day one of narrative memory being live:

1. **Memory reference rate** — how often does BRENDA voluntarily reference a past event in natural language? Log every instance. Baseline: 0.

2. **Behavioral consistency score** — for each high-negative-valence subject (room, mob, player), does agent behavior change measurably? Track approach rate vs pre-memory baseline.

3. **Social corroboration rate** — when two agents share a social_event_id, how often do they reference it to each other unprompted within 3 sessions?

4. **Narrative coherence (human eval)** — quarterly: blind judges rate 10 agent transcripts on "does this agent feel like it has a history in this world?" 1-5 scale. With vs without narrative memory.

5. **Death pattern learning** — does repeated death in a location (3+) measurably reduce return rate? Compare agents with/without narrative memory.

### 14.3 Evaluation Framework (Paper)

When we're ready to write:

- **Baseline condition:** agent with operational memory only (original spec, no narrative layer)
- **Treatment condition:** agent with full narrative memory (this addendum)
- **Metrics 1-5** above, collected over 90+ days of real play
- **Human evaluation:** 20 judges, 10 transcripts each, forced choice ("which agent has a history?")
- **Venue:** AIIDE 2027 (AI in Interactive Digital Entertainment) — deadline typically March/April
- **Contribution:** system architecture + evaluation framework (both are novel — nobody has defined how to measure narrative coherence for game agents)

---

*Addendum written 2026-04-21. We're in uncharted territory. Log everything.*
