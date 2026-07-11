package examples

import (
	"fmt"
	"time"

	"github.com/zax0rz/darkpawns/pkg/metrics"
)

func MetricsIntegration() {
	// Example of integrating metrics into Dark Pawns components

	fmt.Println("Demonstrating Dark Pawns Metrics Integration")

	// Simulate player connections
	metrics.ConnectionOpened()
	metrics.SetPlayersOnline(1)
	time.Sleep(100 * time.Millisecond)

	// Simulate commands
	start := time.Now()
	time.Sleep(50 * time.Millisecond) // Simulate command processing
	metrics.CommandProcessed("look", time.Since(start))

	start = time.Now()
	time.Sleep(30 * time.Millisecond)
	metrics.CommandProcessed("move", time.Since(start))

	// Simulate combat
	metrics.CombatRound()
	metrics.DamageDealt("player", 15)
	metrics.DamageDealt("mob", 8)

	// Simulate errors
	metrics.ErrorOccurred("websocket")
	metrics.ErrorOccurred("database")

	// Simulate database operations
	metrics.DBQuery(20 * time.Millisecond)

	// Simulate memory operations
	metrics.MemoryWrite()
	metrics.MemoryRead()

	// Simulate player disconnection
	metrics.ConnectionClosed()
	metrics.SetPlayersOnline(0)

	fmt.Println("Metrics integration demonstration complete")
	fmt.Println()
	fmt.Println("Note: this example records metrics into the default Prometheus")
	fmt.Println("registry but does not start an HTTP server, so it does not serve")
	fmt.Println("/metrics itself. In the live server, cmd/server registers")
	fmt.Println("/metrics (see main.go), so the recorded values are scraped there.")
	fmt.Println("Typical scrape URLs in a dev setup:")
	fmt.Println("  Dark Pawns /metrics: http://localhost:4350/metrics")
	fmt.Println("  Grafana:             http://localhost:3000")
	fmt.Println("  Prometheus:          http://localhost:9090")
}
