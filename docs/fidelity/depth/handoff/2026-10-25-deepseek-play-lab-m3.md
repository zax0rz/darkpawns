---
tags: [handoff, fidelity, play-lab, differential, progression, dataset]
last_updated: 2026-10-25
author: DeepSeek (play lab)
---
# Play lab, round 3: long-horizon play, six classes, and what stops the climb

Milestone 3 of the play lab (`~/DEEPSEEK-BRIEF-2026-10-25-play-lab.md`, after
milestone 1 merged as #2150 and milestone 2 as #2157). Products this round:

1. **A progression policy** that routes from the world files: hunting grounds
   chosen by mob level, rest and flight, guild visits for `practice`, shop trips
   for gear, and a sub-goal recorded on every decision row.
2. **Six long sessions**, one per class the creation menu offers, 5,100 directed
   decisions in total, plus two random-play sessions for coverage.
3. **A differential sample**: session slices, a staged death, a staged jail
   sequence and the practice catalog, replayed against the C oracle with
   `--draw-parity`.

Findings only, no fixes. Local servers only, frozen clock, pulses pumped as in
milestone 2.

**Headline.** Every class was played for 850 directed decisions (~6–11 minutes
each). One class reached **level 2** (psionic, 1,626 experience); the other five
finished level 1 between 275 and 693 experience against a level-2 threshold of
1,500. Nothing was blocked by a bug in the port: the climb is blocked by the
game's own arithmetic plus the frozen clock (see §3.3), and the policy's own
inefficiency (it spends most decisions walking, not fighting). Two divergences
were reduced and filed: **PL-004** (the practice catalog still reports a learned
*mapped* skill as "(not learned)") and **PL-005** (the death path: the port kills
where C leaves the character incapacitated, with a draw-parity mismatch at the
same step).

## 1. Routing: the world map the policy plays from

`scripts/progression-probe.py` reads the authoritative world files and writes
`world.json`: every room and exit (`lib/world/wld`), every mob prototype's level
and keywords (`lib/world/mob`), the `M`/`O` resets that place mobs and objects
(`lib/world/zon`), the shops with their keeper, room and produce list
(`lib/world/shp`), and the guildmasters from `src/spec_assign.c`
(`ASSIGNMOB(<vnum>, guild)`; names read from `lib/world/mob/80.mob`).

| what the map answers | how |
|---|---|
| where is the hunting, by level | 129 reachable rooms with mob resets, each with its mob list, level band and a door-free route from the newbie start (8162) |
| where do I practise | the six class guildmasters: warrior 8012→8013, thief 8013→8032, cleric 8014→8030, mage 8015→8012, psionic 8024→8149, ninja 8026→8151 |
| where do I buy | 10 reachable shops, including Darek's Happy House of Cutlery (8055, weapons) and Jack's Armor Shop (8037, armour) |
| how do I get there | a door-free BFS over the parsed exit graph, with a door-allowing fallback, and a per-session blocked-edge set for doors that turn out locked |

`internal/agent/worldmap.go` loads it; `internal/agent/progression.go` plays it.

**The policy.** Priorities, in order: recover from death; flee below 30 % health;
rest below 55 % health or below 15 % movement; stop at the target level; wield
and wear what the pack holds; refresh the room's occupant list when it goes
stale; loot a corpse; hunt the strongest mob within one level of the character;
visit the guild (at the start and after every level) to read the catalog and
spend practice sessions; buy a weapon when unarmed; travel to the best hunting
ground in reach; look.

**The big-model layer.** Every decision row carries a `subgoal` string as well as
the step choice, so the dataset has both layers the brief asks for:
`go to Malchor's Monkey World`, `hunt citizen (level 1) in Temple Square`,
`rest to recover movement (12/83)`, `go to the guild (Warrior Practice Room)`,
`list skills at Holy Room Of Clerics`, `practice kick`, `loot the corpse in
Western Market Street`, `buy sword at Darek's Happy House of Cutlery`,
`recover after death`, `open the door`, `milestone reached: level N`.

## 2. Sessions and results

Six long sessions, one per class the creation menu offers (a human's menu is
cleric / thief / warrior / magic-user / ninja / psionic), plus two random-play
sessions for coverage. Every session restarted its own server, ran on a frozen
clock, and drove 850 directed decisions. Wall time is the session's own elapsed
time (the six ran in parallel, one worker per class).

| class | decisions | experience | level reached | kills | jail blocks | practice commands | wall |
|---|---:|---:|---:|---:|---:|---:|---:|
| psionic | 850 | 1,626 | **2** | 12 | 0 | 5 | 693 s |
| thief | 850 | 693 | 1 | 5 | 1 | 4 | 447 s |
| mage | 850 | 573 | 1 | 4 | 1 | 6 | 418 s |
| warrior | 850 | 562 | 1 | 4 | 1 | 5 | 440 s |
| ninja | 850 | 443 | 1 | 3 | 2 | 4 | 439 s |
| cleric | 850 | 275 | 1 | 3 | 1 | 6 | 362 s |
| random warrior | 150 | — | 1 | — | — | — | 70 s |
| random mage | 150 | — | 1 | — | — | — | 70 s |

**5,100 directed decisions** in total (the brief asked for about 5,000), plus
300 random decisions. The level-2 threshold is 1,500 experience
(`score`: "You need 1499 exp to reach your next level" at 1 experience); only the
psionic crossed it. The dataset export keeps 3,564 of the 5,100 directed rows
(686 dropped by the death/outcome filter) and holds one whole session out:
`~/dp-play-lab/dataset/m3`.

**Per-level cost, measured.** No class reached level 3, so there is no honest
level-3/5/10 wall-time figure to report: the milestone's answer to "how far does
each class get" is the table above, and the reason is in §3.3.


## 3. Findings

### 3.1 PL-004 — the practice catalog reports a learned *mapped* skill as "(not learned)"

`RenderSkillList` (`pkg/game/practice.go`) reads each skill with
`p.GetSkill(strings.ToLower(name))`, where `name` is the *display* name from the
spell catalog. `specGuild` (`pkg/game/spec_procs.go`) writes the skill under
`SkillStorageName(skillNum)`, which for the skills in `catalogSkillKeys`
(`pkg/game/skill_catalog_names.go`) is a *different* key — `strike of revenge` is
stored as `strike`, `flesh alter` as `flesh_alter`, `pick lock` as `pick_lock`,
and so on. The catalog therefore reads a skill nothing wrote.

Reproduced on both engines with the same keystrokes
(`scripts/practice-probe.py`, evidence in `~/dp-play-lab/evidence/pl-004-{c,go}.txt`):
a ninja spends all four sessions on `strike of revenge`, the guildmaster answers
"You are now learned in that area.", and then

```
C oracle : strike of revenge     (fair)
Go port  : strike of revenge     (not learned)
```

C stores and reads by skill *number* (`SET_SKILL(ch, skill_num, …)` in
`SPECIAL(guild)`, `src/spec_procs.c:201-247`; `list_skills` →
`how_good(GET_SKILL(ch, spells[i].number))`, `src/spec_procs.c:157-195`), so the
two always agree. This is an R1 divergence in player-facing bytes.

Reduced to `cmd/dp-oracle-diff/scenarios/play-lab-m3-practice-catalog.txt`
(`spawn-mob 8026 1 8151 80` places the ninja guildmaster, so the scenario does not
depend on when the zone reset ran). Verdict today: `normalized divergence
detected`, `draw-parity MATCH`, exit 3 — the one-line diff is the pair above.
The fix is PR **#2160** (`codex/fix-practice-storage`), open, not merged: once it
lands, this scenario is its regression guard.

### 3.2 PL-005 — the death path kills where C leaves the character incapacitated

Staged in `play-lab-m3-death-in-combat.txt`: a level-11 Zoomaster (mob 14516,
placed with `spawn-mob … 8101 145`) fights a level-1 warrior in the Street of
Swords. The same probe step (`~dpclock pulse 100`, step 7) diverges:

```
--- c-oracle
+++ go-port
 The Zoomaster OBLITERATES you with his deadly hit!!
 You are incapacitated an will slowly die, if not aided.
+The Zoomaster hits you with such force that your neck snaps in two!
+You are dead!  Sorry...
 
--<PROMPT>
+Welcome to Dark Pawns!
+0) Exit from Dark Pawns.
…
+   Make your choice:
```

and one step later C shows the suffering ticks the port never prints:

```
-You lie on the ground suffering from your terrible wounds...
-You are incapacitated an will slowly die, if not aided.
```

The run also reports `draw-parity MISMATCH  stream=4093  c=Number(0,18) = 3
go=Number(1,20) = 4` at exactly that step, so the two engines are not drawing the
same call at the same point (R3) — the divergence is not a coincidence of output
ordering.

C's path is `point_update`'s incapacitated branch (`src/limits.c:507-515`:
`POS_INCAP` takes 1 damage per pulse, `POS_MORTALLYW` 2, until `GET_HIT <= -11`
reaches `POS_DEAD`), which is why the C transcript shows the character lying
there suffering rather than dying outright. The port's equivalent step kills
immediately and drops the character to the login menu. The exact origin of the
port-only "neck snaps in two" line is **not yet identified** — it is not a
literal in either tree, so it is assembled or scripted somewhere this pass did not
reach (R5g partially satisfied: the C side of the path is cited, the port side
needs one more pass). Filed as a finding, not fixed.


### 3.3 What actually stops the climb (a policy finding, not a port bug)

Three measured limits, none of them a bug in the port:

1. **The city answers violence with jail, not death.** Attacking a citizen or a
   guard summons a city guard: "A city guard jumps to the aid of the citizen! …
   he carts you off to jail…", and the character sits in The Main Holding Cell
   until the sentence runs out on pulses. The six sessions lost 1–2 sentences
   each; the ninja spent **473 of its 850 decisions** in the cell.
2. **A frozen clock means a finite mob pool.** Zone resets do not repopulate
   within a session's wall time, so every room the policy clears stays cleared;
   the sessions spent most decisions walking between exhausted rooms (the
   psionic: 12 kills in 850 decisions).
3. **Experience is small at level 1.** City mobs pay 2–25 experience; the
   level-2+ wilderness mobs pay 100 (measured: every "You receive N experience"
   kill in the six sessions read exactly 100). Reaching 1,500 therefore needs
   about 15 wilderness kills, and the policy found 3–12.

The honest read: the port is not blocking the climb, the policy and the clock
are. The next round's routing should treat the city as a place to buy and
practise and hunt the 145xx menagerie (levels 1–6, 100 experience a kill, 10–12
steps from the start) as the primary ground.

## 4. Watch items

| watch item | what this round saw |
|---|---|
| practice lists after learning skills | **PL-004**, reproduced on both engines; the fix (#2160) is open. Unmapped skills are correct on both engines: `kick` reads `(not learned)` then `(good)` |
| mana / hit / move regen by position | functions and gates match C line for line (`ManaGain`/`HitGain`/`MoveGain` vs `src/limits.c:59-253`; `PointUpdate` vs `:495-501`). The measured fact that matters for play: regen runs **once per Mud hour = 630 pulses** (`SECS_PER_MUD_HOUR` 63, `src/utils.h:135`), so a rest must pump a whole hour to recover anything — the driver's `-rest-pulses 700` does. No inversion found in this pass |
| death (affects must clear) | #2158 landed the `raw_kill` strip; the staged death scenario is **PL-005** (a divergence in the death *path*, not in the strip) |
| lights | unchanged from round 2: the lantern regression guard (`play-lab-lights-dark-room.txt`) is still green, and the fix for the original finding was **#2151** (`glm/room-light-accounting`, `6e245a2ba`), not #2148 as round 2's report said (see §6) |
| doors while hunting | exercised: the policy walks through doors, opens a closed one (`open door`), and marks a refused walk as a blocked edge for the rest of the session. The jail scenario and the session slices replay door traversals with no divergence |
| the justice system | the guard-assist → jail → release sequence is byte-identical to C at seed 1 (`play-lab-m3-jail-on-attacking-guard.txt`, `draw-parity MATCH`) |

## 5. Differential: scenarios and the census

Added this round (all in `cmd/dp-oracle-diff/scenarios/`):

| scenario | what it replays | verdict at seed 1 |
|---|---|---|
| `play-lab-m3-practice-catalog.txt` | a ninja learns a mapped skill at the guildmaster, then reads the catalog | **divergence** (PL-004), draw parity MATCH |
| `play-lab-m3-death-in-combat.txt` | a level-1 warrior fights a level-11 Zoomaster to the death | **divergence** (PL-005), draw parity MISMATCH |
| `play-lab-m3-jail-on-attacking-guard.txt` | a level-1 warrior swings at a church guard; the city jails him and lets him out | clean, draw parity MATCH |
| `play-lab-m3-warrior-slice.txt`, `-psionic-slice.txt`, `-ninja-slice.txt` | the first 70–90 play commands of a long-horizon session (creation, travel, kills, loot, rest, practice) | produced by `dp-play scenario`; the warrior slice was still running when this report was written, so its verdict is not claimed here |

The census is registered as `m3-play-lab` (targeted, three scenarios) at
`~/Archives/darkpawns/oracle-runs/2026-10-25/m3-play-lab/`, queued behind the
`v015-release-90ff38ff7` release census that was running at the time
(`start` printed `position=2 active=36/72`). Its verdict is not in this report:
the two finding scenarios are expected to be the only red rows, which is what a
findings round should show.

**Lab note (process).** While cleaning up scenario leftovers I ran
`pkill -x circle` / `pkill -f dp-server-prebuilt`, which also matched servers
belonging to the concurrent `v015-release-90ff38ff7` census. I did not touch its
files, and the census wrapper retries INFRA/TIMEOUT rows alone once, but rows
from that run may have been damaged by me and the release owner should re-check
them. This was my error: I should have matched only my own run directories.


## 6. Corrections to round 2

* **The lantern fix was #2151**, not #2148. Round 2's handoff and the PR body for
  #2157 attribute the room-light fix to #2148; #2148 is
  `codex/mob-command-lane`. The fix is `6e245a2ba` — "fix: room light counts a
  lit held lantern like C equip_char/unequip_char" — merged as #2151
  (`glm/room-light-accounting`). Same landing window, different PR.
* **Latency is reported, not a fidelity issue** (Zach's answer to #2157): round
  2's Go median 0.6 ms / p95 1.1 ms against C's ~99 ms is structural
  (`OPT_USEC`, `src/structs.h:628`; the pass loop, `src/comm.c:545-580`) and no
  delay was added anywhere in this round.

## 7. Lab changes this round (`~/dp-play-lab`)

| change | why |
|---|---|
| `scripts/progression-probe.py` + `world.json` | the routing map: rooms, exits, mob levels, resets, shops, guildmasters, and an exit graph the driver routes over |
| `internal/agent/worldmap.go` | loads the map; door-free BFS with a door-allowing fallback and per-session blocked edges |
| `internal/agent/progression.go` | the long-horizon policy: level-aware hunting, rest/flee thresholds, guild visits, shop trips, corpse looting, door opening, death recovery, and a `subgoal` on every decision |
| `sessionlog.Row.Subgoal`, `agent.Subgoaler` | the dataset carries the big-model layer next to the step choice |
| `internal/agent/state.go` | corpses are no longer read as live mobs (the first batch re-issued `kill citizen` at a corpse 77 times); equipped items are tracked by name; the practice catalog is parsed |
| `cmd/dp-play play -class-name -world-map -target-level -rest-pulses` | progression sessions; a rest pumps a whole Mud hour so one regen tick happens |
| `cmd/dp-play/play_steps.go idleFor` | large clock pumps get a 3 s quiet window; small fight batches keep the short one. Without this, a 700-pulse pump returned before the server had written, and the next command's answer was read as that command's output — a run of `stand` commands each reporting zero bytes |
| `scripts/run-progression.sh` | one worker per class, its own run directory, its own telnet and HTTP port, waiting for `world boot complete` before connecting |
| `scripts/stand-probe.py`, `scripts/practice-probe.py` | the wire-level probes behind the position and practice findings |

## 8. Decisions for Zach

1. **PL-004's scenario is red by design until #2160 merges.** Do you want it
   ledgered as an EXPECTED divergence now (with the `#2160` pin), or left out of
   the census until the fix lands and then used as its regression guard?
2. **PL-005 needs a second pass on the port side.** I can cite C's incapacitated
   path but not yet the port's "neck snaps in two" line. Is that worth another
   round, or should it wait for whoever owns the death path?
3. **Jail is a play-lab policy problem, not a port bug.** Should the next
   round's routing treat the city as off-limits for hunting (menagerie only), or
   also measure how long a sentence costs a session?
4. **The frozen clock makes each session a finite mob pool.** For a level-3/5/10
   climb, do you want sessions that advance the clock far enough for zone resets
   (a session would need to pump ~25 game-minutes per repopulate), or longer
   wall-clock runs on an unfrozen server?
5. **Coverage gaps in the sample:** no session produced a shop *purchase* (the
   character starts armed and poor) or a spell *cast* (the policy has no cast
   sub-goal). Do you want the next round to add a purse fixture and a cast
   sub-goal so those surfaces are replayed too?
6. **The two random-play sessions are 150 decisions each.** M2 ran 60 random
   rows; is 300 enough for the coverage set, or should random play scale with the
   directed volume?

