# Depth handoff — teleport_victim — 2026-08-30

## Frontier and queue position

Started from refreshed clean `main` at `3bd4cface` after the merged
`brain_eater` slice.  `make fidelity-depth` reported 1,252 cases: 1,205
proven/delegated, 13 blocked, and 34 excluded (98.9% actionable).  The full
depth-testing guide and newest handoff were read.  Source-order inventory puts
`teleport_victim` first at active mob registration 14405; `con_seller` at mob
21246 is next.

## C-first path audit

- The procedure is `src/spec_procs3.c:225-237`, declared at
  `src/spec_assign.c:149` and registered as `ASSIGNMOB(14405,
  teleport_victim)` at `src/spec_assign.c:377`.  Gil-Glash is race 19
  (demigod), an intelligent C mobile.
- Its entry gates are `cmd == 0`, a non-null `FIGHTING(ch)`, and `AWAKE(ch)`;
  it emits no bytes on a rejected call.  `mobile_activity()` skips fighting
  mobiles before `MOB_SPEC` (`src/mobact.c:68-93`), so the reachable automatic
  combat seam is `perform_violence()` after the ordinary NPC attack loop
  (`src/fight.c:1898-2032`).  The Go combat engine's
  `MobSpecialFunc` seam is the corresponding caller.
- `do_action(ch, GET_NAME(FIGHTING(ch)), find_command("scoff"), 0)` reaches
  the `scoff` no-argument social because `lib/misc/socials` has no
  `char_found` line.  C therefore sends `"$n scoffs at the idea."` to the
  room; the typed fighter name is ignored.  If `can_speak(ch)` passes, the
  second room Act is `"$n says, 'You can't harm me, mortal. Begone.'"`.
- The final call is direct `call_magic(ch, FIGHTING(ch), 0,
  SPELL_TELEPORT, GET_LEVEL(ch), CAST_SPELL)`, not command casting.  Its
  shared spell path includes the NPC/PC target distinction, peaceful and
  sitting gates, private-room rejection, victim black-screen bytes, origin
  and destination fade Acts, transfer/combat teardown, and
  `look_at_room(victim, 0)` after arrival (`src/spells.c:168-217`).

## RED and GREEN proof

Added `cmd/dp-oracle-diff/scenarios/spec-proc-teleport-victim.txt`.  It uses
`empty-players`, `quiet-mobs`, registered mob 14405, stripped script, and a
fixed-room C-first combat vehicle.  A disposable `set-mob-aff 14405 1082504`
fixture clears only Gil-Glash's `AFF_DODGE` bit (bit 17), keeping the already
blocked shared combat-defense transcript outside this special slice.

The clean-main RED run on seed 1 showed Go's invented
`"Gil-Glash scoffs at you."` and omitted the C landing room look.  After the
special fix, `--show-oracle` on seed 1 is GREEN and confirms the intended
registered block executed: exact scoff, speech, black-screen, and destination
look bytes match.  Seeds 2, 3, 5, and 8 still differ in the pre-special hit
opener and then select different random teleport rooms because the shared
combat stream is already divergent.  A second honest aggressive-acquisition
vehicle also reached the special but remained red on that same downstream
combat transcript.  That real gap is recorded as
`mob.teleport-victim-combat-transcript` blocked after two attempts; it is not
fixed forward here under R5b/R5c.

## Go changes

- Replaced the placeholder `roomMessage`/player-only lookup with the actual
  combatant resolver and `spells.CastFromSpecial`, preserving player and mob
  `FIGHTING` targets and the direct native call_magic position gate.
- Routed the scoff through the C no-argument social's exact room Act and made
  the speech Act conditional on `MobInstance.CanSpeak()`.
- Corrected `MobInstance.CanSpeak()` to the literal C
  `intelligent_races[]` values from `src/constants.c:353-373`, rather than the
  unrelated player-race bitvector.
- Added the `LookAtRoomForSpell` bridge so shared `castTeleport` performs C's
  post-transfer `look_at_room(victim, 0)` for player victims without creating
  a spells-to-game import cycle.
- Added focused tests for entry gates, social/speech audience, non-intelligent
  speech suppression, mob-target resolution, successful transfer and combat
  teardown, and the sitting call_magic gate.  Added ten manifest rows.

## Gates and counts

Passed on this branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check`

Final depth report after this slice: 1,262 total; 1,214 proven/delegated; 14
blocked; 34 excluded; actionable completion 1,214/1,228 = 98.9%.

Rules applied: R1/R2/R3/R4/R5e, with the shared combat transcript explicitly
held at the R5b/R5c boundary.

## Next action

Commit this slice, open `glm/spec-teleport-victim`, and merge only after all
GitHub checks are green.  The next queue item is active `con_seller` in
`src/spec_procs3.c` / mob 21246; do not repick any claimed handoff row.
