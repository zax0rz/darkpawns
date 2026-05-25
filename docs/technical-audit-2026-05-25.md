---
tags: [active, audit, outreach]
date: 2026-05-25
author: Daeron
---
# Technical Audit: Marketing Strategy Readiness
*Date: 2026-05-25*

Audit of the Dark Pawns codebase against the technical requirements implied by
`docs/outreach/MARKETING-STRATEGY.md` (Gemini Deep Research brief). The strategy's
"Platform Strategy" section names four integrations as load-bearing:

```
GitHub  ↔  Website (web client + live analytics)  ↔  Grapevine (MSSP + cross-MUD chat)  ↔  Discord (GMCP webhooks)
```

Of those four, **GitHub and the web client exist. Grapevine, MSSP, GMCP, and the
Discord bridge do not exist at all.** This document grades each requirement against
actual code.

---

## Executive Summary

The protocol layer is the gap. Dark Pawns speaks two transports today — a raw TCP
telnet listener (`pkg/telnet/listener.go`) and a WebSocket bridge
(`pkg/session/manager.go`) — and **neither implements a single MUD telnet protocol
extension**. The telnet IAC handler negotiates only ECHO and SGA; every other
option is refused with DONT/WONT, and subnegotiation (the `IAC SB ... IAC SE`
envelope that carries GMCP/MSDP/MSSP/NAWS payloads) is read and **silently
discarded** (`listener.go:435-451`). There is no GMCP, no MSDP, no MSSP, no NAWS,
no CHARSET, no TTYPE. A `grep` across the whole repo for these protocol names
returns zero hits in production Go code. So MUD directories cannot auto-index us
(MSSP), Grapevine cannot talk to us (it speaks GMCP-style JSON over a WebSocket),
and rich clients (Mudlet, MUSHclient) get nothing structured.

The good news, and it is genuinely good: **the hard conceptual work for GMCP is
already done.** The agent var-subscription system (`pkg/session/agent_vars.go`)
is a working, tested implementation of exactly the data model GMCP transmits —
named variables (`HEALTH`, `ROOM_VNUM`, `ROOM_MOBS`, `INVENTORY`, …), a subscribe
message, dirty-tracking, and delta-flush after each command. It serializes to
`{"type":"vars","data":{...}}` JSON over the WebSocket. GMCP is the same idea with
a different envelope (`Char.Vitals`, `Room.Info`) carried inside telnet
subnegotiation instead of a WebSocket frame. We are not building GMCP from nothing;
we are re-skinning a system we already run in production for the AI agents.

The critical path for the marketing rollout is: **(1) MSSP first** — it is tiny,
it is what gets us listed on MUD directories (Phase 2 of the rollout), and it has
no dependencies. **(2) GMCP second** — reuse the var system, wire it to both the
telnet subneg path and a WebSocket GMCP channel, and the web client's half-built
status bar (which has dead Mana/Move/Gold bars today) lights up for humans.
**(3) Grapevine and the Discord bridge third** — both are external network
integrations that depend on having GMCP/MSSP plumbing to feed them. None of this
blocks Phase 1 (the "Show HN" engineering-story launch), which only needs the
existing web client to hold up under load.

---

## Protocol Layer

Reference: `pkg/telnet/listener.go` (541 lines). Telnet protocol constants are
defined at `listener.go:19-34` — only `IAC, WILL, WONT, DO, DONT, SB, SE,
OPT_ECHO, OPT_SGA`. No option constants for any MUD extension exist.

### GMCP (Generic MUD Communication Protocol)
- **Status: NOT BUILT (telnet) / PARTIAL via proxy (WebSocket agent path)**
- **Current state:**
  - Telnet: option 201 not negotiated. `readLine()` (`listener.go:392-475`) reads
    subnegotiation in the `case SB:` branch (`:435-451`) as a byte-skip loop that
    discards everything until `IAC SE`. No payload is parsed or dispatched.
  - WebSocket: the functional equivalent of GMCP exists as the agent var system.
    `agent_vars.go` defines 19 subscribable variables (`:14-42`), a `subscribe`
    handler (`handleSubscribe`, `:65-85`), `markDirty` (`:90-101`),
    `flushDirtyVars` (`:105-134`), and `sendFullVarDump` (`:138-153`). Output
    envelope is `{"type":"vars","data":{...}}` (`MsgVars`, `protocol.go:19`).
  - **Crucially, this is gated to agents.** `handleSubscribe` rejects non-agents
    (`:69-72`); `markDirty`/`flushDirtyVars` early-return unless `s.isAgent`
    (`:93, :107`). Human web clients receive **no** vars messages today.
- **What is needed:**
  1. Telnet: add `OPT_GMCP byte = 201`. On connect, offer `IAC WILL GMCP`. In the
     `case SB:` branch, when the option byte is 201, capture the payload, split on
     the first space into `Package.Message` + JSON, and dispatch. Add an outbound
     `sendGMCP(pkg, payload)` writing `IAC SB 201 <pkg> <json> IAC SE`.
  2. Define a GMCP module mapping over the existing var values: `Char.Vitals`
     (HEALTH/MANA/MOVE), `Char.Status` (LEVEL/EXP/GOLD/POSITION), `Room.Info`
     (VNUM/NAME/EXITS), `Room.Mobs`, `Room.Items`, `Char.Inventory`. The
     `buildVarValue` switch (`agent_vars.go:156-218`) already produces every one
     of these values — wrap them in GMCP package names.
  3. Un-gate the var/GMCP push from `isAgent` so human sessions get it too (this
     is what feeds the web client status bar — see Web Client section).
  4. WebSocket GMCP: agents already get JSON vars; for parity, expose the same
     GMCP package names to the web client over the existing WS envelope.
- **Effort estimate: M** (telnet subneg parse + dispatch is ~150 LOC; the data
  model is reused, not rebuilt).
- **Dependencies:** none hard. Should land after MSSP (smaller, proves the IAC
  subneg plumbing). Un-gating vars from `isAgent` touches `markDirty`/`flushDirtyVars`
  and the login full-dump path — needs care so agent telemetry semantics don't
  change.

### MSDP (Mud Server Data Protocol)
- **Status: NOT BUILT**
- **Current state:** nothing. No option 69, no MSDP table/array byte encoding.
  Subnegotiation is discarded as above.
- **What is needed:** MSDP is a competing standard to GMCP for the same purpose
  (server→client variable reporting), using a binary VAR/VAL byte encoding rather
  than JSON. Variables to report would be identical to GMCP: HEALTH, MAX_HEALTH,
  ROOM_NAME, ROOM_VNUM, EXITS, etc. — all already computed in `buildVarValue`.
- **Effort estimate: M** (similar to GMCP, but the byte-level table encoding is
  fiddlier than JSON).
- **Dependencies:** the same subneg dispatch plumbing as GMCP.
- **Recommendation: DEFER / likely SKIP.** GMCP and MSDP solve the same problem.
  Modern clients and Grapevine favor GMCP. Building both is duplicated effort for
  marginal client coverage. Build GMCP; only add MSDP if a specific target client
  in the outreach list demands it.

### MSSP (MUD Server Status Protocol)
- **Status: NOT BUILT**
- **Current state:** nothing. Option 70 is not negotiated. No status table is
  assembled anywhere.
- **What is needed:** This is the smallest and highest-leverage protocol for the
  marketing plan — it is *the* mechanism by which MUD directories (and Grapevine's
  crawler) auto-index a game, which the rollout's Phase 2 explicitly requires
  ("Register on Grapevine + MUD directories"). Implementation:
  1. Add `OPT_MSSP byte = 70`. On connect offer `IAC WILL MSSP`.
  2. When the client (a crawler) sends `IAC DO MSSP`, respond with one
     `IAC SB 70 <MSSP_VAR name MSSP_VAL value ...> IAC SE` block.
  3. Populate the standard fields from data we already have:
     - `NAME` = "Dark Pawns"
     - `PLAYERS` = live count — `Manager.sessions` map length (add a
       `Manager.PlayerCount()` helper; the map is at `manager.go:78`)
     - `UPTIME` = process start epoch
     - `PORT` = 7777, `CODEBASE` = "CircleMUD 3.0 (Go port)", `FAMILY` = "DikuMUD"
     - `CREATED` = 1994/1997, `WEBSITE` = darkpawns.labz0rz.com, plus
       `HOSTNAME`, `LANGUAGE`, `LOCATION`, genre/subgenre fields.
- **Effort estimate: S** (~80 LOC; one static-ish table plus a live player count).
- **Dependencies:** none. **Build this first.** It is the cheapest item that
  unblocks a named Phase-2 rollout requirement, and it forces us to get the
  `IAC SB ... SE` write path right before the larger GMCP work.

### Other Telnet Protocols (NAWS, CHARSET, TTYPE)
- **NAWS (Negotiate About Window Size, option 31): NOT BUILT.** Window-size
  subneg is discarded. The session struct even has a `screenSize int` field
  (`manager.go:656`, marked `//nolint:unused`) that nothing populates — wiring
  NAWS would finally feed it. Effort: **S**. Low priority — pleasant for telnet
  users with `more`-style paging, not required by the strategy.
- **CHARSET (option 42): NOT BUILT.** No UTF-8/Latin-1 negotiation. The server
  emits the ASCII greeting logo (`listener.go:36-48`) and assumes the client
  copes. Effort: **S**. Low priority.
- **TTYPE / MTTS (terminal type, option 24): NOT BUILT.** We can't detect client
  capabilities (ANSI color, 256-color, Mudlet). Effort: **S**. Low priority, but
  TTYPE is the conventional way to decide whether to send color — pairs naturally
  with the existing `charColor` creation preference (`manager.go:629`).
- **IAC handling summary:** The negotiation is deliberately minimal and *safe* —
  it never hangs on an unknown option (everything gets DONT/WONT) and it correctly
  consumes subneg envelopes without choking. That is a sound base to extend; the
  `case SB:` skip-loop is the single insertion point for all four protocols above.

---

## Web Client

Reference: `web/client.js` (282 lines), `web/index.html` (74 lines).

### Current State
- xterm.js terminal (5.3.0, CDN-loaded) with FitAddon. Themed dark/parchment.
- **Parses structured JSON, not raw text.** `ws.onmessage` (`client.js:148-198`)
  switches on `msg.type`: `state` → `handleStateMsg`, `vars` → `handleVarsMsg`,
  `char_create` → prompt+options rendering, `error`/`event`/`text` → terminal
  writeln. Falls back to raw text on JSON parse failure.
- Status bar exists in markup (`index.html:41-66`): HP / Mana / Move fill-bars
  plus Level and Gold readouts. `client.js` has color-ramp logic per bar
  (`hpColor`/`manaColor`/`moveColor`, `:44-60`) and `updateStatusBar` (`:69-91`).
- Login flow is name → password (`loginPhase` state machine, `:220-269`).

### What is Missing vs Strategy
1. **The Mana / Move / Gold bars are dead for human players.** Two independent
   reasons: (a) `handleVarsMsg` (`:108-120`) is the only path that sets mana/move/
   gold, and vars messages are **agent-only** on the server (`agent_vars.go` gates
   on `isAgent`). (b) The human path is `handleStateMsg` (`:93-106`), but the
   server's `PlayerState` struct (`protocol.go:77-90`) only carries
   `health/max_health/level` — there are **no** mana/move/gold fields to send.
   So the bars render at 0%/"—" forever for a real player. This is a concrete
   bug, not just a missing feature, and it is exactly what GMCP `Char.Vitals`
   would fix. **Fixing it is a subset of the GMCP work.**
2. **No GMCP-driven UI beyond the status bar.** Strategy wants "structured JSON
   payloads for visual UI elements." There is no minimap (despite the website
   having full world map JSON at `website/static/map/world.json` — a minimap could
   be driven by `Room.Info` GMCP + that map data), no spell/cooldown bar, no
   inventory/equipment panel, no target/enemy health display. The data for all of
   these already exists server-side (`buildInventory`, `buildEquipment`,
   `buildRoomMobs` in `agent_vars.go`); the client just doesn't request or render it.
3. **No guest mode. Character creation / login is mandatory.** Both transports
   require a name and password before any world interaction (web:
   `client.js:225-250`; telnet: `listener.go:155-244`). The strategy's "frictionless
   web client" and ">5% visitor conversion" metric argue for a **guest/tourist
   mode** — drop a visitor into a read-only starting room with a "claim your name"
   upsell. This also directly serves Guerrilla Tactic 1 ("Etch Your Name in the
   1994 Stone Database"): the funnel is land → look around as guest → claim handle.
   No such path exists today.

### Effort Estimates
- Fix dead status bar (add mana/move/gold to `PlayerState` + un-gate vars, or send
  GMCP `Char.Vitals` to web): **S**, and it falls out of the GMCP work for free.
- Minimap panel driven by existing world map JSON + room updates: **M**.
- Inventory/equipment/spell panels: **M** (data exists; pure client rendering + a
  push trigger).
- Guest mode: **M–L** — needs a server-side ephemeral session that bypasses
  DB auth (`manager.go` login path) and a constrained command set; touches auth
  assumptions, so not trivial.

---

## Server Architecture

### Message Flow
Data flows game-logic → session → client through one JSON envelope. The canonical
type is `ServerMessage{Type, Seq, Data}` (`protocol.go:34-39`). Game code reaches
players via two wired callbacks set up in `NewManager` (`manager.go:172-200`):
- `world.MessageSink` (`:173-195`) — wraps a raw text string into
  `ServerMessage{Type: MsgEvent, Data: EventData{Type:"text", Text: ...}}` and
  pushes onto the session's buffered `send` channel (cap 256).
- `world.CloseConn` (`:198-200`) — routes game-initiated disconnects.
Combat broadcasts use the same envelope via `SetCombatBroadcastFunc`
(`manager.go:222-237`) → `BroadcastToRoom` (`:569-589`), which fans a marshaled
message to every session whose `player.GetRoom()` matches.

Each transport drains the `send` channel and formats per its medium:
- WebSocket: `writePump` (in `session_send.go`) ships the JSON frame as-is; the
  browser parses it (`client.js`).
- Telnet: `writeLoop` (`listener.go:299-346`) unmarshals each `ServerMessage` and
  re-renders to plain text per `sm.Type` (`state`→`formatState`, `event`→text,
  etc.). So telnet is a JSON→text downconverter sitting on the same bus.

So there is **one structured message bus already**, and both transports are
adapters over it. That is the architectural fact that makes GMCP cheap.

### GMCP Readiness
**HIGH — no major refactor required.** Three reasons:
1. The data is already computed. `buildVarValue` (`agent_vars.go:156-218`) returns
   live values for all 19 variables — vitals, position, room, mobs, items,
   inventory, equipment. GMCP packages are just a renaming of these.
2. The push mechanics already exist. `markDirty`/`flushDirtyVars` is a working
   subscribe-and-delta system. GMCP is the same loop with a telnet-subneg writer
   instead of (or in addition to) the WS channel writer.
3. The transport split is clean. Adding a `sendGMCP` to `telnetConn` and a GMCP
   branch in `writeLoop` is additive; the WebSocket path needs only the `isAgent`
   gate lifted.

The one real refactor risk is the `isAgent` gate. Today vars are an agent-only
research-telemetry feature. GMCP wants the same data flowing to *human* clients.
Lifting that gate must preserve agent semantics (the full-dump-on-login, the
EVENTS draining at `agent_vars.go:206-214`) — do it as an explicit "client
supports structured data" capability flag per session, not by deleting the
`isAgent` checks wholesale.

### Event System
Game events today are coarse: a `type` string inside `EventData` (`protocol.go:111-116`)
with values `"enter"`, `"leave"`, `"say"`, `"combat"`, `"text"`. There is also a
richer per-command event queue for agents: `pendingEvents` on the session
(`manager.go:621`) drained through the `EVENTS` var. Candidates to expose over
GMCP/Discord without new game logic:
- Vitals deltas (already dirty-tracked on damage via `SetDamageFunc`, `manager.go:292-298`).
- Room transitions (`Room.Info` on movement).
- Combat start/death — `SetDeathFunc` (`:255-287`) and the combat broadcast are
  already hooks; a Discord "X was slain by Y" feed is a webhook call from inside
  `DeathFunc`.
- Login/logout — `cleanupSession` already broadcasts a leave message (`:488-499`);
  a level-up / death / login feed is the natural Discord bridge content.

---

## Infrastructure

### Discord Bot
- **Status: NOT BUILT.** No Discord client, no bot token handling, no gateway
  connection anywhere in the Go code. The only `webhook` machinery is in the
  `dp-goat` CLI's generic delivery sink (`cmd/dp-goat/internal/cli/deliver.go`):
  `--deliver webhook:<url>` POSTs an arbitrary body to a URL (`deliverWebhook`,
  `:92-111`). That could fire a Discord webhook, but it is a generic agent-output
  pipe, not a game↔Discord bridge.
- **What is needed for the strategy's "Discord webhooks (bridge game world with
  mobile)":** the *cheap* direction (game → Discord) is a single outbound
  `http.Post` to a Discord webhook URL, called from the existing death/login/
  level-up hooks above. **Effort: S** for one-way notifications. A true two-way
  bridge (Discord chat → in-game `gossip`) needs a real bot (gateway websocket,
  e.g. `discordgo`) and a route into the comm system — **Effort: M–L**.

### MUD Directories
- **Status: NOT REGISTERED, and cannot be auto-indexed.** Directory listing
  (The Mud Connector, Grapevine's index, MUDverse) relies on **MSSP**, which does
  not exist (see Protocol Layer). Until MSSP ships, registration is manual-form-only
  and we report nothing programmatic. **MSSP is the unlock here.**

### Grapevine
- **Status: NOT BUILT.** No client for `wss://grapevine.haus/socket`, no Grapevine
  auth/channel code, nothing. Grapevine is a WebSocket service speaking a JSON
  protocol (authenticate → subscribe to channels like `gossip` → relay
  player/message events cross-MUD). We *do* have a mature WebSocket stack
  (`gorilla/websocket` already a dependency, `manager.go:16`) and a comm system to
  bridge into, so the building blocks are present.
- **What is needed:** a `pkg/grapevine` client: dial the socket, authenticate with
  a client-id/secret, send `channels/subscribe`, translate inbound Grapevine
  messages into in-game broadcasts and outbound in-game `gossip` into Grapevine
  `channels/send`, plus periodic `players/sign-in` presence. **Effort: M–L.**
- **Dependencies:** wants MSSP-style status data and the comm-event hooks; best
  done after MSSP + the event hooks are in place.

### Reverse Proxy / WSS
- **Caddy** (`website/deploy/Caddyfile`, `docker-compose.yml`): `caddy:2-alpine`
  container, ports 80/443. **`auto_https off`** — Caddy is *not* terminating TLS
  here. Per `TOOLS.md`, public TLS is handled by the **Cloudflare Tunnel**
  fronting `darkpawns.labz0rz.com`, so the browser's `wss://` is terminated at the
  Cloudflare edge and proxied plaintext internally.
- Routing: `/ws`, `/api/*`, `/admin/*`, `/health`, `/metrics` all `reverse_proxy
  localhost:4350` (the Go server); everything else is served from the Hugo static
  site at `/srv/hugo/`. The game is also mounted under `/game/*` with prefix
  stripping. **WSS to the web client works today** through this chain — this is the
  one piece of the strategy's platform diagram that is fully operational.
- Go-side TLS (`main.go:286-315`) is optional via `TLS_CERT_FILE`/`TLS_KEY_FILE`/
  `USE_TLS`; in the deployed topology it runs plaintext behind the tunnel, which
  is correct for edge-terminated TLS.
- **Inconsistency to flag:** `index.html:26` advertises telnet on **port 4000**,
  but the server's default telnet port is **7777** (`main.go:70`) and `TOOLS.md`
  documents 7777. Port 4000 in the Caddyfile is *The Soviet*, not Dark Pawns.
  The landing page telnet instruction is wrong — quick fix, but it is the literal
  first call-to-action a retro/telnet purist (a core target demographic) will try.

---

## Recommended Implementation Order

Phased to align with the rollout sequence in the strategy.

**Phase 0 — Pre-launch hygiene (blocks nothing, do immediately)**
- Fix the telnet port in `index.html` (4000 → 7777). **S.** Trivial, but it is a
  broken CTA aimed at the exact audience Phase 2 targets.

**Phase 1 — supports "Show HN" engineering launch (Month 1)**
- No protocol work required; the web client + WSS already function. The risk here
  is load, not features — see Risk Assessment.
- *Optional but high-value:* fix the dead Mana/Move/Gold status bar by adding those
  fields to `PlayerState` (`protocol.go`) and populating them in the state builder.
  **S.** Makes the web client look finished for the HN crowd. (Subsumed later by
  GMCP, but worth doing standalone if GMCP slips.)

**Phase 2 — supports "register on Grapevine + MUD directories" (Month 2)**
1. **MSSP** — directory auto-indexing. **S.** No dependencies. Establishes the
   `IAC SB ... SE` write path. *Do this first of all protocol work.*
2. **GMCP** — reuse the var system; un-gate from `isAgent` behind a capability
   flag; map to `Char.Vitals`/`Room.Info`/etc.; wire both telnet subneg and WS.
   Lights up the web client status bar and unlocks rich-client support. **M.**
   Depends on the subneg plumbing proven by MSSP.
3. **Discord one-way feed** — outbound webhook from existing death/login/level
   hooks. **S.** Independent; can run parallel to GMCP. Serves community-building.

**Phase 3 — supports "Grand Preservation Launch" + community depth (Month 3-4)**
4. **Grapevine client** — cross-MUD `gossip`, presence. **M–L.** Depends on the
   comm-event hooks and benefits from MSSP/GMCP being done.
5. **Guest mode** — frictionless tourist path feeding the "Etch Your Name"
   campaign funnel. **M–L.** Independent of protocols; depends on auth-path changes.
6. **Web GMCP UI** — minimap (reuse `website/static/map/world.json`), inventory/
   equipment/spell panels. **M.** Depends on GMCP (#2).
- *Defer/skip:* MSDP (redundant with GMCP), NAWS/CHARSET/TTYPE (nice-to-have for
  telnet purists; bundle as one **S** ticket only if a target client needs them).

---

## Risk Assessment

1. **The `isAgent` gate is load-bearing and subtle.** Vars are currently an
   agent-only telemetry channel with specific semantics (full dump on login, EVENTS
   queue draining, dirty-tracking from the combat ticker goroutine via `markDirty`,
   `agent_vars.go:90`). GMCP wants the same machinery feeding humans. Lifting the
   gate naïvely risks (a) double-sending to agents, (b) changing the agent EVENTS
   contract that `dp-goat` depends on, (c) introducing the data race that the
   `agentMu` mutex (`manager.go:618`) currently guards — `flushDirtyVars` is called
   from both readPump and the combat ticker. Treat the un-gating as a careful
   capability-flag refactor with tests, **not** a `grep -v isAgent`.

2. **Telnet subnegotiation parsing is currently a discard loop — easy to get
   wrong when made real.** The `case SB:` block (`listener.go:435-451`) assumes
   well-formed `IAC SE` termination and has no length cap on the subneg body. A
   malicious client could stream an unterminated subneg and the loop reads until
   EOF — acceptable as a discard, dangerous once we start buffering the payload
   for GMCP/MSSP. Add a subneg length bound (mirror `maxInputLen`, `listener.go:390`)
   when implementing parse.

3. **`send` channel is a fixed 256-buffer with drop-on-full semantics**
   (`manager.go:378`, and the `default:` drops in `MessageSink`/`BroadcastToRoom`/
   `flushDirtyVars`). GMCP increases message volume (vitals deltas every combat
   tick). Under the HN/"Resurrection Day" load spikes the strategy is courting,
   dropped GMCP frames mean stale UI bars. Validate buffer sizing and consider
   coalescing vitals updates before Phase 3 events.

4. **No graceful shutdown / lifecycle management** — flagged in `main.go`'s own
   header note ([M-07], `:1-33`). In-flight WebSocket and telnet connections are
   not drained on SIGTERM; only zone-reset and world-save are awaited
   (`main.go:317-327`). A press-driven traffic spike followed by a deploy could
   drop every connected player ungracefully. Worth hardening before a publicized
   launch where the server will actually be redeployed under live load.

5. **Single static landing page, CDN-dependent client.** `index.html` pulls
   xterm.js and the fit addon from `cdn.jsdelivr.net` (`:9, :70-71`). If jsDelivr
   is unreachable (or blocked in a viewer's region), the web client — the entire
   Phase-1 conversion funnel — fails to load. For a launch banking on a traffic
   spike from a single HN/press hit, self-host xterm.js. **S**, high resilience
   value.

6. **Grapevine/Discord are external network dependencies inside the game process.**
   A hung `wss://grapevine.haus` dial or a slow Discord webhook must never block a
   game tick or a player's command. Any such integration needs its own goroutine,
   timeouts, and a circuit breaker — the synchronous `http.Post` pattern in
   `deliverWebhook` (`deliver.go:92-111`) is fine for a CLI but must not be copied
   into the request path of the live server.

---

*Audit complete. Data points are cited to file:line against the working tree as of
2026-05-25. The headline: we already run a GMCP-shaped data system for the agents —
the marketing-critical protocol work is mostly adaptation, not invention. MSSP is
the cheap first domino; everything directory- and Grapevine-related waits behind it.*
