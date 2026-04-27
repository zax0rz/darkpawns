# Dark Pawns: Web Admin & Client Architecture Plan

**Status:** Opus-reviewed, vetted  
**Date:** 2026-04-24  
**Author:** BRENDA69  
**Reviewer:** Opus security/architecture audit

---

## 1. Vision

Replace 8,600 lines of CircleMUD C OLC editors and a bare-bones telnet experience with a unified, production-grade web application. The admin panel is not just a game editor — it's a research platform for studying AI agents as narrative participants.

### The Core Insight

No existing platform covers this use case:
- **MUD admin panels** (Evennia, ExVenture, Written Realms) — rooms/mobs/items only, zero AI awareness
- **AI observability** (LangSmith, LangFuse, AgentOps) — trace/cost/evals, zero game awareness
- **Agent management** (Kore.ai AMP, CrewAI) — orchestration only, no game engine

**We build our own.** React SPA with pre-built components where it makes sense, custom where it matters.

---

## 2. Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                        BROWSER (React SPA)                      │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐      │
│  │   Game   │ │   AI     │ │ Research │ │   Operations   │      │
│  │  Panel   │ │  Panel   │ │  Panel   │ │     Panel      │      │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └───────┬────────┘      │
│       │            │            │               │               │
│  ┌────┴────────────┴────────────┴───────────────┴────────┐      │
│  │             WebClient (game terminal SPA tab)          │      │
│  └─────────────────────────┬──────────────────────────────┘      │
└────────────────────────────┼─────────────────────────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │            TLS              │
              │     (Caddy reverse proxy)    │
              └──────────────┬──────────────┘
                             │
┌────────────┐   ┌───────────┴───────────┐   ┌────────────────────┐
│ Game Port  │   │   Admin Port (8081)    │   │   LangFuse (self)  │
│ 8080/WS    │   │   /admin/* REST        │   │   /api/traces      │
│ No admin   │   │   CORS locked to SPA   │   │   AI observability │
└────────────┘   └───────────┬───────────┘   └────────────────────┘
                             │
┌────────────────────────────┼──────────────────────────────────────┐
│                     DARK PAWNS GO SERVER                          │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │              Existing Middleware Stack                     │    │
│  │  ┌────────┐ ┌────────┐ ┌──────────┐ ┌──────────────┐    │    │
│  │  │ CORS   │ │ Sec    │ │ Privacy  │ │ Content-Neg  │    │    │
│  │  │ (web/) │ │ (web/) │ │ (pkg/)   │ │ (web/)       │    │    │
│  │  └────────┘ └────────┘ └──────────┘ └──────────────┘    │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  ┌────────────┐  ┌──────────────────┐  ┌─────────────────────┐   │
│  │ /ws        │  │ /api/*           │  │ /admin/*            │   │
│  │ WebSocket  │  │ Game REST API    │  │ Admin REST API      │   │
│  │ (port 8080)│  │ (existing stubs) │  │ (new, port 8081)    │   │
│  └────────────┘  └──────────────────┘  └─────────────────────┘   │
│                         │                       │                │
│                         ▼                       ▼                │
│              ┌──────────────────────────────────────────┐        │
│              │        Game World (in-memory)           │        │
│              │  Rooms │ Mobs │ Items │ Zones │ Shops   │        │
│              │  Triggers │ Scripts │ Resets            │        │
│              │  Mutex: sync.RWMutex                    │        │
│              │  Reads: SnapshotManager (atomic.Pointer)│        │
│              │  Writers: ZoneDispatcher (per-zone goros)│        │
│              └──────────────────────────────────────────┘        │
│                         │                                        │
│              ┌──────────┴──────────┐  ┌────────────────────┐     │
│              │     PostgreSQL      │  │  Narrative Memory  │     │
│              │  Players │ Saves    │  │  Agent interactions│     │
│              │  Agent keys │Admin  │  │  Event journal     │     │
│              │  World entities     │  │                    │     │
│              └─────────────────────┘  └────────────────────┘     │
└───────────────────────────────────────────────────────────────────┘
```

---

## 3. Backend — Admin REST API (`pkg/admin/`)

### Route Design

All admin routes prefixed `/admin/`:

| Method | Route | Purpose |
|--------|-------|---------|
| GET | `/admin/zones` | List zones with search/filter |
| GET | `/admin/zones/:id` | Zone detail with room list |
| PUT | `/admin/zones/:id` | Update zone metadata |
| POST | `/admin/zones` | Create new zone |
| DELETE | `/admin/zones/:id` | Delete zone |
| GET | `/admin/zones/:id/rooms` | Rooms in zone (graph data) |
| GET | `/admin/rooms/:id` | Room detail (exits, flags, extra descr) |
| PUT | `/admin/rooms/:id` | Update room |
| POST | `/admin/rooms` | Create room |
| POST | `/admin/rooms/:id/link` | Connect two rooms (exit) |
| GET | `/admin/mobs` | Mob list |
| GET | `/admin/mobs/:id` | Mob detail (dice HP, scripts, flags) |
| PUT | `/admin/mobs/:id` | Update mob |
| POST | `/admin/mobs` | Create mob |
| GET | `/admin/objects` | Object list |
| GET | `/admin/objects/:id` | Object detail (type-aware values, applies) |
| PUT | `/admin/objects/:id` | Update object |
| POST | `/admin/objects` | Create object |
| GET | `/admin/shops` | Shop list |
| GET | `/admin/shops/:id` | Shop detail (multi-room, producing, buy) |
| PUT | `/admin/shops/:id` | Update shop |
| GET | `/admin/triggers` | Trigger list |
| PUT | `/admin/triggers/:id` | Update trigger (Lua script) |
| POST | `/admin/triggers` | Create trigger |
| POST | `/admin/zones/:id/reset` | Trigger zone reset (via dispatcher) |
| GET | `/admin/panels/rooms/:id/context` | Room context for AI |

### AI-Specific Routes

| Method | Route | Purpose |
|--------|-------|---------|
| GET | `/admin/agents` | List all AI agents (active/inactive) |
| GET | `/admin/agents/:id` | Agent detail — model, config, state |
| PUT | `/admin/agents/:id/config` | Update agent model/params |
| GET | `/admin/agents/:id/log` | Agent action log (filterable) |
| GET | `/admin/agents/:id/memory` | View agent's narrative memory |
| POST | `/admin/agents/:id/reset` | Reset agent state |
| POST | `/admin/agents` | Spawn new AI agent |
| DELETE | `/admin/agents/:id` | Kill/remove agent |
| GET | `/admin/narrative` | Narrative event feed (all agents) |
| GET | `/admin/narrative/:id` | Single event detail |
| GET | `/admin/conversations` | Cross-agent conversations |
| GET | `/admin/conversations/:id` | Full conversation transcript |
| GET | `/admin/research` | Research data exports |
| POST | `/admin/research/export` | Export interaction dataset |
| GET | `/admin/metrics` | Aggregate agent metrics |
| GET | `/admin/metrics/traces` | LLM trace data (→ LangFuse) |

### Auth Strategy

- **Separate admin auth flow** — admin API keys stored in DB, distinct from player JWTs
- Role levels: `builder` (rooms/mobs/objects), `admin` (zones, agents, players), `research` (read-only narrative data)
- Dual auth: session token (web login) OR API key (automation/scripts)
- JWT key rotation: support `JWT_SECRET_CURRENT` + `JWT_SECRET_PREVIOUS` env vars
- Rate limit admin login: 5 attempts/minute
- CSRF protection required for cookie-based sessions. Prefer `Authorization: Bearer` header over cookies.
- Audit logging for ALL admin mutations (`pkg/audit/logger.go` exists — use it)

### Integration with Existing Middleware

Admin routes are on a **separate port/subdomain** (port 8081, admin.darkpawns.labz0rz.com) with their own TLS termination via reverse proxy (Caddy). Do NOT share CORS config with the game WebSocket.

```go
// cmd/web/main.go — separate binary, separate port
adminMux := http.NewServeMux()
adminMux.Handle("/admin/", adminRouter)
adminMux.Handle("/admin/login", loginHandler)  // get token

// Admin-specific middleware (narrower CORS, rate limited, audit logged)
handler := adminCORSLocked(
    adminRateLimit(
        web.SecurityHeaders(
            adminAuthMiddleware(
                auditLog(
                    adminMux,
                ),
            ),
        ),
    ),
)

srv := &http.Server{
    Addr:    ":8081",
    Handler: handler,
}
```

---

## 4. Frontend — React SPA

### Why React SPA (Not HTMX)

| Criterion | HTMX | React SPA |
|-----------|------|-----------|
| Complex form editing | Poor (full page swaps) | Excellent (controlled forms) |
| Drag-and-drop zone map | Impossible | Excellent (React Flow) |
| Real-time metrics updates | Partial (SSE polling) | Native (WebSocket) |
| Multi-tab editing | Browser-crutch | Natural (tabs) |
| Learning curve to extend | Medium | Medium (already in the toolchain) |
| Initial load time | Faster | Slower (code-split by route) |

This is a tool you use for hours — like a 3D editor or DAW. It deserves a real client.

### Pre-built Components

| Component | Use | Library |
|-----------|-----|---------|
| Zone map / room graph | Visual room editor, drag to connect | **React Flow** (xyflow) |
| Code/description editor | Mob descriptions, room text, Lua scripts | **Monaco Editor** |
| Data table editing | Bulk-edit rooms, mobs, objects | **AG Grid** |
| Metrics charts | Agent stats, system health | **Recharts** or **Nivo** |
| Form library | Complex inline forms | **React Hook Form** |
| Routing | Tabbed panels | **React Router** |
| API client | Type-safe API calls | **TanStack Query** |
| AI trace viewer | LLM call visualization | **LangFuse API → custom component** or embed |

### Route Structure

```
/admin/                    → Dashboard / overview
/admin/game/zones         → Zone browser
/admin/game/zones/:id     → Zone detail + room graph
/admin/game/rooms/:id     → Room editor (exits, flags, extra descr)
/admin/game/mobs          → Mob list (table)
/admin/game/mobs/:id      → Mob editor (dice HP, flags, scripts)
/admin/game/objects       → Object list (table)
/admin/game/objects/:id   → Object editor (type-aware values, applies)
/admin/game/shops         → Shop editor
/admin/game/triggers      → Lua script editor (Monaco)
/admin/ai/agents          → Agent roster
/admin/ai/agents/:id      → Agent detail / config / log
/admin/ai/narrative       → Narrative event feed
/admin/ai/conversations   → Cross-agent chat logs
/admin/ai/traces          → LLM call traces
/admin/research           → Research tools / data export
/admin/operations         → System metrics, backups, resets
/admin/webclient          → Web terminal (game client)
```

### Game Terminal (WebClient Tab)

- ANSI-to-HTML renderer with Xterm.js or custom (we parse ANSI escape codes)
- WebSocket to game port's `/ws` — same protocol the GoMud webclient uses
- Embedded as a tab in the SPA, not a separate page
- Preserves history, supports OLC commands that lack admin UI equivalents

---

## 5. AI Panel — The Differentiator

*Note: Delayed to Phase 6 — not useful until AI agents are running reliably.*

This is where Dark Pawns becomes a research platform, not just a MUD.

### Agent Roster

```
┌──────────────────────────────────────────────────────────────┐
│  Agents                           [Spawn New] [Bulk Kill]   │
├─────┬────────┬─────────┬────────┬────────┬──────────────────┤
│  #  │ Name   │ Model   │ Room   │ Status │ Last Action      │
├─────┼────────┼─────────┼────────┼────────┼──────────────────┤
│  1  │ Alara  │ sonnet  │ Temple │ Active │ 12s ago: examine │
│  2  │ Thorn  │ k2.5    │ Forest │ Idle   │ 2m ago: moved N  │
│  3  │ Nova   │ k2.6    │ Inn    │ Active │ speaking...       │
│  4  │ Rook   │ flash   │ Cave   │ Stuck  │ 5m loop: open    │
└─────┴────────┴─────────┴────────┴────────┴──────────────────┘
```

### Agent Detail / Config

```
┌──────────────────────────────────────────────────────────────┐
│ Agent: Alara                   [Reset] [Kill] [Observe]     │
├───────┬──────────────────────────────────────────────────────┤
│ Model │ claude-sonnet-4-20250514                              │
│ Temp  │ [=====o=====] 0.7                                    │
│ Mem   │ [==============================] 2148 tokens         │
│ Room  │ Temple of the Moon [↗]                                │
│ HP    │ [=================o=======] 67/100                    │
│ Inv   │ short sword, healing potion, torch                   │
├───────┴──────────────────────────────────────────────────────┤
│ Recent Actions                                                │
│ 12:04:23  examine statue → "A marble figure of..."           │
│ 12:04:15  north → enters the Temple courtyard                │
│ 12:03:58  say "The moon is full tonight"                     │
│ 12:03:40  look → sees priest, altar, torchlight              │
├──────────────────────────────────────────────────────────────┤
│ Narrative Memory (last 50 entries)                           │
│ 1. Moved through forest path from clearing to river          │
│ 2. Encountered wolf pack — defended successfully             │
│ 3. Met traveling merchant, traded herbs for potion           │
│ ...                                                          │
└──────────────────────────────────────────────────────────────┘
```

### Narrative Timeline

```
┌──────────────────────────────────────────────────────────────┐
│ Narrative Timeline                     [Filter] [Export]    │
├──────┬──────────┬─────────┬──────────┬───────────────────────┤
│ Time │ Agent    │ Event   │ Location │ Description           │
├──────┼──────────┼─────────┼──────────┼───────────────────────┤
│ 12:04│ Alara    │ perceive│ Temple   │ examines the statue   │
│ 12:04│ Alara    │ move    │ Courtyard│ moves north           │
│ 12:03│ Alara    │ speak   │ Temple   │ "The moon is full..." │
│ 12:03│ Thorn    │ move    │ Forest   │ heads east            │
│ 12:02│ Nova     │ attack  │ Inn      │ swings at the guard   │
│ 12:02│ System   │ reset   │ Cave     │ mobs respawned        │
└──────┴──────────┴─────────┴──────────┴───────────────────────┘
```

---

## 6. Operations Panel

- **System metrics** — memory, goroutines, WebSocket connections, uptime (via existing Prometheus endpoint)
- **Game log** — live tail of server logs with filtering
- **Zone reset control** — manual trigger, schedule override
- **Backup management** — trigger world state save, view backup history
- **Player list** — connected players, recent logins
- **Connection stats** — active WebSocket connections, message rates

---

## 7. Implementation Phases

### Phase 0 — Persistence Layer (Prerequisite)
- [ ] Build DB write path for world entities (rooms, mobs, objects, zones, shops)
- [ ] World file serialization (write world data to file format or PostgreSQL)
- [ ] Write admin mutations through ZoneDispatcher for zone-affecting changes
- [ ] Use SnapshotManager for read endpoints (lock-free reads)
- [ ] Audit logging for all mutations (`pkg/audit/logger.go` exists — wire it)

### Phase 1 — Foundation (Week 1)
- [ ] Create `pkg/admin/` package with REST handler structure
- [ ] Admin auth: API key storage in DB, role-based middleware, key rotation support
- [ ] Rate limiter for admin endpoints
- [ ] Admin-specific CORS config (locked to SPA origin, separate from game)
- [ ] Set up React SPA with Vite + TypeScript + React Router
- [ ] API client layer (TanStack Query)
- [ ] `/admin/game/zones` list + detail (read-only) — validates the full stack

### Phase 2 — Web Terminal (Week 1-2)
*Easiest win, validates full SPA + Go WebSocket integration.*
- [ ] WebSocket connection to game port's `/ws`
- [ ] ANSI-to-HTML renderer (Xterm.js or custom)
- [ ] Command history, input buffer
- [ ] Tab integration in SPA

### Phase 3 — Read-Only Viewers (Week 2)
- [ ] Zone browser (table + detail)
- [ ] Room viewer (exits, flags, extra descriptions)
- [ ] Mob viewer (stats, flags, scripts, dice notation)
- [ ] Object viewer (type-aware value fields, wear, affects)
- [ ] Shop viewer (multi-room, producing array, buy types)
- [ ] Trigger viewer (Lua script display)

### Phase 4 — Game Editors (Weeks 3-5)
- **4a:** Room + zone editor (highest value, most commonly used)
  - Exits (directional with flags: door, locked, secret, etc.)
  - Extra descriptions (keyword → description pairs)
  - 28 room flags as checkbox/toggle groups
- **4b:** Mob editor
  - Dice-based HP/damage (NdS + offset, not flat numbers)
  - 25 mob flags, combat stats, scripts, position tracking
- **4c:** Object editor
  - 24 item types with contextual value field labels (weapon: dice/size/type; container: capacity/locktype/key; etc.)
  - 30 apply types (STR +2, DEX -1, saves, regen, etc.)
  - 29 item flags, wear positions vs wear flags
  - Extra descriptions
- **4d:** Zone reset editor (most complex editor)
  - Commands: M/O/G/E/P/D/R/L(loop)/S(stop)/*(comment)
  - `if_flag` conditional chaining — visual dependency chain
  - Loop start/end pairs (Dark Pawns extension)
- **4e:** Lua script/trigger editor
  - Monaco with Lua syntax highlighting
  - Custom global API surface documentation
  - All trigger types: oncmd, ongive, sound, fight, greet, ondeath, bribe, onpulse
- **4f:** Shop editor (least urgent, lower priority)

### Phase 5 — Operations Panel (Weeks 5-6)
- [ ] System metrics dashboard
- [ ] Live game log tail with filtering
- [ ] Zone reset control (manual trigger + schedule override)
- [ ] Player list (connected players, stats, recent logins)
- [ ] Backup management (trigger save, view history)

### Phase 6 — AI & Research Panel (Post-Agent-Infrastructure)
*Not useful until AI agents are running reliably.*
- [ ] Agent roster (read + kill/spawn)
- [ ] Agent detail + config editing
- [ ] Agent action log streaming
- [ ] Narrative memory viewer
- [ ] Narrative timeline feed
- [ ] LLM trace integration (LangFuse self-host, embed or API)
- [ ] Conversation viewer (cross-agent)
- [ ] Data export (JSON/CSV)
- [ ] Agent analytics dashboard

### Phase 7 — Polish (Ongoing)
- [ ] Mobile responsive layout
- [ ] Keyboard shortcuts
- [ ] Dark/light theme
- [ ] Offline-capable editing with sync

---

## 8. Port Plan Impact

**Waves 8-10 of PORT-PLAN.md are replaced, not ported.** The web admin is the replacement for:
- Wave 8: OLC framework, text editor, Lua editor (~1,608 lines C)
- Wave 9: Room + object editors (~2,642 lines C)
- Wave 10: Mob + shop + zone editors (~3,580 lines C)

That's **~7,800 lines of C we don't need to port.** The data model knowledge from those files still feeds the admin editors, but the in-game teletype OLC commands are dead code.

---

## 9. Security Constraints (Opus Review)

1. **Separate admin binary + port.** Game on 8080, admin on 8081. Different CORS, different auth, different TLS.
2. **No CSRF.** `Authorization: Bearer` header preferred over cookies. If cookies are used, CSRF tokens are mandatory.
3. **JWT key rotation.** Support current + previous secret env vars.
4. **Rate limit admin login.** 5 attempts/minute. Empty `pkg/auth/ratelimit.go` needs implementation.
5. **Input validation.** All admin mutations need server-side sanitization. Especially Lua scripts — stored Lua is code execution. Sandbox enforcement at write time.
6. **Audit logging.** Every admin mutation logged via existing `pkg/audit/logger.go`.
7. **IP allowlisting.** Configurable whitelist for admin endpoints.
8. **No shared CORS.** Game WebSocket CORS is wide open (`*`). Admin CORS is locked to SPA origin only.
9. **TLS at reverse proxy** (Caddy/nginx), not in Go server directly.

---

## 10. Data Model Constraints (from Original C)

### Entity Structure
- **Extra descriptions** (`extra_descr_data`): keyword → description pairs on rooms, mobs, and objects. Used for "examine statue." All editors must handle these.
- **Object values are type-dependent:** 4 generic `value[]` fields whose meaning changes across 24 item types (weapon: dice/size/type, container: capacity/locktype/key-vnum). Editor must show contextual labels per type.
- **Mob combat stats are dice-based:** HP = `num_dice`d`size_dice` + `add_hp`. Damage = NdS + damroll. Editor must present dice notation, not flat integers.
- **Apply flags on objects:** 30 apply types (STR +2, DEX -1, saves, HP regen, etc.). Object editor needs apply-list UI.
- **Wear positions vs wear flags:** Equipment slots (WEAR_LIGHT through WEAR_WIELD) vs bitflags for where an item CAN be worn. Editor must handle both.
- **All bitfields:** 28 room flags, 25 mob flags, 37 affect flags, 29 item flags. Editors need checkbox/toggle arrays for all of these.

### Zone Resets
- Reset commands: M, O, G, E, P, D, R, L (loop), S (stop), * (comment)
- Each command has `if_flag` conditional chaining — commands only execute if the previous succeeded
- The `L` (loop) command is a Dark Pawns extension not in stock CircleMUD. Must support loop start/end pairs.
- Editor needs a visual dependency chain, not a flat table

### Shops
- `shop_data.producing`: `int *producing` — NULL-terminated array of object vnums the shop infinitely restocks. This is an array, not a table.
- `shop_data.type`: buy-type list — what the shop purchases, by item type + keyword
- `shop_data.in_room`: `int *in_room` — shops can operate in MULTIPLE rooms. Go code doesn't model this yet.

### Scripts / Triggers
- Lua scripting is deeply wired. 1300+ lines in C (`scripts.c`), full Lua editor (`luaedit.c`). Go has `pkg/scripting/engine.go` with Lua bridge.
- Trigger editor is a Lua script editor. Monaco with Lua syntax highlighting + custom global API surface documentation.
- All trigger types: `oncmd`, `ongive`, `sound`, `fight`, `greet`, `ondeath`, `bribe`, `onpulse`
- Trigger system uses vnum references — editor needs vnum picker (search by vnum range)

---

## 11. Tech Stack Summary

| Layer | Technology | Status |
|-------|-----------|--------|
| Backend language | Go (existing) | Done |
| HTTP routing | `net/http` + `gorilla/mux` or `chi` | New |
| Admin auth | JWT (separate from game auth, key rotation) | New |
| DB | PostgreSQL (existing) | Done |
| Admin port | 8081 (separate from game 8080) | New |
| TLS | Caddy reverse proxy | New |
| Frontend framework | React 18+ with TypeScript | New |
| Build tool | Vite | New |
| State / API | TanStack Query | New |
| Forms | React Hook Form | New |
| Room graph | React Flow (xyflow) | New |
| Code editor | Monaco (Lua + text) | New |
| Data tables | AG Grid | New |
| Charts | Recharts / Nivo | New |
| AI traces | LangFuse API (self-hosted) | New |
| Terminal | Xterm.js or custom ANSI→HTML | New |
| Styling | Tailwind CSS or styled-components | New |
