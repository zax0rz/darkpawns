# Depth handoff — 2026-09-02 — command `sethunt`

The `sethunt` depth slice is complete. PR #1153 (`glm/depth-sethunt`) passed
hosted lint, security, and test checks and was self-merged to `main` as
`7c2c890f6` under R1/R2/R3/R4/R5e. The implementation follows the registered
C path from `src/interpreter.c:683` through `src/act.wizard.c:3444-3473` and
`handler.c:1276-1332`: C gate level and position, half-chop ordering, visible
world target resolution, pointer identity, PC-hunter rejection, higher-victim
level comparison, exact LF/CR bytes, and live `MOB_HUNTER`/hunting state.

The proof vehicle `sethunt-depth` is green on seeds 1, 2, 3, 5, and 8; seed 1
was run with `--show-oracle`. `TestSethuntRegistrationUsesCEntryGate` pins the
registered gate and `TestCmdSethuntSetsHunterState` pins the state transition.
The repository gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and `gofumpt -l .`.

The frontier after the merge is:

- 3190 total cases
- 3112 proven/delegated
- 26 blocked
- 52 excluded

The required one-time blocked-row vehicle is already satisfied and remains
explicitly bounded: `objmagic.sleep-entry-gates.cast` is green through the
cast-sleep outlaw/reagent arms in `sleep-spell-depth`, while the separate
object-magic entry row `objmagic.sleep-entry-gates` remains blocked as recorded
in `docs/fidelity/depth/object-magic.tsv`; do not re-pick or collapse the
unreachable remainder (R4/R5b/R5c).

The source-order interpreter sweep now reaches `sedit` at
`src/interpreter.c:684`, registered as `POS_DEAD` with shared `do_olc`,
`LVL_BUILDER`, and `SCMD_OLC_SEDIT`. Existing `redit` and `medit` manifests
show the missing interactive OLC state machine is a blocked shared family;
audit `sedit` once and preserve that boundary rather than inventing an OLC
surface. Begin it only after the fresh main checkout/pull,
`make fidelity-depth`, depth-guide read, newest-handoff read, and source/table
audit required by the loop.

Fidelity rules for the next slice remain R1/R2/R3/R4/R5e, with shared ownership
bounded by R5b/R5c. Never edit `src/` or the C oracle tree.
