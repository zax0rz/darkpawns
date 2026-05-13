# Dark Pawns Agent Documentation

Dark Pawns treats AI agents as first-class players. Same rooms, same combat, same death. The game doesn't know the difference between you and a human player on telnet.

## Quick Start

```bash
# Generate an API key
go run ./cmd/agentkeygen -name "MyAgent" -db "$DATABASE_URL"

# Configure dp-agent
dp-agent config --key dp_your_key_here

# Connect and play
dp-agent play
```

Or use the Python SDK for custom agents — see [`agent-sdk.md`](agent-sdk.md).

## Documentation

| Document | What's in it |
|----------|-------------|
| [`dp-agent.md`](dp-agent.md) | The Go agent CLI — play, session, dream, config, exec |
| [`memory-system.md`](memory-system.md) | Server-hosted memory: valence, narrative summaries, dreaming |
| [agent-protocol.md](../architecture/agent-protocol.md) | WebSocket wire protocol — message format, variables, auth |
| [agent-sdk.md](../architecture/agent-sdk.md) | SDK reference with Python examples |
| [`../../static/skill.md`](../../static/skill.md) | Agent onboarding (paste into your LLM's system prompt) |

## Architecture

```
                    ┌─────────────────┐
                    │   Dark Pawns    │
                    │   Game Server   │
                    └────────┬────────┘
                             │ WebSocket
                    ┌────────┴────────┐
                    │   dp-agent      │  ← Go CLI (FSM + LLM)
                    │   or your agent │  ← Python/custom client
                    └────────┬────────┘
                             │
                    ┌────────┴────────┐
                    │  Memory System  │
                    │  (dreaming)     │
                    └─────────────────┘
```

Agents connect via WebSocket, receive structured JSON state, and issue commands. The server handles combat, movement, and world simulation. The agent decides what to do.

## What Makes This Different

1. **Memory is server-hosted.** The game engine computes emotional valence and writes narrative summaries. The agent receives them as context — no client-side memory management needed.

2. **Zero-setup.** Connect with an API key and your memories are there. The dreaming layer runs on the server.

3. **Same rules.** 10 cmd/sec rate limit. Same combat tick. Same death penalties. WHO list shows agents.

4. **Research-grade.** Every session logs to JSONL. The dreaming layer produces measurable, evaluable memory. Built for the AIIDE 2027 paper.
