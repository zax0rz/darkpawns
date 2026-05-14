# File Split Plan — DP-62

Large files in `pkg/game/` that should eventually be split for maintainability.

## Current state

| File | Lines | Contents |
|------|-------|----------|
| spec_procs2.go | 1524 | Zone-specific special procedures |
| spec_procs3.go | 1348 | More zone-specific special procedures |
| world.go | 1279 | World management, zone resets, combat tick |
| socials.go | 1136 | Social command system |
| act_movement.go | 969 | Movement commands |
| objsave.go | 812 | Object save/rent system |
| spec_procs.go | 784 | Special procedure infrastructure |
| death.go | 756 | Death handling |
| constants.go | 689 | Game constants |
| save.go | 674 | Player save/load |
| spawner.go | 673 | Mob spawning |
| mob.go | 656 | Mob type and methods |

## Proposed splits

### world.go (1279 lines) → 3 files
- `world.go` — Core World struct, initialization, room/object management (~400 lines)
- `world_tick.go` — Zone reset, combat tick, weather tick (~500 lines)
- `world_messaging.go` — Room messages, send checks, zone weather messages (~379 lines)

### spec_procs2.go + spec_procs3.go (2872 lines) → by zone group
- `spec_procs_hometown.go` — Hometown zones (Agrabah, Midgaard, etc.)
- `spec_procs_combat.go` — Combat-focused zones (Arena, Battle Grounds)
- `spec_procs_dungeon.go` — Dungeon zones (Mines, Crypt, etc.)
- `spec_procs_misc.go` — Everything else

### socials.go (1136 lines) → 2 files
- `socials.go` — Core social system, registration, lookup (~400 lines)
- `socials_commands.go` — Individual social command implementations (~736 lines)

### act_movement.go (969 lines) → 2 files
- `act_movement.go` — Core movement (north/south/east/west/up/down) (~500 lines)
- `act_movement_special.go` — Special movement (portal, fly, mount, follow) (~469 lines)

## Priority
1. **world.go** — most impactful, touches everything
2. **spec_procs** — largest combined size, zone-specific so low risk
3. **socials.go** — clean separation between infrastructure and commands
4. **act_movement.go** — lowest priority, already manageable

## Risk
All splits are mechanical — move functions, update package doc, no logic changes. Each split should be its own commit with build+test verification.
