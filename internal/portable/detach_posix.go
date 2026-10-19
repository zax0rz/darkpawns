//go:build !windows

package portable

import (
	"os/exec"
	"syscall"
)

func detachProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func backgroundConsole() error    { return nil }
