# Port Fidelity Audit: Module 42 (`poof.c`)

This audit examines the port fidelity between the legacy C source file `src/poof.c` and its Go counterparts in `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/poof.c` (102 lines)
- **Functions & Features**:
  - **Poof Persistence**: Implements `write_poofs` and `read_poofs` to read/write character poof messages directly to `<Player>.poof` files on disk, ensuring persistent teleport custom strings.
  - **Default Humorous Fallbacks**: Defines classic humorous default strings (`rides in on your mom.`, `rides out on your mom.`) if no custom messages are set.

### Go Port Files
- **Go Implementation**:
  - [pkg/session/wiz_info.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/session/wiz_info.go): Implements `cmdPoofset` command, which allows immortals to set their customized teleportation messages.

---

## 2. High-Fidelity Validation & Gaps

The Go implementation contains an architectural **volatility gap** that makes the custom poof set system non-persistent:

### 1. Poofset Session Memory Volatility Bug
- **The Gap**: In legacy C, setting a poof message invokes `write_poofs(ch)` which flushes the custom strings to `<Player>.poof` files. On character load, `read_poofs(ch)` restores them.
- In Go, `cmdPoofset` uses the session-level temp data map:
  ```go
  s.SetTempData("poofin", msg)
  s.SetTempData("poofout", msg)
  ```
  Since `Session` structs are ephemeral and constructed fresh on every socket connection (then destroyed/GCed on logout), **all custom teleport messages are instantly wiped out when a wizard logs out or loses connection**.
- **Impact**: Immortals must run `poofset` to configure their custom messages *every single time they connect* to the MUD.

### 2. Omission of Default Humorous Messages
- **The Gap**: Legacy `src/poof.c#L39-L40` defines humorous fallbacks:
  ```c
  strcpy(default_poofin, "rides in on your mom.");
  strcpy(default_poofout, "rides out on your mom.");
  ```
- In Go, `cmdPoofset` and the underlying teleport handlers do not define these default strings, leading to plain, dry, or absent teleport strings if a wizard uses default poofs.

---

## 3. Go's Architectural Improvements Over C

- **Memory Leak Protection**: In legacy C, `POOFIN(ch) = str_dup(buf)` dynamically allocates memory without any central garbage collection, making it easy to cause memory leaks if poofs are read/written frequently. Go's Garbage Collector handles session map dereferencing automatically.
- **Escape from File Handle Injections**: In C, files were parsed directly with `fscanf` and `fgets` without path validation, which is vulnerable to pathname manipulation if characters have custom characters in their names. Go avoids this by using session-safe keys, though it misses the persistence.

---

## 4. Concurrency & Thread Safety

- **Session Local Safety**: Since `SetTempData` is local to the individual connection session, it is isolated from other sessions and does not require global lock synchronization. However, when the player actually teleports, concurrent command processing on the player struct must ensure the session reads are synchronized if accessed across tickers.

---

## 5. Summary of Recommended Next Steps

1. **Implement Poof Persistence**:
   Save the immortal's custom poofs directly into their player save file (`<name>.json` or the SQLite `PlayerRecord` attributes block) rather than utilizing ephemeral session `TempData`.
2. **Add Default Poof Messages**:
   Restore the classic humorous defaults (`rides in on your mom.`, `rides out on your mom.`) as constant defaults inside the wizard command definitions.
