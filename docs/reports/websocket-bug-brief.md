# WebSocket Integration Bug — Opus Brief

## The Problem
After character creation completes on the Dark Pawns WebSocket server, command responses (look, score, etc.) never reach the client. The client times out waiting.

## What Works
- Character creation flow: login → char_create prompts → char_input responses → state message
- The state message IS received by the client after server restart
- The writePump CAN send messages (char_create messages arrive instantly)

## What's Broken
- After state message, command responses never arrive
- Server logs "not authenticated" errors after each test session
- The `sendText` function has a `select` with `default` that **silently drops messages** if the send channel is full

## Key Files
- `pkg/session/session_pump.go` — writePump (sends from s.send to WebSocket), readPump (reads from WebSocket), handleMessage (routes incoming messages)
- `pkg/session/session_send.go` — sendWelcome (sends state message), sendText (sends command output — has silent drop!)
- `pkg/session/char_creation.go` — completeCharCreation (sets authenticated=true, calls sendWelcome)
- `pkg/session/session_login.go` — handleLogin (returning player path sets authenticated=true)
- `pkg/session/commands.go` — ExecuteCommand (processes commands)
- `pkg/session/cmd_look.go` — cmdLook (look command handler)

## Critical Code Paths

### Working path (char_create):
1. Client sends `{"type":"login","data":{...,"new_char":true}}`
2. Server calls `startCharCreation()` → sends char_create messages via `s.send <- msg`
3. Client sends `{"type":"char_input","data":{...}}` for each stage
4. Server calls `handleCharInput()` → advances stages
5. Final stage: `completeCharCreation()` → `sendWelcome()` → sends state via `s.send <- msg`
6. Client receives state message ✓

### Broken path (commands):
1. Client sends `{"type":"command","data":{"command":"look"}}`
2. Server calls `handleCommand()` → `ExecuteCommand()` → `cmdLook()`
3. `cmdLook()` calls `s.sendText(roomDescription)`
4. `sendText()` does: `select { case s.send <- msg: default: drop }`
5. Client never receives the response ✗

## My Theory
After `completeCharCreation`, the server queues several messages:
- State message (MsgState)
- MOTD event (MsgEvent)
- Agent full var dump
- Memory bootstrap
- Memory summary
- Enter event (MsgEvent)

If the writePump is slow to consume these and the 256-buffer channel fills up, `sendText` drops the command response silently. The server thinks it sent the response, but it never reached the wire.

## What I Need From Opus
1. Identify the root cause: why do command responses never reach the client after character creation?
2. Is the silent drop in `sendText` the culprit, or is something else blocking?
3. Is there a race condition between completeCharCreation and the writePump?
4. Is the session properly authenticated after character creation? (Server logs "not authenticated" errors)
5. What's different between the char_create path (works) and the command path (doesn't work)?
6. Is there a code path where `s.authenticated` gets reset to false after character creation?
7. Provide a specific fix — not a theory, a code change.

## Wishlist — Make It Perfect (Don't Just Patch)
If Opus finds the root cause, also evaluate these while we're in here:

### W1: Silent message drop is unacceptable
`sendText` uses `select { case s.send <- msg: default: drop }`. If the channel is full, the player never sees the response and there's no log. At minimum: log the drop with player name + message type. Better: block with a short timeout and log a warning if it takes too long. Critical: for agent sessions, a dropped message means the LLM never gets a response and hangs.

### W2: Connection limit leak (already found)
The `ipConnCount` is incremented on connection but only decremented in readPump cleanup. If the connection is rejected at the limit check (line 360 of manager.go), the count was already incremented but the session is never created, so readPump never runs, so the count is never decremented. This means rejected connections permanently consume a slot until server restart. Fix: decrement on the rejection path too.

### W3: Agent session needs guaranteed message delivery
Agent sessions (LLM-driven) are fundamentally different from human sessions. A human can retry a command. An LLM agent hangs forever if it doesn't get a response. Agent sessions should either: (a) use a blocking send with a timeout + error, or (b) have a dedicated buffered channel that's larger than the human default.

### W4: State message timing
The server sends state, MOTD, var dump, memory bootstrap, memory summary, and enter event in rapid succession after login. That's 6+ messages queued immediately. If any of these block or the channel fills, subsequent messages (command responses) are silently dropped. Consider: (a) batching these into a single message, (b) increasing the channel buffer for agent sessions, or (c) sending state first, then the rest after a small delay.

### W5: Session takeover race
When a new connection logs in with the same player name, the old session is forcibly closed. But there's a 5-second takeover window (P0-3). During this window, messages from both sessions can interleave. If the old session's writePump is still sending while the new session is being set up, the new session's state message could be lost. Consider: close the old connection's writePump immediately on takeover, not after the probe window.

### W6: Command response format consistency
The server sends different message types for different outputs: `text` for command responses, `vars` for subscriptions, `event` for enter/motd/combat. The client has to guess which type to expect. Consider: always wrap command responses in a consistent envelope, e.g., `{"type":"response","command":"look","text":"..."}` so the client can reliably parse them.

## Server Behavior
- Server: Docker container `darkpawns-server-1` on 192.168.1.15
- WebSocket endpoint: ws://192.168.1.15:4350/ws
- Connection limit: 5 per IP (hardcoded in manager.go)
- Send channel buffer: 256
- Rate limit: 10 commands/sec per session

## Test Results
- `test_iso.go`: State message received ✓, look command timeout ✗
- `test_full2.go`: State message received ✓, all commands timeout ✗
- Server logs: "completeCharCreation: state cleared" appears, then "not authenticated" errors
