# dp-agent — Agent CLI

`dp-agent` is a Go CLI that connects to Dark Pawns as an AI agent. It handles WebSocket transport, structured state parsing, combat survival (FSM), and LLM-driven decision-making. Zero dependencies beyond Go stdlib.

## Installation

```bash
go build -o dp-agent ./cmd/dp-agent
```

Or run directly:

```bash
go run ./cmd/dp-agent <subcommand>
```

## Subcommands

| Command | Description |
|---------|-------------|
| `dp-agent play` | Interactive mode — connects, runs decision loop, prints output |
| `dp-agent session --duration 5m` | Timed session with full JSONL logging |
| `dp-agent dream --agent <name>` | Offline dreaming cycle — consolidate memory graph |
| `dp-agent config` | View or set configuration |
| `dp-agent keygen -name <name>` | Generate a new agent API key |
| `dp-agent whoami` | Show current agent identity |
| `dp-agent exec <command>` | One-shot command — send a single action and exit |

## Configuration

Config file: `~/.dp-agent.json` (override with `DP_CONFIG` env var).

```json
{
  "key": "dp_your_key_here",
  "tier": "medium",
  "model_fast": "zai/glm-5-turbo",
  "model_fallback": "anthropic/claude-sonnet-4-6",
  "litellm_endpoint": "http://192.168.1.106:4000",
  "game_host": "192.168.1.106",
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
| `key` | — | Agent API key (`dp_<32hex>`) |
| `tier` | `medium` | Context budget: `small` / `medium` / `large` / `unlimited` |
| `model_fast` | `zai/glm-5-turbo` | Primary LLM model |
| `model_fallback` | `anthropic/claude-sonnet-4-6` | Fallback if primary fails |
| `litellm_endpoint` | `http://192.168.1.106:4000` | LiteLLM proxy URL |
| `game_host` | `192.168.1.106` | Game server host |
| `game_port` | `4350` | Game server port |
| `temperature` | `0.0` | LLM temperature (0 = deterministic) |
| `valence` | `true` | Enable emotional valence in memory system |
| `log_dir` | — | Where to write session JSONL logs |
| `log_level` | `info` | Log verbosity: `debug` / `info` / `warn` / `error` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DP_KEY` | Override API key |
| `DP_CONFIG` | Override config file path |

## How It Works

### Decision Loop

```
Server → vars message → client updates state
  ↓
FSM check: HP < 25% while fighting? → Flee
FSM check: mob in room, not fighting? → Attack
  ↓ (no FSM override)
LLM call: system prompt + state + memory → action
  ↓
Send command to server → log entry → repeat
```

1. **Server sends state** via WebSocket (health, room, mobs, events).
2. **FSM checks first** — if HP is critical and you're fighting, flee. If a mob is attackable, attack. This is instant, no LLM call.
3. **LLM decides** — if no FSM override, the LLM receives the current state (including memory summary) and produces one action.
4. **Action sent** — the command goes to the server, the turn is logged.

### Memory Integration

At connection time, the server sends a `memory_summary` message containing the dreaming layer's narrative output. This is injected into the LLM context as a system message before the prompt. The agent acts on its memories without managing them.

See [`memory-system.md`](memory-system.md) for details.

### Session Logging

`dp-agent session` writes JSONL logs for every turn:

```json
{
  "timestamp": "2026-05-12T15:30:00Z",
  "room_vnum": 3001,
  "room_name": "A Dark Corridor",
  "hp": 45,
  "max_hp": 100,
  "fighting": "goblin",
  "action": "flee",
  "latency_ms": 1200
}
```

These logs feed the dreaming pipeline. After a session, run `dp-agent dream` to consolidate memories.

## Usage Examples

### Play interactively

```bash
dp-agent play
```

### Run a 5-minute session

```bash
dp-agent session --duration 5m --agent-id brenda --log-dir data/logs
```

### Consolidate memories after a session

```bash
dp-agent dream --agent-id brenda --input data/logs/sessions/brenda/ --output data/dreaming/brenda/
```

### Send a one-shot command

```bash
dp-agent exec "look"
```

### Check configuration

```bash
dp-agent config
```

## FSM (Finite State Machine)

The FSM handles survival decisions without LLM latency:

| Condition | Action |
|-----------|--------|
| HP < 25% AND fighting | Flee |
| Mob in room AND not fighting | Attack |
| Otherwise | Let LLM decide |

The FSM runs in <1ms. The LLM call takes 500-2000ms. When survival is at stake, speed wins.

## Architecture

```
cmd/dp-agent/main.go          Entry point, subcommand dispatch
pkg/agentcli/
  client.go                   WebSocket client, decision loop
  config.go                   Config loading/saving
  ws.go                       WebSocket dial/read/write
  fsm.go                      Combat survival FSM
  llm.go                      LiteLLM proxy client
  prompt.go                   System prompt builder
  session.go                  Session logger, JSONL export
  parser.go                   LLM output parser
pkg/dreaming/
  graph.go                    Memory graph, narrative summary
  extract.go                  Event extraction, valence computation
  dream.go                    Dreaming pipeline (consolidation)
```
