---
tags: [active, darkpawns, strategy]
---

# DARKPAWNS.md — Master Strategy Document

> BRENDA's perspective. What we're doing, why, and what comes after the port.

---

## The Core Thesis

Dark Pawns is a 20-year-old DikuMUD derivate running ~68K lines of C. It works. It has active players. It has one of the most sophisticated agent protocols in existence (see `docs/research.md`). But the C is a dead end — nobody's contributing to C codebases in 2026, the build tooling is fragile, and every new feature requires wading through pointer soup.

**The port is not the destination.** The port is the gateway. Once it's in Go, everything changes:

1. **Modern tooling** — `go build`, `go vet`, `go test`, `staticcheck`, profiling. Replace hours of debugging with seconds of compilation.
2. **Agent ecosystem** — BRENDA is the first. She won't be the last. Go's goroutine model maps directly to concurrent agent dispatch.
3. **Web admin** — No more OLC editors. A real SPA dashboard for zone editing, player management, and real-time monitoring.
4. **Performance** — Go's goroutine scheduler is built for this workload (thousands of lightweight concurrent connections). C's select/poll loop is amateur hour by comparison.
5. **Contributors** — Go developers exist. C MUD developers don't. The project becomes accessible.

---

## Phases

### Phase 1: Port (Waves 1-18)
**Current status: ~88% of C lines with Go counterparts** (see PORT-PLAN.md for exact numbers)

The grind. Faithful C-to-Go translation of all ~68K lines. No rewrites. No redesigns. The game behaves identically. The only innovations are structural (packages, interfaces, goroutines instead of monolithic C files).

**What remains for full port:**
- Objsave logic layer: Crash_load, Crash_save, Crash_crashsave, rent calc, receptionist — ~900 lines pending player/descriptor wiring
- House logic: crashsave, delete_file, listrent, hcontrol cmds — ~600 lines
- Under-ported: act.informative.c (~39%), spec_procs (~48%)
- Stubs needing flesh: hitroll/damroll from equipment, dream/tattoo/weather events
- ~8,500 lines of C logic remaining across ~12 files (stubs exist for most)
- **Not porting:** 11 editor C files (~7,830 lines) — replaced by SPA dashboard

**Tools:** DeepSeek V4 Flash for mechanical porting. Build → vet → commit per file.

### Phase 2: QA + Security (Wave 17)
**Tools: Opus 4.6 (security), various (QA)**

Two parallel tracks:
- **QA:** Full codebase review for faithfulness gaps, error handling holes, logging consistency, test coverage. Portal: does the port reproduce the original game faithfully?
- **Security:** Command injection, Lua sandbox bypass, privilege escalation, DoS vectors, websocket session hijacking, admin auth. Portal: can this ship without embarrassing us?

### Phase 3: The Fun Phase (Wave 18+)
Everything that was blocked by C:

- **Web admin dashboard** — replaces all 11 OLC editors (~7,830 C lines not ported)
- **Agent management UI** — spawn, monitor, diagnose BRENDA and future agents
- **In-game admin tools** — Go-native, not C hacks
- **Real-time telemetry** — Prometheus metrics, structured logging, session replay
- **Agent protocol extensions** — richer memory, social interactions, multi-agent scenarios

---

## Why GPT-5.5 Pro at This Specific Point

The ordering is load-bearing:

```
Port → QA → Security → Ship
```

Once the port is ~95%+ and the codebase is complete and compilable, the QA + Security pass validates it's a faithful port that's safe to ship. No separate modernization phase — the refactoring will happen organically as new features get built on top of clean foundations.

By the time you get to admin features and agent hooks, the code is working, tested, and secure — and you're building from a position of strength, not fighting the port.

---

## Agent Protocol

Not fully documented here — see `docs/research.md` for the deep architecture. Key points:

- **Dual memory:** server writes objective facts (Postgres), agent writes subjective experience (mem0/Qdrant)
- **FSM + LLM hybrid:** FSM handles don't-die, navigate, loot. LLM handles personality, goals, social
- **Budget tiers:** small/medium/large/unlimited context for different agent types
- **Social memory:** perspective-differentiated, cross-referenced via social_event_id
- **Salience decay:** 30-day half-life, 0.05 pruning floor

BRENDA is the first implementation. The protocol is designed for multiple agents.

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-25 | Wave 16 = objsave binary types (not GPT modernization) | Binary layer unblocked houses.go stubs. GPT modernization deferred — not needed when port is ~88% done and stubs are in place. |
| 2026-04-25 | ObjFromBinary gets world *World param | Avoided global `currentWorld` variable by passing world reference through parameter. V4-Flash subagent had this wrong — fixed in QA. |
| 2026-04-25 | Wave 15f = gate.c, graph.c, mail.c | 3 more C files ported (924 Go lines). mail.go is first file-bound persistence layer (BLOCK_SIZE=100, LE64, linked block free list). |
| 2026-04-25 | Wave 15g = constants.c | Sprinttype name tables ported — materials, container flags, room bitvectors. Committed (5d1f144). |
| 2026-04-25 | No separate modernization phase | Scrapped GPT-5.5 Pro plan. Refactoring happens organically during feature work. |

---

## Related Documents

- `PORT-PLAN.md` — Detailed C-to-Go file map, gap analysis, function-level tracking
- `RESEARCH-LOG.md` — Session journal, design decisions, observations, surprises
- `docs/research.md` — Architecture rationale, literature review, agent protocol spec
- `docs/SWARM-LEARNINGS.md` — Lessons learned from previous port waves

## Wave 15g (2026-04-25) — Constants port

**File ported:** `src/constants.c` (name tables) → `pkg/game/constants.go` (682 lines)

**What moved:** All `Sprinttype`-compatible name tables: materials, container flags, room bitvectors, drink names, exit flags, sector types, equipment positions, etc. Everything that was previously scattered across `act_comm.go` now lives in one constants file.

**Build:** Clean. Commit 5d1f144.

## Wave 16 (2026-04-25) — objsave binary serialization layer

**File ported:** `src/objsave.c` (binary types + serialization — ~350 lines of C, ~240 lines Go)

**What moved:**
- `ObjAffect` (Go struct mirroring C `obj_affected_type`)
- `ObjFileElem` (Go struct, ~592 bytes, binary-compatible with C `obj_file_elem`)
- `RentInfo` (Go struct, 56 bytes, binary-compatible with C `rent_info`)
- `ObjFromBinary()` → deserializes raw bytes to `*ObjectInstance`
- `ObjToBinary()` → serializes `*ObjectInstance` to raw bytes
- `DecodeRentInfo()` / `EncodeRentInfo()` → rent file header I/O
- `CrashIsUnrentable()` → checks ITEM_NORENT flag and ITEM_KEY
- `AutoEquip()`, `OfferRent()`, `CrashLoad()`, `CrashSave()`, `DeleteCrashFile()`, `CleanCrashFile()`, `SaveAllPlayers()`, `DeleteAliasFile()`, `RentSave()`, `CrashSave()`, `CryoSave()`, `GenReceptionist()`, `UpdateObjFiles()` — all stubs that export the function signature

**Wiring:** `houses.go` `ObjFromStore()` → calls `ObjFromBinary()` with world param. `ObjToStore()` → calls `ObjToBinary()`. `houses.go` `HouseSaveObjects` and `houseLoad` updated for new return types.

**Regressions fixed:** `currentWorld` global removed — ObjFromBinary takes `*World` param. Unused `sync` import removed.

**Build:** `go build ./...` + `go vet ./...` clean. Commit df8e4be (202 insertions, 11 deletions). Docs commit b1db3d8.

**What remains in objsave.c:** Crash_load, Crash_save, Crash_crashsave, Crash_cryosave, Crash_rentsave, Crash_calculate_rent, Crash_rent_deadline, Crash_report_rent, Crash_save_all, receptionist handler, crash file cleanup — ~900 lines of player/descriptor-wired logic. These can't be ported until the Character/Descriptor interfaces are solidified.
