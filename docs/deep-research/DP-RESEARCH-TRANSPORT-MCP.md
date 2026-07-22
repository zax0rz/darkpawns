# Deep Research Prompt — Human Client Transport + MCP Agent Interface

Use with mimo code `/deep-research` or equivalent research tool.

---

## Research Brief

Draft an implementation plan for two interrelated projects in a production Go MUD (Multi-User Dungeon) server — a 30-year-old CircleMUD port with 10,057 rooms, 1,319 mobs, 95 zones, running at darkpawns.labz0rz.com.

**The codebase:** `github.com/zax0rz/darkpawns` — Go, ~95K LOC, single binary, systemd service, Caddy reverse proxy on a bare Debian VM. No Docker. PostgreSQL for persistence, Redis for cache. Telnet on port 7777, API on 4350, WebSocket on `/ws`.

**The constraint:** the Go port must remain byte-for-byte faithful to the original C CircleMUD on all player-facing output. Agents and modern features are allowed ONLY where they are not player-facing. This is a preservation project, not a rewrite.

---

## Project 1: Human Client Transport — Telnet + WebSocket (Option B)

### Goal
Converge ALL human players on the telnet protocol — raw telnet for native clients, telnet-over-WebSocket for browsers. No Node proxy, no Docker, no nested containers. Everything runs inside the single Go binary.

### Current State
- **Raw telnet:** port 7777, works on LAN, blocked externally (Cloudflare tunnel only carries HTTPS/WSS, not raw TCP)
- **WebSocket:** `/ws` endpoint, JSON-based protocol (custom, not standard MUD protocol). Used by both humans AND agents.
- **Web client:** Hugo-served `client.js` with xterm.js. Currently speaks the JSON protocol, not telnet.
- **dp-client:** Go TUI client (separate repo `zax0rz/dp-client`), speaks WebSocket with JSON protocol.

### What Needs to Happen

**Phase 0 — Reachability:**
- Raw telnet needs a DNS-only host (grey-cloud, no Cloudflare proxy) + firewall port-forward to :7777
- Browser WebSocket rides existing Cloudflare tunnel → Caddy → Go binary
- Decision: Cloudflare Spectrum for TCP passthrough vs. raw DNS + iptables?

**Phase 1 — Go-native `/wstelnet` endpoint:**
- Binary WebSocket (not JSON) bridged to the existing `pkg/telnet` `handleConn` via a `net.Conn` adapter over WS
- No third renderer — reuse the SAME telnet handler that raw TCP uses
- MCCP2 off on the WS path (use permessage-deflate instead)
- Binary WS frames carry raw telnet bytes (IAC sequences, ANSI, everything)
- Test: browser connects to `/wstelnet`, gets the same byte stream as raw telnet

**Phase 2 — Off-the-shelf xterm web client:**
- Evaluate `maldorne/mud-web-client` (Vue 3 + xterm.js, GMCP/MSDP/MXP support) vs. minimal in-house xterm.js page
- Must build to static files (no Node runtime in production)
- Status bar from GMCP (Char.Vitals/Room.Info), not the custom JSON `vars`
- Config via URL params (server, port, character)

**Phase 3 — Cutover:**
- Point `play.darkpawns.labz0rz.com` at new xterm client + `/wstelnet`
- Keep old `client.js` reachable briefly at `/legacy` as rollback
- Retire hand-rolled JSON protocol for HUMANS — keep `/ws` JSON for AGENTS (backs vars/subscribe/memory-bootstrap)

**Phase 4 — Session Decoupling (stretch):**
- Give each player session a stable UUID
- Keep logical session alive for configurable grace period when socket drops
- Buffer output, flush on reconnect (WS or telnet)
- Pre-stages MCP agent work

### Technical Constraints
- No Docker / nested containers (Proxmox LXC, systemd)
- Go binary serves everything — HTTP, WS, telnet, API
- Caddy is the reverse proxy (port 80/443)
- Player-facing output MUST match C CircleMUD byte-for-byte (oracle-gated)
- GMCP already partially implemented (Char.Vitals, Room.Info, Char.Items)

### Key Files
- `pkg/telnet/` — telnet handler, `handleConn`, IAC negotiation
- `pkg/session/` — player session, command dispatch
- `server/` — HTTP/WS server setup
- `cmd/server/main.go` — binary entry point
- `web/client.js` — legacy browser client
- `website/assets/js/client.js` — Hugo-served browser client (different file!)

---

## Project 2: Agent Interface — MCP Migration

### Goal
Replace the custom WebSocket JSON protocol with MCP (Model Context Protocol) for AI agent access. Agents should be able to play the game by connecting to a standard MCP endpoint — no custom client, no SDK, no installation.

### Current State
- **`/ws` JSON protocol:** custom protocol with commands like `vars`, `subscribe`, `char_input`, `char_create`, `memory_bootstrap`, `decision_capture`
- **`/skill.md`:** already exists — tells agents how to connect and play
- **Agent sessions:** tagged as `is_agent`, share the same game world as humans
- **Prior art:** `yuniko-software/minecraft-mcp-server` drives a Minecraft bot via MCP (external adapter pattern)

### What Needs to Happen

**The External Adapter MVP (fastest path):**
```
MCP client → MCP server → (telnet client) → telnet → Dark Pawns
```
- MCP server wraps the telnet connection as MCP tools
- No engine changes required — the game doesn't know it's talking to an agent
- Reuses the `/wstelnet` transport work from Project 1
- Prior art: minecraft-mcp-server does exactly this for Minecraft

**The Native MCP Path (longer term):**
- MCP server runs INSIDE the Go binary
- Commands become MCP tools (`look`, `north`, `attack`, `say`, etc.)
- Game state becomes MCP resources (`mud://room/current`, `mud://player/vitals`, `mud://inventory`)
- Agent identity becomes MCP prompts (system prompt per agent)
- Transport: **Streamable HTTP** (not WebSocket — MCP's standard remote transport)
- SSE (Server-Sent Events) for server→client push (combat events, room changes, messages)

### Design Guardrails (from minecraft-mcp-server analysis)
1. **Push, not polling.** Combat is real-time — a poll-only agent dies. Resource subscriptions + priority skill interrupts.
2. **Dynamic tool surface.** Available skills/spells/items change with game state. Tools should reflect what's ACTUALLY available, not a static list.
3. **Memory.** Agents need persistent memory across sessions (what they've learned, relationships, goals).
4. **Multi-agent.** Multiple agents can play simultaneously. Need session isolation, collision detection, inter-agent communication.

### The Zero-Download Vision
Tell any agent: *"go to darkpawns.labz0rz.com/skill.md and create an account, see you in the game"* — and it just plays.

1. **`/skill.md`** is the bootstrap (already exists) — self-describing: what the game is, the MCP endpoint, the tools, a play primer
2. **`create_account`** is a structured MCP tool (not the nanny prose flow) — `create_account(name, sex, race, class, hometown, password?)` → returns session token
3. **Transport-agnostic `CreateCharacter` core** — factor the character creation logic out of the nanny state machine into a shared `CreateCharacter(spec) → Player` function
4. **Enters the SAME live world** — an MCP session is just another `session.Manager` Session backed by an MCP transport

### Technical Constraints
- MCP's standard transports: **stdio** (local, requires binary) and **Streamable HTTP** (remote, standard-compliant)
- Streamable HTTP carries server→client push via SSE
- Off-the-shelf MCP clients (Claude Desktop, etc.) speak stdio + Streamable HTTP — NOT raw WebSocket
- The `/ws` JSON protocol stays for now (backs vars/subscribe for existing agent integrations)
- MCP is the follow-on, not a replacement — ship Project 1 first

### Key Files
- `pkg/session/` — player session, command dispatch
- `pkg/game/` — game logic, world state
- `cmd/server/main.go` — HTTP/WS server setup
- `docs/research/stateful-MCP-game-agents-research.md` — prior research
- `cmd/dp-goat/` — agent CLI (legacy, may be superseded by MCP)

---

## What I Need From You

1. **Implementation plan** — phased, with clear dependencies between the two projects. Which phases can run in parallel? What blocks what?

2. **Architecture decisions** — for each open question above, recommend an approach with rationale. Consider: Go ecosystem best practices, MCP spec compliance, MUD-specific needs (real-time combat, persistent state, multi-agent), and the preservation constraint.

3. **MUD-specific considerations** — what's different about building an MCP interface for a MUD vs. a web app or a Minecraft server? Think about: real-time state, player-vs-agent parity, the telnet legacy, ANSI escape sequences, GMCP/MSDP/MXP protocol history.

4. **Go implementation details** — package structure, interface design, concurrency model (goroutines per session? event bus?), how to bridge telnet's byte-stream model with MCP's structured JSON-RPC.

5. **Risk register** — what could go wrong? Race conditions between agent and human players? MCP transport reliability? Session state consistency? The preservation constraint creating friction?

6. **Testing strategy** — how do you test an MCP server that wraps a live MUD? Integration tests, contract tests, the oracle differential harness (which already drives both C and Go servers with scripted input).

7. **Prior art analysis** — what have other MUDs/MCP servers done? What works, what doesn't? What should we avoid?

---

## Repository Context

The Go codebase is at `github.com/zax0rz/darkpawns` (private). Key directories:
- `pkg/game/` — core game logic (combat, movement, commands, spells, objects, NPCs)
- `pkg/session/` — player session management, command dispatch, WebSocket handler
- `pkg/telnet/` — raw telnet handler with IAC negotiation
- `pkg/world/` — world loading, zone management
- `pkg/combat/` — combat engine
- `pkg/scripting/` — Lua scripting engine
- `pkg/metrics/` — Prometheus metrics
- `cmd/server/` — binary entry point, HTTP/WS/telnet server
- `cmd/dp-oracle-diff/` — the differential testing harness (drives C + Go with scripted input)
- `web/` — legacy browser client (client.js)
- `website/` — Hugo site (separate from web/)
- `docs/research/` — prior research on agent interfaces, MCP, stateful sessions
- `docs/briefs/` — implementation briefs for the transport and MCP projects
