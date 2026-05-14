# Admin Panel Handoff — Blenda

**Date:** 2026-05-14  
**From:** Daeron + The Architect  
**Status:** Phases 0-7 complete. Agent integration partially wired.

---

## What's Built

The Dark Pawns admin panel is a React SPA served from the Go binary on port 4350 at `/admin/`. All 7 phases are complete. 161 Go tests passing with race detection.

### Backend (pkg/admin/)

| File | Purpose |
|------|---------|
| `router.go` | 20+ routes with role-based auth, rate limiting, CORS |
| `handlers.go` | Full CRUD for zones, mobs, objects, rooms, shops, players, server, logs, metrics, agents, findings, triage |
| `login.go` | JWT login endpoint |
| `log_buffer.go` | Ring buffer for log viewer |
| `agent_store.go` | In-memory store for agent statuses, findings, triage summaries |

### Frontend (admin-ui/src/)

| Page | What it does |
|------|-------------|
| `DashboardPage` | Server status, agent status card, recent findings, stats pills |
| `ZonesPage` / `ZoneDetailPage` | Zone list + detail with reset commands |
| `MobsPage` / `MobDetailPage` / `MobEditPage` | Mob browser + editor (stats, flags, keywords, position) |
| `ObjectsPage` / `ObjectDetailPage` / `ObjectEditPage` | Object browser + editor (values, flags, affects) |
| `RoomDetailPage` / `RoomEditPage` | Room viewer + editor (flags, sector) |
| `ShopEditPage` | Shop editor (buy/sell types, profit margins) |
| `OperationsPage` | Server status, online players, log viewer, metrics card, quick actions |
| `PlayerDetailModal` | Click player name → stats/inventory/equipment tabs |
| `AgentsPage` | Agent cards, findings feed (filterable), triage summaries |
| `TerminalPage` | Embedded Xterm.js WebSocket terminal |
| `LoginPage` | JWT auth with connection status, auto-redirect |

### World Write Methods (pkg/game/world_write.go)

28 methods: room name/description/flags/sector/exits/extra-descs, mob keywords/short/long/level/AC/HP/gold/exp/alignment/action flags/affect flags/6 stats/THAC0/damage/position/sex/race, object short/long/keywords/type/values/weight/cost/wear flags/extra flags/affects/extra-descs, zone lifespan/reset mode/add+remove commands, shop buy/sell types/profit.

### Tests (pkg/admin/)

- `handlers_test.go` — 160 tests: handler table tests, auth/CORS/rate-limit integration
- `world_write_test.go` — all 28 write methods, happy + edge cases
- `log_buffer_test.go` — ring buffer, overflow, thread safety
- `agent_store_test.go` — agent status, findings, triage, thread safety

---

## What's NOT Built (Agent Integration Gap)

The admin API has endpoints for agents to self-report, and the SPA displays agent data. But the agents don't actually call the API yet. The data in the agent store is seeded with static entries and resets on restart.

### The Closed Loop (from spec — not yet closed)

```
Agents do work → POST status/findings to admin API → SPA displays → Architect acts → SPA triggers more agent work
```

### What Exists

**Backend endpoints (all working):**
- `POST /admin/agents/status` — agent self-reports status
- `GET /admin/agents` — list all agents
- `POST /admin/findings` — submit a finding
- `GET /admin/findings` — list findings (filterable by status/severity/source)
- `PUT /admin/findings/:id` — update finding status (confirm/reject/fix)
- `POST /admin/triage/summaries` — submit triage summary
- `GET /admin/triage/summaries` — list triage summaries

**Frontend (all working):**
- AgentsPage: agent cards with status dots, findings feed with 3 filter dropdowns, triage summary list
- Dashboard: agent status card (30s refresh), recent findings card, stats pills

### What's Missing

1. **Agents don't self-report.** Daeron and Reek need to call `POST /admin/agents/status` after each run. Currently they write to Linear/Discord but not the admin API.

2. **Reek doesn't write findings to the API.** His crawl results go to Discord. They should also go to `POST /admin/findings`.

3. **Daeron doesn't write triage summaries to the API.** Triage goes to Linear comments. Should also go to `POST /admin/triage/summaries`.

4. **AgentStore is in-memory.** All data lost on restart. Needs persistence (JSON file, SQLite, or PostgreSQL).

5. **No per-agent detail view.** GET /admin/agents/:id isn't routed. Can't see agent config, memory, or run history.

6. **No trigger endpoints.** POST /admin/trigger/* (reek-crawl, triage, heartbeat) not implemented. These would call OpenClaw Gateway to enqueue agent work.

7. **No agent config editing.** PUT /admin/agents/:id/config not implemented.

### How to Wire It

**Option A: Agents call the API directly**
- Modify Daeron's AGENTS.md standing orders to include `curl POST /admin/agents/status` after each run
- Modify Reek's cron job to POST findings to the admin API after crawling
- Requires: agents need HTTP access to localhost:4350

**Option B: OpenClaw webhook integration**
- Create webhook endpoints (POST /admin/trigger/*) that call OpenClaw Gateway API
- SPA triggers agent work via buttons → webhook → cron/wake → agent runs → agent self-reports back
- Requires: OpenClaw Gateway API URL + auth token

**Option C: Hybrid (recommended)**
- Agents self-report via Option A (direct POST after each run)
- SPA triggers via Option B (webhook endpoints for on-demand work)
- Persistence via JSON file or SQLite (AgentStore writes to disk)

---

## Other Open Items

### From Linear (all closed today)

| Issue | Status | What |
|-------|--------|------|
| DP-93 | ✅ Done | Spec updated to match reality, deferred endpoints documented |
| DP-98 | ✅ Done | 161 admin panel tests with race detection |
| DP-99 | ✅ Done | JWT localStorage threat model documented |

### Deferred Features (from DP-93)

These are documented in the spec as future work, not bugs:

- `POST /admin/restart` — server restart (systemd is sufficient)
- `POST /admin/bans` / `DELETE /admin/bans/:id` — web bans (in-game moderation exists)
- `POST /admin/trigger/*` — webhook triggers (described above)
- `GET /admin/agents/:id/memory` — narrative memory viewer
- `GET /admin/narrative` — cross-agent event feed
- `GET /admin/research/export` — research data export
- Lua script editor (Monaco) — in-game OLC sufficient
- AG Grid inline editing — standard edit pages work fine
- Zone graph (React Flow) — future visualization

---

## Build & Run

```bash
# Backend
cd darkpawns_repo
go build ./... && go vet ./... && go test -race ./pkg/admin/...

# Frontend
cd admin-ui
npm install
npm run dev          # dev server on :5173 (proxies to :4350)
npm run build        # production build → dist/ served by Go binary

# Tests
go test -race ./pkg/admin/... -v
```

---

## Key Files to Read

| File | Why |
|------|-----|
| `PLAN-web-admin-architecture.md` | Full spec — Section 6 is the agent integration architecture |
| `pkg/admin/handlers.go` | All API handlers — read this to understand what's wired |
| `pkg/admin/agent_store.go` | In-memory agent/finding/triage store |
| `pkg/admin/router.go` | Route registration with auth roles |
| `admin-ui/src/api/client.ts` | Frontend API client — all endpoints typed |
| `admin-ui/src/pages/AgentsPage.tsx` | Agent cards + findings feed + triage summaries |
| `admin-ui/src/pages/OperationsPage.tsx` | Server status, players, logs, metrics, quick actions |
| `docs/agents/dp-agent.md` | dp-agent CLI docs — how agents connect to the game |
