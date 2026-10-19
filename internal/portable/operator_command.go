package portable

import (
	"os"
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
	return prefix + quote(executable) + " " + strings.Join(words, " ") + " --instance " + quote(instance)
}
