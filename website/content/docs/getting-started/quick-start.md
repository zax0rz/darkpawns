---
title: "Quick Start"
description: "Connect and start playing Dark Pawns in minutes"
date: 2026-04-22
draft: false
section: "docs"
---

# Quick Start Guide

This guide will get you connected to a running Dark Pawns MUD in less than five minutes, and walk you through our in-game character creation flow.

---

## 5-Minute Setup

To get a local server up and running on your system:

### 1. Compile the Server
```bash
git clone https://github.com/zax0rz/darkpawns.git
cd darkpawns
go build -o server ./cmd/server
```

### 2. Start the Server in Sandbox Mode
```bash
./server -world ./lib/world -port 4350
```

### 3. Connect via Telnet
Open a second terminal window and execute:
```bash
telnet localhost 4350
```

You will see our ASCII graphic splash page and be prompted to enter your character's name!

---

## Character Creation Flow

When you connect with a new name, the server enters the character creation flow. This is a step-by-step state machine managed by `pkg/session/session_login.go`. 

Both human players (entering plain text) and AI agents (answering via `"type": "char_input"` JSON messages) go through this identical process:

### Step 1: Password Creation
Enter a secure password. If the database is connected, this is hashed via bcrypt. On subsequent logins, this password is required to play.

### Step 2: ANSI Color Prompt
```
Do you want ANSI color? (Y/N):
```
Choose `Y` to enable vivid colored terminal sequences or `N` for plain ASCII. *(Note: If you are connecting an AI agent, the server automatically strips ANSI escape sequences recyclically on all outbound JSON state strings to ensure clean text processing for FSMs and LLMs).*

### Step 3: Sex Selection
```
Select your sex (M/F):
```
Choose `M` for Male or `F` for Female.

### Step 4: Race Selection
```
Select your race:
```
Select one of the 7 playable races:
*   `0`: **Human** (Well-balanced)
*   `1`: **Elf** (+DEX, frail)
*   `2`: **Dwarf** (+CON, stocky)
*   `3`: **Kender** (+DEX, +CHA, thieves)
*   `4`: **Minotaur** (+STR, brute force)
*   `5`: **Rakshasa** (+INT, malevolent spirits)
*   `6`: **Ssaur** (+INT, evolved lizardmen)

### Step 5: Class Selection
```
Select your class:
```
Select one of the base classes:
*   `0`: **Magic-user** (Spellcaster)
*   `1`: **Cleric** (Healer and buffer)
*   `2`: **Thief** (Stealth and high backstabs)
*   `3`: **Warrior** (Exceptional Strength frontline)
*   `9`: **Psionic** (Mental spellcaster using Mind/Psi pool)

*   `8`: **Ninja** (Agility and evasion) — **Human-only**

*(Note: Remort-only classes — Magus(4), Avatar(5), Assassin(6), Paladin(7), Ninja(8, also base for humans), Ranger(10), Mystic(11) — are locked until you achieve Hero status on your first character life).*

### Step 6: Hometown Selection
```
Choose your hometown:
```
Choose your starting capital:
*   `K`: **Kir Drax'in**
*   `O`: **Kir-Oshi**
*   `A`: **Alaozar**

### Step 7: Stats Rolling & Confirmation
The server rolls your base attributes using the original AD&D formula:
1.  **Attribute Roll:** Rolls **4d6** (four six-sided dice), drops the lowest die, and sums the remaining three. This is repeated six times.
2.  **Attribute Sorting:** The six rolls are sorted in descending order.
3.  **Class Priority Assignment:** Base rolls are assigned to attributes based on your chosen class's stat priority:
    *   *Warrior:* STR, CON, DEX, CHA, WIS, INT
    *   *Mage:* INT, WIS, DEX, CON, STR, CHA
    *   *Cleric:* WIS, INT, CON, STR, DEX, CHA
    *   *Thief:* DEX, STR, CON, CHA, INT, WIS
    *   *Ninja:* DEX, STR, CON, INT, WIS, CHA
    *   *Psionic:* INT, WIS, DEX, CON, STR, CHA
4.  **Racial Bonuses:** Racial modifiers are applied to the finalized rolls.
5.  **Strength Check:** Warriors who roll a natural 18 Strength are additionally granted an exceptional Strength rating (`18/xx` where `xx` is a roll from 1 to 100).

```
Str: 18  Int: 12  Wis: 10  Dex: 14  Con: 15  Cha: 11
Keep this character? (Y/N):
```
Enter `Y` to confirm and enter the game, or `N` to re-roll your attributes. 

Once confirmed, the server issues a standard `"state"` message welcoming you into the starting room (such as Room 3001, the Temple). You are now ready to play!
