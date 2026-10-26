# Triage — OpenVuln report (16 findings), 2026-10-26

**Report:** VulnHunter/VulnForge `report.yaml`, rendered by OpenVuln. Keys `BUG-R2-C1-A1-H1` … `BUG-R2-C2-A1-H7`.
**Audited snapshot:** `61242f3068ad69d43d958734ba4903b2be856f12` (confirmed ancestor of HEAD).
**Triage target:** HEAD `d055edef8d40790b03ba572f211c78d47a5b2c75`.
**Method:** source verification of every cited call path at HEAD, plus `git log -S` for the fixing commit. Read-only; no files changed except this document.

## Bottom line

**The report is stale.** It was written against an older commit, and a large security-fix
sweep landed in between. Of 16 findings (8 high, 8 medium):

| Verdict | Count | Keys |
|---|---:|---|
| Fixed at HEAD (already remediated) | 6 | C1-A1-H1, C1-A1-H3, C1-A1-H4, C1-A4-H1, C1-A4-H3, C1-A4-H4 |
| False positive at HEAD (claim does not hold) | 1 | C1-A1-H2 |
| Fixed (was real at snapshot) | 2 | C2-A1-H1, C2-A1-H5 |
| **Still open** | **4** | **C2-A1-H3 (residual), C2-A1-H4, C2-A1-H2, C2-A1-H7** |
| Still open — real gap, report's mechanism wrong | 1 | C1-A3-H1 |
| Still open — real leak, mechanism confirmed | 1 | C1-A3-H2 |
| Still open — primitive only, impact neutralised | 1 | C2-A1-H6 |

The commits that did most of the remediation:

| Commit | What it closed |
|---|---|
| `fd7f7bb35` | C's `process_input` printable-ASCII filter at every intake (VULN-047/014/015) — kills all control-byte injection |
| `66ab0b86a` | every guest name is generated; the typed suffix is never used |
| `0f40b20eb` | sanitize inbound Grapevine broadcasts + cap relay frame size (VULN-039/041) |
| `327a0b31f` | every direct session send delivered under the close guard (guarded sends) |
| `be2b62c8e` | lock the world's concurrent gold, room-item, ban and gossip state (VULN-019/020/026/040/053/055) |
| `7c5bf00cd` | DP-912 pre-auth WebSocket idle clock (pongs no longer keep a pre-auth socket alive) |

> **Key-collision warning.** These keys are reused across audit rounds for *different* findings:
> the previous round's `BUG-R2-C1-A3-H1` was the guarded-send wedge (fixed by `327a0b31f`),
> and its `BUG-R2-C2-A1-H1` was mob-social rendering (fixed by `9481465d8`). Key-based dedup
> is therefore unreliable — verdicts below are by content, verified against HEAD.

## CWE-117 content-injection family — all closed

### BUG-R2-C1-A1-H1 (guest name → CRLF/ANSI into other players' terminals) — FIXED / FALSE POSITIVE at HEAD

Two independent layers now block it.

1. `pkg/session/session_login.go:69-87`: the guest branch no longer uses the supplied name at all —
   `guestName := fmt.Sprintf("Guest_%d", guestSeq.Add(1))` (line 77). The typed suffix is discarded;
   the in-code comment cites exactly the control-byte / lookalike-identity risk (DP-912). Fix: `66ab0b86a`.
2. `pkg/session/input_filter.go:15-37` ports C's `process_input` character pass — only `0x20-0x7E`
   survive. It is applied to structured frames at `pkg/session/session_pump.go:275-283`
   (`MsgLogin`, `MsgCommand`, `MsgLine`, `MsgCharInput`, `MsgPagerInput`) and to every telnet line at
   `TerminalLine`. CR/LF, ESC, NUL and high-bit bytes can no longer reach `handleLogin` on either transport.

The report's proposed patch (call `validation.IsValidPlayerName(guestName)` on the typed name) is also
wrong for this codebase: it validates a string the branch discards.

### BUG-R2-C1-A1-H3 (guest name → mudlog / server log / Grapevine presence) — FIXED / FALSE POSITIVE at HEAD

Same two commits. `MudLog` (`pkg/game/logging.go:138`) still does not sanitize `str`, but no
player-controlled control byte can reach it now: names are validated (`validation.IsValidPlayerName`,
`[a-zA-Z0-9_]{2,32}`) or generated, and all input is filtered at intake. Optional defence-in-depth:
sanitize at `MudLog` — not required.

### BUG-R2-C1-A1-H4 (Grapevine inbound/outbound injection) — FIXED

- Inbound: `pkg/grapevine/client.go:422-428` runs `stripControlChars` over `bc.Name`, `bc.Game` and
  `bc.Message` before the `fmt.Sprintf` (`0f40b20eb`, VULN-039), and `conn.SetReadLimit(maxMessageSize)`
  at line 186 (VULN-041) closes the "no read limit" secondary exposure.
- Outbound: `sendGossip` still marshals raw strings, but its only source is `w.OnGossip(ch.Name, argument)`
  (`pkg/game/comm_channel.go:215`) where the name is validated/generated and the argument passed through
  `cInputLine` at intake — interior CR/LF is no longer reachable from the JSON protocol.
- Residual (minor, hardening): `stripControlChars` keeps every rune `>= 0x20 && != 0x7f`, so Unicode
  bidi/zero-width spoofing (U+202E and friends) still passes. Visual spoofing only — not CRLF/ANSI injection.

### BUG-R2-C1-A1-H2 (prefix match → capture a registered `guest*` identity) — FALSE POSITIVE at HEAD

The `strings.HasPrefix(strings.ToLower(login.PlayerName), "guest")` test survives at
`pkg/session/session_login.go:69`, but the supplied name is **discarded** (line 77 generates
`Guest_<counter>`). The claimed capability — an anonymous client becoming a live character under a
registered `guest*` name — no longer exists, so the impersonation, private-tell capture and owner
lockout described in the report do not follow from this code.

Residual (real, narrow, low severity): a stored account whose name merely starts with `guest` — which
the report itself shows can only be minted out-of-band (direct SQL edit, or imported pre-guest-mode
data) — is still routed to the guest branch and receives a generated guest body instead of its account.
That is the documented DP-1379 "approved guest-prefix transport routing", so changing it is a design
decision for Zach, not a security fix.

## CWE-400 resource-exhaustion family — three fixed, two open

### BUG-R2-C1-A4-H1 (WS pre-auth bare blocking sends wedge readPump) — FIXED

`sendError` → `sendGuarded` (`pkg/session/session_send.go:54-64`); `sendGuarded` itself
(`session_send.go:190-204`) takes `sendMu.RLock`, checks `sendClosed` and uses `select`+`default`.
`sendCharCreatePromptWithSecret` and `sendStatsRollPrompt` → `sendGuarded`
(`pkg/session/char_creation.go:453,522`). A full sweep of every `s.send <-` site in non-test code
(18 sites across `session_send.go`, `manager.go`, `session_idle.go`, `session_manager.go`,
`heartbeat_output.go`, `agent_vars.go`, `gmcp.go`, `cmd_group.go`, `terminal.go`) shows **all** of them
inside a `select` with `default`; only `use_cmds_hold_test.go` and `session_send_guarded_test.go` have
bare sends. The `sendGuarded` doc comment names this exact wedge. The park is gone.

### BUG-R2-C1-A4-H3 (telnet wedge → 200-slot total-cap lockout) — FIXED

Same root cause removed: the telnet read goroutine executes the state machine synchronously, but the
pre-auth char-creation sends are now non-blocking, so it can no longer park on a full channel and the
`handleConn` deferred accounting (`connCount` / `connPerIP`) always runs. `stopWriter`
(`pkg/telnet/listener.go:351-356`) already arms a 5 s write deadline before `CloseSend`, and
`readLinePreAuth` keeps the 120 s DP-912 deadline.

Residual hardening (not a wedge): `writeLoop`'s `tc.write` still has no per-write deadline of its own,
so a zero-window client can stall the *writer* goroutine. That now affects only that one goroutine and is
bounded by `stopWriter` + the read deadline. Worth adding a per-message write deadline for symmetry with
`writePump`'s 10 s, but no permanent wedge or slot leak results.

### BUG-R2-C1-A4-H4 (game-loop park → whole-service freeze) — FIXED

`sendObservation` → `sendGuarded` (`pkg/session/observation.go:81`); char-creation prompts → `sendGuarded`;
`heartbeatFrame.deliver` (`pkg/session/heartbeat_output.go:197-213`) uses `select`+`default`; the
broadcast/combat sinks at `manager.go:369,556,2064` are guarded. The single-threaded heartbeat
(`pkg/engine/gameloop.go`) can no longer park on a full channel, so the "frozen world" failure mode is closed.

### BUG-R2-C1-A3-H2 (WS churn strands writePump for ~54 s) — **STILL OPEN** (confirmed)

Mechanism verified at HEAD:

- `readPump`'s deferred teardown (`pkg/session/session_pump.go:83-124`) ends in
  `finishWebSocketTransport()` → `HandleTransportDisconnect` (returns false for `!s.authenticated`,
  before it reaches `DetachTransport`) → `UnregisterSession` → `unregisterSession`, which for an
  unregistered session runs **only `s.cancelFunc()`**. It never closes `s.send`.
- `writePump` (`session_pump.go:199-249`) can therefore only exit via `<-s.send` (never closed),
  `<-s.TransportDone()` (`transportDone` is closed only by `DetachTransport`, unreachable pre-auth),
  or the 54 s ticker (line 178/243) whose ping write to the already-closed socket fails.

Each abruptly-closed pre-auth connection therefore pins one goroutine, the `Session` (256-slot send
channel ≈6 KB) and the retained `*http.Request` (`request: r`) for up to 54 s. `MaxHeaderBytes` is not
overridden (`cmd/server/main.go:1051-1057`), so the retained header map is worth up to Go's 1 MB default
per session. Steady state ≈ `churn_rate × 54 s`. It also fires benignly whenever a browser tab closes
before login. The report's proposed fix — `CloseSend()` in `readPump`'s teardown when the session is not
retained — is sound. **Real finding.**

### BUG-R2-C1-A3-H1 (no process-wide WebSocket total-connection cap) — **STILL OPEN** (gap real, mechanism wrong)

- Gap confirmed: `HandleWebSocket` (`pkg/session/manager.go`) still gates on only
  `webSocketMaxConnsPerIP = 8` (`manager.go:48,1054`), and `GetIPFromRequest`
  (`pkg/auth/ratelimit.go`) returns a full-precision address with no IPv6 /64 aggregation, so source
  rotation defeats it. Telnet has `maxTotalConns = 200`; WS has no equivalent.
- **Mechanism corrected:** the report's central claim — that one 2-byte pong per 54 s keeps a pre-auth
  session alive indefinitely — is false at HEAD. `7c5bf00cd` added the DP-912 pre-auth idle clock
  (`preAuthIdleTimeoutNanos`, default 120 s, `LOGIN_IDLE_TIMEOUT`) and the pong handler deliberately
  does not refresh it before authentication (`session_pump.go:128-136`); the deadline is refreshed only
  by data frames. Holding N sessions now costs one small frame per <120 s each — still cheap, so the
  FD/memory-exhaustion flood remains feasible, but it is no longer "free forever".

Recommend a total cap plus a warn log (mirroring telnet's), which is the correct defence regardless of
the keepalive cost.

## CWE-362/367 economy-race family — two fixed, five open

### BUG-R2-C2-A1-H1 (performGiveGold check-then-debit TOCTOU) — FIXED

`performGiveGold` (`pkg/game/item_transfer.go:499`) no longer does a check followed by a separate
read-modify-write: it calls `ch.SpendGold(amount)`, which takes `p.mu` once across the sufficiency check
and the debit, and credits with the atomic counterpart. `pkg/game/player_identity.go` now provides
`SpendGold` (line 66) and `AddGold` (line 77) alongside `Get/SetGold`. The concurrent-drain-lane
double-credit is closed.

### BUG-R2-C2-A1-H5 (shop_cmds raw `Player.Gold`) — FIXED

`pkg/session/shop_cmds.go` no longer touches the field: `GetGold()` at 440, `SpendGold()` at 474,
`AddGold()` at 488, 574 and 613. No raw `.Gold` arithmetic remains in the file.

### BUG-R2-C2-A1-H3 (unlocked `w.roomItems` reads → unrecoverable runtime fatal error) — MOSTLY FIXED, ONE REAL SITE LEFT

Fixed: `getFromRoom` (`item_transfer.go:150`) now takes a locked snapshot via `w.GetItemsInRoom`
(the VULN-019 fix), and both `doGet` container-lookup arms are covered. Verified locked at HEAD:
`shoot.go:82` (inside `TransferRangedVictim`, which holds `w.mu` from line 64), `movement.go:266`,
`world_scriptable.go:384/415/508`, `char_mgmt.go:265-268`, `world_object.go:65/79/179/191`.

**Residual — `pkg/game/item_container.go:65`:**

```go
for _, obj := range w.roomItems[ch.RoomVNum] {   // doPut, no lock
```

`pkg/game/item_container.go` contains **zero** locking (no `mu.` anywhere in the file), and `doPut` is a
player command handler running on the connection's read goroutine while every `roomItems` writer holds
`w.mu`. This is the same class the finding describes: a `fatal error: concurrent map read and map write`
is a runtime *throw*, not a panic, so neither the `readPump` recover nor `gameloop.safeInvoke` can catch
it — the single process hosting the game, both listeners, the admin API and the DB dies. It also reads
`ch.RoomVNum` raw instead of `ch.GetRoomVNum()`. **Highest-value residual in the batch: a one-line fix
for a whole-process crash class.**

### BUG-R2-C2-A1-H4 (specBank bank-balance TOCTOU) — **STILL OPEN** (confirmed)

`pkg/game/spec_procs4.go:596-635` is unchanged in shape:

```go
if ch.GetGold() < amount { ... }          // deposit check        :612
ch.SetGold(ch.GetGold() - amount)         // :616  separate RMW
ch.SetBankGold(ch.GetBankGold() + amount) // :617  separate RMW
...
if ch.GetBankGold() < amount { ... }      // withdraw check       :629
ch.SetGold(ch.GetGold() + amount)         // :633
ch.SetBankGold(ch.GetBankGold() - amount) // :634
```

`pkg/game/player_identity.go` has no `WithdrawBankGold`/`DepositGold` helper, so the check and both
mutations remain separate `p.mu` acquisitions. Two concurrent `withdraw`/`deposit` commands (heartbeat
drain lane vs readPump fast lane, or the H6 order lane) can both pass the check → minted carried gold,
or a persisted negative `BankGold`. `pkg/game/spec_bank_test.go` exists but tests output fidelity, not
atomicity. Fix: add `WithdrawBankGold`/`DepositGold` returning the amount moved, and call them.

### BUG-R2-C2-A1-H2 (getCheckMoney money-pile TOCTOU) — **STILL OPEN**

The snapshot read is fixed, but the consume-once guard is not:

```go
// item_transfer.go:136 — no obj.Location.Kind == ObjNowhere re-validation
func (w *World) performGetFromRoom(ch *Player, obj *ObjectInstance) { ... }

// item_transfer.go:40 — extraction then an unguarded, non-atomic credit
func (w *World) getCheckMoney(ch *Player, obj *ObjectInstance) {
	amount := obj.GetValue(0)
	if obj.GetTypeFlag() == ITEM_MONEY && amount > 0 {
		w.ExtractObject(obj, ch.GetRoomVNum())   // idempotent: extractObjectLocked no-ops
		...
		ch.SetGold(ch.GetGold() + amount)        // credited for every getter that arrives
	}
}
```

`extractObjectLocked` (`world_object.go`) sets `LocNowhere()` and deletes the registry entry, but the
second getter's `MoveObjectToPlayerInventory` still attaches the instance (detach no-ops on a stale
location, attach has no existence check), so both getters reach the credit line. The report's proposed
fix — skip already-extracted objects in `performGetFromRoom` and make extraction+credit one critical
section under `w.mu` — is the right shape.

### BUG-R2-C2-A1-H7 (stealInventoryItem out-of-lock transfer → item duplication) — **STILL OPEN**

`pkg/game/skill_stealth.go:221` is unchanged:

```go
if !removeCarriedItem(target, item) { ... }   // Inventory.RemoveItem, does not clear Location
if err := ch.Inventory.AddItem(item); err != nil { ... }
item.Location = LocInventoryPlayer(ch.Name)   // re-pointed last, no w.mu anywhere
```

A concurrent `give`/`put`/`drop`'s `MoveObject` (which holds `w.mu`) can detach via the stale
`item.Location` — a silent no-op — and attach the same `*ObjectInstance` to a second inventory, after
which the steal's own `AddItem` also adds it: one instance reachable from two `Inventory.Items` lists,
persisted by the file and DB save seams. Fix as the report proposes: route the transfer through
`world.MoveObject(item, LocInventoryPlayer(ch.Name))`.

### BUG-R2-C2-A1-H6 (order/charm cross-goroutine command re-entry) — **STILL OPEN as a primitive; impact largely neutralised**

`SetCommandExecFunc` (`pkg/session/manager.go:995-1005`) still runs the ordered command synchronously on
the caller's goroutine (`ExecuteCommand(sess, command, nil)`), and `cmd_combat_special.go:117-118` still
dispatches through it, so the same-player two-goroutine primitive exists exactly as described. Its
exploit value has largely gone, though: the give-gold sink it drove is now atomic (H1) and the shop sink
is fixed (H5). It still composes with the still-open bank TOCTOU (H4), which is already reachable
without it. Treat as a latent hardening item (serialize ordered commands onto the victim's own drain
lane), not an independent exploit.

## Open work, ranked

| # | Item | Class | Fix cost | Why this rank |
|---|---|---|---|---|
| 1 | `item_container.go:65` unlocked `roomItems` read (H3 residual) | crash (CWE-362) | 1 line | Unrecoverable whole-process kill; one-line lock |
| 2 | `specBank` TOCTOU (H4) | economy (CWE-367) | ~30 lines | Deterministic minting/negative balance, persists |
| 3 | `getCheckMoney` consume-once (H2) | economy (CWE-367) | ~15 lines | Minting from nothing, persists |
| 4 | `stealInventoryItem` → `MoveObject` (H7) | duplication (CWE-362) | ~20 lines | Arbitrary item duplication, persists |
| 5 | WS writePump 54 s strand (A3-H2) | resource leak (CWE-400) | ~6 lines | Cheap, attacker-scalable leak; also benign-triggered |
| 6 | WS total-connection cap (A3-H1) | flood hardening (CWE-400) | ~25 lines | Correct defence; report's mechanism corrected |
| 7 | Order-lane serialization (H6) | latent (CWE-362) | design choice | Primitive only; sinks fixed |
| 8 | Minor hardening | — | — | Guest-prefix exact-match (H2 residual, design call); Grapevine bidi/zero-width; `writeLoop` per-write deadline; optional `MudLog` sanitize |

## Census guidance (R5h)

These are player-facing behaviour changes, so a **targeted** census is required for the fixes, not a full
one — the final full/combined release census is batched and run by Claude at release time. Name the
scenarios that can fail on each change; the WS/telnet resource items and the container-lock line are
covered by existing session/transport tests plus a `-race` run, and the economy items by
`pkg/game/spec_bank_test.go` plus new atomicity tests. Do not start a full census per fix.

## Addendum — implementation (2026-10-26) and one new finding

### PR 1 — `pkg/game` concurrency/economy — [#2175](https://github.com/zax0rz/darkpawns/pull/2175)

Branch `cline/vuln-races`, worktree `~/wt/cline-vuln-races`, commit `ffc6181ae`.

Closes the four open CWE-362/367 items:

| Finding | Change |
|---|---|
| C2-A1-H4 | new `Player.DepositGold`/`WithdrawBankGold` hold `p.mu` across the check and both writes; `specBank` calls them |
| C2-A1-H2 | `performGetFromRoom` skips an already-extracted object; `getCheckMoney` makes extraction+credit one `w.mu` section |
| C2-A1-H7 | the carried steal's transfer runs under `w.mu` (flags and the capacity-refusal path unchanged) |
| C2-A1-H3 residual | `doPut`'s room-container lookup uses the locked `GetItemsInRoom` snapshot |

R5h evidence — with only those call sites reverted (helpers kept, so it compiles), the new
tests fail:

```
TestBankConcurrentWithdrawPaysOnce: gold=100 bank=-100, want 100/0
TestGetCheckMoneyConsumesOnce:      one 7-coin pile credited 14 gold, want 7
TestGetFromRoomStaleSnapshotCreditedOnce: one 100-coin pile credited 200 gold, want 100
TestPutRoomLookupConcurrentWithRoomWrites: FAIL under -race (unlocked map read)
TestStealCarriedTransferHoldsWorldLock: "completed while w.mu was held"
```

The steal test is deterministic rather than a timing lottery: it holds `w.mu` itself and asserts
the steal cannot complete until the lock is released. A concurrent give/steal race test was
discarded because it passed both before and after the fix — too narrow to be a real gate.

### PR 2 — `pkg/session` WebSocket resources — [#2176](https://github.com/zax0rz/darkpawns/pull/2176)

Branch `cline/vuln-ws-resources`, worktree `~/wt/cline-vuln-ws-resources`, commit `ebe6fc88e`.

Closes C1-A3-H2 (`finishWebSocketTransport` now calls `CloseSend` in the non-retained branch —
the waker the telnet and menu teardown callers already use) and C1-A3-H1 (`webSocketMaxTotalConns`,
default 500, `WEBSOCKET_MAX_TOTAL_CONNS`; counter under `ipConnMu`, released through the existing
`connCountDecremented` latch so exactly one teardown path decrements). Gates re-run independently:
build/vet/`go test ./pkg/session/... ./pkg/telnet/...`/`-race` all clean.

### NEW finding (not in the report) — inventory writes bypass `Inventory.mu`

`go test -race ./pkg/game/` surfaced a genuine, unreported, pre-existing data race:

```
Write at ... by goroutine:
  Inventory.removeItem()          pkg/game/inventory.go:59   <- NO inventory lock held
  World.detachObjectLocked()      pkg/game/movement.go:33
  World.moveObjectLocked()        pkg/game/movement.go:184
  World.MoveObject()              pkg/game/movement.go:240
Previous read at ... by goroutine:
  Inventory.FindItems()           pkg/game/inventory.go:106  <- takes inv.mu.RLock()
  findCarriedItem()               pkg/game/skill_stealth.go:378
```

`moveObjectLocked`/`detachObjectLocked`/`attachObjectLocked` mutate a player's `Inventory.Items`
through the **no-lock** internals (`removeItem`, `addItem`) holding only `w.mu`, while readers use
the locking API (`FindItems`, `Snapshot`, `GetItemCount`). Because the writer never takes `inv.mu`,
the readers' `RLock` excludes nothing, so any give/put/drop/corpse transfer concurrent with any
locking inventory read is a data race — torn slice headers can drop or duplicate items. The same
applies to `extractObjectLocked`'s `p.Inventory.removeItem` calls.

This is independent of the 16 findings (it exists on `origin/main` untouched by PR 1) and needs its
own fix: make the movement path use the locking inventory API (lock order is already
`w.mu → Equipment.mu → Inventory.mu`), after auditing every `*Locked` caller for a caller that
already holds `inv.mu`. Filed here as a lead rather than fixed, because the change is broad and
deadlock-sensitive.

### Still open after both PRs

- **C2-A1-H6** (order/charm cross-goroutine re-entry): primitive remains; its sinks are now atomic.
- **C1-A3 minor**: Grapevine `stripControlChars` still admits bidi/zero-width runes; telnet
  `writeLoop` has no per-write deadline (deliberate — DP-1385 removed the playing-descriptor
  transport deadline, so adding one is a fidelity decision, not a bug fix).
- **Equipped-steal same class as H7**: `stealEquippedItem`'s unequip→add→re-locate is still not
  under `w.mu`, but it stages through a temporary inventory and carries light accounting, so its
  correct fix is not the mechanical one used for the carried path — it needs its own design.
- **C1-A1-H2 residual** and the **A3-H1 mechanism correction** remain as documented above.


## Notes for the reporter

1. The report is against a superseded commit; six of sixteen findings were already remediated before it
   was triaged. Re-audit against HEAD `d055edef8` before the next round.
2. Finding keys collide with the previous round's keys (`BUG-R2-C1-A3-H1`, `BUG-R2-C2-A1-H1` mean
   different things there), so key-based dedup is unsafe — please version the keys.
3. Two findings' mechanisms are wrong even though the underlying gap is real: C1-A3-H1's "one pong keeps
   a pre-auth socket alive forever" is false since `7c5bf00cd`, and C1-A1-H1/H2's payloads cannot reach
   `handleLogin` at all since `fd7f7bb35`.
4. The PoC artefacts the report cites (`findings/…/poc/`) are not in this tree, so the dynamic claims
   could not be re-run; all verdicts above are from source verification at HEAD.



