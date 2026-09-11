# PR #1437 review census — 2026-09-11

This record is written from the isolated
glm/depth-shop-live-inventory worktree after the two focused
shops.live-inventory list corrections. The checkout was not committed or
otherwise changed by the census.

## Exact command

~~~sh
set -o pipefail
export PATH=/usr/local/go/bin:$PATH
export DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
export ORACLE_REGRESSION_GO=/usr/local/go/bin/go
export ORACLE_REGRESSION_TIMEOUT=240s
export ORACLE_REGRESSION_SEED=1
export ORACLE_REGRESSION_JOBS=4
make oracle-regression
~~~

The tested checkout was HEAD 6b29666c2 plus the uncommitted review changes
to pkg/session/shop_cmds.go, pkg/session/shop_cmds_test.go, and
cmd/dp-oracle-diff/scenarios/shop-live-inventory-control.txt.

## Final status

~~~text
oracle-regression: scenarios=938 passed=928 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0 elapsed=7020.863s started=2026-09-10T23:17:54-0400 finished=2026-09-11T01:14:55-0400
make: *** [Makefile:105: oracle-regression] Error 2
~~~

Exit 2 is solely the pre-existing, previously human-cleared
accuse-noarg-depth unpinnable baseline. The nine EXPECTED results are
ledger-backed pinned baselines. There were zero stale, failed, INFRA, or
timed-out results.

Relevant shop lines from the census:

~~~text
PASS shop-do-not-here
PASS shop-live-inventory-control
PASS shop-live-inventory-filters
PASS shop-live-inventory-gate
PASS shop-stack-list-live
~~~

One worker emitted
RETRY spec-proc-wall-guard-ns once (infrastructure-shaped failure); its
bounded retry returned
PASS spec-proc-wall-guard-ns (after infra retry). The aggregate reported
infra=0, so this was manually inspected as a recovered transient and was
not filed as an INFRA result.

The complete raw stream remains outside the harness's self-cleaning
temporary directory at
/tmp/dp-review-1437-oracle-regression.log (949 lines, 22023 bytes,
SHA-256
cf1d722cd7a7aaa452365bf92d295d64e1398a3f14c0c3cd261ee2f3c956979c).

The unrelated make expected-divergences-check blocker was not repaired,
committed, pinned, or excluded.
