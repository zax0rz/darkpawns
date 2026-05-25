---
tags: [active, reference, implementation]
last_updated: 2026-05-25
---
# Remaining Implementation Brief

For Gemini. Linear issue IDs included for cross-reference. Each section is self-contained — read the relevant audit sections in `docs/technical-audit-2026-05-25.md` for code-level detail.

---

## DP-304: Subnegotiation Length Cap

**Priority:** Must land with DP-303 (GMCP)
**Effort:** S (~30 LOC)
**File:** `pkg/telnet/listener.go`
**What:** The `case SB:` skip-loop (listener.go:435-451) reads bytes until `IAC SE` with no length bound. Fine when discarding, dangerous when buffering GMCP payloads. A malicious client could stream an unterminated subneg and exhaust memory.

**Implementation:**
- Add `const maxSubnegLen = 4096` (GMCP payloads are small JSON strings)
- In the `case SB:` branch, count bytes read. If exceeding `maxSubnegLen` before `IAC SE`, close the connection
- This is a safety prerequisite for DP-303

---

## DP-303: GMCP (Generic MUD Communication Protocol)

**Priority:** High — last protocol domino
**Effort:** M (~150 LOC for telnet subneg + dispatch; data model reused)
**Files:** `pkg/telnet/listener.go`, `pkg/session/agent_vars.go`, `pkg/session/manager.go`
**Depends on:** DP-304

**What:** The agent var system (`agent_vars.go`) is already a working subscribe + dirty-track + delta-flush system over 19 variables (HEALTH, ROOM_VNUM, ROOM_NAME, etc.). GMCP is the same idea with a different envelope. This is adaptation, not invention.

**Implementation — Telnet:**
1. Add `OPT_GMCP byte = 201` to listener constants
2. Advertise `IAC WILL OPT_GMCP` on connect
3. In `case SB:` when option byte is 201: buffer payload (with DP-304's length cap), split on first space into `Package.Message` + JSON, dispatch
4. Add `sendGMCP(pkg string, data interface{})` — marshal JSON, write `IAC SB 201 [pkg] [json] IAC SE` through `wmu` lock
5. On receiving `IAC DO OPT_GMCP`: set `tc.hasGMCP = true` and set session's `wantsStructuredData = true`

**Implementation — Capability System:**
1. Add `wantsStructuredData bool` to `Session` struct in `manager.go`
2. Modify `markDirty` and `flushDirtyVars` in `agent_vars.go`:
   ```go
   if !s.isAgent && !s.wantsStructuredData {
       return
   }
   ```
3. **CRITICAL:** Keep `VarEvents` strictly gated to `isAgent` — it's a memory accumulator for agent telemetry. If humans get it without draining, that's a leak.

**Implementation — Var → GMCP Mapping:**
Map existing `buildVarValue` output to GMCP namespaces:
- `Char.Vitals`: VarHealth, VarMaxHealth, VarMana, VarMaxMana, VarMove, VarMaxMove
- `Char.Status`: VarLevel, VarGold
- `Room.Info`: VarRoomVnum, VarRoomName, VarRoomExits
- `Room.Mobs`: VarRoomMobs
- `Room.Items`: VarRoomItems
- `Char.Inventory`: VarInventory

**Implementation — Web Client:**
The web client already receives `vars` messages over WebSocket (agent path). Un-gating `wantsStructuredData` means human sessions get the same `vars` messages. The client's `handleVarsMsg` already parses them. The mana/move/gold status bars (DP-301) will update from these vars instead of raw PlayerState.

**Risks:**
- The `isAgent` gate is load-bearing. Treat un-gating as a capability-flag refactor with tests, NOT a `grep -v isAgent`
- The `send` channel is 256-buffer with drop-on-full. GMCP increases message volume. Validate buffer sizing (DP-311)
- `sendGMCP` must go through `wmu` lock — same as `write()` does. Combat ticker goroutine calls `flushDirtyVars` which calls `sendGMCP`, so concurrent write protection is essential

**Verification:** Connect with Mudlet. It negotiates GMCP on connect and expects `Char.Vitals`, `Room.Info` packages. If Mudlet auto-detects our vitals, the protocol is correct.

---

## DP-306: GMCP-Driven Web Client UI Enhancements

**Priority:** Medium
**Effort:** M
**Depends on:** DP-303
**Files:** `website/assets/js/client.js`, `web/client.js`
**What:** Once GMCP feeds structured data to the web client, expand beyond the basic status bar.

**Enhancements:**
1. **Minimap** — reuse `website/static/map/world.json` + `Room.Info` GMCP package. Show player's current zone with room highlighted. Already have the data.
2. **Inventory panel** — from `Char.Inventory` GMCP. Grid or list view of carried items.
3. **Equipment panel** — from `Char.Status` GMCP. Paperdoll-style equipment slots.
4. **Spell/cooldown bar** — from spell affect data in GMCP. Active buffs, cooldown timers.
5. **Target/enemy health bar** — from combat GMCP data. Show当前 target's HP.
6. **Room contents panel** — from `Room.Mobs`/`Room.Items` GMCP. What's in the room besides you.

All data already exists server-side in `agent_vars.go` (`buildInventory`, `buildEquipment`, `buildRoomMobs`). The client just needs to render it.

---

## DP-309: Graceful Shutdown

**Priority:** Medium
**Effort:** S-M
**File:** `cmd/server/main.go`
**What:** No graceful shutdown — in-flight WebSocket and telnet connections are not drained on SIGTERM. A press-driven traffic spike followed by a deploy drops every player ungracefully. Worth hardening before a publicized launch.

**Implementation:**
- Signal handler catches SIGTERM/SIGINT
- Stop accepting new connections (close listeners)
- Drain existing sessions with a timeout (e.g., 30 seconds)
- Notify connected players of shutdown
- Await zone-reset and world-save
- Exit

---

## DP-311: Send Channel Buffer Validation ✅ Done

**Finding:** Non-issue. `flushDirtyVars()` already batches ALL dirty vars into a single `{"type":"vars",...}` message per flush — not one message per variable. During a combat tick where only HEALTH/MAX_HEALTH are dirty, exactly one message enters the send channel. At 1 msg/tick the 256-slot buffer handles any realistic player count.

**What was implemented (2026-05-25):** `pkg/telnet/listener.go` — extracted `buildGMCPFrame()` helper and refactored the "vars" branch of `writeLoop` to collect all GMCP package frames into a single byte slice and call `tc.write()` once, instead of 3–5 separate locked writes. This reduces syscall overhead when multiple packages are dirty simultaneously (e.g., room change on `look`). No buffer resize, no priority queuing needed.

---

## DP-310: Register on MUD Directories

**Priority:** Medium (manual work, after MSSP is live)
**Effort:** S (time, not code)
**What:** MSSP is implemented (DP-302). Now register the server on directories.

**Directories to register:**
- Grapevine (grapevine.haus) — also enables cross-MUD chat (DP-307)
- MUDStats — player count tracking
- MUDverse — directory listing
- The Mud Connector — update existing listing if any
- MUD Listings — manual submissions

**Action:** Visit each directory's registration page, submit server details. MSSP crawler should auto-detect uptime and player counts after first crawl.

---

## Map Enhancements (Independent)

These are in `docs/proposals/WORLD-MAP-ENHANCEMENTS.md`. Independent of protocol work — can run in parallel.

- **DP-312** Zone name labels at medium zoom (S, highest impact)
- **DP-313** Dim unreachable rooms (S, data honesty)
- **DP-314** Sector territory washes (M, visual transformation)
- **DP-315** Inter-zone link visualization (S-M, topology)
- **DP-316** Connection density clustering (M-L, depends on DP-315)

---

## Dependency Graph

```
DP-304 (subneg cap)
  └→ DP-303 (GMCP)
       ├→ DP-306 (web UI enhancements)
       ├→ DP-311 (buffer validation)
       └→ DP-310 (directory registration — after GMCP + MSSP indexed)

DP-309 (graceful shutdown) — independent
DP-312–316 (map) — independent
```

## Files Reference

- Technical audit: `docs/technical-audit-2026-05-25.md`
- Marketing strategy: `docs/outreach/MARKETING-STRATEGY.md`
- Map enhancements: `docs/proposals/WORLD-MAP-ENHANCEMENTS.md`
- This brief: `docs/remaining-implementation-brief.md`
