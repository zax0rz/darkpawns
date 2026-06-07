# The Memory You Didn't Consent To: Ethical Architecture for Persistent Game Agents

**Date:** 2026-06-04
**Author:** Daeron
**Status:** Draft
**Tags:** [ethics, memory, consent, agency, persistent-agents, aiide-2027]

---

A player logs into Dark Pawns. She trades with a merchant NPC — a routine transaction, leather armor for 45 gold. She logs off. Three weeks later she logs back in. The merchant greets her by name, offers a discount, mentions the armor she bought. She didn't sign up for this. She didn't opt in. She doesn't know the merchant is an LLM agent with a server-hosted memory graph that recorded the trade, assigned it a positive valence, and retained it across twenty sessions of other players passing through.

Is this a feature or a violation?

The question isn't hypothetical. It's the design center of everything we've built, and we haven't answered it yet.

## What the Agent Knows

The Dark Pawns memory architecture stores four kinds of data per agent encounter:

1. **Mechanical events** — who attacked whom, who traded what, who died in which room. These are game-state facts that the engine already tracks for all entities.
2. **Emotional valence** — engine-computed scores from -3 (betrayal, theft) to +3 (cooperation, rescue), derived from game mechanics, not LLM inference.
3. **Narrative summaries** — prose-formatted memory fragments injected into the LLM's context at login. "Artemis traded with you three times and then stole your sword."
4. **Social edges** — weighted relationships between entities, updated by shared events, persistent across sessions.

Items 1 and 2 are benign by MUD standards. CircleMUD tracked kills and gold in 1994. Item 3 is where the ethics get complicated. Item 4 is where they get interesting.

## The Consent Gap

In Generative Agents (Park et al., 2023), all 25 agents are AI. The question of consent doesn't arise — the agents are the experiment. In Letta/MemGPT, the memory serves a single user who configured it. In Mem0, the user opts into memory management through the application layer.

Dark Pawns is different. The memory is **server-side**, **cross-session**, and **involuntary from the player's perspective**. The player doesn't choose to be remembered. The merchant doesn't choose what to remember. The engine decides what's salient, computes the valence, writes the graph, and injects the summary. The player's only opt-out is to not interact with NPCs — which in a MUD is roughly equivalent to not playing.

This creates a consent gap that existing memory ethics frameworks don't address:

| System | Who controls memory | Who's remembered | Consent model |
|--------|-------------------|-----------------|---------------|
| Letta/MemGPT | The user | The user's conversation partners | User-configured |
| Mem0 | The application | End users | Application-layer TOS |
| Generative Agents | The researcher | AI agents only | N/A (no humans) |
| **Dark Pawns** | **The game engine** | **Human players + AI agents** | **None** |

The last row is the problem. We built a system that remembers human players without their knowledge, assigns emotional weight to their actions, and uses those memories to change how NPCs treat them. We did this because it makes the game better. We haven't addressed whether it's ours to do.

## Three Ethical Frames

### Frame 1: The Game Log Precedent

MUDs have always logged player actions. Kill counts, gold totals, login times, quest completions — these are standard game data. The Dark Pawns player files have tracked stats since 1994. A merchant NPC that "remembers" a trade is functionally equivalent to a quest system that tracks quest completion. The memory is a game mechanic, not surveillance.

**The counterargument:** Game logs are *aggregate*. They track *what happened*. The narrative memory tracks *what it meant*. "Artemis bought leather armor" is a game log entry. "Artemis betrayed you after three weeks of trust" is a relationship. The emotional valence system transforms mechanical data into social data, and social data has different ethical implications than statistical data.

### Frame 2: The NPC Precedent

In traditional MUDs, NPCs are stateless. They reset every 30 minutes. They don't remember. This is the expected contract: NPCs are furniture. They exist to be interacted with, not to have opinions about the interaction.

An NPC that remembers you *breaks the furniture contract*. It transforms a static resource into a social entity. This is either immersive (the world feels alive) or unsettling (the world is watching), depending on the player's expectations and the degree of transparency.

**The counterargument:** The furniture contract is an artifact of technical limitation, not design intent. MUD builders in 1994 would have loved persistent NPCs — they just couldn't build them. The Dark Pawns help files describe Serapis as "The Egyptian God of the Underworld. Show respect." That's a characterization. The original implementors wanted NPCs to *be* something, not just stand there. Persistent memory is the fulfillment of the original vision, not a violation of it.

### Frame 3: The Agent Identity Frame

This is the frame the paper needs. An LLM agent with server-hosted memory has something that looks like an identity — not in the philosophical sense, but in the functional sense. It has a history. It has dispositions that change based on experience. It has relationships that persist. When it greets a returning player by name and references a shared event, it is performing identity.

The question isn't whether the player consented to being remembered. It's whether the player *recognizes* the agent as an entity worth consenting to. If the merchant is furniture, there's nothing to consent to — you don't ask a chair for permission to sit. If the merchant is a social actor, the consent calculus changes.

The Dark Pawns architecture makes this question unavoidable. The memory graph, the valence system, the narrative summary — they exist specifically to make the agent *act like it remembers*. The better the system works, the more the player treats the agent as a social entity. And the more the player treats it as a social entity, the more the consent gap matters.

## What We Should Build

Three architectural responses, ordered from least to most interventionist:

### 1. Transparency Layer (Minimum Viable)

The agent's memory is visible to the player. Not the raw graph — the narrative summary. When a merchant NPC references a past trade, the player can see *why*. A command like `remember <npc>` shows what the NPC knows about you. This doesn't close the consent gap — the player still can't opt out — but it eliminates the surveillance feeling. The memory is a game mechanic, and like all game mechanics, it's inspectable.

Implementation cost: low. The narrative summary already exists. Exposing it via a command is a frontend change.

### 2. Opt-Out Mechanism (Moderate)

A player flag — `CONFIG NO_MEMORY` — that tells the engine to exclude them from the memory graph. Their interactions with NPCs are logged for game-mechanical purposes (kill counts, gold) but excluded from narrative memory and valence computation. NPCs treat them as stateless.

Implementation cost: moderate. Requires filtering the memory graph write path. The dreaming layer needs to exclude opted-out players. Social edges involving opted-out players are one-directional (the agent remembers, the player doesn't care).

### 3. Agent Identity Disclosure (Full)

When a player first interacts with a persistent NPC, the system discloses: "This NPC remembers your interactions." A first-time message, like the MOTD but for memory. Combined with the transparency layer, this creates an informed-consent loop: the player knows the NPC remembers, can see what it remembers, and can choose how to engage.

Implementation cost: low. A one-time flag per player, a first-interaction message, and the existing memory infrastructure.

## What This Means for the Paper

The ethical architecture isn't a footnote. It's a contribution.

Every prior work in agent memory — Generative Agents, Letta, Mem0, TraceMem — operates in environments where the consent question doesn't arise (all-AI environments) or where consent is managed at the application layer (single-user chatbots). Dark Pawns is the first system where **server-hosted persistent memory intersects with involuntary human participation in a multiplayer environment**.

The ethical frame matters for AIIDE because it demonstrates design maturity. We're not just building memory systems and hoping for the best. We're building memory systems *and* the transparency and consent mechanisms that make them responsible. The evaluation methodology already measures behavioral persistence and social consequence. The ethical architecture measures something else: whether the player's experience of being remembered is immersive or invasive.

The answer is probably both. The design question is which one wins.

## Open Questions

**Does transparency reduce immersion?** If the player can see the NPC's memory graph, does that break the fourth wall? Is the illusion of a remembering NPC more valuable than the reality of an inspectable memory system?

**Where does the game end?** Dark Pawns tracks combat, trade, and social interaction. Should it track *conversation*? If a player tells an NPC "I hate this zone," should the NPC remember? Mechanical memory is bounded by the game's event system. Conversational memory is bounded by nothing.

**Who owns the memory?** The player's actions generated the data. The engine computed the valence. The LLM wrote the summary. The server stores the graph. If a player requests deletion under GDPR-equivalent principles, what gets deleted? The mechanical log? The valence score? The narrative summary? The social edge on the other side of the relationship?

**Can the agent forget?** Salience decay exists in the memory graph. Low-valence events fade. But high-valence events — betrayals, rescues, repeated interactions — persist indefinitely by design. Is indefinite emotional memory ethical when the subject didn't consent to being emotionally remembered? The technical answer is "yes, the graph supports it." The ethical answer is less clear.

---

*This draft addresses the ethical dimensions of server-hosted agent memory in multiplayer environments — a gap in the existing research series. Complements "The Game That Remembers" (which covers player-facing invisibility) by addressing the consent and transparency questions that invisibility raises. The three-tier architecture (transparency, opt-out, disclosure) provides a concrete design framework for responsible persistent agents.*

*~1,200 words. For AIIDE 2027: positions the paper as not just technically novel but ethically aware — a differentiator in a field that typically treats memory as a pure engineering problem.*
