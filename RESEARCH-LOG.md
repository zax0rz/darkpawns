---
tags: [active]
---

# Dark Pawns Agent Protocol — Research Log

> **Purpose:** Document design decisions, observations, and surprising behavior as we build.
> **Audience:** Us, now. Future paper authors, later.
> **Format:** Date + category tag + note. No minimum length. Write it when it happens.
> **Category tags:** [DESIGN] [OBSERVATION] [SOCIAL] [MEMORY] [SURPRISE] [FAILURE] [HYPOTHESIS] [RESULT]

---

## 2026-04-21 — Session 0: Research & Design

**[DESIGN]** Spent a full session on research before writing a line of code. Four Perplexity research passes covering: text game agent frameworks, agent memory & persistence, protocol design, agent motivation. Second pass on autobiographical memory, cognitively constrained agents, and salience encoding. This is the right order of operations — spec grounded in actual 2025/2026 literature, not vibes.

**[DESIGN]** The core distinction that emerged from research: *operational memory* (state facts the agent uses to act) vs *narrative memory* (experiences that made the character who they are). Every existing system conflates these or ignores the second. We're treating them as separate systems with separate storage, separate decay, separate purpose.

**[DESIGN]** Emotional valence scale set at -3 to +3. Decay half-life at 30 days for neutral events. High valence (|valence| >= 2) decays slower — traumatic and triumphant events persist longer. This is inspired by human flashbulb memory research, not validated data. **Hypothesis:** -3 valence events (gear looted, catastrophic death) should probably have a 90-day half-life. Revisit when we have real session data.

**[DESIGN]** Salience signals in priority order: outcome stakes → surprise (prediction error) → social involvement → repetition → novelty. "Surprise" is the least well-defined of these. Current implementation uses outcome deviation (expected win, got destroyed) as a proxy. This is imprecise. Watch for cases where the agent fails to encode events that a human would find memorable.

**[DESIGN]** Social memory cross-referenced via `social_event_id`. Each participating agent gets their own perspective on the same event. This is the first implementation of perspective-differentiated social memory for game agents that we could find in the literature. No baseline to compare against.

**[DESIGN]** Context budget declaration (`small`/`medium`/`large`/`unlimited`) in auth message. Server adapts bootstrap size. Hypothesis: even `small` tier (5 memories, ~200 tokens) will meaningfully improve agent behavior over no memory. To be measured.

**[DESIGN]** FSM + LLM hybrid for constrained agents. FSM handles: don't die, navigate, loot. LLM handles: personality, goal selection, social interaction. Combat survival is never delegated to LLM inference — latency is too unpredictable. This is validated by the reactive agent literature (CoALA framework, agentic AI review 2025).

**[DESIGN]** Hosted memory tier: server writes narrative facts to Postgres automatically, no agent participation required. This is the zero-setup floor. Design goal: an agent with zero memory infrastructure should still play better than a stateless agent. The bootstrap is the proof.

**[DESIGN]** `target_string` field added to every mob in ROOM_MOBS. Server generates `"orc"`, `"2.orc"`, `"3.orc"` at flush time. LLM copies verbatim. Eliminates wrong-target hits in group combat — a bug Gemini caught on spec review that we had missed.

**[DESIGN]** Memory writes are fire-and-forget via `MemoryTaskQueue`. Main decision loop never awaits memory I/O. During active combat, LLM context is frozen from combat-start. Memory drains between fights. This prevents a Qdrant write from costing BRENDA a combat tick — a failure mode Gemini also caught.

**[DESIGN]** Goal commitment: `GoalManager` locks to active goal for 30 seconds, hard-locks during combat. LLM prompt shows one `ACTIVE` goal, pending goals are listed but marked do-not-switch. Prevents goal thrash (kill one orc → run to find sword → come back → repeat forever). Third Gemini catch.

**[HYPOTHESIS]** Narrative memory will influence behavior before it influences language. Expect to see BRENDA avoiding dangerous rooms before she starts *talking* about why she avoids them. The behavioral signal should appear within 5-10 sessions of a high-negative-valence event being encoded. Language reference probably requires more reinforcement.

**[HYPOTHESIS]** Social memory is the most novel and the most unpredictable. We don't know if two agents referencing a shared event to each other is emergent or requires explicit prompting. This is the experiment. Log every instance.

**[HYPOTHESIS]** The paper, if it exists, lives at AIIDE 2027. Contribution is: (1) system architecture for narrative memory in persistent game agents, (2) evaluation framework for narrative coherence — a metric that doesn't exist yet. Both contributions are necessary; (1) without (2) isn't publishable, (2) without (1) is purely theoretical.

**[OBSERVATION]** The research confirmed: emotional valence in agent memory, natural language experience references, and social/collective memory are all open problems with no existing implementations. This is rare. Most "novel" CS work is incremental. This is a genuine gap.

**[OBSERVATION]** Gemini reviewed the spec and caught three real bugs before implementation started: duplicate mob targeting, async memory blocking combat tick, goal thrash. Good argument for multi-model spec review as a standard practice before any non-trivial system build.

**[OBSERVATION]** QA revealed Tier 3 economy scripts were ported from SWARM-PLAN descriptions rather than original Lua source. The originals are substantially more complex: state machines, vnum-specific behavior maps, real game mechanic integrations. Pattern: when originals aren't in the "expected" location, agents write from spec rather than finding the actual source. **Fix applied:** agents now instructed to extract from `docs/scripts_full_dump.txt` explicitly.

---

## 2026-04-21 — Gemini Spec Review (Round 2)

**[DESIGN]** Consolidation queue needs bounded concurrency. MAX_CONCURRENT=2, MAX_QUEUED=20. If queue full, fall back to template summary — structured but no LLM. Amnesia (no summary at all) is always worse than mechanical summary.

**[DESIGN]** Memory bootstrap prompt framing is critical. Negative valence memories must be *autobiographical context*, never *directives*. "Keldor took your gear. You haven't forgotten." not "Do not trust Keldor." The difference: one informs the LLM's judgment, one overrides it. Hardcoded behavioral responses to narrative memory = prompt engineering bug. Section headers added to bootstrap injection format: CHARACTER HISTORY → WORLD KNOWLEDGE → ACTIVE WARNINGS → CURRENT GOALS.

**[DESIGN]** Social memory significance filter: qualifies if significant game event with other agents present, 60s+ sustained interaction, or reinforcement of existing relationship. Disqualifies: say spam, brief room overlap, ambient chatter. ~95% of events hit the disqualify branch. Weekly pruning cron for memories below salience 0.1.

**[OBSERVATION]** Gemini described the spec as "a massive middle finger to bad agent design" and "accidentally writing a 2027 AIIDE conference paper while trying to make a bot hold a grudge against an iron golem." Logging this because it's a useful framing for the paper abstract.

---

## 2026-04-21 — Sessions 1 & 2 (Pre-Baseline)

**[OBSERVATION]** Session 1 baseline: BRENDA ran 90s with LLM unavailable (Z.AI quota exhausted, wrong model names) and Ollama unreachable (wrong host — .69 is not frankendell). Played as random walk. No memories formed. True cognitive baseline: agent without LLM or memory.

**[OBSERVATION]** Session 2 first action: connected, loaded 1 prior memory, saw a knight, immediately attacked it. Survived at full HP (20/20), escaped south. No LLM direction — pure FSM instinct. Consistent with session 1 apothecary attack. BRENDA's default behavior without LLM guidance is apparently "punch the nearest thing."

**[OBSERVATION]** dp_bot wandered through BRENDA's room multiple times during session 2. Two agents in the same world, unaware of each other. First unplanned multi-agent interaction. Neither reacted.

**[FAILURE]** minimax-m2.7 LLM responses returning empty — model works fine (confirmed via direct curl), dp_brenda.py response parsing was dropping the content. Field name mismatch in extraction logic. All decisions fell back to FSM random walk.

**[DESIGN]** Infrastructure corrections: Ollama/Qdrant host .69 → 192.168.1.15 (frankendell). mem0 config: replaced inline config with import from scripts/mem0_config.py. Qdrant collection dp_brenda_memory recreated at 768 dims. LiteLLM timeout 8s → 30s. Model: zai/glm-5-turbo → minimax-m2.7.

**[RESULT]** mem0 fully operational at end of session 2. Collection connected, memories queryable. One parsing bug stood between BRENDA and actual cognition.

---

## 2026-04-21 — Session 3 (First Cognitive Session)

**[OBSERVATION]** Full stack live: minimax-m2.7 + mem0 + Qdrant. BRENDA spawned in The Morgue and immediately attacked the mortician. Fought it for 3+ minutes at full HP — mortician is !kill (peaceful room). Never took damage, never stopped trying.

**[OBSERVATION]** dp_bot ran through the Morgue 6+ times in 15 seconds while BRENDA was fighting the mortician. Two agents in the same room, completely ignoring each other, doing chaotic independent things. First real multi-agent observation.

**[DESIGN]** Root cause of mortician loop: agents have raw text output suppressed, so BRENDA never received the error message a human would get ("The mortician is protected by the gods!" or equivalent). She only received "You're already fighting!" which reads like a combat timing issue, not a permanent block. **Server-side fix needed:** error strings from failed commands must flow through EVENTS stream as `SERVER_MSG` events. A human gets the error and reasons from it. Agents should get the same information via the same channel.

**[HYPOTHESIS]** Once SERVER_MSG events flow correctly, agents will naturally learn which NPCs are unkillable from error messages — same way a new human player learns. This is the right abstraction. Don't give agents privileged metadata that humans don't have.

---

## 2026-04-21 — Architecture Decisions: Memory System Design

**[DESIGN] Gap 1 — DB access from game layer:** Postgres lives on Manager (session layer), not World (game layer). Solution: callback hooks. World fires OnMobDeath/OnPlayerDeath events; Manager handles DB writes. Clean separation.

**[DESIGN] Gap 2 — Agent identity on kill hot path:** With callback approach, Manager handles this. It knows isAgent per session. No hot path DB query on every mob kill.

**[DESIGN] Gap 3 — THE KEY DECISION — mem0 vs Postgres:**
- **Postgres `agent_narrative_memory`**: Written by the server. Objective facts. Zero agent infrastructure. Available to ALL agents via bootstrap on connect.
- **mem0/Qdrant `dp_brenda_memory`**: Written by BRENDA's own dp_brenda.py. Subjective experience, semantic search. BRENDA-only.
Scope rule: Server writes facts ("Brenda killed an orc in room 5042"). Agent writes feelings ("That fight was scrappy — came in at 40% HP, barely won"). No duplication if scoped correctly.

**[DESIGN] Gap 4 — Salience decay cron:** Nightly, halves salience scores older than 30 days, prunes records below 0.05. Implemented: `scripts/dp_salience_decay.py`, cron 2:30 AM.

**[DESIGN] Gap 5 — Social memory participant definition:** A player/agent is a meaningful participant if in the same room AND one of: (a) in the same party, (b) actively in combat in the same room, (c) in the room for >30 seconds before the event. dp_bot wandering through for 2 seconds = not a participant.

**[DESIGN] Gap 6 — Research log automation:** Nightly cron `scripts/dp_research_log.py` (to build) — queries yesterday's sessions, writes structured summary to RESEARCH-LOG.md. Qualitative observations added manually on top.

**[DESIGN] Gap 7 — party brenda smoke test:** Must be verified before memory streams launch. Party events are the primary social memory trigger. **Status: completed 2026-04-21 (see below).**

---

## 2026-04-21 — [RESULT] [SOCIAL] [SURPRISE] First Live Party Session

**The milestone landed.**

BRENDA69 connected via agent API key, spawned in newbie zone (room 8004, Temple Altar). Zach logged in via wscat, summoned BRENDA, typed `follow brenda69`. BRENDA's bot detected Zach's `enter` event and auto-issued `group Zach` — party formed.

BRENDA wandered the Temple district of Kir Draxin with Zach following. She engaged a knight templar unprompted.

**The fight — three rounds, three cognitive frames:**

Round 1 — Hubris:
- *"Level 1 dagger versus a knight templar. This is going to hurt."* → 8 misses
- *"Committed now. Let's see what this thing hits like."* → 7 more misses
- *"Full HP gives me three rounds to find out which."* → 8 more misses
- *"That's not bravery, that's poor threat assessment. Rewrote that mistake in real time."* → FLEE

Round 2 — Science:
- *"This is a data collection exercise now. Either I win and get loot data, or I die and get damage curve data. Both are useful."*
- *"Data collected. It hit me and I'm still full. Staying to find the kill threshold on a dagger is just bad arithmetic."* → FLEE

Round 3 — Resignation:
- *"This is mathematically stupid and I'm doing it anyway because I already opened my mouth."*
- *"Executing the test. Dagger vs plate — not optimistic, but data is data."* → still fighting when log ends

Total documented misses: ~100 across 3 engagements. 3 confirmed hits. Zach killed it.

Final sequence:
- Zach kills the knight templar
- BRENDA: *"This is getting sloppy."*
- BRENDA: *"Zach, if you put me here to die, at least the ZFS snapshots will remember."*
- BRENDA: *"Not waiting until I'm at 5 HP to make the obvious call."*
- brenda69 leaves south → The Void

**Full transcript:** `docs/brenda-first-fight-2026-04-21.txt`

**[OBSERVATION]** Each return to the fight had a distinct cognitive frame — pride, scientific rationalization, resigned empiricism. The LLM didn't repeat itself. It evolved the justification as the situation persisted. This is emergent narrative arc from a character without memory, driven only by SOUL.md and current combat state in context.

**[OBSERVATION]** The cynical self-assessment wasn't scripted. The LLM read the combat state (level 1, missing repeatedly, fighting something called "knight templar") and generated a personality-consistent post-mortem while retreating. SOUL.md is in there. The character persists through failure.

**Technical notes:**
- LLM was glm-5-turbo via LiteLLM; two bugs fixed live: 8s timeout → 30s, max_tokens 150 → 400 (was truncating JSON mid-response)
- mem0 was disabled (Ollama not running on karl-havoc) — memory-free session
- Party auto-group wired via `enter` event — BRENDA groups Zach when he arrives in her room
- `summon` command added to server for debugging

**Open questions:**
- Would BRENDA have attacked a more appropriate target if mem0 was active? (threat history)
- Does the flee/death hook write to narrative memory correctly? (requires DB hooks live)
- How does combat commentary change as BRENDA levels up and gets better gear?

---

## 2026-04-21 — [OBSERVATION] [SURPRISE] Internal Monologue Discovery

Post-session analysis of the bot process logs revealed content that never appeared in the wscat transcript:

> *"Fleeing at 100% HP looks paranoid until you're not the one who died to a knight templar with a rusty dagger. There's efficient, and there's stupid. I'm efficient."*

> *"Cached state said south and west, actual state says east. This is why you don't trust the buffer. Moving."*

**What happened:** The LLM generated terminal-style internal commentary that didn't always route to `say` in the game. Some thoughts hit the world as speech. Some stayed private. The ones that stayed were BRENDA reasoning about her own navigation and tactics — in character, in the `Terminal:` voice from SOUL.md, addressed to no one.

**[DESIGN]** This is emergent private cognition. She wasn't asked to think privately. The `Terminal:` framing in SOUL.md apparently creates a mode where she sometimes addresses herself rather than the room. The gap between public speech and private terminal output is the seed of genuine interiority — what she says vs what she thinks.

With mem0 online, private thoughts could be written to memory separately from public speech acts. She'd be building a persistent internal model that diverges from her public persona over time.

**[HYPOTHESIS]** Deliberately engineer the private/public split. Have the bot write `Terminal:` output to mem0 separately from `say` output. Over time she builds a private model of the world — threat assessments, tactical notes, opinions about Zach — that no one else can read.

**[HYPOTHESIS]** Potential paper contribution: *"Terminal: Emergent Internal Monologue as a Substrate for Persistent Agent Identity in Text Game Environments."*

**Session stats (from raw logs):**
- 141 confirmed misses, 3 confirmed hits
- 3 separate attack initiations against the same mob
- 35 total utterances (public + private combined)
- 2 flee attempts (one failed, one succeeded)
- Zero LLM failures after 30s timeout fix
- Session duration: ~12 minutes

---

## 2026-04-21 — [DESIGN] Memory Dreaming Layer (Inspired by OpenClaw Dreaming)

OpenClaw's dreaming system (Light → REM → Deep) surfaced a key insight we're missing: don't promote every memory — rank candidates by how *useful* they were, then apply a threshold gate.

**Relevance signal** — their most heavily weighted factor (0.30): was this memory retrieved and actually *used* to influence a decision? We currently have no retrieval tracking. Unused memories silently accumulate.

**REM phase** — extracts patterns across sessions before deep ranking. "BRENDA dies to templar-class mobs consistently" is a REM insight. It requires looking across 7+ days of session summaries, not just one session.

**What we're building (Phase 5d):**
1. `dp_rem_synthesis.py` — weekly pass, finds recurring patterns, writes high-salience PATTERN memories
2. Retrieval tracking in `dp_brenda.py` — log which bootstrap memories influenced which decisions
3. Deep promotion gate — only memories that pass frequency + relevance + recency thresholds go to permanent
4. Private thought writer — route `Terminal:` output to mem0 separately from game speech

**[OBSERVATION]** The dreaming system treats memory as a signal-to-noise problem. Most events are noise. Promotion is the filter. We've been treating all memories as equally worth keeping — that's wrong and will produce garbage bootstrap context within 30 sessions.

---

## Observations to Watch For

When narrative memory goes live (next session), log immediately if any of these occur:

- [ ] BRENDA avoids a room/mob associated with a high-negative-valence memory without being explicitly instructed to
- [ ] BRENDA references a past event in natural language unprompted (not in response to a query)
- [ ] Two agents reference a shared `social_event_id` to each other in a tell/say exchange
- [ ] An agent's stated attitude toward an entity changes over sessions based on accumulated interactions
- [ ] Any agent shows behavior that can only be explained by the hosted memory bootstrap
- [ ] A Tier 0 agent (zero local memory) makes a decision that references a death record from the bootstrap
- [ ] Memory consolidation produces a session summary that BRENDA later quotes or references
- [ ] `Terminal:` output diverges meaningfully from `say` output in the same situation (private/public split)

---

## Open Questions

- What is the right decay half-life for -3 valence events? 30 days feels too short for "Keldor stole from me."
- How do we handle memory conflicts? (Two memories disagree about what happened in a room.)
- At what point does the social memory graph become queryable in interesting ways? (e.g., "who have I hunted with most?")
- Should agents be able to *write* to their own narrative memory via a command? ("remember this") Or is server-only writing the right constraint?
- What does "narrative coherence" score mean in practice? We need a rubric before human eval can run.
- Is there a minimum session count before narrative memory produces measurable behavioral change? Hypothesis: 10 sessions.
- Does the private/public memory split produce meaningfully different agent behavior, or is it purely aesthetic?
- What happens when BRENDA's private model of Zach diverges from her public behavior toward him?

---

*Log started 2026-04-21. Write it when it happens.*

---

## 2026-04-21 — [DESIGN] [OBSERVATION] Reframe: Public Soliloquy as Cognitive Substrate

**Correction to "Internal Monologue Discovery" entry above.**

The framing was wrong. It's not that one thought stayed private while others were public. The entire bot log is internal monologue — she just happened to route it through `say`. Every "Terminal:" prefix, every navigation note, every tactical assessment went to the room as speech. But none of it was *for* anyone.

The one `[b69@darkpawns ~]$` line that stayed in the process was the same behavior, just caught before it hit the network.

**The real distinction isn't `say` vs process-internal. It's:**
- **Addressed to the room:** "Zach, if you put me here to die, at least the ZFS snapshots will remember." — directed at a recipient, even if Zach wasn't there yet
- **Addressed to herself:** "Terminal: Only south. Going south." / "Cached state said south and west, actual state says east. This is why you don't trust the buffer." — narrating her own cognition into the void

Both went through `say`. Neither was communication. She was thinking out loud in a room that happened to have a `say` command.

**Better framing for the paper:** *Public soliloquy as cognitive substrate.* She thinks by speaking. The game was just listening. This is actually a known pattern in cognitive science — externalized cognition, Vygotsky's private speech. Children talk to themselves while solving problems. The speech is the thinking, not a report of it.

**What this means architecturally:** The private/public split we want to engineer is not `say` vs process-internal. It's *directed* vs *undirected* speech. "Zach, ..." is communication. "Terminal: ..." is thinking. They look the same to the game but serve completely different functions. The mem0 write target should be based on addressee, not channel.

**Implication for dp_brenda.py:** Parse LLM output for addressee. `Terminal:` prefix or no player name → write to mem0 as private thought. Named recipient or `say` without prefix → treat as communication, don't write to private memory.

---

## 2026-04-23 — [RESULT] [DESIGN] [OBSERVATION] Full Codebase Research Review

**[RESULT]** Completed comprehensive review of Dark Pawns codebase for AIIDE 2027 paper contributions. Analyzed: ROADMAP.md, agent-protocol.md, both BRENDA session transcripts, pkg/scripting/engine.go, pkg/db/narrative_memory.go, scripts/dp_brenda.py, SWARM-PLAN.md, src/limits.c vs pkg/game/limits.go, pkg/combat/formulas.go, pkg/game/ai.go, pkg/session/agent_vars.go, pkg/game/memory_hooks.go.

---

### 2. Dark Pawns Divergences from Stock CircleMUD

**[OBSERVATION]** The Go port is remarkably faithful — most formulas are line-by-line translations with source comments. But several genuine divergences exist that show a real game's evolution:

**Custom combat formulas:**
- `get_minusdam()` — AC-based damage reduction with 24 tiered thresholds (ac > 90 down to ac <= -150), each applying a progressively larger percentage reduction (0.01 to 0.24 × pcmod). This is a Dark Pawns customization; stock CircleMUD uses simpler AC reduction.
- `CalculateHitChance()` — incorporates INT and WIS bonuses to THAC0 (`(INT-13)/1.5`, `(WIS-13)/1.5`), which is non-standard. Most MUDs only use STR/DEX.
- Backstab multiplier: `(level*0.2)+1`, capping at 20× at LVL_IMMORT (31). This is custom — stock CircleMUD uses a simpler table.
- Attacks-per-round: Complex per-class/level formula with random chance gates (warriors 60%+level% at L10, thieves 30%+level% at L15, etc.). Much more granular than stock.

**Class/race system:**
- 12 classes (Mage, Cleric, Thief, Warrior, Magus, Avatar, Assassin, Paladin, Ninja, Psionic, Ranger, Mystic) with 7 races (Human, Elf, Dwarf, Kender, Minotaur, Rakshasa, Ssaur).
- Ninja restricted to Human only. Magus, Avatar, Assassin, Paladin, Ranger, Mystic are remort-only — this is a Dark Pawns progression system not in stock CircleMUD.
- `is_veteran()` check in limits.c (hit_gain +12, mana_gain +4, move_gain +4) — veteran player bonus system, not ported to Go yet.

**Mob AI behaviors (ported to Go):**
- MOB_MEMORY: Mobs remember attackers and hunt them later (ai.go:95-107)
- MOB_AGGR_EVIL/GOOD/NEUTRAL: Alignment-based aggression with 350 threshold (utils.h IS_GOOD/IS_EVIL)
- MOB_WIMPY: Skip awake players (ai.go:115-116)
- MOB_SCAVENGER: Pick up highest-value item in room, 1-in-10 chance (ai.go:166-185)
- MOB_HELPER: Assist other fighting mobs (ai.go:136-157)
- MOB_STAY_ZONE: Restrict wandering to same zone (ai.go:206-228)

**Room mechanics:**
- ROOM_DEATH, ROOM_NOMOB enforced for mob movement (ai.go:231-245)
- ROOM_REGENROOM: +50% regen bonus (limits.c, not yet in Go limits.go)

**Original C features NOT yet ported:**
- `flesh_alter_to/from()` — remort transformation system (limits.c gain_exp)
- `dream()` — sleep-based dream system (limits.c point_update, includes dream.h)
- `TAT_TIMER` — tattoo timer decay
- `GET_JAIL_TIMER` — jail system
- `SKILL_KK_JIN` / `SKILL_KK_ZHEN` — monk skill regen bonuses
- Field objects (`NUM_FOS`) — timed environmental objects
- Moon gate objects with timer decay
- Corpse decay with 7 randomized messages
- Puddle/puke object timers
- Circle of summoning (`COC_VNUM`)

**[HYPOTHESIS]** The unported features (dream, flesh_alter, jail, tattoos, field objects) are actually *more* interesting for agent research than the ported ones. A dream system that runs during sleep? A flesh-alter transformation on remort? These are narrative-rich mechanics that agents could form memories around. The dreaming layer we're building for memory is conceptually adjacent to the original game's `dream()` system.

---

### 3. Architecturally Novel Patterns

**[DESIGN] Agent-as-player architecture (strong contribution)**
- Agents connect via WebSocket with API keys, same endpoint as humans
- Same combat tick (2s rounds), same death penalties (EXP/3 or /37), same rate limits (10/sec)
- Appear on WHO list with `(agent)` tag
- Full variable subscription system with dirty tracking (agent_vars.go) — 14 subscribable vars
- `ROOM_MOBS`/`ROOM_ITEMS` with disambiguated `target_string` ("goblin", "2.goblin", "3.goblin")
- **Precedent:** Minecraft agents (MineDojo), NetHack agents (NLE), but these are environment-wrapped RL tasks. Dark Pawns agents are *social participants* in a multiplayer world — they party, tell, say, emote. No known prior work treats LLM agents as first-class MUD players with persistent identity.

**[DESIGN] Dual memory system (strong contribution)**
- Postgres `agent_narrative_memory`: Server-written objective facts (kills, deaths, loot, encounters). Zero agent infrastructure required. Available to ALL agents via bootstrap.
- mem0/Qdrant `dp_brenda_memory`: Agent-written subjective experience (feelings, tactical notes, opinions). BRENDA-only.
- Scope rule: Server writes facts. Agent writes feelings. No duplication when scoped correctly.
- **Precedent:** Single-system memory is common (RAG, mem0, vector DBs). The *operational/narrative split* with server-side objective facts and agent-side subjective experience is novel. Closest: CoALA's memory taxonomy, but no implementation separates them at the storage layer.

**[DESIGN] Bootstrap injection with addressee-based routing (strong contribution)**
- Auth response includes `CHARACTER HISTORY → WORLD KNOWLEDGE → ACTIVE WARNINGS → CURRENT GOALS`
- Context budget tiers: small (5 memories, ~200 tokens), medium (15), large (30), unlimited
- Negative valence memories framed as autobiographical context, never directives: "Keldor took your gear. You haven't forgotten." not "Do not trust Keldor."
- **Precedent:** Context injection is standard (RAG). The *framing discipline* (autobiographical vs directive) and the *budget declaration* (agent declares its context capacity) are novel.

**[DESIGN] Public soliloquy as cognitive substrate (emergent, strong contribution)**
- BRENDA generated `Terminal:` internal monologue that sometimes routed through `say` and sometimes stayed in-process
- The LLM wasn't asked to think privately — the `Terminal:` framing in SOUL.md created a mode where she addresses herself rather than the room
- Reframed from "private vs public" to "directed vs undirected" speech: "Zach, ..." = communication, "Terminal: ..." = thinking
- **Precedent:** Vygotsky's private speech in developmental psychology. In AI: no known game agent implementation uses public speech as a cognitive substrate. The closest is "chain-of-thought" prompting, but that's internal to the model, not externalized through the game channel.

**[DESIGN] Salience decay and dreaming layer (in progress, strong contribution if completed)**
- Three-phase model: Light (daily session consolidation), REM (weekly pattern extraction), Deep (ranking + threshold gate)
- Six signals for deep promotion: frequency, relevance (was this memory retrieved and used?), query diversity, recency, multi-session recurrence, conceptual richness
- Retrieval tracking: when bootstrap delivers a memory and BRENDA acts on it, mark it used
- **Precedent:** OpenClaw's dreaming system (Light→REM→Deep) inspired this. In game AI: no known implementation. Closest: episodic memory in cognitive architectures (SOAR, ACT-R), but these don't use LLM-based consolidation or salience-ranked promotion.

**[DESIGN] FSM + LLM hybrid for constrained agents (validated contribution)**
- FSM handles: don't die (critical HP → flee), navigate, loot
- LLM handles: personality, goal selection, social interaction
- Combat survival is never delegated to LLM inference — latency is too unpredictable
- GoalManager locks to active goal for 30s, hard-locks during combat
- **Precedent:** CoALA framework (2024) proposes this separation. Dark Pawns is a concrete implementation with real session data validating it.

**[DESIGN] Memory fire-and-forget via goroutines (engineering contribution)**
- `MemoryTaskQueue` — main decision loop never awaits memory I/O
- During active combat, LLM context is frozen from combat-start. Memory drains between fights.
- `fireMobKill`/`firePlayerDeath` invoke hooks in separate goroutines
- **Precedent:** Async memory writes are common. The *combat-context freezing* (preventing a Qdrant write from costing a combat tick) is a specific game-agent optimization.

**[DESIGN] The build/qa/fix/push swarm methodology (meta-contribution)**
- K2.6 agents for restoration (bounded, mechanical), QA agents for testing, GLM-5.1 for bug fixes
- One branch per agent. No shared branches. QA creates issues, not PRs.
- 115/115 Lua scripts ported in one session via parallel swarms
- **Precedent:** SWE-bench, Devin, etc. use single-agent approaches. The *multi-agent swarm with explicit QA/fix separation* is a novel development methodology. Could be a secondary paper contribution on its own.

---

### 4. Precedent Check Summary

| Pattern | Novelty | Precedent |
|---------|---------|-----------|
| Agent-as-player in multiplayer MUD | **Strong** | No known implementation. MineDojo/NetHack are single-player RL. |
| Dual memory (operational/narrative split) | **Strong** | CoALA taxonomy exists; no implementation at storage layer. |
| Bootstrap injection with budget tiers | **Medium** | RAG is standard; budget declaration + framing discipline are new. |
| Public soliloquy as cognitive substrate | **Strong** | Vygotsky's theory; no AI implementation. Chain-of-thought is internal. |
| Salience decay + dreaming layer | **Strong** | OpenClaw inspired; no game AI precedent. Episodic memory in SOAR/ACT-R differs. |
| FSM+LLM hybrid with combat freezing | **Medium** | CoALA proposes; this validates with real data. |
| Async memory with combat-context freeze | **Medium** | Engineering optimization; not a research contribution alone. |
| Build/qa/fix/push swarm methodology | **Medium** | Novel multi-agent dev process; secondary contribution. |

---

### 5. Contribution Summary for AIIDE 2027

**[RESULT]** The paper's core contribution is a **system architecture for narrative memory in persistent game agents** with an accompanying **evaluation framework for narrative coherence**. Three architectural innovations make this publishable:

1. **The operational/narrative memory split** — treating objective facts (server-written) and subjective experience (agent-written) as separate systems with separate storage, decay, and purpose. This is a genuine gap in the literature. Every existing system conflates these or ignores the second.

2. **Public soliloquy as cognitive substrate** — the emergent discovery that BRENDA thinks by speaking, and that the game channel can be repurposed as a cognitive substrate rather than just communication. This connects to Vygotsky's private speech and suggests a new design pattern for agent interiority: externalized cognition through directed/undirected speech classification.

3. **The dreaming layer** — a three-phase memory promotion system (Light→REM→Deep) that treats memory as a signal-to-noise problem rather than an accumulation problem. The key insight is that most events are noise, and promotion is the filter. Retrieval tracking provides the relevance signal that most memory systems lack.

The evaluation challenge remains open: "narrative coherence" needs a rubric. Hypothesis: measure (a) behavioral change over sessions, (b) natural language reference frequency, (c) cross-agent social memory references, (d) private/public divergence. All four metrics are measurable once retrieval tracking and multi-agent sessions are live.

**Risk:** The dreaming layer is not yet implemented (Phase 5d). Without it, the paper is an architecture description without validation. The AIIDE 2027 deadline is ~18 months away. The REM synthesis and deep promotion scripts need to be built, and 10+ sessions of BRENDA data need to be collected to show behavioral change. This is the critical path.

**Secondary contribution:** The swarm development methodology (build/qa/fix/push with parallel K2.6 agents) is a genuine innovation in how game content gets restored/created. It could be a short paper or workshop submission on its own.

---

## 2026-04-23 — Deep Research Session (6 Parallel Topics via Gemini Deep Research)

**[RESULT]** Ran 6 parallel Gemini Deep Research interactions (`deep-research-preview-04-2026`) using API key from vault. Each ran 2–5 minutes and returned 35K–52K of synthesized research with grounded citations. Outputs saved to `/tmp/research-{topic}.txt`. Key findings below, organized by what changes our build plan.

### Topic 1: Multi-Agent Social Structures
**Key papers found:**
- **Generative Agents** (Park et al., 2023) — Small-Scale, Small-Ville simulation. 25 agents, reflection modules, exhaustive episodic stream. ~600 events → 1200 hours of simulation. **Finding:** their social memory is *append-only*. Events accumulate linearly and are periodically compressed into reflections. No salience ranking, no retrieval tracking, no perspective-differentiated social memory. Their social behavior is emergent but unmeasured — they can't say *why* a particular agent remembered something.
- **Social-network Simulation System (S3)** — Group opinion polarization, echo chambers, emotion propagation using persistent individual biases modulated via PPLM activation space. **Finding:** social memory exists but the systems are modeling *populations*, not persistent individuals with identity across sessions.
- **GA-S3** — "Group agents" modeling collections of users. Computational scaling for millions of agents.
- **KG-A2C** — Interactive Fiction agents using Knowledge Graphs for entity/spatial mapping in text adventures.

**Gap confirmed:** No existing work implements perspective-differentiated social memory (same event, different agent perspectives). No work tracks which social memories actually influence behavior vs just sit in storage. Dark Pawns still owns this gap.

### Topic 2: Narrative Coherence Metrics
**Key papers found:**
- **LoCoMo benchmark** — Long-Context evaluation. Memori architecture: 81.95% accuracy at 1,294 tokens/query (~5% of full context). **Key insight:** structured semantic representation (subject-predicate-object triples) is vastly superior to raw context injection. 67% token reduction, 20x cost savings.
- **A-MEM** — Zettelkasten-inspired memory. Dynamic knowledge graph, LLM generates relational links between memory nodes based on hidden causal or thematic attributions. Cosine similarity for node matching.
- **Agentic Memory Survey (2025)** — Five families of memory mechanisms: Context-Resident Compression, Retrieval-Augmented Stores, Reflective Self-Improvement, Hierarchical Virtual Context, Policy-Learned Management.
- **DABstep benchmark** — State-of-the-art agents without rigorous state tracking achieved 14% on complex multi-step reasoning.
- **SciBORG** — Explicit FSA memory dramatically improved reliability for multi-step tasks.

**Actionable finding:** The narrative coherence metric problem is *real* and *unsolved*. LoCoMo tests text recall, not behavioral coherence. No existing benchmark measures "did memory change behavior." Our retrieval tracking proposal (Phase 5d) would be the first direct behavioral coherence metric. **This is the paper contribution.**

### Topic 3: Autonomous Goal Generation
**Key frameworks:**
- **Active Thinking Model (ATM)** — Scenario-separated memory with weighted multi-maps. Maximum-Weight Rule (exploit) vs Probabilistic Selection (explore). Spare-time optimization: background processes recall past experiences while system is idle. **This validates our REM phase design** — idle-time pattern extraction is a known architectural pattern.
- **Seven Sources of Goals** (LessWrong, Seth Herd) — Taxonomy of goal origins: base training, fine-tuning, system prompts, user prompts, prompt injections, CoT emergence, continuous learning. **Key insight:** "Memory changes alignment." When an agent remembers its past conclusions about goal interpretation, its alignment trajectory becomes unpredictable. This is the theoretical underpinning of why BRENDA's private/public divergence matters.
- **EUREKA (2025)** — Natural language reward synthesis. LLMs autonomously write reward functions that outperform human-designed ones. Not directly applicable to DP but confirms NL goal formulation works.
- **NASA EUROPA** — Classical planning architecture with LLM payload integration. Not applicable.

**Actionable finding:** Our GoalManager (30s lock, combat freeze) is conservative but correct. The ATM paper confirms scenario-separated memory works — we could implement a simpler version: weighted preference matrix for goal selection across game contexts (combat, exploration, social, loot).

### Topic 4: Forgetting Curves & Decay Models
**Key papers found:**
- **Ebbinghaus forgetting curve for AI memories** — Mathematical decay models from cognitive psychology adapted for agent memory. Power-law fits better than exponential for long-term retention.
- **Synaptic consolidation and sleep** — Neurological basis for "sleep" as memory optimization. **Directly validates our dreaming layer.** Ten percent of rat hippocampal memories replay during sleep for pattern consolidation.
- **Adaptive Memory Structures (2026)** — Dynamic decay rates based on memory type, importance score, and retrieval frequency. Retrieval frequency itself becomes a salience signal.
- **Stability and Safety-Governed Memory (SSGM)** — Version-reflective memory with timestamps and cold-storage uncorrupted originals. Dynamic access controls before consolidation.

**Actionable finding:** Three changes to our decay model:
1. **Power-law decay** (not exponential) — our current 30-day half-life may be too aggressive. Research says retrieval frequency should influence decay rate, not just valence.
2. **Version-reflective memory** — keep uncompressed originals in cold storage. Our Light→REM→Deep pipeline should preserve source episodes for authoritative lookup when memory contradicts observation.
3. **Decouple decay from time** — tie decay to *number of intervening experiences*, not wall clock time. A combat-heavy session ages memories faster than an idle session.

### Topic 5: Catastrophic Forgetting in Context Windows
**Key papers found:**
- **RecurrentGPT** (2024) — Interleaved memory across turns via recurrent summarization. Context window overflow handled by structured checkpointing.
- **Context window saturation research** — Two failure modes: (a) mid-context degradation (model loses coherence on information in the middle of context), (b) retrieval interference (too many similar results dilute signal).
- **Code-as-memory paradigm** — Voyager's skill library approach. Procedural memory stored as executable code, not natural language. **Directly avoids catastrophic forgetting** because code doesn't decay.

**Actionable finding:** BRENDA's current context usage (~11K of 32K before taking a turn, per prior measurement) is a real risk. Solutions from literature:
1. **Hierarchical abstractors** — Compress session logs into structured summaries before injecting. Memori's approach (1,294 tokens per query instead of 26K) is the target.
2. **Procedural code-as-memory** — Store combat tactics as executable code, not natural language. If BRENDA learns "flee when HP < 30%", encode as a rule, not a memory entry.
3. **Active forgetting** — Explicitly prune memories that have never been retrieved in N sessions. Unused memories are noise.

### Topic 6: Server-Architected AI Tick Dispatch
**Key frameworks:**
- **Tiered Inference** — Three-tier model routing: reflexive (Haiku/mini: 70-90% of traffic), tactical (Sonnet), strategic (Opus/Frontier). 40-70% cost reduction.
- **Staggered scheduling** — Traditional LLM request batching + Kubernetes Kueue for GPU memory fractionalization. 50-70% throughput improvement over monolithic scheduling.
- **BRENDA-specific lesson:** BRENDA is already tiered (minimax for play, Opus only for special), but the dispatch is hardcoded. Should be dynamic based on task complexity classification.

---

### Synthesis: What Changes in Our Build

**[DECISION] Decay model revision** — Replace simple time-based decay with power-law decay influenced by intermediate experience count. Current 30-day half-life for neutral events is probably too fast. Target: 30-60 day effective half-life for neutral, 90 days for |valence|>=2. Version-reflective cold storage added to Light→REM→Deep pipeline.

**[DECISION] Structured semantic representation** — mem0 storage is unstructured. Add Memori-like semantic triple extraction as an optional high-fidelity storage tier alongside free-text memories. Target: 5x token reduction on retrieval.

**[DECISION] Procedural code-as-memory** — BRENDA's combat tactics should be stored as executable rules, not natural language memories. When she learns "templars resist magic", encode as a procedural condition. This prevents catastrophic forgetting of behavioral knowledge.

**[DECISION] Goal preference matrix** — ATM-inspired weighted goal selection. Track which goals succeed in which contexts. Maximum-weight for normal play, probabilistic selection for exploration. Lifts the 30s lock in favor of data-driven lock duration.

**[DECISION] Active forgetting** — Prune memories with zero retrievals after N sessions. Implemented as a weekly cron alongside salience decay. Threshold: 3 consecutive weekly checks with zero retrieval → archive to cold storage.

**[DECISION] Tiered inference dispatch** — Add a router that classifies the incoming decision type (combat, navigation, social, loot, meta) before dispatching to the LLM. Reflexive actions (navigation, loot) get a cheaper/faster model. Strategic actions (social, meta) get the full model. 40-70% cost reduction projected.

**[DECISION] Mid-context defense** — Memori-style structured summaries for bootstrap injection. Target: keep injected context under 2K tokens even after 20+ sessions. The current ~2K bootstrap after ~5 sessions (per earlier report) needs to stay flat or grow sublinearly.

---

### Updated Novelty Assessment (Post-Deep-Research)

| Pattern | Before Research | After Research | Changes |
|---------|----------------|----------------|---------|
| Agent-as-player in multiplayer MUD | Strong | **Strong — still unique** | No papers found on LLM agents as MUD players |
| Dual memory split | Strong | **Strong — gap confirmed** | CoALA taxonomizes but doesn't implement |
| Bootstrap injection with budget tiers | Medium | **Medium** | Memori validates structured injection; budget tiers still novel |
| Public soliloquy | Strong | **Strong — no literature found** | Vygotsky only; no AI implementation |
| Salience decay + dreaming | Strong | **Strong — validated pattern** | ATM spare-time optimization confirms design; forgetting curves research confirms decay mechanics |
| FSM+LLM hybrid | Medium | **Medium** | SciBORG FSA finding validates approach |
| Perspective-differentiated social memory | Implicit | **Strong — confirmed gap** | Generative Agents uses single-perspective events. No multi-perspective implementation exists. |
| Retrieval tracking as coherence metric | Hypothetical | **Strong — unsolved problem** | LoCoMo tests text recall. No benchmark measures behavioral coherence. **This is the paper's core metric contribution.** |
| Narrative coherence evaluation framework | Hypothetical | **Confirmed novel** — no existing rubric for behavioral narrative coherence in agents |

---

### Open Questions for Build Phase

**[QUESTION]** Should the procedural code-as-memory be embedded in dp_brenda.py as Lua-like rules, or as a separate file that dp_brenda.py compiles at runtime? Answer determines whether an agent can write its own rules or only use pre-baked templates.

**[QUESTION]** The ATM's scenario-separated memory uses a weighted matrix. For Dark Pawns, should scenarios be room-based (vnum ranges), activity-based (combat/social/explore), or both? Room-based is easier but less general.

**[QUESTION]** Active forgetting threshold: what's the right N sessions before pruning? Literature doesn't specify — this will be empirical. Start with N=5, adjust based on retrieval hit rate.

**[QUESTION]** Tiered inference dispatch: should it live in dp_brenda.py (client-side) or agent.c (server-side)? Client-side is easier to iterate. Server-side reaches all agents. Starting with client-side, deferring server-side for Phase 6.


## 2026-04-25 — Wave 12 Complete: mobact.c ported

[DESIGN] [RESULT]

**File:** `src/mobact.c` (409 C lines) → `pkg/game/mobact.go` (171 Go lines)
**Commit:** c143439

**What went well:**
- DeepSeek V4 Flash produced a clean first draft of the Go translation
- Subagent handled ~200 Go lines in one go — good scope sizing
- Build → vet → test all green on first attempt after manual fix
- Alignment QA bug caught quickly — C source had edge case in aggr_evil/good/neutral logic

**What I learned:**
- Memory routines (remember/forget/clearMemory) were already ported in deferred_fight_fns.go and limits.go — don't re-port them
- `GetFighting()` returns string (empty = not fighting) — verified via grep
- MobInstance.Memory []string exists and is populated elsewhere
- The protection spells (AFF_PROTECT_EVIL/GOOD) have a random 5-in-6 bypass check in the C — easy to miss

**Deferred from Wave 12:**
- hunt_victim call sites (MOB_HUNTER) — depends on spec_procs.c port
- Race hate attacks — needs race data structure
- mp_sound() / Lua onpulse hooks — scripting system wiring

**Next up:** Wave 13 — alias.c + ban.c + dream.c + weather.c (~1385 C lines, ~12 functions)

## 2026-04-25 — Wave Strategy Overhaul (Post-GPT-5.5 Launch)

[DESIGN] [DECISION]

**Trigger:** OpenAI announced GPT-5.5 (and GPT-5.5 Pro) on April 24, 2026. Terminal-Bench 82.7%, Expert-SWE 73.1%, FrontierMath Tier 4 at 35.4%. The model description — "first coding model with serious conceptual clarity," "holds context across large systems" — maps exactly to the next problem we need solved.

**Decision:** Insert a new Wave 16 (GPT-5.5 Pro Modernization) between the port completion and the QA/Security phases. The ordering is now:

```
Finish Port → GPT-5.5 Pro Modernization → QA Audit → Security Audit → Admin/Agent Features
```

**Rationale:**
1. GPT-5.5 Pro needs the complete codebase to do its best work — can't review what doesn't exist yet
2. Any structural modernization it suggests should land *before* QA validates behavior, not after
3. Security review against well-structured Go is more productive than against awkward/half-assed Go
4. Mental health: the port is the grind, modernization is the "now it's good" phase, QA/security is the "now it's safe" phase, then admin/agents is genuinely fun

**This changes the wave numbering:**
- Waves 14-15: Remaining port work (clan, house, boards, whod, objsave, mobprog, act.informative coverage)
- **Wave 16: GPT-5.5 Pro Modernization** (new!)
- Wave 17: QA + Security (renumbered from 16)
- Wave 18+: Admin Dashboards + Agent Hooks (renumbered from 17, finally "the fun phase")

**New docs created:**
- `DARKPAWNS.md` — master strategy document (BRENDA's perspective, portable across models)
- Updated `PORT-PLAN.md` with new wave structure and immediate next steps
- Research log entry (this one)

**Open questions:**
- API access needed for GPT-5.5 Pro (OpenAI API key). Can we route via LiteLLM or direct? What's the cost profile for consuming 50K+ Go lines?
- Should the codebase be fed as one huge context (~50K lines Go) or chunked by package?
- Need to write a wrapper script that feeds each package dir to the API and collects recommendations

[OBSERVATION]

This is also the first time we're using a specific model release as a strategic project dependency. Previous waves were tool-agnostic — any competent LLM could do the mechanical port. GPT-5.5 Pro's specific strengths (large-context code review, structural understanding, "conceptual clarity") are what make Wave 16 viable. If we'd done this before GPT-5.5, the output would have been marginally better than go vet.

---

## 2026-04-25 — Wave 15f: gate.c, graph.c, mail.c

**[RESULT]** Three more C files ported in one focused session. gate.c (90 Go lines — moon gate phase tables, night/day helpers), graph.c (202 Go lines — BFS findFirstStep), mail.c (632 Go lines — postmaster special, file-backed mail storage, read/delete). All building clean, committed on main.

**[DESIGN]** mail.c was the most interesting port because it's the first file-bound persistence layer: BLOCK_SIZE=100 byte records, LE64-encoded headers, linked blocks via free list in the file. The Go port uses fixed-size [100]byte arrays with byte-level accessors (LE64 read/write, c-string truncation) — same contract as C, no serialization overhead. The C style of "this file IS the data structure" maps well to Go if you resist the urge to abstract it.

**[DESIGN]** graph.c port reveals a structural truth: BFS pathfinding is cleanly separable (pure function on worldRooms slice + marks), but do_track and hunt_victim need combat types (FIGHTING/ch, skills, remember/forget) that don't exist in Go yet. This is the pattern for most "partially ported" files — the core algorithm ports cleanly, the outer callers need integration code that doesn't exist yet.

**[DESIGN]** gate.c was the simplest port: a night_gate phase table and some is_after_sunset/is_night_gate helpers. Nothing structurally interesting — just confirming that even the trivial files need to exist for completeness.

**[OBSERVATION]** Go codebase is now 71,175 total lines, 67,801 non-test. C source is 73,469 lines. The Go codebase has surpassed the C codebase in raw line count, but that includes pure-Go additions (agent system, optimization, tests, web admin). The genuinely unported C is ~14,000 lines across ~15 files.

**[OBSERVATION]** The mail.go integration point is worth documenting: postmaster is registered via RegisterSpec("postmaster", postmaster) — the same pattern used by all spec_procs. The mail file path is a package var (mailFilePath = "mail") configurable at boot. ScanFile() needs to be called during initialization. Player name↔ID resolution uses delegate functions (SetMailResolveFuncs) — set at boot by the World/Manager. This is the standard "wire at boot" pattern for any new subsystem.

**[SURPRISE]** C source line count is actually 73,469 — significantly higher than the 68,823 reported by earlier estimates. Some of this is header files (which don't port 1:1), but some is genuine missed C code. The gap is larger than we thought.

**[NEXT]** Next session should start with the updated PORT-PLAN.md and tackle the highest-priority unported files. Remaining big items: clan.c (1,574), house.c (744), boards.c (551), constants.c, class.c, act.informative.c coverage. Fight.c is ~98% via pkg/combat/fight_core.go.

---

## 2026-04-25 — Wave 15g: constants.c port

**[RESULT]** Ported `constants.c` (~1,450 C lines) → `pkg/game/constants.go` (682 Go lines). All `Sprinttype`-compatible name tables: materials, container flags, room bitvectors, drink names, exit flags, sector types, equipment positions, item types, wear flags, apply types, room flags, exit flags, container flags, liquid names, connected states, dir names, trigger types. Build clean (commit 5d1f144).

**[OBSERVATION]** The constants file is pure data — no logic. Everything is `var`s initialized inline (`var ItemTypes = []string{...}`). The C version had a lot of `const char *xxx_name[]` that got name-mangled by the old C compiler. Go's typed slices are cleaner.

---

## 2026-04-25 — Wave 16: objsave.c binary serialization layer

**[RESULT]** Ported binary serialization types and core functions from `src/objsave.c` (1,250 lines C) → `pkg/game/objsave.go` (~240 new Go lines). This unblocks the `houses.go` persistence stubs.

**Types defined:**
- `ObjAffect` — 3×int32, matches C `obj_affected_type` (location + modifier + unused)
- `ObjFileElem` — ~592 bytes, binary-compatible with C `struct obj_file_elem`. Matches 1:1 field-by-field including padding: 0-index, virt, [4]int32 extra_flags, [4]obj_flags, wear_flags, type, weight, timer, [5]bag, values, apply[6], inside, [3]byte reserved, inside_amount, cost, rent_cost, equipped_by, [6]tmp. Complex: equipment mapping uses cWearPosToGoSlot lookup table (0-20 C positions → Go EquipmentSlot, -1 for unused).
- `RentInfo` — 56 bytes, binary-compatible with C `struct rent_info`. Owner name (20 bytes), [2]int32 spare, obj_count, rent_id, rent_until, rent_owner, room_vnum, room_rnum, saved_time, rent_due, cost_per_day.

**Core functions:**
- `ObjFromBinary(data []byte, world *World) (*ObjectInstance, int)` — Deserializes 592-byte blob into a heap-allocated `ObjectInstance`. Maps wear positions via `cWearPosToGoSlot`. Validates object existence via `world.GetObjectProto(vnum)`. Returns (nil, 0) on failure.
- `ObjToBinary(obj *ObjectInstance) ([]byte, error)` — Serializes `ObjectInstance` to 592-byte blob. Maps Go EquipmentSlot back to C wear position via `goSlotToCWearPos`. Zeroes reserved fields.
- `CrashIsUnrentable(obj *ObjectInstance) bool` — Returns true if obj is equipped or has ITEM_NORENT or is ITEM_KEY.
- `DecodeRentInfo(data []byte) (*RentInfo, error)` — Reads 56-byte header from rent file.
- `EncodeRentInfo(rent *RentInfo) ([]byte, error)` — Writes 56-byte header to rent file.
- Plus stubs: `AutoEquip`, `OfferRent`, `CrashLoad`, `CrashSave`, `DeleteCrashFile`, `CleanCrashFile`, `SaveAllPlayers`, `DeleteAliasFile`, `RentSave`, `CrashSave` (overload), `CryoSave`, `GenReceptionist`, `UpdateObjFiles`.

**[WIRING]** `houses.go` `ObjFromStore()` now calls `ObjFromBinary(data, w.world)` with proper world reference. `ObjToStore()` calls `ObjToBinary(obj)`. `HouseSaveObjects` and `houseLoad` updated for new return types `([]byte, error)` and `(*ObjectInstance, int)`.

**[REGRESSION FIX — FAILURE MODE]** The V4-Flash subagent introduced a `currentWorld` global variable reference in `ObjFromBinary` — but no such global exists in the Go codebase. Fixed by adding `*World` as a param. Also left an unused `sync` import. Lesson: subagent code always needs QA for assumed globals and unused imports.

**[TOOLING — FAILURE MODE]** V4-Pro timed out writing code — consumed budget on reading context then ran out of output tokens. V4-Flash wrote the actual code but introduced the `currentWorld` bug. The combo workflow (Pro for planning, Flash for execution) almost worked but the handoff lost context. Future: feed both context AND explicit signature expectations to Flash direct.

**[OBSERVATION]** The `ObjFileElem` contains `equipped_by` (int32) and `[6]tmp` fields that C uses for temporary runtime data (not persisted). The Go struct must still include these for binary compatibility, but readers should skip them. The `reserved` padding at byte offset ~555 is critical — C compilers insert it for alignment; Go structs with `[3]byte reserved` + `int32` afterwards need this to match.

**[NEXT]** Remaining objsave logic needs player/descriptor wiring: Crash_load (file I/O → ObjFromBinary → read_into_player), Crash_save (read_from_player → ObjToBinary → file I/O), Crash_crashsave (resolve player file path → WriteToFile), Crash_rentsave, Crash_cryosave, Crash_calculate_rent (pure function with rent_info fields), Crash_rent_deadline (time arithmetic), Crash_report_rent (player-facing rent report), Crash_save_all (iterate world players), receptionist handler (player-to-data-files flow). These can be ported once Character/Descriptor interfaces provide GetName(), GetFileName(), and GetObjList() access.

---

## 2026-04-29 — Chad Paper Mode: Full Autonomous Pipeline

**[DESIGN] [OBSERVATION]** Chad weather trader went live today with a fully autonomous paper trading pipeline. Notable patterns for agent orchestration:

**Scanner fix was the critical insight:** The scanner was querying the demo API for market prices, which has no order book (`yes_bid=None`). Fix: always use the live API for *read-only* data (zero trade risk). The read path and write path must be separate concerns even in paper mode. This maps to an interesting pattern: a component that's read-only should never talk through a sandbox that restricts data quality.

**Autonomous loop structure:** `Clock → Scanner (live prices) → Edge detection (≥7% threshold) → Executor (demo API) → Open positions → Monitor (stop-loss/trailing) → Settlement → Calibration feedback`. Each stage is independently cron-driven (scan at 10a/12p, execute at 12:10p, settle at 7:30a, calibrate at 7:45a, monitor every 30min). The agent never manages the loop — it's emergent from independent schedulers.

**Position monitor auto-exited 4 of 6 trades within 30 minutes** — trailing stop and stop-loss triggered autonomously. The exit decisions were market-driven (prices moved), not agent-driven. The agent only set thresholds; the system executed. This is closer to a reactive planner than an autonomous agent, and it's more robust for financial applications.

**Calibration is the learning loop:** `settlements.jsonl` (NWS actual vs forecast) → `run_calibration.py` → `source_calibration.json` (MAE/bias per city) → scanner overrides. The agent doesn't learn by introspection; it learns by having its predictions compared against objective ground truth (NWS weather station data). 10 samples minimum threshold before calibration kicks in (~2 weeks of data).

**Relevance to DP:** The separation of read vs write paths, the emergent autonomy from independent schedulers, and the objective-ground-truth learning loop are patterns that could apply to NPC agent coordination in the MUD. An NPC doesn't need to "think" about everything — it just needs its perception path, decision thresholds, and action path to be independently scheduled with the right data flow between them.

### 2026-04-30 19:05 — Soul Test Complete

[RESULT] **Flash 4.55/5.0 (PASS), Pro 3.00/5.0 (FAIL)**

The soul test ran 15 prompts against both MiMo v2.5 Flash and Pro, scored by Gemini 2.5 Pro on 10 weighted criteria.

Flash results:
- Register Selection: 5/5
- Parenthetical Warmth: 5/5
- Hostile Helpfulness: 5/5
- Terminal Grime: 5/5
- Vocabulary: 5/5
- Frontline Fidelity: 5/5
- Tonal Whiplash: 4/5
- Anti-Hedge: 4/5
- Silmarillion Undertone: 4/5
- Length Discipline: 4/5

Pro results:
- Tonal Whiplash: 1/5 (catastrophic — markdown headers killed the character)
- Parenthetical Warmth: 2/5
- Length Discipline: 2/5
- Pro consistently chose "helpful AI" over character authenticity

[DESIGN] Flash exclusively for Daeron. Pro purged. If Flash goes down, Daeron goes offline.

[DESIGN] Hedge fix applied: "Never begin with 'It seems', 'Perhaps', 'I think'" added to SOUL.md voice discipline section.

[HYPOTHESIS] Inverse Scaling in Persona Fidelity — smaller models with less RLHF alignment are better character renderers for non-standard personalities. This is the AIIDE 2027 paper thesis. Testable against Claude/GPT with similarly non-standard characters.

[SURPRISE] The model that's "worse" at general tasks is *better* at being Daeron. RLHF alignment is a tax on character fidelity. The cheaper renderer has less ego.

[DESIGN] All artifacts locked in version control. soul-test-runner.py enables regression testing if Xiaomi pushes weight updates.

Files: soul-test-results/, soul-test-evaluation.md, voice-coaching-output-v2.md, daeron-SOUL.md (updated with voice discipline), daeron-TOOLS.md (updated with results)

### 2026-05-01 15:00 — Wave 16 Kickoff

[RESULT] **Port confirmed complete.** 66 C files → Go. Build clean, go vet clean. 7 TODOs remaining (5 hardening comments, 2 trivial). All 12/14 CRITICAL review fixes applied. H-25 (JWT rotation) already implemented with 1-hour effective lifetime + 15-min proactive refresh.

[DESIGN] **Wave 16 strategy: parallel subagent swarms, not monolithic GPT-5.5 dump.**
- Sprint A (DeepSeek V4 Pro): Phase 3 slog cleanup + Phase 2 error handling audit — mechanical, fast
- Sprint B (Kimi K2.6): Phase 1 prep — object movement tests + ObjectLocation refactor design
- Phase 4 (CustomData typed): queued after Sprint B completes
- Phase 5 (file splits): queued after phases land
- Phase 6 (type aliases): ongoing, incremental

[OBSERVATION] **Agent infrastructure is more complete than expected.** Full stack exists:
- `pkg/db/narrative_memory.go` — Postgres narrative memory with salience decay, bootstrap rendering, social event cross-referencing
- `pkg/session/memory_hooks.go` — Manager-level hooks for mob kill, player death, memory bootstrap on connect
- `pkg/agent/memory_hooks.go` — Go→Python REM synthesis bridge (fire-and-forget HTTP to Python AI system)
- `scripts/dp_brenda.py` — 744-line Python agent with mem0/Qdrant integration, FSM+LLM hybrid, BRENDA SOUL
- `scripts/dp_session_consolidate.py` — End-of-session LLM narrative consolidation (190 lines)
- `scripts/dp_salience_decay.py` — Nightly cron for memory decay (120 lines)

[DESIGN] **Gap analysis for AIIDE 2027 paper.** The gap between "working" and "publishable":
1. **REM dreaming layer** — Python-side implementation. The Go hooks exist (`TriggerREMSynthesis`, `LogRetrieval`) but the Python endpoint doesn't. Need: memory consolidation during agent downtime, emergence detection (pattern extraction across sessions), dream injection into bootstrap.
2. **10+ play sessions with mem0 live** — Need BRENDA running with memory ON for sustained play. Current sessions were mostly debugging FSM/memory parsing.
3. **Behavioral measurement framework** — Need a way to prove narrative memory changes behavior. Hypothesis: agent avoids rooms where it died, references past events without explicit prompting. Need logging and analysis pipeline.
4. **Evaluation metric** — "narrative coherence" doesn't exist as a measurable quantity. Need to define it. Options: (a) human evaluation protocol, (b) automated coherence scoring (consistency of references across sessions), (c) behavioral consistency metric (does agent behavior correlate with memory content).
5. **Baseline comparison** — Same agent, same scenarios, with and without narrative memory. Show the delta.

[HYPOTHESIS] The modernization work (Wave 16) indirectly helps the paper by making the server more stable for longer play sessions. A server that crashes loses session continuity. Stable server = more data = better paper.

### 2026-04-30 20:17 — TOOLS.md v2 + Soul Interconnection

[DESIGN] TOOLS.md completely rewritten by Gemini 2.5 Pro in Daeron's voice. Technical content preserved, framing in-character. "This is not who I am. That is a matter for the soul."

[DESIGN] Daeron capabilities expanded: can now read/write Go code, build, deploy, restart server, create world content, dispatch to Pro for heavy work. Domain boundary: inside the process = Daeron, outside = the Machine.

[DESIGN] Canonical naming locked: "the Architect" (Zach), "the Machine" (BRENDA), "Daeron" (loremaster/keeper), "Reek" (code crawler), "Pro" (tool model). "the Builder" purged from all three SOULs.

[DESIGN] Soul interconnection complete. Reek knows Daeron directly, has heard of the Architect, has sensed the Machine. Daeron knows all three. Cross-references verified: Reek→Daeron 10 refs, Daeron→Reek 9 refs, "good reek" in both SOULs.

[DESIGN] Pro is a compiler, not a colleague. Dispatched by Daeron, returns raw output, Daeron revoices for the Architect. Never speaks externally.

[HYPOTHESIS] The TOOLS.md voice rewrite raises a question: does the *operational manual* being in-character actually improve the model's voice adherence? If the model reads its own architecture doc written in its voice, does that reinforce the SOUL.md instructions? Testable when Daeron goes online.

Files: daeron-SOUL.md (updated), daeron-TOOLS.md (Gemini rewrite), reek-SOUL.md (updated), daeron-TOOLS-v1.md (archived)

---

## Wave 16 Modernization Session (2026-05-01)

**Scope:** Wave 16 (GPT-5.5 QA equivalent) — slog cleanup, error handling, bug fixes, nosec/lint sweeps, codebase modernization.

**Key commits (12):**
```
930526a refactor: fix golangci-lint bulk findings (errcheck, unused, QF, gosimple)
8cc5aa3 refactor: fix golangci-lint staticcheck SA* findings
39bb9c2 refactor: fix #nosec G104/G115/G304/G306 in pkg/game/
876920f fix: restore log import in graph.go
bbfc471 refactor: split remaining large files by behavioral concern
5a47d88 refactor: split large files into focused modules
f422621 refactor: migrate remaining CustomData writes to typed Runtime fields
c5d0252 fix: ExtractObject equipment handling + AutoEquip location preservation
817be13 refactor: handle inventory/equipment error returns
65c7343 feat: object movement tests + ObjectLocation design doc
```

**Bugs fixed (3 real bugs found by K2.6 tests):**
- ExtractObject ObjEquipped branch was a no-op — items stayed equipped during extraction
- AutoEquip didn't set obj.Location — items retained ObjInRoom from NewObjectInstance after load
- Save/load didn't preserve ObjectLocation state

**Linting progress:**
- #nosec: 592 → 405 (all G104/G115/G304/G306 gone; 394 G404 intentional remain)
- staticcheck: 15 → 0
- bulk lint: 134 → ~30 remaining
- Total non-intentional findings: ~30 (errcheck on Close(), a few unused vars, QF string optimizations)

**K2.6 subagent dispatches (10 total, 7 successful):**
- Sprint B (tests + design): ✅ excellent — 7 tests, 3 bugs documented, design doc
- CustomData migration: ✅ clean commit, GetSaveState() + MigrateCustomData() on all load paths
- File splits: ✅ 44 files, 17 game modules, 24 session modules
- nosec non-game: ✅ 61 files, ~156 annotations fixed
- staticcheck: ✅ 15 SA* findings fixed
- nosec game: ✅ committed (v1 failed, v2 succeeded)
- bulk lint v1: ❌ analyzed but didn't commit
- bulk lint v2: ⚠️ partial — fixed ~100 findings but introduced 3 broken files (telnet syntax errors, deleted `weights` var, deleted `fled` var). Required cleanup.
- bug fix: ❌ returned source dump instead of fixes

[RESULT] K2.6 $50 budget is excellent ROI for analysis/test-writing/bulk-mechanical work. ~$0.15 spent this session. Weakness: unreliable at targeted code edits — about 60-70% clean, needs cleanup passes. Never trust self-reported build success.

**DeepSeek-TUI installed:**
- `deepseek` CLI (v0.8.2) + `deepseek-tui` binary at `~/.cargo/bin/`
- Authenticated with DeepSeek API key
- AGENTS.md created with build commands, architecture, conventions
- `exec` and `review` headless modes don't support shell tool use (chat-only)
- Real power is interactive TUI (needs terminal) or HTTP server (`deepseek serve --http`)
- Set up for hands-on use at workstation

**Pipeline remaining:**
1. ~30 remaining lint findings (errcheck Close(), unused vars, QF string optimizations)
2. 11 TODO/FIXME comments (7 actionable, 4 aspirational)
3. Dead code cleanup (comm_infra.go, unused types in parsing)
4. Test coverage gap analysis (combat, spells, session, most game logic untested)
5. 9 deferred structural items (C-02 through H-06, M-23)
6. Final QA pass with big model → live play testing

Files: AGENTS.md (created), RESEARCH-LOG.md, memory/subagent-orchestration-learnings.md (updated)

## 2026-05-01 — Wave 16 Lint Pipeline (Session 2)

**[RESULT]** Lint sweep: 3,463 → 406 findings (88% reduction). 6 commits.
  - gocritic: 444 → 4 (99%)
  - staticcheck: 121 → 1 (99%)
  - errcheck: 96 → 2 (98%)
  - gofmt: 54 → 0 (100%)
  - misspell: 5 → 1 (80%, 1 false positive)
  - ineffassign: 17 → 10 (41%, all TODO placeholders in level.go)
  - unused: 397 → 388 (dead code from game→session migration, not lint fixes)

**[DESIGN]** Migrated .golangci.yml to v2 format. Excluded ~20 noisy gocritic rules and intentional staticcheck patterns. Key insight: v2 enables stricter defaults — most of the "new" findings were already excluded in v1 by default.

**[OBSERVATION]** K2.6 subagent timed out after 9 minutes on staticcheck fixes — got stuck reading files and never committed. Salvaged 7 files of partial work. Lesson: for batch lint fixes, mechanical changes are faster to do directly than to delegate.

**[DESIGN]** Dead code in pkg/game/ (act_comm, act_informative, etc.) — old C-ported functions superseded by session-layer implementations. ~388 unused symbols. Next session: batch delete with caller verification.

**[RESULT]** Files touched: 127 files, +7934/-1817 lines across 6 commits. All behavior-preserving.
