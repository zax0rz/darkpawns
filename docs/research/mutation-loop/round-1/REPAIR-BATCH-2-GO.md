# Round 1 repair batch 2: switched shop customer receipt

## Repair

Sample row 2 exposed a switched mobile shop customer who received the keeper's
shop tell but missed C's direct purchase receipt. Filed as
[DP-1606](https://linear.app/labz0rz/issue/DP-1606/switched-mobile-shop-customer-misses-cs-purchase-receipt-r1).
The Go shop path now sends the same receipt to the switched customer's
descriptor after the keeper tell. An ordinary descriptorless mobile remains
silent. The added depth row records the affordability tell and purchase receipt.

The live C path is `SPECIAL(shop_keeper)`'s `buy` branch at
`src/shop.c:981–984`, which calls `shopping_buy`. That function formats the
partial-affordability message at lines 557–562 and uses `do_tell` at line 570.
It sends the purchase tell at lines 579–580, then writes
`You now have %s.\n\r` directly with `send_to_char` at lines 581–582. The
descriptor transfer in `src/act.wizard.c:1196–1200` binds the mobile to the
player's connection while switched.

## Witness

Final Go commit: `3af3e4ba04eb6a03a5c99fc257287eb1d087978a`.

Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-go-shop-final-cite`

Result: `CLEAN`, 1/1 scenario, zero draw divergences, zero hollow scenarios.
The C and Go transcript both contain these lines in order:

1. `Hansen the merchant tells you, 'You can only afford 1.'`
2. `Hansen the merchant tells you, 'That'll be 28 coins, thanks.'`
3. `You now have a golden ring.`

## Revert proof

The pre-fix Go implementation was run against the same fixture in
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-witnesses-v3`.
That run is `NOT_CLEAN` (three scenarios, two passed, zero draw divergences).
Its two `buy 2 gold` attempts both show C's `You now have a golden ring.` and
the Go block missing that receipt.

The receipt-specific one-byte mutant commit
`e34f0542e9803b661a8f333f713f3bd4f072c103` changes byte 13491 in
`pkg/game/shop_npc.go` from `e` to `Q`, turning `You now have` into
`You now havQ`. It builds with `GOEXPERIMENT=nojsonv2 go build ./...`.

Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-receipt-mutant`

Result: `NOT_CLEAN`, one failure, zero draw divergences. Its `buy 2 gold`
diff shows C's `You now have a golden ring.` and Go's mutated
`You now havQ a golden ring.`

The separate Round 1 sampled mutant commit
`496e9c117b7b4c59cc11cf23a93ba8c5f07cac72` changes one byte at offset 11959
in `pkg/game/shop_npc.go`, from `s` to `Q`, changing `%s` to `%Q`. It builds
with `GOEXPERIMENT=nojsonv2 go build ./...`.

Run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-2-final`

Result: `NOT_CLEAN`, 1 failure, zero draw divergences. The Go source diff
contains the mutated `%Q` bytes. The oracle diff for `buy 2 gold` shows Go
losing the affordability tell while C retains
`Hansen the merchant tells you, 'You can only afford 1.'`.

## Gates and review

On Go change commit `3af3e4ba04eb6a03a5c99fc257287eb1d087978a`, each of these
commands exited 0: `make fmt`, `GOEXPERIMENT=nojsonv2 go build ./...`,
`GOEXPERIMENT=nojsonv2 go vet ./...`, `GOEXPERIMENT=nojsonv2 go test ./...`,
`golangci-lint run ./...`, `make modernize-check`, `make deadcode-ratchet`,
`make string-census`, `make fidelity-depth`,
`make expected-divergences-check`, and `make function-index-check`. After the
documentation-only citation edit, `make fmt`, `make fidelity-depth`, and
`make function-index-check` also exited 0.

Read-only Sol review passed all five checklist items on
`3af3e4ba04eb6a03a5c99fc257287eb1d087978a`. The C source citations and final
transcript were checked directly.

## Decisions for Zach

No new ruling or decision entry is needed for this batch.

## For Claude

Please review the row-2 fix, its C-first witness, and the isolated mutant proof.
