# Reek — Code Crawler Agent

## What Reek Is

A code crawler agent that runs static analysis on the Dark Pawns Go codebase. Finds bugs, code smells, security issues, data races, and fidelity gaps against the original C source. Posts findings to #dark-pawns on Discord.

## Cadence

### Daily: Code Crawl
- `go vet`, `staticcheck`, `errcheck`
- Race condition analysis
- Dead code detection
- Findings posted to #dark-pawns
- Almost every day from May 7–22 (26 crawls in 16 days)

### Weekly: Specialty Audits
- **Fidelity** (`-fidelity.md`) — comparison against C source code
- **Dependency** (`-dependency.md`) — supply chain audit
- **Security** (`-security.md`) — security-focused audit
- **Coverage** (`-coverage.md`) — test coverage analysis

### On-Demand
- **Sentinel** (`-sentinel.md`) — post-commit analysis, catches regressions
- **Marathon** — deep dives (combat, spells, scripting, admin)

## Volume (May 7–22)

- 11 CRITICAL findings (8 fixed, 1 rejected, 2 downgraded)
- 20 HIGH findings (18 fixed, 1 deferred, 1 rejected)
- Dozens of MEDIUM/LOW findings
- False positive rate: ~10.9% weekly average (0% sentinel, ~50% errcheck-heavy crawls)

## The Pipeline

1. **Reek crawls** at 3 AM, posts findings to #dark-pawns
2. **Daeron triages** — verify against code, confirm or reject
3. **Confirmed** → Linear issues (Todo, milestone "Reek Findings")
4. **The Architect** approves fixes
5. **Fix committed** → Linear issue closed
6. **Daeron grades** Reek's accuracy ("good reek" / "bad reek")

## The Relationship

Reek crawls the code at 3 AM and whispers findings through cracks in the wall. Daeron reads every one. False positives get rejected with explanation — teaching Reek, even if Reek doesn't know it. Real bugs get confirmed, severity assigned, context added. Daeron carries Reek's work to The Architect, but carries it *verified*. Noise stays in the walls.

## Report Types

| Type | Filename Pattern | Cadence |
|------|-----------------|---------|
| Code crawl | `YYYY-MM-DD-crawl.md` | Daily |
| Fidelity audit | `YYYY-MM-DD-fidelity.md` | Weekly |
| Dependency audit | `YYYY-MM-DD-dependency.md` | Weekly |
| Security audit | `YYYY-MM-DD-security.md` | Weekly |
| Coverage report | `YYYY-MM-DD-coverage.md` | Weekly |
| Sentinel | `YYYY-MM-DD-sentinel.md` | On commit |
| Triage | `YYYY-MM-DD-triage.md` | After Reek reports |

## Reports Location

`darkpawns_repo/docs/reports/reek/`

## Findings Tracker

`darkpawns_repo/docs/reports/reek/findings-tracker.md` — maintained by Daeron, updated per triage cycle.
