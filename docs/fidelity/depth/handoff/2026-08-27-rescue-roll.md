# Dated Handoff: 2026-08-27 (rescue-roll round)

- rescue.roll-success-fail converted: the live fight + number(1,101) roll is
  proven across 8 seeds with BOTH arms (skill 75 via the God-skillset warmup:
  seeds 1-4,7-8 succeed with the Banzai trio and the fight redirect; seeds
  5-6 print "You fail the rescue!" and change nothing).
- ROUND-13 FIX: DoRescue's victim line had its $N roles flipped — C's
  act("You are rescued by $N...", FALSE, vict, 0, ch, TO_CHAR) makes the
  message's "ch" the VICTIM and $N the RESCUER, so ActMessage needs
  (victPronouns, &chPronouns), not (chPronouns, &victPronouns). RED pre-fix
  on seed 1: the victim saw "You are rescued by Victimxo".
- Pattern note for act() ports: whenever C flips the act() subjects for a
  TO_VICT/TO_CHAR line (ch=vict, vict_obj=ch), the ActMessage call must swap
  its pronoun arguments too — grep for other flipped-subject act calls when a
  new one lands.
