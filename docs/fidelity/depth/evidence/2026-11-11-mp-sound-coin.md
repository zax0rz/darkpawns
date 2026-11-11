# `mp_sound` coin-arm audit — 2026-11-11

## Finding and class audit

C's six randomized `mp_sound` cases use `if (number(0,1))` to select the
first line for petitioner, carpenter, town crier, beggar, minstrel, and KD
recruiter (`src/mobprog.c:526-564,569-575,593-604`). C `number()` returns the
integer conversion of its uniform draw (`src/utils.c:52-63`), so the condition
is truthy for every nonzero result. Go had `mobProgNumber(0,1) == 0` at all six
sites, reversing both outcomes whenever the C-compatible draw was 1 (and for
the rare `to+1` boundary value).

The blocked-row note had suspected PF-090's float-rounding boundary. That was
not the cause: `mobProgNumber` delegates to `dprng.Number`, whose existing
boundary test preserves C's `to+1` result. The defect was the opposite branch
predicate in `MpSound`; this fix changes all six predicates to `!= 0` and does
not change the draw stream. The two unit tests that forced zero while claiming
to exercise C's first arm now force one.

The class audit is the complete set of randomized cases in C `mp_sound` and Go
`World.MpSound`: vnums 8066, 8067, 8068, 8071, 8074, and 16300. Fixed-message
cases 8069, 8072, 8079, 14202, 8059, and 8023 contain no coin predicate and are
unchanged by this fix.

## Live transcript proof

Six scenarios each spawn exactly one randomized sound mob in a sealed room,
with `quiet-mobs` removing unrelated activity; C's `mobile_activity` reaches
`mp_sound` on its sound roll (`src/mobact.c:145-147`). Each C transcript shows
the seeded character entering, the requested mob in the room, and both random
lines across seeds 1, 2, 3, 5, and 8. No setup command was refused. Every run
used `--draw-parity`; all six scenarios had zero draw divergence at each seed:

- Seed 1: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-mobprog-coin-branches`
- Seed 2: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-mobprog-coin-branches-s2`
- Seed 3: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-mobprog-coin-branches-s3`
- Seed 5: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-mobprog-coin-branches-s5`
- Seed 8: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-mobprog-coin-branches-s8`

The family revert proof is
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-mobprog-coin-revert-proof`.
For that snapshot, all six predicates were temporarily restored to `== 0` and
the six scenarios ran at seed 2. All six transcript rows failed, while all six
draw streams still matched. The predicates were restored to `!= 0` immediately
after census registration.
