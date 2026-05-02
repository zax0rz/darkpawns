# Overnight Code Review Pipeline — Design Doc

**Status:** Draft  
**Created:** 2026-04-27  
**Repo:** `github.com/zax0rz/darkpawns`

## Goal

Automated nightly code review that catches regressions and maintains quality post-Opus audit. Free to run (local model), no manual intervention required.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐     ┌───────────┐
│  cron (2AM) │────▶│  review.sh   │────▶│  Ollama API  │────▶│  Report   │
│  systemd    │     │  (orchestr.) │     │  qwen3:8b    │     │  Markdown │
└─────────────┘     └──────────────┘     └──────────────┘     └─────┬─────┘
                                                                  │
                                          ┌───────────────────────┘
                                          ▼
                                    ┌───────────┐
                                    │  GBrain   │
                                    │  ingest   │
                                    └───────────┘
```

## Model Choice

**Primary: qwen3:8b** (not qwen2.5-coder 7b)
- Free, runs on workstation 3080 Ti via Ollama
- qwen3 > qwen2.5-coder for review quality (better reasoning, instruction following)
- 8B fits comfortably in 12GB VRAM with context window
- If quality is insufficient, upgrade to qwen3:32b (still fits 12GB)

**Fallback:** deepseek-v4-flash via LiteLLM API (costs money, last resort)

## Segmentation Strategy

**Phase 1 — Diff-based (recommended start):**
```bash
# Review only what changed since last review
git diff main@{24.hours ago}..main -- '*.go'
```
- Fast, focused, catches regressions
- Fails gracefully on first run (no baseline = review nothing, or review all once)

**Phase 2 — Package rotation:**
```bash
# Rotate through packages over a week
PACKAGES=(pkg/combat pkg/game pkg/db pkg/session pkg/command pkg/ai pkg/storage)
TODAY=$(( $(date +%u) - 1 ))  # 0=Mon, 6=Sun
REVIEW_TARGET=${PACKAGES[$TODAY]}
```
- Ensures full codebase gets reviewed weekly
- Good for catching stale code issues

**Phase 3 — Hybrid:**
- Nightly: diff-based review
- Weekly (Sunday): full package rotation review

## Prompt Template

```markdown
You are reviewing Go code for a MUD game engine (Dark Pawns), ported from CircleMUD C code.

Focus areas (in priority order):
1. **Concurrency safety:** Missing mutex locks, data races, goroutine leaks, channel misuse
2. **Error handling:** Unchecked errors, swallowed errors, missing error propagation
3. **Resource leaks:** Unclosed files, connections, channels not drained
4. **Logic bugs:** Off-by-one, nil dereference, incorrect conditionals
5. **Port fidelity:** Behavior that differs from the original CircleMUD C implementation

For each finding:
- SEVERITY: CRITICAL / HIGH / MEDIUM / LOW
- FILE:LINE
- Description of the issue
- Suggested fix (concrete code if possible)

Do NOT report:
- Style formatting (golangci-lint handles this)
- Missing comments
- Naming conventions
- Test coverage gaps

Output in markdown format.
```

## Review Focus (post-Opus)

The Opus audit fixed 75/86 findings. The overnight review should catch:
- **New regressions** from ongoing development
- **Unchecked error returns** (50 errcheck findings in golangci-lint baseline)
- **Ineffectual assignments** (12 findings — dead code masking bugs)
- **Concurrency patterns** that the static linter can't catch (lock ordering, deadlock potential)

## Output & Storage

1. **Markdown report** → `~/brain/inbox/reviews/YYYY-MM-DD-review.md`
2. **GBrain ingestion** → auto-synced, searchable via `gbrain query`
3. **Summary to Telegram** → morning notification with count + any CRITICAL/HIGH findings
4. **Git annotation** (optional) — commit hash + review status tag

## Implementation Plan

### Step 1: Ollama setup (workstation)
```bash
# On workstation (192.168.1.185)
ollama pull qwen3:8b
# Verify: ollama run qwen3:8b "test"
```

### Step 2: Review script
```bash
#!/bin/bash
# darkpawns-review.sh — runs on workstation via cron
REPO_DIR="/home/zach/.openclaw/workspace/darkpawns"
REPORT_DIR="$HOME/brain/inbox/reviews"
DATE=$(date +%Y-%m-%d)
OLLAMA_URL="http://localhost:11434/api/generate"

mkdir -p "$REPORT_DIR"

# Get diff (last 24h)
DIFF=$(git -C "$REPO_DIR" diff "main@{24.hours ago}..main" -- '*.go' 2>/dev/null)

if [ -z "$DIFF" ]; then
    echo "No Go changes in last 24h. Skipping review." > "$REPORT_DIR/$DATE-review.md"
    exit 0
fi

# Send to Ollama
curl -s "$OLLAMA_URL" -d "{
    \"model\": \"qwen3:8b\",
    \"prompt\": \"$PROMPT\n\n$DIFF\",
    \"stream\": false,
    \"options\": {\"temperature\": 0.1}
}" | jq -r '.response' > "$REPORT_DIR/$DATE-review.md"
```

### Step 3: Cron
```cron
# Nightly code review at 2:00 AM ET
0 2 * * * /home/zach/.openclaw/workspace/darkpawns/scripts/darkpawns-review.sh >> /var/log/darkpawns-review.log 2>&1
```

### Step 4: Morning notification
- Integrate with existing BRENDA morning brief
- Parse review for CRITICAL/HIGH, include count in digest
- "Overnight review: 0 CRITICAL, 2 HIGH, 5 MEDIUM — [report link]"

## Metrics to Track

- Review coverage (% of changed files reviewed)
- False positive rate (manually sampled weekly)
- Findings per review (trending down = code improving)
- Time to fix (findings → commit)

## Open Questions

1. **Diff-based vs rotation vs hybrid?** — Recommend hybrid (diff nightly + rotation weekly)
2. **Should CRITICAL findings auto-file GitHub issues?** — Could be noisy, defer
3. **Model evaluation:** Run qwen3:8b on a known Opus finding to validate quality before trusting
4. **golangci-lint integration:** Feed lint output into the review prompt for context

## golangci-lint Baseline (2026-04-27)

| Linter | Findings | Priority |
|--------|----------|----------|
| errcheck | 50 | HIGH — unchecked returns |
| unused | 50 | LOW — dead code |
| staticcheck | 32 | MEDIUM — deprecated APIs |
| ineffassign | 12 | HIGH — may mask bugs |
| govet | 0 | ✅ Clean |

**Total: ~144 findings (excluding likely false positives)**

**Next step:** Fix the 12 ineffassign + high-priority errcheck findings first — these are the most likely to be real bugs. Then remove `continue-on-error` from CI.
