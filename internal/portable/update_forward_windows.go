package portable

import (
	"context"
	"io"
	"os/exec"
	"time"
)

// Windows cannot exec over the anchor. Relay cancellation through a separate
// process group and allow the supervisor's 30-second graceful shutdown window.
func ForwardOperator(ctx context.Context, launcher string, args []string, in io.Reader, out, errOut io.Writer) error {
	for _, arg := range args {
		if arg == "_serve" || arg == "_serve-update" {
			if err := backgroundConsole(); err != nil {
				return err
			}
			break
		}
	}
	child := exec.CommandContext(ctx, launcher, args...)
	prepareProcess(child)
	child.Stdin, child.Stdout, child.Stderr = in, out, errOut
	child.Cancel = func() error { return interruptProcess(child.Process) }
	child.WaitDelay = 45 * time.Second
	return child.Run()
}
