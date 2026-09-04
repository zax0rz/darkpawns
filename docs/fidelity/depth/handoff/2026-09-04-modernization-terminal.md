# Merged fidelity + modernization terminal handoff — 2026-09-04

## Terminal verdict

The merged queue is complete to the RED boundary. The final post-Phase-1
corpus run is green:

```text
make oracle-regression
oracle-regression: scenarios=934 passed=934 failed=0 infra=0 timed_out=0 elapsed=7291.902s
started=2026-09-04T16:40:06-0400
finished=2026-09-04T18:41:38-0400
```

The runner used `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`,
`/usr/local/go/bin/go`, seed 1, four workers, a 240-second per-scenario
timeout, and one retry only for infrastructure-shaped boot/connection errors.
No timeout-kill was classified as a content diff. The focused
`shop-stack-list-live` vehicle is included in the 934 scenario files and is
green.

## Two denominators

The denominators remain intentionally separate:

| ledger | total | proven | blocked | excluded | result |
| --- | ---: | ---: | ---: | ---: | --- |
| modeled depth corpus | 4,758 cases | 4,653 | 54 | 51 | `make fidelity-depth`; 98.9% of actionable cases |
| source-order surface inventory | 4,926 weighted units / 70 rows | 887 / 8 rows | 4,038 / 61 rows | 1 / 1 row | zero blanket proof tokens; blocked is an explicit expected disposition |

The surface inventory is not silently collapsed into the modeled case corpus.
All 61 former `terminal-surface-audit-2026-09-04` markers are replaced by
source/family-specific attempt tokens. The single exclusion retains its C
command-reachability reason. The 61 blocked rows remain blocked where the
whirlpool-clinic sweep could not prove the full source family; this includes
admin/immortal-gated, level-gated, lifecycle, full spell-vector, combat, and
broader shop surfaces. No unprovable surface was strained into an exclusion.

## Lane A disposition

- Phase 0.1: `make oracle-regression` exists, is executable, and has the full
  wall-time above.
- Phase 0.2: the stale 15-social count is corrected. Reachable social coverage
  is present through `yuball`; `hiss`, `kneel`, and `mutter` are in
  `lib/misc/socials` but absent from C's command table and remain excluded under
  R2/R4/R5e. No commands were invented.
- Phase 0.3: the shadow shop stack was confirmed live-broken and fixed at the
  parsed-shop → world/session keeper-list boundary. The focused list vehicle is
  green; broader buy/sell/value/economy parity remains blocked.
- Phase 0.4: stale G104, social-queue, shop-stack, surface, Phase 1, and
  roadmap wording is corrected in the cited handoffs and roadmap.
- Phase 1.1: retired `pkg/game/socials.json` and `pkg/game/socials.txt` were
  removed after the reachability scripts were redirected to authoritative C
  `lib/misc/socials` records.
- Phase 1.2: the 628-line commented-code count was verified as a stale lexical
  heuristic; source-mapping/C-formula/test commentary was retained under R4,
  with no wholesale deletion.
- Phase 1.3: exact-symbol audits confirmed and removed the dead `cmdNotBuy`,
  `cmdInfo`/its dead-only helpers, and `applyAuthFormat`; the U1000 ratchet was
  lowered to the resulting file-ignore count.

## Lane B disposition

The source-order clinic handoff was committed before editing the inventory.
Each blocked row now identifies its own source/family attempt in
`proof_or_attempt`, and its notes retain the C scope and residual owner. The
full modeled run is green, but this does not promote the unmodeled surface
families. A later owner may promote a row only with the oracle/multiseed or C
reachability evidence required by `DEPTH_TESTING.md` and R1–R5.

## Interlock and changed-file lookup

The branch has full corpus and repository gates, but it is **not eligible for
modernization self-merge** under the three-condition amendment: the
behavior-adjacent shop bridge changes files whose direct coverage mapping is
partial/blocked (`list`/`buy` are `DEPTH-PARTIAL-BLOCKED` in
`reports/raw/coverage-mapping.tsv`, and the source-order shop rows remain
blocked). The branch therefore stays open for human merge review. The exact
changed-file lookup against `git diff --name-only origin/main...HEAD` is:

```text
AGENTS.md                                      docs/gates; no player surface
Makefile                                       oracle-regression target/tooling
cmd/dp-goat/internal/config/config.go         confirmed dead helper deletion
cmd/dp-oracle-diff/scenarios/shop-stack-list-live.txt  focused shop vehicle; green
docs/fidelity/depth/handoff/2026-09-04-modernization-phase0.md  handoff/evidence record
docs/fidelity/depth/handoff/2026-09-04-modernization-phase1.md  handoff/evidence record
docs/fidelity/depth/handoff/2026-09-04-modernization-surface-inventory.md  handoff/evidence record
docs/fidelity/depth/handoff/2026-09-04-shadow-shop-stack.md  handoff/evidence record
docs/fidelity/depth/handoff/2026-09-04-social-queue.md  handoff/evidence record
docs/fidelity/depth/handoff/2026-09-04-modernization-terminal.md  terminal handoff record
docs/fidelity/depth/handoff/2026-09-04-terminal-audit.md  opening-boundary wording corrected to point to current clinic
docs/fidelity/depth/surface-inventory.tsv     70-row source-order ledger
docs/research/EVIDENCE_LEDGER.tsv             DP-010 through DP-012 evidence entries
internal/lintguard/u1000_ratchet_test.go      test ratchet for deleted dead files
pkg/game/socials.json                          retired generated artifact; deleted
pkg/game/socials.txt                           retired generated artifact; deleted
pkg/game/world.go                              shop bridge; focused vehicle green; shop row blocked
pkg/game/world_write.go                        shop bridge; focused vehicle green; shop row blocked
pkg/game/world_zone.go                         shop bridge; focused vehicle green; shop row blocked
pkg/parser/shop.go                             parsed C shop fields; focused vehicle green; shop row blocked
pkg/parser/shop_test.go                        parser coverage for parsed fields
pkg/session/info_cmds.go                       confirmed dead command file; deleted
pkg/session/shop.go                            confirmed dead helper file; deleted
pkg/session/shop_cmds.go                       live keeper-list boundary; shop row blocked
scripts/gen_reachability.py                    authoritative C social source
scripts/oracle_regression.sh                   corpus harness
scripts/reachability_ci_gate.py               wording/source correction
```

The raw mapping has no direct row for the new world/parser bridge files; that
absence plus the blocked shop family is an uncovered/partial mapping under the
amendment. This is why the branch is human-merge-only even though the corpus
is green. RED-set files were not changed.

## Gates and stop condition

Before the final corpus run, the full repository gates passed: `make check-fmt`,
`make fidelity-depth`, `/usr/local/go/bin/go build ./...`,
`/usr/local/go/bin/go vet ./...`, `/usr/local/go/bin/go test ./...`,
`PATH=/usr/local/go/bin:$PATH golangci-lint run ./...`, and `gofumpt -l .`
returned clean. The research ledger now contains DP-010 through DP-012.

The required stop state is reached: Lane B has no blanket blocks, Lane A is
complete through the RED boundary, the final full corpus is green, both
denominators are recorded, and the residual blocked families are explicitly
owned. No CI, deployment, secret, website, save-format, or oracle-source work
was performed. STOP.
