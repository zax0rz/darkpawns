## Dated Handoff: 2026-08-26 (affects-state round)
- The deferred affect-state items landed: mag_affects' two pre-apply gates are
  ported (magic.c:1387-1404) — the mob-innate-flag gate and the
  affected-by-non-accumulating-spell gate both answer NOEFFECT to the caster —
  and waterwalk/change-density now carry C's always-20 duration quirk
  ("4+reag?20:0" parses as (4+reag)?20:0). magAffectsApply builds affects into
  a pending list so the gates run after the save-gated case arms (C's
  evaluation order) but before anything is applied; a per-spell gate table
  (`affectGateFlags`) mirrors C's accum flags and first-two bitvectors.
- Proof: recite-armor-reapply is RED on pre-fix main (Go re-applied armor and
  repeated the to_vict line) and GREEN with the gate; unit tests pin both
  gates and the duration quirk; the object-magic scenario set re-ran GREEN.
  One spawn-obj fixture line loads exactly one object (max-existing is the
  cap, not the count) — multi-copy scenarios need one line per copy.
- Still deferred with named owners: the slow AFF_HASTE bit and MIND_BAR's
  AFF_NOTHING (Go writes invented AFFSlow/AFFMindBar bits) — both interlock
  with the combat engine or affect display and belong to a dedicated
  affect-bits round.
