# Acknowledged Gap Sweep — Train 5 (2026-10-14)

## Inventory continuity and scope

The complete pre-fix Go-comment inventory (A), silent heartbeat/object/point
update/mobile-activity/weather inventory, and all-assigned-SPECIAL crosswalk
(B) were captured before the first production fix in
[`2026-10-13-acknowledged-gap-sweep.md`](2026-10-13-acknowledged-gap-sweep.md),
with the itemized comment inventory in
[`2026-10-13-acknowledged-gap-sweep-comments.tsv`](2026-10-13-acknowledged-gap-sweep-comments.tsv)
and the 89-SPECIAL crosswalk in
[`2026-10-13-acknowledged-gap-sweep-specials.tsv`](2026-10-13-acknowledged-gap-sweep-specials.tsv).
This train updates current dispositions in the comments TSV; it preserves the
original pre-fix findings and the assigned-SPECIAL audit.

Train 5 started from fetched `origin/main` at
`d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`, on
`codex/luna-acknowledged-gap-sweep-5`. At the time of the final gate run, sweep
PRs #1953, #1954, and #1955 were open. The code commit is `3edfe2fb0` and is
pushed to `origin/codex/luna-acknowledged-gap-sweep-5`.

## C call path and fix

`src/scripts.c:542-569` implements `lua_follow`: it reads the selected table's
`struct char_data *`, passes `ch->player.name` to `do_follow(me, ..., 0, 0)`,
sets `AFF_CHARM` afterward when requested, and returns the selected table's
struct userdata. For a mobile, `player.name` is its keyword list.

`src/act.movement.c:883-950` parses the first word, resolves it with
`get_char_room_vis` in the follower's room, then checks current master, charm,
self-follow, `circle_follow`, old-master teardown, `AFF_GROUP`, and follower
notices. `src/handler.c:1277-1300` defines room-order, visibility, keyword
abbreviation, ordinal, and player-only lookup semantics. `src/utils.c:365-375`
defines the master-chain cycle check; `397-440` defines follower teardown;
`480-498` defines add-follower notices and their visibility/awake gate. These
call paths were grepped directly in `src/` during this train.

`WorldScriptableAdapter.Follow` now passes the selected character's keyword
name through the same room resolver rather than following the selected remote
mobile pointer or logging that mobile leaders are unsupported. The room
resolver accepts a `combat.Combatant` observer, so mobile callers use the same
visible room-list lookup. The mobile follow path mirrors C's no-op, charm,
self-stop, circle, old-master, group-bit, and add-follower branches. Lua's
post-follow charm bit remains applied after the follow attempt. This adds no
random draws (R1/R3/R5e).

## Proofs and controls

`TestLuaFollowMobileLeaderResolvesKeywordInFollowerRoom` runs the real Lua
engine and adapter. It selects remote mobile 9002 with `inworld`, whose
`guard` keyword resolves to local mobile 9003. It also exercises repeat follow,
reparenting and `AFF_GROUP` clearing, charm refusal, cycle rejection, and
self-stop. The C/Go scenario `lua-follow-mobile-leader` uses stock mobs: remote
gate guard 4818 and local prison guard 8089. The transcript contains the local
guard's room notice.

R5h negative control: temporarily restored the old bridge branch that warned
and ignored a mobile leader. The test failed as intended at the following-body
assertion (`got nil`, wanted the local guard); the old branch also logged
`lua follow of a mobile leader is not ported; ignored`. Restored implementation
passed the test and the focused race run.

The first targeted census attempt stopped during setup because its 13-character
test password exceeded C's 10-character limit; no probe command ran. The
fixture password was shortened, and the corrected targeted census is clean:

```text
oracle-regression: scenarios=1 passed=1 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 elapsed=11.821s verdict=CLEAN
```

Retained run: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train5-follow-mobile-corrected/`.

The required full combined census was started on commit `3edfe2fb0` as
`dp-1371-train5-integrated`. It completed at 2128.796s:

```text
oracle-combined: pairs=3402 deduplicated=1018 full=CLEAN_AFTER_RECHECK claims=CLEAN_AFTER_RECHECK elapsed=2128.796s verdict=CLEAN_AFTER_RECHECK rechecked=combat-entry-charm-relations,dragon-kill-depth,tedit-boundaries-depth,tiger-god-depth
PASS  lua-follow-mobile-leader  seed 10
```

Retained run: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train5-integrated/`.

## Gates

All required gates were run separately after the final integration test was
expanded. They passed:

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...`; `go vet ./...` | PASS separately |
| `go test ./...`; `go test ./pkg/game/...` | PASS separately |
| Focused `go test -race ./pkg/game -run '^TestLuaFollowMobileLeaderResolvesKeywordInFollowerRoom$' -count=1` | PASS |
| `golangci-lint run ./...` | PASS: 0 issues |
| `make deadcode-ratchet` | PASS: 52 baselined, 0 new |
| `make fidelity-depth` | PASS: 5,522 cases; 5,432 proven/delegated; 10 blocked; 68 excluded; 12 divergent-approved |
| `make fidelity-units` | PASS: 1,425 rows, 1,313 symbols, 1,608 claims |
| `make string-census` | PASS: 94 Go-only; 98 known segments, 0 unreviewed. Generated reports were stale; they were not updated. |
| `gosec -severity=high -quiet ./...` | PASS |

## Current disposition and PR coordination

The comments TSV now marks Lua object-target spell dispatch and its location
gate (Train 2 / PR #1953), Kai-zai (Train 3 / PR #1954), the mobile-activity,
scavenger, and player-hunter `set_hunt` fixes (Train 4 / PR #1955), and Lua
mobile follow (this train) as fixed in the sweep. Ordinary stock-object
identify bitvectors are recorded as an evidence-based exclusion; the
unserialized legacy rent-record value remains stop-tier. Water/flying mobile
wandering is reclassified as fixable: Go's `roomActivityHasBoat` already
supports mobile waterwalk, flight, and boats, while `src/utils.h:148-153`
defines the C `CAN_SWIM` and `IS_FLYING` checks. The jail-guard visibility gate
remains a separate fix candidate. The approximate corpse attack mapping is
also fixable: C passes `attacktype` into `make_corpse` on the death path, while
Go maps several different numbers for energy drain (21 vs C 25), shocking
grasp (31 vs C 37), psiblast (34 vs C 100), and petrify (35 vs C 104), and
omits meteor swarm (C 41). AGGR24 selection and global point-update object
timers retain blocked depth rows; memory/helper attack transcript work remains
with its named owner. Further fix candidates for later trains are `DoTurn`,
detailed `show shops`, water/flying wandering, corpse attack mapping,
jail-guard visibility, the Lua `iscorpse`/`canget` bindings, and full Lua
leader-chain tables. Date records,
string editing, help-miss usage logging, house object persistence, and tattoo
timer persistence remain stop-tier under DP-1423/1424/1425 and the goal's
entry/nanny, login, transport, session, and save-format boundaries. The
script-table review found that
production `RunScript` always uses the C-field bridge; the empty `skills` and
mobile `wear` tables are in the legacy test-only fallback, and skill-group
scripts are archived, so those comments are excluded. The DoMindlink
“simplified” label is classified stale: all reachable player/NPC branches have
depth proof, and the apparent success branch is excluded by C's `!IS_NPC`
macros. Stop-tier boundaries are documented in the original handoff.

PR #1955 also adds `lua.bind-follow-mobile-leader` as blocked. This train marks
that same depth row `oracle-green`, so the PRs share a depth-manifest line and
will need merge-order coordination or a rebase. PR #1953 and #1955 also touch
`pkg/game/world_bridge.go`; their changes are independent of this follow path.
