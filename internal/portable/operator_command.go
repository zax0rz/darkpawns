package portable

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// OperatorCommand produces a runnable recovery command for this executable and
// instance. Paths are quoted for POSIX shells or PowerShell, never Go syntax.
func OperatorCommand(instance string, command ...string) string {
	executable, err := os.Executable()
	if err != nil {
		executable = "darkpawns"
	}
	pathCommand, _ := exec.LookPath("darkpawns")
	defaultPath, _ := DefaultInstance()
	return operatorCommandFor(executable, instance, defaultPath, pathCommand, command...)
}

func operatorCommandFor(executable, instance, defaultPath, pathCommand string, command ...string) string {
	short := sameExecutable(pathCommand, executable)
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	prefix := ""
	if runtime.GOOS == "windows" {
		quote = func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
		prefix = "& "
	}
	words := make([]string, len(command))
	for i, word := range command {
		if word != "" && strings.Trim(word, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_") == "" {
			words[i] = word
		} else {
			words[i] = quote(word)
		}
	}
	if short {
		prefix, executable = "", "darkpawns"
	} else {
		executable = quote(executable)
	}
	result := prefix + executable + " " + strings.Join(words, " ")
	if !short || defaultPath == "" || !sameOperatorPath(instance, defaultPath) {
		result += " --instance " + quote(instance)
	}
	return result
}

func sameOperatorPath(left, right string) bool {
	canonical := func(path string) string {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		if absolute, err := filepath.Abs(path); err == nil {
			path = absolute
		}
		return filepath.Clean(path)
	}
	left, right = canonical(left), canonical(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func sameExecutable(pathCommand, executable string) bool {
	if pathCommand == "" {
		return false
	}
	current, err := os.Stat(executable)
	if err != nil {
		return false
	}
	found, err := os.Stat(pathCommand)
	if err != nil {
		return false
	}
	if os.SameFile(current, found) {
		return true
	}
	if runtime.GOOS == "windows" && strings.EqualFold(filepath.Ext(pathCommand), ".cmd") && found.Size() < 4096 {
		data, err := os.ReadFile(pathCommand)
		return err == nil && matchesWindowsLauncherShim(data, executable)
	}
	return false
}

// Recognize only the exact two-line shim emitted by our Windows installer.
// Arbitrary shell scripts on PATH are not evidence that this release is selected.
func matchesWindowsLauncherShim(data []byte, executable string) bool {
	want := "@chcp 65001 >nul\n@\"" + strings.ReplaceAll(executable, "%", "%%") + "\" %*\n"
	return strings.ReplaceAll(string(data), "\r\n", "\n") == want
}
