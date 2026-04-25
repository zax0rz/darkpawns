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
**Current status: ~80-85% complete** (see PORT-PLAN.md for exact numbers)

The grind. Faithful C-to-Go translation of all ~68K lines. No rewrites. No redesigns. The game behaves identically. The only innovations are structural (packages, interfaces, goroutines instead of monolithic C files).

**What remains:**
- Remaining C files: clan.c, house.c, boards.c, whod.c, objsave.c, mobprog.c
- Under-ported areas: act.informative.c (~39%), hitroll/damroll from equipment, dream/weather stubs
- ~5,000-7,000 lines of Go to write

**Tools:** DeepSeek V4 Flash for mechanical porting. Build → vet → commit per file.

### Phase 2: Modernize (Wave 16)
**Tool: GPT-5.5 Pro** — just launched April 24, 2026

Once the port is complete, GPT-5.5 Pro reads the whole codebase and identifies modernization opportunities:
- Go 1.24+ idioms (range over func, clear(), slog improvements)
- Error wrapping patterns (proper `%w` vs `%v` usage)
- Context propagation (where's it missing?)
- Goroutine hygiene (leaks? proper lifecycle?)
- Package boundaries (circular deps? god packages?)
- Dead code / unnecessary indirection

**Zero behavioral change.** This is not a rewrite. It's a polish pass. Everything that compiles and passes tests stays the same game.

**Why GPT-5.5 Pro specifically:** Terminal-Bench 82.7%, Expert-SWE 73.1%. It's the first model that demonstrably holds context across large codebases and makes genuinely good structural suggestions. The "conceptual clarity" improvement is exactly what this job needs.

### Phase 3: QA + Ship (Wave 17)
**Tools: Opus 4.6 (security), various (QA)**

Two parallel tracks:
- **QA:** Full codebase review for faithfulness gaps, error handling holes, logging consistency, test coverage. Portal: does the port reproduce the original game faithfully?
- **Security:** Command injection, Lua sandbox bypass, privilege escalation, DoS vectors, websocket session hijacking, admin auth. Portal: can this ship without embarrassing us?

### Phase 4: The Fun Phase (Wave 18+)
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
Port → Modernize (GPT-5.5 Pro) → QA → Security → Ship
```

GPT-5.5 Pro goes **after** the port but **before** QA and Security because:
1. It can't do its best work on a half-ported codebase — it needs the complete picture
2. If it finds modernization opportunities that need structural changes, those should happen *before* QA validates behavior
3. Security review against well-structured Go is more productive than against awkward Go

This is also a mental health play. The port is the grind. Modernization with GPT-5.5 Pro is the "now it's *good*" phase. QA + Security is the "now it's *safe*" phase. By the time you get to admin features and agent hooks, the code is clean, tested, and secure — and you're building from a position of strength, not fighting the port.

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
| 2026-04-25 | Wave 16 = GPT-5.5 Pro modernization | Best tool for post-port code review. Coming after port complete, before QA/security. |
| 2026-04-25 | Wave 17 split into QA + Security | Different skills, different tools, parallelizable. QA = faithfulness, Security = injection/access. |
| 2026-04-25 | Wave 18 = Admin + Agent features | Lock admin features behind a clean foundation. Don't build dashboard on top of bad code. |

---

## Related Documents

- `PORT-PLAN.md` — Detailed C-to-Go file map, gap analysis, function-level tracking
- `RESEARCH-LOG.md` — Session journal, design decisions, observations, surprises
- `docs/research.md` — Architecture rationale, literature review, agent protocol spec
- `docs/SWARM-LEARNINGS.md` — Lessons learned from previous port waves
