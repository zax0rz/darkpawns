// Package launchctl is the game process's client for the launcher's scoped
// control capability. The capability arrives as DP_CONTROL_ADDR and
// DP_CONTROL_TOKEN from the portable launcher and reaches only the console's
// lifecycle transactions (update check/trigger/status, restart) — never a
// bare stop and never the launcher's file or process operations. This
// package deliberately defines its own wire types: the server binary gains no
// updater code, only an HTTP client to the loopback endpoint.
package launchctl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ErrUnavailable reports that this server runs without a launcher (a
// checkout deployment), so lifecycle transactions have nothing to talk to.
var ErrUnavailable = errors.New("no launcher control capability; this server is not a managed instance")

// Client talks to the launcher's private control endpoint.
type Client struct {
	address string
	token   string
}

// FromEnvironment builds the client from the launcher-supplied capability.
// It returns nil when the process was not started by the launcher.
func FromEnvironment() *Client {
	address := strings.TrimSpace(getenv("DP_CONTROL_ADDR"))
	token := strings.TrimSpace(getenv("DP_CONTROL_TOKEN"))
	if address == "" || token == "" {
		return nil
	}
	return New(address, token)
}

// New validates and builds a client. Anything that is not a loopback
// address with a well-formed 64-hex capability yields nil: the launcher's
// endpoint is private by construction and this client refuses to guess.
func New(address, token string) *Client {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil || host != "127.0.0.1" {
		return nil
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return nil
	}
	if len(token) != 64 {
		return nil
	}
	if _, err := hex.DecodeString(token); err != nil {
		return nil
	}
	return &Client{address: strings.TrimSpace(address), token: token}
}

// getenv is a seam so tests can supply the launcher capability.
var getenv = os.Getenv

// Status is the launcher's view of this run. Managed is the console's flag
// for "a launcher owns lifecycle"; it is true exactly when a client exists.
type Status struct {
	Managed bool   `json:"managed"`
	State   string `json:"state,omitempty"`
	RunID   string `json:"run_id,omitempty"`
	Bundle  string `json:"bundle,omitempty"`
}

// UpdateCheck is the read-only release comparison.
type UpdateCheck struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	UpToDate       bool   `json:"up_to_date"`
	ReleaseURL     string `json:"release_url,omitempty"`
	AssetSize      int64  `json:"asset_size,omitempty"`
	Error          string `json:"error,omitempty"`
}

// UpdateState is the journal-backed transaction status.
type UpdateState struct {
	State            string `json:"state"`
	RuntimeState     string `json:"runtime_state,omitempty"`
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

func (c *Client) call(ctx context.Context, method, path string, timeout time.Duration, out any) error {
	if c == nil {
		return ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, "http://"+c.address+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	client := &http.Client{Timeout: timeout, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("launcher control endpoint unreachable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict {
		var conflict struct{ Error string }
		_ = json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&conflict)
		if conflict.Error == "" {
			conflict.Error = "a lifecycle transaction is already pending"
		}
		return fmt.Errorf("%s", conflict.Error)
	}
	if response.StatusCode == http.StatusForbidden {
		return errors.New("launcher refused the scoped capability")
	}
	if response.StatusCode != http.StatusOK && (method != http.MethodPost || response.StatusCode != http.StatusAccepted) {
		return fmt.Errorf("launcher control returned HTTP %d", response.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(out)
}

// Status reads the run record.
func (c *Client) Status(ctx context.Context) (Status, error) {
	var status Status
	if c == nil {
		return Status{Managed: false}, nil
	}
	if err := c.call(ctx, http.MethodGet, "/status", 3*time.Second, &status); err != nil {
		return Status{Managed: true}, err
	}
	status.Managed = true
	return status, nil
}

// CheckUpdate performs the read-only release lookup on the launcher.
func (c *Client) CheckUpdate(ctx context.Context) (UpdateCheck, error) {
	var check UpdateCheck
	err := c.call(ctx, http.MethodGet, "/update-check", 25*time.Second, &check)
	return check, err
}

// TriggerUpdate asks the launcher to run the update transaction. The 202
// returns before the world stops; the caller must not wait on the outcome.
func (c *Client) TriggerUpdate(ctx context.Context) error {
	return c.call(ctx, http.MethodPost, "/update", 5*time.Second, nil)
}

// UpdateStatus reads the journal-backed state that survives the restart.
func (c *Client) UpdateStatus(ctx context.Context) (UpdateState, error) {
	var state UpdateState
	err := c.call(ctx, http.MethodGet, "/update-status", 3*time.Second, &state)
	return state, err
}

// Restart asks the launcher to run the restart transaction.
func (c *Client) Restart(ctx context.Context) error {
	return c.call(ctx, http.MethodPost, "/restart", 5*time.Second, nil)
}
