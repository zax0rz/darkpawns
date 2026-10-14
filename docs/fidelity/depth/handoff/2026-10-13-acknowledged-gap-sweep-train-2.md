# Acknowledged Gap Sweep — train 2 — 2026-10-13

## Inventory continuity

The complete pre-fix acknowledged-comment inventory, silent-pulse review,
assigned-special crosswalk, triage, and copy-ready Claude issue text are in
`2026-10-13-acknowledged-gap-sweep.md`, `2026-10-13-acknowledged-gap-sweep-comments.tsv`,
and `2026-10-13-acknowledged-gap-sweep-specials.tsv` on train 1 (PR #1943).
This train starts from fresh `origin/main` at
`9b3ecaae3d02cb1f2b066e9dd27c81bb5ec350fc` and continues the Lua object-spell
family recorded there.

## C path and change

Verified in `src/` during this session:

- `src/scripts.c:1428-1489`: `lua_spell` reads both character and object
  arguments. The vocal arm calls `cast_spell(me, vict, obj, spell)`; the silent
  arm calls `call_magic(me, vict, obj, spell, GET_LEVEL(me), CAST_SPELL)`.
- `src/spell_parser.c:291-320`: the vocal object phrase is selected only when
  `tobj->in_room == ch->in_room` or `tobj->carried_by == ch`.
- `src/spell_parser.c:400-557,827-910`: direct `call_magic` position behavior
  and vocal cast gates.
- `src/magic.c:1900-1949`, `src/structs.h:472-476`: object spell behavior and
  the exact extra-flag indices, including remove-poison.
- `src/config.c:94`: `NOEFFECT` is `Nothing seems to happen.\r\n`.

The Go Lua bridge now resolves and forwards `ObjRef` through both cast arms.
`ObjectInstance.InRoom` and `CarriedBy` support the exact say-phrase gate. The
object alteration path now uses the C flag indices, C no-effect bytes, effect
messages to caster and room through `Act`, and the C remove-poison object arm.
No new random draws were added (R3).

## Proofs

- Real dispatch: `TestLuaSpellObjectTargetDispatchAndSpeechGate` runs the real
  Lua engine, bridge, world adapter, vocal `SPELL_INVISIBLE`, silent `SPELL_BLESS`,
  object flags, returned userdata, and room output.
- Speech location cases: `TestSameRoomWithObj` covers same-room and carried
  objects, remote and uncarried objects, and nil inputs.
- Object routine parity: `TestMagAlterObjsUsesCObjectFlagIndices` covers bless,
  curse, invisibility, remove curse, poison, and remove poison; the no-effect
  controls pin C's gates and bytes.
- R5h, dropped target control: restoring the old early return for non-nil
  `obj` made the real Lua test fail at `ITEM_INVISIBLE must be set` (exit 1).
  Restoring object forwarding made the same test pass (exit 0).
- R5h, wrong flag control: setting `ITEM_INVISIBLE` at bit 2 made the real Lua
  test fail at `ITEM_INVISIBLE must be set` (exit 1). Restoring C bit 5 made it
  pass (exit 0).
- R5h, speech gate control: restoring unconditional `sameRoomWithObj` made
  `TestSameRoomWithObj` fail for remote, uncarried, and nil objects (exit 1).
  Restoring the C predicate made it pass (exit 0).

Targeted Go tests and the real Lua integration test pass after restoring each
control.

## Gates

All commands were run separately on the final behavior code tip; all passed:

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS |
| `go test ./pkg/game/...` | PASS |
| `go test -race ./pkg/game/...` | PASS |
| `golangci-lint run ./...` | PASS: 0 issues |
| `make deadcode-ratchet` | PASS: 52 baselined, 0 new |
| `make fidelity-depth` | PASS: 5,514 cases; 5,424 proven/delegated; 10 blocked; 68 excluded; 12 divergent-approved |
| `make fidelity-units` | PASS: 1,420 rows, 1,306 symbols, 1,601 claims |
| `make string-census` | PASS: 98 known Go-only segments, 0 unreviewed; generated reports are already stale, and `make string-census-update` was not run |
| `gosec -severity=high -quiet ./...` | PASS |

## Oracle evidence

The stock-data scenario is `lua-spell-object-target`; it stands the healer with
the real Lua `action` binding, loads bread object 8010 into its inventory with
`oload`, then checks the vocal and silent object spell effects through the
player's transcript. The seed-1 `--show-oracle` run had no normalized
divergence. Its retained C block shows the carried bread, `object invisible=1`,
and `object blessed=1`, confirming both C object-alteration arms executed. A
full output and run manifest are retained under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-13/dp-1371-train2-show-oracle/`.
Targeted census: `dp-1371-train-2-lua-object-target` — CLEAN, 1 passed, 0
expected/failed/stale/infra/timeout/unstable; elapsed 11.866s. The retained
summary and manifest are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-13/dp-1371-train-2-lua-object-target/`.
The manifest row is now `oracle-green`. The train's final combined census on
code tip `7fe4c4c02` completed clean after the infrastructure recheck:
`oracle-combined: pairs=3396 deduplicated=1016 full=CLEAN claims=CLEAN_AFTER_RECHECK elapsed=1965.253s verdict=CLEAN_AFTER_RECHECK`.
The `results.tsv` rows for `lua-bind-actions`, `lua-bind-queries`, and
`lua-spell-object-target` are all `PASS`. The combined run directory is
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-13/dp-1371-train-2/`.
