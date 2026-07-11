package examples

import "testing"

// TestExamples smoke-tests the example entry points, asserting they run to
// completion without panicking. The examples previously had no tests at all,
// so a panic (e.g. the old msg[:20] slice on short messages) could land
// unnoticed.

func TestOptimizationIntegration(t *testing.T) {
	// Should not panic; runs the worker pool, connection pool, AI batch,
	// websocket broadcast, and query optimizer examples.
	OptimizationIntegration()
}

func TestMetricsIntegration(t *testing.T) {
	// Should not panic; records a sequence of metrics into the default
	// Prometheus registry and prints the demonstration summary.
	MetricsIntegration()
}

func TestWebsocketOptimizationExampleShortMessage(t *testing.T) {
	// Regression guard for the msg[:20] panic: the example itself broadcasts
	// a 46-byte message, but the consumer must tolerate shorter messages too.
	// This calls the example (which exercises the min(20,len(msg)) guard) and
	// only asserts it does not panic.
	WebsocketOptimizationExample()
}
