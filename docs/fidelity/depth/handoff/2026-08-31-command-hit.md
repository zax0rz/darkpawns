# Depth handoff — 2026-08-31 — `hit`

## Frontier and queue position

- Started from clean `main` at `4a4c9374e` after the merged `highfive`
  handoff, pulled `main`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus `2026-08-31-command-highfive.md`.
- The starting frontier was 2,157 total, with 2,097 proven/delegated, 16
  blocked, and 44 excluded. The hit manifest adds 6 cases, producing 2,163
  total, 2,103 proven/delegated, 16 blocked, and 44 excluded; actionable
  completion remains 2,103/2,119 = 99.2%.
- `hit` is registered at `src/interpreter.c:495`; `murder` at `:562` enters
  the same `do_hit` handler with a different subcommand that the handler does
  not inspect. The exact interpreter-table sweep leaves `hold` at
  `src/interpreter.c:497` as the next unmanifested command family; the next
  session must return to clean `main`, pull, rerun the frontier check, reread
  this handoff, and begin `hold`.

## C call path and branch inventory

The C source was traced before changing Go:

- `do_hit` calls `one_argument(argument, arg)`, which skips the C fill words
  (`in`, `from`, `with`, `the`, `on`, `at`, `to`), lowercases the selected
  token, and ignores the remainder before `get_char_room_vis`.
- Empty post-fill input emits `Hit who?`; lookup failure emits `They don't
  seem to be here.`; self-target emits the actor/room pair; a charmed
  character's master gets the friendship refusal; a non-standing actor or
  already-current opponent gets `You do the best you can!`.
- The standing, new-target branch dismounts when mounted, calls synchronous
  `hit(ch, vict, TYPE_UNDEFINED)`, then applies `WAIT_STATE` after the call.
  The downstream `damage()` gates, combat enrollment, RNG, audiences, state,
  and death behavior are already owned by `combat-entry` and `combat-swing`
  and remain delegated under R5b/R5c.

## Coverage and confirmed fix

- Added `hit-depth.txt` for `hit the trainee`, proving C fill-word removal,
  and `hit-trailing-depth.txt` for `hit trainee extra words`, proving the
  first-token boundary. Both are GREEN for seeds 1, 2, 3, 5, and 8, and
  `--show-oracle` was used.
- RED on clean `main`: both commands reached C's synchronous miss transcript,
  while Go resolved the full argument and answered `They don't seem to be
  here.`. A serial rerun separated the parallel seed-8 local port collision
  from the fidelity result; seed 8 is GREEN.
- Fixed only `pkg/session/cmdHit` to call the shared C-faithful
  `game.OneArgument` before `ResolveCharInRoom`. Added a focused regression
  test for fill-word/trailing parsing and the fill-only `Hit who?` branch.
- Added `docs/fidelity/depth/hit.tsv`: 2 multiseed oracle cases, 1 unit case,
  the shared murder alias delegation, and explicit delegation of the already
  proven combat-entry and melee matrices.

This follows R1/R2/R3/R4/R5e: player-facing bytes and the registered alias
surface are preserved, seed variation was checked, no C behavior was invented,
and the actual `do_hit` call path was verified. Shared behavior is delegated
under R5b/R5c.

## Changes, gates, and integration

- PR #941 (`glm/depth-hit`, commit `c38380097`) passed hosted `test`, `lint`,
  and `security`; release-only build/deploy jobs were skipped as expected. It
  was merged only after every reported check was green; merged `main` is
  `ca8ca53f4`.
- Local gates passed on the feature branch: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`.

