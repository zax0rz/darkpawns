package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/notify"
)

// WithNotifications supplies the same service used by the live game hooks.
func WithNotifications(service *notify.Service) RouterOption {
	return func(c *routerConfig) { c.notifications = service }
}

func registerNotifications(api huma.API, database *db.DB, logger *audit.AuditLogger, service *notify.Service) {
	gate := staffAdminGate(database, "Notification settings require HIGOD")
	type output struct{ Body notify.View }
	type saveInput struct{ Body notify.Config }
	type testInput struct {
		Provider string `path:"provider" enum:"pushover,discord"`
	}
	available := func() error {
		if service == nil {
			return huma.Error503ServiceUnavailable("Notifications unavailable on this server")
		}
		return nil
	}
	result := func() *output { return &output{Body: service.View()} }
	huma.Register(api, huma.Operation{OperationID: "get-notifications", Method: http.MethodGet, Path: "/admin/notifications", Summary: "Read notification settings and delivery status (HIGOD)", Middlewares: huma.Middlewares{gate}}, func(_ context.Context, _ *struct{}) (*output, error) {
		if err := available(); err != nil {
			return nil, err
		}
		return result(), nil
	})
	huma.Register(api, huma.Operation{OperationID: "save-notifications", Method: http.MethodPut, Path: "/admin/notifications", Summary: "Configure opt-in event subscriptions (HIGOD)", Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *saveInput) (*output, error) {
		if err := available(); err != nil {
			return nil, err
		}
		if err := service.Save(in.Body); err != nil {
			return nil, notificationError(err)
		}
		auditAdmin(ctx, logger, "notifications.save", "event subscriptions updated", true)
		return result(), nil
	})
	type testOutput struct {
		Body struct {
			Status string `json:"status"`
		}
	}
	huma.Register(api, huma.Operation{OperationID: "test-notification", Method: http.MethodPost, Path: "/admin/notifications/{provider}/test", Summary: "Queue a test to a saved destination (HIGOD)", DefaultStatus: 202, Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *testInput) (*testOutput, error) {
		if err := available(); err != nil {
			return nil, err
		}
		if err := service.Test(in.Provider); err != nil {
			return nil, notificationError(err)
		}
		auditAdmin(ctx, logger, "notifications.test", in.Provider, true)
		out := &testOutput{}
		out.Body.Status = "queued"
		return out, nil
	})
	// RawBody plus explicit schema retains Huma/OpenAPI documentation while
	// validating secrets ourselves: generic schema errors may echo bad input values.
	type credentialInput struct {
		Provider string `path:"provider" enum:"pushover,discord,heartbeat"`
		RawBody  []byte `contentType:"application/json"`
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		operationID := "replace-notification-credentials"
		summary := "Save encrypted destination credentials without returning secrets (HIGOD)"
		if method == http.MethodDelete {
			operationID = "remove-notification-credentials"
			summary = "Remove credentials and disable destination (HIGOD)"
		}
		schema := &huma.Schema{Type: "object", Required: []string{"revision"}, Properties: map[string]*huma.Schema{
			"revision": {Type: "integer", Format: "int64"},
		}}
		if method == http.MethodPut {
			for _, key := range []string{"token", "user", "url"} {
				schema.Properties[key] = &huma.Schema{Type: "string", WriteOnly: true}
			}
		}
		huma.Register(api, huma.Operation{OperationID: operationID, Method: method, Path: "/admin/notifications/{provider}/credentials", Summary: summary, MaxBodyBytes: 4096, SkipValidateBody: true, RequestBody: &huma.RequestBody{Required: true, Content: map[string]*huma.MediaType{"application/json": {Schema: schema}}}, Middlewares: huma.Middlewares{gate}}, func(ctx context.Context, in *credentialInput) (*output, error) {
			if err := available(); err != nil {
				return nil, err
			}
			var input struct {
				Revision int64  `json:"revision"`
				Token    string `json:"token"`
				User     string `json:"user"`
				URL      string `json:"url"`
			}
			decoder := json.NewDecoder(bytes.NewReader(in.RawBody))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&input) != nil || decoder.Decode(&struct{}{}) != io.EOF || input.Revision < 1 {
				return nil, huma.Error422UnprocessableEntity("Invalid credential request")
			}
			var credentials *notify.Credentials
			if method == http.MethodPut {
				credentials = &notify.Credentials{Token: input.Token, User: input.User, URL: input.URL}
			}
			if err := service.SaveCredentials(in.Provider, input.Revision, credentials); err != nil {
				return nil, notificationError(err)
			}
			action := "notifications.credentials.replace"
			if credentials == nil {
				action = "notifications.credentials.remove"
			}
			auditAdmin(ctx, logger, action, in.Provider, true)
			return result(), nil
		})
		// RawBody registration assigns a binary schema; restore the explicit
		// write-only JSON contract after Huma has installed the operation.
		item := api.OpenAPI().Paths["/admin/notifications/{provider}/credentials"]
		if method == http.MethodPut {
			item.Put.RequestBody.Content["application/json"].Schema = schema
		} else {
			item.Delete.RequestBody.Content["application/json"].Schema = schema
		}

	}
}

func notificationError(err error) error {
	switch {
	case errors.Is(err, notify.ErrConflict):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, notify.ErrNotReady), errors.Is(err, notify.ErrInvalid), errors.Is(err, notify.ErrCredentials):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, notify.ErrCredentialStorage):
		return huma.Error503ServiceUnavailable("Credential storage unavailable. Check the instance encryption key and permissions.")
	case errors.Is(err, notify.ErrBusy):
		return huma.Error429TooManyRequests(err.Error())
	default:
		return huma.Error500InternalServerError("Could not save notification settings")
	}
}
