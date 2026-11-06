# Brief for MiMo: the census always gives Go a player store (DP-1463, DP-1464)

2026-11-06, by Claude for Zach to hand out. Zach approved the approach: **every scenario gets a real player store**, the way C always has its player file. This is a harness change, so it's **stop tier: Zach merges**.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Note R5i: a fixture that silences a surface needs a scenario that unsilences it. Work in `~/wt/mimo-always-store`, on branch `mimo/always-give-go-a-store` cut from `origin/main`.

## Why

Your DP-1463 diagnosis, verified by Claude:
- The harness boots Go with `deadStorePath` (`cmd/dp-oracle-diff/main.go:42-48`, used at `:477`) and `DP_ALLOW_NO_DB=1` (`withFreshMUDEnv`, `:1333-1340`).
- The only exceptions are scenarios that `needsPlayerStore` (`:2252`) recognises. That's 117 of 1,847 scenario files.
- So any Go code that reads saved players finds nothing during a census, even though C always has its player file and production Go always has a store.

That default dates from `0f6373303` (2026-09-08), when the store was PostgreSQL. The runtime has been SQLite-only since `351707e05` (2026-09-26), so a store now costs one file in the run's scratch directory.

## Do this

1. **Always prepare a store.** In `execute` (`main.go:532-537`), call `prepareGoReloginDB(goWork, scenario.EmptyPlayers, scenario.SeedPlayer, …)` for every scenario. Then delete:
   - `needsPlayerStore` and its tests (`main_test.go:291`, `:295`, `seed_player_test.go:54`);
   - `deadStorePath`;
   - the harness's `DP_ALLOW_NO_DB=1`, and update `withFreshMUDEnv`'s test (`main_test.go:220-226`).
   - Leave the server's own no-database mode alone (`cmd/server/main.go:553-562`).
   - Keep accepting the `live-store` fixture as a harmless no-op, so old scenarios still parse. Note in its parser comment that it's now the default.
2. **Guard against the fallback coming back.** If the Go server starts without persistence during a census, the scenario must not compare. Classify it INFRA, with a message naming the cause:
   - the warning at `cmd/server/main.go:559`, "explicitly running without persistence";
   - or the refusal at `:556`.
   - Add a worker-test case, and a revert proof: remove the store, and the case goes INFRA.
3. **Check that the store matches C's player file at boot:**
   - `empty-players`: both start empty.
   - Otherwise: the shipped first player (`prepareGoReloginDB`'s Testor twin), and nothing else.
   - Write a unit test listing the store's names at boot for each mode.
4. **Re-prove the three scenarios this was hiding:**
   - `clan-offline-applicant-depth` (DP-1463);
   - `show-player-stored` and `show-rent-stored` (DP-1464).
   - Each should go STALE. Move their manifest rows back from `blocked` at the depth and seeds they claim, drop their `divergence: DP-` notes, run `make expected-divergences`, and remove their pins.
   - Revert step 1, and all three must go red again (R5h).
5. **Run the full combined census:**

   ```bash
   scripts/census.sh start --combined --name mimo-always-store
   ```

   This change touches every scenario, so a targeted census can't show its effect. Use `--combined`, without `--release`; Claude runs the release gate.
   - For **every row whose verdict differs from main's**, explain it in `docs/fidelity/evidence/always-store/changed.tsv` (`scenario`, `before`, `after`, `cause`, `action`). Use the most recent combined census on main as "before"; Claude can point you to it. Each change is one of:
     - (a) a divergence the empty store was hiding, like these three;
     - (b) a scenario that relied on the empty store and needs fixing;
     - (c) noise you have evidence for.
   - **Fix (b) in the scenario, never in game code.** For (a), stop on that row and report it, and Claude files it. Possible cases: character creation refusing a name already in the store, the clan fallbacks DP-1463 lists, and the `set file` side effect in DP-1464.
   - **Record the elapsed time** against the previous combined census, in case the stores slow it down.

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
  - `bash scripts/test_oracle_regression_worker.sh`
- **Censuses:** only through `scripts/census.sh`, and use `wait --run`. Don't poll, and don't rerun the whole census for INFRA rows; the wrapper rechecks them.
- **Game code:** don't touch it. If a fix needs it, stop and report.
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`. If R5i should name this case, propose the exact text in the PR.
- **No outside help:** don't invoke another agent, model or paid service.

## Report

Open a PR, ending its description with "Decisions for Zach". Then stop and report:
- the census verdict and run directory;
- the `changed.tsv` summary, by cause;
- the three re-proofs and their revert proof;
- the elapsed-time comparison;
- every type (a) row, for Claude to file.

Claude reviews in Zach's session, and Zach merges.
