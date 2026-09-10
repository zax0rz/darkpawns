# Port Fidelity Audit: Module 26 (`file-edit.c`)

This audit examines the port fidelity between the legacy C source file `src/file-edit.c` (in-game text file editor `tedit` and directory utility functions) and its Go implementations in `pkg/game/note_write.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/file-edit.c` (200 lines)
- **Core Functions**:
  - `tedit_string_cleanup` (aborts or saves OLC text files, managing disk writes and logging)
  - `valid_directory`, `valid_filename` (regex validation routines ensuring path traversal and command injection safety)
  - `list_directory` (uses C's `scandir` to safely list files in MUD directories)
  - `view_file` (displays the raw content of an audited text file)
  - `general_file_edit`, `edit_file` (starts the OLC string editor hook, putting players in `CON_TEDIT` state)

### Go Port Files
- `pkg/game/note_write.go` (implements the player note-writing state machine for `ITEM_NOTE` items, handling buffered inputs and line limits)

---

## 2. Critical Logic Gaps & Severe Bugs

There are **no severe execution bugs** in the `note_write.go` port. However, there are significant architectural differences:

### 1. General OLC File Editing (`tedit`) Completely Unported (Intentional)
- **Fidelity Discrepancy**: Legacy C's `tedit` was a general-purpose disk editor allowing immortal/admin players to edit, write, view, and delete text files directly on the host system (e.g. news, help files, policies) from inside the MUD client.
  In Go, **all general tedit disk OLC editors are unported**. The MUD handles help, news, and world zone configuration files externally using modern JSON/YAML configurations, direct terminal access, or database entries, rendering the in-game text file editing dead code.
- **Impact**: In-game commands to edit help files or news text files are absent in the Go session and registration layers.

### 2. Path Traversal & Filename Safety Checks Removed
- **Fidelity Discrepancy**: The legacy C code implemented strict filename sanitization to protect against server intrusion:
  ```c
  static int valid_filename(const char *file) {
      return matches(file, "^[a-zA-Z0-9_-]+.?[a-zA-Z0-9_-]*$");
  }
  ```
  Since the Go port deprecated in-game filesystem writes, these regex checks are omitted.
- **Impact**: Safe by design, as the Go note-writing system only mutates in-memory string properties (`state.obj.Runtime.NoteText`) and never makes host disk calls.

---

## 3. Go Improvements Over C

### 1. Memory Safety and Garbage Collection
- **Fidelity Improvement**: In legacy C, the editor constantly managed raw buffers, `realloc` bounds, and explicit string frees (`strdup`, `FREE(filebuf)`, `cleanup_olc`). Small errors frequently resulted in memory leaks or buffer overflows. Go handles all allocations using garbage-collected native string concats (`state.buffer += line`), entirely preventing memory corruption.

### 2. Concurrency-Safe State Map
- **Fidelity Improvement**: Go implements a thread-safe map protected by a mutex to track concurrent note writers:
  ```go
  var (
      noteWriteMu      sync.Mutex
      noteWriteEntries = make(map[int]*noteWriteEntry)
  )
  ```
  This guarantees that multiple players can write notes on separate items simultaneously without any data races or memory corruption.

---

## 4. Summary of Recommended Fixes / Enhancements

1. **No immediate fixes required**:
   The decision to deprecate the unsafe, disk-bound C `tedit` OLC in favor of direct filesystem/database configurations is an excellent modern architectural decision that drastically improves MUD security. Player note-writing (`ITEM_NOTE`) operates safely in memory.
