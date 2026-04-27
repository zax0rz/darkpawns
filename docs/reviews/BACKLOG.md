# Dark Pawns — Consolidated Review Backlog

**Generated:** 2026-04-26  
**Source:** Opus code review passes 1–5  
**Codebase:** Dark Pawns MUD server (~80K lines Go)

---

## Summary Statistics

### Raw Findings by Pass

| Pass | CRITICAL | HIGH | MEDIUM | LOW | Total |
|------|----------|------|--------|-----|-------|
| 1 — Architecture | 3 | 6 | 7 | 0 | 16 |
| 2 — Concurrency | 4 | 6 | 6 | 4 | 20 |
| 3 — Security | 2 | 6 | 8 | 6 | 22 |
| 4 — Fidelity | 3 | 6 | 7 | 4 | 20 |
| 5 — QA/Edge Cases | 5 | 5 | 8 | 7 | 25 |
| **Raw Total** | **17** | **29** | **36** | **21** | **103** |

### After Deduplication

| Severity | Count |
|----------|-------|
| CRITICAL | 14 |
| HIGH | 26 |
| MEDIUM | 30 |
| LOW | 16 |
| **Total** | **86** |

### Deduplicated Findings (cross-pass)

| Finding | Passes | Consolidated ID |
|---------|--------|-----------------|
| Dual Send Channel / Player.Send dead | P1-CRITICAL-1, P2-C1, P5-M5-2 | C-01 |
| save.go reads Player without lock | P2-C2, P5-C5-2 | C-04 |
| Telnet login bypasses password auth | P3-C1, P5-C5-3 | C-07 |
| `idlist` arbitrary file write | P3-C2, P5-H5-4 | C-08 |
| Double-close of `s.send` channel | P2-H4, P5-C5-1 | C-12 |

---

## CRITICAL Findings

### ~~C-01✅: Dual Send Channel — Messages Silently Lost~~
**Source:** Pass 1 (CRITICAL-1), Pass 2 (C1), Pass 5 (M5-2)  
**Files:** `pkg/game/player.go:148,795-800`, `pkg/session/manager.go:210,391-420,764-767`, `pkg/game/world.go:438`, `pkg/game/mob.go:175,181`  
**Description:** `Player.Send` (buffered 256) and `Session.send` (buffered 256) are two separate channels. `writePump` reads only from `Session.send`. Nothing ever reads `Player.Send`. All game-layer messages (combat output, room broadcasts, mob actions, death notifications, XP/gold messages) are written to `Player.Send` and silently lost. Blocking sends in `SpawnMob` and `AttackPlayer` can permanently hang goroutines when the buffer fills.  
**Fix:** Remove `Player.Send` entirely. Route all messages through `Session.send` via a `MessageSink` interface or manager method. This also fixes H-11 (SpawnMob deadlock).

### ~~C-02🔶: 70+ Package-Level Mutable `var` Function Hooks in `combat`~~ (deferred — structural refactor)
**Source:** Pass 1 (CRITICAL-2)  
**Files:** `pkg/combat/fight_core.go:15-82`  
**Description:** The `combat` package declares ~70 package-level `var` function pointers (`BroadcastMessage`, `SkillMessageFunc`, `GainExp`, etc.) set at runtime by session/game packages. Race condition: written by one goroutine, read by combat ticker goroutine with no synchronization. Any function called before being set panics (no nil checks). Untestable — can't run two engines with different configs.  
**Fix:** Replace with a `GameCallbacks` struct injected into `CombatEngine` at construction time. Validate all callbacks at construction to prevent nil panics.

### ~~C-03🔶: `game` Package is a 38K-Line God Package~~ (deferred — structural refactor)
**Source:** Pass 1 (CRITICAL-3)  
**Files:** `pkg/game/` (entire directory)  
**Description:** Contains World, Player, MobInstance, Inventory, Equipment, Skills, Spells integration, Movement, Communication, Combat hooks, AI, Zone resets, Spawner, Shops, Houses, Clans, Boards, Bans, Death handling, Dreams, Socials, Spec procs. Every new mechanic increases blast radius.  
**Fix:** Split incrementally: `game/world`, `game/player`, `game/mob`, `game/social`, `game/clan`, `game/housing`, `game/board`, `game/ban`. Each sub-package interacts through interfaces in `common`.

### ~~C-04✅: `save.go` Reads Player Fields Without Any Locking~~
**Source:** Pass 2 (C2), Pass 5 (C5-2)  
**Files:** `pkg/game/save.go:130-200`  
**Description:** `playerToSaveData(p *Player)` reads every Player field — Health, Mana, Gold, Exp, SpellMap (Go map), Inventory.Items (slice), ActiveAffects — without acquiring `p.mu`. Called concurrently with combat, game loop, and commands. Concurrent read of Go map during write = fatal panic. Produces corrupted save files with torn reads.  
**Fix:** Acquire `p.mu.RLock()` in `playerToSaveData`. Must coordinate with C-13 (AdvanceLevel holds lock during save) to avoid deadlock.

### ~~C-05✅: Session Agent Fields Mutated Cross-Goroutine — Map Panic~~
**Source:** Pass 2 (C3)  
**Files:** `pkg/session/manager.go:165,773`, `pkg/session/agent_vars.go:70,77-83,91-98,168-169`  
**Description:** `s.dirtyVars`, `s.subscribedVars` are Go maps accessed from both the readPump goroutine and combat ticker goroutine (via `DamageFunc` callback calling `markDirty`). Concurrent map read/write = fatal runtime panic during any combat round involving an agent session.  
**Fix:** Add `sync.Mutex` to Session for agent state fields, or use a channel to serialize mutations.

### ~~C-06✅: Global Mutable Maps `playerSneakState`/`playerHideState` Without Synchronization~~
**Source:** Pass 2 (C4)  
**Files:** `pkg/game/skills.go:627-650`  
**Description:** Package-level Go maps accessed from command handlers (per-player goroutines), AI ticker goroutine, and zone dispatcher goroutines. Concurrent map access = fatal panic. These maps are redundant — `Player.Affects` already has `AFF_SNEAK`/`AFF_HIDE` bits.  
**Fix:** Remove the global maps. Use existing affect flags on `Player` (which already has `p.mu`).

### ~~C-07✅: Telnet Login Bypasses Password Authentication~~
**Source:** Pass 3 (C1), Pass 5 (C5-3)  
**Files:** `pkg/telnet/listener.go:129-138`, `pkg/session/manager.go:489-499`  
**Description:** Telnet `sendLogin()` sends only `player_name` — no password. In no-DB mode, all logins succeed with zero authentication. In DB mode, telnet is functionally broken (always rejected for existing characters). Additionally serves as a DoS vector via name-squatting rate limiter consumption.  
**Fix:** Add password prompt to telnet `handleConn()` before calling `sendLogin()`. For no-DB mode, either disable telnet or add a server-level password.

### ~~C-08✅: Wizard `idlist` Command — Arbitrary File Write~~
**Source:** Pass 3 (C2), Pass 5 (H5-4)  
**Files:** `pkg/session/wizard_cmds.go:1240-1270` (or :670-696 per Pass 5)  
**Description:** `cmdIdlist` passes user-supplied filename directly to `os.Create(filename)`. A wizard (level 61) can write to any server-writable path: `/etc/cron.d/backdoor`, `../../../root/.ssh/authorized_keys`, etc. `ValidateInput` catches `../` but not absolute paths like `/tmp/evil`.  
**Fix:** Restrict to `filepath.Base(args[0])` and force output to a safe directory like `data/`.

### ~~C-09✅: Saving Throw System Completely Rewritten (d100 → d20, Wrong Tables)~~
**Source:** Pass 4 (C1)  
**Files:** `pkg/spells/saving_throws.go` vs `src/magic.c:83-406`  
**Description:** C uses d100 roll with tables valued 0-90 across 41 levels. Go uses d20 roll with tables valued 5-17 across 21 levels. Every spell that checks a saving throw produces wildly different save rates. Example: C Mage level 1 vs SPELL = ~60% save rate; Go = ~75%. Classes 5-11 are copy-pasted from base classes instead of using actual C tables. Breaks the entire spell balance.  
**Fix:** Port the actual `saving_throws[NUM_CLASSES][5][41]` table from `src/magic.c` verbatim. Change roll to `rand.Intn(100)` and comparison to match C's `MAX(1, save) < roll`.

### ~~C-10✅: Combat Commands Are Stubs — No Damage Calculation~~
**Source:** Pass 4 (C2)  
**Files:** `pkg/session/act_offensive.go` vs `src/act.offensive.c`  
**Description:** All combat commands (backstab, kick, bash, dragon_kick, tiger_punch, disembowel, neckbreak) are display-only stubs that print flavor text and call `StartCombat()` but never calculate damage, check skill percentages, apply WAIT_STATE, or call `improve_skill()`. The entire skill-based combat system is non-functional.  
**Fix:** Port each command's damage formula, skill check, WAIT_STATE, success/failure logic, and `improve_skill()` call from C source.

### ~~C-11✅: Parry/Dodge System Not Implemented~~
**Source:** Pass 4 (C3)  
**Files:** `pkg/session/fight.go` vs `src/fight.c:1958-1975`  
**Description:** `cmdParry()` is a 3-line stub. Parry skill is registered but never used in combat resolution. NPC dodge (AFF_DODGE) is also not checked. A core defensive mechanic is completely missing, making combat significantly more lethal than intended.  
**Fix:** Implement parry check in combat round resolution. Track IS_PARRIED state. Apply attack reduction. Implement NPC dodge check.

### ~~C-12✅: Double-Close of `s.send` Channel Causes Panic~~
**Source:** Pass 2 (H4), Pass 5 (C5-1)  
**Files:** `pkg/session/manager.go:267,843,1054,1209`  
**Description:** `s.send` is closed in four+ locations: `Unregister()`, `CloseSend()`, `UnregisterAndClose()`, `CheckIdlePasswords()`. No `sync.Once` or closed-flag guards. `readPump` defer calls `Unregister`; if another cleanup path fires for the same session, the channel is closed twice → runtime panic. `cmdQuit` (Pass5-L5-7) also triggers this by calling `Unregister` + `conn.Close`, then readPump's defer calls `Unregister` again.  
**Fix:** Add `sendClosed sync.Once` to Session. All close paths use `s.sendClosed.Do(func() { close(s.send) })`.

### ~~C-13✅: `AdvanceLevel` + Save Creates Deadlock When C-04 Is Fixed~~
**Source:** Pass 5 (C5-4)  
**Files:** `pkg/game/level.go:87,307`  
**Description:** `AdvanceLevel` holds `p.mu.Lock()` for ~220 lines including `SavePlayer(p)` at the end. If C-04 is fixed by adding `p.mu.RLock()` to `playerToSaveData`, this will deadlock (RLock under Lock on non-reentrant mutex). Also holds lock during file I/O (Pass5-M5-5), blocking combat reads.  
**Fix:** Restructure `AdvanceLevel` to release lock before saving. Compute all gains under lock, release, then save.

### ~~C-14✅: Combat Engine Holds Stale References to Disconnected Players~~
**Source:** Pass 5 (C5-5)  
**Files:** `pkg/combat/engine.go:165-230`, `pkg/session/manager.go:255-268`  
**Description:** When a player disconnects, `Unregister` removes them from the session map and closes `s.send`, but never calls `combatEngine.StopCombat(playerName)`. The combat engine still holds `Combatant` pointers and will call `SendMessage()` on the dead `Player.Send` channel — writing to a closed channel panics.  
**Fix:** `Unregister` and `UnregisterAndClose` must call `combatEngine.StopCombat(playerName)` before closing `s.send`.

---

## HIGH Findings

### ~~H-01🔶: Package-Level `var` for Cross-Package Wiring Throughout~~ (deferred — structural refactor)
**Source:** Pass 1 (HIGH-1)  
**Files:** `pkg/game/scripts.go:14`, `pkg/game/ai.go:28`, `pkg/game/merge_bridge.go:28`, `cmd/server/main.go:64,78`  
**Description:** Mutable package-level variables (`game.ScriptEngine`, `game.HasActiveCharacter`, `aiCombatEngine`) used for DI. Written at startup, read at runtime — race conditions, test pollution, fragile boot order. Related to C-02 but broader scope.  
**Fix:** Pass dependencies through constructors. `World` accepts `ScriptEngine` in `NewWorld()`.

### ~~H-02🔶: Session/Command Split is Confused~~ (deferred — structural refactor)
**Source:** Pass 1 (HIGH-2)  
**Files:** `pkg/session/commands.go` (1,664 lines), `pkg/session/wizard_cmds.go` (1,792 lines), `pkg/command/registry.go`, `pkg/command/skill_commands.go` (1,591 lines)  
**Description:** Command handlers split between `session/` and `command/` with no clear principle. `command.SessionInterface` imports `*game.Player` directly. Multiple competing session interfaces. The good `Registry` pattern in `command/` is underused.  
**Fix:** Move all command handlers to `pkg/command/`. Session handles only WebSocket lifecycle + dispatch. Use interfaces from `common` instead of importing `game`.

### ~~H-03🔶: `World` Struct Does Too Much (20+ Concerns, Single RWMutex)~~ (deferred — structural refactor)
**Source:** Pass 1 (HIGH-3)  
**Files:** `pkg/game/world.go:21-73`  
**Description:** World owns rooms, mobs, objects, zones, players, AI ticker, spawner, shopManager, EventQueue, event bus, zone dispatcher, houses, clans, boards, bans, snapshot manager — all behind a single `sync.RWMutex`. Any write blocks all reads. `CharTransfer()` holds write lock while iterating all players and mobs.  
**Fix:** Factor out subsystems with own locks: `PlayerRegistry`, `MobManager`, `ItemManager`. World becomes thin coordinator.

### ~~H-04🔶: `interface{}` Used to Break Import Cycles~~ (deferred — structural refactor)
**Source:** Pass 1 (HIGH-4)  
**Files:** `pkg/game/world.go:247-270,451,462,668,699,774`  
**Description:** Methods accept/return `interface{}` to avoid cycles: `ForEachPlayerInRoomInterface`, `LookAtRoomSimple(sender interface{})`, `AddFollowerQuiet(ch, leader interface{})`, `GetAllCharsInRoom` returns `[]interface{}`. Erases type safety, requires runtime assertions that can panic.  
**Fix:** Define narrow interfaces in `common`: `MessageSender`, `Character`, etc.

### ~~H-05🔶: Mutex Discipline Inconsistencies~~ (deferred — structural refactor)
**Source:** Pass 1 (HIGH-5)  
**Files:** `pkg/game/world.go` (various)  
**Description:** `SendToZone` iterates `w.players` without lock. `OnPlayerEnterRoom` calls `GetMobsInRoom` (RLock) while potentially holding write lock → deadlock. `removeItemFromRoomLocked` pattern is fragile with no compile-time enforcement. Overlaps with Pass 2 findings (H-07, H-08, H-09).  
**Fix:** Adopt consistent locking strategy. Add `// requires: w.mu held` comments. Use `go test -race` extensively.

### ~~H-06🔶: `common` Package Interfaces Are Too Wide~~ (deferred — structural refactor)
**Source:** Pass 1 (HIGH-6)  
**Files:** `pkg/common/common.go:7-73`  
**Description:** `Affectable` interface has 27 methods. `CommandManager` leaks `Lock()/Unlock()/Mu()`. `ShopManager` returns `interface{}` everywhere.  
**Fix:** Split `Affectable` into `StatBlock`, `CombatStats`, `StatusEffectable`. Remove mutex methods from interfaces.

### ~~H-07✅: `SendToZone`/`SendToAll` Iterate `w.players` Without Lock~~
**Source:** Pass 2 (H1)  
**Files:** `pkg/game/world.go:333-347`  
**Description:** `w.players` is a Go map mutated by `AddPlayer`/`RemovePlayer` under `w.mu.Lock()`. `SendToZone` and `SendToAll` iterate it without holding `w.mu.RLock()`. Races with any player login/logout. Also reads `p.RoomVNum` directly instead of via `p.GetRoom()`.  
**Fix:** Hold `w.mu.RLock()` for iteration. Use `p.GetRoom()` for field access.

### ~~H-08✅: `mobact.go` Accesses `MobInstance.RoomVNum` Directly, Bypassing Mutex~~
**Source:** Pass 2 (H2)  
**Files:** `pkg/game/mobact.go:116,129,139,148,161,206,212,227,233,241,252,258,274`  
**Description:** `MobileActivity()` accesses `ch.RoomVNum` directly (15 occurrences) instead of `ch.GetRoom()`. Races with `SetRoom()` calls from zone dispatcher, combat movement, player commands.  
**Fix:** Replace all `ch.RoomVNum` with `ch.GetRoom()` in mobact.go.

### ~~H-09✅: `PointUpdate` Reads `p.Flags` Without Lock~~
**Source:** Pass 2 (H3)  
**Files:** `pkg/game/limits.go:455`  
**Description:** Reads `p.Flags` without `p.mu` on PointUpdate timer goroutine. Commands on readPump goroutine may be writing `p.Flags` via `SetPlrFlag()`.  
**Fix:** Use `p.GetFlags()` (which acquires `p.mu.RLock()`).

### ~~H-10✅: Spawner Goroutine Leak — No Shutdown Mechanism~~
**Source:** Pass 2 (H5)  
**Files:** `pkg/game/spawner.go:580-588`  
**Description:** `StartPeriodicResets` spawns a goroutine with no done channel, no context, and no way to stop. Ticker never stopped. On server shutdown, goroutine leaks.  
**Fix:** Add `stopCh` to Spawner and select on it, or use `context.Context`.

### ~~H-11✅: `SpawnMob` Blocking Send While Holding `w.mu.Lock`~~
**Source:** Pass 2 (H6)  
**Files:** `pkg/game/world.go:434-440`  
**Description:** Blocking `player.Send <-` while holding `w.mu.Lock()`. Since Player.Send is never read (C-01), buffer fills and this deadlocks the entire world. Resolves automatically when C-01 is fixed.  
**Fix:** Resolved by C-01 fix. Otherwise: move notification outside lock or use `select`/`default`.

### ~~H-12✅: X-Forwarded-For Header Trusted Without Proxy Validation~~
**Source:** Pass 3 (H1)  
**Files:** `pkg/auth/ratelimit.go:66-77`  
**Description:** `GetIPFromRequest()` blindly trusts `X-Forwarded-For`. Any client can spoof IP, bypassing login rate limiting and per-IP connection limits. Enables brute-force attacks.  
**Fix:** Only trust `X-Forwarded-For` from configured trusted proxy IPs. Take the rightmost untrusted IP.

### ~~H-13✅: WebSocket Allows Connections Without Origin Header in Production~~
**Source:** Pass 3 (H2)  
**Files:** `pkg/session/manager.go:45-51`  
**Description:** When no `Origin` header present, connection is allowed even in production. Removes a defense-in-depth layer.  
**Fix:** In production, reject connections without Origin or require explicit config flag.

### ~~H-14✅: `cmdAt` — Recursive Wizard Command Execution Without Depth Limit~~
**Source:** Pass 3 (H3)  
**Files:** `pkg/session/wizard_cmds.go:70-85`  
**Description:** `at` command allows `at 100 at 200 at 300 shutdown` — no recursion depth limit. Potential stack exhaustion DoS; audit log evasion for intermediate rooms.  
**Fix:** Add recursion counter to Session. Cap at 3 levels.

### ~~H-15✅: No Account-Level Lockout for Failed Login Attempts~~
**Source:** Pass 3 (H6)  
**Files:** `pkg/session/manager.go:461-469`, `pkg/auth/ratelimit.go`  
**Description:** Rate limiter is per-IP only (5 req/s, burst 10). Combined with H-12 (X-Forwarded-For bypass), unlimited password attempts against any account.  
**Fix:** Add per-account rate limiting with escalating lockout durations.

### ~~H-16✅: Affect Durations 75× Too Short (Seconds vs Game-Hours)~~
**Source:** Pass 4 (H4)  
**Files:** `pkg/engine/affect.go:82-85`  
**Description:** Go uses 1 tick = 1 second. C uses 1 tick = 75 seconds (game hour). A spell with `duration = 24` lasts 24 seconds in Go vs 30 minutes in C. Sanctuary lasts 4 seconds instead of 5 minutes. All buffs/debuffs are effectively useless.  
**Fix:** Multiply durations by 75, or set tick interval to 75 seconds, or convert to game-hour durations.

### ~~H-17✅: Position Damage Multiplier Uses Float Division (Should Be Integer)~~
**Source:** Pass 4 (H1)  
**Files:** `pkg/combat/fight_core.go:806` vs `pkg/combat/formulas.go:445`  
**Description:** `MakeHit()` (actual combat path) uses float division: sitting targets take ~33% more damage than C. `formulas.go` has the correct integer version but it's not used in the hot path.  
**Fix:** Use integer division in `MakeHit()`: `dam *= 1 + (PosFighting-defPos)/3`.

### ~~H-18✅: Constitution Loss on Death Is Deterministic (C Is Probabilistic)~~
**Source:** Pass 4 (H2)  
**Files:** `pkg/combat/fight_core.go` (`DieWithKiller`) vs `src/fight.c:601-611`  
**Description:** C: level ≤5 never loses CON; level >5 has 25% chance; level >20 has additional ~17% chance of -2 CON. Go: every death always costs 1 CON regardless of level. Death is vastly more punishing, especially for low-level characters.  
**Fix:** Add level checks and random rolls matching C's formula. Call `affect_total` equivalent.

### ~~H-19✅: `number(1,100)` vs `rand.Intn(100)` Off-By-One in Attack Count~~
**Source:** Pass 4 (H5)  
**Files:** `pkg/combat/formulas.go:527` vs `src/fight.c:1928`  
**Description:** C's `number(1,100)` = 1-100; Go's `rand.Intn(100)` = 0-99. ~1% probability shift per attack-count roll, compounding across multi-attack builds.  
**Fix:** Use `rand.Intn(100) + 1` to match C's range.

### ~~H-20✅: `handlePlayerDeath` Inconsistent Locking on `Stats.Con`~~
**Source:** Pass 5 (H5-1)  
**Files:** `pkg/game/death.go:178-265`  
**Description:** Acquires `player.mu.Lock()` for EXP and gold but modifies `Stats.Con` (lines 206-215) completely unprotected. `Inventory.FindItems` and `Equipment.GetEquippedItems` also called without their locks. Con could go below 1 with simultaneous deaths.  
**Fix:** Consolidate all field modifications into a single locked section.

### ~~H-21✅: `rawKill` Creates Duplicate Corpse~~
**Source:** Pass 5 (H5-2)  
**Files:** `pkg/game/combat_helpers.go:103-109`  
**Description:** `rawKill` calls `makeCorpse` (empty corpse), then `HandleDeath` calls `handlePlayerDeath` which calls `makeCorpse` again (with items). Two corpses per death — one empty, one with inventory. Item duplication vector.  
**Fix:** Remove the `makeCorpse` call from `rawKill`. Let `HandleDeath` handle it exclusively.

### ~~H-22✅: No Position Check Enforced for Combat Commands~~
**Source:** Pass 5 (H5-3)  
**Files:** `pkg/game/combat_basic.go`, `pkg/session/commands.go`  
**Description:** Command registry has `minPosition` field but `ExecuteCommand` never reads or enforces it. Dead players can attack. Sleeping players can backstab. Position-based state machine completely broken.  
**Fix:** Add `if s.player.GetPosition() < entry.MinPosition` check in `ExecuteCommand`. Set correct minPosition for all combat commands.

### ~~H-23✅: `BroadcastToRoom` Silently Drops Messages on Full Channel~~
**Source:** Pass 5 (H5-5)  
**Files:** `pkg/session/manager.go:275-288`  
**Description:** `select/default` pattern drops messages when buffer (256) is full. No logging at warn level. During long fights, players stop receiving combat output, death messages, and room updates while combat continues dealing invisible damage.  
**Fix:** Log dropped messages at Warning level. Consider backpressure or larger buffers for combat-active sessions.

### ~~H-24✅: `force` Command Is a Stub — Dangerous When Implemented~~
**Source:** Pass 3 (H4)  
**Files:** `pkg/session/wizard_cmds.go:450-477`  
**Description:** Logs intent but never calls `ExecuteCommand()`. Currently a security positive, but the code structure suggests it will be filled in without review. Needs transitive force chain prevention, privilege level enforcement, and full arg parsing when implemented.  
**Fix:** Document as intentional stub. When implementing: force runs at target's privilege level, prevent force chains, parse full command string.

### H-25: JWT 24-Hour Lifetime With No Rotation
**Source:** Pass 3 (H5)  
**Files:** `pkg/session/manager.go:383-409`, `pkg/session/protocol.go:61`  
**Description:** JWT sent over WebSocket with 24-hour lifetime and no refresh mechanism. If intercepted (especially over unencrypted WS in dev), provides full API access for a full day.  
**Fix:** Reduce to 1-hour lifetime with refresh mechanism. Enforce WSS in production.

### ~~H-26✅: Counter_procs Fall-Through Not Faithfully Reproduced~~
**Source:** Pass 4 (H6)  
**Files:** `pkg/combat/fight_core.go:984-988` vs `src/fight.c:1283-1296`  
**Description:** C switch has intentional fall-through giving variable stat bonuses. Go always gives +1 to all three stats. Only triggers at kill milestones (1000, 2000, 10000). Minor impact.  
**Fix:** Document as intentional deviation, or implement fall-through logic for strict fidelity.

---

## MEDIUM Findings

### M-01: Error Handling is Inconsistent
**Source:** Pass 1 (MEDIUM-1)  
**Files:** Throughout codebase  
**Description:** No custom error types. All errors are `fmt.Errorf()` strings. `handleLogin` has 140 lines of inconsistent error handling. Cannot programmatically handle different error conditions.  
**Fix:** Define domain-specific error types. Use `errors.Is()`/`errors.As()` consistently. Wrap with `%w`.

### M-02: `engine/comm_infra.go` Contains Dead Code
**Source:** Pass 1 (MEDIUM-2)  
**Files:** `pkg/engine/comm_infra.go`  
**Description:** C-to-Go compat wrappers: `Nonblock()`, `SetupLog()`, `OpenLogfile()` are no-ops. `TxtQ` type never used. `PerformAlias()`/`PerformSubst()` have no callers.  
**Fix:** Remove dead code. Keep `MakePrompt()`.

### M-03: Snapshot Manager Only Covers Rooms
**Source:** Pass 1 (MEDIUM-3)  
**Files:** `pkg/game/snapshot*.go`  
**Description:** Atomic snapshot pattern only applies to rooms. Players, mobs, items still require lock. Inconsistent API with no obvious indicator which reads are lock-free.  
**Fix:** Expand snapshots to cover read-heavy paths, or document clearly which methods use which strategy.

### M-04: `init()` Used for Command Registration
**Source:** Pass 1 (MEDIUM-4)  
**Files:** `pkg/session/commands.go:30-100`  
**Description:** 80+ commands registered in `init()`. Invisible to `main()`, prevents dynamic registration (plugins), can cause test issues.  
**Fix:** Replace with explicit `RegisterBuiltinCommands(r *command.Registry)` called from `main()`.

### M-05: Combatant Interface — Name-Based Lookups, 21 Methods
**Source:** Pass 1 (MEDIUM-5)  
**Files:** `pkg/combat/combatant.go:12-57`  
**Description:** 21-method interface. Combat uses string names (`GetFighting() string`) requiring O(n) name-based lookups through world maps.  
**Fix:** Split interface. Use entity references instead of string names for combat pairs.

### M-06: Two Competing Command Session Interfaces
**Source:** Pass 1 (MEDIUM-6)  
**Files:** `pkg/common/command_interfaces.go:4`, `pkg/command/interface.go:8`, `pkg/common/common.go:38`  
**Description:** `common.CommandSession` (returns `interface{}`), `command.SessionInterface` (returns `*game.Player`), plus `common.Session` — three interfaces for the same concept.  
**Fix:** Consolidate to one interface.

### M-07: `main.go` Manual Wiring With No Lifecycle Management
**Source:** Pass 1 (MEDIUM-7)  
**Files:** `cmd/server/main.go`  
**Description:** Manual wiring via `SetCombatBroadcastFunc()`, `SetDeathFunc()`, etc. No validation — omitted calls leave nil function pointers. No start/stop lifecycle.  
**Fix:** Create `App` struct owning all subsystems. Validate wiring at construction. Managed start/stop.

### M-08: `PerformRound` Uses Write Lock for Read-Only Snapshot
**Source:** Pass 2 (M1)  
**Files:** `pkg/combat/engine.go:193-206`  
**Description:** Uses `Lock()` for a read-only snapshot of `combatPairs`. Should use `RLock()`.  
**Fix:** Change to `ce.mu.RLock()`/`ce.mu.RUnlock()`.

### M-09: `SetTickInterval` Replaces Ticker Without Notifying Goroutine
**Source:** Pass 2 (M3)  
**Files:** `pkg/engine/affect_tick.go:63-73`  
**Description:** `tickLoop` goroutine blocked on old `ticker.C` after `SetTickInterval` replaces the ticker. New ticker's channel never consumed.  
**Fix:** Signal `tickLoop` to restart its select, or use channel to send new interval.

### M-10: Event Bus Unsubscribe Is Broken (Pointer Comparison)
**Source:** Pass 2 (M4)  
**Files:** `pkg/events/bus.go:58-67`  
**Description:** `&h == &handler` compares address of loop variable copy with closure capture. Never equal. Handlers can never be removed. Memory leak.  
**Fix:** Use ID-based subscription or wrapper struct with ID field.

### M-11: `immortalSessionProvider` Global Assigned Without Synchronization
**Source:** Pass 2 (M5)  
**Files:** `pkg/game/logging.go:204,210-212`  
**Description:** Written at startup, read from MudLog on any goroutine. Go memory model doesn't guarantee visibility.  
**Fix:** Use `atomic.Pointer` or `sync.Once`.

### ~~M-12✅: CORS Wildcard Subdomain Matching Overly Permissive~~
**Source:** Pass 3 (M1)  
**Files:** `web/cors.go:53-58`  
**Description:** `HasSuffix` doesn't check dot boundary. `*.darkpawns.example.com` matches `evil-darkpawns.example.com`.  
**Fix:** Check for dot boundary: `strings.HasSuffix(origin, "."+domain)`.

### ~~M-13✅: Development Mode CORS Allows All Origins~~
**Source:** Pass 3 (M2)  
**Files:** `web/cors.go:44-46`  
**Description:** `ENVIRONMENT=development` allows all origins. Risk of accidental deployment with wrong env var.  
**Fix:** Require explicit `ALLOW_ALL_ORIGINS=true`.

### ~~M-14✅: CSP Allows `unsafe-inline` for Scripts~~
**Source:** Pass 3 (M3)  
**Files:** `web/security.go:14-15`  
**Description:** `script-src 'self' 'unsafe-inline'` defeats CSP against XSS. Player names, room descriptions, chat messages could contain XSS payloads.  
**Fix:** Use nonces or hashes instead of `unsafe-inline` for scripts.

### M-15: `cmdUsers` Exposes Player IP Addresses to All Wizards
**Source:** Pass 3 (M4)  
**Files:** `pkg/session/act_informative.go:246-256`  
**Description:** Level 50 immortals can see raw IPs of all online players. Privacy violation.  
**Fix:** Restrict to LVL_GRGOD (61) or redact to network prefix.

### M-16: `cmdSwitch` Doesn't Fully Implement Body Switching
**Source:** Pass 3 (M5)  
**Files:** `pkg/session/wizard_cmds.go:307-357`  
**Description:** Sets `isSwitched = true` but doesn't change `s.player`. All commands still execute as wizard. Audit log confusion.  
**Fix:** Complete implementation or document as cosmetic only.

### M-17: Rate Limiter Cleanup Enables Periodic Bypass
**Source:** Pass 3 (M7)  
**Files:** `pkg/auth/ratelimit.go:30-43`  
**Description:** Wipes entire map at 10,000 entries. All rate limits reset simultaneously. Attacker can trigger reset by creating 10K entries.  
**Fix:** Use LRU cache with TTL per entry.

### M-18: `who` Command Reveals Agent Status to All Players
**Source:** Pass 3 (M8)  
**Files:** `pkg/session/commands.go` (cmdWho)  
**Description:** All players can see which sessions are agent-controlled bots vs human players.  
**Fix:** Remove agent tag from public output, or restrict to wizards.

### M-19: Spell Affect Durations Don't All Match C Values
**Source:** Pass 4 (M1)  
**Files:** `pkg/spells/affect_spells.go`  
**Description:** Curse duration missing `+1` compared to C. Compounds with H-16 (tick timing).  
**Fix:** Add `+1` to curse duration.

### M-20: Bless Spell Applies AC Instead of Saving Throw
**Source:** Pass 4 (M2)  
**Files:** `pkg/spells/affect_spells.go:33-34` vs `src/magic.c:934-942`  
**Description:** Second affect should be `APPLY_SAVING_SPELL -2` but Go applies `AffectArmorClass -2`. Players lose magic defense, gain unintended AC bonus.  
**Fix:** Change to saving throw affect type.

### M-21: Blindness Missing Reagent Bonus and NPC Retaliation
**Source:** Pass 4 (M3)  
**Files:** `pkg/spells/affect_spells.go` vs `src/magic.c:945-972`  
**Description:** Mage reagent system for blindness not implemented. Mobs don't retaliate when they resist.  
**Fix:** Port reagent check and NPC retaliation on save.

### M-22: Movement Missing Sector-Based Move Costs
**Source:** Pass 4 (M4)  
**Files:** `pkg/session/act_movement.go` vs `src/act.movement.c:151-152`  
**Description:** No movement point expenditure. No sector-type based costs. Players move infinitely; terrain has no mechanical effect.  
**Fix:** Implement sector-based movement costs from `constants.c` `movement_loss` table.

### M-23: Mob AI (`mobile_activity`) Not Fully Ported
**Source:** Pass 4 (M5)  
**Files:** Various vs `src/mobact.c`  
**Description:** Missing: double-speed hunting, scavenger behavior, random wandering (1-in-3, respects SENTINEL/STAY_ZONE), aggressive mob attacks, race-hate, memory revenge, helper assist, AGGR24. Mobs are likely static.  
**Fix:** Port `mobile_activity()` loop from C source. High effort.

### M-24: Missing Single-Level XP Cap
**Source:** Pass 4 (M7)  
**Files:** `pkg/combat/fight_core.go` vs `src/limits.c:319-321`  
**Description:** C caps XP gain at `max_exp-1` (can't skip levels). Go only has `max_exp_gain` cap. Players could skip levels with large kills.  
**Fix:** Add `MIN(max_exp-1, gain)` second cap.

### M-25: Inconsistent Cleanup Between `Unregister` Paths
**Source:** Pass 5 (M5-1)  
**Files:** `pkg/session/manager.go:255-268` vs `:1018-1060`  
**Description:** `Unregister` and `UnregisterAndClose` do different things. Neither calls `StopCombat`. `Unregister` doesn't broadcast leave message or clean snoop. Stale snoop references after disconnect.  
**Fix:** Consolidate into single idempotent cleanup function. Use `sync.Once`.

### M-26: Inventory Operations Race Without Locks
**Source:** Pass 5 (M5-3)  
**Files:** `pkg/game/inventory.go:26-33`  
**Description:** `addItem`/`removeItem` (internal) don't hold `inv.mu`. Concurrent `AddItem` (autoloot/script) and `FindItems` (inventory command) race on the `Items` slice. Slice corruption, panic, or item loss.  
**Fix:** `addItem`/`removeItem` must acquire `inv.mu.Lock()`.

### ~~M-27✅: Player Position Not Reset After Death/Respawn~~
**Source:** Pass 5 (M5-4)  
**Files:** `pkg/game/death.go:175-265`  
**Description:** After death, player respawns at full health in temple but position remains `PosFighting`. Incorrect state machine behavior affects position-dependent checks.  
**Fix:** Add `player.SetPosition(PosStanding)` after respawn.

### M-28: No Reconnection/Session-Takeover Mechanism
**Source:** Pass 5 (M5-6)  
**Files:** `pkg/session/manager.go:225-233`  
**Description:** If player disconnects uncleanly, `readPump` holds session for up to 60 seconds (read deadline). Reconnection within that window fails with `ErrPlayerAlreadyOnline`. Character takes combat damage for a minute with no way to respond. Original C MUD had "link-dead" state.  
**Fix:** Implement session takeover (close old session on reconnect) or link-dead state.

### M-29: No Input Length Limit on Telnet
**Source:** Pass 5 (M5-7)  
**Files:** `pkg/telnet/listener.go:210-250`  
**Description:** `readLine` appends bytes indefinitely until `\r`/`\n`. No max length. Memory exhaustion DoS via telnet. WebSocket has 16KB limit.  
**Fix:** Add 4096-byte max line length. Disconnect if exceeded.

### M-30: `ValidateInput` Is Dead Code
**Source:** Pass 5 (M5-8)  
**Files:** `pkg/validation/input.go:35-36`  
**Description:** `ValidateInput` and `ValidateCommand` are defined but never called in command dispatch. Provides zero protection. If it were invoked, it would break legitimate gameplay (blocks `--` and `;`).  
**Fix:** Remove dead code or integrate properly with sensible rules (length limits, control character stripping).

---

## LOW Findings

### L-01: Redundant `GoldMu` on Player
**Source:** Pass 2 (L1)  
**Files:** `pkg/game/player.go:30`  
**Description:** Second lock for Gold field. `Player.mu` already protects all fields. Deadlock hazard if both acquired.  
**Fix:** Remove `GoldMu`. Use `p.mu` consistently.

### L-02: `SpecRegistry` Init-Time Safety Undocumented
**Source:** Pass 2 (L2)  
**Files:** `pkg/game/spec_assign.go:391-395`  
**Fix:** Document that registration must complete before `Start()`, or use `sync.Map`.

### L-03: Zone Worker `ticks` Not Atomic
**Source:** Pass 2 (L4)  
**Files:** `pkg/game/zone_dispatcher.go:35,101`  
**Description:** `w.ticks` written by zone goroutine, read by `ZoneTicks()` from another goroutine.  
**Fix:** Use `atomic.Uint64`.

### L-04: `sysfile` No Size Limit on File Reads
**Source:** Pass 3 (C3 — reclassified LOW-MED)  
**Files:** `pkg/session/wizard_cmds.go:1639-1674`  
**Fix:** Add `io.LimitReader(f, 64*1024)`.

### L-05: Lua `dofile` Re-Registration Pattern Confusing
**Source:** Pass 3 (M6)  
**Files:** `pkg/scripting/engine.go:50,297`  
**Fix:** Add comment explaining intentional sandboxed re-add.

### L-06: `randomString()` in affect.go Uses Broken PRNG
**Source:** Pass 4 (L4)  
**Files:** `pkg/engine/affect.go:163`  
**Description:** All characters generated from same nanosecond timestamp. "Random" string is likely all same character. Affect IDs may collide.  
**Fix:** Use `math/rand` or `crypto/rand` properly.

### L-07: Gender-Unaware Message Token Replacement
**Source:** Pass 4 (L2)  
**Files:** `pkg/combat/fight_core.go` (replaceMessageTokens)  
**Description:** `$e` always "he", `$E` always "him". No gender check.  
**Fix:** Check character sex for pronoun tokens.

### L-08: Missing `flesh_altered_type()` for Unarmed NPC Attacks
**Source:** Pass 4 (L3)  
**Files:** `pkg/combat/fight_core.go`  
**Fix:** Check `AFF_FLESH_ALTER` and call equivalent function.

### L-09: `sendWelcome` Panics if Room Not Found
**Source:** Pass 5 (L5-1)  
**Files:** `pkg/session/manager.go:474`  
**Description:** Error return from `GetRoom` discarded. Nil room → panic on `.VNum`, `.Name`, `.Description`.  
**Fix:** Check `ok` return. Default to `MortalStartRoom`.

### L-10: Inconsistent Send Buffer Sizes (100 vs 256)
**Source:** Pass 5 (L5-2)  
**Files:** `pkg/game/save.go:229` vs `pkg/game/player.go:224`  
**Description:** Loaded players get 100-buffer. New characters get 256.  
**Fix:** Use a constant for buffer size.

### L-11: `ActiveAffects` Not Restored From Save
**Source:** Pass 5 (L5-3)  
**Files:** `pkg/game/save.go:254`  
**Description:** Creates slice of nil pointers. Actual affect data never deserialized. All spell effects lost on save/load.  
**Fix:** Iterate `data.Affects` and create proper `engine.Affect` objects.

### L-12: `cmdForce` Doesn't Execute the Command
**Source:** Pass 5 (L5-6), Pass 3 (H4 — see H-24)  
**Files:** `pkg/session/wizard_cmds.go:432-450`  
**Description:** Logs and confirms but never calls `ExecuteCommand`. Functional stub.  
**Fix:** See H-24 for security considerations when implementing.

### L-13: Session `tempData` Has No Type Safety
**Source:** Pass 3 (L4)  
**Files:** `pkg/session/manager.go`  
**Fix:** Consider typed wrappers or key constants with expected types.

### L-14: Player Name Validation Allows Dots, Spaces, Dashes
**Source:** Pass 3 (L5)  
**Files:** `pkg/validation/validation.go:10`  
**Fix:** Restrict to alphanumeric + underscore for MUD names.

### L-15: Damage Message Thresholds Need Verification
**Source:** Pass 4 (L1)  
**Files:** `pkg/combat/fight_core.go:860-873`  
**Fix:** Line-by-line verification against C source.

### L-16: `handleDeath` Lock Ordering Fragile But Currently Safe
**Source:** Pass 2 (M2)  
**Files:** `pkg/combat/engine.go:283-295`  
**Fix:** Document the interleaving contract.

---

## Recommended Fix Order

The following 10 fixes are the most impactful, ordered by priority and dependency chain.

### 1. C-12: Double-Close of `s.send` Channel + M-25: Consolidate Cleanup
**Why first:** Server crashes on every disconnect where cleanup paths race. Affects all players. Trivial fix (`sync.Once`). Prerequisite for safe work on all other session/channel changes. Consolidating `Unregister`/`UnregisterAndClose` into a single idempotent path prevents the entire class of cleanup bugs.  
**Effort:** Low  
**Files:** `pkg/session/manager.go`

### 2. C-14: Combat Engine Stale Refs + C-01: Dual Send Channel
**Why second:** C-14 causes guaranteed panics when any player disconnects during combat. C-01 is the root cause of all game-layer messages being silently lost — fixing it resolves H-11 (SpawnMob deadlock) as a side effect and is the single most impactful gameplay fix. These should be done together since both affect the session↔player message path.  
**Effort:** Medium  
**Files:** `pkg/game/player.go`, `pkg/session/manager.go`, `pkg/game/world.go`, `pkg/combat/engine.go`

### 3. C-04 + C-13: Save Locking + AdvanceLevel Restructure
**Why third:** Save corruption is silent data loss. Must fix together: adding RLock to save requires releasing lock in AdvanceLevel first to avoid deadlock.  
**Effort:** Medium  
**Files:** `pkg/game/save.go`, `pkg/game/level.go`

### 4. C-05 + C-06: Cross-Goroutine Map Panics (Agent Fields + Sneak/Hide)
**Why fourth:** Fatal runtime panics during any combat with agents (C-05) or any skill check involving sneak/hide (C-06). Both are low-effort fixes.  
**Effort:** Low  
**Files:** `pkg/session/agent_vars.go`, `pkg/game/skills.go`

### 5. C-08 + C-07: Security — File Write + Telnet Auth
**Why fifth:** C-08 is a privilege escalation from wizard to server root. C-07 is complete auth bypass in no-DB mode. Both are low-effort, high-impact security fixes.  
**Effort:** Low (C-08), Medium (C-07)  
**Files:** `pkg/session/wizard_cmds.go`, `pkg/telnet/listener.go`

### 6. C-09 + H-16: Saving Throws + Affect Duration
**Why sixth:** These two findings break the entire spell system. Every spell save rate is wrong (C-09) and every buff/debuff is 75× too short (H-16). Together they make magic non-functional.  
**Effort:** High (C-09), Medium (H-16)  
**Files:** `pkg/spells/saving_throws.go`, `pkg/engine/affect.go`

### 7. C-10 + C-11: Combat Command Stubs + Parry/Dodge
**Why seventh:** The skill-based combat system is entirely non-functional. Skills deal zero damage, parry doesn't work. Requires C-09 first (skill checks involve saving throws).  
**Effort:** High  
**Files:** `pkg/session/act_offensive.go`, `pkg/session/fight.go`, `pkg/combat/`

### 8. H-12 + H-15: Rate Limiting Bypass + Account Lockout
**Why eighth:** X-Forwarded-For spoofing makes all rate limiting useless. Combined with no account lockout = trivially brute-forceable.  
**Effort:** Medium  
**Files:** `pkg/auth/ratelimit.go`, `pkg/session/manager.go`

### 9. H-22 + M-27: Position Check Enforcement + Death Position Reset
**Why ninth:** Dead players can attack. Respawned players stuck in fighting position. Both are easy fixes that restore the position state machine.  
**Effort:** Low  
**Files:** `pkg/session/commands.go`, `pkg/game/death.go`

### 10. C-02 + H-01: Combat Globals → Injected Callbacks
**Why tenth:** 70+ unsynced function pointer vars cause race conditions in every combat round and make the combat engine untestable. Large refactor but enables safe future development. Doing this after the functional combat fixes (step 7) avoids double-work.  
**Effort:** High  
**Files:** `pkg/combat/fight_core.go`, `cmd/server/main.go`, `pkg/session/manager.go`
