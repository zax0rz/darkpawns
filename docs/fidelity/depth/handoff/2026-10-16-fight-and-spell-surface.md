# DP-1371 fight and spell surface handoff — 2026-10-16

## Result and inventory

The starting comparison requested by this goal is PF-075's **2,106 blocked units**. The shared batch had already advanced `origin/main` to `3d4a35ea6`; that measured tree has 1,470 total units and 572 blocked. The two refreshed draft PRs together project to **1,535 total units and 637 blocked**, a net reduction of **1,469 blocked units** against the requested 2,106 baseline.

The delta from the current main snapshot is +65 blocked units: the fight PR adds 78 units of live-dispatch evidence scope, because the helper-only corpus test on main does not satisfy this goal's real registered-command proof bar, while the spell PR closes 13 matrix residuals. This is a deliberate scope correction, not a claim that the live-dispatch corpus regressed. The pair remains in draft and is not yet current main.

| Inventory family | Current main | Updated draft(s) | Reason |
|---|---:|---:|---|
| `fight-skill / lib/misc/messages skill-message corpus` | 0 proven-already units | 78 blocked units | 83 C records are in scope; five are now proved through registered dispatch, and 78 still need their live command-to-damage proof. |
| `fight-skill / src/fight.c act-send family` | 4 blocked | 4 blocked | 98 live sites: 94 proven, 0 gap-reachable, 4 switched-mob session-delivery stop-tier sites. |
| `fight-skill / new_cmds skill/action output` | 0 | 0 | Reconciliation is already in merged PR #2087. |
| `fight-skill / new_cmds2 skill/action output` | 0 | 0 | Reconciliation is already in merged PR #2087. |
| `spell-matrix / CAST_SPELL` | 66 | 64 | Two additional five-seed spoken proofs. |
| `spell-matrix / CAST_POTION` | 103 | 96 | Seven verified delegations, including Protection from Good. |
| `spell-matrix / CAST_SCROLL` | 103 | 101 | Create Food and Meteor Swarm are delegated after source audit. |
| `spell-matrix / CAST_WAND` | 103 | 101 | Create Food and Meteor Swarm are delegated after source audit. |
| `spell-matrix / CAST_STAFF` | 103 | 103 | No staff spell cell was promoted by the failed exploratory fixture. |

The resulting 299×5 spell matrix classifies 39 spoken cells proven, 11 item cells delegated, 464 gap-reachable cells, one C-undefined-behavior gap-blocked cell, and 980 C-unreachable cells. Residual spell inventory is 465. The only item delegations are the seven potion cells and two each for scroll and wand; staff-wide fan-out, item-level duration changes, and other route-sensitive branches remain open.

## Fight message proof

`lib/misc/messages` contains 83 `M` records across 66 attack types; C accepts the whitespace-padded marker at line 1175 (`src/fight.c:152`), so the prior 82-record count was low by one. `skill-message-matrix.tsv` records every record and the attacker, victim, and room result for each defined death, miss, hit, and god outcome. Five records are now fully proven, all 12 outcome/audience cells per record:

- `M@0241` — Bash, through `do_bash`.
- `M@1284` — Tiger Punch, through `do_tiger_punch`.
- `M@1268` — Dragon Kick, through `do_dragon_kick`.
- `M@0257` and `M@0271` — the two Kick variants, through `do_kick`.

Each proof reaches the production command, `damage()`, and `SkillMessage`, with exact C bytes and zero draw divergence. The other 78 records remain gap-reachable; the existing corpus unit test remains supplemental. C parses the records in `src/fight.c:126-175`, dispatches at `src/fight.c:1023-1095`, and reaches the handler from `damage()` at `src/fight.c:1534-1545`. Command call sites are in `src/act.offensive.c:419-741`.

Fresh-main retained censuses `dp1371-final-surface-seed1`, `seed2`, `seed3`, `seed5`, and `seed8` each cover 20 scenarios: all Bash, Tiger Punch, Kick, and Dragon Kick outcomes, Dragon breath, neutral spell rescue, Protection from Good, and Water Breathing. Every run is CLEAN (20/20) with zero draw-divergent cases at Go HEAD `3d4a35ea6`. The five-run set is in `~/Archives/darkpawns/oracle-runs/2026-10-16/`.

## Spell routes and delegations

The five route rows remain separately proven. Spoken samples include the violent `mag_damage` rescue and targeted Water Breathing. The potion fixture dispatches Invigorate, Vitality, and Poison in order; the scroll fixture dispatches Bless, Heal, and Lightning Bolt. The wand cases cover targeted Invisibility and violent targeted Lightning Bolt. Staff cases cover Sleep and Lightning Bolt room fan-out. The existing five-seed route batch is `dp1371-spell-routes-final-seed1/2/3/5/8`; the two newly promoted spoken spells are also included in the fresh-main five-seed runs above.

Delegations are limited to C bodies whose selected branches ignore the item-only inputs: `mag_creations` ignores level, target, object, and save type (`src/magic.c:1968-1998`); Meteor Swarm reads caster level and room victims rather than the selected item arguments (`src/spells.c:1088-1127`); selected self-only `mag_affects` branches for Sense Life, Levitate, Chameleon, Protection from Evil, and Protection from Good preserve recipient, state, and output without reading item level or `SAVING_ROD` (`src/magic.c:1132-1181,1266-1271,1292-1297`). Scroll/wand/staff distinctions remain open where arguments or recipient fan-out change (`src/spell_parser.c:592-817`; `src/act.other.c:897-968`).

The staff Earthquake route experiment repeatedly failed before the C server completed startup (`bind: Address already in use`, classified INFRA). Its fixture and route row were removed, and no staff cell is credited. The `SpellActToRoom`, `SpellActNotVict`, and `SpellActToVictim` adapters for DP-1446 are already present on the base; neither draft changes Go runtime code.

## Remaining owners

At the combined draft projection, the original 58 gap-reachable activity-call-site units are closed; no gap-reachable activity-call-site units remain, and 77 stay gap-blocked. Their remaining owners are the stop-tier surfaces already named in `surface-inventory.tsv`: wizard/session, board and clan persistence, graph command DP-1361, interpreter/session identity, mail persistence/composing lifecycle, object-save persistence, OLC session delivery, special-procedure NPC dispatch, and spell-parser session delivery. Four `src/fight.c` sites remain blocked by switched-mob session delivery. Other retained combat transcript owners are `mob.troll-combat-dispatch` (`spec-procs.tsv:496`) and lifecycle/death/corpse teardown (`surface-inventory.tsv:62`). DP-1426 resolved the teleport-victim red. Dragon-breath draw parity is now CLEAN at seeds 1, 2, 3, 5, and 8.

## PR set and controls

Six PRs were in the DP-1371 batch: #2087 and #2089 are merged; #2088 (fight/message) and #2090 (spell matrix) remain drafts refreshed from `origin/main`; #2091 and #2092 were closed unmerged after their selected changes were carried into main. No PR was merged or deployed by this goal.

Each draft body carries its row deltas, classifications, C paths, retained evidence, and a revert control. The fight draft closes no `fight.c` residuals; the spell draft reduces its five rows by 2, 7, 2, 2, and 0 units respectively. The PR count is six.

## Census and gates

The shared full census for `3d4a35ea6` is retained at `~/Archives/darkpawns/oracle-runs/2026-10-16/batch-u/` and is CLEAN_AFTER_RECHECK for both full and claims verdicts. This goal ran targeted censuses only.

Individually passed on the fresh-main code tree: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`, `make fidelity-units`, `make string-census`, and `gosec -severity=high -quiet ./...`. `make fidelity-depth` passed separately on both draft trees (42 maps, 0 errors); `make fidelity-units` passed on both trees. The string-census ratchet passed with 98 known Go-only segments and 0 unreviewed; generated reports warned of 36 stale baseline entries and were left unchanged. `git diff --check` passed.

**Total goal elapsed at final handoff refresh:** 37,026 seconds (10h 17m 06s). The six-PR count includes the two drafts, two merged PRs, and two closed unmerged PRs.
