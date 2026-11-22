# Strict L1 recount — 2026-11-22

## Method

Strict recount of all uncovered measurable C segments on the retained nightly
combined census dump from `2026-11-22` on main `f8c4ad638cd97edddc5e19d64d178b442655f19f`
(`/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/nightly-20261122`),
verdict `CLEAN` (7,067 pairs across full + claims, 2,248 deduplicated,
`full=CLEAN`, `claims=CLEAN`, `draw_divergent=0` on both corpora, elapsed
6,056.7s). No census was started; the retained dump was read only.

Current `origin/main` at recount time is `a29f11723` (two commits ahead of
`f8c4ad638`: PR #3010, `pkg/game` test-audit batch 17 docs/tests only; zero
changes to C, Go production code, oracle scenarios, or depth manifests).

Follows the 2026-11-19 / 2026-11-20 strict recount precedence
(`docs/research/l1-strict-2026-11-19.md`,
`docs/research/l1-strict-recount-2026-11-20.md`) and Zach's 2026-11-16 strict
buckets, joining:

1. `docs/research/l1-strict-2026-11-19.tsv` (617 uncovered baseline),
2. Current depth manifests (`docs/fidelity/depth/*.tsv`, including #2861's
   manifest exclusions, and `docs/fidelity/depth/surface/shops-behavior.tsv`),
3. All five L1 batch / settlement TSVs in chronological order:
   - `docs/research/l1-strict-batch2-2026-11-20.tsv` (#2891),
   - `docs/research/l1-strict-batch3-2026-11-20.tsv` (#2914),
   - `docs/research/l1-final-2026-11-20.tsv` (#2921, PF-148),
   - `docs/research/l1-settlement-2026-11-21.tsv` (PF-159, DP-1618/DP-1620),
   - `docs/research/l1-final-2026-11-21.tsv` (#2979, last 12 L1 dice rows).

In addition, because `make census-coverage` scans only the top-level seed-1
`census-dump/`, every segment classified `settled-witnessed` was independently
verified directly against the multi-seed `seed*/initial-dump/` C transcripts
retained inside the same `nightly-20261122` combined run.

## Coverage command and output

Command run from a clean worktree at `origin/main` (`a29f11723`):

```bash
make census-coverage CENSUS_DUMP=/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/nightly-20261122/census-dump
```

Output:

```text
go run ./cmd/dp-census-coverage --dump "/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/nightly-20261122/census-dump"
census coverage: 2456 scenarios, 39434 blocks in the dump
c sites:    3622 of 5454 covered (66.4%)
c segments: 4328 of 4902 covered (88.3%), 1336 unverifiable sites
least covered C files (of 60):
     0.0%     0/3    segments  src/spec_assign.c
     0.0%     0/2    segments  src/poof.c
     0.0%     0/0    segments  src/alias.c
     0.0%     0/0    segments  src/luaedit.c
     0.0%     0/0    segments  src/version.c
    18.5%    10/54   segments  src/objsave.c
    19.0%     4/21   segments  src/db.c
    20.0%     4/20   segments  src/scripts.c
    20.0%     1/5    segments  src/ident.c
    54.3%   113/208  segments  src/spec_procs.c
    61.1%    22/36   segments  src/comm.c
    73.7%    14/19   segments  src/whod.c
    76.5%    13/17   segments  src/mail.c
    77.8%    42/54   segments  src/shop.c
    78.1%    25/32   segments  src/boards.c
wrote coverage.tsv and coverage-summary.txt
```

Output archived at `/home/zach/Archives/darkpawns/notes/coverage-2026-11-22/`
(`coverage.tsv` SHA-256
`eb1cddbaf6c0ab3fe39c317b16ee6dbd3dd1bf000ac2a6b1ec45d5e45ff392bc`).

**Reach summary:**

- **C sites (seed-1 dump):** 3,622 of 5,454 covered (**66.4%**).
- **C segments (seed-1 dump):** 4,328 of 4,902 measurable covered (**88.3%**),
  574 uncovered, 1,336 unverifiable short sites.
- **C segments (full combined run, seed-1 + verified multi-seed witnesses):**
  4,328 + 33 = **4,361 of 4,902** measurable segments witnessed (**89.0%**)
  inside `nightly-20261122` alone; the remaining 541 segments are settled as
  307 `settled-excluded`, 226 `settled-unreachable`, and 8 `settled-unit`.

## Bucket table (all 574 seed-1 uncovered segments)

| Bucket | 2026-11-20 | 2026-11-22 | Delta |
|---|---:|---:|---:|
| `settled-excluded` | 304 | 307 | +3 |
| `settled-unreachable` | 225 | 226 | +1 |
| `settled-witnessed` | 22 | 33 | +11 |
| `settled-unit` | 8 | 8 | 0 |
| `cited-not-witnessed` | 11 | 0 | -11 |
| `wide-cite-only` | 5 | 0 | -5 |
| `uncited` | 0 | 0 | 0 |
| `unit-unqualified` | 1 | 0 | -1 |
| **Total uncovered (seed-1)** | **576** | **574** | **-2** |
| **Open L1 segments** | **17** | **0** | **-17** |

## Changes since 2026-11-20

- **New uncovered segments (covered in 11-20 → uncovered in 11-22): 0.**
- **Flipped uncovered → covered in seed-1 dump (2):**
  1. `src/act.wizard.c:1818#1` (`Current machine time:`): was
     `cited-not-witnessed`; now `covered=yes` via `date-depth` and
     `god-system-residuals` after DP-1620 removed `volatileLine` from
     `internal/oraclediff/normalize.go` (PF-158, `l1-settlement-2026-11-21.tsv`).
  2. `src/mobprog.c:577#1` (`tries to escape from an invisible box only he can see.`):
     was `settled-witnessed` via branch evidence in `l1-final-2026-11-20.tsv`;
     now `covered=yes` directly in seed-1 via `mob-sound-mime-box-depth`.
- **Moved from open to settled-excluded (3, `l1-settlement-2026-11-21.tsv`, PF-159):**
  - `src/constants.c:712#1` (`Ident conning`): `wide-cite-only` → `settled-excluded`
    (Zach's 2026-11-20 DP-1605 ident ruling).
  - `src/constants.c:714#1` (`Ident reading`): `wide-cite-only` → `settled-excluded`
    (Zach's 2026-11-20 DP-1605 ident ruling).
  - `src/constants.c:691#1` (`Disconnecting`): `unit-unqualified` → `settled-excluded`
    (Zach's 2026-11-21 DP-1618 option (a) ruling).
- **Moved from open to settled-unreachable (1, `l1-settlement-2026-11-21.tsv`, PF-159):**
  - `src/constants.c:716#1` (`Asking name`): `wide-cite-only` → `settled-unreachable`
    (provably unreachable under `ident == NO`: `src/config.c:283`, `src/comm.c:563-631`,
    `src/ident.c:63,222-242`; `ident == YES` exits covered by DP-1605).
- **Moved from open to settled-witnessed (12, `l1-final-2026-11-21.tsv`, #2979):**
  - `src/act.offensive.c:558#1` (`You fail the rescue!`): `cited-not-witnessed` → `settled-witnessed`.
  - `src/fight.c:1179#1,#2,#3` (`I killed`, `and looted`, `stinkin' corpse!`):
    `cited-not-witnessed` → `settled-witnessed` (note: `l1-final-2026-11-21.tsv`
    indexed `#1` only; `#2` and `#3` are sibling segments of the same C string
    on `src/fight.c:1179`, all three present in
    `nightly-20261122/seed13/initial-dump/attitude-loot-quips-depth.txt`).
  - `src/fight.c:1183#1` (`was tough, but had good eq...`): `cited-not-witnessed` → `settled-witnessed`.
  - `src/fight.c:1187#1` (`was easy xp.`): `cited-not-witnessed` → `settled-witnessed`.
  - `src/fight.c:1196#1` (`Now you will see that evil will always triumph, because good is dumb.`):
    `cited-not-witnessed` → `settled-witnessed`.
  - `src/fight.c:1209#1,#2` (`Oh, did that hurt,`, `? *innocent stare*`):
    `cited-not-witnessed` → `settled-witnessed` (note: `l1-final-2026-11-21.tsv`
    indexed `#1` only; `#2` is the sibling segment on `src/fight.c:1209`, both
    present in `nightly-20261122/seed12/initial-dump/attitude-loot-quips-depth.txt`).
  - `src/fight.c:1553#1` (`You land on your feet!`): `cited-not-witnessed` → `settled-witnessed`.
  - `src/fight.c:1591#1` (`screams in pain!`): `wide-cite-only` → `settled-witnessed`.
  - `src/mobact.c:252#1` (`'Come to destroy my kin? Die!', exclaims`): `wide-cite-only` → `settled-witnessed`.

## Multi-seed witness verification in `nightly-20261122`

Every single one of the **33 `settled-witnessed`** segments (the 21 carried
from batches 2–3 / finish-L1 plus the 12 from #2979) was verified directly in
`nightly-20261122/seed*/initial-dump/` (**33/33 present, 0 missing**):

- `src/act.offensive.c:558#1` → `seed3/initial-dump/combat-entry-rescue-fail-depth.txt`
- `src/fight.c:1179#1,#2,#3` → `seed13/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1183#1` → `seed16/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1187#1` → `seed22/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1196#1` → `seed4/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1209#1,#2` → `seed12/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1223#1,#2` → `seed11/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1238#1` → `seed14/initial-dump/attitude-loot-quips-depth.txt`
- `src/fight.c:1553#1` → `seed23/initial-dump/combat-knockoff-land-depth.txt`
- `src/fight.c:1591#1` → `seed7/initial-dump/spellmatrix-staff-olc-07-depth.txt`
- `src/limits.c:621#1,623#1,629#1,631#1` → `seed2/initial-dump/corpse-decay-depth.txt`
- `src/magic.c:705#1` (disintegration, line 4) and `src/magic.c:717#1` (ray of
  disruption, line 276) → `seed7/initial-dump/magic-backfire-recoil-depth.txt`
- `src/magic.c:813#1` → `seed9/initial-dump/magic-backfire-recoil-depth.txt` (also seeds 12, 14)
- `src/magic.c:981#1,982#1` → `seed2/initial-dump/magic-affect-reagent-lines-depth.txt` (also seeds 3, 5)
- `src/mobact.c:252#1` → `seed3/initial-dump/mobact-race-hate-kin-depth.txt`
- `src/mobprog.c:530#1` → `seed2/initial-dump/mob-sound-petitioner-random.txt` (also seeds 3, 5)
- `src/mobprog.c:537#1` → `seed2/initial-dump/mob-sound-carpenter-random.txt` (also seeds 3, 5)
- `src/mobprog.c:545#1` → `seed2/initial-dump/mob-sound-town-crier-random.txt` (also seeds 3, 5)
- `src/mobprog.c:561#1` → `seed2/initial-dump/mob-sound-beggar-random.txt` (also seeds 3, 5)
- `src/mobprog.c:573#1` → `seed2/initial-dump/mob-sound-minstrel-random.txt` (also seeds 3, 5)
- `src/new_cmds.c:2088#1` → `seed2/initial-dump/compare-band-depth.txt`
- `src/new_cmds.c:2678#1` → `seed3/initial-dump/full-moon-depth.txt`
- `src/spells.c:421#1` → `seed3/initial-dump/charm-charmed-caster-depth.txt`
- `src/weather.c:223#1` → `seed2/initial-dump/weather-lightning-stop-depth.txt`

## Open rows

**None (0 open L1 rows).**

Every measurable C segment in the 2026-11-22 nightly combined dump is either
directly covered in seed-1 (4,328), verified witnessed in a retained multi-seed
dump of the same combined run with an existing revert proof (33), settled as an
approved exclusion (307), provably unreachable with C citation (226), or
settled under the unit exception with a C-cited test and revert proof (8:
`src/spells.c:191#1` and the 7 `src/dream.c` rows).

Minor bookkeeping note (does not reopen any row): `docs/research/l1-final-2026-11-21.tsv`
lists `#1` only for `src/fight.c:1179` and `src/fight.c:1209`, leaving the
same-line sibling segments `src/fight.c:1179#2` (`and looted`),
`src/fight.c:1179#3` (`stinkin' corpse!`), and `src/fight.c:1209#2`
(`? *innocent stare*`) unindexed by segment ID in that TSV; all three are
printed by the exact same witnessed lines in `nightly-20261122/seed13` and
`seed12`.
