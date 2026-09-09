package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests exercise the server entrypoint's boot-validation chain by
// re-running main() in a subprocess with arguments that make it exit before
// any listener binds. They guard the most dangerous startup regressions
// (DP-1250): a broken flag chain or init step that only shows up as the
// server failing to boot.

const bootHelperEnv = "DP_SERVER_BOOT_TEST_HELPER"

// TestServerBootHelper is not a real test: it is the subprocess entrypoint.
// When the env guard is set, it re-runs main() with the args passed after
// "--". main() always reaches os.Exit on the validation paths under test;
// the os.Exit(0) fallback covers an unexpected clean return.
func TestServerBootHelper(t *testing.T) {
	if os.Getenv(bootHelperEnv) != "1" {
		return
	}
	// Reset the default FlagSet: the testing framework already parsed the
	// test binary's flags, and main() re-declares its own on flag.CommandLine.
	flag.CommandLine = flag.NewFlagSet("server", flag.ContinueOnError)
	var args []string
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	os.Args = append([]string{"server"}, args...)
	main()
	os.Exit(0)
}

// bootServer runs main() in a subprocess and returns its exit code and
// combined output.
func bootServer(t *testing.T, args []string, env ...string) (int, string) {
	t.Helper()
	cmdArgs := append([]string{"-test.run=TestServerBootHelper", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...) // #nosec G704 -- test re-executes its own binary
	cmd.Env = append(os.Environ(), append(env, bootHelperEnv+"=1")...)
	out, runErr := cmd.CombinedOutput()
	if runErr == nil {
		return 0, string(out)
	}
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		return exitErr.ExitCode(), string(out)
	}
	t.Fatalf("boot helper failed to run: %v\n%s", runErr, out)
	return -1, string(out)
}

func TestServerBootRequiresWorldFlag(t *testing.T) {
	code, out := bootServer(t, nil,
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "Usage: server -world") {
		t.Errorf("expected usage message, got:\n%s", out)
	}
}

func TestServerBootRequiresDatabaseURL(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-world", t.TempDir()},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "Database URL is required") {
		t.Errorf("expected database URL error, got:\n%s", out)
	}
}

func TestServerBootRejectsShortJWTSecretOutsideDevelopment(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-world", t.TempDir(), "-db", "postgres://unused"},
		"ENVIRONMENT=production",
		"JWT_SECRET=tooshort",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "JWT_SECRET invalid") {
		t.Errorf("expected JWT secret error, got:\n%s", out)
	}
}

func TestServerBootFailsCleanlyOnUnreachableDatabase(t *testing.T) {
	worldDir := filepath.Join("..", "..", "lib", "world")
	if _, err := os.Stat(worldDir); err != nil {
		t.Skipf("repo lib/ not available: %v", err)
	}
	code, out := bootServer(t,
		[]string{"-world", worldDir, "-db", "postgres://127.0.0.1:1/unreachable"},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
		"DP_ALLOW_NO_DB=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	// The deep chain must get past world parse and fail loudly at persistence,
	// not panic or hang.
	if !strings.Contains(out, "Database initialization failed") {
		t.Errorf("expected database initialization error, got:\n%s", out)
	}
}
