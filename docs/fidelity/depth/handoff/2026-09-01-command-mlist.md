# Depth-fidelity handoff — `mlist`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R4, R5e

## Frontier

Started from fresh `main` after the `medit` handoff.  `make fidelity-depth`
reported 2,560 cases: 2,492 proven/delegated, 22 blocked, and 46 excluded
(99.1% actionable).  After the `mlist` feature and evidence were merged, the
frontier is 2,568 total: 2,500 proven/delegated, 22 blocked, and 46 excluded
(99.1% actionable).

The next un-manifested command family is `motd`, the registered C row at
`src/interpreter.c:555` (`POS_DEAD`, unrestricted level, `do_gen_ps`,
`SCMD_MOTD`).  No existing depth manifest row claims `motd`.

## C path and proof

The queue item was the C row `{ "mlist", POS_DEAD, do_mlist, LVL_BUILDER, 0 }`
at `src/interpreter.c:554`.  The handler in `src/act.wizard.c:3376-3404`
consumes one C `one_argument` token, parses it with `atoi`, maps the selected
zone to `[j*100, j*100+99]`, scans `mob_index` in load order, and calls
`page_string` with either the exact empty-zone response or the numbered rows
and count footer.

The first clean-main oracle vehicle exposed three confirmed divergences in Go:
the old implementation treated the input as a keyword, invented usage/no-match
messages, truncated at 50 matches, omitted C paging, and did not use the C
zone range.  The C source also has overlapping `sprintf(buf, "%s...", buf,
...)` calls while appending rows.  The compiled C oracle used by the harness
retains only the final count footer for populated zones; R1/R5e therefore
requires reproducing those observed bytes rather than correcting the C bug.

The corrected Go path now consumes only the first token, matches C signed
decimal-prefix `atoi` behavior (including empty and nonnumeric input), counts
mobiles in the selected VNUM window in parsed order, emits the exact C
empty-zone/footer bytes, and routes the result through `PageString`.  No
`src/` or C-oracle files were edited.

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/mlist-depth.txt` — empty, populated, trailing
  token, and decimal-prefix cases.
- `cmd/dp-oracle-diff/scenarios/mlist-noarg-depth.txt` — bare input to Zone 0.
- `cmd/dp-oracle-diff/scenarios/mlist-nonnumeric-depth.txt` — nonnumeric input
  to Zone 0.
- `pkg/session/mlist_depth_test.go` — C entry gate and decimal-prefix parser.
- `docs/fidelity/depth/mlist.tsv` — eight explicit rows, including the shared
  pager delegation.

All three oracle vehicles were green at seeds 1, 2, 3, 5, and 8; each was run
with `--show-oracle` at seed 1.  The feature was PR #1028 on
`glm/depth-mlist`, merged only after hosted `lint`, `security`, and `test`
checks were green, as squash commit `c5730738c`.

## Gates

The feature branch passed:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

The post-merge `main` frontier was rerun and passed with the counts above.

The next session must start on `main`, pull, rerun `make fidelity-depth`,
re-read `docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only
the unclaimed `motd` family at `src/interpreter.c:555`.
