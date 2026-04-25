# Dark Pawns Research Log

[DESIGN] [2026-04-24] Initial thought on the game...

## 2026-04-24 — Wave 5 Complete: Character Management & Affect Lifecycle

[RESULT] [DESIGN]

**Status:** All 30 C functions from handler.c, limits.c, magic.c mapped to Go equivalents.

**What was written:**
- `pkg/game/char_mgmt.go` — `HasLight`, `UpdateCharObjects`, `UpdateCharObjectsAR`, `ExtractChar`, `ExtractPendingChars`, `UpdateObject`
- `pkg/game/affect_update.go` — `AffectUpdate`, `SpellWearOffMsg`

**Already existed:**
- All 11 C affect helpers (handler.c:136-499) → `pkg/engine/affect_helpers.go`
- Gain functions (limits.c:30-197) → `pkg/game/limits.go`
- `init_char` → `NewPlayer`/`NewCharacter` constructors in `player.go`

**Key design decisions (deviations from C):**
- `init_char` → Go constructors, not "lazy init + blank fields". `NewPlayer()` for loading from DB, `NewCharacter()` for new creation (with `do_start()` equivalent).
- `free_char`/`clear_char` → handled by Go GC and struct zero-value patterns. No explicit free.
- `stop_follower`/`add_follower`/`remove_follower`/`set_hunting` → C linked-list follower system replaced by `Player.Following` string + simplified design. WIP for 6+.
- `obj_from_obj`/`object_list_new_owner` → C linked-list pattern doesn't match Go slice pattern. Not ported.
- `PLR_EXTRACT` bit = 21, `NOWHERE` = -1, `ITEM_LIGHT` type = 1 — constants defined inline in char_mgmt.go.
- `AffectUpdate` uses `AffectFromChar` (from affect_helpers.go) instead of `affect_remove` — same semantics.

**Build:** clean, vet clean.
