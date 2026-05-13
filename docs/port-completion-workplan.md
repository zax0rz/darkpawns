# Port Completion Work Breakdown
# Generated: 2026-05-13
# Status: Ready for agent dispatch

## Context
The Dark Pawns Go port is ~85% complete. Core systems work (combat, movement, spells, socials, clans, houses). The remaining gaps are interface mismatches, missing wiring, and incomplete commands. OLC/editors deliberately skipped (admin panel planned).

---

## Work Package 1: Interface Mismatch Fixes (MiMo v2.5 Pro)
**Priority:** CRITICAL — spells silently fail without these
**Effort:** ~2 hours
**Files:** pkg/game/player_affects.go, pkg/game/player_stats.go, pkg/game/mob.go, pkg/game/object.go, pkg/spells/affect_spells.go

### Task: Add missing methods to match spell system interface assertions

**Player (pkg/game/player*.go) — add 3 methods:**
- `func (p *Player) AddAffect(aff *engine.Affect)` — delegates to affect manager
- `func (p *Player) GetExtraFlags() int` — returns player extra flags as int (check what field exists)
- `func (p *Player) SetExtraFlags(flags int)` — sets player extra flags

**MobInstance (pkg/game/mob*.go) — add 3 methods:**
- `func (m *MobInstance) GetAlignment() int` — returns mob alignment
- `func (m *MobInstance) SetName(name string)` — sets mob name
- `func (m *MobInstance) AddAffect(aff *engine.Affect)` — delegates to affect manager

**ObjectInstance (pkg/game/object.go) — add 5 methods:**
- `func (o *ObjectInstance) GetObjType() int` — alias for GetTypeFlag() (check C ITEM_ types match)
- `func (o *ObjectInstance) GetObjVal(idx int) int` — returns object value by index (vals array)
- `func (o *ObjectInstance) SetObjVal(idx, val int)` — sets object value by index
- `func (o *ObjectInstance) SetExtraFlags(flags int)` — sets extra flags (note: existing GetExtraFlags returns [4]int, may need adapter)
- `func (o *ObjectInstance) GetName() string` — alias for GetKeywords() or GetShortDesc()

**World (pkg/game/world*.go) — add 1 method:**
- `func (w *World) FindObjectByName(name string) []interface{}` — searches all objects in world by name

### Verification
- `go build ./...` passes
- `go vet ./...` passes
- Spell system interface assertions resolve at compile time

---

## Work Package 2: Help File Loading (DeepSeek V4 Pro)
**Priority:** HIGH — `help` command returns nothing without this
**Effort:** ~1 hour
**Files:** pkg/game/world.go, pkg/game/info_commands.go, lib/text/help/ (data)

### Task: Wire help file loading at boot

**Context from C source (db.c):**
- C loads `lib/text/help/*.txt` into `help_table` (binary search array)
- Each help entry has: keyword, entry text, min level
- `do_help` in act.informative.c does binary search on help_table
- `HELP_PAGE_FILE` is the motd/welcome screen

**Go implementation needed:**
1. Create `pkg/game/help.go` with:
   - `HelpEntry` struct (keyword, entry, minLevel)
   - `LoadHelpFiles(dir string) ([]HelpEntry, error)` — reads all .txt files from directory
   - `BinarySearchHelp(table []HelpEntry, keyword string) *HelpEntry`
2. Add `HelpTable []HelpEntry` field to World struct
3. Call `LoadHelpFiles` during world boot (in `LoadWorld` or `BootWorld`)
4. Wire `doHelp` in `pkg/game/info_commands.go` to search HelpTable
5. Also load MOTD from `HELP_PAGE_FILE`

### Data location
- Help files should be at `lib/text/help/` (relative to server binary)
- Check if files exist in repo; if not, document where to get them

### Verification
- `go build ./...` passes
- Help table loads non-empty on boot
- `help newbie` returns content

---

## Work Package 3: House Object Persistence (K2.6)
**Priority:** HIGH — player items in houses lost on reboot
**Effort:** ~3 hours
**Files:** pkg/game/house_save.go, pkg/game/houses.go

### Task: Complete house object save/load

**Context from C source (objsave.c, house.c):**
- C saves object data to rent files (one per player per house)
- Object data includes: vnum, value[4], extra_flags, wear_pos, timer, cost, material, name, description
- `Obj_to_store` serializes object to file
- `Obj_from_store` deserializes object from file
- Auto-equip on load

**Go implementation needed:**
1. Define `SavedObject` struct matching C obj_file_elem
2. `func SaveHouseObjects(playerName string, objs []*ObjectInstance) error` — serializes to JSON/file
3. `func LoadHouseObjects(playerName string) ([]*ObjectInstance, error)` — deserializes from file
4. Wire into house rent system (when player rents a room, save their objects)
5. Wire into house boot (when player enters rented room, load their objects)

### Verification
- `go build ./...` passes
- Save/load round-trip test: save objects, load them back, verify fields match

---

## Work Package 4: Missing Command Implementations (MiMo v2.5 Pro)
**Priority:** MEDIUM — player-facing commands
**Effort:** ~2 hours
**Files:** pkg/game/info_commands.go, pkg/game/other_economy.go, pkg/session/informative_cmds.go

### Task: Complete stubbed commands

**Commands to fix:**
1. `do_diagnose` (pkg/game/info_commands.go:296) — currently prints "not yet fully implemented"
   - C source: shows target's position, HP status, affects
   - Implement: check target's HP, position, list active affects

2. `do_tattoo` (pkg/game/other_economy.go:115) — currently prints "not yet implemented"
   - C source (tattoo.c): tattoo system for immortal markings
   - Implement basic version or document as immortal-only

3. `do_gen_write` (pkg/game/other_settings.go) — bug/typo/idea/TODO submission
   - C source: writes to appropriate file (bugs/typo/ideas)
   - Implement: write to files in lib/ directory

### Verification
- `go build ./...` passes
- Each command produces reasonable output

---

## Work Package 5: Gameplay Wiring (DeepSeek V4 Pro)
**Priority:** MEDIUM — gameplay completeness
**Effort:** ~3 hours
**Files:** pkg/game/graph.go, pkg/scripting/engine.go, pkg/game/comm_channel.go

### Task: Wire remaining gameplay systems

**5a. Weather movement penalty (graph.go:193)**
- C source: movement penalty in bad weather (rain, snow, sandstorm)
- Check weather.go for weather state, apply penalty to move cost

**5b. Lua follow/mount (scripting/engine.go:2137,2148)**
- C source: mobs can follow players and be mounted
- Engine needs World access to call add_follower / mount functions

**5c. Lua shop production (scripting/engine.go:2332)**
- C source: shops produce items on timer
- Wire production tables to Lua callbacks

**5d. Lua carry-weight check (scripting/engine.go:2313)**
- C source: scripts can check player carry weight
- Expose GetCarryWeight to Lua

**5e. Clan channel fix (comm_channel.go:296)**
- Currently broadcasts to all players as fallback
- Wire to actual clan member list

### Verification
- `go build ./...` passes
- Each system responds to game events

---

## Work Package 6: Race Help Text + Data Integrity (MiMo v2.5 Pro)
**Priority:** LOW — polish
**Effort:** ~1 hour
**Files:** pkg/game/constants.go, pkg/game/act_informative.go

### Task: Port remaining data from C constants.c

**Race help strings (constants.c:205-350):**
- help_human, help_dwarf, help_elf, help_kender, help_minotaur, help_rakshasa, help_ssaur
- These are displayed when players type `help <race>`
- Port as Go string constants, wire to help system

**Additional data checks:**
- Verify all 95 zone definitions loaded correctly
- Verify 187 socials registered
- Verify liquid table complete
- Verify class tables complete

### Verification
- `go build ./...` passes
- `help elf` returns racial description

---

## Agent Assignment Recommendation

| WP | Description | Recommended Agent | Why |
|----|-------------|-------------------|-----|
| 1 | Interface fixes | MiMo v2.5 Pro | Mechanical, needs Go precision |
| 2 | Help file loading | DeepSeek V4 Pro | File I/O + data structures |
| 3 | House persistence | K2.6 | Serialization + persistence |
| 4 | Missing commands | MiMo v2.5 Pro | Game logic, needs C knowledge |
| 5 | Gameplay wiring | DeepSeek V4 Pro | Cross-system integration |
| 6 | Race help text | MiMo v2.5 Pro | Simple data porting |

### Execution Order
1. WP1 (interface fixes) — unblocks everything else
2. WP2 (help files) — quick win, high impact
3. WP4 (commands) — quick win
4. WP3 (house persistence) — needs more work
5. WP5 (gameplay wiring) — cross-cutting
6. WP6 (data integrity) — polish

### Total Estimated Effort
~12 hours across 3 agents
