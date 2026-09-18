# Depth-fidelity handoff — `medit` — 2026-09-18

## Queue position

This session implements the `medit` OLC command (CircleMUD/Dark Pawns
`src/medit.c`) on branch `chungus/phase7-olc-medit` from `origin/main` at
`a2172482`. The C registration is `{ "medit", POS_DEAD, do_olc,
LVL_BUILDER, SCMD_OLC_MEDIT }` at `src/interpreter.c:553`.

The explicit September 18 assignment authorizes implementation and **one
local conventional commit only**. Do not push, merge, deploy, or open a
PR; the parent pushes through the GitHub API and creates the draft PR.
Do not alter or rebase `origin/glm/phase7-olc-redit`.

## Implementation summary

New files:
- `pkg/session/olc_shared.go`: C `atoi`, zone lookup, OLC permission gate,
  duplicate-editor scan.
- `pkg/game/world_medit.go`: mob clone/snapshot/commit helpers, live
  script mutation preserving C's shallow-script behavior.
- `pkg/session/medit.go`: descriptor-owned `meditState`, C mode constants,
  entry gates, menus, raw-line parser, save/discard/quit, detailed-desc
  editor, script menu, `.mob` writer, disconnect cleanup.
- `pkg/session/medit_test.go`: 12 tests covering gates, defaults, editing,
  save/discard, scripts, disconnect.

Modified:
- `pkg/session/tedit.go`: descriptor-local `onComplete` for OLC string editing.
- `pkg/session/manager.go`: `mobEdit *meditState` + disconnect cleanup.
- `pkg/session/commands.go`: registered `medit`.
- `pkg/session/session_login.go`: raw-line `medit` routing.
- `cmd/dp-oracle-diff/scenarios/medit-boundaries-depth.txt`: new boundary vehicle.

## C behavior implemented

- Entry gates: no-arg, nonnumeric (`Yikes!  Stop that...`), unknown zone,
  permission (`You do not have permission...`), duplicate editor.
- New mobile defaults: `mob unfinished`, RACE_OTHER (16), ISNPC, 1d1 HP/damage,
  AC 100, standing, weight 200, height 198.
- Field mappings: display hitroll `20-THAC0`, parser AC in tenths, damage
  `Damage.Num/Sides` + damroll `Damage.Plus`, attack `BareHandAttack`.
- Numerical gate: `Field must be numerical, try again : ` with C's empty-input
  and atoi quirks.
- Save confirmation, dirty-quit confirm, clean-quit silent exit.
- Detailed-desc via improved editor; script menu with live shallow-script
  mutation; `.mob` writer with 4 tilde fields and exact C layout.

## Verification

- `go build ./...`: pass.
- `go vet ./...`: pass.
- `go test ./pkg/session/ -run TestMedit`: 12 tests pass.
- `go test ./...`: pass except 2 pre-existing `profiling` failures (root can
  write to read-only dirs; fails on `main` too).
- `make fmt` (gofumpt): clean.

## Oracle status

**BLOCKED**: The C oracle binary cannot be built in this environment.
`src/Makefile` is broken (automake-generated, missing `Makefile.am`).
Manual gcc build fails on missing Lua headers (`scripts.c:27: lua.h` not
found); `src/lua/` has only `.o` files and prebuilt `.a` libs, no headers.
`apt-get install liblua5.4-dev` was attempted.

Without the oracle binary, `DP_ORACLE_BIN` scenarios cannot run. The
`medit-entry-depth`, `medit-session-depth`, and `medit-boundaries-depth`
scenarios are ready but unproven.

Seeds 1, 2, 3, 5, 8: **not run** (oracle binary unavailable).

## Remaining fidelity gaps

1. **Oracle proof**: All menu bytes, prompts, and edge cases are sourced from
   C code reading, not oracle-diff proof. Requires oracle binary.
2. **Level >40 EXP**: C reads beyond its 41-entry `EXP_LOOKUP` array
   (undefined). Go clamps to level 40. Unresolved; documented as gap.
3. **Position 9-14**: C's clamp accepts values beyond the 9-item display
   table. Go accepts but cannot render player-visible bytes without oracle.
   Unresolved; documented as gap.
4. **Save timing**: C queues `OLC_SAVE_MOB` until `medit save <zone>`; Go
   writes immediately on save confirm. Byte-equivalent output but different
   timing. Documented as gap.
5. **`medit.tsv`**: Still shows `blocked` status. Update to `proven` only
   after oracle-diff passes.

## Next steps for parent

1. Build oracle binary (resolve Lua headers) or run on machine with oracle.
2. Run: `DP_ORACLE_BIN=<bin> go run ./cmd/dp-oracle-diff --show-oracle`
   then seeds 1,2,3,5,8 for `medit-entry-depth`, `medit-session-depth`,
   `medit-boundaries-depth`.
3. Update `docs/fidelity/depth/medit.tsv` statuses from `blocked` to `proven`.
4. Run `make fidelity-depth` and `make expected-divergences-check`.
5. Push via GitHub API; create draft PR.
