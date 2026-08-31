# Depth handoff — 2026-08-31 — `groan`

## Frontier and queue position

- Started from clean `main` at `49306d2a7` after the corrected `grimace`
  handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-grimace.md`.
- The starting frontier was 1,960 total, with 1,903 proven/delegated, 16
  blocked, and 41 excluded. The `groan` manifest adds 8 proven/delegated
  cases. The post-slice frontier is 1,968 total, with 1,911
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,911/1,927 (99.2%).
- A fresh source-order audit confirms `groan` at `src/interpreter.c:477` is
  covered. `groinrip` at line 478 is not yet present as its own depth
  manifest and is the next un-manifested command-table family. The existing
  `bite` vehicle's peaceful groinrip witness is incidental evidence only and
  does not claim the family. The next session must return to clean `main`,
  pull, rerun the frontier check, reread this handoff, and begin `groinrip`.

## C call path and branch inventory

`src/interpreter.c:477` registers `groan` as `POS_RESTING`, minimum level 0,
and `do_action`. The C social record at `lib/misc/socials:340-343` is
`groan 0 0`, with no hide flag, no minimum victim position, and no
`char_found` message:

- `do_action` first rejects `PLR_NOSHOUT`, then sees that `char_found` is
  absent and sets its argument buffer empty. It therefore never performs
  target lookup, self handling, minimum-victim-position checks, or not-found
  handling, even when the player types a visible, missing, or self-looking
  name.
- The empty argument path emits the authored `You groan loudly.` actor line
  and `$n groans loudly.` room line. The third record slot is `#`, so there
  are no target-bearing messages.

The registered C row and the Go generic `DoAction` path were compared against
`pkg/session/commands.go`, `pkg/game/act_social.go`, the shared social parser,
and canonical Act delivery. No Go behavior change was confirmed or needed.

## Coverage proof

The C-first `groan-depth --seed 1 --show-oracle` run was GREEN and showed the
same authored actor/room pair for no argument, a visible mob argument, a
missing argument, and a self-looking argument. The vehicle reported no
normalized divergence for seeds `1,2,3,5,8`.

The fixture uses an awake `Groanobserver` player to witness room delivery and
stays within the telnet three-connection same-IP limit. It never edits `src/`
or the C oracle tree.

The slice follows R1/R2/R3/R4/R5e and R5c: C social bytes and the command
surface remain authoritative, deterministic room behavior is proven across
five seeds, no unreachable target branch was invented, the actual
`do_action` path was traced before recording coverage, and shared social
gates/visibility were delegated rather than duplicated.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/groan-depth.txt` with no-argument and
  ignored-argument probes plus an awake room observer.
- Added `docs/fidelity/depth/groan.tsv` with 8 explicit rows, including
  shared delegations for position, noshout, and visibility.
- Added `pkg/session/groan_test.go` to pin the C command and self-only social
  metadata/messages.
- No implementation change was made: the existing Go path matched the C
  oracle on the complete vehicle.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.
- PR #913 (`glm/depth-groan`) passed hosted `test`, `lint`, and `security`
  checks and was merged. No workflow retry was required.
