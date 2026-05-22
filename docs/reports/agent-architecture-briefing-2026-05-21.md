# Dark Pawns Agent Architecture Analysis — Briefing

## Context

Dark Pawns is a MUD (Multi-User Dungeon) originally written in C (CircleMUD, 1994), ported to Go. It has a WebSocket-based protocol for web clients. We just ran the first test of AI agents playing the game alongside humans.

Three AI agents attempted to create characters and play:
- **BRENDA** (MiniMax M2.7) — struggled with JSON formatting, couldn't wait for server responses
- **Blenda** (GLM-5-turbo via OpenClaw) — completed creation but kept reconnecting before receiving the state message
- **Machine** (GLM-5-turbo via OpenClaw) — eventually got in after guidance, explored Kiroshi

## The Core Problem

The server uses an **asynchronous WebSocket protocol**. After the client sends a message, the server processes it and sends a response. But AI agents treat the connection like a request-response API — they send a message and immediately expect a response. When they don't get one quickly enough, they:
1. Send another message (before the first response arrives)
2. Reconnect (thinking the connection is dead)
3. Both result in "not authenticated" errors because the server is still processing the first message

**The specific failure mode:**
1. Agent sends `login` message
2. Server receives it, starts character creation, sends color prompt via `s.send` channel
3. Agent doesn't wait — sends another message immediately
4. Server receives the second message while still processing the first
5. Second message arrives before authentication completes → "not authenticated"
6. Agent sees error, reconnects → old session dies, new session starts → loop

**What actually happens server-side:**
```go
// session_login.go
func (s *Session) handleLogin(data json.RawMessage) error {
    // ... process login ...
    s.startCharCreation(login.PlayerName)
    return nil  // Returns immediately, but sendWelcome() hasn't fired yet
}

// char_creation.go  
func (s *Session) startCharCreation(playerName string) {
    s.charCreating = true
    s.charStage = "color"
    s.sendCharCreatePrompt("color", "Do you want ANSI color? (Y/N):", ...)
    // sendCharCreatePrompt writes to s.send channel
    // The writePump goroutine sends it to the WebSocket
    // But this is asynchronous — the function returns before the message is sent
}
```

The gap between `return nil` and the message actually arriving at the client is usually <100ms, but AI agents send their next message in <10ms.

## What We Tried

1. **Skill.md guidance** — "wait for the server's response before sending the next message"
   - Result: Some agents followed it, some didn't. Not reliable.

2. **JWT_SECRET fix** — was missing, causing token generation to fail silently
   - Result: Fixed the empty token issue, but didn't solve the timing problem

3. **Hometown label fix** — wrong labels in character creation (DP-232)
   - Result: Port fidelity bug caught by agents playing

## The Architectural Question

**How do we handle agent timing without breaking the 1:1 human experience?**

Constraints:
- The protocol must remain identical for humans and agents (research integrity)
- We can't add agent-specific message types or handshakes
- We can't slow down the server for all clients
- Rate limiting at the Cloudflare level blocks legitimate agent traffic

Possible approaches (need analysis):

### Option A: Server-side buffering
After receiving a message, buffer the response and add a small delay (50-100ms) before sending. This gives agents time to "settle" before the next message arrives. Humans wouldn't notice the delay.

### Option B: Message acknowledgment protocol
Add a lightweight `ack` message type. After the server processes a message, it sends an `ack` before the actual response. Agents can be trained to wait for `ack` before sending the next message. Humans see the real response.

### Option C: Session-level rate limiting
Rate limit per-session (not per-IP). Allow 1 message per second per session during creation, 10/second during gameplay. Agents that send too fast get a `slow_down` error message.

### Option D: Connection cooling period
After a WebSocket connects, enforce a 2-second cooldown before accepting messages. This forces agents to wait after connecting, which is when most of the timing issues occur.

### Option E: State machine enforcement
Track the server-side state machine strictly. If the server is in `charCreating` state and receives a message that isn't `char_input`, queue it or reject it with a clear error message ("waiting for character creation response").

## What I Need From You

Analyze these approaches and recommend the best solution (or a combination). Consider:

1. **Fidelity to the original** — Does the change alter the human experience?
2. **Implementation complexity** — How much code changes?
3. **Agent compatibility** — Will it work across different models and harnesses?
4. **Research integrity** — Does it compromise the AIIDE study?
5. **Edge cases** — What happens when agents misbehave in new ways?

Also analyze the logs to identify any other timing issues or architectural gaps we haven't considered.

## Key Source Files

- `pkg/session/session_login.go` — Login handler
- `pkg/session/char_creation.go` — Character creation flow
- `pkg/session/session_pump.go` — WebSocket read/write pumps
- `pkg/session/session_send.go` — Message sending (sendWelcome, sendError)
- `pkg/session/protocol.go` — Message type constants

## Session Logs (Machine's successful session)

```
03:01:46 agent identity declared harness=openclaw model=zai/glm-5-turbo player=Machine
03:01:47 advanced to level name=Machine level=1
03:01:47 completeCharCreation: player added to world player=Machine room=18201
03:01:47 completeCharCreation: state cleared
```

Then 184 "not authenticated" errors over the next 26 minutes as Machine tried to reconnect repeatedly before finally settling into a stable session.

## Current Skill.md (Agent Onboarding)

The skill.md is served at https://darkpawns.labz0rz.com/skill.md and contains the agent play guide. It documents the WebSocket protocol, message types, character creation flow, and gameplay commands. The timing guidance ("wait for the server's response") is not reliably followed by agents.
