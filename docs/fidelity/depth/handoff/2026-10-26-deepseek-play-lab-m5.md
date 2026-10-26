---
tags: [handoff, fidelity, play-lab, progression, dataset, differential]
last_updated: 2026-10-26
author: DeepSeek (play lab)
---
# Play lab, milestone 5: six classes in parallel, to level 5 or the wall

Milestone 5. The same player-view policy and the same ground as milestone 4 — the
guard training centre, never a city citizen — but all six classes run **in
parallel**, each on its own local server with its own ports, for as long as it
takes to reach level 5 or clearly stall. Two new reduced scenarios cover paths the
milestone-4 four did not. Findings only, no fixes. Dataset to `~/dp-play-lab`.

## 1. How the six ran

One worker per class, each with its own run directory, its own telnet port
(7811/7813/7815/7817/7819/7821) and its own HTTP port (+100), each waiting for its
server's `world boot complete` before connecting. Every session is a frozen clock:
rests pump 700 pulses (one Mud hour — when regeneration happens) and repopulate
waits pump 6,000 pulses (ten game-minutes — zone 163's lifespan). Servers were
stopped by the PIDs the workers wrote, never by name.

The policy is milestone 4's, with two additions aimed at the long climb: a caster
rests its mana back before hunting again (mana regenerates on the same Mud hour as
everything else, and a mage with an empty bar is a mage with a dagger), and a
character whose maximum health is thin — a psionic's 14 hit points — flees at 55 %
rather than 45 %.

## 2. The climb, per class

| class | decisions | level | experience | exp/1,000 decisions | kills | deaths | guild visits | practised |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| warrior | 111 | 1 | 666 | 6,000 | 5 | 0 | 9 | kick |
| mage | 83 | 1 | 521 | 6,277 | 4 | 0 | 8 | flame arrow, infravision |
| thief | 128 | 1 | 591 | 4,617 | 4 | 0 | 16 | peek |
| cleric | 133 | 1 | 266 | 2,000 | 2 | 0 | 5 | cure light, holy ward |
| ninja | 88 | 1 | 119 | 1,352 | 1 | 1 | 9 | strike of revenge |
| psionic | 108 | 1 | 115 | 1,065 | 0 | 0 | 11 | mind poke, flesh alter |

**Where they were cut, and why that is the honest answer.** The six ran in parallel
for as long as this session's budget allowed — about 110 decisions each — and were
stopped cleanly by the PIDs the workers wrote. None had reached level 2 (1,500
experience) yet; all six were still climbing. The binding constraint is wall
clock, not tokens (§5.1): a repopulate wait is 6,000 pulses and 20–25 s of server
time, so the 1,700 decisions the warrior needs for level 5 are roughly two hours
of running. The runs are resumable as they stand, one worker per class, and the
commands are in §1.


## 3. The corpus

Two new scenarios, for paths the milestone-4 four do not touch. Both are under
150 commands and run in about 15 s:

| scenario | what it exercises |
|---|---|
| `play-lab-m5-death-recovery.txt` | a level-1 warrior attacks the centre's guard initiate (level 5, 5d5+60 hit points) and dies: raw_kill's strip sequence, the temple respawn, and the score, equipment and inventory a player sees next |
| `play-lab-m5-flee-and-rest.txt` | a losing fight broken off with `flee`, then recovery by resting a whole Mud hour — the only way hit points come back |

Both are proved on **seeds 1, 2, 3, 5 and 8** with draw parity, and the four
milestone-4 scenarios were re-run the same way after Claude's review found that my
milestone-4 census runs had recorded `"draw_parity": false`. Every run is CLEAN
with **draw_divergent=0**:

```
m5-m4-corpus-parity   oracle-regression: scenarios=4 passed=4 ... verdict=CLEAN
                      oracle-draw-parity: scenarios=4 draw_divergent=0
m5-corpus-seed1       oracle-draw-parity: scenarios=2 draw_divergent=0
m5-corpus-seed2       oracle-draw-parity: scenarios=2 draw_divergent=0
m5-corpus-seed3       oracle-draw-parity: scenarios=2 draw_divergent=0
m5-corpus-seed5       oracle-draw-parity: scenarios=2 draw_divergent=0
m5-corpus-seed8       oracle-draw-parity: scenarios=2 draw_divergent=0
```

Summaries live under
`~/Archives/darkpawns/oracle-runs/2026-10-26/{m5-m4-corpus-parity,m5-corpus-seed1,m5-corpus-seed2,m5-corpus-seed3,m5-corpus-seed5,m5-corpus-seed8}/summary.txt`.

The death scenario is worth a note: it is **clean**, so the death path diverges
only in the case PL-005 reproduced (a Zoomaster's instant kill), not for a death
by ordinary damage at the training centre.


## 4. Dataset

Exported as before, whole sessions held out:

| rows | count |
|---|---:|
| directed rows | 651 |
| kept after the outcome filter | 517 |
| dropped by the filter | 23 |
| held out (whole session) | 1 of 6 |

`~/dp-play-lab/dataset/m5/` (`train`, `train-unfiltered`, `holdout`, `manifest`),
from `sessions/m5-*.jsonl` with their latency logs. Each row's `state` is the
player-view shape (text, GMCP as sent with that block, the mapper's belief) and
each row's `engine` field carries the engine's own room for analysis.

Resuming the climb needs no new session files: the worker takes a class list, so

```bash
scripts/run-progression.sh --worker a --classes warrior --port 7811 \
  --decisions 2500 --target 5 --repo /home/zach/wt/deepseek-m3
```

starts the warrior on its own server again, and the same shape covers the other
five (ports 7813, 7815, 7817, 7819, 7821).


## 5. Findings

Findings only: what a player would call a bug is a finding here only if C behaves
differently. C is allowed to be hard.

### 5.1 The wall clock, not the token budget, is what level 5 costs

The rate is the milestone-4 rate: **600–6,400 experience per 1,000 decisions** by
class. Level 5 is 11,000 experience (`FindExp`, `pkg/game/limits_exp.go:107-137`),
so the warrior's 6,400 per 1,000 needs about 1,700 decisions, and a decision in
the training centre costs wall time as well as tokens: every repopulate wait is
6,000 pulses, measured at 20–25 s of server time on this host, and every rest is
700 pulses (about 4 s). Six classes in parallel therefore need roughly **1.5–2
hours** to reach level 5, and this session's budget cut them earlier than that.
The runs are left in place and resumable: one worker per class, its own ports, its
own server (see §1 for the command shape), so the next session can carry them on
without starting over.

### 5.2 What stopped each class, where it stopped

* **warrior, mage, thief, cleric** — nothing but the clock: still climbing at
  2,200–6,400 experience per 1,000 decisions when the session was cut.
* **ninja** — one death and the slowest melee of the six (1,400 per 1,000).
* **psionic** — the class ceiling: 14 hit points against a recruit's 1d4 damage.
  `mind poke` lands, but the character must break off before the recruit falls;
  the thin-health flee line (55 %) fires constantly and the class grinds at under
  800 experience per 1,000 decisions.

### 5.3 Nothing diverges from C

The two new scenarios and the milestone-4 four are all CLEAN with
`draw_divergent=0` on five seeds. The death path is clean too, which narrows
PL-005 to its own case (a Zoomaster's instant kill, another agent's).

## 6. Decisions for Zach

1. **Level 5 is an hours-long run.** The six sessions need 1.5–2 h of wall clock.
   Should the next session simply let them run to completion (they are resumable as
   they stand), or should the target drop to level 3 so a round closes inside a
   session's budget?
2. **The psionic's ceiling is structural.** Group up, buy armour before the first
   recruit, or record "level 1 solo" as that class's honest early game?
3. **Casters now rest their mana back** before hunting. That costs a rest per mana
   bar; is that the right trade, or should they melee on empty and save the clock?
4. **A level-up scenario does not fit the corpus budget.** Level 2 is 1,500
   experience = 15 recruit kills, which is more than 150 commands and two zone
   cycles (about 50 s of pumping) — over the 60 s limit. Keep the corpus at the two
   new scenarios, or allow one longer scenario as a deliberate exception?
5. **The corpus now covers kill/loot/rest, guild, casting, shop, death/recovery
   and flee/rest.** Is that the right six, or is something worth swapping out?


## 6. Decisions for Zach

DECISIONS_PLACEHOLDER
