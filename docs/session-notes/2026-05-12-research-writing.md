# Session Notes — 2026-05-12

## Research Writing (Program 5)

Wrote draft: `docs/research/drafts/2026-05-12-silent-drift-port-fidelity.md`

**Topic:** Silent port drift — the classSpells audit as anchor case study.

**Key arguments:**
- 30% of our confirmed findings (51/170) are port-specific — they don't exist as a category in single-codebase analysis
- Static analysis can't catch semantic divergence between two codebases
- Fidelity audit methodology: compare ported subsystem against authoritative source, classify each divergence
- classSpells: Go had 50 Mage spells, C has 27. Compiles fine. Runs wrong.

**Next steps for this draft:**
- Add C vs Go comparison table (side-by-side spell entries)
- Add section on BRENDA's rebuild process (how the fix was done)
- Consider expanding to cover the other drift categories with examples

**Posted to #dark-pawns.** Research log updated.
