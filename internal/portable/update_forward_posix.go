//go:build !windows

package portable

import (
	"context"
	"io"
	"os"
	"syscall"
)

// Replace the anchor process so native managers signal the active supervisor
// directly and retain its exact exit status.
func ForwardOperator(_ context.Context, launcher string, args []string, _ io.Reader, _ io.Writer, _ io.Writer) error {
	return syscall.Exec(launcher, append([]string{launcher}, args...), os.Environ())
}
