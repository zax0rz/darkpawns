package agentcli

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"
)

// ReconnectConfig controls reconnection behavior.
type ReconnectConfig struct {
	InitialBackoff time.Duration // starting backoff (default: 1s)
	MaxBackoff     time.Duration // cap (default: 60s)
	Multiplier     float64       // backoff multiplier per attempt (default: 2.0)
	Jitter         float64       // random jitter fraction 0-1 (default: 0.3)
	MaxAttempts    int           // 0 = unlimited
}

// DefaultReconnectConfig returns sensible defaults.
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     60 * time.Second,
		Multiplier:     2.0,
		Jitter:         0.3,
		MaxAttempts:    0, // unlimited
	}
}

// Reconnect wraps AgentClient.Connect with exponential backoff + jitter.
// On each failed attempt it logs the error and sleeps before retrying.
// The context can be cancelled to stop retrying.
func (a *AgentClient) Reconnect(ctx context.Context, cfg ReconnectConfig) error {
	attempt := 0
	backoff := cfg.InitialBackoff

	for {
		attempt++

		// Check context before attempting
		select {
		case <-ctx.Done():
			return fmt.Errorf("reconnect cancelled: %w", ctx.Err())
		default:
		}

		slog.Info("connecting", "attempt", attempt, "backoff", backoff)
		err := a.Connect(ctx)
		if err == nil {
			slog.Info("connected", "attempt", attempt)
			return nil
		}

		slog.Warn("connect failed", "attempt", attempt, "error", err, "retry_in", backoff)

		// Check max attempts
		if cfg.MaxAttempts > 0 && attempt >= cfg.MaxAttempts {
			return fmt.Errorf("max reconnect attempts (%d) exceeded: %w", cfg.MaxAttempts, err)
		}

		// Wait with jitter
		jitter := time.Duration(float64(backoff) * cfg.Jitter * (rand.Float64()*2 - 1))
		wait := backoff + jitter
		if wait < 0 {
			wait = 0
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("reconnect cancelled during backoff: %w", ctx.Err())
		case <-time.After(wait):
		}

		// Exponential backoff with cap
		backoff = time.Duration(float64(backoff) * cfg.Multiplier)
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}
}

// RunConnected runs the decision loop with automatic reconnection.
// On disconnect it reconnects and resumes. Only returns on context
// cancellation or fatal error.
func (a *AgentClient) RunConnected(ctx context.Context, rcfg ReconnectConfig) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Connect (with retries)
		if err := a.Reconnect(ctx, rcfg); err != nil {
			return fmt.Errorf("reconnect: %w", err)
		}

		// Run until disconnect
		slog.Info("decision loop starting")
		err := a.RunDecisionLoop(ctx)
		if err == nil {
			// Clean shutdown
			return nil
		}

		slog.Warn("disconnected", "error", err)

		// Close old connection cleanly
		if a.conn != nil {
			_ = a.conn.Close()
			a.conn = nil
		}

		// Brief pause before reconnect attempt (avoid tight loop)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
