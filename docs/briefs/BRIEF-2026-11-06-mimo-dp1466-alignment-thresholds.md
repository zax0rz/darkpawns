# Brief for MiMo: DP-1466, spells treat any nonzero alignment as good or evil

Linear: DP-1466. Rulebook: R1 (bytes), R5b and R5c (find the class), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

Work in a worktree: `git worktree add ~/wt/mimo-dp-1466 -b mimo/dp-1466-alignment-thresholds origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

**Other agents are in `pkg/spells` at the same time.** DeepSeek's #2318 changes `call_magic.go` and `inflictDamage` in `damage_spells.go`, GLM is changing level reads in `affect_spells.go`, and Kimi is changing `checkReagents`. Stay inside the alignment checks. Rebase onto `origin/main` before opening the PR. If a conflict lands in someone else's hunk, stop and report.

## Ground truth (verified by Claude at `origin/main` 3298d1de9)

**C:** `#define IS_GOOD(ch) (GET_ALIGNMENT(ch) >= 350)` and `#define IS_EVIL(ch) (GET_ALIGNMENT(ch) <= -350)` (`src/utils.h:454-455`). Alignments from -349 to 349 are neutral.

C's uses in the spell code:
- `src/spells.c:644,649` and `:855,860`: enchant weapon and enchant armor pick the glow and the anti-good/anti-evil flag;
- `src/spells.c:1189,1192`;
- `src/magic.c:725-750`: dispel evil/good;
- `src/magic.c:1143,1163,1187`: affect gates.

**Go:**
- `isEvil` and `isGood` (`pkg/spells/damage_spells.go:441-453`) test `< 0` and `> 0`.
- The enchant spells check the sign inline (`pkg/spells/affect_spells.go:2302,2305` and `:2369,2372`). A caster at alignment 100 gets "$p glows blue." and an anti-evil weapon in Go; C says "glows yellow." and sets neither flag.
- Elsewhere the port already uses ±350: `pkg/game/player_affects.go:193,196` and `pkg/game/mobact.go:96,103`.

**Tests pin the bug:** `TestIsEvil` and `TestIsGood` require -1 and +1 to count.

## Do

1. **Audit (R5b):** list every alignment test in `pkg/spells`, plus any `GetAlignment()` compared with 0 anywhere in `pkg/`, against its C site. Write `docs/fidelity/evidence/dp-1466/alignment-sites.tsv` with columns `Go file:line`, `test`, `C file:line`, `C macro or test`, `match`.
2. **Fix every mismatch.** Use C's thresholds, and reuse one predicate rather than inlining the numbers again.
3. **Rewrite `TestIsEvil` and `TestIsGood`** to C's thresholds: -349, -350, 349 and 350.

## Proof

1. **Unit tests** at the boundaries: enchant weapon at alignment 100 glows yellow and sets no flag, and dispel evil on a target at -100 does what C does. Revert proof for each; paste the failures.
2. **Oracle:** a scenario where a caster at an alignment between 1 and 349 casts enchant weapon and then `identify`s or `look`s at the weapon. It must be red before the fix and green after (R5h). Add one for dispel evil at -100 if a consumer is reachable.
3. **A targeted census** of your new scenarios plus every scenario that casts enchant weapon, enchant armor, dispel evil or dispel good (grep for them, and name the set and why): `scripts/census.sh start --name mimo-dp1466-alignment --scenarios …`, then `wait --run <dir>`; don't poll.

## Rules

- Cite C only from `src/`, grepped in this session.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`. CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1466` in its title** (Linear closes issues from the PR title), containing the TSV, the fix, the tests and revert proofs, the scenarios, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
