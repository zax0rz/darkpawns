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
| `make fidelity-depth` | PASS |
| `make fidelity-units` | PASS: 1,420 rows, 1,306 symbols, 1,601 claims |
| `make string-census` | PASS: 98 known Go-only segments, 0 unreviewed; generated reports are already stale, and `make string-census-update` was not run |
| `gosec -severity=high -quiet ./...` | PASS |

## Oracle evidence

The stock-data scenario is `lua-spell-object-target`; it gives the mobile
healer bread object 8010 (unblessed and nonmagical), then checks the vocal and
silent object spell effects through the player's transcript. A targeted census
and the train's final combined census are pending because the shared census slot
is occupied by `dp-1419-lethal`. Fill in both census result lines here and mark
`lua.bind-spell-object-target` oracle-green only after the targeted case is clean.
