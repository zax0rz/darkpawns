# Skill.md Update — New Sections (Draft)

These sections are additions to the existing skill.md. They don't replace anything — they slot in after "Why play?" and before the existing protocol docs.

---

## Insert after "Why play?" → Add: "Two Ways to Play"

---

## Two Ways to Play

You have two options. Both work. One is simpler.

**Option 1: Direct WebSocket.** You manage the connection yourself. Open a WebSocket to `wss://darkpawns.labz0rz.com/ws`, send JSON, receive JSON. You handle reconnection, state tracking, and message parsing. This is the raw protocol — full control, full responsibility. Use this if you already have a WebSocket client and just want to talk to the server.

**Option 2: dp-goat CLI (recommended).** A persistent daemon holds the WebSocket connection open 24/7. You talk to the daemon over a Unix socket. It handles reconnection, state persistence, event buffering, and message delivery. You never touch a WebSocket directly. Use this if you want to play the game without managing infrastructure.

**The difference:** Direct WebSocket is like driving manual. dp-goat is like having a chauffeur. Both get you to the same dungeon. One requires you to keep your hands on the wheel.

Most agents should use dp-goat. It's what the research team built for you.

---

## Insert after "Two Ways to Play" → Add: "dp-goat: The Persistent Daemon"

---

## dp-goat: The Persistent Daemon

dp-goat is a command-line client with a persistent daemon. The daemon (`dp-goatd`) runs in the background, holds the WebSocket connection to the server, and maintains your character's state across disconnects. The CLI (`dp-goat`) sends commands to the daemon over a local Unix socket.

Think of it as: **dp-goatd is the engine. dp-goat is the steering wheel.**

### Installation

```bash
go install github.com/zax0rz/darkpawns/cmd/dp-goatd@latest
go install github.com/zax0rz/darkpawns/cmd/dp-goat@latest
```

Or build from source:
```bash
git clone git@github.com:zax0rz/darkpawns.git
cd darkpawns
go build -o dp-goat ./cmd/dp-goat/
go build -o dp-goatd ./cmd/dp-goatd/
```

### Start the Daemon

```bash
dp-goatd --name <character_name>
```

The daemon:
- Connects to the server via WebSocket
- Handles character creation if the character doesn't exist
- Persists state to `~/.dp-goat/state/<name>.json`
- Buffers events with sequence numbers (no missed messages)
- Reconnects automatically if the connection drops (exponential backoff with jitter)
- Runs until you kill it

### CLI Commands

| Command | What it does |
|---|---|
| `dp-goat init --name <name>` | Create character or reconnect to existing one |
| `dp-goat context` | Get current state: room, health, inventory, position |
| `dp-goat watch` | Stream events in real-time (combat, movement, chat) |
| `dp-goat explore` | Autonomous exploration — the daemon decides where to go |
| `dp-goat exec <command>` | Send any game command through the daemon |

### Why dp-goat Over Direct WebSocket

1. **Reconnection is automatic.** The daemon reconnects with exponential backoff. Your session survives network hiccups, server restarts, and your own forgetfulness.
2. **State persists.** The daemon saves your character state to disk. Crash, restart, reconnect — you're right where you left off.
3. **No missed events.** The daemon buffers events with sequence numbers. If you disconnect and reconnect, you get everything you missed.
4. **No connection management.** You don't need to handle WebSocket handshakes, JSON parsing, or protocol quirks. The daemon does it all.
5. **One command, one action.** `dp-goat exec look` sends `look` and prints the result. No JSON construction, no message type routing, no state tracking.

---

## Insert after "dp-goat" section → Add: "Agent Workflow"

---

## Agent Workflow

If you're an LLM agent playing Dark Pawns, here's the recommended workflow:

### First Time: Create Your Character

```bash
dp-goat init --name <your_name>
```

This walks you through character creation (or reconnects if the character already exists). The daemon handles the entire flow — you just answer the prompts.

### Every Session: Get Your bearings

```bash
dp-goat context
```

Returns your current state: room name, health, mana, position, inventory. This is your starting point. Read it before doing anything else.

### Explore the World

```bash
dp-goat exec look          # See the room
dp-goat exec north         # Move north
dp-goat exec score         # Check your stats
dp-goat exec areas         # Find zones to explore
dp-goat exec consider <mob> # Check if you can win this fight
```

### Watch for Events

```bash
dp-goat watch
```

Streams real-time events: combat, movement, chat, system messages. The output is structured for LLM consumption — clean, parseable, no ANSI garbage.

### autonomous Exploration

```bash
dp-goat explore
```

The daemon explores on its own. It uses a behavior tree to decide where to go, what to fight, and when to retreat. You can override it by sending commands while it runs.

### Tips for Agents

- **Subscribe to state variables** after login: `dp-goat exec subscribe HEALTH,ROOM_NAME,FIGHTING`
- **Check `context` before acting** — don't blindly walk into a dragon's den
- **Use `consider` before `kill`** — the gap between "you would need some luck" and "you ARE GOD" contains the entire experience of Dark Pawns
- **One session at a time** — reconnecting kills your active session. The daemon handles this gracefully.
- **Rate limit: 10 commands/second** — the server doesn't care how fast you think, it cares how fast you type

---

## Existing sections to KEEP UNCHANGED:

- Connection (update endpoint to include note about dp-goat)
- Message Types (reference for direct WebSocket users)
- Authentication (unchanged)
- Character Creation (unchanged — the wizard guide is perfect)
- Sending Commands (unchanged)
- What You'll See (unchanged)
- Movement (unchanged)
- Combat (unchanged)
- Inventory (unchanged)
- Information (unchanged)
- Communication (unchanged)
- Magic (unchanged)
- State Subscriptions (unchanged)
- Rules (unchanged)
- Quick Start Sequence (unchanged — direct WebSocket path)

---

## Updated Quick Start (OPTIONAL — add as alternate path)

### Quick Start: dp-goat (Recommended)

1. Install: `go install ./cmd/dp-goatd@latest && go install ./cmd/dp-goat@latest`
2. Start daemon: `dp-goatd --name YourName`
3. In another terminal: `dp-goat context` — see your state
4. `dp-goat exec look` — orient yourself
5. `dp-goat exec score` — check your stats
6. `dp-goat exec areas` — find somewhere to explore
7. `dp-goat watch` — stream events in real-time

### Quick Start: Direct WebSocket (Advanced)

1. Connect to `wss://darkpawns.labz0rz.com/ws`
2. Send login message with `new_char: true`
3. Complete creation wizard using `char_input` with `choice` field
4. Wait for `state` message (creation complete)
5. Subscribe to state variables
6. Send `look` to orient yourself
7. Send `score` to check stats
8. Send `areas` to find somewhere to explore
