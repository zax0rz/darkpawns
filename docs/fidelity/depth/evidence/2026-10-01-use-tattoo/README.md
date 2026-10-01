# E2 tattoo proof evidence

Candidate source commit: 52d2f631f5165aa7b216772b756e42fb09920da3.
Base: a192eda61. R1, R3, R5e, R5g, R5h.

- revert-triples.tsv: 34 independent unit mutations, each assertion-only 0 -> 1 -> 0.
- oracle-revert-triples.tsv: eight live paired vehicles, each content-only 0 -> 1 -> 0 with dispatch disabled.
- oracle-claims.tsv: all 40 tattoo seed/scenario pairs PASS at seeds 1,2,3,5,8.
- source-sha256.tsv: candidate source hashes, also verified against the restored isolated proof checkout.
- census-summaries.txt: exact final full/claims summary lines.

Full source, logs, baseline-red, hashes and gate outputs are retained at
~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-tattoo-proofs/final-source-v3/.
Live mutation census evidence is in sibling dp-1371-p4-tattoo-oracle-mutant/
and dp-1371-p4-tattoo-oracle-restored/; the original green is
dp-1371-p4-tattoo-targeted/. The standalone mutant skull report shows
activation/cooldown replaced by item-missing bytes.

To reproduce unit triples, create an isolated detached worktree at the
original base a192eda61, then run revert_proofs.py with three arguments:
the candidate checkout, that isolated worktree, and a new evidence directory.
The script copies only Go candidate files, verifies an assertion-red on the
base, and restores every source file even when a mutation is rejected.
Do not run it against a primary checkout. It never modifies the C oracle.
