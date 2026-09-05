# Special-procedure depth handoff — 2026-08-27

## Checkpoint

The first special-procedure depth slice is complete on `main` at
`0cbcb32ef` (PR #687, self-merged after hosted checks):

- `make fidelity-depth`: **517 total, 506 proven/delegated, 1 blocked, 10
  excluded**; exit 0.
- Actionable completion: **506/507 = 99.8%**.
- The only remaining blocked row is the intentional object-magic sleep entry
  gap; this special-procedure slice added no new blocker.

The governing fidelity rulebook and depth-testing workflow were re-read before
the round. The existing special-procedure scout remains the inventory source:
113 C `SPECIAL` definitions, 228 unique active mob assignments, and 66 final
procedure names. `spec-procs.tsv` was the owning manifest for this work.

## Selected deterministic family

The smallest reachable family was the directional `no_move_*` command
interceptor. `no_move_west` and `no_move_east` were already live-proven, so the
next C-active sibling was `no_move_south`:

- C assigns mob vnum 16308 to `no_move_south` at `src/spec_assign.c:399`.
- C's handler is `src/spec_procs2.c:2092-2110`.
- Go dispatches the same vnum through `pkg/game/spec_assign.go` and invokes
  the procedure through the player command special-procedure seam.

The defined `no_move_north` procedure was deliberately not promoted. A source
census found no active C `ASSIGNMOB` for it, so R5e treats it as outside the
reachable proof surface even though Go contains an unassigned implementation.

## RED → GREEN

The C-first vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-no-move-south.txt`, annotated as
`mob.spec-no-move-south`. It spawns the assigned guard in a controlled room,
uses a mortal actor and a peer, and probes `south` before movement can occur.

On clean `main`, RED showed Go's invented
`You try to go south but are blocked by a heavy object.` while C emitted:

- the actor-facing `A burly guard blocks your way.`;
- the room-facing `A burly guard says 'Thou shalt not pass.'`;
- the peer-facing `A burly guard blocks Specactor's way.` and the same room
  line.

The Go-only fix uses the shared `Act` path for C's `TO_NOTVICT`, `TO_VICT`,
and `TO_ROOM` calls. The vehicle is GREEN with `--show-oracle --seed 1`,
including both audience transcripts. The owning `spec-procs.tsv` row is now
`oracle-green` with proof `spec-proc-no-move-south`.

No `src/` or `darkpawns-c-oracle/` files were edited. The change follows R1,
R4, R5c, and R5e: exact player bytes, no invented behavior, class-level
assignment census, and a verified reachable call path.

## Verification

- `make fidelity-depth` — pass, counts above.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- PR #687 hosted test, lint, and security checks — pass.

## Next frontier

The program should continue one small C-active deterministic assigned
procedure at a time. After that inventory is exhausted, move to fight-driven
procedures, then percent-driven procedures with multi-seed draw evidence, and
finally heartbeat/timer procedures with explicit pulse fixtures. Do not count
an unassigned Go registry entry as reachable proof, and do not open the
fight/percent/heartbeat backlog until its fixture controls the relevant state
and draw order.
