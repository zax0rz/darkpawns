# Section seed: when the tests are the bug

2026-11-06. Zach proposed a paper section on regression tests, mutation testing, and tests that lock in wrong behavior. This is an outline from rows already in `EVIDENCE_LEDGER.tsv`, not a draft. Every claim below is a ledger row; check each row's status before quoting it.

## The question

Seven months of AI-written tests sit beside the port: about 196,000 lines of them, against about 165,000 lines of Go. How often does a passing test, or a green census, prove what it claims? And when it doesn't, what shape does the failure take?

## A working taxonomy, with evidence

1. **Tests that pin a divergence from C.** The test is correct about the code and wrong about the game, so fixing the game turns the test red.
   - `counter_procs` duplicate in combat (PF-103).
   - Mirror image's clone level (PF-107).
   - Spell alignment's sign check (PF-108).
   - Three tests written to consume an invented "appears." line (PF-037).
   - A Lua stub that encoded the implementer's guess at the Mudlet API (MA-006).
   - The pattern: an auditor reads each one as a keeper, or as redundant junk to delete. Either way the divergence survives.
2. **Tests that can't fail.** Mutations in the code they name stay green:
   - the three spells cases in PF-108;
   - the five survivors of the 100-symbol unit sample (PF-052);
   - a fidelity fix tested in a function nothing calls (PF-003).
3. **Scenarios that pass without running.** The census compares two identical non-events:
   - refused peer logins (RO-012);
   - 52 spell-item scenarios whose editor never ran, backing 269 rows in a release (PF-095);
   - a new test that hollowed an old one (PF-096);
   - a death scenario where the character always fled (PF-013);
   - casters silent on both sides (PF-039).
   - Rule R5j came from this class.
4. **Fixtures that silence a whole surface.**
   - Quiet-mobs hid literal `$n` on every city mob (PF-100).
   - No-cost practice classes hid a missing column (PF-099).
   - Rule R5i came from this class.
5. **Comparisons too coarse to see the bug.**
   - Matching transcripts over diverging draws (PF-068).
   - A comment that was meant to be a directive, so color was never compared (PF-015).
   - Prompts that are uncertified by construction (RO-011).
6. **An expected divergence with the wrong cause attached.** A pinned divergence kept a data-loss bug green under a note that blamed C (RO-009).
7. **What the tests and census can't see at all.**
   - Human playtests found what a green census missed (PF-008, PF-054).
   - So did a second transport (RO-008).
   - So did Go-only state with no C counterpart (PF-053).
   - So did identity-conditional code (MA-010).

## Measurement, not anecdote

- **Random mutation samples.**
  - Census catch rate 23 of 40 for player-output functions (PF-102), against 7 of 7 for probes an agent chose itself.
  - 94 of 100 for unit proofs (PF-052).
  - Round three, in pkg/session and pkg/spells, is running now (Luna).
- **Blind test audits by three model families** (PF-103, PF-104, PF-107, PF-108).
  - Kappa ranges from 0.30 to 0.70 by pair and package.
  - Most weak tests predate the first oracle (2026-07-12).
  - Each auditor alone found real divergences the others marked keep.
- **The rule trail:** R5h, R5i and R5j each came from a recurrence of one of the classes above.

## Open questions for the section

- Is "a test that pins a divergence" more common in tests written before the oracle, or only easier to find there?
- Do mutation survivors and audit "F" marks pick out the same tests? The spells audits and the round-three sample cover the same package, so this can be tested directly.
- How much of this is the AI author, and how much is any team writing tests without a reference implementation to check against? The answer needs a comparison point.

## Related

- [Field note, 2026-11-06](../field-notes/2026-11-06-the-frontier-pass-finds-the-live-divergences.md).
- `docs/fidelity/RULEBOOK.md` R5h–R5j.
