# Depth-fidelity handoff — `levels` — 2026-09-01

## Queue position

This session started from clean `main` at the post-`leer` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md` plus
`docs/fidelity/depth/handoff/2026-08-31-command-leer.md`. The next
unproven source-table command was `levels` at `src/interpreter.c:538`; the
next command after this slice is `list` at `src/interpreter.c:539`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked.

## C call path and branch inventory

The C registration is `{ "levels", POS_DEAD, do_levels, 0, 0 }` in
`src/interpreter.c:538`. `comm.c:910-947` applies the zero level and
`POS_DEAD` gates before invoking `ACMD(do_levels)` in
`src/act.informative.c:2311-2328`. The handler does not parse arguments.
It has two branches:

1. `IS_NPC(ch)`: emit only `You ain't nothin' but a hound-dog.\r\n`;
2. a player character: append rows 1 through `LVL_IMMORT-1` using
   `find_exp(GET_CLASS(ch), i-1)` and `find_exp(GET_CLASS(ch), i)`, then pass
   the complete 30-row buffer to `page_string`.

The fixed XP values and class-specific formula are the C
`src/class.c:1089-1166` path. Pager navigation and page boundaries are a
shared `modify.c` behavior, so their complete matrix remains delegated to
the existing pager tests.

## RED diagnosis and confirmed fix

`cmd/dp-oracle-diff/scenarios/levels-depth.txt` proves the non-NPC table and
pager page 1/page 2. `levels-arguments-depth.txt` proves C's ignored
argument behavior through both pages. `levels-npc-depth.txt` spawns the
disposable cleaner mob, switches the first-player Implementor into it, and
reaches the `IS_NPC(ch)` branch.

The player and argument vehicles were green at seed 1 with `--show-oracle`;
the NPC vehicle was RED on main: C emitted the hound-dog line while Go
rendered the original wizard's XP table. The confirmed Go fix in
`pkg/session/cmd_info.go` checks the active switched-mob state and emits
that exact C early-return line before any player table or pager setup.
`pkg/session/levels_depth_test.go` pins the `POS_DEAD` entry gate and the
switched-mob early return. No `src/` or `darkpawns-c-oracle/` file was edited.

## Durable proof and verification

`docs/fidelity/depth/levels.tsv` contains six rows: entry gate, non-NPC
output, ignored arguments, NPC branch, delegated `FindExp` arithmetic, and
delegated pager behavior. The post-slice frontier is `2448 total, 2387
proven/delegated, 17 blocked, 44 excluded`; actionable completion is
`2387/2404 = 99.3%`, and `do_levels` is `6/6`.

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #1001 (`glm/depth-levels-20260901`) merged with hosted `test`,
`security`, and `lint` green in run `33468515615` at main commit
`736ff1651`; optional build/deploy jobs were skipped by workflow policy. The
checks required the one permitted workflow retry because no checks initially
appeared. No merge was performed while a required check was pending.

This note is the required dated handoff for the session. Continue from clean
`main`, recheck the frontier and newest handoff, and take `list` next. The
slice follows R1/R2/R3/R4/R5e; shared XP and pager behavior is bounded under
R5b/R5c.
