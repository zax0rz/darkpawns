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
