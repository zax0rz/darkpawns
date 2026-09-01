# Depth-fidelity handoff — `massage`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R4, R5b, R5c, R5e

## Frontier

Started from fresh `main` after the `motd` handoff.  `make fidelity-depth`
reported 2,572 cases: 2,504 proven/delegated, 22 blocked, and 46 excluded
(99.1% actionable).  After the `massage` evidence was merged, the frontier is
2,583 total: 2,515 proven/delegated, 22 blocked, and 46 excluded (99.1%
actionable).

The previous `motd` handoff named `mail` as next, but `mail` is already claimed
by the existing generic `do_not_here` manifest (`docs/fidelity/depth/do-not-here.tsv`)
and was not re-picked.  The next actually un-manifested source-table family is
`mount` at `src/interpreter.c:558` (`POS_STANDING`, unrestricted level,
`do_ride`).

## C path and proof

The queue item was `{ "massage", POS_RESTING, do_action, 0, 0 }` at
`src/interpreter.c:557`.  C `do_action` in `src/act.social.c:102-151` checks
`PLR_NOSHOUT`, consumes one argument only when the social record has
`char_found`, then branches through no-argument, not-found, self, victim
position, and actor/observer/victim audience paths.  The `massage` record at
`lib/misc/socials:480-488` has `hide=0`, no victim-position restriction, and
eight authored messages, with a self-only `#` no-argument room branch.

The existing Go social path already matched this C call path.  No behavioral
fix was warranted after the C-first comparison; R4 forbids inventing one.

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/massage-depth.txt` — no argument, first-token
  parsing, successful actor/observer/victim audiences, not-found, self, and
  sleeping-target branches.
- `pkg/session/massage_depth_test.go` — C command gate and complete authored
  social metadata/message record.
- `docs/fidelity/depth/massage.tsv` — 11 explicit rows with shared position,
  visibility, audience, and noshout delegation.

The live vehicle was green at seeds 1, 2, 3, 5, and 8, and was run with
`--show-oracle` at seed 1.  The C blocks prove the actor, peer, and sleeping
target audiences; no `src/` or C-oracle files were edited.

## Gates and merge

Local gates passed on `glm/depth-massage`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature/evidence PR #1032 (`glm/depth-massage`) had green hosted `lint`,
`security`, and `test` checks; build/deploy were skipped by the PR workflow.
It was self-merged as squash commit `d84af7cd4`.

The post-merge `main` frontier was rerun and passed with the counts above.

The next session must start on `main`, pull, rerun `make fidelity-depth`,
re-read `docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only
the unclaimed `mount` family at `src/interpreter.c:558`.
