# Brief for DeepSeek (via Cline): DP-1460 remainder, the first round after a scroll hit has too few attacks

Linear: DP-1460. Rulebook: R1 (bytes), R3 (determinism and draw parity), R5e (verify the call path), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

Work in a worktree: `git worktree add ~/wt/cline-dp-1460 -b cline/dp-1460-first-round-attacks origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

## Ground truth (verified by Claude at `origin/main` 6d87d8bcb)

You know this code: #2318 made item-damage victims fight back. One pinned scenario still differs, `spellmatrix-damage-scroll-group-04-depth`, with 7 blocks, pinned citing DP-1460. Evidence: `~/Archives/darkpawns/oracle-runs/2026-11-06/cline-dp1481b-full`, capture1 `blocks.json`.

On the first `~dpclock pulse 20` after `recite book ItemPeer`:
```
C : ItemPeer hits you very hard.
    ItemPeer hits you.
    ItemPeer barely hits you.
    You barely hit ItemPeer.
    You scratch ItemPeer as you hit her.
Go: ItemPeer scratches you as she hits you.
    You scratch ItemPeer as you hit her.
```
C gives ItemPeer three attacks and ItemMage two; Go gives each one. Everything after follows from that: ItemMage ends at 2485 HP in C and 2504 in Go.

**C's attack count** comes from `perform_violence` (`src/fight.c`, around 1904-1940). It starts from a level band (`GET_LEVEL(ch) >= 31` gives 5, `<= 30` gives 4, down to 1), applies `number()` bonus rolls, and has a separate NPC branch. Read it all; don't trust this summary (R5e). Then find Go's twin.

**Hypotheses to test, not answers:**
- Go may count attacks with a branch C doesn't take for this pair, for example the NPC branch for a player, or the wrong level.
- Go may enroll the scroll victim so that its first round runs a different code path from a normal round.
- Go may consume a different number of draws before the count. Run with `--draw-parity` and find the first draw that differs; that's usually the answer (R3).

## Do

1. **Run `spellmatrix-damage-scroll-group-04-depth` with `--draw-parity`** and find the first divergent draw. Name the C and Go functions that make it.
2. **Fix the cause.** If the cause is an attack-count path, audit every caller of that path (R5b): melee, item-damage enrollment, and specials that start fights.
3. **Re-pin or promote:** if the scenario goes green, remove its pin and set its manifest rows to `oracle-green`, with a C witness line (R5j). If part of the divergence is left, re-mint the pin from your census run and say what's left.

## Proof

1. **A unit test** for the attack count of the case that differed, with a revert proof. Paste the failure.
2. **The scenario** green, and red again with the fix reverted (R5h).
3. **A targeted census** of every `spellmatrix-damage-*` scenario, every `spellmatrix-*olc*` scenario, and every scenario with a `~dpclock pulse` during melee (grep for them, and name the set): `scripts/census.sh start --name cline-dp1460-attacks --scenarios …`, with `--draw-parity`. Then `wait --run <dir>`; don't poll. This touches every combat round, so if the targeted set shows any changed row outside the cluster, run the full census and attribute it.

## Rules

- Cite C only from `src/`, grepped in this session, never from memory or from `darkpawns-c-oracle/`.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, and `make db-migrate-test` if an exported `pkg/game` symbol changes. CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1460` in its title**, containing the diagnosis (the first divergent draw and the C and Go functions), the fix, the test and revert proof, the census directory with attribution, and a **Decisions for Zach** section. If the cause turns out to be design-level, such as a whole round-ordering model, stop after the diagnosis and report before changing anything. Then stop and report. Claude reviews in Zach's session, and Zach merges.
