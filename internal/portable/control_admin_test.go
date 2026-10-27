package portable

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gofrs/flock"
)

func startControl(t *testing.T, instance string) (ScopedCapability, controlRecord) {
	t.Helper()
	_, cancel := context.WithCancel(context.Background())
	_, closeControl, scoped, err := serveControl(instance, "release", 4350, cancel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = closeControl(nil) })
	var record controlRecord
	if err := readJSON(filepath.Join(instance, ".control.json"), &record); err != nil {
		t.Fatal(err)
	}
	return scoped, record
}

func controlCall(t *testing.T, address, token, method, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, "http://"+address+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func TestScopedCapabilityCannotStopButReachesTransactions(t *testing.T) {
	instance := t.TempDir()
	scoped, record := startControl(t, instance)
	if scoped.Token == "" || scoped.Token == record.Token {
		t.Fatal("scoped token missing or identical to the full capability")
	}
	if err := os.MkdirAll(updatePath(instance), 0o700); err != nil {
		t.Fatal(err)
	}
	writeJournal(t, instance, updateRecord{ID: strings.Repeat("c", 32), Phase: "prepared", Previous: instance, Candidate: filepath.Join(updatePath(instance), strings.Repeat("c", 32), "release")})
	if response := controlCall(t, record.Address, scoped.Token, http.MethodPost, "/stop"); response.StatusCode != http.StatusForbidden {
		t.Fatalf("scoped capability invoked a bare stop: HTTP %d", response.StatusCode)
	}
	for _, probe := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/status", http.StatusOK},
		{http.MethodGet, "/update-status", http.StatusOK},
		{http.MethodGet, "/update-check", http.StatusOK}, // fixture endpoint below
		// A pending recovery journal must conflict both transactions before
		// anything is spawned: single-sourced recovery stays the rule.
		{http.MethodPost, "/update", http.StatusConflict},
		{http.MethodPost, "/restart", http.StatusConflict},
	} {
		fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(githubRelease{Tag: "v0.0.0", Assets: []releaseAsset{}})
		}))
		t.Cleanup(fixture.Close)
		original := updateCheckEndpoint
		updateCheckEndpoint = fixture.URL
		defer func() { updateCheckEndpoint = original }()
		response := controlCall(t, record.Address, scoped.Token, probe.method, probe.path)
		if response.StatusCode != probe.want {
			t.Fatalf("%s %s via scoped token: HTTP %d, want %d", probe.method, probe.path, response.StatusCode, probe.want)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, "http://"+record.Address+"/status", nil)
	req.Header.Set("Authorization", "Bearer "+scoped.Token)
	req.Header.Set("Origin", "https://example.com")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatal("a browser Origin must be refused even with a valid token")
	}
}

func journalRecord(instance, id, phase string) updateRecord {
	return updateRecord{ID: id, Phase: phase, Previous: instance, Candidate: filepath.Join(updatePath(instance), id, "release")}
}

func writeJournal(t *testing.T, instance string, record updateRecord) {
	t.Helper()
	if err := os.MkdirAll(updatePath(instance), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := saveUpdate(instance, record); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateStatusStates(t *testing.T) {
	valid := func(instance, phase string) updateRecord {
		return journalRecord(instance, strings.Repeat("a", 32), phase)
	}
	cases := []struct {
		name    string
		prepare func(t *testing.T, instance string)
		want    string
	}{
		{"idle", func(*testing.T, string) {}, "idle"},
		{"staging marker", func(t *testing.T, instance string) { writeUpdateMarker(instance) }, "staging"},
		{"committed", func(t *testing.T, instance string) { writeJournal(t, instance, valid(instance, "committed")) }, "committed"},
		{"rolled-back", func(t *testing.T, instance string) { writeJournal(t, instance, valid(instance, "rolled-back")) }, "rolled-back"},
		{"applying while lock held", func(t *testing.T, instance string) {
			writeJournal(t, instance, valid(instance, "backed-up"))
			lock := flock.New(filepath.Join(instance, ".lifecycle.lock"))
			if locked, err := lock.TryLock(); err != nil || !locked {
				t.Fatalf("could not take the lifecycle lock: %v %v", locked, err)
			}
			t.Cleanup(func() { _ = lock.Close() })
		}, "applying"},
		{"interrupted becomes pending recovery", func(t *testing.T, instance string) {
			writeJournal(t, instance, valid(instance, "prepared"))
		}, "pending-recovery"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			instance := t.TempDir()
			handlers := controlAdminHandlers(instance, "release", func() RuntimeStatus { return RuntimeStatus{State: "running"} })
			test.prepare(t, instance)
			recorder := httptest.NewRecorder()
			handlers["/update-status"](recorder, httptest.NewRequest(http.MethodGet, "/update-status", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP %d", recorder.Code)
			}
			var status UpdateStatus
			if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
				t.Fatal(err)
			}
			if status.State != test.want {
				t.Fatalf("state %q, want %q (body %s)", status.State, test.want, recorder.Body.String())
			}
			if status.State == "pending-recovery" && !strings.Contains(status.RecoverCommand, "update recover") {
				t.Fatalf("recovery command missing: %q", status.RecoverCommand)
			}
		})
	}
}

func TestUpdateStatusClearsStagingMarkerOnTerminalPhase(t *testing.T) {
	instance := t.TempDir()
	writeUpdateMarker(instance)
	writeJournal(t, instance, journalRecord(instance, strings.Repeat("b", 32), "committed"))
	handlers := controlAdminHandlers(instance, "release", func() RuntimeStatus { return RuntimeStatus{State: "stopped"} })
	recorder := httptest.NewRecorder()
	handlers["/update-status"](recorder, httptest.NewRequest(http.MethodGet, "/update-status", nil))
	if _, err := os.Stat(markerPath(instance)); !os.IsNotExist(err) {
		t.Fatal("staging marker survived a terminal journal")
	}
}

func TestUpdateCheckIsReadOnlyAndReportsDrift(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("update check must be read-only, saw %s", r.Method)
		}
		extension := ".tar.gz"
		if runtime.GOOS == "windows" {
			extension = ".zip"
		}
		name := "darkpawns_9.9.9_" + runtime.GOOS + "_" + runtime.GOARCH + extension
		_ = json.NewEncoder(w).Encode(githubRelease{
			Tag:     "v9.9.9",
			HTMLURL: "https://github.com/zax0rz/darkpawns/releases/tag/v9.9.9",
			Assets:  []releaseAsset{{Name: name, URL: "https://github.com/zax0rz/darkpawns/releases/download/v9.9.9/" + name, Digest: "sha256:" + strings.Repeat("0", 64), Size: 42}},
		})
	}))
	defer fixture.Close()
	original := updateCheckEndpoint
	updateCheckEndpoint = fixture.URL
	defer func() { updateCheckEndpoint = original }()

	instance := t.TempDir()
	handlers := controlAdminHandlers(instance, "release", func() RuntimeStatus { return RuntimeStatus{State: "running"} })
	recorder := httptest.NewRecorder()
	handlers["/update-check"](recorder, httptest.NewRequest(http.MethodGet, "/update-check", nil))
	var result UpdateCheckResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.LatestVersion != "v9.9.9" || result.UpToDate {
		t.Fatalf("check result wrong: %+v (body %s)", result, recorder.Body.String())
	}
	if result.CurrentVersion != "" {
		t.Fatal("fixture bundle has no manifest; current version must be empty, not invented")
	}
}

func TestReleaseEndpointOverrideFeedsCheckAndDownloadBase(t *testing.T) {
	if endpoint := releaseEndpoint(); endpoint != "" {
		t.Fatal("without the environment set there is no override")
	}
	if got := effectiveReleaseAPI(); got != releaseAPI {
		t.Fatalf("default endpoint changed: %s", got)
	}
	t.Setenv("DP_RELEASE_ENDPOINT", "http://127.0.0.1:8399")
	if got := effectiveReleaseAPI(); got != "http://127.0.0.1:8399/releases/latest" {
		t.Fatalf("override endpoint wrong: %s", got)
	}
	if got := downloadBaseFor("v9.9.9"); got != "http://127.0.0.1:8399/releases/download/v9.9.9/" {
		t.Fatalf("override download base wrong: %s", got)
	}
	// A mirror-hosted asset passes selection only under the override.
	tag := "v9.9.9"
	name := "darkpawns_9.9.9_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	asset := releaseAsset{Name: name, URL: downloadBaseFor(tag) + name, Digest: "sha256:" + strings.Repeat("0", 64), Size: 42}
	if _, err := selectAsset(githubRelease{Tag: tag, Assets: []releaseAsset{asset}}); err != nil {
		t.Fatalf("mirror asset rejected under override: %v", err)
	}
	// Without the override the same local URL must be refused.
	t.Setenv("DP_RELEASE_ENDPOINT", "")
	if _, err := selectAsset(githubRelease{Tag: tag, Assets: []releaseAsset{asset}}); err == nil {
		t.Fatal("non-GitHub asset accepted without the override")
	}
	// Malformed overrides are ignored, never guessed.
	for _, bad := range []string{"ftp://example.com", "http://example.com/path", "http://user@example.com", "http://example.com/?q=1"} {
		t.Setenv("DP_RELEASE_ENDPOINT", bad)
		if got := releaseEndpoint(); got != "" {
			t.Fatalf("accepted malformed override %q", bad)
		}
	}
}
