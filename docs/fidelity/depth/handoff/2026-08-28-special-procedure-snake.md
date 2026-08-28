# Dated Handoff: 2026-08-28 (special-procedure snake reachability slice)

The active C `snake` procedure has been audited and explicitly excluded on
`main` at `36c8cabd0` (PR #692, self-merged after all hosted checks passed).

## Queue and inventory

The round started from current `main`, pulled with `--ff-only`, reconfirmed the
depth frontier with `make fidelity-depth`, and re-read
`docs/fidelity/DEPTH_TESTING.md` and the newest handoff. The refreshed C
inventory remains:

- 113 `SPECIAL` definitions across `spec_procs.c`, `spec_procs2.c`, and
  `spec_procs3.c`;
- 233 active `ASSIGNMOB` calls;
- 228 unique active mob VNUMs;
- 66 unique final mob-procedure names after later C assignments win.

The source-ordered queue reached `snake` at `src/spec_procs.c:330`. C assigns
it to mob VNUMs 14103, 14127, and 14415.

## R5e call-path finding

The actual C dispatch path was traced before changing the manifest:

- command input reaches `src/interpreter.c:947`, which calls `special()`;
- `src/interpreter.c:1451-1456` invokes the mob procedure with a nonzero
  command ID, and `src/spec_procs.c:332-333` makes `snake` return `FALSE`;
- the only commandless special invocation is the mobile-activity call at
  `src/mobact.c:82-92`, but `src/mobact.c:68-72` skips every mob already in
  combat, while `src/spec_procs.c:335` requires `GET_POS(ch) == POS_FIGHTING`.

Thus the poison-bite body cannot be reached from the registered C dispatch
surface. A spawned fighting snake vehicle would prove combat behavior, not the
`snake` special, and would violate R5e. The owning manifest records one sharp
D5 `excluded` row and deliberately claims no vacuous oracle scenario or Go
change.

## Verification

The manifest frontier is now 535 total cases: 523 proven/delegated, 1 blocked,
and 11 excluded; actionable completion remains 523/524 (99.8%). All local
slice gates passed:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (0 issues);
- `gofumpt -l .` (empty).

Hosted PR #692 lint, security, and test checks passed; build/deploy jobs were
correctly skipped for the fidelity-only manifest change. Neither `src/` nor
the C-oracle tree was edited.

## Next action

Return to current `main` and continue with the next active, unclaimed
source-ordered procedure, `thief` (`src/spec_procs.c:389`; mob VNUMs 3119,
12127, 18218, and 21242). Leave `objmagic.sleep-entry-gates` blocked until
the special-procedure inventory is exhausted.
