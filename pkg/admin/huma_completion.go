package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func registerLoginOperation(api huma.API, database loginPlayerDB, attempts *auth.LoginAttemptTracker) {
	type input struct{ Body loginRequest }
	type output struct{ Body loginResponse }
	huma.Register(api, huma.Operation{OperationID: "admin-login", Method: http.MethodPost, Path: "/admin/login", Summary: "Authenticate to the admin console"},
		func(ctx context.Context, in *input) (*output, error) {
			body, err := json.Marshal(in.Body)
			if err != nil {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid json"}`)
			}
			rec := runLegacy(ctx, handleLogin(database, attempts), http.MethodPost, "/admin/login", body)
			if rec.Code != http.StatusOK {
				return nil, recorderError(rec)
			}
			var response loginResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				return nil, err
			}
			return &output{Body: response}, nil
		})
}

func runLegacy(ctx context.Context, handler http.HandlerFunc, method, target string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader(body)).WithContext(ctx)
	req.RemoteAddr = clientIPFrom(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func recorderError(rec *httptest.ResponseRecorder) error {
	body := strings.TrimSuffix(rec.Body.String(), "\n")
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		return apidoc.NewPlainError(rec.Code, body)
	}
	var value any
	if json.Unmarshal(rec.Body.Bytes(), &value) != nil {
		value = body
	}
	return &jsonResponseError{status: rec.Code, body: value}
}

type (
	agentListOutput   struct{ Body []*AgentStatus }
	agentOutput       struct{ Body *AgentStatus }
	findingListOutput struct{ Body []Finding }
	findingOutput     struct {
		Status int `status:"201"`
		Body   Finding
	}
)

type (
	findingUpdateOutput struct{ Body *Finding }
	triageListOutput    struct{ Body []TriageSummary }
	triageOutput        struct {
		Status int `status:"201"`
		Body   TriageSummary
	}
)

func registerAgentStoreCompletion(api huma.API, store *AgentStore) {
	huma.Register(api, huma.Operation{OperationID: "list-agents", Method: http.MethodGet, Path: "/admin/agents", Summary: "List research agents"},
		func(ctx context.Context, in *struct{}) (*agentListOutput, error) {
			return &agentListOutput{Body: store.GetAgents()}, nil
		})
	type statusInput struct {
		Body struct {
			AgentID string `json:"agent_id"`
			Status  string `json:"status"`
		}
	}
	huma.Register(api, huma.Operation{OperationID: "update-agent-status", Method: http.MethodPost, Path: "/admin/agents/status", Summary: "Update an agent status"},
		func(ctx context.Context, in *statusInput) (*agentOutput, error) {
			if in.Body.AgentID == "" || in.Body.Status == "" {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"agent_id and status are required"}`)
			}
			a, ok, err := store.UpdateAgentStatus(in.Body.AgentID, in.Body.Status)
			if err != nil {
				slog.Error("admin agent status save failed", "error", err)
				return nil, apidoc.NewPlainError(http.StatusInternalServerError, `{"error":"failed to persist agent status"}`)
			}
			if !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"agent not found"}`)
			}
			return &agentOutput{Body: a}, nil
		})
	type findingsInput struct {
		Status   string `query:"status"`
		Severity string `query:"severity"`
		Source   string `query:"source"`
	}
	huma.Register(api, huma.Operation{OperationID: "list-findings", Method: http.MethodGet, Path: "/admin/findings", Summary: "List agent findings"},
		func(ctx context.Context, in *findingsInput) (*findingListOutput, error) {
			return &findingListOutput{Body: store.GetFindings(in.Status, in.Severity, in.Source)}, nil
		})
	type createFindingInput struct {
		Body struct {
			Source      string `json:"source"`
			Severity    string `json:"severity"`
			Title       string `json:"title"`
			File        string `json:"file"`
			Line        int    `json:"line"`
			Description string `json:"description"`
		}
	}
	huma.Register(api, huma.Operation{OperationID: "create-finding", Method: http.MethodPost, Path: "/admin/findings", Summary: "Create an agent finding"},
		func(ctx context.Context, in *createFindingInput) (*findingOutput, error) {
			if in.Body.Source == "" || in.Body.Severity == "" || in.Body.Title == "" {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"source, severity, and title are required"}`)
			}
			f, err := store.AddFinding(in.Body.Source, in.Body.Severity, in.Body.Title, in.Body.File, in.Body.Line, in.Body.Description)
			if err != nil {
				slog.Error("admin finding save failed", "error", err)
				return nil, apidoc.NewPlainError(http.StatusInternalServerError, `{"error":"failed to persist finding"}`)
			}
			return &findingOutput{Status: http.StatusCreated, Body: f}, nil
		})
	type findingIDInput struct {
		ID   string `path:"id"`
		Body struct {
			Status        string `json:"status"`
			LinearIssueID string `json:"linear_issue_id"`
		}
	}
	huma.Register(api, huma.Operation{OperationID: "update-finding", Method: http.MethodPut, Path: "/admin/findings/{id}", Summary: "Update an agent finding"},
		func(ctx context.Context, in *findingIDInput) (*findingUpdateOutput, error) {
			id, err := strconv.Atoi(in.ID)
			if err != nil {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid finding id"}`)
			}
			if in.Body.Status == "" && in.Body.LinearIssueID == "" {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"status or linear_issue_id is required"}`)
			}
			var f *Finding
			var ok bool
			if in.Body.LinearIssueID != "" {
				f, ok, err = store.UpdateFinding(id, in.Body.Status, in.Body.LinearIssueID)
			} else {
				f, ok, err = store.UpdateFindingStatus(id, in.Body.Status)
			}
			if err != nil {
				slog.Error("admin finding update save failed", "error", err)
				return nil, apidoc.NewPlainError(http.StatusInternalServerError, `{"error":"failed to persist finding update"}`)
			}
			if !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"finding not found"}`)
			}
			return &findingUpdateOutput{Body: f}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "list-triage-summaries", Method: http.MethodGet, Path: "/admin/triage/summaries", Summary: "List triage summaries"},
		func(ctx context.Context, in *struct{}) (*triageListOutput, error) {
			return &triageListOutput{Body: store.GetTriageSummaries()}, nil
		})
	type triageInput struct {
		Body struct {
			Date      string `json:"date"`
			Confirmed int    `json:"confirmed"`
			Rejected  int    `json:"rejected"`
			Pending   int    `json:"pending"`
			Summary   string `json:"summary"`
		}
	}
	huma.Register(api, huma.Operation{OperationID: "create-triage-summary", Method: http.MethodPost, Path: "/admin/triage/summaries", Summary: "Create a triage summary"},
		func(ctx context.Context, in *triageInput) (*triageOutput, error) {
			if in.Body.Date == "" {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"date is required"}`)
			}
			s, err := store.AddTriageSummary(in.Body.Date, in.Body.Summary, in.Body.Confirmed, in.Body.Rejected, in.Body.Pending)
			if err != nil {
				slog.Error("admin triage summary save failed", "error", err)
				return nil, apidoc.NewPlainError(http.StatusInternalServerError, `{"error":"failed to persist triage summary"}`)
			}
			return &triageOutput{Status: http.StatusCreated, Body: s}, nil
		})
}

type zoneNumberInput struct {
	Number string `path:"number"`
}

type (
	zoneOutput   struct{ Body zoneResponse }
	statusOutput struct {
		Body struct {
			Status string `json:"status"`
		}
	}
)

func registerWorldCompletion(api huma.API, world *game.World, auditLogger *audit.AuditLogger) {
	huma.Register(api, huma.Operation{OperationID: "get-zone", Method: http.MethodGet, Path: "/admin/zones/{number}", Summary: "Get one zone"},
		func(ctx context.Context, in *zoneNumberInput) (*zoneOutput, error) {
			n, err := strconv.Atoi(in.Number)
			if err != nil {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid zone number"}`)
			}
			zone, ok := world.GetZone(n)
			if !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"zone not found"}`)
			}
			return &zoneOutput{Body: zoneResponse{Number: zone.Number, Name: zone.Name, TopRoom: zone.TopRoom, Lifespan: zone.Lifespan, ResetMode: zone.ResetMode}}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "reset-zone-placeholder", Method: http.MethodPost, Path: "/admin/zones/reset", Summary: "Legacy zone reset trigger"},
		func(ctx context.Context, in *struct{}) (*struct{}, error) {
			return nil, &jsonResponseError{status: http.StatusNotImplemented, body: map[string]string{"error": "not implemented", "message": "Zone reset trigger is not yet wired to the zone dispatcher"}}
		})

	huma.Register(api, huma.Operation{OperationID: "reset-zone", Method: http.MethodPost, Path: "/admin/zones/{number}/reset", Summary: "Reset one zone"},
		func(ctx context.Context, in *zoneNumberInput) (*statusOutput, error) {
			n, err := strconv.Atoi(in.Number)
			if err != nil {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid zone number"}`)
			}
			if err := world.ResetZone(n); err != nil {
				return nil, apidoc.NewPlainError(http.StatusInternalServerError, `{"error":"`+err.Error()+`"}`)
			}
			auditAdmin(ctx, auditLogger, "admin_zone_reset", fmt.Sprintf("manual reset zone %d", n), true)
			out := &statusOutput{}
			out.Body.Status = "reset triggered"
			return out, nil
		})

	type playerInput struct {
		Name string `path:"name"`
	}
	type playerOutput struct{ Body playerDetailResponse }
	huma.Register(api, huma.Operation{OperationID: "get-player", Method: http.MethodGet, Path: "/admin/players/{name}", Summary: "Get one online player"},
		func(ctx context.Context, in *playerInput) (*playerOutput, error) {
			p, ok := world.GetPlayer(in.Name)
			if !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"player not found"}`)
			}
			return &playerOutput{Body: playerDetailToResponse(p)}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "save-player", Method: http.MethodPost, Path: "/admin/players/{name}/save", Summary: "Force-save one online player"},
		func(ctx context.Context, in *playerInput) (*statusOutput, error) {
			p, ok := world.GetPlayer(in.Name)
			if !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"player not found"}`)
			}
			if err := game.SavePlayer(p); err != nil {
				slog.Error("admin player save failed", "name", in.Name, "error", err)
				return nil, &jsonResponseError{status: http.StatusInternalServerError, body: map[string]string{"error": fmt.Sprintf("save failed: %v", err)}}
			}
			auditAdmin(ctx, auditLogger, "admin_force_save", fmt.Sprintf("forced save player %s", in.Name), true)
			out := &statusOutput{}
			out.Body.Status = "saved"
			return out, nil
		})
	huma.Register(api, huma.Operation{OperationID: "kick-player", Method: http.MethodPost, Path: "/admin/players/{name}/kick", Summary: "Disconnect one online player"},
		func(ctx context.Context, in *playerInput) (*struct{}, error) {
			if _, ok := world.GetPlayer(in.Name); !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"player not found"}`)
			}
			return nil, &jsonResponseError{status: http.StatusNotImplemented, body: map[string]string{"error": "kick not yet implemented"}}
		})

	type shopInput struct {
		Keeper string `path:"keeper"`
	}
	type shopOutput struct{ Body shopResponse }
	huma.Register(api, huma.Operation{OperationID: "get-shop", Method: http.MethodGet, Path: "/admin/shops/{keeper}", Summary: "Get a shop by keeper vnum"},
		func(ctx context.Context, in *shopInput) (*shopOutput, error) {
			n, err := strconv.Atoi(in.Keeper)
			if err != nil {
				return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid vnum"}`)
			}
			s, ok := world.GetShopByKeeper(n)
			if !ok {
				return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"shop not found"}`)
			}
			return &shopOutput{Body: shopResponse{VNum: s.VNum, KeeperVNum: s.KeeperVNum, BuyTypes: s.BuyTypes, SellTypes: s.SellTypes, ProfitBuy: s.ProfitBuy, ProfitSell: s.ProfitSell, KeeperName: s.KeeperName, RoomVNum: s.RoomVNum}}, nil
		})

	huma.Register(api, huma.Operation{OperationID: "save-world", Method: http.MethodPost, Path: "/admin/save-world", Summary: "Save world state"},
		func(ctx context.Context, in *struct{}) (*statusOutput, error) {
			if err := game.SaveWorld(world); err != nil {
				slog.Error("admin save world failed", "error", err)
				return nil, &jsonResponseError{status: http.StatusInternalServerError, body: map[string]string{"error": fmt.Sprintf("save failed: %v", err)}}
			}
			auditAdmin(ctx, auditLogger, "admin_save_world", "saved world state", true)
			out := &statusOutput{}
			out.Body.Status = "saved"
			return out, nil
		})

	type resetAllOutput struct {
		Body struct {
			Errors     []string `json:"errors,omitempty"`
			Status     string   `json:"status"`
			ZonesReset int      `json:"zones_reset"`
			ZonesTotal int      `json:"zones_total"`
		}
	}
	huma.Register(api, huma.Operation{OperationID: "reset-all-zones", Method: http.MethodPost, Path: "/admin/reset-all-zones", Summary: "Reset every zone"},
		func(ctx context.Context, in *struct{}) (*resetAllOutput, error) {
			zones := world.GetAllZones()
			out := &resetAllOutput{}
			out.Body.Status = "reset triggered"
			out.Body.ZonesTotal = len(zones)
			for _, z := range zones {
				if err := world.ResetZone(z.Number); err != nil {
					out.Body.Errors = append(out.Body.Errors, fmt.Sprintf("zone %d: %v", z.Number, err))
				} else {
					out.Body.ZonesReset++
				}
			}
			auditAdmin(ctx, auditLogger, "admin_reset_all_zones", fmt.Sprintf("reset %d zones, %d errors", out.Body.ZonesReset, len(out.Body.Errors)), len(out.Body.Errors) == 0)
			return out, nil
		})
}

func auditAdmin(ctx context.Context, logger *audit.AuditLogger, action, details string, success bool) {
	if logger == nil {
		return
	}
	name := ""
	if claims, ok := auth.GetClaimsFromContext(ctx); ok {
		name = claims.PlayerName
	}
	logger.Log(audit.AuditEvent{IPAddress: clientIPFrom(ctx), EventType: "administration", User: name, Action: action, Details: details, Success: success})
}

// jsonResponseError preserves the application/json error bodies emitted by
// legacy handlers which used json.Encoder instead of http.Error.
type jsonResponseError struct {
	status int
	body   any
}

func (e *jsonResponseError) Error() string                { return fmt.Sprint(e.body) }
func (e *jsonResponseError) GetStatus() int               { return e.status }
func (e *jsonResponseError) ContentType(string) string    { return "application/json" }
func (e *jsonResponseError) GetHeaders() http.Header      { return nil }
func (e *jsonResponseError) MarshalJSON() ([]byte, error) { return json.Marshal(e.body) }
