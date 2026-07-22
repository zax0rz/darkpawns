# Linear Issue Reconciliation — Transport & MCP Report (2026-07-17)

> Source: `research/transport-mcp-2026-07-17/REPORT.md` + findings F1–F8
> Generated: 2026-07-17 · target projects: Human Client Transport, Agent Interface — MCP

---

## A. ENRICH EXISTING ISSUES

### DP-1140 — Expose native telnet :7777 externally (DNS-only host + port-forward)

- **Add:** "Cloudflare Spectrum requires an Enterprise plan with custom pricing (estimated $1,000+/mo) for generic TCP passthrough — Pro/Business plans only support Minecraft, SSH, and RDP as hardcoded exceptions." [7]
- **Add:** "Cloudflare Tunnel (`cloudflared`) can carry TCP but requires end users to install the WARP client, making it unsuitable for public MUD access." [8]
- **Add:** "Create a DNS-only (grey-cloud) A record for `telnet.darkpawns.labz0rz.com` pointing to the VM's real IP; keep the existing Cloudflare tunnel (orange-cloud) for `play.darkpawns.labz0rz.com` → Caddy → Go binary (HTTPS/WSS)." [7][8]

### DP-1142 — Design browser WSS route + public hostnames (Cloudflare tunnel → Caddy)

- **Add:** "The WSS endpoint is `wss://play.darkpawns.labz0rz.com/wstelnet` riding the existing Cloudflare tunnel (orange-cloud), while raw telnet uses a separate DNS-only (grey-cloud) A record on `telnet.darkpawns.labz0rz.com:7777`." [7][8]
- **Add:** "Caddy has no capability to proxy raw TCP connections — it is an HTTP/HTTPS/gRPC/WebSocket reverse proxy only and cannot serve as a telnet relay." [F7:7]

### DP-1143 — Implement /wstelnet: binary WS ↔ telnet session bridge

- **Add:** "Use `coder/websocket` v1.8.15, not gorilla/websocket. gorilla's `NetConn()` returns the raw underlying TCP connection — direct I/O on it corrupts the WebSocket state. `coder/websocket`'s `NetConn()` wraps `*websocket.Conn` as `net.Conn` with each `Write` becoming a binary WS message and each `Read` receiving one, purpose-built for protocol tunneling." [1][2][3]
- **Add:** "The bridge is a one-liner: `rawConn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)`. `handleConn(rawConn net.Conn, manager *session.Manager, banLevel int)` is immediately compatible — zero refactoring of the telnet handler internals." [2][F2:4]
- **Add:** "Use `AcceptOptions{CompressionMode: websocket.CompressionContextTakeover}` for full RFC 7692 permessage-deflate with context takeover." [1]
- **Add:** "Caveat: `coder/websocket`'s `NetConn` closes the connection when a deadline fires — different from typical `net.Conn` which only interrupts the goroutine. For long-lived MUD sessions this may require a custom adapter with softer deadline semantics, or accepting that WS connections are re-established on timeout." [2][F2:5]
- **Add:** "Use `MessageBinary` frames, not `TextMessage` — telnet IAC sequences (0xFF) and ANSI escapes are arbitrary byte sequences, not valid UTF-8 text." [F2:9]
- **Add:** "Add `coder/websocket` as a new dependency alongside the existing `gorilla/websocket` v1.5.3 (different import paths). The `/wstelnet` handler uses `coder/websocket`; the existing `/ws` JSON handler continues using gorilla until later migration." [2][3]

### DP-1144 — Disable MCCP2 on the WS path; rely on permessage-deflate

- **Add:** "`coder/websocket` supports full RFC 7692 permessage-deflate with both context takeover and no-context-takeover modes, while gorilla/websocket only supports no-context-takeover." [3][F2:6]
- **Add:** "MCCP2 is `IAC WILL COMPRESS2` (telnet option) offered during initial negotiation using compress/zlib. Suppress via a boolean flag or interface type assertion on the `telnetConn` to skip the MCCP2 offer for WS-sourced connections." [F2:8]
- **Add:** "Permessage-deflate context takeover safety for MUD traffic patterns (small frequent messages vs. large bursts) is unverified — needs benchmarking." [3]

### DP-1145 — e2e tests: telnet-over-WS login + char-creation parity

- **Add:** "Extend the existing oracle differential harness (`cmd/dp-oracle-diff/`) to drive both raw telnet and `/wstelnet` with the same scripted input, comparing output byte-for-byte." [REPORT: testing strategy]
- **Add:** "The oracle harness already drives both C and Go servers with scripted input; extend it to verify that WS-sourced sessions produce identical output to native telnet sessions for login, char-creation, and basic gameplay." [REPORT: testing strategy]

### DP-1146 — Adopt/fork an xterm.js MUD client pointed at /wstelnet

- **Add:** "Adopt `maldorne/mud-web-client` v4 — Vue 3 + TypeScript + xterm.js v6.0.0, the only actively maintained modern xterm.js MUD client with GMCP, MSDP, and MXP protocol support." [9][10][F3:12]
- **Add:** "All configuration via URL query parameters: `?proxy=wss://play.darkpawns.labz0rz.com/wstelnet`. No config files or environment variables needed." [9][F3:5]
- **Add:** "Companion `maldorne/mud-web-proxy` is NOT needed — Dark Pawns' `/wstelnet` endpoint IS the WebSocket-to-Telnet proxy, embedded in the Go binary." [9]
- **Add:** "Upstream builds to static `/dist` via `vue-tsc --noEmit && vite build`; multi-stage Docker build (node:20-alpine + nginx:alpine) in upstream, but no Docker runtime needed in production — just the static files." [9][F3:7]
- **Add:** "`useTelnetParser.ts` composable handles IAC subnegotiation parsing at byte level before UTF-8 decoding, preventing IAC bytes (0xFF) from being mangled by `TextDecoder`. This is cleanly factored and reusable even if the rest of the client is custom." [10][F3:3]

### DP-1147 — GMCP-driven status bar (replace JSON vars)

- **Add:** "maldorne client already parses GMCP events (`Char.Vitals`, `Room.Info`) via `useTelnetParser.ts` but includes no built-in status bar component — this is the only customization needed." [9][10]
- **Add:** "Implementation: a thin Vue component wrapping GMCP events from `useTelnetParser.ts`, displaying HP/mana/movement/XP from `Char.Vitals` and room name/exits from `Room.Info`." [9]

### DP-1148 — Build to static files under /opt/darkpawns/web (no Docker)

- **Add:** "maldorne produces `/dist` via `npm run build` (which runs `vue-tsc --noEmit && vite build`). Hugo can serve these as static assets, or they can be placed directly in the website's static directory under `/opt/darkpawns/web`." [9][F3:7]
- **Add:** "No Node runtime in production — static files only. The upstream Docker build uses `nginx:alpine` at runtime, but for Dark Pawns this is served directly by the existing Caddy reverse proxy." [9]

### DP-1149 — Cut public web terminal over to new client + /wstelnet

- **Add:** "Keep old `client.js` reachable at `/legacy` briefly as a rollback path during cutover, before full retirement." [REPORT: Phase 3]

### DP-1150 — Retire hand-rolled client.js JSON path for humans

- **Add:** "Keep `/ws` JSON endpoint for agents (not humans). The JSON protocol continues serving agent access even after all humans move to `/wstelnet` + xterm client. This means `/ws` is repurposed, not deleted." [REPORT: Phase 3]

### DP-1152 — Decouple physical socket from logical session (UUID + grace-period)

- **Add:** "Use UUIDv7 via `google/uuid` v1.6.0 (113K+ importers) — time-ordered by creation time, sortable, globally unique. UUIDv7 is preferred over UUIDv1/v6 per RFC 9562." [17][18]
- **Add:** "MCP's `Mcp-Session-Id` header + SSE `Last-Event-ID` replay pattern defines the exact resumability target: server-assigned session ID on `InitializeResult`, client reconnects with `Last-Event-ID` to replay missed events." [14][16]
- **Add:** "Dark Pawns' existing `ServerMessage.Seq` field (uint64) can serve as the cursor for MCP-style resumable event streams — tag each output event with a monotonic sequence number." [F6:9]
- **Add:** "Evennia's Portal/Server split is the reference architecture: network sessions (Portal) are separate from game logic sessions (Server) via AMP protocol, allowing sessions to survive server restarts." [17]
- **Add:** "Current state: `Session` struct is keyed by `playerName` string with no UUID; `UnregisterAndClose` immediately deletes the session from the map and closes the `*websocket.Conn`. Target: stable UUID identity with configurable grace period and ring-buffered output for reconnection." [16][F6:8]

### DP-1153 — [MVP] External MCP↔telnet adapter — agent plays over MCP without engine changes

- **Add:** "The external adapter can run inside the Go binary as a goroutine connecting to `localhost:7777` — avoids separate-process overhead while still keeping the game server unaware of MCP." [11]
- **Add:** "Tool list (imperative primitives, same pattern as minecraft-mcp-server): `look`, `north`, `south`, `east`, `west`, `up`, `down`, `attack`, `cast`, `say`, `get`, `drop`, `wear`, `wield`, `flee`, `inventory`, `score`, `who`, `create_account`." [12]
- **Add:** "The adapter negotiates GMCP on its telnet connection and surfaces GMCP data as structured MCP resources (`mud://session/gmcp`) — giving agents the same structured data that GUI clients get, without modifying the game server." [11]
- **Add:** "Auto-reconnection pattern from minecraft-mcp-server: every tool call first checks connection state and attempts reconnect if disconnected, polling up to `reconnectDelayMs + 5000ms`." [F4:2]
- **Add:** "Key limitation: the adapter can only respond to tool calls — it cannot push combat events or other proactive notifications to the MCP client. Text output must be parsed (ANSI-stripped) to extract game state." [11][F4:8]

### DP-1154 — Use Streamable HTTP as the MCP transport (standard-client compat + SSE push)

- **Add:** "Use `mark3labs/mcp-go` v0.56.0 (8.9k stars, 563 commits, 1.6k importers) — implements MCP spec 2025-11-25 with backward compatibility for 2024-11-05, 2025-03-26, and 2025-06-18." [4][5]
- **Add:** "`StreamableHTTPServer` implements `http.Handler` and mounts directly on the existing `net/http` mux at `/mcp` alongside `/ws`, `/wstelnet`, and `/api` — no separate port or process needed." [6]
- **Add:** "Single `/mcp` endpoint handles both: POST for client→server JSON-RPC messages, GET for server→client SSE stream (server-initiated push without client first sending data)." [F1:1][F1:2]
- **Add:** "Session management: server assigns a `Mcp-Session-Id` header (globally unique, cryptographically secure — UUID, JWT, or cryptographic hash) on the `InitializeResult` response. Client includes it in subsequent requests." [F1:3]
- **Add:** "SSE resumability: server attaches `id` fields to SSE events; client reconnects via GET with `Last-Event-ID` header to replay missed messages after disconnection." [F1:10]
- **Add:** "Server→client push: `SendNotificationToSpecificClient(sessionID, ...)` and `SendNotificationToAllClients(...)` for targeted and broadcast notifications." [6]
- **Add:** "Official Go SDK alternative: `modelcontextprotocol/go-sdk` (4.8k stars, maintained with Google, also implements spec 2025-11-25) — worth evaluating against mark3labs for API ergonomics and http.Handler integration before committing." [F1:8][F1:9]

### DP-1155 — Design guardrails: beat naive game-MCP servers (push, dynamic tools, memory, multi-agent)

- **Add:** "Per-session dynamic tool surface via `AddSessionTool`/`DeleteSessionTools` + `WithToolFilter` (list-time filtering based on session context). Example: `open_chest` tool is only registered/visible when the player is in a room containing a chest." [5][13]
- **Add:** "Combat push MUST use MCP resource subscriptions, not tool results — MCP tools are strictly request-response with no built-in event stream per invocation. The agent subscribes to `mud://player/combat` via `resources/subscribe` and receives `notifications/resources/updated` when combat state changes, then calls `resources/read` to get the updated state." [14][15]
- **Add:** "MSDP combat variables serve as the payload template: `OPPONENT_NAME`, `OPPONENT_HEALTH`, `OPPONENT_LEVEL`, `HEALTH`, `MANA`, `MOVEMENT` — these are battle-tested naming conventions across the MUD ecosystem." [F8:7]
- **Add:** "MSDP's REPORT mechanism (server auto-resends a variable whenever it changes) is the MUD-world precedent for MCP resource subscriptions — the same push-not-polling pattern, now over HTTP SSE instead of telnet subnegotiation." [F8:2]
- **Add:** "Multi-agent race conditions: the existing `session.Manager` mutex protects session state; game engine commands are already serialized through the command queue; MCP tool calls are just additional commands queued in the same sequence." [REPORT: risk register]
- **Add:** "Stretch: MCP sampling — the server can request AI-generated content FROM the client (NPC dialogue, quest text, dynamic room descriptions), creating a bidirectional AI loop. Nexlen/mud-mcp implements this pattern; it is NOT needed for MVP." [F4:10]

### DP-1174 — Zero-download agent onboarding: create_account MCP tool + transport-agnostic CreateCharacter core

- **Add:** "Core architecture: `CreateCharacter(spec) → Player` factored out of the nanny state machine into a shared function — transport-agnostic, same code path for telnet nanny and MCP `create_account` tool handler." [REPORT: zero-download vision]
- **Add:** "`create_account` tool signature: `create_account(name, sex, race, class, hometown, password?)` → returns session token. The password parameter is optional for agents (API-key auth) but preserved for human compatibility." [REPORT: zero-download vision]
- **Add:** "An MCP session is just another `session.Manager` Session backed by an MCP transport — the same live world, the same mutex-protected session map, not a separate instance." [REPORT: zero-download vision]
- **Add:** "Agent auth pattern from EllyMUD: 256-bit API keys (64 hex characters) stored server-side in `.env`, validated on all MCP requests except `/health`. All requests logged with IP addresses." [F4:9]
- **Add:** "`/skill.md` already exists and tells agents how to connect and play — `create_account` is the structured MCP tool that makes this zero-download." [REPORT: zero-download vision]

---

## B. NET-NEW ISSUES

### Native In-Process MCP Server (mark3labs/mcp-go — tools/resources/prompts, StreamableHTTPServer on http mux)

- **Project:** Agent Interface — MCP
- **Blocked by:** DP-1152 (session decoupling with UUID identity), DP-1153 (learnings from external adapter MVP)
- **Body:** Build the MCP server directly into the Dark Pawns binary as Phase 6, using `mark3labs/mcp-go` v0.56.0 [4][5]. Mount `StreamableHTTPServer` on the existing `net/http` mux at `/mcp` alongside `/ws`, `/wstelnet`, and `/api` — `StreamableHTTPServer` implements `http.Handler` so no separate process or port is needed [6]. Map MUD commands to MCP tools with declarative JSON schema input validation using mcp-go's schema builders [5]. Expose game state as MCP resources (`mud://room/current`, `mud://player/vitals`, `mud://player/inventory`, `mud://player/equipment`, `mud://world/time`) and agent identity as MCP prompts [REPORT: native MCP path]. Leverage mcp-go's per-session tool customization (`AddSessionTool`, `WithToolFilter`) for dynamic tool availability — e.g., `open_chest` only when the player is in a room with a chest [5][13]. Combat events flow through MCP resource subscriptions (`resources/subscribe` to `mud://player/combat` + `notifications/resources/updated`), not tool results, since MCP tools are strictly request-response [14][15]. The native server supersedes the external adapter (DP-1153) once session decoupling (DP-1152) is complete and the external adapter's tool surface has been validated.

### Transport-Abstraction Package Structure (pkg/mcp, pkg/transport)

- **Project:** Human Client Transport | Agent Interface — MCP
- **Blocked by:** (none — can start in parallel with Phase 1)
- **Body:** Establish the Go package structure for the transport and MCP layers before implementation begins, to avoid circular dependencies and ensure clean separation. Create `pkg/transport/` with `conn.go` (shared `net.Conn` interface and adapter types for both telnet and WS connections) and `wstelnet.go` (the `/wstelnet` HTTP handler using `coder/websocket`'s `NetConn()` to bridge WS binary frames to the existing `handleConn(rawConn net.Conn, ...)` telnet handler) [1][2][REPORT: package structure]. Create `pkg/mcp/` with `server.go` (`mark3labs/mcp-go` `StreamableHTTPServer` setup), `tools.go` (MUD command → MCP tool mapping with JSON schema input validation), `resources.go` (game state → MCP resource URI mapping: `mud://room/current`, `mud://player/vitals`, `mud://player/inventory`, `mud://player/equipment`, `mud://world/time`), `prompts.go` (agent identity/system prompt mapping), and `adapter.go` (the Phase 5 external adapter: MCP server wrapping telnet client connection to `localhost:7777`) [REPORT: package structure]. Register all handlers on a single `net/http` mux in `cmd/server/main.go`: `/ws` (existing JSON), `/wstelnet` (new binary WS), `/mcp` (new MCP), `/api` (existing API) [6]. Also define the shared interface types: `SessionID` (UUIDv7), `GameSession` (decoupled from transport with ring buffer), and `MUDToolHandler` function signature [REPORT: interface design].
