# Whirlpool clinic round: spec parity, room_activity, and the prompt chase — 2026-09-03

Branch `glm/depth-whirlpool` (from `origin/main` @ b5641d9b7). Converts the six
`mob.whirlpool-*` rows of `docs/fidelity/depth/spec-procs.tsv` from `blocked`
to `oracle-green-multiseed` via the new `spec-proc-whirlpool` scenario
(seeds 1,2,3,5,8), and ports C `room_activity` plus the `process_output`
prompt frame that pulse-time output requires.

## What actually shipped (five fixes, one was pure coverage)

1. **Async prompt chase (the true form of the "vanishing messages" ghost).**
   The pickup note diagnosed a writeLoop "prompt-gating" bug. On current main
   that mechanism does not exist: a standalone two-connection raw-socket repro
   (doctored world copy, `M 0 12200 999 8162` injected in 80.zon, RANDZON
   cleared) proved async pulse output **is delivered immediately** to an idle
   player. The real fidelity gap revealed itself only when a pumped pulse
   produced output that survived normalization: C's `process_output`
   (comm.c:1615-1660) appends `\r\n` + `make_prompt` to **every** output
   flush, so pulse-time output is always chased by a prompt — including the
   vitals fields (`23H 100M 83V > `, negative HP at seed 8 renders `-2H…`,
   whose leading dash defeats the normalizer's prompt-dropping). Go delivered
   the bytes but never the trailing prompt. Fixed by:
   - `Session.outputSincePrompt` counter (incremented in `SendMessage` and the
     manager `MessageSink`);
   - `SendPrompt` prepends C's non-compact `\r\n` frame when output was
     flushed since the previous prompt;
   - `Manager.PumpPulses` sweeps sessions after the pumped heartbeats and
     emits the owed prompt;
   - `promptText` now renders the `PRF_DISPHP/DISPMANA/DISPMOVE` vitals
     (C comm.c:1070-1105 order) when the infobar is off. Colors are omitted
     (ANSI is normalized away by the differential and Go's prompt never
     carried them; raw-wire color parity remains a pre-existing gap).
   Verified standalone (idle peer receives pull lines + `\r\n24H 100M 86V > `
   with no input of its own) and by the scenario's seed-8 drowning transcript.
   `pkg/telnet`'s `TestPromptAfterCommandOutput` was updated to the faithful
   frame (its race-detection intent is unchanged).

2. **Entry/creation-time spec dispatch: already faithful, no source change.**
   Static audit of every dispatch site on main: command-time
   (`pkg/session/commands.go:701+`, mirrors interpreter.c:947 `special()`),
   pulse-time (`pkg/game/mobact.go:193`, mirrors mobact.c:82-93), combat-time
   (`pkg/session/manager.go` `SetMobSpecialFunc`, mirrors the
   `MOB_FLAGGED(ch, MOB_SPEC)` call at the end of fight.c `perform_violence`),
   and follower movement (`pkg/game/act_movement.go:472` mirrors
   act.movement.c:349 `perform_move(k->follower, dir, 1)` → do_simple_move's
   special check). No entry/creation path exists; the raw-socket repro
   confirmed no pull during creation or idle. The note's root cause 2 was an
   artifact of the older clinic environment (or a desynced creation dance
   leaving post-entry commands, which trigger C's legitimate command-time
   dispatch).

3. **specWhirlpool destination draw (R3/R4).** Go drew `dprng.Number(4600,
   4699)` over vnums with an invented 100-iteration cap and a nil-room skip.
   C draws `number(real_room(4600), real_room(4699))` — an integer over the
   vnum-sorted room-index (rnum) space, rejection loop unbounded. Added
   `World.sortedRoomVNums` (the vnum-ascending room set; C's world[] is
   binary-searched ascending, db.c:3083) with `RealRoomIndex` /
   `RoomVNumByIndex` helpers; the spec now draws the same absolute index
   bounds and redraws unboundedly. Also fixed the victim lines' doubled
   terminators (old `sendToChar` appended a second CRLF; now routed through
   `SendMessage` with C's spec_procs2.c:267-268 framing).

4. **PRF_NOHASSLE victim gate (spec_procs2.c:256).** `specWhirlpool` now
   skips NOHASSLE players exactly like C. The scenario proves it live: a God
   (`nohassle` + `goto 8161`) stands with the mortal peer when the pulse
   lands; pre-fix Go pulled the God, desyncing the draw stream.

5. **room_activity port (comm.c:690-756) into the `OnRoomActivity` seam.**
   New `pkg/game/room_activity.go` + wiring in `cmd/server/main.go`: AFF_FLAMING
   → 15 SPELL_FLAMESTRIKE; SECT_UNDERWATER without WATERBREATHE → 25
   SPELL_DROWNING; SECT_WATER_NOSWIM without WATERWALK/FLY/boat → 25
   SPELL_DROWNING (all NOHASSLE-exempt); PC-only pulse-time room specs (TRUE
   aborts the pass); SECT_FLYING falls (move down + look, else real_room(5)
   DT + abort). Damage rides the shared fight.c machinery: modifier funnel,
   position update, the M-103/M-96 `skill_message` blocks from
   lib/misc/messages, the stun/incap/death position bytes, and `HandleDeath`
   (corpse wording via the existing `case 103` mapping). Seed 8 of the
   scenario drowns the peer live ("You sure are DROWNING!", incapacitated,
   prompt parity at negative HP).

   Deliberately not ported (pre-existing gaps, unreachable or invisible in
   the current corpus): DG room scripts (RS_ONPULSE), `flow_room` (no room in
   the stock world carries a ROOM_FLOW_* flag, so C's gated `number(0,1)`
   never draws), `loud_mobs`, the CON<=0 croak, damage()'s dismount block,
   and the jail/neutral-room redirect arms of self-damage. Room-activity
   iterates players (by name) then mobs (by VNum) — a deterministic
   approximation of C's prepend-ordered character_list; multi-occupant water
   rooms could interleave differently (not oracle-observable today).

## The vehicle (spec-proc-whirlpool)

Fixtures: `empty-players`, `quiet-mobs`, `spawn-mob 12200 999 8161 80`,
`clear-mob-flag 12200 RANDZON`, `replace-room-exits 8161 none` (12200 has no
SENTINEL flag; stripping 8161's exits pins it against the wander draw at
every seed — without this the vehicle is seed-fragile). The peer walks
`north` from the hometown infirmary 8162 into 8161 **post-settle**; the mob
must NOT sit in 8162 itself because Go's birth is synchronous while C's is
pulse-driven — a hometown-room mob makes the first pull land one PULSE_MOBILE
earlier on Go and desyncs everything. Pulse ledger: actor settle → pulse 40,
peer settle → pulse 80 (NOHASSLE God alone with the mob: the gate proof),
probe pump → pulse 120 (the pull + room_activity drowning).

## Debugging lessons this round (beyond the pickup note's)

- **`git stash` reverted the tracked scenario file** mid-round: every harness
  run after the stash silently tested the OLD committed vehicle (mob in 8105,
  zone 183) and produced a consistent-but-meaningless "peer in the Street of
  Wizardry" transcript that burned hours. When stashing code, check whether
  the scenario file is tracked and got caught too.
- **Dirty oracle player files poison later standalone runs**: a re-created
  name hits the password prompt and the whole dance desyncs ("Bad PW",
  relogin-disconnect in the C log). Copy a fresh lib and truncate
  `etc/players` for every standalone oracle boot.
- The C log lines "entering game with no equipment" (objsave.c Crash_load)
  fire for every new char — they do not mean do_start ran.
- The harness's `--show-oracle` "no divergence" verdict is only meaningful
  with the intended block inspected; the seed-8 prompt divergence was the
  only red among five seeds and would have been missed by spot-checking dry
  destinations only.
- `dp-oracle-diff` temp dirs are deleted on exit; a temporary
  `ORACLE_DIFF_KEEP_DATA`-style copy hook (since reverted) was what exposed
  the stash-reverted fixture lines. Do not commit debug hooks.

## Second wave: the birth transition and the e2e world

The first corpus pass went red on `character-creation` — the port made the
pulse-time `start_room` dispatch live, so `completeCharCreation`'s
frozen-clock *synchronous* birth compensation (full-length message +
hometown observation at creation) became a duplicate. The compensation was
removed: creation now leaves the new mortal in the Burning Hut (8099) and the
first PULSE_MOBILE delivers the (oracle-observed, libc-UB-truncated) birth
message plus the hometown relocation — exactly C. The `start_room` port's
forced `PRF_AUTOEXIT` off was also dropped: C's fresh mortal renders the
exits line. Two more prompt-frame details fell out of the same scenario: the
oracle's `~dpclock` line is input on the pumping descriptor, so C clears
`has_prompt` in the input branch and that session's next flush carries
`process_output`'s interruption CRLF — `Manager.PumpPulsesFrom` threads the
pumping session through and the first player-bound output after it claims the
prefix (idle other sessions flush without it, matching per-descriptor state).

The birth change rippled into `tests/e2e` (telnet + websocket smoke): the
walk helpers now trigger the birth with a command from the Burning Hut (C's
command-time `special()` fires it instantly instead of waiting up to
PULSE_MOBILE), and the engage loops were hardened — C renders room people
*after* the `[ Exits: ]` line, so the capture runs through the vitals prompt;
the presence parser learned the stock zone-80 phrases (warg, mercenary,
janitor, petitioner, priestess, prostitute); retries are paced; and a boot
whose Temple Square never shows an NPC skips as world state instead of
failing as a pipeline break.

## Corpus regression

Full re-run of all scenarios in `cmd/dp-oracle-diff/scenarios/` against the
branch (sequential, 240s timeout each, timeout-kills and boot-crash EOFs
classified as infra and retried once, not diffs) — required because the
prompt-frame change touches every command's transcript tail and room_activity
runs on every pulse.

Every red was re-run against pristine main @ b5641d9b7 before acceptance:

- `accuse-noarg-depth` — red on main, identical shape; the manifested
  `blocked` row already documents the C infobar binary-bytes anomaly.
- `force-mob`, `medit-entry-depth`, `medit-session-depth` — red on main,
  identical shape; pre-existing regressions from other rounds (the medit pair
  sits in the OLC/sedit family that is a fenced blocked cluster). None
  introduced by this branch.

- FINAL TALLY (2026-09-03 20:53 EDT): **855/866 pass, 11 red, 0 infra**
  (all four infra rows — equipment-glance, pant, recall, review — passed
  their single retry). All 11 reds reproduce identically on pristine main:
  accuse-noarg-depth, force-mob, medit-entry/session, redit-entry/session,
  sedit-entry/session, spec-proc-cityguard, spec-proc-cityguard-breed,
  spec-proc-dragon-breath-combat. Zero reds introduced by this branch;
  855 green includes every scenario the prompt-frame and room_activity
  changes could touch.

## Gates

`make fidelity-depth` exit 0; `go build ./...`, `go vet ./...`,
`go test ./...` green; `golangci-lint run ./...` 0 issues; `gofumpt -l .`
clean.
