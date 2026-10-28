package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/internal/launchctl"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/settings"
)

// WithSettings serves the console's configuration pages from the game-owned
// settings store. Without it the operations return 503.
func WithSettings(manager *settings.Manager) RouterOption {
	return func(c *routerConfig) { c.settings = manager }
}

// WithLaunchctl gives the console the launcher's scoped control capability
// for update and restart transactions. Without it those pages report an
// unmanaged server.
func WithLaunchctl(client *launchctl.Client) RouterOption {
	return func(c *routerConfig) { c.launchctl = client }
}

// WithBuildInfo reports the release identity on /admin/server and the
// updates page.
func WithBuildInfo(version, commit string) RouterOption {
	return func(c *routerConfig) { c.buildVersion, c.buildCommit = version, commit }
}

// WithAccountBudget supplies the login failure budget so settings saves can
// reconfigure it live. Without it the router builds one from the environment
// exactly as before.
func WithAccountBudget(tracker *auth.AccountBudgetTracker) RouterOption {
	return func(c *routerConfig) { c.accountBudget = tracker }
}

type auditView struct {
	Body []audit.AuditEvent `json:"events"`
}

// registerSettings serves GET/PUT per group, export, restore and the audit
// history. Every mount is behind requireRole("admin") — the implementor tier
// the configuration design fixed for deployment-level settings.
func registerSettings(api huma.API, logger *audit.AuditLogger, manager *settings.Manager, control *launchctl.Client, buildVersion string) {
	available := func() error {
		if manager == nil {
			return huma.Error503ServiceUnavailable("Settings unavailable on this server")
		}
		return nil
	}
	settingsError := func(err error) error {
		switch {
		case errors.Is(err, settings.ErrConflict):
			return huma.Error409Conflict("settings changed; reload before saving")
		case errors.Is(err, settings.ErrStorage):
			return huma.Error503ServiceUnavailable("Settings storage is unavailable. Repair or restore the settings file and its key.")
		default:
			return huma.Error422UnprocessableEntity(err.Error())
		}
	}

	type viewOutput struct{ Body settings.SettingsView }
	huma.Register(api, huma.Operation{
		OperationID: "get-settings",
		Method:      http.MethodGet,
		Path:        "/admin/settings",
		Summary:     "Read every settings group with restart and override metadata (implementor)",
		Description: "The game-owned settings document: session limits, access, integrations. Secret fields report presence only; meta lists restart-marked fields and environment overrides.",
	}, func(_ context.Context, _ *struct{}) (*viewOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		return &viewOutput{Body: manager.SettingsView()}, nil
	})

	save := func(result settings.SaveResult, view settings.SettingsView, err error) (*settingsSaveOutput, error) {
		if err != nil {
			return nil, settingsError(err)
		}
		return &settingsSaveOutput{Body: settingsSaveResponse{Result: result, SettingsView: view}}, nil
	}

	type sessionsInput struct {
		Body struct {
			Revision int64 `json:"revision"`
			settings.Sessions
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "put-settings-sessions",
		Method:      http.MethodPut,
		Path:        "/admin/settings/sessions",
		Summary:     "Update session and connection limits (implementor)",
		Description: "All fields apply live: the next connection sees the new limits. Revision must match the loaded one.",
	}, func(ctx context.Context, in *sessionsInput) (*settingsSaveOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		result, err := manager.SaveSessions(in.Body.Revision, in.Body.Sessions)
		if err == nil {
			auditAdmin(ctx, logger, "settings.save.sessions", settingsAuditDetails(result), true)
		}
		return save(result, manager.SettingsView(), err)
	})

	type accessInput struct {
		Body struct {
			Revision int64 `json:"revision"`
			settings.Access
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "put-settings-access",
		Method:      http.MethodPut,
		Path:        "/admin/settings/access",
		Summary:     "Update proxy trust and CORS origins (implementor)",
		Description: "CORS origins apply live; trusted proxies are read at boot, so changing them marks a restart.",
	}, func(ctx context.Context, in *accessInput) (*settingsSaveOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		result, err := manager.SaveAccess(in.Body.Revision, in.Body.Access)
		if err == nil {
			auditAdmin(ctx, logger, "settings.save.access", settingsAuditDetails(result), true)
		}
		return save(result, manager.SettingsView(), err)
	})

	type grapevineInput struct {
		Body struct {
			Revision int64 `json:"revision"`
			settings.Grapevine
			// ClientSecret is write-only: nil keeps the sealed value, an empty
			// string clears it. It is never echoed; the view reports presence.
			ClientSecret *string `json:"client_secret,omitempty"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "put-settings-grapevine",
		Method:      http.MethodPut,
		Path:        "/admin/settings/grapevine",
		Summary:     "Update the Grapevine intermud relay (implementor)",
		Description: "Every field needs a restart; the relay connects at boot. The client secret is write-only.",
	}, func(ctx context.Context, in *grapevineInput) (*settingsSaveOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		result, err := manager.SaveGrapevine(in.Body.Revision, in.Body.Grapevine, settings.SecretUpdate{settings.SecretGrapevineClient: in.Body.ClientSecret})
		if err == nil {
			auditAdmin(ctx, logger, "settings.save.grapevine", settingsAuditDetails(result), true)
		}
		return save(result, manager.SettingsView(), err)
	})

	type staffPushInput struct {
		Body struct {
			Revision int64 `json:"revision"`
			settings.StaffPush
			// Token is write-only: nil keeps, empty clears.
			Token *string `json:"token,omitempty"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "put-settings-staff-push",
		Method:      http.MethodPut,
		Path:        "/admin/settings/staff_push",
		Summary:     "Update the staff-log ntfy push (implementor)",
		Description: "Every field needs a restart; the pusher starts at boot. The token is write-only.",
	}, func(ctx context.Context, in *staffPushInput) (*settingsSaveOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		result, err := manager.SaveStaffPush(in.Body.Revision, in.Body.StaffPush, settings.SecretUpdate{settings.SecretStaffPushToken: in.Body.Token})
		if err == nil {
			auditAdmin(ctx, logger, "settings.save.staff_push", settingsAuditDetails(result), true)
		}
		return save(result, manager.SettingsView(), err)
	})

	type contactInput struct {
		Body struct {
			Revision int64 `json:"revision"`
			settings.Contact
			// SMTPPassword and TurnstileSecret are write-only: nil keeps,
			// empty clears.
			SMTPPassword    *string `json:"smtp_password,omitempty"`
			TurnstileSecret *string `json:"turnstile_secret,omitempty"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "put-settings-contact",
		Method:      http.MethodPut,
		Path:        "/admin/settings/contact",
		Summary:     "Update the contact form delivery (implementor)",
		Description: "Every field needs a restart; the form stays disabled until the whole set is present. Both secrets are write-only.",
	}, func(ctx context.Context, in *contactInput) (*settingsSaveOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		result, err := manager.SaveContact(in.Body.Revision, in.Body.Contact, settings.SecretUpdate{
			settings.SecretContactSMTP:      in.Body.SMTPPassword,
			settings.SecretContactTurnstile: in.Body.TurnstileSecret,
		})
		if err == nil {
			auditAdmin(ctx, logger, "settings.save.contact", settingsAuditDetails(result), true)
		}
		return save(result, manager.SettingsView(), err)
	})

	type exportOutput struct{ Body settings.Bundle }
	huma.Register(api, huma.Operation{
		OperationID: "get-settings-export",
		Method:      http.MethodGet,
		Path:        "/admin/settings/export",
		Summary:     "Download the settings bundle (implementor)",
		Description: "Every group with its revision and server version. Secrets are never part of an export.",
	}, func(_ context.Context, _ *struct{}) (*exportOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		return &exportOutput{Body: manager.Export(buildVersion, time.Now().UTC())}, nil
	})

	type restoreValidateInput struct {
		RawBody []byte `contentType:"application/json"`
	}
	type restoreValidateOutput struct{ Body settings.RestorePreview }
	huma.Register(api, huma.Operation{
		OperationID:      "post-settings-restore-validate",
		Method:           http.MethodPost,
		Path:             "/admin/settings/restore/validate",
		Summary:          "Validate an uploaded bundle and preview its diff (implementor)",
		Description:      "Parses and validates the bundle server-side, then reports the field-level diff, unknown keys, untouched secrets and restart requirements. Nothing is applied.",
		MaxBodyBytes:     1 << 20,
		SkipValidateBody: true,
		RequestBody:      &huma.RequestBody{Required: true, Content: map[string]*huma.MediaType{"application/json": {Schema: &huma.Schema{Type: "object"}}}},
	}, func(ctx context.Context, in *restoreValidateInput) (*restoreValidateOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		preview, err := manager.RestoreValidate(in.RawBody)
		if err != nil {
			return nil, settingsError(err)
		}
		auditAdmin(ctx, logger, "settings.restore.validate", summarizePreview(preview), true)
		return &restoreValidateOutput{Body: preview}, nil
	})

	type restoreApplyInput struct {
		Body struct {
			Revision int64           `json:"revision"`
			Bundle   json.RawMessage `json:"bundle"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID:  "post-settings-restore-apply",
		Method:       http.MethodPost,
		Path:         "/admin/settings/restore/apply",
		Summary:      "Apply a validated bundle (implementor)",
		Description:  "Replaces the bundle's groups wholesale, keeping secrets. The previewed revision must still be current.",
		MaxBodyBytes: 1 << 20,
	}, func(ctx context.Context, in *restoreApplyInput) (*settingsSaveOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		if len(in.Body.Bundle) == 0 {
			return nil, huma.Error422UnprocessableEntity("bundle is required")
		}
		result, err := manager.RestoreApply(in.Body.Revision, in.Body.Bundle)
		if err != nil {
			return nil, settingsError(err)
		}
		auditAdmin(ctx, logger, "settings.restore.apply", settingsAuditDetails(result), true)
		return &settingsSaveOutput{Body: settingsSaveResponse{Result: result, SettingsView: manager.SettingsView()}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-settings-audit",
		Method:      http.MethodGet,
		Path:        "/admin/settings/audit",
		Summary:     "Settings and update change history (implementor)",
		Description: "The settings-relevant tail of the audit log: every settings save, restore, restart and update action.",
	}, func(_ context.Context, _ *struct{}) (*auditView, error) {
		events, err := settingsAuditHistory(logger)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Audit log unavailable")
		}
		return &auditView{Body: events}, nil
	})

	type restartOutput struct {
		Body struct {
			Status string `json:"status"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID:   "post-restart",
		Method:        http.MethodPost,
		Path:          "/admin/restart",
		Summary:       "Apply pending restart-marked settings (implementor)",
		DefaultStatus: http.StatusAccepted,
		Description:   "Hands the restart to the launcher's transaction: graceful stop, fresh start, health-gated. Only on managed instances.",
	}, func(ctx context.Context, _ *struct{}) (*restartOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		if control == nil {
			return nil, huma.Error503ServiceUnavailable("Restart requires a managed instance; use your service manager")
		}
		if err := control.Restart(ctx); err != nil {
			return nil, controlError(err)
		}
		auditAdmin(ctx, logger, "settings.restart", "restart transaction requested", true)
		out := &restartOutput{}
		out.Body.Status = "restart requested"
		return out, nil
	})
}

func controlError(err error) error {
	if errors.Is(err, launchctl.ErrUnavailable) {
		return huma.Error503ServiceUnavailable("Restart requires a managed instance; use your service manager")
	}
	return huma.Error409Conflict(err.Error())
}

type settingsSaveResponse struct {
	Result       settings.SaveResult   `json:"result"`
	SettingsView settings.SettingsView `json:"view"`
}

type settingsSaveOutput struct{ Body settingsSaveResponse }

// settingsAuditDetails records what a save changed without any secret value:
// field names only, plus the revision and restart list.
func settingsAuditDetails(result settings.SaveResult) string {
	details := "revision " + strconvFormat(int(result.Revision))
	if len(result.RestartRequired) > 0 {
		details += "; restart required: " + strings.Join(result.RestartRequired, ", ")
	}
	if len(result.EnvOverridden) > 0 {
		details += "; still overridden by environment: " + strings.Join(result.EnvOverridden, ", ")
	}
	if result.Warning != "" {
		details += "; " + result.Warning
	}
	return details
}

func summarizePreview(preview settings.RestorePreview) string {
	details := "diff fields: " + strconvFormat(len(preview.Diff))
	if len(preview.UnknownKeys) > 0 {
		details += "; unknown keys ignored: " + strings.Join(preview.UnknownKeys, ", ")
	}
	return details
}

func strconvFormat(value int) string { return strconv.Itoa(value) }

// settingsAuditHistory returns the last 100 settings/updates/restart events,
// newest first, from the append-only audit log.
func settingsAuditHistory(logger *audit.AuditLogger) ([]audit.AuditEvent, error) {
	if logger == nil {
		return nil, errors.New("no audit logger")
	}
	path := logger.Filename()
	data, err := os.ReadFile(path) // #nosec G304 -- operator-configured audit path
	if err != nil {
		if os.IsNotExist(err) {
			return []audit.AuditEvent{}, nil
		}
		return nil, err
	}
	var events []audit.AuditEvent
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var event audit.AuditEvent
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		if event.EventType != "administration" {
			continue
		}
		if !strings.HasPrefix(event.Action, "settings.") && !strings.HasPrefix(event.Action, "update.") {
			continue
		}
		events = append(events, event)
	}
	if len(events) > 100 {
		events = events[len(events)-100:]
	}
	// Newest first for the console's history list.
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events, nil
}
