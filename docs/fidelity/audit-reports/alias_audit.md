# Port Fidelity Audit: Module 13 (`alias.c`)

This audit examines the port fidelity between the legacy C source file `src/alias.c` and its Go counterparts in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/alias.c` (110 lines)
- **Functions**: `write_aliases` (saves player aliases to flat files), `read_aliases` (restores player aliases from flat files at login).

### Go Port Files
- **Session Commands**:
  - `pkg/session/act_social.go` (Structurally misplaced; implements `cmdAlias` to add/delete/update aliases)
- **Game Logic**:
  - `pkg/game/aliases.go` (Defines `Alias`, `WriteAliases`, `ReadAliases`, `FindAlias`, `PerformAlias`)
  - `pkg/game/player.go` (Declares `Aliases []Alias` field in the player struct)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. `PerformAlias` is Never Called (Alias System is Completely Dead)
- **Source Context**: `pkg/session/commands.go#L325-L497` (`ExecuteCommand`), `pkg/game/aliases.go#L193` (`PerformAlias`)
- **Fidelity Bug**: The command processing loop in `ExecuteCommand` reads raw player input and dispatches it directly to the command registry or special procedures. It **never** calls `PerformAlias` to expand user-defined aliases. As a result, the alias system is entirely dead: players can define aliases, but typing them in-game is ignored and yields `"Unknown command"`.

### 2. `ReadAliases` is Never Called (Aliases are Never Loaded)
- **Source Context**: `pkg/session/session_login.go#L151-L295` (`handleLogin`)
- **Fidelity Bug**: Upon a returning player logging in, their character record is deserialized from the SQLite database via `db.RecordToPlayer`. However, the system never calls `game.ReadAliases(p.Name)` to populate the `player.Aliases` slice. As a result, all saved player aliases are lost from session memory upon logging out and logging back in.

### 3. Semicolon Multi-Command Complex Aliases Bypassed
- **Source Context**: `pkg/game/aliases.go#L190-L192` (`PerformAlias`)
- **Logic Gap**: In legacy Diku/CircleMUD, aliases can be "complex" (type `ALIAS_COMPLEX = 1`) which expand multi-command triggers separated by semicolons (e.g. `alias retreat sit;stand;north`). Semicolons are parsed and split into separate command executions.
- **Fidelity Bug**: Go's `PerformAlias` explicitly defers complex multi-command alias expansion because the command pipeline cannot parse or execute multiple command blocks sequentially yet. Complex aliases remain stubbed.

### 4. Structural Misplacement of `cmdAlias`
- **Source Context**: `pkg/session/act_social.go#L15-L80` (`cmdAlias`)
- **Fidelity Bug**: The active command handler `cmdAlias` (which handles player setting/deleting of aliases and calls `game.WriteAliases`) is implemented in `pkg/session/act_social.go` (mapped to Module 12 `act.social.c`). This is a highly misleading file structure that makes codebase navigation confusing.

---

## 3. Go Improvements Over C

### 1. Memory Safety
- **Fidelity Improvement**: Legacy C managed aliases as a singly-linked list (`struct alias *next`) dynamically allocated via `CREATE()`, which had high potential for memory leaks and memory fragmentation. Go uses a clean, safe slice `[]Alias` inside the `Player` struct, completely managed by Go's garbage collector.

### 2. Overflow/Truncation Immunity
- **Fidelity Improvement**: Legacy C opened and read files using fixed `127-byte` buffers (`fscanf` and `fgets`), creating potential buffer overrun or string truncation bugs. Go utilizes `bufio.Scanner` to safely parse lines of arbitrary length.

---

## 4. Concurrency & Thread Safety

- **Concurrent Slice Modifications**:
  - `cmdAlias` modifies a player's `player.Aliases` slice (deleting, adding, or replacing elements) while the player session is active.
  - Since this slice is accessed concurrently by the main session pump and potential logging cycles, lock protection must be ensured on `Player` struct modifications.

---

## 5. Summary of Recommended Fixes

1. **Wire Alias Expansion to the Interpreter**:
   Update `ExecuteCommand` in `pkg/session/commands.go` to call `game.PerformAlias(s.player.Aliases, cmdStr)` at the very beginning of the parsing routine (before matching command registry entries).
2. **Wire Alias Loading on Player Login**:
   Update the returning-player login sequence in `pkg/session/session_login.go#handleLogin` (after `RecordToPlayer` successfully loads the player struct) to call `game.ReadAliases(p.Name)` and assign the result to `p.Aliases`.
3. **Implement Complex Alias Semicolon Splitting**:
   Enhance `PerformAlias` to check `AliasComplex` triggers and split command strings by semicolons into multiple consecutive session command dispatches.
4. **Move `cmdAlias` to `pkg/session/cmd_alias.go`**:
   Relocate the misplaced `cmdAlias` function out of `pkg/session/act_social.go` into a new, logically-named command file `cmd_alias.go` to maintain codebase architectural integrity.
