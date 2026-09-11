# Shop live-inventory review follow-up — 2026-09-11

Branch: `glm/depth-shop-live-inventory`, review head `6b29666c2` plus the
uncommitted correction and coverage changes for PR #1437.

## Scope

This follow-up addresses only the two stocked-control failures reported in
PR #1437. It does not alter the empty-keeper fixture, shop consolidation, or
other modernization work.

The C path is `shopping_list()` in `src/shop.c:877-925`: it calls
`is_ok()`, walks `keeper->carrying`, filters `CAN_SEE_OBJ` and positive cost,
groups with `same_obj()`, and checks `found` before processing the final
group. C keyword matching is `isname()` in `src/handler.c:81-110`, which
requires a complete token. The live Go path is `cmdList()` and
`shopListKeywordMatches()` in `pkg/session/shop_cmds.go`; the correction is
isolated there.

## Before and after

The supplied reproductions remain preserved at
`/tmp/dp-review-1437-final-group.log` and
`/tmp/dp-review-1437-prefix.log`. Before the correction, Go printed the
final cinnamon row for `list cinnamon` and matched black pepper for `list
pep`; C printed `Presently, none of those are for sale.` for both commands.

The stocked-control fixture now runs `list`, `list pepper`, `list pep`, and
`list cinnamon`. Fresh `--show-oracle` runs at seeds 1 and 2 are preserved in
`control-final-seed-1.txt` and `control-final-seed-2.txt`, and the raw logs
are also retained at:

```text
/tmp/dp-review-1437-shop-live-inventory-control-seed1-final.log
/tmp/dp-review-1437-shop-live-inventory-control-seed2-final.log
```

Both seeds report `result: no normalized divergence`; the control proves the
keeper is reached, the shop is associated, the carried inventory is stocked,
`list pepper` returns ordinal 2, `list pep` returns the fixed C no-match
bytes, and `list cinnamon` returns the same no-match bytes despite the final
group existing. The complete relevant scenario census entries are PASS for
`shop-live-inventory-gate`, `shop-live-inventory-control`,
`shop-live-inventory-filters`, and `shop-stack-list-live`.

## Coverage and evidence

The manifest row `shops.live-inventory-control` remains D4
`oracle-green-multiseed`; its note now cites `src/handler.c:81-110` and
names both quirks. The new focused unit test is
`pkg/session/shop_cmds_test.go`. The final census record is
`docs/fidelity/depth/evidence/2026-09-10-shop-live-inventory/oracle-regression-review-1437.md`.

## Gates and census

On the uncommitted candidate, these gates passed with
`PATH=/usr/local/go/bin:$PATH`:

```text
make check-fmt
go build ./...
go vet ./...
go test ./...
go test ./pkg/game/...
make lint
make fidelity-depth
```

The final census was run with the exact command and environment preserved in
`docs/fidelity/depth/evidence/2026-09-10-shop-live-inventory/oracle-regression-review-1437.md`:

```text
scenarios=938 passed=928 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
exit=2
```

Exit 2 is solely the previously human-cleared `accuse-noarg-depth` baseline.
The census had zero failed, infra, timed-out, or stale results. Its one
transient infrastructure-shaped retry recovered to PASS and was manually
inspected; it was not filed as INFRA.

`make expected-divergences-check` remains a separate blocker. The fresh
command exited 2 because it regenerated the unrelated
`character-creation-name-retry / entry.motd-color` row from `entry.tsv`,
making the committed ledger stale. The raw output is preserved at
`/tmp/dp-review-1437-expected-divergences-final.log` with SHA-256
`b0a3e6f89f2c0fcad6415e359f071b80944ec71759273ccc2de3a9e43006ac98b`.
That generated row was removed again; no pin or exclusion was added.
