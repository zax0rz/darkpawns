# Daeron — Operational Reference

Your AGENTS.md points here. One page, verbs only. Created 2026-07-22 (the pointer
dangled before that — if something's missing, add it rather than working from memory).

## Fidelity verdicts

- **The law:** `docs/fidelity/RULEBOOK.md` (R1–R5). Cite rules by number in briefs,
  reviews, and Linear comments — "violates R2a" is a complete verdict. A repeated
  failure indicts the rule, not the file: amend the rulebook + audit the class (R5b/R5c).
  gbrain mirror: `gbrain get darkpawns/rulebook` (repo copy is canonical).
- **The judge:** `DP_ORACLE_BIN=~/.openclaw/workspace/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario <name>` (~40s; exit 0 even on divergence —
  read the `result:` line). Scenarios: `cmd/dp-oracle-diff/scenarios/` — format is
  `[fixture]` / `[setup:oracle]` / `[setup:port]` / `[setup:*:observer]` / `[probe]`;
  copy `communication.txt`'s creation sequences verbatim. Fidelity fixes merge only on
  RED (pre-fix main) → GREEN (branch). Run it yourself; never trust the claim.

## Port progress

- `make reachability` — deterministic C-table-vs-registry report (~2s), dated TSV in
  `docs/reports/`. `make reachability-weekly` — adds delta + JSONL append + commit.
- Time series: `docs/research/metrics/reachability-history.jsonl`. Quick answer:
  `gbrain get darkpawns/port-reachability` (auto-updated Mondays by your cron).
- CI enforces the floor: `scripts/reachability_ci_gate.py` fails any PR that makes a
  reachable command unreachable.

## Brief pipeline

- Process + templates + executor rotation: `docs/briefs/README.md` (kept current as of
  2026-07-22 — codex/GLM rows, oracle gate, git-isolation warning).
- Reference fidelity briefs: `BRIEF-2026-07-22-codex-r2-command-surface.md`,
  `BRIEF-2026-07-22-glm-prefix-matching.md`. Every brief carries a **Git:** section;
  executors have flipped `.git/HEAD` in the live repo twice — check `git status -sb`
  before committing after any agent ran.

## Review mechanics

- GitHub can't take a formal approval on agent PRs (author and reviewer are both
  zax0rz) — post the review as a comment on the driving DP issue in Linear instead;
  that's the audit trail the paper mines anyway.
- Review checklist lives in `docs/briefs/README.md`; the fidelity additions are the
  oracle red→green check and the reachability zero-regression check.

## Paper evidence

- `RESEARCH-LOG.md` — session entries, newest last. The 2026-07-22 entry (migration-kit
  test drive) is the current reference for methodology claims; your weekly digest and
  this log should cross-reference each other.
