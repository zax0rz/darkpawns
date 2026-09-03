# Depth handoff — skillset

Date: 2026-09-02  
Branch: `glm/depth-skillset`  
Feature PR: #1241 (merged green)  
Feature commit: `ceea2d288`  
Main merge: `b2981e100`

## Queue position

The special-procedure inventory remains exhausted. The single blocked row,
`objmagic.sleep-entry-gates`, was attempted once through the cast-sleep
outlaw/reagent vehicle and remains blocked for the unreachable entry gates.
After the required refresh and source-order audit, the `skillset` row at
`src/interpreter.c:704` was the next genuinely unmanifested command family;
the earlier `sleep` command is already covered by `position.tsv` and shared
sleep object-magic coverage.

Pre-slice frontier: 3,682 total, 3,581 proven/delegated, 48 blocked, 53
excluded.

The skillset slice adds 18 manifest cases. Post-slice frontier: 3,700 total,
3,599 proven/delegated, 48 blocked, 53 excluded; actionable completion is
3,599/3,647 = 98.7%.

The next session must checkout and refresh `main`, rerun `make fidelity-depth`,
reread `DEPTH_TESTING.md` and the newest handoff, then continue after
`skillset` in interpreter-table order. The next source-order claim is
`do_sleeper` for `sleeper` at `src/interpreter.c:706`; confirm it against the
manifests before working it. Do not repick the already-covered `sleep` family.

## C call path and observable contract

The command table registers:

```c
{ "skillset" , POS_SLEEPING, do_skillset , LVL_GRGOD, 0 },
```

at `src/interpreter.c:704`. `do_skillset` in `src/modify.c:255-341` first
handles the no-argument syntax/list path, then resolves `get_char_vis` before
parsing the quoted skill and numeric value. The target lookup checks visible
in-room players/mobs with keyword abbreviations and self/me/ordinal forms,
then exact global character names. Its parser preserves C's mixed `\n\r`
terminators, lowercases quoted skill text for `find_skill_num`, consumes only
the next `one_argument` token, and uses C `atoi` prefix semantics. Numeric
range checks precede the NPC refusal; a valid player receives the mutation and
exact confirmation.

The initial RED on `main` found two confirmed divergences: telnet framing
treated C's `\n\r` as two line breaks in the no-argument skill list, and
`1.orc` used Go's global substring mob lookup instead of C's visible ordinal
character resolver. Go now preserves LFCR as one transport break, routes
skillset through `ResolveCharWorld`, and uses the shared C-compatible `cAtoi`
for leading digits and first-token behavior.

## Proof artifacts

Scenario: `cmd/dp-oracle-diff/scenarios/skillset-depth.txt`. It covers the
no-argument list, target miss, every direct parser/value error, ordinal NPC
rejection, player mutation, multiword skill lookup, self alias, in-room name
abbreviation, `0.<name>` player-only selection, and `atoi` first-token/prefix
behavior.

Manifest: `docs/fidelity/depth/skillset.tsv` (18 rows).

Focused tests:

- `pkg/session/skillset_depth_test.go` — C registration gates and visible
  target resolution/mutation.
- `pkg/telnet/normalize_crlf_test.go` — LFCR/CRLF/LF/CR all represent one
  canonical telnet line break.
- Existing `pkg/session/cmd_skillset_test.go` — exact list, parser, mutation,
  and authorization cases, updated to register the fixture players in the
  world resolver.

With `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, the scenario
produced `result: no normalized divergence` for seeds 1, 2, 3, 5, and 8;
seed 1 was inspected with `--show-oracle`. No source or oracle-tree file was
edited.

## Gates and review

All local gates passed on the final feature commit:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` (clean)
- `git diff --check`

PR #1241's hosted lint, security, and test checks were green; build/deploy
were skipped by the workflow. It was self-merged only after the applicable
checks were green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (ordering, parser, and mutation parity), R4 (no invented behavior), R5
and R5e (verify the actual C path and let C win), with the shared telnet and
target classes corrected under R5b/R5c.
