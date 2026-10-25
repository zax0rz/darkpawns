---
tags: [handoff, fidelity, play-lab, differential]
last_updated: 2026-10-25
author: DeepSeek (play lab)
---
# Play lab, round 1: a recorded session that replays, and one transport finding

The play lab is a new lane with two products: **differential findings** (play at
volume, report where C and Go differ) and **training data** for the planned
System One decision model. This round is **milestone 1 only**: prove that a
recorded play session replays as an oracle scenario where C and Go agree, then
stop and report. No fixes ship from this lane.

- **Lab repo (research tooling, not part of the port):** `~/dp-play-lab`,
  private, no remote. Session driver, scenario generator, session logs, probes.
- **Into `darkpawns` this round:** one scenario
  (`cmd/dp-oracle-diff/scenarios/play-lab-m1-walk-look-get.txt`) plus this note.
- **Findings:** one, transport-level, not scenario-reducible. Filed for an owner;
  not fixed here.

## 1. The replay proof

| | |
|---|---|
| scenario | `cmd/dp-oracle-diff/scenarios/play-lab-m1-walk-look-get.txt` |
| recorded session | `~/dp-play-lab/sessions/2026-10-25T10-20-m1.jsonl` (live clock) and `…T10-30-m1-frozen.jsonl` (frozen clock) |
| recorded against | local Go port, head `4fc5c8ab2`, `DP_ALLOW_NO_DB=1`, copied world |
| census head | `4fc5c8ab2` (the worktree `~/wt/deepseek-play-lab`) |
| fixture | `quiet-mobs` |
| probe | `look`, `north`, `look`, `south`, `inventory`, `drop sword`, `get sword`, `inventory` |

It plays a fresh Human Warrior from Kir Drax'in: the nanny creation flow, then
the newbie intro's landing room (8162 Temple Infirmary), one step north into
8161 Western Vestibule and back, and the character's own starting short sword
dropped and picked up.

**Census result — CLEAN on seeds 1, 2, 3, 5 and 8 with `--draw-parity`:**

| seed | run directory | verdict | draw |
|---|---|---|---|
| 1 | `2026-10-25/deepseek-playlab-m1-seed1` | CLEAN | `draw_divergent=0` |
| 2 | `2026-10-25/deepseek-playlab-m1-s2` | CLEAN | `draw_divergent=0` |
| 3 | `2026-10-25/deepseek-playlab-m1-s3` | CLEAN | `draw_divergent=0` |
| 5 | `2026-10-25/deepseek-playlab-m1-s5` | CLEAN | `draw_divergent=0` |
| 8 | `2026-10-25/deepseek-playlab-m1-s8` | CLEAN | `draw_divergent=0` |

One run directory is a mislabeled duplicate and should be ignored:
`deepseek-playlab-m1-seed2` was started without `ORACLE_REGRESSION_SEED=2` in the
environment, so it re-ran seed 1. It is left in place rather than deleted; the
real seed-2 run is `deepseek-playlab-m1-s2`.

Two independent sessions of the same script produced **byte-identical probe
output** (the only difference anywhere in either transcript is the RNG-rolled
vitals prompt, which Tier-1 normalization masks as `<PROMPT>`), so the replay is
not luck: the same command sequence reproduces the same bytes.


## 2. What makes a recorded session replayable

The brief asked for this specifically. Seven requirements, in the order they
bite:

1. **The character must be a mortal on both engines.** C crowns the first
   character of an empty player file Implementor (`src/db.c:3016`), and so does
   the port (`pkg/session/char_creation.go:575-596`). The lab runs the port with
   `-db /nonexistent-dp-play-lab/players.db` and `DP_ALLOW_NO_DB=1` so no store
   is opened and `isGod` stays false; the oracle's copied `lib/etc/players`
   already holds players. Without this the actor is an immortal in room 1204
   with no starter kit, and the scenario's mortal is a different character.
2. **The creation keystrokes must be identical on both sides.** The nanny draws
   RNG (stat rolls) from process start; an asymmetric setup shifts one engine's
   stream and every later draw is compared at the wrong position (R3a). The
   generator emits the recorded sequence into both `[setup:oracle]` and
   `[setup:port]`, so the two setups are the same keystrokes.
3. **The rooms the session touches must hold nothing RNG-placed.** `quiet-mobs`
   suppresses every mobile reset, and rooms 8161/8162 have no object reset in
   `lib/world/zon/80.zon` at all, so both rooms are empty apart from the actor.
4. **The first play command must come after the newbie intro pulse.** Both
   engines move a brand-new mortal out of the Burning Hut (8099) on the first
   `PULSE_MOBILE` — four seconds of wall clock
   (`pkg/engine/gameloop.go:33`) — and the harness's own settle pumps 40 pulses
   for exactly that reason (`cmd/dp-oracle-diff/main.go:36`).
5. **A frozen clock is the better default.** Against a `DP_CLOCK=1` server the
   driver sends `~dpclock pulse 40` instead of waiting, so the session's timing
   *is* the scenario's timing: the entry settle drops from 6.3s to 0.65s and no
   wall-clock race remains. Both variants replay (verified), but only the frozen
   one is reproducible pulse-for-pulse. `DP_FIXED_TIME=1770838461` should be set
   too, so the game calendar matches the harness's.
6. **One command per quiescence block.** A read ends after 500 ms of silence
   (the harness's own model); sending the next command before the block closes
   is how a session desynchronizes.
7. **The probe must avoid clock- and RNG-derived output.** `score`, `time`,
   `date`, `uptime` and the vitals prompt either need the frozen calendar or are
   masked by normalization. Anything a pulse or a draw can move does not belong
   in a replay proof.

**Objects.** No fixture is needed to get an item: the character's own starting
kit comes from `do_start()` (`src/class.c:501-535`, warrior → object 8037), which
is deterministic on both engines, so `drop sword` / `get sword` proves the
ground-item path with no `spawn-obj`. Using an RNG-placed world object instead
would not replay.

**The lab trap that nearly invalidated the first recording.** A world-only copy
(`-world lib/world`) makes every login text file read as missing: the port
derives its static-text root from the world flag's parent
(`pkg/game/world.go:381-383`, `World.LibTextDir = <world>/../text`), so the MOTD
silently became empty and the only symptom was a server-side
`could not read static login text` log line. The oracle harness copies
`lib/text` beside the world for the same reason
(`cmd/dp-oracle-diff/main.go:334-338`); the lab now copies the whole `lib` tree.
Worth knowing for anyone standing up a server by hand: a missing `lib/text`
produces a silently different login, not a boot failure.

## 3. Finding PL-001 — a rule-following client loops the port's telnet negotiation

**Class:** bytes (transport framing). **Not reducible to a scenario.**
**Suspect:** `pkg/telnet/listener.go` `readLineBody()` option handling —
`case DO` at `:596`, `case OPT_ECHO` at `:602-607`, `case OPT_MSSP` at `:608-610`
(`sendMSSP()` is `:821`).
**C reference:** `src/comm.c` `process_input()` (`:1828`) handles exactly two
client negotiations, by substring search: `IAC DO COMPRESS2` (`:1892`) and
`IAC DONT COMPRESS2` (`:1915`). Every other `WILL`/`WONT`/`DO`/`DONT` is ignored.

### What happens

The port answers a client's acknowledgement with the same request again:

```go
case OPT_ECHO:
    // Server manages ECHO explicitly (password masking).
    tc.write([]byte{IAC, WILL, opt})      // listener.go:602-607
case OPT_MSSP:
    tc.write([]byte{IAC, WILL, OPT_MSSP})
    tc.sendMSSP()                          // listener.go:608-610
```

A client that follows the telnet option rules — answer `WILL x` with `DO x`,
answer `DO x` with `WILL x` — therefore never stops: the server's `WILL ECHO`
gets a `DO ECHO`, which gets another `WILL ECHO`, and so on without bound. The
same loop runs for MSSP, and MSSP re-sends its whole payload each round, so the
loop is amplified.

Measured with the lab's probe (`~/dp-play-lab/scripts/nego-probe.py`, run at the
password prompt — the point where the server first sends `WILL ECHO`, C's
`echo_off()` at `src/comm.c:937`):

| engine | IAC frames in 3 s | bytes in 3 s |
|---|---|---|
| Go port | 245,591 (`WILL 70` ×122,796, `WILL 1` ×122,795) | **48,381,621** (~16 MB/s) |
| C oracle | 0 | 0 |

Raw probe output is retained at `~/dp-play-lab/sessions/nego-go.txt` and
`nego-c.txt`. The loop is also what first hung the lab's own session driver: it
sits in the password prompt forever, never reaching the game.

### Why no scenario can see it

The oracle harness's transport consumes IAC sequences and answers none of them
(`internal/oraclediff/conn.go:77-121`, `consumeIAC` at `:127`), so both engines
are driven by a client that never acknowledges an option. C's and the port's
*text* bytes therefore match — which is why 1,610 hand-written scenarios are
clean — while the port's *framing* behaviour diverges. This is the play lab's
first concrete answer to "what does the corpus miss": anything whose only
observer is a real client's telnet state machine.

Not in the ledger: `cmd/dp-oracle-diff/expected_divergences.tsv` has two lines
(a header and one `alias-cap` row) and neither mentions telnet, ECHO or MSSP.

### Find one, find the class (R5c)

The port already knows this failure mode. `case OPT_GMCP` (`:611-617`) is
guarded — "Answer only a DO the server has not already agreed to … re-asserting
it on the client's DO would start a negotiation loop" — but the identical
re-assertion two lines above it (ECHO, `:602-607`) and below it (MSSP, `:608-610`)
is not. The guard was written for one option and its siblings were left. The
class is: *every* server-initiated offer whose answer is met with a fresh
request. The three sites above are the whole class in this handler.

### Shape of a fix (not applied here)

An owner's fix is small and local: track the option state per descriptor and
answer a `DO` with `WILL` at most once per state change (the guard GMCP already
has), and the same for `DO MSSP`. It changes no game bytes, so it needs a
targeted census over the login scenarios plus a transport unit test that fails
on the reverted guard. **No fix, no rule text, and no ledger row is proposed as
accepted here — see Decisions for Zach.**


## 4. Observations that are not findings

- **Framing comparison, C vs Go.** C's greeting offers one option
  (`IAC WILL COMPRESS2`); the port offers `WONT ECHO`, `WILL MSSP`, `WILL GMCP`,
  `WILL COMPRESS2`, `WILL EOR`. GMCP/MSSP are documented port features
  (`docs/gmcp.md`), so this is an addition rather than a divergence — recorded
  here because it is the surface PL-001 lives on.
- **The password prompt's stray bytes already match.** C's `echo_on()` sends
  `IAC WONT ECHO` followed by two stray option bytes (`TELOPT_NAOFFD`,
  `TELOPT_NAOCRD` = `0x0D 0x0A`) as data (`src/comm.c:954-966`); the port sends
  `IAC WONT ECHO` and the next prompt's own leading CRLF. The two byte streams
  are identical at that point, verified by dumping both raw streams side by side
  (`~/dp-play-lab/sessions/raw-creation-c.txt` vs `raw-creation-go.txt`).
- **The name prompt's clear-line is a raw difference normalization erases.**
  C sends `"\x1B[2K\n\rBy what name do you wish to be known? "` (`src/ident.c:240`);
  the port sends a plain CRLF with no `ESC[2K` (no `2K` literal exists in the Go
  tree). The streams still normalize equal — Tier-1 strips the escape and its
  `\r\n`/`\n\r`/`\r` canonicalization makes both sides four newlines — which is
  why `character-creation` PASSes in the 2026-10-25 full census. Recorded as an
  observation, not a finding: raw colour bytes at the name prompt are outside what
  the census certifies, exactly as `docs/fidelity/depth/entry.tsv` already says
  for `entry.motd-mortal-color-on-normalized`.
- **A linkless leftover is visible in later sessions' room output.** In the raw
  dump, C's first room line includes `Rawdump the Warrior (linkless) is standing
  here.` — a character from an earlier probe that disconnected without quitting.
  The lab restarts its server before every recorded session for this reason
  (`scripts/run-session.sh`); a session recorded against a dirty world is not a
  replayable session.
- **The MOTD is byte-identical** at the PRESS RETURN step once the lab supplies
  `lib/text` (see §2): `Welcome to Dark Pawns: darkpawns.com 4300` through
  `**** So go there immediately, if not sooner.` plus the bare `\n` before
  `*** PRESS RETURN: `.

## Decisions for Zach

1. **The class name for PL-001.** The brief's classes are bytes, draw, state and
   timing. PL-001 is not game text; it is wire framing that only a real client's
   telnet state machine observes. I have filed it as **bytes (transport
   framing)**. If that stretches "bytes" past usefulness, say what the class
   should be — this is the first finding in the lane and it will set the pattern.
2. **Whether PL-001 gets a fix at all, and by whom.** It is a real DoS-shaped
   amplification (16 MB/s to one client, ~86k frames/s) reachable pre-auth at the
   password prompt, but it changes no game byte and no scenario can prove it. If
   it is in scope, the fix is the GMCP-style guard applied to ECHO and MSSP; I
   have not written it, per the brief's "no fixes".
3. **Whether the ledger wants a row.** `expected_divergences.tsv` cannot express
   this (no scenario reproduces it), so if PL-001 stays open it is invisible to
   every census. A durable note — or a transport unit test — would keep it from
   being lost.
4. **A rulebook proposal, for you to accept, edit or reject.** I am not adding
   it. Exact proposed text, if you want one:
   > **R6. A client's transport state machine is part of the game surface.** A
   > server negotiation the C original never makes — or an answer C never gives —
   > is player-facing even when the census cannot see it: the harness's transport
   > acknowledges no telnet option, so framing differences are invisible to every
   > scenario. Reproduce C's negotiation exactly, and guard every option against
   > the re-assertion loop with the same care as GMCP.
   >
   > Taught by PL-001 (2026-10-25): the port answered `DO ECHO` with `WILL ECHO`,
   > so a rule-following client looped at 16 MB/s while C was silent, and the
   > 1,610-scenario corpus was green throughout.
5. **Milestone 2 defaults.** I propose: record against a `DP_CLOCK=1` +
   `DP_FIXED_TIME=1770838461` server, drive pulses with `~dpclock pulse 40` for
   the entry settle (and for anything else that needs time), and treat "one
   command per quiescence block, no clock-derived output in the probe" as the
   replay contract. Say if you want a different default.
6. **The scenario's place in the corpus.** It is unclaimed (no depth manifest
   row), so it runs in every full census and never in a `--claims` run. Say
   whether you want it claimed at seeds 1,2,3,5,8 — that would need a manifest
   row, which I have not added.

## Handoff numbers (milestone 1)

- **Sessions recorded:** 3 (one live-clock probe run, two recorded sessions:
  live clock and frozen clock). 12 creation keystrokes + 8 play commands each.
- **Decisions recorded:** 0 — milestone 2's option enumeration and policies are
  not built yet, so there is no dataset and no policy mix to report.
- **Findings by class:** bytes (transport framing) 1; draw 0; state 0; timing 0.
- **Dataset size:** none yet (milestone 2).
- **Census:** 5/5 seeds CLEAN with `--draw-parity` (`draw_divergent=0` each).
- **Lab artifacts:** `~/dp-play-lab` (`0cc5c92` + this round's follow-up commit);
  sessions under `sessions/`, probe outputs `nego-go.txt` / `nego-c.txt`.

