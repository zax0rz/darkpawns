package admin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/audit"
)

// This file holds the tranche-1 Huma operations: the routes migrated off the
// plain admin mux onto generated, spec-documented operations. The router
// mounts the Huma API's mux behind the same rate-limit/CORS/role chain the
// hand-rolled handlers used, so the middleware composition is unchanged.

// liveAgentSessionsOutput is the JSON array returned by GET
// /admin/sessions/agents. It reuses LiveAgentSession so the wire shape is
// identical to the handler it replaces; a nil slice marshals to `null`, as
// before.
type liveAgentSessionsOutput struct {
	Body []LiveAgentSession
}

// registerLiveAgentSessions registers GET /admin/sessions/agents on api.
func registerLiveAgentSessions(api huma.API, provider LiveSessionProvider) {
	huma.Register(api, huma.Operation{
		OperationID: "list-live-agent-sessions",
		Method:      http.MethodGet,
		Path:        "/admin/sessions/agents",
		Summary:     "List connected game agent sessions",
		Description: "Live view of the AI agents currently connected to the game server, for the admin console.",
	}, func(ctx context.Context, in *struct{}) (*liveAgentSessionsOutput, error) {
		return &liveAgentSessionsOutput{Body: provider.GetLiveAgentSessions()}, nil
	})
}

// researchCaptureInput is the POST body for /admin/research/capture. Enabled
// is a pointer so that "field absent" is distinguishable from false; the
// handler rejects absent explicitly, matching the old handler's contract
// (a body that does not say which way is a 400, not a silent false). The
// omitempty keeps the property optional in the generated schema — required is
// enforced by the handler so the failure stays a 400, as it was on the plain
// mux, rather than a schema-validation 422.
type researchCaptureInput struct {
	Body struct {
		Enabled *bool `json:"enabled,omitempty" description:"true starts recording decision capture, false stops and flushes it"`
	}
}

// researchCaptureOutput reuses researchCaptureResponse so the wire shape —
// including records' omitempty — is identical to the old handler.
type researchCaptureOutput struct {
	Body researchCaptureResponse
}

// registerResearchCapture registers GET and POST /admin/research/capture on
// api. GET reports the capture state; POST toggles it. The audit logging and
// the response wording ("tells and says") carry over from the old handler
// unchanged.
func registerResearchCapture(api huma.API, provider LiveSessionProvider, auditLogger *audit.AuditLogger) {
	current := func() researchCaptureOutput {
		out := researchCaptureOutput{}
		out.Body.Enabled = provider.DecisionCaptureEnabled()
		out.Body.Available = provider.DecisionCaptureAvailable()
		if out.Body.Available {
			out.Body.Records = researchCaptureRecords
		}
		return out
	}

	huma.Register(api, huma.Operation{
		OperationID: "get-decision-capture",
		Method:      http.MethodGet,
		Path:        "/admin/research/capture",
		Summary:     "Get decision capture state",
		Description: "Reports whether decision capture is enabled and whether a research store is configured.",
	}, func(ctx context.Context, in *struct{}) (*researchCaptureOutput, error) {
		out := current()
		return &out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "set-decision-capture",
		Method:      http.MethodPost,
		Path:        "/admin/research/capture",
		Summary:     "Enable or disable decision capture",
		Description: "Starts or stops recording of command text including tells and says. Enabling without a configured research store fails with 409.",
	}, func(ctx context.Context, in *researchCaptureInput) (*researchCaptureOutput, error) {
		if in.Body.Enabled == nil {
			return nil, huma.Error400BadRequest(`body must be {"enabled": true|false}`)
		}
		if *in.Body.Enabled {
			if !provider.EnableDecisionCapture() {
				return nil, huma.NewError(http.StatusConflict,
					"no research store configured; set DP_RESEARCH_URL and restart")
			}
		} else {
			// Flushes what is buffered; see Manager.DisableDecisionCapture.
			provider.DisableDecisionCapture()
		}
		if auditLogger != nil {
			auditLogger.Log(audit.AuditEvent{
				EventType: "research",
				Action:    "decision_capture",
				Success:   true,
				Details:   fmt.Sprintf("enabled=%t records=%q", *in.Body.Enabled, researchCaptureRecords),
			})
		}
		slog.Info("decision capture toggled", "enabled", *in.Body.Enabled, "records", researchCaptureRecords)
		out := current()
		return &out, nil
	})
}
