## Dark Pawns Wave 18 — Spell System Porting (Continued)

Context is at ~/darkpawns/. Start here:

1. GBrain page: `gbrain get_page darkpawns/wave18-spell-port` — full session state, what's done, what's blocked
2. `docs/SPELL_PORT_SCOPE.md` — master scope of all 24 manual spells
3. `git log --oneline -10` — recent commits

### What Was Completed (4 commits, pushed to main)
- PLR flags (PlayerFlags uint64, 22 constants, HasPLRFlag/SetPLRFlag/ClearPLRFlag)
- MOB flags (Flags uint64 on MobInstance, 26 bit constants, HasMobFlag/SetMobFlag)
- Wave A: 9 LOW spells (sobriety, zen, detect_poison, calliope, lycanthropy, vampirism, control_weather, coc, mental_lapse)
- Wave B: 4 MEDIUM spells (enchant_weapon, enchant_armor, create_water, identify) + silken_missile
- ObjectInstance copy-on-write overrides (ExtraFlags, Affects, Values)
- SetValue on ObjectInstance

### Current State: 15/24 manual spells implemented

### Priority: Infrastructure unblocking

The remaining 9 spells all need one or more of these infrastructure pieces. Pick the one with the most unblocking power:

1. **Room character iteration** (`world.PlayersInRoom()` exists but mobs aren't included) — unblocks hellfire, meteor_swarm, mindsight (3 spells)
2. **Room transfer** (char_from_room/char_to_room) — unblocks recall, teleport, summon, mindsight (4 spells)
3. **Follower system** — unblocks charm, conjure_elemental, divine_int (3 spells)

### Key Files
- `pkg/spells/affect_spells.go` — all manual spell implementations + dispatch
- `pkg/spells/spell_info.go` — spell template system
- `pkg/spells/call_magic.go` — main spell dispatch
- `pkg/game/world.go` — World struct with SpawnObject, ExtractObject, AddItemToRoom, GetPlayersInRoom
- `pkg/game/player.go` — Player struct (PlayerFlags, conditions, affects)
- `pkg/game/mob.go` — MobInstance (Flags, Hunting, Following)
- `src/spells.c` — C source (authoritative)
- `src/structs.h` — C flag/type constants (authoritative)

### Rules
- Read before write. Check interfaces before calling them.
- `go build ./...` after every significant change.
- Use `slog` not `log` or `fmt` for logging.
- No globals — pass dependencies through params.
- C source is authoritative: `src/` directory. Fix C bugs noted in scope doc during port.
- **Import cycle**: spells→game creates a cycle. Use interface assertions on all game types.
- For Claude Code subagents: always prepend `unset ANTHROPIC_API_KEY &&` to use subscription OAuth.
- Claude Code --print mode OOMs on large prompts here. Prefer direct implementation or `sessions_spawn` with bounded tasks.
