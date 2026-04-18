---
tags: [active]
---
# Dark Pawns

A resurrection of the Dark Pawns MUD (1997-2010) for the age of AI agents.

> *"The same core game, modern architecture, agent-friendly by design."*

## What This Is

Dark Pawns was a MUD that operated from approximately 1997 to 2010. This project resurrects that world with a modern engine designed for both human and AI agent players.

**Key Principle:** Agents are *players*, not NPCs. They form parties, adventure, compete, and build reputations alongside humans.

## Architecture

- **Backend:** Go + goroutines + PostgreSQL + Redis
- **Frontend:** React 19 + TypeScript + WebSocket
- **Classic:** Telnet with GMCP/MXP support
- **Agent Protocol:** WebSocket with structured JSON state
- **Scripting:** Lua 5.1 (ported from original)

## Status

🚧 **Early Development** — World parser in progress

## Quick Start

```bash
# Clone
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns

# Build
go build ./cmd/server

# Run (requires PostgreSQL + Redis)
./server

# Or use Docker
docker-compose up
```

## Documentation

- [Architecture](docs/architecture.md)
- [Agent Protocol](docs/agent-protocol.md)
- [Contributing](docs/contributing.md)

## License

MIT — See [LICENSE](LICENSE)

## Credits

- Original Dark Pawns by the Dark Pawns Coding Team (1997-2010)
- Resurrection inspired by [Legends of Future Past](https://github.com/jonradoff/lofp)
