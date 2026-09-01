# Depth-fidelity handoff — `leave` — 2026-08-31

## Queue position

This session started from clean `main` at the post-`last` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md` plus
`docs/fidelity/depth/handoff/2026-08-31-command-last.md`. The next
unproven source-table command was `leave` at `src/interpreter.c:536`; the
next command after this slice is `leer` at `src/interpreter.c:537`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked.

## C call path and branch inventory

The C registration is `{ "leave", POS_STANDING, do_leave, 0, 0 }` in
`src/interpreter.c:536`. `comm.c:910-947` applies the command position and
level gates before invoking `ACMD(do_leave)` in
`src/act.movement.c:676-694`. The handler ignores command arguments. It
first emits `You are outside.. where do you want to go?\r\n` unless the
current room is indoors, then scans exits in direction order for the first
valid open exit to a non-indoors destination and delegates to
`perform_move(ch, door, 1)`. If no such exit exists it emits
`I see no obvious exits to the outside.\r\n`.

## Proof and implementation

The four durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/leave-outside-depth.txt` — outdoor refusal
  and ignored trailing arguments;
- `cmd/dp-oracle-diff/scenarios/leave-no-exit-depth.txt` — indoor room with
  no qualifying outside exit;
- `cmd/dp-oracle-diff/scenarios/leave-success-depth.txt` — first qualifying
  indoor-to-outdoor exit, actor observation, origin audience, and destination
  audience;
- `cmd/dp-oracle-diff/scenarios/leave-position-depth.txt` — sleeping actor
  rejected by the registered entry gate.

All intended branches were green against the C oracle at seed 1, including
the success vehicle with origin/destination peers and the position vehicle's
`In your dreams, or what?` gate response. No Go behavior change was needed:
`cmdLeave` already delegates to `World.DoLeave`, whose selector and canonical
`perform_move` path match C. Shared movement audience and movement-state
branches remain delegated to `docs/fidelity/depth/movement.tsv`.

## Durable proof and verification

`docs/fidelity/depth/leave.tsv` contains eight proven rows: the entry gate,
outside refusal, ignored arguments, no qualifying exit, success actor,
origin audience, destination audience, and position gate. The focused test
`pkg/session/leave_depth_test.go` pins the C registration gate and registry
parity. The scenario annotations are declared in their respective vehicles.

The feature branch passed all local gates:

- `make fidelity-depth`: `2431 total, 2370 proven/delegated, 17 blocked, 44 excluded`; `do_leave: 8/8`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #997 (`glm/depth-leave-20260831`) merged with hosted `test`,
`security`, and `lint` green in run `33466742784` at main commit
`e44b73ba0`; optional build/deploy jobs were skipped by workflow policy. No
merge was performed while a required check was pending. No `src/` or
`darkpawns-c-oracle/` file was edited.

This note is the required dated handoff for the session. Continue from clean
`main`, recheck the frontier and newest handoff, and take `leer` next. The
slice follows R1/R2/R4/R5e; canonical movement delegation is bounded under
R5b/R5c.
