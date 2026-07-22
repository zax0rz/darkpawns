# BRIEF (glm) — CI reachability ratchet

**Owner:** glm-5.2. **Gate:** CI must pass on the PR itself; Claude reviews both failure
directions before merge.
**Git:** branch off `main` as `glm/ci-reachability-ratchet` (house convention, cf.
`glm/reset-time-port` → PR #401). Commit to that branch, push it, open a PR against
`main` with the deliverable's both-directions test output in the description. Do NOT
merge — review is Claude/Daeron's. Sized to one small PR.
**Closes:** DP-1188.
**Cite:** rules **R2**, **R5c** (`docs/fidelity/RULEBOOK.md`).

## What

Add a CI job step that fails the build when the count of unreachable commands
(status `implemented-unwired` or `missing`) increases relative to the last committed
snapshot. Same spirit as the existing coverage ratchet (commits a9139c30 + e3a44e59):
a floor that only moves in one direction.

## How

1. In the CI workflow (see the existing security/coverage jobs in
   `.github/workflows/`), add a step:
   - `python3 scripts/gen_reachability.py --out /tmp/reachability-ci.tsv` (deterministic,
     ~2s, stdlib-only — no new deps, no cache needed)
   - Baseline = the newest committed `docs/reports/reachability-*.tsv` (sort by the date
     in the filename, not mtime)
   - Compare unreachable counts (column 6 = status). If CI count > baseline count: print
     each command that is unreachable now but wasn't in the baseline, then exit 1.
     Equal or lower: exit 0, print both counts.
2. Floor as of 2026-07-22: **61** (25 unwired + 36 missing).
3. Put the comparison logic in `scripts/reachability_ci_gate.py` (stdlib only), not
   inline YAML — CI YAML gets a one-line invocation. Reuse the status constants from
   `scripts/reachability_weekly.py` if practical; do not import from it if that drags in
   git/subprocess behavior — a small duplicated constant set with a comment is fine.

## Guardrails — read before writing

- **The awk-exit inversion (e3a44e59) is the named failure mode for this class of gate:**
  the coverage ratchet shipped with pass/fail flipped and broke CI on healthy commits.
  You MUST test both directions locally and show the output in the PR description:
  - healthy: run the gate as-is → exit 0
  - regression: run with a doctored baseline (edit a copy — never a committed TSV — to
    claim fewer unreachable) → exit 1 with the offending commands listed
- New-in-C-table commands (added rows) count toward whichever status they land in — no
  special-casing; the gate is a pure count comparison plus a name diff for the message.
- Don't touch `scripts/gen_reachability.py` or `scripts/reachability_weekly.py`.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

`scripts/reachability_ci_gate.py` + the workflow step + both-directions test output in
the PR description.
