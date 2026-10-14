# Acknowledged Gap Sweep — Train 9 — 2026-10-14

## Scope and inherited inventory

Train 9 starts from freshly fetched `origin/main` at
`d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`, on
`codex/luna-acknowledged-gap-sweep-9`. It closes the inventoried detailed
`show shops` report gap as one wizard shop-report family. The full pre-edit A/B
inventory and triage remain in the base handoff
[`2026-10-13-acknowledged-gap-sweep.md`](2026-10-13-acknowledged-gap-sweep.md),
with the complete comment and assigned-special inventories in
[`2026-10-13-acknowledged-gap-sweep-comments.tsv`](2026-10-13-acknowledged-gap-sweep-comments.tsv)
and
[`2026-10-13-acknowledged-gap-sweep-specials.tsv`](2026-10-13-acknowledged-gap-sweep-specials.tsv).
Those inventories were captured before the first production edit and are not
rewritten by this train.

## Triage

| Inventory item | Disposition | Evidence |
|---|---|---|
| Detailed `show shops` report | Fixed in this train; was the copy-ready fix candidate in the base handoff | C `do_show` routes the value to `show_shops` (`src/act.wizard.c:2413-2415`); `show_shops` selects list, current-room, or 1-based numeric detail (`src/shop.c:1413-1444`). |
| Live shop bank/cash dependency | Required runtime support for the report | C initializes bank state to zero at boot (`src/shop.c:1206`), credits the keeper and sweeps bank-enabled overflow after buys (`src/shop.c:584-589`), pays sales from keeper cash plus bank and refills after sales (`src/shop.c:763-812`). |
| Other A/B candidates | Retain their base-handoff dispositions and owner boundaries | The complete table and Claude-ready issue text stay in the base handoff; this train does not alter those items. |

## Implementation

- `cmdShow` now handles `show shops <number>` and `show shops .`; no-argument
  listing and its pager are unchanged. The detailed report reproduces C field
  order, line endings, 78-byte list wrapping, room names/unknown markers,
  customers, products, buy types/keywords, rates/hours, and bit labels.
- Room selection preserves `ok_shop_room`'s comparison with `real_room(1)`
  (`src/shop.c:932-939`). Room specials use `COrderedRoomMobs` per
  `src/interpreter.c:1452-1466`; the report's live keeper lookup uses the
  newest matching mobile in C's global character-list order
  (`src/handler.c:887-895`, with `read_mobile` prepending at
  `src/db.c:1745-1750`). `SHOP_FUNC` reports an already-assigned keeper special
  (`src/shop.c:1233-1237`; VNUM 1 is assigned `puff` at
  `src/spec_assign.c:182`). The detail uses the keeper NPC's short description:
  `list_detailed_shop` calls `GET_NAME(&mob_proto[...])` (`src/shop.c:1342-1344`),
  and `GET_NAME` resolves NPC names to `player.short_descr`
  (`src/utils.h:275-276`).
- `game.Shop.Bank` is runtime-only and starts at zero, matching C boot. Buy and
  sell dispatch now account for keeper coins and bank state; world-lock and
  mob-lock operations keep each balance transition atomic. Existing shop edits
  preserve bank state as C `copy_shop` does (`src/sedit.c:143-158`). No save
  format or C/oracle source changed.
- The oracle vehicle `show-shops-detail` warms with `goto 12133` and probes
  `show shops .`. C's `ok_shop_room` fallback compares a shop room to
  `real_room(1)`, so the first matching shop is VNUM 2508, room Limbo (#1),
  which the report exposes. Unit proofs cover numeric and room selectors,
  empty/missing fields, assigned and ordinary keepers, wrapping, real buy/sell/
  show dispatch, and bank transitions.

## R5h controls

Both controls were run against the reverted behavior and then restored:

| Removed behavior | Control result |
|---|---|
| Detailed selector/report | `go test ./pkg/session -run '^TestShowShopDetailedReportAndLiveBankThroughDispatch$' -count=1` failed: got only `"\r\n"` instead of the detailed report including `Coins: [     7777], Bank: [     1000] (Total: 8777)`. Restored selector; focused test passes. |
| Buy-side bank overflow | `go test ./pkg/game -run '^TestShopAccountPurchaseAndSaleBanking$' -count=1` failed at the overflow assertion: `coins=16000 bank=0`, wanted `15000/1000`. Restored bank sweep; focused test passes. |

The report, account, sale-affordability and keeper-order proofs pass after
restoration. The report/account code adds no RNG draws (R3).

## Gates

Each gate ran separately on the restored implementation:

| Gate | Result |
|---|---|
| `make fmt` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./...` | PASS |
| `go test ./pkg/game/...` | PASS |
| `go test -race ./pkg/game/... ./pkg/session/...` | PASS |
| `golangci-lint run ./...` | PASS, 0 issues |
| `make deadcode-ratchet` | PASS, 52 baselined and 0 new |
| `make fidelity-depth` | PASS, 5,522 cases; 5,432 proven/delegated; 99.8% actionable completion |
| `make fidelity-units` | PASS, 1,425 rows, 1,313 symbols, 1,608 claims |
| `make string-census` | PASS, 98 known Go-only segments, 0 unreviewed; existing 24 stale baseline entries and generated reports noted |
| `gosec -severity=high -quiet ./...` | PASS |

## Census and delivery

The first targeted census, `dp-1371-train9-show-shops-detail`, found a real
name-field divergence: Go printed keeper keywords (`island shaman man`) where C
printed the NPC short description (`the island shaman`). After matching C's
`GET_NAME` call path, targeted census
`dp-1371-train9-show-shops-detail-recheck` passed: 1 scenario passed, no
failures, verdict `CLEAN`, at Go HEAD `d8e8024dd`; retained at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train9-show-shops-detail-recheck/`.
The final combined census ran on committed Train 9 tip `8aa563a33`:
`oracle-combined: pairs=3402 deduplicated=1018 full=CLEAN claims=CLEAN
elapsed=1961.399s verdict=CLEAN`. The `show-shops-detail` row passed. Evidence
is retained at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train9-final/`.

This is one shop-report/account family, below the train boundary. Open its PR
without merging. Train 8's targeted Medusa census is CLEAN, and its final
combined census is CLEAN_AFTER_RECHECK for both full and claims, with only the
known infrastructure row `clan-plan-mortal-depth` rechecked; those results are
recorded in the Train 8 handoff. The next train starts from freshly fetched
`origin/main` and remains separate from this PR.
