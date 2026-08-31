# Depth handoff — 2026-08-31 — `greet`

## Frontier and queue position

- Started from clean `main` at `e89aa4c35` after merging the `grab` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus
  `2026-08-31-command-grab.md`.
- The starting frontier was 1,930 total, with 1,873 proven/delegated, 16
  blocked, and 41 excluded. The dedicated `greet` manifest adds 11
  proven/delegated cases. The post-slice frontier remains 1,941 total, with
  1,884 proven/delegated, 16 blocked, and 41 excluded; actionable completion
  is 1,884/1,900 (99.2%).
- PR #907 initially merged the direct greet vehicle and metadata test. A
  post-merge evidence audit found that its only non-victim peer was asleep,
  so the room-audience witness was not actually visible. This handoff branch
  corrects the disposable vehicle to use an awake observer, an awake player
  target, and a sleeping target; it changes no game implementation behavior.
- A fresh source-order audit confirms `greet` at `src/interpreter.c:474` is
  covered. `grats` at line 473 is covered by `channels.tsv`; the next
  un-manifested command-table family is `grin` at
  `src/interpreter.c:475`. The next session must return to clean `main`,
  pull, rerun the frontier check, reread this handoff, and begin `grin`.

## C call path and branch inventory

`src/interpreter.c:474` registers `greet` as `POS_RESTING`, minimum level 0,
and `do_action`. The C social record at `lib/misc/socials:315-323` is
`greet 0 0`, so it has no hide flag, no minimum victim position, and all eight
social message positions populated except the ordinary `others_no_arg` slot,
which is `#`:

- `do_action` first rejects `PLR_NOSHOUT`, then applies `one_argument` because
  `char_found` exists. Empty input emits `Greet Who?` and the authored
  no-argument room branch.
- `get_char_room_vis` resolves a visible mob or player in the actor's room.
  A miss emits `Please -- try someone who is here?`.
- A self target selects `So, you've finally discovered yourself!` and the
  authored `$mself` room line.
- A different target passes directly because the record's minimum victim
  position is zero, then emits actor, non-victim room, and victim lines. A
  sleeping target still passes the C position test, but `act` suppresses its
  `TO_VICT` delivery while preserving eligible room output.

The registered C row and the Go generic `DoAction` path were compared against
`pkg/session/commands.go`, `pkg/game/act_social.go`, the typed mob/player
room lookup, the shared one-argument parser, and canonical Act delivery. No
Go behavior change was confirmed or needed.

## Coverage proof

The corrected C-first `greet-depth --seed 1 --show-oracle` run was GREEN and
showed the intended C blocks for no argument, a fill-word/trailing-input mob
target with an awake room observer, an awake player victim, self target,
missing target, and sleeping target. The corrected vehicle reported no
normalized divergence for seeds `1,2,3,5,8`.

The fixture uses registered mob 18305 (`cleaner`) for the non-victim room
audience, an awake `Greetobserver` player for the victim audience, and a
sleeping `Greetsleeper` player for C's sleeping-delivery boundary. It stays
within the telnet three-connection same-IP limit and never edits `src/` or
the C oracle tree.

The slice follows R1/R2/R3/R4/R5e and R5c: C social bytes and the command
surface remain authoritative, deterministic audience behavior is proven
across five seeds, no branch was invented, the actual `do_action` path was
traced before recording coverage, and shared social gates/visibility were
delegated rather than duplicated.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/greet-depth.txt` with the corrected
  actor/awake-observer/sleeping-target topology.
- Added `docs/fidelity/depth/greet.tsv` with 11 explicit rows and corrected
  the target-audience note to identify the awake observer's mob-target
  witness.
- Added `pkg/session/greet_test.go` to pin the C command and social metadata.
- No implementation change was made: the existing Go path matched the C
  oracle on the corrected vehicle.
- Local gates passed: `make fidelity-depth` — 1,941 total /
  1,884 proven-or-delegated / 16 blocked / 41 excluded; `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (0 issues),
  `gofumpt -l .`, and `git diff --check`.

PR #907 (`glm/depth-greet`) was self-merged only after hosted `lint`,
`security`, and `test` checks were green following the one permitted
workflow retry. Its `build-and-push` and `deploy` jobs were skipped by policy;
merge commit `bd7902ac8` is on `main`. This handoff carries only the
post-merge evidence correction and the durable queue record.
