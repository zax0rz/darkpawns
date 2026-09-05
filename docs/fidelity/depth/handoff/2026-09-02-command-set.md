# Depth handoff — 2026-09-02 — command `set`

The `set` depth slice is complete. PR #1151 (`glm/depth-set`) passed hosted
lint, security, and test checks and was self-merged to `main` as
`b53a4d6c1` under R1/R2/R3/R4/R5e. The implementation follows the registered
C path from `src/interpreter.c:682` through `src/act.wizard.c:2523-3069`,
including the ordered field table, target resolution, optional `player`/`mob`
prefixes, C remainder parsing and `atoi` behavior, target and per-field level
gates, PC/NPC gates, binary/misc/numeric branches, clamps, exact acknowledgements,
and live NPC state overrides. The old partial Go handler was removed so
`cmdSetText` is the sole authoritative path.

The proof set is `set-depth`, `set-gate-depth`, and `set-extended-depth`, each
green on seeds 1, 2, 3, 5, and 8; focused session tests pin the ordered field
table and parser contracts. The repository gates passed: `make fidelity-depth`,
`go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and
`gofumpt -l .`.

The frontier after the merge is:

- 3181 total cases
- 3103 proven/delegated
- 26 blocked
- 52 excluded

The source-order interpreter sweep rechecked the command table after `set`.
The next genuinely un-manifested command family is `sethunt` at
`src/interpreter.c:683`, registered as `POS_DEAD` with `do_sethunt` and
`LVL_GRGOD`. Begin it only after the fresh main checkout/pull,
`make fidelity-depth`, depth-guide read, newest-handoff read, and C-path/
manifest audit required by the loop. The blocked
`objmagic.sleep-entry-gates` row remains pending its one cast-sleep vehicle.

Fidelity rules for the next slice remain R1/R2/R3/R4/R5e, with shared ownership
bounded by R5b/R5c. Never edit `src/` or the C oracle tree.
