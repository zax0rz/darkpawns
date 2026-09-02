# Backlog: Human Client Transport — Telnet + WebSocket (Option B)

> Linear project: [Human Client Transport — Telnet + WebSocket (Option B)](https://linear.app/labz0rz/project/human-client-transport-telnet-websocket-option-b-0306488bf643)
> Project ID: `af87d84d-5a21-4aa7-8254-0f7d76ae533b` · Team: DP · Status: Backlog · 14 issues
>
> This file is a durable copy of the Linear project backlog, generated from the
> Linear GraphQL API on 2026-08-15 via `linear` CLI. The full design research is
> in [`TRANSPORT-MCP-RESEARCH.md`](TRANSPORT-MCP-RESEARCH.md) (`research/transport-mcp-2026-07-17/`).

## Project description

Converge human clients on telnet: a Go-native telnet-over-WebSocket endpoint (no Node proxy, no Docker), an off-the-shelf xterm client, and DNS/port reachability so telnet + browser both "just work". Agent /ws JSON stays until the MCP follow-on.

## Issues (14, all Backlog)

Phases map ~1:1 to the implementation plan's dependency graph: 0 Reachability · 1 `/wstelnet` · 2 xterm client · 3 cutover · 4 session decoupling.

### DP-1176 — Transport-abstraction package structure (pkg/transport, pkg/mcp)

**State:** Backlog

Establish the Go package structure for the transport + MCP layers before implementation, to avoid circular deps and keep clean separation. Can start in parallel with Phase 1.

`pkg/transport/`

* `conn.go` — shared `net.Conn` interface + adapter types for telnet and WS connections.
* `wstelnet.go` — the `/wstelnet` handler using `coder/websocket`'s `NetConn()` to bridge WS binary frames to the existing `handleConn(rawConn net.Conn, …)` telnet handler (one-liner: `websocket.NetConn(ctx, wsConn, websocket.MessageBinary)`).

`pkg/mcp/` (stubs, filled by the native-MCP-server work)

* `server.go` (mark3labs/mcp-go `StreamableHTTPServer`), `tools.go` (command→tool), `resources.go` (state→`mud://` URIs), `prompts.go` (agent identity), `adapter.go` (the Phase 5 external adapter → `localhost:7777`).

Register all handlers on one `net/http` mux in `cmd/server/main.go`: `/ws` (existing JSON), `/wstelnet` (new binary WS), `/mcp` (new MCP), `/api` (existing). Define shared interface types: `SessionID` (UUIDv7), `GameSession` (transport-decoupled + ring buffer), `MUDToolHandler`.

Source: `research/transport-mcp-2026-07-17/REPORT.md` (§Go implementation details) + `linear-reconciliation.md`.

### DP-1152 — Decouple physical socket from logical session (UUID + grace-period hold)

**State:** Backlog

Stretch. Give each player session a stable UUID and keep the logical session alive in the world for a configurable grace period when the socket drops, buffering output. On reconnect (WS or telnet), re-attach by UUID and flush buffered events. Improves link-dead recovery for humans on flaky mobile/WS, and directly pre-stages the MCP agent work (research doc, Recommendation 1 — stateful session bridge). Coordinate with the existing link-dead reaper.

---

**Deep research enrichment (2026-07-17):**

* **UUID format: UUIDv7 via** `google/uuid` **v1.6.0** (113K+ importers) — time-ordered by creation time, sortable, globally unique. Preferred over UUIDv1/v6 per RFC 9562.
* **MCP alignment:** MCP's `Mcp-Session-Id` header + SSE `Last-Event-ID` replay pattern defines the exact resumability target. Server-assigned session ID on `InitializeResult`, client reconnects with `Last-Event-ID` to replay missed events.
* **Dark Pawns' existing** `ServerMessage.Seq` **field** (uint64) can serve as the cursor for MCP-style resumable event streams — tag each output event with a monotonic sequence number.
* **Evennia's Portal/Server split** is the reference architecture: network sessions (Portal) are separate from game logic sessions (Server) via AMP protocol, allowing sessions to survive server restarts. Dark Pawns doesn't need the full AMP-based split, but the conceptual separation of "transport session" from "game session" is the right model.
* **Current state:** `Session` struct keyed by `playerName` string with no UUID; `UnregisterAndClose` immediately deletes session from map and closes `*websocket.Conn`. Target: stable UUID identity with configurable grace period and ring-buffered output for reconnection.

### DP-1151 — Regression: telnet ↔ web feature-parity checklist + DEPLOYMENT.md update

**State:** Backlog

Verify parity across both human transports before declaring done:

* ANSI/256 color, room rendering (no staircase), char-creation menus.
* GMCP-driven status bar (web) vs prompt/GMCP (native).
* MCCP2 for native clients; permessage-deflate for web.
* Link-dead behavior sane on both.

Then update **DEPLOYMENT.md**: new endpoints, hostnames, client build/deploy, networking (DNS-only telnet host + port-forward).

### DP-1150 — Retire hand-rolled client.js JSON path for humans (keep /ws for agents)

**State:** Backlog

Remove the bespoke JSON login/command/char_input/state/char_create protocol usage for **human** browsers (the old `web/client.js`). **Keep the** `/ws` **JSON protocol for AI agents** — it backs vars/subscribe/memory-bootstrap/decision-capture and is slated for the MCP follow-on, not this project. Make the split explicit in code + docs so the agent path isn't accidentally broken.

---

**Deep research enrichment (2026-07-17):**

* `/ws` **JSON endpoint is repurposed, not deleted.** It continues serving agent access even after all humans move to `/wstelnet` + xterm client.

### DP-1149 — Cut public web terminal over to new client + /wstelnet

**State:** Backlog

Point `play.darkpawns…` at the new xterm client and `/wstelnet` endpoint. Keep the old `client.js` reachable briefly (e.g. `/legacy`) as a rollback until the new path is proven, then remove.

---

**Deep research enrichment (2026-07-17):**

* **Keep old** `client.js` **reachable at** `/legacy` **briefly as a rollback path** during cutover, before full retirement.

### DP-1148 — Build to static files under /opt/darkpawns/web (no Docker)

**State:** Backlog

Produce a static build (Vite → static assets, or a hand-rolled page) served from `/opt/darkpawns/web` by the existing Go/Caddy static serving. No Nginx-in-Docker, no separate container. Document the build+deploy step in DEPLOYMENT.md style (build on mac-mini, scp assets, no service restart needed for pure asset changes).

---

**Deep research enrichment (2026-07-17):**

* **maldorne produces** `/dist` **via** `npm run build` (`vue-tsc --noEmit && vite build`). No Node runtime in production — static files only.
* Hugo can serve these as static assets, or they can be placed directly in `/opt/darkpawns/web`. The upstream Docker build uses `nginx:alpine` at runtime, but for Dark Pawns this is served directly by the existing Caddy reverse proxy.

### DP-1147 — GMCP-driven status bar (replace JSON vars)

**State:** Backlog

Rebuild the HP/mana/move/room status UI from the **GMCP** frames the telnet server already emits — `Char.Vitals`, `Char.Status`, `Room.Info`, `Char.Items` — instead of the bespoke JSON `vars` message the old `client.js` used. The web client parses GMCP subnegotiations client-side. Confirms feature parity with the retired status bar.

---

**Deep research enrichment (2026-07-17):**

* **maldorne client already parses GMCP events** (`Char.Vitals`, `Room.Info`) via `useTelnetParser.ts` but includes no built-in status bar component — this is the only customization needed.
* **Implementation:** a thin Vue component wrapping GMCP events from `useTelnetParser.ts`, displaying HP/mana/movement/XP from `Char.Vitals` and room name/exits from `Room.Info`.

### DP-1146 — Adopt/fork an xterm.js MUD client pointed at /wstelnet

**State:** Backlog

Evaluate **maldorne/mud-web-client** (Vue 3 + xterm.js, GMCP/MSDP/MXP, URL-param config) as the browser client, configured to hit our `/wstelnet` directly — **no Node proxy** (Option B). If the Vue app is heavier than we want, a minimal xterm.js page speaking binary WS is an acceptable fallback.

Decision to record: fork maldorne vs. minimal in-house xterm page. Either way it must be buildable to **static files** (no Docker).

---

**Deep research enrichment (2026-07-17):**

* **Adopt** `maldorne/mud-web-client` **v4** — Vue 3 + TypeScript + xterm.js v6.0.0, the only actively maintained modern xterm.js MUD client with GMCP, MSDP, and MXP protocol support.
* **All configuration via URL query parameters:** `?proxy=wss://play.darkpawns.labz0rz.com/wstelnet`. No config files or environment variables needed.
* **Companion** `maldorne/mud-web-proxy` **is NOT needed** — Dark Pawns' `/wstelnet` endpoint IS the WebSocket-to-Telnet proxy, embedded in the Go binary.
* **Upstream builds to static** `/dist` via `vue-tsc --noEmit && vite build`; no Docker runtime needed — just the static files served by Caddy or the Go binary.
* `useTelnetParser.ts` **composable** handles IAC subnegotiation parsing at byte level before UTF-8 decoding, preventing IAC bytes (0xFF) from being mangled by `TextDecoder`. Cleanly factored and reusable even if the rest of the client is custom.
* **Responsive layout** from 800x600 iframe to fullscreen.

### DP-1145 — e2e tests: telnet-over-WS login + char-creation parity with raw telnet

**State:** Backlog

Add e2e coverage proving `/wstelnet` behaves identically to raw telnet:

* Connect via WS binary, drive name → password → char creation → into world.
* Assert the byte stream matches the raw-telnet path (modulo compression negotiation).
* Reuse the session/telnet test harness pattern (`pkg/session/websocket_e2e_test.go`, `tests/e2e/telnet_smoke_test.go`).

---

**Deep research enrichment (2026-07-17):**

* **Extend the existing oracle differential harness (**`cmd/dp-oracle-diff/`**)** to drive both raw telnet and `/wstelnet` with the same scripted input, comparing output byte-for-byte.
* The oracle harness already drives both C and Go servers with scripted input; extend it to verify that WS-sourced sessions produce identical output to native telnet sessions for login, char-creation, and basic gameplay.

### DP-1144 — Disable MCCP2 on the WS path; rely on permessage-deflate

**State:** Backlog

MCCP2 (zlib) over a WebSocket is redundant and fights the browser's decompression. On the `/wstelnet` transport:

* Do **not** offer `IAC WILL COMPRESS2` (or refuse the client's DO).
* Enable WS **permessage-deflate** instead if compression is wanted.
* Keep MCCP2 on the raw TCP telnet path for native clients.

---

**Deep research enrichment (2026-07-17):**

* `coder/websocket` **supports full RFC 7692 permessage-deflate** with both context takeover and no-context-takeover modes. gorilla/websocket only supports no-context-takeover.
* **MCCP2 suppression mechanism:** Suppress via a boolean flag or interface type assertion on the `telnetConn` to skip the MCCP2 offer for WS-sourced connections. The telnet handler currently offers `IAC WILL COMPRESS2` to all connections during initial negotiation.
* **Open question:** permessage-deflate context takeover safety for MUD traffic patterns (small frequent messages vs. large bursts) is unverified — needs benchmarking.

### DP-1143 — Implement /wstelnet: binary WS ↔ telnet session bridge

**State:** Backlog

Add a `/wstelnet` handler to the Go HTTP server that speaks **binary WebSocket** carrying raw telnet bytes, bridged to the **existing** telnet session logic.

* Reuse `pkg/telnet` `handleConn` / `readLine` / `writeLoop` — abstract the `net.Conn` so a WS connection can back it (a `net.Conn` adapter over the WS binary stream is the cleanest seam). **Do not fork a third renderer** — the whole point is one telnet codepath.
* Login is the ordinary telnet name/password flow; no bespoke handshake.
* Server-side output already normalizes CRLF (PR [#314](<https://github.com/zax0rz/darkpawns/issues/314>)), so xterm renders correctly.

---

**Deep research enrichment (2026-07-17):**

* **Use** `coder/websocket` **v1.8.15, NOT gorilla/websocket.** gorilla's `NetConn()` returns the raw underlying TCP connection — direct I/O on it corrupts the WebSocket state. `coder/websocket`'s `NetConn()` wraps `*websocket.Conn` as `net.Conn` with each `Write` becoming a binary WS message and each `Read` receiving one — purpose-built for protocol tunneling. `coder/websocket` was created specifically because gorilla never addressed issues [#282](https://linear.app/labz0rz/issue/DP-1117/fidelity-o39-cast-bypasses-c-targeteligibilitygating-contract-pre) and [#441](<https://github.com/zax0rz/darkpawns/issues/441>) requesting this pattern.
* **The bridge is a one-liner:** `rawConn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)` → pass to `handleConn(rawConn net.Conn, manager, banLevel)`. Zero refactoring of telnet handler internals.
* **Compression:** `AcceptOptions{CompressionMode: websocket.CompressionContextTakeover}` for full RFC 7692 permessage-deflate with context takeover.
* **Deadline caveat:** `coder/websocket`'s `NetConn` closes the connection when a deadline fires (unlike typical `net.Conn` which only interrupts the goroutine). For long-lived MUD sessions, may need a custom adapter with softer deadline semantics, or accept WS reconnection on timeout.
* **Use** `MessageBinary` **frames** — telnet IAC sequences (0xFF) and ANSI escapes are arbitrary byte sequences, not valid UTF-8 text.
* **Migration path:** Add `coder/websocket` as a new dependency alongside the existing `gorilla/websocket` v1.5.3 (different import paths). `/wstelnet` uses `coder/websocket`; existing `/ws` JSON handler continues using gorilla until migrated.

### DP-1142 — Design browser WSS route + public hostnames (Cloudflare tunnel → Caddy)

**State:** Backlog

Plan the browser path before writing the endpoint:

* Pick hostnames: e.g. `play.darkpawns…` (web client, proxied/orange) and `mud.darkpawns…` (telnet, DNS-only/grey).
* Map `wss://play…/wstelnet` → Cloudflare tunnel → Caddy → Go `:4350`. Confirm WS upgrade survives the tunnel (it should — it's HTTPS).
* Note Caddy config + `cloudflared` ingress changes needed (documented, not yet applied).

---

**Deep research enrichment (2026-07-17):**

* **WSS endpoint:** `wss://play.darkpawns.labz0rz.com/wstelnet` rides the existing Cloudflare tunnel (orange-cloud).
* **Raw telnet:** separate DNS-only (grey-cloud) A record on `telnet.darkpawns.labz0rz.com:7777`.
* **Caddy cannot proxy raw TCP** — it is HTTP/HTTPS/gRPC/WebSocket only, cannot serve as a telnet relay.

### DP-1141 — Verify off-LAN connect with Mudlet / TinTin++ / raw telnet

**State:** Backlog

Once telnet is externally reachable, confirm real off-the-shelf clients "just work" end-to-end from outside the LAN:

* **Raw telnet / nc**, **Mudlet**, **TinTin++** (at least).
* Check: banner, name/password (ECHO off), char creation menus, gameplay, ANSI color.
* Confirm MCCP2 negotiation works with clients that request it (Mudlet), and MSSP shows in listings.

Capture any client-specific quirks.

### DP-1140 — Expose native telnet :7777 externally (DNS-only host + port-forward)

**State:** Backlog

Cloudflare's proxy only carries HTTP(S)/WSS — raw TCP 7777 through a proxied hostname gets reset/mangled (a plausible cause of the "disconnect at password" reports). Make native telnet reachable from the open internet:

* Add a **DNS-only (grey-cloud)** A/AAAA record, e.g. `mud.darkpawns…`, pointing at the origin's public IP.
* Firewall/router **port-forward TCP 7777** → CT 120 (192.168.1.121:7777).
* Alternative if we won't expose origin IP: Cloudflare **Spectrum** (paid) — evaluate cost vs. the port-forward.

Done = `telnet mud.darkpawns… 7777` connects from off-LAN.

---

**Deep research enrichment (2026-07-17):**

* **Cloudflare Spectrum requires Enterprise ($1,000+/mo)** for generic TCP passthrough — Pro/Business plans only support Minecraft, SSH, and RDP as hardcoded exceptions. Not viable.
* **Cloudflare Tunnel (**`cloudflared`**)** can carry TCP but requires end users to install the WARP client — unsuitable for public MUD access.
* **Practical setup:** DNS-only (grey-cloud) A record for `telnet.darkpawns.labz0rz.com` → VM's real IP. Keep existing Cloudflare tunnel (orange-cloud) for `play.darkpawns.labz0rz.com` → Caddy → Go binary (HTTPS/WSS). The origin IP exposure is acceptable for a MUD.
