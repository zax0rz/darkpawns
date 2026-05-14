# Session Notes — 2026-05-14 (Opus QA Fixes)

## What Happened

Knocked out 18 of 21 issues from Opus's admin panel QA review. Subagents dispatched for parallel work, inline fixes for smaller changes. All Go + TypeScript compiles clean. Committed as `05879b8`.

## Issues Closed (18)

### CRITICAL (2)
- **DP-81:** CORS rewritten — exact-origin matching (localhost:5173/4350 + `ADMIN_CORS_ORIGIN` env). Substring match eliminated.
- **DP-82:** `/admin/health` moved outside AuthMiddleware in main.go. Registered on main mux before the auth-wrapped catch-all.

### HIGH (4)
- **DP-80:** Full `POST /admin/login` endpoint. Validates bcrypt against player DB, returns JWT with role from level. Frontend LoginPage + useAuth hook wired.
- **DP-83:** React setState-during-render fixed in MobEditPage, ObjectEditPage, RoomEditPage. Moved to `useEffect`. Subagent (DeepSeek).
- **DP-84:** AgentStore `nextID` split into `nextFindingID` + `nextTriageID`. Subagent (GLM-5-turbo).
- **DP-85:** `auth.NewIPRateLimiter()` wired to all admin routes via `wrap()` middleware.

### MEDIUM (8)
- **DP-86:** `HasRole` returns false for unknown roles (defense-in-depth against typos).
- **DP-87:** HP regex updated to handle negative plus: `/(\d+)d(\d+)([+-]\d+)?/`
- **DP-88:** `validateStringField` helper added. PUT handlers reject empty strings, cap length (256 names, 8192 descriptions), strip control chars. Subagent (DeepSeek).
- **DP-89:** LogBuffer now includes level + RFC3339 time + attrs in entries.
- **DP-90:** Zone 0 (Limbo) reachable — removed `idStr == "0"` redirect clause.
- **DP-91:** All 4 audit event sites now include `auth.GetIPFromRequest(r)`.
- **DP-92:** Already fixed by DP-84 subagent (split ID counters).
- **DP-94:** Frontend Agents nav changed from `role: 'research'` to `role: 'builder'`.

### LOW (4)
- **DP-95:** Server-side alignment clamped to [-1000, 1000] in `SetMobAlignment`.
- **DP-96:** PUT with empty vnum returns 400, not redirect (redirect loses body).
- **DP-97:** Server-side clamps on gold, exp, weight, cost (all >= 0).
- **DP-100:** `processStartTime` captured at package init, uptime computed in `handleServerInfo`.

## What's Left (3)
- **DP-93:** MEDIUM — Spec fidelity gap (Phase 4-6 partial). Big scope — new endpoints needed.
- **DP-98:** LOW — Zero test coverage on admin panel. Needs dedicated test-writing session.
- **DP-99:** LOW — JWT in localStorage. Documented as standard SPA tradeoff, not a real fix.

## Models Used
- DeepSeek V4 Flash: DP-83 React setState fix, DP-88 PUT validation (subagents)
- GLM-5-turbo: DP-84 AgentStore race fix (subagent)
- MiMo v2.5 Base: main session, all Go backend fixes

## Key Files Modified
- `pkg/admin/router.go` — CORS, rate limiting, login route, health bypass
- `pkg/admin/handlers.go` — login handler, validation, audit IP, uptime, clamps
- `pkg/admin/login.go` — new file, POST /admin/login endpoint
- `pkg/admin/log_buffer.go` — level/time/attrs in log entries
- `pkg/admin/agent_store.go` — split ID counters
- `pkg/auth/jwt.go` — HasRole defense-in-depth
- `pkg/game/world_write.go` — alignment/gold/exp/weight/cost clamps
- `cmd/server/main.go` — health outside auth, pass db to router
- `admin-ui/src/pages/LoginPage.tsx` — wired to real login endpoint
- `admin-ui/src/hooks/useAuth.ts` — login calls POST /admin/login
- `admin-ui/src/pages/MobEditPage.tsx` — useEffect, HP regex fix
- `admin-ui/src/pages/ObjectEditPage.tsx` — useEffect fix
- `admin-ui/src/pages/RoomEditPage.tsx` — useEffect fix
- `admin-ui/src/components/Layout.tsx` — role mismatch fix

## What's Next
1. Push to GitHub when ready
2. DP-93 (spec fidelity) — big, needs planning
3. DP-98 (test coverage) — write tests for admin handlers
4. Domain-expansion back online → rebuild binary with tick fix
