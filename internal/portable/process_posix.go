//go:build !windows

package portable

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func prepareProcess(_ *exec.Cmd)                 {}
func interruptProcess(process *os.Process) error { return process.Signal(os.Interrupt) }

// interruptedByRequest reports whether the child died from the SIGINT the
// launcher sends as its graceful stop (interruptProcess), instead of exiting on
// its own. A child that has not installed a handler yet — the bundled server
// installs one only after its boot, as C does in src/comm.c:269,277 — dies from
// the default action, which is what wait(2) then reports as "signal: interrupt".
func interruptedByRequest(waitErr error) bool {
	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		return false
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGINT
}
