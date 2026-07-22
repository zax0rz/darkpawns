# BRIEF — Combat command message + gate parity (`hit` / `kill`)

**For:** codex (frontier). **Owner of gate:** Claude (oracle red→green + review vs C).
**Branch:** `refactor/combat-command-messages` off `main`.
**Deterministic / Tier-1** — no RNG, no live-combat harness needed (that's the separate B-2 track).
**Method rules:** read `src/act.offensive.c` `do_hit` (101-131) + `do_kill` (134-161) and the
`damage()` gate ladder (`src/fight.c` ~1330-1360) directly. Gate = oracle red→green + unit tests.

---

## 0. Context
DP-1163 (PR #351) fixed room-flag lookups, so the combat *gates* now fire. This brief closes the
remaining **message-text drift** in the player `hit`/`kill` command path (pkg/session/combat_cmds.go)
against C. Pure string/branch parity — the faithful gate logic already exists; the wording is off.

## 1. Oracle-PROVEN RED (verified 2026-07-15, ×3)
`hit <absent-target>`:
```
-They don't seem to be here.      (C do_hit, act.offensive.c:110)
+They aren't here.                (Go cmdHit)
```
Go uses `do_kill`'s wording for `do_hit`. They are **different** in C (see §2/§3).

## 2. `cmdHit` → align to C `do_hit` (act.offensive.c:101-131), exact strings
- no arg: `"Hit who?\r\n"` (Go has "Hit whom?").
- not found (`!get_char_room_vis`): `"They don't seem to be here.\r\n"` (Go has "They aren't here." — the RED).
- self (`vict == ch`): `"You hit yourself...OUCH!.\r\n"` (note the trailing `.` after `OUCH!`) **and**
  `act("$n hits $mself, and says OUCH!", FALSE, ch, 0, vict, TO_ROOM)` — verify Go emits both.
- charm-friend (`IS_AFFECTED(ch, AFF_CHARM) && ch->master == vict`):
  `act("$N is just such a good friend, you simply can't hit $M.", FALSE, ch, 0, vict, TO_CHAR)`.
- otherwise: if `POS_STANDING && vict != FIGHTING(ch)` → start combat (dismount first if mounted);
  else `"You do the best you can!\r\n"`. **C has no "You're already fighting!" string** — Go invented
  it; replace with C's `"You do the best you can!\r\n"` branch semantics.
Use `Act()` for the `$n/$N/$M/$mself` forms (F0a) so pronouns are correct, not Sprintf.

## 3. `cmdKill` → align to C `do_kill` (act.offensive.c:134-161)
- mortal (`GET_LEVEL(ch) < LVL_IMPL-1 || IS_NPC`) → delegate to `do_hit` (Go already does — keep).
- immortal (level ≥ LVL_IMPL-1) path, exact strings:
  - no arg: `"Kill who?\r\n"`.
  - not found: `"They aren't here.\r\n"` (**note: `do_kill` really does say this — different from
    `do_hit`; don't "unify" them**).
  - self (`ch == vict`): `"Your mother would be so sad.. :(\r\n"`.
  - equal level (`GET_LEVEL(vict) == GET_LEVEL(ch)`): `"No can do, buddy.. \r\n"` (⚠ **trailing space**
    before `\r\n` — Go currently `"No can do, buddy..\r\n"` drops it).
  - else: `act("You chop $M to pieces!  Ah!  The blood!", FALSE, ch, 0, vict, TO_CHAR)` (⚠ **two
    spaces** `pieces!  Ah!`), `act("$N chops you to pieces!", FALSE, vict, 0, ch, TO_CHAR)`,
    `act("$n brutally slays $N!", FALSE, ch, 0, vict, TO_NOTVICT)`, then `raw_kill`. Go uses Sprintf
    with `%s` and single spaces + omits the TO_NOTVICT room line — switch to `Act()` and match spacing.

## 4. Gate ladder (verify, likely already correct post-DP-1163)
`damage()` order (fight.c ~1336): peaceful (unless victim outlaw / already fighting ch) → attacker
newbie (`GET_LEVEL(ch)<=10` vs PC) "You are not experienced enough to attack $N!" → victim newbie
"Ancient forces protect $N from your wrath!" → shopkeeper `"Ha ha... Don't think so.\r\n"`. Confirm
Go's cmdHit reproduces these strings/order; add the shopkeeper gate if missing.

## 5. Acceptance gate
1. **Oracle:** `--scenario combat-round` stays green (peaceful), **plus** a `hit nobody` probe →
   `"They don't seem to be here."` (Claude will add the probe when gating).
2. **Unit tests** (exact C strings): every message in §2/§3 incl. the trailing-space/double-space/
   trailing-period quirks; self-hit room broadcast; charm-friend; immortal chop trio via Act().
3. `make check-fmt vet` + `go test ./...` green; import guard green; no WS schema break.

## 6. Gotchas
- **`do_hit` "They don't seem to be here." ≠ `do_kill` "They aren't here."** — keep them distinct.
- **Whitespace is fidelity:** `"buddy.. \r\n"` (trailing space), `"pieces!  Ah!"` (two spaces),
  `"OUCH!.\r\n"` (trailing period). Copy from source; don't normalize.
- **Use `Act()`** for all `$n/$N/$M/$mself` — no Sprintf pronoun guessing.
- Deterministic only — do NOT touch combat RNG/draw-order (that's B-2, still blocked on harness).
