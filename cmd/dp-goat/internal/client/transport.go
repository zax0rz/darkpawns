package client

import "encoding/json"

// Transport is the interface both DaemonClient and *Client satisfy.
// The generated CLI calls newClient() which returns Transport.
type Transport interface {
	PostWithParams(path string, params map[string]string, body any) (json.RawMessage, int, error)
	Post(path string, body any) (json.RawMessage, int, error)
	Get(path string, params map[string]string) (json.RawMessage, error)
	GetWithHeaders(path string, params map[string]string, headers map[string]string) (json.RawMessage, error)
}
