package portable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperatorCommandSelectsOnlyMatchingPathCommand(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "launcher")
	other := filepath.Join(root, "other")
	for _, p := range []string{exe, other} {
		if err := os.WriteFile(p, []byte("fixture"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	instance := filepath.Join(root, "world")
	if got := operatorCommandFor(exe, instance, instance, exe, "stop"); got != "darkpawns stop" {
		t.Fatal(got)
	}
	custom := filepath.Join(root, "custom world")
	if got := operatorCommandFor(exe, custom, instance, exe, "restart"); !strings.HasPrefix(got, "darkpawns restart --instance ") || !strings.Contains(got, custom) {
		t.Fatal(got)
	}
	for _, path := range []string{"", other} {
		got := operatorCommandFor(exe, instance, instance, path, "stop")
		if !strings.Contains(got, exe) || !strings.Contains(got, "--instance") {
			t.Fatal(got)
		}
	}
}

func TestWindowsLauncherShimRequiresExactOwnedCommand(t *testing.T) {
	exe := `C:\owner's %home%\darkpawns.exe`
	shim := "@chcp 65001 >nul\r\n@\"" + strings.ReplaceAll(exe, "%", "%%") + "\" %*\r\n"
	if !matchesWindowsLauncherShim([]byte(shim), exe) {
		t.Fatal("owned shim not recognized")
	}
	for _, bad := range []string{shim + "echo extra\r\n", strings.ReplaceAll(shim, "%%", "%"), "@darkpawns.exe %*\r\n"} {
		if matchesWindowsLauncherShim([]byte(bad), exe) {
			t.Fatal("unrecognized shim accepted")
		}
	}
}
