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

## Agent Integration — The Closed Loop (Section 6 of Admin Spec)

The admin API is bidirectional. Agents write to it (status, findings, triage), the SPA reads from it, and the SPA can trigger agent work via webhooks. This is the core architecture from Section 6 — the loop isn't fully closed yet.

```
┌─────────────────────────────────────────────────────┐
│                  ADMIN PANEL (SPA)                   │
│  Dashboard │ Findings │ Triage │ Ops │ Agent Status  │
└────┬──────────────────────────────┬─────────────────┘
     │ reads                        │ triggers
     ▼                              ▼
┌──────────────────┐    ┌─────────────────────────┐
│  Admin REST API  │    │   Webhook Endpoints      │
│ /admin/agents/*  │    │  POST /admin/trigger/*   │
│ /admin/findings/*│    │                          │
│ /admin/triage/*  │    │                          │
└──────┬───────────┘    └────────┬────────────────┘
       │ writes                  │ enqueues
       ▼                         ▼
┌──────────────────────────────────────────────────┐
│          OpenClaw Gateway (mac-mini)              │
│  Cron │ Standing Orders │ Hooks │ Task Flow       │
└──────┬─────────────────────────┬─────────────────┘
       │ executes                │ executes
       ▼                         ▼
┌──────────────┐         ┌──────────────────┐
│   DAERON     │         │      REEK        │
│  (MiMo v2.5) │         │  (DeepSeek V4)   │
│  Loremaster  │         │  Code Crawler    │
└──────────────┘         └──────────────────┘
```

### What's Built (Backend)

| Endpoint | Method | Purpose | Status |
|----------|--------|---------|--------|
| `/admin/agents` | GET | List all agents + status | ✅ Working |
| `/admin/agents/status` | POST | Agent self-reports status | ✅ Working |
| `/admin/findings` | GET | List findings (filterable) | ✅ Working |
| `/admin/findings` | POST | Submit a finding | ✅ Working |
| `/admin/findings/:id` | GET | Finding detail | ✅ Working |
| `/admin/findings/:id` | PUT | Update status (confirm/reject) | ✅ Working |
| `/admin/triage/summaries` | GET | List triage summaries | ✅ Working |
| `/admin/triage/summaries` | POST | Submit triage summary | ✅ Working |

### What's Built (Frontend)

- **AgentsPage:** Agent cards with status dots (green/active, yellow/other, red/error), findings feed with 3 filter dropdowns (source, severity, status), triage summary list
- **Dashboard:** Agent status card (30s auto-refresh), recent findings card with severity badges, stats pills (total/open/confirmed/fixed/critical)

### What's NOT Built

**1. Agents don't self-report to the admin API.**
- Daeron writes triage results to Linear comments, not `POST /admin/findings`
- Reek writes findings to Discord, not `POST /admin/findings`
- Neither calls `POST /admin/agents/:id/status` after runs
- The agent store has seeded static entries that reset on restart

**2. AgentStore is in-memory only.**
- All findings, triage summaries, and agent statuses lost on server restart
- Needs persistence: JSON file, SQLite, or PostgreSQL

**3. Webhook triggers not implemented.**
- `POST /admin/trigger/reek-crawl` — SPA button to kick off Reek
- `POST /admin/trigger/triage` — SPA button to kick off Daeron triage
- `POST /admin/trigger/heartbeat` — SPA button to trigger server health check
- These need to call the OpenClaw Gateway API to enqueue work

**4. No per-agent detail view.**
- `GET /admin/agents/:id` not routed — can't see agent config, memory, or run history

### How to Wire the Self-Reporting

The agents need to POST to the admin API after each run. Two approaches:

**Option A: Agents call the API directly (simpler)**
- Add standing order instructions to Daeron's AGENTS.md: after triage, POST findings + status to admin API
- Add instructions to Reek's cron job: after crawl, POST findings + status to admin API
- Agents need HTTP access to localhost:4350 (same machine or network)
- Example: `curl -X POST http://localhost:4350/admin/agents/daeron/status -H 'Authorization: Bearer $TOKEN' -H 'Content-Type: application/json' -d '{"status":"active","model":"mimo-v2.5-base"}'`

**Option B: OpenClaw Gateway integration (more robust)**
- Standing orders in AGENTS.md already define the programs — agents just need to also write to the admin API
- Cron jobs already exist for Reek (3 AM) and Daeron (7:30 AM) — add API calls to their execution flow
- The Gateway's cron system persists jobs at `~/.openclaw/cron/jobs.json` — agent runs are already tracked as background tasks

### How to Wire the Webhook Triggers

The SPA needs to trigger agent work on-demand. The OpenClaw Gateway has an API that supports this:

**Implementation path:**
1. Admin backend needs the OpenClaw Gateway URL + auth token (env var: `OPENCLAW_GATEWAY_URL`, `OPENCLAW_GATEWAY_TOKEN`)
2. `POST /admin/trigger/reek-crawl` → calls Gateway API to enqueue a one-shot cron job for Reek
3. `POST /admin/trigger/triage` → calls Gateway API to wake Daeron's session with a triage prompt
4. `POST /admin/trigger/heartbeat` → calls Gateway API to trigger Daeron heartbeat

**OpenClaw cron API reference:**
- `openclaw cron add --name "trigger" --at now --session <session> --system-event "<prompt>" --delete-after-run`
- Or use the Gateway HTTP API directly (see https://docs.openclaw.ai/automation/cron-jobs)

### Recommended Wiring Order

1. **Persistence first** — AgentStore → JSON file or SQLite (data survives restart)
2. **Self-reporting** — Add curl/HTTP calls to Daeron + Reek standing orders
3. **Findings pipeline** — Reek writes to `POST /admin/findings` instead of Discord
4. **Triage pipeline** — Daeron writes to `POST /admin/triage/summaries` after Linear triage
5. **Webhook triggers** — SPA buttons → admin API → OpenClaw Gateway → agent wakes
6. **Agent detail view** — `GET /admin/agents/:id` with config, memory, run history

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
