# Depth-fidelity handoff — `mosh`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R5b, R5c, R5e

## Frontier

Started from fresh `main` after the `mold` handoff.  `make fidelity-depth`
reported 2,544 cases: 2,480 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).  After adding the `mosh` slice and merging it, the
frontier is 2,556 total: 2,492 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).

The next un-manifested command family is `medit`, the registered C row at
`src/interpreter.c:553` (`POS_DEAD`, `LVL_BUILDER`, `do_olc`,
`SCMD_OLC_MEDIT`).  No existing depth manifest row claims `medit`.

## C path and proof

The queue item was `{ "mosh", POS_RESTING, do_action, 0, 0 }` at
`src/interpreter.c:552`.  Its social record is `mosh 0 5` with the eight
authored messages at `lib/misc/socials:1302-1310`.  The actual
`src/act.social.c:102-151` path is:

- reject `PLR_NOSHOUT` before social lookup output;
- because `char_found` exists, parse only the first target token with
  `one_argument`;
- emit the no-argument actor/room pair when no target is supplied;
- return the record-specific missing-target bytes for an unresolved target;
- emit `char_auto`/`others_auto` for self-targeting;
- reject a victim below the record's `min_victim_position` (`POS_RESTING`);
- otherwise emit the actor, non-victim room, and victim messages with C's
  substitution and audience rules.

Clean-main proof was initially blocked by a fixture role typo: the harness
does not recognize an ad hoc `target` role, so that setup ended before command
dispatch.  The role was corrected to the established `peer` role; this was a
vehicle correction, not a C/Go behavior result.  A second clean-main run with
`--show-oracle` was GREEN across the full matrix.  A dedicated audience
vehicle was added because the sleeping peer in the full matrix correctly
receives no ordinary room act; it separately proves the awake third-player
room audience.

Added:

- `cmd/dp-oracle-diff/scenarios/mosh-depth.txt` — no argument, first-token
  target parsing, target success/victim output, missing target, self-target,
  and sleeping-victim position refusal.
- `cmd/dp-oracle-diff/scenarios/mosh-audience-depth.txt` — awake actor/target/
  observer target-success audience topology.
- `pkg/session/mosh_depth_test.go` — C command gate plus generated social
  metadata and all eight authored messages.
- `docs/fidelity/depth/mosh.tsv` — 12 manifest rows, with shared position,
  noshout, and visibility behavior delegated to existing proof owners.

Both scenarios passed at seeds 1, 2, 3, 5, and 8.  Seed 1 was run with
`--show-oracle`; all normalized blocks matched, including actor, target,
awake-room observer, self-target, missing-target, and sleeping-victim paths.
No Go implementation divergence was found, so no player-facing Go code was
changed.  No C source or oracle-tree files were edited.

## Gates and merge

Local gates passed on `glm/depth-mosh`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature commit `fdcef5a53` was submitted as PR #1024 (`glm/depth-mosh`).
Hosted `test`, `lint`, and `security` checks were green; `build-and-push` and
`deploy` were skipped by the workflow for this PR.  PR #1024 was self-merged
to `main` as `cbb5bdcde`.
