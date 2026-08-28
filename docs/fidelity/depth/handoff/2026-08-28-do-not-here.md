# Depth-fidelity handoff — 2026-08-28 — `do_not_here`

## Queue position

- The special-procedure inventory and the single `objmagic.sleep-entry-gates`
  attempt are complete.
- The interpreter-table sweep has completed `ambush`, `appraise`, `auto`,
  `backstab`, `ban`, and now the generic `do_not_here` family reached by
  `balance` in table order. The next family must be selected from a fresh main
  checkout, pull, frontier report, and C-table/manifest comparison.

## C path verified

- `src/interpreter.c:346` reaches `do_not_here` for `balance`; the same handler
  is registered for `check`, `collect`, `deposit`, `hire`, `mail`, `offer`,
  `recharge`, `receive`, `remort`, `rent`, `retrieve`, `stable`, `value`, and
  `withdraw` (through `src/interpreter.c:829`). `list`, `buy`, and `sell` also
  use this fallback when their room special procedure does not intercept.
- `src/act.other.c:206-211` never parses arguments and always sends exactly
  `Sorry, but you cannot do that here!\r\n`.

## RED finding and Go fix

- The C-first `not-here-depth` vehicle showed all 15 unregistered fallback
  names returning Go's `Huh?!?` while C returned the generic refusal.
- Go now registers those names with the authoritative C-derived gates and a
  shared `cmdNotHere` handler. Existing shop handlers continue to run after
  special-procedure dispatch and retain the same fallback bytes when no shop is
  present. No files under `src/` or `darkpawns-c-oracle/` were changed.

## Proof

- New vehicle: `cmd/dp-oracle-diff/scenarios/not-here-depth.txt` probes all 15
  names with ignored arguments. Seed 1 used `--show-oracle`; seeds 1, 2, 3, 5,
  and 8 all have no normalized divergence.
- Existing `shop-do-not-here` now contributes the no-shop `list/buy/sell`
  boundary and is green with `--show-oracle`.
- Manifest: `docs/fidelity/depth/do-not-here.tsv`, 3 rows — all proven.

## Gates

- `make fidelity-depth`: PASS — 719 total, 698 proven/delegated, 6 blocked,
  15 excluded; actionable 698/704 (99.1%).
- `go build ./...`: PASS.
- `go vet ./...`: PASS.
- `go test ./...`: PASS.
- `golangci-lint run ./...`: PASS — 0 issues.
- `gofumpt -l .`: PASS — clean.

## Carry-forward

- Branch: `glm/depth-not-here`.
- Base: main `2fd67530c` (ban merged); the unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains untouched.
- Open exactly one `glm/depth-not-here` family PR. Do not merge it unless every
  hosted check is green; if CI fails to fire, retry once with the prescribed
  workflow command and leave the PR open if it remains not-green.
