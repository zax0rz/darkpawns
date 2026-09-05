# Depth-fidelity handoff — `last` — 2026-08-31

## Queue position

This session started from clean `main` at the post-`lambada` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md` plus
`docs/fidelity/depth/handoff/2026-08-31-command-lambada.md`. The next
unproven source-table command was `last` at `src/interpreter.c:535`; the next
command after this slice is `leave` at `src/interpreter.c:536`.

## C call path and branch inventory

The C registration is `{ "last", POS_DEAD, do_last, LVL_GOD-1, 0 }` in
`src/interpreter.c:535`. `comm.c:910-947` resolves the command, applies the
level/position/frozen gates, and invokes `ACMD(do_last)` in
`src/act.wizard.c:1834-1854`. The handler calls
`one_argument()` (`src/interpreter.c:1267-1286`), which skips fill words,
lowercases the first token, and ignores the remainder. It then calls
`load_char()` (`src/db.c:2342-2356`) and emits one of:

1. no target: `For whom do you wish to search?\r\n`;
2. missing pfile: `There is no such player.\r\n`;
3. found pfile: fixed-width idnum, level, two-character class abbreviation,
   twelve-column name, eighteen-column host, and `ctime(last_logon)`:
   `[%5ld] [%2d %s] %-12s : %-18s : %-20s\r\n`.

The found branch is materially data-dependent: C reads durable `idnum`,
`class`, `host`, and `last_logon` from `char_file_u`, while the disposable Go
oracle vehicle starts with its database deliberately unreachable and the Go
player/save model has no equivalent durable host or last-login fields.

## Proof and implementation

Added `cmd/dp-oracle-diff/scenarios/last-depth.txt` with a God actor and
persisted C peer. The vehicle covers no argument, found target, fill-word
first-token parsing, ignored trailing input, and missing target. The seed-1
`--show-oracle` run reached every intended block.

The unchanged Go baseline was RED: found and missing lookups returned
`No database available.`. A second honest attempt used the live peer as a
temporary source and formatted the host/class fields, but remained RED on the
id and login timestamp (C peer pfile id `2`/`23:16:43`; the approximation used
descriptor id `1`/`23:16:48`). That approximation was removed under R4: it
would invent a pfile identity and `last_logon` value. The confirmed changes in
`pkg/session/wiz_system.go` are therefore limited to:

- C's `LVL_GOD-1` command-level gate;
- C's `one_argument` target selection;
- C's missing-player bytes when the Go persistence backend is unavailable.

`pkg/session/last_depth_test.go` pins the command registry gate and the
missing-player/one-argument early return. `docs/fidelity/depth/last.tsv` has
five proven rows and one blocked found-player row. The blocked row is retained
as a real proof gap after two attempts, with the persistence seam explicitly
identified; do not change the Go save format or invent host/id/time bytes.

## Verification and review

Local gates passed on the feature branch:

- `make fidelity-depth`: `2423 total, 2362 proven/delegated, 17 blocked, 44 excluded`;
  `do_last: 5/6`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #995 (`glm/depth-last-20260831`) merged with hosted `test`,
`security`, and `lint` green in run `33465872810` at main commit
`0a9ef064c`. Optional build/deploy jobs were skipped by workflow policy.

This note is the required dated handoff for the session. Continue from clean
`main`, recheck the frontier and newest handoff, and take `leave` next. All
claims follow R1/R2/R4/R5e; the blocked persistence seam is retained under
R5b/R5c.
