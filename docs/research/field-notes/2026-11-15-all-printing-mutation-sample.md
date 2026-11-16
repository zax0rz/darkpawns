# Fresh all-printing mutation sample (seed 20261115)

2026-11-15. Measurement field note for the roadmap §6 completion bar ("Fresh mutation sample, all printing packages: lower end of the 95% interval at or above 80%, and every catchable miss repaired").

## Population and sample

Using the round-two AST collector (`docs/research/idiom-experiment/round2/run/population.go`) across every package under `pkg/` that prints player output at Go HEAD `4ff2dfaca`:

| Package | Functions | Sampled | Killed | Survived |
|---|---:|---:|---:|---:|
| `pkg/boards` | 6 | 0 | 0 | 0 |
| `pkg/combat` | 13 | 1 | 1 | 0 |
| `pkg/command` | 51 | 4 | 2 | 2 |
| `pkg/game` | 400 | 39 | 17 | 22 |
| `pkg/session` | 156 | 12 | 7 | 5 |
| `pkg/spells` | 41 | 4 | 4 | 0 |
| **Total** | **667** | **60** | **31** | **29** |

- Frozen population: `docs/research/mutation-sample-2026-11-15/population.tsv` (`sha256: a5aca24e03c95ebb6e250d992b6d4e386d7fd1f46278e09b43b8e94858997e61`) and `population-literals.tsv` (`sha256: 70f7ffe655938bb0dcd86ebec1cdc2d143994bbb61ea8dc5564d561bf43d47ed`).
- Fixed preregistered seed: `20261115` (`random.Random(20261115).sample(population, 60)` without replacement, then `random.Random(20261115 + i).choice(literals)` for one literal per function, mutating its first ASCII letter to `Q` or `Z` as in `round3/run/draw_sample.py`).
- Sample & probe ledger: `docs/research/mutation-sample-2026-11-15/sample.tsv` and `docs/research/mutation-sample-2026-11-15/probes.tsv`.

## Unmutated control and measurement method

Following the `2026-11-08` correction, an unmutated control run was executed first across all `118` covering scenarios in `cmd/dp-oracle-diff/scenarios/` (`-seed 1` against `~/darkpawns-c-oracle/bin/circle`). `117` passed cleanly and `1` pre-existing failure (`postmaster-saved-recipient-depth`, exit 3) was excluded before any mutant ran, leaving `117` clean control scenarios. Each mutant was compiled once into `/tmp/mut_srv` and tested via `ORACLE_REGRESSION_SERVER=/tmp/mut_srv` against its covering scenarios.

## Result

- **Killed**: **31 / 60 = 51.7%**
- **95% Wilson score interval**: **[39.3%, 63.8%]**
- **Equivalent**: **0 / 60** (every sampled literal alters a live format or message string; none is output-equivalent on all inputs).
- **Survived**: **29 / 60 = 48.3%** (`27` C-backed catchable misses, `2` uncatchable Go-only sites).
- **Roadmap §6 bar verdict**: **Not met** — the lower end (`39.3%`) is well below the `80%` threshold (and the upper end `63.8%` is also below `80%`).

## Every survivor, its C site, and catchability

| `i` | Package | Go site | Mutated literal | Scenarios run | C site | Catchable by scenario? |
|---:|---|---|---|---|---|---|
| 0 | `pkg/game` | `*World.doRideMob` (`pkg/game/other_mount.go:118`) | `"That's disgusting!"` | `mount-depth;combat-entry-mounted` | `src/act.other.c:1567` | Yes (mount an unmountable mob) |
| 1 | `pkg/command` | `*AdminCommands.cmdListReports` (`pkg/command/admin_commands.go:546`) | `"Jan 02 15:04"` | — | None (Go-only admin command) | No (Go-only admin console formatter, not MUD telnet/WS) |
| 2 | `pkg/session` | `cmdAssist` (`pkg/session/cmd_combat_basic.go:25`) | `"Whom do you wish to assist?\r\n"` | `assist-resolve-depth;combat-entry-assist;invisibility-assist-depth` | `src/act.offensive.c:73` | Yes (`assist` with no argument) |
| 3 | `pkg/session` | `cmdOlist` (`pkg/session/wiz_zone.go:168`) | `"Huh?!?"` | `olist-depth` | `src/interpreter.c:591` (table `min_level` shadows body) | No (Go-only defensive level guard inside `cmdOlist`; command table rejects mortals first) |
| 5 | `pkg/game` | `npcSteal` (`pkg/game/spec_procs.go:350`) | `"$n tries to steal gold from $N."` | `steal-depth` | `src/spec_procs.c:314` | Yes (thief mob `spec_proc` attempting steal) |
| 6 | `pkg/game` | `*World.CastGateSpell` (`pkg/game/gates.go:109`) | `"The magic flows through you, but nothing else happens.\r\n"` | `cast-depth` | `src/gate.c:383-390` | Yes (cast `gate` when no valid room/portal applies) |
| 12 | `pkg/game` | `*World.decayObject` (`pkg/game/limits_condition.go:379`) | `"%s fades out of existence.\r\n"` | `death-corpse-decay-npc;death-corpse-decay-pc` | `src/limits.c:543` | Yes (object timer decay in player room/inventory) |
| 16 | `pkg/game` | `*World.findDoorMobForCharacter` (`pkg/game/act_movement.go:1030`) | `"What is it you want to %s?\r\n"` | `door-basic;door-bad-direction-depth;door-fillword-depth` | `src/act.movement.c:409` | Yes (switched mob executing `open`/`close`/`lock` with no arg) |
| 17 | `pkg/session` | `forceSessionCommand` (`pkg/session/wiz_communication.go:278`) | `"$n has forced you to '%s'."` | `force-depth;force-gates;force-mob` | `src/act.wizard.c:1864` | Yes (`force` another player and observe victim output) |
| 18 | `pkg/session` | `cmdRetreat` (`pkg/session/combat_cmds.go:337`) | `"Get on your feet first!\r\n"` | `retreat-closed-door` | `src/act.offensive.c:1021` | Yes (`retreat` while sitting/resting) |
| 19 | `pkg/game` | `*World.DoSpecCommMobile` (`pkg/game/directed_speech.go:436`) | `"$n %s you, '%s'"` | `comm-depth;comm-color-complete;comm-echo-immort` | `src/act.comm.c:1007` | Yes (switched mob `whisper`/`ask` to player) |
| 21 | `pkg/game` | `*World.shopNpcValue` (`pkg/game/shop_npc.go:234`) | `"%s What do you want me to valuate??"` | `shop-bank-state;play-lab-m2-s4-shop` | `src/shop.c:828` | Yes (`value` with no argument at shopkeeper) |
| 22 | `pkg/game` | `CharacterFleshAlter` (`pkg/game/skill_actor_selected.go:105`) | `"$n's hand reverts from a %s!"` | `flesh-alter-depth;flesh-alter-failure;flesh-alter-gates` | `src/new_cmds.c:1920` | Yes (room observer when `flesh alter` reverts) |
| 24 | `pkg/game` | `specMickey` (`pkg/game/spec_procs.go:1203`) | `"$n says, 'You're not centered.'"` | — | `src/spec_procs.c:1293` | Yes (trigger `specMickey` special proc) |
| 27 | `pkg/game` | `StopFollowerMob` (`pkg/game/follow.go:248`) | `"$n stops following you."` | `death-followers-depth;move-npc-follower-depth;movement-followers` | `src/utils.c:423` | Yes (mob stops following player) |
| 29 | `pkg/command` | `CmdShoot` (`pkg/command/skill_commands.go:737`) | `"You must wield a bow or sling to fire a projectile.\r\n"` | `shoot-entry-depth;shoot-item-gates-depth;shoot-mob-hit-depth` | `src/act.offensive.c:826` | Yes (`shoot` without wielding bow or sling) |
| 30 | `pkg/session` | `*Session.finishZeditLocked` (`pkg/session/zedit.go:215`) | `"$n stops using OLC."` | `olc-save-depth;mudlog-olc-disk-depth;zedit-boundaries-depth` | `src/olc.c:514` | Yes (room observer when builder exits `zedit`) |
| 35 | `pkg/game` | `*World.useWandPoint` (`pkg/game/other_economy.go:523`) | `"You point $p at yourself."` | `spellmatrix-wand-flamestrike-depth;spellmatrix-wand-mirror-image-depth;wand-lightning-route-depth` | `src/spell_parser.c:615` | Yes (`use` wand targeting self) |
| 37 | `pkg/session` | `cmdThaw` (`pkg/session/wiz_system.go:671`) | `"Yes, but for whom?!?\r\n"` | `wizutil-thaw-higher-freeze-depth` | `src/act.wizard.c:2091` | Yes (`thaw` with no argument) |
| 40 | `pkg/game` | `*World.doMobReply` (`pkg/game/reply_body.go:64`) | `"They are no longer playing.\r\n"` | `comm-depth` | `src/act.comm.c:957` | Yes (switched mob `reply` after target logs out) |
| 41 | `pkg/game` | `*World.DoWrite` (`pkg/game/act_write.go:74`) | `"You can't write on $p."` | `write-gates-depth;write-already-written-depth;gen-write-depth` | `src/act.comm.c:1093` | Yes (`write` on non-note object) |
| 43 | `pkg/game` | `specElementsMinionBody` (`pkg/game/actor_specials_rooms.go:281`) | `"$n utters the words 'eradico paratus' and $p disintegrates."` | `spec-npc-elements-minion-depth` | `src/spec_procs3.c:1232` | Yes (elements minion spec proc disintegrating item) |
| 48 | `pkg/game` | `specMoonGate` (`pkg/game/spec_procs_missing.go:312`) | `"You enter the portal and are transferred...\n\r"` | `full-moon-depth` | `src/gate.c:278` | Yes (player enters moon gate portal) |
| 50 | `pkg/game` | `lunarHunterForWorld` (`pkg/game/weather.go:666`) | `"By the light of the full moon.. DIE!"` | `full-moon-depth` | `src/new_cmds.c:2678` | Yes (lunar hunter attacks during full moon) |
| 51 | `pkg/game` | `mobDisarm` (`pkg/game/spec_paladin.go:97`) | `"$n tries to disarm $N, but fails and falls flat on $s face!"` | `disarm-depth;combat-zone-equipped-mobile` | `src/new_cmds2.c:246` | Yes (mob fails disarm with room observer) |
| 52 | `pkg/session` | `cmdRecite` (`pkg/session/use_cmds.go:57`) | `"You can't concentrate enough!."` | `recite-effect;recite-damage-spell;recite-armor-target` | `src/spell_parser.c:663` | Yes (low-skill/class character fails scroll recite check) |
| 53 | `pkg/game` | `*World.doClanEnroll` (`pkg/game/clan_membership.go:25`) | `"You're not influent enough in the clan to do that!\r\n"` | `clan-applicant-depth;clan-callsite-authorization-gates-depth` | `src/clan.c:317` | Yes (insufficient-rank clan member runs `clan enroll`) |
| 54 | `pkg/game` | `performDropGoldMobile` (`pkg/game/item_transfer_actor.go:351`) | `"$n drops %s."` | `drop-gold-depth;drop-gold` | `src/act.item.c:459` | Yes (switched mob drops gold with room observer) |
| 55 | `pkg/game` | `wakeBody` (`pkg/game/character_command.go:306`) | `"$n tosses and turns uncomfortably."` | `dream-point-update-wake-depth;position-wake-target` | `src/act.movement.c:871` | Yes (wake `AFF_SLEEP` victim with room observer) |

## Next

No code or scenario fixes were made in this PR (measurement only). The `27` catchable C-backed survivors above are the concrete repair list for closing the roadmap §6 bar.
