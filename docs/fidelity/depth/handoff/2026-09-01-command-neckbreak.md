# Depth-fidelity handoff — `neckbreak`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R4, R5b, R5c, R5e

## Frontier

This session began on a clean, pulled `main` after rereading
`docs/fidelity/DEPTH_TESTING.md` and the `mute` handoff.  The starting
frontier was 2,616 cases: 2,546 proven/delegated, 22 blocked, and 48
excluded (99.1% actionable).  After `neckbreak` was merged, fresh `main`
reports:

```text
Cases: 2634 total, 2564 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2564/2586 = 99.1%
```

The queue correction from the prior handoff remains in force: `mail` is
already claimed by the existing `do_not_here` family proof; `muhaha` and
`mumble` are social-family claims; and `murder` is owned by the existing
`do_hit`/combat-entry proof.  The next actually unclaimed table row is
`news` at `src/interpreter.c:565` (`POS_SLEEPING`, `do_gen_ps`,
`SCMD_NEWS`).  `newbie` at line 566 follows it.

## C path and RED → GREEN proof

The source registration is `{ "neckbreak", POS_FIGHTING, do_neckbreak, 0,
0 }` at `src/interpreter.c:564`.  The C call path in
`src/act.offensive.c:1295-1376` was mapped before implementation changes:

1. Skill knowledge and wielded-weapon gates run before `one_argument()` and
   target lookup.
2. C `one_argument()` skips fill words and retains only the first target
   token.  Visibility failure, shopkeeper protection, self-targeting, and
   `ROOM_PEACEFUL` then run in that order.
3. A mounted actor is refused; otherwise C spends 51 movement only when the
   attempt is affordable.
4. The skill roll failure emits all three set-190 miss audiences and then
   immediately calls `hit(vict, ch, TYPE_UNDEFINED)`.
5. The success arm rolls `18d(level)`, calls `damage()` with attack type 190,
   enters combat, waits three violence rounds, and improves the skill only
   after the damage/message boundary.

The first clean-main vehicles were RED on the C target/gate order, exact
lookup bytes, shopkeeper and mounted refusals, set-190 success messages, and
failure retaliation.  The success continuation also exposed a shared
`damage()` pain-message draw: C consumes `number(0,2)` after the severe-hit
line, so the common combat seam was corrected only after the draw trace
confirmed that divergence.  No `src/` or C-oracle files were edited.

## Durable proof

Added:

- `cmd/dp-oracle-diff/scenarios/neckbreak-depth.txt` — no-argument,
  missing-target, visible mob/player, self, fill-word/trailing-argument, and
  peaceful branches.
- `cmd/dp-oracle-diff/scenarios/neckbreak-wielded-depth.txt` — pre-lookup
  wielded-weapon gate.
- `cmd/dp-oracle-diff/scenarios/neckbreak-shopkeeper-depth.txt` — boot-assigned
  shopkeeper gate using mob 8003.
- `cmd/dp-oracle-diff/scenarios/neckbreak-mounted-depth.txt` — legal ride
  followed by the mounted refusal.
- `cmd/dp-oracle-diff/scenarios/neckbreak-low-move-depth.txt` — exact 51-move
  affordability branch.
- `cmd/dp-oracle-diff/scenarios/neckbreak-failure-depth.txt` — numbered miss
  audiences and immediate victim retaliation.
- `cmd/dp-oracle-diff/scenarios/neckbreak-success-depth.txt` — numbered hit,
  combat state, and three `~dpclock pulse 20` continuation pulses.
- `pkg/game/neckbreak_depth_test.go` and
  `pkg/command/neckbreak_depth_test.go` — gate order, parser boundary,
  result contract, deferred improvement, retaliation, and draw-count proofs.
- `docs/fidelity/depth/neckbreak.tsv` — 18 rows covering the reachable
  command, audience, state, damage-seam, and delegation cases.

All seven vehicles were GREEN at seeds 1, 2, 3, 5, and 8: 35/35 runs with
no normalized divergence.  The success vehicle was also inspected with the
oracle transcript and the shared draw trace.  The implementation changes are
limited to the confirmed C divergences: wrapper gate/parser ordering,
runtime shopkeeper membership, set-190 damage routing, failure retaliation,
and the shared severe-damage pain draw (R1/R3/R5e; shared seam review under
R5b/R5c).

## Gates and merge

Local gates passed on `glm/depth-neckbreak`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean

Feature PR #1038 (`glm/depth-neckbreak`) had green hosted `lint`, `security`,
and `test` checks; build/deploy were skipped by the PR workflow.  It was
self-merged under the 2026-08-27 amendment as squash commit `8faa4b99c`.

The post-merge `main` checkout/pull and frontier rerun passed with the counts
above.  The next session must start on `main`, pull, rerun
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this
handoff, then map and attempt only `news` at `src/interpreter.c:565`.
