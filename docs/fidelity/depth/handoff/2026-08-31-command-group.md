# Depth handoff — 2026-08-31 — `group`

## Frontier and queue position

- Started from clean `main` at `60ecdfda2` after merging the `goose`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-goose.md`.
- The starting frontier was 1,890 total, with 1,833 proven/delegated, 16
  blocked, and 41 excluded. The `group` manifest adds 19 proven/delegated
  cases. The post-slice frontier is 1,909 total, with 1,852 proven/delegated,
  16 blocked, and 41 excluded; actionable completion is 1,852/1,868 (99.1%).
- A fresh source-order audit confirms `group` at `src/interpreter.c:471` is
  covered. `goto` is covered by `goto.tsv`, `gossip` by `channels.tsv`, and
  `gold` by `spec-procs.tsv`; the next un-manifested command-table family is
  `grab` at `src/interpreter.c:472`. The next session must return to clean
  `main`, pull, rerun the frontier check, reread this handoff, and begin
  `grab` while leaving glare PR #896 open.

## C call path and branch inventory

`src/interpreter.c:471` registers `group` as `POS_SLEEPING`, minimum level 1,
and `do_group`. The handler path is `src/act.other.c:624-740`:

- `perform_group` rejects an already grouped or unseen victim, sets
  `AFF_GROUP`, and emits the actor, victim, and non-victim room Act messages.
- `print_group` rejects a non-member, then renders the grouped head and each
  grouped follower with HP, mana, move, level, class, and the NPC redaction
  fields.
- `do_group` applies C `one_argument`; no input prints the group; a character
  with `master` cannot enroll; `all` self-enrolls and scans same-room
  followers while excluding `IS_SHADOWING`; a single target uses
  `get_char_room_vis`, requires the target to follow the actor (or be self),
  then accepts or kicks with the corresponding three audiences.
- `src/utils.h:420` defines `IS_SHADOWING` as a player with `AFF_DODGE`, and
  `src/config.c:93` defines the exact `NOPERSON` text.

The unchanged-main RED vehicle confirmed the old Go path joined all argument
words, used the wrong missing-target text, resolved only global players,
omitted room broadcasts, incorrectly self-enrolled on a single-target accept,
and rendered a shortened group listing. The corrected Go path uses
`game.OneArgument`, `ResolveCharInRoom`, typed player/mob AFF_GROUP state,
canonical `game.Act`, and deterministic follower enumeration. The existing
agent-only auto-follow behavior remains isolated after C target resolution and
was not treated as a C branch.

## Coverage proof

The RED `group-depth --seed 1 --show-oracle` run exposed the full membership
divergence before implementation. After the fix, these vehicles reported no
normalized divergence for seeds 1, 2, 3, 5, and 8:

- `group-depth`: no-argument, C one-argument parsing, accept without leader
  self-enrollment, `all`, print listing, kick/reaccept, already-grouped, and
  self-kick audiences.
- `group-gates-depth`: fill-only no-argument, missing target, target-must-
  follow, and master refusal.
- `group-room-depth`: room-scoped target lookup versus an online player in
  another room.
- `group-mob-depth`: visible NPC resolution before the follower gate.

The slice follows R1/R3/R4/R5e and R5c: C player-facing bytes and Act
audiences remain authoritative, deterministic state/order is proven across
five seeds, no new C behavior was invented, and the actual `do_group` call
path was traced before changing the port.

## Changes and gates

- Added four annotated oracle vehicles and `docs/fidelity/depth/group.tsv`
  with 19 rows.
- Added `pkg/session/group_test.go` for the C entry and position gate.
- Updated group handling to preserve C parsing, room/mob resolution,
  follower/membership gates, audiences, and formatted group output; added
  typed follower enumeration and exported AFF constants needed at the session
  boundary.
- Local gates passed: `make fidelity-depth` — 1,909 total /
  1,852 proven-or-delegated / 16 blocked / 41 excluded; `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (0 issues),
  `gofumpt -l .`, and `git diff --check`.

Implementation PR #902 (`glm/depth-group`) was self-merged only after hosted
`lint`, `security`, and `test` checks were green. Its `build-and-push` and
`deploy` jobs were skipped by policy. Merge commit: `935d5952c`. The glare
implementation PR #896 remains open because its hosted test failure is the
unrelated pre-existing retry-based `pkg/spells/TestMagAffects_Sleep`; it was
not retried or fixed forward.
