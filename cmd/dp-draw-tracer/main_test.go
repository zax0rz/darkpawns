package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyTarget(t *testing.T) {
	dir := t.TempDir()
	goodPath := filepath.Join(dir, "circle-good")
	goodBytes := []byte("reference-oracle-fixture")
	if err := os.WriteFile(goodPath, goodBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	goodSum := sha256.Sum256(goodBytes)
	goodSHA := hex.EncodeToString(goodSum[:])

	badPath := filepath.Join(dir, "circle-bad")
	badBytes := append(append([]byte(nil), goodBytes...), 'X')
	if err := os.WriteFile(badPath, badBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	badSum := sha256.Sum256(badBytes)
	badSHA := hex.EncodeToString(badSum[:])

	t.Run("accepts reference digest", func(t *testing.T) {
		gotSHA, isRef, err := verifyTarget(goodPath, goodSHA, false)
		if err != nil {
			t.Fatalf("verifyTarget(good) error = %v, want nil", err)
		}
		if !isRef || gotSHA != goodSHA {
			t.Fatalf("verifyTarget(good) = (%s, %v), want (%s, true)", gotSHA, isRef, goodSHA)
		}
	})

	t.Run("refuses non-reference digest with both digests and path", func(t *testing.T) {
		gotSHA, isRef, err := verifyTarget(badPath, goodSHA, false)
		if err == nil {
			t.Fatal("verifyTarget(bad) error = nil, want refusal")
		}
		if isRef || gotSHA != badSHA {
			t.Fatalf("verifyTarget(bad) = (%s, %v), want (%s, false)", gotSHA, isRef, badSHA)
		}
		msg := err.Error()
		for _, want := range []string{badPath, badSHA, goodSHA} {
			if !strings.Contains(msg, want) {
				t.Errorf("error %q missing %q", msg, want)
			}
		}
	})

	t.Run("allows non-reference digest when override is set", func(t *testing.T) {
		gotSHA, isRef, err := verifyTarget(badPath, goodSHA, true)
		if err != nil {
			t.Fatalf("verifyTarget(bad, allow=true) error = %v, want nil", err)
		}
		if isRef || gotSHA != badSHA {
			t.Fatalf("verifyTarget(bad, allow=true) = (%s, %v), want (%s, false)", gotSHA, isRef, badSHA)
		}
	})
}

func TestWriteTargetSidecar(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "cap1", "c-draw-sites.log")
	if err := writeTargetSidecar(logPath, "/path/to/circle", "abcd1234", true); err != nil {
		t.Fatalf("writeTargetSidecar: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "cap1", targetSidecarName))
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	want := "target=/path/to/circle sha256=abcd1234 reference=yes\n"
	if string(data) != want {
		t.Fatalf("sidecar = %q, want %q", string(data), want)
	}
}

func TestReferenceOracleSHA256MatchesFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "dp-oracle-diff", "reference-oracle.sha256"))
	if err != nil {
		t.Fatalf("read reference-oracle.sha256: %v", err)
	}
	want := strings.TrimSpace(string(raw))
	if referenceOracleSHA256 != want {
		t.Fatalf("referenceOracleSHA256 = %q, want %q (from cmd/dp-oracle-diff/reference-oracle.sha256)", referenceOracleSHA256, want)
	}
}
