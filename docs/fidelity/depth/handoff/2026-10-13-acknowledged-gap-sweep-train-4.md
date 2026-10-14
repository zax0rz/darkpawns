# Acknowledged Gap Sweep — Train 4 (2026-10-13)

## Scope and source audit

This train started from freshly fetched `origin/main` at
`9b3ecaae3d02cb1f2b066e9dd27c81bb5ec350fc`. It was rebased onto fresh
`origin/main` at `d8e8024ddbfe52fbde36d16e0b02e2d7606fac71` after PRs #1943,
#1944, and #1945 merged, on `codex/luna-acknowledged-gap-sweep-4`. PRs #1953
and #1954 were open and are not part of this base at handoff time. The pre-fix inventory is committed in
Train 1 and PR #1943: `2026-10-13-acknowledged-gap-sweep.md`, its comments TSV,
and its assigned-special crosswalk.

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

## Remaining triage and copy-ready issue text

The mobile-leader `lua_follow` candidate remains for the next Lua action train;
`lua.bind-follow-mobile-leader` is now a blocked row. The object-timer and
AGGR24 mobile-selection gaps also have blocked rows before any future fix.
The combat memory/helper branches call the shared combat transcript path. Keep
that common attack/message/RNG behavior with its owner; the missing source-
specific target gates and pre-hit acts remain separately inventoried pending
an explicit owner-boundary decision. Existing copy-ready issues for
`DoTurn`, `show shops`, the Lua character/mobile table, water capability, and
jail-guard visibility remain in the Train 1 handoff.

### Complete point-update timers across the global object list

Go's `decayObjectsInRoom` only traverses room-floor objects
(`pkg/game/limits_condition.go:327-363`), while C `point_update` walks
`object_list` (`src/limits.c:529-684`) and also handles worn/carried objects in
`src/handler.c:1046-1082`. Please add live object-location coverage for moon
gates, puddles/puke, dust, circles, corpses and field objects. Preserve global
list order, timer semantics, corpse-content spill destinations, the carried
corpse message, room audiences, and the corpse decay draw. The blocked
`point-update.global-object-list-timers` row names the required proof. Cite
R1/R3/R5e.

### Apply visibility before the AGGR24 mobile target is selected

C's AGGR24+AGGRESSIVE full-health branch skips a mobile that fails
`CAN_SEE(ch, vict)` before choosing the first weaker target
(`src/mobact.c:324-336`; visibility predicates at `src/utils.h:515-530`). Go's
`pkg/game/mobact.go:471-485` selects the first lower-level mobile without that
gate. Please add a fail-capable pulse proof with an invisible first candidate
and visible later candidate. Preserve C room order and coordinate with DP-1364
and the `mob.teleport-victim-combat-transcript` owner before changing combat
entry or transcript behavior. Cite R1/R3/R5e.

### Restore the missing memory and helper mobile pulse arms

C `mobile_activity` checks memory targets by player ID, rejects invisible and
`PRF_NOHASSLE` targets, then emits the speech/growl room act before `hit`
(`src/mobact.c:261-282`). Its helper branch first assists a fighting charmed
master, then scans room order and requires a visible player target before its
room act and `hit` (`src/mobact.c:284-302`). Go's corresponding branches at
`pkg/game/mobact.go:382-439` start combat without those C pre-hit acts and use
different helper gates. Please handle these branches with the shared combat
transcript owner, preserve C's scan order and draw timing, and add pulse proofs
for each audience and refusal gate. Cite R1/R3/R5e.

### Identify's object affect bitvector is zero for ordinary stock objects

C `spell_identify` prints `obj->obj_flags.bitvector` through `affected_bits`
(`src/spells.c:476-510`). `src/db.c:1372,1399-1457` clears each object
prototype and parses type/extra/wear flags, values, and `A` affect records;
there is no object affect-bitvector field in that format. `read_object`
clears then copies the prototype (`src/db.c:1947-1951`), so ordinary spawned
stock objects print NOBITS in C. A source-wide `rg -n 'obj_flags\.bitvector'
src` found no gameplay setter: the only writes are loading and saving the word
in player object records (`src/objsave.c:60-88,102-120`); other hits read it.
Go's parser likewise has no object bitvector field, and its `AffectFlags`
runtime slot has no production writer; `SaveItemData` does not serialize it.
Therefore keep the current NOBITS result for ordinary stock objects. Any
nonzero legacy rent-record value is a persistence/save-format question, which
this sweep must not change. Disposition: evidence-based exclusion for ordinary
world instances (R1/R4).

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
reported no normalized divergence. Their targeted census is clean: two passed,
zero expected, unpinnable, stale, failed, infra, timeout, or unstable; elapsed
22.024s. The retained summary and manifest are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train4-lua-bindings/`.
The `lua.bind-set-hunt-player-hunter` and `lua.bind-canget-world-actor` rows
are now `oracle-green`. The retained relevant normalized blocks are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-13/dp-1371-train4-show-oracle/`;
the `lua.bind-canget-world-actor` probe reports `canget invisible=nil` after
the live mobile's detect-invisible effect is removed. The train's final
combined census remains pending after rebase.

Required gates, all run independently after `make fmt`:

The branch is based on the rebased `origin/main` noted above. All gates below
were rerun separately after that rebase and after the two targeted rows became
`oracle-green`.

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...`; `go vet ./...` | PASS separately |
| `go test ./...`; `go test ./pkg/game/...`; `go test -race ./pkg/game/...` | PASS separately |
| `golangci-lint run ./...` | PASS: 0 issues |
| `make deadcode-ratchet` | PASS: 52 baselined, 0 new |
| `make fidelity-depth` | PASS: 5,529 cases; 5,436 proven/delegated; 13 blocked; 68 excluded; 12 divergent-approved |
| `make fidelity-units` | PASS: 1,428 rows, 1,316 symbols, 1,611 claims |
| `make string-census` | PASS: 94 Go-only, ratchet 98 known segments / 0 unreviewed; generated reports were stale, and were not updated |
| `gosec -severity=high -quiet ./...` | PASS |

The two targeted Lua depth rows are oracle-green. Keep
`lua.bind-follow-mobile-leader` blocked until a real mobile-leader proof exists.
Do not open this train's PR until the final combined census and evidence are
recorded.
