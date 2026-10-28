package portable

import (
	"strings"
	"testing"
)

func TestCompareReleaseVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.8", "v0.1.9", -1},
		{"v0.1.9", "v0.1.8", 1},
		{"v0.1.8", "v0.1.8", 0},
		{"v0.1.9", "v0.2.0", -1},
		{"v0.9.0", "v0.10.0", -1},
		{"v1.0.0", "v0.99.99", 1},
		{"v2.0.0", "v10.0.0", -1},
	}
	for _, tc := range cases {
		got, err := compareReleaseVersions(tc.a, tc.b)
		if err != nil || got != tc.want {
			t.Fatalf("%s vs %s = %d, %v; want %d", tc.a, tc.b, got, err, tc.want)
		}
	}
	for _, pair := range [][2]string{{"v0.1.7-local.1", "v0.1.8"}, {"v0.1.8", "0.1.9"}, {"dev", "v0.1.8"}, {"v1.2", "v1.2.3"}} {
		if _, err := compareReleaseVersions(pair[0], pair[1]); err == nil {
			t.Fatalf("accepted a non-stable tag pair: %v", pair)
		}
	}
}

func manifest(version, commit string) Manifest {
	return Manifest{Format: 1, Version: version, Commit: commit, Files: map[string]string{"lib/world/wld/index": "x"}}
}

func TestVersionChangeDecisionRefusesDowngrades(t *testing.T) {
	if err := versionChangeDecision(manifest("v0.1.9", "c1"), manifest("v0.1.8", "c2"), false); err == nil || !strings.Contains(err.Error(), "refusing to downgrade from v0.1.9 to v0.1.8") {
		t.Fatalf("older release accepted: %v", err)
	}
	if err := versionChangeDecision(manifest("v0.1.9", "c1"), manifest("v0.1.8", "c2"), true); err != nil {
		t.Fatalf("--allow-downgrade refused an older release: %v", err)
	}
}

func TestVersionChangeDecisionRefusesEqualVersionDifferentCommit(t *testing.T) {
	err := versionChangeDecision(manifest("v0.1.8", "aaaa"), manifest("v0.1.8", "bbbb"), false)
	if err == nil || !strings.Contains(err.Error(), "different build") {
		t.Fatalf("equal version with a different commit accepted: %v", err)
	}
	if err := versionChangeDecision(manifest("v0.1.8", "aaaa"), manifest("v0.1.8", "bbbb"), true); err != nil {
		t.Fatalf("--allow-downgrade refused an equal rebuild: %v", err)
	}
	// Same version and commit is the up-to-date case, never a refusal.
	if err := versionChangeDecision(manifest("v0.1.8", "aaaa"), manifest("v0.1.8", "aaaa"), false); err != nil {
		t.Fatalf("identical release refused: %v", err)
	}
}

func TestVersionChangeDecisionAcceptsNewerAndSkipsNonStable(t *testing.T) {
	if err := versionChangeDecision(manifest("v0.1.8", "c1"), manifest("v0.1.9", "c2"), false); err != nil {
		t.Fatalf("newer release refused: %v", err)
	}
	// Rehearsal builds are not stable tags; the guard cannot judge them and
	// must not guess the operator into a corner.
	if err := versionChangeDecision(manifest("v0.1.7-local.1", "c1"), manifest("v0.1.8", "c2"), false); err != nil {
		t.Fatalf("non-stable current version refused: %v", err)
	}
}
