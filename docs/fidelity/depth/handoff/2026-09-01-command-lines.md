# Depth-fidelity handoff — `lines` — 2026-09-01

## Queue position

This session started from clean `main` at the post-`lick` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md`, the standing brief, and
`docs/fidelity/depth/handoff/2026-09-01-command-lick.md`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked. The
source-order sweep selected `lines` at `src/interpreter.c:542`, after the
claimed `lick` row at line 541. The next unmanifested command is `lock` at
`src/interpreter.c:543`.

## C call path and branch inventory

The C registration is `{ "lines", POS_DEAD, do_lines, 0, 0 }` in
`src/interpreter.c:542`; `comm.c:910-947` applies the all-position and zero
level entry gates before invoking `ACMD(do_lines)` in `src/act.display.c:112-141`.
`do_lines` calls `one_argument`, reports the current `GET_SCREENSIZE` when no
token is present, then calls raw C `atoi`. Its reachable branches are:

- nonnumeric input, which becomes zero and reaches the `< 7` rejection;
- a value below 7, with the exact minimum-size rejection;
- a value above 50, with the exact upper-bound rejection;
- an accepted value from 7 through 50, which updates screen size and emits
  the confirmation;
- decimal-prefix input, because C `atoi` ignores a trailing suffix;
- trailing arguments ignored by `one_argument`; and
- an active-infobar update, where `InfoBarOn` redraws the frame before the
  confirmation.

## RED to GREEN and implementation

The unchanged main vehicle was RED in two C-verified branches:

- `lines foo`: C emitted `Screen size must be at least 7 lines.`, while Go
  emitted the Go-only `Usage: lines <number>`;
- `lines 25abc`: C emitted `Your new lines count is 25.`, while Go emitted
  `Usage: lines <number>`.

The Go-only fix in `pkg/session/display_cmds.go` adds a local
`parseLinesSize` that mirrors the C `atoi` sign, leading-decimal, suffix, and
zero-result behavior. Added `cmd/dp-oracle-diff/scenarios/lines-depth.txt`,
`lines-infobar-depth.txt`, `pkg/session/lines_depth_test.go`, and 11 rows in
`docs/fidelity/depth/lines.tsv`. Both vehicles were run with `--show-oracle`
and were green at seeds 1, 2, 3, 5, and 8. No `src/` or
`darkpawns-c-oracle/` file was edited.

The focused test pins the POS_DEAD entry gate and C-`atoi` parser semantics.
The active-infobar frame is live-proven at the `do_lines` boundary and its
shared frame implementation is delegated to `infobar.tsv` under R5b/R5c.

## Durable proof and verification

Post-slice frontier: `2478 total, 2417 proven/delegated, 17 blocked, 44
excluded`; actionable completion is `2417/2434 = 99.3%`, and `do_lines` is
`11/11`.

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #1007 (`glm/depth-lines-20260901`) was initially missing checks.
The one permitted exact retry, `gh workflow run "Dark Pawns CI/CD" --ref
glm/depth-lines-20260901`, started run `33471218842`; required `lint`,
`security`, and `test` checks finished green, with optional build/deploy
skipped by workflow policy. PR #1007 was then self-merged at main commit
`35f948e04`. No merge was performed while a required check was pending.

This note is the required dated handoff for the session. The slice follows
R1/R2/R3/R4/R5e; shared entry and infobar behavior is bounded under R5b/R5c.
