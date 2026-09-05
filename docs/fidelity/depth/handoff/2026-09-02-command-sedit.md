# Depth handoff — 2026-09-02 — command `sedit`

The `sedit` depth slice is blocked after two honest RED attempts. The C path
was audited from `src/interpreter.c:684` through `src/olc.c:74-277`,
`src/sedit.c:817-1179`, and the `CON_SEDIT` dispatch at
`src/interpreter.c:1738-1739`. C reaches `do_olc` for the no-argument,
nonnumeric, and unknown-zone entry branches, then enters the descriptor-driven
SEDIT menu and consumes `q` through `sedit_parse`; Go has no matching OLC
state-machine surface and falls through to `Huh?!?`.

The two proof vehicles are `sedit-entry-depth` and `sedit-session-depth`, both
RED on main with `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and
seed 1. The first proves the exact no-argument, nonnumeric, unknown-zone, and
interactive menu divergence. The second proves the interactive boundary and
that Go misroutes the following `q` to quaff while C consumes it in
`CON_SEDIT`. Existing `redit` and `medit` manifests establish this as a shared
interactive OLC blocker. No Go behavior was invented; neither `src/` nor the C
oracle tree was edited. This preserves R1/R2/R4/R5e and the shared-class rules
R5b/R5c.

The manifest is `docs/fidelity/depth/sedit.tsv`, with four blocked rows. Local
gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`; the
oracle-tree diff check was also clean. Feature/evidence PR #1155
(`glm/depth-sedit`) passed hosted lint, security, and test checks and was
self-merged to `main`.

The frontier after this slice is:

- 3194 total cases
- 3112 proven/delegated
- 30 blocked
- 52 excluded

The one-time blocked-row vehicle remains satisfied: the cast-sleep outlaw and
reagent arms prove `objmagic.sleep-entry-gates.cast` through
`sleep-spell-depth`; the separate object-magic entry row remains blocked.

The next unclaimed source-order interpreter entry is `serpent` at
`src/interpreter.c:685`, registered `POS_FIGHTING` with
`do_serpent_kick` and minimum level 1. Begin it only after the required fresh
main checkout/pull, `make fidelity-depth`, depth-guide read, newest-handoff
read, and source/table audit. Never edit `src/` or the C oracle tree.
