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
		return func(s common.CommandSession, cmdStr string, args []string) error {
			start := time.Now()
			err := next(s, cmdStr, args)
			duration := time.Since(start)

			if err != nil {
				slog.Debug(
					"command failed",
					"cmd", cmdStr,
					"duration", duration,
					"error", err,
				)
			} else {
				slog.Debug(
					"command executed",
					"cmd", cmdStr,
					"duration", duration,
				)
			}
			return err
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
		return func(s common.CommandSession, cmdStr string, args []string) error {
			if !allowedSet[strings.ToLower(cmdStr)] {
				return fmt.Errorf("unknown command: %s", cmdStr)
			}
			return next(s, cmdStr, args)
		}
	}
}
