# Entry cleanup review — 2026-09-08

This is a follow-up to Luna's lifecycle proof, not production certification.
Production data, deployment, oracle source and normalizers are unchanged.

## Confirmed findings and repairs

1. **Stale oracle inputs (R1/R5c/R5f).** The five color-related census failures
   were caused by old `[setup:port]` inputs: name, Y, password twice, **Y**, N,
   sex, race, class, hometown, accept, return, enter. The extra Y formerly
   answered a port-only question; now it answers the shared color prompt. C
   receives N. `src/interpreter.c:1992` sets color bits on Y, as Go now does.
   Removed the extra input from 1,757 port setup blocks in 932 files. Of these,
   1,737 whole setup streams then match their C stream except confirmation case.
   The other 20 have existing differences such as room alignment commands,
   synthetic passwords or a saved C identity; only the same obsolete creation
   input was removed. No setup parser, normalizer or game color behavior changed.
2. **Pre-world TCP EOF (R5e).** `src/comm.c:2129` retains only CON_PLAYING as
   linkdead. Authentication now precedes world entry, so the old Go auth-only
   gate incorrectly retained new characters at MOTD/menu. The telnet handler
   now rejects creation/menu states from linkdead handling, using normal cleanup.
   Real TCP/PostgreSQL tests reproduce both old failures and prove both repaired
   reconnect paths. They also prove playing EOF still retains a linkdead player.
3. **Transient entry-save failure.** Luna's permanent-failure test missed the
   teardown retry: a one-shot SavePlayer failure was followed by cleanup saving
   the already-advanced level-one candidate, overwriting the accepted level-zero
   row. `abortEntry` now clears the candidate before unregistering it. The new
   one-shot failure test reproduces the old overwrite and proves the durable row
   remains level zero. Session cleanup and closure still run.
4. **Proof precision.** Renamed the original single-login telnet test so its name
   does not claim reconnect. The new WebSocket reconnect assertion now checks the
   exact accepted ID, rather than merely checking for a positive ID. Terminal
   room observation no longer assumes separate TCP reads for welcome and room.

## Validation

Historical red output is retained alongside this report. The opt-in PostgreSQL
tests used only isolated schemas on the disposable loopback cluster at port 55439.

- `go build ./...`, `go vet ./...`, `go test ./...`: pass on the final Go code.
- `golangci-lint run ./...`: 0 issues with a fresh task-local lint cache.
- `make fmt`, `make check-fmt`, `make fidelity-depth`: pass.
- `node --test scripts/entry-client.test.mjs`: 4 passed.
- `DP_ENTRY_TEST_DATABASE_URL=... go test -race ./pkg/telnet ./pkg/session
  -run '^TestEntry' -count=1`: pass against the disposable PostgreSQL cluster.
- Focused oracle regression: all five previously failing color/report scenarios,
  creation-name-retry and God smoke pass (7/7, no exceptions).
- Final-code `character-creation`, `character-creation-name-retry`, and
  `god-harness-smoke` each pass seeds 1,2,3,5,8 (15/15), all with
  `--show-oracle`. See cleanup-oracle-green.txt.
- Complete fixture census: result pending below.

The full census binary was built after the telnet state-gate repair and before
the later abortEntry transient-failure repair. The latter branch requires a
persistence failure and is not reachable in the census's no-database vehicle;
final-code PostgreSQL race tests and repository gates cover that branch. The
seeded creation runs use a server rebuilt after both repairs.

## Remaining boundaries

The broader blocked rows in entry.tsv remain open. These checks do not certify
all menu actions, restrictions, usurp/unswitch, frozen/unhealthy entry, every
identity consumer, or full visual-browser and preference-persistence fidelity.
The three production identity collisions still require owner-reviewed resolution.
