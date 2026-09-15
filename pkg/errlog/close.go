// Package errlog contains small helpers for reporting best-effort cleanup
// failures without changing the primary operation's result.
package errlog

import (
	"io"
	"log/slog"
)

// Close closes resource and records a diagnostic if cleanup fails. Callers use
// this only when the operation's primary result is already determined and a
// close failure must not replace it.
func Close(resource io.Closer, operation string, attrs ...any) {
	if err := resource.Close(); err != nil {
		args := make([]any, 0, 2+len(attrs))
		args = append(args, "operation", operation)
		args = append(args, attrs...)
		args = append(args, "error", err)
		slog.Warn("resource close failed", args...)
	}
}
