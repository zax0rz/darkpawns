# Tier B target review

This records the second pass over ranks 6–17 of the round-one blueprint. Each
kept code target has its own commit, local gates, targeted census, and one-byte
probe in `telemetry.tsv`. The round-one blueprint itself is part of open PR
#2275, so this review is kept here instead of editing it.

| Rank | Target | Disposition | Evidence or reason |
|---:|---|---|---|
| 6 | `World.doSimpleMove`, `World.doSimpleMoveMob` | Kept | Extracted the player and mob arrival Act branches; commit `24d6ddb47`. Four movement scenarios were CLEAN; the arrival-byte probe was caught. |
| 7 | `CmdShoot` | Kept | Extracted target gates; commit `988e7db11`. All 19 listed shoot scenarios were CLEAN; the refusal-byte probe was caught. |
| 8 | `DoBackstab`, `DoCircle` | Kept | Separate outcome helpers in `bf8f18351` and `de38f2ad3`. Their real scenario sets were CLEAN. Both guard probes stayed green and are in `findings.md`; draw and return order stayed at the original call point. |
| 9 | `DoKujiKiri` | Kept | Extracted the post-precondition body in `b0798dea9`; `rin-depth` was CLEAN and caught the message probe. |
| 10 | `DoScrounge` | Kept | Extracted the sector outcome in `c935a561f`; all ten scrounge scenarios were CLEAN and caught the failure-message probe. The canceled unmodified setup attempt is documented in telemetry. |
| 11 | `World.HandleDeath`, `handleMobDeath`, `handlePlayerDeath`, `World.AwardMobKillXP` | Kept / probe-ineligible | Kept the mob-death tail and solo/group XP/gold blocks in `be2ff3210`, `7776f8462`, `f103d9298`, and `f9dbad601`. `handleMobDeath` and `handlePlayerDeath` have no direct player-output literal in the frozen population, so they had no valid one-byte probe target. |
| 12 | `World.PointUpdate`, `World.decayObject`, `World.CheckIdling` | Kept / probe-ineligible | Kept the vnum-18 decay block and the two idle branches in `49569e87a`, `ffe000b0e`, and `d38d199e9`. `PointUpdate` has no direct player-output literal in the frozen population; its downstream output paths are handled by their owning functions. |
| 13 | `processCombatPair`, `performOneHit`, `takeDamageFrom`, `getMinusDam` | Kept / probe-ineligible | Extracted the damage-position message switch from `takeDamageFrom` in `b59832762`, using the ambush call path. `processCombatPair`, `performOneHit`, and `getMinusDam` have no direct output literal suitable for the required probe. The initial backstab fixture mismatch and corrected ambush census are documented in telemetry and `findings.md`. |
| 14 | `executeZoneResetLocked`, `NewWorld`, `NewMob` | Skipped: probe-ineligible | None contains a direct player-output literal under the frozen population rule; these are reset/constructor paths with no same-function output byte to mutate. |
| 15 | `World.mobileActivityForMob`, `MpGreetActor`, `specCleric`, `specStableboy`, `specRoach`, other spec procs | Kept / skipped by named disposition | Kept scavenger pickup (`6770fdc60`), stableboy purchase (`d0504ce71`), roach feeding (`50e79f78f`), and cleric combat (`0657340d8`). `MpGreetActor` and `MpSound` are in `pkg/game/mobprogs.go`, which open PR #2271 edits. `specSummoner`, `specCouch`, and `specCastleGuardEast` have no exact live scenario/call path confirmed in the round-one blueprint. |
| 16 | `sendSkillResult`, `CmdListSkills` | Skipped: probe-ineligible | `sendSkillResult` emits message values supplied by `SkillResult`; it has no direct output literal for a one-byte probe. `CmdListSkills` is only called by `CmdLearn` with no arguments, and `CmdLearn` has no non-test caller. The live `practice` route uses `pkg/session/practice.go`, not this function. |
| 17 | `performAct`, `Dream`, `huntVictim`, `RaceHelpEntries`, `World.doGenTog` | Kept / skipped by named disposition | Kept act substitution (`647e9615f`), level-dream branch (`cfe4d1684`), and hunt evasion (`d0d5798d0`). The first two probes and the hunt probe stayed green; see `findings.md`. `RaceHelpEntries` is static catalog text with no executable control-flow block to extract. `doGenTog` was already kept and reviewed in inherited PR #2275 (`5f1f4e5cb`). |

No target was labeled `Skipped: unsure`: each unattempted subtarget has a specific
measurement limitation, open-PR exclusion, missing live call path, or static
data-only reason. No Tier B target was abandoned.
