package oraclediff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// defaultOracleBinary is the oracle path the regression driver assumes when
// DP_ORACLE_BIN is unset (scripts/oracle_regression.sh).
const defaultOracleBinary = "/home/zach/darkpawns-c-oracle/bin/circle"

// oracleBinary resolves the C oracle binary, or skips: CI has no oracle
// checkout, exactly like the differential harness.
func oracleBinary(t *testing.T) string {
	t.Helper()
	if bin := os.Getenv("DP_ORACLE_BIN"); bin != "" {
		return bin
	}
	if _, err := os.Stat(defaultOracleBinary); err == nil {
		return defaultOracleBinary
	}
	t.Skip("no C oracle binary: set DP_ORACLE_BIN or build the dp-oracle-seam checkout")
	return ""
}

// oracleRoot derives the oracle checkout root from the binary, or skips.
func oracleRoot(t *testing.T) string {
	t.Helper()
	bin := oracleBinary(t)
	root := filepath.Dir(filepath.Dir(bin))
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skipf("no git oracle checkout beside %s", bin)
	}
	return root
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test source file")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test source file")
		}
		dir = parent
	}
}

// fileSHA256 names the exact artifact in skip/failure messages.
func fileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestOracleSeamSelftest executes the shipped seam against the built oracle.
// The seam prints its own observations under DP_SEAM_SELFTEST=1 and exits
// before booting, so this is a real run of the patched C, not a text match: a
// valid control queues its count without running a heartbeat (proved by `pulse`
// being untouched, which only the drain loop advances), the drain yields that
// count exactly once, an overflowing control is refused without wrapping, and
// controls are ordinary input in normal (non-DP_CLOCK) mode and when malformed.
func TestOracleSeamSelftest(t *testing.T) {
	bin := oracleBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-d", t.TempDir(), "4000")
	cmd.Env = append(os.Environ(), "DP_SEAM_SELFTEST=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("the oracle did not exit for DP_SEAM_SELFTEST: the seam is not draining the self-test\noutput:\n%s", out)
	}
	if !strings.Contains(string(out), "dp-seam-selftest:") {
		// A binary without the hook boots the game instead of self-testing. The
		// default path is whatever build happens to sit in the shared checkout,
		// so name it, hash it and skip; an explicitly named DP_ORACLE_BIN is the
		// checkout the caller means to certify, so that one must carry the hook.
		message := fmt.Sprintf(
			"oracle at %s (sha256 %s) has no DP_SEAM_SELFTEST hook; rebuild it from tools/oracle-seam/dp-determinism.patch",
			bin, fileSHA256(bin))
		if os.Getenv("DP_ORACLE_BIN") != "" {
			t.Fatalf("%s\noutput:\n%s", message, out)
		}
		t.Skip(message)
	}
	if err != nil {
		t.Fatalf("oracle seam self-test failed: %v\noutput:\n%s", err, out)
	}
	for _, want := range []string{
		"dp-seam-selftest: control-queues queued=1260 heartbeats=0",
		"dp-seam-selftest: drain-once drained=1260 pending=0",
		"dp-seam-selftest: overflow-rejected pending=100000",
		"dp-seam-selftest: normal-mode-refused pending=0",
		"dp-seam-selftest: invalid-refused accepted=0",
		"dp-seam-selftest: result=ok",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("self-test output lacks %q\noutput:\n%s", want, out)
		}
	}
}

// TestSeamPatchDefersPulsesOutOfInputProcessing applies the repository's seam
// patch to the documented pristine oracle base and inspects the result. The
// executable proof of the same contract is TestOracleSeamSelftest; this test
// adds the two properties only the patch source can show: the control handler
// contains no heartbeat call at all, and the production (non-DP_CLOCK)
// heartbeat loop is preserved verbatim.
func TestSeamPatchDefersPulsesOutOfInputProcessing(t *testing.T) {
	root := oracleRoot(t)
	patchPath := filepath.Join(repoRoot(t), "tools", "oracle-seam", "dp-determinism.patch")
	if _, err := os.Stat(patchPath); err != nil {
		t.Fatalf("read seam patch: %v", err)
	}

	base, err := exec.Command("git", "-C", root, "show", "d2cb13e:src/comm.c").Output()
	if err != nil {
		t.Skipf("oracle checkout has no pristine base d2cb13e: %v", err)
	}

	work := t.TempDir()
	if err := os.MkdirAll(filepath.Join(work, "src"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "src", "comm.c"), base, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-q", "."},
		{"add", "-A"},
		{"-c", "user.email=seam@test", "-c", "user.name=seam", "commit", "-qm", "pristine"},
	} {
		cmd := exec.Command("git", append([]string{"-C", work}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v unavailable in the test environment: %v\n%s", args, err, out)
		}
	}
	if out, err := exec.Command("git", "-C", work, "apply", "--check", patchPath).CombinedOutput(); err != nil {
		t.Fatalf("seam patch does not apply to the pristine base: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", work, "apply", patchPath).CombinedOutput(); err != nil {
		t.Fatalf("apply seam patch: %v\n%s", err, out)
	}

	applied, err := os.ReadFile(filepath.Join(work, "src", "comm.c"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(applied)

	control := cFunctionBody(t, source, "int process_dpclock_control(char *input)\n{")
	if strings.Contains(control, "heartbeat(") {
		t.Errorf("the control handler must queue pulses only, but its body pumps heartbeats:\n%s", control)
	}
	if !strings.Contains(control, "dp_pending_pulses += count;") {
		t.Errorf("the control handler does not queue its pulses:\n%s", control)
	}
	if drain := cFunctionBody(t, source, "int dp_seam_take(void)\n{"); !strings.Contains(drain, "dp_pending_pulses--;") {
		t.Errorf("the drain primitive does not consume one queued pulse:\n%s", drain)
	}
	if !strings.Contains(source, "while (dp_seam_take())\n        heartbeat(++pulse);") {
		t.Error("the game loop does not drain the queued pulses at its heartbeat slot")
	}
	if !strings.Contains(source, "    } else {\n      while (missed_pulses--)\n        heartbeat(++pulse);\n    }") {
		t.Error("the production (non-DP_CLOCK) heartbeat loop is not preserved verbatim")
	}
}

// cFunctionBody returns the brace-balanced body of the C function that starts
// with header, so a test can assert what that function may and may not call.
func cFunctionBody(t *testing.T, source, header string) string {
	t.Helper()
	start := strings.Index(source, header)
	if start < 0 {
		t.Fatalf("patched source has no function %q", header)
	}
	open := start + len(header) - 1
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[open : i+1]
			}
		}
	}
	t.Fatalf("unterminated body for %q", header)
	return ""
}
