# Brief for DeepSeek (via Cline): DP-1480, players in a room come back in random order

Linear: DP-1480. Rulebook: R3 (determinism), R5b and R5c (find one, find the class), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

Work in a worktree: `git worktree add ~/wt/cline-dp-1480 -b cline/dp-1480-room-player-order origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

## Ground truth (verified by Claude at `origin/main` 15b7bd6ad)

**The symptom.** `attitude-loot-gold-depth` and `attitude-loot-switched-output-depth` pass on some attempts and fail on others, on any base. In the failing attempt (`~/Archives/darkpawns/oracle-runs/2026-11-06/mimo-always-store/seed1/initial-logs/attempt.combined/attitude-loot-gold-depth.attempt1.log`), C's aggressive watcher kills the observer, Goldwatch, while Go's watcher kills the actor. The aggressive mob picked a different player.

**C** (`src/mobact.c:198-233`): an aggressive mob walks `world[ch->in_room].people` from the head and attacks the **first** eligible player (`!found` stops the loop). `char_to_room` front-inserts, so the list is newest arrival first.

**Go** (`pkg/game/mobact.go:259` onward): the same scan loops over `w.GetPlayersInRoom(ch.RoomVNum)`. `GetPlayersInRoom` (`pkg/game/world.go:924`) builds its slice by ranging over the `w.players` **map**, so with two players in the room Go picks either one, run to run.

**The port already knows this pattern.** `COrderedRoomMobs` (`pkg/game/world.go`, after `GetMobsInRoom`) and `cRoomPeople` (`pkg/game/actor_specials_rooms.go:33`) sort by `GetRoomEntrySequence()`, newest first, with name as the tie-break. `COrderedRoomMobs`'s comment says it outright: "GetMobsInRoom walks a map and must not be used where the order is observable." Players never got the same treatment.

## The fix: order the player list at its source (decision made)

`GetPlayersInRoom` has about 70 call sites. Don't fix them one at a time. Make `GetPlayersInRoom` itself return C's order: `RoomEntrySequence` descending, then name ascending, the same comparator as `COrderedRoomMobs`. Do the same for `GetPlayersInRoomScriptable` (`pkg/game/world_scriptable.go:15`). Update both doc comments, citing `char_to_room`'s front insert in `src/handler.c` (grep the line yourself).

- **Locking:** sort after you release `w.mu`. Read each player's sequence through its accessor; don't read the raw field.
- **Leave `GetMobsInRoom` as it is.** Its callers already chose between it and `COrderedRoomMobs`.
- **Other `range w.players` loops:** list every loop in `pkg/` that filters `w.players` by room itself instead of calling `GetPlayersInRoom`. Put them in `docs/fidelity/evidence/dp-1480/room-scans.tsv` with columns `file:line`, `function`, `order observable? (yes/no)`, `why`. Fix the ones marked yes by calling the ordered function. Leave the no rows alone.

## Proof

1. **A unit test that can fail.** Two players enter a room in a known order, then an aggressive mob scans. Assert it attacks the newest arrival, and run the scan 200 times in the test, so map randomness can't let a wrong implementation pass by luck. Revert proof: with `GetPlayersInRoom` restored, the test fails. Paste the failure.
2. **The two attitude-loot scenarios are stable.** Run each 10 times at seed 1 and once each at seeds 2, 3, 5 and 8. Every run must report no divergence. Paste a one-line tally.
3. **A full census, because this changes the order at every room scan:** `scripts/census.sh start --name cline-dp1480-room-order`, not `--combined`. Then `scripts/census.sh wait --run <dir>`; don't poll. For every changed row against the newest full census on main, say whether it's a fix (FAIL→PASS), a flake, or a regression. Stop and report any regression; don't pin it, opt it out or edit the scenario to make it pass.
4. **Draw parity:** if any scenario moves on draw counts, report it with its draw logs (R3).

## Rules

- Cite C only from `src/`, grepped in this session, never from memory or from `darkpawns-c-oracle/`.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`. If you think a rule is missing, propose its exact text in the PR description.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status. Never chain them with `;`: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt` (gofumpt), `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`. If an exported `pkg/game` symbol changes, also run `make db-migrate-test`.

## Done means

A PR against `main` containing:
- the ordered functions;
- the scans TSV;
- the unit test with its revert proof;
- the scenario tally;
- the census run directory with its changed-row attribution;
- a **Decisions for Zach** section.

Then stop and report. Claude reviews in Zach's session, and Zach merges.
