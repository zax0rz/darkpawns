# Handoff — oracle retry classification

Date: 2026-09-12
Branch: `glm/fix-oracle-retry-classification`
Checkpoint: `8c8089d96`

## Completed milestone

The oracle regression worker now applies one bounded three-attempt budget to
recovery and content confirmation. Infrastructure attempts supply no content
proof. Successful attempts honor baseline membership (`PASS` versus `STALE`),
and status-3 attempts require valid, non-empty fingerprints plus a repeated
identical content attempt before ledger-backed exact pin agreement can produce
`EXPECTED`. Unstable, unpinned, pin-mismatched, malformed, timeout, and
infrastructure outcomes retain distinct gate categories. Every attempt log is
preserved separately.

The deterministic fake-harness suite passes 29 cases, including aggregate
classification. The real smoke matrix passes `yuball-depth` as PASS and
`medit-entry-depth` as EXPECTED; it reports only the established
`accuse-noarg-depth` UNPINNABLE boundary. Evidence and durable logs are in
`docs/fidelity/evidence/2026-09-12-oracle-retry-classification/README.md` and
`/home/zach/oracle-retry-evidence-2026-09-12/`.

No production game, oracle, normalizer, ledger, pin, exclusion, CI, or
scenario input changed. This PR does not complete #1447 and does not provide
its full saving-throw census.

## Next integration step

After human review, bring the approved runner fix onto PR #1447 without
changing its saving-table scope. Freeze the scripts, ledger, pins, scenarios,
and seed, then rerun #1447’s full parallel census. Require the complete
current tally with `failed=0 infra=0 timed_out=0 stale=0`, allowing only the
specifically human-cleared `accuse-noarg-depth` UNPINNABLE result. Keep both
PRs unmerged until that review and integration gate are complete.
