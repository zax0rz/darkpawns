package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/testutil"
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

// fakeWorld builds the minimum tree validateWorldDir accepts: <root>/lib/world
// holding a wld/ directory. Boot stops at the database or JWT check long before
// the parser opens a file, so no room data is needed.
func fakeWorld(t *testing.T) string {
	t.Helper()
	worldDir := filepath.Join(t.TempDir(), "lib", "world")
	if err := os.MkdirAll(filepath.Join(worldDir, "wld"), 0o700); err != nil {
		t.Fatalf("create fake world: %v", err)
	}
	return worldDir
}

// repoWorld returns the checkout's world directory, skipping if the test is not
// running inside a full checkout.
func repoWorld(t *testing.T) string {
	t.Helper()
	worldDir := filepath.Join("..", "..", "lib", "world")
	if _, err := os.Stat(worldDir); err != nil {
		t.Skipf("repo lib/ not available: %v", err)
	}
	return worldDir
}

// TestServerBootNoFlagsNamesTheCommand covers the first thing a new operator
// hits: `./server` with no arguments. No flag may be silently required, and the
// refusal has to name the exact command to run instead of just failing.
func TestServerBootNoFlagsNamesTheCommand(t *testing.T) {
	t.Chdir(t.TempDir()) // a directory that is not a checkout
	code, out := bootServer(t, nil,
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "world directory unusable") {
		t.Errorf("expected actionable world-directory error, got:\n%s", out)
	}
	if !strings.Contains(out, "./server -world ./lib/world") {
		t.Errorf("expected the command to run in the message, got:\n%s", out)
	}
}

// TestServerBootNoFlagsFindsCheckoutWorld proves the -world default does not
// need an operator to know the layout: from a checkout root, no flags get as
// far as the database check. That is as far as boot can go without an operator
// supplying a URL, and it is the point.
func TestServerBootNoFlagsFindsCheckoutWorld(t *testing.T) {
	repoWorld(t)
	t.Chdir(filepath.Join("..", ".."))
	code, out := bootServer(t, nil,
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if strings.Contains(out, "world directory unusable") {
		t.Errorf("default -world did not resolve in the checkout root:\n%s", out)
	}
	if !strings.Contains(out, "database URL required") {
		t.Errorf("expected the database refusal, got:\n%s", out)
	}
}

// TestServerBootRejectsParentOfWorldDir pins the level mistake that reads as a
// parser bug: -world lib/ instead of lib/world.
func TestServerBootRejectsParentOfWorldDir(t *testing.T) {
	worldDir := fakeWorld(t) // .../lib/world
	code, out := bootServer(t,
		[]string{"-world", filepath.Dir(worldDir)}, // .../lib
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "no wld/ subdirectory") {
		t.Errorf("expected the world-level hint, got:\n%s", out)
	}
}

func TestServerBootRequiresDatabaseURL(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-world", fakeWorld(t)},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "database URL required") {
		t.Errorf("expected database URL error, got:\n%s", out)
	}
	// The refusal has to carry the URL that actually authenticates on a stock
	// local PostgreSQL, not just the fault.
	if !strings.Contains(out, "host=/var/run/postgresql") {
		t.Errorf("expected the local socket URL in the message, got:\n%s", out)
	}
}

// The database refusal advertises DP_ALLOW_NO_DB. Before this test, following
// that hint exactly reproduced the same refusal: the missing-URL check ignored
// the variable, which was only consulted later on connection failure. A message
// that names an escape hatch and then ignores it is the defect this whole pass
// exists to remove.
func TestAllowNoDBHonouredWhenURLIsMissing(t *testing.T) {
	_, out := bootServer(t,
		[]string{"-world", fakeWorld(t)},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
		"DP_ALLOW_NO_DB=1",
	)
	if strings.Contains(out, "database URL required") {
		t.Fatalf("DP_ALLOW_NO_DB=1 was ignored; the hint the refusal prints does not work\n%s", out)
	}
	// Boot gets past the database gate and on to the world, which is as far as a
	// fake world tree goes: the parse happens before db.New, so the explicit
	// no-persistence warning is not reachable from here. Passing the gate is the
	// whole claim.
	if !strings.Contains(out, "Loading world") {
		t.Errorf("expected boot to reach the world load, got:\n%s", out)
	}
}

func TestServerBootRejectsShortJWTSecretOutsideDevelopment(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-world", fakeWorld(t), "-db", "postgres://unused"},
		"ENVIRONMENT=production",
		"JWT_SECRET=tooshort",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "JWT_SECRET invalid") {
		t.Errorf("expected JWT secret error, got:\n%s", out)
	}
	if !strings.Contains(out, "openssl rand -hex 32") {
		t.Errorf("expected the fix command in the message, got:\n%s", out)
	}
}

// TestStaticSiteFlagReachesBoot proves -static is accepted and validated before
// the world parse begins. The fake world then fails to parse, which is the
// sentinel: a rejected -static would have exited before "Loading world".
func TestStaticSiteFlagReachesBoot(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-static", t.TempDir(), "-world", fakeWorld(t), "-db", "postgres://unused"},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if strings.Contains(out, "static site directory unusable") {
		t.Errorf("-static was rejected despite an existing directory:\n%s", out)
	}
	if !strings.Contains(out, "Loading world") {
		t.Errorf("expected boot to proceed past the site directory check, got:\n%s", out)
	}
}

// TestHugoFlagStillWorksAndWarns guards scripts and units that still pass the
// pre-rename spelling: it must work, and it must say it is on the way out.
func TestHugoFlagStillWorksAndWarns(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-hugo", t.TempDir(), "-world", fakeWorld(t), "-db", "postgres://unused"},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "-hugo is deprecated") {
		t.Errorf("expected a deprecation warning, got:\n%s", out)
	}
	if strings.Contains(out, "static site directory unusable") {
		t.Errorf("-hugo value was not adopted as -static:\n%s", out)
	}
	if !strings.Contains(out, "Loading world") {
		t.Errorf("expected -hugo to keep working, got:\n%s", out)
	}
}

// TestUnusableStaticDirRefusesBeforeParse covers the explicit bad path: an
// operator-supplied directory that does not exist is a refusal, not a silently
// dark front door.
func TestUnusableStaticDirRefusesBeforeParse(t *testing.T) {
	code, out := bootServer(t,
		[]string{"-static", filepath.Join(t.TempDir(), "missing"), "-world", fakeWorld(t), "-db", "postgres://unused"},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", code, out)
	}
	if !strings.Contains(out, "static site directory unusable") {
		t.Errorf("expected static directory refusal, got:\n%s", out)
	}
}

func TestServerBootFailsCleanlyOnUnreachableDatabase(t *testing.T) {
	worldDir := repoWorld(t)
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

func TestMailBootFailureIsNonFatalAndDisablesMail(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("data", "mail"), 0o700); err != nil {
		t.Fatalf("create unusable mail store: %v", err)
	}

	err := initializePersistentMail(testutil.NewMockDatabase())
	if err == nil {
		t.Fatal("mail boot unexpectedly succeeded with a directory mail store")
	}
	// The caller logs this error and continues boot; the initializer itself
	// must leave mail disabled so later sessions cannot use partial state.
	if !strings.Contains(err.Error(), "mail storage initialization failed") {
		t.Fatalf("mail boot error = %v, want storage failure", err)
	}
}
