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
