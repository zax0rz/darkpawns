//go:build !windows

package portable

import (
	"os"
	"os/exec"
)

func prepareProcess(_ *exec.Cmd)                 {}
func interruptProcess(process *os.Process) error { return process.Signal(os.Interrupt) }
