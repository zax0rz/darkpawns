# 2026-10-04 — A 30-minute playtest, and a level scale from someone else's MUD

## Scope

- **Tracks:** legacy-port fidelity; test-oracle validity; AI-assisted porting
- **Window:** 2026-10-04, the live playtest gate for zax0rz/darkpawns#1774 (heartbeat output staging)
- **Primary artifacts:** Linear DP-1383, DP-1384, DP-1385 and DP-1386; zax0rz/darkpawns#1775 (level thresholds) and its census `~/Archives/darkpawns/oracle-runs/2026-10-04/level-scale-class/`; the playtest report comment on #1774; earlier commit `2bda896de` (admin panel role ladder); the playtest server under `~/Archives/darkpawns/playtests/2026-10-03-pr-1774/`
- **Status:** artifact-grounded field note. It complements PF-053 (where Phase 4 defects were found) with a different detection method.

## What the playtest found

Zach played a disposable local server for about 30 minutes in a browser and telnet, using an 18-item checklist written to stress #1774. Every check on #1774 passed. The same session found five defects, none caused by #1774, and every one invisible to the combined census, which was green across roughly 3,200 scenario/seed pairs:

| Found as | Defect | C authority |
|---|---|---|
| An Implementor is refused the immortal board ("You try but fail to understand the holy words.") | Board read/write/remove levels 50/60/61 on a ladder that ends at 40 | `src/boards.c:93-98` |
| A follower kept following after its master died and went back to loot | Followers not released on death (DP-1383) | `src/fight.c:580` → `src/handler.c:1116-1117` |
| The newbie channel wasn't yellow | No gen_comm channel colored at all (DP-1384) | `src/act.comm.c:1257-1265` |
| A character was disconnected after 5 idle minutes with no void | Invented telnet read timeout on in-game players (DP-1385) | `src/limits.c:424-451` |
| The combat prompt lacked the condition readout; AFK wasn't red | Prompt target/tank segments missing, AFK/INACTIVE uncolored (DP-1386) | `src/comm.c:1120-1154` |

Each one is either Go-only (the timeout) or on a path no scenario exercised: an immortal reading a board, a follower present at a player's death, a colored channel listener, an idle telnet session, a combat prompt with display flags set. That matches PF-053's split exactly, found by a human instead of a reviewer.

## The foreign level scale

The board table was one instance of a class. A sweep (#1775) found the same mistake in thief NPC stealing (`>= 50`, C `>= LVL_IMMORT`, `src/spec_procs.c:307`) and four spells that treat immortals as level 100 or more (hellfire, meteor swarm, mass affects, mindsight; C uses `LVL_IMMORT`/`LEVEL_IMMORT` at `src/spells.c:727`, `:922-923`, `:1120` and `src/magic.c:1537`). Effects in play: thieves stole from immortals, area spells damaged them, and the immortal recoil in mindsight never fired. Commit `2bda896de` had earlier removed an admin panel role ladder built on a nonexistent level 50.

Dark Pawns ends at `LVL_IMPL` = 40 (`src/structs.h:610-623`). The values 50, 60, 61 and 100 match other DikuMUD derivatives' level ladders, not this one. These constants were plausible MUD knowledge applied where the port's own C said otherwise. It's the same failure shape as the `yank` paraphrase (PF-002): the code reads correctly, carries a confident comment ("Skip immortals"), and is wrong by the port's own law (R4). No oracle scenario had an immortal in the affected role, so nothing failed.

## Methodological observations

- **A short human playtest is a distinct detector.** It found five defects in one sitting that weeks of differential testing had not, because a person exercises the everyday paths (prompts, colors, idling, following) that scenarios are rarely written for.
- **Write the checklist around the change, then let the player wander.** The checklist proved #1774. Everything else came from the player doing ordinary things in between.
- **Domain-knowledge leakage is a reusable search.** Once one foreign constant is found, grep for the whole value set (41–199 used as levels). The #1775 sweep found five more sites and cleared the rest with stated reasons.
