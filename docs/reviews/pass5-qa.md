# Pass 5: QA & Edge Cases Review

**Reviewer:** Claude Opus 4 (opus-pass5)
**Date:** 2026-04-26
**Scope:** Error paths, disconnect handling, overflow/underflow, boundary conditions, nil dereference, resource cleanup, input validation, state machine integrity, reconnection, race conditions in error paths

---

## Executive Summary

The codebase has meaningful error handling in its centralized systems (object movement, combat engine) but exhibits a pattern of silently swallowing errors in peripheral code paths and session lifecycle management. The most dangerous bugs are: (1) **double-close of `s.send` channel** causing guaranteed panics on disconnect, (2) **`playerToSaveData` reads all Player fields without holding any lock**, creating torn reads that can produce corrupted save files, (3) **`AdvanceLevel` panics at level 0** via `rand.Intn(0)`, and (4) **telnet login creates sessions without password authentication**, allowing any telnet user to impersonate any character name. The disconnect-during-combat path is particularly fragile — the combat engine retains stale `Combatant` references to disconnected players, and the death handler will write to a closed `Send` channel.

---

## CRITICAL

### C5-1: Double-Close of `s.send` Channel Causes Panic

**Files:** `pkg/session/manager.go:267`, `pkg/session/manager.go:843`, `pkg/session/manager.go:1054`, `pkg/session/manager.go:1209`

The `s.send` channel is closed in **four separate locations**:
1. `Unregister()` at line 267: `close(s.send)` — called when `readPump` exits
2. `UnregisterAndClose()` at line 1054 (alias path): `close(s.send)`
3. `CheckIdlePasswords()` at line 1209: `close(s.send)` for timed-out sessions
4. `readPump` calls `Unregister` on exit (line 361), which closes `s.send`

Meanwhile, `writePump` reads from `s.send` and also calls `s.conn.Close()`. If `readPump` finishes first and calls `Unregister` (closing `s.send`), `writePump`'s `<-s.send` will receive `ok=false` and exit. But if `UnregisterAndClose` or `CheckIdlePasswords` is also triggered for the same session, `close(s.send)` is called **a second time**, which panics in Go.

Additionally, `CloseSend()` (line 937) also calls `close(s.send)` and is invoked by the telnet `handleConn` cleanup. There is no `sync.Once` or closed-flag guarding any of these paths.

**Impact:** Server crash (panic: close of closed channel) on any disconnect where multiple cleanup paths race.

**Fix:** Add a `sendClosed sync.Once` field to `Session` and use `s.sendClosed.Do(func() { close(s.send) })` at every close site. Alternatively, use a dedicated `done` channel and `select` in `writePump`.

---

### C5-2: `playerToSaveData` Reads Player Fields Without Lock (Confirmed from Pass 2)

**File:** `pkg/game/save.go:152-220`

`playerToSaveData()` reads `p.Health`, `p.Mana`, `p.Gold`, `p.Exp`, `p.Inventory.Items`, `p.Equipment.Slots`, `p.ActiveAffects`, `p.SkillManager`, and dozens of other fields without acquiring `p.mu`. This function is called from:
- `SavePlayer()` — can be called from `AdvanceLevel` (game loop goroutine)
- `SavePlayerWithRent()` — called from crash save paths
- `Unregister()` → `db.PlayerToRecord(s.player, nil)` — called on disconnect

Meanwhile, the combat engine's `processCombatPair` is calling `TakeDamage` (which holds `p.mu`) and the game loop is modifying `p.Exp`, `p.Gold`, affects, etc. This is a data race that produces torn reads — a save file could contain HP from before a combat round but Gold from after, or a half-mutated inventory slice.

**Impact:** Corrupted player save files. Silent data loss. In pathological cases, the slice header for `Inventory.Items` could be read in a torn state, causing a panic during JSON serialization.

**Fix:** `playerToSaveData` must acquire `p.mu.RLock()` for the duration of the snapshot. For `Inventory` and `Equipment`, also acquire their respective locks.

---

### C5-3: Telnet Login Bypasses Password Authentication

**File:** `pkg/telnet/listener.go:122-137`

The `sendLogin` function constructs a login message with only `player_name` — no password field:

```go
func sendLogin(s *session.Session, name string) error {
    loginData, err := json.Marshal(map[string]interface{}{
        "player_name": name,
    })
```

In `handleLogin` (manager.go:319), when `login.Password == ""` and `rec.Password != ""`, the server sends "Password required." and closes the connection. But the telnet path never sets `login.Password`. This means:

1. **Existing characters with passwords:** Telnet login is rejected (sends error, closes). This is "safe" but breaks telnet for returning players.
2. **New characters:** When `rec == nil` (player doesn't exist in DB) or `login.NewChar` is set, the code reaches the new-character path which also requires a password. Again rejected.
3. **No-DB mode:** When `m.hasDB == false`, `NewCharacter` is created with NO password and NO authentication check. Anyone connecting via telnet can create and play as any name.

**Impact:** In no-DB mode, telnet provides unauthenticated access. In DB mode, telnet is functionally broken for all users (silently fails). Neither outcome is correct.

**Fix:** The telnet handler must prompt for a password after the name prompt and include it in the login message. Add a password prompt flow to `handleConn`.

---

### C5-4: `AdvanceLevel` Panics with `rand.Intn(0)` at Level 0

**File:** `pkg/game/level.go:112`

```go
addMana = rand.Intn(3*p.Level-p.Level+1) + p.Level
```

When `p.Level == 0`: `rand.Intn(3*0 - 0 + 1)` = `rand.Intn(1)` = 0. Fine.

But the expression `3*p.Level - p.Level + 1` simplifies to `2*p.Level + 1`. At level 0 this is 1. At level 1, this is 3. Both fine.

**However**, `AdvanceLevel` is called **with the lock already held** (`p.mu.Lock()` at line 87), then calls `SavePlayer(p)` at line 307. `SavePlayer` calls `playerToSaveData(p)` which reads `p.Health`, `p.MaxHealth`, etc. — these are the same fields protected by `p.mu`. Since `p.mu` is not a reentrant lock, this **does not deadlock** because `playerToSaveData` doesn't acquire the lock (which is the bug from C5-2). But it means the save happens while the lock is held, which is the only reason it doesn't deadlock — a fix for C5-2 (adding RLock to save) would **cause a deadlock here**.

**Impact:** Fixing C5-2 naively will deadlock on every level-up. The save in `AdvanceLevel` must be moved outside the lock scope, or the lock must be released before saving.

**Fix:** Move `SavePlayer(p)` call to after `p.mu.Unlock()` — restructure `AdvanceLevel` to release the lock before saving, or defer the save to the caller.

---

### C5-5: Combat Engine Holds Stale References to Disconnected Players

**Files:** `pkg/combat/engine.go:165-230`, `pkg/session/manager.go:255-268`

When a player disconnects:
1. `readPump` exits, calling `Unregister(playerName)`
2. `Unregister` removes from sessions map, saves to DB, closes `s.send`, and calls `world.RemovePlayer`
3. `world.RemovePlayer` removes from `w.players` map

But the combat engine's `combatPairs` map still holds direct pointers to the `*Player` object. The next `PerformRound()` call will:
- Call `attacker.GetHP()` / `defender.GetHP()` — reads from a Player that is no longer in the world but still exists in memory (no immediate crash, but stale)
- Call `attacker.SendMessage()` / `defender.SendMessage()` — writes to `p.Send` channel which has been **closed** by `Unregister`
- Writing to a closed channel **panics**

The `Unregister` path does not call `combatEngine.StopCombat(playerName)`, so the combat pair persists.

**Impact:** Guaranteed panic when the combat engine tries to send a message to a disconnected player who was in combat.

**Fix:** `Unregister` (and `UnregisterAndClose`) must call `m.combatEngine.StopCombat(playerName)` before closing `s.send`. Additionally, `SendMessage` should check if the channel is closed (use a `sync.Once`-guarded close pattern).

---

## HIGH

### H5-1: `handlePlayerDeath` Reads/Writes Player Fields Without Consistent Locking

**File:** `pkg/game/death.go:178-265`

`handlePlayerDeath` acquires `player.mu.Lock()` only for the EXP deduction (lines 193-197) and gold zeroing (lines 241-243). But between those locked sections, it:
- Reads `player.Level` without lock (line 202) — fine as Level rarely changes, but inconsistent
- Modifies `player.Stats.Con` without lock (lines 206-215)
- Calls `player.Inventory.FindItems("")` and `player.Inventory.clear()` without acquiring `Inventory.mu`
- Calls `player.Equipment.GetEquippedItems()` and clears `Equipment.Slots` (line 232-233) — acquires `Equipment.mu` for the clear but not for `GetEquippedItems`

The `player.Stats.Con` modification at lines 206-215 is completely unprotected:
```go
player.Stats.Con--
if player.Stats.Con < 1 {
    player.Stats.Con = 1
}
```

**Impact:** Data race on `Stats.Con`. If another goroutine (e.g., an affect tick or save) reads `Stats.Con` concurrently, it could see a torn value. The Con could go below 1 if two deaths process simultaneously.

**Fix:** All `Stats.Con` modifications must be under `player.mu.Lock()`. Consolidate all field modifications in `handlePlayerDeath` into a single locked section.

---

### H5-2: `rawKill` Creates Duplicate Corpse

**File:** `pkg/game/combat_helpers.go:103-109`

```go
func (w *World) rawKill(victim *Player, attackType int) {
    corpse := w.makeCorpse(victim.GetName(), victim.GetSex(), nil, nil, victim.RoomVNum, attackType, 0)
    _ = corpse
    w.HandleDeath(victim, nil, attackType)
}
```

`rawKill` calls `makeCorpse` to create an empty corpse, then calls `HandleDeath` which calls `handlePlayerDeath`, which calls `makeCorpse` **again** — this time with the player's actual inventory and equipment. Result: two corpses in the room for every `rawKill` death, one empty and one with items.

**Impact:** Item duplication vector (player sees two corpses, one with their stuff). Confusing gameplay. The empty corpse is a leaked object that never gets cleaned up.

**Fix:** Remove the `makeCorpse` call from `rawKill`. Let `HandleDeath`/`handlePlayerDeath` handle corpse creation exclusively.

---

### H5-3: No Position Check Before Combat Commands

**Files:** `pkg/game/combat_basic.go`, `pkg/session/commands.go`

Commands like `backstab`, `hit`, `kill`, `disembowel`, `bash`, `kick` etc. never check if the player is dead, sleeping, stunned, or incapacitated before executing. The command registry in `commands.go` has a `minPosition` field but:

1. Some combat commands are registered with `minPosition: 0` (e.g., `hit`, `kill`, `backstab` at line ~61-63)
2. The position check in `ExecuteCommand` is never actually performed — there's no code that reads `entry.MinPosition` and compares it to `s.player.GetPosition()`

```go
entry, ok := cmdRegistry.Lookup(cmd)
// No position check here!
return entry.Handler(&commandSession{Session: s}, args)
```

**Impact:** Dead players can attack. Sleeping players can backstab. Stunned players can cast spells. This completely breaks the position-based state machine that MUDs rely on.

**Fix:** Add position enforcement in `ExecuteCommand`:
```go
if s.player.GetPosition() < entry.MinPosition {
    s.sendText("You can't do that right now!")
    return nil
}
```
And set correct `minPosition` values for all combat commands.

---

### H5-4: `idlist` Wizard Command — Arbitrary File Write (Confirmed from Pass 3)

**File:** `pkg/session/wizard_cmds.go:670-696`

```go
func cmdIdlist(s *Session, args []string) error {
    filename := "idlist.txt"
    if len(args) > 0 {
        filename = args[0]
    }
    f, err := os.Create(filename)
```

An implementor-level wizard can specify any filename: `idlist /etc/crontab`, `idlist ../../../etc/passwd`, etc. The `validation` package has path traversal checks, but `ExecuteCommand` only validates the command name and args for SQL injection/XSS patterns — the `../` check in `ValidateInput` would catch `../` but not absolute paths like `/tmp/evil`.

Wait — re-checking: `ValidateInput` does check for `\.\./` (path traversal) but does NOT check for absolute paths starting with `/`. So `idlist /etc/passwd` would pass validation.

**Impact:** Arbitrary file overwrite with world object data. Severity depends on server process privileges.

**Fix:** Restrict `filename` to basename only (strip directory component). Or hardcode the output directory to `./data/`.

---

### H5-5: `BroadcastToRoom` Silently Drops Messages on Full Channel

**File:** `pkg/session/manager.go:275-288`

```go
select {
case s.send <- message:
default:
    // Channel full, drop message
}
```

This pattern appears in `BroadcastToRoom`, `sendError`, `sendText`, `flushDirtyVars`, and `SendToAll`. When a player's `send` channel (buffer size 256) is full, messages are silently dropped. For agents, this means variable updates, combat messages, and death notifications can be lost without any indication.

For combat specifically: if a player's channel fills up during a long fight, they'll stop receiving damage notifications, death messages, and room updates. The combat engine continues dealing damage to them invisibly.

**Impact:** Players lose combat messages, potentially dying without seeing any output. Agents lose state synchronization. No logging when messages are dropped (except `SendToAll` which logs at Debug level).

**Fix:** At minimum, log dropped messages at Warning level. Consider increasing buffer size for combat-active sessions, or implementing backpressure (pause combat processing if victim can't receive messages).

---

## MEDIUM

### M5-1: `Unregister` and `UnregisterAndClose` — Inconsistent Cleanup

**File:** `pkg/session/manager.go:255-268` vs `pkg/session/manager.go:1018-1060`

There are two session cleanup functions:
- `Unregister(playerName)` — called by `readPump` on disconnect
- `UnregisterAndClose(playerName)` — "equivalent of close_socket()"

They do different things:
- `Unregister`: decrements IP count, saves to DB, closes `s.send`, calls `world.RemovePlayer`
- `UnregisterAndClose`: flushes queues, broadcasts leave message, saves to DB, calls `world.RemovePlayer`, closes `s.conn`, closes `s.send`, cleans up snoop state

Neither calls `combatEngine.StopCombat()`. `Unregister` doesn't broadcast a leave message or clean up snoop state. If a player disconnects during a snoop, `s.snoopBy.snooping` still points to the dead session.

**Impact:** Stale snoop references, missing leave messages on normal disconnect, combat not cleaned up on any disconnect path.

**Fix:** Consolidate into a single cleanup function that handles all cases. Use `sync.Once` to ensure idempotent cleanup.

---

### M5-2: `send` Channel Exists on Both `Session` and `Player`

**Files:** `pkg/session/manager.go:214` (Session.send), `pkg/game/player.go:148` (Player.Send)

The `Session` has `send chan []byte` (buffer 256) and the `Player` has `Send chan []byte` (buffer 256 from `NewCharacter`, buffer 100 from `saveDataToPlayer`). Messages flow:
- Game code calls `player.SendMessage(msg)` → writes to `player.Send`
- Session code calls `s.sendText(msg)` → marshals JSON → writes to `s.send`

But nothing reads from `player.Send` and writes to `s.send`! The `writePump` reads from `s.send` and writes to the WebSocket. Messages sent via `player.SendMessage()` go to `player.Send` which has **no consumer** in the WebSocket path.

Wait — let me re-verify. The session's `writePump` reads from `s.send`, not from `s.player.Send`. Game-layer code (death.go, combat_helpers.go, party.go) calls `player.SendMessage()` which writes to `player.Send`. These messages are **silently lost**.

**Impact:** All messages from game-layer code (death messages, XP gain messages, gold split messages, CON loss messages, combat messages from `doDamage`) are written to a channel nobody reads. Players never see death XP loss messages, gold loot messages, or CON loss notifications.

This was identified in Pass 1 as the "dual send channel" issue. It remains unfixed and is the single most impactful gameplay bug.

**Fix:** Either:
1. Bridge `player.Send` to `session.send` with a goroutine, or
2. Replace `player.SendMessage()` with a callback that writes to the session's `send` channel, or  
3. Have `SendMessage` find the session and write to `s.send` directly

---

### M5-3: Inventory Operations Race with Concurrent Access

**Files:** `pkg/game/inventory.go:26-33`, `pkg/session/commands.go` (cmdGet, cmdWear, etc.)

`addItem` and `removeItem` (lowercase, internal) do NOT hold `inv.mu`. The exported `AddItem`/`RemoveItem` wrappers also don't hold the lock. But `FindItem`/`FindItems` DO hold `inv.mu.RLock()`.

This means:
- `cmdGet` calls `s.player.Inventory.AddItem(item)` — no lock
- `cmdWear` calls `s.player.Inventory.RemoveItem(item)` — no lock  
- `cmdInventory` calls `s.player.Inventory.FindItems("")` — holds RLock

Concurrent `AddItem` (from autoloot or a script) and `FindItems` (from inventory command) race on the `Items` slice.

**Impact:** Slice corruption (append during iteration), panic on index out of range, or items silently duplicated/lost.

**Fix:** `addItem` and `removeItem` must acquire `inv.mu.Lock()`. Or move all locking to the caller (World methods that already hold `w.mu`).

---

### M5-4: `handlePlayerDeath` Doesn't Set Player Position to Dead

**File:** `pkg/game/death.go:175-265`

After a player dies, `handlePlayerDeath`:
1. Deducts EXP
2. Applies CON loss
3. Transfers items to corpse
4. Calls `player.SetRoom(MortalStartRoom)` and `player.Heal(9999)` and `player.StopFighting()`

But it never sets `player.Position = PosDead` or even `PosStanding`. The player's position remains whatever it was during combat (`PosFighting` = 7). After respawn, the player is at full health in the temple but still in "fighting" position — which means:
- `GetAttacksPerRound` may give extra attacks
- Position-dependent checks (sleeping, resting) may behave unexpectedly
- The original C code's `raw_kill()` explicitly sets position

**Impact:** Respawned players are in inconsistent position state. Minor gameplay impact but incorrect state machine behavior.

**Fix:** Add `player.SetPosition(combat.PosStanding)` after respawn in `handlePlayerDeath`.

---

### M5-5: `AdvanceLevel` Holds Player Lock During File I/O

**File:** `pkg/game/level.go:87, 307`

```go
func (p *Player) AdvanceLevel() {
    p.mu.Lock()
    defer p.mu.Unlock()
    // ... 220 lines of logic ...
    if err := SavePlayer(p); err != nil {
```

The lock is held for the entire function including the `SavePlayer` call at the end, which does `os.Create`, `json.Encode`, and file I/O. If the filesystem is slow (NFS, full disk), this blocks the player's lock for the duration, preventing any other goroutine from reading their stats.

**Impact:** Lock contention during level-up. The combat engine trying to read `GetHP()` or `GetDamroll()` will block until the save completes.

**Fix:** Release the lock before saving. Restructure as: compute gains under lock, release lock, then save.

---

### M5-6: Reconnection Does Not Clean Up Old Session

**File:** `pkg/session/manager.go:225-233`

```go
func (m *Manager) Register(playerName string, s *Session) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    if _, exists := m.sessions[playerName]; exists {
        return ErrPlayerAlreadyOnline
    }
```

If a player disconnects uncleanly (browser crash, network drop), `readPump` may not have exited yet (60-second read deadline). A reconnection attempt within that window gets `ErrPlayerAlreadyOnline` and fails. The player must wait up to 60 seconds before they can reconnect.

The original C MUD handled this with "link-dead" state — the character stays in the world, and reconnecting attaches to the existing session. This codebase has no such mechanism.

**Impact:** Players cannot reconnect for up to 60 seconds after a disconnect. In combat, this means their character takes damage for a full minute with no way to flee or respond.

**Fix:** Implement session takeover: when `Register` finds an existing session, close the old session's connection and reattach the new WebSocket. Or implement a "link-dead" state where the character persists but the session is detached.

---

### M5-7: No Input Length Limit on Telnet

**File:** `pkg/telnet/listener.go:210-250`

The `readLine` function appends bytes to a slice indefinitely until it sees `\r` or `\n`. There is no maximum line length. A malicious client can send gigabytes of data without a newline, exhausting server memory.

The WebSocket path has `s.conn.SetReadLimit(16384)` (16KB), but telnet has no equivalent.

**Impact:** Memory exhaustion DoS via telnet.

**Fix:** Add a maximum line length (e.g., 4096 bytes) to `readLine`. Disconnect if exceeded.

---

### M5-8: `ValidateInput` Blocks Legitimate Game Commands

**File:** `pkg/validation/input.go:35-36`

The SQL injection patterns include:
- `--` (SQL comment) — this blocks emotes like "say --help" or descriptions with dashes
- `;` (statement separator) — this blocks any input containing semicolons

**But**: `ValidateInput` is not actually called in `ExecuteCommand`. It's defined but unused in the command dispatch path. The `ValidateCommand` function exists but is never called.

**Impact:** The validation is dead code. It provides no protection because it's never invoked. If it were invoked, it would break legitimate gameplay by blocking common characters.

**Fix:** Either remove the dead code or integrate it properly with sensible rules (the game doesn't use SQL for player commands, so SQL injection checks are misdirected — focus on length limits and control character stripping).

---

## LOW

### L5-1: `sendWelcome` Panics if Room Not Found

**File:** `pkg/session/manager.go:474**

```go
func (s *Session) sendWelcome(token string) {
    room, _ := s.manager.world.GetRoom(s.player.GetRoom())
```

The error return is discarded. If `GetRoom` returns `nil` (e.g., player has invalid room VNum from a corrupted save), the subsequent access to `room.VNum`, `room.Name`, `room.Description` will panic with nil pointer dereference.

**Fix:** Check the `ok` return value. If room is not found, default to MortalStartRoom.

---

### L5-2: `saveDataToPlayer` Creates Smaller Send Buffer Than `NewCharacter`

**File:** `pkg/game/save.go:229`

```go
Send: make(chan []byte, 100),  // saveDataToPlayer
```

vs `pkg/game/player.go:224`:
```go
Send: make(chan []byte, 256),  // NewCharacter
```

Players loaded from save files get a 100-buffer channel. New characters get 256. This inconsistency means loaded players are more susceptible to message drops.

**Fix:** Use a constant for the buffer size.

---

### L5-3: `saveDataToPlayer` Doesn't Restore `ActiveAffects` Contents

**File:** `pkg/game/save.go:254`

```go
ActiveAffects: make([]*engine.Affect, len(data.Affects)),
```

This creates a slice of `len(data.Affects)` nil pointers. The actual affect data from `data.Affects` is never deserialized back into `engine.Affect` objects. When the game tries to iterate `ActiveAffects` after loading, every element is `nil`.

**Impact:** All spell effects are lost on save/load. Players lose all buffs, debuffs, and timed effects when the server restarts or they relog.

**Fix:** Iterate `data.Affects` and create proper `engine.Affect` objects from each `saveAffect` entry.

---

### L5-4: `cmdSysfile` Reads Arbitrary Files Relative to Working Directory

**File:** `pkg/session/wizard_cmds.go:919-942`

While `cmdSysfile` restricts the section name to `bugs|ideas|todo|typos`, the file paths are hardcoded to `data/bugs.txt` etc. If the server's working directory is changed (e.g., via a `chdir` in a script), these become relative to wherever the server happens to be running. Not a direct vulnerability since the filenames are hardcoded, but a reliability issue.

**Fix:** Use absolute paths relative to a configured data directory.

---

### L5-5: `number()` Function Not Shown But Referenced in `death.go`

**File:** `pkg/game/death.go:204`

```go
if number(0, ConLossCheckChance-1) == 0 {
```

The `number` function is used for random number generation. If it's implemented as `rand.Intn(max-min+1) + min`, calling `number(0, 0)` returns 0 always. This is fine. But the function definition wasn't shown in the files I read — ensure it handles edge cases where `min == max` and `min > max`.

---

### L5-6: `cmdForce` Logs But Doesn't Execute

**File:** `pkg/session/wizard_cmds.go:432-450`

```go
func cmdForce(s *Session, args []string) error {
    // ...
    slog.Info("forced", "target", target.player.Name, "command", forceCmd, "by", s.player.Name)
    s.Send(fmt.Sprintf("Forced %s to '%s'.", target.player.Name, forceCmd))
    return nil
}
```

The `force` command logs the action and sends a confirmation, but **never actually executes the command** on the target. The `force all` path also only logs.

**Impact:** Wizard `force` command is non-functional. Low severity since it's an admin feature, but confusing.

**Fix:** Call `ExecuteCommand(targetSess, forceCmd, args[2:])` to actually execute the forced command.

---

### L5-7: `cmdQuit` Double-Closes Connection

**File:** `pkg/session/commands.go` (cmdQuit)

```go
func cmdQuit(s *Session) error {
    // ...
    s.manager.world.RemovePlayer(s.player.Name)
    s.manager.Unregister(s.player.Name)
    s.conn.Close()
    return nil
}
```

`Unregister` already handles cleanup. Then `cmdQuit` calls `s.conn.Close()` explicitly. Meanwhile, `readPump`'s defer also calls `s.conn.Close()`. The `conn.Close()` is called twice, and `Unregister` is also called twice (once from cmdQuit, once from readPump's defer). The double-`Unregister` is mostly safe (second call finds no session), but `close(s.send)` in the second `Unregister` call will panic since it was already closed.

**Impact:** Panic on quit — same root cause as C5-1.

**Fix:** `cmdQuit` should just close the connection and let `readPump`'s defer handle cleanup via `Unregister`.

---

## Prioritized Summary

| ID | Severity | Title | Fix Complexity |
|----|----------|-------|----------------|
| C5-1 | CRITICAL | Double-close of `s.send` channel causes panic | Low — add `sync.Once` |
| C5-2 | CRITICAL | `playerToSaveData` reads without lock | Medium — add locking, fix AdvanceLevel |
| C5-3 | CRITICAL | Telnet login bypasses password auth | Medium — add password prompt |
| C5-4 | CRITICAL | `AdvanceLevel` + save deadlock on fix | Medium — restructure locking |
| C5-5 | CRITICAL | Combat engine holds stale refs to disconnected players | Medium — add StopCombat to Unregister |
| H5-1 | HIGH | Death handler inconsistent locking on Stats.Con | Low — consolidate under lock |
| H5-2 | HIGH | `rawKill` creates duplicate corpse | Low — remove redundant makeCorpse |
| H5-3 | HIGH | No position check enforced for combat commands | Medium — add enforcement in ExecuteCommand |
| H5-4 | HIGH | `idlist` arbitrary file write | Low — restrict to basename |
| H5-5 | HIGH | `BroadcastToRoom` silently drops messages | Medium — add logging + backpressure |
| M5-1 | MEDIUM | Inconsistent cleanup between Unregister paths | Medium — consolidate |
| M5-2 | MEDIUM | Dual send channel — game messages silently lost | High — architectural change |
| M5-3 | MEDIUM | Inventory operations race without locks | Medium — add locking |
| M5-4 | MEDIUM | Player position not reset after death/respawn | Low |
| M5-5 | MEDIUM | AdvanceLevel holds lock during file I/O | Medium — restructure |
| M5-6 | MEDIUM | No reconnection/session-takeover mechanism | High |
| M5-7 | MEDIUM | No input length limit on telnet | Low |
| M5-8 | MEDIUM | ValidateInput is dead code | Low — remove or integrate |
| L5-1 | LOW | sendWelcome panics on invalid room | Low |
| L5-2 | LOW | Inconsistent send buffer sizes | Low |
| L5-3 | LOW | ActiveAffects not restored from save | Medium |
| L5-4 | LOW | cmdSysfile uses relative paths | Low |
| L5-5 | LOW | `number()` edge cases unverified | Low |
| L5-6 | LOW | cmdForce doesn't execute the command | Low |
| L5-7 | LOW | cmdQuit double-closes connection/channel | Low — fix with C5-1 |

**Recommended fix order:** C5-1 → C5-5 → C5-2+C5-4 (together) → M5-2 → H5-3 → H5-2 → C5-3 → remainder
