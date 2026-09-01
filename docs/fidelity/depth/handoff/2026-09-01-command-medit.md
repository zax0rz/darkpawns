# Depth-fidelity handoff — `medit`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R4, R5e

## Frontier

Started from fresh `main` after the `mosh` handoff.  `make fidelity-depth`
reported 2,556 cases: 2,492 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).  After the two honest `medit` attempts and merging the
blocked evidence, the frontier is 2,560 total: 2,492 proven/delegated, 22
blocked, and 46 excluded (99.1% actionable).

The next un-manifested command family is `mlist`, the registered C row at
`src/interpreter.c:554` (`POS_DEAD`, `LVL_BUILDER`, `do_mlist`).  No existing
depth manifest row claims `mlist`.

## C path and proof

The queue item was `{ "medit", POS_DEAD, do_olc, LVL_BUILDER,
SCMD_OLC_MEDIT }` at `src/interpreter.c:553`.  C `do_olc` in `src/olc.c:80-275`
performs command argument parsing, save handling, zone and permission checks,
duplicate-editor checks, descriptor OLC allocation, and dispatch into
`medit_setup_new`/`medit_setup_existing`.  The valid path then enters the
descriptor-driven `MEDIT_MAIN_MENU` and `medit_parse` state machine in
`src/medit.c:612-1121`, covering mobile text, numeric fields, flags, scripts,
save confirmation, and cleanup.

The first honest clean-main vehicle covered no argument, nonnumeric input, and
an unknown-zone numeric VNUM.  C emitted respectively:

- `Specify a mobile VNUM to edit.`
- `Yikes!  Stop that, someone will get hurt!`
- `Sorry, there is no zone for that number!`

Go had no `medit` command registry entry or OLC handler and emitted `Huh?!?`
for all three.  The second honest vehicle supplied valid mobile VNUM `18305`:
C entered and printed the full `MEDIT_MAIN_MENU`, while Go again emitted
`Huh?!?`; the following `q` was misrouted to Go's unrelated quaff alias.  This
confirms a missing descriptor/state-machine surface, not a safe one-line
message correction.

After exactly two genuine RED attempts, the four observed cases are marked
`blocked` in `docs/fidelity/depth/medit.tsv` with the exact C sites and the
reason no substitute was invented.  No Go implementation was changed; no C
source or oracle-tree files were edited.  The queue advances under the
two-attempt rule.

Added:

- `cmd/dp-oracle-diff/scenarios/medit-entry-depth.txt` — the three entry
  branches and exact C/Go divergence.
- `cmd/dp-oracle-diff/scenarios/medit-session-depth.txt` — valid mobile OLC
  entry and the missing Go descriptor surface.
- `docs/fidelity/depth/medit.tsv` — four sharp blocked rows.

## Gates and merge

Local gates passed on `glm/depth-medit`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature/evidence commit `12363a40a` was submitted as PR #1026
(`glm/depth-medit`).  Hosted `test`, `lint`, and `security` checks were green;
`build-and-push` and `deploy` were skipped by the workflow for this PR.  PR
#1026 was self-merged to `main` as `c6bbbc717`.
