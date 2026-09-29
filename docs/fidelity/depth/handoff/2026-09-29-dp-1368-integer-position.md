# DP-1368 item 5: integer position damage (R1/R4/R5c/R5h)

Addenda 3 and 4 authorize repairing the DP-515 float multiplier class. C's executed expression is `dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3` (src/fight.c:1854-1855), before backstab_mult at :1867. The shared combat.ApplyPositionDamageMultiplier helper preserves integer truncation and makes no RNG calls.

| Site | Disposition | Table proof |
|---|---|---|
| combat.CalculateDamage | Replaced float expression and removed intent rationale | Extended existing TestCalculateDamage_VictimPositionMultiplier, base 12, every position |
| TestPositionDamageMultiplier | Deleted intent NOTE; corrected base-100 golden to C | Existing golden retained |
| World.mobBackstab | Replaced float expression with shared helper | TestMobBackstab_PositionMultiplier, base 12 before backstab_mult; dead victim still rejected by damage gate |
| DoCircle | Already integer-correct; now calls shared helper to prevent splitting | TestDoCircle_PositionMultiplier, same base-12 table |

The final pkg-wide search for float64 adjacent to PosFighting/GetPosition has no hits. The only additional float code hit was mobBackstab; no other float multiplier remains. Positions dead/mortally/incap/stunned/sleeping/resting/sitting/fighting/standing produce base-12 results 36/36/24/24/24/12/12/12/12.

Literal reversion of mobBackstab to float fails its table. Literal reversion of the shared helper to float fails the CalculateDamage, golden, circle and backstab tables. Evidence is retained under ~/Archives/darkpawns/oracle-runs/2026-09-29/dp-1368-development/addendum4-{position-green,backstab-reverted,helper-reverted}.log.

The known DP-1363 TestEntryWebSocketSavedIdentityAndMenuResume flake passed the first isolated retry (addendum4-websocket-retry1.log). Full build, vet, test and lint gates passed after these changes. DP-1369 is unchanged. Candidate and multiseed census evidence follows in the final review handoff.
