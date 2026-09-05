# Depth-fidelity handoff — `credits`

Date: 2026-09-02

## Queue position and result

This round began from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the newest prior handoff, `2026-09-02-command-cheer.md`. The special-procedure
inventory remains exhausted. The one-time blocked
`objmagic.sleep-entry-gates` row was already attempted through the cast-sleep
outlaw/reagent vehicle and was not repicked. The interpreter sweep consumed the
next genuinely unmanifested family after the covered rows through `clap`,
`credits`, at `src/interpreter.c:396`.

The pre-slice frontier was 3,603 total cases, with 3,502 proven/delegated, 48
blocked, and 53 excluded. The credits manifest contributes four fully
proven/delegated cases. The resulting frontier is:

- 3,607 total cases
- 3,506 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,506/3,554 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:396 */
{ "credits"  , POS_DEAD    , do_gen_ps   , 0, SCMD_CREDITS },
```

The handler is `src/act.informative.c:2117-2130`. Its
`SCMD_CREDITS` arm passes the boot-cached `credits` string to
`page_string(ch->desc, credits, 0)` and does not inspect the command argument.
The source text is `lib/text/credits`, a 32-line, two-page static payload.
The Go entry is already registered with the same unrestricted `POS_DEAD` gate;
`cmdCredits` reads the configured `lib/text/credits` file and routes it through
the shared Go pager. No player-facing Go behavior needed correction.

## Evidence and verification

The C-first vehicle is `cmd/dp-oracle-diff/scenarios/credits-depth.txt`. It
probes the static page, completes the first invocation with RETURN, invokes the
same command with trailing words, completes the second pager pass, and runs
`score` to prove normal dispatch resumes. The focused unit proof is
`pkg/session/credits_depth_test.go`, which pins the C entry gate and the Go
registry gate. The vehicle was run on clean pre-change `main` with
`--show-oracle` at seed 1 and at seeds 2, 3, 5, and 8. Every run reported
`result: no normalized divergence`; the shown C blocks confirm the exact two
pages and post-pager command boundary.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/credits-depth.txt`;
- `docs/fidelity/depth/credits.tsv`; and
- `pkg/session/credits_depth_test.go`.

The manifest delegates shared pager quit/post-pager ownership to the existing
help pager proof under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-credits`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...  # 0 issues
gofumpt -l .             # clean
git diff --check
```

Feature commit: `4df415582` (`test: prove credits depth fidelity (R1 R2 R3 R5)`).

Feature PR: #1227 (`glm/depth-credits`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting `main`
merge commit is `94a70a20d`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed/pager evidence), R4 (no invented output), R5/R5e (the actual C
call path), and R5b/R5c (shared pager ownership).

## Continuation

The live interpreter-table-versus-manifest sweep expands slash-separated
families, skips the `RESERVED` sentinel and covered aliases, and finds the next
genuinely unmanifested command in table order:

```c
/* src/interpreter.c:411 */
{ "search"   , POS_STANDING, do_detect   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`search` before advancing in interpreter-table order.
