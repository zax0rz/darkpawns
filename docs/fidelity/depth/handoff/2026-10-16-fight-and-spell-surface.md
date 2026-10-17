# DP-1371 fight and spell surface handoff — 2026-10-16

## Result and inventory

The starting comparison requested by this goal is PF-075's **2,106 blocked units**. The shared batch had already advanced `origin/main` to `3d4a35ea6`; that measured tree has 1,470 total units and 572 blocked. The two refreshed draft PRs together project to **1,526 total units and 628 blocked**, a net reduction of **1,478 blocked units** against the requested 2,106 baseline.

The delta from the current main snapshot is +56 blocked units: the fight PR adds 75 units of live-dispatch evidence scope, because the helper-only corpus test on main does not satisfy this goal's real registered-command proof bar and eight records are now proven through registered commands, while the spell PR closes 19 matrix residuals. This is a deliberate scope correction, not a claim that the live-dispatch corpus regressed. The pair remains in draft and is not yet current main.

| Inventory family | Current main | Updated draft(s) | Reason |
|---|---:|---:|---|
| `fight-skill / lib/misc/messages skill-message corpus` | 0 proven-already units | 75 blocked units | 83 C records are in scope; eight are now proved through registered dispatch, and 75 still need their live command-to-damage proof. |
| `fight-skill / src/fight.c act-send family` | 4 blocked | 4 blocked | 98 live sites: 94 proven, 0 gap-reachable, 4 switched-mob session-delivery stop-tier sites. |
| `fight-skill / new_cmds skill/action output` | 0 | 0 | Reconciliation is already in merged PR #2087. |
| `fight-skill / new_cmds2 skill/action output` | 0 | 0 | Reconciliation is already in merged PR #2087. |
| `spell-matrix / CAST_SPELL` | 66 | 58 | Eight spoken cells are proven: Protection from Good, Water Breathing, Bless, Cure Blind, Sleep, Animate Dead, Metalskin, and Mirror Image. |
| `spell-matrix / CAST_POTION` | 103 | 96 | Seven verified delegations, including Protection from Good. |
| `spell-matrix / CAST_SCROLL` | 103 | 101 | Create Food and Meteor Swarm are delegated after source audit. |
| `spell-matrix / CAST_WAND` | 103 | 101 | Create Food and Meteor Swarm are delegated after source audit. |
| `spell-matrix / CAST_STAFF` | 103 | 103 | No staff spell cell was promoted by the failed exploratory fixture. |

The resulting 299×5 spell matrix classifies 45 spoken cells proven, 11 item cells delegated, 458 gap-reachable cells, one C-undefined-behavior gap-blocked cell, and 980 C-unreachable cells. Residual spell inventory is 459. The only item delegations are the seven potion cells and two each for scroll and wand; staff-wide fan-out, item-level duration changes, and other route-sensitive branches remain open.

## Fight message proof

`lib/misc/messages` contains 83 `M` records across 66 attack types; C accepts the whitespace-padded marker at line 1175 (`src/fight.c:152`), so the prior 82-record count was low by one. `skill-message-matrix.tsv` records every record and the attacker, victim, and room result for each defined death, miss, hit, and god outcome. Eight records are now proven through registered command dispatch. The five earlier records prove their defined outcome/audience cells; Trip proves its nine non-# death/miss/hit cells; Circle proves its defined death, miss, hit, and God lines for all three audiences; Headbutt proves its death, miss, and hit lines for all three audiences, while its God triplet is C-unreachable because `do_headbutt` rejects non-NPC immortal targets before `damage()`:

- `M@0241` — Bash, through `do_bash`.
- `M@1284` — Tiger Punch, through `do_tiger_punch`.
- `M@1268` — Dragon Kick, through `do_dragon_kick`.
- `M@0257` and `M@0271` — the two Kick variants, through `do_kick`.
- `M@0335` — Trip, through `do_trip`; the three cases cover miss, hit, and death for attacker, victim, and room observer, while the God fields are `#`.
- `M@1221` — Circle, through `do_circle`; four outcome cases cover attacker, connected victim, and room observer.
- `M@0287` — Headbutt, through `do_headbutt`; miss, hit, and death cases cover attacker, connected victim, and room observer. Direct `--show-oracle` confirms the lethal arm at seed 1; five-seed runs are CLEAN with zero draw divergence. The God triplet is unreachable at `src/new_cmds.c:410-419`; `headbutt-immortal-depth` proves the pre-damage refusal.

Each proof reaches the production command, `damage()`, and `SkillMessage`, with exact C bytes and zero draw divergence. The other 75 records remain gap-reachable; the existing corpus unit test remains supplemental. C parses the records in `src/fight.c:126-175`, dispatches at `src/fight.c:1023-1095`, and reaches the handler from `damage()` at `src/fight.c:1534-1545`. Command call sites are in `src/act.offensive.c:419-741` and `src/new_cmds.c:735-815` for Trip and `:368-460` for Headbutt.

Fresh-main retained censuses `dp1371-final-surface-seed1`, `seed2`, `seed3`, `seed5`, and `seed8` each cover 20 scenarios: all Bash, Tiger Punch, Kick, and Dragon Kick outcomes, Dragon breath, neutral spell rescue, Protection from Good, and Water Breathing. Every run is CLEAN (20/20) with zero draw-divergent cases at Go HEAD `3d4a35ea6`; the five-run set is retained under `~/Archives/darkpawns/oracle-runs/2026-10-16/`. Trip's three cases pass at seeds 1, 2, 3, 5, and 8 with zero draw divergence (`dp1371-trip-skill-message-final-seed1/2/3/5/8`, Go HEAD `f233a290a`). Circle's four cases also pass at those seeds (`dp1371-circle-skill-message-revised-seed1/2/3/5/8`, Go HEAD `f049c9f81`); direct `--show-oracle` confirms its lethal block at seed 1, and seed 8 follows the miss arm. The first Circle seed-8 attempt exposed C's low-health warning on a miss; the final fixture sets victim max HP to 3 and current HP to 1, and the NOT_CLEAN run remains retained as `dp1371-circle-skill-message-seed8`.
Headbutt's miss/hit and lethal live-player fixtures are CLEAN (2/2) with zero draw divergence at actual seeds 1, 2, 3, 5, and 8, Go HEAD `4e0a8e63a`; the final `dp1371-headbutt-draw-seed1/2/3/5/8` runs set `ORACLE_REGRESSION_SEED` explicitly and enable `--draw-parity`. C dumps show miss on seeds 1, 2, 5, and 8 and hit on seed 3 for the live-health fixture, plus death on all five seeds for the sleeping one-HP victim fixture. Direct `--show-oracle` at seed 1 shows the death lines at all three recipients. Earlier runs remain diagnostic only: `dp1371-headbutt-skill-message-seed1` is NOT_CLEAN because a low-HP wimpy victim fled after a miss; the `v2`, `v3`, and `actual-seed` named batches did not enable draw parity, and `v2`/`v3` manifests show the default seed 1. The final fixture sleeps the victim so C forces Headbutt's success roll before lethal damage.

## Spell routes and delegations

The five route rows remain separately proven. Spoken samples include the violent `mag_damage` rescue and targeted Water Breathing. The potion fixture dispatches Invigorate, Vitality, and Poison in order; the scroll fixture dispatches Bless, Heal, and Lightning Bolt. The wand cases cover targeted Invisibility and violent targeted Lightning Bolt. Staff cases cover Sleep and Lightning Bolt room fan-out. The existing five-seed route batch is `dp1371-spell-routes-final-seed1/2/3/5/8` and is CLEAN with draw parity.

Six further per-spell spoken cells now have registered-cast output and state evidence: Bless (`wizard.unaffect-active-spells`), Cure Blind (`objmagic.mag-unaffects-output-sites`), Sleep (`objmagic.sleep-entry-gates.cast`), Animate Dead (`magic.animate-dead-message-pair`), Metalskin (`objmagic.metalskin-reagent-output`), and Mirror Image (`spell.mirror-image-output-audience`). Their five-seed targeted batches are `dp1371-bless-current-seed1/2/3/5/8`, `dp1371-cure-blind-current-seed1/2/3/5/8`, `dp1371-sleep-spell-current-seed1/2/3/5/8`, `dp1371-magic-surface-current-seed1/2/3/5/8`, `dp1371-metalskin-current-seed1/2/3/5/8`, and `dp1371-mirror-image-depth-seed1/2/3/5/8`. All are CLEAN at Go HEAD `9b87c8dff` with zero seed-1 draw divergence; the Go runtime is unchanged from `origin/main` `3d4a35ea6`. The runs are retained under `~/Archives/darkpawns/oracle-runs/2026-10-16/` and `2026-10-17/`.

Delegations are limited to C bodies whose selected branches ignore the item-only inputs: `mag_creations` ignores level, target, object, and save type (`src/magic.c:1968-1998`); Meteor Swarm reads caster level and room victims rather than the selected item arguments (`src/spells.c:1088-1127`); selected self-only `mag_affects` branches for Sense Life, Levitate, Chameleon, Protection from Evil, and Protection from Good preserve recipient, state, and output without reading item level or `SAVING_ROD` (`src/magic.c:1132-1181,1266-1271,1292-1297`). Scroll/wand/staff distinctions remain open where arguments or recipient fan-out change (`src/spell_parser.c:592-817`; `src/act.other.c:897-968`).

The staff Earthquake route experiment repeatedly failed before the C server completed startup (`bind: Address already in use`, classified INFRA). Its fixture and route row were removed, and no staff cell is credited. The `SpellActToRoom`, `SpellActNotVict`, and `SpellActToVictim` adapters for DP-1446 are already present on the base; neither draft changes Go runtime code.

## Remaining owners

At the combined draft projection, the original 58 gap-reachable activity-call-site units are closed; no gap-reachable activity-call-site units remain, and 77 stay gap-blocked. Their remaining owners are the stop-tier surfaces already named in `surface-inventory.tsv`: wizard/session, board and clan persistence, graph command DP-1361, interpreter/session identity, mail persistence/composing lifecycle, object-save persistence, OLC session delivery, special-procedure NPC dispatch, and spell-parser session delivery. Four `src/fight.c` sites remain blocked by switched-mob session delivery. Other retained combat transcript owners are `mob.troll-combat-dispatch` (`spec-procs.tsv:496`) and lifecycle/death/corpse teardown (`surface-inventory.tsv:62`). DP-1426 resolved the teleport-victim red. Dragon-breath draw parity is now CLEAN at seeds 1, 2, 3, 5, and 8.

## PR set and controls

Six PRs were in the DP-1371 batch: #2087 and #2089 are merged; #2088 (fight/message) and #2090 (spell matrix) remain drafts refreshed from `origin/main`; #2091 and #2092 were closed unmerged after their selected changes were carried into main. No PR was merged or deployed by this goal.

Each draft body carries its row deltas, classifications, C paths, retained evidence, and a revert control. The fight draft closes no `fight.c` residuals; the spell draft reduces its five rows by 8, 7, 2, 2, and 0 units respectively. The PR count is six.

## Census and gates

The shared full census for `3d4a35ea6` is retained at `~/Archives/darkpawns/oracle-runs/2026-10-16/batch-u/` and is CLEAN_AFTER_RECHECK for both full and claims verdicts. This goal ran targeted censuses only.

Individually passed on the fresh-main code tree: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`, `make fidelity-units`, `make string-census`, and `gosec -severity=high -quiet ./...`. `make fidelity-depth` passed separately on both draft trees (42 maps, 0 errors); `make fidelity-units` passed on both trees. The string-census ratchet passed with 98 known Go-only segments and 0 unreviewed; generated reports warned of 36 stale baseline entries and were left unchanged. `git diff --check` passed. On the refreshed fight worktree, the full gate set passed again after adding Trip, Circle, and Headbutt evidence: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth` (6,070 proven/delegated, 11 blocked, 65 excluded, 12 divergent-approved; 42 maps, 0 errors), `make fidelity-units` (1,960 claims PASS), `make string-census`, and CI's `gosec -severity=high -quiet ./...`. The string-census ratchet remains clean with its 36 stale generated-report entries untouched. `git diff --check` passed. The final targeted Headbutt censuses use `--draw-parity` at actual seeds 1, 2, 3, 5, and 8; each is CLEAN (2/2) with zero draw-divergent cases at Go HEAD `4e0a8e63a`.

**Total goal elapsed at this handoff refresh:** 44,255 seconds (12h 17m 35s). The six-PR count includes the two drafts, two merged PRs, and two closed unmerged PRs.
