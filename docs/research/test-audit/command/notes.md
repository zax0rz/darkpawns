# Test audit: pkg/command — notes

Audit run: 2026-11-20, branch `kimi/test-audit-command`, base `origin/main` (b2e884aca).
Auditor: Kimi. Method: `docs/research/test-audit/pilot-combat/BRIEF.md` with the three
changes from PROMPT-2026-11-20 (single agent self-checks non-R marks; fix-as-you-go;
output in-repo).

Baseline: `go test ./pkg/command/...` passes at b2e884aca.

Ground truth: 173 test functions in 34 files (5,025 test lines) against 14 production
files (7,385 lines). 37 tests born before the first oracle harness (2026-07-12,
ba05d9906): 6 registry/middleware (a767d832c, 2026-05-27), 1 rescue (7465d8aef,
2026-06-26), 3 bash (54cd125d9/02b111ae4, 2026-07-05/11), 27 kill-payout (8183a9039,
2026-07-06).

## Predictions (written before reading any test)

Both earlier audits (PF-103 combat, PF-107 spells) put nearly all weak marks on
pre-oracle tests: 93 of 94 later combat tests were keepers. I expect the same shape
here, slightly milder — pkg/command's later tests are mostly oracle-era depth tests
with C citations, and its one big pre-oracle block (27 kill-payout tests from a
single 2026-07-06 commit) is exactly the kind of go-generated-expectation suite the
pilot flagged as J1.

| Mark | Pre-oracle (37) | Later (136) | Total (173) |
|---|---|---|---|
| R keep | 24 | 127 | 151 (87%) |
| F fix | 7 | 5 | 12 (7%) |
| C merge | 4 | 3 | 7 (4%) |
| D delete | 2 | 1 | 3 (2%) |

Junk codes expected: J1 (Go-generated expected values, especially kill-payout XP
numbers and gold) dominates pre-oracle; J2 (assertions that cannot fail) and J8
(mock/fixture supplies the asserted behavior) next; a little J5 (implementation-shape
assertions) in the registry/middleware block. At least one live Go/C divergence
pinned by a test (J10) — both earlier audits found exactly one each.

Confidence: low on the F/C/D split between pre-oracle and later (the totals are a
firmer guess than the partition).

## Baseline

`GOEXPERIMENT=nojsonv2 go test ./pkg/command/...` → ok (2026-11-20).

## Batches

- Batch 1: the 37 pre-oracle tests.
- Later batches: remaining 136, oldest first, ~50–100 per batch.
