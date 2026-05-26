# Audit Report: act.display.c vs Go Display Engines

**C file:** `src/act.display.c` (718 lines)
**Go file(s):** `pkg/game/act_display.go` (73 lines), `pkg/session/display_cmds.go` (525 lines)
**Mapping type:** 1:N
**Functions audited:** 28 C functions / ~5 Go command entrypoints & helpers

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Completely Un-registered / Dead Screen and Infobar Commands
- **Location:** `pkg/session/commands.go` (command registry) and `pkg/session/display_cmds.go` (entire file).
- **C behavior:** In `act.display.c:80` and `112`, the commands `do_lines` and `do_infobar` are fully registered MUD commands that let players customize their screen sizes (getting/setting terminal row count between 7 and 50) and toggle the immersive, dynamic VT100-based bottom stat infobar.
- **Go behavior:** The Go codebase has fully ported the logic into `cmdLines()` and `cmdInfoBar()` inside `pkg/session/display_cmds.go`. However, they are **completely dead and un-wired** (marked with `//nolint:unused` and `//lint:file-ignore U1000 ... not yet wired to command registry`). Neither command is registered in `pkg/session/commands.go`.
- **Discrepancy:** The MUD has lost its classic VT100 terminal customization layer. Players are completely unable to toggle the bottom infobar or modify their screen line counts.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-002]: Math Inconsistency: Experience Needed Update Bug
- **Location:** `pkg/session/display_cmds.go:476` inside `cmdInfoBarUpdate()`.
- **C behavior:** In C, `InfoBarUpdate()` relies on the character's real, authoritative experience structure to print the required remaining experience.
- **Go behavior:** Go's display package implements a highly detailed class-specific experience formula table via `findExp()` matching the legacy progression. This is correctly called when the infobar is initially rendered:
  ```go
  // In newInfobarState:
  expNeeded := findExp(p.Class, p.Level+1)
  ```
  However, in `cmdInfoBarUpdate` (which handles the live dynamic updates of the infobar), the calculation is brutally oversimplified:
  ```go
  // In cmdInfoBarUpdate:
  is.expNeededForLevel = 1000 * is.level
  ```
- **Discrepancy:** Severe mathematical progression divergence. If dynamic updates were wired, as soon as a player gained any experience points, their "Needed for Level" display would instantly jump from their real class progression threshold (e.g., 650,000 for level 11 mage) down to a flat linear scale (e.g., 10,000). Because their current exp is much higher than 10,000, the infobar would begin displaying massive, garbage negative numbers for the remaining experience.
- **Severity:** HIGH
- **Type:** DRIFT / BUG

---

## Type & Boundary Vulnerabilities

### [FINDING-003]: VT100 Scroll Region Boundary Mismatch
- **Location:** `pkg/session/display_cmds.go:405` inside `cmdInfoBarOn()`.
- **C behavior:** C configures the scroll margins to lock the bottom 5 lines for stats:
  `sprintf(buf, VT_MARGSET, 0, size - 5)` (top margin is 0, bottom is row - 5).
- **Go behavior:** Go matches this by writing:
  `output += fmt.Sprintf(vtMarSet, 0, is.screenSize-5)`
- **Risk:** Potential boundary incompatibility. Standard VT100 scrolling region escape codes expect 1-based indexing for both top and bottom rows: `\033[top;bottomr`. While many terminals treat `0` as `1` for the top row, a strictly standard implementation would use `1` explicitly. Additionally, if a player's screensize is smaller than 7, the scrolling boundary math `size - 5` collapses. Although `cmdLines` restricts size to >=7, the default screensize falls back to 0 (`s.screenSize == 0`), which will trigger a crash or term lockup if `cmdInfoBarOn` is called before any manual lines setting.
- **Severity:** MEDIUM
- **Type:** DRIFT

---

## Control Flow & Mathematical Fidelity

### [FINDING-004]: Isolated Dynamic Info Updates (No Game Loop Triggers)
- **Location:** `pkg/game/act_display.go` and `pkg/session/display_cmds.go`.
- **C behavior:** In C, dynamic updates `InfoBarUpdate(ch, INFO_HIT | INFO_MANA)` are raised in character ticks, spells, and damage loops whenever attributes change, forcing the terminal socket to flush individual refreshed cells.
- **Go behavior:** Go has no active routines in `pkg/game/` that invoke `cmdInfoBarUpdate` or track changed states. The entire updating loop is functionally isolated from character status updates and combat ticks.
- **Severity:** MEDIUM
- **Type:** STUB

---

## Concurrency & Mutex Safety

### [FINDING-005]: Concurrency Data Race on Player Live Stats
- **Location:** `pkg/session/display_cmds.go:453` in `cmdInfoBarUpdate()`.
- **C behavior:** Strictly single-threaded synchronous loop; no concurrency safety required.
- **Go behavior:** `cmdInfoBarUpdate` is triggered on individual player session goroutines. It reads active player stats directly to build VT100 templates:
  ```go
  is := &infobarState{
      screenSize:  s.screenSize,
      lastHit:     p.Health,
      lastMaxHit:  p.MaxHealth,
      lastMana:    p.Mana,
      ...
  }
  ```
  It accesses these stats concurrently while they are written to by the combat ticker (`processCombatPair`) or regen loops on separate goroutines without acquiring the player's lock (`p.mu`).
- **Impact:** Classic concurrent read/write data race. Can result in reading torn words, inconsistent stat values, or thread corruption on the `Player` state.
- **Severity:** HIGH
- **Type:** CONCURRENCY

---

## Unported Functions

All behavior inside `act.display.c` was structurally ported into Go files, although completely un-wired from active game loops. There are no completely unported C display functions.

---

## Summary

- **Total findings:** 5
- **Critical:** 0
- **High:** 3
- **Medium:** 2
- **Low:** 0
- **Unported functions:** 0

---

## Verification Plan

### Automated Verification
Verify the compilation safety of the display commands:
```bash
go build ./pkg/session/...
go build ./pkg/game/...
```
