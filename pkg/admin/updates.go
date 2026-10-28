package admin

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/internal/launchctl"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// registerUpdates serves the console's update page: read-only release
// comparison, the update-now transaction handoff, and the journal-backed
// status that survives the restart. Every mount sits behind
// requireRole("admin") with the settings page: both are deployment-level.
func registerUpdates(api huma.API, logger *audit.AuditLogger, control *launchctl.Client, world *game.World, buildVersion, buildCommit string) {
	// One release query per minute: the check hits GitHub, and an implementor
	// mashing the button must not become a traffic amplifier.
	var checkMu sync.Mutex
	var lastCheck time.Time

	type statusBody struct {
		Managed       bool                   `json:"managed"`
		ServerVersion string                 `json:"server_version"`
		Commit        string                 `json:"commit,omitempty"`
		OnlinePlayers int                    `json:"online_players"`
		Update        *launchctl.UpdateState `json:"update,omitempty"`
	}
	type statusOutput struct{ Body statusBody }
	huma.Register(api, huma.Operation{
		OperationID: "get-updates-status",
		Method:      http.MethodGet,
		Path:        "/admin/updates/status",
		Summary:     "Release state and update transaction status (implementor)",
		Description: "On a managed instance, the launcher's journal-backed status: staging, applying, testing, committed, rolled back or pending recovery. Refresh-safe across the restart.",
	}, func(ctx context.Context, _ *struct{}) (*statusOutput, error) {
		out := statusOutput{Body: statusBody{
			Managed:       control != nil,
			ServerVersion: buildVersion,
			Commit:        buildCommit,
		}}
		if world != nil {
			out.Body.OnlinePlayers = world.GetPlayerCount()
		}
		if control != nil {
			state, err := control.UpdateStatus(ctx)
			if err != nil {
				return nil, huma.Error503ServiceUnavailable("Launcher status unavailable: " + err.Error())
			}
			out.Body.Update = &state
		}
		return &out, nil
	})

	type checkBody struct {
		launchctl.UpdateCheck
		Managed bool `json:"managed"`
	}
	type checkOutput struct{ Body checkBody }
	huma.Register(api, huma.Operation{
		OperationID:   "post-updates-check",
		Method:        http.MethodPost,
		Path:          "/admin/updates/check",
		Summary:       "Check for a newer release, read-only (implementor)",
		Description:   "The launcher performs the lookup; nothing is downloaded, staged or stopped. Publisher verification happens at staging time, as always.",
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, _ *struct{}) (*checkOutput, error) {
		checkMu.Lock()
		if !lastCheck.IsZero() && time.Since(lastCheck) < time.Minute {
			checkMu.Unlock()
			return nil, huma.Error429TooManyRequests("A check just ran; wait a minute before checking again")
		}
		lastCheck = time.Now()
		checkMu.Unlock()
		if control == nil {
			return nil, huma.Error503ServiceUnavailable("Update checks require a managed instance; releases are listed on GitHub")
		}
		check, err := control.CheckUpdate(ctx)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Release lookup failed: " + err.Error())
		}
		auditAdmin(ctx, logger, "update.check", "latest "+check.LatestVersion+" (current "+check.CurrentVersion+")", true)
		out := checkOutput{Body: checkBody{UpdateCheck: check, Managed: true}}
		return &out, nil
	})

	type applyBody struct {
		Status string `json:"status"`
	}
	type applyOutput struct{ Body applyBody }
	huma.Register(api, huma.Operation{
		OperationID:   "post-updates-apply",
		Method:        http.MethodPost,
		Path:          "/admin/updates/apply",
		Summary:       "Start the update transaction now (implementor)",
		Description:   "The launcher stages and verifies while the world runs, then stops, snapshots, swaps and health-checks the candidate with automatic rollback. The request returns 202 before the world stops; poll /admin/updates/status.",
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, _ *struct{}) (*applyOutput, error) {
		if control == nil {
			return nil, huma.Error503ServiceUnavailable("Updates require a managed instance; deploy with the portable launcher or your existing procedure")
		}
		online := 0
		if world != nil {
			online = world.GetPlayerCount()
		}
		if err := control.TriggerUpdate(ctx); err != nil {
			return nil, controlError(err)
		}
		auditAdmin(ctx, logger, "update.apply", "update transaction requested with "+strconv.Itoa(online)+" players online", true)
		out := &applyOutput{}
		out.Body.Status = "update requested"
		return out, nil
	})
}
