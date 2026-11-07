# Brief for DeepSeek: mirror image's silent guard, and `create_mobile` scaling on every spawn (DP-1487)

Linear: DP-1487. Rulebook: R1, R4 (no invention), R5b/R5c (the class), R5e (verify the call path), R5h, R5j.

## Ground truth (verified by Claude at `origin/main` b7fdd6ef5)

**The divergence.** Scenario `spellmatrix-wand-mirror-image-depth`, census `~/Archives/darkpawns/oracle-runs/2026-11-06/glm-dp1468-caster-level-r3`:
- On `use wand0990 ItemMage`, C prints nothing past the wand line. Go prints "You divide yourself in two!" plus the room line.
- On `stat clone`: C shows `MOB 'a clone'`, `XP: [0]`; Go shows `MOB 'ItemMage'`, `XP: [100]`, and other lines differ too.

**C `spell_mirror_image`** (`src/spells.c:1131-1162`) runs in this order:
1. `create_mobile(ch, MOB_CLONE, 1, FALSE)` first. The clone already exists and is placed by now.
2. Then it returns **silently** if `GET_NAME(ch)`, `ch->player.title` or `ch->player.description` is NULL (`:1138-1142`).
3. Only after that does it rename the clone, print both messages, and move memory and hunting from caster to clone (`:1144-1161`).

The C transcript says that for this user one of the three guards fires: the clone keeps the prototype's name and the messages never print. **Find which guard fires, and why (R5e).** The likely candidate is a character that never set a description, which leaves `player.description` NULL. Then decide how Go represents "NULL" faithfully: is an empty description in Go always C's NULL, or are there paths where C has `""`? Trace where C sets and clears `player.description` (character creation, the `description` editor, loading from the pfile) and cite each one.

**Go** (`pkg/spells/affect_spells.go` around `:3870-3905`):
- It has none of the three guards.
- It prints an invented "You fail to divide yourself." when the spawn errors (R4). C just returns, because `NULL == clone` (`:1138`).
- It names the clone, but doesn't set the short description, long description, description or sex the way C does (`:1144-1149`).
- Check whether it ports the memory and hunting hand-off loop (`:1154-1161`).

**`create_mobile`** (`src/new_cmds2.c:588-612`) sets the instance's level, `EXP 0`, damroll, NDD, SDD, hitroll, AC, max hit and hit, and zeroes the carry counts. Go's port, `(*MobInstance).ConfigureCreatedMobile` (`pkg/game/mob.go:703`), is called only from `pkg/command/skill_commands.go:1655` and one test. **The spell spawn paths don't call it** (registered but never called). `XP: [100]` vs `[0]` is the visible symptom.

**C `create_mobile` call sites (the class):**
- `src/spells.c:1071`: elementals
- `src/spells.c:1136`: mirror image
- `src/spells.c:1212`: divine intervention angel
- `src/magic.c:1726`
- `src/new_cmds.c:2675`
- `src/new_cmds2.c:643-648`, `:739`

## Do

1. **Port the guards** in C order, with the clone created before them. Delete the invented failure message, and copy C's field copies and the memory/hunting loop. Cite each line.
2. **The spawn class.** For every C call site above, find the Go twin. Make sure `create_mobile`'s scaling runs on it, including the `hunting` branch's room placement and `set_hunting` (`src/new_cmds2.c:608-611`). Use one shared Go path rather than five hand copies. Write the table to `docs/fidelity/evidence/dp-1487/create-mobile-sites.tsv` with columns `C site`, `Go site`, `before (scaled? y/n)`, `after`.
3. **Tests.** A unit test for each guard, one for the scaling on each spawn path, and one for the memory hand-off. Each needs a revert proof.
4. **Scenarios.** Get `spellmatrix-wand-mirror-image-depth` green (remove its pin). Add a scenario where the caster **has** a description, so C's positive branch prints and renames: that's the witness for the other half. Add one spawn-stat scenario for elementals or the angel if a vehicle reaches it (`stat <mob>` after the cast).
5. **Census.** Every scenario that casts a spawning spell or reaches a `create_mobile` site (grep `mirror`, `elemental`, `divine`, `conjure`, and the skill at `skill_commands.go:1655`). Name the set. **It must include** `spellmatrix-potion-olc-03-depth`, `spellmatrix-scroll-olc-03-depth` and `spellmatrix-wand-olc-08-depth`. Each of those carries mirror image **and** divine intervention, is pinned today, and may clear with your fix. Report each one's new first divergent block, or report it green with its pin removed. GLM is working on the staff-only run-varying cluster under DP-1461 at the same time and won't touch spawning code.

## Rules (all briefs)

- Read `AGENTS.md`, `docs/fidelity/RULEBOOK.md` (R1–R5j) and `docs/fidelity/DEPTH_TESTING.md` first.
- Work in your own worktree: `git worktree add ~/wt/<agent>-<issue> -b <agent>/<issue-slug> origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, never stop processes by name. Other agents are working in `pkg/` today, so rebase on `origin/main` before you report.
- `src/` and `darkpawns-c-oracle/` are read-only. Cite C only from `src/`, grepped in this session (R5g), never from memory.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`. If a rule needs changing, propose the exact text in your report.
- Never invoke another model, agent or paid service.
- Every fix gets a unit test **and a revert proof**: revert the fix, show the test fail, and quote the failure (R5h). Every scenario that you turn green needs a C witness line, meaning C's own transcript shows the event happening (R5j).
- **Pins.** If you clear a divergence, its pin goes STALE. Remove the pin. Don't re-mint it. If you find a new divergence, don't pin it, opt it out or edit the scenario to pass: report it with its first divergent block, and Claude files it. If a PR legitimately needs new pins, mint them only with `scripts/pin_expected_divergences.py --from-run <run-dir>`, run from your worktree. Then run `make expected-divergences` and commit.
- **Census** only through `scripts/census.sh` (`start --name … --scenarios …`, then `wait --run <dir>`). Don't poll it, and don't `cat` or `tail` a running `census.log`. Attribute every changed row against the rehearsal baseline `~/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107` (a full combined census of main at `b7fdd6ef5`, with draw parity).
- Run every gate on its own and check its exit status. Never chain gates with `;`. The gates: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt` (gofumpt), `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`. CI `test` must be green before you report.

## Done means

A PR against `main` with **the Linear ID in its title** (GitHub→Linear closes issues only from the title). The PR body has the trace, the tests, the revert proofs, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
