# Round 1 repair batch 2: scenario witnesses

The final scenario fixture commit is `60de2cdbe3096f21e72d2f495623f51b88c6d616`,
parented on `4de255f5941603cadd29113260b7868987d91727`. The final targeted run
used that commit and all five changed scenarios:
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-scenarios-final-v3`.
It is CLEAN: 5/5 passed, draw divergent 0, no hollow steps.

## C-first witnesses

| Sample row | Scenario | C transcript line |
|---:|---|---|
| 12 | `switched-social-sleeping-victim-depth` | `Socialsleep is not in a proper position for that.` |
| 21 | `tedit-disconnect-output-depth` | `Teditpeer stops using OLC.` |
| 23 | `switched-immortal-visible-depth` | `You feel a strange presence as a clone appears, seemingly from nowhere.` |
| 24 | `switched-body-scrounge-mountain-depth` | `You find some edible berries.` |
| 26 | `switched-body-tag-player-depth` | `A clone taps you and screams, 'TAG! You're it!'` |

The row 12 fixture was corrected after the first revert attempt stayed CLEAN.
That earlier scenario used a player-issued `hug guard`, which reached the player
social implementation and did not execute the sampled `npcSocial` literal. The
retained run `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-row12`
is CLEAN with draw divergent 0. The corrected fixture has a second God force
the switched clone to hug a sleeping player; C sends the refusal through
`TO_CHAR` to the clone's attached descriptor (`src/act.wizard.c:1189-1200` for
switch binding and `1862-1880` for force, `src/act.social.c:132-143`,
`src/comm.c:2503-2510`, `lib/misc/socials:410-418`). The corrected baseline
and mutant proof below now exercise the sampled path.

The other witnesses are C-first depth cases. Their source traces are recorded
in the scenario headers: tedit disconnect cleanup (`src/comm.c:2107-2119`,
`src/olc.c:447-518`); switched immortal visibility (`src/act.other.c:557-585`,
`src/act.wizard.c:1622-1635`, `src/fight.c:108-121`, `src/structs.h:248,311,329`,
`src/mobact.c:121`); switched mountain scrounge (`src/new_cmds2.c:64-130`,
`src/act.wizard.c:1189-1203,1584-1618`, `src/db.c:104,1281`); and switched
mobile tag against a player (`src/new_cmds.c:2198-2264`,
`src/act.wizard.c:1189-1203`).

## Revert proofs

Each isolated mutant is a one-byte change parented directly on the final
scenario fixture commit above. Each passed `GOEXPERIMENT=nojsonv2 go build ./...`
before its single-scenario census. Every run is NOT_CLEAN with one failing
scenario and draw divergent 0.

| Sample row | Mutant commit | Byte change | Run directory | C line → mutated Go line |
|---:|---|---|---|---|
| 12 | `3474da5f540819d146459917b87b8e2aa67626bb` | `pkg/game/npc_command.go`, offset 17619, `N→Q` in `$N` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-row12-v2` | `Socialsleep is not in a proper position for that.` → ` is not in a proper position for that.` |
| 21 | `1ddb364fc5037e81ec4cc57054f5ba4650bc1f99` | `pkg/session/tedit.go`, offset 14409, `n→Q` in `$n` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-row21-v2` | `Teditpeer stops using OLC.` → ` stops using OLC.` |
| 23 | `9a94a812d2b40968527ba306c37f8a672ecb1677` | `pkg/game/character_command.go`, offset 7553, `Y→Q` in `You` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-row23-v2` | `You feel a strange presence as a clone appears, seemingly from nowhere.` → `Qou feel a strange presence as a clone appears, seemingly from nowhere.` |
| 24 | `77ee55583cd2e1061fc2f974d886a1743f2b92e4` | `pkg/game/skill_actor_waits.go`, offset 1768, `Y→Q` in `You` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-row24-v2` | `You find some edible berries.` → `Qou find some edible berries.` |
| 26 | `28ecdb46e8e3afea71afa73c160ce0c7d2ff284f` | `pkg/game/skill_actor_combat_next.go`, offset 30993, `n→Q` in `$n` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-20/luna-r1-repair2-mutant-row26-v2` | `A clone taps you and screams, 'TAG! You're it!'` → ` taps you and screams, 'TAG! You're it!'` |

## Decisions for Zach

No new decision is needed; see [`DECISIONS.md`](../DECISIONS.md).

## For Claude

Please review the corrected row 12 fixture alongside the five isolated mutant
proofs. The earlier CLEAN attempt is retained and explained above; it is not
counted as a kill.
