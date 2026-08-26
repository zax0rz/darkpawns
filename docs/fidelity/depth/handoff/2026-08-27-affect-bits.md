# Dated Handoff: 2026-08-27 (affect-bits round)

- The last two deferred affect-bit rows converted: SPELL_SLOW now applies
  engine.AFFHaste (C magic.c:1051 sets bitvector AFF_HASTE, so a slowed
  combatant carries the haste bit and fight.c's attacks++ haste check fires —
  the AFF_SLOW attacks-- check never does, because the spell never sets it;
  slow makes you swing more in C), and MIND_BAR is now the pure -18 INT stat
  affect C applies (APPLY_INT, bitvector AFF_NOTHING, no status flag) instead
  of ApplyNone with an invented AFFMindBar bit.
- Unit-proven via the golden table (no item vehicle exists for either spell);
  the engine's existing AFF_HASTE reader lands the slow quirk with no further
  wiring.
- Process change recorded in the charter amendment (same PR): green + evidence
  + rebased glm/depth-* PRs are now self-merged by the loop agent under a
  written protocol, with post-hoc model review of merged PRs; DEPTH_TESTING.md
  dated sections moved to per-round files under docs/fidelity/depth/handoff/
  so consecutive rounds no longer share an insertion point.
