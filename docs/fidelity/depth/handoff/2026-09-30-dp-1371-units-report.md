# DP-1371 phase 2: unit-proof integrity

Measured 2026-09-30 after PR #1709 merged. Findings are unchanged: this PR
adds tooling and its tests, the Makefile entry points, and this report.
No game, harness, scenario, manifest, ledger or governing-document changes.

## Main measurement and evidence

Main/test/manifest HEAD: `34d7173b0a4179b08142422fd14d7a6737d6d968`.
Tooling HEAD: `8b52cf25d2570436fdc70dd10d573c69bd7a5b72`.
Both trees were clean at run start. Main was a separate detached checkout,
`/home/zach/dp-integrity-2-reference`; the tool ran from `sol/integrity-2`.

```sh
make fidelity-units UNIT_ROOT=/home/zach/dp-integrity-2-reference OUT=/home/zach/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-units-baseline
```

One full run, **17.835 seconds**, Go 1.26.6 linux/amd64. The Python tool
returned 1 (Make returned 2) because two claims are unproven. Every one of
the nine package commands returned 0. Test caching was disabled with
`-count=1`; package selection and anchored root-test expressions were sorted.
The shared `fidelity_manifest.load_rows` supplies both validation and execution.

Evidence directory:
`/home/zach/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-units-baseline/`.
It retains `MANIFEST.md` with exact commands and HEADs, `summary.txt`,
`test-index.json` with declarations and source snapshots, nine package JSON
logs, all 1,144 rows in `unit-results.tsv`, and
`classified-unit-claims.tsv` with class and likely cause.

| Measure | Count |
|---|---:|
| unit-green rows / row-symbol claims | 1,144 |
| Distinct claimed symbols | 925 |
| Distinct resolvable symbols executed | 923 |
| Packages | 9 |
| Passing row claims | 1,142 |
| Prefix-only string matches | 2 |
| Missing declarations or emitted runtime tests | 0 |
| Wrong test signatures | 0 |
| Skipped roots or descendants | 0 |
| Failing tests / package failures | 0 |
| Confirmed asserts-nothing proofs | 0 |

There are no existing multi-symbol or subtest-qualified claims. The tooling
supports both, and checks subtest claims against actual terminal JSON events:
an unreachable literal `t.Run` is insufficient. A skipped descendant makes a
root proof unproven. Build failures, absent events and `TestMain` failures
cannot become passing claims.

| Package | Passing row claims |
|---|---:|
| pkg/game | 604 |
| pkg/session | 461 |
| pkg/spells | 31 |
| pkg/command | 28 |
| pkg/scripting | 7 |
| pkg/combat | 6 |
| pkg/boards | 2 |
| pkg/telnet | 2 |
| pkg/parser | 1 |

## Findings: one class, two rows

Both claims were accepted by the old substring search because each symbol is
a prefix of a longer real function name. Neither claimed exact test exists.
The strict validator reports both rows as errors. No proof was renamed here.

| Manifest row / case | Claimed symbol | Problem / class | Likely cause and source |
|---|---|---|---|
| `dns.tsv:8`, `dns.add-three-octet-entry` | `TestExecDNSAddDeletePrefixAndPersistence` | matched only as a string / false claim | Prefix matching accepted `TestExecDNSAddDeletePrefixAndPersistenceState`, declared at `pkg/game/dns_test.go:11`. |
| `spec-procs.tsv:308`, `room.carrion-spawn-state` | `TestSpecCarrion_SuccessfulSpawnState` | matched only as a string / false claim | Prefix matching accepted `TestSpecCarrion_SuccessfulSpawnStateAndAudience`, declared at `pkg/game/spec_carrion_test.go:107`. |

All citations refer to the measured main HEAD above. The full paths for these
manifests are under `docs/fidelity/depth/`. Evidence establishes incorrect
symbol claims; it does not establish whether either longer test proves the
claimed behavior. That question belongs to the next approved repair phase.

## Separate class: tests that assert nothing

**Zero confirmed findings.** The AST check recognizes empty, bare-return-only
and literal-logging-only bodies, including literal subtest callbacks. It does
not label an unknown helper or a panic-sensitive operation as vacuous.

As a further screen, 922 of the 923 referenced functions contain a call named
`Fatal`, `Fatalf`, `Error`, `Errorf`, `Fail` or `FailNow`. The remaining function,
`TestSkillMessageColorFramingAtCMP` at
`pkg/combat/skill_message_branch_test.go:140`, calls
`assertSkillMessageEvents` at line 144. That helper at line 68 compares the
captured event sequence and calls `t.Errorf` at line 71 when it differs.
It is not an empty proof.

This is a conservative vacuity screen, **not an R5h mutation audit**. A failure
method name alone does not prove its receiver, reachability, assertion strength,
or connection to a manifest row. Tests can still assert the wrong thing,
check an unrelated path, or contain an unreachable assertion. Those questions
remain unmeasured. Passing execution must not be presented as proof that every
one of these tests can fail on its claimed regression.

## Tooling proof and gates

Seven isolated break/restore checks each recorded exit **0 → 1 → 0**:
real-signature enforcement; empty-body detection; exact-name resolution;
skip rejection; anchored selection; build-failure rejection; runtime subtest
verification. Go checks run alone with exact `-run` names; Python methods run
alone by their unittest names. Every Python mutation run used a separate bytecode
cache to avoid stale imports. Real Go fixture modules exercise failures and
skips; production tests were not changed.

Evidence:
`/home/zach/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-unit-tooling-proofs/`:
`break-restore.json` plus each named green, broken and restored log.

`make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint cache clean`, `golangci-lint run ./...` (0 issues),
`git diff --check`, and all five Python script tests passed.
`make fidelity-depth` intentionally reports precisely the two unchanged
prefix-only claims and exits nonzero. No finding was edited to turn that gate
green. No oracle census was run: this changes unit-proof tooling, not the
server, oracle harness, or scenario corpus; the requested measurement is the
unit run on main.

## R5h audit proposal — approval before implementation

Deduplicate execution by symbol, but retain a row-to-assertion map. There are
925 named symbols, two unresolved, and 219 additional row references to the
923 runnable symbols. One killed mutation in a reused test does not establish
every associated row: require the mutation and failing assertion to cover the
specific claimed behavior.

The costs below are planning estimates, not measured audit times. The only
measured runtime is the 17.835-second nine-package baseline above; focused
mutation runs may have different compilation and setup costs. Run mutants in
isolated worktrees; retain clean/broken/restored commands, terminal events,
HEADs and diffs. A crash or timeout needs explanation, not automatic credit as
a meaningful assertion failure.

| Option | Estimated cost | What it catches | What it leaves open |
|---|---|---|---|
| Stratified sample of 100 runnable symbols, covering all nine packages and major behavior classes; one behavior-specific production mutation each | At an assumed 10–20 minutes of mutation design, row mapping and review per symbol: about 17–33 person-hours, plus execution. If each mutant conservatively reran the entire measured baseline, 100 broken/restored pairs would cost about 59 minutes of machine time before extra builds. | Weak or unreachable assertions, wrong call paths, and classes of over-claims; produces retained R5h evidence for the sampled rows. | Unsampled symbols and behaviors; a sample cannot certify all 1,144 rows. |
| Per-row historical revert notes and reruns | At an assumed 5–15 minutes per row: about 95–286 person-hours, plus reruns. Reused symbols can share execution, but each row still needs a mapping. | Whether the claimed original fix is necessary for that row's test; catches tests added without exposing the original regression. | Changes without an isolated historical fix, inseparable commits, and new regression classes; a note alone is insufficient without a failing rerun. |
| Full targeted mutation audit of the 923 runnable symbols with explicit row-to-assertion mapping | At an assumed 10–20 minutes per symbol: about 154–308 person-hours, plus extra work for multi-behavior tests and 219 repeated row references. Baseline-cost broken/restored pairs would be about 9.1 machine-hours; this is only an extrapolation. | Direct evidence for every mapped behavior, including helpers and reused tests; surviving mutants expose specific weak proofs. | A mutation set never proves all conceivable regressions; irrelevant mutants must be excluded with reasons rather than counted as evidence. |

Recommendation: approve the 100-symbol stratified sample first, after the two
unresolved names get their separately approved repair. Record the sampling
frame and all outcomes, including survivors. Use its measured design cost and
shared-cause findings to propose batches for the full audit. Do not call the
remaining rows R5h-verified based on a sample. No mutation campaign or governing
rule change is implemented in this PR.
