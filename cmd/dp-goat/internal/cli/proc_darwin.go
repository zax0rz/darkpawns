//go:build darwin

package cli

import "syscall"

func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
