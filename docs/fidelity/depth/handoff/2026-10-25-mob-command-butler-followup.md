# Mobile command-call lane — butler follow-up — 2026-10-25

This follow-up starts from the merged mobile command-call lane on
`origin/main` (`44eb9c360`) and covers the six `SPECIAL(butler)` door calls in
`src/spec_procs3.c:172-192`.

## Inventory correction

The six `gen_door` rows now pass C's exact literals (`"case"`, `"cabinet"`,
and `"chest"`) through `World.mobGenDoor` for open and close. The map now has
173 rows: 133 fixed, 1 proven, 10 delegated, 28 unreachable, and 1 blocked
(`breed_killer`).

`TestSpecButler_GenDoorCallBytesMatchC` checks the exact room bytes for all six
calls in C's order, including each open/close message and all six successful
`ok_pick` draws. `TestSpecButler_GenDoorUsesGenericFindInventoryFirst` covers
each literal with an inventory shadow container. C's `generic_find` selects
that inventory object before the room object, so no door act reaches the room;
the room object stays closed while `perform_put` still uses the selected room
container pointer. Temporarily restoring the direct pointer toggle made all
three table cases fail on the extra open/close bytes. The route was restored,
and the focused butler tests pass.

`mobGenDoor` now consumes `number(1, 101)` after a command passes the C gate
ladder, matching `ok_pick` at `src/act.movement.c:536-542`; refused commands
consume no draw. This is required for C/Go draw parity on the new open/close
route.

## Verification

Passed individually: `make fmt`, `go build ./...`, `go vet ./...`,
`go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...` (0
issues; one warning names an absent sibling worktree), `make deadcode-ratchet`
(52 baselined, 0 new), `make fidelity-depth` (42 maps, 0 errors),
`make fidelity-units` (1,949 claims), `make string-census` (ratchet passes;
generated reports are stale), and `gosec -severity=high -quiet ./...`.

The targeted `spec-proc-butler` census verdicts with `--draw-parity`:

| Seed | Census | Verdict | Draw divergence |
|---:|---|---|---:|
| 1 | `luna-mob-command-butler-followup-seed1` | CLEAN | 0 |
| 2 | `luna-mob-command-butler-followup-seed2` | CLEAN | 0 |
| 3 | `luna-mob-command-butler-followup-seed3` | CLEAN | 0 |
| 5 | `luna-mob-command-butler-followup-seed5` | CLEAN | 0 |
| 8 | `luna-mob-command-butler-followup-seed8` | CLEAN | 0 |

## Delivery

The follow-up is PR #2153 for Claude review. No full census, merge, or deploy
is included in this handoff.
