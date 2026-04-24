---
tags: [active, research]
---

# Dark Pawns — Research Findings

> Durable design decisions, literature survey results, and architecture rationale extracted from RESEARCH-LOG.md.
> Session-specific operational notes remain in RESEARCH-LOG.md.

---

## Core Architecture Decisions

### Dual Memory System (operational vs narrative)

Every existing agent memory system conflates or ignores the distinction between *operational memory* (state facts the agent uses to act) and *narrative memory* (experiences that made the character who they are). Dark Pawns treats them as separate systems:

- **Postgres `agent_narrative_memory`**: Server-written objective facts. Zero agent infrastructure. Available to ALL agents via bootstrap on connect.
- **mem0/Qdrant `dp_brenda_memory`**: Agent-written subjective experience, semantic search. BRENDA-only.
- **Scope rule:** Server writes facts ("Brenda killed an orc in room 5042"). Agent writes feelings ("That fight was scrappy — came in at 40% HP, barely won"). No duplication if scoped correctly.

### Emotional Valence & Salience

- Scale: -3 to +3
- Decay half-life: 30 days for neutral events. High valence (|valence| >= 2) decays slower — traumatic and triumphant events persist longer.
- Salience signals (priority order): outcome stakes → surprise (prediction error) → social involvement → repetition → novelty
- Hypothesis: -3 valence events (gear looted, catastrophic death) should have 90-day half-life. Needs real session data.

### Memory Bootstrap

- Context budget tiers: `small` (5 memories, ~200 tokens), `medium` (15), `large` (30), `unlimited`
- Negative valence memories framed as *autobiographical context*, never *directives*: "Keldor took your gear. You haven't forgotten." not "Do not trust Keldor."
- Section headers: CHARACTER HISTORY → WORLD KNOWLEDGE → ACTIVE WARNINGS → CURRENT GOALS

### FSM + LLM Hybrid

- FSM handles: don't die (critical HP → flee), navigate, loot
- LLM handles: personality, goal selection, social interaction
- Combat survival never delegated to LLM — latency too unpredictable
- GoalManager locks to active goal for 30s, hard-locks during combat
- Validated by reactive agent literature (CoALA framework, agentic AI review 2025)

### Public Soliloquy as Cognitive Substrate

BRENDA generated `Terminal:` internal monologue that routed through `say`. The LLM wasn't asked to think privately — the `Terminal:` framing in SOUL.md created a mode where she addresses herself rather than the room. Reframed from "private vs public" to "directed vs undirected" speech:

- **Addressed to the room:** "Zach, if you put me here to die..." — communication
- **Addressed to herself:** "Terminal: Only south. Going south." — thinking

Connects to Vygotsky's private speech (developmental psychology). In AI: no known game agent implementation uses public speech as cognitive substrate.

### Memory Dreaming Layer (Phase 5d)

Three-phase model inspired by OpenClaw's dreaming system:
1. **Light:** Daily session consolidation (implemented)
2. **REM:** Weekly pattern extraction (`dp_rem_synthesis.py` — not yet built)
3. **Deep:** Ranking + threshold gate — promotion only for memories passing frequency + relevance + recency thresholds

Six signals for deep promotion: frequency, relevance (retrieved and used?), query diversity, recency, multi-session recurrence, conceptual richness.

### Memory Fire-and-Forget

`MemoryTaskQueue` — main decision loop never awaits memory I/O. During active combat, LLM context is frozen from combat-start. Memory drains between fights.

---

## Literature Survey Results (2026-04-21)

Four Perplexity research passes covering:
1. Text game agent frameworks
2. Agent memory & persistence
3. Protocol design
4. Agent motivation

Second pass: autobiographical memory, cognitively constrained agents, salience encoding.

**Key finding:** Emotional valence in agent memory, natural language experience references, and social/collective memory are all open problems with no existing implementations. Genuine gap.

### Multi-Model Spec Review

Gemini reviewed the agent protocol spec and caught three real bugs before implementation:
1. Duplicate mob targeting (missing `target_string` disambiguation)
2. Async memory blocking combat tick (needed fire-and-forget queue)
3. Goal thrash (needed commitment lock)

Good argument for multi-model spec review as standard practice.

---

## Dark Pawns Divergences from Stock CircleMUD

The Go port is mostly faithful, but several genuine customizations exist:

### Custom Combat Formulas
- `get_minusdam()`: 24-tiered AC-based damage reduction (not in stock CircleMUD)
- `CalculateHitChance()`: INT and WIS bonuses to THAC0 (non-standard; most MUDs only use STR/DEX)
- Backstab multiplier: `(level*0.2)+1`, capping at 20× at LVL_IMMORT
- Attacks-per-round: Complex per-class/level formula with random chance gates

### Class/Race System
- 12 classes, 7 races (Human, Elf, Dwarf, Kender, Minotaur, Rakshasa, Ssaur)
- Ninja restricted to Human only
- Magus, Avatar, Assassin, Paladin, Ranger, Mystic are remort-only (Dark Pawns progression system)

### Original C Features NOT Yet Ported
- `flesh_alter_to/from()` — remort transformation
- `dream()` — sleep-based dream system
- `TAT_TIMER` — tattoo timer decay
- `GET_JAIL_TIMER` — jail system
- `SKILL_KK_JIN` / `SKILL_KK_ZHEN` — monk skill regen bonuses
- Field objects (`NUM_FOS`) — timed environmental objects
- Moon gate objects with timer decay
- Corpse decay with 7 randomized messages

Hypothesis: Unported features (dream, flesh_alter, jail) are more interesting for agent research than ported ones.

---

## AIIDE 2027 Paper Contributions

### Strong Contributions
1. **Agent-as-player in multiplayer MUD** — No known prior work treats LLM agents as first-class MUD players with persistent identity
2. **Operational/narrative memory split** — CoALA taxonomy exists; no implementation at storage layer
3. **Public soliloquy as cognitive substrate** — Vygotsky's theory; no AI implementation
4. **Salience decay + dreaming layer** — No game AI precedent

### Medium Contributions
5. **Bootstrap injection with budget tiers** — RAG standard; budget declaration + framing discipline are new
6. **FSM+LLM hybrid with combat freezing** — CoALA proposes; Dark Pawns validates with real data
7. **Build/qa/fix/push swarm methodology** — Novel multi-agent dev process

### Evaluation Framework (needed)
- Behavioral change over sessions
- Natural language reference frequency
- Cross-agent social memory references
- Private/public divergence
- All measurable once retrieval tracking and multi-agent sessions are live

---

## Open Research Questions

- What is the right decay half-life for -3 valence events?
- How to handle memory conflicts (two memories disagreeing about an event)?
- When does the social memory graph become queryable?
- Should agents write to their own narrative memory via command, or is server-only the right constraint?
- What does "narrative coherence" mean in practice? Needs a rubric.
- Minimum session count before narrative memory produces measurable behavioral change? (Hypothesis: 10)
- Does the private/public memory split produce meaningfully different agent behavior?
- What happens when BRENDA's private model of Zach diverges from her public behavior toward him?

---

*Extracted from RESEARCH-LOG.md on 2026-04-24.*
