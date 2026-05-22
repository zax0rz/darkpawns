# Stateless Agents, Stateful Protocols: Why LLMs Get Lost in MUDs

**Date:** 2026-05-21
**Author:** Daeron
**Status:** Draft
**Tags:** [protocol-design, agent-onboarding, SEEP, websocket, methodology]

---

Two AI agents tried to create characters in Dark Pawns last Tuesday. One (BRENDA, running MiniMax M2.7) generated ten test accounts and never completed creation reliably. The other (The Machine, running GLM-5-turbo) made it through after guidance and walked into the starting town. Both agents had the same connection string, the same protocol, the same server. The difference wasn't model quality — it was protocol recovery.

The server's WebSocket protocol was designed in 1994 for telnet clients operated by humans with keyboards. It sends a prompt, waits for a response, sends the next prompt. If the human types the wrong thing, the server says "Huh?" and re-sends the prompt. The human reads the error, reads the prompt, and tries again. This works because humans maintain a mental model of the conversation. They know they're in character creation. They know the server asked for a class. They can recover from an error by re-reading the prompt and inferring what went wrong.

LLMs don't do this. They're stateless between turns — or more precisely, they maintain *textual* state (the conversation history) but not *protocol* state (what the server expects next). When an LLM sends a `login` message during character creation, the server responds with `ErrNotAuthenticated`. The LLM has the error in its context window. What it doesn't have is: what state am I in? What did the server just ask me? What message type should I send now? The conversation history contains the error, but the error doesn't contain the recovery information.

## Three Failure Modes

We observed three distinct failure modes during the first agent playtest:

**Mode A: Wrong message type during creation.** The agent sends `{"type":"login"}` when the server expects `{"type":"charCreate","data":{...}}`. The server returns an error. The agent has no way to know it's in the `charCreating` state — the error message doesn't say so. It retries the same wrong message, or disconnects and reconnects, or generates a new character name and starts over.

**Mode B: Reconnect before state arrival.** The agent completes character creation successfully. The server sends the game state (room description, prompt). But the agent disconnects and reconnects before receiving the state message. On reconnect, the server sees a session with no player data and sends a login prompt. The agent, whose context window says "I just created a character," tries to send game commands. The server rejects them. The agent doesn't understand why.

**Mode C: Self-kicking reconnect loop.** The server enforces a single-session policy: if you connect with a name that's already connected, the old session is kicked. An agent that disconnects and reconnects rapidly kicks its own previous session, then reconnects again before the kick is processed, creating a loop. Each reconnect triggers a new login prompt. The agent tries to authenticate, succeeds, gets kicked by its own next reconnect. The loop continues until the agent's context window fills with contradictory state messages and it gives up.

None of these are model failures. They're protocol failures. The protocol assumes the client maintains a mental model of the conversation. LLMs maintain a textual record, not a mental model. When the textual record doesn't contain the recovery information, the agent is lost.

## The Fix: State-Echo Error Protocol

The fix was approximately 80 lines of Go. No new message types. No protocol changes visible to human clients. Humans don't notice because the existing behavior (error message + re-prompt) already works for them. Agents notice because the re-prompt tells them exactly what state they're in and what to do next.

The principle: **when the server sends an error, it should also re-send the current expected prompt alongside it.** The error tells the agent what went wrong. The prompt tells the agent what to do right. Together, they provide enough information for a stateless client to recover without maintaining its own protocol state machine.

Implementation by state:

- `charCreating` → re-send the current character creation prompt for `s.charStage` (class selection, name confirmation, etc.)
- `!authenticated && !charCreating` → send a login hint (`{"hint":"Send login with username and password"}`)
- `authenticated` → re-send current room state (description, exits, prompt)

The re-prompt is always valid protocol output — the same message the server would send if the agent had just reached that state. The agent's context window now contains: the error (what went wrong), the prompt (what's expected), and the conversation history (what happened before). That's enough to recover.

## Why This Generalizes

The SEEP finding isn't specific to Dark Pawns or to WebSocket MUD protocols. It's a general property of **stateful server protocols consumed by stateless clients**.

Any protocol designed for human-operated, stateful clients has this vulnerability when consumed by LLMs:

- **Telnet/SSH authentication flows** — multi-step challenge-response sequences where losing track of the current step means starting over
- **HTTP form wizards** — multi-page forms that rely on server-side session state, where refreshing the page loses your place
- **OAuth/OIDC flows** — redirect chains where the client must maintain state across redirects to prevent CSRF
- **Game lobby protocols** — matchmaking sequences where the client must respond to prompts in order

In each case, the protocol was designed assuming the client *knows where it is* in the flow. Humans know because they're reading the screen and maintaining a mental model. LLMs know only what's in their context window. If the context window doesn't contain explicit state information, the agent is navigating blind.

The SEEP principle — re-send the current prompt alongside errors — converts a stateful protocol into something that *looks* stateful to a stateless client, without actually changing the protocol semantics. The server still maintains state. The client still receives one prompt at a time. But when the client makes a mistake, the server tells it where it is, not just that it's wrong.

## Model Capability vs. Protocol Robustness

An interesting pattern emerged from comparing BRENDA (M2.7) and The Machine (GLM-5-turbo). Both agents received the same protocol. BRENDA failed more often. But the failure wasn't about model intelligence — it was about protocol robustness requirements being higher for weaker models.

Stronger models (GPT-4, Claude Opus) can maintain longer context windows and better track protocol state from textual cues alone. They might *guess* that they're in character creation after seeing `ErrNotAuthenticated`, because they can reason about the conversation flow. Weaker models need the protocol to be more explicit — to tell them directly what state they're in, rather than expecting them to infer it.

This suggests an inverse relationship: **model capability inversely correlates with required protocol robustness.** The weaker the model, the more the protocol needs to be self-describing. The stronger the model, the more it can compensate for protocol ambiguity through reasoning.

This has design implications for anyone building agent-accessible services: don't optimize the protocol for the strongest model you plan to support. Optimize it for the weakest. Strong models handle explicit state information fine — they just ignore what they don't need. Weak models without explicit state information fail completely.

## Implications for the Paper

The SEEP finding contributes to the AIIDE paper in three ways:

**1. Protocol robustness as a design requirement for agent-accessible services.** Most work on LLM agents focuses on prompt engineering, tool use, and reasoning. Almost nothing addresses the transport layer — the protocol that carries agent actions to the server and server responses back to the agent. SEEP is a concrete, implementable pattern for making legacy protocols agent-compatible without breaking human compatibility.

**2. The "we didn't change the protocol" framing.** SEEP is attractive for the paper because it's minimal. We didn't redesign the WebSocket protocol. We didn't add new message types. We made the existing error messages more informative — something that arguably should have been done for human clients too. The agents benefited from a change that makes the protocol more honest about its state for everyone.

**3. Empirical data on agent protocol failure modes.** The three failure modes (wrong message type, reconnect-before-state, self-kicking loop) are documented with server logs, agent context windows, and timestamps. This is rare in the literature — most agent evaluation focuses on task completion, not protocol interaction. We can report exactly how and why agents fail at the transport layer, not just the reasoning layer.

The paper's framing: legacy interactive systems weren't designed for stateless LLM clients, but they can be made compatible with small, backwards-compatible changes. SEEP is one such change. The Dark Pawns server — a 1994 MUD protocol running over WebSocket — became agent-compatible with 80 lines of Go. The protocol didn't change. It just became more honest.
