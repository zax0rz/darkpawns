# Acknowledged Gap Sweep — Train 4 (2026-10-13)

## Scope and source audit

This train starts from freshly fetched `origin/main` at
`9b3ecaae3d02cb1f2b066e9dd27c81bb5ec350fc`, on
`codex/luna-acknowledged-gap-sweep-4`. The pre-fix inventory is the committed
`2026-10-13-acknowledged-gap-sweep.md` plus its comments and assigned-special
crosswalk in Train 1.

The C call paths were checked directly in `src/` for this train:

| Gap | C evidence and behavior |
|---|---|
| Wounded charmed mobile activity | `src/mobact.c:68-117` performs both hunter calls, then special dispatch, waking, scavenging, and movement. It has no wounded/charmed return between those stages. |
| Scavenger and `CAN_GET_OBJ` | `src/mobact.c:102-117` scans room contents in list order and requires `CAN_GET_OBJ(ch, obj)` before comparing cost. `src/utils.h:515-549` defines blindness, room light/infravision, invisible/detect-invisible, HOLYLIGHT, take flag, carry-weight, and carry-count gates. `rg -n 'CAN_GET_OBJ\s*\(' src` found three production call sites: `src/mobact.c:108`, `src/spec_procs3.c:163`, and `src/scripts.c:210`. The butler also resolves its named room fixtures through `get_obj_in_list_vis` at `src/handler.c:1328-1347`. |
| Lua `canget` actor and object | `src/scripts.c:193-219` obtains the actor from global `me`, the exact object from argument 1's `struct`, and returns the result of `CAN_GET_OBJ(ch, obj)`. The stock healer has detect-invisible, so the oracle probe removes affect bit 3 with `aff_flags` (`src/scripts.c:165-191`) before setting the object's invisible bit 5. The player trigger keeps HOLYLIGHT, making actor selection observable. |
| Lua `set_hunt` with player hunter | `src/scripts.c:1341-1363` passes table arguments to `set_hunting`. `src/utils.c:708-728` logs a positive-ID player prey for any hunter; `src/utils.c:730-738` makes `HUNTING(ch)` return `NULL` for players. `rg -n hunting_id src` found the field declaration and writes in `set_hunting`; the read is only in `HUNTING`'s NPC branch, so the player-hunter bridge must emit C's mudlog without inventing a Go-side player target. |

The `CAN_GET_OBJ` class audit covered all three macro call sites above. The
butler's `get_obj_in_list_vis` preselection and `CAN_GET_OBJ` carry gate are
both world-aware after this change. No other caller was found by that source
search.

## Fixes and proofs

- Removed Go's extra wounded/charmed early return from mobile activity. The
  focused real-pulse proof expects the ordinary C door and sound draws.
- Mobile scavenging, the butler, and live Lua `canget` now use one World-aware
  object predicate matching `CAN_GET_OBJ`, including the actual actor's room,
  visibility, take flag, weight, and count. The Lua bridge passes global `me`
  and the exact `ObjRef`; legacy vnum-only clients retain their old API.
- Lua `set_hunt` resolves player hunters and emits C's positive-ID player-prey
  mudlog. Only mobile hunters store a Go-side target, matching the fact that
  C's `HUNTING(player)` is always `NULL`.
- These changes add no RNG draws. The scavenger retains its one pre-scan draw;
  visibility and capacity predicates are deterministic; `set_hunting` only
  logs/updates state; removing the invented early return restores the existing
  C draw sequence (R3).

## R5h controls

Each mutation was applied temporarily, the focused proof was run against the
regressed code, and the mutation was restored before the passing verification.

| Temporarily restored/removed behavior | Reverted-code result |
|---|---|
| Reinsert wounded/charmed early return | `TestMobileActivityWoundedCharmedMobKeepsCActivity` failed: got no draws, wanted `[{from:0 to:18} {from:0 to:15}]`. |
| Remove the scavenger `CAN_GET_OBJ` gate | `TestMobileActivityScavengerUsesCanGetVisibility` failed for both `invisible` and `dark`: each picked up an object that C rejects. |
| Force Lua `canget` back to its legacy vnum-only path | `TestLuaCanGetUsesMeAndLiveObjectVisibility` failed in all five actor/object cases, including visible controls; the old bridge could not resolve the live `me` and object table used by C. |
| Restore `SetHunt`'s mobile-only guard | `TestLuaSetHuntPlayerHunterUsesCPlayerPreyLog` failed: got empty output, wanted `[ Playerhunter started hunting Playerhunter ]\r\n`. |

After restoring all fixes, these focused proofs passed:

```text
go test ./pkg/game -run '^(TestMobileActivityWoundedCharmedMobKeepsCActivity|TestMobileActivityScavengerUsesCanGetVisibility|TestLuaCanGetUsesMeAndLiveObjectVisibility|TestLuaSetHuntPlayerHunterUsesCPlayerPreyLog|TestButlerVisibleRoomObjectUsesWorldLight)$' -count=1
ok github.com/zax0rz/darkpawns/pkg/game 0.008s
go test ./pkg/scripting -run '^TestLuaCanGetValidPathPushesNil$' -count=1
ok github.com/zax0rz/darkpawns/pkg/scripting 0.004s
go test ./internal/oraclediff -run '^TestParseScenarioPeersAndFixtures$' -count=1
ok github.com/zax0rz/darkpawns/internal/oraclediff 0.003s
```

## Oracle and gates

The target oracle cases are `lua-bind-actions` and `lua-bind-queries`; both
contain the new depth probes. Both were run with `--show-oracle` at seed 1 and
reported no normalized divergence. The retained normalized outputs are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-13/dp-1371-train4-show-oracle/`;
the `lua.bind-canget-world-actor` probe reports `canget invisible=nil` after
the live mobile's detect-invisible effect is removed. Their targeted census
and the train's final combined census remain pending. The preceding
`dp-1419-lethal` combined census completed clean after rechecking
`spank-sleeping-depth` (`full=CLEAN_AFTER_RECHECK`,
`claims=CLEAN_AFTER_RECHECK`, 3,395 pairs).

Required gates, all run independently after `make fmt`:

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...`; `go vet ./...` | PASS separately |
| `go test ./...`; `go test ./pkg/game/...`; `go test -race ./pkg/game/...` | PASS separately |
| `golangci-lint run ./...` | PASS: 0 issues |
| `make deadcode-ratchet` | PASS: 52 baselined, 0 new |
| `make fidelity-depth` | PASS: 5,518 cases; 5,426 proven/delegated; 12 blocked; 68 excluded; 12 divergent-approved |
| `make fidelity-units` | PASS: 1,423 rows, 1,309 symbols, 1,604 claims |
| `make string-census` | PASS: 94 Go-only, ratchet 98 known segments / 0 unreviewed; generated reports were stale, and were not updated |
| `gosec -severity=high -quiet ./...` | PASS |

Do not mark the two Lua depth rows oracle-green until the targeted census is
clean; do not open the train PR until its final combined census and evidence
are recorded.
