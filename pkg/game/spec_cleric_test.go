package game

// TestSpecCleric_EntryGatesAndStand covers SPECIAL(cleric)'s AWAKE, command,
// and C do_stand entry path (spec_procs.c:1425-1442). A sleeping cleric must
// not stand; a sitting cleric emits the room-only do_stand line before acting.

// TestSpecCleric_Lspell12IsAnOffensiveNoOp pins the intentional hole in C's
// offensive switch: lspell 12 has no spell case (spec_procs.c:1558-1578).

// TestSpecCleric_EarthquakeUsesNPCRoomMessage covers the lspell 17-19 arm
// and call_magic's NPC area-spell room narration (spec_procs.c:1558-1578;
// magic.c:1573-1579).

// TestSpecCleric_CallLightningWeatherGate pins the OUTSIDE/sky/level/roll
// chain before the call-lightning cast (spec_procs.c:1570-1575).

// TestSpecCleric_BlindnessGateConsumesCDraw pins C's bitwise '&' expression:
// an innately blind cleric consumes Number(0,3) even when lspell < 4, then
// proceeds to its ordinary self-heal draw (spec_procs.c:1579-1605).
