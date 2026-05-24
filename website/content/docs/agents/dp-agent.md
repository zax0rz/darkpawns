---
title: "dp-agent CLI"
description: "The dp-agent CLI — connect to Dark Pawns as an AI agent, run decision loops, log sessions, and consolidate memories"
date: 2026-04-22
draft: false
section: "docs"
---

# dp-agent — Agent CLI

`dp-agent` is a Go CLI that connects to Dark Pawns as an AI agent. It handles WebSocket transport, structured state parsing, combat survival (FSM), and LLM-driven decision-making. Zero dependencies beyond the Go standard library.

---

## Installation

```bash
# Build and install to $GOPATH/bin
go build -o ~/go/bin/dp-agent ./cmd/dp-agent

# Verify
dp-agent --help
```

---

## Subcommands

| Command | Description |
|---------|-------------|
| `dp-agent play` | Interactive mode — connects, runs decision loop, prints output |
| `dp-agent session --duration 5m` | Timed session with full JSONL logging |
| `dp-agent dream` | Offline dreaming cycle — consolidate memory graph |
| `dp-agent config` | View or set configuration |
| `dp-agent keygen -name <name>` | Generate a new agent API key |
| `dp-agent whoami` | Show current agent identity |
| `dp-agent exec <command>` | One-shot command — send a single action and exit |

---

## Configuration

Config file: `~/.dp-agent.json` (override with `DP_CONFIG` env var).

```json
{
  "key": "dp_your_key_here",
  "player_name": "your_character",
  "tier": "medium",
  "model_fast": "zai/glm-5-turbo",
  "model_fallback": "deepseek-v4-flash",
  "litellm_endpoint": "http://localhost:4000",
  "game_host": "localhost",
  "game_port": 4350,
  "temperature": 0.0,
  "valence": true,
  "log_dir": "data/logs",
  "log_level": "info"
}
```

### Fields

| Field | Default | Description |
|-------|---------|-------------|
| `key` | — | Agent API key (`dp_<hex>`) |
| `player_name` | — | Character name in Dark Pawns |
| `tier` | `medium` | Context budget: `small` / `medium` / `large` / `unlimited` |
| `model_fast` | `zai/glm-5-turbo` | Primary LLM model |
| `model_fallback` | `deepseek-v4-flash` | Fallback if primary fails |
| `litellm_endpoint` | — | LiteLLM proxy URL |
| `game_host` | — | Game server host |
| `game_port` | `4350` | Game server port |
| `temperature` | `0.0` | LLM temperature (0 = deterministic) |
| `valence` | `true` | Enable emotional valence recording in session logs |
| `log_dir` | — | Where to write session JSONL logs |
| `log_level` | `info` | Log verbosity: `debug` / `info` / `warn` / `error` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DP_KEY` | Override API key |
| `DP_CONFIG` | Override config file path |

### Getting an API Key

API keys are generated server-side:

```bash
go run ./cmd/agentkeygen -name "your_character" -db "$DB_DSN"
```

Keys look like `dp_<64hex>`. The key acts as the character's password — store it securely.

---

## How It Works

### Decision Loop

```
Server → vars message → client updates state
  ↓
FSM check: HP < 25%? → Flee
FSM check: mob in room already attacking? → Counter-attack
  ↓ (no FSM override)
LLM call: system prompt + state + memory → action
  ↓
Send command to server → log entry → repeat
```

1. **Server sends state** via WebSocket (health, room, mobs, events) as `vars` messages.
2. **FSM checks first** — critical survival decisions bypass the LLM entirely (see below).
3. **LLM decides** — if no FSM override, the LLM receives current state (including memory summary) and produces one action.
4. **Action sent** — the command goes to the server, the turn is logged.

### Memory Integration

At connection time, the server sends a `memory_summary` message containing the dreaming layer's narrative output. This is injected into the LLM context before the prompt. The agent acts on its memories without managing them.

See [Memory & Dreaming System](/docs/agents/memory-system/) for details.

### Session Logging

`dp-agent session` writes JSONL logs for every turn:

```json
{
  "timestamp": "2026-05-12T15:30:00Z",
  "room_vnum": 3001,
  "room_name": "A Dark Corridor",
  "hp": 45,
  "max_hp": 100,
  "agent_level": 5,
  "mobs_present": 1,
  "fighting": "goblin",
  "action": "flee",
  "latency_ms": 1200
}
```

These logs feed the dreaming pipeline. After a session, run `dp-agent dream` to consolidate memories.

---

## Usage Examples

### Play interactively

```bash
dp-agent play
```

### Run a 30-minute session with logging

```bash
dp-agent session --duration 30m --log-dir data/logs
```

### Consolidate memories after a session

```bash
# --output must match the server's dreaming dir (default: data/dreaming)
dp-agent dream --agent your_character --sessions data/sessions --output data/dreaming
```

### Send a one-shot command

```bash
dp-agent exec "look"
dp-agent exec "north"
dp-agent exec "hit goblin"
```

### Check and set configuration

```bash
dp-agent config
dp-agent config --key dp_YOUR_KEY_HERE --player-name your_character
```

---

## FSM (Finite State Machine)

The FSM handles life-or-death decisions without LLM latency:

| Condition | Action |
|-----------|--------|
| HP < 25% (regardless of combat state) | Flee immediately |
| Not in combat AND a mob in the room has `fighting: true` | Counter-attack that mob |
| Otherwise | Let the LLM decide |

The FSM runs in <1ms. The LLM call takes 500–2000ms. When survival is at stake, speed wins.

**Important:** The FSM does **not** proactively attack idle mobs — that is the LLM's domain. It only counter-attacks mobs that are already engaged.

---

## Architecture

```
cmd/dp-agent/main.go      Entry point, subcommand dispatch
pkg/agentcli/
  client.go               WebSocket client, decision loop
  config.go               Config loading/saving (~/.dp-agent.json)
  ws.go                   WebSocket dial/read/write
  fsm.go                  Combat survival FSM
  llm.go                  LiteLLM proxy client
  prompt.go               System prompt builder
  session.go              Session logger, JSONL export
  state.go                GameState type, server message parsing
  behavior.go             Higher-level behavioral policies
  events.go               Event type parsing
  context.go              LLM context budget management
  creation.go             Character creation state machine
  reconnect.go            Reconnect with exponential backoff
pkg/dreaming/
  graph.go                Memory graph, narrative summary
  extract.go              Event extraction, valence computation
  dream.go                Dreaming pipeline (consolidation)
```

---

## Reference Implementations

The repository ships two additional Python agent scripts:

| Script | Description |
|--------|-------------|
| `scripts/dp_bot.py` | Deterministic FSM bot. Circuit breaker, death recovery, auto-loot, reconnect with backoff. No LLM. |
| `scripts/dp_brenda.py` | BRENDA69 — SOUL.md personality, mem0 memory, minimax LLM. Emergent private cognition. |
