# Implementation Plan: Human Client Transport + MCP Agent Interface for Dark Pawns

> Generated 2026-07-17 · depth: deep · 87 sources · workspace: research/transport-mcp-2026-07-17/

## Executive summary

- **Use `coder/websocket` (not gorilla) for the `/wstelnet` endpoint** — it has a first-class `NetConn()` adapter that wraps `*websocket.Conn` as `net.Conn`, purpose-built for protocol tunneling. gorilla/websocket's `NetConn()` returns the raw TCP socket and is useless for this pattern. [1][2][3]
- **`handleConn(rawConn net.Conn, ...)` is immediately compatible** with the `coder/websocket` `NetConn()` adapter — pass `websocket.NetConn(ctx, wsConn, websocket.MessageBinary)` as the first argument. No refactoring of the telnet handler internals needed. [2]
- **mark3labs/mcp-go (8.9k stars, v0.56.0)** is the Go MCP library to use — it implements MCP spec 2025-11-25, its `StreamableHTTPServer` implements `http.Handler` (mounts on any `net/http` mux), and it has built-in per-session tool customization via `AddSessionTool`/`ToolFilterFunc`. [4][5][6]
- **Cloudflare Spectrum requires Enterprise ($1,000+/mo)** for generic TCP passthrough. The practical path for raw telnet reachability is DNS-only A record + iptables port-forward to :7777, keeping the Cloudflare tunnel for HTTPS/WSS. [7][8]
- **maldorne/mud-web-client v4** is the only actively maintained modern xterm.js MUD web client with GMCP/MSDP/MXP support — Vue 3, TypeScript, builds to static files via Vite, all config via URL params. [9][10]
- **External MCP adapter (MCP server wrapping telnet) is the fastest path to agents** — requires zero game server changes. minecraft-mcp-server (662 stars) proves this pattern works. Native in-process MCP is the follow-on. [11][12][13]
- **MCP resource subscriptions (`resources/subscribe` + `notifications/resources/updated`) are the native push mechanism** for combat events — MCP tools are request-response only, so combat rounds must flow through SSE notifications, not tool results. [14][15]
- **Session decoupling is the critical prerequisite** for both projects — Dark Pawns' current `Session` struct is tightly coupled to `*websocket.Conn` with no UUID identity. MCP's `Mcp-Session-Id` header + SSE `Last-Event-ID` resumability defines the target pattern. [16][17]
- **Ship Project 1 (transport) first, Project 2 (MCP) second** — the `/wstelnet` endpoint is a prerequisite for both the xterm web client AND the external MCP adapter. The two projects share the WS-telnet bridge as common infrastructure. [speculative]

## Background & scope

Dark Pawns is a Go MUD server ported from C CircleMUD/Merc 2.2, running ~95K LOC as a single binary on a bare Debian VM with Caddy reverse proxy. Two interrelated projects are planned: (1) converging all human players on telnet-over-WebSocket with an off-the-shelf xterm web client, and (2) replacing the custom WebSocket JSON protocol with MCP for AI agent access. The preservation constraint requires byte-for-byte fidelity to the original C CircleMUD on all player-facing output — agents and modern features are allowed only where non-player-facing.

## Project 1: Human Client Transport

### Phase 0 — Reachability

**Recommendation: DNS-only A record + iptables** for raw telnet, riding the existing Cloudflare tunnel for WSS.

Cloudflare Spectrum is the only Cloudflare product that proxies raw TCP, but it requires an Enterprise plan with custom pricing (estimated $1,000+/mo). Pro/Business plans only support Minecraft, SSH, and RDP as hardcoded exceptions — no generic TCP passthrough. [7][8] Cloudflare Tunnel (`cloudflared`) can carry TCP but requires end users to install the WARP client, making it unsuitable for public MUD access. [8]

**The practical setup:**
1. Create a DNS-only (grey-cloud) A record for `telnet.darkpawns.labz0rz.com` pointing to the VM's real IP
2. Open port 7777 in iptables/firewall
3. Keep the existing Cloudflare tunnel (orange-cloud) for `play.darkpawns.labz0rz.com` → Caddy → Go binary (HTTPS/WSS)
4. Native telnet clients connect to `telnet.darkpawns.labz0rz.com:7777` directly
5. Browser clients connect via `wss://play.darkpawns.labz0rz.com/wstelnet` through the tunnel

This avoids Spectrum's cost while giving both paths external reachability. The tradeoff is exposing the origin IP on the DNS-only record — acceptable for a MUD.

### Phase 1 — Go-native `/wstelnet` endpoint

**Library choice: `coder/websocket` v1.8.15** (not gorilla/websocket).

The critical finding: `coder/websocket` provides a first-class `NetConn()` adapter that converts `*websocket.Conn` into `net.Conn`, purpose-built for protocol tunneling. Each `Write` becomes a binary WS message; each `Read` receives one. [1] gorilla/websocket (which Dark Pawns currently uses) does NOT provide this — its `NetConn()` returns the raw underlying TCP connection, and direct I/O on it corrupts the WebSocket state. [2] The `coder/websocket` `NetConn` was specifically created because gorilla never addressed issues #282 and #441 requesting this pattern. [3]

**The bridge is a one-liner:**
```go
// In the /wstelnet handler:
wsConn, _ := websocket.Accept(w, r, &websocket.AcceptOptions{
    CompressionMode: websocket.CompressionContextTakeover,
})
rawConn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)
handleConn(rawConn, manager, banLevel) // reuse existing telnet handler
```

Dark Pawns' `handleConn` signature is `handleConn(rawConn net.Conn, manager *session.Manager, banLevel int)` — it accepts `net.Conn` directly, making it immediately compatible with no refactoring. [2]

**MCCP2 suppression:** The telnet handler offers `IAC WILL COMPRESS2` to all connections during initial negotiation. WS clients use permessage-deflate (RFC 7692), not MCCP2. A boolean flag or interface type assertion on the `telnetConn` can skip the MCCP2 offer for WS connections. `coder/websocket` supports full permessage-deflate with context takeover; gorilla only supports no-context-takeover. [3][2]

**Deadline behavior caveat:** `coder/websocket`'s `NetConn` closes the connection when a deadline fires (unlike typical `net.Conn` which just interrupts the goroutine). For long-lived MUD sessions, this may require a custom adapter with softer deadline semantics, or accepting that WS connections are re-established on timeout. [2]

**Migration path:** Dark Pawns currently uses `gorilla/websocket` v1.5.3. The `coder/websocket` dependency can be added alongside gorilla (different import path) — the `/wstelnet` handler uses `coder/websocket`, while the existing `/ws` JSON handler continues using gorilla until migrated.

### Phase 2 — xterm web client

**Recommendation: Adopt maldorne/mud-web-client v4** (not build in-house).

maldorne/mud-web-client v4 (2026) is a full Vue 3 + TypeScript + xterm.js v6 rewrite that builds to static files via Vite. [9] It supports GMCP, MSDP, and MXP protocols with a byte-level IAC parser cleanly factored into `useTelnetParser.ts`. [10] All configuration is via URL query parameters (`?proxy=wss://...&host=...&port=...`), and it has responsive layout from 800x600 iframe to fullscreen. [9]

The companion `maldorne/mud-web-proxy` is a WebSocket-to-Telnet proxy — but Dark Pawns won't need it because the `/wstelnet` endpoint IS the proxy, embedded in the Go binary. The maldorne client connects to a WebSocket endpoint that carries raw telnet bytes, which is exactly what `/wstelnet` provides.

**Status bar from GMCP:** The maldorne client parses GMCP events (`Char.Vitals`, `Room.Info`) but doesn't include a built-in status bar component. A thin Vue component wrapping the GMCP events from `useTelnetParser.ts` would provide the status bar. This is the only customization needed.

**Build pipeline:** `npm run build` produces a `/dist` directory. Hugo can serve these as static assets, or they can be placed directly in the website's static directory. No Node runtime in production.

**Alternative considered:** Building a minimal in-house xterm.js client (~200 lines). The telnet parser is the hardest part and is already factored into maldorne's `useTelnetParser.ts` — reusable even if the rest of the client is custom. However, adopting maldorne gives GMCP/MSDP/MXP support for free, which a minimal client would need to implement from scratch.

### Phase 3 — Cutover

1. Deploy `/wstelnet` endpoint and verify byte-for-byte parity with raw telnet using the existing oracle differential harness
2. Build maldorne client with `?proxy=wss://play.darkpawns.labz0rz.com/wstelnet`
3. Point `play.darkpawns.labz0rz.com` at the new xterm client + `/wstelnet`
4. Keep old `client.js` reachable at `/legacy` briefly as rollback
5. Retire hand-rolled JSON protocol for humans — keep `/ws` JSON for agents

### Phase 4 — Session decoupling (stretch)

**Current state:** Dark Pawns' `Session` struct is keyed by `playerName` string with no UUID. The session lifecycle is tightly coupled to `*websocket.Conn` — `UnregisterAndClose` immediately deletes the session from the map and closes the connection. [16]

**Target state:** Each player session gets a stable UUID (UUIDv7 via `google/uuid` v1.6.0, 113K+ importers, time-ordered and sortable). [17] The logical session stays alive for a configurable grace period when the socket drops, buffering output in a ring buffer. On reconnect (WS or telnet), the player re-authenticates with their UUID and receives buffered output.

**MCP alignment:** MCP's Streamable HTTP already defines the exact pattern needed — server-assigned `Mcp-Session-Id` header + SSE event IDs with `Last-Event-ID` replay for resumability. [14][16] Dark Pawns' existing `ServerMessage.Seq` field (uint64) could serve as the cursor for resumable event streams. [16]

**Evennia's Portal/Server split** is the reference architecture: network sessions are separate from game logic sessions, allowing sessions to survive server restarts. [17] Dark Pawns doesn't need the full AMP-based split, but the conceptual separation of "transport session" from "game session" is the right model.

## Project 2: MCP Agent Interface

### The External Adapter MVP

**Architecture:** `MCP client → MCP server (Go binary) → telnet client → telnet → Dark Pawns`

The fastest path is an external adapter: a standalone Go MCP server that wraps a telnet client connection. The game server doesn't know it's talking to an agent. This is exactly the pattern used by minecraft-mcp-server (662 stars) for Minecraft. [11][12]

**However, there's a better option for Dark Pawns:** Instead of a separate process, the external adapter can run INSIDE the Go binary as a goroutine that connects to its own telnet port (`localhost:7777`). This avoids the separate-process overhead while still keeping the game server unaware of MCP.

**What the adapter exposes:**
- **Tools:** `look`, `north`, `south`, `east`, `west`, `up`, `down`, `attack`, `cast`, `say`, `get`, `drop`, `wear`, `wield`, `flee`, `inventory`, `score`, `who`, `create_account` — all imperative commands, same as the minecraft-mcp-server pattern. [12]
- **Resources:** `mud://session/output` (streaming text output), `mud://session/gmcp` (structured GMCP data if the telnet connection negotiates GMCP)
- **Prompts:** System prompt describing the game, available commands, and the agent's character

**Limitations of the external adapter:**
- No structured resource access — the adapter can only observe what a normal player observes (text output)
- No proactive notifications — the adapter can't push combat events to the MCP client; it can only respond to tool calls
- Text parsing required — the adapter must parse ANSI-stripped text to extract game state

**Mitigation:** The adapter can negotiate GMCP on its telnet connection and surface GMCP data as structured MCP resources. This gives agents the same structured data that GUI clients get, without modifying the game server.

### The Native MCP Path (longer term)

**Library: mark3labs/mcp-go** (8.9k stars, v0.56.0, implements MCP spec 2025-11-25). [4][5]

**Integration is trivial:** `StreamableHTTPServer` implements `http.Handler`, so it mounts directly on the existing `net/http` mux alongside the MUD server. [6]

```go
mux := http.NewServeMux()
mux.Handle("/ws", existingWSHandler)          // existing JSON protocol for agents
mux.Handle("/wstelnet", newWSTelnetHandler)   // new binary WS for humans
mux.Handle("/mcp", mcpStreamableHandler)      // new MCP endpoint
mux.HandleFunc("/api", existingAPIHandler)    // existing API
```

**Tool mapping:** Each MUD command becomes an MCP tool with JSON schema input validation. mcp-go's declarative schema builders make this straightforward. [5]

**Dynamic tool surface:** mcp-go supports three mechanisms for per-session tool customization — `AddSessionTool`/`DeleteSessionTools` for adding/removing tools at runtime, `WithToolFilter` for context-based filtering at list-time, and `SessionWithTools` interface for full per-session customization. [5] This maps directly to "the `open_chest` tool is only available when the player is in a room with a chest." [13]

**Resource mapping:**
- `mud://room/current` — current room state (description, exits, items, mobs)
- `mud://player/vitals` — HP, mana, movement, level, XP
- `mud://player/inventory` — carried items
- `mud://player/equipment` — worn/wielded items
- `mud://world/time` — in-game time and weather

**Combat push via resource subscriptions:** MCP tools are request-response only — combat events cannot be pushed as tool results. [14] Instead, agents subscribe to `mud://player/combat` and receive `notifications/resources/updated` when combat state changes. The agent then calls `resources/read` to get the updated state. [15] This is the MCP equivalent of MSDP's REPORT mechanism. [14]

**SSE for server→client push:** mcp-go supports `SendNotificationToSpecificClient` for pushing events to individual sessions. [6] The GET SSE stream on the MCP endpoint carries these notifications.

### The Zero-Download Vision

1. **`/skill.md`** — already exists, tells agents how to connect and play
2. **`create_account` tool** — structured MCP tool: `create_account(name, sex, race, class, hometown, password?)` → returns session token. Factored out of the nanny state machine into a shared `CreateCharacter(spec) → Player` function.
3. **Transport-agnostic character creation** — the `CreateCharacter` core is independent of whether the caller is a telnet nanny or an MCP tool handler
4. **Same live world** — an MCP session is just another `session.Manager` Session backed by an MCP transport

## Phased implementation plan with dependencies

```
Phase 0: Reachability (DNS-only + iptables)
    └── no dependencies, can start immediately

Phase 1: /wstelnet endpoint (coder/websocket + NetConn adapter)
    └── depends on: nothing (but benefits from Phase 0 for external testing)
    └── blocks: Phase 2, Phase 3, External MCP adapter

Phase 2: xterm web client (adopt maldorne/mud-web-client v4)
    └── depends on: Phase 1 (/wstelnet must exist)
    └── blocks: Phase 3

Phase 3: Cutover (point play.darkpawns.labz0rz.com at new client)
    └── depends on: Phase 1 + Phase 2
    └── blocks: nothing

Phase 4: Session decoupling (UUID sessions, grace period, ring buffer)
    └── depends on: nothing (can run in parallel with Phases 1-3)
    └── blocks: Native MCP server (benefits from UUID sessions)

Phase 5: External MCP adapter (standalone MCP server wrapping telnet)
    └── depends on: Phase 1 (/wstelnet or direct telnet)
    └── blocks: nothing (can ship independently)

Phase 6: Native MCP server (in-process, mark3labs/mcp-go)
    └── depends on: Phase 4 (UUID sessions) + Phase 5 (learnings from external adapter)
    └── blocks: nothing

Phase 7: Cutover to native MCP (deprecate external adapter)
    └── depends on: Phase 6
```

**Parallel tracks:**
- Track A: Phases 0 → 1 → 2 → 3 (human client transport)
- Track B: Phase 4 (session decoupling — can start in parallel with Track A)
- Track C: Phase 5 (external MCP adapter — can start after Phase 1)
- Track D: Phases 6 → 7 (native MCP — starts after Phase 4 + Phase 5)

## Architecture decisions

| Decision | Recommendation | Rationale |
|----------|---------------|-----------|
| WS library | `coder/websocket` v1.8.15 | First-class `NetConn()` adapter for protocol tunneling. gorilla's `NetConn()` returns raw TCP. [1][2] |
| MCP Go library | `mark3labs/mcp-go` v0.56.0 | 8.9k stars, implements spec 2025-11-25, `http.Handler` integration, per-session tools. [4][5] |
| MCP transport | Streamable HTTP | Only viable transport for a telnet MUD — stdio is incompatible with the existing binary stream. [14] |
| Web client | maldorne/mud-web-client v4 | Only modern xterm.js MUD client with GMCP/MSDP/MXP. Builds to static files. [9][10] |
| Raw telnet reachability | DNS-only A record + iptables | Spectrum requires Enterprise ($1000+/mo). DNS-only is free and simple. [7][8] |
| Session ID format | UUIDv7 (`google/uuid` v1.6.0) | Time-ordered, sortable, globally unique, 113K+ importers. [17] |
| Compression on WS path | permessage-deflate (RFC 7692) | `coder/websocket` supports full context takeover. Suppress MCCP2 on WS path. [2][3] |
| MCP agent entry point | External adapter first, native second | Zero game server changes for MVP. minecraft-mcp-server proves the pattern. [11][12] |

## Go implementation details

### Package structure

```
pkg/
  telnet/          # existing — add MCCP2 suppression flag for WS connections
  session/         # existing — add UUID identity, decouple from websocket.Conn
  mcp/             # NEW — MCP server implementation
    server.go      # mark3labs/mcp-go setup, StreamableHTTPServer
    tools.go       # MUD command → MCP tool mapping
    resources.go   # Game state → MCP resource mapping
    prompts.go     # Agent identity → MCP prompt mapping
    adapter.go     # External adapter: MCP server wrapping telnet client
  transport/       # NEW — transport abstraction layer
    conn.go        # net.Conn interface for both telnet and WS connections
    wstelnet.go    # /wstelnet handler using coder/websocket NetConn()
cmd/
  server/
    main.go        # register /wstelnet, /mcp, /ws, /api routes on single mux
```

### Interface design

```go
// Transport-agnostic session identity
type SessionID struct {
    UUID    uuid.UUID  // UUIDv7, time-ordered
    Created time.Time
}

// Session decoupled from transport
type GameSession struct {
    ID          SessionID
    PlayerName  string
    Transport   net.Conn          // telnet, WS, or MCP-backed conn
    OutputRing  *RingBuffer       // buffered output for reconnection
    LastSeq     uint64            // cursor for resumable event streams
    IsAgent     bool
    GracePeriod time.Duration     // link-dead timeout
}

// MCP tool handler signature
type MUDToolHandler func(ctx context.Context, session *GameSession, args map[string]any) (*mcp.CallToolResult, error)
```

### Concurrency model

- **Goroutine per session** (existing pattern) — each telnet/WS connection gets its own goroutine running `handleConn`
- **MCP sessions** use the same pattern: each MCP client's SSE stream gets a goroutine that pushes events from the game's event bus
- **Event bus:** the game engine publishes events (combat rounds, room changes, channel messages) to a per-session channel. Both telnet/WS handlers and MCP SSE streams drain from this channel.
- **No shared mutable state** between sessions — the existing `session.Manager` with its mutex-protected map is the right pattern

### Bridging telnet's byte-stream with MCP's structured JSON-RPC

The fundamental tension: telnet is a byte-stream with IAC negotiation, ANSI escapes, and interleaved GMCP subnegotiations. MCP is structured JSON-RPC over HTTP.

**For the external adapter:** The adapter negotiates GMCP on its telnet connection and surfaces GMCP data as MCP resources. Text output is streamed as-is (with ANSI stripped for agent consumption). The adapter maintains a telnet parser that extracts structured data from the byte stream.

**For the native MCP server:** The game engine emits structured events (combat results, room state changes) that are directly mapped to MCP resources and notifications. No byte-stream parsing needed — the game engine's internal event model IS the MCP data model. The GMCP schemas (`Char.Vitals`, `Room.Info`) are the bridge — they already define JSON structures for game state.

## Risk register

| Risk | Severity | Mitigation |
|------|----------|------------|
| **MCCP2/WS compression conflict** | Medium | Suppress MCCP2 offer on WS path via boolean flag in `handleConn`. Test with permessage-deflate context takeover. [2][3] |
| **coder/websocket deadline closes connection** | Medium | For long-lived MUD sessions, implement a custom net.Conn wrapper with softer deadline semantics, or accept WS reconnection on timeout. [2] |
| **gorilla→coder/websocket migration breaks existing /ws** | Low | Add `coder/websocket` as new dependency alongside gorilla. Migrate `/ws` to coder/websocket later, or keep both. |
| **maldorne client needs customization** | Low | Status bar from GMCP is the only gap. The `useTelnetParser.ts` composable already extracts GMCP events — just need a Vue component to display them. [9][10] |
| **External MCP adapter can't push combat events** | High | Mitigate by having the adapter subscribe to its own telnet connection's output stream and buffer events. The agent polls via `resources/read`. For true push, native MCP is needed. [11][14] |
| **MCP resource subscription latency for combat** | Medium | MCP SSE streams have inherent latency (HTTP round-trip). For real-time combat, the agent may miss 1-2 rounds. Mitigate with priority interrupts and defensive combat tools (`flee`, `heal`). [14][15] |
| **Multi-agent race conditions** | Medium | The existing `session.Manager` mutex protects session state. Game engine commands are already serialized through the command queue. MCP tool calls are just additional commands in the same queue. [speculative] |
| **Preservation constraint violation** | High | The oracle differential harness already drives both C and Go servers with scripted input. Extend it to verify that MCP agent sessions produce identical game state as human sessions. All MCP tools must go through the same command dispatch as telnet commands. [speculative] |
| **Session decoupling complexity** | Medium | Start with a simple grace period (keep session alive for N seconds after disconnect). Full ring-buffer resumability is a stretch goal. The MCP `Mcp-Session-Id` + `Last-Event-ID` pattern is well-defined. [14][16] |
| **DNS-only record exposes origin IP** | Low | Acceptable for a MUD. The real IP is already known to anyone who connects via telnet. No sensitive services on the same IP. [speculative] |

## Testing strategy

### Unit tests
- `NetConn` adapter: verify binary WS frames carry raw telnet bytes (IAC sequences, ANSI)
- MCCP2 suppression: verify WS connections don't receive `IAC WILL COMPRESS2`
- MCP tool schemas: verify each tool's JSON schema matches the game command's expected input
- Session UUID: verify UUIDv7 generation, time-ordering, uniqueness

### Integration tests
- **Oracle differential harness:** extend the existing `cmd/dp-oracle-diff/` to drive both telnet and `/wstelnet` with the same scripted input, comparing output byte-for-byte
- **MCP tool contract tests:** for each MCP tool, verify that calling the tool produces the same game state change as typing the equivalent telnet command
- **Session reconnection:** connect via WS, disconnect, reconnect with same UUID, verify buffered output is flushed

### End-to-end tests
- **Human path:** browser → `wss://play.darkpawns.labz0rz.com/wstelnet` → xterm.js → verify GMCP status bar updates
- **Agent path:** MCP client → `https://play.darkpawns.labz0rz.com/mcp` → create_account → look → move → verify game state
- **Mixed session:** human and agent in same room, verify both see each other's actions

### Performance tests
- **Concurrent sessions:** 50 simultaneous WS connections (25 human, 25 agent), verify no race conditions
- **Combat latency:** agent subscribes to combat resource, measure time from combat event to SSE notification delivery
- **Session reconnection under load:** disconnect and reconnect 10 sessions simultaneously, verify output buffer integrity

## Prior art analysis

| Project | Pattern | What works | What to avoid |
|---------|---------|------------|---------------|
| **minecraft-mcp-server** (662 stars) | External adapter wrapping Mineflayer | Proves the pattern. Automatic reconnection on tool calls. ~20 imperative tools. [11][12] | All tools are low-level primitives — agents must compose multi-step plans. No structured resource access. |
| **FundamentalLabs/minecraft-mcp** (74 stars) | External adapter with higher-level skills | 30 "verified skills" that encapsulate multi-step logic (mineResource, craftItems). Multi-bot support. [12] | Higher-level abstractions may not map cleanly to MUD commands. |
| **Nexlen/mud-mcp** (32 stars) | Native MCP server | Dynamic tool availability (`open_chest` only when chest present), MCP sampling for bidirectional AI, resource URIs (`mud://room/current`). [13] | Small project, untested at scale. Sampling may not be needed for MVP. |
| **EllyMUD** (6 stars) | Hybrid: native MCP + virtual sessions | "Virtual session" pattern bridges external adapter and native integration. API-key auth. [13] | Small project. MCP implementation details not fully readable. |
| **Evennia** (Python MUD framework) | Portal/Server split | Network sessions separate from game sessions via AMP protocol. Sessions survive server restarts. [17] | Python-specific, not directly applicable to Go. The conceptual model is right. |
| **Aardwolf MUD** | GMCP over telnet | Battle-tested JSON schemas for Room.Info, Char.Vitals. Push-based updates via telnet subnegotiation. [14] | GMCP interleaves with text stream — MCP's HTTP transport is architecturally cleaner. |

## Open questions

1. **permessage-deflate context takeover safety** — is context takeover safe for MUD traffic patterns (small frequent messages vs. large bursts)? Needs benchmarking. [3]
2. **MCP SSE performance under combat load** — how does mcp-go handle 10+ concurrent SSE streams with high-frequency notifications (combat rounds every 1-2 seconds)? Goroutine-per-stream or fan-out channel? [6]
3. **GMCP → MCP resource mapping** — should GMCP packages (`Char.Vitals`, `Room.Info`) map 1:1 to MCP resource URIs, or should MCP define its own resource schema? The GMCP schemas are battle-tested but may not be optimal for MCP's resource model. [14]
4. **Agent identity and authentication** — EllyMUD uses 256-bit API keys. MCP's `Mcp-Session-Id` is server-assigned. How does `create_account` issue credentials that the agent uses to reconnect? [13][16]
5. **Grace period length** — CircleMUD used 15 minutes for link-dead, Merc used ~3 minutes. What makes sense for Dark Pawns with AI agent sessions that may intentionally disconnect/reconnect? [16]

## Sources

[1] coder/websocket NetConn() documentation — https://pkg.go.dev/github.com/coder/websocket#NetConn (published 2026-06-15, accessed 2026-07-17)

[2] gorilla/websocket NetConn() documentation — https://pkg.go.dev/github.com/gorilla/websocket@v1.5.3#Conn.NetConn (published 2024-06-14, accessed 2026-07-17)

[3] coder/websocket issue #100 (NetConn motivation) — https://github.com/coder/websocket/issues/100 (published 2019-06-24, accessed 2026-07-17)

[4] mark3labs/mcp-go — https://github.com/mark3labs/mcp-go (published 2026-07-09, accessed 2026-07-17)

[5] mark3labs/mcp-go session tools documentation — https://github.com/mark3labs/mcp-go (published 2026-07-09, accessed 2026-07-17)

[6] mcp-go.dev HTTP transport docs — https://mcp-go.dev/transports/http (accessed 2026-07-17)

[7] Cloudflare Spectrum protocols per plan — https://developers.cloudflare.com/spectrum/protocols-per-plan/ (published 2026-04-16, accessed 2026-07-17)

[8] Cloudflare Tunnel for private networks — https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/private-net/cloudflared/ (published 2026-04-17, accessed 2026-07-17)

[9] maldorne/mud-web-client — https://github.com/maldorne/mud-web-client (published 2026, accessed 2026-07-17)

[10] maldorne useTelnetParser.ts — https://raw.githubusercontent.com/maldorne/mud-web-client/master/src/composables/useTelnetParser.ts (published 2026, accessed 2026-07-17)

[11] yuniko-software/minecraft-mcp-server — https://github.com/yuniko-software/minecraft-mcp-server (published 2026-02-26, accessed 2026-07-17)

[12] FundamentalLabs/minecraft-mcp — https://github.com/FundamentalLabs/minecraft-mcp (accessed 2026-07-17)

[13] Nexlen/mud-mcp — https://github.com/Nexlen/mud-mcp (accessed 2026-07-17)

[14] MCP spec 2025-03-26 — resources — https://modelcontextprotocol.io/specification/2025-03-26/server/resources (published 2025-03-26, accessed 2026-07-17)

[15] MCP spec 2025-03-26 — tools — https://modelcontextprotocol.io/specification/2025-03-26/server/tools (published 2025-03-26, accessed 2026-07-17)

[16] MCP spec 2025-03-26 — transports — https://modelcontextprotocol.io/specification/2025-03-26/basic/transports (published 2025-03-26, accessed 2026-07-17)

[17] Evennia session handler — https://github.com/evennia/evennia/blob/main/evennia/server/sessionhandler.py (accessed 2026-07-17)

[18] google/uuid v1.6.0 — https://pkg.go.dev/github.com/google/uuid@v1.6.0 (published 2024-01-23, accessed 2026-07-17)

[19] GMCP protocol documentation — https://www.gammon.com.au/gmcp (published 2015-04-08, accessed 2026-07-17)

[20] MSDP protocol specification — https://tintin.sourceforge.net/protocols/msdp/ (accessed 2026-07-17)

[21] ellyseum/ellymud — https://github.com/ellyseum/ellymud (published 2026-05-04, accessed 2026-07-17)

[22] metoro-io/mcp-golang — https://github.com/metoro-io/mcp-golang (published 2026-02-25, accessed 2026-07-17)

[23] modelcontextprotocol/go-sdk — https://github.com/modelcontextprotocol/go-sdk (published 2026-05-22, accessed 2026-07-17)

[24] maldorne/mud-web-proxy — https://github.com/maldorne/mud-web-proxy (published 2026, accessed 2026-07-17)

[25] coder/websocket — https://github.com/coder/websocket (published 2026-06-15, accessed 2026-07-17)

[26] WHATWG SSE specification — https://html.spec.whatwg.org/multipage/server-sent-events.html (living standard, accessed 2026-07-17)

[27] Dark Pawns prior research: stateful MCP game agents — docs/research/stateful-MCP-game-agents-research.md (50 citations, accessed 2026-07-17)

[28] Dark Pawns prior research: agent-friendly interfaces — docs/research/foundations/agent-friendly-interfaces.md (29 citations, accessed 2026-07-17)

[29] Dark Pawns pkg/telnet/listener.go — handleConn signature (project source, accessed 2026-07-17)

[30] Dark Pawns pkg/session/session_manager.go — UnregisterAndClose (project source, accessed 2026-07-17)

[31] Dark Pawns pkg/session/protocol.go — ServerMessage.Seq field (project source, accessed 2026-07-17)

[32] xterm.js v6.0.0 — https://github.com/xtermjs/xterm.js (published 2025-12-22, accessed 2026-07-17)
