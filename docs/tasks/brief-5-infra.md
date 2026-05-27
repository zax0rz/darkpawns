# Brief 5: Infrastructure Fixes (Sonnet — 7 issues)

Various subsystem fixes. Less interdependent than the save/mob clusters.

## 1. DP-466 — House save mutates shared prototype weight (CRITICAL)

**File:** `pkg/game/` — find the house save function that sets item weights
**Bug:** `item.Weight = float64(...)` mutates the shared prototype, causing global weight corruption.

**Fix:** The weight-setting code in house save is unnecessary because JSON serialization doesn't use the in-memory Weight field for storage. Delete the ~30 lines that set/recalculate weight during save. The JSON save uses `GetSaveState()` which handles serialization correctly.

**Verify:** Find the exact function by searching for house save or `Weight =` in the house-related code.

---

## 2. DP-464 — Track skill cannot find mobs in other rooms

**File:** Find `getCharVis` or `GetCharVis` — the function used by the Track skill
**Bug:** `getCharVis` only searches the current room for NPCs. Track needs to find mobs across zones.

**Fix:** The Track skill's mob lookup needs a separate function that searches across rooms (like C's `get_char_vis` which checks `world_list`). Find where Track looks up its target and add cross-room NPC search.

---

## 3. DP-500 — Migrate math/rand to math/rand/v2

**Scope:** ~40 files use `"math/rand"`, only 3 use `"math/rand/v2"`
**Change:**
1. Create `pkg/common/random.go` with unified helper:
```go
package common

import "math/rand/v2"

func Number(min, max int) int {
    return rand.IntN(max-min+1) + min
}
```
2. Migrate each file's import from `"math/rand"` to `"math/rand/v2"`
3. Replace `rand.Intn(N)` with `rand.IntN(N)` (capital N in v2)
4. Replace ad-hoc `rand.Intn(max-min+1) + min` with `common.Number(min, max)`

**Note:** This is mechanical but touches many files. Do it carefully. `math/rand/v2` uses `IntN` (capital N) not `Intn`.

---

## 4. DP-460 — specMoonGate hardcodes MortalStartRoom

**File:** Find `specMoonGate` in spec_procs
**Change:** Replace hardcoded destination with gate_phases table lookup.
**Reference:** `src/spec_procs.c` for the gate_phases table structure.

---

## 5. DP-462 — Red portals never decay

**File:** Find portal object creation
**Change:** Set decay timer on red portals: `portal.SetTimer(ObjectTimer, 2)`

---

## 6. DP-473 — Commands dropped during cooldown

**File:** Find the command cooldown check
**Change:** Non-combat commands (look, inventory, say, etc.) should bypass the cooldown gate.

---

## 7. DP-476 — Mail system has no global mutex

**File:** `pkg/game/mail.go`
**Change:** Add `var mailGlobalMu sync.Mutex` and lock/unlock around `storeMail` and `readDelete`.

---

## Build verification
```bash
go build ./... && go vet ./... && go test ./...
```
