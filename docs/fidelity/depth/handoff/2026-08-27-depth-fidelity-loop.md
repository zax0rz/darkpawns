# Dated Handoff: 2026-08-27 (depth-fidelity loop)

This wave started from a clean, pulled `main` and confirmed the handoff
frontier with `make fidelity-depth`: **476 total, 444 proven/delegated, 22
blocked, 10 excluded** (95.3% actionable). Three planned rounds then ran in
order, with one post-audit corrective proof before handoff. All four PRs were
self-merged only after their GitHub checks were green.

## Round 1 — god-set stat/position vehicle (PR #669)

Branch: `glm/depth-godset`; merged as `83a65e9a0`.

- `godset-vehicle` was RED on pre-fix `main`: the C oracle accepts `set Mordecai
  wis 0` and emits `Mordecai's wis set to 0.`, while the old Go `cmdSet` treated
  it as unknown; C rejects `set Mordecai position 3` with `Can't set that!`.
- C source inspection (`src/act.wizard.c`, `src/act.comm.c`) established the
  stat field set and C's mortal/high-level clamp ranges, plus the WIS/INT zero
  stupid-speech gate. Go `cmdSet` now matches those stat acknowledgements and
  clamps, while preserving C's unsupported-field response; no `src/` or oracle
  file was edited (R1/R4/R5e).
- `say.stupid-gate` is now oracle-green through the WIS-zero vehicle.
- The four `stand/sit/rest/sleep.stunned-or-worse` arms are unit-green through
  `TestPositionCommandsStunnedOrWorse`, proving exact actor/room bytes and
  resulting positions. C has no usable `set position` field, so this is the
  deterministic D4 state proof required by R4/R5e.

## Round 2 — cheap stragglers (PR #670)

Branch: `glm/depth-cheap-stragglers`; merged as `21cdd9ff1`.

- `recite.wrong-type` was confirmed RED: C says `You can only recite scrolls.`
  for a bread object; Go had invented flexible non-scroll/spell-value behavior.
  Go now follows C's `ITEM_SCROLL` gate (R1/R4), and the case is oracle-green.
- `reply.no-arg` was proven with a told peer followed by bare `reply` and is
  oracle-green.
- `where.immort-zone-arg` was proven with the named-argument fixture. R5e
  source tracing showed that the legacy row name was misleading: C's optional
  argument searches visible character names, not zones. Go now consumes and
  matches that argument and emits the C-formatted targeted listing.

## Round 3 — pulse-driven info, then pivot to `consider` (PR #671)

Branch: `glm/depth-pulse-info`; merged as `85e33d8be`.

- `info-pulse-variants` clears the indoor flag and pumps `~dpclock pulse 630`
  before `time` and `weather`. Seeds 1–5 matched C and Go, advancing the MUD
  clock and exercising rainy/cloudy/lightning weather text. The time and
  weather rows are now oracle-green-multiseed under R3.
- The first new command-family manifest, `docs/fidelity/depth/consider.tsv`,
  opens `src/act.informative.c:2330-2431` at D1–D3: no argument, not found,
  self, target level/description, and private output. A spawnable horse mob
  vehicle was used; seeds 1–5 matched, including the peer-silent private
  audience. The existing Go implementation was already faithful, so this was
  pure coverage with no source change.

## Post-audit correction — `wake.target-bad-shape` (PR #672)

The final audit found that the initial god-set round had converted the four
stunned position arms and `say.stupid-gate` but had not converted the explicit
`wake.target-bad-shape` row. A deterministic unit vehicle was added on branch
`glm/depth-wake-bad-shape`, merged as `6f76a9efe`.

C `src/act.movement.c:858-859` emits `$E's in pretty bad shape!` when a target
is below `POS_SLEEPING`, then returns without changing the target. The focused
`TestPositionCommandsMountAndWake` subtest drives a female target at
`POS_STUNNED`, proves exact `She's in pretty bad shape!\r\n`, and proves the
position is unchanged. The Go behavior was already faithful; this PR adds the
missing D2 state proof and changes the owning manifest row to `unit-green`
(R1/R4/R5e).

## Final frontier

Final `make fidelity-depth`:

```text
Cases: 481 total, 460 proven/delegated, 11 blocked, 10 excluded
Actionable completion: 460/471 = 97.7%
```

The remaining blocked rows are deliberately left blocked with an explicit
owning manifest/next vehicle. The requested deep-engine backlog was not
attempted:

- `combat-entry.tsv`: `kill.immortal-postdeath-menu` — deferred extraction and
  session menu return; `hit.charm-master` — charmed attacker/master vehicle.
- `comm.tsv`: `tell.linkless`, `tell.writing` — linkless descriptor and board
  writing-state vehicles.
- `combat-entry.tsv`: `assist.cant-see`, `assist.mob-helpee-pers` — invisible
  opponent and mob-helpee audience vehicles.
- `door.tsv`: `door.lock-unlock-pick` — keyed, pickproof, and skill-gated door
  vehicle.
- `use.tsv`: `use.effect` — wand/staff `mag_objectmagic` vehicle.
- `info.tsv`: `score.state-variants` — affect, position, mount/pet, tattoo, and
  naked/armed state matrix.
- `object-magic.tsv`: `objmagic.sleep-entry-gates` — spell-entry gate vehicle.
- `position.tsv`: `wake.cant-wake-aff-sleep` — save-gated magical-sleep vehicle.

These owners are manifest/vehicle owners for the next depth round, not claims
that those deep rows were completed here.
No `src/` or `darkpawns-c-oracle/` files were changed.

## Verification

On the merged correction branch, all required local gates passed:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` — `0 issues.`
- `gofumpt -l .` — no output

GitHub checks for PR #672 also passed: test, lint, security (`govulncheck` and
`gosec`). The first security attempt failed only on transient
`sum.golang.org` HTTP/2 verification errors; the failed job was rerun and
passed. Final repository state is clean `main` at `6f76a9efe`.
