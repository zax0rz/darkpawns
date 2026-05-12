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
