You are conducting academic research for a paper targeting AIIDE 2027.

Project context: We are building Dark Pawns, a Multi-User Dungeon (MUD) where autonomous AI agents participate as first-class citizens alongside human players. The game is written in Go with a custom Lua scripting engine. Our novel contribution is a narrative/autobiographical memory system for game agents — emotionally valenced, socially cross-referenced, server-hosted, zero-setup.

Research the following areas thoroughly, with citations to specific papers, systems, and benchmarks:

1. **Mixed Human-AI Game Environments**: What academic work exists on environments where LLM-based agents and humans coexist in the same game world? Include TextArena, lmgame-Bench, CALYPSO, TILE/DiscoveryWorld, AgentBench, and any MUD-specific agent work (e.g., MUD-MCP, EllyMUD). What evaluation frameworks are used?

2. **Agent Memory Systems in Games**: Survey memory architectures for game agents — episodic memory, autobiographical memory, emotional valence, social memory (remembering other players). Include mem0, Letta (MemGPT), Memoria, SNAP, and any system that specifically targets persistent identity across sessions. What's the state of the art in making agents "remember" who they are?

3. **Structured vs. Prose Interfaces for Agents**: Research on how game state representation affects agent performance. GMCP/MSDP protocols, MCP (Model Context Protocol), JSON-RPC for game APIs, object trees, tile-based representations. When does structured state outperform prose parsing for LLM agents?

4. **LLM Agents in Interactive Fiction / Text Games**: Jericho, TextWorld, ScienceWorld, and other benchmarks. What are the known failure modes? What's the current SOTA on these benchmarks? How do agents handle spatial reasoning, disambiguation, and long-horizon planning?

5. **Our Positioning**: Given the above, where would a MUD with persistent narrative memory agents fit in the research landscape? What's genuinely novel vs. what's been done? What are the most relevant prior works we MUST cite?

For each paper/system referenced, include:
- Title, authors, venue/year (if known)
- One-sentence summary of contribution
- Relevance to our work (high/medium/low)

Format as a structured research report suitable for forming the Related Work section of an academic paper. Prioritize papers from 2023-2026 but include foundational work as needed.
# Dark Pawns: A Foundation for Narrative Memory and Human-AI Coexistence in Multi-User Dungeons

*Disclaimer: The following architectural models, hardware logistics, and theoretical frameworks are presented for informational purposes in the context of academic game design research and do not constitute production software deployment or financial advice.*

## Executive Summary

The integration of Large Language Models (LLMs) into interactive digital environments represents a paradigm shift from traditional finite-state game design toward dynamic, open-ended virtual ecosystems. This report details the state of the art in this rapidly evolving field to establish the academic foundation for *Dark Pawns*, a novel Multi-User Dungeon (MUD) featuring agents with emotionally valenced, autobiographical memory. The following core areas summarize the current landscape and outline our systemic response to its limitations:

*   **Mixed Human-AI Game Environments:** Academic focus has transitioned from isolated puzzle-solvers to models coexisting in shared social worlds. Modern evaluation frameworks such as TextArena and CALYPSO assess dynamic social skills and cooperation. However, when deployed as continuous participants, agents rapidly exhaust their context windows, exposing critical flaws in persistent identity and long-term goal-oriented behavior.
*   **Agent Memory Systems in Games:** Standard Retrieval-Augmented Generation (RAG) and raw context-window scaling are insufficient for game agents, failing to capture temporal relationships and emotional weight. The state of the art has progressed from operating-system-inspired memory management (like Letta) toward sophisticated Agentic Memory Graphs (like Memoria and mem0) that decouple memory from the context window, dramatically reducing token costs while enabling true long-term persona retention.
*   **Structured vs. Prose Interfaces:** For AI agents, structured data strictly outperforms natural language prose parsing. Representing the world via schemas such as JSON-RPC, GMCP, or the Model Context Protocol (MCP) prevents the "world-update problem," mitigating severe architectural drift and spatial hallucination by explicitly defining object hierarchies and physical boundaries.
*   **LLM Agents in Interactive Fiction:** Despite massive parameter scaling, frontier LLMs still fail catastrophically at multi-hop spatial reasoning and long-horizon planning in interactive text environments. Benchmarks like ScienceWorld and TALES prove that without heavy environmental scaffolding or structured spatial injection, models remain remarkably inept compared to basic human baselines.
*   **Our Positioning:** *Dark Pawns* introduces a genuinely novel solution to agent continuity. Rather than forcing the AI client to manage its own complex memory loop, *Dark Pawns* utilizes a server-hosted, zero-setup, emotionally valenced autobiographical memory system natively embedded within the MUD engine. This architecture offloads cognitive overhead, solving the problem of context amnesia while achieving cross-referenced social persistence in real-time multiplayer spaces.

The integration of autonomous artificial intelligence into shared interactive spaces represents a paradigm shift in game design. Historically, game AI relied on finite state machines or behavior trees, offering predictable but rigid interactions. With the advent of Large Language Models (LLMs), developers gained the ability to create agents capable of open-domain conversation and dynamic reasoning. However, as these agents are deployed into persistent virtual worlds, critical challenges have emerged regarding memory decay, architectural integration, and behavioral believability. This report surveys the state-of-the-art literature across mixed human-AI game environments, agentic memory architectures, interface representations, and text-based evaluation benchmarks. It is designed to situate *Dark Pawns*—a novel Multi-User Dungeon (MUD) featuring agents with emotionally valenced, autobiographical memory—within the broader landscape of Artificial Intelligence and Interactive Digital Entertainment (AIIDE) targeting the 2027 academic horizon.

## 1. Mixed Human-AI Game Environments

The trajectory of academic research in game AI has shifted dramatically from evaluating models in isolated, single-player vacuums to assessing them in dynamic ecosystems where they must coexist, cooperate, or compete with humans. This transition from "AI as a solver" to "AI as a participant" introduces chaotic variables—primarily the unpredictability of human actors and the necessity for the agent to maintain a coherent social presence over time.

### Core Literature: Human-AI Ecosystems and Evaluation

The following systems and benchmarks represent the foundational and contemporary attempts to measure and implement LLMs in shared or highly complex interactive environments:

*   **Title:** TextArena: A Collection of Competitive Text-Based Games for Language Model Evaluation and Reinforcement Learning
    *   **Authors:** Guertler, L., et al.
    *   **Venue/Year:** arXiv, 2025
    *   **Summary:** Introduces an open-source framework of over 57 competitive text-based games to evaluate LLMs against humans and other models using real-time TrueSkill (a Bayesian skill rating system originally developed for Xbox Live matchmaking) scoring to measure dynamic social skills like negotiation and deception. [cite: 1, 2]
    *   **Relevance:** High. TextArena provides the evaluation methodology (model-to-human interaction in text games) that *Dark Pawns* can utilize to measure the effectiveness of its social memory systems.

*   **Title:** lmgame-Bench: How Good are LLMs at Playing Games?
    *   **Authors:** Hu, L., et al.
    *   **Venue/Year:** arXiv, 2025
    *   **Summary:** A comprehensive evaluation suite that transforms real video games into contamination-robust benchmarks using lightweight scaffolds that decouple visual perception from an LLM's high-level planning and strategic reasoning. [cite: 3, 4]
    *   **Relevance:** Medium. While visually focused, its underlying philosophy of using specific scaffolding to isolate reasoning capabilities informs how *Dark Pawns* should pass environmental data to its agents.

*   **Title:** CALYPSO: LLMs as Dungeon Masters' Assistants
    *   **Authors:** Zhu, A., et al.
    *   **Venue/Year:** AIIDE, 2023
    *   **Summary:** Presents an LLM-powered interface that assists human Dungeon Masters in Dungeons & Dragons by generating context-aware game descriptions and brainstorming encounters while preserving human creative agency. [cite: 5, 6]
    *   **Relevance:** High. Demonstrates how humans and LLMs can synchronously co-create narratives in tabletop/text environments, establishing a baseline for cooperative human-AI storytelling.

*   **Title:** TILE / DiscoveryWorld: A Virtual Environment for Developing and Evaluating Automated Scientific Discovery Agents
    *   **Authors:** Jansen, P., et al.
    *   **Venue/Year:** NeurIPS, 2024
    *   **Summary:** Introduces a multi-modal, simulated environment structured strictly as a 32x32 tile grid representing 120 long-horizon scientific discovery tasks to benchmark an agent's ability to autonomously hypothesize, experiment, and analyze results. [cite: 7, 8]
    *   **Relevance:** Medium. Validates the use of complex, virtual sandbox environments and explicit tile-based constraints as rigorous academic benchmarks for agentic reasoning and long-term planning.

*   **Title:** AgentBench: Evaluating LLMs as Agents
    *   **Authors:** Liu, Z., et al.
    *   **Venue/Year:** ICLR, 2024
    *   **Summary:** A systematic benchmark evaluating LLM-as-Agent capabilities across eight distinct environments, revealing that poor long-term reasoning and instruction-following are the main obstacles to usable LLM agents. [cite: 9, 10]
    *   **Relevance:** High. Highlights the exact failure modes (long-term reasoning, memory loss) that the *Dark Pawns* architecture is explicitly designed to solve.

*   **Title:** EllyMUD / MUD-MCP
    *   **Authors:** Ellyseum (EllyMUD) / Nexlen (MUD-MCP)
    *   **Venue/Year:** GitHub Open Source, 2025/2026
    *   **Summary:** Open-source Multi-User Dungeon implementations that integrate the Model Context Protocol (MCP) to allow AI agents to autonomously connect, parse structured game state, and execute commands within the MUD environment. [cite: 11, 12]
    *   **Relevance:** High. These systems serve as the direct technical predecessors to *Dark Pawns*, proving the viability of using modern API protocols to treat LLMs as standard MUD clients.

### Synthesis and Implications for Dark Pawns

The evolution from solitary puzzle-solving (as seen in early benchmarks) to the dynamic, multi-agent scenarios of TextArena [cite: 1] and CALYPSO [cite: 6] illustrates a critical shift: the "intelligence" of an agent is increasingly measured by its social competence and adaptability. In environments where LLM-based agents and humans coexist, standard static benchmarks fail. As Guertler et al. demonstrate with TextArena, traditional benchmarks like Massive Multitask Language Understanding (MMLU) are nearing saturation (where frontier models like OpenAI's o3 now achieve a 92.9% state-of-the-art score, effectively saturating the benchmark and eliminating discriminative power), necessitating environments that test soft skills such as theory of mind, negotiation, and long-term bluffing [cite: 1, 13, 14]. 

However, introducing agents into multiplayer spaces exposes significant architectural weaknesses. AgentBench [cite: 9] clearly delineates a severe performance gap between an LLM's ability to answer static questions and its ability to maintain coherent, goal-oriented behavior over extended sessions. When an agent is forced to interact with humans over hours or days, its context window rapidly fills, leading to catastrophic forgetting. 

Implementations like EllyMUD and MUD-MCP have solved the *connectivity* problem, utilizing Anthropic's Model Context Protocol (MCP) to securely network LLMs into the MUD ecosystem as autonomous clients [cite: 11, 12]. Yet, these implementations largely leave the *cognitive* problem unsolved; they provide the interface but not the persistent identity. *Dark Pawns* bridges this gap. By utilizing a shared game world where the engine itself hosts the agent's memory, *Dark Pawns* moves beyond the transient "Agent Mode" interactions of MUD-MCP [cite: 12] and establishes an environment where long-term social consequences are mechanically enforced.

## 2. Agent Memory Systems in Games

The core technological hurdle for autonomous agents in persistent games is memory. Without a robust memory architecture, agents suffer from "conversational amnesia," acting as ephemeral entities that reset after their context windows are exhausted [cite: 15, 16]. Traditional Retrieval-Augmented Generation (RAG)—where past interactions are embedded as vectors and retrieved based on semantic similarity—is highly efficient but fails at complex reasoning. Standard RAG (often termed "System-1" retrieval) struggles to capture temporal relationships, causal chains, and the emotional weight of past events [cite: 17, 18]. To make agents "remember who they are," the state-of-the-art has shifted toward structured, graph-based, and agentic memory systems.

### Core Literature: State-of-the-Art Agent Memory

*   **Title:** Letta (formerly MemGPT): Towards LLMs as Operating Systems
    *   **Authors:** Packer, C., Wooders, S., et al. (Letta)
    *   **Venue/Year:** arXiv / Open Source, 2024/2025
    *   **Summary:** Implements a memory hierarchy inspired by operating systems, enabling agents to autonomously manage their context window via tool calls to move data between fast, in-context "core memory" and slow, external "archival memory." [cite: 19, 20]
    *   **Relevance:** High. Provides the foundational mechanical architecture for context window management and the concept of self-editing "memory blocks" for persona retention.

*   **Title:** Mem0: The Memory Layer for Personalized AI
    *   **Authors:** Yadav, D., Singh, T., et al.
    *   **Venue/Year:** Open Source / Y Combinator, 2024/2025
    *   **Summary:** An open-source, hybrid memory orchestration layer combining vector databases, knowledge graphs, and key-value stores to continuously adapt to user interactions without unbounded context growth. [cite: 21, 22, 23]
    *   **Relevance:** High. Provides an enterprise-grade comparison point for decoupling memory from the LLM context, achieving a 93% reduction in token costs and 91% lower latency compared to full-context approaches. [cite: 23, 24]

*   **Title:** Memoria: A Scalable Agentic Memory Framework for Personalized Conversational AI
    *   **Authors:** Sarin, S., et al.
    *   **Venue/Year:** AIMLSystems, 2025
    *   **Summary:** A modular framework combining dynamic session-level summarization with a weighted Knowledge Graph to incrementally capture user traits, preferences, and behavioral patterns for long-term dialogue coherence. [cite: 25]
    *   **Relevance:** High. Validates the necessity of combining semantic summarization with a structured graph to overcome the limitations of standard RAG systems.

*   **Title:** Evaluating Very Long-Term Conversational Memory of LLM Agents (LoCoMo)
    *   **Authors:** Maharana, A., et al. (Snap Inc.)
    *   **Venue/Year:** ACL, 2024
    *   **Summary:** Introduces a benchmark and dataset for evaluating true long-term memory across up to 35 sessions, proving that current long-context and RAG models severely lag behind humans in multi-hop and temporal reasoning. [cite: 26]
    *   **Relevance:** High. Defines the evaluation criteria for long-term memory and highlights the specific temporal reasoning failures that *Dark Pawns* must overcome.

*   **Title:** TraceMem: Weaving Narrative Memory Schemata from User Conversational Traces
    *   **Authors:** Shu, Y., et al.
    *   **Venue/Year:** arXiv, 2026
    *   **Summary:** A cognitively inspired framework that uses hierarchical clustering to transform fragmented dialogue histories into coherent, self-evolving narrative memory schemata, mimicking human synaptic and systems memory consolidation. [cite: 27, 28]
    *   **Relevance:** High. Provides a direct theoretical model for structuring autobiographical memory as an evolving narrative rather than a static database.

*   **Title:** From Simulated Empathy to Structural Attunement: Realtime Editable Memory Topology (REMT)
    *   **Authors:** Albanese, J.
    *   **Venue/Year:** Frontiers in Artificial Intelligence, 2026
    *   **Summary:** Introduces an architectural framework for persistent autobiographical memory organized as an evolving graph of emotionally valenced nodes, formalized through synthetic neuroplasticity and a Mood Index that modulates retrieval bias. [cite: 29, 30]
    *   **Relevance:** Critical. REMT is the exact theoretical analog to the *Dark Pawns* memory system, demonstrating how emotional valence can dynamically alter the topology of an agent's memory graph to influence long-term behavior.

### Synthesis and Implications for Dark Pawns

The literature clearly delineates three distinct epochs of LLM memory design. The first epoch relied on expanding the raw context window or using basic RAG, which the LoCoMo benchmark proved was insufficient for maintaining a coherent persona over dozens of sessions due to hallucinations and an inability to perform temporal reasoning [cite: 26]. The second epoch, popularized by Letta (MemGPT), treats the LLM like an operating system. Letta provides agents with explicit memory blocks (e.g., "Persona" and "Human" blocks) that live in the context window (RAM), allowing the agent to use tool calls to page data in and out of external databases (Disk) [cite: 19, 31]. While highly effective for functional tasks, this approach relies entirely on the LLM's own, often flawed, judgment to decide what is worth remembering, incurring massive token costs for continuous self-reflection. For instance, a 50-step agent workflow maintaining a 20K token context per step consumes 1 million tokens per task execution—equating to roughly $2.50 in input costs alone at standard GPT-4o pricing, which becomes financially devastating at scale [cite: 16, 20].




The current, third epoch focuses on *Agentic Memory Graphs* and *Autobiographical Memory*. Frameworks like Memoria [cite: 25], Mem0 [cite: 21], and TraceMem [cite: 27] recognize that human memory is not a searchable database but a continuously consolidating narrative. TraceMem's approach of using clustering to uncover latent narrative threads mimics human synaptic consolidation, providing a mechanism for agents to form a holistic sense of "self" [cite: 27, 32]. 

*Dark Pawns* is perfectly aligned with the bleeding edge of this third epoch, specifically drawing parallels to Albanese's Realtime Editable Memory Topology (REMT) [cite: 29]. REMT argues that true persona fidelity requires **synthetic neuroplasticity**. 
1.  **Technical Definition:** Synthetic neuroplasticity is the programmatic algorithm by which a graph database dynamically adjusts the traversal weights (edge strengths) and node salience based on the calculated emotional intensity of a recorded event. 
2.  **Analogy:** Imagine a physical journal where emotionally intense memories are written in bold, glowing ink that naturally causes the reader's eye to gravitate toward them first, while mundane memories fade to a light gray, requiring conscious effort to read. 
3.  **Game Relevance:** In *Dark Pawns*, this means the MUD server does not just store chat logs; it mechanically calculates the emotional valence of a player's action and permanently thickens the graph edge between that player and the agent. When the agent is prompted, the server natively prioritizes traversing these thickened edges, ensuring the agent retrieves a major emotional event instantly without wasting context tokens on mundane greetings [cite: 30, 33].

By implementing an emotionally valenced, socially cross-referenced memory system natively on the MUD server, *Dark Pawns* eliminates the computational overhead of the Letta (MemGPT) agent-loop paradigm. Instead of the agent constantly polling its own memory, the game engine directly curates the agent's context based on the emotional salience and spatial proximity of the current interaction, representing a highly novel, zero-setup integration of cognitive science into multiplayer game architecture.

*(Note regarding data limitations: Precise real-time benchmarks comparing REMT's token efficiency against Letta in production environments are currently unavailable, as REMT is a theoretical framework pending empirical validation in future studies [cite: 29]. Therefore, Dark Pawns will serve as a crucial empirical testbed for these topological memory theories.)*

## 3. Structured vs. Prose Interfaces for Agents

A fundamental debate in AI-driven interactive environments is how to best communicate game state to the agent. In classic interactive fiction (IF), the agent reads natural language text ("You are in a dark room. There is a sword here.") and outputs text commands ("take sword"). However, as environments scale in complexity, relying purely on natural language prose introduces severe reliability issues. 

### Core Literature: Interface Representation and Parsing

*   **Title:** Model Context Protocol (MCP)
    *   **Authors:** Anthropic (Open Standard)
    *   **Venue/Year:** Open Source, 2024
    *   **Summary:** An open standard and framework that provides a universal, bidirectional interface (typically via JSON-RPC, a stateless, light-weight remote procedure call protocol) connecting AI assistants to external data sources, eliminating the need for fragmented, custom tool integrations. [cite: 34, 35]
    *   **Relevance:** High. MCP has rapidly become the industry standard for granting LLMs structured access to external environments and filesystems, heavily utilized by MUD-MCP.

*   **Title:** PAYADOR: A Minimalist Approach to Grounding Language Models on Structured Data for Interactive Storytelling and Role-playing Games
    *   **Authors:** Góngora, S., et al.
    *   **Venue/Year:** ICCC, 2024
    *   **Summary:** Addresses the "world-update problem" by shifting focus from parsing raw textual actions to using LLMs grounded in a minimal, structured representation (JSON/dictionaries) to predict and output exact world state changes. [cite: 36, 37, 38]
    *   **Relevance:** High. Empirically demonstrates that LLMs hallucinate less and maintain better world consistency when reading and writing structured data rather than pure narrative prose.

*   **Title:** Semia: Auditing Agent Skills via Constraint-Guided Representation Synthesis
    *   **Authors:** Wen, H., et al.
    *   **Venue/Year:** arXiv, 2026
    *   **Summary:** A static analyzer that uses an LLM-driven synthesis loop to translate hybrid prose-code agent skills into a structured Datalog (a declarative logic programming language often used as a query language for deductive databases) fact base (SDL) to perform deterministic security and reachability audits. [cite: 39, 40]
    *   **Relevance:** Medium. Proves that translating vague prose into strict, structured relational languages allows for deterministic, reliable logic operations that LLMs otherwise fail to execute safely.

*   **Title:** ReasonPlanner: Enhancing Autonomous Planning in Dynamic Environments with Temporal Knowledge Graphs and LLMs
    *   **Authors:** Dinh, et al.
    *   **Venue/Year:** arXiv, 2024
    *   **Summary:** Proposes an agent that builds a Temporal Knowledge Graph to serve as its world model, allowing it to accurately plan hypothetical trajectories in the ScienceWorld benchmark, vastly outperforming pure prompt-based methods. [cite: 41, 42]
    *   **Relevance:** Medium. Reinforces the necessity of structured graphs for long-horizon planning in text games.

*   **Title:** Structure Beats Prose: Specs for Coding Agents That Actually Work
    *   **Authors:** van Egmond, S.
    *   **Venue/Year:** Developer Publications, 2026
    *   **Summary:** An industry analysis demonstrating that structured, parseable schemas for AI agents dramatically reduce architectural drift and hallucination compared to natural language specifications. [cite: 43, 44]
    *   **Relevance:** High. Provides strong, recent empirical backing for the use of structured data (like object trees) over prose to guide agent behavior.

### Synthesis and Implications for Dark Pawns

The overarching consensus in the literature is that for autonomous agents, **structure beats prose** [cite: 43, 44]. In natural language environments, agents frequently succumb to the "world-update problem," as identified by Góngora et al. in the PAYADOR framework [cite: 36, 38]. When an agent reads prose, it must implicitly track the state of the world (e.g., remembering that a door is locked or an item is in its inventory). LLMs are notoriously poor at this implicit state tracking, often hallucinating items (like suddenly possessing a bazooka) or ignoring physical constraints [cite: 36, 38]. This happens at an architectural level because transformer models suffer from token attention dilution over long contexts; as the sequence grows, the self-attention mechanism struggles to persistently weigh older, implicitly tracked state changes against newer tokens. Furthermore, the standard transformer architecture lacks explicit relational grounding—it predicts the next statistically likely token rather than maintaining a rigid internal database of entity relationships and physical constraints.

To circumvent this, systems like ReasonPlanner force the LLM to maintain a Temporal Knowledge Graph of the world [cite: 41], while static analyzers like Semia translate prose directly into relational Datalog facts before executing logic [cite: 39]. Structured implementations consistently prove their superiority, improving task performance by 40% to 70% without modifying any model parameters by enabling deterministic verification and eliminating assumption variance [cite: 43, 44]. 

The recent widespread adoption of the Model Context Protocol (MCP) by Anthropic formalizes this structured approach [cite: 34, 45]. By using JSON-RPC messages over standard transport layers [cite: 35], MCP allows the LLM to query the exact state of an object tree, significantly reducing token overhead and entirely bypassing the ambiguity of natural language parsing [cite: 46]. 

| Feature Category | Natural Language Prose | Structured Interfaces (JSON/MCP/GMCP) |
| :--- | :--- | :--- |
| **State Tracking** | Implicit, highly prone to "world-update" hallucinations (e.g., duplicating items). | Explicit, rigid schemas explicitly defining object trees and boolean constraints. |
| **Parsing Reliability** | Low; subject to ambiguity and LLM attention dilution over long context sequences. | Deterministic; allows exact mechanical verification of what the agent observes. |
| **Latency/Overhead** | High; requires vast token blocks to describe complex environments textually. | Low; minimal token expenditure using constrained KV pairs or Datalog facts. |
| **Target Output** | Open-ended narrative requiring regex extraction. | Machine-readable API calls mapping directly to engine functions. |

For *Dark Pawns*, implementing structured interfaces (such as GMCP, *Generic Mud Communication Protocol*, and MSDP, *Mud Server Data Protocol*, which are out-of-band JSON/telnet protocols native to MUDs for transmitting structured data, or the modern MCP) is not just an optimization; it is a prerequisite for agent sanity. While humans will interact with the MUD via prose, the AI agents must receive the world state as an object tree. This dual-interface architecture ensures the agents can accurately navigate spatial reasoning and logic puzzles without the cognitive burden of natural language extraction, allowing their limited context windows to be reserved entirely for social and emotional reasoning.

## 4. LLM Agents in Interactive Fiction / Text Games

Interactive Fiction (IF) and text-based games have historically served as the ultimate litmus test for natural language understanding and sequential decision-making. Unlike physical robotics, IF provides a safe, contained environment where an agent's ability to reason, explore, and manipulate state can be rigorously quantified.

### Core Literature: Text Game Benchmarks

*   **Title:** Interactive Fiction Games: A Colossal Adventure (Jericho)
    *   **Authors:** Hausknecht, M., et al. (Microsoft Research)
    *   **Venue/Year:** AAAI, 2020
    *   **Summary:** Introduces Jericho, an evaluation framework and learning environment for reinforcement learning agents across 56 classic parser-based interactive fiction games (like Zork), exposing action templates and vocabulary to make action spaces tractable. [cite: 47, 48]
    *   **Relevance:** High. Serves as the fundamental bedrock for standardizing text-based game evaluation, defining the baseline challenges of combinatoric action spaces and partial observability.

*   **Title:** TextWorld: A Learning Environment for Text-based Games
    *   **Authors:** Côté, M.-A., et al.
    *   **Venue/Year:** Workshop on Computer Games / CGW, 2018
    *   **Summary:** A foundational sandbox learning environment that procedurally generates parser-based IF games to train and evaluate reinforcement learning agents on combinatorial action spaces. [cite: 49, 50]
    *   **Relevance:** Medium. The foundational paper establishing text games as a legitimate benchmark for AI, defining the baseline challenges of partial observability and sparse rewards.

*   **Title:** ScienceWorld: Is your Agent Smarter than a 5th Grader?
    *   **Authors:** Wang, R., et al.
    *   **Venue/Year:** EMNLP, 2022
    *   **Summary:** An interactive text environment featuring 30 tasks across elementary science topics, proving that while LLMs excel at static QA, they fail to apply scientific concepts in grounded, multi-step interactive simulations. [cite: 51, 52]
    *   **Relevance:** High. Exposes the severe gap between an LLM's "knowledge" and its "interactive reasoning" capabilities. 

*   **Title:** TALES: A Unified Benchmark for LLM Agents in Text-Adventure Game Environments
    *   **Authors:** Liu, et al. / Authors of TALES
    *   **Venue/Year:** arXiv / OpenReview, 2025/2026
    *   **Summary:** Unifies existing frameworks (Jericho, TextWorld, ScienceWorld, ALFWorld—an aligned environment combining TextWorld and embodied AI tasks) into a comprehensive gauntlet that evaluates an agent's deductive, inductive, spatial, and grounded reasoning capabilities. [cite: 53]
    *   **Relevance:** High. The most recent and exhaustive benchmark, demonstrating that even state-of-the-art frontier models fail at zero-shot composite reasoning in text adventures without heavy scaffolding.

### Synthesis and Implications for Dark Pawns

Despite the massive leap in parameter counts and pre-training data over the last few years, the TALES benchmark [cite: 53] and ScienceWorld [cite: 51] reveal a stark reality: LLMs are still remarkably inept at playing text-based games and fall drastically short of human baselines. For example, in **ScienceWorld**, human baselines achieve an 80th-percentile aggregate score of approximately 0.94 (94%), whereas frontier models lag significantly, especially in multi-step application [cite: 54]. In the **TALES** benchmark, state-of-the-art LLM-driven agents fail to achieve a 15% completion rate on games designed for human enjoyment [cite: 53]. Similarly, in the **TILE/DiscoveryWorld** simulation, expert humans achieve a 66% completion rate, while the best baseline agents (like REACT) complete only 38% of easy tasks and a dismal 18% of challenge tasks [cite: 55].

The known failure modes are heavily concentrated in **spatial reasoning** and **long-horizon planning**. Agents suffer from "affordance blindness," struggling to distinguish between actionable objects and set dressing, and they frequently enter endless loops of repeating failed commands due to an inability to disambiguate identical objects (e.g., "open door" when there are three doors) [cite: 56]. Furthermore, because IF games are partially observable Markov decision processes (POMDPs) [cite: 57], agents must rely on their context window to remember the map layout. Once the layout scrolls out of context, the agent becomes lost. 

The current State of the Art (SOTA) approaches on these benchmarks rely heavily on external scaffolding. Agents that dynamically build and query Knowledge Graphs (like ReasonPlanner [cite: 41]) or utilize extensive Chain-of-Thought prompting generally achieve the highest scores, but still fall drastically short of human baselines in complex environments [cite: 52, 58]. 

For *Dark Pawns*, these findings dictate that relying on the LLM to autonomously map the MUD through trial and error is a guaranteed failure path. The game engine must provide the agent with immediate, structured spatial awareness (e.g., tile-based adjacent room data or absolute coordinate tracking) to bypass the LLM's inherent spatial reasoning deficits.

## 5. Our Positioning: Dark Pawns in the Research Landscape

As AIIDE 2027 approaches, the research community is moving away from testing whether an LLM *can* act as an agent, toward understanding *how* communities of agents and humans can sustainably interact over months or years.

### Must-Cite Prior Works

To properly position *Dark Pawns*, any academic publication must heavily cite the following foundational pillars:
1.  **Generative Agents (Park et al., 2023):** The absolute foundational baseline for simulating daily lives and emergent social behavior among LLM agents in a sandbox. [cite: 59, 60]
2.  **Letta / MemGPT (Packer et al., 2024):** The current mechanical industry standard for LLM memory management and tiered storage. [cite: 19, 20]
3.  **REMT (Albanese, 2026):** The closest theoretical counterpart to the *Dark Pawns* emotional memory architecture. [cite: 29, 30]
4.  **PAYADOR (Góngora et al., 2024):** For establishing the absolute necessity of structured state representations in IF to solve the world-update problem. [cite: 36, 38]
5.  **CALYPSO (Zhu et al., 2023):** For establishing the framework of human-AI coexistence in tabletop and role-playing spaces. [cite: 5, 6]

### Novelty and Positioning

*Dark Pawns* represents a significant evolutionary step beyond existing paradigms. While Park et al.'s *Generative Agents* [cite: 59] popularized the concept of agents living persistent lives, those agents existed in a closed simulation, insulated from human intervention and real-time multiplayer complexities. Conversely, environments that *do* mix humans and AI, such as CALYPSO [cite: 6] or MUD-MCP implementations [cite: 12], treat the AI as an external tool or a transient client operating via a localized script. 

**What has been done:**
*   Connecting LLMs to MUDs via MCP and structured JSON interfaces (EllyMUD, MUD-MCP).
*   Giving agents basic persistent memory via local vector databases or self-editing context blocks (Letta/MemGPT).
*   Evaluating agent reasoning in isolated text games (ScienceWorld, TextWorld, TALES).

**What is genuinely novel in *Dark Pawns*:**
The defining innovation of *Dark Pawns* is the inversion of the memory hosting architecture. Currently, agent developers must build complex, client-side orchestration layers (like LangChain or Letta) to manage an agent's memory, which is highly computationally expensive and prone to context failure [cite: 20, 61]. *Dark Pawns* pioneers a **server-hosted, zero-setup, emotionally valenced autobiographical memory system** natively embedded within the MUD engine itself. 

By writing the game in Go with a custom Lua scripting engine, the MUD tracks the social interactions, computes the emotional valence of events, and maintains the memory graph for the agent on the server side. When the agent is pinged for an action, the server directly injects a highly curated, topologically relevant memory state into the prompt. This seamlessly operationalizes Albanese's theoretical REMT framework [cite: 29] into a playable, real-time multiplayer game. Consequently, *Dark Pawns* eliminates the need for agents to act as their own database managers, freeing their cognitive overhead entirely for deep, socially cross-referenced roleplay alongside human players.

### Reality Check: Graph Memory and Emotion at Play

To ground this theoretical model in reality, consider the following hypothetical case study within *Dark Pawns*: A human player, "Artemis," cooperates with an AI merchant agent, "Silas," for three weeks, successfully trading rare goods. The MUD server maintains this ongoing interaction as positive, low-valence nodes in Silas's memory graph. 

One day, Artemis steals a vital artifact from Silas's shop. In a standard RAG framework, this theft might simply be appended to a text log and rapidly pushed out of the context window by mundane follow-up greetings, causing Silas to forget the betrayal within hours. In *Dark Pawns*, the game engine detects the "theft" event state change and calculates its negative emotional valence as extreme. The Go backend mathematically updates the memory graph, heavily weighting the "Artemis - Betrayer" edge connection. Two months later, when Artemis approaches Silas, the server performs a localized graph traversal. Because the "betrayal" edge is permanently thickened due to its high emotional valence, it is instantly retrieved and injected into the prompt, overriding all recent mundane greetings. Silas naturally and predictably refuses service, generating narrative consequences that require zero prompt engineering or context-window self-editing from the LLM client.

### Engineering Logistics and Asynchronous Concurrency

Hosting a complex memory graph and coordinating LLM generation on the backend introduces severe logistical considerations. Relying on an external API (like OpenAI or Anthropic) for agent responses inherently incurs latency ranging from 500ms to over 5 seconds. If *Dark Pawns* executed these requests synchronously, every time an AI agent spoke, the entire server tick-rate would freeze, causing game-blocking latency for all human players.

To circumvent this, *Dark Pawns* leverages Go's native concurrency model (goroutines) combined with the isolated state environments of its custom Lua scripting engine. 

1.  **The Event Trigger:** When an in-game event requires an AI response (e.g., a player speaking to the agent), the primary Go game loop captures the event and immediately passes it to an asynchronous job queue.
2.  **Concurrent Memory Retrieval:** A dedicated goroutine pulls the request, queries the server's internal graph database for relevant context (the REMT traversal), and compiles the structured JSON prompt.
3.  **Non-Blocking API Call:** The goroutine fires the API request to the external LLM provider. The main game loop continues to tick at 60Hz, entirely unbothered by the waiting process.
4.  **Lua State Injection:** Upon receiving the response, the goroutine securely invokes a restricted, agent-specific Lua state to translate the LLM's structured JSON output into explicit game engine commands (e.g., moving rooms, attacking, or broadcasting dialogue).
5.  **Hardware Overhead:** Because memory curation and graph traversal are executed efficiently in compiled Go rather than through tokenized LLM loops, the hardware requirement for the server remains exceptionally low—a standard mid-tier cloud instance (e.g., 4 vCPUs, 16GB RAM) can easily manage the concurrent memory graphs for hundreds of agents, while inference costs scale linearly strictly based on active conversational engagement.




**Sources:**
1. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE_xgmUMX9uNzQVvhkZlwPSUcfGlDib3ut0RlxwVpK2SGrvAM3WgLI7O0wAgzCFOVhFruKAdudA4jfNIRd0acg1MfJZsWEQ73n_aPoZ7kVZ2S-i_SJf5SNN0Q==)
2. [researchgate.net](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFrRjtn3W7fEX5XR1w-SzV6Bdkkrmw19nEm5HMcg-hBEnCrqb4lqDuaa8kG1UQVG_RvsVKAwBHur5NsvlP-nSobiITQtUG0rl5DaVhZPqtgwRsSUju2emQf_XL47QtyWT2X5mHSnKUZ7v-Wj7JvyiTy6YE=)
3. [emergentmind.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHmF7bW6Ua7FTcFHqOkLCex-cc-JrgKEiJNOttkGPMPl5karG4gjtUuXgptp353BGQW0OfWcd-70-YQys1th3nahsqrFgEr7qDS9lVf3sIbRJ751A4h7T7NTIxTpAFM9o8n8iz1NmQ=)
4. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHBCTTIYp6kCOJ3X55q2yua1SkvkNqfsFo8pDvfoFXk9UbrGtb1wnsQ3UZQksi1mDO4t2fIP722HxDalu8ViCZLm9M_3lrXS1x_laweYVjxegALOy8_iX8=)
5. [andrewhead.info](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEnX_VQz5Rs5A2sv5HnyPG9FAgbKv7CtWPTqbSehvqTg8F-1DJmoxApLTg5aRcHccKV9_ZSTft2P7Vx6YXULdnGndBowZbZcaaFAdYwJWDfqANG70x1n22BnaA-WdpU_0FhS7Cl)
6. [upenn.edu](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQH_il7BZqLtAfZdo_Kh6qhKuinlcovGPHKv-5Q8YJTNxBy6s3LiUO6xqFk9xQIwlmlwxOBNnjgj7uOrHU4qXnusO1Lp0-_4rb99sqNgKHoO9knIdTRq-g5xb2W5a2XUBtMWW7JZ1HkxUbJivK-w8phw1BjaDnGsM6b6mg==)
7. [neurips.cc](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE_sGoDot-gi_JaDtVu_8jyFn3aloen7cr6JQzdw5ga1uGCnNuLH9Y18nx7KvI68iqEmcotQqI_9rxU2F47ophhe0705zcXwJIAudr24Vtcv7VaYhO2BSq_1LJNUz1b9G3BAy_62Pn7kZzQHndSk6IQR4Zbg0aVDi7mV46vkL-BNRYof9ulHWa1PFYZQLfSaoyxzVMsizBlHL6zBXuNP1YJOhaQsmdq4CdNW0GbdgXtMUMbfJYg5FqDGLLWbxOp)
8. [neurips.cc](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFoupgJ31PPb_GKGHSj4C6j4NKzh9jl0d0mNPvcWe3al4sTGBAWoLeDzEOhLLuYU7kh1_zTJtozOidX2duVQfWiMhdftqQh-IZKR0voOZ5CbkSEbqqwNLRuYtIb6KirH4z-O6sHTxUrFk4BDjRRGvcIFV57GzaUdv8MlrijptL6u8Y3MykgqcNPefv47FUfJt3PuvPxvMGjbVcxoU17IIqDcHbcwHNk2FWGO1wA4mH0WegRapSJs6FRoF8=)
9. [takara.ai](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFozsfp0y-3TAh_JVdK4m-Frrp2EiCngcDOSocHLucqfH0khKoUXQNrKc8FS6vSnfpkzx70CQn07f85orNvRpyLvm5G0bnKZS-UweZdi8S70bGXNedegsrPCQ==)
10. [iclr.cc](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFVPyP4CO2KR___JTuVIZCNNGX2scdogqTqFav52XSFD9FJvMBkQjmKB0UhbZOQkqA8EfZ_Cm44EunUyXDcsHLJcxKvUhuxQzdOOL8lyT1EjSFfXUVDYa1OzJpP9qr2YreO_XEYTFQ4k_no4HBkdBRMTCMCy1dfM4o5wwPxZzwxwg1W06knXXHtDQ213CJxsnpSZvcbfEVdAXwd5WLBuJIKMWGB)
11. [github.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEOsm0-Pw7M6gCsrJiFsLpfl-v-QdMxJKsFhOhJJG0VnYndADJMrWMsRLot4qeW2-EZcR0yqM4xJSSCZxf96r1Fra9vAqA0oJX4bTCUBHKgfSf4hBQavHighg==)
12. [github.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEuqQiZwyxDJ0aD0oPMvGMkCF1bWLqHQvn-VQSfooPJN7RM2gehoWpwhqmbb6nJthG-AYKGZRxlyFz0KLdI7JjJ-AjNt05qsd6X1ncENLXm8CoGJAMTtPI=)
13. [aimodels.fyi](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGOEhjccwKWgLLRZv3SnivRYr-6mmsT8EUZ9grOjG4y_IpyHHmuWy3ichr9GSFcq11ovSpopaldRKB-zCaVCRmVebmH4Lf9KtoET6MSDkPousL-7pHtJFOJL0dDAKxE5jUYNLq2Hw==)
14. [codesota.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGy3YFPN3edcrc60OwnU9y_J7ym22HA0umuVZW8C-e4_KPY5V0_3IdrF-c3MUmNMlBVciLDiSOCpmp4YXwyEsQymcr1_wJrL-0jVcfV_a0AGG8s)
15. [themoonlight.io](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG1XScTnNkXue5Q26rLITacPYXbiP5SdzpUInkinCjUapqffffosA3-qrdQQ0NQyAHB2P-9DN5THzLfukQSU5yqHOpxC-dQdcn1f-2ZM2nlUhyxFTywS6Eu3nvqmPFdWiysMduAvDzOHcFKYTBQPYeZ5u8dq__ky8J0goo8vJ6SsA5AJ3C3bU5zw7OUSlYfcqbGuXdqd8rA63s=)
16. [genmind.ch](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG5igCkJVeMhgdycmGwdxsHoPKfivi5_r1u_hj2zC-ACrnD9kMXKqWlBYJCiOvjWYK4vGZZBwH16xKtsb50FLHgJf6nsIKlynFqNUAq8EjhIWGOwWCrQTJO7VU0KJVwh4M6hcOYfm5R1DEJtCWYNHFd2sHMJuHc3vZ86HVUeswy1C5s6F-VhuGxXgboRTO6mAsE2Ta-_A==)
17. [huggingface.co](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHWHybPfhSmW6_HKW2fEPoxbu5OiuhodJMj061-XGpY6JuDreuX5FswnR4ej19C4UH_WRk3N9g94PWAVy1cLjjqmqQzqkJxe_T3TQQMf4JWae_ql6Gqf7weHsZ3ucZWeukawWPkVpgXpHWQsQcpojUM4qU9iQ==)
18. [venturebeat.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGRvApTTIQ52Dn0qCu-j4-Xp3R1SwCpXJ4WQNwew89-YI9e7Ktn922bCtPacDqRU3JpY8ykHMjpEGoP6fdBi9KYXi1mrEGFao58ZwH4MX4_xlk4VXQ9Ikl9cj9KX68Vj6JdK0tS5ub4jbF2n9nCBBbujk0-fG6KK2gt20WY3uKJOBtEnwb8kfQga4RcSfjCcTCctnaH4Iz8)
19. [letta.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE9M26wMkQk1yBJpPBPzEbv8LHvck_afjzB_UaWKUQpEA_K7v9qdN8OjH9qQzlXUk6xfyyTB65rnkCzZpe9yLBh_vsDn84G_ZV6-FigvQSlJoFeFGfZuqqw6cb6pDI=)
20. [vectorize.io](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQH-sPxv95TnhXms-UP6_DQN4r9KbIPZWBG3B5FB7RONjGtkSHtQsya4eiDAZgDSxNvlEBooTQ8kEGlo3Zfj1horJZqKOR7grL0-pj4ZkQqvaZMzqOexGH9jEKMeLE8KaRcX)
21. [amazon.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFV7NII1kVVgLXHKR4XqFfpLrRi0F0PL95gCJN7L1ToKLsLGp7RBFCCgBUvNC10jG_8AS1zi9Z4KRLgzhdyHF-J65P2CdCGujKsdbpXWs4P0UVsryAwpX0eNxLprOLTGjM-1iDRpcu4g6aipZ0dWg83naokDYKiwLpI424BAVsCNMu0mzxGrG45RR8XdzYjiPEa37kt9H55yCVEUm8UnPUICZ-ThPQhAmLnJ71v8r6B28G5KufxNlXW3xxGkJiny6WrtNzG2XFfkABRLJ0bSvAbgc-LyfKxW8Ye0CQJli-9SV6z)
22. [ycombinator.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHMRxplABERq89kRlEnJK3sYQ3g7yU33FlmjqcECSQxQtO51Y5ParEVMCm-IBQIsFbkmvecvgLfFzXGXfhwNIj50KdUowTnnqEW7otGsGbfdTsj4ICHuN4FMY4NHIy71cI=)
23. [github.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGMfqSvk9Qw95jQDlh9yM3kuOO4IRBI3cPnyqkKc4kCVSeUX7dx63-MddR9aCUMeexHYFyNLNCr9YjQlHIiLxSyR-jmNRcV4di3mQFMyDUZTE_19tM=)
24. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG3YciJqGWDyTyS9oAwmPpQUDZ-tBL3BcskRPH_y1yTHJhdIX8NvNScJNAQUrkcJsbyarNEyLKseh4jUxsKPGsTRP73xYGl6j6tokPZOLZJzfKXrsEzHUEyYA==)
25. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG3_Szn5wt3i13r-b-NxIEXWDQAmSKDMZOwwV3nerGgUxLQaTrHRgB3HpOu15z1hBPYEsnb7hqPPUeYab9ox6VAASHfKtVP0-OemsQozYmbkgNrdZ4KGQ==)
26. [github.io](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHi31quHgk89eUMb0xKMFpoYg_o6uspcmI-Uf-xqGeopL-VvMHz9cpWE-VU3yBQJIT_6mnMZPK5ZkNigWiW3Ggrgyec2nAcpK21RpLmPfKBuklvV49ektjzdMDxhRY=)
27. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFSSXZ81uXMc1foHwzpeAEA1IjrGD1ERChRhS9XdTDgeJL0azfkz42ijQWsMSX6kOXHdHiK-gMWVBIGqnm-ODskCXKALVvUxtz9MBCBIXPhRmnqmQW1ZuzlHg==)
28. [takara.ai](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHOlFZhnAQBco7OM-cjD4WynWDjSCFjymNWbv0Aelc-N-AC9TctpESjT4IAAGTz2LImdc00LB2L6WzOqOiJJXUB_8L0T76ncij3nrbStz5ENNS7kEWCH1ibcg==)
29. [nih.gov](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEAdMTL-m3KIlyTKC5V2rmNFbruV2BLQmTymeae_Eml3dTJBuRL-_lYYNh0Y55k1_i3EnSIugUu5RdKSbyFvZ4CWboyGRHeW7NaHbANugw34iBMCfsy557tpJ9duEpU7g==)
30. [nih.gov](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGiZayTQggjI9usTjmucnv2_iJVpBchCafDsFJVn-iOqmmMCN1eyXk-th_D3bhxgVkuzxXTX7ZA7AVb8j9a6gOTUvYCP3HFdclFuTR6IdisXZWUM3u4c8e9lejcWTQE73UweSnWYGO5DQ==)
31. [letta.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEgbmrI3jERJyDXhthfxMjNFOxMTF2b2MTGPmgUzN6ue9_p7jHu7SVpTLszh4Rj7TgG0xkqvlK2A-hf2Mt-NzoDgkDau-IA6KAfgD9ykW1E9QAw7_SeTWEDpUbewI6K)
32. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE82ATq3BEEeJndMo6-SjSQ745h4EX3Epp4_6rC5ov3DppQS4ciu3xd_MzywQaLH7aL5Wr6FnLne3CLRCqocvcHmIUop_0NMDP1Pcm6xGu-PLZljL-4mg==)
33. [researchgate.net](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG-itGkNh9AMojpXBEwQ30NFRdT3Hr_7NQQIDb9Yc-qkj5rWfDGDYFL4QMcnkIlusKjj6I6FN2PVRXwCQSsFIGeg089bWCpoz6qUeEKSfXDX_JecBSNvBMDCp-lBE7SVL6sDrKF19akAH3-yb2ePl_NTC62fC9IpwAZjtolW5u-odPTc3kOyh0ZumFOcSkWz0epIyLAHNLlakKscrnDZ2Sn_uu4PQaCkRRirBAZjj2_Tp03dDtHy6wRgDTdvUmNBCDEHWt3ZStTtB725ZLafOgB6_qFmf4lOGX3Mb42LoOGD1M9byhyApffd8Svnl6zOsdN)
34. [wikipedia.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG2YN7kmbHjKDn9bBlbczeOOajk0s5ygaE_0TkGeqH2Uf560YywkObaGMCPY--91AWvAATeE0qSfiQxsL57ANG1I5O3dc6xMX8cAcwwjpaY0t359J7IothJgiDeV3uMjfg2NMY5soC4pOx3)
35. [google.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQF-7lnyeFNejbGJtPdEOpCN35t0JBj3heztWvUgQT3OdpYILfOMvqfeUdbXwfpNdr36g7sieAJFUnTc-q5veTJUlQLPSfzzEmdQ2ukmgvfB6TqIkp8MEgNFC_E3cLRKynlhYgUXmXkPZrsVqTu4HTcXAD2dHkjt)
36. [computationalcreativity.net](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQG_GTSIjT9p5c62JWE9gntEUW0VT1BJoDv9GQSNartzr-Vfpy3dop46NFs1kOVRgmoas0lVKk1v9Od_uoHa9n8JKXkf4MPvRSm--DcNRjn1X--Q1J_9BSxT2mL_O6cOX6d0axKzlaGcCN5yPF2cN4jo664XIGME2TJDGloM)
37. [github.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHUooh4plD7-Jl8nJeAc2HBSfR0zCQ0IeN3AyZVpmLOI69Op901S6w5n2a667XMsLTu0YydUgOQwvKgqf6Pdx66gzCOojpShyOf-oJyc_DmhdjU95Ok-Ut8DjvTBn053sw=)
38. [researchgate.net](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE5ewoPiTPiqJxQQndqHvL_IhflIpGTLkKjPa0m_Rn0bsKo1hms3CLYi4nD0MyQ6gD7bj5cB18xqgqYQOHSuwe2SZatLrdX-plydBy4MLYL2hPeBECh1_IOq4wb8yT_N-hwMEG_OfJWHzENi5xpVLLZcnE3lGSjDtzl7fdtNnWE1d7rAyx45TTGH9dGQAgmpld8tlJHZJzvsIGKNnt6tpeVKUPxkIQXN3PEd-k-8HghluYVSsvKGWy5EQdC-tMKMx10mpehkhbbPM_1XZd94SDz267GobzI1o-62VR_M0IMTSnoRlfUj9whneD1)
39. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE6cwKkW3WqZOa5n8fzqX0tIyMCp7Ig6hVjiiuLozzM078FXiyp2LKND75OFuM_8K4x0bUb46v0wZEr5RvRAu2KpF22w_XXm5aN9V2N6rTay9Ne1t0g_VLXHg==)
40. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFZCB-KrBQRZ6_zV8ImbNW4788Rb8PLlcS519HCujVcjQdHVowYWkVGfvXT-A1--QzYkC7WCkYwogJBq7NZGDDpIxAWQ80TxJ_8y8nM55SHkVnNpnTQMQ==)
41. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEzwzhheThZcARRh_8aSNe0h6rjeQaEYgEf2Wgf4qbC-lc7ZAMEZFT-HQmRIx9g52ELKzCN6bK5LIAyN8nfSDc9VLJ_ve3h6hsZK2lFoyeEw9zuQRCezYpvAA==)
42. [researchgate.net](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGoc3CL0SNOtnonr6Hidaot_d9z2qCcGAOvKs5fAgGkM-iNZL0kTiqbQ0zPh-fnvQOA-c49qOrhaMc7D4LSa_jNdBm-z0arwiq2lAv6ER6oM57-JcrP5X5Q3DHowNmHOZZSvkECwqomUzr0wblCgJKR84ULqE011ev8NVB7FLLIehnoWTbqzNtDXlPOE1qDC1KtuCNDAlXtFsHti1OcxlIAGg52JP6AryL0jCpGIwIASJjKZ-9D4CfsJ2jitQPtBw3kXanpBYrtJM47UPeoVjEIAHph9B4=)
43. [medium.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQE82V-8wYBgEBeZkDM8-NGINCNHSXlNekAJPO4dpYCO0T7X-5oiyoMtEzuGVmD91buMtFagluy5qup15-PEgRexK5DRep9zm4ZbWYlbaljnFItjKA_eiZmnTOQkmY8jstHpDdvL4fuoaWiuLHcltUmFkyPu6fGKgqKMmoX_-IN9LBFA1ZlFaJcc6Rn0G8BGoRYZ9LTjvTkeQZu_86--c2j-4a6aukoJ3g==)
44. [meta-intelligence.tech](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQELgnUj1nwD5t2v8YNWPOAQE92rN_LNXMhRB6bYeIOECnauIP-9MKZJALfNi5r12oKGRe03hlkGEVx6s5HJ_F4YtbcW8w_qjpfZIByDduy6Ah1xCS18H7eZCiKpEAUXMMUDfJByFPGAyRUVctssM2yK_ZdnSv_Y)
45. [anthropic.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGC3hANgeU5uMqmmx6TB6j5BXRGabxmWlb8kyIrCirhOA4nEBJ6rNmSMcZ0h9yf7WAqdqHJ0kQkflSeuNFaYil2JOSxVzsWj-fzCxrdh9RNfChwLqOJqFHOHDeghx1SaUiZFElUfon4tpx9mw==)
46. [anthropic.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEEybqVm3cSmZ4Mbh4s_04YhgC0wmCkC49bmBpbV0MV01N8FSkUUt0szeLydesh-Y3EIlwO6J-_yWybI_RWYQJ7Ov9bRSjd641ibRgzgkgDCVBZEtO-jdA6GBV00-hbrsdY1haNEx9iWSa7oQuyXyPkz4vG)
47. [microsoft.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQEoGYtPbxZi-w3YNxNhsioISTO9EPnezAvf0T40lSDPAZoYEsoz0_hhZifBAOq0cLQta2FLN3J7omYc9oQsouWR-no4P1NAxbl0VLs2t78Ra8AHMeOJ4axaLVbrpT8kjhPb6fZH5dV1zn1r3UK2FULfqL11_6kBRLGnLoXeyx_n-OM90BS6bOIQsYLiJ1wguhhSzI7rkIIjME6FU6egcJnqhBcd-BqDlkjRWC7iQorcT1_VNjeQYHxxAAolIlfeAvXaVcHmZeN9krm6gbcCxN-5_9oW8KIgRSivhmIldw==)
48. [neurips.cc](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHVk5HyJU-JQAUjCsGbXz0rtFpmeU8jNIJvDdf2LxiZYdxIVd6eBD6uqkAOrtTmZ8mOFdAKXOQDp3hx-PQa6cIyByPRbg02KWK1oUBmOekwDVZ-dQ0mCI1bGoJR)
49. [scispace.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFNNGfk8RAcUd-aOMf-tDGUrt3VoThiof4Df6IgByRsyQz18ncf4QWKXGpdoQScn4e0Lt1YaxNQzKr-LeaKfLGrYVWALH3Rw_GhJ8jaAA7cpRd_OEoy2QRVIw9GpH7PzGzMvGao_3sguOXCDAEJd8cCYdnCt8Bb034ZIF8dQciRDq6uPrEBjPnGotVwBg==)
50. [researchgate.net](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFnn08YnlnL4zsBcP8dXQdBD_GVrHSHYwFbzZtDHhNTOvP0lrXG_e5i-xGJVgJecRZDLQjkIeISyw4aFGSZiDhWUgKW1oV_7rFYv7cCe85EL28-DFySlw2yO8jSF-7ycHEjAmyxXxqhjguogEL-XrfucKBr5YD4t4cCbJnSga-Xrlwtw_rKXhLG3XpCLK-T1cGrYorZqWA2kth4a5vLbg==)
51. [allenai.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGuGp6TaSdlEslyER9nIz_kPYF17k78V1YChLvL1ORXNBVE-Kg7bP_7cpvSaqFTNNJSSxehWIu_YsCRlXCLpVxD9DBp08d5rgAi8M-9Gl3GPQQpKWnlhE7i)
52. [aclanthology.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHDIGnXhhQbBFJIibeukTqoKysLVPXuOurgsMyZ8voUHfIcXzsEbcVjwYKa06CYW3JhsdsCs_7zOWvpPzMqO0AVBm56voGQvfN42HJtljO6yQsZQxBbOO59oTgutLmCsMDYp-kE-pk=)
53. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGsWt1u3dW-Q9ZgiZ7dd78IOoLOaiSY-G_joM6UdrQTKgfL0-rcQXo2mADJFOX9UiISyKZX5orJVniPNUG7TNhoXxjVM8J9cUr0qqPFuQtkx8B2Lpgx5woBZw==)
54. [emergentmind.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGlsfNs_Pu-X-SCY_ePQ-JitTQ8mKa30jNIid2T3nW6JNuWOMvmKdNZxC_S0v7RZD2wY6Q4swR6iC6BdLuKYytibng_8p-Bi84weLHmxGmkGBa_PSUhidEZx35aVS4L5_wY0dTJBg==)
55. [marktechpost.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHeCS7bQwIOkzX9C36b1JGYSV0mUVf-UPZgLXVHNPQ7CuNeUKLjuMcJ6_xIMxS7H7hhRIsr5gE1n2PAAgHl_dbPqkcSTOzfYce0idL7EBSqGXrLfLj39yjKlIPECoHKclWqGZknLtnpT6Zh7wl_ky8evfidw4231NGwgrsMFjJ1iASG7PShxBcHt6a6pR5eJa5WvrVA5OCENtusJeUC6AJh9ZQUafC9fMqbSQQpNgWWzLkVSqE0TTy2YVgo42nv2xfjjZCPEY6RpdqWa-2RVUdWLBbJlJlf6c74KjSOgvPoXtc-XeK87QK5UqiHCnIWmkLzWV9b)
56. [mcgill.ca](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFAJWB6818zZD-VoW1A4pW0nlb661_RILF7AKBTfvRtBLzIxjqlURIbUAgjPyRNvgqydFLNs7iweWTspEkqaU6jWFPG0aCVv-tcEpJwLZ4T6AjJ22rhsbdoPr9cQhfj1f6H2gJyQ42hUsaGjQq_JmXxvS7hQph-DUtRQWmWDIIE63c=)
57. [mit.edu](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQHeKCV0ga579ID6DA4-SUz0Mey7U2qadYmMQHPZB6IqpHzqkbYa7yfp0_Ca59D094HI4OHrhQ0oBfFzYLVn3WxnT18vpiMXtf7IZsa73OibLpgW-zHlyG36tDBURBdCGwanGjRjwYEevotyXgZh01hFFk-BJ1nbCW6O3gyMHEXwiWozZbuX6N7AfJOTtRbnv08kz53rVd2OCbE85Q9_RKpVaMjrjSd1pCo3dic=)
58. [emergentmind.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQF2EIPje1WyKzZlKn2JSWQVZSWRG14_qcAx28ukkRaCwXKt0rjVsu0RH-HJQP34iKDsMev-RQUryA9JUZowIQplVkn_q-3skZ0AOXJzn76tV4zrpxfU-jt79dak-1x3BvNDkxzAePeg4Qaq7iCrPwwF)
59. [arxiv.org](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQGEizY9tRywf9HekRGWxFhvVOuvkTIQfgNn6Min2n41zdD2odi81dwBb4mAVLDIpOkDn6VyP3HOIQAOknaPpfb8MVAiDT5PTf5WWSxhMOJ0Yh4_3bn4SQ==)
60. [3dvar.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQFE1IqyzbjVnX9MSjUskVnIebrEQr69u6FjxBfG6pYQm2RWwHD4E5xRITpbdsQ9kUGYchEzuNwWcz6IYAaL2ERV0JLDG8ssCyqzIE4ER8hUJCCBVXQY8B2e8A43JsMR)
61. [letta.com](https://vertexaisearch.cloud.google.com/grounding-api-redirect/AUZIYQExP7wSccWSQnGhDcHJVKNMb-9P4s3I6qptLs1ke56FHnuoR2R_sKo6bez_P0gdMswhlfiLc-vfeXg2NRxDJSlW9LtmR0JHebFqns9ysX4hSLp5C3t1zslqNrtLBSX5depn)

