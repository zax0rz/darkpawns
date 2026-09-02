# Depth-fidelity handoff — `shishkabob`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `shame` handoff. The special-procedure inventory remains exhausted.
The one-time blocked row `objmagic.sleep-entry-gates` remains bounded after its
cast-sleep outlaw/reagent vehicle; the separate object-magic entry row must not
be re-picked. The interpreter sweep advanced from `shame` to `shishkabob` in
table order.

The frontier before this slice was 3,260 total; 3,178 proven/delegated; 30
blocked; 52 excluded. The shishkabob manifest contributes 11 cases, producing
the current frontier:

- 3271 total cases
- 3189 proven/delegated
- 30 blocked
- 52 excluded

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:689 */
{ "shishkabob",POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. It resolves the `shishkabob`
social, checks `PLR_NOSHOUT`, parses the first argument with `one_argument`,
performs visible-room target lookup, and selects the no-argument, target-found,
not-found, self-target, or victim-position branches. The record in
`lib/misc/socials:675-685` is `shishkabob 0 5`: hide-invisible is false and the
minimum victim position is POS_RESTING. Its complete authored record emits the
actor/ordinary-room no-argument pair, actor/TO_NOTVICT/victim target trio,
`Sorry good chef, but that person doesn't seem to be here.` for a missing
target, and the authored self-target pair. A sleeping visible target is
rejected before any success audience act. Shared POS_RESTING, PLR_NOSHOUT,
target visibility, and Act audience behavior remain delegated to their owning
evidence under R5b/R5c.

The clean-main probes found no confirmed Go divergence, so no implementation
change was warranted under R4. No `src/` or `darkpawns-c-oracle/` file was
edited.

## Evidence and confirmed parity

Manifest: `docs/fidelity/depth/shishkabob.tsv` (11 rows)

Vehicles:

- `cmd/dp-oracle-diff/scenarios/shishkabob-depth.txt`
- `cmd/dp-oracle-diff/scenarios/shishkabob-sleeping-depth.txt`

The normal vehicle is green at seeds 1, 2, 3, 5, and 8; seed 1 used
`--show-oracle` and showed the intended actor, target, observer, self, and
not-found C blocks. The sleeping-target vehicle is also green at seeds 1, 2,
3, 5, and 8; seed 1 used `--show-oracle` and showed C's proper-position refusal
with no success audience bytes. The focused
`TestShishkabobRegistrationUsesCEntryGate` test pins the POS_RESTING command
gate, zero social hide metadata, the POS_RESTING victim minimum, and all eight
C-authored message slots.

## Verification and integration

All required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-shishkabob`

Feature commit: `0ddbf69c3` (`test: prove shishkabob depth fidelity`)

Feature PR: #1165 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. It was self-merged only after all
required checks were green, as main commit `5ca494776`.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). Shared do_action gates, target lookup, and
Act audience behavior were handled as class-level evidence under R5b/R5c.

The next unclaimed source-order interpreter entry is `shiver` at
`src/interpreter.c:693`, registered to `do_action` with POS_RESTING, level 0,
and no fighting restriction. Begin it only after a fresh `main` checkout/pull,
`make fidelity-depth`, depth-guide read, newest-handoff read, and source/table
audit.
