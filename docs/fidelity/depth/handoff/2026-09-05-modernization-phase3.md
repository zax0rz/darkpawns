# Modernization Phase 3 — shop stack handoff

Date: 2026-09-05
Branch: `codex/modernize-phase3-shops`
Base: `origin/main` after PR #1387 merge (`340d129f4c875e53d5158093c27ba9b3fd952c98`)

## Scope and ruling

This round implements the Phase 3 row in `docs/modernization/06-roadmap.md`:
consolidate the remaining shop duplication to one engine, while keeping claims
limited to C-byte/state proof. The shadow-stack ruling retains the live
`pkg/game` shop engine and removes the dead duplicate in `pkg/game/systems` plus
the command layer that depended on it. The session commands now resolve against
the parsed shops held by the world's one `*game.ShopManager`.

The bridge corrections landed before consolidation in this same change:

- `ShopProto.Products` is retained as `Shop.SellTypes` instead of being
  discarded. The parser also carries buy types and the C shop profits.
- All seven C shop messages are retained and the no-such-item/list branches in
  the live commands use the C strings. The keeper tell path strips the C
  buyer-name prefix before rendering the already-prefixed tell line.
- `shop-stack-list-live` was changed from a bare `spawn-mob` keeper to the
  authored zone-121 M/G reset. Reset commands load keeper 12100 and four spice
  objects into room 12133 before the player arrives. Removing the parsed
  products or bypassing live keeper lookup makes the scenario fail; the former
  pre-bridge output was:

  ```text
  [list sword] C Presently, none of those are for sale.
             Go Sorry, but you cannot do that here!
  [buy sword] C The fat merchant tells you, 'Sorry, I don't stock that item.'
             Go Sorry, but you cannot do that here!
  ```

  After the bridge and consolidation, the focused oracle diff reports no
  normalized divergence and preserves those C blocks.

## Deliberate depth boundary

The scenario proves the stocked-definition bridge and the C no-such-item/list
messages. It does not yet prove the full C live-inventory gate. In
`src/shop.c:877-925`, `shopping_list` iterates the keeper's carried objects and
applies C's `is_ok`/`CAN_SEE_OBJ` and positive-cost checks; static `.shp`
products alone are not sufficient. The current vehicle uses the same authored
reset to make the keeper's inventory real, but does not create the required
static-definition/live-carried-inventory mismatch. That is the real depth item,
not a reason to relabel the coincidence-green predecessor scenario.

The item-3 case is therefore blocked for Lane B. Existing residual rows remain
blocked at `docs/fidelity/depth/surface-inventory.tsv:69-71` (core and adjacent
shop surfaces) and `docs/fidelity/depth/show.tsv:24` (`show.shops-list`).

The new coverage map is `docs/fidelity/depth/shop.tsv`:

- `shops.bridge.stocked-definition` is oracle-green through
  `shop-stack-list-live`.
- `shops.live-inventory-gate` is explicitly blocked pending a YELLOW vehicle
  that separates static products from keeper-carried stock.

## Changed-file coverage map

- `cmd/dp-oracle-diff/scenarios/shop-stack-list-live.txt` —
  `shops.bridge.stocked-definition`.
- `pkg/parser/shop.go` and `pkg/parser/shop_test.go` — parser field-order/unit
  proof; the parsed fields feed the stocked-definition scenario.
- `pkg/game/shop.go` and `pkg/game/world.go` — one retained engine and the
  parsed product/message bridge; `shops.bridge.stocked-definition`.
- `pkg/session/shop_cmds.go` — live list/buy output; the stocked-definition row
  is proven, while the live-inventory row remains blocked.
- `pkg/session/manager.go`, `pkg/session/combat_command_messages_test.go`, and
  `cmd/server/main.go` — single-manager wiring and regression compatibility;
  exercised by the focused scenario and package tests.
- Deleted `pkg/game/systems/shop*.go` and `pkg/command/shop_commands*.go` —
  dead shadow stack removed per the handoff at
  `docs/fidelity/depth/handoff/2026-09-04-shadow-shop-stack.md`; no separate
  player-facing engine remains.

Because core shop transaction rows remain blocked and the changed live shop
files map to that uncovered surface, this branch is HUMAN-MERGE-ONLY under the
three-condition self-merge amendment. Do not self-merge unless later proof
maps every changed shop file to a proven row.

## Verification

Focused post-consolidation regression:

```text
oracle-regression: scenarios=1 passed=1 failed=0 infra=0 timed_out=0 elapsed=15.867s started=2026-09-05T02:43:42-0400 finished=2026-09-05T02:43:58-0400
```

Full `make oracle-regression` tally:

```text
oracle-regression: scenarios=934 passed=934 failed=0 infra=0 timed_out=0 elapsed=7316.966s started=2026-09-05T02:49:39-0400 finished=2026-09-05T04:51:36-0400
```

The runner recovered eight transient infrastructure-shaped retries; the final
authoritative tally is `infra=0` and `timed_out=0`.

Standard gates passed before the corpus run: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...` (`0 issues`), gofumpt, and
`git diff --check`.

Rules applied: R1 player-facing bytes, R4 no invention, and R5e actual call
path/source verification.
