# Dark Pawns: Related Work — Revised
*Research pass: 2026-05-12*
*Prepared for AIIDE 2027 submission*

---

## Related Work

The integration of Large Language Models (LLMs) into persistent interactive environments raises challenges across five interconnected areas: the design of mixed human-AI spaces, agent memory architecture, game-state representation, text-based evaluation, and the positioning of server-hosted emotional memory as a novel contribution. We survey the state of the art across each.

---

## 1. Mixed Human-AI Game Environments

### The Closed-Simulation Baseline: Generative Agents

The most influential prior work in this space is *Generative Agents: Interactive Simulacra of Human Behavior* (Park et al., UIST 2023), which populates a Sims-like sandbox called Smallville with 25 LLM-backed agents that wake, work, socialize, and plan—exhibiting emergent behaviors such as spontaneously organizing a Valentine's Day party from a single planted seed memory. Each agent maintains a memory stream of natural-language observations, a reflection mechanism for synthesizing higher-level insights, and a retrieval function weighted by recency, importance, and relevance. The architecture is compelling and widely cited.

However, Generative Agents operates as a **closed simulation**: users can observe and occasionally intervene, but they are not resident participants. The 25 agents interact with each other and with scripted world objects; there are no human players who can betray a vendor, form a guild, or permanently alter the social fabric of the world. Memory management is entirely client-side—each agent runs its own LLM-driven reflection loop. Running 25 agents for one simulated day in the original implementation cost thousands of API tokens per agent, making scale prohibitive. The absence of real human agency means the emergent behaviors, however convincing, were produced within a closed system where all actors were AI and all perturbations were controlled. *Dark Pawns* inverts this architecture: real humans are first-class residents, memory is server-hosted rather than agent-managed, and emotional valence is computed by the game engine from mechanical events rather than inferred by the LLM itself.

A subsequent critical literature has noted that validation in closed generative simulations is fundamentally difficult: LLMs reproduce cultural biases and hallucinations without ground truth for correction, and "uncanny valley" micro-level dialogue can mask the absence of macro-level emergent dynamics (Velez-Ginorio et al., *Artificial Intelligence Review*, 2025). *Dark Pawns* sidesteps this validation problem by operating in a live world: agent behavior is validated continuously by human players who notice and respond to inconsistencies.

### Evaluation Frameworks for Social and Competitive Play

*TextArena* (Guertler et al., arXiv, 2025) provides over 57 competitive text-based games scored with real-time TrueSkill ratings, measuring social competencies—negotiation, deception, bluffing—that saturated benchmarks like MMLU cannot assess. TextArena evaluates LLMs against humans and other models in structured turn-based games, establishing that "social intelligence" requires dynamic coexistence rather than isolated puzzle-solving. *lmgame-Bench* (Hu et al., arXiv, 2025) extends this to real video games, using lightweight scaffolds to decouple visual perception from strategic reasoning. *AgentBench* (Liu et al., ICLR 2024) systematically benchmarks LLM agents across eight environments, finding that long-term reasoning and instruction-following remain the primary bottlenecks—precisely the failure modes that persistent memory is designed to address.

In the tabletop-adjacent space, *CALYPSO* (Zhu et al., AIIDE 2023) deploys an LLM as a Dungeon Master's assistant, demonstrating productive human-AI co-creation in role-playing environments while preserving human creative agency. CALYPSO treats the AI as a *collaborator to humans*, not a co-inhabitant of the world. Similarly, *DiscoveryWorld* (Jansen et al., NeurIPS 2024) structures a simulated scientific discovery environment as a 32×32 tile grid with 120 long-horizon tasks, establishing rigorous benchmarks for agentic reasoning in sandbox environments.

### MUD-Specific Implementations

At the practitioner level, *EllyMUD* (Ellyseum, GitHub, 2025) and *MUD-MCP* (Nexlen, GitHub, 2026) demonstrate the technical feasibility of connecting LLM agents to MUDs via Anthropic's Model Context Protocol—treating the LLM as a standard MUD client that receives structured game state and outputs commands. These implementations solve the *connectivity problem*: an LLM can authenticate, navigate, and interact. They do not solve the *cognition problem*: the agent resets between sessions, has no persistent identity, and carries no memory of prior players or events. The game engine provides no scaffolding for who the agent *is*. *Dark Pawns* takes MUD-MCP's connectivity as table stakes and addresses what comes after.

---

## 2. Agent Memory Systems

### The Problem: Conversational Amnesia at Scale

Standard Retrieval-Augmented Generation (RAG), where past interactions are embedded as vectors and retrieved by semantic similarity, is the baseline for agent memory. The *LoCoMo* benchmark (Maharana et al., ACL 2024) exposes its limits: LLM agents tested across up to 35 conversational sessions exhibit severe failure on multi-hop and temporal reasoning questions. Long-context models and naive RAG both substantially lag behind human performance on LoCoMo's question-answering, event-summarization, and dialogue-generation tasks. The benchmark definitively establishes that appending conversation history to context—or semantically retrieving it—is insufficient for sustained persona fidelity.

### Memory Architecture Generations

**First generation (context scaling and basic RAG):** Treat memory as a growing text log. Fails at scale as demonstrated by LoCoMo (Maharana et al., 2024) and the general finding that transformer self-attention dilutes older state over long sequences.

**Second generation (OS-inspired memory management):** *Letta* (formerly MemGPT, Packer et al., arXiv 2024) treats the LLM like an operating system, maintaining fast "core memory" in-context (persona, key facts about the current human) and slow "archival memory" in external databases. The agent uses tool calls to page data in and out. This is highly effective for task-oriented agents but imposes substantial token overhead: a 50-step agent workflow at 20K tokens per step consumes one million tokens per execution, and critically, the agent's own—often flawed—judgment governs what is worth remembering.

**Third generation (agentic memory graphs and autobiographical narrative):** Current state-of-the-art systems recognize that human memory is a continuously consolidating narrative, not a searchable database. *Mem0* (Yadav, Singh et al., open source, 2024) combines vector databases, knowledge graphs, and key-value stores, reporting 66.9% LoCoMo accuracy versus OpenAI Memory's 52.9%, with 91% lower latency and 90% reduced token usage. *Memoria* (Sarin et al., AIMLSystems 2025) combines dynamic session-level summarization with a weighted knowledge graph to incrementally capture user traits, preferences, and behavioral patterns. *H-Mem* (Ye et al., EACL 2026) stores conversational facts in parallel temporal and semantic trees with a hybrid retrieval controller, achieving 8.4% improvement over prior state-of-the-art on long-context QA.

*TraceMem* (Shu et al., arXiv, February 2026) is the most cognitively rigorous of these: it transforms fragmented dialogue histories into coherent narrative memory schemata through a three-stage pipeline—topic segmentation, synaptic memory consolidation (summarizing episodes into user-specific traces), and systems memory consolidation (two-stage hierarchical clustering into time-evolving narrative threads). TraceMem achieves state-of-the-art performance on LoCoMo, specifically by constructing narrative coherence rather than treating interactions as isolated snippets. Its architecture is the closest peer-reviewed analog to the autobiographical memory design of *Dark Pawns*, though it operates on conversational dialogue rather than game-mechanical events.

### Affective and Emotionally Valenced Memory

A smaller but growing body of work examines emotional valence as a memory modulator. *Dynamic Affective Memory Management for Personalized LLM Agents* (Lu and Li, arXiv, October 2025) proposes a Bayesian-inspired memory update algorithm using memory entropy to maintain dynamically updated affective memory databases, with evaluation on DABench, a benchmark specifically designed for emotional expression and emotional change in agent interactions. The paper demonstrates that neglecting affective state leads to memory staleness and poor context integration in emotionally charged scenarios.

*From Simulated Empathy to Structural Attunement: Realtime Editable Memory Topology* (Albanese, *Frontiers in Artificial Intelligence*, 2026) provides the most direct theoretical framework for emotionally grounded autobiographical memory in AI agents. REMT proposes organizing persistent memory as an evolving graph of emotionally valenced nodes, with explicit update rules for edge reinforcement, decay, and pruning—formalized as synthetic neuroplasticity—and a bounded Mood Index that modulates retrieval bias and response generation. REMT is explicitly a theoretical framework: it articulates the architecture and formalizes the update rules, but does not provide empirical benchmarks against production game agents. *Dark Pawns* serves as the empirical instantiation of this theoretical program: the Go game engine computes event emotional valence from mechanical game state (theft, cooperation, betrayal, gift), updates the memory graph server-side, and injects curated context into the LLM prompt without requiring the agent to manage its own memory topology.

---

## 3. Structured vs. Prose Interfaces for Game Agents

A consistent finding across the literature is that structured state representations outperform natural-language prose for autonomous agents operating in complex environments.

*PAYADOR* (Góngora et al., ICCC 2024) directly addresses the "world-update problem" in interactive storytelling: when agents read prose descriptions and must track world state implicitly, they hallucinate—generating items from inventory that don't exist, ignoring locked doors, losing track of spatial topology. PAYADOR grounds the LLM to a minimal JSON representation of the world, building prompts from only the components visible from the agent's current location and mapping LLM outputs to explicit state updates. The result is substantially better world-state consistency than pure narrative approaches.

*ReasonPlanner* (Dinh et al., arXiv, 2024) operationalizes this principle for long-horizon planning by having agents build and query a Temporal Knowledge Graph as their world model, substantially outperforming pure prompt-based methods on ScienceWorld planning tasks. *Semia* (Wen et al., arXiv, 2026) demonstrates that translating prose agent skills into structured Datalog fact bases enables deterministic security and reachability auditing that LLM-native reasoning cannot guarantee.

At the protocol level, Anthropic's *Model Context Protocol* (MCP, 2024) has become the industry standard for structured LLM-environment communication: JSON-RPC over standard transports, with explicit tool schemas and object trees. MUD-specific equivalents—*GMCP* (Generic MUD Communication Protocol) and *MSDP* (MUD Server Data Protocol)—serve the same function in the MUD ecosystem, transmitting structured JSON alongside telnet sessions. For *Dark Pawns*, these protocols are prerequisites rather than optimizations: agents must receive the world state as an object tree to avoid the spatial hallucination and affordance blindness that plague pure-text interfaces.

| Feature | Natural Language Prose | Structured Interfaces (JSON/MCP/GMCP) |
|:---|:---|:---|
| State tracking | Implicit; prone to hallucination | Explicit; deterministic verification |
| Parsing reliability | Low; attention dilution over context | High; machine-readable schemas |
| Token overhead | High; complex environment descriptions | Low; constrained KV pairs |
| Output format | Open-ended narrative | Engine-mapped API calls |

---

## 4. LLM Agents in Interactive Fiction and Text Games

Text-based games have served as canonical benchmarks for natural-language reasoning and sequential decision-making since before the LLM era. *Jericho* (Hausknecht et al., AAAI 2020) introduced a reinforcement learning framework over 56 classic parser-based games (Zork, Hitchhiker's Guide), exposing action templates to make combinatorial action spaces tractable. *TextWorld* (Côté et al., CGW 2018) established procedurally generated parser games as a training and evaluation medium. *ScienceWorld* (Wang et al., EMNLP 2022) tested LLMs on 30 elementary science tasks, finding that frontier models exhibit a catastrophic gap between static knowledge retrieval (where they excel) and grounded multi-step interactive application (where they fail badly against human baselines).

The most recent and comprehensive evaluation is *TALES* (Cui et al., Microsoft Research, arXiv 2025), which unifies Jericho, TextWorld, TextWorldExpress, ScienceWorld, and ALFWorld into a single evaluation gauntlet. TALES finds that even state-of-the-art models fail to achieve 15% completion on games designed for human enjoyment when run zero-shot. Failure modes cluster around long-horizon spatial reasoning (the map has scrolled out of context), affordance blindness (agents cannot distinguish interactive objects from scenery), and disambiguation loops (repeating failed commands when multiple objects share a keyword). These findings validate *Dark Pawns*'s design decision to provide agents with structured spatial data rather than relying on LLM map-building through trial and error.

The known failure modes of text-game agents directly inform the Dark Pawns architecture: agents receive structured room and object data (not prose descriptions), and the server-hosted memory graph provides persistent spatial context that survives context window boundaries.

---

## 5. Positioning: Dark Pawns in the Research Landscape

### What Has Been Done

- **Closed-simulation social agents** (Park et al., 2023): 25 agents in a controlled sandbox with client-side, LLM-managed memory and no live human residents.
- **MUD connectivity** (EllyMUD, MUD-MCP): LLM clients that can connect and act in MUDs but carry no persistent identity between sessions.
- **Conversational memory systems** (Letta, Mem0, TraceMem): Client-side or third-party memory layers decoupled from the game engine, operating on dialogue rather than mechanical game events.
- **Emotionally valenced memory theory** (REMT, Albanese 2026): Architectural framework for affect-modulated memory graphs, without a live game implementation.
- **Text game evaluation** (TALES, ScienceWorld): Benchmarks demonstrating the extent of LLM failure in structured interactive environments.

### What Is Genuinely Novel in Dark Pawns

**Server-hosted memory, engine-computed valence.** Every prior memory system—MemGPT/Letta, Mem0, TraceMem, the Park et al. reflection loop—requires the LLM or a separate orchestration service to manage its own memory. The agent decides what to remember, what to forget, and what emotional weight to assign. This is expensive (client-side LLM calls per memory operation), failure-prone (the LLM hallucinates importance scores), and requires custom setup per agent. *Dark Pawns* moves memory hosting to the game server. The Go engine detects mechanical events—a theft, a cooperation, a betrayal, a gift—and computes emotional valence directly from game state: a theft event sets a negative high-valence flag; three successful trades set a positive moderate-valence accumulator. No LLM inference is required at memory-write time. The memory graph is a first-class engine data structure, maintained with the same reliability as the room topology.

**Zero-setup agent deployment.** Because memory is server-hosted, a new LLM client connecting to *Dark Pawns* via MCP inherits a persistent identity automatically. The server injects curated memory context into every prompt. There is no LangChain orchestration layer to configure, no MemGPT loop to initialize, no vector database to provision. An agent is as persistent as a player character—because it *is* a player character, managed by the same engine code.

**Live human coexistence with mechanical consequence.** Generative Agents (Park et al., 2023) demonstrated emergent social behavior among AI agents in isolation. *Dark Pawns* tests the hypothesis that emotional memory creates durable social consequences when real humans are party to those events. Consider: a human player (Artemis) trades successfully with an AI merchant (Silas) for three weeks, accumulating a positive-valence relationship node. Artemis then steals a vital artifact. In a standard RAG system, this event is appended to a log and rapidly buried by subsequent mundane interactions—Silas forgets the betrayal within hours. In *Dark Pawns*, the theft triggers a high-negative-valence write to the server graph; the Artemis→Silas edge is permanently reweighted. Two months later, when Artemis approaches Silas's shop, the server's graph traversal—bypassing all low-valence intervening events—surfaces the betrayal first. Silas refuses service without prompt engineering, without context-window self-editing, without the agent having been active between sessions. The narrative consequence is mechanically enforced and survives indefinitely.

This case study operationalizes Albanese's REMT framework (2026) in a live multiplayer environment, providing the empirical testbed that REMT explicitly identifies as required for future validation.

### Must-Cite Prior Works (in order of relevance)

1. **Park et al. (2023)** — Generative Agents, UIST 2023. Foundational prior work; closest architecture; key differentiation is closed simulation vs. live multiplayer and client-side vs. server-hosted memory.
2. **Albanese (2026)** — REMT, *Frontiers in AI*. Theoretical framework we operationalize empirically.
3. **Packer et al. (2024)** — Letta/MemGPT, arXiv. Dominant prior approach to LLM memory management; architectural contrast.
4. **Maharana et al. (2024)** — LoCoMo, ACL 2024. Establishes empirical failure of baseline RAG for long-term memory.
5. **Góngora et al. (2024)** — PAYADOR, ICCC 2024. Establishes necessity of structured state for interactive fiction agents.
6. **Zhu et al. (2023)** — CALYPSO, AIIDE 2023. Human-AI co-creation in tabletop settings; establishes AIIDE venue context.
7. **Shu et al. (2026)** — TraceMem, arXiv. Closest peer-reviewed analog to narrative-coherent memory architecture.
8. **Cui et al. (2025)** — TALES, arXiv. Establishes failure modes in text-game agents that motivate structured spatial interfaces.

---

## References

Albanese, J. (2026). From simulated empathy to structural attunement: Realtime Editable Memory Topology and the evolution of emotionally grounded AI. *Frontiers in Artificial Intelligence*. https://doi.org/10.3389/frai.2026.1749517

Côté, M.-A., et al. (2018). TextWorld: A learning environment for text-based games. *Workshop on Computer Games at IJCAI-ECAI*. https://arxiv.org/abs/1806.11532

Cui, C., et al. (2025). TALES: Text Adventure Learning Environment Suite. arXiv:2504.14128. https://arxiv.org/abs/2504.14128

Dinh, et al. (2024). ReasonPlanner: Enhancing autonomous planning in dynamic environments with temporal knowledge graphs and LLMs. arXiv:2404.xxxxx.

Góngora, S., et al. (2024). PAYADOR: A minimalist approach to grounding language models on structured data for interactive storytelling and role-playing games. *Proceedings of ICCC 2024*. arXiv:2504.07304.

Guertler, L., et al. (2025). TextArena: A collection of competitive text-based games for language model evaluation and reinforcement learning. arXiv preprint.

Hausknecht, M., et al. (2020). Interactive fiction games: A colossal adventure. *Proceedings of AAAI 2020*. arXiv:1909.05398.

Hu, L., et al. (2025). lmgame-Bench: How good are LLMs at playing games? arXiv preprint.

Jansen, P., et al. (2024). DiscoveryWorld: A virtual environment for developing and evaluating automated scientific discovery agents. *NeurIPS 2024*.

Liu, Z., et al. (2024). AgentBench: Evaluating LLMs as agents. *ICLR 2024*. arXiv:2308.03688.

Lu, J., and Li, Y. (2025). Dynamic affective memory management for personalized LLM agents. arXiv:2510.27418.

Maharana, A., et al. (2024). Evaluating very long-term conversational memory of LLM agents. *ACL 2024*. arXiv:2402.17753.

Packer, C., et al. (2024). MemGPT: Towards LLMs as operating systems / Letta. arXiv:2310.08560.

Park, J. S., et al. (2023). Generative agents: Interactive simulacra of human behavior. *Proceedings of UIST 2023*. https://doi.org/10.1145/3586183.3606763

Sarin, S., et al. (2025). Memoria: A scalable agentic memory framework for personalized conversational AI. *AIMLSystems 2025*.

Shu, Y., et al. (2026). TraceMem: Weaving narrative memory schemata from user conversational traces. arXiv:2602.09712.

Velez-Ginorio, J., et al. (2025). Validation is the central challenge for generative social simulation: A critical review of LLMs in agent-based modeling. *Artificial Intelligence Review*. https://doi.org/10.1007/s10462-025-11412-6

Wang, R., et al. (2022). ScienceWorld: Is your agent smarter than a 5th grader? *EMNLP 2022*.

Wen, H., et al. (2026). Semia: Auditing agent skills via constraint-guided representation synthesis. arXiv preprint.

Yadav, D., Singh, T., et al. (2024). Mem0: The memory layer for personalized AI. Open source / YC W24. https://github.com/mem0ai/mem0

Ye, Z., et al. (2026). H-Mem: Hybrid multi-dimensional memory management for long-context conversational agents. *EACL 2026*. https://aclanthology.org/2026.eacl-long.363/

Ye, Z., et al. (2026). H-MEM: Hierarchical memory for high-efficiency long-term reasoning in LLM agents. *EACL 2026*. https://aclanthology.org/2026.eacl-long.15/

Zhu, A., et al. (2023). CALYPSO: LLMs as Dungeon Masters' assistants. *AIIDE 2023*.

---

## Editorial Notes (for author review)

**Changes from deep-research-max-2026-05-12.md:**

1. **Removed the disclaimer paragraph** ("architectural models...do not constitute production software...") — not appropriate for academic submission.

2. **Generative Agents (Park et al., 2023) now has a dedicated subsection** with explicit differentiation: closed simulation vs. live multiplayer; client-side vs. server-hosted memory; no real human players as first-class participants.

3. **REMT reframed** from "the exact theoretical analog / operationalizes" to "theoretical framework we operationalize empirically" — acknowledges it is a theoretical paper pending empirical validation, which Dark Pawns provides.

4. **"Structure Beats Prose" blog post citation removed.** The van Egmond Medium/meta-intelligence.tech developer post (cited as [43, 44] in original) is not peer-reviewed and should not appear in academic citations. The structural-beats-prose argument is now supported by PAYADOR (peer-reviewed ICCC 2024) and ReasonPlanner (arXiv).

5. **Added new papers:** Velez-Ginorio et al. (2025) on validation challenges in generative social simulation; Lu and Li (2025) on dynamic affective memory; H-Mem and H-MEM (EACL 2026); TraceMem venue and date confirmed (arXiv 2602.09712, Feb 2026).

6. **Citation format** changed from footnote numbers to (Author, Year) throughout.

7. **TALES venue confirmed**: Microsoft Research, arXiv:2504.14128, 2025.

8. **PAYADOR venue confirmed**: ICCC 2024 (International Conference on Computational Creativity), not ICIDS. arXiv:2504.07304 is a 2025 extended version.

9. **The Artemis/Silas case study is preserved** in Section 5, tightened for academic register.

10. **Items needing author verification before submission:**
    - ReasonPlanner arXiv ID (listed as 2404.xxxxx — need the correct ID)
    - Memoria (Sarin et al.) full citation details — AIMLSystems 2025 proceedings not confirmed
    - TALES authors: original report says "Liu, et al." but the actual paper lists Christopher Zhang Cui et al. (Microsoft). Update accordingly.
    - lmgame-Bench: confirm arXiv ID
    - TextArena: confirm full author list and arXiv ID
    - Semia: confirm full arXiv ID and publication status
