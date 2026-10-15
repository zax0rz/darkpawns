# DeepSeek — setup streams that could not be compared (draw-parity fixture class)

Date: 2026-10-15 · Branch: `deepseek/draw-parity-setup-symmetry` (off `origin/main`
`f1bf62126`) · Rules: R3a, R3b, R5c, R5g, R5h

## The finding

`-draw-parity` compares the two draw logs **from process start**. A scenario whose
`[setup:oracle]` and `[setup:port]` run a different number of post-entry commands
therefore desynchronizes the two streams *before the first probe step*: every
probe draw then lands on a different stream position, the probe's RNG state is
drifted, and its byte-green verdict is a coincidence green (R3a, R5h). Setup
output is drained, so nothing else in the run shows it.

**Fourteen** of the 1,171 seed-1 scenarios had asymmetric primary setups, and all
fourteen are DRAW rows in `finish-baseline-draw-parity`. Two shapes account for
them, and both are leftovers of the pre-rewrite navigation dance — the fixture
that used to walk the character from the temple, later replaced by `recall` /
`goto`. `observation-roomflags`' own header records the transition ("where the old
actor walked to (w, s from the temple)").

| Shape | Scenarios | Draw delta |
|---|---|---|
| oracle `1 / look / recall`, port `1 / recall` | `look-start-room`, `object-doors`, `object-give-mob`, `object-give-nobody`, `object-inventory`, `object-door-messages` (3 clients) | C +1 |
| oracle `1 / look`, port `1 / w / s` | `observation`, `position-gates`, `security-gates`, `act-messages` (3 clients) | Go +1 |
| oracle `1 / look`, port `1` | `observation-roomflags` | C +1 |

## The fix

`[setup:port]` now runs the same post-entry commands as `[setup:oracle]`: the
vestigial `look` is added where the oracle has it, and the stale `w / s` walk is
replaced by the `look` the oracle actually issues. The oracle's own setup is
untouched, so the reference transcript and reference stream do not move; only the
port's setup is brought back in line.

The invariant is written up in `cmd/dp-oracle-diff/README.md` → *Fixture rule: the
setups must draw alike*, with the repair recipe (make `[setup:port]` mirror the
oracle and re-check `no normalized divergence` + `draw-parity MATCH`).

## Evidence (seed 1, reference oracle `bd6c548c…`)

Before = serial `--scenario <name> -draw-parity` on `origin/main`; after = the same
command on this branch. All eleven keep `result: no normalized divergence`.

| Scenario | Before first mismatch | After |
|---|---|---|
| `look-start-room` | `stream=64813 c=Number(0,3) = 1 go=(log ends) c-step=probe step 2: look sign` | `MATCH entries=31313 stream=64814` |
| `object-doors` | `stream=4059 c=Number(0,3) = 1 go=Number(0,2) = 0 c-step=warmup` | `MATCH entries=4108 stream=4108` |
| `object-give-mob` | `stream=4076 c=Number(0,3) = 2 go=(log ends) c-step=probe step 1: give sword dolkvin` | `MATCH entries=4067 stream=4077` |
| `object-give-nobody` | `stream=4060 c=Number(0,3) = 0 go=(log ends) c-step=probe step 1: give sword nobody` | `MATCH entries=4061 stream=4061` |
| `object-inventory` | `stream=4061 c=Number(0,3) = 1 go=Number(1,101) = 48 c-step=probe step 2: open pack` | `MATCH entries=4075 stream=4075` |
| `object-door-messages` | `stream=4063 c=Number(0,3) = 2 go=Number(1,6) = 5 c-step=setup` | `MATCH entries=4103 stream=4103` |
| `observation` | `stream=4059 c=Number(1,6) = 2 go=Number(0,3) = 1 c-step=setup` | `MATCH entries=4096 stream=4096` |
| `position-gates` | `stream=64819 c=(log ends) go=Number(0,3) = 0 c-step=after the last recorded step go-step=probe step 8: north` | `MATCH entries=31318 stream=64819` |
| `security-gates` | `stream=64816 c=(log ends) go=Number(0,3) = 3 c-step=after the last recorded step go-step=probe step 5: todo` | `MATCH entries=31315 stream=64816` |
| `act-messages` | `stream=64811 c=Number(0,18) = 17 go=Number(0,3) = 3 c-step=setup` | `MATCH entries=42394 stream=75895` |
| `observation-roomflags` | `stream=4064 c=Number(0,3) = 1 go=(log ends) c-step=probe step 4: exits` | `MATCH entries=4065 stream=4065` |

The draw boundaries make the mechanism exact, not inferred. `object-give-nobody`,
whose oracle setup is `1 / look / recall` and whose port setup was `1 / recall`:

```
cbound  setup 4060   probe 1 (give sword nobody) 4061
gobound setup 4059   probe 1 (give sword nobody) 4060
```

C's setup holds one more entry, and the extra entry is the `Number(0,3)` at entry
index 4059 — the `look` command's leading hide roll. Entry-by-entry the logs agree
until Go's ends, which is why the reported first mismatch sits at the last probe
step rather than at the setup boundary.

## Rows this class does *not* cover — the three remaining asymmetric setups

`house-depth`, `spec-proc-key-seller` and `spec-proc-key-seller-non-owner` are also
DRAW rows with asymmetric setups, but their asymmetry is the *login flow*, not a
stray command: the oracle reattaches to a pre-seeded character (`Testor /
testpassword / <ENTER> / 1` — zero creation draws) while the port creates one
through the full creation dance. Their setups contain **zero** post-entry commands
on both sides, so there is nothing to mirror; the offset comes from the creation
path itself. They cannot be fixed in the corpus and need the probe-boundary
comparison planned for DP-1427 v2 (compare the streams from the probe boundary
rather than from process start).

## Gates

See the PR body. `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`,
`go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`,
`make fidelity-depth`, `make fidelity-units`, `make string-census`,
`gosec -severity=high -quiet ./...`, `go test -race ./pkg/game/...`.

Corpus note: no census was launched from this branch — the batched census
`dp-1427-v2-combined-draw` (Claude) held the census lock for the whole session, so
`scripts/census.sh start` would have refused. Evidence is the same harness run
serially per scenario; the corpus-level confirmation belongs to the next batch.

