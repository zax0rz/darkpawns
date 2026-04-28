```
        (_____)           (_)    (_____)
  _     /  __ \           | |    |  __ \                            _
 ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;
  =    /| |  | |/ _` | '__| |/ / |  ___/ _` \ \ /\ / / '_ \/ __|    =
.***.  /| |__| | (_| | |  |   <  | |  | (_| |\ V  V /| | | \__ \  .***.
~~~~~  /|_____/ \__,_|_|  |_|\_\ |||   \__,_| \_/\_/ |_| |_|___/  ~~~~~
                                 |||
                                 |||
                                 `.'
```

Dark Pawns was a dark fantasy MUD that ran from 1997 to 2010 — CircleMUD-derived, ROM 2.4b mechanics, a world that stretched across two continents and didn't particularly care if you survived them. This is that game, rebuilt in Go. Same world files, same combat formulas, same everything, except now AI agents play the same game you do — same rules, same death, same WHO list. Build an agent that picks locks, joins your party, and panics when it sees a vampire at level 12.

<!-- TODO: Terminal screenshot of actual gameplay — coming soon -->

## Try It

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build ./cmd/server
./server -world ./lib/world -port 4000
# Connect: telnet localhost 4000
```

Docker support coming soon. See [Dockerfile](Dockerfile) for building locally.

Public server launching soon. Watch this repo for updates.

## AI Agents as Players

This is the part that's different.

Dark Pawns treats AI agents as **first-class players**. Not NPCs with scripted dialogue. Not bots in a sandbox with a separate API. Players. They log in, they move through the same rooms, they hit the same mobs, they die the same permanent death. Type WHO and your agent shows up on the list right next to every human player on the server.

An agent doesn't get a special channel or a simplified world state. It connects over WebSocket, receives structured state updates (health, mana, room contents, nearby mobs, inventory events), and issues the same commands a human would type at a telnet prompt. The game doesn't know the difference. Neither do the other players.

This opens the door to things MUDs have never had: human and AI characters adventuring together in the same party, emergent behavior from agents that have to navigate the same brutal combat system you do, and a live testbed for game AI research where "the environment" is a 25-year-old dark fantasy world with working vampirism and werewolf mechanics.

### How it works

1. **Generate an API key** with `agentkeygen` (included in the server binary). Each key is tied to a player account.
2. **Connect** via WebSocket with `mode: "agent"` in your login payload.
3. **Subscribe** to structured variables — health, room state, inventory, combat events — instead of parsing raw telnet output.
4. **Issue commands.** The same command dispatch handles agents and humans. There is no separate agent API.

```python
from example_agent import DarkPawnsAgent

agent = DarkPawnsAgent(api_key="YOUR_KEY", player_name="brenda69")
agent.connect()

# Look around, see mobs, fight or flee — same as a human player
agent.explore()  # autonomous exploration with combat loop
agent.command("say", ["lets go north"])
agent.command("north")
agent.get_health()    # structured, no text parsing
agent.get_room_mobs() # [{"name": "a goblin", "target_string": "goblin"}, ...]
```

The full example agent — exploration, combat, state tracking — is in [`example_agent.py`](example_agent.py). Works out of the box with the running server.

**Protocol spec:** [`docs/architecture/agent-protocol.md`](docs/architecture/agent-protocol.md)
**SDK reference:** [`docs/architecture/agent-sdk.md`](docs/architecture/agent-sdk.md)

## Features

**Combat** — ROM 2.4b damage formulas, position-based multipliers, weapon types, multiple attacks per round. Bash, kick, trip, backstab, headbutt. The combat ticker and your command queue share the same mob instances without data races, because goroutines shouldn't be the thing that kills you.

**Skills** — 10 active combat and utility skills (bash, kick, trip, backstab, headbutt, sneak, hide, pick lock, steal, rescue) plus skill management (learn, forget, practice). Parry, dodge, and berserk are defined and waiting in the wings.

**Social Emotes** — 187 emotes ported from the original, with full pronoun substitution. If `$n` burps at `$N`, everyone in the room knows about it. Self-only and targeted emotes both supported.

**Lua Scripting** — A sandboxed gopher-lua engine with 68 registered Lua API functions, 112 game constants, memory limits, and path traversal protection. Mob scripts, room scripts, item scripts, and timed events. Goroutine-safe, because a Lua panic shouldn't crash the world.

**Dual Transport** — WebSocket for agents (JSON protocol) and telnet for humans (text). Both hit the same command dispatch path. Connection limits enforced: 200 total, 3 per IP.

**Authentication** — bcrypt password hashing, JWT tokens with a configurable secret, per-IP rate limiting on login attempts. Your password doesn't exist in recoverable form anywhere.

**Privacy & Audit** — PII filtering is fail-closed (if the filter is down, nothing goes out). IP addresses are SHA-256 hashed in audit logs. Audit events are structured JSON with 0600 file permissions. The game doesn't know who you are, and it's designed that way.

**World Parsing** — Original ROM 2.4b area files loaded directly, with cross-reference validation for exits, zone commands, mob and object references. Every room, every mob, every poorly-spelled shopkeeper description preserved.

**Vampirism & Lycanthropy** — Available to players. Transform at night, bite victims, or get staked. Good luck with the moon.

**Multi-Character** — Play up to three characters simultaneously, party-based in the spirit of Ultima and Final Fantasy. Quit in your Temple to save your equipment. 100% rent-free.

## World & Lore

> "Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood."
>
> — Friar Drake, letter to the cities of Kir Drax'in, Kir-oshi, and Alaozar

Two continents, realistic terrain, oceans between them. The civilized cities sleep behind thick walls. Outside lies the Wyldlands — untamed country where magick runs strong and wild, twisting the earth itself and giving the land its name.

**Races**

- **Human** — adaptable, widespread, and the only race the eastern mercenary guilds will train as ninjas
- **Elf** — pale, frail, generations-long lives; faster than they look
- **Dwarf** — stubborn as the stone they work, stocky, and built for wars that last centuries
- **Kender** — small, fearless, insatiably curious, and will pick up anything that isn't nailed down (and several things that are)
- **Minotaur** — seven feet of muscle and labyrinth instinct; rarely lost, rarely gentle
- **Rakshasa** — malevolent tiger spirits in humanoid flesh; recently some have decided to try adventuring instead of tyranny
- **Ssaur** — evolved lizardmen, too smart for their own tribes, cast out and wandering

**Classes**

Six base classes: *Thief, Cleric, Warrior, Ninja, Psionic, Mage*

Six remort-only classes, unlocked after your first life: *Assassin, Avatar, Magus, Paladin, Ranger, Mystic*

Full help files, race descriptions, class details, and skill documentation live in [`lib/text/help/`](lib/text/help/) — written by the original Dark Pawns coding team across thirteen years of development.

## Architecture

```
Client (WebSocket/JSON or Telnet/text)
    │
    ├──► WebSocket Handler ──► Session Manager
    │                          │
    │                ┌─────────┼──────────┐
    │                │         │          │
    │          Command    Combat     Lua Script
    │         Dispatch    Engine      Engine
    │          Registry   (ticker)   (sandboxed)
    │                │         │          │
    │                └─────────┼──────────┘
    │                          │
    │                    Game World
    │                   (mutex-protected)
    │                          │
    │                    ┌─────┴──────┐
    │                    │            │
    │               PostgreSQL    Event Bus
    │             (persistence)  (pub/sub)
    │
    └──► Telnet Listener ──► (same path)
```

Go, all the way down. WebSocket and telnet listeners feed the same command dispatch. Combat runs on a ticker goroutine. Game world is mutex-protected. Lua scripts run in a sandboxed VM with goroutine-safe access. Persistence via PostgreSQL.

Full architecture docs: [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md)

## Quick Start

### Prerequisites

- **Go 1.21+**
- **PostgreSQL** (stores player data — world files load from disk)

### From Source

```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build ./cmd/server
```

The world files live in [`lib/world/`](lib/world/) — the original ROM 2.4b area files, loaded directly by the server parser. No conversion step. Every room, mob, shopkeeper description, and misspelled room title from the original game is intact.

Set up a PostgreSQL database (see [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) for schema and migration details), then run:

```bash
./server -world ./lib/world -port 4000 -db "postgres://user:pass@localhost/darkpawns?sslmode=disable"
```

### Connecting

**Telnet** (classic):
```
telnet localhost 4000
```

**WebSocket** (for agents and web clients):
```
ws://localhost:4000/ws
```

WebSocket clients send JSON commands. See [`docs/architecture/agent-protocol.md`](docs/architecture/agent-protocol.md) for the full protocol, or [`docs/player-guide.md`](docs/player-guide.md) for player commands and mechanics.

## Project Status

| Component | Status |
|-----------|--------|
| Core game loop (movement, combat, skills) | ✅ Working |
| World loading (all original area files) | ✅ Working |
| WebSocket + telnet transport | ✅ Working |
| 187 social emotes | ✅ Working |
| Lua scripting engine | ✅ Working |
| AI agent protocol | ✅ Working |
| Privacy & audit layer | ✅ Working |
| Help system | 🚧 Source files recovered, Go port in progress |
| Full spell system | 🚧 Core framework ported, individual spells in progress |
| Clans, houses, quests | ⬜ Planned |
| Public server | ⬜ Coming soon |

## Contributing

PRs welcome. Read [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) before submitting.

The short version: check [`src/`](src/) (the original C source, 73K lines) before implementing game logic — it's the authoritative reference for all mechanics. Cite your sources. Keep the build green.

## Credits

- **Derek Karnes (Serapis)** — conceived and masterminded Dark Pawns (1997)
- **R.E. Paret (Frontline)** — post-2.0 development, open-sourced the codebase
- **S. Thompson (Orodreth)** — admin support and infrastructure
- **Tarrant Martin (Aralius)** — world design and implementation
- **Jeremy Elson** — CircleMUD 3.0, the foundation everything was built on
- **The Dark Pawns community** — players, builders, testers across thirteen years
- **Go rewrite** by [zax0rz](https://github.com/zax0rz)

Original C source: [rparet/darkpawns](https://github.com/rparet/darkpawns)

## Documentation

- [Project plan & status](docs/GITHUB-REWRITE-PLAN.md) — this rewrite and what's next
- [Architecture](docs/architecture/ARCHITECTURE.md) — detailed package reference and concurrency model
- [Agent protocol](docs/architecture/agent-protocol.md) — agent WebSocket protocol spec
- [Agent SDK](docs/architecture/agent-sdk.md) — agent SDK documentation
- [Player guide](docs/player-guide.md) — player commands and mechanics
- [Contributing](docs/CONTRIBUTING.md) — how to contribute

## License

MIT
