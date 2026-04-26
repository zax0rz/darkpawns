# Dark Pawns Security Audit Report

**Date:** 2026-04-25
**Scope:** 12-commit modernization sprint (C→Go MUD codebase)
**Auditor:** Claude Opus 4.6 (automated)
**Commit:** 0b9e6b5 (main)

---

## Executive Summary

This audit identified **7 CRITICAL**, **10 HIGH**, **11 MEDIUM**, and **5 LOW** severity findings across authentication, concurrency, input handling, DoS resistance, and economy systems. The most dangerous class of bugs involves **unsynchronized access to shared state** — room item maps, player maps, inventory slices, and gold fields are read and mutated without holding the World mutex, creating exploitable race conditions for item duplication, server crashes, and gold underflow.

---

## CRITICAL Findings

### C1. Slice Modification During Iteration — Item Duplication/Crash
**File:** `pkg/game/item_transfer.go:238-240, 251-255, 413-420`
**Severity:** CRITICAL

`doDrop("drop all")` and `doGive("give all")` iterate `ch.Inventory.Items` with `range` while `performDrop`/`performGive` calls `MoveObject()` which modifies the underlying slice via `detachObjectLocked()`. Go's `range` captures the slice header at loop start, but the backing array is mutated mid-iteration.

```go
// line 238 — doDrop
for _, obj := range ch.Inventory.Items {
    w.performDrop(ch, obj)  // modifies ch.Inventory.Items via MoveObject
}
```

Same pattern at lines 251-255 (drop all.keyword) and 413-420 (give all).

**Impact:** Skipped items, double-processing of items (duplication), or index-out-of-bounds panic.
**Fix:** Copy the slice before iterating (as correctly done in `doGet` at line 102-103).

---

### C2. Unprotected `w.players` Map Iteration — Server Crash
**File:** `pkg/game/world.go:233`
**Severity:** CRITICAL

`sendToZone()` iterates `w.players` without holding `w.mu`:

```go
func (w *World) sendToZone(roomVNum int, msg string) {
    // ...
    for _, p := range w.players {  // NO LOCK
```

Concurrent `RemovePlayer()` (called from session `Unregister`) deletes from the same map. Go's runtime detects concurrent map read+write and panics with `fatal error: concurrent map iteration and map write`.

**Same pattern in:**
- `pkg/game/modify.go` (doEmote)
- `pkg/game/party.go` (multiple functions)
- `pkg/game/clans.go` (multiple functions)
- `pkg/game/affect_update.go`
- `pkg/game/act_comm.go`

**Impact:** Any player disconnect during zone-broadcast causes immediate server crash.
**Fix:** Hold `w.mu.RLock()` around all `w.players` iterations.

---

### C3. Unprotected `w.roomItems` Access — Item Duplication via TOCTOU
**File:** `pkg/game/item_transfer.go:98, 102-103, 120-121`
**Severity:** CRITICAL

`doGet()` reads `w.roomItems[ch.RoomVNum]` and copies it WITHOUT holding `w.mu`:

```go
// line 102-103 — NO LOCK
items := make([]*ObjectInstance, len(w.roomItems[ch.RoomVNum]))
copy(items, w.roomItems[ch.RoomVNum])
```

Two players executing `get all` simultaneously in the same room both copy the same item list. Both then call `performGetFromRoom()` → `MoveObject()` (which does hold `w.mu`). The first succeeds; the second attempts to detach an already-detached object.

**Impact:** Depends on `detachObjectLocked` behavior — if it fails silently, the item ends up in only one inventory (benign). If it succeeds because the object's `Location` field wasn't updated between copy and move, duplication occurs.
**Fix:** Hold `w.mu.RLock()` while copying `roomItems`, or move the copy inside `MoveObject`.

---

### C4. No WebSocket Message Size Limit — Memory Exhaustion DoS
**File:** `pkg/session/manager.go:28-30, 310`
**Severity:** CRITICAL

The WebSocket upgrader sets `ReadBufferSize: 1024` (I/O buffer only) but never calls `conn.SetReadLimit()`. The gorilla/websocket default read limit is effectively unbounded — a client can send arbitrarily large frames.

```go
// line 310 — only sets deadline, no SetReadLimit
s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
```

**Impact:** A single malicious client sending a 1GB WebSocket frame exhausts server memory.
**Fix:** Add `s.conn.SetReadLimit(16384)` (16KB) in `readPump()` before the read loop.

---

### C5. No Per-IP Connection Limit — Connection Flood DoS
**File:** `pkg/session/manager.go:63-67, 162-185`
**Severity:** CRITICAL

The `Manager` tracks sessions by player name but does not count connections per IP address. `HandleWebSocket()` creates a new session and spawns goroutines for every connection, regardless of how many connections already exist from that IP.

```go
// line 170-184 — no IP connection counting
session := &Session{
    conn: conn,
    // ...
}
go session.writePump()
go session.readPump()
```

**Impact:** Attacker opens thousands of WebSocket connections from one IP, exhausting goroutines and file descriptors.
**Fix:** Track connections per IP in `HandleWebSocket`; reject after threshold (e.g., 5 per IP).

---

### C6. Gold Transfer Race Condition — Negative Gold Exploit
**File:** `pkg/game/item_transfer.go:328-338`
**Severity:** CRITICAL

`performGiveGold()` checks gold balance and deducts without any lock:

```go
// line 328 — CHECK (no lock)
if ch.Gold < amount && ch.GetLevel() < lvlGod {
    return
}
// line 336 — ACT (no lock, TOCTOU gap)
ch.Gold -= amount
// line 338 — no lock on victim either
vict.Gold += amount
```

Two concurrent `give 50 coins player2` commands from a player with 50 gold: both pass the check (Gold=50 >= 50), both deduct, Gold becomes -50. Victim gains 100 gold from thin air.

**Impact:** Unlimited gold generation. Two colluding players can create arbitrary gold.
**Fix:** Acquire `ch.mu.Lock()` around the entire check-and-deduct sequence. Also protect `vict.Gold`.

---

### C7. Player Death Lock Ordering Violation — Deadlock Risk
**File:** `pkg/game/death.go:218-235`
**Severity:** CRITICAL

`handlePlayerDeath()` accesses inventory then equipment, violating the documented lock order (`w.mu → Equipment.mu → Inventory.mu`):

```go
// line 220 — touches Inventory first
inventoryItems = player.Inventory.FindItems("")
player.Inventory.clear()              // no lock held

// line 233 — then Equipment
player.Equipment.mu.Lock()            // LOCK ORDER VIOLATION
player.Equipment.Slots = make(...)
player.Equipment.mu.Unlock()
```

Additionally, `player.Inventory.clear()` at line 222 has no synchronization with concurrent inventory reads from command handlers.

**Impact:** Deadlock under concurrent death + inventory access. Also: `clear()` racing with `doGet` or `doDrop` corrupts inventory state.
**Fix:** Acquire locks in documented order. Wrap entire death item transfer in `w.mu.Lock()`.

---

## HIGH Findings

### H1. X-Forwarded-For Header Spoofable — Rate Limit Bypass
**File:** `pkg/auth/ratelimit.go:68-71`
**Severity:** HIGH

```go
if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
    return strings.TrimSpace(ips[0])  // trusts client-supplied header
}
```

Without a trusted proxy, attackers forge `X-Forwarded-For` to get a fresh rate limiter bucket per request, bypassing the 5 req/sec login limit entirely.

**Fix:** Only trust X-Forwarded-For behind a known reverse proxy. Default to `RemoteAddr`.

---

### H2. Communication Commands Skip Input Sanitization — XSS Risk
**File:** `pkg/session/comm_cmds.go:116, 170, 245`
**Severity:** HIGH

`cmdSay`, `cmdShout`, `cmdGossip`, and `cmdTell` pass player messages directly to other players without calling `validation.SanitizeInput()`:

```go
// cmdShout line 116
message := strings.Join(args, " ")
// ... broadcast raw message to all zone players
```

**Impact:** If any client renders HTML (web-based MUD client), `<script>` tags or event handlers in messages execute in other players' browsers.
**Fix:** Call `SanitizeInput()` on all player-generated message content before broadcast.

---

### H3. No Message Size Limit on Communication Commands — Amplification DoS
**File:** `pkg/session/comm_cmds.go:111-200`
**Severity:** HIGH

`cmdShout` and `cmdGossip` accept arbitrarily long messages (`strings.Join(args, " ")`) and broadcast them to every player in zone/server. At 10 commands/sec rate limit, a player can broadcast ~10 large messages/sec to N players.

**Impact:** Bandwidth amplification. One player saturates all other players' send channels.
**Fix:** Enforce max message length (e.g., 512 bytes) in communication commands.

---

### H4. Inventory Capacity Check Outside Lock — Overflow
**File:** `pkg/game/item_transfer.go:64-74`
**Severity:** HIGH

`performGetFromRoom()` calls `canTakeObj()` to check inventory capacity BEFORE calling `MoveObject()` which acquires `w.mu`. Between check and move, another goroutine can add items to the player's inventory.

```go
func (w *World) performGetFromRoom(ch *Player, obj *ObjectInstance) {
    if w.canTakeObj(ch, obj) {           // capacity check WITHOUT lock
        w.MoveObjectToPlayerInventory(obj, ch)  // lock acquired inside
    }
}
```

**Impact:** Inventory can exceed capacity. Minor game balance issue.
**Fix:** Move capacity check inside `attachObjectLocked()`.

---

### H5. Session Register/AddPlayer Race — Ghost Players
**File:** `pkg/session/manager.go:507-514`
**Severity:** HIGH

Between `Register()` (adds to sessions map) and `AddPlayer()` (adds to world), a disconnect triggers `Unregister()` which removes the session, but the player may already be in `w.players` without a session.

```go
if err := s.manager.Register(login.PlayerName, s); err != nil { return err }
// GAP: disconnect here → Unregister removes session
if err := s.manager.world.AddPlayer(s.player); err != nil {
    s.manager.Unregister(login.PlayerName)  // already unregistered
    return err
}
```

**Impact:** Ghost player stuck in `w.players` forever. Name is permanently taken until server restart.
**Fix:** Make Register+AddPlayer atomic, or add cleanup that removes orphaned players.

---

### H6. Equipment Equip/Unequip Race — State Corruption
**File:** `pkg/game/equipment.go:173-244`
**Severity:** HIGH

`Equip()` and `Unequip()` each acquire `Equipment.mu`, but concurrent calls can interleave. `equip()` internally calls `unequip()` (line 215) which modifies `Slots` — if another goroutine calls `Unequip()` simultaneously, they both read/write `Equipment.Slots`.

**Impact:** Equipment slot corruption — item appears equipped but isn't in the slot map, or vice versa.
**Fix:** Ensure all equip/unequip operations acquire `w.mu` first (following documented lock order).

---

### H7. Mob Death Item Race — Corpse Duplication Window
**File:** `pkg/game/death.go:148-174`
**Severity:** HIGH

`handleMobDeath()` copies mob inventory/equipment (lines 148-155), creates corpse (line 161), THEN removes mob from world (line 173). Between copy and removal, another goroutine could access the mob's inventory.

```go
for _, item := range deadMob.Inventory {     // line 149 — copy
    inventoryItems = append(inventoryItems, item)
}
// ... items exist in BOTH mob inventory AND local slice ...
corpse := w.makeCorpse(...)                   // line 161
w.MoveObjectToRoom(corpse, roomVNum)          // items in corpse
w.mu.Lock()
delete(w.activeMobs, deadMobID)               // line 173 — NOW removed
w.mu.Unlock()
```

**Impact:** Items could theoretically appear in both corpse and mob if accessed between copy and delete.
**Fix:** Hold `w.mu` across the entire death sequence, or clear mob inventory immediately after copying.

---

### H8. Event Queue Unbounded Growth — Memory Exhaustion
**File:** `pkg/events/queue.go:69-71, 96`
**Severity:** HIGH

`EventQueue.events` is a heap with no size cap. `Create()` pushes events without checking queue length.

**Impact:** Scripts or game mechanics scheduling large numbers of events exhaust heap memory.
**Fix:** Add a max queue size (e.g., 10000 events) and reject/log when exceeded.

---

### H9. Object Pool Creates Unbounded Temporary Objects
**File:** `pkg/optimization/object_pool.go:79-93`
**Severity:** HIGH

When the pool is exhausted, `Get()` creates unlimited temporary objects outside the pool with no tracking.

**Impact:** Under load, memory grows without bound.
**Fix:** Return error when pool is exhausted, or track temporary objects with a hard cap.

---

### H10. `cmdForce "all"` Bypasses Authorization Check
**File:** `pkg/session/wizard_cmds.go:470-481`
**Severity:** HIGH

When `force all <command>` is used, the level comparison check at line 489 is in the single-target branch only. The "all" branch (lines 470-481) just logs and returns — but if command execution is ever wired in, it would bypass the `target.player.Level >= s.player.Level` check.

```go
if targetName == "all" {
    for _, sess := range s.manager.sessions {
        slog.Info("force all", ...)  // no level check here
    }
    return nil  // currently no execution, but no auth check either
}
```

**Impact:** Currently a no-op (not implemented), but a latent authorization bypass when implemented.
**Fix:** Add level check inside the "all" loop now, before execution is wired in.

---

## MEDIUM Findings

### M1. CORS Allows All Origins in Development Mode
**File:** `pkg/session/manager.go:33-34`
**Severity:** MEDIUM

```go
if os.Getenv("ENVIRONMENT") == "development" {
    return true  // all origins accepted
}
```

**Impact:** CSRF attacks against the WebSocket endpoint in development.
**Fix:** Use localhost-only whitelist even in development.

---

### M2. Missing Origin Header Allowed Through
**File:** `pkg/session/manager.go:45-49`
**Severity:** MEDIUM

Connections without an Origin header are accepted with only a warning log. Direct WebSocket clients (non-browser) don't send Origin, so this is by design, but it weakens origin-based protections.

---

### M3. TLS Optional by Default
**File:** `cmd/server/main.go:145-168`
**Severity:** MEDIUM

TLS requires `USE_TLS=true` env var plus cert/key paths. Default is plaintext HTTP.

**Impact:** Passwords and JWT tokens transmitted in cleartext.
**Fix:** Require TLS in production or document the requirement prominently.

---

### M4. Predictable Session IDs (Narrative Memory)
**File:** `pkg/session/memory_hooks.go:120-122`
**Severity:** MEDIUM

Session IDs are `playerName-unixNano`, which is guessable if login time is known.

**Impact:** Not used for authentication (JWT is separate), but could leak via narrative memory records.
**Fix:** Use UUID4 or `crypto/rand` for session IDs.

---

### M5. `ValidateCommand()` Defined But Not Called
**File:** `pkg/validation/input.go:96-111`, `pkg/session/commands.go:291`
**Severity:** MEDIUM

The validation package defines `ValidateCommand()` with SQL injection, XSS, and path traversal pattern matching, but `ExecuteCommand()` in `commands.go` does not call it.

**Impact:** Input validation exists but is not enforced at the command dispatch level.
**Fix:** Call `ValidateCommand()` at the entry point of `ExecuteCommand()`.

---

### M6. Room VNum Not Range-Validated in Wizard Commands
**File:** `pkg/session/wizard_cmds.go:62, 86, 145-149`
**Severity:** MEDIUM

`cmdGoto`, `cmdAt`, `cmdTeleport` parse room VNums with `strconv.Atoi()` but don't validate the destination room exists before acting.

**Impact:** Players teleported to nonexistent rooms. Game state corruption (player in room that doesn't exist).
**Fix:** Validate room existence via `world.GetRoom()` before any teleport.

---

### M7. `cmdSet` Stat Bounds Incomplete
**File:** `pkg/session/wizard_cmds.go:238-252`
**Severity:** MEDIUM

The `set` command caps HP/mana at 10000 only for players below level 60. Alignment has no bounds at all. High-level immortals can set any stat to `math.MaxInt`.

**Fix:** Enforce hard bounds regardless of level.

---

### M8. Equipment Nil Pointer Risk in `performRemove`
**File:** `pkg/game/item_equipment.go:273-290`
**Severity:** MEDIUM

`GetItemInSlot()` returns `(*ObjectInstance, bool)` — if a slot maps to nil, `objHasFlag(nil, ...)` would panic.

**Fix:** Add `if obj == nil { return }` after the found check.

---

### M9. Player Movement Not Atomic
**File:** `pkg/game/act_movement.go:287-293`
**Severity:** MEDIUM

Movement point deduction uses separate `GetMove()` and `SetMove()` calls, each with their own lock/unlock cycle. Another goroutine can modify Move between the two calls.

**Impact:** Movement points can go negative or be lost.
**Fix:** Use a single locked operation for check-and-deduct.

---

### M10. Mob Memory List Unbounded
**File:** `pkg/game/mobact.go:209`
**Severity:** MEDIUM

Mob AI iterates a memory list (`ch.Memory`) per mob per AI tick. No size cap found on the memory list.

**Impact:** If mobs accumulate unbounded memory entries, AI tick becomes expensive (O(memory × players_per_room) per mob).

---

### M11. `where` Command Ignores Invisibility
**File:** `pkg/session/commands.go:1339-1367`
**Severity:** MEDIUM

The `where` command shows all player locations without checking AFF_INVISIBLE.

**Impact:** Invisible players' locations are revealed.
**Fix:** Filter out invisible players unless viewer has detect-invisible.

---

## LOW Findings

### L1. Rate Limiter IP Map Cleanup Threshold Too High
**File:** `pkg/auth/ratelimit.go:35`
**Severity:** LOW

IP map is only cleaned when `len(ips) > 10000`. With X-Forwarded-For spoofing, this allows 10000 entries before cleanup — ~1MB of memory.

---

### L2. Legacy `ShopManager` in `pkg/game/shop.go` Lacks Mutex
**File:** `pkg/game/shop.go:14-37`
**Severity:** LOW

The legacy `ShopManager` has no mutex. The modern replacement in `pkg/game/systems/shop.go` does. Verify only the modern version is used in hot paths.

---

### L3. `SanitizeInput()` Uses Manual HTML Escaping
**File:** `pkg/validation/input.go:82-86`
**Severity:** LOW

Manual string replacement for `<`, `>`, `&` instead of using `html.EscapeString()`. Works but is less maintainable.

---

### L4. Database Connection Uses `sslmode=disable`
**File:** CLAUDE.md connection string
**Severity:** LOW

PostgreSQL connection string uses `sslmode=disable`. Acceptable for localhost development; must be `sslmode=require` in production.

---

### L5. Incomplete `handleCharInput()` Stub
**File:** `pkg/session/char_creation.go:15-26`
**Severity:** LOW

The character creation state machine is a no-op stub. The normal login path at `manager.go:468-484` correctly requires passwords. However, `completeCharCreation()` (line 93-119) would create passwordless accounts if ever called — currently unreachable dead code, but a latent vulnerability.

---

## INFO Findings

### I1. Bcrypt password hashing properly implemented (manager.go:451, 474)
### I2. Agent API keys use SHA-256 hash storage with crypto/rand generation (db/player.go:145-191)
### I3. JWT uses HS256 with mandatory 32+ char secret, 24h expiry (auth/jwt.go)
### I4. IP-based login rate limiting at 5 req/sec (auth/ratelimit.go:48)
### I5. Per-session command rate limiting at 10 cmd/sec (session/manager.go:175)
### I6. SQL queries use parameterized placeholders throughout (db/player.go)
### I7. Security headers (CSP, X-Frame-Options, HSTS) properly configured (web/security.go)
### I8. Audit logging with IP hashing for privacy (audit/logger.go)
### I9. Player name validation with reserved name rejection (validation/validation.go)
### I10. WebSocket read/write deadlines set (60s read, 10s write)
### I11. No pprof or debug endpoints exposed (cmd/server/main.go)

---

## Summary by Category

| Category | CRITICAL | HIGH | MEDIUM | LOW | Total |
|----------|----------|------|--------|-----|-------|
| Concurrency & Races | 4 | 3 | 1 | 0 | 8 |
| DoS / Resource Exhaustion | 1 | 4 | 1 | 1 | 7 |
| Economy / Item Integrity | 1 | 1 | 0 | 1 | 3 |
| Authentication & Sessions | 0 | 1 | 4 | 1 | 6 |
| Input Sanitization | 0 | 1 | 3 | 1 | 5 |
| Information Leaks | 0 | 0 | 1 | 1 | 2 |
| Misc | 1 | 0 | 1 | 0 | 2 |

---

## Recommended Fix Priority

### Immediate (before any public deployment)
1. **C4** — Add `conn.SetReadLimit(16384)` to prevent memory exhaustion
2. **C5** — Add per-IP connection limits in `HandleWebSocket`
3. **C1** — Copy inventory slice before iterating in `doDrop`/`doGive`
4. **C2** — Add `w.mu.RLock()` to all `w.players` iterations
5. **C6** — Mutex-protect gold transfer check-and-deduct

### Next sprint
6. **C3** — Protect `roomItems` reads in `doGet`
7. **C7** — Fix lock ordering in `handlePlayerDeath`
8. **H1** — Don't trust X-Forwarded-For without trusted proxy config
9. **H2** — Sanitize all communication command messages
10. **H3** — Enforce message size limits
11. **H5** — Make Register+AddPlayer atomic
12. **H7** — Lock mob during death item transfer

### Backlog
13. All MEDIUM findings
14. Wire `ValidateCommand()` into command dispatch (M5)
15. Add event queue size cap (H8)
16. Object pool overflow handling (H9)
