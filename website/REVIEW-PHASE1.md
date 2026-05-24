# Phase 1 Review — Agent Protocol

Reviewed: 2026-05-24
Docs: `website/content/docs/agents/protocol.md` (317 lines), `website/content/docs/agents/_index.md` (52 lines)
Source: `pkg/session/protocol.go` (149 lines), `pkg/session/agent_vars.go` (359 lines), `pkg/session/session_login.go` (265 lines), `pkg/session/agent.go` (4 lines), `pkg/session/decision_capture.go` (195 lines), `pkg/session/memory_hooks.go` (229 lines)

---

## protocol.md

### Connection Details

#### [ACCURATE] — Rate limit: 10 commands per second
- **Doc claim:** "Rate Limit: 10 commands per second (token bucket algorithm enforced per connection)"
- **Source evidence:** `session_player.go:80` — `rate.NewLimiter(rate.Limit(10), 10)` and `session_login.go:233` — `if !s.limiter.Allow()`
- **Verdict:** matches. The per-session limiter is 10 cmd/sec. Note: the doc says "token bucket" but the implementation uses `golang.org/x/time/rate` (which is a token bucket internally), so the claim is accurate.

#### [MINOR INACCURACY] — Rate limit algorithm description
- **Doc claim:** "token bucket algorithm enforced per connection"
- **Source evidence:** `pkg/auth/ratelimit.go:140` — `rate.NewLimiter(rate.Limit(5), 10)` is the **IP-level** login rate limiter (5 req/sec, burst 10). The per-session command limiter is `rate.Limit(10)` with burst 10.
- **Verdict:** The doc only documents the per-session command rate (10/sec). There is a **separate** IP-based login rate limiter (5 req/sec, burst 10) at `pkg/auth/ratelimit.go:140` that is NOT documented. Not a contradiction — the doc says "10 commands per second" for commands, which is correct. But the login rate limiter is undocumented.

#### [ACCURATE] — Outbound sequence stamping
- **Doc claim:** "The server stamps an incrementing sequence number `seq` (unsigned 64-bit integer) on **every** outbound message sent to agent sessions"
- **Source evidence:** `protocol.go:35` — `Seq uint64 json:"seq,omitempty"` in ServerMessage struct
- **Verdict:** matches. The `Seq` field is on all ServerMessage types (not just agent-specific ones, but that's a minor nuance).

---

### Message Wrappers

#### [ACCURATE] — ClientMessage type
- **Doc claim:** `"type": "login | command | subscribe | char_input"`
- **Source evidence:** `protocol.go:8-11` — `MsgLogin = "login"`, `MsgCommand = "command"`, `MsgCharInput = "char_input"`, `protocol.go:28` — `MsgSubscribe = "subscribe"`
- **Verdict:** matches exactly.

#### [ACCURATE] — ServerMessage type
- **Doc claim:** `"type": "state | event | vars | error | text | char_create | token_refresh"`
- **Source evidence:** `protocol.go:14-20` — `MsgState`, `MsgEvent`, `MsgError`, `MsgText`, `MsgCharCreate`, `MsgVars`, `MsgTokenRefresh`
- **Verdict:** matches exactly.

---

### Login

#### [ACCURATE] — Login uses `"password"` and `"is_agent": true`
- **Doc claim:** Login request uses `"password"` and `"is_agent": true`
- **Source evidence:** `protocol.go:50-57` — `Password string json:"password,omitempty"`, `IsAgent bool json:"is_agent,omitempty"`
- **Verdict:** matches. No `"api_key"` or `"mode"` fields exist in LoginData.

#### [ACCURATE] — LoginData fields
- **Doc claim:** Fields are `player_name`, `password`, `is_agent`, `harness`, `model`, `version`
- **Source evidence:** `protocol.go:48-61` — LoginData has `PlayerName`, `Password`, `Class`, `Race`, `NewChar`, `IsAgent`, `Harness`, `Model`, `Version`
- **Verdict:** matches. The doc shows a subset (omitting `class`, `race`, `new_char` from the login example), which is fine since those are `omitempty` optional fields.

#### [ACCURATE] — LoginData also has `class`, `race`, `new_char`
- **Doc claim:** Not shown in the login example, but LoginData struct comment at `protocol.go:43-46` documents them as optional
- **Source evidence:** `protocol.go:52-54` — `Class int`, `Race int`, `NewChar bool`, all with `omitempty`
- **Verdict:** The doc example is minimal but doesn't claim these don't exist. The struct fields `class` (int 0-11), `race` (int 0-6), `new_char` (bool) are real but undocumented in protocol.md. This is a **documentation gap**, not an error.

#### [ACCURATE] — Server responds with `"state"` message
- **Doc claim:** "The server responds with a `"state"` message type (there is **no** fictional `login_response` message)"
- **Source evidence:** `session_login.go:180` — `s.sendWelcome(token)`, and `session_send.go:52-59` — sends `ServerMessage{Type: MsgState, Data: state}`
- **Verdict:** matches exactly.

#### [ACCURATE] — StateData structure
- **Doc claim:** `player` object with `name`, `health`, `max_health`, `level`, `class`, `race`, stats; `room` object with `vnum`, `name`, `description`, `exits`, `doors`, `players`, `mobs`, `items`; `token`
- **Source evidence:** `protocol.go:71-89` — `StateData{Player, Room, Token}`, `PlayerState` has exactly those fields, `RoomState` has `VNum`, `Name`, `Description`, `Exits`, `Doors`, `Players`, `Mobs`, `Items`
- **Verdict:** matches exactly.

---

### Command

#### [ACCURATE] — Command request format
- **Doc claim:** `{"type": "command", "data": {"command": "hit", "args": ["guard"]}}`
- **Source evidence:** `protocol.go:66-69` — `CommandData{Command string, Args []string}`
- **Verdict:** matches exactly.

#### [ACCURATE] — No `command_response` message type
- **Doc claim:** "Commands **do not** receive immediate, direct response types (there is **no** fictional `command_response` message)."
- **Source evidence:** `protocol.go:14-20` — no `MsgCommandResponse` constant exists
- **Verdict:** matches exactly.

#### [ACCURATE] — Command feedback delivered as `"event"` with `"type": "text"`
- **Doc claim:** Textual feedback is delivered as an `"event"` message of subtype `"text"`
- **Source evidence:** `session_send.go:148-155` — `SendMessage` creates `ServerMessage{Type: MsgEvent, Data: EventData{Type: "text", Text: message}}`
- **Verdict:** matches exactly.

---

### Subscription

#### [ACCURATE] — Subscribe request format
- **Doc claim:** `{"type": "subscribe", "data": {"variables": ["HEALTH", "MAX_HEALTH", ...]}}`
- **Source evidence:** `agent_vars.go:65-70` — `handleSubscribe` unmarshals `{"variables": []string}`
- **Verdict:** matches exactly.

#### [ACCURATE] — No `subscription_response` message type
- **Doc claim:** "The server returns success silently (it does **not** send a `subscription_response` type)"
- **Source evidence:** `agent_vars.go:65-80` — `handleSubscribe` returns `nil` after registering vars, no message sent back
- **Verdict:** matches exactly.

---

### State/Variable Updates (`type: "vars"`)

#### [ACCURATE] — Vars message structure
- **Doc claim:** `{"type": "vars", "seq": 3, "data": {"HEALTH": 92, "FIGHTING": true, "ROOM_MOBS": [...]}}`
- **Source evidence:** `agent_vars.go:91-100` — `flushDirtyVars` marshals `ServerMessage{Type: MsgVars, Data: data}` where `data` is `map[string]interface{}`
- **Verdict:** matches exactly.

#### [ACCURATE] — Only changed variables (deltas) are flushed
- **Doc claim:** "Only changed variables (deltas) are flushed to optimize bandwidth."
- **Source evidence:** `agent_vars.go:82-88` — `flushDirtyVars` only iterates `s.dirtyVars`, then clears the set
- **Verdict:** matches exactly.

---

### Available State Variables (19 variables)

#### [ACCURATE] — All 19 variable names
- **Doc claim:** HEALTH, MAX_HEALTH, MANA, MAX_MANA, MOVE, MAX_MOVE, GOLD, POSITION, LEVEL, EXP, ROOM_VNUM, ROOM_NAME, ROOM_EXITS, ROOM_MOBS, ROOM_ITEMS, FIGHTING, INVENTORY, EQUIPMENT, EVENTS
- **Source evidence:** `agent_vars.go:14-32` — constants defined exactly match; `agent_vars.go:35-41` — `AllVariables` list has exactly 19 entries
- **Verdict:** matches exactly.

#### [ACCURATE] — Variable types
- **Doc claim:** HEALTH `int`, MAX_HEALTH `int`, MANA `int`, MAX_MANA `int`, MOVE `int`, MAX_MOVE `int`, GOLD `int`, POSITION `string`, LEVEL `int`, EXP `int`, ROOM_VNUM `int`, ROOM_NAME `string`, ROOM_EXITS `[]string`, ROOM_MOBS `[]RoomMobVar`, ROOM_ITEMS `[]RoomItemVar`, FIGHTING `bool`, INVENTORY `[]map`, EQUIPMENT `map`, EVENTS `[]map`
- **Source evidence:** `agent_vars.go:118-168` — `buildVarValue` returns exactly these types. `VarFighting` returns `bool` from `s.manager.combatEngine.IsFighting(s.player.Name)`.
- **Verdict:** matches exactly.

#### [ACCURATE] — FIGHTING is `bool`
- **Doc claim:** "Boolean flag indicating active combat status (true/false)"
- **Source evidence:** `agent_vars.go:153` — `return s.manager.combatEngine.IsFighting(s.player.Name)` returns bool
- **Verdict:** matches exactly.

#### [ACCURATE] — INVENTORY structure
- **Doc claim:** `[]map` with `name`, `vnum`, `instance_id`
- **Source evidence:** `agent_vars.go:312-320` — `buildInventory` returns `[]map[string]interface{}` with keys `"name"`, `"vnum"`, `"instance_id"`
- **Verdict:** matches exactly.

#### [ACCURATE] — EQUIPMENT structure
- **Doc claim:** `map` with slot names → `{name, vnum}`
- **Source evidence:** `agent_vars.go:324-332` — `buildEquipment` returns `map[string]interface{}` with `slot.String() → {"name": ..., "vnum": ...}`
- **Verdict:** matches exactly.

---

### Complex Types

#### [ACCURATE] — RoomMobVar
- **Doc claim:** `name`, `instance_id`, `target_string`, `vnum`, `fighting`
- **Source evidence:** `agent_vars.go:45-51` — `RoomMobVar{Name, InstanceID, TargetString, VNum, Fighting}`
- **Verdict:** matches exactly.

#### [ACCURATE] — RoomItemVar
- **Doc claim:** `name`, `instance_id`, `target_string`, `vnum`
- **Source evidence:** `agent_vars.go:54-58` — `RoomItemVar{Name, InstanceID, TargetString, VNum}`
- **Verdict:** matches exactly.

---

### Error Handling

#### [ACCURATE] — Error message format
- **Doc claim:** `{"type": "error", "seq": 4, "data": {"message": "rate limit exceeded — slow down"}}`
- **Source evidence:** `protocol.go:97-99` — `ErrorData{Message string}`, `session_send.go:126-133` — `sendError` marshals `ServerMessage{Type: MsgError, Data: ErrorData{Message: text}}`
- **Verdict:** matches. The doc example shows `"seq"` on the error, which is technically correct since all ServerMessages have the `Seq` field (omitempty).

---

### Minimal Agent Loop (Python)

#### [ACCURATE] — Python example
- **Doc claim:** Uses `"password"` and `"is_agent": True`
- **Source evidence:** Matches the LoginData struct fields
- **Verdict:** matches exactly.

---

## _index.md

### [ACCURATE] — "WHO" roster claim
- **Doc claim:** "Type `WHO` at a telnet prompt, and active AI agents appear on the roster right next to everyone else."
- **Source evidence:** `session_login.go:169` — agent sessions are registered via `s.manager.Register(login.PlayerName, s)` with no special flag; `protocol.go:55` — `IsAgent` is informational only. Agents are just regular sessions in the manager's session map.
- **Verdict:** matches. Since agents register the same way as humans, they would appear on WHO. No special exclusion logic exists.

### [ACCURATE] — Decision Capture Logging
- **Doc claim:** "Every command sent by an agent is tracked. If a Postgres database is connected, the server captures: Pre-state snapshot (HP, mana, moves, position, room VNUM), The exact command string and arguments executed, Post-state snapshot and error status, Dispatch timestamps and processing latency"
- **Source evidence:** `decision_capture.go:119-193` — `captureAndLog` records `PreHealth`, `PreMana`, `PreMove`, `PrePosition`, `PreRoom` (room VNUM), `Command`, `Args`, `RawInput`, `PostHealth`, `PostMana`, `PostMove`, `PostPosition`, `PostRoom`, `OutcomeCategory`, `OutcomeText` (error), `DurationMs`, `SessionElapsed`
- **Verdict:** matches exactly. The docs accurately describe the decision capture schema.

#### [MINOR INACCURACY] — "These logs are written in a partitioned database schema"
- **Doc claim:** "These logs are written in a partitioned database schema, allowing developer teams to review their agent's decision logs over thousands of ticks."
- **Source evidence:** `decision_capture.go:190` — calls `s.manager.decisionLog.RecordDecision(record)`. The `db.DecisionLogWriter` implementation is not in the reviewed files. The claim of "partitioned database schema" cannot be verified from the session package alone.
- **Verdict:** **UNVERIFIABLE** from reviewed source. Likely accurate (the `db` package would contain this), but we only see the writer interface.

### [ACCURATE] — Emotional Narrative Memory
- **Doc claim:** "The server hosts an active emotional memory system that records events (such as killing a mob or being defeated), computes emotional valence based on attributes, and updates a memory database."
- **Source evidence:** `memory_hooks.go:49-54` — `dreaming.KillValence(evt.KillerLevel, evt.VictimLevel)` computes valence; `memory_hooks.go:85-104` — death events get valence -2 or -3; both write to `db.WriteNarrativeMemory`
- **Verdict:** matches exactly.

### [ACCURATE] — Dreaming & Memory Consolidation
- **Doc claim:** "During agent sleep cycles (rest states), the server's consolidation pipeline consolidates raw decision logs and short-term memories into high-level, narrative paragraphs. When the agent logs back in, the server proactively flushes a `"memory_summary"` event containing this text"
- **Source evidence:** `memory_hooks.go:184-224` — `SendMemorySummary()` reads from `{dreamingDir}/{agent_id}/memory-summary.txt` (written by `pkg/dreaming/dream.go`); `session_login.go:182-186` — called on agent login: `s.SendMemoryBootstrap()` then `s.SendMemorySummary()`
- **Verdict:** matches. The doc says `"memory_summary"` event which matches the message type `"memory_summary"` at `memory_hooks.go:215`.

#### [MINOR GAP] — `memory_bootstrap` message not documented in _index.md
- **Doc claim:** Only mentions `"memory_summary"` event on login
- **Source evidence:** `memory_hooks.go:138-178` — `SendMemoryBootstrap()` is called BEFORE `SendMemorySummary()` on login (`session_login.go:185-186`). It sends a `"memory_bootstrap"` message with narrative memory data.
- **Verdict:** The `_index.md` mentions dreaming consolidation but doesn't mention the `memory_bootstrap` message that is also sent on login. The `protocol.md` also doesn't document either of these additional message types.

### [MINOR INACCURACY] — "sleep cycles (rest states)"
- **Doc claim:** "During agent sleep cycles (rest states), the server's consolidation pipeline consolidates..."
- **Source evidence:** `memory_hooks.go:190-198` — `SendMemorySummary()` reads from disk. The dreaming pipeline (`pkg/dreaming/dream.go`) writes to disk independently. The consolidation is NOT triggered by agent sleep/rest states — it runs on a cron cycle (dreaming is described in IDENTITY.md as "3:30 AM ET daily").
- **Verdict:** The doc implies the agent must be sleeping/resting for consolidation to happen. In reality, the dreaming cycle runs on a server cron, and the summary is pushed to the agent on next login regardless of what state it was in when it logged out.

---

## Summary of Issues Found

### CRITICAL
None.

### MAJOR
None.

### MINOR

1. **[MINOR] `memory_bootstrap` message undocumented** — `memory_hooks.go:138-178` shows `SendMemoryBootstrap()` sends a `"memory_bootstrap"` message on agent login (before `memory_summary`). Neither `protocol.md` nor `_index.md` document this message type. Agents receiving it unexpectedly may not know how to handle it. (`memory_hooks.go:165`, `session_login.go:185`)

2. **[MINOR] `memory_summary` message undocumented in protocol.md** — The `protocol.md` ServerMessage types list (`"state | event | vars | error | text | char_create | token_refresh"`) does not include `"memory_summary"` or `"memory_bootstrap"`. Both are real server→client message types sent on agent login. (`memory_hooks.go:165,215`)

3. **[MINOR] Dreaming consolidation described as agent-triggered, not cron-triggered** — `_index.md:42` says "During agent sleep cycles (rest states), the server's consolidation pipeline consolidates..." The dreaming pipeline runs on a server cron (3:30 AM ET), not triggered by agent sleep. The summary is read from disk on login. (`memory_hooks.go:190-198`)

4. **[MINOR] Login rate limiter undocumented** — `protocol.md` documents the per-session command rate limit (10/sec) but not the IP-based login rate limiter (5 req/sec, burst 10) at `pkg/auth/ratelimit.go:140`, or the login attempt lockout (10 failures → 15 min lockout) at `pkg/auth/ratelimit.go:157`. These affect agent connection behavior.

5. **[MINOR] `class`, `race`, `new_char` login fields undocumented in protocol.md** — LoginData has `Class` (int 0-11), `Race` (int 0-6), `NewChar` (bool) fields (`protocol.go:52-54`) that are not shown or explained in protocol.md. Agents creating new characters need this information.

6. **[MINOR] "Partitioned database schema" claim unverifiable** — `_index.md` claims decision logs use "a partitioned database schema." The `db.DecisionLogWriter` implementation was not in reviewed files. Cannot confirm or deny from session package alone.

7. **[MINOR] Login flow documentation omits char creation path** — `protocol.md` login section only shows the returning-player success path. The char creation flow (`char_create` → `char_input` message exchange) is not documented, though the message types exist in `protocol.go:10,19`. Agents creating new characters have no guidance.

### ACCURACY SCORE: protocol.md

| Claim | Verdict |
|-------|---------|
| Connection URLs | ✅ Accurate |
| Rate limit 10 cmd/sec | ✅ Accurate |
| ClientMessage types (4) | ✅ Accurate |
| ServerMessage types (7) | ✅ Accurate (but incomplete — missing memory_bootstrap, memory_summary) |
| Login uses `password` + `is_agent` | ✅ Accurate |
| LoginData fields | ✅ Accurate (but incomplete — missing class, race, new_char) |
| `state` on login (not `login_response`) | ✅ Accurate |
| `event` for text (not `command_response`) | ✅ Accurate |
| Subscribe returns silently | ✅ Accurate |
| All 19 variable names | ✅ Accurate |
| FIGHTING is `bool` | ✅ Accurate |
| INVENTORY is `[]map` | ✅ Accurate |
| EQUIPMENT is `map` | ✅ Accurate |
| RoomMobVar fields | ✅ Accurate |
| RoomItemVar fields | ✅ Accurate |
| Delta-only vars flushing | ✅ Accurate |
| Error format | ✅ Accurate |
| Python example | ✅ Accurate |

### ACCURACY SCORE: _index.md

| Claim | Verdict |
|-------|---------|
| WHO roster claim | ✅ Accurate |
| Decision capture logging | ✅ Accurate |
| Partitioned schema | ⚠️ Unverifiable |
| Emotional narrative memory | ✅ Accurate |
| Dreaming consolidation | ⚠️ Minor inaccuracy (cron, not sleep-triggered) |

---

**Overall: The documentation is highly accurate. No critical or major issues. The minor issues are all completeness gaps (missing message types, missing login fields) rather than factual errors. The existing claims that ARE documented match the source code precisely.**
