# Item sleep proof evidence

Base: fresh origin/main `78a0a2785` (after #1741). No production behavior,
harness, shipped world, save format, reference binary or governing document
changes. The only production-source edit corrects the stale reachability comment.

`TestItemSleepEntryDepth` drives real cmdQuaff with a level-40 sleep potion and
level-8 mage, independently crossing outlaw, exact-vnum sand and saving outcome
controls at seeds 1,2,3,5,8 (40 combinations). Opposite ROD/SPELL save modifiers
make the cast type observable. Raw next-draw equality checks zero draws on
outlaw refusal and exactly one draw on allowed sleep. The real reagent consumer
must remove 1226 before the outlaw gate/save and preserve keyword-matching 1227.
The test also checks exact self/observer spell messages, position/affect bit and
duration 6 or 7 (caster level, not potion level), potion extraction and 20 wait
pulses. Quaff audiences and wake bytes are independently live-proven.

`revert-triples.tsv`: 16 assertion-only 0/1/0 triples. `revert_proofs.py` runs
in `/home/zach/dp-p4-item-sleep-proof`, an isolated detached origin/main checkout
with the candidate test/comment copied in. Mutation stages never alter the
concurrent census source. Logs are retained under
`~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-item-sleep-proofs/`.

The live vehicle uses existing OLC commands in paired disposable runtime copies:
4399 remains its real potion type/name, but gets values 40/38/-1/-1 in memory.
No disk save or repository world edit occurs. A mortal drinks it, with the God
as named observer. Four separate outlaw/refusal × sand/bare scenarios compare
actual quaff, component narration, both self-refusal messages, save/sleep room
messages, consumed inventory and wake. `oracle-claims.tsv` records inspected C
captures at all 20 claimed pairs, including both saving outcomes.

`oracle_proofs.py` uses only census start/wait. It runs the remaining final seed
sets, then removes sleep dispatch temporarily: every vehicle must content-FAIL,
and the restored source must PASS. `oracle-revert-triples.tsv` records the four
live 0/1/0 triples; no compile error, infrastructure issue or timeout is a proof.
Census wrapper manifests/captures and separate attempt logs are retained under
`dp-1371-p4-item-sleep-*`. Exact normal/restored/mutant summary lines are in
`census-summaries.txt`. Seed 2 also rechecks sleep-spell-depth because this PR
corrects its misleading comment without changing the cast vehicle.

The proof stops at standing PC potion self-targets. General call_magic room,
position and command-cast target gates, NPC NOSLEEP/retaliation, and shared
spell visibility remain owned by their existing manifests. The level-window
refusal is unreachable for this self-target because caster and victim are the
same character; the independent cast row retains its non-self level-window
unit proofs. TAR_NOT_SELF is checked in cast_spell, never this item entry.

A class search found the stale active reachability claim in the Go comment,
sleep-spell-depth comment and object-magic manifest; those are corrected here.
The historical 2026-08-28 handoff is preserved, superseded by the new dated
handoff and actual C self-quaff captures (R5e/R5f/R5g/R5h).
