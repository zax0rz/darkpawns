---
tags: [handoff, fidelity, play-lab, progression, dataset, differential]
last_updated: 2026-10-26
author: DeepSeek (play lab)
---
# Play lab, milestone 4: the real climb, and the rate the game allows

Milestone 4 of the play lab. Every class starts in the guard training centre and
climbs; the corpus gains four reduced scenarios, each proved on five seeds; the
dataset is re-exported. Findings only, no fixes. Local servers only, frozen
clock, census concurrent.

**Headline.** Under the player-view policy every class climbs the recruit ladder
without touching the city, and the rate is measurable: **925–5,982 exp per 1,000
decisions** by class, which puts level 2 within a session's budget and level 5
several sessions outside it. What stops the climb is the game's arithmetic — one mob per room per ten
game-minute zone cycle, and a level-5 threshold of 11,000 experience — not a bug.

## 1. What the policy does now

* **Start:** the guard training centre, every class, from the newbie start
  (8162 → north, east, south, west, north, west, south through the closed door at
  16300, which `open door` opens). City zones are never hunted: the six mobs that
  carry a player to jail all stand in Kir Drax'in.
* **The ladder:** inside the centre the kill band is the character's *own* level
  (recruits at 1, white trainees at 2, green trainees at 3, then 4, then the
  guard initiates at 5), because the centre is a ladder and a level-1 character
  that attacks a level-2 trainee loses. Outside it, one level above.
* **The clock:** a rest pumps 700 pulses (one Mud hour, which is when
  regeneration happens: `SECS_PER_MUD_HOUR` 63 × 10), and a repopulate wait pumps
  6,000 pulses (ten game-minutes, zone 163's lifespan), recorded as its own
  sub-goal.
* **Onward when it stops paying:** `calc_level_diff` (`src/fight.c:659-685`) keeps
  a mob's full value until the character is eight levels above it; when the
  character passes that, the sub-goal becomes
  `"<room> (level N mobs) no longer pays at level M; go to <next room>"` and the
  ladder moves up a band.
* **Powers:** casters cast, psionics **will** (`pkg/session/cast_cmds.go` refuses
  the wrong verb by name), and a power the game refuses — unknown, unlearned, or
  answered with "Cast what?!?" — is retired after two tries so the character falls
  back to steel.

## 2. The climb, per class

| class | decisions | level | experience | exp/1,000 decisions | kills | deaths | guild visits | skills/powers practised |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| warrior | 111 | 1 | 664 | 5,982 | 5 | 0 | 9 | kick |
| mage | 86 | 1 | 511 | 5,942 | 3 | 0 | 8 | flame arrow, infravision |
| thief | 126 | 1 | 591 | 4,690 | 4 | 0 | 16 | peek |
| cleric | 133 | 1 | 266 | 2,000 | 2 | 0 | 5 | cure light, holy ward |
| ninja | 88 | 1 | 119 | 1,352 | 1 | 1 | 9 | strike of revenge |
| psionic | 107 | 1 | 99 | 925 | 0 | 0 | 11 | mind poke, flesh alter |

**All six stopped at level 1**, and the honest reason is the budget: level 2 is
1,500 experience and the best of them had 664 when the sessions were cut at
86–133 decisions rather than the 600 planned. A repopulate wait is 6,000 pulses
and costs 20–25 s of server time, so a decision in the centre is expensive in
wall time even when it is cheap in tokens; six sessions in parallel had spent
about half an hour to get this far. The measurement that matters is the rate,
and it is in the table.

**What stopped further progress**, per class:

* **warrior, mage, thief, cleric** — nothing but the clock: each was still
  climbing at 2,000–6,000 experience per 1,000 decisions when the session ended.
  At the warrior's rate level 2 is about 220 more decisions and level 5 about
  1,800.
* **ninja** — one death (a recruit that got the better of a 16-hit-point
  character) and the slowest melee of the six.
* **psionic** — the class ceiling of §5.2: 14 hit points against a recruit's
  1d4, so every fight is a flight; `mind poke` lands but the class cannot finish
  a kill before it has to break off.


## 3. The corpus: four reduced scenarios, five seeds each

| scenario | what it exercises | commands |
|---|---|---:|
| `play-lab-m4-recruit-loop.txt` | the recruit kill → corpse → rest loop, and the closed door on the way in | ~40 |
| `play-lab-m4-guild-practice.txt` | the guild visit: catalog, four practices, catalog again | ~30 |
| `play-lab-m4-mage-cast.txt` | casting in combat: an unknown spell refused, then flame arrow, then steel | ~35 |
| `play-lab-m4-shop-buy-wear.txt` | the shop: sell the starting sword, list, buy a club, wield it | ~28 |

Proved with a targeted census on **seeds 1, 2, 3, 5 and 8**, `--draw-parity`:
every seed CLEAN, four scenarios passed, no timeouts, 56–75 s for the whole
four-scenario set (about 15 s each, against the 240 s per-scenario limit).

| seed | verdict | scenarios | elapsed |
|---|---|---:|---:|
| 1 | CLEAN | 4 passed | 56 s |
| 2 | CLEAN | 4 passed | 67 s |
| 3 | CLEAN | 4 passed | 70 s |
| 5 | CLEAN | 4 passed | 72 s |
| 8 | CLEAN | 4 passed | 75 s |

## 4. Dataset

Exported as before, whole sessions held out:

| rows | count |
|---|---:|
| directed rows | 651 |
| kept after the outcome filter | 517 |
| dropped by the filter | 23 |
| held out (whole session) | 1 of 6 |

The dataset is `~/dp-play-lab/dataset/m4/` (`train`, `train-unfiltered`,
`holdout`, `manifest`), and the sessions it came from are
`sessions/m4-*.jsonl` with their latency logs. Each row's `state` is the
player-view shape (text, GMCP as sent, the mapper's belief) and each row's
`engine` field carries the true room for analysis.


## 5. Findings

Findings only: what a *player* would call a bug is a finding here only if C
behaves differently. C is allowed to be hard, and the climb's ceiling is the
game's arithmetic.

### 5.1 The rate is the zone cycle, not the policy

The training centre's resets add **one mob per room per ten game-minute cycle**
(`M <if> <mob> <max> <room>` loads one while the world count is under the cap;
zone 163's lifespan is 10, `lib/world/zon/163.zon`). A repopulate wait is 6,000
pulses, measured at 20–25 s of server time on this host. The centre's level-1
band is four rooms of recruits and farmers at 100 experience each, so a cycle
yields a few hundred experience, and the rest of a decision goes to rests (700
pulses — one Mud hour, which is when regeneration happens) and guild visits.

Level 2 is 1,500 experience, level 3 is 3,000 and level 5 is 11,000
(`FindExp`, `pkg/game/limits_exp.go:107-137`). At the measured rate the lower
levels fit in a session and level 5 needs several: a budget fact, not a defect.
The measured rates are 925–5,982 experience per 1,000 decisions (§2).
Nothing in this round diverges from the C oracle — every scenario the corpus
added is clean on five seeds with draw parity.

### 5.2 Low-hit-point classes cannot solo a recruit at level 1

A psionic starts with 14 hit points; a recruit has 21–25 and hits for 1d4. The
psionic's level-1 power is `mind poke`, and its first two attempts at the
stronger `mind attack` are refused ("You are not learned in that power!") before
the policy retires it and falls back to steel. The result is a losing fight that
the policy escapes by fleeing at 45 % and re-entering after a rest — correct
play that still costs the session decisions. The mage has 100 mana and
`flame arrow`, so it kills; warrior, thief, cleric and ninja fight with steel and
win. Balance, not a bug: the same fight in C ends the same way.

### 5.3 The power verb differs by class

Psionics (and mystics) **will** their powers: `cast 'mind poke' recruit` answers
"Psionics 'will' things, not 'cast' them!" (`pkg/session/cast_cmds.go:342`), and
the power's catalog name is "mind poke", with a space. A lab-side bug until this
round — the runner passed the wrong verb and the wrong name — recorded here
because any future spell work has to respect it.

### 5.4 A fight in progress is not re-attackable

`kill <target>` while already fighting that target is answered with **nothing at
all**: no message, just the prompt. The warrior's session stalled on this in the
last round. The driver now advances a fight whenever one is in progress, whatever
the decision was, and the session no longer dies there.

## 6. Decisions for Zach

1. **Level 5 needs a longer budget than one session.** The measured rate puts it
   several sessions out. Should the next round run one class for hours (a
   long-horizon single-class session), or accept level 2–3 per class and report
   the rate?
2. **Low-hit-point classes stall on recruits.** A solo psionic cannot beat one at
   level 1. Should the policy group up (a lab-side party), buy armour first, or
   record the ceiling as that class's honest early game?
3. **The guild trip costs about ten decisions per level.** Keep visiting on every
   level-up, or batch to every third?
4. **Self-healing is not in the policy.** A cleric can cast `cure light` on itself
   but the policy only casts at targets. Worth adding for survival, or out of
   scope?
5. **The four corpus scenarios cover kill/loot/rest, guild, casting and shop.**
   Is that the right four, or should a door-and-dark-room scenario replace one?
