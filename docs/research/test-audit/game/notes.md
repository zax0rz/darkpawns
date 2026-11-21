
## Post-oracle redo, batch 1 (tests 1–50, T339–T388; 2026-07-12..13)

The redo per the follow-up method: every test read in full with the
production code, C cites grepped this session (door precondition strings at
src/act.movement.c:545,627,630,633; say's INT-0 refusal at src/act.comm.c:647;
str_app 18/50 carry 280 at src/constants.c:1023-1049; group-share math
src/fight.c:661-685 with the 693 float-truncation pin; solo XP line
src/fight.c:1646; CAN_GET_OBJ at src/utils.h:544-549; social hide/
min_victim_position at src/act.social.c:53-54).

Marks: **R 49 / F 1 / C 0 / D 0.**

**F: TestConsumables_InstanceIsolation (J9/J3)** — all three halves were
vacuous: the default conditions (full=36, thirst=36) gated the drink shut
before any write, the eat arm only removed the item, and the pour resolved
one object for both shared keywords. No SetValue ever ran, so the isolation
assertions held with nothing happening. Fixed to drive real mutations
(taste decrement; conditions cleared before the sip; a distinct-keyword
pour target), honoring inventory's prepend resolution order. Proof: the
SetValue→prototype leak keeps the old body green and reds the fixed test.

R-mutation spot-checks (per the pace rule, 3 seeded-random picks):
TestPerformActSubstitutesEveryCode ($N renders the actor — red),
TestCanTakeObjUsesTakeBit (bit test → exact match — red; the first attempt,
skipping the check, stayed green because the test pins the positive arm —
recorded as a lesson), TestConsumables_InstanceIsolation (the F proof).
