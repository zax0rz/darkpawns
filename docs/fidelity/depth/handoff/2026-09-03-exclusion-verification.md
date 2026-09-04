# Exclusion verification handoff — 2026-09-03

## Scope

This slice rechecks the depth manifest exclusions against the C source and the
actual player-facing dispatch paths. The checkout was pristine apart from the
pre-existing untracked `website/static/images/` directory. The corpus baseline
was 4,111 cases: 4,013 proven/delegated, 45 blocked, and 53 excluded.

No C oracle scenario was added here. These are reachability decisions, so the
evidence is the C call path and the Go dispatch path; `src/` and
`darkpawns-c-oracle/` were not edited.

## Verified exclusions

- The two assigned 158xx mob rows remain unreachable as specials. `src/spec_assign.c:421-422`
  assigns vnums 15808 and 15814, but their action flags in `lib/world/mob/158.mob`
  do not set `MOB_SPEC` (`src/structs.h:247`). No reclassification is warranted.
- The vampire and werewolf corpse item paths are already `blocked`, not
  excluded. The prior audit commit `07c3a687e` correctly moved
  `drink.vampire`, `eat.vampire`, and `eat.werewolf-corpse` after confirming
  reachable C setters and game-state gates. They are not changed in this slice.
- Descriptorless and undefined-behavior rows retain their exclusions. C
  explicitly returns for missing descriptors in `do_quit` and `do_save`
  (`src/act.other.c:84-85,188-189`), `do_abils` and `do_help`
  (`src/act.informative.c:1079,1570-1572`), and the descriptorless
  `headbutt`/`groinrip` gates (`src/new_cmds.c:378,2565-2567`). NPC page
  handling is an explicit C rejection (`src/act.comm.c:1112-1114`). The
  overlapping `sprintf` paths in `do_auto` and `do_dns`
  (`src/act.other.c:1308-1341`, `src/act.wizard.c:3163-3208`) have no faithful
  deterministic byte target, so they remain excluded under R3/R4/R5d.
- `shoot.handler-fighting-unreachable` remains excluded: the command table
  registers `shoot` at `POS_STANDING` (`src/interpreter.c:695`), and the
  interpreter position gate rejects a fighting player before
  `src/act.offensive.c:746-775` can reach its local fighting branch.
- `mindlink.success-psionic-mob-unreachable` remains excluded. The success
  arm is guarded by `IS_PSIONIC`/`IS_MYSTIC`, while those macros require
  `!IS_NPC` (`src/new_cmds2.c:274-291`, `src/utils.h:366,418`), after the
  command has already selected an NPC target.
- `obj.field-object-special-unreachable` remains excluded. The special in
  `src/spec_procs3.c:456-514` returns on a nonzero command, and its assigned
  objects (`src/spec_assign.c:524-529`) have no commandless object-special
  dispatcher; `object_activity` handles object scripts instead.
- The remaining exclusions recorded by the Section 5.2 audit retain their
  cited C reachability reasons, including NPC/descriptor early returns,
  script-only entry points, the two socials with no target vehicle, and the
  explicit Go-only `use.skill-extension` extension. No new false exclusion was
  found in those rows.

## False exclusions corrected to blocked

- `use.tattoo` is a reachable player command: `src/act.other.c:920-924`
  routes `use tattoo` to `use_tattoo` (`src/tattoo.c:31-91`), and the Go port
  has the corresponding implementation in `pkg/game/other_economy.go:108-168`.
  It lacks an independent depth vehicle, so it is `blocked`, not proven.
- `room.jail-commandless-body` is a reachable pulse path. C
  `room_activity()` invokes the room special for every player with command 0
  (`src/comm.c:690-756`), and `SPECIAL(jail)` contains the timer, audience,
  relocation, and look sequence (`src/spec_procs2.c:1470-1493`). Go already
  dispatches room activity with an empty command, but `specJail` is currently a
  no-op. The row is therefore `blocked`, pending a dedicated pulse vehicle;
  it must not be excluded as unreachable.

These two reclassifications change the accounting to 4,111 total, 4,013
proven/delegated, 47 blocked, and 51 excluded. This is a classification fix,
not a claim of behavioral parity. The decision applies R1/R2/R4/R5e: preserve
the real player-visible surface, do not invent a commandless substitute, and
verify the actual dispatcher before excluding a path.

