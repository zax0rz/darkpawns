# Brief for DeepSeek: DP-1467, Hellfire knocks victims to resting, plus the position-literal class

2026-11-06, by Claude for Zach to hand out. A one-line fix, a scenario that can see it, and a mechanical check over the whole class.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Work in `~/wt/cline-dp-1467`, on branch `cline/dp-1467-hellfire-position` cut from `origin/main`.

## Ground truth (grepped by Claude at `origin/main`, 2026-11-06)

**C:** Hellfire's knockdown, for victims above level 4, is in `src/spells.c:735-744`:
```c
if (number(1,20) > GET_DEX(ch))
{
   GET_POS(tmp_victim) = POS_SITTING;
   act("The fires of hell bring you to your knees!\r\n", ...TO_VICT);
   act("The fires of hell bring $N to $S knees!\r\n", ...TO_NOTVICT);
}
damage(ch, tmp_victim, dam, SPELL_HELLFIRE);
```
`POS_SITTING` is 6 (`src/structs.h:215`).

**Go:** `pkg/spells/affect_spells.go:3144` has `cn.SetPosition(5) // POS_SITTING`. Go's own constants agree with C (`spells.PosSitting = 6`, `pkg/spells/spell_info.go:72`; `combat.PosSitting = 6`, `pkg/combat/formulas.go:39`), so 5 is resting. The comment is right and the number is wrong.

**Why the census missed it:** seven scenarios cast Hellfire:
- `cast-hellfire-area-depth`;
- `cast-hellfire-peaceful-depth`;
- `spellmatrix-hellfire-spoken-depth`;
- `spellmatrix-world-scroll-hellfire-depth`;
- `spellmatrix-world-wand-hellfire-depth`;
- `spellmatrix-potion-olc-02-depth`;
- `spellmatrix-staff-olc-09-depth`.

The position only shows in a later command, such as another character's `look` ("…is sitting here." against "…is resting here.") or the victim's own `stand`. None of them runs one.

**The class:** Go has numeric position literals with `// POS_…` or `// Pos…` comments in several places, for example:
- `pkg/game/spec_procs.go:943`, `:946`;
- `pkg/game/player.go:338`;
- the whole `pkg/game/social_positions.go` table.

One of them is wrong. The rest haven't been checked.

## Do this

1. **The fix:** `cn.SetPosition(int(PosSitting))`, or whatever the package's named-constant form is. Use the constant, not a corrected literal.
2. **The scenario:** a new `hellfire-knockdown-position-depth`, or an extension of `cast-hellfire-area-depth` if it fits cleanly.
   - A victim above level 4 fails the dex check, then an observer runs `look`.
   - Pick the caster's dex and seed so `number(1,20) > GET_DEX(ch)` actually fires, and show it with a C witness line: "The fires of hell bring … to … knees!" (R5j).
   - If an NPC victim stands back up before the look (its fight round can change position), use a player victim, or put the `look` in the same pulse. Explain what you chose.
3. **The class check:** write `scripts/check_position_literals.py`.
   - Find every numeric position literal in non-test Go under `pkg/` that carries a `POS_` or `Pos` comment.
   - Compare the number with C's `#define` for the named position (`src/structs.h:210-218`). Print every mismatch, and exit non-zero if there are any.
   - Wire it into `make fidelity-depth`, or the nearest existing check target, and quote its output in the PR.
   - **For every mismatch:**
     - If it's a wrong value at a C call site (like Hellfire), fix it to the named constant, with its own revert proof.
     - If the comment is wrong and the value right, fix the comment.
     - If you can't tell which side is wrong, report it, and don't change it.
   - **For `social_positions.go`:** also find where C's social minimum positions come from (the socials data file under the oracle's `lib/`, or `src/act.social.c`). Check the table against that, not just against its comments. Report the source and any mismatches; fix only those you can cite.
4. **Prove it:**
   - The new scenario is green at its claimed seeds.
   - Revert the Hellfire line to `5`, and the scenario goes red (R5h).
   - Corrupt one literal, and the checker fails.
5. **Census:** targeted, through `scripts/census.sh`: your new scenario, the seven Hellfire scenarios above, and every scenario that sends a social, if you changed the social table.
   - Two Hellfire scenarios (`spellmatrix-potion-olc-02-depth`, `spellmatrix-staff-olc-09-depth`) are pinned expected divergences for DP-1460, and `staff-olc-09` is declared run-varying. If either one's shape changes, the pin goes FAIL. If that happens, report it rather than re-pinning it; Claude decides.

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
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`.
- **No outside help:** don't invoke another agent, model or paid service.

Open a PR, ending its description with "Decisions for Zach". Then stop and report:
- the fix;
- the checker's output and every mismatch with its disposition;
- the social-table source;
- the census run directories;
- the revert proofs.

Claude reviews in Zach's session.
