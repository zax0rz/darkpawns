# Acknowledged Gap Sweep — Train 6 — 2026-10-14

## Inventory record and current triage

The complete pre-fix inventories are retained in the original dated
[handoff](2026-10-13-acknowledged-gap-sweep.md): section A is the admitted-comment
inventory, section B covers heartbeat, room/object/point/mobile/weather behavior
and assigned `SPECIAL`s. The itemized A inventory is
[2026-10-13-acknowledged-gap-sweep-comments.tsv](2026-10-13-acknowledged-gap-sweep-comments.tsv);
the complete 89-name C assignment/definition/Go registration crosswalk is
[2026-10-13-acknowledged-gap-sweep-specials.tsv](2026-10-13-acknowledged-gap-sweep-specials.tsv).
The baseline files remain unchanged in this train so their before-first-edit
capture remains auditable. PR #1956 carries the current shared-TSV dispositions
through train 5; this handoff records the new train 6 delta.

| Triage | Current disposition |
|---|---|
| Fixed in earlier sweep PRs | Player `APPLY_RACE_HATE` (#1943); Lua object-target spell and location gate (#1953); Kai-zai spell-ID gate (#1954); mobile-activity, scavenger visibility, player-hunter `set_hunt`, and live mobile-actor `canget` (#1955); Lua mobile follow (#1956). These PRs remain open except #1943, which is merged. |
| Fixed in train 6 | Lua `iscorpse` now matches the C type/value predicate. Lua character tables now serialize the complete acyclic master chain rather than stopping after 32 links. |
| Fixable in later trains | `show shops` report (`pkg/session/wiz_info.go:237`; `src/shop.c:1413`, `src/act.wizard.c:2414`); water/flying wandering gates (`pkg/game/ai.go:79`; `src/mobact.c:130-138`, `src/utils.h:148-153`); `DoTurn` room-wide race and mana/RNG flow (`pkg/game/skills2.go:492-567`; `src/new_cmds2.c:960-1030`); corpse attack-number mapping (`pkg/game/death.go`; C death path in `src/fight.c`); jail-guard visibility (`pkg/combat/engine.go:954`; `src/fight.c:1374`). |
| Blocked depth rows | AGGR24+AGGRESSIVE mobile target visibility and global point-update object timers remain blocked pending their recorded breadth/depth work. Do not claim the assigned-special registration crosswalk proves branch behavior. |
| Owned/tracked | DP-1426 (`flow_room`, `loud_mobs`, CON croak); DP-1419 lethal login replay; DP-1364 ordering; DP-493 `hunt_items`; shared combat attack/message/RNG transcript; generic NPC command interpreter. Preserve their existing owners and rows. |
| Stop tier | DP-1423/1424/1425; entry/nanny, persistence, login, session lifecycle, transport, and save formats. The date-record, help-usage, `do_write` editor, tattoo timer, and house-object persistence findings remain here. |
| Exclusions | `get_obj` has no callers; ordinary stock object prototypes have no identify bitvector field or gameplay setter; stale comments and security sandbox stubs do not admit missing gameplay behavior. Evidence is in the baseline handoff and train 5 handoff. |

For B, the original inventory maps heartbeat order and character point-update
to lifecycle/DP-1364 rows; `room_activity`'s owned omissions to DP-1426;
`object_activity`'s global object walk and branch gates to existing rows;
point-update object timers to a blocked depth row; and mobile activity to the
533-row `spec-procs.tsv` ledger plus the 89-name assignment crosswalk. No
additional missing Go registration was found. Core weather progression is
mapped; full-moon descriptor work remains blocked/stop-tier. Train 6 changes no
pulse or special-procedure behavior.

## Train 6 findings and C call paths

`luaIsCorpse` in `pkg/scripting/engine.go` compared the Lua `type` field to 18,
which C names `ITEM_KEY`, and ignored the fourth Lua value. C
`lua_iscorpse` calls `IS_CORPSE` (`src/scripts.c:636-654`); the macro is exactly
`GET_OBJ_TYPE == ITEM_CONTAINER && GET_OBJ_VAL(obj, 3) == 1`
(`src/utils.h:490-491`; `ITEM_CONTAINER` is 15 at `src/structs.h:434`). The
bridge's object table stores C value slots 0–3 at Lua indices 1–4. The fix
requires type 15 and Lua `val[4] == 1`; it does not infer corpse state from
keywords or the Go-only `IsCorpse` field. It retains C's nil return for a valid
non-corpse table and the existing invalid-argument diagnostic/no-values arm.

`charToTable` in `pkg/scripting/bridge.go` stopped following `Master` at depth
32. C's `char_to_table` recursively emits `leader` until `master` is nil or
self (`src/scripts.c:1823-1891`, especially 1870-1873). C's follower path
rejects loops in `circle_follow` (`src/utils.c:365-375`). The fix follows the
live chain and explicitly omits a self-master, matching C. It adds no RNG
draws (R3).

## Proofs and controls

| Proof | Result |
|---|---|
| `TestBridgeIsCorpseUsesContainerValue` | PASS for container/value 1, container/value 0, container/value 2, and non-container/value 1 through `Engine.RunScript`. |
| Corpse R5h control | RED after changing the type predicate back to 18: container/value 1 returned `nil` instead of `1`, and non-container/value 1 returned `1` instead of `nil`. Restored code passes. |
| `TestBridgePreservesLeaderChainsBeyondFormerLimit` | PASS for a 40-link chain through the Lua engine. |
| `TestLuaLeaderChainBeyondFormerLimit` | PASS through the real `WorldScriptableAdapter`, live player follower state, mobile `say`, and Lua `oncmd` execution. |
| Leader-chain R5h control | RED with the former 32-level cap restored: both engine and real-world tests failed at the missing `leader` table. Restored code passes both. |
| `lua-iscorpse-container-value` | CLEAN targeted census, one scenario passed. It uses Lua `oload(ch, 1210, "char")`; the stock object has type 15, value[3] 1, and no corpse keyword. |

An exploratory 33-peer leader-chain oracle scenario returned `INFRA` on its
initial attempt and automatic recheck, not a content divergence. It required 34
same-IP logins; the Go entry limiter allows a burst of 10 and refills at 5/sec
(`pkg/auth/ratelimit.go:189-190`). The per-attempt runner logs were cleaned, so
the exact failed login was not retained. The fixture was removed rather than
claiming an oracle green. The 40-link real-world adapter test is the retained
proof; `lua.bridge-leader-chain` is therefore `unit-green`, not `oracle-green`.

Retained targeted census evidence:

- `luna-train6-lua-iscorpse`: preliminary CLEAN for an added `lua-bind-queries`
  probe; that shared scenario edit was removed after overlap review with PR
  #1955, and the dedicated scenario above is the final oracle proof.
- `luna-train6-leader-chain`: NOT_CLEAN, one `INFRA` row after automatic
  recheck; no FAIL/STALE/content divergence. The oversized peer scenario was
  removed.
- `luna-train6-iscorpse-container-value`: CLEAN, `PASS lua-iscorpse-container-value`.

## Open-PR coordination

Train 6 starts from freshly fetched `origin/main` at
`d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`. PR #1955 edits
`pkg/scripting/engine.go` for `lua_canget`; train 6 edits the separate
`luaIsCorpse` function in the same file and deliberately does not duplicate the
`canget` fix. If either PR lands first, rebase the other and retain both
function bodies. PR #1955 and PR #1956 also add rows to `docs/fidelity/depth/lua.tsv`;
train 6 adds distinct leader-chain and corpse-value rows. Train 6 leaves the
shared comments TSV unchanged while PR #1956 carries its train 5 updates. It
also uses a dedicated corpse scenario, avoiding PR #1955's edit to
`lua-bind-queries.txt`.

PRs #1953-#1956 were open when this train began; #1943 was merged. Do not merge
any PR as part of this handoff.

## Required gates and combined census

The goal's required gates are recorded separately below; this section is
completed before opening the train 6 PR. The full combined census runs once on
the final code tip and retains separate full/claims verdicts.

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS |
| `go test ./pkg/game/...` | PASS |
| `golangci-lint run ./...` | PASS, 0 issues |
| `make deadcode-ratchet` | PASS, 52 baselined and 0 new |
| `make fidelity-depth` | PASS, 5,523 cases; 5,433 proven/delegated; 10 blocked; 99.8% actionable completion |
| `make fidelity-units` | PASS, 1,426 rows, 1,314 symbols, 1,609 claims |
| `make string-census` | PASS, 98 known go-only segments and 0 unreviewed; generated reports remain stale and the check warned to run `make string-census-update` |
| `gosec -severity=high -quiet ./...` | PASS |
| Full combined census on code tip `5444ac29356e9aacbc91bb34095ef176ba489666` | CLEAN_AFTER_RECHECK for full and claims; 3,402 pairs, 1,018 deduplicated, 2,002.366s; `mudlog-local-depth` was rechecked |

The retained run is
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train6-integrated`.
Its `summary.txt` records `verdict=CLEAN_AFTER_RECHECK`; `results.tsv` records
`PASS lua-iscorpse-container-value 10`. Both combined verdicts are clean after
the one permitted infrastructure recheck.
