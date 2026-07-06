# Dark Pawns — Full Code Review
**Date:** 2026-07-05
**Reviewer:** Claude Fable 5
**Scope:** Full codebase (~104K lines non-test Go, 27 pkg/ packages + cmd/server)
**Brief:** docs/briefs/BRIEF-2026-07-05-fable-full-audit.md

---

## Phase 0 — Evidence Run

| Check | Result |
|-------|--------|
| `go build ./...` | ✅ clean (exit 0) |
| `go vet ./...` | ✅ clean (exit 0) |
| `go test -race ./...` | ✅ **35 packages pass, zero DATA RACE reports, zero FAILs** |
| e2e smoke (`scripts/smoke_test_2b.py`) | not run this session (requires live server + DB); see Phase 3 coverage roadmap |

This is a materially different baseline from prior audits: the April review found panics-on-disconnect and map races that `-race` now confirms are gone from the tested paths. Caveat: race detection only covers exercised code, and overall coverage is 29.7% — the race-clean signal is strongest for combat/session paths that now have tests, weakest for `pkg/command` (8.8%).

---

## Phase 1 — Sweep

### 1A. Architecture Assessment

Package health (inspection depth varies — marked where classification is from survey only):

| Package | Lines | Status | Notes |
|---------|-------|--------|-------|
| `pkg/game` | 41,531 (130 files) | **FRAGILE** | One flat package, no internal boundaries; C-03 god package carried from April. Correctness has improved markedly but blast radius unchanged. |
| `pkg/session` | 16,344 (65 files) | **FRAGILE** | Session lifecycle now solid (reaper, sync.Once close); command dispatch, wizard commands, and connection plumbing still one namespace. |
| `pkg/spells` | 4,557 | **STABLE** | Golden-tested (108 metadata, 267 level tuples, damage/heal/affect formulas). Clean internal structure. |
| `pkg/combat` | 3,596 | **STABLE w/ caveat** | Formulas golden-tested, engine logic sound; caveat = ~70 package-level function hooks (C-02, fight_core.go:14–80). |
| `pkg/command` | 3,513 | **FRAGILE** | 8.8% coverage; the DP-901 class of bug lived here undetected. Registry has `MinLevel` field that is never enforced (see 1D). |
| `pkg/scripting` | 3,415 | **STABLE** | Sandbox is genuinely good (see 1D); per-run global scrubbing, 5s timeout, load-failure caching (DP-903). |
| `pkg/admin` | 3,117 | STABLE (survey) | JWT + role-gated router; audit logger races fixed (DP-830). |
| `pkg/agentcli` | 3,025 | not inspected | AI-agent tooling; out of gameplay blast radius. |
| `pkg/optimization` | 2,488 | STABLE (survey) | Worker-pool shutdown deadlock fixed via clawpatch (f703052). |
| `pkg/parser` | 1,707 | **STABLE** | Diku format parsing, tilde bug fixed (DP-843). |
| `pkg/engine` | 1,683 | **STABLE** | GameLoop pulse architecture matches comm.c; affect system tick-based. |
| `pkg/db` | 1,479 | STABLE (survey) | DSN escaping fixed (DP-845); decision-log partitioning guarded. |
| `pkg/telnet` | 1,145 | **STABLE** | Input bounded (maxInputLen, 4096 subneg cap), write deadline (DP-804), pre-auth idle timeout (DP-912). |
| `pkg/dreaming` | 960 | not inspected | Background narrative; non-critical path. |
| `pkg/privacy` | 835 | not inspected | |
| `pkg/moderation` | 777 | STABLE (survey) | Penalty audit trail added (DP-831). |
| `pkg/auth` | 562 | **STABLE** | JWT secret validated at boot (DP-910); H-25 (rotation) still open. |
| `pkg/events` | 523 | STABLE | Bus unsubscribe bug (M-10) fixed. |
| `pkg/agent` | 414 | STABLE (survey) | Memory hooks, 5 goroutine launch sites. |
| `pkg/testutil` | 337 | STABLE | |
| `pkg/grapevine` | 327 | STABLE (survey) | Clean Start/Stop lifecycle. |
| `pkg/metrics` | 249 | STABLE | promauto refactor (DP-827). |
| `pkg/storage` | 236 | STABLE (survey) | |
| `pkg/audit` | 164 | STABLE | Init leak fixed (DP-799). |
| `pkg/common` | 160 | STABLE | Much thinner than April (was 27-method interfaces). |
| `pkg/secrets` | 127 | STABLE (survey) | |
| `pkg/validation` | 65 | STABLE | Tiny; ValidateInput integration per M-30. |
| `cmd/server` | 453 | **FRAGILE** | Ordered graceful shutdown exists but misses three goroutine families (Finding F3); M-07 manual wiring acknowledged in file header. |

**Global state:** ~150 package-level `var` declarations excluding error values/regexes. Dominated by: the ~70 combat hooks (C-02), `game.ScriptEngine`, spec/social registries (init-time write, runtime read — documented safe), and session command registry.

**CustomData:** 74 non-test uses, concentrated in `pkg/game/object.go` (37) and `mob.go` (11) — consistent with its role as the tagged-union escape hatch. No new sprawl since the RuntimeState refactor.

**Import graph:** No cycles (`go list ./...` clean). `pkg/game` has highest fan-in (13 importers). Direction is clean: session→game→combat/engine/parser; spells self-contained.

### Top 10 architecture risks

1. `pkg/combat`'s ~70 package-level function hooks (C-02) — untestable wiring, nil-panic risk on partial init, write-at-boot/read-at-runtime with no synchronization (`fight_core.go:14-80`).
2. `pkg/game` flat god package (C-03) — 130 files, one namespace.
3. Shutdown doesn't stop AI ticker / point ticker / periodic resets → world mutates during shutdown save (F3).
4. `PointUpdate` double-driven by two independent tickers (F4).
5. Dual damage pipelines (engine tick vs `damage_stubs` path) have diverged: the skill/spell path drops XP award (F1) and player death isn't idempotent across them (F2).
6. Dispatch-level privilege enforcement absent — 48 per-handler `checkLevel` calls are the only wall (F10).
7. `ZoneDispatcher` implemented but unwired (`main.go:339-341`) — the "two-layer dead code" pattern from the last review persists here.
8. Live combat messages bypass the golden-tested message tables (F7).
9. `pkg/command` at 8.8% coverage while owning skill execution.
10. Manual wiring in `main.go` (M-07, acknowledged) — hook omission is a silent failure mode.

### 1B. C Port Completeness

- **Spec procs:** C sources define **122 unique `SPECIAL()`** functions (41 in spec_procs.c, 50 in spec_procs2.c, 30 in spec_procs3.c, plus scattered). Go registers **119**, including all 41 from spec_procs.c and 2 Go-originals (`couch`, `mini_thief`). Unregistered: `gen_board` (→ boards.go system), `postmaster` (→ mail.go system), `receptionist`/`cryogenicist` (→ rent system in objsave.go — parity unverified), and **`black_horn` (src/new_cmds2.c:624) which is genuinely missing** — a held horn item that summons zone mobs on `use`.
- **Stale marker:** `spec_procs.go:944` still says "Wave 4a: remaining functions will be added in Wave 5" — obsolete, all 41 are ported; delete the comment.
- **Commands:** 55 `Do*` handlers in game/session; grep for stub/TODO markers shows concentrations in `wiz_zone.go`, `houses.go`, `act_item_stubs.go` — none are player-facing crash risks, but `houses.go` and board-write flows deserve a functional pass.

### 1C. Concurrency Model

**Goroutine inventory:** engine GameLoop (100ms pulse, owns point/violence/mobact/weather/affect/idle/reaper callbacks), CombatEngine tick (2s) + mob position recovery (3s), World AI ticker (10s), World point-update ticker (30s — F4 duplicate), zone-reset + periodic-reset goroutines, telnet accept loop + per-conn goroutines, WebSocket read/write pumps per session, Grapevine client, event queue, agent memory hooks, decision-log writer.

**Shutdown:** main.go has a properly *ordered* sequence (gameLoop → telnet → HTTP → session drain → decision-log flush → zone wg → SaveWorld) — a big improvement. But `StopAITicker()` and `StopPeriodicResets()` exist and are **never called** (F3), and the 30s point ticker has no stop handle at all.

**Lock hierarchy:** documented contract exists at `pkg/combat/engine.go:449-463` (ce.mu → World.mu → Player.mu) — this is the right pattern and it's followed in the death path. `-race` suite is clean.

**RWMutex usage:** read-heavy paths (PerformRound snapshot, IsFighting, playerToSaveData) correctly use RLock.

### 1D. Security Surface

- **Lua sandbox** (`pkg/scripting/engine.go:56-110`): `dofile/loadfile/load/loadstring/package/debug/io` removed; registry `_LOADED`/`_PRELOAD` cleared (kills `require`); `os` selectively stripped (execute/exit/getenv/setenv/remove/rename/tmpname/clock/setlocale); `string.dump` and `math.randomseed` removed; per-run global scrubbing prevents cross-script leaks; 5s context timeout. **Residual:** no memory cap (gopher-lua limitation — a script can allocate aggressively for its 5s window), `collectgarbage` still callable (F16). The ~60 registered Go bridge functions were not individually audited this session.
- **Privilege:** all 48 wizard handlers begin with `checkLevel(s, LVL_IMMORT)` (verified per-file counts). But the registry's `MinLevel` field is dead — dispatch enforces MinPosition and WaitState only (`commands.go:592-635`). One future handler without `checkLevel` = mortal escalation (F10).
- **Input:** telnet line length bounded, subnegotiation capped at 4096, WebSocket 16KB limit; name validation present.
- **Errcheck:** **zero `#nosec G104` remain in non-test code** (down from ~135). Remaining swallowed errors are in corpse/donation item transfer (F9).
- **Open from prior review:** H-25 JWT 24h lifetime without rotation.

### 1E. Prior Findings Regression Check

**April backlog (docs/reviews/BACKLOG.md, 86 findings, Opus passes 1–5):** status table claims 75 fixed / 9 deferred / 2 open. Spot verification of the highest-impact items against current HEAD:

| ID | Finding | Status at HEAD |
|----|---------|----------------|
| C-01 | Dual send channel, messages lost | **CLOSED** |
| C-04 | save.go reads Player without lock | **CLOSED** — `playerToSaveData` takes RLock (save.go:162-164) |
| C-12 | double-close of s.send | **CLOSED** — sync.Once + reaper consolidation (f919677) |
| C-02 | ~70 package-level func hooks in `pkg/combat` | **NEVER FIXED (deferred)** — fight_core.go:14-80 unchanged |
| C-03 | `pkg/game` god package | **NEVER FIXED (deferred)** — still flat, 130 files |
| H-25 | JWT 24h lifetime, no rotation | **OPEN** |
| M-23 | mobile_activity not fully ported | **PARTIALLY CLOSED** — wander cadence + SENTINEL/STAY_ZONE fixed (DP-908); scavenger/memory/helper unverified |
| L-11 | ActiveAffects not restored from save | **CLOSED** — restoreAffects (save.go:315,327) |
| M-29 | telnet unbounded input | **CLOSED** — maxInputLen enforced |

**Sprint 2 / Fable review fixes (2026-07-03) — all verified landed at HEAD:** DP-898/899 (flag pipeline), DP-900 (combat reciprocity — verified in engine.go PerformRound), DP-901 (skill damage death pipeline — verified in skill_commands.go:1558/damage_stubs.go), DP-902/928 (session reaper/ghosts), DP-903 (script-failure caching), DP-904 (dead-code ratchet), DP-905 (shadow damage path deleted), DP-906 (backstab gates), DP-907–913 (resolver, wander, char-create, JWT boot check, save error logging, idle timeout, cosmetics).

**Verdict:** No REGRESSED items. The carried debt is C-02, C-03, H-25, and the unverified halves of M-23.

### Phase 1 Prioritization → Phase 2 targets

1. Kill/death pipeline (both damage paths) — player-facing correctness
2. Shutdown/tick hygiene — data-loss risk on every restart
3. Spec proc fidelity sample — largest untested C surface
4. Lua sandbox — security surface
5. Save/load roundtrip — data-loss risk

---

## Phase 2 — Deep Dive

### Depth achieved

| Area | Depth |
|------|-------|
| B1 Combat data race (engine.go, skill_combat.go, player_combat.go, damage_stubs.go, death.go, combat_helpers.go) | **FULL** (fight_core.go PARTIAL — hooks + structure) |
| B2 Lua sandbox (engine.go sandbox setup, timeout, dispatch) | **PARTIAL** — sandbox FULL, 60+ bridge functions not individually audited |
| B3 Save/load (save.go structure, locking, affect restore) | **PARTIAL** — locking + restore verified; no field-by-field roundtrip audit |
| B4 Error handling (repo-wide grep + spot reads) | **FULL** at grep level |
| B5 Spec procs (registration cross-ref + 3-proc fidelity sample) | **PARTIAL** — cross-ref FULL; per-proc fidelity sampled (dump, snake, summoner) of 117 |
| Session lifecycle | **SKIMMED** — regression-verified via commits, not re-read |
| pkg/session full read, mobact behaviors, shop math | **NOT REACHED** → Phase 3 coverage roadmap |

### Findings

**[F1] [HIGH] [fidelity] Skill and spell kills award zero XP, no kill counter, no events**
- File: `pkg/game/damage_stubs.go:60,104` (also :53,97 for the player branch's missing killer)
- Description: `DoSpellDamage` and `doDamage` call `w.handleMobDeath(v, nil, 303)` — killer explicitly nil — instead of `w.HandleDeath(victim, killer, attackType)`. XP award (`AwardMobKillXP`), kill counter + `counter_procs` milestones, `MobKilledEvent`, memory hooks, and AutoGold/AutoLoot killer logic all live in `HandleDeath` (death.go:112-168) and are skipped. Any mob whose killing blow is a backstab/kick/bash/circle/charge or damage spell yields nothing. The DP-901 comment at `skill_commands.go:1558` claims XP is handled — it is not.
- C reference: fight.c `damage()` → `die_with_killer()` → `group_gain()` — XP flows regardless of what dealt the killing blow.
- Suggested fix: replace both `handleMobDeath(v, nil, 303)` calls with `w.HandleDeath(v, attackerAsCombatant, attackType)` (attacker is already in scope; assert to `combat.Combatant`); pass the real attack type through instead of hardcoded 303.
- Effort: S

**[F2] [HIGH] [concurrency] Player death is not idempotent — concurrent kills double-punish**
- File: `pkg/game/death.go:359` (`handlePlayerDeath`)
- Description: mob death is guarded (activeMobs delete under w.mu, death.go:236-241, second caller no-ops) but player death has no equivalent. The combat engine tick (its own goroutine) and a skill/spell kill (session goroutine) can both drop the same player to ≤0 and both run the full death path: double EXP loss, double CON roll, two corpses (second empty), double respawn. Window is small but real; `-race` won't flag it (it's logical, not a data race).
- Suggested fix: add a `dying atomic.Bool` (or flag under p.mu) checked-and-set at the top of `handlePlayerDeath`; reset on respawn.
- Effort: S

**[F3] [HIGH] [concurrency] Shutdown never stops AI ticker, point ticker, or periodic resets — world mutates during shutdown save**
- File: `cmd/server/main.go:412-440`; `pkg/game/world.go:929-934` (`StopAITicker`, never called); `pkg/game/spawner.go:686-688` (`StopPeriodicResets`, never called); `pkg/game/ai.go:206-219` (point ticker, no external stop)
- Description: the shutdown sequence stops the game loop, telnet, HTTP, and sessions — but `AITick` (mob AI mutating rooms/mobs), the 30s `PointUpdate` ticker, and 60s periodic zone resets keep running while `game.SaveWorld` serializes world state. Same class as the clawpatch fix 02b452e ("game loop continues mutating world during shutdown save") — that fix covered the engine loop only.
- Suggested fix: call `gameWorld.StopAITicker()` (which closes `w.done`, also stopping the point ticker) and `StopPeriodicResets()` in main.go's shutdown sequence, before `wg.Wait()`.
- Effort: S

**[F4] [MEDIUM] [fidelity] PointUpdate double-driven — regen/hunger ticks from two independent clocks**
- File: `pkg/game/world.go:193` (NewWorld starts 30s ticker) vs `cmd/server/main.go:221-223` (gameLoop OnPointUpdate, every mud hour per engine pulse constants)
- Description: two drivers for the same function. Players regen/hunger at roughly 3.5× the C rate (C: every 75s tick). Comment at ai.go:205 says the 30s tick is deliberate ("faster tick"), but having *both* cannot be.
- Suggested fix: pick one driver. If the 30s cadence is a deliberate design deviation, remove the gameLoop callback; if C fidelity is wanted, remove the NewWorld ticker.
- Effort: S

**[F5] [HIGH] [fidelity] specDump swallows the drop command and the award can never fire**
- File: `pkg/game/spec_procs.go:155-174`; dispatch at `pkg/session/commands.go:526-548`
- Description: specs run *before* command execution (matching C). C's `dump` calls `do_drop(ch,...)` itself, then values what landed in the room. Go's `specDump` never performs the drop: on `drop` it vacuums the room (`roomCleanup` at :157, discarding value), re-vacuums (:163 — the item is still in the player's inventory, so value=0), then returns true, which consumes the command. Net effect: in a dump room, `drop` does nothing at all and the XP/gold award is dead code. Also the value formula differs (C: `MAX(1,MIN(10,COST/10))` per item).
- Suggested fix: have specDump invoke the real drop (via `w.executeCommand` or by moving the named item to the room), then value+extract room contents with the C formula.
- Effort: S

**[F6] [MEDIUM] [fidelity] Systemic off-by-one: `randN(N)` ports C's inclusive `number(0,N)`**
- File: `pkg/game/spec_procs.go:39-46` (helper); ~42 call sites across spec_procs*.go (e.g. specSnake :181); C has 44 `number(0,` sites in spec_procs*.c
- Description: `number(0,N)` yields N+1 outcomes; `randN(N)` yields N. Every `== 0` gate fires slightly too often (snake poison bite: 1/32 instead of 1/33 at level 0). Same class as H-19 which was fixed in combat.
- Suggested fix: audit each site; where C is `number(0,N)==0`, use `randN(N+1)==0`. Consider adding `number(from,to)` (already exists in remort_helpers.go:4) as the only porting primitive and deprecating bare `randN` for C-ported code.
- Effort: S (mechanical, ~42 sites)

**[F7] [MEDIUM] [fidelity] Live combat messages bypass the C damage-message system**
- File: `pkg/combat/engine.go:414-431` (`sendHitMessage`), :434-447 (`sendMissMessage`)
- Description: the live combat tick prints "You hit X for 7 damage!" — numeric damage, single generic verb. C's `dam_message`/skill message tables (severity strings like "barely scratch", weapon-type verbs — which have golden tests in this repo!) exist in fight_core.go but the engine path doesn't use them. Combat *feels* wrong even though formulas are right.
- C reference: fight.c dam_message + attack_hit_text usage.
- Suggested fix: route `sendHitMessage` through `SkillMessageFunc`/the damage-message tables keyed by damage severity and weapon type; keep numeric output behind a PRF flag if desired for agents.
- Effort: M

**[F8] [MEDIUM] [architecture] C-02 carried: ~70 unsynchronized package-level function hooks in pkg/combat**
- File: `pkg/combat/fight_core.go:14-80`
- Description: unchanged from April (deliberately deferred). Wired at boot from main/session, read from combat goroutines; no nil-guards at construction; blocks testing two engines with different wiring.
- Suggested fix: incremental — introduce a `GameCallbacks` struct on `CombatEngine`, migrate hooks in batches, validate non-nil at construction. Pairs naturally with F7's message routing work.
- Effort: L

**[F9] [MEDIUM] [error-handling] Corpse/donation item transfers swallow errors — silent item loss**
- File: `pkg/game/death.go:795-834` (six `_ =` on MoveObjectToContainer/MoveObjectToRoom), `pkg/game/item_donate.go:34,65`
- Description: if a move fails during corpse creation, the item vanishes with no log and no fallback — invisible player-facing data loss in the most emotionally charged moment the game has.
- Suggested fix: log at Error with item vnum + player, and fall back to dropping the object in the room on container-move failure.
- Effort: S

**[F10] [MEDIUM] [security] No dispatch-level privilege gate — registry MinLevel is dead code**
- File: `pkg/command/registry.go:28` (field), `pkg/session/commands.go:592-635` (dispatch enforces MinPosition/WaitState only)
- Description: all 48 current wizard handlers self-gate with `checkLevel` (verified), so there is no exploit *today* — but the invariant lives in 48 places instead of 1. The next wizard command added without the boilerplate is a silent mortal-escalation.
- Suggested fix: populate MinLevel on wizard registrations and enforce `s.player.GetLevel() >= entry.MinLevel` in ExecuteCommand alongside the MinPosition gate. Keep the per-handler checks as defense in depth.
- Effort: S

**[F11] [LOW] [fidelity] specSnake victim messaging loses TO_VICT form**
- File: `pkg/game/spec_procs.go:188`
- Description: C sends "$n bites you!" to the victim and "$n bites $N!" to others; Go `roomMessage` sends third-person to everyone including the victim. Pattern likely repeats in other roomMessage-based specs.
- Suggested fix: victim-aware message helper (ActMessage already exists in skill_combat.go — reuse it).
- Effort: S

**[F12] [LOW] [concurrency] DoBash move-point check-then-act**
- File: `pkg/game/skill_combat.go:144-147`
- Description: `GetMove()<10` then `SetMove(GetMove()-10)` — two lock acquisitions; a concurrent regen tick between them is lost/overwritten. Cosmetic-scale impact.
- Suggested fix: `Player.SpendMove(n int) bool` doing check+deduct under one lock.
- Effort: S

**[F13] [LOW] [dead-code] ZoneDispatcher implemented but unwired**
- File: `cmd/server/main.go:339-341`, `pkg/game/zone_dispatcher.go`
- Description: the per-zone goroutine dispatcher exists with tests but production uses serial periodic resets. This is the surviving instance of the "two-layer architecture" pattern the last review flagged. Either wire it in (after F3) or delete it — the U1000 ratchet (DP-904) argues for a decision, not limbo.
- Effort: S (delete) / M (wire in)

**[F14] [MEDIUM] [architecture] C-03 carried: pkg/game flat god package**
- File: `pkg/game/` (130 non-test files, one namespace)
- Description: unchanged from April by explicit deferral. Correctness fixes have de-risked it, but every mechanic still compiles against every other. The `game/systems` and `game/data` subdirs show the intended direction.
- Suggested fix: extract leaf clusters first (boards, socials, houses, clans — low fan-in), not core (world/player/mob). One extraction per sprint; don't big-bang.
- Effort: L (incremental)

**[F15] [LOW] [dead-code] Stale "Wave 4a/Wave 5" comment**
- File: `pkg/game/spec_procs.go:944`
- Description: claims spec procs remain to port; all 41 from spec_procs.c are registered. Misleads future contributors (and future audits).
- Effort: trivial

**[F16] [LOW] [security] Lua sandbox has no memory ceiling**
- File: `pkg/scripting/engine.go:56-110,376`
- Description: 5s CPU timeout is enforced via context, but gopher-lua has no allocation cap — a hostile script can allocate large tables for its 5s window; `collectgarbage` is also still exposed. Scripts are operator-authored (not player-authored) so exposure is low.
- Suggested fix: document the trust model in engine.go; optionally nil `collectgarbage` and add a table-size guard in bridge functions that build tables from world state.
- Effort: S

**[F17] [LOW] [performance] Spec dispatch scans all mobs + equipment + inventory + room items on every command**
- File: `pkg/session/commands.go:526-590`
- Description: four linear scans with map lookups per keystroke per player. Fine at current scale; will show up under load. Cheap win: skip scan when the registries are empty for the room's contents (precomputed per-room spec presence bit).
- Effort: M

**[F18] [LOW] [error-handling] Player save format has no version field**
- File: `pkg/game/save.go` (savePlayerData)
- Description: field additions are backward-tolerant via JSON zero values, but silent — a renamed field drops data with no warning (the L-11 affect-loss bug was this class). No version/migration hook exists.
- Suggested fix: add `SaveVersion int` + a load-time check that logs on mismatch.
- Effort: S

**[F19] [LOW] [fidelity] black_horn spec proc missing; rent-system parity unverified**
- File: C `src/new_cmds2.c:624`; no Go registration
- Description: the black horn (held item, `use` summons one of six zone mobs) is the only genuinely unported spec of 122. `receptionist`/`cryogenicist` (rent) are claimed by the objsave.go rent system but parity wasn't verified this session.
- Effort: S (horn) + verification task (rent)

### Positives worth recording

- `-race` clean across 35 packages; zero `#nosec G104` in non-test code (was ~135).
- DP-900 reciprocity implementation in PerformRound is *better* than the brief's suggested code — snapshot-under-RLock with FIGHTING-target resolution mirrors C's semantics correctly.
- Mob death is properly idempotent (MED-011 guard); lock hierarchy is documented and followed in the death path.
- Sandbox quality is well above typical MUD ports.
- Save path locking (C-04/C-13) done right: RLock in playerToSaveData, no save-under-lock in AdvanceLevel.

---

## Phase 3 — Roadmap

### 3A. Prioritization Matrix

| Pri | Finding | Category | Severity | Effort | Package | One-line |
|-----|---------|----------|----------|--------|---------|----------|
| 1 | F1 | fidelity | HIGH | S | game | Skill/spell kills award no XP/kills/events |
| 2 | F2 | concurrency | HIGH | S | game | Player death not idempotent — double death |
| 3 | F3 | concurrency | HIGH | S | cmd/server, game | Shutdown leaves 3 ticker families mutating world during save |
| 4 | F5 | fidelity | HIGH | S | game | specDump eats `drop`; award unreachable |
| 5 | F4 | fidelity | MED | S | game, cmd/server | PointUpdate double-driven (regen ~3.5×) |
| 6 | F9 | error-handling | MED | S | game | Corpse transfer errors = silent item loss |
| 7 | F10 | security | MED | S | command, session | Enforce MinLevel at dispatch |
| 8 | F7 | fidelity | MED | M | combat | Live combat text bypasses dam_message tables |
| 9 | F6 | fidelity | MED | S | game | randN off-by-one, ~42 spec proc sites |
| 10 | F19 | fidelity | LOW | S | game | black_horn missing; rent parity unverified |
| 11 | F8 | architecture | MED | L | combat | C-02 hooks → GameCallbacks struct |
| 12 | F14 | architecture | MED | L | game | C-03 god package — leaf extractions |
| 13 | F13 | dead-code | LOW | S/M | game | ZoneDispatcher: wire or delete |
| 14 | F16 | security | LOW | S | scripting | Lua memory ceiling / trust-model doc |
| 15 | F18 | error-handling | LOW | S | game | Save version field |
| 16 | F11 | fidelity | LOW | S | game | Spec messages lose TO_VICT form |
| 17 | F12 | concurrency | LOW | S | game | SpendMove check-then-act |
| 18 | F17 | performance | LOW | M | session | Per-command spec scans |
| 19 | F15 | dead-code | LOW | trivial | game | Stale Wave comment |
| — | H-25 | security | HIGH* | M | auth/session | JWT rotation (carried; *severity per original review) |

### 3B. Work Streams

**Stream 1 — Kill Pipeline Correctness (F1, F2, F5, F11)** — effort S×4, `pkg/game` + `pkg/command`. The player-visible payoff stream: kills pay out, deaths are single, dump rooms work. No dependencies; do first. *All four fit one session.*

**Stream 2 — Tick & Shutdown Hygiene (F3, F4, F13)** — effort S+S+S/M, `cmd/server` + `pkg/game`. Prevents shutdown save corruption and normalizes regen. F13's decision (wire vs delete ZoneDispatcher) should follow F3 since wiring it changes the shutdown set.

**Stream 3 — Fidelity Polish (F6, F7, F19)** — effort S+M+S, `pkg/game` + `pkg/combat`. Combat feel and spec proc statistics. F7 pairs with Stream 4's hook work if scheduled together.

**Stream 4 — Architecture Debt (F8, F14, F10, F17)** — effort L, long-running. Order: F10 (dispatch gate, small and closes a security class) → F8 (GameCallbacks, unlocks combat testability) → F14 leaf extractions (boards/socials/houses first) → F17 opportunistically.

**Stream 5 — Hardening (F9, F16, F18, H-25)** — effort S×3+M. Independent; good background-agent fodder (Reek/Daeron sized).

**Quick wins (single session):** F1 + F2 + F5 (Stream 1 core), or F3 + F4 + F15 + F9 (hygiene batch).

### 3C. Coverage Roadmap (next 30 days)

Current: total 29.7% — combat 59.7, spells 36.1, session 24.1, scripting 24.1, game 21.4, command **8.8**.

1. **`pkg/command` functional tests** (8.8% → ~40%): drive skill commands end-to-end against a test world and *assert XP/gold/kill-counter deltas* — F1 existed precisely because no test asserts the payout. This is the highest-yield test investment in the repo. **(COV-1, open)**
2. **Death-path concurrency tests** (`-race`): two goroutines delivering lethal damage to the same player/mob simultaneously; assert exactly one corpse, one EXP loss. Locks in F2's fix.
3. **Shutdown drain test**: boot world, start tickers, trigger shutdown, assert no goroutine mutates world after SaveWorld begins (can use a mutation counter). Locks in F3.
4. ~~**Spec proc table test**~~: ✅ **LANDED (COV-4)** — `TestSpecProc_SmokeAll` covers all 122 specs, golden fidelity tests for top-10. Also fixed 2 live panics (specTeleporter/specPetShops nil guards).
5. ~~**e2e smoke in CI**~~: ✅ **LANDED (COV-5)** — Telnet e2e tests (`TestTelnetSmoke_*`) in CI with Postgres. Persistence round-trip test (`TestTelnetSmoke_PersistenceRoundTrip`) gated on `DP_TEST_DB_URL`. Fixed DP-591 race: `handleLogin` now uses `CloseSend()` instead of `Close()` so error messages flush before disconnect. Python WS smoke (`smoke_test_2b.py`) soft-gated.
6. **Lua adversarial suite**: ✅ **LANDED (COV-6)** — `__index` metamethod bypass fixed; all sandbox tests pass with stub functions. **(also covered by initial sandbox hardening)**

Realistic 30-day targets: total ~38-40%, command ~40%, game ~28%, with CI smoke as the true gate. **COV-4/5/6 complete — COV-1 (command functional tests) is the remaining high-yield coverage investment.**

### Executive Summary

Dark Pawns is in the best shape it has been: the build, vet, and full `-race` suite are clean; the April backlog's 14 CRITICALs are verifiably fixed with only the two explicitly-deferred structural items (combat hook globals, game god package) carried; and the Sprint 2 fixes from the last Fable review all landed and held — combat is reciprocal, the skill damage path handles death, sessions reap cleanly. The audit's significant new findings cluster in one theme: **the two damage pipelines have drifted apart** — skill/spell killing blows award no XP and skip the event/memory system (F1), player death isn't idempotent across the two paths (F2), and live combat text bypasses the golden-tested message tables (F7). Add three shutdown-hygiene gaps (F3–F4) that can corrupt the world save on restart, one confirmed-broken spec proc (F5, dump rooms eat `drop`), and a systemic RNG off-by-one across ~42 spec proc sites (F6) — all small-effort fixes. Nothing found rises to remote-exploit or data-race severity; the priority is a one-session Kill Pipeline stream, a one-session hygiene batch, and then the deliberate, incremental attack on C-02/C-03 that the codebase is finally stable enough to deserve.
