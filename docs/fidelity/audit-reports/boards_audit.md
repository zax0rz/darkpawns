# Port Fidelity Audit: Module 15 (`boards.c`)

This audit examines the port fidelity between the legacy C source file `src/boards.c` and its Go counterparts in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/boards.c` (552 lines)
- **Functions**: `find_slot`, `find_board` (looks up board in the room), `init_boards` (resolves board VNUMs and loads binary files), `gen_board` (board object special procedure), `Board_write_message`, `Board_show_board`, `Board_display_msg`, `Board_remove_msg`, `Board_save_board`, `Board_load_board`, `Board_reset_board`.

### Go Port Files
- **Game Logic**:
  - `pkg/game/boards.go` (Defines `BoardMsgInfo`, `BoardInfo`, `BoardSystem`, `InitBoards`, `findSlot`, `WriteMessage`, `ShowBoard`, `DisplayMsg`, `RemoveMsg`, `FindBoard`, `genBoard`, and `GetOrInitBoards`)
  - `pkg/game/player.go` (Declares `WriteMagic` field inside the `Player` struct)
  - `pkg/game/spec_assign.go` (Maps board VNUMs to the `"gen_board"` spec proc string)
- **Session Layer**:
  - `pkg/session/commands.go` (Directly routes look/examine/read/write/remove commands to room item spec procedures, but completely lacks board editor integration)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Board System is Never Initialized (Boards are Completely Dead)
- **Source Context**: `pkg/game/world.go#L96-L97` (`Boards`), `pkg/game/boards.go#L549-L581` (`genBoard`)
- **Fidelity Bug**: The MUD's world boot sequence **never** calls `GetOrInitBoards()` or `InitBoards()` to bootstrap the board system. As a result, the `world.Boards` pointer remains `nil` forever in production. 
  When players interact with boards, the `genBoard` spec procedure immediately aborts because the database is uninitialized:
  ```go
  if w.Boards == nil {
      return false
  }
  ```
  This causes all board interactions (`read`, `write`, `remove`, `look board`) to fall back to generic item descriptions or fail entirely with `"Unknown command"`, rendering the entire bulletin board system completely inactive.

### 2. Dead Editor Hook: Session Layer Bypasses Board Composition (`WriteMagic`)
- **Source Context**: `pkg/game/boards.go#L565-L571` (`genBoard`), `pkg/session/`
- **Fidelity Bug**: 
  - In `genBoard`'s `write` case, when a player attempts to write a post on a board, the system allocates a slot and assigns `ch.WriteMagic = magic`.
  - However, the session layer (`pkg/session/`) completely ignores the `WriteMagic` field. It never intercepts `WriteMagic`, never flags `PlrWriting`, and never activates the text editor for the player.
  - Even if the board system were initialized, a player typing `write hello` would create a blank post but would be unable to write any message body, as the composition screen fails to trigger.

### 3. Binary File Layout Alignment Danger
- **Source Context**: `pkg/game/boards.go#L138-L145` (`loadBoard`)
- **Logic Gap**: Legacy C saved boards as raw binary structures directly to disk using `fwrite`. The Go port replicates this binary structure to maintain backward file compatibility:
  ```go
  var info struct {
      SlotNum    int32
      _          [4]byte // padding for pointer (heading) — not serialized
      Level      int32
      HeadingLen int32
      MessageLen int32
  }
  ```
- **Fidelity Bug**: Relying on raw binary structs with hardcoded padding `_ [4]byte` is extremely fragile. Variations in structure alignment, integer sizing, or machine architecture (e.g. 32-bit vs 64-bit builds, compiler optimizations) between the original C compiles and the new Go binary will corrupt offsets, trigger invalid slice bounds on `make([]byte, info.HeadingLen)`, and forcibly reset all boards with `Board file corrupt. Resetting.`

---

## 3. Go Improvements Over C

### 1. Memory Safety
- **Fidelity Improvement**: Legacy C utilized raw `malloc`/`FREE` dynamically allocated character buffers for `msg_storage` and headings, presenting high risks for double-free or buffer leakage vulnerabilities. Go replaces this with standard GC-managed string pointers and boolean flags in the `BoardSystem` storage buffers.

### 2. Thread-Safe Mutexing
- **Fidelity Improvement**: Legacy C had no thread protection for board index changes, vulnerable to race conditions if multiple threads updated the index. Go encapsulates board state inside the `BoardSystem` and implements a read-write lock (`sync.RWMutex`) to synchronize concurrent board access safely.

---

## 4. Concurrency & Thread Safety

- **Mutex Lock-Reacquisition Bug**:
  - In `RemoveMsg` inside `boards.go`, the code attempts to unlock and re-lock the RWMutex during file saves to avoid holding the lock during slow Disk I/O:
    ```go
    // Save after removal (release lock first)
    bs.mu.Unlock()
    bs.saveBoard(boardType)
    bs.mu.Lock()
    ```
  - However, because the RWMutex was initially locked using a write lock `bs.mu.Lock()` at the start of `RemoveMsg`, unlocking via `bs.mu.Unlock()` is fine, but wait!
  - `saveBoard()` itself acquires a read lock `bs.mu.RLock()`:
    ```go
    bs.mu.RLock()
    num := bs.numOfMsgs[boardType]
    ...
    bs.mu.RUnlock()
    ```
    This nested locking works cleanly *only* because the lock is temporarily released beforehand. However, if the lock was *not* released, it would deadlock. Temporary lock release pattern is prone to race conditions (another thread can modify `bs.numOfMsgs` between `Unlock` and `Lock`), suggesting `saveBoard` should not acquire its own internal locks when called internally.

---

## 5. Summary of Recommended Fixes

1. **Bootstrap the Board System at World Boot**:
   Update `pkg/game/world.go` during world initialization (e.g. inside `world.go#NewWorld` or a boot sequence) to call `w.GetOrInitBoards("lib")` so that `w.Boards` is properly instantiated and loaded.
2. **Wire `WriteMagic` to the Session Editor**:
   Add support in `pkg/session/session_login.go#handleCommand` or `pkg/session/commands.go` to check `s.player.WriteMagic` after executing room spec procedures. If a magic board token is set, activate the line-editor and route player inputs to the board's storage buffer (similar to how `PlrWriting` works for mail and notes).
3. **Migrate to JSON Serialization**:
   Retire the unsafe binary disk loader (`binary.Read`/`binary.Write`) and transition the board persistence layer to standard JSON or YAML. This guarantees cross-architecture compatibility, eliminates struct padding corruption, and makes board files easily editable by administrators.
4. **Refactor Concurrency Locks**:
   Refactor `saveBoard` to be a lock-free internal helper `saveBoardUnlocked`, and have public methods handle synchronization cleanly to eliminate lock-reacquisition race conditions.
