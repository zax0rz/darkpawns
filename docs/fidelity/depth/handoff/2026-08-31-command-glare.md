# Depth handoff — 2026-08-31 — `glare`

## Frontier and queue position

- Started from clean `main` at `38bc63669` after merging the `giggle`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-giggle.md`.
- The starting frontier was 1,868 total, with 1,811 proven/delegated, 16
  blocked, and 41 excluded. The `glare` branch adds ten proven/delegated
  cases, but its evidence PR #896 remains open because the hosted test job
  failed in the unrelated pre-existing retry-based
  `pkg/spells/TestMagAffects_Sleep`; the same focused race test passed locally.
  No retry-into-green or unrelated fix was made.
- A fresh source-order audit confirms `glare` at `src/interpreter.c:465` is
  covered by PR #896. `glance` is covered by `equipment-glance-depth`,
  `gossip` by `channels-depth`, and `gold` by `spec-proc-bank`; the next
  un-manifested command-table family is `goto` at line 467. The next session
  must return to clean `main`, pull, rerun the frontier check, reread this
  handoff, and begin `goto` while leaving #896 open.

## C call path and branch inventory

`src/interpreter.c:465` registers `glare` with `POS_RESTING`, no command
minimum level, and `do_action`. The shared handler path is
`src/act.social.c:102-151`: it checks the social lookup and `PLR_NOSHOUT`,
uses `one_argument` for this record, resolves a visible target, then selects
the no-argument, not-found, self-target, minimum-victim-position, or target
actor/room/victim audience branch.

The authored record at `lib/misc/socials:305-314` is `glare 0 5` with these
messages:

```
You glare at nothing in particular.
$n glares around $m.
You glare icily at $M.
$n glares at $N.
$n glares icily at you, you feel cold to your bones.
You try to glare at somebody who is not present.
You glare icily at your feet, they are suddenly very cold.
$n glares at $s feet, what is bothering $m?
```

The `5` is C's minimum victim position. A sleeping target therefore receives
the exact position rejection before any audience act. The record's hide flag
is zero; shared position, noshout, and Act visibility behavior is delegated to
existing manifests.

## Coverage proof

The unchanged-main `glare-depth --seed 1 --show-oracle` run was GREEN and
showed the intended C blocks for no argument, a visible target, self target,
missing target, and leading fill-word/trailing-input parsing. The separate
`glare-sleeping-depth --seed 1 --show-oracle` run was GREEN and showed C's
`Glaresleeptarget is not in a proper position for that.` actor output with
empty observer/target blocks. Both vehicles reported no normalized divergence
for seeds `1,2,3,5,8`.

The slice follows R1/R2/R4, R5e, and R5c: the C message table and registered
surface remain authoritative, the actual `do_action` target and position paths
were verified, no target behavior was invented, and shared gates remain owned
by their established behavior classes.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/glare-depth.txt` and
  `glare-sleeping-depth.txt` with six annotated direct cases and named peers.
- Added `docs/fidelity/depth/glare.tsv` with ten explicit rows.
- Added `pkg/session/glare_test.go` to pin the C command and social metadata.
- Local gates passed on the PR branch: `make fidelity-depth` — 1,878 total /
  1,821 proven-or-delegated / 16 blocked / 41 excluded; `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (0 issues),
  `gofumpt -l .`, and `git diff --check`.

PR #896 was opened as `glm/depth-glare`; hosted lint and security passed, but
hosted test failed only at the unrelated pre-existing
`TestMagAffects_Sleep` retry loop. It is intentionally left open and must not
be merged unless its checks become green through an appropriate external
resolution. This handoff is the durable record for advancing to `goto`.
