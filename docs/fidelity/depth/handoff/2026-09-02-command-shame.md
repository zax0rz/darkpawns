# Depth-fidelity handoff — `shame`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `shadow` handoff. The special-procedure inventory remains exhausted.
The one-time blocked row `objmagic.sleep-entry-gates` remains bounded after its
cast-sleep outlaw/reagent vehicle; the separate object-magic entry row must not
be re-picked. The interpreter sweep advanced from `shadow` to `shame` in table
order.

The frontier immediately before the shame slice was 3,238 total; 3,156
proven/delegated; 30 blocked; 52 excluded. The shame manifest contributes 11
cases, producing 3,249 total and 3,167 proven/delegated before the concurrently
observed main update that also integrated the qecho evidence. The fresh
post-merge checkpoint reports the current repository frontier as:

- 3260 total cases
- 3178 proven/delegated
- 30 blocked
- 52 excluded

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:688 */
{ "shame"    , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. It resolves the `shame` social,
checks `PLR_NOSHOUT`, parses the first argument with `one_argument`, performs
visible-room target lookup, and selects the no-argument, target-found,
not-found, self-target, or victim-position branches. The record in
`lib/misc/socials:1366-1374` is `shame 0 0`: hide-invisible is false and the
minimum victim position is zero. Its complete authored record therefore emits
the actor/ordinary-room no-argument pair, actor/TO_NOTVICT/victim target trio,
`Who?!?` for a missing target, and the authored self-target pair. A sleeping
visible target is admitted; the plain TO_VICT delivery is suppressed for that
sleeping recipient. Shared POS_RESTING, PLR_NOSHOUT, target visibility, and Act
audience behavior remain delegated to their owning evidence under R5b/R5c.

The clean-main probes found no confirmed Go divergence, so no implementation
change was warranted under R4. No `src/` or `darkpawns-c-oracle/` file was
edited.

## Evidence and confirmed parity

Manifest: `docs/fidelity/depth/shame.tsv` (11 rows)

Vehicles:

- `cmd/dp-oracle-diff/scenarios/shame-depth.txt`
- `cmd/dp-oracle-diff/scenarios/shame-sleeping-depth.txt`

The normal vehicle is green at seeds 1, 2, 3, 5, and 8; seed 1 used
`--show-oracle` and showed the intended actor, target, observer, self, and
not-found C blocks. The sleeping-target vehicle is also green at seeds 1, 2,
3, 5, and 8; seed 1 used `--show-oracle` and showed the actor/observer pair
with no sleeping victim byte. The focused
`TestShameRegistrationUsesCEntryGate` test pins the POS_RESTING command gate,
zero social hide/victim metadata, and all eight C-authored message slots.

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

Feature branch: `glm/depth-shame`

Feature commit: `275beb78d` (`test: prove shame depth fidelity`)

Feature PR: #1163 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. It was self-merged only after all
required checks were green, as main commit `8fe6beb36`.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). Shared do_action gates, target lookup, and
Act audience behavior were handled as class-level evidence under R5b/R5c.

The next unclaimed source-order interpreter entry is `shishkabob` at
`src/interpreter.c:689`, registered to `do_action` with POS_RESTING, level 0,
and no fighting restriction. Begin it only after a fresh `main` checkout/pull,
`make fidelity-depth`, depth-guide read, newest-handoff read, and source/table
audit.
