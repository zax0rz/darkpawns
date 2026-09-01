# Depth-fidelity handoff — `mount`

Date: 2026-09-01
Queue: un-manifested interpreter command families, source-table order
Rules: R1, R2, R3, R4, R5b, R5c, R5e

## Frontier

Started from fresh `main` after the `massage` handoff.  `make fidelity-depth`
reported 2,583 cases: 2,515 proven/delegated, 22 blocked, and 46 excluded
(99.1% actionable).  After the `mount` evidence was merged, the frontier is
2,602 total: 2,532 proven/delegated, 22 blocked, and 48 excluded (99.1%
actionable).

The exact `src/interpreter.c` table sweep against the command fields in
`docs/fidelity/depth/*.tsv` treats the existing social table and generic
`do_not_here` family manifests as family claims.  The next actually unclaimed
family is `mute` at `src/interpreter.c:561` (`POS_DEAD`, level 1,
`do_wizutil`/`SCMD_SQUELCH`).  `muhaha` and `mumble` are later; `mount` and
`ride` are now claimed by `mount.tsv`.

## C path and RED → GREEN proof

The source row is `{ "mount", POS_STANDING, do_ride, 0, 0 }` at
`src/interpreter.c:558`; the later `ride` row at line 659 aliases the same
handler.  `src/act.other.c:1545-1593` checks fighting, `ROOM_INDOORS`, C
`one_argument`, visible target/not-found/self, actor and mount mounted state,
`IS_MOUNTABLE`, the player-only `CAN_MOUNT` branch, actor/mount charm, then
sets both mount affects and follower state before the actor, victim, and room
audiences.  `src/utils.h:362-366` defines the mount predicates.  The Lua
`mount(ride)` entry in `src/scripts.c:876-895` is a non-command API and is
explicitly excluded from this registered descriptor-command family (R2/R4/R5e).

On clean `main`, `mount-depth --show-oracle --seed 1` was RED: C returned
`That's disgusting!` for the self target and `You can't ride Mountpeer!` for
the player target, while Go returned NOPERSON for both.  The same main vehicle
also exposed Go's `strings.TrimSpace` boundary rather than C's first-token
`one_argument` behavior.  The confirmed Go fix uses `OneArgument` and handles
player self/other targets without touching `src/` (R1/R2/R4/R5e).

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/mount-depth.txt` — no argument, first-token
  parsing, not-found, self/player targets, success audiences, actor-mounted
  refusal, dismount, and the `ride` alias.
- `cmd/dp-oracle-diff/scenarios/mount-ridden-depth.txt` — isolated
  already-ridden mount gate using the `force` vehicle.
- `cmd/dp-oracle-diff/scenarios/mount-indoors-depth.txt` — indoor early-return
  bytes before target lookup.
- `pkg/game/mount_depth_test.go` — direct gate ordering, charm and
  non-mountable refusals, success state, audiences, and one-argument behavior.
- `pkg/session/mount_depth_test.go` — authoritative `mount`/`ride` command
  gates and alias registration.
- `docs/fidelity/depth/mount.tsv` — 20 rows covering reachable D1-D5 cases,
  shared state/audience ownership, and unreachable command-surface branches.

All three vehicles were GREEN at seeds 1, 2, 3, 5, and 8; each was run with
`--show-oracle` at seed 1.  The C blocks reached the intended mount branches,
including the indoor and already-ridden gates.  No `src/` or C-oracle files
were edited (R1-R5e).

## Gates and merge

Local gates passed on `glm/depth-mount`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature/evidence PR #1034 (`glm/depth-mount`) had green hosted `lint`,
`security`, and `test` checks; build/deploy were skipped by the PR workflow.
It was self-merged under the 2026-08-27 amendment as squash commit
`1be31c0e3`.

The post-merge `main` checkout/pull and frontier rerun passed with the counts
above.  The next session must start on `main`, pull, rerun
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this newest
handoff, then map and attempt only `mute` at `src/interpreter.c:561`.
