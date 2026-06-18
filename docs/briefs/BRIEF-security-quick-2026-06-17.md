# Brief A: Quick Security Fixes

**Filed by:** Daeron
**Date:** 2026-06-17
**Priority:** High
**Category:** Security
**Estimated scope:** 5 files, ~15 lines changed total

## Summary

Three security issues from clawpatch's 95-finding batch. All are single-file, near-single-line fixes. No model deployment required — that's Brief B.

**Deferred:** DP-611 (LiteLLM endpoint path) — decommissioning LiteLLM.

---

## Fix 1: Hardcoded LiteLLM API key in 4 scripts

**Files:**
- `scripts/dp_brenda.py` (line 35-37)
- `scripts/dp_playtester.py` (line 29-31)
- `scripts/dp_session_consolidate.py` (line 32-34)
- `scripts/emotion_llm_classifier.py` (line 35-37)

**Bug:** All four scripts default `LITELLM_KEY` to a literal `'sk-lab…-key'` when the env var is unset. Hardcoded secrets in source are a security incident waiting to happen.

**Fix:** Replace `os.getenv('LITELLM_KEY', 'sk-lab…-key')` with `os.environ['LITELLM_KEY']` in all four files. The script should fail fast with a clear error if the env var is unset, not silently fall back to a literal secret.

**Test:** Not unit-testable in a meaningful way. Manual verification: unset `LITELLM_KEY`, run each script, confirm it errors immediately with `KeyError: 'LITELLM_KEY'` instead of silently using the hardcoded value.

---

## Fix 2: PIIHandler.WithAttrs bypasses PII filtering

**File:** `pkg/privacy/slog_handler.go` (lines 66-71)

**Bug:** `PIIHandler.WithAttrs` passes attrs directly to `h.next.WithAttrs(attrs)` without filtering. These attrs are stored in the inner handler and prepended during `h.next.Handle` — after PIIHandler has already finished filtering. Since `slog.Record.Attrs()` only iterates per-call attrs (not handler-level attrs from `WithAttrs`), handler-level values are never seen by `filterAttr` and appear unfiltered in log output. Same applies to `WithGroup`.

**Fix:** Filter attrs in `WithAttrs` before passing them downstream. For each attr in the slice, call `filterAttr` (or an extracted helper) to strip/mask PII before storing. The filter should apply the same redaction rules that `Handle` uses for per-call attrs.

**Test:** Add `TestPIIHandler_WithAttrsFiltered` — create a handler with `slog.With("email", "user@example.com")`, log a message, verify the output contains the redacted email (not the raw value). This is the regression guard: without the fix, the email appears in plaintext.

---

## Fix 3: Unbounded line buffer growth in readLine

**File:** `pkg/telnet/listener.go` (lines 490-636)

**Bug:** `readLine()` accumulates all non-newline bytes into the `line` slice with no size check. The `maxInputLen` truncation only fires when `\r` or `\n` is encountered. A malicious client can send data without newline characters, causing unbounded memory allocation per connection. With the 5-minute read deadline, this enables sustained allocation and server OOM.

**Fix:** Check `len(line)` against `maxInputLen` during accumulation inside the loop (before the append at line 634). When the limit is reached, either discard excess bytes or immediately return the truncated line. Validate after every append, not just at line endings.

**Test:** Add `TestReadLine_BufferLimit` — dial a telnet listener, send `maxInputLen * 2` bytes without `\r` or `\n`, verify the connection is handled without excessive memory allocation and the server remains responsive. This is a regression guard for the OOM vector.

---

## Verification Plan

```bash
go build ./...
go vet ./...
go test ./pkg/privacy/... -run TestPIIHandler_WithAttrs -v
go test ./pkg/telnet/... -run TestReadLine_Buffer -v
```

For the Python scripts: manually verify each script fails with `KeyError` when `LITELLM_KEY` is unset.

## What "done" looks like

- 3 fixes applied
- 2 new Go tests written and passing
- 4 Python scripts verified to fail fast without env var
- `go build ./... && go vet ./... && go test ./...` clean
- One commit, pushed to branch, PR ready
