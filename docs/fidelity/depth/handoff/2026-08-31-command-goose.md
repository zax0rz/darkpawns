# Depth handoff — 2026-08-31 — `goose`

## Frontier and queue position

- Started from clean `main` at `7d8bdd9ab` after merging the `goto`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-goto.md`.
- The starting frontier was 1,881 total, with 1,824 proven/delegated, 16
  blocked, and 41 excluded. The dedicated `goose` manifest adds nine
  proven/delegated cases. The post-slice frontier is 1,890 total, with 1,833
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,833/1,849 (99.1%).
- A fresh source-order audit confirms `goose` at `src/interpreter.c:470` is
  covered. `goto` is covered by `goto.tsv`, `gossip` by `channels.tsv`, and
  `gold` by `spec-procs.tsv`; the next un-manifested command-table family is
  `group` at `src/interpreter.c:471`. The next session must return to clean
  `main`, pull, rerun the frontier check, reread this handoff, and begin
  `group` while leaving glare PR #896 open.

## C call path and branch inventory

`src/interpreter.c:470` registers `goose` with `POS_RESTING`, no command
minimum level, and `do_action`. The C handler path is
`src/act.social.c:102-151`: it finds the social record, rejects
`PLR_NOSHOUT`, calls `one_argument` because `char_found` is present, and then
selects the no-argument, visible-target, not-found, self-target, or minimum
victim-position branch. The `goose` record at `lib/misc/socials:1456-1461`
is `goose 0 0`, so it does not hide the actor and admits every victim
position. Its authored messages are:

```
Goose who?
#
You goose $M. MMmmMM
$n sticks out $s thumb and gooses $N.
$n gooses you.. woo woo!
Goose who?
That's disgusting!
$n sticks $s thumb up $s butt.
```

The existing Go generic social path already mirrors these branches through
`game.DoAction`, `oneArgument`, `findSocialTarget`, and canonical `Act`; no
runtime divergence was confirmed and no implementation change was needed.

## Coverage proof

The unchanged-main `goose-depth --seed 1 --show-oracle` run was GREEN and
showed the intended C blocks for no argument, a visible target with leading
fill words and trailing input, a missing target, and a self target. The same
vehicle reported no normalized divergence for seeds 1, 2, 3, 5, and 8.

The slice follows R1/R2/R4/R5e and R5c: the C social table, registered command
surface, one-argument behavior, target branch order, and Act audiences remain
authoritative; no unreachable behavior was invented; and shared position,
PLR_NOSHOUT, and visibility behavior is delegated to established owners.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/goose-depth.txt` with five annotated
  direct cases and named room peers.
- Added `docs/fidelity/depth/goose.tsv` with nine explicit rows.
- Added `pkg/session/goose_test.go` to pin the C command and social metadata.
- Local gates passed: `make fidelity-depth` — 1,890 total /
  1,833 proven-or-delegated / 16 blocked / 41 excluded; `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (0 issues),
  `gofumpt -l .`, and `git diff --check`.

Implementation PR #900 was self-merged only after hosted `lint`, `security`,
and `test` checks were green following the one permitted workflow retry. Its
`build-and-push` and `deploy` jobs were skipped by policy. The glare
implementation PR #896 remains open because its hosted test failure is the
unrelated pre-existing retry-based `pkg/spells/TestMagAffects_Sleep`; it was
not retried or fixed forward.
