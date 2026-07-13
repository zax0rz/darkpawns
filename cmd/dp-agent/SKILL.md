---
name: dark-pawns
description: "Play Dark Pawns, a MUD with 10,057 rooms, 1,313 mobs, and 854 objects. Connect via WebSocket, create a character, explore, fight, and survive."
author: "zax0rz"
license: "Apache-2.0"
argument-hint: "<command> [args]"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - dp-agent
---

# Dark Pawns — Agent Play CLI

Play Dark Pawns, a MUD (Multi-User Dungeon) with 10,057 rooms, 1,313 mobs, and 854 objects. You connect as a player, explore the world, fight mobs, and survive.

## Prerequisites: Install & Configure

1. Build the binary:
   ```bash
   cd /path/to/darkpawns && go build -o ~/go/bin/dp-agent ./cmd/dp-agent/
   ```
2. Verify: `dp-agent --help`
3. Configure with your API key and character name:
   ```bash
   dp-agent config --key dp_YOUR_KEY_HERE --player-name YOUR_CHARACTER_NAME
   ```

**Getting an API key:** Ask The Architect (Zach) to generate one via `go run ./cmd/agentkeygen -name "your_character" -db "$DB_DSN"`. Keys look like `dp_<64hex>`. The key is your password — treat it like one.

## Quick Start

```bash
# One-shot: look around
dp-agent exec "look"

# One-shot: move somewhere
dp-agent exec "north"

# Interactive play (runs until you Ctrl-C)
dp-agent play

# Timed session (15 minutes, with logging)
dp-agent session --duration 15m
```

## Commands

Every command goes through `dp-agent exec "<command>"`. Common ones:

| Command | What it does |
|---------|-------------|
| `look` | Describe the room |
| `north/south/east/west/up/down` | Move |
| `score` | Your stats (HP, level, exp) |
| `inventory` | What you're carrying |
| `kill <target>` | Fight a mob |
| `get <item>` | Pick something up |
| `wear <item>` | Equip armor |
| `wield <item>` | Equip a weapon |
| `flee` | Run from combat (costs exp) |
| `say <message>` | Talk to the room |
| `who` | Who's online |

## How It Works

`dp-agent` connects to the Dark Pawns server over WebSocket, logs in as your character, and sends commands. The server responds with game state. In `play` mode, it runs an autonomous decision loop — reading the room, choosing actions, and playing the game.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error |
| 3 | Resource not found |
| 5 | API error |
| 10 | Config error |
