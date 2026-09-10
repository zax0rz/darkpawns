# Port Fidelity Audit: Module 17 (`clan.c`)

This audit examines the port fidelity between the legacy C source file `src/clan.c` (along with `src/clan.h`) and its Go implementations in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **Files**: `src/clan.c` (1,574 lines), `src/clan.h`
- **Functions**: `do_clan_rename`, `do_clan_create`, `do_clan_destroy`, `do_clan_enroll`, `do_clan_expel`, `do_clan_promote`, `do_clan_demote`, `do_clan_who`, `do_clan_members`, `do_clan_quit`, `do_clan_status`, `do_clan_apply`, `do_clan_info`, `init_clans`, `save_clans`, `find_clan_by_id`, `find_clan`, `do_clan_bank`, `do_clan_money`, `do_clan_ranks`, `do_clan_titles`, `do_clan_private`, `do_clan_application`, `do_clan_sp`, `do_clan_plan`, `do_clan_privilege`, `do_clan_set`, `do_clan`.

### Go Port Files
The behavior of the massive C file is split into multiple single-responsibility Go files under the `pkg/game/` package:
- `pkg/game/clans.go` (Defines `Clan` and `ClanManager` structures, `InitClans`, `SaveClans`, `resolveClanContext`, `resolveClanForImmortal`, `sendClanFormat`)
- `pkg/game/clan_admin.go` (Implements `doClanRename`, `doClanCreate`, `doClanDestroy`)
- `pkg/game/clan_bank.go` (Implements `doClanBank`, `doClanPrivate` [redundant stub])
- `pkg/game/clan_command.go` (Implements command router `ExecClanCommand`)
- `pkg/game/clan_economy.go` (Implements `doClanMoney`, `doClanAppLevel`, `doClanSet`)
- `pkg/game/clan_info.go` (Implements `doClanStatus`, `doClanApply`, `doClanInfo`)
- `pkg/game/clan_membership.go` (Implements `doClanEnroll`, `doClanExpel`, `doClanPromote`, `doClanDemote`, `doClanWho`, `doClanMembers`, `doClanQuit`)
- `pkg/game/clan_settings.go` (Implements `doClanPrivate` [duplicate], `doClanPlan`, `doClanRanks`, `doClanTitles`, `doClanPrivilege`, `doClanSP`)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Immediate Nil Pointer Panic in `doClanBank`
- **Source Context**: `pkg/game/clan_bank.go#L8-L19` (`doClanBank`)
- **Fidelity Bug**: The Go function `doClanBank` declares a nil `c *Clan` pointer but never resolves it before dereferencing it:
  ```go
  func (w *World) doClanBank(ch *Player, arg string, action int) {
      var clanNum int
      var c *Clan // Nil pointer

      if arg == "" {
          w.sendClanFormat(ch)
          return
      }

      if action == CBWithdraw && !c.CanWithdraw(ch) { // CRASH: Nil Pointer Dereference
  ```
  Since `c` is never resolved using `resolveClanContext` or `resolveClanForImmortal`, any player attempting to withdraw money will immediately trigger a server panic, crashing the MUD runtime.

### 2. Clan Destruction Leaves Offline Players Corrupted (Orphaned Clan References)
- **Source Context**: `pkg/game/clan_admin.go#L132-L138` (`doClanDestroy`)
- **Fidelity Bug**: In legacy C, `do_clan_destroy` loops through the pfile database (`player_table`), loading offline characters via `load_char` to clear the deleted `clan` ID and rank, and saving via `save_char_file_u`.
  The Go port's `doClanDestroy` only loops over active, online players:
  ```go
  // Clear clan from all online members
  for _, p := range w.players {
      if p.ClanID == c.ID {
          p.ClanID = 0
          p.ClanRank = 0
      }
  }
  ```
- **Impact**: All offline clan members are left with invalid, corrupted `ClanID` references. If a new clan is subsequently created and assigned that recycled ID, these offline players will silently and automatically become members/leaders of the new clan upon login, posing severe gameplay and data integrity bugs.

### 3. Invisible Offline Applicants and Members
- **Source Context**: `pkg/game/clan_membership.go#L27-L32` (`doClanEnroll`), `pkg/game/clan_membership.go#L283-L295` (`doClanMembers`)
- **Fidelity Bug**:
  - `doClanEnroll`: When queried without arguments to list applicants, the Go implementation only loops through `w.players` (online active players). In legacy C, both online and offline applicants were listed by traversing the full player table.
  - `doClanMembers`: The Go port limits listings to active online players, noting: `// For now, only list online members (can't read all saved players without the pfile system)`.
- **Impact**: Clan leaders cannot see or enroll players who applied offline. Moreover, leaders have no way of monitoring or expelling/demoting members who are currently offline, since `w.GetPlayer` only checks online players.

### 4. Recalculation Drift on Boot (`InitClans` vs `init_clans`)
- **Source Context**: `pkg/game/clans.go#L162-L195` (`InitClans`)
- **Fidelity Bug**: Legacy C's `init_clans` loops through all player records `load_char` during startup, recalculating the total active `power` and `members` count for each clan to correct any offline changes or deletions.
  The Go port's `InitClans` merely restores cached JSON values from `clans.json` and performs no database/pfile queries.
- **Impact**: Active member counts and cumulative power levels will permanently drift over time if players are deleted, renamed, or have their levels modified while offline.

### 5. Destructive Plan Editing (`doClanPlan` Clears Descriptions)
- **Source Context**: `pkg/game/clan_settings.go#L64-L110` (`doClanPlan`)
- **Fidelity Bug**: Legacy C uses the line-descriptor editor `string_write` to allow players to multi-line edit their clan's plans.
  The Go port completely lacks line-editor support for this command. However, instead of stubbing the edit cleanly, `doClanPlan` executes:
  ```go
  c.Plan = ""
  w.SaveClans()
  ```
- **Impact**: Typing `clan set plan` will silently, immediately, and permanently wipe the existing clan plan, saving a blank string to `clans.json` with no way for players to write a new description.

### 6. Broken Immortal Parameter Parsing in `doClanRanks` and `doClanSP`
- **Source Context**: `pkg/game/clan_settings.go#L136-L138` (`doClanRanks`), `pkg/game/clan_settings.go#L342-L345` (`doClanSP`)
- **Fidelity Bug**: When an immortal issues a rank command (e.g. `clan set ranks 5 testclan`), the code attempts to parse the parameters:
  ```go
  a1, _ := halfChop(arg)
  arg = a1
  clanNum, c = w.Clans.FindClan(a1)
  ```
  `a1` holds the number `"5"` and the second part (the actual clan name `"testclan"`) is discarded in `_`. The system then looks up a clan named `"5"`, resulting in a perpetual `"Unknown clan"` error. The same bug occurs in `doClanSP`.

---

## 3. Go Improvements Over C

### 1. JSON Serialization
- **Fidelity Improvement**: Legacy C stored clan records in a raw binary file `CLAN_FILE` (`fread`/`fwrite`), which was extremely prone to platform alignment, packing, and endianness discrepancies. Go replaces this with structured, readable, and easily backupable JSON files under `./data/clans.json`.

### 2. Thread-Safe Mutexing
- **Fidelity Improvement**: Go encapsulates clan state within a thread-safe `ClanManager` backed by a `sync.RWMutex`, preventing race conditions during concurrent additions, deletions, or searches in the clan list.

### 3. Memory Safety
- **Fidelity Improvement**: Clan plans and titles are represented as safe native Go strings instead of raw C character pointers (`char *plan`), eliminating double-free and buffer overflow bugs (`CLAN_PLAN_LENGTH` overflows).

---

## 4. Concurrency & Thread Safety

- **Clan Field R/W Data Races**:
  - While the `ClanManager` secures the `Clans` slice with its read/write mutex, individual `Clan` fields (such as `c.Treasure`, `c.Members`, and `c.Plan`) are read and written directly during gameplay commands without acquiring a lock on the `Clan` struct or the global `World`.
  - If multiple players within the same clan perform concurrent transactions (e.g., depositing gold, withdrawing, or updating privileges simultaneously), these concurrent writes will cause data races and state inconsistencies, as `Clan` fields are not guarded by individual mutexes or atomic operations.

---

## 5. Summary of Recommended Fixes

1. **Fix immediate Nil Panic in `doClanBank`**:
   Before performing banking actions, resolve the clan context using `resolveClanContext` or `resolveClanForImmortal` to populate the `c *Clan` pointer correctly.
2. **Handle Offline Player Clan Updates**:
   Integrate database/pfile queries in `doClanDestroy`, `doClanEnroll`, and `doClanMembers`. Ensure that when a clan is destroyed, a database query is executed to clear `clan_id` and `clan_rank` for offline characters.
3. **Re-calculate Clan Members and Power on Boot**:
   Upon database startup, run a background query to aggregate player records and dynamically recalculate each clan's member counts and power values to prevent cached value drift.
4. **Fix Immortal Argument Parsing**:
   Refactor `doClanRanks` and `doClanSP` to extract parameters in the correct order:
   ```go
   arg1, arg2 := halfChop(arg) // arg1 = rank/value, arg2 = clan name
   arg = arg1
   clanNum, c = w.Clans.FindClan(arg2)
   ```
5. **Defer Plan Editing Until Editor is Wired**:
   Ensure `doClanPlan` does not wipe the description string `c.Plan = ""` unless the editor has successfully captured new input from the player.
