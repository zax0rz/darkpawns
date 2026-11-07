# Brief for Kimi: DP-1483, a hidden attacker doesn't fade into existence when it hits

Linear: DP-1483. Rulebook: R1 (bytes), R5b and R5c (find the class), R5e (verify the call path), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

**Start a fresh session for this brief.** Read `AGENTS.md` and `docs/fidelity/RULEBOOK.md` first.

**Wait for PR #2318 to merge.** It makes damage-spell victims fight back, and this bug only shows once they do. If #2318 isn't on `origin/main` when you start, stop and say so. Then work in a worktree: `git worktree add ~/wt/kimi-dp-1483 -b kimi/dp-1483-hidden-attacker-fade origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

## Ground truth (verified by Claude at `origin/main` daa83851e)

**The scenario:** `spellmatrix-staff-olc-10-depth`. `staff1002` is built with spell 63, chameleon (`src/spells.h:128`). Chameleon sets `AFF_HIDE` (`src/magic.c:1292-1294`). Used from a staff, it lands on everyone in the room. After shape 2, those players attack the staff's user, and on the next `~dpclock pulse 20` the output differs.

In PR #2318's run `~/Archives/darkpawns/oracle-runs/2026-11-06/cline-dp1481b-cluster-s4` (capture1 `blocks.json`, block 96), **C** sends `ItemBuilder slowly fades into existence.` before ItemBuilder's first hit, and the same for ItemPeer and ItemView. **Go** sends the hits with no fade lines.

**C:** `damage()` (`src/fight.c:1459-1463`):
```c
if (IS_AFFECTED(ch, AFF_HIDE))  /* we do this inline now, instead of calling appear() */
{
  REMOVE_BIT_AR(AFF_FLAGS(ch), AFF_HIDE);
  act("$n slowly fades into existence.", FALSE, ch, 0, 0, TO_ROOM);
}
```
This clears **only the bit**, not the chameleon affect record. Check what that means later, for example whether the wear-off message still arrives when the affect expires, and keep it. `appear()` (`src/fight.c:108-120`) is the full version, and other callers use it. Don't mix the two up.

**Go:** `pkg/combat/fight_core.go:510-515` has the same branch: `cbHasAffect(ch, AFF_HIDE)`, then `cbRemoveAffect`, then a broadcast. So either this branch isn't reached for these melee hits, or chameleon's `engine.AFFHide` (`pkg/spells/affect_spells.go:413`) isn't the bit `cbHasAffect(ch, AFF_HIDE)` reads. **Find which** (R5e): trace a melee hit from the violence round to the damage function and check the bit values. Don't guess.

**A second bug on the same line:** Go broadcasts `fmt.Sprintf("%s slowly fades into existence.", chName)` with the real name. C's `act()` substitutes `$n` through PERS, so an observer who can't see the actor reads "Someone slowly fades into existence.". PR #2318 added the per-receiver path: `cbBroadcastPerReceiver` and `persName` in `pkg/combat/callbacks.go`. Use it.

## Do

1. **Fix the fade line** so it fires where C's does, with C's bit-only removal.
2. **Use the per-receiver broadcast for it.**
3. **Audit (R5b):** every `cbBroadcast` call in `pkg/combat` whose message contains an actor's or victim's name. For each one, find the C `act()` it ports and say whether PERS applies. Write `docs/fidelity/evidence/dp-1483/broadcast-names.tsv` with columns `Go file:line`, `message`, `C file:line`, `PERS applies (yes/no)`, `fixed (yes/no)`. Fix the ones that are clear. If the list grows past about 10, fix the fade line plus the three most common, list the rest, and stop.
4. **Manifest:** `staff-olc-10` carries a pin citing DP-1483 from #2318. When it goes green, remove the pin and set the row to `oracle-green`, with a C witness line (R5j).

## Proof

1. **Unit tests:**
   - a hidden attacker's first hit sends the fade line and clears `AFF_HIDE`, but keeps the chameleon affect record;
   - an observer who can't see the attacker reads "Someone".

   Revert proof for each; paste the failures.
2. **Oracle:** `spellmatrix-staff-olc-10-depth` turns green. Revert the fix and show it goes red again (R5h).
3. **A targeted census** of staff-olc-10, every `spellmatrix-*` scenario, and every scenario that uses chameleon, hide or sneak (grep for them; name the set and why): `scripts/census.sh start --name kimi-dp1483-fade --scenarios …`, then `scripts/census.sh wait --run <dir>`. Don't poll it in a loop. If the census's own setup refuses a step (`setup-refused`), that run proves nothing (R5j): fix the setup and rerun.

## Rules

- Cite C only from `src/`, grepped in this session.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, and `make db-migrate-test` if an exported `pkg/game` symbol changes. CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1483` in its title**, containing the fix, the TSV, the tests and revert proofs, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
