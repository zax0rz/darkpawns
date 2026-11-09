# Brief for MiMo: DP-1474, a switched immortal reaches specials as the player, not the mob

Linear: DP-1474. Rulebook: R1 (bytes), R5b and R5c (find the class), R5e (verify the call path), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

Work in a worktree: `git worktree add ~/wt/mimo-dp-1474 -b mimo/dp-1474-switched-actor-specials origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

## Ground truth (verified by Claude at `origin/main` 6d87d8bcb)

Luna's PF-102 probes 2 and 8 found two scenarios that are red without any mutation. Both scenario files are retained in the run sources:
- `~/Archives/darkpawns/oracle-runs/2026-11-06/luna-pf102-2-clean/source/cmd/dp-oracle-diff/scenarios/newbie-zone-entrance-switched-depth.txt`
- `~/Archives/darkpawns/oracle-runs/2026-11-06/luna-pf102-8-clean/source/cmd/dp-oracle-diff/scenarios/shopkeeper-switched-steal-depth.txt`

Copy them into your branch. Each is the vehicle.

**1. Newbie zone entrance** (switched into a mob in room 16300, then `south`):
```
C : Nah, you're too much of a badass to go in there!
Go: The door seems to be closed.
```
C (`src/spec_procs.c:2262-2274`) reads `GET_LEVEL(ch)`, and `ch` is the **mob body**, because a switched descriptor's character is the mob. Go's `specNewbieZoneEntranceActor` (`pkg/game/actor_specials_guards.go:198`) has the right body, but returns `SpecialUnsupported` unless `ctx.Actor.IsNPC()`. Go prints "door closed", so the actor reaching it is apparently the player, or the special never runs. Find which (R5e).

**2. Shopkeeper steal** (switched into a horse, then `steal barrel`):
```
C : Thib the jeweler shouts 'a horse is a bloody thief!'
Go: Thib the jeweler slaps KeeperPet.
    You are slapped by Thib the jeweler.
    Thib the jeweler shouts 'KeeperPet is a bloody thief!'
```
C (`src/shop.c:971-978`) does `do_action(keeper, GET_NAME(ch), cmd_slap, 0)` and then `act("$N shouts '%s'", ..., TO_CHAR)`. `GET_NAME` of the horse is its short description, "a horse". The slap goes through the social's normal target lookup, and **C sends no slap lines at all**, so that lookup finds no one. Work out why from C (grep the social's victim lookup) and match it. Go's `specShopKeeperActor` (`pkg/game/actor_specials_shop.go:19`) names the switched **player**, KeeperPet, in both lines.

**Also in that function:** the Kender check reads `me.Proto().Race`, the keeper's race. C's `GET_RACE(ch)` is the **actor's** race (`src/shop.c:973`). Fix it, and add a unit test with a Kender actor.

**The likely class (R5b):** when a session is switched, the special-procedure and shop dispatch hands the player in as the actor (its name, level and `IsNPC`), not the mob body.

## Do

1. **Trace the switched-actor dispatch** from command input to `CharacterSpecialContext.Actor` (R5e), and fix it at the source, so every special sees the mob body as C does. Don't patch the two specials one by one.
2. **Audit (R5b):** list every special and shop path that reads the actor's name, level, race, `IsNPC` or position, with what it receives when switched before and after your fix. Write `docs/fidelity/evidence/dp-1474/switched-actor.tsv` with columns `Go file:line`, `reads`, `C file:line`, `before`, `after`.
3. **Fix the Kender race check.**

## Proof

1. **Unit tests:** the switched-actor context carries the mob body, and a Kender actor triggers the shout. Revert proof for each; paste the failures.
2. **Oracle:** both scenarios green, each with a C witness line in the manifest (R5j), and each red again with your fix reverted (R5h). Add both to the manifest as `oracle-green`.
3. **A targeted census** of the two scenarios plus every scenario that uses `switch` (grep for it; name the set and why): `scripts/census.sh start --name mimo-dp1474-switched --scenarios …`, then `wait --run <dir>`; don't poll. Switched sessions touch many paths, so look hard at any changed row.

## Rules

- Cite C only from `src/`, grepped in this session.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, and `make db-migrate-test` if an exported `pkg/game` symbol changes. CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1474` in its title**, containing the fix, the TSV, the tests and revert proofs, both scenarios, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
