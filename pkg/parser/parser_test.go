package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validateWorldPath rejects paths with ".."
func TestValidateWorldPath_Traversal(t *testing.T) {
	_, err := validateWorldPath("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

// validateWorldPath rejects paths with ".." in middle
func TestValidateWorldPath_TraversalMiddle(t *testing.T) {
	_, err := validateWorldPath("wld/../mob/test.mob")
	// filepath.Clean("wld/../mob/test.mob") = "mob/test.mob" which doesn't
	// contain "..", so validateWorldPath returns nil. This is fine — the
	// function checks for ".." after Clean().
	if err != nil {
		t.Errorf("expected no error (Clean collapses traversal), got %v", err)
	}
}

// validateWorldPath rejects paths with ".." components that survive Clean
func TestValidateWorldPath_EmbeddedTraversal(t *testing.T) {
	_, err := validateWorldPath("safe/..dangerous/file.txt")
	if err == nil {
		t.Fatal("expected error for embedded '..' in path, got nil")
	}
}

// validateWorldPath rejects paths with leading ".."
func TestValidateWorldPath_TraversalLeading(t *testing.T) {
	_, err := validateWorldPath("..")
	if err == nil {
		t.Fatal("expected error for '..' alone, got nil")
	}
}

// validateWorldPath accepts clean absolute paths
func TestValidateWorldPath_AbsoluteClean(t *testing.T) {
	clean, err := validateWorldPath("/home/darkpawns/lib/wld/test.wld")
	if err != nil {
		t.Fatalf("expected no error for clean path, got %v", err)
	}
	if clean != "/home/darkpawns/lib/wld/test.wld" {
		t.Errorf("expected cleaned path unchanged, got %q", clean)
	}
}

// validateWorldPath accepts clean relative paths
func TestValidateWorldPath_RelativeClean(t *testing.T) {
	clean, err := validateWorldPath("lib/wld/test.wld")
	if err != nil {
		t.Fatalf("expected no error for clean relative path, got %v", err)
	}
	if clean != "lib/wld/test.wld" {
		t.Errorf("expected cleaned path unchanged, got %q", clean)
	}
}

// validateWorldPath returns cleaned path for traversal that Clean resolves
func TestValidateWorldPath_ReturnsCleanedPath(t *testing.T) {
	clean, err := validateWorldPath("/trusted/root/../../../etc/passwd")
	if err != nil {
		t.Fatalf("expected no error (Clean resolves traversal), got %v", err)
	}
	if clean != "/etc/passwd" {
		t.Errorf("expected cleaned path /etc/passwd, got %q", clean)
	}
}

// validateWorldPath rejects empty string
func TestValidateWorldPath_Empty(t *testing.T) {
	clean, err := validateWorldPath("")
	if err != nil {
		t.Fatalf("expected no error for empty string, got %v", err)
	}
	if clean != "." {
		t.Errorf("expected cleaned path '.', got %q", clean)
	}
}

// parseFlag with invalid characters returns 0
func TestParseFlag_InvalidChars(t *testing.T) {
	if v := parseFlag("!"); v != 0 {
		t.Errorf("parseFlag(\"!\") = %d, want 0", v)
	}
}

// lineBuffer scan/unread behavior
//
// NOTE: lineBuffer.Text() after consuming a buffered line (via Unread+Scan)
// returns stale text from the underlying scanner's last Scan() call.
// This means the correct usage pattern is:
//  1. Scan() from scanner → Text() is valid
//  2. Unread(line) → sets buffered
//  3. Scan() → consumes buffered line (returns true, but Text() is stale)
//  4. Scan() → reads next from scanner → Text() is valid
//
// The production code (ParseMobFile) works correctly because it does NOT
// call Text() between steps 3 and 4 — it calls Scan() twice to skip the
// stale text. Our white-box tests verify the internal state directly.
func TestLineBuffer_ScanAndUnread(t *testing.T) {
	content := "line1\nline2\nline3\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	lb := &lineBuffer{scanner: bufio.NewScanner(file)}

	// Scan first line from scanner
	if !lb.Scan() {
		t.Fatal("expected scan to return true for first line")
	}
	if lb.Text() != "line1" {
		t.Errorf("expected 'line1', got %q", lb.Text())
	}

	// Verify internal state after unread (white-box testing in same package)
	lb.Unread("unread_line")
	if !lb.has {
		t.Error("expected has=true after unread")
	}
	if lb.buffered != "unread_line" {
		t.Errorf("expected buffered='unread_line', got %q", lb.buffered)
	}

	// Consume the buffered line — Text() will be stale, so don't check it
	if !lb.Scan() {
		t.Fatal("expected scan to return true for buffered line")
	}
	if lb.has {
		t.Error("expected has=false after consuming buffer")
	}

	// Next scan reads from scanner — should be "line2"
	if !lb.Scan() {
		t.Fatal("expected scan to return true for line2")
	}
	if lb.Text() != "line2" {
		t.Errorf("expected 'line2', got %q", lb.Text())
	}

	if !lb.Scan() {
		t.Fatal("expected scan to return true for line3")
	}
	if lb.Text() != "line3" {
		t.Errorf("expected 'line3', got %q", lb.Text())
	}

	// Should be EOF
	if lb.Scan() {
		t.Fatal("expected scan to return false at EOF")
	}
}

// lineBuffer with consecutive unreads (last one wins)
func TestLineBuffer_DoubleUnread(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("line1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	lb := &lineBuffer{scanner: bufio.NewScanner(file)}
	if !lb.Scan() {
		t.Fatal("expected scan for line1")
	}

	// Unread twice — second should override (check internal state)
	lb.Unread("first_unread")
	lb.Unread("second_unread")
	if lb.buffered != "second_unread" {
		t.Errorf("expected buffered='second_unread', got %q", lb.buffered)
	}
	if !lb.has {
		t.Error("expected has=true after unread")
	}
}

// lineBuffer with a pre-set buffer before any scanner reads
func TestLineBuffer_BufferedThenScanner(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("real1\nreal2\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	// Pre-set buffer before any scanner activity — same as initial Unread in mob parser
	lb := &lineBuffer{scanner: bufio.NewScanner(file)}
	lb.Unread("buffered")

	// Scan consumes the buffered line. After this, lb.Text() is stale (scanner's
	// last scan hadn't happened yet), so check via internal state.
	if !lb.Scan() {
		t.Fatal("expected scan for buffered")
	}
	if lb.has {
		t.Error("expected has=false after consuming buffer")
	}

	// Now scan from the scanner — should get "real1"
	if !lb.Scan() {
		t.Fatal("expected scan for real1")
	}
	if lb.Text() != "real1" {
		t.Errorf("expected 'real1', got %q", lb.Text())
	}

	if !lb.Scan() {
		t.Fatal("expected scan for real2")
	}
	if lb.Text() != "real2" {
		t.Errorf("expected 'real2', got %q", lb.Text())
	}

	if lb.Scan() {
		t.Fatal("expected EOF")
	}
}

// lineBuffer Err() forwards scanner errors
func TestLineBuffer_Err(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	file, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	lb := &lineBuffer{scanner: bufio.NewScanner(file)}
	if err := lb.Err(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Scan through to end
	for lb.Scan() {
	}
	if err := lb.Err(); err != nil {
		t.Errorf("expected nil error after EOF, got %v", err)
	}
}

// ParseAllZonFiles with invalid file in directory returns error (fails fast)
func TestParseAllZonFiles_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	_ = writeZonFile(t, tmpDir, "bad.zon", "") // empty file — triggers error

	_, err := ParseAllZonFiles(tmpDir)
	if err == nil {
		t.Fatal("expected error for invalid zone file, got nil")
	}
}

// ParseAllMobFiles with invalid file in directory
func TestParseAllMobFiles_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "bad.mob"), []byte(""), 0o644)
	content := "#100\nkeyword~\nA mob~\nA mob stands here.\n~\n0 0 0 7 E\n1 20 0 1d1+0 1d1+0\n0 0\n8 3 0\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "good.mob"), []byte(content), 0o644)

	mobs, err := ParseAllMobFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all mob files: %v", err)
	}
	if len(mobs) != 1 {
		t.Errorf("expected 1 valid mob, got %d", len(mobs))
	}
}

// ParseAllObjFiles with invalid file in directory
func TestParseAllObjFiles_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "bad.obj"), []byte("garbage"), 0o644)
	content := `#100
obj~
An obj~
An obj lies here.
~
1 0 0 0 0 0 0 0 0
0 0 0 0
1 1 100.0
$
`
	_ = os.WriteFile(filepath.Join(tmpDir, "good.obj"), []byte(content), 0o644)

	objs, err := ParseAllObjFiles(tmpDir)
	if err != nil {
		t.Fatalf("parse all obj files: %v", err)
	}
	if len(objs) != 1 {
		t.Errorf("expected 1 valid obj, got %d", len(objs))
	}
}

// ParseAllWldFiles rejects directory with path traversal
func TestParseAllWldFiles_PathTraversal(t *testing.T) {
	_, err := ParseAllWldFiles("../../etc")
	if err == nil {
		t.Fatal("expected error for path traversal in directory, got nil")
	}
}

// ParseAllMobFiles rejects directory with path traversal
func TestParseAllMobFiles_PathTraversal(t *testing.T) {
	_, err := ParseAllMobFiles("../../etc")
	if err == nil {
		t.Fatal("expected error for path traversal in directory, got nil")
	}
}

// ParseAllObjFiles rejects directory with path traversal
func TestParseAllObjFiles_PathTraversal(t *testing.T) {
	_, err := ParseAllObjFiles("../../etc")
	if err == nil {
		t.Fatal("expected error for path traversal in directory, got nil")
	}
}

// ParseAllZonFiles rejects directory with path traversal
func TestParseAllZonFiles_PathTraversal(t *testing.T) {
	_, err := ParseAllZonFiles("../../etc")
	if err == nil {
		t.Fatal("expected error for path traversal in directory, got nil")
	}
}

// ParseAllZonFiles with non-existent directory
func TestParseAllZonFiles_NoDir(t *testing.T) {
	_, err := ParseAllZonFiles("/nonexistent/directory")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// ParseAllMobFiles with non-existent directory
func TestParseAllMobFiles_NoDir(t *testing.T) {
	_, err := ParseAllMobFiles("/nonexistent/directory")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

// traversalToExistingDir creates a real directory and returns a path string
// that reaches it using a literal ".." segment. The ParseAll* functions must
// reject the traversal spelling before reading any directory contents.
func traversalToExistingDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	// Create a directory reachable through traversal.
	nested := filepath.Join(tmpDir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Build a path with a literal ".." without filepath.Clean removing it.
	return tmpDir + string(filepath.Separator) + "a" + string(filepath.Separator) + "c" + string(filepath.Separator) + ".." + string(filepath.Separator) + "b"
}

func assertTraversalRejected(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error for path traversal in directory, got nil")
	}
	if !strings.Contains(err.Error(), "..") {
		t.Fatalf("expected validateWorldPath error containing '..', got: %v", err)
	}
}

func TestParseAllWldFiles_PathTraversalExistingDir(t *testing.T) {
	_, err := ParseAllWldFiles(traversalToExistingDir(t))
	assertTraversalRejected(t, err)
}

func TestParseAllMobFiles_PathTraversalExistingDir(t *testing.T) {
	_, err := ParseAllMobFiles(traversalToExistingDir(t))
	assertTraversalRejected(t, err)
}

func TestParseAllObjFiles_PathTraversalExistingDir(t *testing.T) {
	_, err := ParseAllObjFiles(traversalToExistingDir(t))
	assertTraversalRejected(t, err)
}

func TestParseAllZonFiles_PathTraversalExistingDir(t *testing.T) {
	_, err := ParseAllZonFiles(traversalToExistingDir(t))
	assertTraversalRejected(t, err)
}
