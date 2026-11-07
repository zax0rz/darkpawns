# DP-1493: Divine Intervention angels enter combat in Go

## Root cause: C's macro tests MEMORY

C defines `MOB_AGGR_TO_ALIGN (MOB_AGGR_EVIL | MOB_AGGR_NEUTRAL | MOB_AGGR_GOOD)` at `src/mobact.c:52`. Those constants are bit numbers 8, 10 and 9 (`src/structs.h:255-257`), so the expression is 11: `MOB_MEMORY` (`src/structs.h:258`). `MOB_FLAGGED` takes a bit number (`src/utils.h:211,233`). Consequently the aggression block at `src/mobact.c:198-231` is entered only for `MOB_AGGRESSIVE` or `MOB_MEMORY`, not for an alignment-only mobile.

Go instead entered on `aggressive || aggrEvil || aggrGood || aggrNeutral`. The Divine Intervention angels have only their alignment aggression flag, not AGGRESSIVE or MEMORY, so C never starts their autonomous attack. The fix ports C's effective MEMORY gate; it does not add a charm gate.

## Spawn and retained oracle witness

C's `spell_divine_int` chooses mob 85 or 86, applies `AFF_CHARM`, and adds a follower (`src/spells.c:1170-1216`). Go follows that sequence in `castDivineInt` (`pkg/spells/affect_spells.go:3546-3576`). Neither state is missing. C has no charm gate around aggression, so a charmed AGGRESSIVE follower must still attack.

The prototypes are `lib/world/mob/0.mob:367-393`: mob 85 has mask 20745 and mob 86 has 21001. Each has SPEC, ISNPC, one alignment flag, HELPER and NOSUMMON; neither has AGGRESSIVE or MEMORY.

The retained C witnesses show successful casts and two angel arrivals without combat in the following pulse:

- Potion: `~/Archives/darkpawns/oracle-runs/2026-11-07/kimi-dp1493-divine-angel/census-dump/spellmatrix-potion-olc-03-depth.txt`, lines 13-16 and empty pulse block at line 41.
- Wand: the same run's `census-dump/spellmatrix-wand-olc-08-depth.txt`, lines 732-735 and empty pulse block at line 760.
- Staff: the same run's `census-dump/spellmatrix-staff-olc-13-depth.txt`, lines 380-383 and following pulse blocks without angel combat.

## Whole-block parity (R5b)

The fix also ports two in-block differences:

1. The first alignment clause is `!AGGR_EVIL && AGGR_NEUTRAL && AGGR_GOOD`, which hits any victim alignment (`src/mobact.c:217-219`).
2. The alignment arm and AGGRESSIVE arm each call `hit()` (`src/mobact.c:222-228`). A matching AGGRESSIVE mobile therefore makes two calls in one iteration. `hit()` always consumes its swing draw before `damage()` enrolment/refusal (`src/fight.c:1763-1869`), so Go calls `mobFirstHit` independently in both arms.

CAN_SEE, NOHASSLE, WIMPY, protection and sneak checks remain in C's order.

## World-mob class audit

Parsing every `lib/world/mob/*.mob` action mask finds 59 mobiles with an alignment-aggression flag. Twenty-three have neither AGGRESSIVE nor MEMORY and are listed in `alignment-only-mobs.tsv`. The new `mobact-alignment-only-evil-depth` scenario places world mob 85 beside an evil player and requires the pulse and subsequent look to remain free of combat.

## Unit proofs and reverts

The real `World.MobileActivity` / `mobileActivityForMob` tests cover:

- `TestMobileActivityDivineAngelsStayOutOfCombat` and `TestMobileActivityAlignmentOnlyMobDoesNotAttack`: alignment-only mobs do not attack. Reverting to the old alignment gate failed both, including `{attacker:a Dominion Angel defender:Victim}`.
- `TestMobileActivityMemoryEnablesAlignmentAttack`: adding MEMORY enables the matching attack. Removing MEMORY from the outer gate failed with zero combat starts.
- `TestMobileActivityCharmedAggressiveFollowerStillAttacks`: charm does not suppress AGGRESSIVE. Reintroducing a charm gate failed with zero combat starts.
- `TestMobileActivityNeutralGoodWithoutEvilHitsAnyAlignment`: the first clause hits an evil victim. Removing that clause failed with zero combat starts.
- `TestMobileActivityAggressiveAlignmentMatchHitsTwice`: matching AGGRESSIVE makes two hit calls. Suppressing the second call failed with one start instead of two.

## Oracle scope

The original targeted run is `~/Archives/darkpawns/oracle-runs/2026-11-07/kimi-dp1493-divine-angel`. The corrected 52-scenario run is `~/Archives/darkpawns/oracle-runs/2026-11-07/kimi-dp1493-memory-aggression`: 49 PASS, 1 STALE, 2 EXPECTED_UNSTABLE, and zero FAIL/INFRA/TIMEOUT. The non-PASS rows are the previously pinned/run-varying spell-matrix scenarios. `mobact-alignment-only-evil-depth` passed, and the Divine Intervention divergence is cleared.

The corrected five-seed claims run is `~/Archives/darkpawns/oracle-runs/2026-11-07/kimi-dp1493-memory-aggression-claims`: 6033 PASS, 1 EXPECTED, 10 EXPECTED_UNSTABLE, 5 STALE and 4 FAIL across 6053 pairs. The three Divine Intervention matrix scenarios retain their non-FAIL verdicts at every seed (1, 2, 3, 5 and 8), with no angel-combat divergence. The four unrelated failures are Go warmup setup refusals in `dispel-evil-skill-message-hit-depth@8` and `eat-werewolf-corpse@8`, plus existing third-person swordplay-message differences in `spellmatrix-wand-olc-01-depth@2` and `spellmatrix-staff-olc-01-depth@8`.
