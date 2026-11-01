# The witness audit finds hollow proofs the record never noticed

**Date:** 2026-11-01. **Source:** Sol's witness audit (`BRIEF-2026-11-01-audits.md`), reviewed by Claude. **Ledger:** PF-095, PF-096.

The screening register counts the checking failures the project noticed. The witness audit looked for unnoticed ones: does each scenario's C output show what the scenario claims? It read the retained v0.1.8 release census (`~/Archives/darkpawns/oracle-runs/2026-10-31/v0.1.8-release/`), with no new runs.

## Most scenarios are honest

In a random sample of 150 of the 1,841 scenarios (seed `20261102`), 142 witnessed their claim, 2 did not, 4 were refused and 2 were unclear. That's 4% failing, below Sol's prediction of 5–15%, which was written before any output was read.

## The spell-item matrix: 52 scenarios, 269 rows, a release

A mechanical sweep of all 1,841 found a cluster the sample missed. In 52 `spellmatrix-*` scenarios, the builder character's first setup command, `oedit 4399`, gets `Huh?!?` from both servers, and every following editor line gets `Huh?!?` too. The item is never configured. The scenario then quaffs, recites or uses the unconfigured prototype, so the spells it names (fireball, harm, lightning bolt and so on) never fire. Both servers refuse identically, so the comparison passes.

- 269 manifest rows in `docs/fidelity/depth/` cite these scenarios as `oracle-green-multiseed` proof.
- The refusal is present in every retained run of the scenario from the day the matrix was added (`351418e87`, 2026-10-18), including the R5h runs of that work.
- The rows shipped in release v0.1.8.

This is the peer-login mechanism (RO-012, September: a setup step refused, two refusals compared) recurring a month later in another surface. STUDY.md's setup rule makes it group C.

## Transfer: a new test hollowed an old one

`transfer-depth` printed `No-one by that name here.` for its not-found probe in every retained run through 2026-10-15. From 2026-10-16 on, every transfer probe gets `Huh?!?`. The boundary is `4c9e21d26` ("test: prove wizard transfer refusal outputs"), which added a low-level character to the scenario's setup; after it, the actor no longer has the level for a wizard command. Three rows, including `transfer.success-audience`, still cite the scenario as `oracle-green-multiseed`. Adding a proof broke an existing one, and the census stayed green.

## What it means for the study

- Noticed C incidents: 5 in the screening register. Unnoticed, found by one audit: a 52-scenario cluster, plus 12 A-candidate scenarios such as transfer. The ratio is itself a finding about how much a green census can hide.
- For RQ4: the peer-login fix in September didn't stop the class from recurring. No check asserts that setup succeeded.
- Neither finding is fixed here. Both need a harness check and re-proof of the affected rows, tracked as port work.
