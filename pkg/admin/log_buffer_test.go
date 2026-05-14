package admin

import (
	"strings"
	"sync"
	"testing"
)

func TestLogBuffer_WriteAndRead(t *testing.T) {
	lb := NewLogBuffer(10)

	// Write entries and verify
	for i := 0; i < 5; i++ {
		_, err := lb.Write([]byte("entry " + string(rune('A'+i))))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	entries := lb.GetRecent(5)
	if len(entries) != 5 {
		t.Errorf("GetRecent(5) returned %d entries, want 5", len(entries))
	}
}

func TestLogBuffer_RingOverflow(t *testing.T) {
	lb := NewLogBuffer(3)

	for i := 0; i < 10; i++ {
		lb.Write([]byte("line " + string(rune('0'+i))))
	}

	entries := lb.GetRecent(10)
	if len(entries) != 3 {
		t.Errorf("GetRecent(10) returned %d entries, want 3 (buffer size)", len(entries))
	}

	// The oldest entry should have been evicted
	for _, e := range entries {
		if strings.Contains(e, "line 0") {
			t.Error("oldest entry should have been evicted from ring buffer")
		}
	}
}

func TestLogBuffer_GetRecentNegative(t *testing.T) {
	lb := NewLogBuffer(5)
	for i := 0; i < 3; i++ {
		lb.Write([]byte("entry"))
	}

	// Negative n should return all entries
	entries := lb.GetRecent(-1)
	if len(entries) != 3 {
		t.Errorf("GetRecent(-1) returned %d entries, want 3", len(entries))
	}
}

func TestLogBuffer_EmptyBuffer(t *testing.T) {
	lb := NewLogBuffer(5)
	entries := lb.GetRecent(10)
	if len(entries) != 0 {
		t.Errorf("GetRecent on empty buffer returned %d entries, want 0", len(entries))
	}
}

func TestLogBuffer_TrimmedWrite(t *testing.T) {
	lb := NewLogBuffer(5)

	// Write with extra whitespace
	n, err := lb.Write([]byte("  hello world  \n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len("  hello world  \n") {
		t.Errorf("Write returned n=%d, expected %d", n, len("  hello world  \n"))
	}

	entries := lb.GetRecent(1)
	if len(entries) != 1 {
		t.Fatalf("GetRecent returned %d entries, want 1", len(entries))
	}
	if entries[0] != "hello world" {
		t.Errorf("entry = %q, want %q", entries[0], "hello world")
	}
}

func TestLogBuffer_EmptyLineIgnored(t *testing.T) {
	lb := NewLogBuffer(5)

	// Empty string after trimming should be skipped
	n, err := lb.Write([]byte("  \n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len("  \n") {
		t.Errorf("Write returned n=%d, expected %d", n, len("  \n"))
	}

	entries := lb.GetRecent(1)
	if len(entries) != 0 {
		t.Errorf("empty line should not be stored, got %d entries", len(entries))
	}
}

func TestLogBuffer_ThreadSafety(t *testing.T) {
	lb := NewLogBuffer(100)
	var wg sync.WaitGroup

	// Concurrent writes from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				lb.Write([]byte("goroutine"))
			}
		}(i)
	}

	wg.Wait()

	entries := lb.GetRecent(200)
	if len(entries) != 100 {
		t.Errorf("expected 100 entries, got %d", len(entries))
	}
}

func TestLogBuffer_SlogHandlerIntegration(t *testing.T) {
	lb := NewLogBuffer(10)
	entries := lb.GetRecent(1)
	if len(entries) != 0 {
		t.Errorf("buffer should start empty")
	}
}
