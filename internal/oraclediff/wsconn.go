package oraclediff

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// WSConn drives the Go port through its WebSocket endpoint by way of the real
// browser client: wsdriver/driver.mjs runs web/public/mud-client.js headless
// under Node and relays what the client writes to its terminal. The transcript
// is therefore what a /play player sees, not the raw JSON frames.
type WSConn struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	mu      sync.Mutex
	buf     bytes.Buffer
	changed chan struct{} // signalled (non-blocking) on every stdout write
	done    chan struct{} // closed when the driver's stdout ends
	stderr  bytes.Buffer

	// pendingEcho is the client's local echo of the last line sent. A telnet
	// client echoes typed input too, and the telnet transcript never holds
	// it, so it is dropped here to keep the two transports comparable. The
	// client does not echo secret input (passwords); nothing is dropped then.
	pendingEcho string
	// chrome strips what the client writes about its own socket (see
	// clientChrome); it names the URL, so it is built per connection.
	chrome *strings.Replacer
}

// NewWSConn starts driverPath under node against wsURL, loading the client at
// clientPath.
func NewWSConn(node, driverPath, clientPath, wsURL string) (*WSConn, error) {
	c := &WSConn{
		changed: make(chan struct{}, 1),
		done:    make(chan struct{}),
		chrome:  clientChrome(wsURL),
	}
	// #nosec G204 -- the harness chooses node, the driver and the client.
	c.cmd = exec.CommandContext(context.Background(), node, driverPath, clientPath, wsURL)
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c.cmd.Stderr = &lockedWriter{mu: &c.mu, w: &c.stderr}
	if err := c.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start websocket driver: %w", err)
	}
	c.stdin = stdin
	go c.pump(stdout)
	return c, nil
}

func (c *WSConn) pump(r io.Reader) {
	defer close(c.done)
	chunk := make([]byte, 4096)
	for {
		n, err := r.Read(chunk)
		if n > 0 {
			c.mu.Lock()
			c.buf.Write(chunk[:n])
			c.mu.Unlock()
			select {
			case c.changed <- struct{}{}:
			default:
			}
		}
		if err != nil {
			return
		}
	}
}

// Send types line into the client and presses Enter.
func (c *WSConn) Send(line string) error {
	c.pendingEcho = line + "\r\n"
	_, err := io.WriteString(c.stdin, line+"\n")
	return err
}

// ReadUntilQuiescent returns after the client has written nothing for d.
func (c *WSConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case <-c.changed:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(d)
		case <-c.done:
			out := c.take()
			if out == "" {
				return "", fmt.Errorf("websocket driver exited: %w\n%s", io.EOF, c.stderrText())
			}
			return out, nil
		case <-timer.C:
			return c.take(), nil
		}
	}
}

func (c *WSConn) take() string {
	c.mu.Lock()
	out := c.buf.String()
	c.buf.Reset()
	c.mu.Unlock()
	out = strings.TrimPrefix(out, c.pendingEcho)
	c.pendingEcho = ""
	return c.chrome.Replace(out)
}

// clientChrome is what mud-client.js writes about its own socket, never game
// text: a telnet client reports connecting and a dropped connection too, and
// the telnet transcript holds neither. Matched exactly, colour codes included,
// so game text that happens to share the words is untouched.
func clientChrome(wsURL string) *strings.Replacer {
	return strings.NewReplacer(
		"\x1b[2mConnecting to "+wsURL+"...\x1b[0m\r\n", "",
		"\x1b[32mConnected.\x1b[0m\r\n\r\n", "",
		"\x1b[31m\r\n--- Connection lost ---\x1b[0m\r\n", "",
	)
}

func (c *WSConn) stderrText() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stderr.String()
}

// Close ends the driver, which closes the client's WebSocket.
func (c *WSConn) Close() error {
	_ = c.stdin.Close()
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		_ = c.cmd.Process.Kill()
	}
	err := c.cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil // killed or exited non-zero after close; nothing to report
	}
	return err
}

type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
