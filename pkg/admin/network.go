package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/internal/launchctl"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// The Network card (docs/design/admin-network-settings.md, Zach's 2026-10-28
// decisions): v1 edits plaintext telnet, TLS telnet and WHOD; HTTP stays
// read-only so the management route never moves under the operator. Saving
// is a draft through the launcher's typed control channel; applying runs
// the launcher's restart transaction, which restores the previous
// configuration automatically when a requested listener fails to bind.

// WithNetworkObserved supplies configured-and-observed listener data the
// game process knows from its own flags and sockets. The console shows the
// two separately: a failed bind must never appear as a running listener.
func WithNetworkObserved(observed func() NetworkObserved) RouterOption {
	return func(c *routerConfig) { c.networkObserved = observed }
}

// NetworkObserved is the game's own view of its listeners.
type NetworkObserved struct {
	HTTPPort  int
	HTTPBind  string
	Telnet    string // observed "host:port" or ""
	TelnetTLS string
	Whod      string
}

type networkListenerResponse struct {
	Enabled  bool   `json:"enabled"`
	Port     int    `json:"port"`
	Scope    string `json:"scope"`
	Address  string `json:"address,omitempty"`
	Observed string `json:"observed,omitempty"`
}

type networkView struct {
	Managed                  bool                    `json:"managed"`
	Revision                 int64                   `json:"revision"`
	HTTP                     networkHTTPResponse     `json:"http"`
	TLSCertificateConfigured bool                    `json:"tls_certificate_configured"`
	Telnet                   networkListenerResponse `json:"telnet"`
	TelnetTLS                networkListenerResponse `json:"telnet_tls"`
	Whod                     networkListenerResponse `json:"whod"`
}

type networkHTTPResponse struct {
	Port        int    `json:"port"`
	Bind        string `json:"bind"`
	ReadOnly    bool   `json:"read_only"`
	HowToChange string `json:"how_to_change"`
}

type networkSaveInput struct {
	Body struct {
		Revision  int64                     `json:"revision"`
		Telnet    networkListenerChangeBody `json:"telnet"`
		TelnetTLS networkListenerChangeBody `json:"telnet_tls"`
		Whod      networkListenerChangeBody `json:"whod"`
	}
}

// The apply transaction is bound to the revision the operator reviewed: a
// save by anyone else makes the apply conflict instead of silently running
// (or rolling back) onto a configuration nobody approved.
type networkApplyInput struct {
	Body struct {
		Revision int64 `json:"revision" doc:"The configuration revision the operator reviewed and is approving"`
	}
}

type networkListenerChangeBody struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
	Public  bool `json:"public"`
}

type networkSaveOutput struct{ Body networkSaveResponse }

type networkSaveResponse struct {
	Config          networkConfigResponse `json:"config"`
	Diff            []networkDiffResponse `json:"diff"`
	Warnings        []string              `json:"warnings"`
	RestartRequired bool                  `json:"restart_required"`
}

type networkConfigResponse struct {
	Revision int64 `json:"revision"`
}

type networkDiffResponse struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type networkApplyOutput struct {
	Body struct {
		Status string `json:"status"`
	}
}

type networkStatusOutput struct {
	Body networkStatusResponse
}

type networkStatusResponse struct {
	Managed        bool   `json:"managed"`
	State          string `json:"state"`
	Error          string `json:"error,omitempty"`
	FailedListener string `json:"failed_listener,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
}

// registerNetwork serves the console's Network card. The implementor tier
// (role admin, level 40) matches the configuration design: these controls
// expose sockets and restart the entire world.
func registerNetwork(api huma.API, logger *audit.AuditLogger, control *launchctl.Client, observed func() NetworkObserved, world *game.World) {
	nonManaged := huma.Error503ServiceUnavailable("Network settings need a managed instance. On a checkout deployment, stop the server and change the -telnet-port, -telnet-bind and -whod flags (or instance.json on a portable instance)")

	type viewOutput struct{ Body networkView }
	huma.Register(api, huma.Operation{
		OperationID: "get-network",
		Method:      http.MethodGet,
		Path:        "/admin/network",
		Summary:     "Read configured and observed listeners (implementor)",
		Description: "Configured values come from the launcher's instance configuration on a managed instance; observed values are the sockets the game actually bound. HTTP is read-only: it is the management route itself.",
	}, func(ctx context.Context, _ *struct{}) (*viewOutput, error) {
		view := networkView{Managed: control != nil}
		if observed != nil {
			local := observed()
			view.HTTP = networkHTTPResponse{
				Port: local.HTTPPort, Bind: local.HTTPBind, ReadOnly: true,
				HowToChange: "Read only. Change it by stopping the server and editing http_port in instance.json, or the -port and -http-bind command-line flags.",
			}
			view.Telnet.Observed = local.Telnet
			view.TelnetTLS.Observed = local.TelnetTLS
			view.Whod.Observed = local.Whod
		}
		if control != nil {
			config, err := control.GetNetworkConfig(ctx)
			if err != nil {
				return nil, huma.Error503ServiceUnavailable("Launcher configuration unavailable: " + err.Error())
			}
			view.Revision = config.Revision
			view.TLSCertificateConfigured = config.TLSCertificateConfigured
			view.Telnet.Enabled, view.Telnet.Port, view.Telnet.Scope, view.Telnet.Address = config.Telnet.Enabled, config.Telnet.Port, config.Telnet.Scope, config.Telnet.Address
			view.TelnetTLS.Enabled, view.TelnetTLS.Port, view.TelnetTLS.Scope, view.TelnetTLS.Address = config.TelnetTLS.Enabled, config.TelnetTLS.Port, config.TelnetTLS.Scope, config.TelnetTLS.Address
			view.Whod.Enabled, view.Whod.Port, view.Whod.Scope, view.Whod.Address = config.Whod.Enabled, config.Whod.Port, config.Whod.Scope, config.Whod.Address
			view.HTTP.Port = config.HTTPPort
		}
		return &viewOutput{Body: view}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "put-network",
		Method:      http.MethodPut,
		Path:        "/admin/network",
		Summary:     "Save a network change as a draft (implementor)",
		Description: "Validates enable, port and exposure for plaintext telnet, TLS telnet and WHOD. Saving changes nothing live: the response carries the exposure diff and warnings, and the change becomes real only through the apply-and-restart transaction.",
	}, func(ctx context.Context, in *networkSaveInput) (*networkSaveOutput, error) {
		if control == nil {
			return nil, nonManaged
		}
		proposal := launchctl.NetworkProposal{
			Revision:  in.Body.Revision,
			Telnet:    launchctl.ListenerChange{Enabled: in.Body.Telnet.Enabled, Port: in.Body.Telnet.Port, Public: in.Body.Telnet.Public},
			TelnetTLS: launchctl.ListenerChange{Enabled: in.Body.TelnetTLS.Enabled, Port: in.Body.TelnetTLS.Port, Public: in.Body.TelnetTLS.Public},
			Whod:      launchctl.ListenerChange{Enabled: in.Body.Whod.Enabled, Port: in.Body.Whod.Port, Public: in.Body.Whod.Public},
		}
		result, err := control.SaveNetworkConfig(ctx, proposal)
		if err != nil {
			return nil, networkError(err)
		}
		details := networkAuditDetails(result)
		auditAdmin(ctx, logger, "network.save", details, true)
		out := &networkSaveOutput{}
		out.Body.RestartRequired = result.RestartRequired
		out.Body.Config.Revision = result.Config.Revision
		// Never null: an empty diff and an empty warning list are valid
		// states the card renders directly.
		out.Body.Warnings = []string{}
		out.Body.Warnings = append(out.Body.Warnings, result.Warnings...)
		out.Body.Diff = []networkDiffResponse{}
		for _, entry := range result.Diff {
			out.Body.Diff = append(out.Body.Diff, networkDiffResponse{Field: entry.Field, From: entry.From, To: entry.To})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "post-network-apply",
		Method:        http.MethodPost,
		Path:          "/admin/network/apply",
		Summary:       "Apply the saved network configuration and restart (implementor)",
		DefaultStatus: http.StatusAccepted,
		Description:   "The launcher stops the world gracefully, starts it on the saved configuration, and verifies every requested listener bound. A failed bind restores the previous configuration automatically. The apply is bound to the revision the operator reviewed: if the configuration changed since, the request conflicts and nothing restarts. The request returns before the world stops; poll /admin/network/status.",
	}, func(ctx context.Context, in *networkApplyInput) (*networkApplyOutput, error) {
		if control == nil {
			return nil, nonManaged
		}
		online := 0
		if world != nil {
			online = world.GetPlayerCount()
		}
		if err := control.ApplyNetworkConfig(ctx, in.Body.Revision); err != nil {
			if errors.Is(err, launchctl.ErrConfigConflict) {
				return nil, huma.Error409Conflict("configuration changed; reload before applying")
			}
			return nil, networkError(err)
		}
		auditAdmin(ctx, logger, "network.apply", "revision "+strconv.FormatInt(in.Body.Revision, 10)+"; "+strconv.Itoa(online)+" players online at apply", true)
		out := &networkApplyOutput{}
		out.Body.Status = "apply requested"
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-network-status",
		Method:      http.MethodGet,
		Path:        "/admin/network/status",
		Summary:     "Network apply transaction status (implementor)",
		Description: "The launcher's receipt: applying, verifying, committed or rolled-back. Survives the restart; refresh-safe.",
	}, func(ctx context.Context, _ *struct{}) (*networkStatusOutput, error) {
		out := &networkStatusOutput{}
		out.Body.Managed = control != nil
		if control == nil {
			out.Body.State = "unmanaged"
			return out, nil
		}
		status, err := control.ConfigApplyStatus(ctx)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Launcher status unavailable: " + err.Error())
		}
		out.Body.State = status.State
		out.Body.Error = status.Error
		out.Body.FailedListener = status.FailedListener
		out.Body.StartedAt = status.StartedAt
		return out, nil
	})
}

func networkError(err error) error {
	switch {
	case errors.Is(err, launchctl.ErrUnavailable):
		return huma.Error503ServiceUnavailable("Network settings need a managed instance")
	case errors.Is(err, launchctl.ErrConfigConflict):
		return huma.Error409Conflict("configuration changed; reload before saving")
	default:
		return huma.Error422UnprocessableEntity(err.Error())
	}
}

func networkAuditDetails(result launchctl.SaveNetworkResult) string {
	details := "revision " + strconv.Itoa(int(result.Config.Revision))
	for _, entry := range result.Diff {
		details += "; " + entry.Field + ": " + entry.From + " -> " + entry.To
	}
	return details
}
