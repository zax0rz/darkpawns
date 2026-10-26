---
tags: [handoff, fidelity, play-lab, progression, dataset]
last_updated: 2026-10-26
author: DeepSeek (play lab)
---
# Play lab, milestone 3 follow-up: the newbie route, and a climb that stays out of jail

Follow-up to `2026-10-25-deepseek-play-lab-m3.md`, on Zach's instructions of
2026-10-25. Two changes to the play lab, then the climb redone:

1. **Routing.** The directed policy's default early game is now the guard
   training centre: Temple of the Cross (8008) → west (8101) → north (8102) →
   west (16300 Guard Barracks) → `open door` → south (16301 The New Recruit
   Training Area), kill recruits.
2. **A hard rule.** The policy never attacks a mob in a city zone. Kir Drax'in
   answers violence with the jail, and a sentence costs a session far more than
   the kill pays. A jail test is a scenario, not a policy.
3. **The climb, redone** for level 3 (then 5), with the frozen clock pumped far
   enough for zone resets so killed rooms refill.

Findings only, no fixes. Local servers only. PL-005 (the death-path draw
divergence) is another agent's; nothing here touches it.

## 1. The route, and why the city was the wrong ground

Round 3's sessions hunted city citizens and spent up to 473 of 850 decisions in
The Main Holding Cell. The mechanic behind that: six mobs carry the specs that
cart a player off to jail — 8001, 8002, 8020, 8027, 8059 (`take_to_jail`) and
8060 (`wall_guard_ns`), all assigned in `src/spec_assign.c:285-291`, all standing
in Kir Drax'in. When one of them engages, `src/fight.c:1370-1396` sets the
victim to 1 hit point, moves them to room 8118 and sets
`GET_JAIL_TIMER = MAX(2, level/2)`.

The lab's world map (`world.json`) now carries `city_zones` (70 Kir Drax'in
Slums, 80 Kir Drax'in, 85 Kir Drax'in Extention — the zones named in the zon
headers) and a zone number per mob prototype, and `huntHere` skips any mob in
one of them. Nothing in the training centre carries a jail spec, so the new
ground is safe.

**Verified on the wire** (`scripts/recruit-probe.py`, and as a scenario):
`16300 → south` answers "The door seems to be closed.", `open door` answers
"Okay.", `south` enters the training area, and a recruit is there. The reduced
scenario `play-lab-m3b-recruit-route.txt` replays exactly that and is **clean**
against the C oracle at seed 1 (`draw-parity MATCH entries=4081`).

## 2. What the training centre pays, and how it refills

| mob | level | hit points | damage | experience |
|---|---:|---|---|---:|
| 16301 dirty farmer | 1 | 1d5+20 | 1d4+0 | 100 |
| 16302 new recruit | 1 | 1d5+20 | 1d4+0 | 100 |
| 16305 white guard trainee | 2 | 2d5+30 | 1d4+1 | 245 |
| 16306 green guard trainee | 3 | 3d5+40 | 1d4+1 | 400 |
| 16309 guard member initiate | 5 | 5d5+60 | 2d4+2 | 1,100 |
| 16350 priest fanatic | 6 | 6d5+70 | 3d4+3 | 1,800 |

Read from `lib/world/mob/163.mob`; the experience column is what every kill in
this round actually paid (every "You receive N experience" line in the six
sessions reads exactly the prototype's value).

The centre is a ladder, and the policy walks it by matching the room's top mob
level to the character's own level (a level-1 character fights level-1 recruits,
not the level-2 trainees that share the same first room).

**Repopulation.** Zone 163's lifespan is 10 (`lib/world/zon/163.zon`), and the
port ages a zone one game-minute per six zone ticks of 100 pulses
(`pkg/game/zone_clock.go:39-52`), so a reset is due after about **6,000 pulses**
(10 game-minutes). Reset mode 2 fires even with a player standing inside
(`zone_clock.go:64`), which is what makes waiting in the room work. Measured
with `scripts/pulse-cost-probe.py`: **6,000 pulses cost 20–25 s** of server time
on this host (about 6.4 ms per pulse while the v0.1.5 release census was
running), and a direct probe watched a room go from 2 recruits to 3 after one
pump. The policy's `wait` action pumps exactly that and is recorded as its own
sub-goal (`wait for the training centre to repopulate`).

**What stops paying.** `calc_level_diff` (`src/fight.c:659-685`) gives a solo
player a two-level slack and then cuts: full pay within five levels of the mob,
then −30 % above five, −50 % above ten, −70 % above fifteen. A level-1 recruit
therefore pays its full 100 experience up to **character level 8**, 70 % to
about 13, and 50 % to about 18. That is the answer to "where next": the centre
carries a class to about level 8 on recruits alone, and the trainees/initiates
carry it further without leaving the zone.

## 3. The climb, redone

Six sessions, one per class the creation menu offers, 900 directed decisions
each, target level 3. Same setup as round 3: own server per class, frozen clock,
`-entry-pulse 40`, a rest pumps 700 pulses (one Mud hour of regeneration) and a
repopulate wait pumps 6,000.

| class | decisions | experience | level | kills | deaths | exp/hour in the centre |
|---|---:|---:|---:|---:|---:|---:|
| thief | 213 | 1,033 | 1 | 8 | 0 | 18,412 |
| psionic | 288 | 761 | 1 | 6 | 0 | 12,557 |
| cleric | 292 | 732 | 1 | 4 | 0 | 11,315 |
| ninja | 252 | 670 | 1 | 5 | 0 | 10,978 |
| mage | 90 | 141 | 1 | 1 | 0 | 2,639 |
| warrior | 42 | 11 | 1 | 0 | 0 | stalled, see below |

**No class reached level 3 in this budget, and that is the honest headline.** The
level-2 threshold is 1,500 experience and level 3 is 3,000 (`FindExp`,
`pkg/game/limits_exp.go:107-137`); at the measured rates that is 6–24 minutes to
level 2 and 12–48 minutes to level 3 *per class*, before counting the guild
trips. The sessions were stopped at 90–292 decisions each once the rate was
established, because the deliverable Zach asked for is the yield, not the
wall-clock.

Two lab limitations show up in the table and are worth fixing before a level-5
attempt:

* **The warrior stalled.** Its session sits in 16301 with a recruit visible and
  the state agreeing, but `kill recruit` answers with nothing but the prompt —
  the character is already fighting that recruit, and the game says nothing when
  a kill repeats a fight in progress. The policy's refresh fires, looks, kills
  again, and the fight never resolves inside the decision budget. Every other
  class kills normally. Not a port issue (the same commands against the C oracle
  in `play-lab-m3b-recruit-route.txt` are byte-identical); a policy gap.
* **The mage is slow.** One kill in 90 decisions: the class starts with a dagger
  and no spell in the policy's repertoire, so its melee round is weak and its
  rests are long.



## 4. Experience per hour in the recruit area, and where next

`scripts/recruit-yield.py`, over the decisions whose room was in zone 163. Wall
time is **measured**: the session writes one latency row per send in the same
order as its step rows, so the rows inside the phase window pair up by index and
their quiet times sum to the phase's real cost.

| class | kills in the centre | experience | phase wall | exp/hour | kills/hour |
|---|---:|---:|---:|---:|---:|
| thief | 8 | 800 | 156 s | 18,412 | 184 |
| psionic | 6 | 600 | 172 s | 12,557 | 126 |
| cleric | 4 | 545 | 173 s | 11,315 | 83 |
| ninja | 5 | 500 | 164 s | 10,978 | 110 |
| mage | 1 | 100 | 136 s | 2,639 | 26 |

The centre pays the prototype's experience exactly (100 for a recruit; the
cleric's 545 over four kills includes one 245-experience white trainee). The
spread (2.6k–18.4k) is mostly how much of the phase went on walking and resting
rather than killing: the thief found the room full and stayed, the mage spent its
phase resting at low health with a dagger.



**Where each class should go next.** The centre is a ladder and the policy walks
it by matching the room's top mob level to the character's own level:

| character level | ground | why |
|---|---|---|
| 1–2 | 16301–16304 recruits and farmers (level 1, 100 exp) | full pay, 21–25 hit points, safe with the starting kit |
| 2–3 | 16305–16311 white trainees (level 2, 245 exp) | 32–40 hit points: winnable, and the pay doubles |
| 3–4 | 16312–16315 green trainees (level 3, 400 exp) | |
| 4–5 | 16316–16319 (level 4) | |
| 5–8 | 16320–16324 guard initiates and the priest fanatic (level 5–6, 1,100–1,800 exp) | the centre's top band |
| 9+ | leave the centre: `calc_level_diff` starts cutting recruit pay at level 9 | the 145xx menagerie (levels 1–6) and beyond, chosen by the level band |

The whole centre repopulates on one 10-game-minute cycle, so a class can climb
inside it without walking anywhere; the trip that costs time is the guild visit
for practices (the warrior's guild is nine steps away, in the city).

## 5. Lab changes this follow-up (`~/dp-play-lab`)

| change | why |
|---|---|
| `scripts/progression-probe.py` | the map now carries each mob prototype's experience, zone and **long description**, each room's zone, `city_zones`, and rooms reachable **through doors** (the training centre is behind a closed door, and a door-free-only map silently dropped the whole newbie ground) |
| `internal/agent/progression.go` | the newbie ladder (training centre first while it pays, matched to the character's level), the `wait` action that pumps a zone lifespan, the city-zone ban, `open door` when every exit is blocked, a flee streak that gives up after three attempts and fights, rest allowed beside a mob that is not attacking |
| `internal/agent/state.go` | a mob line is now recognised by its **exact long description** from the world files; objects and mobs both render as a long description ("A new recruit is wandering around here." / "A small fountain is here.") and a shared keyword cannot tell them apart |
| `cmd/dp-play play_steps.go` | `requireGMCP`: a session that loses the GMCP handshake fails loudly. Room.Info is the only source of room vnums and four rooms in the training centre share the name "Level One", so a session without it plays blind — which is what one probe did under load |
| `cmd/dp-play` `-rest-pulses`, per-option `Pulses` | a rest pumps one Mud hour; a repopulate wait pumps ten game-minutes |
| `scripts/recruit-probe.py`, `scripts/pulse-cost-probe.py`, `scripts/recruit-yield.py` | the wire probe for the route, the measured cost of a pulse pump, and the per-class yield report |

## 6. Differential

| scenario | what it replays | verdict at seed 1 |
|---|---|---|
| `play-lab-m3b-recruit-route.txt` | the newbie route: the closed door, `open door`, the walk in, the first recruit fight | **clean**, `draw-parity MATCH entries=4081` |

Round 3's scenarios are unchanged: `play-lab-m3-practice-catalog.txt` clean
(fixed by #2160), `play-lab-m3-jail-on-attacking-guard.txt` clean,
`play-lab-m3-death-in-combat.txt` still red (PL-005, another agent's).

## 7. Decisions for Zach

1. **The training centre is now the default early game.** Should the policy also
   *buy* the cheap weapon and armour before the first trainee band (Darek's and
   Jack's are eight to ten steps from the centre), or is the starting kit enough
   for recruits?
2. **Recruits are a coin flip at level 1** (21 hit points, a 1d4 mob over a ten
   round fight). The policy now rests to 85 % and flees at 45 % before level 3.
   Do you want the next round to measure deaths per 100 kills as a balance
   signal, or is that outside the lab's remit?
3. **The guild trip costs a round walk from the centre.** Keep visiting on every
   level-up, or batch practices to every third level?
4. **Levels 9+ need a new ground** (`calc_level_diff` cuts recruit pay from
   level 9). Should the next round's ladder continue into the menagerie and the
   desert, or stop at the centre and report the handoff point?
5. **PL-005 remains another agent's**; nothing in this follow-up touches the
   death path. Confirm that is still the split.

