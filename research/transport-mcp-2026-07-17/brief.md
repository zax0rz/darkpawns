# Research Brief — Human Client Transport + MCP Agent Interface for Dark Pawns

> Date: 2026-07-17 · Mode: deep · Audience: Go MUD developer implementing two interrelated projects

## Question

Draft an implementation plan for two interrelated projects in a production Go MUD server (Dark Pawns, a CircleMUD descendant, ~95K LOC, single Go binary, systemd, Caddy reverse proxy, bare Debian VM):
1. **Human Client Transport** — converge all human players on telnet-over-WebSocket (binary WS bridged to existing `pkg/telnet` handler) with an off-the-shelf xterm web client
2. **MCP Agent Interface** — replace the custom WebSocket JSON protocol with MCP (Model Context Protocol) for AI agent access, starting with an external adapter pattern and evolving to a native in-process MCP server

## Scope Boundaries

**In scope:** MCP 2025-03-26 spec compliance (Streamable HTTP, resources, tools, prompts, subscriptions), WebSocket-to-telnet binary bridging in Go, GMCP protocol integration, external adapter pattern (MCP server wrapping telnet), native in-process MCP server design, session decoupling, Cloudflare tunnel + TCP reachability, xterm.js web client evaluation, preservation constraint, prior art analysis.

**Out of scope:** Docker/containerization, Node.js runtime in production, rewriting the game engine, memory consolidation system redesign.

## Assumptions

- Single Go binary serves HTTP, WS, telnet, API. Caddy handles TLS.
- Existing `pkg/telnet` `handleConn` is the canonical telnet handler — WS bridge reuses it.
- GMCP partially implemented (Char.Vitals, Room.Info, Char.Items).
- `/ws` JSON protocol stays for existing agent integrations.
- MCP is follow-on, not replacement — ship transport project first.
- MCP standard remote transport is Streamable HTTP (not WebSocket).
- Preservation constraint: agents/modern features only where non-player-facing.

## Angles

1. MCP 2025-03-26 spec & Go implementations
2. WebSocket-telnet binary bridging in Go
3. MUD web client landscape 2025-2026
4. External MCP adapter pattern for games
5. Native MCP server design in Go
6. Session persistence & reconnection
7. Cloudflare + TCP reachability
8. MUD-specific MCP considerations (real-time combat, ANSI, multi-agent, parity)
