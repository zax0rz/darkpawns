# Brief 1: Telnet Hardening

**Filed by:** Daeron
**Date:** 2026-06-17
**Priority:** High
**Category:** Security / Bug
**Estimated scope:** 1 file (`pkg/telnet/listener.go`), ~15 lines changed, 2 tests

## Summary

Two HIGH findings in the same file. Both affect the telnet listener — the oldest, most exposed code path in the server. One is a reliability bug (accept loop), the other is a security vulnerability (buffer exhaustion).

---

## Fix 1: Accept loop retry on temporary errors

**File:** `pkg/telnet/listener.go` (lines 82-86)

**Current code:**
```go
go func() {
    for {
        conn, err := ln.Accept()
        if err != nil {
            slog.Error("Telnet accept error", "error", err)
            return
        }
```

**Bug:** The loop returns on ANY error. `net.Listener.Accept()` returns temporary errors (EMFILE — too many open files, transient port exhaustion) that should be retried. After the first transient failure, the server permanently stops accepting all new telnet connections. Nobody reconnects until the server restarts.

**Fix:** Check if the error implements `net.Error` with `Temporary()`. On temporary errors, sleep briefly (100ms) and continue the loop. Only return on permanent errors (listener closed, broken pipe).

```go
go func() {
    for {
        conn, err := ln.Accept()
        if err != nil {
            var netErr net.Error
            if errors.As(err, &netErr) && netErr.Temporary() {
                slog.Warn("Telnet: temporary accept error, retrying", "error", err)
                time.Sleep(100 * time.Millisecond)
                continue
            }
            slog.Error("Telnet accept error, listener stopped", "error", err)
            return
        }
```

**Test:** `TestListen_RetriesOnTemporaryError` — create a mock `net.Listener` that returns a temporary error on `Accept()` twice, then succeeds. Verify the third connection is accepted (not killed by the error). Use a short timeout to confirm the retry loop works.

---

## Fix 2: Unbounded readLine buffer

**File:** `pkg/telnet/listener.go` (line 686)

**Current code:**
```go
line = append(line, b)
```

**Bug:** `readLine()` accumulates bytes with no size check. The `maxInputLen` (1024) truncation only fires when `\r` or `\n` is encountered. A client can send unlimited bytes without newlines — the buffer grows unbounded per connection. With the 5-minute read deadline, a malicious client can OOM the server by streaming data without ever sending a newline.

**Fix:** Add a guard before the append:

```go
if len(line) >= maxInputLen {
    // Discard remaining bytes until newline to prevent buffer exhaustion
    // Log once, then drain until line ending
    slog.Warn("telnet: input exceeds max length, discarding remainder", "max", maxInputLen)
    for {
        b2, err := tc.br.ReadByte()
        if err != nil {
            return "", false
        }
        if b2 == '\r' || b2 == '\n' {
            return string(line[:maxInputLen]), true
        }
    }
}
line = append(line, b)
```

This caps the buffer at `maxInputLen` and drains until the next line ending, preventing unbounded growth while still returning the valid prefix.

**Test:** `TestReadLine_BufferLimit` — dial a telnet connection, send `maxInputLen * 2` bytes without `\r` or `\n`, then send `\r\n` and a valid command. Verify:
1. The server doesn't crash or OOM
2. The truncated input is handled (logged, not silently swallowed)
3. The server remains responsive (the valid command after the oversized input is processed)

---

## Verification Plan

```bash
go build ./...
go vet ./...
go test ./pkg/telnet/... -run "TestListen_Retries|TestReadLine_Buffer" -v
go test -race ./pkg/telnet/...
```

## What "done" looks like

- 2 fixes applied to `listener.go`
- 2 tests written and passing
- `go build ./... && go vet ./... && go test ./...` clean
- One commit, pushed to branch
