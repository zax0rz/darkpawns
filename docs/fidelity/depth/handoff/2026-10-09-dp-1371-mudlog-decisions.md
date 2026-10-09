# DP-1371 mudlog decisions — D7 closeout (Zach rules here)

Every inventory row that this train is **not** porting, with what C does, why the
port cannot or should not reproduce it, the evidence, and a recommendation. The
`decision` column is yours. Rows you have to settle come first.

Everything below is a *recommendation*: no status moves until you rule. The two
rows already settled (`src/spell_parser.c:534`, and the `comm.c:2235` SIGUSR1
correction) were applied in the PR-3 inventory.

## 1. Yours to decide

| row | what C does and when | why the port can't / shouldn't | evidence | recommendation | decision |
|---|---|---|---|---|---|
| `objsave.c:489` | `"%s entering game with no equipment."` at NRM / MAX(LVL_IMMORT,invis) / TRUE when the crash file is missing (`fopen` fails). **Every new character's first entry.** | Go keeps no rent file; the inventory is the SQLite row's blobs, so there is no rent code to select any arm. | `2026-10-09-dp-1371-mudlog-admission.md` §5; row evidence in the inventory | `blocked-representation` until DP-1404 rules the representation | |
| `objsave.c:504` | `"%s entering game, rented equipment lost (no $)."` at BRF when `rentcode` is RENT_RENTED/RENT_TIMEDOUT and the accrued cost exceeds gold+bank | same: no stored rent code | as above | as above | |
| `objsave.c:519` | `"%s un-renting and entering game."` at NRM for `RENT_RENTED` — **every ordinary re-login after renting in C** | same | as above | as above | |
| `objsave.c:524` | `"%s retrieving crash-saved items and entering game."` for `RENT_CRASH` | same | as above | as above | |
| `objsave.c:528` | `"%s un-cryo'ing and entering game."` for `RENT_CRYO` | same (Go has no cryo path) | as above | as above | |
| `objsave.c:534` | `"%s retrieving force-saved items and entering game."` for `RENT_FORCED`/`RENT_TIMEDOUT` | same | as above | as above | |
| `objsave.c:539` | `"WARNING: %s entering game with undefined rent code."` at BRF, the default arm | same | as above | as above | |
| `objsave.c:1186` | the receptionist's `"%s has rented (%d/day, %d tot.)"` and `"%s has cryo-rented."` at NRM | Go has no receptionist or cryogenicist command path; `quit`'s RentOut is the quit path only | as above | as above | |
| `whod.c:250`, `:267`, `:296` | `"WHOD port opened."` / `"WHOD port closed."` at BRF / LVL_GOD / TRUE from `init_whod`, `close_whod` and `whod_loop` (the reboot arm) | WHOD is a live C service started at boot (`src/comm.c:274`, `:290`, `:555`) with its own listener; Go has none. Its "who" surfaces are the web, MSSP and Grapevine — an approved divergence, not an omission | `LOG` is `mudlog(msg, BRF, LVL_GOD, TRUE)` (`src/whod.c:39`); no Go listener | **`divergent-approved`**, citing a new DP issue ("WHOD service not ported") that Claude files | |
| `whod.c:335` | `"WHO request from %d.%d.%d.%d served."` / `"… from %s served."` at BRF / LVL_GOD / TRUE per served request | same service; the request it logs cannot occur without it | as above | as above | |
| `oedit.c:422` | `"SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!"` at BRF / LVL_BUILDER / TRUE when a saved extra description is NULL | A representation question: C distinguishes a loaded NULL from a live allocated-empty extra description. The port's OLC writer collapses them, so the arm has no reachable state until that is decided | `2026-10-07-dp-1371-oedit-null-C.py`, `2026-10-07-dp-1371-oedit-empty-Go.py`, `2026-10-07-dp-1371-olc-control-design.md` | keep `blocked-representation`; that is a save-shape decision, not a producer | |

Rent rows are otherwise ready to port the day DP-1404 gives them a stored code;
none of them is a lock or lifecycle problem.

## 2. Recommendations I can support with a bounded proof

| row | what C does and when | why the port can't / shouldn't | evidence | recommendation | decision |
|---|---|---|---|---|---|
| `spell_parser.c:534` | `"SYSERR: Unknown spellnum %d in manual assign"` at BRF / LVL_GOD / TRUE, `call_magic`'s default arm for a `MAG_MANUAL` spell with no switch case | C has 26 `MAG_MANUAL` spells (`src/spell_parser.c:502-536`) and 26 cases, so the default is dead. **Applied in this PR** (row relabelled from `missing`) | PR-1a handoff; the 26/26 enumeration in the row | `excluded-valid-play` (applied) | done |
| `comm.c:2235` | `"Signal received - rereading wizlists."` at CMP / LVL_IMMORT / TRUE from the SIGUSR1 handler (the signal `re-read wizlist`, `src/comm.c:2303`) | Only an operator can send SIGUSR1: in C its normal sender was the autowiz child, and there is no autowiz binary or source. Go handles only SIGINT/SIGTERM (`cmd/server/main.go`) | `src/comm.c:2303`; inventory note corrected from "SIGHUP" | `excluded-valid-play` | |
| `comm.c:2245` | `"Received SIGUSR2 - completely unrestricting game (emergent)"` at BRF / LVL_IMMORT / TRUE from the SIGUSR2 handler (`src/comm.c:2309`) | Same: an operator-only signal, outside valid play | `src/comm.c:2309` | `excluded-valid-play` | |
| `spec_procs2.c:371` | `"SYSERR: …"` at BRF / LVL_GRGOD / TRUE in `stableboy`'s `else` arm, reachable only when `read_mobile(HORSE_VNUM, VIRTUAL)` fails | `HORSE_VNUM` is 8021 (`src/spec_procs2.c:314`) and mob 8021 ships in `lib/world/mob/80.mob`, so the load cannot fail in valid play. The arm also logs a stale `buf` (it formats `msg`, then logs `buf`) — undefined behaviour (R1a) | `src/spec_procs2.c:314`, `:365-371`; `lib/world/mob/80.mob` | `excluded-valid-play` | |
| `zedit.c:264`, `:270` | `"SYSERR: OLC: Failed to open %s"` at BRF / LVL_IMPL / TRUE when the zone index cannot be opened | Reached only by a filesystem fault, not by play; the payload is also a self-overlapping `sprintf(buf, "…%s", buf)` (R1a) | #1825's unchanged-C `/dev/full` predecessor; `2026-10-07-dp-1371-zedit-unknown-C.py` | `excluded-valid-play` | |
| `medit.c:364`, `sedit.c:486` (cases `olc-medit-checked-write`, `olc-sedit-checked-write`) | `"SYSERR: OLC: Cannot write mob file!"` / `"… to shop file!"` at BRF / LVL_BUILDER / TRUE when the OLC checked write fails | A filesystem fault. #1825's unchanged-C `/dev/full` experiment produced no diagnostic at all, so there is no C behaviour here to match. **No new experiments were run** | #1825; `2026-10-07-dp-1371-olc-checked-write-C.py` | `excluded-valid-play` | |
| `ident.c:235` | `"Connection attempt denied from [%s]"` at CMP / LVL_GOD / TRUE when the ident reply names a banned site | Go has no ident worker and C's path needs a responding identd on the client's side. A banned connection is refused by comm.c's own `isbanned` path in both | PR-2 handoff §5 | `excluded-valid-play` | |
| `scripts.c:1765` | `"SYSERR: Attempting to call unassigned script for %s (#%d)."` at BRF / LVL_IMMORT / TRUE when a mobile has no script for the trigger | The 1b bounded audit is complete: the port gates on the script name before the trigger flag, so C's unassigned-script state cannot be constructed (`pkg/game/scripts.go:25-28`, `pkg/game/room_obj_scripts.go:36-39`) | 1b handoff; the audit in the row | `excluded-valid-play` | |
| `fight.c:513` | `"death_cry() in fight.c called with ch->in_room = NOWHERE"` at BRF / LVL_IMMORT / TRUE, then `char_to_room(ch, 0)` and return | See §3 below for the caller audit. Go cannot hand `death_cry` a NOWHERE room: the death-trap path is gated on a resolved ROOM_DEATH room and Go re-resolves the mount in the rider's room instead of reusing C's pre-entry pointer | `2026-10-09-dp-1371-mudlog-closeout.md` §3; `TestDeathCryBoundedCallers` | `excluded-valid-play` (see the residual in §3) | |

## 3. `fight.c:513` — the bounded caller audit and its residual

C has three call sites (`grep -n death_cry src/*.c`):

1. `raw_kill` (`src/fight.c:573`) — **guarded**: the function opens with
   `if (ch->in_room == NOWHERE) return;` (`:536`), and nothing between it and
   the call moves `ch` (stop_fighting, affect_remove, tattoo_af, unmount,
   forget/set_hunting).
2. `do_move`'s death-trap arm for the rider (`src/act.movement.c:293`) —
   **guarded** immediately above by `if (ch->in_room == -1) return(0);`
   (`:279-281`), which re-checks after the entry chain.
3. `do_move`'s death-trap arm for the **mount** (`:298`) — **not re-checked**.
   C captured the pointer before the move (`:195`) and the mount is not part of
   the `ch->in_room` guard. This is the arm the 10-07 handoff left open.

The mount can only hold NOWHERE at `:298` if its room is cleared during the
rider's entry window (`char_to_room` at `:202` … `:298`): the room's
`entry_prog`, its `mp_greet` programs, and the Lua `greet` scripts of the mobs
standing there (`:266-284`). Two mechanisms exist on that surface:

- `extchar(<char table>)` calls `extract_char` on any character a script can
  address (`src/scripts.c:480-494`), and `inworld()` looks a character up by
  name (`src/scripts.c:1610`, `:1630`), so a greet script could extract the
  mount by name.
- `tport` **cannot** do it: `lua_tport` refuses an invalid destination before it
  moves anything (`if (real_room(to_room) < 0) { mudlog(…); return -1; }`,
  `src/scripts.c:1531-1534`).

Shipped content: no non-archive script under `lib/world/scripts/` calls
`extchar` or `raw_kill`; the only extractors are archive content
(`room/archive/citadel.lua`, `room/archive/sacrifice.lua`,
`mob/archive/*`, `obj/archive/daemonic_focus.lua`). So no shipped greet or entry
script can clear the mount's room in this window.

**Residual, stated plainly:** the exclusion rests on the absence of shipped
content plus the two guards above, not on a mechanism that makes the mount arm
impossible — a future greet script in a ROOM_DEATH room could reach it in C. If
you want the stricter reading, keep `blocked-reachability` with this audit as
its evidence; my recommendation is `excluded-valid-play` on the grounds that
authored content reaching it is a content change, not valid play.

Go's side is stronger than C's and is what the proof pins: `deathTrapMount`
re-resolves the mount **in the rider's room** and returns early when it is gone
(`pkg/game/death.go:775-800`), so Go never holds a pre-entry mount pointer and
never calls `deathCry` with a cleared room. `TestDeathCryBoundedCallers`
(`pkg/game`) asserts both halves — the rider's cry lands in the real trap room,
and a rider whose mount has already left produces no mount cry at all.

Note for the record, not repaired here: Go's mount arm emits one
`"The sound of a death cry is heard as %s enters the room!"` line, where C calls
`log_death_trap(mount)` and then `death_cry(mount)`. That is the "mount caller
not claimed" gap the `act.movement.c:261-292` row already records; it needs its
own train, not this one.

## 4. Hunting call sites that carry only a name (rule 7: no guessing)

`utils.c:724`'s producer needs to know whether the prey is a mobile
(`IS_MOB`) and whether it has an id (`GET_IDNUM > 0`). These Go paths cannot
answer that, so they emit nothing and are listed rather than patched:

| Go site | why it cannot decide | disposition |
|---|---|---|
| `pkg/game/world.go:1731` — `World.SetHunting(hunterName, preyName, hunterIsMob)` | takes two names and a hunter flag; the prey's kind is unknown | its single caller (`pkg/command/skill_commands.go:1915`, `new_cmds2.c:612`) emits the producer at the call site, where the prey is a typed `*Player`, so the name-only layer never needs to log |
| `pkg/scripting/engine.go:2532` — `luaSetHunt`, the no-bridge fallback for the Lua `set_hunt` global | the victim arrives as `L.ToString(2)`, a name, and the hunter is found by vnum+room | engine-test-only path: the bridge (`bridgeSetHunt`, `pkg/scripting/bindings_bridge.go`) runs for every script in the server and logs with the resolved prey (`world_bridge.go:1073-1092`) |
| `pkg/game/mobile_equipment.go:230` — `pest.SetHunting(m.GetName())` | the prey here is the equipped **mobile** (`handler.c:664-666` calls it with `ch`; the port's `finishMobileEquipment` is the mobile path only) | C's `IS_MOB(vict)` arm stores the pointer **without** the producer, so this site must stay silent. C's *player* equipment path is not ported, so its arm has no Go state |
| `pkg/game/graph.go:149`, `:176`; `pkg/game/combat_wire.go:313-314`; `pkg/game/damage_before_message.go:45-46` | these clear hunting (`set_hunting(x, NULL)` in C: `fight.c:571`, `:1386`, `:1508`) | C's `if (vict)` guard means no producer |

C callers with no Go counterpart at all (nothing to emit, recorded for the
next sweep): `spells.c:1161` (`spell_mirror_image`'s clone, whose idnum is 0 —
its own arm is the silent one), `spec_procs.c:1419` (mickey), `new_cmds2.c:807`
(`hunt_items`), `handler.c:665` (pestilence, mobile-only in the port).


