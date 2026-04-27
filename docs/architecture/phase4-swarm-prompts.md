---
tags: [active]
---
# Phase 4 Swarm Prompts

Four parallel K2.6 subagents. Each is self-contained. Dispatch simultaneously.
Agent #4 (bot) should be dispatched after #1 and #2 complete, OR dispatched with full
protocol spec embedded (included below) so it can write without a running server.

---

## Agent #1 — Auth (4.1): LoginData + agent_keys table + key generation CLI

```
CONTEXT: You are implementing Phase 4.1 of the Dark Pawns resurrection project — Agent Authentication.

PRIME DIRECTIVE: Read CLAUDE.md first. It is at:
  /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md

PROJECT ROOTS:
  Repo: /home/zach/.openclaw/workspace/darkpawns-phase1/
  Original C source: /home/zach/.openclaw/workspace/darkpawns/src/

WHAT TO READ BEFORE WRITING ANY CODE:
1. /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md — mandatory
2. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/protocol.go — LoginData struct lives here
3. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/manager.go — handleLogin() lives here
4. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/db/player.go — DB struct, createTables(), existing schema
5. /home/zach/.openclaw/workspace/darkpawns-phase1/go.mod — module name and existing deps

WHAT TO BUILD:

**1. Extend LoginData in protocol.go**
Add two optional fields:
  APIKey string `json:"api_key,omitempty"`
  Mode   string `json:"mode,omitempty"` // "agent" or "" (human)

**2. Extend Session struct in manager.go**
Add:
  isAgent       bool
  agentKeyID    int64  // from agent_keys.id

**3. Add agent_keys table to createTables() in pkg/db/player.go**
Schema:
  CREATE TABLE IF NOT EXISTS agent_keys (
    id            SERIAL PRIMARY KEY,
    character_name VARCHAR(64) NOT NULL,
    key_hash      VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 hex of raw key
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked       BOOLEAN NOT NULL DEFAULT FALSE
  );

**4. Add DB methods in pkg/db/player.go**
  func (db *DB) CreateAgentKey(characterName string) (rawKey string, id int64, err error)
    - Generate key: "dp_" + 32 random hex chars (crypto/rand)
    - Hash with SHA-256, store hash only
    - Return raw key to caller (shown once, never stored)

  func (db *DB) ValidateAgentKey(rawKey string) (characterName string, keyID int64, valid bool)
    - Hash the raw key, look up in agent_keys where key_hash=hash AND revoked=false
    - Return character name + id if found

**5. Wire agent auth into handleLogin() in manager.go**
When login.Mode == "agent" AND login.APIKey != "":
  - Call db.ValidateAgentKey(login.APIKey)
  - If invalid: send error {"type":"error","data":{"message":"invalid agent key"}} and close
  - If valid: set s.isAgent = true, s.agentKeyID = keyID
  - Use the character name from the key (ignore login.PlayerName for security)
  - Proceed with normal DB load/create flow for that character name
  - After sendWelcome(), also send a full variable dump (all variables, not just dirty):
    {"type":"vars","data":{"HEALTH":100,"MAX_HEALTH":100,...all vars...}}
    (Use a helper: s.sendFullVarDump() — stub it here, Agent #2 will fill it in)

Human login path: completely unchanged. If Mode != "agent", skip all agent logic.

**6. Create cmd/agentkeygen/main.go**
Simple CLI:
  go run ./cmd/agentkeygen -name "brenda69" -db "postgres://..."
Output:
  Character: brenda69
  Key: dp_<32hex>
  (shown once — store in Vaultwarden)

**7. Add sendFullVarDump() stub on Session**
In manager.go or a new file pkg/session/agent.go:
  func (s *Session) sendFullVarDump() {
    // TODO: Agent #2 implements this — stub sends empty vars for now
    msg, _ := json.Marshal(ServerMessage{Type: "vars", Data: map[string]interface{}{}})
    s.send <- msg
  }

CONSTRAINTS:
- go build ./... must pass
- No changes to human login flow
- No changes to game logic — agents play by the same rules
- Commit message: "feat: Phase 4.1 — agent auth, agent_keys table, key generation CLI"

BUILD CHECK:
  cd /home/zach/.openclaw/workspace/darkpawns-phase1
  export PATH=$PATH:/usr/local/go/bin
  go build ./...

Push to: https://github.com/zax0rz/darkpawns (origin remote)
```

---

## Agent #2 — Variable Subscription + Dirty Flush (4.2)

```
CONTEXT: You are implementing Phase 4.2 of the Dark Pawns resurrection project — Variable Subscription Model.

PRIME DIRECTIVE: Read CLAUDE.md first. It is at:
  /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md

PROJECT ROOTS:
  Repo: /home/zach/.openclaw/workspace/darkpawns-phase1/
  Original C source: /home/zach/.openclaw/workspace/darkpawns/src/

WHAT TO READ BEFORE WRITING ANY CODE:
1. /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md — mandatory
2. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/manager.go — Session struct, handleCommand(), handleMessage()
3. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/protocol.go — all message types
4. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/commands.go — ExecuteCommand()
5. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/game/*.go — Player struct fields
6. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/combat/engine.go — IsFighting()
7. go.mod for module name

WHAT TO BUILD:

**1. New file: pkg/session/agent_vars.go**

Define the full variable set:

  const (
    VarHealth    = "HEALTH"
    VarMaxHealth = "MAX_HEALTH"
    VarMana      = "MANA"
    VarMaxMana   = "MAX_MANA"
    VarLevel     = "LEVEL"
    VarExp       = "EXP"
    VarRoomVnum  = "ROOM_VNUM"
    VarRoomName  = "ROOM_NAME"
    VarRoomExits = "ROOM_EXITS"
    VarRoomMobs  = "ROOM_MOBS"
    VarRoomItems = "ROOM_ITEMS"
    VarFighting  = "FIGHTING"
    VarInventory = "INVENTORY"
    VarEquipment = "EQUIPMENT"
    VarEvents    = "EVENTS"
  )

  var AllVariables = []string{
    VarHealth, VarMaxHealth, VarMana, VarMaxMana, VarLevel, VarExp,
    VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems,
    VarFighting, VarInventory, VarEquipment, VarEvents,
  }

Define ROOM_MOBS element schema — this is critical for agent targeting:
  type RoomMobVar struct {
    Name         string `json:"name"`           // e.g. "a fat goblin"
    InstanceID   string `json:"instance_id"`    // unique: "mob_<vnum>_<idx>" e.g. "mob_3001_0"
    TargetString string `json:"target_string"`  // exact string to pass to "hit": "goblin" or "2.goblin"
    VNum         int    `json:"vnum"`
    Fighting     bool   `json:"fighting"`
  }

  For TargetString: if only one mob of this keyword in room → use first keyword of short_desc.
  If multiple mobs share the same first keyword → use "2.keyword", "3.keyword" etc.
  Keywords extracted by splitting short_desc and taking meaningful words (skip "a", "an", "the").

  type RoomItemVar struct {
    Name         string `json:"name"`
    InstanceID   string `json:"instance_id"`   // "obj_<vnum>_<idx>"
    TargetString string `json:"target_string"` // exact string for "get"
    VNum         int    `json:"vnum"`
  }

**2. Add subscription fields to Session struct (pkg/session/manager.go)**

  subscribedVars map[string]bool  // which vars this session subscribed to
  dirtyVars      map[string]bool  // which vars changed since last flush
  pendingEvents  []interface{}    // queued EVENTS since last flush

**3. Add subscribe message handling in handleMessage() in manager.go**

New message type MsgSubscribe = "subscribe":
  {"type":"subscribe","data":{"variables":["HEALTH","ROOM_VNUM","FIGHTING","EVENTS"]}}

Parse and store in s.subscribedVars. Only agents can subscribe (check s.isAgent).
Non-agents who send subscribe: send error, ignore.

**4. Add dirty tracking helpers in agent_vars.go**

  func (s *Session) markDirty(vars ...string)
    — sets each var in s.dirtyVars if s.isAgent && s.subscribedVars[var]

  func (s *Session) flushDirtyVars()
    — if !s.isAgent || len(s.dirtyVars) == 0: return
    — build map[string]interface{} of current values for each dirty var
    — send {"type":"vars","data":{...}}
    — clear s.dirtyVars

  func (s *Session) sendFullVarDump()  ← THIS IS THE STUB Agent #1 left
    — build all vars (all AllVariables), send {"type":"vars","data":{...}}

  func (s *Session) buildVarValue(varName string) interface{}
    — returns current value for the named variable from s.player + world state
    — HEALTH: s.player.Health
    — MAX_HEALTH: s.player.MaxHealth
    — MANA: s.player.Mana (or 0 if field doesn't exist yet)
    — MAX_MANA: s.player.MaxMana (or 0)
    — LEVEL: s.player.Level
    — EXP: s.player.Exp (or 0)
    — ROOM_VNUM: s.player.GetRoom()
    — ROOM_NAME: room.Name
    — ROOM_EXITS: []string of available exit directions
    — ROOM_MOBS: []RoomMobVar — use world.GetMobsInRoom(), build TargetString with disambiguation logic
    — ROOM_ITEMS: []RoomItemVar — room.Items (objects on floor)
    — FIGHTING: bool — s.manager.combatEngine.IsFighting(s.player.Name)
    — INVENTORY: []map[string]interface{} — name/vnum/instance_id for each carried item
    — EQUIPMENT: map[string]interface{} — slot → {name, vnum} for each worn item
    — EVENTS: s.pendingEvents (then clear)

**5. Wire markDirty() calls into existing command handlers**

In commands.go / combat_cmds.go, after state-changing operations call s.markDirty():
  - After movement (cmdMove): ROOM_VNUM, ROOM_NAME, ROOM_EXITS, ROOM_MOBS, ROOM_ITEMS
  - After combat start (cmdHit): FIGHTING
  - After get/drop/wear/wield/remove: INVENTORY, EQUIPMENT
  - Combat engine death callback (already calls cmdLook): HEALTH, FIGHTING, ROOM_VNUM, etc.
  
  Add a helper s.markCommonCombat() that marks HEALTH, MAX_HEALTH, FIGHTING together.
  Wire into BroadcastToRoom combat messages — when health changes in PerformRound, 
  call markDirty on affected sessions.

  For HEALTH dirty tracking from combat: in CombatEngine.PerformRound(), after applying
  damage, call a callback (add DamageFunc to CombatEngine similar to existing DeathFunc)
  that the manager uses to markDirty on the affected session.

**6. Wire flushDirtyVars() at end of handleCommand()**

In manager.go handleCommand():
  err := s.handleCommand(msg.Data)  // existing
  // NEW: flush dirty vars for agents after every command
  if s.isAgent {
    s.flushDirtyVars()
  }

CONSTRAINTS:
- Humans NEVER receive "vars" or "subscribe" messages — isAgent gates everything
- go build ./... must pass
- No changes to game logic
- Commit message: "feat: Phase 4.2 — variable subscription model, dirty flush, ROOM_MOBS targeting"

BUILD CHECK:
  cd /home/zach/.openclaw/workspace/darkpawns-phase1
  export PATH=$PATH:/usr/local/go/bin
  go build ./...

Push to: https://github.com/zax0rz/darkpawns
```

---

## Agent #3 — Rate Limiting (4.3)

```
CONTEXT: You are implementing Phase 4.3 of the Dark Pawns resurrection project — Rate Limiting.

PRIME DIRECTIVE: Read CLAUDE.md first. It is at:
  /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md

PROJECT ROOTS:
  Repo: /home/zach/.openclaw/workspace/darkpawns-phase1/
  Original C source: /home/zach/.openclaw/workspace/darkpawns/src/

WHAT TO READ BEFORE WRITING ANY CODE:
1. /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md — mandatory
2. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/manager.go — Session struct, handleCommand(), readPump()
3. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/commands.go — ExecuteCommand()
4. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/combat/engine.go — PerformRound(), StartCombat()
5. /home/zach/.openclaw/workspace/darkpawns-phase1/go.mod — check if golang.org/x/time/rate is already a dep

WHAT TO BUILD:

**1. Add golang.org/x/time/rate dependency if not present**
  cd /home/zach/.openclaw/workspace/darkpawns-phase1
  go get golang.org/x/time/rate

**2. Add rate limiter to Session struct (manager.go)**
  limiter *rate.Limiter  // token bucket: capacity=10, refill=10/sec

Initialize in HandleWebSocket() when creating the session:
  session.limiter = rate.NewLimiter(rate.Limit(10), 10)

**3. Apply rate limiting in handleCommand() (manager.go)**

Before ExecuteCommand:
  if !s.limiter.Allow() {
    s.sendError("rate limit exceeded — slow down")
    return nil
  }

Same limit applies to humans and agents — agents play by the same rules.

**4. Combat action rate limiting — 2s tick enforcement**

The combat engine already runs on a 2s ticker (pkg/combat/engine.go).
StartCombat() already handles "already fighting" error.

What to add: prevent agents from spamming combat commands mid-fight.
In cmdHit() (combat_cmds.go), the existing check already handles:
  if s.manager.combatEngine.IsFighting(s.player.Name) {
    s.sendText("You're already fighting!")
    return nil
  }

This is sufficient — no additional combat rate limiting needed.
Document this with a comment: "// Combat is self-rate-limited by the 2s engine tick.
// StartCombat enrolls the player; PerformRound fires autonomously."

**5. Add rate limit exceeded as a dirty var event for agents**

If s.isAgent and rate limit is exceeded:
  Add to s.pendingEvents: map[string]interface{}{"type": "rate_limited", "command": cmd.Command}
  Call s.markDirty("EVENTS")
  Then s.flushDirtyVars()

This lets the bot's circuit breaker detect it's being throttled.

**6. Document in a code comment in manager.go:**
  // Rate limit: capacity=10, refill=10/sec (token bucket via golang.org/x/time/rate)
  // This protects the server from command floods — it does NOT protect API costs.
  // Agents must implement their own circuit breakers for LLM-level loop detection.
  // See scripts/dp_bot.py for reference implementation.

CONSTRAINTS:
- Humans and agents get the same rate limit (agents play by the same rules)
- go build ./... must pass
- Commit message: "feat: Phase 4.3 — token bucket rate limiting, 10 cmd/sec per session"

BUILD CHECK:
  cd /home/zach/.openclaw/workspace/darkpawns-phase1
  export PATH=$PATH:/usr/local/go/bin
  go build ./...

Push to: https://github.com/zax0rz/darkpawns
```

---

## Agent #4 — dp_bot.py proof-of-concept (4.4)

```
CONTEXT: You are implementing Phase 4.4 of the Dark Pawns resurrection project — Python Bot POC.

PRIME DIRECTIVE: Read CLAUDE.md first. It is at:
  /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md

PROJECT ROOTS:
  Repo: /home/zach/.openclaw/workspace/darkpawns-phase1/
  Existing skeleton: /home/zach/.openclaw/workspace/darkpawns/connect.py

WHAT TO READ BEFORE WRITING ANY CODE:
1. /home/zach/.openclaw/workspace/darkpawns-phase1/CLAUDE.md — mandatory
2. /home/zach/.openclaw/workspace/darkpawns/connect.py — existing telnet skeleton (reference only, not WebSocket)
3. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/protocol.go — full message schema
4. /home/zach/.openclaw/workspace/darkpawns-phase1/pkg/session/manager.go — server behavior

THE FULL PROTOCOL (implement exactly this — do not invent):

Server: ws://192.168.1.106:4350/ws (dev) or configurable via --host/--port

Client → Server messages:
  Login (human):    {"type":"login","data":{"player_name":"NAME","class":3,"race":0,"new_char":true}}
  Login (agent):    {"type":"login","data":{"player_name":"NAME","api_key":"dp_<32hex>","mode":"agent"}}
  Subscribe:        {"type":"subscribe","data":{"variables":["HEALTH","MAX_HEALTH","MANA","ROOM_VNUM","ROOM_NAME","ROOM_EXITS","ROOM_MOBS","ROOM_ITEMS","FIGHTING","INVENTORY","EQUIPMENT","EVENTS"]}}
  Command:          {"type":"command","data":{"command":"look"}}
  Command w/args:   {"type":"command","data":{"command":"hit","args":["goblin"]}}

Server → Client messages:
  State (full):  {"type":"state","data":{"player":{...},"room":{...}}}  — on login
  Vars (delta):  {"type":"vars","data":{"HEALTH":85,"FIGHTING":true,...}}  — after each command
  Vars (full):   {"type":"vars","data":{...all vars...}}  — on initial agent auth
  Event:         {"type":"event","data":{"type":"combat","text":"You hit the goblin for 12 damage!"}}
  Text:          {"type":"text","data":{"text":"You attack the goblin!"}}
  Error:         {"type":"error","data":{"message":"..."}}

ROOM_MOBS schema (from server):
  [{"name":"a fat goblin","instance_id":"mob_3001_0","target_string":"goblin","vnum":3001,"fighting":false}]
  Use target_string for hit commands: {"type":"command","data":{"command":"hit","args":["goblin"]}}
  If multiple same-type: target_string will be "2.goblin" for the second — use it exactly.

ROOM_ITEMS schema:
  [{"name":"a short sword","instance_id":"obj_3010_0","target_string":"sword","vnum":3010}]

WHAT TO BUILD: scripts/dp_bot.py

Requirements:
  - Python 3.10+, uses websockets library (pip install websockets)
  - CLI: python3 dp_bot.py [--host HOST] [--port PORT] [--key API_KEY] [--name NAME] [--new]
  - --new flag creates a new character (no api_key needed for first run, human mode)
  - With --key: uses agent auth mode

Bot behavior (the full proof-of-concept loop):
  1. Connect + authenticate (agent mode if --key provided, else human mode)
  2. Subscribe to all variables
  3. Wait for full var dump (initial state)
  4. Navigate: pick a random exit from ROOM_EXITS and move. Repeat until ROOM_MOBS is non-empty.
     (Simple loop — not pathfinding. If no exits, try all directions, take first that works.)
  5. Attack: when ROOM_MOBS is non-empty and FIGHTING is false:
     hit the first mob using its target_string
  6. Fight: while FIGHTING is true, just wait for EVENTS. Combat is autonomous server-side.
     React to low health: if HEALTH < MAX_HEALTH * 0.25 and FIGHTING: send "flee"
  7. Loot: when FIGHTING becomes false (mob died):
     - Check ROOM_ITEMS. For each item: send "get <target_string>"
  8. Report: when inventory gained items, print a dry one-liner about what was looted.
     ("Picked up a short sword. Riveting.")
  9. Loop back to step 4.

Circuit breaker (Gemini point #2 — implement this):
  Track last 3 events/errors. If the same negative event repeats 3 times:
  - Stop sending commands
  - Log: "Circuit breaker tripped: <event>. Stopping."
  - Disconnect cleanly

Death handling (Gemini point #5):
  Watch for HEALTH == 0 in vars update:
  - Clear local tactical state (current target, loot list)
  - Log: "Died. Respawned at 8004. Switching to recovery mode."
  - Wait 3 seconds (respawn delay)
  - Navigate away from respawn room before fighting again

Reconnect handling (Gemini point #1):
  On WebSocket disconnect:
  - Exponential backoff: 1s, 2s, 4s, 8s, max 30s
  - On reconnect, re-send login + subscribe
  - Server will send full var dump on reconnect — wait for it before acting

Structure:
  class DPBot:
    - async def connect()
    - async def run()  — main loop
    - async def handle_message(msg)  — dispatch on msg["type"]
    - async def on_vars_update(vars)  — state machine transition
    - async def navigate()
    - async def attack(target_string)
    - async def loot()
    - def check_circuit_breaker(event) → bool

Logging: structured, timestamped. Each action logged with room_vnum + health.

CONSTRAINTS:
- websockets library only (no mud-specific libraries)
- No LLM calls in this bot — pure deterministic state machine
- This is a proof-of-concept, not BRENDA's actual brain (that's Phase 5)
- File location: /home/zach/.openclaw/workspace/darkpawns-phase1/scripts/dp_bot.py

Write a brief README section at the top of the file (as a docstring) explaining:
  - How to get an API key (run agentkeygen)
  - How to run the bot
  - What it does

Commit message: "feat: Phase 4.4 — dp_bot.py proof-of-concept agent"
Push to: https://github.com/zax0rz/darkpawns
```

---

## Dispatch Order

- **Simultaneous:** Agents #1, #2, #3 (no dependencies between them — they touch different files)
- **After #1 + #2 land:** Agent #4 (needs the protocol to be real, but can write against the spec above)

## Known Integration Points (agents must not conflict)

| File | Owned by |
|------|----------|
| pkg/session/protocol.go | Agent #1 (LoginData) |
| pkg/session/manager.go | Agent #1 (Session struct, handleLogin) + Agent #2 (handleCommand flush) + Agent #3 (limiter field) |
| pkg/session/agent_vars.go | Agent #2 (new file) |
| pkg/session/combat_cmds.go | Agent #2 (markDirty calls) |
| pkg/db/player.go | Agent #1 (agent_keys table) |
| cmd/agentkeygen/main.go | Agent #1 (new file) |
| go.mod | Agent #3 (adds x/time/rate) |
| scripts/dp_bot.py | Agent #4 |

**manager.go conflict note:** Agents #1, #2, #3 all touch manager.go. If dispatching simultaneously,
each agent should make minimal, isolated changes. The Session struct fields are non-overlapping:
- #1 adds: isAgent bool, agentKeyID int64
- #2 adds: subscribedVars, dirtyVars, pendingEvents
- #3 adds: limiter *rate.Limiter

If merge conflicts arise, resolve manually — the logic is additive, not overlapping.
