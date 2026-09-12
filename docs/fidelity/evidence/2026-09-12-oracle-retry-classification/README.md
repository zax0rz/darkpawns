# Oracle regression retry-classification evidence

Date: 2026-09-12
Branch: `glm/fix-oracle-retry-classification`
Base: `origin/main` at `e54301fb4abfea6bf0c4811111d244c4ae141912`
Code checkpoint: `8c8089d96`

## Scope and before/after

This bounded change repairs only `scripts/oracle_regression_worker.sh`, its
deterministic test harness, and the test target that runs it. It does not
change production game code, saving tables, oracle normalization, ledger
membership, pins, exclusions, CI workflows, or the C oracle.

The pre-fix worker had two inconsistent paths:

| deterministic sequence | pre-fix result | pre-fix attempts | correct result |
| --- | --- | ---: | --- |
| infrastructure signature → status 3 with the exact pinned set | `FAIL\tfake\t3` | 2 | `EXPECTED\tfake` only after a confirming content attempt |
| infrastructure signature → status 0 with a baseline row | `PASS\tfake` | 2 | `STALE\tfake` |

These results were reproduced by the real pre-fix worker with the checked-in
fake harness. The durable logs, result files, and attempt counts are under
`/home/zach/oracle-retry-evidence-2026-09-12/pre-fix-v2/`. The historical
`medit-entry-depth` reproduction is also preserved in
`/home/zach/saving-throw-evidence-2026-09-12/oracle-regression.log`: its
infrastructure-shaped first attempt was followed by status 3 and the three
fingerprints matched the current pins, but the old worker printed `FAIL`.

## Outcome and attempt matrix

The implementation uses one explicit maximum of three process attempts per
scenario. Recovery attempts and content-confirmation attempts consume the
same counter; no path resets it. A status-3 attempt is content evidence only
when its log contains a non-empty set of unique labels whose digests are
exactly 64 lowercase hexadecimal characters. A valid content pair must have
identical sorted sets before it can be compared with pins.

| Evidence observed so far | Baseline/pins condition | Result |
| --- | --- | --- |
| status 0, no prior content divergence | no ledger row | `PASS` |
| status 0, including after recovery or content attempts | ledger row exists | `STALE` |
| infrastructure signature, budget remains | any | run the next attempt; infrastructure supplies no content proof |
| status 3 with valid fingerprints, no confirming content attempt before budget ends | any | `INFRA` |
| two valid content attempts with identical fingerprints | ledger row and exact complete pin set | `EXPECTED` |
| two valid content attempts with identical fingerprints | no ledger row | `FAIL` |
| two valid content attempts with identical fingerprints | ledger row but missing, extra, malformed, or changed pins | `FAIL` |
| two valid content attempts with different fingerprints | ledger row | `UNPINNABLE` |
| two valid content attempts with different fingerprints | no ledger row | `INFRA` |
| content divergence followed by status 0 | ledger row | `STALE` |
| content divergence followed by status 0 | no ledger row | `INFRA` (flaky-red, not stable proof) |
| timeout status 124, 137, or 143 | any | `TIMEOUT` |
| infrastructure signature at the end of the three-attempt budget | any | `INFRA` |
| other nonzero status without an infrastructure signature | any | `FAIL` |
| status 3 with missing or malformed fingerprint records | any | `FAIL`; empty-set equality cannot pass |

For example, infrastructure → valid content → identical content uses all three
attempts and can become `EXPECTED` only when the ledger and complete pins also
agree. Infrastructure → valid content → success cannot become `EXPECTED`.

The aggregate runner already counts each worker result category explicitly;
the deterministic aggregate checks exercise `PASS`, `STALE`, `EXPECTED`,
`UNPINNABLE`, `FAIL`, `INFRA`, and `TIMEOUT` with their established aggregate
exit codes. No aggregate defect was found.

## Deterministic proof

The real worker is invoked by
`scripts/test_oracle_regression_worker.sh` through
`scripts/testdata/oracle_regression_fake_harness.sh`. The fake harness uses an
explicit sequence and exits with controlled statuses; it performs no network
I/O, launches no game server, and does not depend on timing. Each attempt is
written to a distinct `fake.attemptN.log`, and the tests assert the exact
result file, invocation count, and preserved log count.

The test covers immediate PASS/STALE, recovery to PASS/STALE, pinned,
unpinned, and pin-mismatched content, divergence followed by success or
different fingerprints, content → infrastructure → content confirmation,
budget exhaustion, timeouts, missing/malformed fingerprints, and multi-block
pin sets with missing/extra/changed blocks. It also runs the aggregate script
with the real worker and fake build wrapper.

Runnable command:

```bash
export PATH=/usr/local/go/bin:$PATH
make oracle-regression-worker-test
```

Result: `29 deterministic oracle regression cases` passed (22 direct worker
cases and 7 aggregate cases).

## Real smoke

Before selecting the smoke cases, current source membership was checked:
`yuball-depth` has no expected-divergence row; `medit-entry-depth` is
ledger-backed with three exact pins; `accuse-noarg-depth` is ledger-backed with
the established run-varying boundary.

Exact command:

```bash
export PATH=/usr/local/go/bin:$PATH
export DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
ORACLE_REGRESSION_SCENARIOS='yuball-depth,medit-entry-depth,accuse-noarg-depth' \
ORACLE_REGRESSION_JOBS=1 \
ORACLE_REGRESSION_TIMEOUT=240s \
make oracle-regression
```

The run was performed at checkpoint `8c8089d96` with frozen scripts and
inputs. It produced:

```text
scenarios=3 passed=1 expected=1 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The make target exited 2 solely because `accuse-noarg-depth` is the named,
human-cleared `UNPINNABLE` case. Durable output is in
`/home/zach/oracle-retry-evidence-2026-09-12/smoke-8c8089d96/run.log`, with
the recorded exit in `status`.

This is a runner smoke result, not the missing saving-throw census. The full
940-scenario census remains the subsequent integration step after review of
this runner fix.

## Validation and boundaries

The implementation checkpoint passed shell syntax checks, `make
oracle-regression-worker-test`, and `git diff --check`. The final branch
validation additionally runs the repository build, vet, tests, lint,
`make fidelity-depth`, and `make expected-divergences-check`.

Changed files are:

- `Makefile` — adds the explicit deterministic runner-test target.
- `scripts/oracle_regression_worker.sh` — bounded state machine, fingerprint
  validation, consistent baseline/pin classification, and per-attempt logs.
- `scripts/test_oracle_regression_worker.sh` — deterministic worker and
  aggregate regression cases.
- `scripts/testdata/oracle_regression_fake_harness.sh` — controlled attempt
  sequence harness.
- `scripts/testdata/oracle_regression_fake_go.sh` — deterministic aggregate
  build shim.
- `docs/fidelity/evidence/2026-09-12-oracle-retry-classification/README.md` —
  this evidence record.
- `docs/fidelity/depth/handoff/2026-09-12-oracle-retry-classification.md` —
  dated integration handoff.

No pins, ledger rows, normalization code, expected-divergence membership,
scenario fixtures, production game behavior, or save format changed.
