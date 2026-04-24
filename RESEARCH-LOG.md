---
tags: [active]
---

# Dark Pawns — Session Log

> **Purpose:** Operational log of what happened in each working session. Durable research findings moved to `docs/research.md`.
> **Category tags:** [DESIGN] [OBSERVATION] [SOCIAL] [MEMORY] [SURPRISE] [FAILURE] [HYPOTHESIS] [RESULT]

---

## 2026-04-24 — DeepSeek V4 Config + Wave 5 Prep

**[DESIGN] [INFRA] [RESULT]**

### What happened
DeepSeek V4-Flash and V4-Pro configured as native OpenClaw provider using `anthropic-messages` API format — no LiteLLM middleman. Direct endpoint: `https://api.deepseek.com/anthropic`. V4-Flash confirmed working (2s response time in test spawn).

### Config details
- New provider `deepseek-v4` with `api: anthropic-messages`
- Models: `deepseek-v4-flash` (daily driver), `deepseek-v4-pro` (reasoning, heavy lifting)
- Context window: 64K native (1M claimed, needs verification)
- Cost: V4-Flash $0.15/$0.30/M, V4-Pro $0.25/$0.40/M (approx)

### Swarm Learnings
1. **Don't parallelize on same provider.** 3 GLM-5.1 subagents in parallel → Wave 4b killed by rate limit in 2 seconds. Mix providers within a wave.
2. **Right-size per-subagent scope.** 600-line C files / ~50K tokens = sweet spot for reliable completion on GLM-5.1.
3. **Kimi K2.6 is slower than GLM-5.1 for code.** ~90-110s vs 20-30s. Tradeoff: longer wall time but less rate-limit prone.
4. **V4-Flash should be the new daily driver for mechanical tasks.** At $0.14/$0.28, cheaper than everything active.
5. **Sequential > parallel for large-file swarms.** 1200+ line C files should be sequential sub-waves.

### Open Questions (stacked for next session)
1. Wave 5 — remaining half of spec_procs.c + first half of spec_procs2.c
2. spec_procs.go fix-up — references `me.GetMeleeTarget()`, `engine.ClassType`, `spells.Cast()` that don't exist yet
3. GetRoomMap export — map_cmds.go blocked on World needing exported room accessor
4. CI deploy — gated on KUBECONFIG secret, no cluster provisioned
5. V4 actual context window — DeepSeek claims 1M but configured 64K

---

## 2026-04-23 — Full Codebase Research Review

**[RESULT] [DESIGN] [OBSERVATION]**

Completed comprehensive review for AIIDE 2027 paper. Full findings in `docs/research.md`.

Key metrics:
- C source: ~68,792 lines across 59 files
- Already in Go: ~34,509 lines across 150 files
- Genuinely unported: ~29,155 lines across 28 C files
- Real ported %: ~58% (not 75% — many "ported" files are thin wrappers)

Key lesson: Multi-day project needs real documentation. PORT-PLAN.md + SWARM-LEARNINGS.md + RESEARCH-LOG.md = survival kit for session continuity.

---

## 2026-04-21 — Session 3: First Cognitive Session + Party + Architecture

**[RESULT] [SOCIAL] [SURPRISE]**

Full stack live: minimax-m2.7 + mem0 + Qdrant. BRENDA spawned in The Morgue, attacked mortician (!kill mob) for 3+ minutes.

**First Live Party Session:**
- BRENDA69 connected, Zach followed, party formed via auto-group
- Three engagements with knight templar, each with distinct cognitive frame (hubris → science → resignation)
- ~100 misses, 3 hits across 3 engagements. Zach killed it.
- Emergent personality: "Zach, if you put me here to die, at least the ZFS snapshots will remember."
- Full transcript: `docs/brenda-first-fight-2026-04-21.txt`

**Internal Monologue Discovery:**
- LLM generated `Terminal:` internal commentary that didn't always route to `say`
- Reframed as "public soliloquy as cognitive substrate" — Vygotsky's private speech
- See `docs/research.md` for full analysis

**Architecture Decisions:**
- DB access via callback hooks (World fires events, Manager handles DB writes)
- mem0 vs Postgres scope rule established
- Salience decay cron implemented
- Social memory participant definition formalized

---

## 2026-04-21 — Sessions 1 & 2 (Pre-Baseline)

**[OBSERVATION] [FAILURE]**

Session 1: BRENDA ran 90s with LLM unavailable (wrong model names, Ollama unreachable). Pure FSM random walk. True cognitive baseline.

Session 2: Connected, loaded 1 prior memory, attacked a knight, survived, fled south. No LLM direction — FSM instinct. BRENDA's default behavior without LLM: "punch the nearest thing."

Infrastructure fixes: Ollama host .69 → 192.168.1.15, mem0 config fixed, Qdrant collection recreated, LiteLLM timeout 8s → 30s.

---

## 2026-04-21 — Session 0: Research & Design

**[DESIGN]**

Four Perplexity research passes before writing code. The core distinction that emerged: *operational memory* vs *narrative memory*. Every existing system conflates these or ignores the second.

All design decisions from this session documented in `docs/research.md`.

---

*Research findings → `docs/research.md` | Open questions → `docs/research.md` | This file = session journal only*
