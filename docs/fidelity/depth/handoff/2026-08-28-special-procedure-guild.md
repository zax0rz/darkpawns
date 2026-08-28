# Dated Handoff: 2026-08-28 (special-procedure guild slice)

The `guild` special-procedure slice is complete on `main` at `49c4d1167`
(PR #691, self-merged after all hosted checks passed).

## Frontier and inventory

This session started from a clean, current `main`, ran `make fidelity-depth`,
and re-read `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff. The
C census was refreshed from the active `ASSIGNMOB` dispatch lines in
`src/spec_assign.c` and the `SPECIAL` definitions in
`src/spec_procs.c`, `src/spec_procs2.c`, and `src/spec_procs3.c`:

- 113 `SPECIAL` definitions across the three C files;
- 233 active `ASSIGNMOB` calls;
- 228 unique active mob VNUMs;
- 66 unique final mob-procedure names after C's later-assignment wins.

The procedure queue is source-definition ordered within the C files, filtered
to active C assignments and excluding procedures already claimed by a prior
handoff. `guild` was the next item after the prior `key_seller` slice. The next
unclaimed source-ordered item is `snake` (`src/spec_procs.c:330`, active mob
assignments 14103, 14127, and 14415).

## `guild` call path and proof

C's reachable path was verified before changing Go:

- `src/interpreter.c:947` calls `special()` before the registered command;
- `src/interpreter.c:1407-1456` scans room mobs and invokes the assigned
  procedure;
- `src/spec_procs.c:201-249` implements `SPECIAL(guild)`, which handles
  `practice`, calls `list_skills()`, and owns the zero-practice, unknown-skill,
  already-learned, ordinary-practice, and learned-cap branches.

The existing `guild-practice` vehicle was RED on clean main: Go emitted the
practice-count prelude while the live C oracle emitted the catalog beginning
with `You know of the following skills:`. The same renderer-class RED appeared
in the standalone `skills-practice` vehicle.

The Go-only fix removes that invented prelude from the shared renderer, avoids
double-CRLF framing in the guild gate messages, and adds exact focused tests
for the remaining gate and state branches. The owning manifest gained five
rows: three oracle-green rows and two unit-green rows.

GREEN evidence:

- `guild-practice --show-oracle --seed 1`: no normalized divergence;
- `skills-practice --show-oracle --seed 1`: no normalized divergence;
- `TestSpecGuild_Golden` and `TestSpecGuild_Gates`: pass.

This follows R1, R4, R5c, and R5e. Neither `src/` nor the C-oracle tree was
edited.

## Gates

All local gates passed on the branch:

- `make fidelity-depth`: 534 total, 523 proven/delegated, 1 blocked, 10 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (0 issues);
- `gofumpt -l .` (empty).

Hosted PR #691 checks were all green: lint, security, and test passed; the
deployment/build jobs were correctly skipped for this fidelity-only change.

## Next action

Return to `main`, pull before the next slice, and continue with active C
procedure `snake` in source-definition order. The one blocked
`objmagic.sleep-entry-gates` row remains untouched until the special-procedure
inventory is exhausted, per the standing queue.
