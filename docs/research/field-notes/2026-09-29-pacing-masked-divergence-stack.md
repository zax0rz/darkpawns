# 2026-09-29 — A pacing-dependent oracle hid a stack of three Go divergences

## Scope

- **Tracks:** legacy-port fidelity; test-oracle validity; multi-agent software engineering
- **Window:** 2026-09-29
- **Primary artifacts:** DP-1366 (census speed-up), DP-1368 (C-seam wait-state fix), zax0rz/darkpawns#1701, parked branch `glm/census-speed-concurrency` (`e9dd1b833`), the `sol/wait-pulses` branches (main-repo `befcb2a57`, oracle `d2ace874`), stop reports under `docs/fidelity/depth/handoff/2026-09-29-dp-1368-*` on that branch, and run evidence in `~/Archives/darkpawns/oracle-runs/2026-09-29/dp-1366-*`, `dp-1368-*`
- **Status:** artifact-grounded field note. It records engineering evidence, not a paper conclusion. The Go fixes were not merged when this was written.

## A speed-up that should have been neutral was not

Running the C and Go engines concurrently, and draining peer connections in parallel, cut a full census from 920 s to 495 s. These changes should leave every scenario's result the same. Instead eight combat and wait-state scenarios that pass on main started to fail. Isolating each change: concurrent engines with sequential drains failed four; sequential engines with parallel drains still failed; both together failed all eight; main's harness passed all eight twice.

The cause was in the oracle, not the harness. Under `DP_CLOCK`, the C seam still decremented a player's wait state once per real 100 ms game-loop pass (`~/darkpawns-c-oracle src/comm.c:613`), while heartbeats ran only when pumped. A command sent during a combat `WAIT_STATE` therefore ran whenever enough wall time had passed. Main's slower step pacing let the wait expire before the next send. The eight greens were coincidence greens (RULEBOOK R5h): pacing, not fidelity, made them pass. This is the same blind spot as DP-1202.

## Making the oracle deterministic exposed what the pacing had masked

With the seam changed so a player's wait counts pumped pulses (real C: one pass is one pulse; Go already did this), the census exposed Go divergences that had been invisible, each stacked on the next:

1. **Wait length.** C's `do_hit` sets `WAIT_STATE(ch, PULSE_VIOLENCE + 2)` = 22 pulses (`src/act.offensive.c:127`); Go stored 3 rounds = 60 pulses. The same `+N`-as-whole-rounds shape recurred in kick, circle, strike, and retreat's failure arm (R5c class).
2. **Stand gate.** C's `damage()` stands a victim via `set_fighting` only when `!FIGHTING(victim)` (`src/fight.c:1443`); Go stood every struck defender, silently suppressing C's later "You drag yourself to your feet."
3. **An "intent" fix hiding under the stand gate.** Once an already-fighting victim correctly stayed sitting, seed 8 of `bash-failure-depth` exposed Go's position damage multiplier: C computes `dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3` in integer arithmetic (`src/fight.c:1855`), so sitting and resting multiply by 1; Go used float math, citing the C comment's "x1.33" as "developer intent" (commit `0210c7ff9`, DP-515, 2026-05-27). That is an R4 violation, a deliberate plausible-looking improvement over the executed C.

Each layer masked the next. With the old oracle, the pacing hid (1) and (2). With (2) wrong, the victim stood before the damage multiplier read its position, so (3) never had a sitting victim to multiply. A control on seed 8 shows the interaction: old oracle + pre-fix Go passes, the deterministic oracle + pre-fix Go fails on the stand line, the deterministic oracle + fixed stand gate fails on the damage label.

## Methodological observations

- **A harness change that should be neutral is itself a probe.** Changing only the pacing, and requiring identical results, found an oracle nondeterminism that a green census had hidden since the seam was written.
- **Greens can stack.** A correct fix at one layer can turn a passing scenario red because it removes the error that was masking another one. The right response was to trace, not to revert the correct fix (the agent's stop report says this explicitly).
- **"Restore developer intent" is the tell.** The float multiplier sits in the same class as the `yank` paraphrase (PF-002): plausible, well-commented, and wrong by the port's own law.
- **Multiple seeds mattered.** Seed 1's full census was clean; the multiplier surfaced only on seed 8 of an existing five-seed proof.
- **Agent conduct.** The implementing agent stopped three times at brief-defined conditions rather than tuning pump counts or pinning divergences, and in one case discarded its own test vehicle because it passed on the old binary by coincidence.
