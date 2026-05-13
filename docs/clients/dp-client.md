# dp-client — Human MUD Client

A modern terminal MUD client for Dark Pawns, built on [Zif](https://github.com/perlsaiyan/zif) with bubbletea. Connects via WebSocket, renders structured state in split panes, logs sessions to JSONL for the dreaming pipeline.

## Installation

### Pre-built Binaries

Download from [Releases](https://github.com/zax0rz/darkpawns/releases):

| Platform | Binary |
|----------|--------|
| macOS (arm64) | `dp-client-darwin-arm64` |
| Linux (amd64) | `dp-client-linux-amd64` |
| Windows (amd64) | `dp-client-windows-amd64.exe` |

### Build from Source

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns/dp-client
go build -o dp-client .
```

Requires Go 1.24+. No CGO. Cross-compiles anywhere.

## Quick Start

```bash
# 1. Generate an API key
go run ./cmd/agentkeygen -name "YourName" -db "$DATABASE_URL"

# 2. Connect
dp-client --dp --key dp_your_key_here --character Aidan

# 3. Play
# Type commands as you would in telnet. Same commands, same world.
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `--dp` | Connect to Dark Pawns (sets default host/port) |
| `--key <key>` | Agent API key for DP authentication |
| `--character <name>` | Character name |
| `--log-dir <dir>` | Directory for JSONL session logs |
| `--valence` | Enable emotional valence in memory (default: true) |
| `--version` | Display version |

## Configuration

### sessions.yaml

Located at `~/.config/zif/sessions.yaml`:

```yaml
sessions:
  - name: "aidan"
    address: "darkpawns.labz0rz.com:4350"
    autostart: true
    api_key: "dp_your_key_here"
    character: "Aidan"
    log_dir: "data/logs"
```

### Config Fields

| Field | Description |
|-------|-------------|
| `name` | Session name (used in TUI tabs) |
| `address` | `host:port` of the Dark Pawns server |
| `autostart` | Connect on client launch |
| `api_key` | DP agent API key (triggers WebSocket mode) |
| `character` | Character name for logging |
| `log_dir` | JSONL log directory (empty = disabled) |

## Features

### DP Integration

- **HP bar** — real-time health display in the status pane
- **Mob list** — shows attackable mobs with `target_string` for easy combat
- **Room tracking** — room name, vnum, and exits on every transition
- **Memory summary** — server-hosted narrative memory displayed in a prominent block

### Session Logging

Every command is logged to JSONL for the dreaming pipeline:

```json
{
  "timestamp": "2026-05-12T23:30:00Z",
  "room_vnum": 3001,
  "room_name": "A Dark Corridor",
  "hp": 85,
  "max_hp": 100,
  "agent_level": 12,
  "fighting": "goblin",
  "action": "hit",
  "args": ["goblin"]
}
```

Same format as `dp-agent`. The dreaming pipeline reads both interchangeably.

### Lua Engine

Zif's Lua scripting engine works out of the box. Write triggers for combat messages, aliases for common commands, or automated healing scripts. See [Zif's Lua docs](https://github.com/perlsaiyan/zif#lua-modules).

### Split Panes

```
#split h main sidebar 30    # 30% sidebar on the right
#unsplit sidebar            # close the sidebar
#focus main                 # switch focus to main pane
```

### Built-in Commands

| Command | Description |
|---------|-------------|
| `#help` | Show all commands |
| `#session <name> <host:port>` | Create/switch session |
| `#sessions` | List active sessions |
| `#split [h\|v] <pane> <type> <percent>` | Split pane |
| `#unsplit <pane>` | Remove pane |
| `#modules` | List Lua modules |
| `#tickers` | List active timers |

## Architecture

```
dp-client
├── main.go                    Entry point, bubbletea model
├── session/
│   ├── handler.go             Session struct, DPGameState, WebSocket connect
│   ├── reader.go              mudReader() — JSON message dispatch
│   ├── jsonl_logger.go        JSONL session logging
│   ├── commands.go            Internal commands (#help, #split, etc.)
│   ├── lua_api.go             Lua API bindings
│   ├── actions.go             Trigger system
│   ├── aliases.go             Alias system
│   ├── events.go              Event system
│   ├── queue.go               Command queue
│   ├── tickers.go             Timer system
│   └── ringlog.go             Ring buffer for recent output
├── plugins/
│   └── kallisti/              Map plugin (reference implementation)
├── layout/                    Pane management
├── config/                    XDG config handling
└── protocol/                  MSDP (kept for plugin compat)
```

## Development

```bash
# Build
go build -o dp-client .

# Run
./dp-client --dp --key dp_your_key_here --character Test

# Test against local server
./dp-client --dp --key dp_test --character Test --log-dir /tmp/logs
```

## Relationship to dp-agent

| | dp-client | dp-agent |
|---|-----------|----------|
| **User** | Human player | AI agent |
| **Interface** | bubbletea TUI | Headless CLI |
| **Decision** | Human types commands | FSM + LLM decides |
| **Memory** | Human remembers | Server injects summary |
| **Logging** | JSONL (same format) | JSONL (same format) |
| **Dreaming** | Feeds same pipeline | Feeds same pipeline |

Both clients produce the same JSONL output. The dreaming pipeline doesn't know (or care) which one generated the log. Human sessions and agent sessions are measured the same way.

## Research

This client is part of the AIIDE 2027 paper evaluation. Human sessions provide the baseline for comparing agent behavior. The JSONL logs feed the same dreaming pipeline, enabling direct comparison of human vs agent memory and behavioral persistence.

See [`memory-system.md`](../agents/memory-system.md) for the full memory documentation.
