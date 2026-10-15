# 2026-10-15: The finish line, measured against C instead of against our cases

## Scope

- **Tracks:** legacy-port fidelity (completion measurement, "what does done mean")
- **Measured:** main `fd327ae1a` (= production as of 2026-10-15 10:07 ET)
- **Primary artifacts:**
  - `docs/fidelity/depth/*.tsv` (5,659 case rows)
  - `docs/fidelity/depth/surface-inventory.tsv` (73 surface families)
  - the draw-parity corpus run `~/Archives/darkpawns/oracle-runs/2026-10-15/finish-baseline-draw-parity/`
  - the measurement script below
- **Status:** verified (counts reproducible with the script on that commit).

## What was observed

Zach: "the last time I checked we were at 98.8% behavior cases ... that last .2% is feeling like 10%." Both statements are true, because they measure different denominators.

**By authored case** (the depth manifests, excluding the surface inventory): 5,659 rows, of which 71 are `blocked` (1.3%).

| Status | Rows | Share |
|---|---|---|
| oracle-green-multiseed | 2,289 | 40.4% |
| unit-green | 1,461 | 25.8% |
| oracle-green | 1,098 | 19.4% |
| delegated | 648 | 11.5% |
| blocked | 71 | 1.3% |
| excluded (+ with C reason) | 68 | 1.2% |
| proven-already | 12 | 0.2% |
| divergent-approved | 12 | 0.2% |

Note: that table counts the 60 blocked surface-inventory rows among the 71, as the raw TSV scan does. Excluding them, **11** case rows are blocked.

**By C surface** (`surface-inventory.tsv`, where one row is a whole family of C code and `units` counts its call sites or vectors): **4,929 units, of which 4,031 (82%) are `blocked`.**

| Surface class | Blocked units | Largest families |
|---|---|---|
| activity-call-sites (`act`/`send_to_char`) | 2,060 | `act.wizard.c` 352, `new_cmds.c` 204, `spec_procs.c` 179, `clan.c` 165, `spec_procs2.c` 129, `spells.c` 102 |
| spell-matrix (5 cast routes × 299 spells) | 1,495 | all five routes |
| fight-skill output | 463 | `new_cmds` 204, `fight.c` 97, message corpus 82, `new_cmds2` 80 |
| lifecycle | 10 | death, rent, save/load, pulse order, descriptors |
| shops | 3 | core, specials, rent-adjacent |

**Draw parity** (`--draw-parity` over the 1,171-scenario corpus): DRAW_RESULT.

## Interpretation

1. **The case percentage measures what we wrote down.** A case exists because someone authored it, so the manifest can be 98.7% green while most of C has never been asked about. The surface inventory is the denominator that C defines. It was added during the DP-1426-era audits precisely because gaps were hiding outside the authored cases.
2. **"Blocked" in the surface inventory mostly means unmapped, not broken.** Every blocked family's notes date from the terminal audit of **2026-09-04**, before DP-1371 added thousands of proofs. When `src/act.comm.c` (99 sites) was mapped, it flipped to `proven-already` without new code. Expect most of the 82% to move by mapping alone, and the residue to be real gaps.
3. **Byte-green is not done.** DRAW rows are scenarios whose transcripts match on a drifted RNG stream (PF-067, PF-068). Done requires draw parity too (R3).
4. **What remains is mostly structural** (PF-069): the residual defects are about sequencing and delivery, not text.

## Baseline table (re-measure weekly)

| Date | Commit | Case rows blocked | Surface units blocked | DRAW rows |
|---|---|---|---|---|
| 2026-10-15 | `fd327ae1a` | 11 (+60 surface rows) | 4,031 / 4,929 | DRAW_COUNT |

## Measurement script

Run from the repository root:

```python
import csv, glob, collections
st = collections.Counter()
for f in glob.glob('docs/fidelity/depth/*.tsv'):
    with open(f, newline='') as fh:
        r = csv.reader(fh, delimiter='\t'); hdr = next(r, None)
        if not hdr or 'status' not in hdr: continue
        i = hdr.index('status')
        for row in r:
            if len(row) > i and row and not row[0].startswith('#'): st[row[i]] += 1
print(st)
with open('docs/fidelity/depth/surface-inventory.tsv', newline='') as fh:
    r = csv.DictReader(fh, delimiter='\t')
    rows = list(r)
units = lambda rs: sum(int(x['units']) for x in rs if x['units'].isdigit())
print('surface units', units(rows), 'blocked', units([x for x in rows if x['status'].startswith('blocked')]))
```

DRAW rows: `scripts/census.sh start --draw-parity --name <run>`, then the `oracle-draw-parity:` line in `summary.txt`.

## Caveats

- `units` are the audit's counts of call sites or vectors, not weighted by player exposure. A wizard-only `act()` counts the same as a combat message.
- The draw-parity run is seed 1 only. DP-1427 v1 has no multi-seed mode.
- "Done" here is still the rulebook's definition (R1–R5). This note changes the measurement, not the target.
