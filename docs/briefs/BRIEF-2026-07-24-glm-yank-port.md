# BRIEF (glm) — DP-1214: port `do_yank` byte-for-byte (currently a loose paraphrase)

**Owner:** glm-5.2. **Gate:** byte-exact unit tests per branch now; Claude can add
a `yank` oracle fixture later (God/skillset harness with a follower mob). CI green.
**Git:** branch off `main` as `glm/yank-port`. `git fetch origin && rebase onto
origin/main` first; confirm merge-base == origin/main tip. Edit → commit → push →
PR. Do NOT merge. Sized S.
**Finding:** DP-1214. `yank` (haul a sitting **follower** to their feet — e.g. after
a trip) is registered/reachable in Go but its player-facing bytes were rewritten,
not ported. Same class as `grats`/`gratz` (R2a); surfaced from player memory, not
the oracle (nothing exercises `yank`).
**Cite:** C `src/act.other.c:1620-1662` (do_yank), `NOPERSON` global; Go
`pkg/game/other_mount.go:99` (`doYank`). Rules **R1/R2a/R4/R5e**.

---

## The C truth (port this exactly — R5e, read it yourself)
`do_yank` (`act.other.c:1620-1662`), in order:
```c
one_argument(argument, arg);
if (!*arg)                          send_to_char("Who do you wish to yank?\r\n", ch);
else if (!(victim = get_char_room_vis(ch, arg)))  send_to_char(NOPERSON, ch);   // "No-one by that name here.\r\n"
else if (victim == ch)             send_to_char("That's wierd.\r\n", ch);       // sic: "wierd"
else {
  if (victim->master != ch) { send_to_char("That probably wouldn't be appreciated.\r\n", ch); return; }
  if (GET_POS(victim) > POS_SITTING) {
     if (!IS_MOUNTED(victim)) act("$N is already on $S feet.", TRUE, ch, 0, victim, TO_CHAR);
     else                     act("You can't yank $M off $S mount!", TRUE, ch, 0, victim, TO_CHAR);
     return;
  }
  if (GET_POS(victim) <= POS_SLEEPING) { act("$N is is no position to be yanked around!", TRUE, ch, 0, victim, TO_CHAR); return; }  // sic: "is is"
  act("You yank $M to $S feet.",     TRUE, ch, 0, victim, TO_CHAR);
  act("$n yanks you to your feet.",  TRUE, ch, 0, victim, TO_VICT);
  act("$n yanks $N to $S feet.",     TRUE, ch, 0, victim, TO_NOTVICT);
  GET_POS(victim) = POS_STANDING;
}
```

## What Go's `doYank` gets wrong (the whole set — see DP-1214 table)
Every error string is paraphrased and two branches are missing. Port to exact C:

| case | must emit (C) | Go now (wrong) |
|---|---|---|
| no arg | `Who do you wish to yank?\r\n` | `Yank whom from what?\r\n` |
| not found | `NOPERSON` (`No-one by that name here.\r\n` — the existing global; do NOT invent) | `They aren't here.\r\n` |
| `victim == ch` | `That's wierd.\r\n` *(keep the typo)* | **missing** |
| not your follower (`victim->master != ch`) | `That probably wouldn't be appreciated.\r\n` | `They aren't following you!\r\n` |
| already up (`GET_POS(victim) > POS_SITTING`) | not mounted → `$N is already on $S feet.`; mounted → `You can't yank $M off $S mount!` | `They're already on their feet.\r\n` (no mount branch; wrong threshold) |
| `GET_POS(victim) <= POS_SLEEPING` | `$N is is no position to be yanked around!` *(keep "is is")* | **missing** |
| success | the 3 `act()` lines above, then `POS_STANDING` | uses the NAME where C uses `$M`/`$S` pronouns — verify |

## Port notes (verify each — R5e)
- **Branch ORDER and the exact `\r\n`** must match C. The self-check (`victim==ch`)
  comes BEFORE the follower check; the already-up check BEFORE the sleeping check.
- **Position threshold:** "already up" is `GET_POS > POS_SITTING` (so FIGHTING and
  STANDING both count as up), NOT `>= PosStanding`. A sitting follower (POS_SITTING)
  is exactly the yankable case; the sleeping/below case (`<= POS_SLEEPING`) is
  rejected with the "is is" line. So the yankable window is `POS_RESTING`/
  `POS_SITTING` (above SLEEPING, at or below SITTING). Match C's bounds precisely.
- **Follower relationship:** C requires `victim->master == ch` (the victim is
  *your* follower/charmed pet). Confirm Go's equivalent (`GetFollowing()`/master)
  expresses the same relationship, not merely "is following."
- **`act()` codes for the success + already-up lines:** `$N` = victim NAME,
  `$M` = victim objective pronoun (him/her/it), `$S` = victim possessive
  (his/her/its). So `"You yank $M to $S feet."` renders `"You yank him to his
  feet."` — a **pronoun**, not the name. Use Go's `act`/`ActMessage` port so
  `$M`/`$S` expand correctly (mirror how other ported `act()` sites do it); do NOT
  substitute the victim's name. Verify the exact expansion (visibility rules) against
  how Go renders `$M`/`$S` elsewhere.
- **Mount guard:** the already-up branch splits on `IS_MOUNTED(victim)` — port that
  sub-branch (the "You can't yank $M off $S mount!" line) using Go's mount check.
- On success set the victim to `POS_STANDING` (already correct in Go).

## Tests (byte-exact, one per branch)
Assert the EXACT bytes incl. `\r\n` and the typos (`wierd`, `is is`): no-arg,
not-found (NOPERSON), self, not-follower, already-up (both mounted + not),
sleeping, and success (the 3 messages with correct pronoun expansion + POS_STANDING).
Mirror an existing `act()`-based command test for the pronoun rendering.

## Gate
No oracle coverage today. Unit tests are the interim gate; note in the PR that
Claude may add a `yank` oracle fixture. **R5c:** while here, note (don't fix) any
sibling follower/position/social commands that look similarly paraphrased — a
candidate audit class.

## Guardrails
- **Never** edit `src/`, `darkpawns-c-oracle/`, `lib/` — read-only.
- All gates (GLM.md §gates): `make fmt`, build, vet, `test ./... -race`,
  `golangci-lint run`, `make reachability` (`yank` stays `registered`).
- Don't stage `.zcode/`, generated reachability reports,
  `website/static/map/world-sphere.json`, `docs/reports/reek/*`.

## Deliverable
`doYank` ported byte-for-byte from `do_yank`: exact strings (incl. `wierd`/`is is`
typos), all branches (self, follower-master, already-up + mount sub-branch,
sleeping), NOPERSON on miss, correct `> POS_SITTING` threshold, the 3 success
`act()` lines with `$M`/`$S` pronoun expansion, `POS_STANDING`, and per-branch
byte-exact tests.
