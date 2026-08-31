# Depth handoff — 2026-08-31 — `grimace`

## Frontier and queue position

- Started from clean `main` at `b39d8521f` after merging the `grin` evidence,
  pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-grin.md`.
- The starting frontier was 1,952 total, with 1,895 proven/delegated, 16
  blocked, and 41 excluded. The `grimace` manifest adds 8 proven/delegated
  cases. The post-slice frontier is 1,960 total, with 1,903
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,903/1,919 (99.2%).
- A fresh source-order audit confirms `grimace` at
  `src/interpreter.c:476` is covered. `groan` at line 477 is not present in
  any depth manifest and is the next un-manifested command-table family. The
  next session must return to clean `main`, pull, rerun the frontier check,
  reread this handoff, and begin `groan`.

## C call path and branch inventory

`src/interpreter.c:476` registers `grimace` as `POS_RESTING`, minimum level 0,
and `do_action`. The C social record at `lib/misc/socials:325-328` is
`grimace 0 0`, with no hide flag, no minimum victim position, and no
`char_found` message:

- `do_action` first rejects `PLR_NOSHOUT`, then sees that `char_found` is
  absent and sets its argument buffer empty. It therefore never performs
  target lookup, self handling, minimum-victim-position checks, or
  not-found handling, even when the player types a visible, missing, or
  self-looking name.
- The empty argument path emits `You grimace.` to the actor and the authored
  `$n grimaces.` room line. The third record slot is `#`, so there are no
  target-bearing messages.

The registered C row and the Go generic `DoAction` path were compared against
`pkg/session/commands.go`, `pkg/game/act_social.go`, the shared social parser,
and canonical Act delivery. No Go behavior change was confirmed or needed.

## Coverage proof

The C-first `grimace-depth --seed 1 --show-oracle` run was GREEN and showed
the intended C blocks for no argument, a visible player argument, a missing
argument, and a self-looking argument. Every probe emitted the exact same
actor/room pair. The vehicle reported no normalized divergence for seeds
`1,2,3,5,8`.

The fixture uses an awake `Grimaceobserver` player to witness room delivery,
and stays within the telnet three-connection same-IP limit. It never edits
`src/` or the C oracle tree.

The slice follows R1/R2/R3/R4/R5e and R5c: C social bytes and the command
surface remain authoritative, deterministic room behavior is proven across
five seeds, no unreachable target branch was invented, the actual
`do_action` path was traced before recording coverage, and shared social
gates/visibility were delegated rather than duplicated.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/grimace-depth.txt` with no-argument and
  ignored-argument probes plus an awake room observer.
- Added `docs/fidelity/depth/grimace.tsv` with 8 explicit rows, including
  shared delegations for position, noshout, and visibility.
- Added `pkg/session/grimace_test.go` to pin the C command and self-only social
  metadata/messages.
- No implementation change was made: the existing Go path matched the C
  oracle on the complete vehicle.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.
- PR #911 (`glm/depth-grimace`) passed hosted lint, security, and test checks
  after the mandated one-time workflow retry and was merged.
