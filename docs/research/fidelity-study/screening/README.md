# Incident screening, v2 (frozen 2026-11-01)

The adjudicated incident register for the port-fidelity study. Sol (OpenResearch, gpt-6.1-sol) screened it; Claude reviewed it in Zach's session. Coding follows the definitions in [`../STUDY.md`](../STUDY.md) at commit `56571bbc6`.

## Counts

| Outcome | Count |
|---|---:|
| Core A: a named check stopped short of its claim | 16 |
| Core B: the comparison was wrong | 17 |
| Core C: a failed or unrun check recorded as a non-failure | 5 (7 counting load mob and accuse as secondary C) |
| **Core incidents** | **38** |
| Supporting (outside the game port) | 6 |
| Uncovered: no check claimed the behaviour | 43 |
| Excluded | 68 |
| Unresolved | 5 |
| Aggregate contexts (not incidents) | 4 |

The 164 records reconcile. Another 306 git-only leads were routed but not screened (260 tagged for the slop track); the counts above don't include them.

## What the review checked

- The v1 exclusion sample (20 of 69, seed `20261101`) reproduces from its sorted, hashed frame.
- All 167 v1 files still match their recorded hashes; v1 stays in Sol's workspace as the superseded freeze.
- Every A names its claiming check; the moves to uncovered were spot-checked against their sources.
- The C cases were checked against primary artifacts (commits, PRs, the units report).

## Limits

- Single screener, with review by Claude; no second independent coder.
- The git search is date-biased: 604 of 606 matching commits are on or after 2026-09-28, so a low early count doesn't mean fewer early problems.
- Incidents were reconstructed from commits, PRs, field notes and the ledger. Some receipts (Linear tickets) weren't available.

## Files

- `screening-handoff.md`: Sol's summary of the freeze.
- `register.md`, `register.tsv`, `register.jsonl`: the register (the `.jsonl` is authoritative).
- `counts.json`: the counts.
- `A-operational-definition.json`, `A-named-check-audit.md`: the A rule and the re-coding of the 57 v1 A records.
- `C-operational-definition.json`, `C-coding-guide.md`: the C rule.
- `uncovered-records.jsonl`, `slop-track-register.jsonl`: input for the slop track.
- `unresolved.md`, `v1-to-v2-changes.jsonl`, `merge-log.jsonl`: open items, the revision and the duplicate merges.

Source: `~/.local/share/openresearch/files/darkpawns/research-scoping/screening-final-v2/` (`register.jsonl` sha256 `26bea8dbd16a271e2648576ea40fc166a3aeaf8d2d51499053ef634ee9516b84`).
