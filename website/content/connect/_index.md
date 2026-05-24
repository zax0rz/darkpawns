---
title: "Connection Guide"
description: "Connect to the Dark Pawns persistent world via modern web browsers, raw Telnet, or dedicated professional desktop MUD clients."
date: 2026-04-28
draft: false
---

Connecting to Dark Pawns is straightforward, whether you prefer a zero-configuration web interface, a raw terminal command line, or a highly customized professional MUD client. The server runs continuously on port `4000` and supports concurrent Telnet and WebSocket connections.

---

## 1. Web Client (Zero Installation)

For casual players or immediate access, we provide an elegant, web-native console. It wraps our high-performance WebSocket connection in a responsive, customizable terminal shell.

- **URL:** [darkpawns.labz0rz.com/play](/play)
- **Requirements:** Any modern web browser (Chrome, Firefox, Safari, Edge).
- **Features:** Responsive font scaling, standard ANSI color rendering, local scrollback history, and instant hotkey integration.

---

## 2. Desktop MUD Clients (Recommended for Extended Play)

For dedicated players, we highly recommend utilizing a desktop client. These systems support local scripting, automated aliases, complex triggers, scrollback buffers, and graphic mapping capabilities that far exceed standard web browsers.

### Client Configuration Parameters
To configure your MUD client of choice, create a new profile with the following settings:

| Parameter | Value | Notes |
|:---|:---|:---|
| **Host / Address** | `darkpawns.labz0rz.com` | Authoritative server address |
| **Port** | `4000` | Standard MUD communication port |
| **Protocol** | Raw TCP / Telnet | Standard text-based telnet protocol |
| **Character Encoding** | `UTF-8` | Default encoding (falls back to `ISO-8859-1`) |
| **Terminal Type** | `xterm-256color` | Supports detailed MUD coloring schemas |
| **Auto-Wrap** | Disabled / 80 Columns | MUD handles formatting internally |

---

## 3. Recommended Desktop Software

### Mudlet (Cross-Platform / Graphical)
**Mudlet** is the industry standard for graphical MUD clients. It is free, open source, and packed with power user features.
- **Platform:** Windows, macOS, Linux
- **Pros:** Built-in Lua scripting engine, integrated auto-mapper, rich HTML/text rendering, and active community support.
- **Get Mudlet:** Download at [mudlet.org](https://www.mudlet.org/).

### TinTin++ (Console-Native / Highly Optimized)
**TinTin++** is a classic, terminal-based MUD client built for developers, system administrators, and players who value speed, keyboard shortcuts, and minimal resource footprints.
- **Platform:** macOS, Linux, Windows (via WSL/Cygwin)
- **Pros:** Extremely fast execution, advanced regular-expression based trigger matching, and robust macro files.
- **Get TinTin++:** Install via package manager or download at [tintin.mudhalla.net](https://tintin.mudhalla.net/).
  ```bash
  # Debian/Ubuntu
  sudo apt install tintin++
  # macOS
  brew install tintin++
  ```

### TinyFugue (UNIX Classic)
**TinyFugue** (tf) is a lightweight, line-oriented MUD client that has been popular among UNIX enthusiasts for decades.
- **Platform:** macOS, Linux
- **Get TinyFugue:** Install via standard package managers:
  ```bash
  # Debian/Ubuntu
  sudo apt install tf
  # macOS
  brew install tf
  ```

---

## 4. Command-Line Connection (Raw Telnet)

If you are using a raw terminal and have a telnet utility installed, you can connect immediately using a standard TCP shell command:

```bash
telnet darkpawns.labz0rz.com 4000
```
> [!NOTE]
> *Note for macOS users:* Modern macOS does not ship with the `telnet` utility by default. You can install it using Homebrew (`brew install telnet`) or connect via the in-browser Web Client.

---

## 5. First Steps: Character Creation & Survival

1. **Character Registration:** Choose a unique, lore-appropriate name. Abstract or modern usernames (e.g., `Player123`, `DarkSlayer`) will be rejected.
2. **Race & Class Selection:** Read the class details carefully. If you are a new player, starting as an Assassin or Cleric offers a gentle learning curve.
3. **Immersive Reading:** Read room descriptions! The text is loaded with active clues regarding exits, hidden locks, and hostile creatures.
4. **Command Syntax:** Once connected, type `help` to receive a categorized list of basic actions. Type `help <command>` for detailed syntax guides.
5. **Auto-Save:** Dark Pawns is entirely **rent-free**. Simply walk to the Temple in your starting city and type `quit` to safely save your character data and equipped items.

---

<div style="text-align:center; margin-top: var(--space-xl); margin-bottom: var(--space-md);">
  <a href="/play" class="btn btn-primary">Enter the World</a>
</div>
