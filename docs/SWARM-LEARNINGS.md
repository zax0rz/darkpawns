# Swarm Learnings — April 2026

Lessons from the 15-agent parallel c-to-go port (3-day sprint, ~26 commits, 45+ command categories).

---

## What Worked

### The core pattern: spawn → receive → fix → merge → commit
Holding the full state in context and doing surgical repair on subagent output was the single most important ability. Most models would hit a compile error and keep retrying the same broken thing; the differentiator was reading the actual file on disk, finding exact byte ranges, and using targeted sed/Python fixes.

### K2.6 for writing agents
Moonshot's Kimi K2.6 was the most reliable subagent model. It produced complete, well-structured files with sensible API choices. It cost less than Sonnet and was more reliable than Step 3.5 Flash.

### Command registry pattern
Replacing the 100-line switch statement with `pkg/command/registry.go` was a good early decision. It made adding new commands trivial — just an `init()` registration — and avoided the pileup of case statements that would have been a nightmare with 15 agents.

### File-lock naming convention
`*_cmds.go` per agent worked. Informative commands → `informative_cmds.go`, items → `item_cmds.go`, etc. Made it easy to see who wrote what and find the right file for fixes.

### Build check after every agent
Failing fast let me fix errors while the agent's output was still fresh in my working set.

---

## What Didn't Work

### Spawning 15 agents simultaneously
Biggest mistake. They wrote to overlapping files (especially `commands.go`), created duplicate function declarations, and their output quality varied wildly. The triage overhead was significant and some agents' work was completely discarded.

**Fix for next time:** Spawn in waves of 3-5, with explicit file ownership per agent and staggered timing.

### Agents writing to `commands.go`
Having every agent add its registrations to the same file was a coordination nightmare. Some agents rewrote the entire `init()` block, overwriting previous registrations. Some agents wrote their command functions into `commands.go` *and* into their `*_cmds.go` files, creating duplicate declarations.

**Fix for next time:** Use a separate registration file per package (`registrations.go`) that just imports and calls registrations, or use a post-merge agent that reads all registration stanzas and deduplicates them.

### Truncated output
Multiple agents cut off at line ~700 of 1500+ due to context window limits, leaving incomplete function bodies. This was the source of most compile errors.

**Fix for next time:** Set explicit output size expectations. Break large files into multiple smaller agents. Use a "completeness check" — verify the file ends with valid Go and has proper `}` nesting.

### Wrong branch commits
About 4 subagents committed to nonexistent branches like `feat/regen-limits` instead of `feat/doors`. Had to cherry-pick or ignore.

**Fix for next time:** `git checkout feat/doors` as the first instruction, and confirm the branch name in the task prompt's assumptions.

### Empty output (cryptic failures)
Some agents returned "empty file" with no error message. Others showed internal status messages as their "result" instead of summarizing what was accomplished.

**Fix for next time:** Require agents to end with a structured summary: files written, functions defined, build status.

### Z.AI token quotas
Several agents hit token quotas mid-run on the Z.AI API (GLM-5.1, Step 3.5 Flash). This caused silent failures where work was partially done but not committed.

**Fix for next time:** Use K2.6 or other models with generous quotas for write agents. Reserve Z.AI for analysis/audit tasks that need fewer tokens.

### Stale notification spam
Subagent completion notifications (OpenClaw inter-session messages) often arrived long after the work was already handled. The per-session event inbox has no deduplication or staleness tracking.

**Workaround:** Active polling — check `git log` after big batches instead of relying on notifications.

---

## Data Points

| Metric | Value |
|--------|-------|
| Agents spawned | ~15 |
| Clean commits | ~26 |
| Success rate | ~60% (9 of 15 landed clean code) |
| Avg agent runtime | 20-60 min |
| Compaction recoveries | 6+ |
| Push-to-remote errors | 0 ✅ |
| Lines of Go added | ~10,000+ (estimate) |

---

## Recommended Process for Next Big Swarm

1. **Inventory first** — read all files that will be touched, list their public API
2. **Wave 1** — structural changes only (data models, interfaces)
3. **Wave 2** — command implementations with explicit file assignment
4. **Post-merge agent** — deduplicate registrations, check for orphaned stubs
5. **Build and test sweep** — single agent, no file writes allowed
6. **Commit sweep** — squash, message cleanup

Spawn 3-5 per wave, not 15 all at once. Use one model consistently per wave. Verify branch before every commit instruction.

---

## Early Wave Learnings (Waves 0–1, 2026-04-22)

Archived from root-level SWARM-LEARNINGS.md. These are the foundational lessons that shaped the swarm methodology documented above.

### WAVE 0: Initial swarm dispatch

- **Model routing broken:** All subagents ran on DeepSeek Chat instead of their assigned models (K2.6, GLM-5.1). Reason: default model is DeepSeek Chat, and `model` wasn't explicitly passed to `sessions_spawn`.
  - *Fix:* Always pass `model=` when spawning subagents.
- **QA waves skipped:** Wave 3 (Sonnet QA) and Wave 4 (Opus security) didn't run. Push included un-reviewed code.
  - *Fix:* Enforce wave gating. Next wave doesn't start until QA signs off.
- **Session context lost on compaction:** Long threads get compacted, bot loses the plan.
  - *Fix:* Write plans to files. PORT-PLAN.md and this file survive compaction.
- **Files compiled but incomplete:** Go files had struct/enum equivalents but C business logic wasn't ported to Go functions — just references to C/Lua.
  - *Lesson:* Don't count a C file as "ported" until there's a Go function that implements the same logic, not just references the C.

### QA GATE RULE (enforced from Wave 3 onward)

Build subagents must NOT commit. Their job ends at `go build ./... passes`.

1. Build agents write Go files and report build result — no git operations.
2. Orchestrator waits for all waves in a batch to go green.
3. Orchestrator spawns a QA agent with the full diff and relevant C source files.
4. QA outputs APPROVED or BLOCKED + issues.
5. Commit happens only after APPROVED.

### WAVE 1: First real C-to-Go translation

- **Subagent model matters less than context:** GLM-5.1 and K2.6 both produced reasonable translations when given the full C file content.
  - *Takeaway:* Prompt engineering > model selection for translation work.
- **Build + fix pattern works:** K2.6 does fast first pass, GLM-5.1 does cleanup and compilation fixes.
- **Control flow diverging:** Go uses goroutines/maps/slices where C uses arrays/loops. Edge cases (null pointers, negative indices) need explicit bounds checks.
