# Brief: DP-1463, `clan enroll` leaves out the offline applicant (suggested agent: MiMo)

2026-11-06, by Claude for Zach to hand out. **Diagnose this first.** The fix may be one line, or it may sit in the persistence layer, which is stop tier. The rule for which is below.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Work in `~/wt/<agent>-dp-1463`, on branch `<agent>/dp-1463-clan-offline-applicant` cut from `origin/main`.

## Ground truth (grepped by Claude at `origin/main`, 2026-11-06)

**C: the applicant list.** `do_clan_enroll` with no argument prints the applicant list (`src/clan.c:322-346`):
- It walks `player_table` **in index order**: `for (j = 0; j <= top_of_p_table; j++)`, at `:326`.
- For each entry:
  - if `is_playing(name)` finds the player, it tests the live character (`:328`);
  - otherwise it calls `load_char` and tests the saved clan and rank (`:339`).
- `is_playing` (`src/utils.c:755`) matches only `CON_PLAYING` descriptors, so a quit or linkless player is read from file.
- **Note:** C compares against `GET_CLAN(ch)`, the caller's own clan, even for an immortal, who names a clan (`immcom`, `:307-310`); the comparisons are at `:330` and `:340`. Go compares against the named clan `c.ID`. The scenario uses a clan member, so both agree there. List this as a sibling in your report; don't fix it unless a scenario proves it.

**Go: the applicant list.** `pkg/game/clan_membership.go:31` onward:
- it lists the players in `w.players` (a map, so in random order);
- then it lists stored players not found in `w.players`, from `w.StoredPlayers()` (`pkg/game/persistence_seam.go:138`).

So Go already has a fallback, and the bug is that it doesn't print `Clanoffline`.

**The failing scenario:** `clan-offline-applicant-depth`. It's pinned as an expected divergence that cites DP-1463. Its warmup ends with:
1. `force Clanoffline clan apply Wolves`
2. `teleport Clanoffline 8004`
3. `send:applicant quit`
4. `~dpclock pulse 1`

Then the probe runs `force Clanmate clan enroll`.

**Where to look first:**
1. **`StoredPlayers` returns `nil, nil` when `w.PlayerStoreList` is nil.** If the oracle server never wires it, the fallback is silently empty.
2. **Quit may not save the applied clan and rank** before the list is read.
3. **The quit player may still be in `w.players`** (linkless or pending removal), so the online branch skips it and the stored branch skips it too. Compare DP-1423.

**A likely sibling:** DP-1464 (`show player` and `show rent` can't find a just-saved player) may share the root cause.

## Decide which case it is

- **The cause is in the clan listing logic** (cases 3 or similar, inside `pkg/game`): fix it.
  - Also make the listing walk in C's order: online players are tested in player-table order, not map order. Find how other Go ports reproduce `player_table` order, and reuse that.
- **The cause is the store wiring or save timing** (cases 1 or 2): **stop and report, and change nothing in persistence.** That's stop tier, and it probably fixes DP-1464 too, so Zach decides its scope.

## Prove the fix (the listing case)

- `clan-offline-applicant-depth` stops matching its pin: it goes STALE.
- Move its `clan.tsv` row back from `blocked`, at the depth and seeds it claims, and drop the `divergence: DP-1463` note.
- Run `make expected-divergences`.
- Revert the fix, and the scenario must go red (R5h).
- **Census:** targeted, through `scripts/census.sh`: this scenario plus every `clan-*` scenario.

## Rules

- **Gates:** run each on its own and check its exit status:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
  - `golangci-lint run ./...`
  - `make fmt`
  - `make deadcode-ratchet`
  - `make fidelity-depth`
  - `make expected-divergences-check`
  - `make db-migrate-test`, if you change an exported `pkg/game` symbol.
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`.
- **No outside help:** don't invoke another agent, model or paid service.

Open a PR if you fixed it, ending its description with "Decisions for Zach". Either way, stop and report:
- the root cause, with the evidence;
- which case it was;
- the census run directories;
- the siblings you found.

Claude reviews in Zach's session.
