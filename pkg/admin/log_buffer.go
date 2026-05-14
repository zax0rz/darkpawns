package admin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// LogBuffer is a simple ring buffer that captures slog output while the server is running.
// It implements io.Writer so it can be attached as a slog handler.
type LogBuffer struct {
	mu      sync.RWMutex
	entries []string
	max     int
}

// NewLogBuffer creates a log buffer with the given capacity.
func NewLogBuffer(maxEntries int) *LogBuffer {
	return &LogBuffer{
		entries: make([]string, 0, maxEntries),
		max:     maxEntries,
	}
}

// Write implements io.Writer. Each Write call appends a trimmed line to the buffer.
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	line := strings.TrimSpace(string(p))
	if line == "" {
		return len(p), nil
	}
	lb.entries = append(lb.entries, line)
	if len(lb.entries) > lb.max {
		lb.entries = lb.entries[len(lb.entries)-lb.max:]
	}
	return len(p), nil
}

// GetRecent returns the last n log entries. If n <= 0 or n > len(entries), returns all.
func (lb *LogBuffer) GetRecent(n int) []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	if n <= 0 || n > len(lb.entries) {
		n = len(lb.entries)
	}
	start := len(lb.entries) - n
	result := make([]string, n)
	copy(result, lb.entries[start:])
	return result
}

// NewSlogHandler creates a slog.Handler that writes to the LogBuffer.
// It wraps a base handler so logs still go to the original destination.
func NewSlogHandler(base slog.Handler, buf *LogBuffer) slog.Handler {
	return &logBufferHandler{
		base: base,
		buf:  buf,
	}
}

type logBufferHandler struct {
	base slog.Handler
	buf  *LogBuffer
}

func (h *logBufferHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *logBufferHandler) Handle(ctx context.Context, record slog.Record) error {
	// Write to base handler
	err := h.base.Handle(ctx, record)
	if err != nil {
		return err
	}
	// Build a one-line entry: LEVEL time message [attrs]
	var sb strings.Builder
	sb.WriteString(record.Level.String())
	sb.WriteByte(' ')
	sb.WriteString(record.Time.Format(time.RFC3339))
	sb.WriteByte(' ')
	sb.WriteString(record.Message)
	record.Attrs(func(a slog.Attr) bool {
		sb.WriteString(fmt.Sprintf(" %s=%v", a.Key, a.Value))
		return true
	})
	h.buf.Write([]byte(sb.String()))
	return nil
}

func (h *logBufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &logBufferHandler{
		base: h.base.WithAttrs(attrs),
		buf:  h.buf,
	}
}

func (h *logBufferHandler) WithGroup(name string) slog.Handler {
	return &logBufferHandler{
		base: h.base.WithGroup(name),
		buf:  h.buf,
	}
}
