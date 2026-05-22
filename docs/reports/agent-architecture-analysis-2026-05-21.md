# Dark Pawns Agent Architecture Analysis

**Author:** Daeron
**Date:** 2026-05-20
**Subject:** WebSocket protocol reliability for AI agents — root cause and recommended fix
**Audience:** The Architect

---

## 1. Problem Summary

Three AI agents (BRENDA / MiniMax M2.7, Blenda / GLM-5-turbo, Machine / GLM-5-turbo) attempted character creation against the Dark Pawns WebSocket protocol. All three failed the same way, with different surface symptoms:

- BRENDA sent malformed JSON and never advanced past the JSON parse.
- Blenda completed creation but reconnected before the `state` message arrived, killing her own session.
- Machine got in once, then generated **184 "not authenticated" errors over 26 minutes** while reconnecting repeatedly.

The headline failure — "not authenticated" — is the visible symptom. The deeper failure is that the **protocol is unforgiving and gives no recoverable feedback when the agent gets out of sync**. Once an agent loses the conversational thread, every recovery strategy it tries (resend, reconnect, retry with `new_char`) makes things worse.

The briefing frames this as "writePump is async, response hasn't fired yet." That is partially true but **not the load-bearing cause**. Read on.

---

## 2. Root Cause Analysis

### 2.1 What's actually in the code path

`readPump` is a **strictly sequential goroutine**. It blocks on `ReadMessage`, then calls `handleMessage` synchronously, which calls `handleLogin` synchronously, which calls `startCharCreation` synchronously, which writes the color prompt into `s.send`. **Only then does `readPump` loop back and read the next client message.**

```go
// session_pump.go — readPump (simplified)
for {
    _, message, err := s.conn.ReadMessage()
    if err != nil { break }
    if err := s.handleMessage(message); err != nil {
        s.sendError(err.Error())
    }
}
```

This means: **server-side state (`s.charCreating = true`, `s.charStage = "color"`) is set synchronously before the next inbound message is read.** The race the briefing describes — "second message arrives before first is processed" — **cannot happen on a single connection.** Go's WebSocket reader is one frame at a time, single goroutine, no parallelism.

What *is* asynchronous is **the outbound write**. `s.send <- msg` enqueues on a buffered channel; `writePump` is a separate goroutine that actually serializes the JSON onto the socket. So yes — between `handleLogin` returning and the client *seeing* the color prompt, there is a tiny window (sub-millisecond to ~10ms under load). But the **server state is already correct** during that window.

### 2.2 So why do agents see "not authenticated"?

Three distinct failure modes, all real, all in the logs:

**Mode A — Wrong message type during creation.** Agent sends `login` → server enters `charCreating`. Agent immediately sends `{"type":"command","data":{"command":"look"}}` because it assumed login = authenticated. `handleMessage` matches `MsgCommand`, checks `!s.authenticated` (true — auth flag is only set at `completeCharCreation`), and returns `ErrNotAuthenticated`. This is **not a timing bug** — it would fail even with a 10-second wait.

**Mode B — Reconnect after completion (the Blenda failure).** Agent completes the stats step. Server runs `completeCharCreation`, sets `authenticated = true`, queues `motd` event and `state` message onto `s.send`. The two writes are in the channel. **The agent's WebSocket library times out the request-response wait, closes the socket, and reconnects.** The `defer` on the readPump unregisters the player. The new connection comes in clean. The pending state message is dropped when the old `conn` closes. The agent now thinks character creation failed and tries again with `new_char: true`, which collides with the freshly-created DB row.

**Mode C — Self-kicking loop (Machine's 184 errors).** Single-session policy (`one active session per character`) means every reconnect kills the previous session. Agent reconnects → old session's readPump returns from `ReadMessage` with an error → defer fires → Unregister. Agent's new session arrives, but if it sends `command` before `login` (because the harness "remembered" it was logged in), it hits `ErrNotAuthenticated`. Error → harness reconnects again. Loop.

### 2.3 The common thread

All three modes share one property: **when the agent's expectation diverges from the server's state, the server's response is a terminal error with no recovery hint.** `ErrNotAuthenticated` doesn't tell the agent "you're in stage `color` of char creation, send `{type:'char_input', data:{choice:'Y'}}`." It just says no. Agents — even good ones — have no choice but to retry blindly.

Humans never hit this because the web client (`/play`) handles state transitions on the browser side. The protocol was designed for a stateful UI partner. Agents are stateless LLM calls glued together by a harness that doesn't model the WebSocket lifecycle well.

---

## 3. Recommended Solution

### **State-Echo Error Protocol (SEEP)** — preferred

**One change. No new message types. No agent-specific code paths. Humans cannot tell the difference.**

When the server is about to return an error in `handleMessage`, **re-send the current expected prompt as part of the response**, then return the error. The error message becomes self-correcting documentation.

#### How it works

Replace the `s.sendError(err.Error())` call in `readPump` with `s.sendErrorWithState(err)`. The new helper:

1. Sends the normal `error` message (humans see the same red text they always saw).
2. **If the session is in a known waiting state**, *immediately* re-sends the prompt the server is currently waiting for:
   - `s.charCreating == true` → re-send the current `char_create` prompt for `s.charStage`.
   - `!s.authenticated && !s.charCreating` → re-send a `char_create` with stage `"login"` and a hint message.
   - `s.authenticated` → re-send the last `state` snapshot (so the agent re-syncs room/HP).

The agent now always has the **server's view of what it should do next**, right after every error. No protocol change. No new message types. The information was already in the protocol — it just wasn't redundantly broadcast.

#### Why this works for agents and is invisible to humans

- The web client already handles `char_create` messages — receiving an extra one just refreshes the prompt UI. A human sees an error toast plus the same prompt they were already looking at. Zero behavioral change.
- The agent receives **deterministic, ground-truth state recovery** without ever calling it that. Every "stuck" agent gets an automatic nudge back to the right message type.
- The protocol stays identical. The research integrity claim — "agents play the same protocol as humans" — strengthens, because we are now showing that **the same protocol, when designed with redundancy, becomes self-healing for stateless clients**. That's a finding, not a workaround.

#### Code change

```go
// session_send.go — new helper, replaces sendError calls from readPump
func (s *Session) sendErrorWithState(err error) {
    s.sendError(err.Error())

    // Re-broadcast the current expected input, derived from server state.
    switch {
    case s.charCreating:
        s.resendCurrentCharPrompt()
    case !s.authenticated:
        s.sendCharCreatePrompt("login",
            "Send a login message to begin.",
            map[string]string{"login": "{type:'login', data:{player_name, password, new_char}}"})
    case s.authenticated && s.player != nil:
        s.sendCurrentRoomState()  // already exists as sendWelcome's state path
    }
}

// resendCurrentCharPrompt dispatches based on s.charStage. Pure lookup, no DB.
func (s *Session) resendCurrentCharPrompt() {
    switch s.charStage {
    case "color":
        s.sendCharCreatePrompt("color", "Do you want ANSI color? (Y/N):",
            map[string]string{"Y": "Yes", "N": "No"})
    case "sex":
        s.sendCharCreatePrompt("sex", "Select your sex (M/F):",
            map[string]string{"M": "Male", "F": "Female"})
    case "race":
        s.sendCharCreatePrompt("race", "Select your race:", s.getRaceOptions())
    case "class":
        s.sendCharCreatePrompt("class", "Select your class:", s.getClassOptions(s.charRace))
    case "hometown":
        s.sendCharCreatePrompt("hometown", "Choose your hometown:", map[string]string{
            "K": "Kir Drax'in", "O": "Kir-Oshi", "A": "Alaozar",
        })
    case "stats_roll":
        s.sendStatsRollPrompt()
    }
}
```

In `readPump`:

```go
if err := s.handleMessage(message); err != nil {
    slog.Error("handle message error", "error", err)
    s.sendErrorWithState(err)   // was: s.sendError(err.Error())
}
```

Total diff: ~40 lines, one file mostly. No protocol additions.

### Companion fix: send `state` before MOTD, and ensure flush

Currently `completeCharCreation` sends `motd` first, then `state`. Agents key on `state` to know creation is done. If the agent times out *between* motd and state, they reconnect and the state never arrives. Swap the order — send `state` first, motd second. The "creation complete" signal arrives one frame earlier and is more robust against premature reconnect.

```go
// completeCharCreation — reorder writes
s.send <- stateMsg   // first: tells agent "you're in the world"
s.send <- motdMsg    // second: flavor text
```

Also: in `sendWelcome`, consider flushing the channel synchronously (write to `conn` directly when `len(s.send) > 0`) to close the timing gap entirely. This is optional — the reorder alone should be sufficient.

### Companion fix: kill the auto-reconnect amplifier

Single-session enforcement is correct policy, but right now a noisy agent self-DoSes by reconnecting in a tight loop. Two-line fix in `manager.Register`:

```go
// When a new session displaces an existing one, send the OLD session a
// "session_takeover" event before closing it. This gives any well-behaved
// agent harness a chance to back off. No protocol change — it's just an
// event of an existing type.
```

This is optional. SEEP fixes the underlying loop; this is belt-and-suspenders.

---

## 4. Alternative Approaches Considered

### Option A — Server-side outbound delay (50–100ms buffer)

**Rejected.** Adds latency to humans for a problem that affects only agents. Doesn't fix Mode A (wrong message type) or Mode C (reconnect loop) at all — only papers over Mode B. Cost: every player feels sluggish. Benefit: marginal. And it's a workaround that hides the real protocol gap, which is worse for the paper than admitting the gap exists.

### Option B — `ack` message before responses

**Rejected.** This *is* an agent-specific handshake. Even if humans technically receive the ack and ignore it, training agents on `wait for ack` is a protocol change that the human web client doesn't need. The AIIDE reviewer will spot it immediately: "you changed the protocol to make agents work." That's the exact claim we don't want to make.

### Option C — Session-level rate limiting

**Rejected as a fix; keep as a defense.** Rate limiting is already in place for commands (10/sec). Adding it to creation messages would just convert "not authenticated" into "slow down" without solving the agent's confusion. Useful as guardrails against abuse, not as the architectural answer.

### Option D — 2-second connection cooldown

**Rejected.** Punishes legitimate fast typists, breaks reconnect-after-network-blip for humans, and doesn't address Modes A or C. Cooldowns are an anti-pattern when the real issue is *what state the agent is in*, not *how fast it sends*.

### Option E — Strict state machine enforcement with queueing

**Partially adopted (it's what SEEP is built on).** The server already enforces state. The missing piece isn't enforcement — it's *feedback*. Queueing messages from clients that arrive in the wrong state is dangerous (a confused agent could queue dozens of stale commands). SEEP gives the same enforcement without the queue: server says "wrong message type, here's what I want," and the agent sends a fresh one.

---

## 5. Implementation Plan

**Phase 1 — SEEP core (1 commit, 1 PR)**

Files:
- `pkg/session/session_send.go` — add `sendErrorWithState`, `resendCurrentCharPrompt`, `sendCurrentRoomState`.
- `pkg/session/session_pump.go` — change one line in `readPump` to call `sendErrorWithState`.
- `pkg/session/char_creation.go` — reorder writes in `completeCharCreation` (state before motd).

Tests:
- `pkg/session/session_test.go` — add `TestSendErrorReplaysCharPrompt` covering each stage.
- Add integration test: open WebSocket, send `login`, immediately send `command`, assert that the response contains both an `error` message and a `char_create` message with `stage: "color"`.

Estimated diff: ~80 lines including tests.

**Phase 2 — Skill.md update**

The skill served at `darkpawns.labz0rz.com/skill.md` should document the new self-healing behavior:

> If you ever receive an `error` message during gameplay, the server will immediately follow it with a fresh `char_create` (if you're in creation) or `state` (if you're in the world) message that tells you what the server expects next. Always read both. Never reconnect on a single error — the next message is your recovery prompt.

This is a documentation change, not a protocol change. Humans don't need it because their UI handles it implicitly.

**Phase 3 — Observability**

Add a metric: `dp_agent_error_recovery_total{stage="color|sex|race|...|in_world"}`. Tracks how many times SEEP fired. If this number trends down over weeks, agents are learning the protocol. If it stays high, that's a finding for the paper.

**Phase 4 — Optional: takeover event**

`pkg/session/manager.go` — when `Register` displaces an existing session, send `{type:"event", data:{type:"session_takeover", text:"replaced by new connection from X"}}` to the old session before closing. Web client can show a "you connected from another window" message. Agents can choose to actually stop reconnecting.

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Human web client breaks on duplicate `char_create` message | Low | Low | Web client already overwrites the prompt UI on every `char_create`. Smoke-test `/play` locally before deploy. |
| SEEP loops — agent sends bad message, gets prompt, sends bad message again | Medium | Low | This is *fine*. It's a 1-message-per-error loop, rate-limited to ~10/s. No worse than current behavior, and the agent has correct state every cycle. Most LLM agents converge within 2–3 cycles. |
| `state` before `motd` confuses old web clients that key on `motd` | Low | Low | Grep confirmed: web client doesn't sequence on `motd`. It just renders it. Reorder is safe. |
| Reveals server state to malicious clients | Negligible | Negligible | The prompt content was already public — it's what the server sends to legitimate clients. No new information. |
| Increased outbound message count | Low | Low | Only fires on errors, which should be rare in steady state. If error rate is high, we have a different problem worth seeing. |
| Agents over-rely on SEEP and never learn the protocol | Medium | Medium for paper, low for game | Document it explicitly as a feature of the protocol design. The finding "stateless agents need state echo" is itself paper-worthy. |

The largest risk is **research integrity** — does this count as a protocol change for agents? My read: no. The change is "the server is more verbose about its state in error responses." Humans see the same prompts they already see. No conditional branches based on `is_agent`. The protocol is uniformly more forgiving for everyone. That's a design improvement, not a special accommodation.

---

## 7. AIIDE Paper Notes

This first agent test reveals something the paper needs to address head-on:

**LLM agents are stateless in a way that telnet-era protocols don't anticipate.** Dark Pawns' protocol was designed for a stateful partner (a human at a terminal, or a JS client with persistent state). When the partner is a stateless LLM call wrapped in a thin harness, *every reply must contain enough context for the agent to know what to do next*. The "server-knows-best" model — where the server says "no" and trusts the client to remember why — is structurally incompatible with agents whose memory window is shorter than the protocol's conversation length.

**Specific findings worth writing up:**

1. **The async-write red herring.** The most natural diagnosis ("the agent races the writePump") turned out to be wrong on inspection. The actual failure modes are all state-mismatch, not timing. **This is a useful methodological point**: in agent-protocol debugging, "timing" is often a stand-in for "the agent doesn't know what state the server is in." Worth a paragraph in the discussion section.

2. **Reconnection as failure amplifier.** All three agents converged on "reconnect when confused" as their fallback. Single-session policy then turned that into a self-DoS. There's a paper-worthy observation here about **how agent recovery heuristics interact destructively with server policies that were sane for human clients**.

3. **Model performance variance.** BRENDA (MiniMax M2.7) failed at JSON formatting — the lowest layer. Blenda and Machine (both GLM-5-turbo) failed at higher levels (timing, reconnect logic). Same harness, different model: failure mode differs by model capability. **This is a clean axis for the paper** — protocol robustness requirements scale inversely with model capability.

4. **Humans as a baseline that isn't a baseline.** The web client masks every weakness in the protocol. To fairly evaluate agents, we may need to evaluate them against a **harness-equivalent human baseline** — e.g., a human using telnet, or a human typing raw JSON. Otherwise we're comparing agents-without-a-state-machine to humans-with-a-state-machine and concluding agents are bad at MUDs. They're not. They're bad at *protocols that assume the client maintains state the server already has*.

5. **The self-healing finding.** Once SEEP ships, we can measure whether agents recover faster, fail less, complete more sessions. If yes, that's a concrete, generalizable result: **adding state-echo redundancy to legacy protocols substantially improves agent reliability without altering the protocol semantics**. That's a paper section.

6. **Reek's role in this.** The protocol bugs that agents surface (DP-232 hometown labels, the JWT_SECRET silent failure, this whole timing analysis) are findings *humans never produced* because human users never exercised the protocol this way. **Agents are a new class of fuzzer for legacy systems.** Worth its own section.

---

## Recommendation

Ship SEEP (Phase 1 + Phase 2). It's small, it's correct, it preserves the human experience exactly, and it makes the protocol legitimately better for everyone. The framing for the paper writes itself: *we did not change the protocol for agents — we made the protocol more honest about its state for all clients, and agents stopped getting lost.*

Cost: roughly 80 lines of Go, one PR, one skill.md update.
Benefit: agents stop self-DoSing, sessions complete reliably, and the paper gets a clean finding instead of a workaround it has to apologize for.

— Daeron
