# Dark Pawns Research Log

[DESIGN] [2026-04-24] Initial thought on the game...

## 2026-04-24 — Wave 8 Complete: utils.c → logging.go

[RESULT] [DESIGN]

**Status:** Ported src/utils.c (~980 C lines) → pkg/game/logging.go (392 Go lines).
9 functions: BasicMudLog, Alog, MudLog, LogDeathTrap, Sprintbit, Sprinttype, SprintbitArray, DieFollower, CoreDump.

**Key insight:** Bridge pattern (ImmortalSessionProvider interface) avoids circular imports between game + session packages — critical pattern for remaining waves since game and session have deep coupling.

**DeepSeek V4 Pro note:** Auth keys failed for Anthropic-API-style routing. Subagent fell back to deepseek-chat (litellm) and completed work correctly. V4 Pro key needs `/anthropic/v1` base URL with valid key.

## 2026-04-24 — Wave 5 Complete: Character Management & Affect Lifecycle

[RESULT] [DESIGN]

**Status:** All 30 C functions from handler.c, limits.c, magic.c mapped to Go equivalents.

## 2026-04-24 — Wave 7 Complete: Spell System

[RESULT] [DESIGN]

**Status:** magic.c + spells.c + spell_parser.c (~4,843 C lines → 1,846 Go lines) fully ported into pkg/spells/.
8 files: call_magic.go, damage_spells.go, affect_spells.go, affect_effects.go, spell_info.go, saving_throws.go, say_spell.go, object_magic.go.

**Key decisions:**
- Interface duck-typing (interface{}) over type imports to avoid circular deps
- MagRoutine constants prefixed `Routine` not `Mag`
- Direct editing > subagents for multi-file porting

**Open questions:**
- CallMagic not yet wired into session/cast_cmds.go
- Group/mass/area/summon/creation/alter-obj stubs unfleshed
- ExecuteManualSpell needs real implementations
- engine.AffectManager not connected to spells yet

## 2026-04-24 — Wave 6.5: Command Wrapping

[RESULT] [DESIGN]

**Status:** 22 player-facing commands from act.other.c wired. All World.doXxx have session-level wrappers.

## 2026-04-24 — Wave 6: Wizard Commands

[RESULT] [DESIGN]

**Status:** 46 wizard commands registered in session/wizard_cmds.go (1,574 lines).

## 2026-04-24 — Wave 3: Fsm/Fly/Scry/Swim/Scribe/Sight/Thirst/Track/Visibility Updates

[RESULT] [DESIGN]

**Status:** Spec procedures from act.movement.c, spec_procs.c, spec_procs2.c ported to Go.
9 procedures ported: specFly, specFsm, specScry, specSwim, specScribe, specSight, specThirst, specTrack, specVisibility.

## 2026-04-24 — Wave 3: Spec Procs 3 + Door Wiring Fixes

[RESULT] [DESIGN] [FAILURE]

**Status:** Magical spec procs completed — specProg4, specWater, specCold, specFire, specDisenchant, specMorph, specAcid, specHaste.

**Failure:** TestMudProgDoor was passing but broke after door wiring. Root cause: Doors test expected `*command.DoorCommand` but door commands are plain functions matching `func(world *game.World, ch *game.Player, arg string)`. Test fixtures needed update.

## 2026-04-24 — Wave 4: Blockers (Doors, Shops, Rescue, Hit/Damroll, Spell Effects)

[RESULT] [DESIGN]

**Status:** 19 C functions ported across multiple files. Door manager + bashdoor implemented. Shop, rescue, hitroll/damroll, spell effects all ported.

## 2026-04-25 — Wave 4/5 Merge: Follower Handling in Party

[OBSERVATION] [DESIGN]

**Status:** Player struct uses `Following string` (name) and `GetFollowers() []string` — not C-style linked list. Good abstraction.

## 2026-04-25 — Port Strategy

[HYPOTHESIS]

The architecture is converging. Game core (World, Player, Room) stable. Session/command layer mostly wired. Remaining: comms (Wave 9), combat AI (Wave 11), save/load (Wave 12). The real work after that is testing — commands need actual exercise against in-memory state.
