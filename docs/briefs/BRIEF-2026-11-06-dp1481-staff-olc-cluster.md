# Brief: DP-1481, the staff/wand OLC cluster is three stable divergences

For DeepSeek (via Cline) or MiMo; Zach assigns. Linear: DP-1481. Rulebook: R1 (bytes), R3 (determinism), R5b and R5c (find one, find the class), R5e (verify the call path), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

Work in a worktree: `git worktree add ~/wt/<agent>-dp-1481 -b <agent>/dp-1481-staff-olc-cluster origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

## Why this matters

The spellmatrix staff/wand OLC cluster turns every full census red, so the v0.1.9 release gate can't go green until it's fixed. DP-1456 recorded it as instability under census load, and no one has found a cause. Claude has now looked, and in the latest run it isn't unstable at all: three ordinary divergences are hiding under the label.

## Ground truth (verified by Claude at `origin/main` b772d843a)

**The evidence:** `~/Archives/darkpawns/oracle-runs/2026-11-06/cline-dp1480-room-order-commit/initial-logs/attempt.ith67x/spellmatrix-staff-olc-01-depth.capture{1,2}/blocks.json`. Each file is a JSON list of `{Command, Oracle, Go, Diff}`. Attempts 1 and 2 are **byte-identical on both engines**, with 40 divergent blocks. Read `cmd/dp-oracle-diff/scenarios/spellmatrix-staff-olc-01-depth.txt` for the setup. ItemMage uses staffs; ItemPeer, ItemView (the observer) and ItemBuilder stand in the room. `staff0102` casts blindness on everyone else in the room, including the observer.

### Shape 1: Go adds "Someone …" lines a blind observer shouldn't get

From `remove staff0102` on, the blind observer gets, in Go only: `Someone stops using something.`, `Someone drops something.`, `Someone gets something.`, `Someone grabs something.`, `Someone starts using OLC.` and `Someone stops using OLC.`

C sends each of these through `act()` with `hide_invisible = TRUE`, which skips any receiver who can't see the actor:
- `src/act.item.c:1727` (`$n stops using $p.`)
- `src/act.item.c:279` (`$n gets $p.`)
- `src/act.item.c:1394` and `:1397` (`$n grabs $p.`)
- `src/olc.c:275` (`$n starts using OLC.`)
- `src/olc.c:514` (`$n stops using OLC.`)
- **Find the drop line yourself.**

**One cause, found:** `(*World).actToRoom` (`pkg/game/item_helpers.go:377`) passes `hideInvisible = false` to `Act` for all 14 of its callers, so C's flag is lost. The adapters next to it (`actToChar`, `actToVictim`, `actToRoomExclude`) may do the same. Check the OLC path too (`pkg/session/oedit.go:287,320`, `medit.go:189,230`, `olc_registry.go:202`).

### Shape 2: player victims of staff damage don't fight back

In C, after `use staff0103` (burning hands) and `use staff0104` (call lightning), ItemBuilder and ItemPeer attack ItemMage at the next `~dpclock pulse 20`. ItemMage's `score` shows 4947, then 4915, then 4870 of 5000. In Go, nobody attacks and ItemMage stays at 5000.

C's `damage()` starts the victim fighting: `src/fight.c:1443-1445`, `if (GET_POS(victim) > POS_STUNNED && !FIGHTING(victim)) set_fighting(victim, ch)`. Follow the Go call path for a `CAST_STAFF` area or fan-out damage spell on a **player** victim, from `useStaff` (`pkg/game/other_economy.go:397`) to wherever damage is applied, and find where the victim's fight never starts. This is likely a class across every non-melee damage source that hits a player: other cast types, wands, scrolls, spells cast directly. Audit it (R5b).

### Shape 3: Go drops a blind victim's own damage lines

On `use staff0103`, C sends the blind observer `You cry out in pain as someone grabs you with burning hands!` plus the room lines. Go sends nothing to the observer, although its HP drops in both engines. C's skill-message lines go out with `hide_invisible = FALSE`; check `skill_message` in `src/fight.c` yourself. Find what suppresses them in Go.

### What may be left after the fixes

DP-1456 saw C's own combat lines differ between census runs (the fan-out order View/Peer/Builder vs Peer/Builder/View). Shape 2 is all combat lines, so part of that "instability" may just be Go lacking the combat. Fix the three shapes first, then measure again.

## Do

1. **Confirm each shape against the C call path** before changing anything (R5e). Write `docs/fidelity/evidence/dp-1481/shapes.tsv` with columns `shape`, `C file:line`, `C hide flag or condition`, `Go file:line`, `Go behaviour`, `fix`.
2. **Shape 1 class:** give the adapters a hide flag, or call `Act` directly, and pass C's flag at every one of their call sites. List each call site with its C line and flag in `docs/fidelity/evidence/dp-1481/act-adapters.tsv`. Don't guess any flag; grep each one.
3. **Shape 2:** start the victim fighting where C's `damage()` does. Find every damage path in Go that hits a player and skips this; fix them, and list them in the PR.
4. **Shape 3:** match C's hide flags on the damage and skill messages.
5. **Re-measure the whole cluster.** Run staff-olc-01..16, wand-olc-01..10 and potion-olc-03 at seed 1. Then, for every scenario that is still red, compare attempt 1 with attempt 2 on both engines.
   - If they're identical, it's a stable divergence: report it with its first divergent block.
   - If they differ, find the first block where **C** differs between attempts, and report the command before it and what changed.

   Don't pin, opt out, or edit a scenario to make it pass. Stop and report before filing anything new; Claude files issues.
6. **Manifest rows:**
   - A scenario that now passes with a C witness line (R5j) returns to `oracle-green`. Remove any pin or `run-varying` declaration it no longer needs.
   - DP-1460 and DP-1461 are open divergence issues on some of these scenarios. If a fix changes their pinned shapes, re-mint the pins with `scripts/pin_expected_divergences.py`, and say which issue each shape still belongs to.

## Proof

1. **Unit tests, one per shape, each with a revert proof showing it fails without the fix:**
   - a blind observer gets nothing from a `hide_invisible` item act;
   - a player hit by a staff damage spell is fighting the caster afterwards;
   - a blind victim receives its own damage line.
2. **Revert proof at the scenario level:** with all fixes reverted, `spellmatrix-staff-olc-01-depth` shows the same 40 divergent blocks again.
3. **A full census**, because shapes 1 and 3 change who receives `act()` output across the game: `scripts/census.sh start --name <agent>-dp1481-staff-olc`, then `wait --run <dir>`; don't poll. Attribute every changed row against the newest full census on main: a fix, a flake, or a regression. Stop on any regression.
4. **Draw parity:** shape 2 adds combat rounds in Go, so run the fixed cluster with `--draw-parity`, and report any `draw_divergent` row (R3).

## Rules

- Cite C only from `src/`, grepped in this session, never from memory or from `darkpawns-c-oracle/`.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`. Propose any rule text in the PR description.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, and `make db-migrate-test` if an exported `pkg/game` symbol changes.

## Done means

A PR against `main` containing:
- both TSVs;
- the fixes and their unit tests, with revert proofs;
- the cluster re-measurement: what passes now, and what's left, each with its first divergent block;
- the census run directory with its changed-row attribution;
- a **Decisions for Zach** section.

If the fixes grow past about 600 changed lines, stop after shape 1 and report; the rest can be a second PR. Then stop and report. Claude reviews in Zach's session, and Zach merges.
