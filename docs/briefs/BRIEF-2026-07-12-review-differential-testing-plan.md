# Brief: Review — Differential Testing Plan — 2026-07-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Model:** Third-party reviewer (Gemini, Kimi, or MiMo — NOT Sol or Fable, who execute the other two briefs)
**Purpose:** Poke holes in the plan before we spend expensive model tokens executing it.

---

## Context

Dark Pawns is a CircleMUD-based MUD, ported from ~73,000 lines of C to ~95,000 lines of Go. The C source lives at `src/` in this repo (rparet/darkpawns, the 2010 shutdown dump). The Go port lives at `pkg/`, `cmd/`, `server/`.

We've been doing targeted fidelity audits (reading C files, comparing to Go, finding drift). We've closed 200+ issues this way. But it's ad hoc — we find what we look for, we miss what we don't.

**The goal:** Prove behavioral equivalence between the C and Go servers. Not "the code looks similar" — "given the same inputs, both servers produce the same outputs."

**The plan:** Three models, split by comparative advantage:
1. **GPT-5.6 Sol** (OpenAI frontier) — Build the C server, map C→Go correspondences, create mechanical infrastructure
2. **Claude Fable** (Anthropic frontier) — Design the differential test harness, write behavioral test specifications
3. **You** (reviewer) — Poke holes in this plan before execution

---

## Your Task

Read the two execution briefs (Sol's and Fable's) and the context below. Then answer these questions:

### 1. Feasibility

- Is getting the C server (rparet/darkpawns) compiling and running on macOS realistic? The autotools build checks for libm, libcrypt, libz. Any blockers you see?
- The C server uses flat-file player storage (no database). Can it run standalone with just the `lib/` directory?
- CircleMUD's `comm.c` uses `select()` for multiplexing. Any macOS compatibility issues?

### 2. Mapping Completeness

- Sol's brief asks for a function-level C→Go mapping. Is file-level sufficient, or do we need function-level? What granularity catches the most drift?
- The C codebase has 66 .c files. The Go codebase has ~211 .go files across ~20 packages. How do you handle one C file mapping to multiple Go packages (e.g., `act.item.c` → `pkg/game/` + `pkg/session/`)?
- Should the mapping include a "confidence score" for each correspondence?

### 3. Test Design

- Fable's brief proposes scripting telnet sessions and diffing output. Is there a better approach? What about:
  - Unit tests that call the same internal functions with the same inputs?
  - Instrumented builds that log every decision point?
  - Property-based testing (given random inputs, do both servers agree)?
- What's the right granularity for "behavioral equivalence"? Room descriptions? Combat outcomes? Stat calculations? All of the above?
- How do we handle intentional divergences (Go adds features C didn't have, like Lua scripting, the dream system, agent vars)?

### 4. Risk Assessment

- What's the most likely failure mode?
- What's the most likely source of false positives (differences that look like bugs but aren't)?
- What's the most likely source of false negatives (bugs we still won't find)?
- Is there a simpler version of this plan that gets 80% of the value?

### 5. Ordering

- Should the review happen before Sol starts building, or in parallel?
- Should Fable wait for Sol's mapping, or start designing the harness framework independently?
- What's the critical path?

---

## Context Files to Read

- `docs/briefs/BRIEF-2026-07-12-sol-c-build-and-mapping.md` — Sol's execution brief
- `docs/briefs/BRIEF-2026-07-12-fable-differential-test-design.md` — Fable's execution brief
- `src/Makefile` — C build system (autotools-generated)
- `docs/briefs/README.md` — Our brief workflow and standards
- `RESEARCH-LOG.md` — What we've documented so far about the port

## Output

Write your review to `docs/briefs/REVIEW-2026-07-12-differential-testing-plan.md`. Be blunt. If something won't work, say so. If there's a better approach, describe it. The goal is to catch problems before we spend $50+ in model tokens executing a flawed plan.

---

## After Review

Daeron reads the review, incorporates feedback, revises the Sol and Fable briefs if needed, then dispatches execution.
