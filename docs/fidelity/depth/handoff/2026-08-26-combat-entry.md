## Dated Handoff: 2026-08-26 (combat-entry round)
- The combat-entry family (`hit`/`murder`, `kill` routing, `assist`, `rescue`)
  is captured in `docs/fidelity/depth/combat-entry.tsv`, 27/38 proven/delegated.
  Five of six scenarios were RED on pre-fix main; the fixes: do_hit's
  WAIT_STATE(PULSE_VIOLENCE+2) now fires even when damage()'s peaceful/newbie
  gates block the swing; StartCombat stands the ATTACKER to POS_FIGHTING at
  entry (C set_fighting, fight.c:223), which do_hit's "You do the best you
  can!" branch depends on; assist's gates were rewritten (C's two-space
  already-fighting text, NOPERSON, self gate, $M pronoun, TO_NOTVICT room
  exclusion, and the immediate hit() swing with its no-wait quirk); rescue's
  resolution failure answers with the same prompt as no-arg and the
  nobody-fighting line uses $M.
- The defender-side POS_FIGHTING stand is deliberately deferred to a
  retaliation/damage-matrix round — standing the defender at entry interlocks
  with the damage path's position model (AWAKE/wimpy gates) and several
  engine tests model downed combatants via enrollment.
- Scenario shape notes: a gate-blocked swing still costs the attacker the
  round on both servers, so queued commands lag symmetrically behind the
  frozen clock — keep the wait-setting command last or accept identical lag;
  NEVER pump pulses in a room with a live fight unless you want violence
  rounds in the transcript.
