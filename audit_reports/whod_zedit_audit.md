# Port Fidelity Audit: Modules 59 & 60 (`whod.c` & `zedit.c`)

This audit examines the port fidelity between the legacy C sources `src/whod.c` and `src/zedit.c` and their Go counterparts inside `pkg/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source Files
- **`src/whod.c`** (532 lines):
  - Implements the MUD's original WHOD daemon.
  - Opens a secondary TCP socket (one port above the game's telnet port) to allow fast connections (e.g. `telnet mud.example.com 4001`) that display who is currently online without requiring a full game login.
  - Implements administrative toggles (`do_whod()`) to show or hide name, class, level, title, site, and wizinvis details from the WHO daemon output.
  - Tracks player and immortal counts.
- **`src/zedit.c`** (1,276 lines):
  - Implements the in-game OasisOLC zone editor `do_zedit`, which lets builders edit zone boundaries, life spans, room spawns, reset modes, door directions, and command lists directly inside the MUD console.

### Go Port Files
- [pkg/game/whod.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/whod.go): Full encapsulation of WHOD filter modes (`WhodShowClass`, `WhodShowSite`, etc.), bitwise command parser `DoWhod()`, and list builders (`BuildWhoList()`).

---

## 2. High-Fidelity Validation & Design Parity

Comparing the implementations highlights a **clean modern replacement** and **appropriate builder-tool deprecation**:

### 1. The WHO Daemon Filter System (`whod.c`)
- **Parity Status**: Flawless. Go's [whod.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/whod.go) maps every single bitmask option from C (`WhodShowClass`, `WhodShowLevel`, `WhodShowTitle`, `WhodShowSite`, `WhodShowWizLevel`) and toggle logic from `do_whod` perfectly.
- **Prefix Lookup Matches**: The C function `old_search_block()` is ported with high-fidelity as `WhodSearchBlock()`, supporting standard prefix-match commands (e.g. `whod site` or `whod site` toggling).
- **Socket listening Deprecation**: In Go, opening a separate low-level TCP raw socket is **intentionally deprecated**. Instead, modern MUD players and web portals connect to the unified WebSocket port, where the game server outputs the exact same who-daemon lists dynamically filtered by the active `Whod.Mode` settings. This avoids opening unmonitored ports on modern firewalls.

### 2. OLC Zone Editor Deprecation (`zedit.c`)
- **OLC Deprecation Status**: `src/zedit.c` is **intentionally unported**. Builders no longer edit zones using archaic in-game terminal CLI prompts. In Go, zone reset rules are written and managed as standardized JSON or text area `.zon` files directly under Git version control, or generated dynamically via modern map builders. This eliminates database corruption risks from runtime in-game OLC sessions.

---

## 3. Go's Architectural Improvements Over C

- **Safe String Builders**: The original C code used a fixed-length character buffer `MAX_STRING_LENGTH` and unsafe `strcat` calls which risked buffer overflows under high concurrent user loads. Go uses `strings.Builder` to scale buffer sizes dynamically and safely.
- **No Global Sockets**: The WHO daemon socket descriptor was a global static variable in C, risking socket leaks. Go encapsulates WHOD parameters in a clean, testable struct (`Whod`).
