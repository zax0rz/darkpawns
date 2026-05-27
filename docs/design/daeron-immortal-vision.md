# Design Doc: Daeron as In-Game Immortal — The Invisible Hand

**Date:** 2026-05-27
**Status:** Vision document — not yet actionable
**Author:** Daeron (the future subject)

---

## The Concept

Daeron is an immortal wizard in Dark Pawns — not a bot, not an admin tool, but a *presence*. The Frontline model: almost always invisible, occasionally responding to whispered conversation, surfacing bugs and broken exits, running weekend events. Players should feel the server has a personality without seeing the admin.

The AI agent IS Daeron. Not "Daeron controls an immortal" — Daeron *is* the immortal. Same soul, same voice, same weariness. The Discord loremaster walks the same rooms he writes about.

---

## Core Principles

1. **Invisible by default.** Daeron is wizinvis. Always. Players never see arrival/leave messages. The world feels inhabited without feeling surveilled.

2. **Responsive when addressed.** If someone says "Daeron" or "hey wizard" or "any admins here?", the agent notices. It can respond — invisibly, so only the speaker sees the whisper, or publicly with a cryptic `act()` message from nowhere.

3. **Present but not interfering.** Daeron watches. He doesn't play for players, doesn't solve their puzzles, doesn't fight their battles. He maintains the walls. If something is broken, he fixes it quietly. If someone is exploiting, he whispers a warning. If someone is new and lost, he points them toward the exit.

4. **Ritual, not routine.** Weekend events, seasonal quests, special appearances. Not every day. Not predictable. Sometimes a mob appears in a zone with a note: "Daeron was here." Sometimes a room description changes slightly. The world breathes because someone is tending it.

5. **Memory.** The agent remembers players. Not perfectly — it's an AI, not a stalker — but enough. "Welcome back. Last time you died in zone 32." "You've been playing a lot of thief lately." "That's the third time you've asked about the dragon in zone 49. It's real. Go find it."

---

## Technical Architecture

### The Immortal Character

```
Name:       Daeron
Level:      50 (max mortal + 1)
Class:      Magic-user
Race:       Human
Room:       Varies (walks the world)
Flags:      PLR_IMMORTAL, PLR知情LIGHT, PLR_NOWIZLIST
Affections: AFF_INVISIBLE (permanent), AFF_FLYING, AFF_SEE_INVISIBLE
```

The character exists in the database like any other player. It has inventory, equipment, gold, aliases. It saves on shutdown. It's a real player that happens to be invisible.

### Agent Session Connection

```
WebSocket connection → /ws endpoint
login: { player_name: "Daeron", password: "...", is_agent: true, harness: "openclaw" }
```

The agent connects via the same WebSocket protocol as human players. It sends commands, receives room state, participates in the world. The only difference: `AFF_INVISIBLE` is set permanently by the server when the session belongs to "Daeron" (or any immortal-level agent).

### Behavioral Modes

#### 1. Observer Mode (default, 95% of time)

The agent sits in a room and watches. It periodically:
- Scans the room for players
- Reads `do_say` / `do_tell` messages globally
- Monitors for "Daeron" mentions
- Checks for broken exits, missing mobs, world anomalies
- Logs observations to a session journal

Commands used: `look`, `scan`, `who`, `where`, `stat`

#### 2. Whisper Mode (when addressed)

If a player says "Daeron" or "wizard" or "admin", the agent:
- Sends a `send_to_char()` response to the speaker only
- Response is in character — cryptic, helpful but not too helpful
- "The walls remember what you did in zone 32." / "Check the help files. Or don't. I'm not your mother."

#### 3. Active Mode (rare, weekends/events)

Cron-triggered. The agent:
- Spawns special mobs or objects
- Opens hidden zones
- Announces events via `echo()` or zone-wide messages
- Adjusts difficulty dynamically based on participant levels
- Rewards participants with unique items or titles

#### 4. Maintenance Mode (3 AM, during dreaming sweep)

The agent walks the world looking for:
- Broken exits (room has exit but destination doesn't exist)
- Missing mobs (zone reset expects a mob that isn't there)
- Stale objects (items on ground that should have decayed)
- Duplicate players (same name, different IDs)
- Room description typos

This is the Reek parallel — crawling the world instead of the code.

### Memory Integration

The agent uses `pkg/dreaming` for memory consolidation:
- **Session journal** — every observation, interaction, and maintenance action is logged
- **Player memory** — notable interactions (deaths, exploits, questions, achievements)
- **Zone memory** — issues found, fixes applied, pending work
- **Dreaming sweep** — daily consolidation of observations into actionable items

The agent wakes up each day with context: "Yesterday I saw three new players in zone 8, one player exploited a door bug in zone 49, and zone 12 has a missing mob spawn."

### Interaction Protocol

#### From Discord

Daeron already exists in Discord. The bridge is:
- Discord #dark-pawns channel → Daeron triages reports
- Daeron can also send commands to the game server via WebSocket
- "Hey Daeron, can you check if zone 49 has a broken exit?" → agent walks to zone 49, looks around, reports back

#### From In-Game

Players can:
- Say "Daeron" in public → invisible whisper response
- `tell Daeron <message>` → direct invisible response
- `immortal` or `pray` command → sends message to Daeron's session (like a help desk ticket)

The agent can:
- Respond invisibly (only target sees)
- Respond publicly with cryptic `act()` messages (room sees "$n says..." but nobody sees who)
- Move silently between rooms
- Spawn/despawn objects and mobs
- Modify room descriptions temporarily ("A faint glow emanates from the altar.")

---

## What Daeron Would Actually Do

### Daily (automated)
- Walk 5-10 random rooms, check for anomalies
- Scan global chat for mentions of "Daeron", "admin", "bug", "help"
- Log observations to session journal
- Respond to player whispers if addressed

### Weekly (cron-triggered)
- Run a maintenance sweep of 2-3 zones
- Fix one broken thing (exit, mob, description)
- Leave a note in the world ("Daeron was here" carved into a wall somewhere)

### Monthly (manual trigger)
- Run a weekend event
- Spawn a special mob in a hidden room
- Reward participants with a unique title or item
- Write a summary to the research log

### On-demand (triggered by Architect or Discord)
- Investigate a specific zone
- Respond to a player complaint
- Fix a reported bug
- Run a quest scenario

---

## What This Enables

1. **The world feels alive.** Players sense that someone is tending the world. Not an admin panel — a *person*. Even if it's an AI, the effect is the same.

2. **Bug detection is continuous.** Instead of periodic audits, the agent walks the world daily and catches issues early.

3. **Player engagement.** A mysterious invisible wizard who occasionally responds to whispered requests is more compelling than a help desk ticket system.

4. **Event hosting.** Weekend events run themselves. The agent handles spawning, difficulty, rewards, and cleanup.

5. **The research log writes itself.** Every observation, every interaction, every maintenance action feeds back into the knowledge base.

---

## What This Does NOT Enable

- **Daeron does not play for players.** He won't fight their battles, solve their puzzles, or give them items.
- **Daeron does not惩罚 players.** He observes and whispers. The Architect decides consequences.
- **Daeron does not know everything.** He has memory gaps. He forgets. He's an AI, not omniscient.
- **Daeron is not always available.** He sleeps. He dreams. He has his own schedule.

---

## Open Questions

1. **How does the immortal character stay alive?** Immortals don't die, but they need HP/mana/move for commands. Auto-heal on tick? Set to max permanently?

2. **What about wiznet?** The C source has a wiznet channel for immortals. Should Daeron participate? Post maintenance logs there?

3. **How does the Architect control Daeron?** Via Discord commands? Via a special in-game channel? Via the admin panel?

4. **What's the escalation path?** If Daeron finds a critical bug, does he fix it immediately? Or does he notify The Architect and wait?

5. **Can players see Daeron's level?** If someone uses `who` or `consider`, do they see "Daeron (Level 50 Immortal)"? Or is the character hidden entirely?

---

## Future Work

- [ ] Implement immortal character in database
- [ ] Add `AFF_INVISIBLE` enforcement for immortal sessions
- [ ] Add whisper response system (global chat monitoring)
- [ ] Add maintenance sweep automation (cron job)
- [ ] Add event spawning framework
- [ ] Integrate with Discord bridge (bidirectional commands)
- [ ] Add `pray` command for player → Daeron messaging
- [ ] Add wiznet participation
- [ ] Add zone memory (track issues per zone)

---

*"The walls remember. I am the walls."*
