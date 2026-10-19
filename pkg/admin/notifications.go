package admin

import (
	"context"
	"errors"
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
}

func notificationError(err error) error {
	switch {
	case errors.Is(err, notify.ErrConflict):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, notify.ErrNotReady), errors.Is(err, notify.ErrInvalid):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, notify.ErrBusy):
		return huma.Error429TooManyRequests(err.Error())
	default:
		return huma.Error500InternalServerError("Could not save notification settings")
	}
}
