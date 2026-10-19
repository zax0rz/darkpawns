package portable

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NEW_CONSOLE, HideWindow: true}
}

// A private console lets the background supervisor forward CTRL_BREAK to the
// game even when launched by Task Scheduler, without exposing a console window.
func backgroundConsole() error {
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	handle, _, _ := kernel.NewProc("GetConsoleWindow").Call()
	if handle != 0 {
		return nil
	}
	ok, _, err := kernel.NewProc("AllocConsole").Call()
	if ok == 0 {
		return fmt.Errorf("allocate background shutdown console: %w", err)
	}
	handle, _, _ = kernel.NewProc("GetConsoleWindow").Call()
	windows.NewLazySystemDLL("user32.dll").NewProc("ShowWindow").Call(handle, 0)
	return nil
}
