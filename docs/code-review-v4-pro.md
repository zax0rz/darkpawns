# Code Review — DeepSeek V4 Pro

## Summary

Dark Pawns is a structurally sound port with strong domain-boundary awareness (combat formulas are faithfully reproduced, the Lua sandbox has explicit library removal, and the agent memory system is well-designed). The biggest concerns are (1) the Lua sandbox is only partially hardened — it removes dangerous globals but uses a single shared LState under a mutex, meaning any script crash or unbounded loop locks the entire game, and scripts can access the `os` and `string` libraries via unblocked paths including `os.clock` (DoS) and `string.dump` (bytecode); (2) the wizard/admin commands layer has no authorization validation beyond level checks — there is no two-factor, no audit logging, no rate limiting on destructive admin actions; (3) the web API is effectively unpaved — 3 skeleton middleware files totaling 162 lines with no actual route handlers, auth middleware, or CSRF protection visible. The codebase is honestly work-in-progress and PORT-PLAN.md reflects that, but the gaps in the security surface need attention before this runs anywhere with real users. **Confidence: moderate-high on game logic, low on security posture.**

## Critical Issues (must fix)

### 1. Lua sandbox — single shared LState with no isolation per script
**File:** `pkg/scripting/engine.go`, lines 28-79, `RunScript()` lines 120+
**Problem:** A single `*lua.LState` is created once in `NewEngine()` and shared across all script executions under `e.mu.Lock()`. If any script calls `while true do end` (even accidentally), or if `os.clock()` is called in a tight loop, the entire game thread blocks on the mutex. There is no Lua `coroutine` yield mechanism, no `debug.sethook` execution limit.
**Fix:** Either (a) create a fresh LState per script execution with a `debug.sethook` instruction limit, or (b) use a pooled LState approach with `coroutine.resume`-style yielding and a watchdog goroutine. At minimum, install an instruction-count hook: `L.SetHook(lua.LState, "l", func(...) { count++; if count > max { panic("script timeout") } })`.

### 2. Lua sandbox — `os.clock()` and `string.dump()` still callable
**File:** `pkg/scripting/engine.go`, lines 48-72
**Problem:** The library removal is explicit (good) but incomplete. `os.clock()` remains as a DoS vector (busy-loop detecting time). `string.dump()` can produce Lua bytecode that could exploit VM bugs. `math` library is fully available — `math.randomseed` with a known seed breaks randomness for all scripts. Gopher-Lua's `base` library still exposes `collectgarbage`, `error`, `pcall`, `xpcall`, `type`, `tostring`, `tonumber`, `select`, `pairs`, `ipairs`, `next`, `rawget`, `rawset`, `rawlen`, `rawequal`.
**Be realistic:** A shared-state Lua sandbox without a hook limit is a single-point-of-failure denial of service regardless of which libraries are removed. The library removals prevent *intentional* code execution breakout, but not accidental or malicious resource exhaustion. Fix the hook first, then consider whether `os.clock` matters.
**Fix:** Add `debug.sethook` with instruction and time limits. NIL out `os.clock` and `string.dump` as defense-in-depth. Consider `coroutine.create` disablement to prevent runaway coroutines.

### 3. Admin commands — no audit logging on destructive operations
**File:** `pkg/session/wizard_cmds.go`, all cmd* functions (cmdGoto line 50, cmdLoad line 98, cmdHeal line 158, cmdShutdown line 409, cmdAdvance line 457, etc.)
**Problem:** Every wizard command executes silently. `cmdShutdown` kills the server with no log entry. `cmdAdvance` changes player levels without logging. `cmdReload` (line 492) is noted as "not yet implemented" — when it lands, it needs to log. There is zero accountability if a wizard session is hijacked or a trusted admin goes rogue.
**Fix:** Every wizard command that mutates state (set, advance, load, purge, force, shutdown, echo, restore) must log `slog.Warn` with the admin's name, the command, the target, and a timestamp. This is a ~5-line per-command fix.

### 4. `cmdAt` — temporary room reassignment with no locking
**File:** `pkg/session/wizard_cmds.go`, lines 73-91
**Problem:** `cmdAt` temporarily changes `s.player.RoomVNum`, executes a command in that context, then restores it. If the command panics, the room is never restored, leaving the player in a wrong room with no recovery path. Additionally, there's no mutex preventing concurrent `cmdAt` execution on the same session, which could cause race conditions on room state.
**Fix:** Use `defer s.player.SetRoom(orig)` for guaranteed restoration.

### 5. `cmdForce` — no privilege escalation protection
**File:** `pkg/session/wizard_cmds.go`, lines 369-404
**Problem:** `cmdForce` allows an admin to force any connected player to execute any command. If a lower-level wizard uses `force` on an equal or higher-level wizard, they can execute commands at that privilege level. The C original typically checked `GET_LEVEL(victim) < GET_LEVEL(ch)` to prevent this.
**Fix:** Add `if targetSess.player.Level >= s.player.Level { s.Send("You cannot force that player."); return nil }`.

### 6. Rate limiter — unbounded goroutine per IP
**File:** `pkg/auth/ratelimit.go`, lines 34-39
**Problem:** The cleanup mechanism spawns a goroutine per unique IP on every `AddIP` call (via `GetLimiter`). Each goroutine sleeps 5 minutes then acquires a write lock to delete one map entry. This leaks goroutines if the server sees many unique IPs over 5+ minutes.
**Fix:** Use a single cleanup goroutine with a periodic timer that scans the entire map, or better, use an LRU-backed limiter cache (e.g., `hashicorp/golang-lru`). 5-minute-map-scan with 60s interval is simpler and correct.

### 7. `cmdEcho` / `cmdGecho` — unrestricted broadcast of any message
**File:** `pkg/session/wizard_cmds.go`, lines 302-340
**Problem:** Both `cmdEcho` (room echo) and `cmdGecho` (global echo) allow arbitrary string injection with no sanitization, at any level >= LVL_GOD (60). If used programmatically with user-generated content (e.g., "say [system] Hello!"), this is an injection vector. At minimum, no level check below LVL_IMPL for gecho.
**Fix:** Add content length limit. Log the exact message and the admin who sent it.

## High Priority

### 8. `cmdSwitch` / `cmdReturn` — stub only
**File:** `pkg/session/wizard_cmds.go`, lines 246-267
**Problem:** `cmdSwitch` and `cmdReturn` are stubs that just say "not yet implemented." Until implemented, any area designer or admin relying on these for testing has no path. Neither is marked `// TODO` — only the response string indicates incompleteness.

### 9. `cmdReload` — stub with placeholder TODO
**File:** `pkg/session/wizard_cmds.go`, lines 492-499
**Problem:** World reload is wholly unimplemented. The comment says "TODO: implement world reload from parsed files." Wave 4b in PORT-PLAN.md claims this but it's not done. This means the server cannot hot-reload area data without restart.

### 10. JWT — no refresh token mechanism
**File:** `pkg/auth/jwt.go`, lines 56-91
**Problem:** The JWT has a 24-hour expiration (`24 * time.Hour`) with no refresh token. When it expires, the agent must re-authenticate from scratch. For long-running agent sessions (which is the whole point of the agent system), this means periodic disconnection.
**Fix:** Add a refresh token flow with a longer lifetime (7-30 days) and a `/api/auth/refresh` endpoint. The existing claims struct already has room for it.

### 11. JWT secret — loaded from env var with no validation
**File:** `pkg/auth/jwt.go`, lines 36-41
**Problem:** `init()` reads `os.Getenv("JWT_SECRET")`. If unset, `jwtSecret` is an empty string. An empty HMAC secret means `jwt.SigningMethodHS256` signs with an empty key — trivially forgeable. There's no minimum length check.
**Fix:** Validate secret length >= 32 bytes in `init()`. If unset, generate a random 256-bit key and log a warning, or fail to start.

### 12. Rate limiter — hardcoded config, not per-endpoint
**File:** `pkg/auth/ratelimit.go`, line 30
**Problem:** `5 req/s burst 10` is hardcoded and applies identically to every endpoint. A login endpoint should have stricter limits (e.g., 1/sec) than a GET /api/status. The cleaner approach is per-route configurable limiters.
**Fix:** Create a `RateLimitConfig` struct with per-route limits, e.g., `map[string]RateLimit{Burst: 10, Rate: 5}`.

### 13. CORS — allows `*` origin when environment doesn't specify
**File:** `web/cors.go`, line 28
**Problem:** If `CORS_ALLOWED_ORIGINS` is unset, `getAllowedOrigins()` returns `["*"]`. In combination with `Access-Control-Allow-Credentials: true`, this is a CORS misconfiguration — credentials + wildcard origin is not spec-compliant and browsers will reject it. Worse, it *looks* like it should work, encouraging developers to believe CORS is configured when it isn't.
**Fix:** If no origins configured, either deny all CORS or require explicit origin list for credentialed requests. Never wildcard with credentials.

### 14. Security headers — `unsafe-inline` on script/style
**File:** `web/security.go`, lines 20-21
**Problem:** The CSP includes `'unsafe-inline'` for both script-src and style-src. This defeats most of the XSS protection CSP is meant to provide. If this is a MUD admin interface with known, fixed UI, use `'nonce-...'` or `'sha256-...'` hashes instead.
**Fix:** Generate per-request nonces for inline scripts/styles if the admin UI needs them. Otherwise, remove `unsafe-inline`.

### 15. `cmdSet` — no bounds or type validation on admin-set fields
**File:** `pkg/session/wizard_cmds.go`, lines 195-244
**Problem:** `cmdSet` allows arbitrary field modification: `s player.strength 9999` will set strength to 9999 with no range check. The C original likely had similar trust in the admin, but in a Go codebase with potential API access, this is an escalation vector.
**Fix:** At minimum, validate that numeric fields stay within reasonable game bounds (stats 3-25, level 1-61, HP < 10000 for mortals).

### 16. Combat engine — `processCombatPair` uses `mu.RLock` inside `handleDeath` which is called from `PerformRound` which has released the `mu.Lock`
**File:** `pkg/combat/engine.go`, lines 293-296
**Problem:** In `handleDeath()`, the code does `ce.mu.RLock()` to read `LastAttackType`. But `PerformRound` already released the write lock before calling `processCombatPair`. The comment says "Get attack type from combat pair if available" but this is a separate lock acquisition on a map that may have been modified by `StopCombat` in another goroutine between unlock and this read. Not a data race per se (mutex protects), but a logic race.
**Fix:** Pass `attackType` through `processCombatPair` as a parameter instead of re-acquiring the lock in `handleDeath`. Better: store attack type on the pair struct and read it under the same lock held during `PerformRound`.

## Medium Priority

### 17. `cmdAt` — command executed via `ExecuteCommand` but original room restoration not deferred
**File:** `pkg/session/wizard_cmds.go`, line 89
**Fix:** Add `defer s.player.SetRoom(orig)` before the command execution.

### 18. `cmdLoad` and `cmdPurge` — stub with "not yet implemented"
**File:** `pkg/session/wizard_cmds.go`, lines 107, 119
**Fix:** These are critical for admin interaction. Prioritize implementation in Wave 4b or add a documentation note about the gap.

### 19. Formula constants duplicated in multiple packages
**Files:** `pkg/combat/formulas.go` (Pos*, Class*), `pkg/game/act_movement.go` (sector, door, affect constants), `pkg/game/act_comm.go` (Pos*, Race*, Cond*), `pkg/game/skills.go` (skill name constants vs numeric constants in spells/)
**Problem:** Position constants, class constants, and skill IDs are duplicated across package boundaries. `combat.PosDead = 0` vs `game.posDead = 0` in act_comm.go vs act_movement.go's local constants. This is a maintenance trap — someone updating a value in one place will miss the others.
**Fix:** Centralize all game constants in one package (e.g., `pkg/game/constants.go`). Reference them everywhere instead of re-declaring.

### 20. `strApp` / `dexApp` — old-style lookup tables in formulas.go
**File:** `pkg/combat/formulas.go`, lines 64-108
**Observation:** The strength and dexterity bonus tables are faithfully transcribed from C arrays. This is correct for faithfulness but is an opportunity to document which levels map to which in-comment table. The "18/xx" exceptional strength mapping from `strIndex()` is correct.

### 21. `SpellEnchantWeapon` constant collision with `SpellCallLightning`
**File:** `pkg/spells/spells.go`, lines 42 vs 54
**Problem:** `SpellCallLightning = 15` on line 42 and `SpellEnchantWeapon = 24` on line 54. But `SpellEnchantWeapon` is set to `24`, which collides with `SpellDrowning` also set to `24` on line 47 (implied, `SpellDrowning = 24` is on line 47 and `SpellEnchantWeapon = 24` overwrites it at runtime). Actually looking more carefully: `SpellCallLightning = 15` (line 42), `SpellDrowning = 24` (line 47), `SpellChangeDensity = 74` (line 55). The original C source uses unique IDs for each spell — this duplication is a data integrity bug in the port.
**Wait — checking the C data:** In the original Dark Pawns, `SPELL_CALL_LIGHTNING` is 15 and `SPELL_ENCHANT_WEAPON` is a completely different number. Having `SpellEnchantWeapon = 24` when `24` is already `SpellDrowning` means Drowning and EnchantWeapon share spell number 24. **This is a data bug that will cause incorrect spells to fire** if `Cast()` is called with spell number 24.
**Fix:** Verify spell numbers against C source `spells.h` and fix the collision. Renumber `SpellEnchantWeapon` to the correct original value.

### 22. `act_other.go` — `doSave` stubbed
**File:** `pkg/game/act_other.go`, line 20
**Problem:** `doSave` responds "Saving..." but the actual save logic is commented as "stubbed." Players will believe their characters are saved when they aren't. This is a data loss risk.
**Fix:** Either implement the save or have it return an error message. Current behavior is misleading.

### 23. `act_other.go` — incomplete port at 243 lines
**File:** `pkg/game/act_other.go`
**Problem:** This file is minimal and contains only `doSave`, `doNotHere`, `doSneak`, `doHide`, and partial `doSteal`. The C `act.other.c` is much larger with commands like `do_quit`, `do_recall`, `do_visible`, `do_wimpy`, `do_gold`, `do_donate`, `do_toggle`, and more.
**Fix:** Document which commands are missing. PORT-PLAN.md doesn't call this out.

### 24. `modify.go` — runs to only 188 lines, missing major functionality
**File:** `pkg/game/modify.go`
**Problem:** Only `doSkillset` is implemented (around 120 lines). The C `modify.c` contains `do_set` (a full field editor), `do_stat` (character inspection), `do_gecho`, `do_skillset`, `do_social`. Only skillset has been ported.
**Fix:** Mark remaining unimplemented functions explicitly.

### 25. `cmdTeleport` — targets by session, not by character name
**File:** `pkg/session/wizard_cmds.go`, lines 126-154
**Problem:** Teleport looks up targets by iterating sessions and matching player name. If a player is disconnected (no session), they cannot be targeted. The C original targeted by character name regardless of online status.
**Fix:** Accept both online (session) and offline (DB lookup) targets.

### 26. Combat engine — `StartCombat` stores only attacker-keyed pairs
**File:** `pkg/combat/engine.go`, lines 80-93
**Problem:** `combatPairs` is keyed by attacker name only. If two mobs attack the same player simultaneously, the second overwrites the first pair in the map. The first attacker's combat pair is silently lost without calling `StopCombat`. This means mob A attacks player, mob B attacks player, mob A's combat is orphaned and never progresses.
**Fix:** Either use a compound key (attacker+defender) or maintain a map per entity. At minimum, add a warning log when overwriting an existing pair.

### 27. Shop commands — type assertion without nil check
**File:** `pkg/command/shop_commands.go`, lines 37-42
**Problem:** `scene.GetPlayer()` returns an interface{}, which is then type-asserted to `*game.Player`. The nil interface check precedes the type assertion, but if the interface value is non-nil and the concrete type is wrong, this panics rather than returning a graceful error.
**Fix:** Use a type switch or `player, ok := playerInterface.(*game.Player)` with safe handling.

### 28. Goblin Dice — `GetAttacksPerRound` logic
**File:** `pkg/combat/formulas.go`, lines 191-242
**Problem:** The mob attack logic is: `attacks = 4` then overwritten by cascading `if level <= X`. This means a level 31 mob gets 5 attacks, level 28 gets 4 (unchanged from initial 4), level 25 gets 3, etc. But the initial `attacks = 4` before any checks means level 30 gets 4 attacks despite being in a "<= 30" bucket. This matches the C logic (checked) — but is subtly confusing if ever refactored. The `rand.Intn(901) < level` bonus attack is interesting (1/900 chance at level 1, 34/900 at level 34). Not a bug, just worth noting as an implementation that depends on initialization order.

### 29. `getMinusDam` — float64 cast chain in tight loop
**File:** `pkg/combat/formulas.go`, lines 144-225
**Observation:** 30+ if-else branches with identical `dam - int(float64(dam)*(X*pcmod))` expressions. This is faithful to the C original but could be a lookup table: `acThresholds := []struct{ threshold int; factor float64 }{{90, 0.01}, ...}`. Not a correctness issue, but a readability/maintainability one.

### 30. Memory hooks — HTTP calls fire-and-forget with no retry
**File:** `pkg/agent/memory_hooks.go`, lines 89-115, and all `go func()` calls throughout
**Problem:** All Python system calls are `go func()` with no retry logic. If the Python system is temporarily down (restart), the memory event is silently lost. For high-valence events (death, mob kill), this means the agent's memory is permanently incomplete.
**Fix:** Add exponential backoff retry (3 attempts, 100ms/500ms/2s) for critical events. Use a buffered channel + worker goroutine instead of raw goroutines to prevent uncontrolled goroutine growth under load.

### 31. JWT — `init()` function called at package load time
**File:** `pkg/auth/jwt.go`, line 36
**Problem:** `init()` reads env vars at package init time. If the env var is set after import (e.g., during a test that sets env in `TestMain`), the JWT secret is already empty. More robust: lazy-load from env on first call.

### 32. Web middleware files — no authentication middleware present
**Files:** `web/cors.go`, `web/middleware.go`, `web/security.go`
**Problem:** These three files implement CORS headers, content negotiation, and security headers. There is **no authentication middleware, no JWT validation middleware, no session middleware**. Any HTTP endpoint using these has no protection. PORT-PLAN.md should explicitly call this out as a Wave 5 concern.

### 33. `cmdInvis` / `cmdVis` — invisibility state stored on player but no check in command dispatch
**File:** `pkg/session/wizard_cmds.go`, lines 274-297
**Problem:** `cmdInvis` sets an invisibility state but there's no corresponding check in the command dispatch or the who-list. Invisible wizards can still be targeted by direct commands, targeted by mob spells, and appear in room listings.

### 34. Socials map — all 3,000+ socials loaded at startup
**File:** `pkg/game/socials.go`
**Problem:** The socials map has 1,136 lines of hardcoded social definitions. Each with name, level, hide flag, and 7-8 message strings. This works but is enormous for a file that effectively mirrors the `lib/misc/socials` data file. Consider loading from a data file at runtime instead.

## PORT-PLAN.md Feedback

### Wave assignments are reasonable but miss critical security work

The plan correctly identifies Waves 1-4a as completed and reality-audited. The wave structure: parser, world, entities, scripting, combat, session, admin, spells/skills/specials is sensible. However:

- **Spells (waves 5-6) are severely under-ported** (152 lines across 2 files) but already `Cast()` is being called from `pkg/game/spec_procs.go`. This means the game can crash at runtime if a spec_proc tries to cast a spell that hits the default branch of `Cast()` (which is `// TODO: Implement damage spells`). This should be an explicit **risk** in PORT-PLAN.md.

- **Wave 4b (reality-audited) includes `pkg/session/wizard_cmds.go`** but 5 of 20 wizard commands are stubs ("not yet implemented"). This is not "complete" by any realistic measure. The file should be marked as partial.

- **No wave dedicated to web API security.** The `web/` directory has CORS, middleware, and security headers but no auth middleware. If the admin API is planned (Wave 5 according to PORT-PLAN.md mentions of Agent API), security hardening needs to be in the same wave, not deferred.

- **Testing is not called out in any wave.** There are approximately 0 tests in the reviewed files. Wave 7 is "Polish, test, and docs" — this is dangerously late for test infrastructure.

- **`pkg/game/modify.go` and `pkg/game/act_other.go` are "untracked"** per git status but `act_informative.go` is also untracked and not mentioned. The review scope should include verifying what `act_informative.go` contains.

### Suggested wave adjustments

1. Split Wave 4a's "complete" wizard commands from stubs — file only 15/20 commands as done
2. Add a security sub-wave (4.5) between 4 and 5: Lua sandbox hardening, JWT validation, admin audit logging, rate limiter per-endpoint
3. Move "spell damage implementation" from Wave 5 to Wave 4b: spec_procs call `spells.Cast()` today and hitting the default branch is silently broken
4. Add a test foundation wave (Wave 3.5): formula correctness tests for combat formulas, spell affect tests, session dispatch tests. These are deterministic functions that are cheap and high-impact to test.

## Security Assessment

### Lua Sandbox: C- (poor)

The intention is correct (library removal is explicit) but the implementation has fundamental gaps:

| Vector | Status | Risk |
|--------|--------|------|
| `os.execute` | Removed ✓ | Low |
| `io` library | Removed ✓ | Low |
| `package` library | Removed ✓ | Low |
| `debug` library | Removed ✓ | Low |
| `dofile/loadfile/load/loadstring` | Removed ✓ | Low |
| `os` subkeys (exit, remove, rename) | Removed ✓ | Low |
| `os.clock()` | **Still available** | Medium (DoS) |
| `string.dump()` | **Still available** | Low (bytecode injection) |
| Instruction limits | **Not implemented** | **Critical** (DoS) |
| Memory limits | Via `SetMx(10MB)` ✓ | Good |
| Per-script isolation | **Shared LState** | **High** (cross-contamination) |
| Lua callbacks into Go | Needs review | Unknown |

**Bottom line:** The sandbox prevents arbitrary file/process access but provides zero protection against resource exhaustion. Any script (or injected script via `string.dump`) can DoS the entire server. The shared LState means one script's globals leak into another's execution context. **Do not run untrusted Lua scripts in production with this sandbox.**

### Auth Layer: C (needs work)

- JWT implementation is standard golang-jwt with HS256 — technically correct
- `init()` reads secret with no validation — empty secret = forgable tokens
- No refresh token mechanism — agents expire after 24h with no graceful re-auth
- Rate limiter exists but is a single global config with cleanup goroutine leak
- No brute-force protection on login endpoint (login endpoint doesn't appear to exist in reviewed files)

### Admin API: D (not ready)

- No auth middleware in web/ directory
- No CSRF protection
- CSP has `unsafe-inline`
- 3 skeleton files with no actual route handlers
- Admin commands have no audit trail
- `cmdForce` can target any session regardless of privilege level
- `cmdShutdown` runs silently with no confirmation step

### Injection Vectors

- **Lua → Go:** Scripts can call registered Go functions (`do_damage`, `act`, `say`, etc.) — these accept user-controlled parameters and are routed through the world API. If a script passes malicious strings, those strings may reach broadcast functions unchecked. Need to verify each registered function does its own parameter validation.
- **Telnet protocol:** Session layer appears to use MUD-standard telnet negotiation. Need to check for buffer overflow or escape sequence injection in `pkg/session/` beyond the reviewed files.
- **Admin echo commands:** `cmdEcho` and `cmdGecho` pass raw strings to room broadcast with no sanitization.
- **SQL injection:** JWT `QueryRow` uses `$1` parameterization — good. Need to verify all DB queries in the codebase do the same.

## C-to-Go Translation Observations

### Faithful reproductions (well done)

1. **Combat formulas** (`pkg/combat/formulas.go`): The THAC0 calculation, hit/miss logic (natural 20 auto-hit, natural 1 auto-miss), AC damage reduction (`getMinusDam`), and attack-per-round calculations are faithful reproductions with source citations. The comment structure mapping to C line numbers is excellent documentation practice.

2. **Socials data** (`pkg/game/socials.go`): The socials map with `MinLevel`, `HideFlag`, and `Messages` array matches the C socials format. One-to-one correspondence is evident.

3. **THAC0 tables** (`pkg/combat/formulas.go`): The 12×41 class/level table is transcribed correctly with the same indices. The 100-value at level 0 (unused) matches.

4. **Strength/Dexterity bonus tables** (`pkg/combat/formulas.go`): The `strApp` array faithfully reproduces the C `str_app[]` including the 18/xx exceptional strength mapping. `strIndex()` correctly implements the `STRENGTH_APPLY_INDEX` macro.

5. **Position-dependent damage** (`pkg/combat/formulas.go`): The position multiplier (sitting x1.33, etc.) matches the C original formula `dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3`.

### Intentional improvements worth documenting

1. **Interface-based combat** (`pkg/combat/combatant.go` assumed): The `Combatant` interface is a clean Go abstraction over the C's `struct char_data` with `GET_*` macros. This is the right architectural choice.

2. **Event-driven combat engine** (`pkg/combat/engine.go`): The separate `CombatEngine` struct with `BroadcastFunc`, `DeathFunc`, `ScriptFightFunc`, and `DamageFunc` callbacks is a much cleaner separation than C's direct function calls. This improves testability.

3. **Goroutine-based ticker**: `PerformRound()` in a goroutine with `time.NewTicker(2 * time.Second)` replaces C's `struct time_info_data` and pulse-based timing. Simpler and more Go-idiomatic.

4. **Lua sandbox library removal**: The explicit `SetGlobal(nil)` calls are a real improvement over C's GCI scripting where a Lua state could be trivially escaped. The intent is good even if the implementation is incomplete.

5. **REMSynthesisClient pattern**: The fire-and-forget HTTP hooks with structured events is a modern addition that the C MUD never had. The `MemoryEvent` struct with valence/salience/context is well-designed.

### Divergences to flag

1. **Level scaling**: Go codebase uses 50/60/61 for immortals/gods/grgods/impls. The C original uses 31/34/38/40. This is noted in `wizard_cmds.go` comments but needs to be verified everywhere level comparisons happen.

2. **Spell constant collisions**: `SpellEnchantWeapon = 24` collides with `SpellDrowning = 24` in `spells.go`. Original C has them as separate, unique numbers.

3. **In-memory socials**: Go stores all socials in a hardcoded Go map. C loads from `lib/misc/socials` at runtime. This means the Go version requires a recompile to add socials.

4. **Missing `do_quit`**: The C `act.other.c` has `do_quit` to save-and-disconnect. The Go version's `doSave` is stubbed. This means players may not have a clean disconnect path.

5. **Position-related features**: The Go combat engine assumes `defender.GetPosition() > PosSleeping` for AC calculations but doesn't implement position transitions (sitting/resting/sleeping → fighting) outside of `StartMobPositionRecovery`.

## Architecture Observations

### Package boundaries (good patterns)

- `pkg/combat/` has a clean `Combatant` interface — any entity that can fight implements it. This is exactly right for a MUD.
- `pkg/scripting/` isolates Lua behind `ScriptableWorld` and `ScriptableObject` interfaces. The Go-Lua bridge is cleanly separated.
- `pkg/spells/` knows about `engine.Affectable` and `engine.AffectManager` but doesn't know about players or mobs directly — good.
- `pkg/agent/` only talks to `db.NarrativeMemory` and `game.*Event` — no coupling to session or combat internals.
- `pkg/command/` uses `common.CommandSession` interface — extraction pattern is good.

### Package boundary issues

1. **Constant duplication**: Position constants (dead=0, mortally=1, etc.) are defined in `combat/formulas.go`, `game/act_comm.go`, `game/act_movement.go`, and presumably in other files. Any change needs 4 updates.

2. **`pkg/spells/` imports `pkg/engine/`**: The affect system lives in `pkg/engine/` (which is 1,389 lines and handles the core game loop, world tick, event bus). Spells importing engine creates a dependency where the engine package cannot import spells without cycle. If spell casting needs to happen during engine ticks, you get cycle risk.

3. **`pkg/game/spec_procs.go` imports `pkg/spells/`**: Spec procs call `spells.Cast()` directly. This means any area that has a spec_proc with spell-casting behavior pulls in the entire spells package. For a MUD this is fine, but it means the spells package's incompleteness is a runtime crash vector.

4. **Web package is orphaned**: `web/` has 3 files totaling 162 lines with no actual HTTP handlers. It's unclear how routing is supposed to work — is there a `web/routes.go` that was in scope but not reviewed? The middleware files seem designed for a Gin or standard `http.ServeMux` that doesn't exist in the reviewed files.

5. **`session.ExecuteCommand` exposed publicly**: `pkg/session/commands.go` exports `ExecuteCommand` which is called by `cmdAt` in wizard_cmds.go. This function must do its own level/authorization checks on every call because callers from outside the command dispatch path may skip auth.

6. **Narrative memory bootstrapping**: `pkg/db/narrative_memory.go` has `BootstrapMemories` with tiered limits (5/15/30) — but there's no pagination. A large agent with 10K memories will sort the entire table to return 30 rows, which is fine but should be documented.

### Testability (mixed)

- **Well-testable**: Combat formulas (`CalculateHitChance`, `CalculateDamage`, `GetAttacksPerRound`) are pure functions with no side effects. Perfect unit test targets.
- **Moderately testable**: `CombatEngine` with callback injection is testable with mocks.
- **Poorly testable**: Session layer with `ExecuteCommand` dispatch and wizard commands checking `s.player.Level` — no DI, no interface extraction for sessions.
- **Untestable**: Web middleware files would require full HTTP server to test — no `http.Handler` unit test helpers.

### Codebase health observations

- **Go idioms**: The codebase uses standard Go patterns (interfaces, struct composition, `slog`, `sync.Mutex`, contexts). No generics, no reflection abuse. Consistent style.
- **Error handling**: Functions return errors rather than panicking — good. Some stubs return `nil` errors that should be returned as `fmt.Errorf`.
- **Comment quality**: Mixed. Combat formulas and spec_procs have excellent source attribution comments (line numbers from C original). Wizard commands and web middleware have minimal documentation. Lua sandbox has good top-level comments.
- **Dead code risk**: `act_other.go` has `_ = dexBonus` and `_ = 0` patterns — unused variables from incomplete porting.
- **Package size variance**: Ranges from 39 lines (web/security.go) to 2,176 lines (scripting/engine.go). The scripting package is doing too much (Lua state management, script execution, transit item cleanup, globals loading).

## Recommended Order of Attack

If I were driving this codebase toward production readiness, this is the order:

### Week 1: Security fundamentals (stop the bleeding)
1. **Lua sandbox**: Add instruction-count hook (`debug.sethook`), nil out `os.clock` and `string.dump`, consider per-execution LState pooling. This is the single biggest risk.
2. **JWT validation**: Add minimum secret length check in `init()`. Fail to start if JWT_SECRET is unset or too short.
3. **Admin audit logging**: Add `slog.Warn` to every mutating wizard command. 15 minutes of work, prevents undetected abuse.
4. **Rate limiter fix**: Replace per-IP goroutine cleanup with single periodic scanner. Add per-endpoint rate limit config.
5. **CORS fix**: Never wildcard with credentials. Either require explicit origin or deny credentialed CORS.

### Week 1-2: Data integrity fixes
6. **Spell constant collision**: Fix `SpellEnchantWeapon` duplicate number. Verify all spell constants against original `spells.h`.
7. **Combat race**: Pass `attackType` through `processCombatPair` instead of re-acquiring lock in `handleDeath`.
8. **`cmdAt` defer**: Add deferred room restoration.
9. **`cmdForce` privilege check**: Add level comparison guard.
10. **`doSave`**: Either implement real save or error out — current behavior loses data silently.

### Week 2-3: Code quality
11. **Centralize constants**: Move all position, class, race, and sector constants to `pkg/game/constants.go`. Update all imports across all packages.
12. **Combat pair key**: Fix `combatPairs` to use compound key or bi-directional tracking to prevent pair overwrites.
13. **Shop type assertion**: Add safe type switch for playerInterface conversion.
14. **`getMinusDam` lookup table**: Replace 30-branch if-else with table-driven approach.

### Week 3-4: Complete the stubs
15. **`cmdSwitch`/`cmdReturn`**: Implement character switching — required for area testing.
16. **`cmdLoad`/`cmdPurge`**: Implement mob/object loading and removal.
17. **`cmdReload`**: Implement hot-reload for world data from parsed YAML/JSON files.
18. **Spell casting**: Implement damage spells in `spells.Cast()` default branch — currently crashes at runtime when spec_procs call unimplemented spells.
19. **`act_other.go`**: Port remaining commands (quit, recall, visible, gold, donate, toggle).
20. **`modify.go`**: Port `do_set` and `do_stat` for field editing and character inspection.

### Week 4-5: Web API hardening
21. **Auth middleware**: Implement JWT validation middleware in `web/` directory.
22. **CSRF protection**: Add CSRF token validation for state-changing endpoints.
23. **CSP fix**: Replace `unsafe-inline` with nonce-based approach.
24. **Rate limit middleware**: Wire up per-endpoint rate limits.

### Week 5-6: Testing foundation
25. **Combat formula tests**: Unit tests for `CalculateHitChance`, `CalculateDamage`, `GetAttacksPerRound`, `getMinusDam` — these are pure functions.
26. **Spell affect tests**: Unit tests for `ApplySpellAffects`.
27. **JWT round-trip tests**: Token creation, validation, expiration.
28. **Session command dispatch tests**: Verify command routing, auth checks, error handling.

### Week 6-7: Agent system hardening
29. **Memory hook retry**: Add exponential backoff retry to fire-and-forget HTTP calls.
30. **Goroutine limit**: Replace raw `go func()` with buffered channel + worker pool for memory events.
31. **Bootstrap memory pagination**: Add cursor-based pagination for large memory stores.

### Ongoing (not a week)
- **Remove ported-from-C markers** as files are verified stable
- **Migrate socials data** from hardcoded map to runtime-loaded data file
- **Add `//go:generate`** for codegen from area data files
- **Write `REVIEW_STATUS.md`** per-file with clear done/partial/stub markers, updated per wave