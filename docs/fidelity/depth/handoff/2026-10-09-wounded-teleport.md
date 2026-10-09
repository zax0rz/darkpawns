# Wounded teleport prerequisite

Stop tier: wizard teleport now renders the destination to a wounded player,
matching src/act.wizard.c:365-389 (R1/R5e/R5g). Zach approved this prerequisite
on 2026-10-09 after the corrected DP-1406/1414 combined census exposed it.

C calls look_at_room(victim, 0), bypassing do_look's position test and honoring
BRIEF. Go now calls existing cmdMovementLook. A second suppression existed in
RenderObservationMessages: room literals traveled through Act without ToSleep.
C look_at_room sends its room text through send_to_char (act.informative.c:725
onward), including to wounded/sleeping viewers. Room observations now mark that
delivery explicitly and use ToSleep in their existing single-recipient renderer.
No position is changed; blind/dark visibility and BRIEF remain in DoLookRoom.

Other readers (R5c): DoLookRoom/DoLookRoomAt, movement, goto, spell landing,
DP-1402 rescue and Lua room views share the renderer. Ordinary awake output
uses the existing formatter. Explicit do_look retains its own position gate.
transferCharacter still calls cmdLook despite C do_trans calling look_at_room
(act.wizard.c:339,357); this related transfer gap requires a separate follow-up.
The later cmdLook in wiz_movement.go is wizard dig's own view, not teleport.

Locks: cmdMovementLook snapshots the body under manager RLock, releases it,
then builds and renders the room view. Observation/Act/player getters retain
their existing locking. No World or player lock is held across session delivery.

TestTeleportMortallyWoundedRoomView uses a registered session, checks actual
terminal room title/exits and unchanged HP -7, mortal position and destination.
2026-10-09-wounded-teleport-controls.py separately reinstates cmdLook and the
room delivery suppression; each overlay must compile and fail its room-view
assertion, then pass restored (R5h).

Census evidence: ~/Archives/darkpawns/oracle-runs/2026-10-09/
dp-1406-1414-corrected-combined/summary.txt: 3453 pairs, both NOT_CLEAN.
Poison passes at seeds 1,2,3,5,8. Mortal reaches drowning and mortal tick at
all five seeds; only the teleport destination view differs. New full combined
census is required on the prerequisite plus the DP-1406/1414 tip.

Gates before commit: fmt, build, vet, full tests, game tests, lint, fidelity-depth,
all 1462 fidelity-unit claims, string-census, focused game/session race tests
and both compiling removal/restoration controls pass. Existing stale string
reports warning is inherited from main. Logs retained with the candidate
manifest snapshot for the final combined run.
