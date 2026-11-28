# C bug decisions inventory — 2026-11

Audit of every known bug in the C original (`src/`) and what the Go port decided
to do about it: reproduce it (Classic fidelity: RULEBOOK R1), fix it on purpose
(security, R1a undefined behaviour, or an approved divergence), patch the
reference oracle to the original 32-bit build's semantics (R1a), or depart
accidentally. **Audit only: no code changes.**

Brief: `docs/briefs/BRIEF-2026-11-28-c-bug-decisions-inventory.md`.

## Sources searched

1. **Go code and test comments** across `pkg/`, `cmd/`, and `internal/`
   (case-insensitive scan for bug/quirk/R1a/UB/typo/overflow/dangling/fallthrough
   terms — 696 non-test, 190 test, and 59 secondary hits reviewed by hand).
2. **Fidelity ledgers and audits:**
   `cmd/dp-oracle-diff/expected_divergences.tsv`,
   `docs/fidelity/world-exceptions.tsv`,
   `docs/fidelity/strings/` (`divergence:approved`),
   `docs/fidelity/audits/2026-11-special-procs/`,
   `docs/fidelity/C-ERRATA.md`,
   `docs/fidelity/depth/*.tsv` (`divergent-approved` and `R1a` rows),
   `docs/briefs/PROPOSAL-2026-11-14-shop-r1a.md`, and
   `docs/research/mutation-loop/DECISIONS.md`.
3. **`docs/fidelity/RULEBOOK.md`** (R1a and amendment log) and
   **`docs/research/EVIDENCE_LEDGER.tsv`**.
4. **Git history:** `git log --all --grep` on `darkpawns` and the reference
   oracle checkout `~/darkpawns-c-oracle` (`dp-oracle-seam`).
5. **Linear:** skipped — Linear tools are not available in this environment.

Every `c_site` citation was grepped/read from `src/` during this session (R5g).

## Scope: C bugs vs. design choices

Per the brief, a C bug is something C's own authors would call wrong (a typo,
an argument mismatch, overflow, UB, a wrong target, an infinite loop, a missing
`break`). Deliberate approved divergences that address **design choices or
modern hardening** rather than bugs in C are excluded from the TSV:

- `entry.security-reserved-names` (DP-1378) — extra reserved names overlay.
- `entry.security-full-password-comparison` (DP-1380) — bcrypt vs. C's 8-char `crypt()`.
- `entry.security-ip-admission` (DP-1387) — per-IP login rate limiting / lockout.
- `entry.security-pending-command-cap` (DP-1400) — 128-command queue cap.
- `ident.please-wait-notice` (DP-1529) — leaving obsolete RFC 931 ident unported.
- `alias.cap-refusal` (L2-006) — 500-alias / 64 KiB cap.
- `DP-1496` — `identify` age wall-clock seam (harness issue; Go matches C).
- `BUG-R2-C1-A2-H1` — empty-hash login fail-open (Go-only bug, not C).
- `src/dream.c:36` `i <= NUM_DREAMS` — `NUM_DREAMS` is 8 for a 9-element array
  (`0..8`), so `<=` is a max-index convention, not an out-of-bounds bug.

## Counts

Total rows: **91** (`CB-001` through `CB-091`).

### By `decision`

| `decision` | Count |
|---|---:|
| `fixed` | 45 |
| `reproduced` | 37 |
| `oracle-patched` | 7 |
| `accidental` | 2 |
| `unclear` | 0 |
| **Total** | **91** |

### By `who_decided`

| `who_decided` | Count |
|---|---:|
| `agent` | 32 |
| `rule` | 30 |
| `Zach` | 29 |
| `unknown` | 0 |
| **Total** | **91** |

### By `reachable` (Classic player reachability)

| `reachable` | Count |
|---|---:|
| `yes` | 56 |
| `no` | 19 |
| `immortal` | 16 |
| **Total** | **91** |

### Found only in code comments

**40** of the 91 decisions were found only in Go code/test comments (not
recorded in `RULEBOOK.md`, `C-ERRATA.md`, `EVIDENCE_LEDGER.tsv`, depth
manifests, or prior audits):

`CB-026`, `CB-028`, `CB-029`, `CB-030`, `CB-033`, `CB-034`, `CB-035`, `CB-036`,
`CB-039`, `CB-043`, `CB-046`, `CB-054`, `CB-055`, `CB-056`, `CB-057`, `CB-058`,
`CB-059`, `CB-060`, `CB-061`, `CB-062`, `CB-063`, `CB-064`, `CB-065`, `CB-068`,
`CB-069`, `CB-070`, `CB-071`, `CB-073`, `CB-077`, `CB-078`, `CB-080`, `CB-081`,
`CB-082`, `CB-083`, `CB-084`, `CB-085`, `CB-086`, `CB-089`, `CB-090`, `CB-091`.

## Accidental departures (2)

1. **`CB-090` — `check_kk_success` (`src/new_cmds.c:1514-1516`) — hallucinated C bug.**
   `pkg/game/skill_berserk_kuji.go:126-150` comments that C's switch omits
   `SKILL_KK_TOH` ("Ported faithfully; do not 'fix'") and leaves `SkillKkToh`
   out of Go's switch, making the Toh seal always fail. In reality,
   `src/new_cmds.c:1514-1516` **has** `case SKILL_KK_TOH: prob = GET_SKILL(ch, SKILL_KK_TOH); break;`.
   Go broke a working C skill on a false claim of C parity.
2. **`CB-091` — `spell_sobriety` (`src/spells.c:690`) — unapproved R4 string.**
   C has `assert(victim)` and aborts on a NULL victim. `pkg/spells/affect_spells.go:1755-1760`
   avoids the crash by returning early and sending an invented `"No target.\r\n"`
   message with no C counterpart and no ruling (violates R4).

## Bookkeeping (verified before push)

```
$ awk -F'\t' 'NR>1' c-bug-decisions.tsv | wc -l
91
$ awk -F'\t' 'NR>1 {print $6}' c-bug-decisions.tsv | sort | uniq -c
      2 accidental
     45 fixed
      7 oracle-patched
     37 reproduced
$ awk -F'\t' 'NR>1 {print $8}' c-bug-decisions.tsv | sort | uniq -c
     32 agent
     30 rule
     29 Zach
$ awk -F'\t' 'NF != 10 {print NR, NF}' c-bug-decisions.tsv | wc -l
0
```

Every row has 10 tab-separated columns; `id` values are unique.
