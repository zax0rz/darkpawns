# DP-1371: repair the old steal branch witnesses

Tier: proof-only. This changes one scenario, its existing manifest notes and
retained proof controls; no game, harness, oracle or census code changes.

## Finding and bounded repair

The previous `steal-depth@1,2` C transcripts reach the peaceful-room refusal
for every target-present probe. That proves the refusal, not the inventory,
missing-item or zero-gold branches claimed by three rows (R5e/R5h).
`src/act.other.c:340-343` precedes all three branches.

Keep the original commands and both claimed seeds. In disposable copies of
both engines, clear only ROOM_PEACEFUL in room 8162 and make bread prototype
8010 available. Set the actor strength to 25 so carry capacity does not mask
transfer; allow 20 pulses after successful theft so WAIT_STATE cannot swallow
the following command; set the trainee gold to zero explicitly. These are
paired fixture changes, not permanent world edits or new divergences.

The required C witnesses are `Got it!` at the fill-word/trailing-input probe
(`src/act.other.c:471-485`), both missing-item refusals at the next two probes
(`src/act.other.c:400-403`), and the zero-gold message at the coins probe
(`src/act.other.c:506-524`). The zero-gold draw is retained. The missing-item
arm returns before assigning wait; one intervening 20-pulse advance is enough.

## Reproduce the controls

```sh
DP_ORACLE_BIN="$HOME/darkpawns-c-oracle/bin/circle" \
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-steal-proof-controls.py \
  --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-steal-proof-repair/controls"
```

The controls replace each claimed Go branch's bytes independently and require
an actual differential failure, then restore the exact source and re-prove it.
Build failures do not count. A separate witness reader rejects the retained old
C captures, so a plain no-diff result cannot certify branch coverage.

## Evidence and remaining boundary

Evidence root:
`~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-steal-proof-repair/`.
Old seed-1 and seed-2 C captures and their failed witness checks are retained.
New C branch checks pass at seeds 1 and 2. All three compiling mutation
triples pass 1 → 0 → 1, and the temporary game-source edits are restored.
Committed-tip targeted results are recorded in the PR and retained run summaries. The targeted set is exactly
`steal-depth`, at both originally claimed seeds 1 and 2. No other scenario or
claim pair changes. This repairs three existing claims, not a new completion
of the entire steal handler or hidden transfer state.
