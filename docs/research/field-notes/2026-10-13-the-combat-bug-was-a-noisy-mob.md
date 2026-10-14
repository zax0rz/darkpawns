# 2026-10-13: The combat bug was a noisy mob

## Scope

- **Tracks:** legacy-port fidelity (RNG parity, coincidence greens, diagnosis method)
- **Window:** diagnosed 2026-10-13
- **Primary artifacts:** `~/Archives/darkpawns/oracle-runs/2026-10-13/teleport-combat-diagnosis/` (paired C/Go draw traces for seeds 1 and 2, the seed-2 traces with the probe applied, `drawdiff.py`, `loud-mobs-probe.diff`, `c-oracle-draw-trace-instrumentation.diff`); `src/comm.c:743`; `src/spec_procs2.c:1532-1536`; `lib/world/mob/144.mob` (#14405); `pkg/game/room_activity.go`; depth row `mob.teleport-victim-combat-transcript`; Linear DP-1426
- **Status:** verified. The cause is shown by a draw-for-draw trace, and a probe adding only the missing draw restores full parity.

## What was observed

Since 2026-08-30, `mob.teleport-victim-combat-transcript` was blocked as a "pre-existing shared combat attack/message/RNG transcript" divergence. Two honest vehicles reached Gil-Glash's `teleport_victim` special, but seeds 2, 3, 5 and 8 diverged on the **player's opening hit**: C landed it, Go missed. The handoff recorded it as a combat problem and declined to "fix unrelated combat in this slice". That was the right call with what was known.

On 2026-10-13 the vehicle was still red on today's `-O0` reference, so it wasn't the `-O2` artifact that explained `wiznet @` the same day. A paired draw trace settled it. The C oracle copy was instrumented to log every `number`/`dice`/`uniform` call in the same format as Go's existing `DP_DRAW_LOG` seam:

- The streams agree for **4,096 draws**: boot, zone resets, creation, warmup.
- At draw 4,097, **C draws `number(0,20)` and Go doesn't.** Every later roll shifts by one, and the opening hit flips.
- That draw is the first statement of `loud_mobs` (`src/spec_procs2.c:1536`): `if (!GET_NOISE(ch) || ch->in_room == NOWHERE || (number(0,20)>0)) return;`. `room_activity` calls it for every NPC on every mobile pulse (`src/comm.c:743`). Gil-Glash (#14405) has a noise string, "Waves of evil wash over you.", and a mobile pulse during the scenario's **setup** makes the extra draw. The player's `hit` is only the first roll whose outcome shows the drift. (This sentence originally said the pulse fell inside the `hit` step; corrected 2026-10-14, see below.)
- The Go port lists `loud_mobs` under *Deliberately NOT ported* in `room_activity.go`. It had no row and no issue, the same pattern as the CON death found the same evening (DP-1426).

**Proof.** A throwaway probe adding only `dprng.Number(0,20)` for each noisy mob in a room turns all five seeds green. The paired trace of seed 2 then matches for all 4,085 draws.

## Two findings that matter more than the fix

1. **The combat diagnosis was wrong for six weeks, honestly.** The symptom was in combat (a hit became a miss), but the cause was in ambient world behaviour that doesn't print anything here, because the room's exits were replaced with none. RNG parity makes the whole game one shared stream. A missing draw anywhere shows up wherever the next consequential roll happens. Byte diffs locate the symptom; only a draw trace locates the cause. (Same lesson as DP-1212, 2026-07-24.)
2. **Seed 1 was a coincidence green, and the rows built on it claimed too much.** Seed 1 carries the same drift, but C's to-hit roll (9) and Go's (20) both hit, and a later path-dependent draw happens to put the streams back in step. Several `mob.teleport-victim-*` rows cite the seed-1 vehicle. Their assertions held, but on a shifted stream. **A byte-level census can't detect a scenario whose stream drifts but whose bytes land the same.** The only check that can is draw-stream parity. That suggests a harness capability: compare the draw sequences, not just the transcripts. It's harness work, which is Zach's stop tier, so it's recorded as a proposal.

## Open

- The real port of `loud_mobs` (the gate, plus noise into adjacent rooms) belongs to DP-1426 and its PR unblocks the row. With two or more noisy mobs, parity also depends on C's `character_list` walk order (DP-1364).
- **Proposal for Zach:** a draw-parity census mode. Run each scenario with both draw logs, and report the first draw mismatch as a finding even when the bytes match. It would turn coincidence greens into reds across the whole corpus.

## Correction (2026-10-14)

The first version of this note placed the extra `loud_mobs` draw "inside the `hit` step". That was an inference from where the transcripts first differed. The draw-parity harness built the next day (DP-1427) records each side's draw count after setup, warmup and every probe step. Run against this scenario with the `loud_mobs` draw removed, it reports `MISMATCH c=Number(0,20)` with `c-step=setup`. The extra draw happens during setup, and the hit is just the first consequential roll afterwards. The rest of the finding stands. The correction illustrates the note's own first point: a byte diff locates the symptom, and only the draw stream locates the cause.
