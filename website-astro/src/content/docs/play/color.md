---
title: "Color"
description: "Set your ANSI color level in Dark Pawns with color off, sparse, normal, or complete, and see which screens light up at each threshold."
type: guide
audience: players
verified: v0.1.8
---

Dark Pawns uses four ANSI color levels: `off`, `sparse`, `normal`, and `complete`. Answering `Y` to `Do you want ANSI color (Y/N)?` when you roll a character sets you straight to `complete`, and you can dial it up or down at any time with the `color` command.

> If you have a color-capable terminal and wish to see useful color-coding of information, use the COLOR command to set the level of coloring you see. ... 'color' with no argument will display your current color level. 'color on' defaults to 'complete'.
>
> `help COLOR`

## Before you start

- A logged-in character in any terminal or client that understands standard ANSI escape codes (all five options in [Connect a client](/docs/play/connect-client/) do).

## Set your color level

1. Type `color` with no arguments to check your current setting.

   The server replies with `Your current color level is <level>.`

2. Type `color <level>` using `off`, `sparse`, `normal`, or `complete` (or `color on`, which is a shortcut for `complete`).

   The server saves your preference on your character and replies `Your color is now <level>.`

:::tip[Prefixes work too]
The command matches prefixes, so `color c` sets `complete` and `color s` sets `sparse`.
:::

## What changes at each level

Every colored message in the C source checks your character's level before emitting ANSI codes (`src/screen.h`). Higher levels include everything from the levels below them.

| Level | Command | What it colors |
| --- | --- | --- |
| **0: Off** | `color off` | Disables all ANSI color escapes. Everything prints as plain text. |
| **1: Sparse** | `color sparse` | Minimum emergency and status highlights: the `You wish that your wounds would stop BLEEDING so much!` warning in combat, the `who` and `users` lists, your own `gsay` echo, and the login rent/crash notice. Also expands inline `&` color codes (such as `&r` for red or `&Y` for bright yellow) embedded in world text, `map`, and `quest-say`. |
| **2: Normal** | `color normal` | Everything in `sparse`, plus room names, `[ Exits: ... ]` lines, room object and character lists, `report` output, incoming communication channels (`gossip`, `auction`, `shout`, `tell`, `newbie`, `ctell`), and the VT100 `infobar` (plus OLC building menus and `stat` screens for staff). |
| **3: Complete** | `color complete` | Everything in `normal`, plus combat hit and skill damage lines (yellow when your swing lands, red when you get hit), your prompt vitals (`H`, `M`, `V` colored green, yellow, or red by health percentage), enemy condition gauges in the prompt, `score` numbers, the `group` table, and your own outgoing channel echoes. |

:::note[Why `who` shows `&r` and `&y` when color is off]
If you run `color off` and type `who`, `[ Wizard ]` and `(Chosen)` lines show raw `&r`, `&y`, and `&n` codes. That is not a display bug: the original C server wrote those literal `&` tags into the `who` buffer and only ran its `&`-tag expander when at least one color bit was turned on (`src/comm.c`).
:::

## Did it work?

Type `color complete`, then type `score` and `look`. Room titles appear in cyan, items on the floor in green, mobs in yellow, and the word `color` in `Your color is now complete.` prints in red.

## If it didn't work

- **You see raw escape codes like `[36m` or `[0m`.** Your terminal or telnet client has ANSI color parsing turned off. Enable ANSI color in your client preferences, or type `color off` until you switch clients.
- **Your prompt or `group` list isn't colored even though rooms are.** You are at `color normal` (`level 2`). Prompt vitals, `score`, `group`, and melee damage lines require `color complete`.

## How it works

Under the hood, the server stores your color level in two preference flags on your character: `PRF_COLOR_1` (worth `1`) and `PRF_COLOR_2` (worth `2`). Adding them together produces a level from `0` (`C_OFF`) to `3` (`C_CMP`). Inline `&` tags (`&r`, `&Y`, `&c`, `&n`, and 13 others) expand in `vwrite_to_output` whenever either bit is set (`level >= 1`), while hardcoded game messages check `C_SPR` (`1`), `C_NRM` (`2`), or `C_CMP` (`3`) macro by macro.

## Related

- [Your first hour](/docs/play/first-hour/)
- [Connect a client](/docs/play/connect-client/)
- [Commands](/docs/reference/commands/)
