# Port Fidelity Audit: Module 38 (`modify.c`)

This audit examines the port fidelity between the legacy C source file `src/modify.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/modify.c` (870 lines)
- **Functions & Features**:
  - **Interactive Multiline Editor**: Implements `string_write`, `string_add`, and `smash_tilde` which intercept connection inputs line-by-line until `@` is sent. It saves or aborts compositions (mail, boards, descriptions) depending on state.
  - **Paging / Pagination System**: Implements `page_string`, `paginate_string`, `next_page`, `count_pages`, and `show_string`. This system splits long text blocks (such as help files or news) into interactive pages, allowing scrolling (RETURN), quitting (Q), refreshing (R), backing up (B), or jumping to specific page numbers.
  - **Character / Object String Field Editing**: Command `do_string` allows live string editing of mob/player attributes and object keywords, short/long/extra descriptions.
  - **Word Wrapping**: `clean_up` wraps text at `width = 78` columns while maintaining custom return characters (`|`).
  - **Character Skill Modification**: Command `do_skillset` allows immortals to manually set skill levels.

### Go Port Files
- **Wizard & Editor Commands**:
  - [pkg/game/modify.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/modify.go): Ported implementation containing the immortal commands `doSet` (ported from `act.wizard.c`), `doStat` (from `act.wizard.c`), `doGecho` (from `act.wizard.c`), `doSocial` (from `act.social.c`), and `doSkillset` + `doString` (from `modify.c`).

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Complete Omission of Pagination/Pager System (Severe UX Gap)
- **Source Context**: `src/modify.c#L351-L528` (`show_string` and paging family), `pkg/session/session_manager.go#L47` (noting `showstr` is omitted).
- **Fidelity Gap**: In legacy C, any long message blocks (such as reading posts on boards, reading help files, or checking server rules) are buffered and fed to the client page-by-page. Players can browse them interactively using RETURN, Q, B, R, or numbers.
  In Go, **the entire paging system is omitted**. All outputs are written to player sockets in their entirety, potentially flooding the player's Telnet buffer and destroying readability for large help files.

### 2. Omission of Interactive Multiline Editor
- **Source Context**: `pkg/game/modify.go#L475` (`doString`).
- **Fidelity Gap**: In legacy C, omitting the inline string value in `do_string` places the descriptor into `string_add` composition mode, letting builders type a multiline block.
  The Go port has completely omitted the interactive multiline composition mode for string editing, forcing inline parameter edits and outputting a stub message when a value is omitted:
  ```go
  if valueArg == "" {
      ch.SendMessage("Enter string mode not yet supported — use 'string <obj> <field> <value>' directly.\r\n")
      return true
  }
  ```

### 3. Omission of 78-Column Word Wrapping (`clean_up`)
- **Source Context**: `src/modify.c#L775-L869` (`clean_up`).
- **Fidelity Gap**: In legacy C, the `clean_up` function processes inputted string blocks to automatically wrap words at a standard width of 78 characters. This functionality has been omitted in Go, meaning description changes lack auto-formatting.

### 4. Limited `doString` Scope (Objects Only)
- **Source Context**: `pkg/game/modify.go#L450` (`doString`).
- **Fidelity Gap**: In legacy C, `do_string` can edit both mobs (`TP_MOB`) and objects (`TP_OBJ`). In the Go port, `doString` only searches for objects (`w.findObjNear`) and completely omits character/mob field changes.

---

## 3. Go Improvements Over C

### 1. Clean Parameter Parsing
- **Go Enhancement**: Legacy C relied on a rigid argument splitting routine `quad_arg` which depended on index values of static array constants (`string_fields`). Go replaces this with clean, error-handled string splits (`splitArg`) and robust conditional blocks.

### 2. Safe Value Conversions
- **Go Enhancement**: Go utilizes type-safe conversions (`strconv.Atoi`) for numerical limits and clamps skill values cleanly to `0-100` bounds with descriptive player warnings, preventing silent failures or integer overflow bugs that C had.

---

## 4. Concurrency & Thread Safety

- **Read-Lock Player Iterations**:
  - The `doGecho` command acquires a read lock on the global player collection (`w.mu.RLock()`) prior to broadcasting messages, isolating iterations from concurrent logins or logouts.
- **RWMutex Isolation**:
  - Character field edits in `doSet` utilize thread-safe setters (`target.SetLevel`, `target.SetGold`, `target.SetHP`) to avoid race conditions with tick updates.

---

## 5. Summary of Recommended Next Steps

1. **Restore Pager Support**:
   Implement a lightweight pagination buffer in `pkg/session/` to paginate long output sequences (like help files) page-by-page.
2. **Expand `doString` to Mobs**:
   Add mob lookup and field mutations to `doString` to restore parity with C's mob-string edit capabilities.
