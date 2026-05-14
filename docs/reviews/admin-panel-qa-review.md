# Admin Panel QA Review — 2026-05-14

**Reviewer:** Opus QA subagent (Daeron-dispatched)
**Scope:** `pkg/admin/` (1,335 LOC Go), `pkg/game/world_write.go` (174 LOC), `admin-ui/src/` (3,549 LOC TS/TSX), `cmd/server/main.go` wiring, `pkg/auth/jwt.go`, `web/auth.go`.
**Build status:** `go build ./...` clean. `go vet ./...` clean. No tests for `pkg/admin/` or `admin-ui/`.

---

## Executive Summary

The admin panel is **architecturally sound** and broadly matches the revised spec: single binary, `/admin/` prefix on port 4350, JWT-Bearer auth on top of existing `web.AuthMiddleware`, World read methods reused, eleven new World write methods that hold `w.mu`, and a TanStack-Query SPA with role-aware navigation. Build is green and vet is clean. The code style is consistent and the layering is clean — `router.go` → `handlers.go` → `agent_store.go` / `world_write.go` with no circular imports.

The concerning side is the **security + spec-fidelity gap** between what the doc described and what was actually wired. There are **two CRITICAL** issues (the production CORS middleware is gated on `strings.Contains(origin, "localhost")` and effectively breaks production-mode admin entirely *and* widens dev mode to any host containing the substring "localhost"; the existing `web.AuthMiddleware` is also blanket-applied to `/admin/health` which the spec explicitly carved out as unauthenticated), **four HIGH** (no admin login endpoint despite the SPA shipping a login form, no rate-limit wiring on admin routes even though `pkg/auth/ratelimit.go` is referenced as already-wired, render-phase `setState` calls in three edit pages, and a data-race-prone agent status path that hands out shared pointers under a copied slice), and a handful of MEDIUM/LOW correctness and fidelity issues. Many spec-listed endpoints simply do not exist (`/admin/zones/:id/rooms`, room/zone flag editors, `/admin/triage/summary`, `/admin/trigger/*`, etc.) — Phase 4-6 is partial.

Total issues found: **2 CRITICAL, 4 HIGH, 9 MEDIUM, 6 LOW**.

---

## Issues by Severity

### CRITICAL (must fix before merge)

#### [ISSUE-001] Admin CORS middleware is broken in production AND too permissive in dev
- **File:** `pkg/admin/router.go:75-89`
- **Category:** Security / Fidelity
- **Description:** `corsMiddleware` only sets `Access-Control-Allow-Origin` if `strings.Contains(origin, "localhost")`. Two problems:
  1. In production, the admin SPA origin (`https://admin.darkpawns.labz0rz.com`) will *never* get CORS headers, so every cross-origin admin call from the SPA fails. The admin panel is non-functional in any deploy that isn't from localhost.
  2. `strings.Contains` is a substring match. An attacker-controlled origin like `https://localhost.evil.com` or `https://my-localhost-thing.attacker.tld` would be reflected back as `Access-Control-Allow-Origin`. Combined with the `pkg/auth` JWT going through localStorage (so an attacker would need to exfiltrate it via XSS), this is not an immediate RCE, but the CORS contract is wrong: it should be an exact-origin allowlist driven by env (mirroring `web/cors.go`'s `CORSMiddleware`, which already exists).
- **Impact:** Production admin panel unreachable; dev mode rubber-stamps any origin containing "localhost".
- **Fix:** Replace the local `corsMiddleware` with `web.CORSMiddleware` from `web/cors.go` (it already supports `CORS_ALLOWED_ORIGINS` env, exact matches, and a dev-mode override gated on `ENVIRONMENT=development`). Drop the substring match. Add `admin.darkpawns.labz0rz.com` (or whatever the prod admin host will be) to the production default list or set it via env.

#### [ISSUE-002] `/admin/health` requires authentication, contradicting its registration intent
- **File:** `pkg/admin/router.go:18` + `cmd/server/main.go:205`
- **Category:** Functionality / Security (DoS-friendly health probe)
- **Description:** Router registers `/admin/health` with no `requireRole`/auth wrapper ("Health (unauthenticated)" per the comment). But `main.go:205` mounts the whole admin router behind `web.AuthMiddleware`: `http.Handle("/admin/", web.AuthMiddleware(adminRouter))`. So a curl `/admin/health` without a JWT returns `401 Unauthorized`, defeating the carve-out and breaking any external probe (Kubernetes/Caddy upstream health check, etc.).
- **Impact:** Liveness checks fail; the spec's intent ("Health (unauthenticated)") is not realized. Externalized monitoring requires either creating a service-token JWT for the prober (overhead) or routing the probe to the existing `/health` instead — fine if intended, but then `/admin/health` is dead code.
- **Fix:** Either (a) delete `/admin/health` and direct probes to the existing `/health`, or (b) mount the admin router in `main.go` with the auth-bypass applied just to `/admin/health` (e.g., serve `/admin/health` directly on the root mux before the auth wrap).

---

### HIGH (should fix before merge)

#### [ISSUE-003] No admin login endpoint exists; LoginPage is a dead form
- **File:** `admin-ui/src/pages/LoginPage.tsx:8-13`, `admin-ui/src/hooks/useAuth.ts:27-34`, `pkg/admin/router.go` (no `/admin/login` route)
- **Category:** Functionality / Fidelity
- **Description:** Spec §3 ("Auth Strategy") calls out `POST /admin/login` as the credential exchange endpoint, and §8 says admin login must be rate-limited (5/min via `pkg/auth/ratelimit.go`). Neither exists. The SPA login form's submit handler does `setError('Login not yet implemented. Run this in the browser console to set a token…')`. `useAuth.login()` literally `throw new Error('Login not yet implemented — set admin_token in localStorage')`.
- **Impact:** No way to authenticate through the UI. The only path is for an operator to (a) generate a JWT via some out-of-band tool that knows `JWT_SECRET`, and (b) paste it into localStorage. That's a Phase 0 stub being shipped as Phase 7 "production-ready polish."
- **Fix:** Implement `POST /admin/login` that validates a player credential (DB lookup via `db.SQLDB()` against player saves or an admin keys table) and returns a signed JWT with the appropriate `role` claim. Wire `pkg/auth/ratelimit.go` per-IP to the login route specifically. Update `useAuth.login` to call it and stash the token in localStorage.

#### [ISSUE-004] Rate limiting is not wired to admin routes at all
- **File:** `pkg/admin/router.go` (no rate-limit middleware), `cmd/server/main.go:204-205`
- **Category:** Security
- **Description:** Spec §8.3 mandates rate-limited admin login; the original architectural narrative also lists `pkg/auth/ratelimit.go` as "✅ Exists." Neither the (missing) login route nor any other `/admin/*` route runs through a limiter. Combined with JWT being long-lived (24h) and the absence of replay defenses, a leaked token plus zero rate limiting on mutations means a single XSS can drain or scribble world data without throttle.
- **Impact:** Brute-force the (not-yet-existent) login. Once login lands, unlimited PUTs against `/admin/rooms/:vnum`, `/admin/mobs/:vnum`, `/admin/objects/:vnum`.
- **Fix:** Add a per-IP limiter middleware to `pkg/admin/router.go` (stricter for write methods, lenient for reads). Reuse `auth.NewIPRateLimiter()`.

#### [ISSUE-005] Three edit pages call `setState` during render — React anti-pattern, will warn/break under StrictMode
- **File:** `admin-ui/src/pages/RoomEditPage.tsx:24-28`, `admin-ui/src/pages/MobEditPage.tsx:31-47`, `admin-ui/src/pages/ObjectEditPage.tsx:25-31`
- **Category:** Functionality / Code Quality
- **Description:** Each edit page does:
  ```tsx
  if (mob && !initialized) {
    setShortDesc(mob.short_desc);
    // ... seven more setStates ...
    setInitialized(true);
  }
  ```
  This calls `setState` during the render phase, which React explicitly warns against ("Cannot update a component while rendering a different component"). It happens to work in practice because React re-renders, but under React 18 StrictMode every component renders twice in dev, so all initial form values get set twice and any flicker is observable. It will also fight TanStack Query refetches — if `useQuery` returns updated `mob` data, `initialized` is already `true` and the form never re-syncs with server state.
- **Impact:** Stale edits after refetch; dev-mode console noise; latent bug if anyone enables `<React.StrictMode>` in `main.tsx` (currently not enabled — checked).
- **Fix:** Move the form initialization into `useEffect(() => { setShortDesc(mob.short_desc); ... }, [mob])`. Drop the `initialized` boolean.

#### [ISSUE-006] `handleAgents` race: returns shared `*AgentStatus` pointers while `UpdateAgentStatus` mutates them under a separate lock window
- **File:** `pkg/admin/agent_store.go:79-88` and `:91-102`
- **Category:** Functionality (data race)
- **Description:** `GetAgents` copies the map's values into a `[]*AgentStatus` *under read lock*, then returns. The returned slice contains pointers to the same `AgentStatus` structs the store owns. The handler then iterates and JSON-encodes them *without* the lock. Meanwhile `UpdateAgentStatus` takes the write lock and mutates `agent.Status = ...` / `agent.LastRun = time.Now()` in place. The JSON encoder reads those fields concurrently — Go's race detector will flag this on any concurrent GET/POST.
- **Impact:** Data race. Almost certainly benign at runtime (pointer/int writes are word-aligned), but it will fail `go test -race` once tests exist, and it's a real correctness issue if `AgentStatus` ever grows a string longer than one word or any composite field.
- **Fix:** In `GetAgents`, copy each struct by value into a `[]AgentStatus`. The handler then encodes the snapshot without sharing memory with the store.

---

### MEDIUM (fix in follow-up)

#### [ISSUE-007] `requireRole("typo")` is a soft-bypass: unknown required-role maps to 0, so any authenticated user passes
- **File:** `pkg/auth/jwt.go:54-57`
- **Category:** Security (defense-in-depth)
- **Description:** `HasRole` uses `hierarchy[c.Role] >= hierarchy[required]`. Both lookups return zero value `0` on miss. So `RequireRole("admni")` or any typo silently treats every authenticated request as authorized. The router currently passes valid strings, but the construct is fragile.
- **Fix:** Make `HasRole` return false when `required` isn't in the hierarchy map, e.g. `req, ok := hierarchy[required]; if !ok { return false }`. Same for `c.Role` (unknown role → deny).

#### [ISSUE-008] `PUT` handlers accept empty strings for required fields; no input validation
- **File:** `pkg/admin/handlers.go:447-465` (room), `:534-590` (mob), `:679-710` (object)
- **Category:** Functionality / Data Integrity
- **Description:** Pointer-to-string fields are "set if present," but an empty string `""` is treated as "set this to empty." A builder can blank out a room name or mob short_desc. No length cap either — a 5MB description is accepted. No sanitization for control characters that would break the MUD's text protocol (e.g. embedded `\r\n` could spoof MUD output).
- **Fix:** Add minimal validation in the handlers (or in `world_write.go`): reject empty strings, cap length (e.g. 256 chars for names, 8KB for descriptions), strip or reject ANSI/control bytes.

#### [ISSUE-009] Mob HP regex doesn't handle non-positive `Plus` — parsing breaks for `5d8+0` (matches as 0, ok) and any future `5d8-2`
- **File:** `admin-ui/src/pages/MobEditPage.tsx:37-42`
- **Category:** Functionality
- **Description:** `mob.hp.match(/(\d+)d(\d+)\+(\d+)/)` requires a literal `+` and a digits-only group. `DiceRoll.String` (Go side, `pkg/parser/mob.go:91-92`) uses `fmt.Sprintf("%dd%d+%d", …)`, which for a negative `Plus` would emit e.g. `5d8+-2`. The regex won't match → `hpNumDice/hpSizeDice/hpAdd` all stay at 0, and on save the user accidentally zeros out the mob's HP. Existing prototypes appear to be non-negative, but nothing enforces it.
- **Fix:** Use `/(\d+)d(\d+)([+-]?\d+)/` and `Number(hpMatch[3])` for the add value. Show a warning if parsing fails instead of silently zeroing.

#### [ISSUE-010] Admin audit events have no IP address — privacy hash is moot
- **File:** `pkg/admin/handlers.go:476-482, :604-610, :716-727, :778-784`
- **Category:** Fidelity (Section 8 says "Audit logging — Every mutation logged via existing pkg/audit/logger.go")
- **Description:** Every admin audit event built in `handlers.go` sets `User` and `Action` but leaves `IPAddress` empty. The audit logger then hashes an empty string (or skips it). When investigating an incident, you can't trace a mutation back to a network origin.
- **Fix:** `audit.AuditEvent{ ..., IPAddress: auth.GetIPFromRequest(r) }` (the helper exists at `pkg/auth/ratelimit.go:61`).

#### [ISSUE-011] `handleZoneByID` redirects on `/admin/zones/0` instead of looking up zone 0
- **File:** `pkg/admin/handlers.go:70-74`
- **Category:** Functionality
- **Description:** `if idStr == "" || idStr == "0"` redirects to the list. Zone 0 is a valid CircleMUD zone vnum (the Limbo zone in canonical worlds). It will never be reachable through the API.
- **Fix:** Drop the `idStr == "0"` clause. Let `strconv.Atoi("0")` return 0 and `world.GetZone(0)` decide whether it exists.

#### [ISSUE-012] LogBuffer entries drop slog level, time, and attrs — only the message survives
- **File:** `pkg/admin/log_buffer.go:72-80`
- **Category:** Functionality
- **Description:** `logBufferHandler.Handle` forwards the full record to the base handler, then writes only `record.Message` to the in-memory buffer. The Operations page therefore shows raw messages with no severity, no timestamp, no error attrs. Triage value of the live log tail is sharply reduced.
- **Fix:** Build a one-line string from the record (level + RFC3339 timestamp + message + attrs) before writing to the buffer. Or just `slog.NewTextHandler(buf, …)` as the buffer's writer and let slog format it for you.

#### [ISSUE-013] Backend role for agent routes is `builder`, frontend nav shows them at `research`
- **File:** `admin-ui/src/components/Layout.tsx:21` vs `pkg/admin/router.go:47-51`
- **Category:** Integration / UX
- **Description:** Layout lists Agents nav item as `role: 'research'` (i.e., visible to research+). Backend `/admin/agents*` requires `builder`. A `research`-role user clicks "Agents" → empty page with red "Failed to load" boxes.
- **Fix:** Decide which is canonical. Spec §3 says agents are `research` territory; backend should drop to `requireRole("research", ...)` for read endpoints, and the frontend nav stays as-is. If `builder`-only is intended, raise the nav `role` to `builder`.

#### [ISSUE-014] AgentStore shares `nextID` between `findings` and `triages` — counter intermixes resource types
- **File:** `pkg/admin/agent_store.go:50, 130, 178`
- **Category:** Code Quality / API Hygiene
- **Description:** A new finding might get ID 5; the next new triage summary gets ID 6 from the same counter. Findings IDs become non-contiguous, and a client that assumes contiguous IDs (e.g. for caching) will be surprised. Functionally fine, but a sign of carelessness.
- **Fix:** Two counters, `nextFindingID` and `nextTriageID`. Cheap.

#### [ISSUE-015] Many spec endpoints + features simply don't exist (Phases 4–6 are partial)
- **File:** spec §3 route table vs `pkg/admin/router.go`
- **Category:** Fidelity
- **Description:** Missing endpoints called out as in-scope:
  - `GET /admin/zones/:id/rooms`, `GET /admin/rooms/:vnum/players|mobs|items` (live state per-room)
  - Room flag edits, exit edits, extra-desc edits (only name/desc are writable)
  - Mob flag edits, affect flags, stats edits (STR/INT/etc) — only descs, level, AC, HP, gold, exp, alignment
  - Object value edits (the type-dependent `Values[4]`), wear flags, extra flags, applies
  - Shops endpoints (`/admin/shops*`) — entirely missing
  - `POST /admin/zones/:id/reset` returns 501 Not Implemented (acceptable Phase 5 placeholder, but the SPA wires a disabled "Zone Reset All" button instead of per-zone)
  - `/admin/restart`, `/admin/bans*` — missing
  - `/admin/triage/summary` (singular, POST per spec) is `/admin/triage/summaries` (plural, POST list)
  - `/admin/trigger/reek-crawl`, `/admin/trigger/triage`, `/admin/trigger/heartbeat` (Phase 6 webhooks) — missing
- **Impact:** Phase 4 (Game Editors) is roughly 25% complete; Phase 5 (Operations) ~50%; Phase 6 (AI/Research) ~40%.
- **Fix:** Either ship as "Phase 4 partial" and update the spec to match, or land the missing endpoints in follow-ups with clear tickets.

---

### LOW (nice to fix)

#### [ISSUE-016] `objUpdateRequest` declares no validation; `weight` and `cost` accept negative integers
- **File:** `pkg/admin/handlers.go:414-420`
- **Category:** Data Integrity
- **Fix:** Clamp non-negative or reject. Same for `mob.gold`, `mob.exp`.

#### [ISSUE-017] `mob.alignment` is clamped to `[-1000,1000]` only in the frontend (MobEditPage.tsx:151) — backend accepts anything
- **File:** `admin-ui/src/pages/MobEditPage.tsx:151`, `pkg/game/world_write.go:117-126`
- **Category:** Data Integrity
- **Fix:** Server-side clamp in `SetMobAlignment` or in the handler.

#### [ISSUE-018] `handleObjectByVnum` redirects on `PUT /admin/objects/` with empty vnum — should be 400
- **File:** `pkg/admin/handlers.go:516-520, :661-665`
- **Category:** Functionality (minor)
- **Description:** `http.Redirect(w, r, "/admin/mobs", http.StatusMovedPermanently)` on a PUT loses the body. Same issue in `handleMobByVnum` PUT path.
- **Fix:** Return `400 Bad Request` for write methods with missing path param.

#### [ISSUE-019] `serverInfoResponse.Uptime` is always `""` — field declared but never populated
- **File:** `pkg/admin/handlers.go:26-31, :766-770`
- **Category:** Functionality (minor — Dashboard shows nothing for uptime)
- **Fix:** Capture process start time at server boot and compute `time.Since(start).Round(time.Second).String()` in the handler. Or drop the field from the response type until Phase 5.

#### [ISSUE-020] JWT in `localStorage` is XSS-exfiltrable; SPA has no CSP for inline scripts because admin SPA is served by Vite/Caddy, not Go
- **File:** `admin-ui/src/api/client.ts:4`, `admin-ui/src/hooks/useAuth.ts:11`
- **Category:** Security (standard SPA tradeoff)
- **Description:** Standard SPA tradeoff. The Go server's CSP (`web/security.go`) only protects the *server-served* content; the admin SPA lives at a different origin once deployed. With React 18 and JSX, the XSS surface is small but real (e.g. `dangerouslySetInnerHTML` is not used — confirmed via grep).
- **Fix:** Document the threat model. Consider moving to httpOnly cookie sessions for production (will require CSRF protection, which the spec explicitly rejected — so this is a deliberate decision, just call it out).

#### [ISSUE-021] No tests at all for `pkg/admin/`, `pkg/game/world_write.go`, or any admin-ui component
- **File:** entire scope
- **Category:** Testing Gap
- **Description:** `go test ./pkg/admin/...` returns "no test files." No vitest/jest config in `admin-ui/package.json`. World write methods are untested even though they hold the world lock and mutate shared state.
- **Fix:** See Testing Gaps section below.

---

## Fidelity Checklist

| Spec Requirement | Implemented? | Notes |
|---|---|---|
| `/admin/` prefix on port 4350 | ✅ | `cmd/server/main.go:205` |
| `web.AuthMiddleware` reused for admin | ✅ | `cmd/server/main.go:205` |
| `Role` claim added to JWT | ✅ | `pkg/auth/jwt.go:48` |
| `RequireRole` middleware | ✅ | `pkg/admin/router.go:58` (named `requireRole`, in admin pkg not web) |
| Audit logger wired to mutations | 🟡 | Logged, but IPAddress always empty (ISSUE-010) |
| Rate-limit admin login | ❌ | No login route, no limiter (ISSUE-003, ISSUE-004) |
| CORS locked to admin SPA origin | ❌ | Substring `localhost` match instead (ISSUE-001) |
| `GET /admin/zones` + `:id` | ✅ | `:0` shadowed by redirect (ISSUE-011) |
| `PUT /admin/zones/:id` | ❌ | Missing |
| `GET /admin/zones/:id/rooms` | ❌ | Missing |
| `GET/PUT /admin/rooms/:vnum` | 🟡 | Only name + description writable; no flags/exits/extra descs |
| `GET/PUT /admin/mobs/:vnum` | 🟡 | Stats partial; no flag/affect/script edits |
| `GET/PUT /admin/objects/:vnum` | 🟡 | Only name/desc/weight/cost; no Values[]/flags/applies |
| `GET/PUT /admin/shops/:id` | ❌ | Missing entirely |
| `GET /admin/players` | ✅ | Returns name/level/room |
| Live state per-room (`/players`, `/mobs`, `/items`) | ❌ | Missing |
| `POST /admin/zones/:id/reset` | 🟡 | 501 Not Implemented placeholder |
| `POST /admin/restart` | ❌ | Missing |
| `GET /admin/server` | ✅ | Uptime field empty (ISSUE-019) |
| `GET /admin/logs` | 🟡 | Wired, but messages only (ISSUE-012) |
| `POST/DELETE /admin/bans` | ❌ | Missing |
| `GET /admin/agents` | ✅ | + seed data for daeron/reek |
| Agent self-report POST status | 🟡 | Implemented as `POST /admin/agents/status` with body, not `:id/status` |
| `GET/POST /admin/findings` | ✅ | Reasonable shape |
| `PUT /admin/findings/:id` | ✅ | No status-value validation |
| `POST/GET /admin/triage/summaries` | ✅ | Plural (spec said singular) |
| `POST /admin/trigger/*` webhooks | ❌ | Missing |
| `GET /admin/research/export` | ❌ | Missing |
| `GET /admin/agents/:id/memory` | ❌ | Missing |
| React SPA: Vite + TS + Tailwind v4 | ✅ | `vite.config.ts`, `package.json`, `@tailwindcss/vite` |
| TanStack Query | ✅ | Used across all pages |
| React Router | ✅ | `App.tsx` |
| Xterm embedded as tab | ✅ | `MudTerminal.tsx` |
| React Flow zone graph | ❌ | Not used |
| Monaco editor for Lua | ❌ | Not used |
| AG Grid data tables | ❌ | Plain `<table>` instead |
| Recharts metrics | ❌ | Not used |
| Login flow (admin JWT) | ❌ | Stub form, manual localStorage (ISSUE-003) |
| Dev proxy `/admin/*` and `/ws` → :4350 | ✅ | `vite.config.ts` |
| Role-based UI hiding | 🟡 | Implemented, but research/builder mismatch (ISSUE-013) |
| Command palette (Ctrl+K) | ✅ | `CommandPalette.tsx` |
| Dark/light toggle | ✅ | `useTheme.ts` |
| Error boundaries + loading states | ✅ | `ErrorBoundary.tsx`, `Skeleton.tsx` |
| WebSocket reconnect for terminal | 🟡 | Manual reconnect button only; no auto-retry/backoff |

Legend: ✅ done · 🟡 partial · ❌ missing

---

## Testing Gaps

Prioritized list — ship with at least the top three before any production rollout.

1. **`pkg/admin` HTTP handler tests** (highest) — table-driven tests using `httptest.NewRecorder` for every route: auth-missing returns 401, role-too-low returns 403, valid-but-not-found returns 404, valid mutations apply and return updated entity. Especially:
   - `requireRole` against unknown / typo'd required role (catches ISSUE-007).
   - PUT with empty body, malformed JSON, partial fields.
   - Concurrent GET `/admin/agents` + POST `/admin/agents/status` under `-race` (catches ISSUE-006).

2. **`pkg/game/world_write.go` tests** — happy path + missing-vnum for each setter. Important because these hold `w.mu` and any future regression could deadlock the game loop. Use `game.NewWorld` with a tiny seed or a test helper.

3. **CORS + auth integration test** — spin up the full mux from `main.go` (factor out an `App.NewMux()` helper while you're at it — addresses the architectural note at the top of `main.go`) and assert: prod origin gets CORS headers; localhost origin in prod mode does not; preflight OPTIONS works; `/admin/health` auth carve-out works (after ISSUE-002 fix).

4. **`admin-ui` smoke tests** — vitest + React Testing Library for the three edit pages (catches ISSUE-005 after fix), Login page, ProtectedRoute redirect.

5. **`MudTerminal` reconnect behavior** — at minimum a manual reconnect test in the running app; ideally a Playwright spec that kills the WS and asserts the SPA recovers without a full page reload.

6. **AgentsPage filter logic** — verify queries serialize filters correctly and the table renders empty/error/loaded states.

---

## What's Solid

A lot, actually — flag this when reporting back to The Architect.

- **Clean Go layering.** `router.go` (wiring) → `handlers.go` (HTTP shape) → `agent_store.go` / `world_write.go` (state). No circular imports, no hidden side effects. Build and vet are clean.
- **Write methods are correctly locked.** Every setter in `world_write.go` follows the same `w.mu.Lock()` / `defer w.mu.Unlock()` / map lookup / mutate pattern. Consistent and correct for the existing concurrency model.
- **JWT `Role` claim added thoughtfully.** Defaults to `"player"` on empty (`pkg/auth/jwt.go:71-74`), so existing tokens don't accidentally elevate. `HasRole` is one place to maintain the hierarchy (modulo ISSUE-007).
- **TanStack Query is used correctly.** Cache keys include filter params (`['findings', filterStatus, filterSeverity, filterSource]`), refetch intervals on the right pages (agents and findings at 30s, logs at 10s), `invalidateQueries` after writes.
- **Error boundaries wrap every route.** `App.tsx:33-46` is meticulous about this. Skeleton components for loading states. Toast provider in place.
- **Auth flow handles 401 cleanly.** `api/client.ts:13-17` clears the token and bounces to `/login` on any 401 response — defends against expired tokens without prompting the user.
- **Dev proxy is correctly configured.** `vite.config.ts` proxies both `/admin/*` and `/ws` to :4350 with `ws: true` for the upgrade. Terminal works in dev without CORS.
- **Mobile/tablet/desktop breakpoints** with a real responsive nav (sidebar + bottom tab bar). More care than typical for an admin UI.
- **Type safety.** Backend response types in `api/client.ts` mirror the Go JSON shapes exactly (I spot-checked all five: Zone, Mob, Obj, Room, AgentStatus). No `any` in the API layer.
- **Audit events fire on every mutation handler.** Even if `IPAddress` is missing, the structure is right and the action names are descriptive (`admin_room_update`, `admin_mob_update`, `admin_object_update`, `admin_server_info`).
- **The dashboard is honest about Phase 6 status.** "Coming in Phase 6" placeholders rather than pretending to ship empty real components.

This is a serviceable Phase 0+1+2+3 foundation with selective Phase 4 work. The CRITICAL CORS issue is the only thing that genuinely blocks merge once you decide on the production origin; everything else can land as follow-up tickets (DP-XXX in Linear) without breaking anything that's already working.
