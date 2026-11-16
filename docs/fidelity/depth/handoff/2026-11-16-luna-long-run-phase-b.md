# Luna long-run port completion — Phase B handoff

## Result

The current L1 recount reaches zero uncited rows. The latest full census dump
contains 2,184 scenarios and 36,749 blocks. `dp-census-coverage` found 5,454 C
sites (3,290 covered) and 4,902 output segments (3,949 covered; 1,336
unverifiable, excluded by the frozen method). Of the 953 uncovered segments,
951 already had depth-manifest line citations. The remaining two were both
`buffered:sprintf` segments at `src/comm.c:783` in `object_activity`.

The initial valid witness showed a real R1 divergence: after `get helm`, `put
helm basket`, `get helm basket`, and the next mobile pulse, C sent
`[ SYSERR: Object 'a basket of apples' weight incorrect, location 'in room' ]`
to the complete-syslog immortal; Go sent no line. C's path is
`do_get` → `perform_put`/`perform_get_from_container` → `obj_to_obj` /
`obj_from_obj` → `object_activity` on the next `PULSE_MOBILE`.

The final fix records C's sub-one weight floor until the pulse diagnostic is
emitted, without adding that floor to Go's derived carry weight. A separate
carried-rack witness confirmed that both engines accept the second 5-pound
sword at the exact 10-pound limit and emit the same diagnostic with location
`carried`. The code fix is commit `dfd1a0771` on
`codex/luna-dp1535-weightfix`.

The final 315-scenario get/put selector passed with draw parity:

- Run: `~/Archives/darkpawns/oracle-runs/2026-11-16/luna-l1-weight-command-parity-r3`
- Verdict: 315/315 passed, 0 failed, 0 draw divergences; the wrapper marked
  one existing scenario (`money-exdesc-depth`) with 20 empty steps. Both new
  L1 witness scenarios had zero hollow steps.
- The frozen run's five changed Go files match fix commit `dfd1a0771`, and its
  two scenario files match the proof branch byte for byte.

The final recount scanned 462 top-level manifests and 20,452 citation ranges;
all 953 currently uncovered segments are cited, leaving 0 L1 rows. The new
rows in `mudlog.tsv` preserve separate in-room and carried location witnesses
for the same C site. The two new scenario files and
`docs/research/l1-recount-2026-11-16.tsv` are on the proof branch.

## Frozen baseline limitation

The frozen 2026-11-11 README records 508 cited segments and 739 remaining on
its specified manifest snapshot (`122615de3`). Reapplying the same
inclusive-line method to the old census dump with the workspace's current
top-level manifests gives 934 cited and 313 remaining. The historical
subtraction cannot be reconstructed from the current manifest set; the frozen
README was left unchanged. The 1,336 unverifiable segments remain outside L1,
as the frozen method requires.

## Phase boundary counts

- Sites settled: 1 C site / 2 output segments; current L1 residual 0.
- Divergences found: 2. The original C-only diagnostic was fixed. A first
  weight-override implementation caused a carry-limit refusal; that candidate
  was discarded before commit, and the final marker implementation passed the
  carried-weight witness.
- Phase B PRs opened: 0. There were already four open repository PRs (#2712,
  #2725, #2726, #2727), above the prompt's three-PR cap. No PR was opened over
  that limit. The behavior-fix and proof branches are prepared separately.
