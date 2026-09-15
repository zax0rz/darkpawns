package errlog

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
)

type failingCloser struct{ err error }

func (f failingCloser) Close() error { return f.err }

func TestCloseReportsFailureWithoutReturningIt(t *testing.T) {
	var logOutput bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	Close(failingCloser{err: errors.New("disk full")}, "write fixture", "path", "/tmp/fixture")

	got := logOutput.String()
	if !strings.Contains(got, "resource close failed") || !strings.Contains(got, "write fixture") || !strings.Contains(got, "disk full") {
		t.Fatalf("close failure was not logged with operation context: %q", got)
	}
}

func TestCloseAcceptsNormalCloser(t *testing.T) {
	Close(io.NopCloser(strings.NewReader("ok")), "read fixture")
}
