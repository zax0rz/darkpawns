# Audit Report: act.informative.c vs Go Informative Engines

**C file:** `src/act.informative.c` (2,804 lines)
**Go file(s):** `pkg/game/act_informative.go` (13,843 bytes), `pkg/game/look.go` (12,859 bytes), `pkg/game/help.go` (9,358 bytes), `pkg/session/act_informative.go` (4,252 bytes), `pkg/session/informative_cmds.go` (948 bytes), `pkg/session/cmd_look.go` (8,269 bytes), `pkg/session/examine.go` (9,095 bytes), `pkg/session/cmd_info.go` (12,113 bytes)
**Mapping type:** 1:N
**Functions audited:** 24 C commands and helpers / ~15 Go command entrypoints

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Bare Look Command Sends JSON Instead of Formatted MUD Text to Telnet
- **Location:** `pkg/session/cmd_look.go:13` (`cmdLook()`) and `pkg/game/look.go` (`doLook()`).
- **C behavior:** In `act.informative.c:725` (`look_at_room`), the bare `look` command renders gorgeous, fully formatted ANSI text describing the room name, description, obvious exits, dropped items, and present characters directly to the player's output buffer.
- **Go behavior:** Go's session-side `cmdLook` (which is wired to the active command parser) constructs a structured JSON `ServerMessage` of type `MsgState` containing raw room metadata, and pushes it directly into the session's socket send channel:
  ```go
  state := StateData{
      Player: PlayerState{ ... },
      Room: RoomState{ ... },
  }
  msg, _ := json.Marshal(ServerMessage{Type: MsgState, Data: state})
  s.send <- msg
  ```
  Although the Go game-side implementation `pkg/game/look.go:lookAtRoom` correctly implements traditional text rendering, it is **completely dead and un-wired** (marked with `//lint:file-ignore U1000 ... not yet wired to command registry`).
- **Discrepancy:** For standard Telnet/terminal MUD clients, typing `look` or moving between rooms flushes a raw, minified JSON block onto the player's screen rather than readable room descriptions, completely breaking standard terminal MUD playability.
- **Severity:** CRITICAL
- **Type:** BYPASS / PROTOCOL

### [FINDING-002]: Severe Calculation & Text Mismatch in Consider Command
- **Location:** `pkg/session/consider.go:14` (`cmdConsider()`) and `src/act.informative.c:2330` (`do_consider`).
- **C behavior:** In C, `consider` evaluates combat difficulty dynamically using precise formulas:
  - It loads the player's and target's actively wielded weapons, rolling their actual damage dice (e.g. `2d6`), or generates bare-handed random damage ranges `number(0, level/3)`.
  - It constructs a descriptive three-part sentence representing:
    - **Part 1:** Strength & weapon damage difference (`damdiff`).
    - **Part 2:** Hit point health difference relative to the *player's current HP* (`hitdiff` comparison with `GET_HIT(ch)`).
    - **Part 3:** Level and experience confidence difference relative to the *player's level* (`leveldiff`).
- **Go behavior:** Go's active `cmdConsider` uses simplified, simulated damage rolls (`Level * 2` + static Str modifiers) that **completely ignore wielded weapons** and bare-handed ranges. Furthermore, its sentence assembly is deeply buggy:
  - **Part 2** switches on static, hardcoded thresholds (`hitdiff > 30`, `hitdiff > -10`) instead of current HP ratios, producing completely different messages: `"and you would need a lot of help to beat $M."`
  - **Part 3** (the level-confidence check) is **completely omitted and dropped**.
- **Discrepancy:** The `consider` command generates inaccurate, fabricated combat difficulty assessments that differ structurally and textually from the original MUD, depriving players of reliable tactical feedback.
- **Severity:** HIGH
- **Type:** DRIFT / BUG

### [FINDING-003]: Character Sheet (`score`) is a Highly Abbreviated Debug Stub
- **Location:** `pkg/session/cmd_info.go:9` inside `cmdScore()`.
- **C behavior:** In C `act.informative.c:1168` (`do_score`), the score sheet prints a highly descriptive RPG layout containing Name, Age, birthdate indicators, textual alignment thresholds (e.g. "Epitome of Righteousness"), textual AC status (e.g. "armored like a wyvern"), Gold in Bank vs Gold in Hand, detailed Mob Kills, PKs, Deaths, citizenship, clan rank and affiliation, pack weight descriptions, position state, status conditions (intoxicated, hungry, thirsty), and active spells/equipment affects.
- **Go behavior:** Go's session-side `cmdScore` prints a tiny, barebones numeric debug layout. It reports only raw numeric fields: Name, Level, XP, HP, Mana, Move, Stats (Str/Int/Wis/Dex/Con/Cha), and numeric AC/Hitroll/Damroll/Align/Gold.
- **Discrepancy:** The MUD has lost its classic roleplay immersion panel. All textual status descriptors, bank savings, kills/deaths stats, active spell effects, and hunger/thirst warnings are completely hidden from the character sheet.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-004]: Complete Omission of Four Main Informative Commands
- **Location:** `pkg/session/commands.go` (command registry) and `src/act.informative.c`.
- **C behavior:** C registers and exposes:
  - `do_coins` (line 2743) — displays gold carried in a simple statement.
  - `do_abils` (line 1077) — prints ability scores as descriptive textual names (e.g. "superhuman").
  - `do_levels` (line 2311) — lists the experience progression table for the player's class.
  - `do_toggle` (line 2500) — displays and allows toggling up to 24 different preferences.
- **Go behavior:** The Go codebase has completely omitted the `coins`, `abils`, and `levels` commands from player sessions. Additionally, `cmdToggle` is a total stub that supports toggling ONLY `autoexit`, completely ignoring the other 23 legacy preferences.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-005]: Examine Command Bypasses progression by Revealing Identifications
- **Location:** `pkg/session/examine.go:86` inside `examineItem()`.
- **C behavior:** In C `act.informative.c:1137`, the `examine` command is simply a macro shortcut that runs `look <target>` and, if it is a container, automatically runs `look in <target>`.
- **Go behavior:** Go's `examineItem` prints highly detailed mechanical stats directly to the player: keywords, type, weight, valid wear slots, damage dice (for weapons), AC bonuses (for armor), and all enchanted affects.
- **Discrepancy:** Massive progression balance bypass. Players can identify the hidden magical attributes, weapon dice, and AC of any item in the game immediately for free, completely bypassing the necessity of casting the `identify` spell or paying oracle identify costs.
- **Severity:** HIGH
- **Type:** DRIFT

---

## Type & Boundary Vulnerabilities

### [FINDING-006]: Invisible and Sneaking Mob Visibility Checks Mismatched
- **Location:** `pkg/game/look.go:166` inside `listCharToChar()`.
- **C behavior:** In C, characters that are sneaking (`AFF_SNEAK`) or invisible are completely omitted from look results in dark or normal rooms unless the looker has the appropriate detection spells (`AFF_DETECT_INVIS`, etc.).
- **Go behavior:** Go's active `listCharToChar` in the game layer contains **no checks** for `affSneak` or `affInvisible` when printing the characters present in the room! It only skips players who are level >= 31 or players who are hidden (`affHide`).
- **Discrepancy:** Sneaking and invisible characters/mobs are fully visible in the room list to anyone who enters, rendering sneaking and invisibility spells useless for evasion or stealth.
- **Severity:** HIGH
- **Type:** DRIFT

---

## Concurrency & Mutex Safety

### [FINDING-007]: Concurrency Data Race on Look and Examine Targets
- **Location:** `pkg/session/cmd_look.go` and `pkg/session/examine.go`.
- **C behavior:** Synchronous single-threaded loop; thread-safe.
- **Go behavior:** Session-side look (`cmdLookAt`) and examine (`cmdExamine`) commands traverse lists of online players (`world.GetPlayersInRoom`) and read their attributes concurrently. They do so directly on separate player session goroutines without acquiring the player lock (`p.mu`).
- **Impact:** Classic read/write data race condition on player and mob instances.
- **Severity:** HIGH
- **Type:** CONCURRENCY

---

## Unported Functions

All core informative utilities in `act.informative.c` have been ported structurally, though with major logic drifts, stubs, and JSON serialization shifts.

---

## Summary

- **Total findings:** 7
- **Critical:** 1
- **High:** 5
- **Medium:** 1
- **Low:** 0
- **Unported functions:** 0

---

## Verification Plan

### Automated Verification
Verify structural safety and successful builds:
```bash
go build ./pkg/session/...
go build ./pkg/game/...
```
