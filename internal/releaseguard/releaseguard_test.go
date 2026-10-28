package releaseguard

// Guards the protected publish job's code-execution boundary: no npm, pnpm
// or yarn may appear in any publish-job step, because package-manager
// install scripts are arbitrary third-party code and the job holds the
// release signing key and the package token. The admin UI is prebuilt by a
// secret-less job and DP_PREBUILT_UI collapses the goreleaser npm hooks to
// no-ops; these tests fail the build if any of that regresses.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repoFile(t *testing.T, elements ...string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(append([]string{root}, elements...)...)
	data, err := os.ReadFile(path) // #nosec G304 -- fixed repository paths
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestPublishJobRunsNoPackageManager(t *testing.T) {
	workflow := repoFile(t, ".github", "workflows", "release.yml")
	start := strings.Index(workflow, "\n  publish:")
	if start < 0 {
		t.Fatal("publish job not found in release.yml")
	}
	rest := workflow[start+1:]
	next := regexp.MustCompile(`\n  [a-z-]+:`).FindStringIndex(rest[1:])
	if next != nil {
		rest = rest[:next[0]+1]
	}
	// Comments explain the rule; only executable text is scanned.
	executable := regexp.MustCompile(`(?m)^\s*#.*$|(?m)\s#\s[^'"]*$`).ReplaceAllString(rest, "")
	banned := regexp.MustCompile(`\b(npm|pnpm|yarn)\b`)
	if found := banned.FindString(executable); found != "" {
		t.Fatalf("publish job references %q: package-manager install scripts are third-party code and must never run where the signing key or package token is present", found)
	}
	if !strings.Contains(rest, "DP_PREBUILT_UI") {
		t.Fatal("publish job must set DP_PREBUILT_UI so the goreleaser npm before-hooks collapse to no-ops")
	}
	if strings.Contains(rest, "setup-node") {
		t.Fatal("publish job must not set up Node: nothing there may run npm")
	}
	// Package channels must go out only after the GitHub release is live,
	// so no formula or manifest ever points at a URL that 404s.
	releaseLive := strings.Index(rest, "gh release edit")
	channels := strings.Index(rest, "publish-package-channels.sh")
	if releaseLive < 0 || channels < 0 || channels < releaseLive {
		t.Fatal("the package channels must be published after gh release edit publishes the release")
	}
}

func TestGoreleaserNpmHooksAreConditional(t *testing.T) {
	config := repoFile(t, ".goreleaser.yaml")
	for _, line := range strings.Split(config, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, "npm") {
			if !strings.Contains(trimmed, `{{ if index .Env "DP_PREBUILT_UI" }}`) {
				t.Fatalf("npm hook outside the DP_PREBUILT_UI conditional: %s", trimmed)
			}
			if strings.Contains(trimmed, "npm --prefix admin-ui ci") && !strings.Contains(trimmed, "--ignore-scripts") {
				t.Fatalf("npm ci without --ignore-scripts: %s", trimmed)
			}
		}
	}
}
