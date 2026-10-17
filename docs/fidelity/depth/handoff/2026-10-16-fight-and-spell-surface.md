# DP-1371 fight and spell surface handoff — final follow-up 2026-10-17

## Result and inventory

The requested comparison starts at **2,106 blocked inventory units**. On the original goal baseline `3d4a35ea6`, the measured inventory is **1,470 units / 572 blocked**. The earlier #2088 and #2090 family PRs have since merged together into `main` at `4bf0899be`; current `origin/main` is **1,465 units / 567 blocked**. These final follow-up changes project to **1,392 units / 494 blocked**, 73 fewer blocked units than current `main` and 1,612 fewer than the requested baseline. Across the full effort, 52 newly counted live SkillMessage gaps were added while the earlier spell work removed 57 matrix gaps; these follow-ups close one SkillMessage gap and 72 spell-matrix cells.

| Inventory family | `3d4a35ea6` baseline | Final projection | Evidence |
|---|---:|---:|---|
| `lib/misc/messages` SkillMessage corpus | 0 counted units | 51 blocked | 83 C records: 32 complete, 50 gap-reachable, one gap-blocked |
| `src/fight.c` act/send map | 4 blocked | 4 blocked | 98 live calls: 94 proven, 0 reachable gaps, 4 switched-mob session-delivery gaps |
| CAST_SPELL matrix | 66 blocked | 10 blocked | 93 proven, 9 reachable gaps, 1 blocked, 196 C-unreachable |
| CAST_POTION matrix | 103 blocked | 89 blocked | 1 proven, 13 delegated, 89 reachable gaps, 196 C-unreachable |
| CAST_SCROLL matrix | 103 blocked | 79 blocked | 24 delegated, 79 reachable gaps, 196 C-unreachable |
| CAST_WAND matrix | 103 blocked | 78 blocked | 1 proven, 24 delegated, 78 reachable gaps, 196 C-unreachable |
| CAST_STAFF matrix | 103 blocked | 93 blocked | 1 proven, 9 delegated, 93 reachable gaps, 196 C-unreachable |
| Activity call-site gaps | 77 blocked | 77 blocked | The prior 58 reachable gaps are closed; all remaining owners are named in the inventory |

The 299×5 spell matrix now classifies 96 cells as proven, 70 delegated, 348 gap-reachable, one gap-blocked, and 980 C-unreachable. Its remaining inventory is 349 units. The 58 reachable activity gaps were closed in merged PRs #2087 and #2089; 77 stop-tier gaps remain blocked and named. The duplicated `new_cmds`/`new_cmds2` rows were reconciled to the source-order maps in #2087.

## Fight messages

`surface/fight.c.tsv` enumerates all 98 live `act()`/`send_to_char()`/`stc()` sites: 94 proven, none gap-reachable, and four blocked at switched-mob session delivery. Its shared combat, death/corpse, and lifecycle owners remain named in `surface-inventory.tsv`.

The C corpus has 83 `M` records across 66 attack types. The 32 fully proven records include M@0141 Harm; 50 records remain gap-reachable and M@0045 Call Lightning remains gap-blocked because its only miss caller is the DP-1448 NPC/switch actor lane (`src/spec_procs.c:1490-1496`). The actor-special path was not changed. Harm now has four registered spoken-cast cases proving its miss, hit, lethal-death, and immortal-God triplets for attacker, victim, and room. Each case matches C and draw parity at seeds 1, 2, 3, 5, and 8. The miss reaches `do_cast`'s concentration-failure call; all cases then follow `mag_damage` → `damage()` → `SkillMessage` (`src/spell_parser.c:953-960,1085-1095,1332`; `src/magic.c:762`; `src/fight.c:1023-1095,1534-1545`; `lib/misc/messages:143-154`).

Retained manifests: `dp1371-harm-skill-message-seed1`, `seed2`, `seed3`, `seed5`, and `seed8` under `~/Archives/darkpawns/oracle-runs/2026-10-17/`; each is CLEAN (4/4) with zero draw-divergent scenarios. The four scenarios are annotated in `cast.tsv` and the 12 defined M@0141 audience/outcome cells are marked proven in `matrices/skill-message-matrix.tsv`.

## Spell routes and audited delegations

The five route proofs remain independent of per-spell matrix cells. Spoken, potion, scroll, wand, and staff dispatch are recorded in `spell-route.tsv`; the representative route layer cases prove item gates, charges, levels, recipient selection, and multi-spell dispatch before `call_magic` (`src/act.other.c:897-968`; `src/spell_parser.c:544-817`). The spell follow-up restores C's `MANUAL_SPELL` argument buffer through `CallMagicWithArgument` for spoken casts and item routes, including C's `one_argument` fill-word parsing (`src/interpreter.c:1251-1281`; `src/spell_parser.c:525,610-658,777-817,963-982`).

The new spoken proofs cover Heal, Mass Heal, Group Heal, Holy Shield, Group Invisibility, Earthquake, Acid Blast, Chill Touch, Control Weather, Remove Curse, and Remove Poison. The Control Weather wand fixture passes the manual argument `worse` through `do_use` and changes live weather state (`src/spells.c:997-1010`). The Create Food staff fixture reaches `mag_creations` twice for two room peers and verifies both caster lines and both created inventory objects (`src/magic.c:1968-1998`). The score output now matches C's visible fighting-target name and its thin-air fallback, with probes for visible and hidden targets (`src/act.informative.c:1318-1340`).

Remove Curse and Remove Poison are delegated for scroll and wand after source audit: each route names the same affected character, `mag_unaffects` ignores item level and save type, and `mag_alter_objs` returns for the NULL object argument (`src/magic.c:1834-1900`). Staff remains a gap for those spells because the spoken fixture has an unaffected room observer; C staff fan-out adds a separate `NOEFFECT` message for that observer. Potion cells remain gaps where the self-target recipient differs. Create Food staff is directly proven rather than delegated because its two-peer fan-out and repeated object creation are the behavior under proof. All five route census manifests (`dp1371-spell-surface-seed1/2/3/5/8`) are CLEAN (4/4) with zero draw divergence. Five-seed runs also cover mag-points, mag-groups, and mag-areas; Chill Touch has a CLEAN seed-1 run. Retained evidence is under `~/Archives/darkpawns/oracle-runs/2026-10-17/`.

## Activity owners and PR set

The 77 remaining activity-call-site units are blocked at named stop-tier owners: wizard/session, board and clan persistence, graph command DP-1361, interpreter/session identity, mail lifecycle, object-save persistence, OLC session delivery, NPC special dispatch, and spell-parser session delivery. Four additional `src/fight.c` sites are blocked at switched-mob session delivery. DP-1446 formatter behavior is represented in the existing spell proofs; no C source, oracle, or governing document was edited.

Eight PRs belong to the DP-1371 batch: #2087 and #2089 are merged; #2088 and #2090 were merged together at `4bf0899be`; #2091 and #2092 were closed unmerged after selected changes were carried into main. The final follow-ups are draft [#2093, Harm SkillMessage](https://github.com/zax0rz/darkpawns/pull/2093) and [#2094, spell-route matrix](https://github.com/zax0rz/darkpawns/pull/2094). No PR from this final increment was merged and nothing was deployed.

## Census and gates

Claude's shared combined census is retained at `~/Archives/darkpawns/oracle-runs/2026-10-17/batch-v2/`: 4,323 pairs, 1,258 deduplicated; full verdict CLEAN and claims CLEAN_AFTER_RECHECK. The five Harm runs and five spell-route runs above are the targeted censuses for these docs/scenario additions.

All required build, test, lint, formatting, deadcode, string-census, and gosec gates passed individually before the rebase; the rebase brought in the already-merged spell work and a docs-only census-name change. After rebasing, both worktrees passed `make fidelity-depth`, `make fidelity-units`, and `git diff --check`. Fight fidelity-depth reports 6,289 cases, 6,202 proven/delegated, 9 blocked, 66 excluded, and 12 divergent-approved; its 42 maps validate with 0 errors. Fight fidelity-units passes 1,946 claims. Spell fidelity-depth reports 6,298 cases, 6,211 proven/delegated, 9 blocked, 66 excluded, and 12 divergent-approved; the matrix has 299 rows and 0 errors, and its 42 maps validate with 0 errors. Spell fidelity-units passes 1,946 claims. String-census passes with 98 known Go-only segments and 0 unreviewed; its generated reports still note 36 stale baseline entries and were left unchanged.

**Elapsed goal time at final status read:** 20h 10m 44s (72,644 seconds). The eight-PR count includes the six earlier PRs and the two final follow-up drafts.
