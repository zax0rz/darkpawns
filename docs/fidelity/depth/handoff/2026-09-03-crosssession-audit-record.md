# WHIRLPOOL CLINIC BREAKTHROUGH (2026-09-02/03, ZCode) — read before section 3
The 7-row blocked cluster is CONVERTIBLE: vehicle proven RED on main. Artifacts:
- Working vehicle: /home/zach/dp-whirlpool-vehicle.txt (copy into
  cmd/dp-oracle-diff/scenarios/, rename per convention, add depth-case rows).

## Root cause of the 4-day-old "unmaterialized vehicle" (2026-08-29 handoff was wrong)
1. Room 8105 is the Street of Wizardry (the GOD creation room), NOT where walks end.
   The n/e/s/e walk ends in the Board Room of the Immortals — the 8-29 peer walked
   AWAY from the spawn room every time. Mortal creation room = Temple Infirmary 8162
   (in 80.wld). Nobody was ever co-located with the mob.
2. Mob 12200's act flags 1335373 include bit 20 = MOB_RANDZON: at boot-reset C
   scatters it to a random room of its zone (src/db.c reset_zone M-handler). Fix:
   clear-mob-flag 12200 RANDZON (fixture exists).
3. `vnum mob` in C lists PROTOTYPES ONLY — it can never show instances; the 8-29
   round misread it as "no instance present."
4. spawn-mob injection works fine on current main (stableboy's stablehand answers
   tells live; parse+renum accept injected M lines). No harness regression. NOTE:
   stableboy's green proofs are real (tells are global) but none prove room placement.

## Proven RED divergences (fresh main, DP_SEED=1)
Vehicle: spawn-mob 12200 999 8162 80 + clear-mob-flag 12200 RANDZON + quiet-mobs +
empty-players; mortal peer created in 8162, no walk; probe:peer = ~dpclock pulse 40 + look.
C: pull fires ON THE PULSE ("A ravaging whirlpool sucks you under!" + "You finally
surface, sputtering...\r\n\r\n" + look_at_room of a 4600-4699 room). C ALSO dispatches
at COMMAND time (any command in-room triggers it). Post-relocation C applies water-
sector effects: "You sure are DROWNING!" + stun ("You're stunned, but will probably
regain consciousness again."), and stun blocks look ("All you can do right now is
think about the stars!").
Go fixes needed (pkg/game/spec_procs2.go specWhirlpool + its dispatch):
a. TIMING: Go's spec dispatch runs during setup/settle (pulled the peer before the
   probe; messages lost in the drain). C's DP_CLOCK freeze means mobile_activity
   does not advance until ~dpclock. Freeze Go's mobact spec dispatch identically.
b. DRAW PARITY: with timing fixed, destination draws must consume identical draws
   per seed (Go dprng.Number(4600,4699) + rejection vs C number(real_room(4600),
   real_room(4699)) rejecting PRIVATE/GODROOM/DEATH/NOMOB). Also remove Go's
   invented 100-iteration cap — C's loop is unbounded (R4).
c. MISSING SECTOR EFFECTS: Go has no drowning/stun for water-sector rooms. Port
   from C (R4 — no invented messages); may deserve its own manifest rows.
d. NOHASSLE: Go's specWhirlpool lacks C's PRF_NOHASSLE victim gate — a God-in-room
   variant of the vehicle will prove it RED (converts row nohassle-gate).
After fixes: multi-seed (1,2,3,5,8) for random-destination rows; convert the six
mob.whirlpool-* rows blocked -> oracle-green(-multiseed).

## Incidental real divergences found while building the vehicle
- Room-content leading space: C renders " A large bulletin board..." (leading space)
  in the 8105-class rooms; Go drops it.
- `vnum mob <name>`: C "  1. [12200] a magical whirlpool" (numbered, indented, no
  header); Go invents "mob matching ... (N found):" header (R4). Check for an
  existing wizard-command manifest row before fixing.

## Debugging lessons (do not repeat)
- probe:peer labels: [x [actor]] = the PEER (probe client), [x [primary]] = the
  actor; empty primary blocks are normal.
- pkill -f matching the running command kills the whole Bash tool call silently;
  use [b]racket patterns in a SEPARATE command.
- The oracle logs local time (EDT), not UTC.
- Scenario files are go:embed'd — rebuild the harness after editing them.

---

# Fidelity re-audit — reconstructed record & pickup
Written 2026-09-02 by ZCode; replaces /tmp/dp-audit-PICKUP.md (lost to reboot).
Audience: Claude (post-hoc review stream), future ZCode/Codex sessions, Zach.

## 1. Mid-flight full regression (2026-09-01) — the headline evidence
- Why: CI cannot run the C oracle (needs the local binary), so "oracle-green" is only
  proven at slice time; later merges could silently regress earlier scenarios with no
  gate catching it. This run closed that blind spot for the a-m corpus.
- Method: detached git worktree pinned at main #1045 (nibble handoff); ALL 634
  scenarios re-run sequentially via cmd/dp-oracle-diff
  (DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle; go at /usr/local/go/bin),
  nice'd alongside the then-live sweep executor.
- Prior stratified sample (2026-08-31, main #946): 15/15 pass, all eras.
- RESULT: **625/634 pass (98.6%). 9 failures, ALL infra-shaped — zero content diffs
  between Go and C anywhere in the corpus.**
- The 9: clan-depth, clan-member-depth, gag-depth, grin-depth, headbutt-depth,
  hop-depth, immlist-depth, imotd-depth, spec-proc-no-get-palm.
- Failure modes: C oracle boot crash ("exited before readiness: exit status 1") or
  EOF mid-probe on either side. Contention evidence: grin/headbutt/hop boot crashes
  clustered inside 22 minutes; headbutt-depth had been slice-proven green on the same
  code the day before; the audit ran concurrently with the sweep executor.
- Reboot casualty note: /tmp/dp-audit-summary.txt and /tmp/dp-audit-failures.log are
  gone. The tally and names above are the authoritative record (from the session that
  ran it). Lost per-failure captures were all infra messages; no diff evidence existed.

## 2. Triage of the 9 — RESOLVED 2026-09-02, all flakes, zero real divergences
Re-run on current main (post-#1231): gag, grin, headbutt, hop, immlist, imotd,
spec-proc-no-get-palm passed immediately; clan-member-depth and clan-depth passed
with a 240s timeout (they are long scenarios; 90s under load was never enough).
Final verdict: 634/634 of the 2026-09-01 corpus green — no content divergence
anywhere. Lessons for any future bulk re-run:
- per-scenario timeout must be generous (240s+) under a loaded box, and the runner
  may be running — expect slowdowns, not failures
- classify non-zero exits properly: harness timeout-kill (exit 124/SIGTERM, empty
  output) is NOT a content diff; only an actual normalized diff is
- the long clan-* scenarios are the known flake-prone pair under load

## 3. Stranded-PR sweep — known process gap, recurring
- Root cause: CI no-fire race. Runner dispatches CI, moves on if not green; PRs that
  go green late are never revisited. Runner handoffs confirm this is by its own
  protocol ("bellow and bird remain claimed by their open feature handoffs after
  their one-time CI retries").
- 2026-09-02 ZCode sweep: merged #1095, #1097, #1130, #1064 (green, direct).
  #1096 (qecho fix) had rotted into merge conflicts after 17 slices of drift ->
  rebased in an isolated worktree (single additive conflict in
  pkg/session/commands.go registry: main's rsay/qsay routing vs qecho's; kept both),
  local gates + oracle scenarios qecho-depth AND send-depth green, CI green,
  merged 2026-09-02T11:30Z.
- Standing task for the review stream: periodically `gh pr list --state open`; merge
  green fidelity PRs (rebase first if conflicted); each idle hour makes a stranded
  fix's eventual rebase costlier.
- Protocol amendment to fold into the NEXT standing goal: before starting each slice,
  re-check own open PRs and merge any that went green.
- Stranded at the time of this note: #1190, #1191, #1192, #1193, #1199
  (beg / bellow / bird / bleed era) — verify green before merging.

## 4. Frontier at last check (2026-09-02 18:28 UTC)
- 3,435 total / 3,334 proven+delegated / 48 blocked / 53 excluded; repo tip ~#1199.
  The standing goal's run: PR ~692 -> ~1199 and counting.
- Queue state: special-procedure inventory EXHAUSTED; interpreter-table sweep
  EXHAUSTED (through z); now consuming the SOCIALS inventory alphabetically
  (bitch, blame, bellow, bird, beg, bleed, ...). Terminal condition = socials
  inventory exhausted -> runner writes a terminal handoff and stops.
- Notable blocked rows: sedit (pending OLC state machine — a subsystem, correctly
  fenced), objmagic.sleep-entry-gates (attempted once via cast-sleep vehicle, not
  repicked per queue rules).
- Caveat that keeps getting re-learned: manifest "actionable completion" is not port
  completion. The real denominator is enumerated by the round in section 5.4.

## 5. Post-terminal sequence (agreed with Zach)
1. Triage the 9 (section 2).
2. Exclusion spot-checks: ~10 excluded rows verified against src/ (start with
   spec-procs.tsv *-unassigned cluster); any exclusion that doesn't hold converts to
   proven or blocked with notes.
   STATUS 2026-09-02 (ZCode): sample verified, all held. mayor, tipster, eviltrade,
   little_boy, ira, couch, elemental_room, enter_circle, elevator appear ONLY in the
   spec_assign.c prototype block (lines ~96-589), never in an ASSIGN* line ->
   genuinely unassigned. snake IS assigned (14103/14127/14415) exactly as its note
   claims; its unreachability rests on the mobact gate, confirmed verbatim:
   mobile_activity skips any mob with FIGHTING(ch) before special dispatch
   (src/mobact.c:68+). Remaining unverified: the 158.mob no-spec-flag pair
   (rescuer-15808, pissedalchemist-15814) and the non-spec exclusions
   (drink.vampire, eat.werewolf-corpse, dns/auto/help no-descriptor/UB rows).
3. Blocked clinic: vehicle attempts for the mob.whirlpool-* 7-row cluster and
   force.npc-command-interpreter; two honest attempts each, else sharpen notes.
   WHIRLPOOL VEHICLE RESEARCH DONE 2026-09-02 — it is feasible:
   - C body: src/spec_procs2.c:244-279. Registered ASSIGNMOB(12200, whirlpool)
     (spec_assign.c:342); mob 12200 act flags 1335373 (odd -> MOB_SPEC bit 0 set,
     so the rescuer-15808 no-dispatch trap does NOT apply).
   - Fires on the autonomous pulse: mobact calls func(ch, ch, 0, "") for SPEC-flagged,
     awake, non-fighting mobs. No command needed.
   - Victim: any PC in the mob's room without PRF_NOHASSLE. Use a MORTAL peer as
     victim (Gods may carry NOHASSLE; mortals don't) — covers whirlpool-autonomous-entry
     (peer present when pulse lands) and whirlpool-state-transition (victim ends in
     a 4600-4699 room).
   - Output per victim: "A ravaging whirlpool sucks you under!" + "You finally
     surface, sputtering...\r\n\r" + look_at_room of destination. NO to-room message;
     old-room peers see nothing. Covers victim-output + destination-look.
   - Destination: do { to_room = number(real_room(4600), real_room(4699)); } while
     (PRIVATE|GODROOM|DEATH|NOMOB). Draw parity is the whole game: redraw count must
     match C per seed -> run multi-seed (1,2,3,5,8); any redraw divergence is a real
     finding. Covers random-destination.
   - nohassle-gate row: needs a NOHASSLE-set victim that is NOT pulled — requires
     pref control on a peer (hassle toggle) or falls back to unit-proof.
   - Vehicle: spawn-mob 12200 into a fixed room (it does not appear in any zon load
     grep), mortal peer present, ~dpclock pulse to advance, capture both sides.
   Remaining clinic research: force.npc-command-interpreter sketch.
4. Enumeration round: inventory the player-visible surface that lives OFF the command
   tables — every act()/send_to_char call-site family, the full spell matrix across
   all cast vectors (cast/potion/scroll/wand/staff), fight/skill message corpus
   breadth, lifecycle systems (death cycle, rent, save/load round-trips, regen,
   weather pulses), shop breadth. Output: a measurable denominator so "port
   completion %" finally means what it says.
   Preview sizing done 2026-09-02: ~2,006 send_to_char + ~992 act() call sites in
   src/*.c. Density leaders: spec_procs{,2,3}.c (237 act sites), new_cmds{,2}.c
   (120), act.item.c (76), act.movement.c (61), fight.c (49), spell_parser.c (47),
   act.offensive.c (45), spells.c + act.other.c (44 each). Note this is an upper
   bound — many sites live in already-manifested handlers; the enumeration round's
   job is bucketing them proven/unproven/excluded.
Goal texts for steps 1-3 exist in the 2026-09-01/02 ZCode conversation.

## 6. Standing rules (unchanged)
- src/ and darkpawns-c-oracle/ are the read-only oracle. Never edit. Cite R1-R5e.
- Flaky-red = stop and report; never retry-into-green (and never declare red on
  contention-only evidence — see section 2).
- CI/deploy/secrets/save-format/website changes are human-only. (#945 CI flake fix
  was Claude at Zach's direction — within scope. feat/web PR #1075 is Zach/Claude's,
  not the loop's.)
- Post-hoc review by another model is part of the charter. This note is for that stream.
[fixture]
empty-players
quiet-mobs
spawn-mob 12200 999 8162 80
clear-mob-flag 12200 RANDZON

[setup:oracle]
Debugactor
Y
oraclepass
oraclepass
N
M
H
W
K
Y
<ENTER>
1

[setup:port]
Debugactor
y
oraclepass
oraclepass
Y
N
M
H
W
K
Y
<ENTER>
1

[setup:oracle:peer]
Debugpeer
Y
oraclepass
oraclepass
N
M
H
W
K
Y
<ENTER>
1

[setup:port:peer]
Debugpeer
y
oraclepass
oraclepass
Y
N
M
H
W
K
Y
<ENTER>
1

[probe:peer]
~dpclock pulse 40
look

## Appendix: provenance
Promoted from session-local notes (/home/zach/dp-audit-PICKUP.md, dp-whirlpool-vehicle.txt)
so the record survives reboots and becomes a citable repo artifact per
docs/research/README.md. Originals retained in ~ until superseded.
