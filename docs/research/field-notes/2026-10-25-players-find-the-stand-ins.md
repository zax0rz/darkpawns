# Players find the stand-ins; the play lab finds them too

2026-10-25. Field note by Claude.

## Players found what the census couldn't

Over 24 and 25 October, three returning players on production (Egon, Lucci and Sai) reported five bugs, and the fidelity census had missed all of them. Each traced to the same shape: **a faithfully ported C path sitting next to a hand-written stand-in that paraphrased it.**

| Report | The faithful path | The stand-in | PR |
|---|---|---|---|
| "You flee into Torak" | C hunters call `do_gen_door` + `perform_move` (`src/graph.c:320-340`) | `mobOpenDoor` (cleared the lock bit), `mobPerformMove` (no closed-exit check) | #2146, #2147 |
| Lantern doesn't light the room | movement-path light accounting | `EquipItem` never counted lights (`src/handler.c:737-740`) | #2151 |
| Spells survive death | the `combat.RawKill` callback bridge | `handlePlayerDeath` skipped `raw_kill`'s strip (`src/fight.c:541-572`) | #2158 |
| Mana regen higher standing than sleeping | hit/move regen gear | the mana call inverted the sleeping-only rule (`src/limits.c:89-95`) | in flight |
| Learned skills show "(not learned)" | guild learning via `SkillStorageName` | the practice list read the display name | in flight |

The census only checks what someone thought to script. These bugs live in what players do constantly: walking, dying, holding a light, sleeping to regen, typing `practice`. In the doors case the faithful path was the *rare* one. The stand-ins sat on the hot paths.

A follow-up audit (#2148) inventoried all 173 C sites where a mob calls a player command directly. **127 were stand-ins and were replaced** with the real command bodies.

## The play lab finds the same class at volume

DeepSeek's play lab (#2150, #2157) plays the game in a closed loop: read the state, list the options, choose, act, observe, log. It then replays the commands as oracle scenarios. Milestone 2 (340 decisions, 8 sessions) independently rediscovered the lantern bug from a "hold a light in a dark room" goal, and found PL-001: a telnet negotiation loop of about 16 MB/s, invisible to the harness, which swallows negotiation bytes. The same sessions produce a `{state, questions, gold}` dataset for a planned small decision model, so one activity yields both differential findings and training data.

## Latency: the port is faster than the original

Measured in the same seeded sessions, on a frozen clock: Go's median time from send to first byte was **0.6 ms**, C's was **99.3 ms**. C reads input once per 100 ms pass (`OPT_USEC`, `src/structs.h:628`; `src/comm.c:545`, `:580`). Go answers immediately when nothing is queued, and still drains one command per pulse once a queue builds (`pkg/session/manager.go:1184-1190`). R1 governs bytes, not timing, so this is recorded as a difference, not a violation.
