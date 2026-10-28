package apidoc

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

type ConnectionListener struct {
	Bind string `json:"bind"`
	Port int    `json:"port"`
}

type ConnectionInfo struct {
	Telnet    *ConnectionListener `json:"telnet,omitempty"`
	TelnetTLS *ConnectionListener `json:"telnet_tls,omitempty"`
	Whod      *ConnectionListener `json:"whod,omitempty"`
}

// ConnectionAt accepts only an observed listener address; no requested defaults
// are substituted when startup failed or a listener is disabled.
func ConnectionAt(address string) *ConnectionListener {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return nil
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return nil
	}
	return &ConnectionListener{Bind: host, Port: port}
}

func RegisterConnections(api huma.API, observed func() ConnectionInfo) {
	huma.Register(api, huma.Operation{OperationID: "get-connections", Method: http.MethodGet, Path: "/connection-info", Summary: "Active game listeners", Description: "Public listener addresses for this server's connection page. Disabled or failed listeners are omitted."}, func(_ context.Context, _ *struct{}) (*struct{ Body ConnectionInfo }, error) {
		return &struct{ Body ConnectionInfo }{Body: observed()}, nil
	})
}
