package portable

import (
	"errors"
	"io/fs"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
)

// failingFirstVerify fails the first call with the given error and passes the
// second, recording every call.
type failingFirstVerify struct {
	failWith error
	calls    atomic.Int32
}

func (v *failingFirstVerify) verify(string, ...string) error {
	if v.calls.Add(1) == 1 {
		return v.failWith
	}
	return nil
}

func (v *failingFirstVerify) callsMade() int { return int(v.calls.Load()) }

func withSubstitutedVerify(t *testing.T, substitute func(string, ...string) error) {
	t.Helper()
	previous := PublisherVerifyExecutables
	PublisherVerifyExecutables = substitute
	t.Cleanup(func() { PublisherVerifyExecutables = previous })
}

// TestPublisherVerifyForLaunchRetriesTransientOpen pins the CI windows race:
// the candidate update launcher's pre-spawn verification dies when a
// just-extracted executable is momentarily unopenable (real-time scanner),
// the rollback retains the previous lib with no candidate marker, and the
// update smoke's marker assertion FileNotFoundErrors. The retry makes the
// launch path survive the transient open — without the retry (call it
// directly, or revert publisherVerifyForLaunch to the plain call) the first
// error propagates and this test fails.
func TestPublisherVerifyForLaunchRetriesTransientOpen(t *testing.T) {
	flaky := &failingFirstVerify{failWith: &fs.PathError{Op: "open", Path: "darkpawns-server.exe", Err: fs.ErrPermission}}
	withSubstitutedVerify(t, flaky.verify)

	if err := publisherVerifyForLaunch("bundle"); err != nil {
		t.Fatalf("transient open was not retried: %v", err)
	}
	if got := flaky.callsMade(); got != 2 {
		t.Fatalf("verify calls = %d, want 2 (one attempt plus the retry)", got)
	}
}

// TestPublisherVerifyForLaunchRetriesWindowsSharingViolation covers the
// ERROR_SHARING_VIOLATION errno shape, which does not map to fs.ErrPermission.
func TestPublisherVerifyForLaunchRetriesWindowsSharingViolation(t *testing.T) {
	// errno 32 is only interpreted as a sharing violation on windows; the
	// transientOpenError gate must match it there and nowhere else.
	flaky := &failingFirstVerify{failWith: &fs.PathError{Op: "open", Path: "x.exe", Err: syscall.Errno(32)}}
	withSubstitutedVerify(t, flaky.verify)

	err := publisherVerifyForLaunch("bundle")
	if runtimeIsWindows() {
		if err != nil {
			t.Fatalf("sharing violation was not retried: %v", err)
		}
		if got := flaky.callsMade(); got != 2 {
			t.Fatalf("verify calls = %d, want 2", got)
		}
		return
	}
	if err == nil {
		t.Fatal("errno 32 was retried on a non-windows platform; it is not a transient open there")
	}
	if got := flaky.callsMade(); got != 1 {
		t.Fatalf("verify calls = %d, want 1 (no retry off windows)", got)
	}
}

// TestPublisherVerifyForLaunchNeverRetriesVerdicts keeps the security gate
// intact: a verification failure (checksum mismatch, unsigned bundle, missing
// coverage) is a verdict, not a race, and must surface on the first attempt.
func TestPublisherVerifyForLaunchNeverRetriesVerdicts(t *testing.T) {
	verdict := &failingFirstVerify{failWith: errors.New("publisher checksum mismatch: darkpawns-server")}
	withSubstitutedVerify(t, verdict.verify)

	if err := publisherVerifyForLaunch("bundle"); err == nil {
		t.Fatal("a verification verdict was swallowed")
	}
	if got := verdict.callsMade(); got != 1 {
		t.Fatalf("verify calls = %d, want 1: verdicts are never retried", got)
	}
}

// TestTransientOpenErrorShapes pins the classifier: permission denials and
// windows sharing violations are transient; missing files and plain errors
// are not.
func TestTransientOpenErrorShapes(t *testing.T) {
	if !transientOpenError(&fs.PathError{Op: "open", Path: "x", Err: fs.ErrPermission}) {
		t.Error("fs.ErrPermission should classify as transient")
	}
	if transientOpenError(&fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}) {
		t.Error("a missing file is a verdict, not a race")
	}
	if transientOpenError(errors.New("publisher checksum mismatch")) {
		t.Error("a checksum mismatch is a verdict, not a race")
	}
}

func runtimeIsWindows() bool {
	return os.PathSeparator == '\\'
}
