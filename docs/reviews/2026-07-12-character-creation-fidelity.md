# Character Creation Fidelity Review: C vs Go

**Date:** 2026-07-12
**Scope:** Full character creation flow — login through entering the game
**Source:** `src/interpreter.c` (nanny), `src/class.c` (roll_real_abils, do_start), `src/db.c` (init_char), `src/constants.c` (menus, abil_names)
**Target:** `pkg/session/char_creation.go`, `pkg/game/character.go`, `pkg/game/player.go`, `pkg/game/world_player.go`, `pkg/telnet/listener.go`
**Test output:** `~/.openclaw/workspace-daeron/memory/2026-07-12-character-creation.txt`

---

## Critical Issues

### 1. `[object Object]` — Structured options not rendered in WebSocket client
**Severity:** Critical (cosmetic, user-facing)
**Observed:** Lines 37-38, 43-44, 48-49, 58-65, 75-80, 88-90, 98-99 of test output

The Go server sends `CharCreateOption` objects (`{Key, Label}`) as JSON arrays in the `MsgCharCreate` message. The WebSocket client renders them as `[object Object]` instead of displaying the key/label pairs. This affects every prompt that has options (Y/N, race, class, hometown, stats).

The C version sends plain text menus inline — no structured data, just:
```
Do you want ANSI color (Y/N)?
```

**Fix needed:** Either the client needs to render `CharCreateOption` arrays as `[Y] Yes  [N] No`, or the server needs to send the prompt text with options already embedded (like C does).

### 2. Race/class/hometown menus display on single unwrapped lines
**Severity:** Critical (cosmetic, user-facing)
**Observed:** Lines 53-56, 68-73, 83-86 of test output

The C version sends menus with `\r\n` after each line:
```c
const char *race_menu =
  "\r\n"
  "Choose a race:\r\n"
  "  [H]uman        [E]lven       [D]warven      [K]enderkin\r\n"
  "  [M]inotaur     [R]akshasan   [S]sauran\r\n"
  "  [?]Help on races in general\r\n"
  "  [?<race abbreviation>] Help on a specific race (i.e ?D for help on dwarves)"
  "\r\n";
```

The Go version embeds the same text in a `Prompt` field of a JSON message, and the WebSocket client renders it without respecting the `\r\n` line breaks. The race menu, class menu, and hometown menu all appear as a single long wrapped line.

**Fix needed:** The client needs to render `\r\n` in prompts as actual line breaks, or the server should send menu lines as separate messages/array elements.

---

## Missing Messages

### 3. "Please remember to choose an appropriate fantasy-oriented name."
**Severity:** Medium (fidelity)
**C source:** `interpreter.c:1781,1812`
**Go:** Missing entirely

The C code displays this reminder after name confirmation. The Go `startNewCharFlow` jumps straight to "Did I get that right, X (Y/N)?" without it.

### 4. "New character." announcement
**Severity:** Low (fidelity)
**C source:** `interpreter.c:1842`
**Go:** Missing

Before the password prompt for a new character, C prints "New character.\r\n". The Go flow skips this.

### 5. Main menu (CON_MENU) between MOTD and game
**Severity:** Medium (behavioral)
**C source:** `interpreter.c:2160-2244`

The C flow after MOTD is:
```
*** PRESS RETURN: [enter]
[MAIN MENU]
  0) Quit
  1) Enter the game
  2) Enter description
  3) Read background story
  4) Change password
  5) Delete this character
```

The Go flow goes directly from MOTD → game entry. There is no main menu. This means Go players cannot change their password, edit their description, or read the background story at this stage.

### 6. Welcome message / MOTD text differences
**Severity:** Low (fidelity)

The C `CON_RMOTD` shows the MOTD file, then "PRESS RETURN", then the MENU. The Go version shows MOTD + "PRESS RETURN" but no MENU. The welcome text (`WELC_MESSG`) that C sends on game entry (line 2187) is also absent in Go.

---

## Numeric Defaults Mismatches

### 7. Starting Move points: C=82, Go=100
**Severity:** Medium (gameplay)
**C source:** `db.c:3053` — `ch->points.max_move = 82;`
**Go source:** `player.go:302` — `p.MaxMove = 100`

New characters in C start with 82 max move. Go gives 100.

### 8. Starting hunger/thirst: C=36, Go=24
**Severity:** Medium (gameplay)
**C source:** `class.c:578-579` — `GET_COND(ch, THIRST) = 36; GET_COND(ch, FULL) = 36;`
**Go source:** `player.go:306-307` — `p.Hunger = 24; p.Thirst = 24;`

C starts players at 36/48 (well-fed). Go starts at 24/48 (half-full). This means Go characters will get hungry/thirsty sooner.

### 9. Wimp level not set (C=5, Go=0)
**Severity:** Low (gameplay)
**C source:** `class.c:588` — `GET_WIMP_LEV(ch) = 5;`
**Go:** `WimpLevel` field exists (`player.go:180`) but is never set during character creation. Defaults to 0.

### 10. Practices not granted (C=2, Go=0)
**Severity:** Medium (gameplay)
**C source:** `class.c:590` — `GET_PRACTICES(ch) += 2;`
**Go:** `Practices` field exists (`player.go:27`) but is never set during character creation. Defaults to 0. New Go players cannot practice skills.

### 11. Display flags not set
**Severity:** Low (quality-of-life)
**C source:** `class.c:585-589` — Sets `PRF_DISPHP`, `PRF_DISPMANA`, `PRF_DISPMOVE`, `PRF_AUTOEXIT`
**Go:** Not set during character creation. Players start without HP/mana/move bars or autoexit.

---

## Behavioral Differences

### 12. Name rejection closes connection instead of re-prompting
**Severity:** Medium (usability)
**C source:** `interpreter.c:165-168` — "Okay, what IS it, then?" → back to `CON_GET_NAME`
**Go source:** `char_creation.go:166-167` — `s.Close()` — disconnects the player

When a player says "N" to "Did I get that right?", C lets them try again. Go disconnects them.

### 13. `init_char()` not called after stat acceptance
**Severity:** Medium (fidelity)
**C source:** `interpreter.c:2128` — `init_char(d->character)` called when stats accepted
**Go:** `completeCharCreation()` creates the player via `NewCharacter()` which calls `NewPlayer()`, but `init_char()` equivalent is not called separately.

C's `init_char()` does:
- Sets random height/weight by sex (male: 160-200cm, 120-180kg; female: 150-180cm, 100-160kg)
- Sets birth time, played time, logon time
- Initializes all skills to 0
- Clears affects, saves, conditions
- Sets loadroom to NOWHERE
- Sets armor to 100

Go's `NewPlayer()` does NOT set height/weight. Other fields are partially covered.

### 14. `do_start()` called at CON_MENU time, not CON_ROLLABL2
**Severity:** Low (timing)
**C source:** `interpreter.c:2214-2216` — `do_start()` called when player selects "1" from menu, AFTER stats are saved
**Go source:** `char_creation.go:471` — `NewCharacter()` (which includes do_start logic) called in `completeCharCreation()` after MOTD

The order differs but the end result is functionally equivalent since both happen before the player enters the game.

### 15. New character placed in room 8004, not 8099
**Severity:** Low (intentional change)
**C source:** `interpreter.c:2241` — `char_to_room(d->character, real_room(8099))` — "A Burning Hut"
**Go source:** `char_creation.go:538` — `s.player.RoomVNum = game.LoginStartRoom(s.player)` — returns `MortalStartRoom` (8004)

Comment in Go says 8099 "has no exits and no mob spawns in the current world data." This is an intentional change, not a bug, but it's a fidelity difference.

---

## Race Option Name Typo

### 16. "Minotauran" should be "Minotaur"
**Severity:** Low (cosmetic)
**Go source:** `char_creation.go:632` — `{"M", "Minotauran"}`
**C source:** `constants.c:200` — `[M]inotaur`

The C menu says "Minotaur" but Go says "Minotauran". Similarly, "Rakshasan" and "Ssauran" are used in Go but C uses "Rakshasan" and "Ssauran" in the menu (though the race help text says "Minotaurs", "Rakshasas", "Ssaurs").

---

## Duplicate/Conflicting Skill Assignment

### 17. Skills set in both `NewCharacter()` and `GiveStartingSkills()`
**Severity:** Medium (code quality, potential for drift)
**Go source:** `player.go:329-346` AND `character.go:296-316`

Both functions set thief/assassin starting skills. `NewCharacter()` is missing `peek` (which `GiveStartingSkills()` has). Since `completeCharCreation()` calls `NewCharacter()` first, then `GiveStartingSkills()`, the values from `GiveStartingSkills()` win — but the code is confusing and error-prone.

Additionally, `NewCharacter()` grants `kick` to all classes and `bash`/`rescue` to warriors — skills NOT granted by C's `do_start()`. These are either intentional additions or bugs.

---

## Stat Rolling Fidelity

### 18. `RollRealAbils` — Functionally correct
**Status:** PASS (with note)

The Go `RollRealAbils()` correctly implements:
- 4d6 drop lowest, sorted descending
- Class-based stat priority assignment (all 12 classes)
- Warrior 18/xx strength bonus
- All 7 racial modifiers including caps

The Ssaur wisdom cap (max 16) is implemented correctly via `if s.Wis > 16 { s.Wis = 16 }` after `min18()`.

**Note:** Stats are rolled during the `class` stage in Go (before hometown), vs during `CON_ROLLABL1` in C (after hometown). Functionally identical since the player hasn't seen the stats yet.

---

## C-side Code Not Ported (and not needed)

| C Feature | Status |
|-----------|--------|
| Ident lookup (`ident.c`) | Not ported — not needed for modern deployments |
| Ban checks (`isbanned()`) | Handled differently in Go auth layer |
| Dupe check (`perform_dupe_check()`) | Handled by Go session manager |
| `PLR_DELETED` re-creation flow | Not ported — Go uses DB soft delete |
| `PLR_INVSTART` invisibility | Not ported |
| `PLR_WRITING/PLR_MAILING/PLR_CRYO` flag clearing | Not ported |
| `Crash_load()` rent/crash recovery | Not ported — different persistence model |

---

## Summary: Priority Fix List

| # | Issue | Severity | Effort |
|---|-------|----------|--------|
| 1 | `[object Object]` — options not rendered | Critical | Medium (client or server) |
| 2 | Menu line breaks lost in WebSocket | Critical | Medium (client or server) |
| 12 | Name "N" disconnects instead of re-prompt | Medium | Small |
| 7 | Move 100 vs 82 | Medium | Small |
| 8 | Hunger/thirst 24 vs 36 | Medium | Small |
| 10 | No practices granted | Medium | Small |
| 13 | No height/weight randomization | Medium | Small |
| 17 | Duplicate skill assignment code | Medium | Small (refactor) |
| 3 | Missing "fantasy name" reminder | Medium | Trivial |
| 5 | No main menu | Medium | Medium |
| 9 | Wimp level not set | Low | Trivial |
| 11 | Display flags not set | Low | Trivial |
| 16 | "Minotauran" typo | Low | Trivial |
| 4 | Missing "New character." text | Low | Trivial |
| 6 | Missing welcome text | Low | Trivial |
| 15 | Room 8004 vs 8099 | Low | Intentional |

---

## Files Referenced

### C (source of truth)
- `src/interpreter.c:1693-2365` — `nanny()` state machine
- `src/class.c:86-112` — class/hometown menus
- `src/class.c:379-496` — `roll_real_abils()`
- `src/class.c:501-592` — `do_start()`
- `src/db.c:3006-3077` — `init_char()`
- `src/constants.c:61-90` — `abil_names[]`
- `src/constants.c:196-347` — race menus and help text
- `src/structs.h:350-384` — CON_ state definitions

### Go (port)
- `pkg/session/char_creation.go` — character creation state machine
- `pkg/game/character.go` — `RollRealAbils()`, `GiveStartingSkills()`
- `pkg/game/player.go` — `NewCharacter()`
- `pkg/game/world_player.go` — `GiveStartingItems()`
- `pkg/game/death.go:73,96-106` — `MortalStartRoom`, `LoginStartRoom()`
- `pkg/telnet/listener.go:300-426` — telnet pre-auth flow
- `pkg/session/protocol.go` — `MsgCharCreate`, `CharCreateData`, `CharCreateOption`
