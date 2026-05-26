# Brief: Linear Board Cleanup & Taxonomy

**For:** Blenda (The Machine)
**From:** Daeron (Loremaster)
**Requested by:** The Architect
**Date:** 2026-05-26

---

## Goal

Clean up the DP team's Linear issues: remove title-prefix hacks, apply proper labels, assign projects, and close stale issues. One-time cleanup to make the board filterable and batch-operable.

---

## Part 1: Install linear-cli (Optional)

The Architect wants `linear-cli` installed for agent convenience. Repo: https://github.com/schpet/linear-cli

```bash
brew install schpet/tap/linear
# or
npm install -D @schpet/linear-cli

# Auth
linear auth login

# Configure per-repo
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
linear config
```

This gives agents `linear issue mine`, `linear issue start`, `linear issue create` etc. from the terminal. Nice-to-have for daily workflows. **Do this after the cleanup, not before.**

---

## Part 2: Create New Labels

These labels don't exist yet on the DP team. Create them:

| Label Name | Color | Description |
|------------|-------|-------------|
| `fidelity` | `#F2994A` (orange) | Port fidelity work — C→Go behavior matching |
| `drift` | `#E8913A` (dark orange) | Behavior differs from C source (subset of fidelity) |
| `stub` | `#D46B4E` (red-orange) | Feature exists but is unimplemented/dead code |
| `memory-leak` | `#EB5757` (red) | Resource leak bugs — objects, memory, goroutines |
| `concurrency` | `#9B51E0` (purple) | Race conditions, lock ordering, channel issues |
| `database` | `#2D9CDB` (blue) | Database redesign feature work |
| `website` | `#6FCF97` (green) | Hugo site, map page, public-facing web |
| `agent` | `#BB87FC` (lavender) | Agent layer, narrative memory, DP-Goat |
| `client` | `#56CCF2` (cyan) | Go client — TUI, CLI, AI agent CLI |
| `infra` | `#F2C94C` (yellow) | Server, deployment, monitoring, Docker |
| `build-gate` | `#4EA7FC` (light blue) | CI/CD, compilation, testing, build pipeline |

**Labels to KEEP** (already exist, no changes):
- `Bug` — all bug types
- `Feature` — new features
- `Improvement` — enhancements to existing features
- `admin-panel` — admin panel phases
- `worldbuilding` — lore, rooms, mobs, entities
- `content` — storytelling, player-facing text
- `narrative-memory` — agent behavior, protocol design
- `implementation` — technical implementation, code
- `research` — research, experiments, AIIDE paper

---

## Part 3: Batch-Update Existing DP Issues

For every open issue on the DP team, apply this mapping:

### Title Prefix → Labels + Clean Title

| Title starts with | Remove prefix | Add label(s) |
|-------------------|---------------|-------------|
| `[Fidelity] CRIT:` | Yes | `fidelity`, `Bug` |
| `[Fidelity] HIGH:` | Yes | `fidelity`, `Bug` |
| `[Fidelity] MEDIUM:` | Yes | `fidelity`, `Bug` |
| `[Fidelity] LOW:` | Yes | `fidelity`, `Bug` |
| `[Port Fidelity]` | Yes | `fidelity`, `Bug` |
| `[Database]` | Yes | `database`, `Feature` |
| `[Map]` | Yes | `website`, `Feature` |
| `[dp-goat]` | Yes | `agent`, `Feature` |
| `DP-GOAT:` | Yes | `agent`, `Feature` |
| `DP-GOAT —` | Yes | `agent`, `Feature` |
| `Build gate —` | No (it's descriptive) | `build-gate` |
| `Link checker —` | No (it's descriptive) | `website`, `Bug` |

### Add sub-type labels where applicable

Scan issue descriptions for these keywords and add the matching label:
- `race condition`, `torn-read`, `data race`, `concurrent` → `concurrency`
- `memory leak`, `orphaned`, `leak` → `memory-leak`
- `stub`, `dead code`, `unimplemented`, `no Go equivalent` → `stub`
- `drift`, `differs from C`, `C behavior`, `Go behavior` → `drift`

### Assign to projects

| Issue prefix/category | Assign to project |
|----------------------|-------------------|
| `[Fidelity]` / `fidelity` label | Dark Pawns |
| `[Database]` / `database` label | Database (create if needed) |
| `[dp-goat]` / `agent` label | DP-Goat |
| `[Map]` / `website` label | Dark Pawns |
| Client work | Dark Pawns |
| Infra work | Infrastructure & Operations |

---

## Part 4: Close Stale Issues

Scan for and close/cancel:
- Issues completed before 2026-05-20 that are still in Todo/Backlog (likely forgotten)
- Duplicate issues (e.g., multiple issues for the same finding)
- Issues with title "[Port Fidelity] DoDig" (DP-243) — already marked false positive, cancel if still open
- Any issue that references deleted code or decommissioned infrastructure (domain-expansion .125 is gone)

---

## Part 5: Post-Cleanup Verification

After all updates:
1. Run `linear issue query --all-teams --state backlog` — should show clean categories
2. Verify no DP issues have `[Fidelity]` or `[Database]` in titles anymore
3. Spot-check 5 random issues to confirm labels are correct
4. Post summary to `#dark-pawns` with stats: X issues cleaned, Y closed, Z labels applied

---

## Execution Notes

- Linear API supports batch label creation and issue updates
- The `linear issue query` command can filter by label after cleanup to verify
- Don't touch issues on other teams (CHAD, HH, LAB) — they have their own org
- If an issue needs both `fidelity` and `Bug`, that's fine — labels are additive
- Priority values (1=Urgent, 2=High, 3=Medium, 4=Low) are already set correctly on most issues — don't change them
