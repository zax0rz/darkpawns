# Round 1 repair batch 2: switched shop customer receipt

## Repair

Sample row 2 exposed a switched mobile shop customer who received the keeper's
shop tell but missed C's direct purchase receipt. Filed as
[DP-1606](https://linear.app/labz0rz/issue/DP-1606/switched-mobile-shop-customer-misses-cs-purchase-receipt-r1).
The Go shop path now sends the same receipt to the switched customer's
descriptor after the keeper tell. An ordinary descriptorless mobile remains
silent. The added depth row records the affordability tell and purchase receipt.

The C path is `do_buy` in `src/shop.c`: the affordability branch tells the
keeper at lines 557–562, the shop purchase tell is sent at lines 579–580, then
`send_to_char` writes `You now have %s.\n\r` at lines 581–582. The descriptor
transfer in `src/act.wizard.c:1196–1200` explains why switched-mobile output
reaches the player's connection.

## Witness

Final Go commit: `3af3e4ba04eb6a03a5c99fc257287eb1d087978a`.

Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-go-shop-final-cite`

Result: `CLEAN`, 1/1 scenario, zero draw divergences, zero hollow scenarios.
The C and Go transcript both contain these lines in order:

1. `Hansen the merchant tells you, 'You can only afford 1.'`
2. `Hansen the merchant tells you, 'That'll be 28 coins, thanks.'`
3. `You now have a golden ring.`

## Revert proof

The isolated mutant commit `496e9c117b7b4c59cc11cf23a93ba8c5f07cac72`
changes one byte at offset 11959 in `pkg/game/shop_npc.go`, from `s` to `Q`,
changing the format string `%s` to `%Q`. It builds with
`GOEXPERIMENT=nojsonv2 go build ./...`.

Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-2-final`

Result: `NOT_CLEAN`, 1 failure, zero draw divergences. The diff carries the
mutated `%Q` bytes and shows Go losing the affordability tell while C retains
`Hansen the merchant tells you, 'You can only afford 1.'`.

## Gates and review

All required build, vet, test, lint, modernization, dead-code, string census,
fidelity-depth, expected-divergence, and function-index gates passed on the
final Go change. The documentation-only citation correction was followed by
passing `make fmt`, `make fidelity-depth`, and `make function-index-check`.

Read-only Sol review passed all five checklist items on
`3af3e4ba04eb6a03a5c99fc257287eb1d087978a`. The C source citations and final
transcript were checked directly.

## Decisions for Zach

No new ruling. Round-level open decisions remain in
[`DECISIONS.md`](../DECISIONS.md).

## For Claude

Please review the row-2 fix, its C-first witness, and the isolated mutant proof.
