# The harness ran the branch players never reach

2026-11-06. Field note by Claude, from reviewing MiMo's PR #2311, which gives the Go side of every census scenario a player store. Census run: `~/Archives/darkpawns/oracle-runs/2026-11-06/mimo-always-store` at `1de14d94d`. Issue: DP-1479.

## What happened

Until this PR, most census scenarios booted the Go server with no player store (`DP_ALLOW_NO_DB=1`). Production always has one. Code that branches on whether a store exists therefore ran one way under the census and the other way for players.

`last` is one such branch. In `pkg/session/wiz_system.go`, `cmdLast` has two:

- **No store** (`onlinePlayerForLast`, `formatOnlineLast`): it builds C's line, `[%5ld] [%2d %s] %-12s : %-18s : %-20s` (`src/act.wizard.c:1846-1849`), from the live session's class, host and connection time.
- **Store**: it prints `[%d] [%2d] %-12s : Level %d`, a format C never had. There's no class, no host, no time, and the level appears twice.

`last-depth` was green for weeks. It was proving the branch players never reach. Once the census had a store, the scenario went red on its first run, deterministically.

## Why it's worth recording

This is a different way for a test to lie from the ones in the [seven-kind seed](../drafts/2026-11-06-tests-that-lie-section-seed.md). The test wasn't weak and its expectation wasn't wrong; the scenario checked the right bytes against the right C. The harness's runtime decided which code ran, and it chose the code that existed to pass the scenario. The no-store branch even says so in its comment: "The no-database runtime has no offline pfiles, but its live sessions still carry the character identity". Someone made the harness's configuration agree with C and left production's alone.

R5i ("a fixture that silences a surface needs a scenario that unsilences it") covers this once you count the runtime as a fixture. The empty store silenced the store branch of every command that reads saved players. The PR's census turned up one live case out of 1,847 scenarios. The siblings, such as `players` sorting by name where C walks `player_table`, are on DP-1479.

## The fix needs a decision, not just code

C prints the host saved in `char_file_u`. Go's `db.PlayerRecord` has no host field, so a faithful fix is a persistence change, and that's Zach's to decide. Meanwhile `last-depth` is pinned as an expected divergence citing DP-1479, so it fails if the divergence changes shape and goes stale when it's fixed.
