// DaemonClient implements the same interface as the HTTP Client but
// communicates with the dp-goat daemon over a Unix domain socket.
// This is the transport patch: generated CLI structure + WebSocket daemon.
package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DaemonRequest is a CLI→daemon command.
type DaemonRequest struct {
	Cmd  string         `json:"cmd"`
	Args []string       `json:"args,omitempty"`
	Body map[string]any `json:"body,omitempty"`
}

// DaemonResponse is a daemon→CLI response.
type DaemonResponse struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// DaemonClient connects to the dp-goat daemon over a Unix socket.
type DaemonClient struct {
	socketPath string
	timeout    time.Duration
}

// NewDaemonClient creates a client that talks to the daemon over Unix socket.
func NewDaemonClient(playerName string, timeout time.Duration) *DaemonClient {
	home, _ := os.UserHomeDir()
	return &DaemonClient{
		socketPath: filepath.Join(home, ".dp-goat", "sock", playerName+".sock"),
		timeout:    timeout,
	}
}

// dial connects to the Unix socket.
func (d *DaemonClient) dial() (net.Conn, error) {
	return net.DialTimeout("unix", d.socketPath, d.timeout)
}

// sendRequest sends a request to the daemon and reads the response.
func (d *DaemonClient) sendRequest(req DaemonRequest) (*DaemonResponse, error) {
	conn, err := d.dial()
	if err != nil {
		return nil, fmt.Errorf("connect to daemon: %w\nhint: is the daemon running? Start with: dp-goatd start --name <player>", err)
	}
	defer conn.Close()

	// Set deadline
	_ = conn.SetDeadline(time.Now().Add(d.timeout))

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no response from daemon")
	}

	var resp DaemonResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &resp, nil
}

// doCommand sends a command to the daemon and returns the response.
// Implements the same signature as Client.PostWithParams.
func (d *DaemonClient) doCommand(cmd string, body any) (json.RawMessage, int, error) {
	// Extract args from body (the generated CLI sends body as map[string]any)
	var args []string
	var bodyMap map[string]any
	if body != nil {
		bodyMap, _ = body.(map[string]any)
		if bodyMap == nil {
			// Try to marshal/unmarshal to get a map
			raw, _ := json.Marshal(body)
			json.Unmarshal(raw, &bodyMap)
		}
	}

	// Convert body map to args list
	if bodyMap != nil {
		for _, v := range bodyMap {
			if s, ok := v.(string); ok && s != "" {
				args = append(args, s)
			}
		}
	}

	resp, err := d.sendRequest(DaemonRequest{Cmd: cmd, Args: args})
	if err != nil {
		return nil, 500, err
	}

	if !resp.OK {
		return nil, 400, fmt.Errorf("daemon error: %s", resp.Error)
	}

	return resp.Data, 200, nil
}

// PostWithParams implements the client.Client interface.
// Path becomes the MUD command, body becomes arguments.
func (d *DaemonClient) PostWithParams(path string, params map[string]string, body any) (json.RawMessage, int, error) {
	cmd := strings.TrimPrefix(path, "/")
	return d.doCommand(cmd, body)
}

// Post implements the client.Client interface.
func (d *DaemonClient) Post(path string, body any) (json.RawMessage, int, error) {
	return d.PostWithParams(path, nil, body)
}

// Get implements the client.Transport interface.
// For MUD commands, GET and POST are the same — both send a command.
func (d *DaemonClient) Get(path string, params map[string]string) (json.RawMessage, error) {
	data, _, err := d.PostWithParams(path, params, nil)
	return data, err
}

// GetWithHeaders implements the client.Transport interface.
func (d *DaemonClient) GetWithHeaders(path string, params map[string]string, headers map[string]string) (json.RawMessage, error) {
	return d.Get(path, params)
}

// GetNoCache implements the client.Client interface.
func (d *DaemonClient) GetNoCache(path string, params map[string]string) (json.RawMessage, error) {
	return d.Get(path, params)
}

// GetWithHeadersNoCache implements the client.Client interface.
func (d *DaemonClient) GetWithHeadersNoCache(path string, params map[string]string, headers map[string]string) (json.RawMessage, error) {
	return d.Get(path, params)
}

// ProbeGet implements the client.Client interface.
func (d *DaemonClient) ProbeGet(path string) (int, error) {
	conn, err := d.dial()
	if err != nil {
		return 500, err
	}
	conn.Close()
	return 200, nil
}

// PostWithHeaders implements the client.Client interface.
func (d *DaemonClient) PostWithHeaders(path string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.Post(path, body)
}

// PostWithParamsAndHeaders implements the client.Client interface.
func (d *DaemonClient) PostWithParamsAndHeaders(path string, params map[string]string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.PostWithParams(path, params, body)
}

// Delete implements the client.Client interface.
func (d *DaemonClient) Delete(path string) (json.RawMessage, int, error) {
	return d.doCommand("delete:"+strings.TrimPrefix(path, "/"), nil)
}

// DeleteWithParams implements the client.Client interface.
func (d *DaemonClient) DeleteWithParams(path string, params map[string]string) (json.RawMessage, int, error) {
	return d.Delete(path)
}

// DeleteWithHeaders implements the client.Client interface.
func (d *DaemonClient) DeleteWithHeaders(path string, headers map[string]string) (json.RawMessage, int, error) {
	return d.Delete(path)
}

// DeleteWithParamsAndHeaders implements the client.Client interface.
func (d *DaemonClient) DeleteWithParamsAndHeaders(path string, params map[string]string, headers map[string]string) (json.RawMessage, int, error) {
	return d.Delete(path)
}

// Put implements the client.Client interface.
func (d *DaemonClient) Put(path string, body any) (json.RawMessage, int, error) {
	return d.Post(path, body)
}

// PutWithHeaders implements the client.Client interface.
func (d *DaemonClient) PutWithHeaders(path string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.Post(path, body)
}

// PutWithParamsAndHeaders implements the client.Client interface.
func (d *DaemonClient) PutWithParamsAndHeaders(path string, params map[string]string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.PostWithParams(path, params, body)
}

// Patch implements the client.Client interface.
func (d *DaemonClient) Patch(path string, body any) (json.RawMessage, int, error) {
	return d.Post(path, body)
}

// PatchWithHeaders implements the client.Client interface.
func (d *DaemonClient) PatchWithHeaders(path string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.Post(path, body)
}

// PatchWithParamsAndHeaders implements the client.Client interface.
func (d *DaemonClient) PatchWithParamsAndHeaders(path string, params map[string]string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.PostWithParams(path, params, body)
}

// RawRequest implements the client.Client interface.
func (d *DaemonClient) RawRequest(method, path string, params map[string]string, body any, headers map[string]string) (json.RawMessage, int, error) {
	return d.PostWithParams(path, params, body)
}
