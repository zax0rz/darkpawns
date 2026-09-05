# Depth-fidelity handoff: `hush`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family sweep after
the special-procedure inventory and the one-time `objmagic.sleep-entry-gates` attempt.
The preceding command handoff was `hump`; `hush` is the registered social at
`src/interpreter.c:507`. Before this slice, the depth manifest frontier was 2,249 total
cases: 2,189 proven/delegated, 16 blocked, and 44 excluded. This slice adds 11
manifest rows, bringing the frontier to 2,260 total: 2,200 proven/delegated, 16
blocked, and 44 excluded (2,200/2,216 actionable, 99.3%). The howl handoff PR (#953)
remains open because its hosted checks did not fire after the single permitted retry;
that unresolved PR is carried forward and was not merged.

The next source-order unclaimed command is `idlist` at `src/interpreter.c:512`.
`inventory`, `idea`, and `ident` are already represented by `inventory.tsv`,
`gen-write.tsv`, and `gen-tog.tsv`, respectively.

## C call path and branch inventory

The command table registers `hush` as `POS_RESTING`, `do_action`, with no level
minimum. `do_action` in `src/act.social.c:102-151` uses the social record in
`lib/misc/socials:1506-1514`: `hush 0 0`, with the authored no-argument,
target-success, self-target, missing-target, and target-audience messages. The C
`one_argument` path consumes the first token and ignores trailing words before
visible-room target lookup. A target is admitted at any position because the social
record's victim-position minimum is zero. The `TO_NOTVICT`, `TO_VICT`, and ordinary
room/actor audience branches are shared communication behavior; the sleeping-target
vehicle verifies that C emits the actor and room lines while the sleeping target
receives no `TO_VICT` bytes. The command position rejection and `PLR_NOSHOUT`
refusal are delegated to existing depth vehicles.

## RED/GREEN result

The focused Go test and oracle scenarios were run on main before the slice and were
GREEN across deterministic seeds 1, 2, 3, 5, and 8. The awake vehicle covers no
argument, fill-word/trailing argument, visible target, self target, and missing target;
the sleeping vehicle covers the zero victim-position gate and audience behavior.
No confirmed Go divergence was found, so this was a proof-only slice. No `src/` or
`darkpawns-c-oracle/` files were edited.

## Verification

All required local gates passed on the completed feature slice: `make fidelity-depth`,
`go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a
clean `gofumpt -l .` check. The feature was merged as PR #960 (`7b5cc7d86`) after
its hosted test, lint, and security checks were green; build/deploy were skipped by
the workflow. The proof obeys R1/R3 and the C-path audit obeys R4/R5e.

## Next session

Start from clean `main`, pull, rerun `make fidelity-depth`, reread the depth-testing
guide and newest handoff, then take `idlist` in source-table order. Keep the open,
not-green howl PR unmerged.
