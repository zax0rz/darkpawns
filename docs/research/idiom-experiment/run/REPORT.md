# Idiomatic Go experiment report

Status: ready for draft PR and review. Seven one-target refactor commits are
integrated. Each passed its local gates, each targeted real census is CLEAN,
and each one-byte mutation probe was caught by its target scenarios.

The integrated branch passed `make fmt`, `go build ./...`, `go vet ./...`,
`go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, and
`make deadcode-ratchet` (52 baselined, zero new). The focused prediction lint
count command returned its expected nonzero status because findings remain:
dupl 26, funlen 191, gocognit 111, gocyclo 4; baseline counts were 26, 190,
112, and 4. Its output and exit status are retained in `lint-after.txt`.
The full linter exited 0 with no issues; it emitted a nonfatal warning about a
missing source file in a different worktree. Baseline focused-lint output also
contains nine non-target findings absent from the after output, so only the
four pre-registered categories are compared.

The targets are `DoDrink` sustenance extraction, `DoPour` pour-out extraction,
`doPut` put-all extraction, `doGive` coin-handling extraction, `doDoorcmd`
repeated state-update consolidation, `doGenTog` preference-toggle extraction,
and `HcontrolBuildHouse` direction-lookup extraction. All seven remain in the
review branch. Their census and probe outcomes are in `telemetry.tsv`.

This runtime is Codex based on GPT-6, while the pre-registered brief names
GPT-Luna. The results must not be reported as a GPT-Luna replication. See
[`RESULTS.md`](RESULTS.md) for the row-by-row prediction comparison and census
directories.

Two `give` behavior discrepancies were read through the Go and C call paths and
recorded in [`findings.md`](findings.md). Neither was fixed. No rulebook text is
proposed at this stage.

The evidence-ledger update is deferred because open PR #2271 owns
`docs/research/EVIDENCE_LEDGER.tsv`. The final branch-tip combined census is
reserved for Claude's reviewed PR batch by the repository instructions; all
targeted real and mutation-probe censuses for this run have completed.

The scope fell by 17 lines (73,557 to 73,540); functions over 100 lines fell
from 48 to 45. The longest function stayed at 747 lines, while the focused
`funlen` count rose by one and `gocognit` fell by one. Seven commits were kept,
below the pre-registered estimate of 10–45. No first real census was red, and
none of the seven probes stayed green. The two `give` findings are the only
latent behavior discrepancies logged. The 31.8% readiness improvement is
confounded by a change in host memory load; RSS changed by +0.15% and binary
size by −0.026%.

Protected-path audit on the final branch tip: passed, no protected paths
changed. No rulebook amendment is proposed. The evidence ledger remains for
integration after open PR #2271; the final combined census remains with
Claude's batch review.
