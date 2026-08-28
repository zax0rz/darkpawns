# Depth-fidelity handoff: `fido`

Date: 2026-08-28
Branch: `glm/spec-fido`
Starting main: `db1f68d8a` (`fix: deepen puff special procedure (#700)`)

## Frontier

Before this slice, `make fidelity-depth` reported 569 total cases, 556 proven/delegated, 2 blocked, and 11 excluded: 556/558 actionable, or 99.6%.

This slice adds five manifest rows. The expected post-slice frontier is 574 total, 561 proven/delegated, 2 blocked, and 11 excluded: 561/563 actionable, or 99.6%.

## Queue position and inventory

`fido` is the next unproven special procedure in file order after `puff` in `src/spec_procs.c:724-748`. Its active registrations are mob VNums 8063 (`src/spec_assign.c:294`), 12115 (`:349`), 15108 (`:394`), and 18203 (`:403`). The next special-procedure queue item is `janitor` at `src/spec_procs.c:750`, registered first at VNum 8061 (`src/spec_assign.c:292`) and also at 21229 (`:505`).

## C call path and player-visible partition

The C mobile loop in `src/mobact.c:68-93` skips fighting or non-awake mobs, then invokes the registered special with `cmd == 0` and an empty argument. The command path in `src/interpreter.c:1407-1456` can also dispatch a mobile special, but `fido` rejects nonzero `cmd`.

`SPECIAL(fido)` at `src/spec_procs.c:724-748` therefore has these observable branches:

- return `FALSE` for fighting, non-commandless, asleep, or negative-HP entry;
- scan room contents in C linked-list order;
- handle only `ITEM_CONTAINER` objects with nonzero `GET_OBJ_VAL(i, 3)`;
- emit `"$n savagely devours a corpse."` to `TO_ROOM`;
- detach every contained object with `obj_from_obj`, move it to the room with `obj_to_room`, extract the corpse, and return `TRUE`;
- otherwise return `FALSE` without output.

The Go change in `pkg/game/spec_procs.go` removes the old random gate and keyword heuristic, uses the typed container/value predicate, emits the canonical `Act` room bytes, and performs the content transfer and corpse extraction through the ObjectLocation movement APIs with checked errors. No C or oracle-tree files were changed.

## Proof

Clean-main RED was run from `db1f68d8a` with `spec-proc-fido`, seed 1. The C oracle emitted `The hybrid warg savagely devours a corpse.` to both actor and peer on the pulse; the pre-fix Go port emitted nothing.

Focused GREEN tests:

- `TestSpecFido_EntryGates` covers command, fighting, sleeping, and negative-HP gates.
- `TestSpecFido_UsesContainerValueNotKeyword` rejects keyword-only and non-container false positives.
- `TestSpecFido_CorpsePredicateAndTransfer` verifies exact room output, canonical object locations, empty corpse contents, and corpse removal.

The C-first scenario is `cmd/dp-oracle-diff/scenarios/spec-proc-fido.txt`. It uses active mob 8063, strips its script, creates a native corpse through the God vehicle, and pads one autonomous pulse. The pulse block is GREEN with no normalized divergence for seeds 1, 2, 3, 5, and 8. An early probe placed the disposable `kill trainee` command in the compared block and exposed an unrelated pre-existing C/Go death-message mismatch; it was moved to warmup so the fido proof does not claim or fix that outside-slice divergence.

Manifest rows added: `mob.fido-entry-gates`, `mob.fido-corpse-predicate`, `mob.fido-corpse-devour`, `mob.fido-content-transfer`, and `mob.fido-pulse-dispatch`.

This slice follows R1 (player-facing bytes), R2 (command surface), R4 (no invention), R5b/R5c (audit the whole reachable class and delegate shared paths), and R5e (verify the actual C call path).

## Verification and handoff

`make fidelity-depth` passes at 574/561/2/11. Focused fido tests and the five oracle seeds pass. The final diff also passes `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`.

After this PR is merged only when all GitHub checks are green, refresh `main`, rerun `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take `janitor`. The blocked `objmagic.sleep-entry-gates` row remains after the special-procedure inventory and before the interpreter command-family sweep.
