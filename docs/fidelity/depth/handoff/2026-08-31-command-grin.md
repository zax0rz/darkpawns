# Depth handoff — 2026-08-31 — `grin`

## Frontier and queue position

- Started from clean `main` at `ad7555d23` after merging the corrected
  `greet` handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the latest dated queue-correction
  handoff.
- The starting frontier was 1,941 total, with 1,884 proven/delegated, 16
  blocked, and 41 excluded. The `grin` manifest adds 11 proven/delegated
  cases. The post-slice frontier is 1,952 total, with 1,895
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,895/1,911 (99.2%).
- A fresh source-order audit confirms `grin` at `src/interpreter.c:475` is
  covered. `grimace` at line 476 is not present in any depth manifest and is
  the next un-manifested command-table family. The next session must return
  to clean `main`, pull, rerun the frontier check, reread this handoff, and
  begin `grimace`.

## C call path and branch inventory

`src/interpreter.c:475` registers `grin` as `POS_RESTING`, minimum level 0,
and `do_action`. The C social record at `lib/misc/socials:330-338` is
`grin 0 0`, so it has no hide flag, no minimum victim position, and all eight
social message positions populated:

- `do_action` first rejects `PLR_NOSHOUT`, then applies `one_argument`
  because `char_found` exists. Empty input emits `You grin evilly.` and the
  authored room branch.
- `get_char_room_vis` resolves a visible mob or player in the actor's room.
  A miss emits `Grin at whom?`.
- A self target selects `You sit back with a smug grin.` and the authored
  `$mself` room line.
- A different target passes directly because the record's minimum victim
  position is zero, then emits actor, non-victim room, and victim lines. A
  sleeping target still passes the C position test, but `act` suppresses its
  `TO_VICT` delivery while preserving eligible room output.

The registered C row and the Go generic `DoAction` path were compared against
`pkg/session/commands.go`, `pkg/game/act_social.go`, the typed mob/player room
lookup, the shared one-argument parser, and canonical Act delivery. No Go
behavior change was confirmed or needed.

## Coverage proof

The C-first `grin-depth --seed 1 --show-oracle` run was GREEN and showed the
intended C blocks for no argument, a fill-word/trailing-input cleaner-mob
target with an awake room observer, an awake player victim, self target,
missing target, and sleeping target. The vehicle reported no normalized
divergence for seeds `1,2,3,5,8`.

The fixture uses registered mob 18305 (`cleaner`) for the non-victim room
audience, an awake `Grinobserver` player for the victim audience, and a
sleeping `Grinsleeper` player for C's sleeping-delivery boundary. It stays
within the telnet three-connection same-IP limit and never edits `src/` or
the C oracle tree.

The slice follows R1/R2/R3/R4/R5e and R5c: C social bytes and the command
surface remain authoritative, deterministic audience behavior is proven
across five seeds, no branch was invented, the actual `do_action` path was
traced before recording coverage, and shared social gates/visibility were
delegated rather than duplicated.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/grin-depth.txt` with the actor,
  observer, sleeping-target, and registered-cleaner topology.
- Added `docs/fidelity/depth/grin.tsv` with 11 explicit rows, including
  shared delegations for position, noshout, and visibility.
- Added `pkg/session/grin_test.go` to pin the C command and social metadata.
- No implementation change was made: the existing Go path matched the C
  oracle on the complete vehicle.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and
  `git diff --check`.
- PR #909 (`glm/depth-grin`) passed hosted lint, security, and test checks and
  was merged. No workflow retry was required.
