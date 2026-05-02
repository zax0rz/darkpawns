# Dark Pawns — Session Handoff

**Last updated:** 2026-04-25  
**Status:** Repo cleanup done, Opus fidelity audit complete, fix tasks defined, site brief drafted

---

## Where Everything Lives

| Thing | Path |
|-------|------|
| **Go repo** | `~/darkpawns/` (or `/home/zach/.openclaw/workspace/darkpawns/`) |
| **Original C source** | https://github.com/rparet/darkpawns |
| **Cleanup branch** | `chore/repo-cleanup` (commit 6cc5a5d, **not merged to main yet**) |
| **Fidelity audit** | `darkpawns/archive/internal-docs/OPUS-FIDELITY-AUDIT.md` |
| **Fix task briefs** | `darkpawns/archive/internal-docs/FIX-TASK-BRIEFS.md` |
| **GPT-5.5 doc audit** | See subagent sessions `0c240db6` and `7d552183` for raw results |
| **Site design brief** | `darkpawns/docs-site/brief.md` |
| **Research log** | `darkpawns/RESEARCH-LOG.md` |
| **Project stats** | 69K lines C → 69K lines Go, 189 Go files, 169 commits, 8 days (Apr 17–25, 2026) |

---

## What's Done

- [x] Full repo audit (all root files, docs, scripts categorized)
- [x] Repo cleanup on `chore/repo-cleanup` branch (junk removed, internal docs archived, scripts moved, build verified)
- [x] GPT-5.5 doc-vs-source audit (~50 issues across all docs)
- [x] Opus C→Go fidelity audit (6 parallel subagents, 66 C files)
- [x] Fix task briefs written (24 tasks, 10 batches, prioritized)
- [x] Site design brief drafted (darkpawns.io, Claude Design handoff ready)

## What's Next

### Session Start: Fidelity Fixes
1. Merge `chore/repo-cleanup` to main (or rebase)
2. Start with **Task 1.1: Port `act()` engine** — this is the keystone
3. Fan out batches 2–8 in parallel after act() lands
4. Each task brief in FIX-TASK-BRIEFS.md is self-contained

### After Fidelity: Doc Fixes
- Rewrite `docs/architecture.md` (React client doesn't exist, Redis not used, telnet not wired)
- Fix `docs/agent-protocol/` and `docs/agent-sdk/` contradictions (FIGHTING var type)
- Fix SOURCE_ACCURACY_AUDIT.md (many issues already fixed)
- Clean up SECURITY_HARDENING_GUIDE.md (aspirational, not actual)
- Fix path references in docs (doors, shops)

### After Docs: Website
- Buy darkpawns.io ($35/year)
- Build site per brief in `docs-site/brief.md`
- Static-first, Hugo or similar

---

## Key Technical Details

- **Build:** `cd ~/darkpawns && export PATH=$PATH:/usr/local/go/bin && go build ./...`
- **Go version:** 1.23.2
- **Server port:** 8080 (HTTP/WebSocket)
- **Telnet port:** 4000 (exists but not wired in main.go)
- **OpenClaw model for subagents:** GLM-5-Turbo for orchestration, DeepSeek V4 Flash for mechanical tasks, Sonnet for judgment-critical work
- **Opus available** for C→Go comparison tasks (agentId: `opus-reviewer`)
