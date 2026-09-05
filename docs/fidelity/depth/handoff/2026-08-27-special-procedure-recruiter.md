# Special-procedure depth handoff — 2026-08-27 recruiter slice

## Checkpoint

The recruiter deterministic slice is complete on `main` at `b75b643ea`
(PR #688, self-merged after hosted checks):

- `make fidelity-depth`: **519 total, 508 proven/delegated, 1 blocked, 10
  excluded**; exit 0.
- Actionable completion: **508/509 = 99.8%**.
- The only blocked row remains the intentional object-magic sleep entry gap.

## Vehicle and call path

The C-active assignment is mob vnum 16300 → `recruiter` at
`src/spec_assign.c:398`. The procedure is
`src/spec_procs3.c:905-930`; it intercepts `hit`/`kill` and `cast`/`will`,
constructs a message whose first token is the player name, and calls C
`do_tell`. That target parser removes the first token, leaving a direct
`perform_tell` message to the actor (`src/act.comm.c:905-930`).

The Go vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-recruiter.txt`, with a primary actor,
a peer, and all four command arms. On clean `main`, RED showed four confirmed
classes of drift: the player name leaked into the tell body, C's double space
was collapsed, the mob label was lowercased, and Go broadcast the tell to the
peer. The Go-only fix uses the canonical `Act` direct-victim path with
`ToSleep`, preserving C's exact bytes and private audience.

`--show-oracle --seed 1` is GREEN for `hit recruiter`, `kill recruiter`,
`cast fireball recruiter`, and `will fireball recruiter`; the peer receives no
message. The owning `spec-procs.tsv` manifest now contains the two grouped D3
cases `mob.spec-recruiter-hit-kill` and `mob.spec-recruiter-cast-will`.

No `src/` or `darkpawns-c-oracle/` files were edited. The work follows R1, R4,
R5c, and R5e: exact player bytes, no invented behavior, class-aware source
inspection, and a verified reachable C assignment/call path.

## Verification

- `make fidelity-depth` — pass, counts above.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- PR #688 hosted test, lint, and security checks — pass.

## Next frontier

Continue the deterministic assigned-procedure inventory one focused vehicle at
a time. Candidates should be selected from active C `ASSIGNMOB` entries, not
from Go-only or unassigned registry names. Once the cheap deterministic arms
are exhausted, begin fight-driven procedures with explicit combat state, then
percent-driven procedures with multi-seed draw evidence, and only then
heartbeat/timer procedures with controlled pulse fixtures.
