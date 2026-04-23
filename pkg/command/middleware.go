package command

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
)

// LoggingMiddleware returns a middleware that logs every command execution.
func LoggingMiddleware() Middleware {
	return func(next Handler) Handler {
		return func(s common.CommandSession, args []string) error {
			start := time.Now()
			err := next(s, args)
			duration := time.Since(start)

			cmdStr := ""
			if len(args) > 0 {
				cmdStr = args[0]
			}

			if err != nil {
				slog.Debug("command failed",
					"cmd", cmdStr,
					"duration", duration,
					"error", err,
				)
			} else {
				slog.Debug("command executed",
					"cmd", cmdStr,
					"duration", duration,
				)
			}
			return err
		}
	}
}

// RateLimitMiddleware returns a middleware that prevents command spam.
// minInterval is the minimum time between commands from the same session.
func RateLimitMiddleware(minInterval time.Duration) Middleware {
	return func(next Handler) Handler {
		var lastCommandTime time.Time

		return func(s common.CommandSession, args []string) error {
			now := time.Now()
			if !lastCommandTime.IsZero() && now.Sub(lastCommandTime) < minInterval {
				// Silently drop — rate limited
				return nil
			}
			lastCommandTime = now
			return next(s, args)
		}
	}
}

// WhitelistMiddleware returns a middleware that only allows specific commands.
func WhitelistMiddleware(allowed ...string) Middleware {
	allowedSet := make(map[string]bool)
	for _, cmd := range allowed {
		allowedSet[strings.ToLower(cmd)] = true
	}

	return func(next Handler) Handler {
		return func(s common.CommandSession, args []string) error {
			if len(args) > 0 && !allowedSet[strings.ToLower(args[0])] {
				return fmt.Errorf("unknown command: %s", args[0])
			}
			return next(s, args)
		}
	}
}
