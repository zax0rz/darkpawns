# Dark Pawns: Web Admin & Client Architecture Plan (Revised)

**Status:** Phases 0-7 COMPLETE (2026-05-14)  
**Original:** BRENDA69, Opus-reviewed (2026-04-24)  
**Revised by:** Daeron  
**What changed:** Routing, port model, data structures, phase ordering, JWT role model, World write API gaps, agent integration architecture — all updated to match actual codebase state.

---

## 0. Implementation Status (2026-05-14)

All 7 phases are complete. The admin panel is live on port 4350 at `/admin/`.

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 0: Admin API Foundation | ✅ Complete | `pkg/admin/` router, JWT roles, login, CORS, rate limiting, audit |
| Phase 1: React SPA Scaffold | ✅ Complete | Vite + React + TypeScript + Tailwind v4, auth, routing |
| Phase 2: Web Terminal Tab | ✅ Complete | Xterm.js embedded, WebSocket, status bar |
| Phase 3: Read-Only Viewers | ✅ Complete | Zones, mobs, objects, rooms, shops, players, server status |
| Phase 4: Game Editors | ✅ Complete | 28 world write methods, PUT handlers for rooms/mobs/objects/shops/zones |
| Phase 5: Operations Panel | ✅ Complete | Player detail, metrics, save-world, reset-all-zones, log viewer |
| Phase 6: AI & Research Panel | ✅ Complete | Agent cards, findings feed, triage summaries, dashboard integration |
| Phase 7: Polish | ✅ Complete | Login flow, error states, responsive layout verified |

### Scope Decisions

Some endpoints from the original spec were deferred as not needed for initial launch:

| Endpoint | Status | Rationale |
|----------|--------|-----------|
| `POST /admin/restart` | Deferred | Server restart via systemd is sufficient |
| `POST /admin/bans`, `DELETE /admin/bans/:id` | Deferred | Moderation system exists in-game; web bans future work |
| `POST /admin/trigger/*` | Deferred | Agent triggering via OpenClaw cron is sufficient |
| `GET /admin/agents/:id/memory` | Deferred | Agent memory is internal; future narrative viewer |
| `GET /admin/narrative` | Deferred | Cross-agent event feed — future enhancement |
| `GET /admin/research/export` | Deferred | Research export via RESEARCH-LOG.md for now |
| Lua script editor (Monaco) | Deferred | In-game OLC is sufficient for now |
| AG Grid inline editing | Deferred | Standard edit pages work fine |
| Zone graph (React Flow) | Deferred | Future visualization enhancement |

### What's Built

**Backend (pkg/admin/):** router.go, handlers.go, login.go, log_buffer.go, agent_store.go  
**Frontend (admin-ui/):** 17 pages, 8 components, API client with 25+ endpoints  
**World writes (pkg/game/world_write.go):** 28 methods for rooms, mobs, objects, shops, zones

---

## 1. What Actually Exists (as of 2026-05-14)

### Server Binary (`cmd/server/main.go`)

Single binary. Not two. The server handles game traffic, WebSocket, API, and static web assets all in one process. Key facts:

| Component | Reality |
|-----------|---------|
| Routing | `net/http` + `http.NewServeMux()` — no gorilla/mux, no chi |
| Game port | Configurable via `-port` flag (default `4350`) |
| WebSocket | `/ws` — JSON-RPC game protocol |
| API | `/api/` — auth-protected, currently just OpenAPI spec + 404 stub |
| Health | `/health` — plain `OK` |
| Metrics | `/metrics` — Prometheus handler |
| Static | `/` — serves web client files if `-web` flag set |
| TLS | Auto-detect from `TLS_CERT_FILE`/`TLS_KEY_FILE` env vars |

### Middleware Stack (`web/`)

Already built, already wired:

| File | Function | Status |
|------|----------|--------|
| `web/auth.go` | `AuthMiddleware` — JWT Bearer token validation | ✅ Wired to `/api/` |
| `web/cors.go` | `CORSMiddleware` — configurable origins, dev mode | ✅ Exists, not currently in chain |
| `web/security.go` | `SecurityHeaders` — CSP, HSTS, X-Frame-Options | ✅ Wired to all routes |
| `pkg/auth/jwt.go` | JWT creation/validation | ✅ Used by WebSocket + API |
| `pkg/auth/ratelimit.go` | IP-based rate limiter + login attempt tracker | ✅ Exists |
| `pkg/audit/logger.go` | Structured audit event logging (append-only file) | ✅ Exists, not yet wired to admin |

### World Read Methods (`pkg/game/`)

These exist and work — the admin API can call them directly:

| Method | Returns | Package |
|--------|---------|---------|
| `World.GetRoomInWorld(vnum)` | `*parser.Room` | `pkg/game/world.go` |
| `World.GetMobPrototype(vnum)` | ``parser.Mob, bool` | `pkg/game/world_zone.go` |
| `World.GetObjPrototype(vnum)` | `*parser.Obj, bool` | `pkg/game/world_zone.go` |
| `World.GetZone(number)` | `*parser.Zone, bool` | `pkg/game/world_zone.go` |
| `World.GetAllZones()` | `[]*parser.Zone` | `pkg/game/world_zone.go` |
| `World.GetShopByKeeper(vnum)` | `*Shop, bool` | `pkg/game/world_zone.go` |
| `World.GetAllObjects()` | `[]*ObjectInstance` | `pkg/game/world.go` |
| `World.GetAllMobs()` | `[]*MobInstance` | `pkg/game/world.go` |
| `World.GetPlayersInRoom(vnum)` | `[]*Player` | `pkg/game/world.go` |
| `World.GetMobsInRoom(vnum)` | `[]*MobInstance` | `pkg/game/world.go` |
| `World.GetItemsInRoom(vnum)` | `[]*ObjectInstance` | `pkg/game/world_object.go` |
| `World.GetRoomCount()` | `int` | `pkg/game/world.go` |

### World Data Structures (`pkg/parser/`)

```go
// parser.Room — 10,057 instances
type Room struct {
    VNum, Name, Description, Zone, Flags []string, Sector int
    Exits map[string]Exit, ExtraDescs []ExtraDesc
    ScriptName string, ScriptFunctions int
}

// parser.Mob — 1,319 prototypes
type Mob struct {
    VNum, Keywords, ShortDesc, LongDesc, DetailedDesc string
    ActionFlags, AffectFlags []string
    Alignment, Race, Level, THAC0, AC int
    HP, Damage DiceRoll
    Gold, Exp, Position, DefaultPos, Sex, Weight, Height int
    Str/StrAdd/Int/Wis/Dex/Con/Cha int
    ScriptName string, LuaFunctions int
}

// parser.Obj — 1,661 prototypes
type Obj struct {
    VNum, Keywords, ShortDesc, LongDesc, ActionDesc string
    TypeFlag int, ExtraFlags [4]int, WearFlags [4]int
    Values [4]int, Weight, Cost int, LoadPercent float64
    Affects []ObjAffect, ExtraDescs []ExtraDesc
    ScriptName string, LuaFunctions int
}

// parser.Zone — 95 zones
type Zone struct {
    Number, Name, TopRoom, Lifespan, ResetMode int
    Commands []ZoneCommand
}

// game.Shop — loaded from shop files
type Shop struct {
    ID, KeeperVNum int
    BuyTypes, SellTypes []int
    ProfitBuy, ProfitSell float64
    Flags, RoomVNum int, KeeperName string
}
```

### Web Client (`web/`)

Partially built:

| File | Purpose |
|------|---------|
| `web/index.html` | Landing page with embedded terminal |
| `web/client.js` | Xterm.js WebSocket client (7.3KB) |
| `web/style.css` | Dark fantasy theme (5KB) |
| `web/onboarding/` | Agent onboarding docs |
| `web/api/openapi.json` | Agent API spec (WebSocket JSON-RPC) |

### Concurrency Model

The World struct uses `sync.RWMutex` for concurrent access. ZoneDispatcher exists (`pkg/game/zone_dispatcher.go`) for per-zone goroutine processing. There is no `SnapshotManager` or `atomic.Pointer` — the spec's assumed concurrency model was aspirational, not implemented.

### Database

PostgreSQL is optional. If connection fails, server runs without persistence. The `db` package provides `DB` with `SQLDB()` accessor. Session manager holds the DB reference.

---

## 2. Architecture (Revised)

```
┌──────────────────────────────────────────────────────────────┐
│                     BROWSER (React SPA)                      │
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │   Game   │ │   AI     │ │ Research │ │  Operations  │   │
│  │  Panel   │ │  Panel   │ │  Panel   │ │    Panel     │   │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘   │
│       │            │            │               │            │
│  ┌────┴────────────┴────────────┴───────────────┴──────┐    │
│  │           WebClient (Xterm.js, existing)             │    │
│  └───────────────────────┬──────────────────────────────┘    │
└──────────────────────────┼───────────────────────────────────┘
                           │
            ┌──────────────┴──────────────┐
            │          TLS (Caddy)         │
            └──────────────┬──────────────┘
                           │
┌──────────────────────────┼───────────────────────────────────┐
│                 DARK PAWNS GO SERVER (single binary)          │
│                          port 4350                            │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              Middleware Chain (existing)                │  │
│  │  SecurityHeaders → CORS → Auth → Audit                │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                               │
│  ┌──────────┐ ┌──────────────┐ ┌───────────────────────┐    │
│  │  /ws     │ │ /api/*       │ │ /admin/*              │    │
│  │  Game WS │ │ Agent API    │ │ Admin REST API        │    │
│  │  (exist) │ │ (auth, stub) │ │ (NEW)                 │    │
│  └──────────┘ └──────────────┘ └───────────┬───────────┘    │
│                                             │                │
│  ┌──────────────────────────────────────────┴─────────────┐  │
│  │              World (sync.RWMutex)                       │  │
│  │  Rooms │ Mob Prototypes │ Obj Prototypes │ Zones       │  │
│  │  Shops │ Players │ Mob Instances │ Object Instances     │  │
│  │  ZoneDispatcher │ EventQueue │ ScriptEngine             │  │
│  └────────────────────────────────────────────────────────┘  │
│                          │                                    │
│  ┌───────────────────────┴─────────────┐ ┌───────────────┐  │
│  │         PostgreSQL (optional)       │ │  Narrative    │  │
│  │  Players │ Saves │ Admin keys │Mod  │ │  Memory       │  │
│  └─────────────────────────────────────┘ └───────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### Key Architecture Decision: Same Port, `/admin/` Prefix

The original spec proposed a separate port (8081) and separate binary. **Reality says no.** Here's why:

1. The server is already a single binary on a single port.
2. Splitting to two ports means two TLS certs, two Caddy routes, two deploy targets.
3. The admin panel needs the same `World` object the game uses — sharing a process is simpler than IPC.
4. CORS can be locked to `/admin/` paths without a separate port.

**Decision:** Admin routes live at `/admin/*` on the same port (4350). CORS is locked to the admin SPA origin. No separate binary. If we ever need to split, we can do it later without changing the API contract.

---

## 3. Backend — Admin REST API (`pkg/admin/`)

### New Package: `pkg/admin/`

A new Go package that registers admin HTTP handlers. The server's `main.go` imports it and mounts the router.

```go
// pkg/admin/router.go
func NewRouter(world *game.World, mgr *session.Manager, audit *audit.AuditLogger) http.Handler {
    r := http.NewServeMux()
    // ... register routes ...
    return r
}
```

### Route Design

All routes prefixed `/admin/`. Auth required (admin JWT or API key).

**World Entity Routes (read + write):**

| Method | Route | Purpose | World Method |
|--------|-------|---------|--------------|
| GET | `/admin/zones` | List all zones | `GetAllZones()` |
| GET | `/admin/zones/:id` | Zone detail + commands | `GetZone(id)` |
| PUT | `/admin/zones/:id` | Update zone metadata | Zone write |
| GET | `/admin/zones/:id/rooms` | Rooms in zone | Filter `GetRoomInWorld` by zone |
| GET | `/admin/rooms/:vnum` | Room detail | `GetRoomInWorld(vnum)` |
| PUT | `/admin/rooms/:vnum` | Update room | Room write |
| GET | `/admin/mobs` | List mob prototypes | `GetAllMobs()` |
| GET | `/admin/mobs/:vnum` | Mob detail | `GetMobPrototype(vnum)` |
| PUT | `/admin/mobs/:vnum` | Update mob | Mob write |
| GET | `/admin/objects` | List obj prototypes | `GetAllObjects()` |
| GET | `/admin/objects/:vnum` | Object detail | `GetObjPrototype(vnum)` |
| PUT | `/admin/objects/:vnum` | Update object | Obj write |
| GET | `/admin/shops` | List shops | `GetShopManager()` |
| GET | `/admin/shops/:id` | Shop detail | `GetShopByKeeper(vnum)` |
| PUT | `/admin/shops/:id` | Update shop | Shop write |

**Live State Routes (read-only):**

| Method | Route | Purpose |
|--------|-------|---------|
| GET | `/admin/players` | Online players + stats |
| GET | `/admin/players/:name` | Player detail (session state) |
| GET | `/admin/rooms/:vnum/players` | Players in room |
| GET | `/admin/rooms/:vnum/mobs` | Mobs in room |
| GET | `/admin/rooms/:vnum/items` | Items in room |

**Operations Routes:**

| Method | Route | Purpose |
|--------|-------|---------|
| POST | `/admin/zones/:id/reset` | Trigger zone reset |
| POST | `/admin/restart` | Graceful server restart |
| GET | `/admin/server` | Server status (uptime, memory, connections) |
| GET | `/admin/logs` | Tail server log |
| POST | `/admin/bans` | Ban player/IP |
| DELETE | `/admin/bans/:id` | Unban |

**AI Agent Routes (Phase 6):**

| Method | Route | Purpose |
|--------|-------|---------|
| GET | `/admin/agents` | List active agents |
| GET | `/admin/agents/:id` | Agent detail + config |
| PUT | `/admin/agents/:id/config` | Update agent params |
| GET | `/admin/agents/:id/memory` | View narrative memory |
| GET | `/admin/narrative` | Cross-agent event feed |
| GET | `/admin/research/export` | Export interaction data |

### Auth Strategy

Reuse existing infrastructure:

- `web.AuthMiddleware` validates JWT Bearer tokens — admin routes use the same middleware
- Add role claim to JWT: `{ player_name, is_agent, agent_key_id, role }` where role is `player`, `builder`, `admin`, or `research`
- Admin login endpoint: `POST /admin/login` — validates credentials, returns admin-scoped JWT
- API key fallback: `X-Admin-Key` header for automation (stored in DB, hashed)
- Rate limit: reuse `pkg/auth/ratelimit.go` IP rate limiter
- Audit: wire `pkg/audit/logger.go` to log ALL admin mutations

**⚠️ Codebase correction:** The current `auth.Claims` struct is `{ player_name, is_agent, agent_key_id }`. Phase 0 must add a `Role string` field. Existing tokens will have empty role → defaults to `player`. The `RequireRole()` middleware enforces:

```go
// Admin routes require role >= "builder". Empty role = player = no admin access.
var roleHierarchy = map[string]int{"player": 0, "research": 1, "builder": 2, "admin": 3}

func RequireRole(role string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims, ok := auth.GetClaimsFromContext(r.Context())
        if !ok || roleHierarchy[claims.Role] < roleHierarchy[role] {
            http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### CORS Strategy

Admin CORS locked to SPA origin only. Development mode more permissive.

```go
// CORS for /admin/ — locked to admin.darkpawns.labz0rz.com
adminCORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin == "https://admin.darkpawns.labz0rz.com" || 
           (isDevMode() && strings.HasPrefix(origin, "http://localhost:")) {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }
        // ... preflight handling ...
        next.ServeHTTP(w, r)
    })
}
```

### Mutation Logging

Every admin write operation goes through an audit wrapper:

```go
func auditMutation(auditLogger *audit.AuditLogger, action string, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user, _ := web.GetPlayerNameFromContext(r)
        auditLogger.Log(audit.AuditEvent{
            EventType: "admin_mutation",
            User:      user,
            IPAddress: r.RemoteAddr,
            Action:    action,
            Details:   fmt.Sprintf("%s %s", r.Method, r.URL.Path),
        })
        next(w, r)
    }
}
```

---

## 4. Frontend — React SPA

### Stack (from original spec, unchanged)

| Tool | Purpose |
|------|---------|
| React 18+ / TypeScript | Framework |
| Vite | Build tool |
| TanStack Query | API state / caching |
| React Router | Tabbed panels |
| React Hook Form | Complex forms |
| React Flow | Zone map / room graph |
| Monaco Editor | Lua script editing |
| AG Grid | Bulk data tables |
| Recharts | Metrics charts |
| Xterm.js | Web terminal (existing, embed) |
| Tailwind CSS | Styling |

### Why React SPA (unchanged from original)

The admin panel is a tool you use for hours. Complex form editing, drag-and-drop zone maps, real-time metrics, multi-tab editing — these need a real client, not HTMX.

### Route Structure

```
/admin/                      → Dashboard (overview)
/admin/game/zones            → Zone browser (AG Grid)
/admin/game/zones/:id        → Zone detail + room graph (React Flow)
/admin/game/rooms/:vnum      → Room editor (exits, flags, extra descr)
/admin/game/mobs             → Mob list (AG Grid)
/admin/game/mobs/:vnum       → Mob editor (dice HP, flags, scripts)
/admin/game/objects          → Object list (AG Grid)
/admin/game/objects/:vnum    → Object editor (type-aware values, applies)
/admin/game/shops            → Shop editor
/admin/game/triggers         → Lua script editor (Monaco)
/admin/ai/agents             → Agent roster (Phase 6)
/admin/ai/agents/:id         → Agent detail / config / log
/admin/ai/narrative          → Narrative event feed
/admin/ai/conversations      → Cross-agent chat logs
/admin/operations            → System metrics, backups, resets
/admin/webclient             → Web terminal (Xterm.js, existing)
```

### Web Terminal Integration

The existing `web/client.js` (Xterm.js + WebSocket) gets embedded as a tab in the SPA, not a separate page. The terminal preserves its existing behavior — it's just one panel among many.

---

## 5. Implementation Phases (Revised)

### Phase 0: Admin API Foundation
**Time:** 1 day  
**Depends on:** Nothing  
**Delivers:** `/admin/` routes with auth, first read-only endpoint

1. Create `pkg/admin/` package with router
2. Add role claim to JWT system (`pkg/auth/jwt.go`)
3. Wire admin routes in `cmd/server/main.go`
4. Implement `GET /admin/zones` (read-only, uses `World.GetAllZones()`)
5. Add admin CORS middleware to chain
6. Wire audit logger to admin mutations

### Phase 1: React SPA Scaffold
**Time:** 1-2 days  
**Depends on:** Phase 0  
**Delivers:** SPA shell with routing, auth, first data view

1. `create vite@latest` with React + TypeScript + Tailwind
2. TanStack Query setup with API client
3. Auth flow (login form → admin JWT → stored in memory)
4. React Router with panel tabs
5. Zone list page (first real data from API)
6. Dev proxy to game server port

### Phase 2: Web Terminal Tab
**Time:** 0.5 day  
**Depends on:** Phase 1  
**Delivers:** Xterm.js embedded in SPA

1. Embed existing `client.js` logic into a React component
2. WebSocket connection to `/ws`
3. Tab in the SPA alongside other panels
4. Status bar (HP/Mana/Move) from existing client.js state

### Phase 3: Read-Only Viewers
**Time:** 2-3 days  
**Depends on:** Phase 1  
**Delivers:** Browse all world entities

1. Zone detail (room list, reset commands)
2. Room viewer (exits, flags, extra descriptions)
3. Mob list + detail (dice HP, flags, scripts, affects)
4. Object list + detail (type-aware values, wear flags, applies)
5. Shop viewer (buy types, profit margins, keeper)
6. Player list (online, with room/level/class)
7. Server status (uptime, memory, connections)

### Phase 4: Game Editors
**Time:** 5-7 days (revised — write API is new work)  
**Depends on:** Phase 3  
**Delivers:** Write world entities from the browser

**⚠️ Codebase correction:** The World struct has limited write methods. The spec assumed existing write APIs — reality says otherwise. Phase 4 includes building the World write surface:

| Existing Write Methods | Missing (must build) |
|------------------------|---------------------|
| `SetObjectExtraDesc()` | `SetRoomName()`, `SetRoomDescription()` |
| `SetObjectExtraFlag()` | `SetRoomFlags()`, `SetExitDoorState()` (exists) |
| `SetExitDoorState()` | `SetMobShortDesc()`, `SetMobLongDesc()` |
| | `SetMobStats()` (HP, level, AC, THAC0) |
| | `SetMobFlags()`, `SetMobAffectFlags()` |
| | `SetObjShortDesc()`, `SetObjLongDesc()` |
| | `SetObjValues()` (type-dependent), `SetObjCost()` |
| | `SetObjWearFlags()`, `SetObjExtraFlags()` |
| | `SetZoneLifespan()`, `SetZoneResetMode()` |
| | `AddZoneCommand()`, `RemoveZoneCommand()` |

All writes must hold `World.mu` (write lock). The admin API handlers call these methods. Consider a `world_write.go` file to keep write methods separate from read logic.

1. Room editor (name, description, exits, flags, extra descr)
2. Mob editor (keywords, descriptions, dice HP, stats, flags, scripts)
3. Object editor (type selector → contextual value fields, applies, wear flags)
4. Shop editor (buy types, profit margins, producing list)
5. Zone reset editor (visual command chain with if_flag dependencies)
6. Lua trigger editor (Monaco with Dark Pawns API docs autocomplete)
7. Bulk operations (AG Grid inline editing + batch save)

**Data model constraints (from C source):**
- Object `Values[4]` meaning changes per TypeFlag (weapon: dice/size/type, container: capacity/locktype/key)
- Mob HP is dice: `num_dice`d`size_dice` + `add_hp`
- 28 room flags, 25 mob flags, 37 affect flags, 29 item flags — all need checkbox arrays
- Extra descriptions: keyword → description pairs on rooms, mobs, objects
- Zone reset commands: M/O/G/E/P/D/R/L/S/* with if_flag chaining

### Phase 5: Operations Panel
**Time:** 2-3 days  
**Depends on:** Phase 3  
**Delivers:** Live server monitoring + control

1. Live log tail (WebSocket to server log)
2. Zone reset trigger (manual reset button per zone)
3. Player management (kick, ban, mute — via existing moderation system)
4. Server metrics (Prometheus `/metrics` → Recharts)
5. Backup trigger (save world state)

### Phase 6: AI & Research Panel
**Time:** 3-5 days  
**Depends on:** Phase 3 + AI agents deployed on domain-expansion  
**Delivers:** Agent observability + research data export

**⚠️ Dependency note:** The agent CLI (`dp-agent`) and dreaming layer exist in the repo but are not yet running on domain-expansion. Phase 6 can build the UI with mock data and wire to live agents later. The admin API endpoints for agent status should return graceful empty states when agents aren't reporting in.

The AI systems are already built (memory, dreaming, agent CLI, dp-agent). This phase surfaces them in the admin UI. **See Section 6 (Agent Integration Architecture) for how Daeron and Reek wire into the admin panel.**

1. Agent roster (list active agents, model, status, room)
2. Agent detail (config, current state, combat log)
3. Narrative memory viewer (per-agent memory graph)
4. Cross-agent event feed (kill/death/give/say/etc.)
5. LLM trace viewer (API calls, latency, tokens, cost)
6. Research data export (JSON/CSV interaction datasets)
7. Dreaming layer status (last consolidation, memory summary)

### Phase 7: Polish
**Time:** 2-3 days  
**Depends on:** All previous phases  
**Delivers:** Production-ready admin panel

1. Responsive layout (breakpoints for tablet/desktop)
2. Keyboard shortcuts (Ctrl+K command palette)
3. Dark/light theme toggle
4. Error boundaries + loading states
5. WebSocket reconnection for terminal + logs
6. Role-based UI hiding (builder can't see agent config)

---

## 6. Agent Integration Architecture (New)

The admin panel isn't just a web UI — it's the operational surface for Daeron and Reek. The agents are API clients: they write to the admin API (findings, status, triage results) and the SPA reads from it. The SPA can also trigger agent work via webhooks.

### The Model

```
┌─────────────────────────────────────────────────────┐
│                  ADMIN PANEL (SPA)                   │
│  Dashboard │ Findings │ Triage │ Ops │ Agent Status  │
└────────┬──────────────────────────────┬──────────────┘
         │ reads                        │ triggers
         ▼                              ▼
┌──────────────────────┐    ┌─────────────────────────┐
│    Admin REST API     │    │   Webhook Endpoints      │
│  /admin/agents/*      │    │  POST /admin/trigger/*   │
│  /admin/findings/*    │    │                          │
│  /admin/triage/*      │    │                          │
└───────┬──────────────┘    └────────┬────────────────┘
        │ writes                     │ enqueues
        ▼                            ▼
┌──────────────────────────────────────────────────────┐
│              OpenClaw Gateway (mac-mini)              │
│  Cron │ Standing Orders │ Hooks │ Task Flow │ Tasks   │
└──────┬──────────────────────────┬────────────────────┘
       │ executes                 │ executes
       ▼                          ▼
┌──────────────┐          ┌──────────────────┐
│   DAERON     │          │      REEK        │
│  (MiMo v2.5) │          │  (DeepSeek V4)   │
│  Loremaster  │          │  Code Crawler    │
└──────────────┘          └──────────────────┘
```

### How Daeron Wires In

Daeron is the admin operator. Standing orders in AGENTS.md already define programs (Morning Triage, Weekly Digest, Dependency Audit, Server Monitoring). The admin API gives programmatic access to the game world for these programs.

| Standing Order | Admin API Usage | Automation Mechanism |
|----------------|-----------------|----------------------|
| Morning Triage | Read Linear issues → verify against code → write triage comments | Cron (7:30 AM) + Linear MCP |
| Weekly Digest | Read Reek reports, findings tracker, git log → write digest | Cron (Sunday 6 PM) |
| Server Monitoring | `GET /admin/server`, `GET /admin/logs` → check health | Heartbeat cycle |
| Dependency Audit | Read go.mod, check versions → update + verify | Cron (1st Monday) |
| Research Writing | Read codebase, write to RESEARCH-LOG.md | Cron (biweekly Tuesday) |

**Daeron writes to the admin API:**
- `POST /admin/agents/daeron/status` — current operation, last run time
- `POST /admin/findings` — triaged findings (confirmed/rejected/escalated)
- `POST /admin/triage/summary` — daily triage summary for dashboard

**Daeron reads from the admin API:**
- `GET /admin/server` — server health for monitoring checks
- `GET /admin/rooms/:vnum` — verify room state during triage
- `GET /admin/mobs/:vnum` — verify mob state during triage
- `GET /admin/logs` — check for errors during monitoring

### How Reek Wires In

Reek is the code crawler, expanded role. Officially on DeepSeek V4 Flash. Reek's primary job is code review, but with admin API access, he can also do server health checks and log analysis.

| Task | Admin API Usage | Automation Mechanism |
|------|-----------------|----------------------|
| Nightly Code Crawl | Read source via workspace files | Cron (3 AM) |
| Log Analysis | `GET /admin/logs` → scan for patterns | Cron (post-crawl) |
| Server Health Check | `GET /admin/server`, `GET /admin/players` | Cron (post-crawl) |
| Findings Report | `POST /admin/findings` → write findings to admin API | Cron (post-crawl) |

**Reek writes to the admin API:**
- `POST /admin/findings` — code review findings (severity, file:line, description)
- `POST /admin/agents/reek/status` — last crawl time, findings count

**Reek reads from the admin API:**
- `GET /admin/logs` — server logs for error pattern detection
- `GET /admin/server` — uptime, memory, connection stats

### Admin API Endpoints for Agents

These endpoints serve both the SPA (reads) and the agents (writes). They're standard REST with JWT auth.

**Agent Self-Reporting:**

| Method | Route | Purpose | Writer |
|--------|-------|---------|--------|
| POST | `/admin/agents/:id/status` | Agent reports current status | Daeron/Reek |
| GET | `/admin/agents/:id/status` | Read agent status | SPA |
| GET | `/admin/agents` | List all agents + status | SPA |

**Findings Feed:**

| Method | Route | Purpose | Writer |
|--------|-------|---------|--------|
| POST | `/admin/findings` | Submit a finding (from Reek or Daeron triage) | Reek/Daeron |
| GET | `/admin/findings` | List findings (filterable by severity, status, source) | SPA |
| GET | `/admin/findings/:id` | Finding detail + triage history | SPA |
| PUT | `/admin/findings/:id` | Update finding status (confirm/reject/escalate) | Daeron |

**Triage Results:**

| Method | Route | Purpose | Writer |
|--------|-------|---------|--------|
| POST | `/admin/triage/summary` | Daily triage summary | Daeron |
| GET | `/admin/triage/summaries` | List triage summaries | SPA |

### Webhook Triggers (SPA → Agents)

The SPA can trigger agent work on-demand via webhook endpoints. These enqueue cron jobs or wake events in the OpenClaw Gateway.

| Method | Route | Purpose | Effect |
|--------|-------|---------|--------|
| POST | `/admin/trigger/reek-crawl` | Request on-demand code crawl | Enqueues Reek cron job |
| POST | `/admin/trigger/triage` | Request on-demand triage | Enqueues Daeron triage |
| POST | `/admin/trigger/heartbeat` | Request server health check | Enqueues Daeron heartbeat |

**Implementation:** These endpoints call the OpenClaw Gateway API (or write to a webhook URL) to trigger the appropriate cron job or wake event. The SPA shows a "triggered" confirmation; the actual work happens asynchronously.

### Dashboard Layout (Phase 1-2)

The admin dashboard surfaces agent activity at a glance:

```
┌──────────────────────────────────────────────────────────────┐
│  DARK PAWNS — ADMIN DASHBOARD                                │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │
│  │ DAERON      │ │ REEK        │ │ SERVER              │   │
│  │ 🟢 active   │ │ 🟢 active   │ │ 🟢 running          │   │
│  │ Last: 2m ago│ │ Last: 4h ago│ │ Uptime: 3d 14h      │   │
│  │ Triage: 3   │ │ Findings: 12│ │ Memory: 287MB       │   │
│  │ confirmed   │ │ 8 confirmed │ │ Players: 0          │   │
│  └─────────────┘ └─────────────┘ └─────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ RECENT FINDINGS                                      │   │
│  │ 🔴 HIGH   DP-12  nil panic in combat.go:342          │   │
│  │ 🟡 MEDIUM DP-14  unused variable in parser.go:89     │   │
│  │ 🟢 LOW    DP-15  style: duplicate nosec tag           │   │
│  │ ✅ CONFIRMED DP-11  dead code in comm_infra.go       │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ QUICK ACTIONS                                        │   │
│  │ [Trigger Reek Crawl] [Run Triage] [Zone Reset All]  │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Data Flow Summary

1. **Reek crawls** (3 AM cron) → writes findings to `POST /admin/findings`
2. **Daeron triages** (7:30 AM cron) → reads findings, verifies against code, updates status via `PUT /admin/findings/:id`
3. **SPA displays** findings feed + triage summaries on dashboard
4. **Architect uses SPA** to review findings, trigger actions, monitor server
5. **SPA triggers** on-demand work via `POST /admin/trigger/*` → webhook → OpenClaw → agent
6. **Agents report status** via `POST /admin/agents/:id/status` → SPA shows live agent state

This creates a closed loop: agents do work → admin API surfaces it → SPA displays it → Architect acts on it → SPA triggers more agent work.

---

## 7. Data Model Constraints (from C Source, Unchanged)

These are real and must be respected by the editors:

- **Object values are type-dependent:** 4 generic `value[]` fields whose meaning changes across 24 item types
- **Mob combat stats are dice-based:** HP = `num_dice`d`size_dice` + `add_hp`
- **Apply flags on objects:** 30 apply types (STR +2, DEX -1, saves, HP regen, etc.)
- **All bitfields:** 28 room flags, 25 mob flags, 37 affect flags, 29 item flags
- **Zone resets:** M/O/G/E/P/D/R/L/S/* commands with if_flag conditional chaining
- **Shops:** `producing` array (infinite restock), `buy_types` list, multi-room support
- **Lua triggers:** 8 types (oncmd, ongive, sound, fight, greet, ondeath, bribe, onpulse)

---

## 8. Security Constraints (Unchanged from Original)

1. **No CSRF.** `Authorization: Bearer` header preferred over cookies.
2. **JWT key rotation.** Support current + previous secret env vars.
3. **Rate limit admin login.** 5 attempts/minute via existing `pkg/auth/ratelimit.go`.
4. **Input validation.** All mutations server-side sanitized. Lua scripts sandboxed at write time.
5. **Audit logging.** Every mutation logged via existing `pkg/audit/logger.go`.
6. **IP allowlisting.** Configurable whitelist for admin endpoints.
7. **CORS locked.** Admin SPA origin only. No wildcards.
8. **TLS at reverse proxy.** Caddy terminates TLS, Go server runs plaintext internally.


### JWT Storage Threat Model (DP-99)

The admin SPA stores JWTs in `localStorage` (see `admin-ui/src/api/client.ts:4` and `admin-ui/src/hooks/useAuth.ts:11`). This is a standard SPA security tradeoff.

**Threat:** If an XSS vulnerability exists in the SPA, an attacker can read `localStorage` and exfiltrate the JWT. The token grants full admin API access until expiry.

**Mitigations in place:**
- React does not use `dangerouslySetInnerHTML` — XSS surface is minimal
- CSP headers on server-served content (not the SPA, which is a separate build)
- CORS locked to admin SPA origin — no cross-origin API requests
- Rate limiting on auth endpoints
- Audit logging on all mutations
- Short-lived tokens (configurable expiry)

**Risk assessment:** LOW. The admin panel is an internal tool on a private network. The XSS surface is small (no innerHTML, no user-rendered HTML). The JWT approach is consistent with industry-standard SPA patterns (used by most admin panels).

**Future consideration:** If the panel is ever exposed publicly, migrate to httpOnly cookie-based sessions. This eliminates XSS exfiltration of tokens but adds CSRF complexity. For an internal tool on a private network, localStorage is the simpler choice.

**Decision:** Accept localStorage JWT storage. Document the tradeoff. Revisit if deployment posture changes.

---

## 9. Dependencies (Revised)

### Already in go.mod (no new deps for backend)

| Package | Used For |
|---------|----------|
| `net/http` | Routing (stdlib) |
| `database/sql` + `github.com/lib/pq` | PostgreSQL |
| `github.com/golang-jwt/jwt` | JWT auth |

### New deps (frontend only)

| Package | Purpose | Size |
|---------|---------|------|
| react, react-dom | UI framework | — |
| vite | Build tool | — |
| @tanstack/react-query | API state | — |
| react-router-dom | Routing | — |
| react-hook-form | Forms | — |
| @xyflow/react | Zone map graph | ~200KB |
| @monaco-editor/react | Lua code editor | ~300KB |
| ag-grid-react | Data tables | ~200KB |
| recharts | Charts | ~150KB |
| xterm, xterm-addon-fit | Terminal (already used) | ~100KB |
| tailwindcss | Styling | — |

**No new Go dependencies required.** The entire backend is stdlib + existing packages.

---

## 10. What Changed vs Original Spec

| Original Assumption | Reality | Impact |
|---------------------|---------|--------|
| gorilla/mux or chi routing | `net/http` + `http.NewServeMux()` | Use stdlib, no new dep |
| Ports 8080/8081 (separate) | Single port 4350, `/admin/` prefix | Simpler deploy, same CORS chain |
| Separate admin binary | Single binary | No IPC needed, share World object |
| SnapshotManager (atomic.Pointer) | sync.RWMutex on World | Direct method calls |
| ZoneDispatcher for reads | Direct World methods | Simpler read path |
| AI deferred to Phase 6 | AI systems already built | Phase 6 surfaces existing data, doesn't build new infra |
| "Existing stubs" for API | OpenAPI spec + 404 stub | Build from scratch but wire into existing auth middleware |
| `pkg/admin/` doesn't exist | Confirmed — new package | Clean start |
| Audit logger doesn't exist | `pkg/audit/logger.go` exists | Wire it, don't build it |
| Rate limiter doesn't exist | `pkg/auth/ratelimit.go` exists | Wire it, don't build it |
| CORS middleware doesn't exist | `web/cors.go` exists | Wire it, don't build it |
