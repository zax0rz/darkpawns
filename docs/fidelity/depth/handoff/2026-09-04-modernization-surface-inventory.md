# Lane B surface-inventory clinic handoff — 2026-09-04

This handoff opens the source-order Lane B sweep required by the merged
fidelity + modernization queue. It uses the 2026-09-03 whirlpool clinic recipe:
fresh creation-room actors (mortal 8162, God 8105), asynchronous output flushed
by `writeLoop`, and the C oracle as the adjudicator. The rows are handled in
`docs/fidelity/depth/surface-inventory.tsv` file order.

## Disposition contract

The inventory's blanket reason `terminal-surface-audit-2026-09-04` is only a
round marker, not evidence. During this round every blocked row will receive a
family-specific attempt token and a note naming the C scope and residual owner.
The final inventory must contain zero instances of that blanket token.

Each row ends in one of three valid states:

- `proven` / `oracle-green-multiseed`: the Go transcript matches the C oracle
  across the required seeds;
- `excluded-with-C-reason`: C source and command reachability prove the surface
  is not a player-facing call path under R2/R4/R5e; or
- `blocked`: the clinic attempt is recorded per family and the remaining
  surface is explicitly owned for later proof. Blocked is an expected and
  correct disposition for admin, immortal-gated, level-gated, lifecycle, and
  other surfaces the current corpus cannot safely prove. No exclusion is
  asserted without C evidence.

This sweep does not invent commands, alter the C oracle, change save format, or
promote a no-diff spot check into depth proof. The separate modeled-depth
denominator remains 4,758 cases (4,653 proven/delegated, 54 blocked, 51
excluded). The weighted source-order denominator is 70 rows / 4,926 units
(8 proven-already, 61 blocked, 1 excluded-with-C-reason).

## Round boundary

The handoff is committed before the inventory proof fields are changed. The
full post-Phase-1 `make oracle-regression` run is the final corpus gate for the
branch; its exact scenario count, pass/infra/timeout result, and wall-time will
be appended to the dated terminal handoff after completion.
