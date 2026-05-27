# Port Fidelity Audit: Module 33 (`mail.c`)

This audit examines the port fidelity between the legacy C source file `src/mail.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source Files
- **File**: `src/mail.c` (597 lines)
- **Functions**: 
  - `push_free_list` & `pop_free_list`: Manage free 512-byte block structures in the binary mail file.
  - `find_char_in_index`: Search memory indices for first mail entry by player ID.
  - `write_to_file` & `read_from_file`: Seeks and performs raw block disk reads/writes.
  - `index_mail`: Registers and appends block positions to memory index.
  - `scan_file`: Performs MUD boot-up scanning and indexing of existing binary mail messages.
  - `store_mail` & `read_delete`: Core engines for packed block generation/chaining and extraction.
  - `postmaster` (SpecProc) & `postmaster_send_mail` / `postmaster_receive_mail` / `postmaster_check_mail`: Post office NPC actions.

### Go Port Files
- **Packed Binary System**:
  - [mail.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/mail.go): High-fidelity port of the 512-byte block format, `storeMail`, `readDelete`, index buffers, and serialization helper functions.
- **SpecProc Mapping**:
  - [postmaster.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/postmaster.go): Registers `"postmaster"` SpecProc and delegates to `mail.go` helper methods.

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Critical Concurrency Race Condition (Vulnerability & Severe Bug)
- **Source Context**: `pkg/game/mail.go#L263` (`storeMail`), `pkg/game/mail.go#L336` (`readDelete`), `pkg/game/mail.go#L73-L76` (Global state).
- **Fidelity Bug**: The legacy C MUD operated strictly on a single-threaded cooperative select loop, ensuring no concurrent access to the mail file or index structures. In Go, player session commands execute concurrently on separate goroutines.
  When Player A sends a mail, `storeMail` is called concurrently with Player B potentially reading/deleting mail via `readDelete`. Both methods read and write the global variables:
  - `mailIndex` (singly-linked memory list of indices)
  - `freeList` (singly-linked memory list of free blocks)
  - `fileEndPos` (file size tracker)
  - `data/mail` (shared disk file)
  No mutex lock protects the files or lists in `storeMail` or `readDelete` (unlike the session-buffered `mailWriteMu` which only protects draft compositions).
- **Impact**: Under concurrent traffic (two players sending mail or reading mail simultaneously), `freeList` and `mailIndex` pointers will race and corrupt. This leads to cyclic pointer reference hangs, lost blocks (memory/disk leaks), and severe on-disk corruption of the `data/mail` file.

### 2. Year 2038 Sign-Extension Timestamp Corruption (Severe Serialization Bug)
- **Source Context**: `pkg/game/mail.go#L628` (`readInt64`).
- **Fidelity Bug**: Go parses 64-bit int fields (specifically the Unix `MailTime` timestamp) from the binary stream using:
  ```go
  func readInt64(buf []byte, off int) int64 {
      return int64(readInt32(buf, off)) | int64(readInt32(buf, off+4))<<32
  }
  ```
  If `readInt32(buf, off)` returns a value whose most significant bit is `1` (which occurs when the 32-bit signed int represents a negative number), `int64(v)` performs a **sign-extension**, filling all upper 32 bits of the `int64` with `1`s.
  The subsequent bitwise OR `|` is corrupted because the upper 32 bits are already filled with `1`s, completely overriding and erasing the upper 32-bit integer offset.
- **Impact**: Currently, current timestamps do not set the most significant bit of the lower 32-bit int. However, in **Year 2038**, when Unix timestamps exceed `2,147,483,647` (`0x7FFFFFFF`), the signed lower 32-bit integer will overflow to a negative value. The sign-extension bug will trigger instantly, corrupting all read timestamps.
- **Fix**: Mask the lower 32 bits to prevent sign-extension before ORing:
  ```go
  return (int64(readInt32(buf, off)) & 0xFFFFFFFF) | (int64(readInt32(buf, off+4)) << 32)
  ```

---

## 3. Go Improvements Over C

### 1. Robust File System Creation
- **Go Enhancement**: C’s `scan_file` created a file using custom `touch(MAIL_FILE)`. Go uses modern robust `os.OpenFile` calls with clean permission modes (`0600`), avoiding potential multi-user security exposure.

### 2. Clear Session buffering
- **Go Enhancement**: decoulping the draft composition states (PLR_WRITING buffer accumulators) into memory-backed `sync.Mutex` maps (`mailWriteEntries`) is a major improvement over C's legacy static character buffer arrays, making memory bounds overflows impossible during composition.

---

## 4. Concurrency & Thread Safety

- **Unprotected Global Variables**:
  - The variables `mailIndex`, `freeList`, and `fileEndPos` are accessed concurrently in multiple goroutines, creating a severe data race.
  - Adding a global package-level `mailLock sync.Mutex` is mandatory to make the Go port's file and state operations thread-safe.

---

## 5. Summary of Recommended Fixes

1. **Introduce Global Mail Mutex**:
   Declare a global `mailSystemLock sync.Mutex` in `pkg/game/mail.go`. Protect all file access, free-list alterations, and index manipulations in `storeMail`, `readDelete`, and `scanFile` with `mailSystemLock.Lock()` and `mailSystemLock.Unlock()`.
2. **Fix Sign-Extension in `readInt64`**:
   Refine `readInt64` in `pkg/game/mail.go` to mask the lower 32-bit integer using `& 0xFFFFFFFF`, neutralizing the sign-extension corruption for Year 2038 compatibility.
