# Pre-Audit Predictions — Blind Cross-Audit (2026-11-24)

- **Auditor:** Gemini 4 Carbon
- **Recorded:** 2026-11-24 (before opening `CROSS-AUDIT-2026-11-24-sample.tsv` or reading any Go test file)
- **Base commit (`origin/main`):** `ef2b14138`
- **Sample file:** `/home/zach/darkpawns/docs/briefs/CROSS-AUDIT-2026-11-24-sample.tsv`
- **Verified SHA-256:** `aade2e6c07143df528954ea776402ff30b9cf47d18474f28181bd5b44a42b9cb` (matches brief)

## Point Predictions (non-R = F + C + D)

| Package | Sampled tests | Predicted non-R | Predicted non-R % | Plausible range (non-R) |
|---|---:|---:|---:|---:|
| `pkg/session` | 60 | 14 | 23.3% | 9–20 (15%–33%) |
| `pkg/game` | 60 | 10 | 16.7% | 6–16 (10%–27%) |
| **Total** | **120** | **24** | **20.0%** | **15–36 (12.5%–30%)** |

## Prior Reasoning

1. **Strict mutation gate bounds the non-R rate.** Because every F, C, or D mark requires a concrete production-code mutation under which the audited test stays green (or, for C, where a named sibling test catches everything it catches), tests that are merely verbose or imperfectly styled remain `R` if breaking their target branch turns them red.
2. **`pkg/session` (~23% non-R expected):** Command and session handlers emit multi-line player text and depend on session/world fixtures. Common failure modes in MUD command tests are overly broad `strings.Contains` checks (matching a word echoed from input or a shared header while missing the actual branch), tests asserting only non-empty output or no-error return, fixtures that pre-populate the expected post-condition, or narrow single-case tests fully subsumed by an adjacent table-driven test.
3. **`pkg/game` (~17% non-R expected):** Core game mechanics and state structures (`ObjectLocation`, `Equipment`, combat/world helpers) tend to have more direct state assertions that fail under mutation, but large ported packages still accumulate struct-literal/getter tautologies (`D`), tests exercising test-only helpers or stand-in paths rather than the live production function (`F`), and duplicate micro-tests (`C`).
