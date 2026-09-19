# Depth-fidelity handoff — `oedit` — 2026-09-19

## Queue position

This session implements the `oedit` OLC command (CircleMUD/Dark Pawns
`src/oedit.c`, 1,564 lines) on branch `cline/phase7-olc-oedit` from
`origin/main` at `48040e534`. The C registration is
`{ "oedit", POS_DEAD, do_olc, LVL_BUILDER, SCMD_OLC_OEDIT }` at
`src/interpreter.c:592`.

This is the third and largest OLC editor family. It reuses the machinery the
merged `redit` and `medit` slices established: `pkg/session/olc_shared.go`
(`atoiC`, `olcZoneForVNum`, `olcAuthorized`), the atomic claim/release
admission pattern, and the descriptor-owned session state shape.

## Implementation summary

New files:

- `pkg/game/world_oedit.go`: object clone/snapshot/commit helpers,
  live-instance prototype refresh preserving runtime placement, and live
  object script mutation (C's `obj_index[rnum].script` indirection).
- `pkg/session/oedit.go`: descriptor-owned `oeditState`, the OEDIT_* mode
  table, entry gates, every C menu (main, type, extras, wear, container
  flags, apply prompt, apply, perm-spell, race, weapon, spells, liquid type,
  script menu/flags, extra-desc menu), the raw-line parser, save/discard/
  quit, the two improved-editor string fields, the `.obj` writer, and
  disconnect cleanup.
- `pkg/session/oedit_test.go`: 20 tests covering the entry gates, every
  verified C trap, the value cascade, the working copy, the disk format with
  a parse-back, live-instance refresh, script shallow authority, disconnect
  cleanup, color levels 0/2/3, and the concurrent-entry reservation.
- `cmd/dp-oracle-diff/scenarios/oedit-entry-depth.txt`
- `cmd/dp-oracle-diff/scenarios/oedit-boundaries-depth.txt`
- `cmd/dp-oracle-diff/scenarios/oedit-menu-color-on.txt` (keep-ansi)
- `cmd/dp-oracle-diff/scenarios/oedit-menu-color-off.txt` (keep-ansi)
- `docs/fidelity/depth/oedit.tsv`: 27 manifested cases.

Modified:

- `pkg/session/commands.go`: registered `oedit`.
- `pkg/session/manager.go`: `oedit *oeditState`, the `objEdits` reservation
  map, and disconnect cleanup.
- `pkg/session/session_login.go`: raw-line `oedit` routing, placed beside the
  CON_REDIT route because the editor owns its own improved-editor buffer and
  flushes one output turn at a time.
- `pkg/session/session_send.go`: prompt suppression for the object editor.
- `pkg/telnet/listener.go`: extended the existing
  `IsRoomEditing() || IsMeditEditing()` branch with `IsOeditEditing()`.

## C behavior implemented (and the traps verified)

- Entry gates: no-arg (`Specify a object VNUM to edit.` — the C literal, from
  `olc_scmd_info[SCMD_OLC_OEDIT].text`), nonnumeric, unknown zone, permission,
  duplicate editor.
- **Weapon val3 fallthrough** (`oedit.c:1333-1340`): `ITEM_WEAPON` sets
  `min_val = 1; max_val = 50;` and has no `break`, so it falls into
  `ITEM_WAND`/`ITEM_STAFF` which overwrite to 0..20. Weapons clamp 0-20.
  Reproduced deliberately, with a `//nolint:ineffassign` and a comment, and
  proven both by unit test and by the oracle vehicle's main-menu bytes.
- **The cost prompt lies** (`:1105` vs `:1243-1245`): the prompt claims a
  10000 maximum and `OEDIT_COST` is a bare `atoi`. Both kept.
- **Type 0 is unselectable** (`:1182`): the menu lists entries from 0, but
  `number < 1` is refused with `Invalid choice, try again : `.
- **Extras vs wear error asymmetry** (`:1191` vs `:1215`): an invalid
  extra-flag number redisplays the menu with no message; the wear path emits
  `That's not a valid choice!\r\n` first.
- **Percent load is a float** (`:1004-1010`, `:1248-1257`, `:967-976`):
  `atof`, `round_float(0.01)` in single precision (the `(int)` cast truncates
  before the multiply), clamp 0..100, and the menu's two renderings
  (`%.2f` and `1 in %d` from `(int)(100.0/load + 0.5)`, or `Never`).
- **Value cascade** (`:717-858`): LIGHT jumps to val3, WEAPON/MISSILE/
  FIREWEAPON skip val0, FOOD skips vals 2-3, CONTAINER's val2 is the
  container-flag menu with `1 << (n-1)` toggling in val 1.
- **Save-confirmation prompts** (`:1043-1045`): the first prompt ends with
  ` : `, the invalid-input retry does not.
- **Extra-desc menu is not redit's** (`:530-556`): its own header,
  `0) Quit` line, `Enter choice : ` prompt, and the `\r\n`-bearing
  `<Not set>` string; keywords go through `str_udup` (`olc.c:520`), so an
  empty keyword becomes the literal `undefined`.
- **Applies** (`:560-593`, `:1394-1426`): per-slot prompt with `%+d` plain
  modifiers and `None.` empty slots; `APPLY_RACE_HATE` and `APPLY_SPELL` route
  to their own menus and print `mob_races[modifier]` / `affected_bits[modifier]`;
  `Modifier : ` has no validation.
- **save_internally** (`:180-198`): C whole-struct-swaps every standing
  instance while preserving `in_room`, `carried_by`, `worn_on`, `contains`,
  `next`. In Go those placement fields live on `ObjectInstance`, so repointing
  `Prototype` preserves them; the instance-level override fields stand in for
  C struct fields a full assignment *would* overwrite, so they are cleared.
- **`.obj` format** (`:372-448`): four tilde strings (name, short, long,
  action; action `\r`-stripped; empty writes nothing between its tildes —
  `undefined` is only the missing name/short/long fallback), the 9-int
  type/extra/wear line, the 4-int values line, `weight cost %.2f load`,
  optional `S`/`E`/`A` blocks written in that order, and `$~`.

## Go-side field mapping (`pkg/parser/obj.go`)

| C field (`struct obj_data` / `obj_flag_data`) | Go field (`parser.Obj`) |
|---|---|
| `name` | `Keywords` |
| `description` | `LongDesc` |
| `short_description` | `ShortDesc` |
| `action_description` | `ActionDesc` |
| `obj_flags.type_flag` | `TypeFlag` |
| `obj_flags.extra_flags[4]` | `ExtraFlags [4]int` |
| `obj_flags.wear_flags[4]` | `WearFlags [4]int` |
| `obj_flags.value[4]` | `Values [4]int` |
| `obj_flags.weight` | `Weight` |
| `obj_flags.cost` | `Cost` |
| `obj_flags.load` (float) | `LoadPercent float64` |
| `affected[MAX_OBJ_AFFECT]` | `Affects []ObjAffect` |
| `ex_description` | `ExtraDescs []ExtraDesc` |
| `obj_index[rnum].script->name` | `ScriptName` |
| `obj_index[rnum].script->lua_functions` | `LuaFunctions` |
| `obj_flags.timer` | **no Go field** — see findings |
| placement fields (`in_room`, `carried_by`, `worn_by`, `worn_on`, `in_obj`, `contains`, `next`) | `ObjectInstance` (`Location`, `RoomVNum`, `Contains`, `Runtime`) |

## Findings (flagged, not silently accommodated)

1. **`obj_flags.timer` has no Go home.** The C `.obj` reader never loads a
   timer (`db.c:1431` reads `"%d %d %f"` only) and neither `oedit_save_to_disk`
   nor `db.c` ever writes one, so a freshly booted prototype is always 0. The
   port keeps the value session-local (`oeditState.timer`) so the in-editor
   bytes match C; a restart cannot resurrect a nonzero timer in C either, so
   nothing observable is lost. Recorded as `oedit.timer-session-local`.
2. **`medit`'s disk path is wrong; `oedit` does not copy it.**
   `saveMeditZone` writes `filepath.Join(parsed.SourceDir, "..", "mob")`,
   which resolves to `<lib>/mob` — but Dark Pawns keeps mob files at
   `<lib>/world/mob`, and C's `MOB_PREFIX` is `world/mob` relative to the lib
   root. `redit` uses `world.WorldPath + "/wld"` (correct) and `oedit` uses
   `parsed.SourceDir + "/obj"` (correct, since `SourceDir` is the `world`
   directory). The medit writer is out of this slice's scope and is **not**
   fixed here; it needs its own single-concern change.
3. **`OEDIT_ACTDESC` uses `MAX_MESSAGE_LENGTH`** (4096, `boards.h:34`), not a
   room/mob-desc bound; both oedit string fields use 4096.
4. **A never-saved object cannot be given a script**, because C resolves
   `GET_OBJ_SCRIPT` through `obj_index[rnum]` and a new object has
   `rnum == NOTHING`. The port keys on `isNew`.

## Verification

- `go build ./...`, `go vet ./...`: pass.
- `go test ./... -count=1`: pass (43 packages, exit 0).
- `go test -race ./pkg/session ./pkg/telnet`: pass.
- `golangci-lint run ./pkg/session/... ./pkg/game/... ./pkg/telnet/...`:
  `0 issues`.
- `gofumpt -l .`: clean.
- `make fidelity-depth`: green; 4,912 cases, 4,801 proven/delegated, 98.8%.
- Oracle vehicles, all at seeds 1, 2, 3, 5, 8 with
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`:
  - `oedit-entry-depth`: green at all five seeds.
  - `oedit-boundaries-depth`: green at all five seeds.
  - `oedit-menu-color-on`: green at all five seeds (keep-ansi).
  - `oedit-menu-color-off`: green at all five seeds (keep-ansi).
- `--show-oracle` inspection (R5e: verify the intended C block ran) confirms
  the C transcripts carry the trap bytes the port matches: `Values      : 0 3 20 0`
  after a weapon val3 of 50 (the fallthrough clamp), `Invalid choice, try
  again : ` twice, `Enter cost (10000 max): `, `That's not a valid choice!`,
  the `undefined` keyword, `12.57% (1 in 8)`, `Rakshasa to RACE_HATE`, and
  both save prompts; the keep-ansi transcript carries live `ESC[32m` bytes.

## Handoff frontier

The OLC editor family now has three merged members (`redit`, `medit`,
`oedit`). Remaining families: `sedit` (`SCMD_OLC_SEDIT`) and `zedit`
(`SCMD_OLC_ZEDIT`), plus the shared `olc_saveinfo`/`olc_saveall` surface in
`src/olc.c`. `oedit`'s zone disk save is unit-proven but not oracle-proven:
the C oracle harness runs with a CWD where `world/obj` does not resolve, so
the C file write fails while the player-facing message stays identical.