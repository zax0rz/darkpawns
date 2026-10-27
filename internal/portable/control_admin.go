package portable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// The admin console's browser-facing update flow runs through these control
// messages. The launcher stays the only process that touches releases: the
// game proxies a request, the launcher spawns the same `darkpawns update` /
// `restart` CLI the operator would type, detached exactly like the
// background supervisor. The dying game process never runs its own update.

// UpdateCheckResult is the read-only answer to "is there a newer release".
type UpdateCheckResult struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpToDate       bool   `json:"up_to_date"`
	ReleaseURL     string `json:"release_url,omitempty"`
	AssetSize      int64  `json:"asset_size,omitempty"`
}

// UpdateStatus is the journal-backed state the console polls across a
// restart. State values: idle, staging, applying, testing, recovering,
// committed, rolled-back, pending-recovery.
type UpdateStatus struct {
	State            string `json:"state"`
	RuntimeState     string `json:"runtime_state"`
	RunID            string `json:"run_id,omitempty"`
	CurrentVersion   string `json:"current_version,omitempty"`
	TransactionID    string `json:"transaction_id,omitempty"`
	Phase            string `json:"phase,omitempty"`
	PreviousVersion  string `json:"previous_version,omitempty"`
	CandidateVersion string `json:"candidate_version,omitempty"`
	Error            string `json:"error,omitempty"`
	RecoverCommand   string `json:"recover_command,omitempty"`
	RequestedAt      string `json:"requested_at,omitempty"`
}

// updateCheckEndpoint is a seam so tests can point the check at a fixture
// server instead of GitHub. DP_RELEASE_ENDPOINT takes precedence over it, so
// a mirror or rehearsal feed governs both the check and the transaction.
var updateCheckEndpoint = releaseAPI

func checkEndpoint() string {
	if endpoint := releaseEndpoint(); endpoint != "" {
		return endpoint + "/releases/latest"
	}
	return updateCheckEndpoint
}

// browserUpdateMarker records that the console asked for an update while the
// CLI was still staging (the durable journal only appears once Apply begins).
type browserUpdateMarker struct {
	RequestedAt time.Time `json:"requested_at"`
}

func markerPath(instance string) string {
	return filepath.Join(updatePath(instance), "browser-update.json")
}

func writeUpdateMarker(instance string) {
	_ = os.MkdirAll(updatePath(instance), 0o700)
	data, err := json.Marshal(browserUpdateMarker{RequestedAt: time.Now().UTC()})
	if err != nil {
		return
	}
	_ = updateBytes(markerPath(instance), append(data, '\n'))
}

func readUpdateMarker(instance string) (browserUpdateMarker, bool) {
	var marker browserUpdateMarker
	if err := readJSON(markerPath(instance), &marker); err != nil {
		return marker, false
	}
	return marker, true
}

// lifecycleBusy reports whether another launcher process holds the lifecycle
// lock (an updater mid-Apply, a start, a stop). The probe never steals the
// lock: it acquires only when free and releases immediately.
func lifecycleBusy(instance string) bool {
	lock := flock.New(filepath.Join(instance, ".lifecycle.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return true
	}
	if locked {
		_ = lock.Close()
		return false
	}
	return true
}

func bundleVersion(bundle string) string {
	manifest, err := ReadManifest(bundle)
	if err != nil {
		return ""
	}
	return manifest.Version
}

// controlAdminHandlers builds the four scoped control operations. state
// snapshots the live runtime record under its mutex.
func controlAdminHandlers(instance, bundle string, state func() RuntimeStatus) map[string]http.HandlerFunc {
	writeJSON := func(w http.ResponseWriter, value any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(value)
	}
	return map[string]http.HandlerFunc{
		"/update-check": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			current := bundleVersion(bundle)
			release, err := fetchLatestRelease(ctx, updateClient(), checkEndpoint())
			if err != nil {
				writeJSON(w, map[string]any{"error": fmt.Sprintf("release lookup failed: %v", err), "current_version": current})
				return
			}
			asset, err := selectAsset(release)
			if err != nil {
				writeJSON(w, map[string]any{"error": err.Error(), "current_version": current})
				return
			}
			writeJSON(w, UpdateCheckResult{
				CurrentVersion: current,
				LatestVersion:  release.Tag,
				UpToDate:       current == release.Tag,
				ReleaseURL:     release.HTMLURL,
				AssetSize:      asset.Size,
			})
		},
		"/update": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if err := pendingUpdate(instance); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			self, err := os.Executable()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := os.MkdirAll(filepath.Join(instance, "lib", "logs"), 0o700); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			writeUpdateMarker(instance)
			if err := spawnSupervisor(self, instance, "update", "--yes"); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("could not start the update command: %v", err)})
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "update requested"})
		},
		"/update-status": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			runtime := state()
			status := UpdateStatus{State: "idle", RuntimeState: runtime.State, RunID: runtime.RunID, CurrentVersion: bundleVersion(bundle)}
			record, err := readUpdate(instance)
			switch {
			case errors.Is(err, os.ErrNotExist):
				if marker, ok := readUpdateMarker(instance); ok {
					status.State = "staging"
					status.RequestedAt = marker.RequestedAt.Format(time.RFC3339)
				}
			case err != nil:
				status.State = "unavailable"
				status.Error = err.Error()
			default:
				status.TransactionID = record.ID
				status.Phase = record.Phase
				status.PreviousVersion = bundleVersion(record.Previous)
				status.CandidateVersion = bundleVersion(record.Candidate)
				status.Error = record.Error
				switch record.Phase {
				case "committed", "rolled-back":
					status.State = record.Phase
					status.CurrentVersion = bundleVersion(bundle)
					// The terminal journal outlives the staging marker.
					_ = os.Remove(markerPath(instance))
				case "testing", "recovering":
					if lifecycleBusy(instance) {
						status.State = record.Phase
					} else {
						status.State = "pending-recovery"
						status.RecoverCommand = OperatorCommand(instance, "update", "recover")
					}
				default: // prepared, backing-up, backed-up
					if lifecycleBusy(instance) {
						status.State = "applying"
					} else {
						status.State = "pending-recovery"
						status.RecoverCommand = OperatorCommand(instance, "update", "recover")
					}
				}
			}
			writeJSON(w, status)
		},
		"/restart": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if err := pendingUpdate(instance); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			self, err := os.Executable()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if err := os.MkdirAll(filepath.Join(instance, "lib", "logs"), 0o700); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// `darkpawns restart` stops this very process through the full
			// token, then starts a fresh supervisor; it is detached, so it
			// survives our exit.
			if err := spawnSupervisor(self, instance, "restart"); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("could not start the restart command: %v", err)})
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "restart requested"})
		},
	}
}

// scopedControlPaths are the operations the game process's scoped capability
// may reach. Raw /stop is deliberately absent: the admin console triggers
// lifecycle changes as transactions (restart, update), never a bare stop.
var scopedControlPaths = map[string]bool{
	"/status":        true,
	"/update-check":  true,
	"/update":        true,
	"/update-status": true,
	"/restart":       true,
}
