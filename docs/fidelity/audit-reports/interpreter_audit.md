# Port Fidelity Audit: Module 30 (`interpreter.c`)

This audit examines the port fidelity between the legacy C source file `src/interpreter.c` and its Go counterparts in `pkg/session/` and `pkg/command/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source Files
- **File**: `src/interpreter.c` (2,365 lines)
- **Functions**: 
  - `command_interpreter`: Handles command lookup via prefix abbreviations, enforces level/position gates, invokes special procedures, and dispatches to ACMD functions.
  - `nanny`: State machine for connected descriptors (handles login, character creation menus, and main menu choices).
  - `find_alias` & `perform_complex_alias`: Manage alias lookup and multi-command semicolon parsing.
- **Header**: `src/interpreter.h` (Defines command structures and ACMD function macro templates).

### Go Port Files
- **Command Registry**:
  - [registry.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/command/registry.go): Thread-safe `Registry` implementing middleware execution, exact command lookups, and dispatch.
- **Session Commands**:
  - [commands.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/session/commands.go): Registers built-in commands and implements `ExecuteCommand` dispatcher (handles guest restrictions, mob command scripts, spec procedures, and position/cooldown gates).
  - [cast_cmds.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/session/cast_cmds.go): Implements magic cast command and registers the `"cast"` endpoint.
- **Login and Character Creation Connection States**:
  - [session_login.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/session/session_login.go): Implements JSON-based authentication and nanny state handlers.
  - [char_creation.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/session/char_creation.go): Modern JSON-centric character creation flow.
- **Alias Logic**:
  - [aliases.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/aliases.go): Manages alias file reading, writing, and simple expansion.

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Dropped Player Inputs during Cooldowns (Lack of Command Queueing)
- **Source Context**: `src/comm.c#L597` (C queue reader), `pkg/session/commands.go#L517` (`ExecuteCommand` wait-state check).
- **Fidelity Bug**: In legacy C, if a player is in a wait state (cooldown lag), the engine continues to read their network input and appends those commands to the descriptor's input queue (`d->input`). The main game loop decrements the wait ticks every pulse and executes the queued commands sequentially as lag expires.
- **Impact**: In the Go port, `ExecuteCommand` immediately intercepts wait states and **drops** the input, sending a `"You're too busy!"` message. This requires players to spam commands to execute them after cooldowns, resulting in a clunky combat experience. Furthermore, this blocks communication (`say`, `gossip`) and informative commands (`look`, `score`, `who`) completely during combat lag—actions that were either bypassed or queued safely in the C original.

### 2. Missing Command Prefix Abbreviation Support
- **Source Context**: `src/interpreter.c#L910` (Prefix matching), `pkg/command/registry.go#L91` (`Lookup`).
- **Fidelity Bug**: In Diku/CircleMUD, the command interpreter uses `strncmp` to match prefixes. A player typing `sa` will execute the `say` command, and `lo` will execute `look`. This enables arbitrary partial command abbreviations out-of-the-box.
- **Impact**: The Go command registry performs a strict hash-map look-up (`r.commands[strings.ToLower(cmd)]`). Consequently, unregistered command abbreviations (e.g. `pra` for `practice`, `sa` for `say`, `ex` for `examine`) fail, generating an `"Unknown command"` error. This severely restricts standard MUD keyboard fluidity.

### 3. Missing Non-Alphanumeric Shortcut Symbols (`'` and `.`)
- **Source Context**: `src/interpreter.c#L902` (Symbol parser), `pkg/session/commands.go#L329` (Go splitter).
- **Fidelity Bug**: Legacy C checks `if (!isalpha(*argument))` and splits single-character symbol commands (like `'` for `say` and `.` for `reply`) from their arguments, even without a space boundary. For example, `'hello` is parsed as command `'` with argument `hello`.
- **Impact**: In Go, `'hello` is treated as a single command string, which fails exact lookups. Symbol shortcut dispatches are completely unimplemented, meaning players cannot use the standard `'` and `.` communication shortcuts.

### 4. Semicolon Command Stacking Bypass in Aliases
- **Source Context**: `src/interpreter.c#L1068` (C semicolon parser), `pkg/game/aliases.go#L190` (`PerformAlias`).
- **Fidelity Bug**: Legacy C supports complex aliases containing semicolons (e.g. `alias flee sit;stand;west`). Semicolons are parsed as `ALIAS_SEP_CHAR` (`;`) and split into multiple separate commands pushed consecutively onto the network input queue.
- **Impact**: Go's `PerformAlias` explicitly stubs out semicolon splitting. Because Go lacks both a descriptor command queue and multi-command parsing, stacked complex aliases are completely broken and execute as a single malformed string.

### 5. Position Gate Flavor Text Omissions
- **Source Context**: `src/interpreter.c#L924` (C positions), `pkg/session/commands.go#L18` (`positionFailMessage`).
- **Fidelity Bug**: Go's `positionFailMessage` misses multiple position cases and returns modernized, generic messages.
- **Impact**:
  - `POS_SITTING` is completely missing in Go's switch, returning the generic `"You are in no position to do that!"` instead of C's `"Maybe you should get on your feet first?"`.
  - `POS_FIGHTING` is missing, returning the generic message instead of C's `"No way!  You're fighting for your life!"`.
  - Missing flavor like `"Lie still; you are DEAD!!! :-("` (Go returns `"You are dead! You can't do that."`).

---

## 3. Go Improvements Over C

### 1. Middleware Architecture
- **Go Enhancement**: Go's registry uses a clean, functional middleware stack (`Registry.Use(mw)`). Cross-cutting concerns such as security checks, audit logging, and rate-limiting are elegantly wrapped around command handlers rather than being manually hardcoded into the parser body.

### 2. Thread-Safe Lookup
- **Go Enhancement**: Command registries in Go are equipped with read-write mutex locks (`sync.RWMutex`), safeguarding concurrent command maps from data races during lookup operations.

### 3. Modern Decoupled Connection State Machine
- **Go Enhancement**: Decoupling the character creation flow into clean JSON-based prompt stages (`char_creation.go`) represents a major architectural upgrade over C's massive, nested, 600-line `nanny()` switch block. This makes character validation and persistence straightforward.

---

## 4. Concurrency & Thread Safety

- **Package Initialization Registrations**:
  - Commands register themselves concurrently at boot time using `init()` functions across multiple packages (`pkg/session`, `pkg/command`). While map writes are not thread-safe, Go's single-threaded package initialization model ensures this is safe at startup. However, post-boot registrations are guarded by `Registry.mu.Lock()`.
- **Command Session Safety**:
  - `commandSession` adapts session details thread-safely via safe adapters.

---

## 5. Summary of Recommended Fixes

1. **Implement Command Prefix/Abbreviation Matching**:
   Update `Registry.Lookup()` in `pkg/command/registry.go` to fall back to a prefix search if an exact match fails. It should scan all registered commands and aliases for matching prefixes that satisfy the player's level restrictions.
2. **Implement Non-Alphanumeric Symbol Splitter**:
   Update `ExecuteCommand` in `pkg/session/commands.go` to parse single-character non-alphanumeric shortcuts (`'`, `.`, `;`) and map them directly to `say`, `reply`, and `wiznet` shortcuts.
3. **Queue Stacked Commands on Semicolon Split**:
   Introduce a simple input buffer/queue in `pkg/session/Session` to allow semicolon-separated commands and combat-wait states to be processed sequentially instead of instantly discarded.
4. **Complete Position Flavor Switch**:
   Refine `positionFailMessage` in `pkg/session/commands.go` to re-introduce the full classic Diku position flavor text for sitting and fighting states.
