# Swarm Learnings — Dark Pawns Port

> Collection of lessons from each swarm wave. Read at session startup.

## WAVE 0 (2026-04-22): Initial swarm dispatch

- **Model routing broken:** All subagents ran on DeepSeek Chat instead of their assigned models (K2.6, GLM-5.1). Reason: my default model is DeepSeek Chat, and I didn't explicitly pass `model` to `sessions_spawn`.
  - *Fix:* Always pass `model="moonshot/kimi-k2.6"` etc. when spawning subagents.
- **QA waves skipped:** Wave 3 (Sonnet QA) and Wave 4 (Opus security) didn't run. Push included un-reviewed code.
  - *Fix:* Enforce wave gating. Next wave doesn't start until QA signs off.
- **Session context lost on compaction:** Long threads get compacted, bot loses the plan.
  - *Fix:* Write plans to files. PORT-PLAN.md and this file survive compaction.
- **Files compiled but incomplete:** Go files had struct/enum equivalents but C business logic wasn't ported to Go functions — just references to C/Lua.
  - *Lesson:* Don't count a C file as "ported" until there's a Go function that implements the same logic, not just references the C.

## QA GATE RULE (enforced from Wave 3 onward)

**Build subagents must NOT commit.** Their job ends at `go build ./... passes`.

Workflow:
1. Build agents write Go files and report `go build ./...` result — no `git add`, no `git commit`.
2. Orchestrator waits for all waves in a batch to go green.
3. Orchestrator spawns a **Sonnet QA agent** with the full diff (`git diff HEAD`) and the relevant C source files for faithfulness review.
4. QA agent outputs either: `APPROVED` + any notes, or `BLOCKED` + issues list.
5. Commit happens only after `APPROVED`. If `BLOCKED`, build agents fix and repeat from step 1.

**Why:** Wave 0 skipped QA and pushed broken code. Wave 3 had a failed commit. One Sonnet QA call is cheap insurance.

---

## WAVE 1 (2026-04-22): First wave of real C-to-Go translation

- **Subagent model matters less than context:** GLM-5.1 and K2.6 both produced reasonable translations when given the full C file content in the prompt.
  - *Takeaway:* Prompt engineering > model selection for translation work. Give the agent the actual C file contents.
- **Build + fix pattern works:** K2.6 does a fast first pass, GLM-5.1 (slow but steady, long context) does cleanup and compilation fixes.
  - *Best flow:* K2.6 writes the Go → GLM-5.1 reads it, checks against C original, fixes compile errors, verifies logic.
- **Control flow diverging:** Go uses goroutines/maps/slices where C uses arrays/loops. This is fine for ported behavior but we should check that edge cases (null pointers, negative array indices) are handled.
  - *Check:* Every `ch->array[index]` in C has a bounds check or equivalent in Go.
  - *Check:* Every C `NULL` check maps to a Go `nil` check for the same case.
