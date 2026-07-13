package oraclediff

import (
	"bufio"
	"errors"
	"io"
	"net"
	"time"
)

// Conn is the transport surface used by the scenario driver.
type Conn interface {
	Send(line string) error
	ReadUntilQuiescent(d time.Duration) (string, error)
	Close() error
}

// TCPConn drives either server over its telnet TCP listener.
type TCPConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewTCPConn(conn net.Conn) *TCPConn {
	return &TCPConn{conn: conn, reader: bufio.NewReader(conn)}
}

func (c *TCPConn) Send(line string) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	_, err := io.WriteString(c.conn, line+"\r\n")
	return err
}

// ReadUntilQuiescent returns after the server has emitted no bytes for d. Telnet
// negotiation is transport framing, so it is consumed here and never reaches
// the transcript normalizer.
func (c *TCPConn) ReadUntilQuiescent(d time.Duration) (string, error) {
	var out []byte
	for {
		if err := c.conn.SetReadDeadline(time.Now().Add(d)); err != nil {
			return string(out), err
		}
		b, err := c.reader.ReadByte()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return string(out), nil
			}
			if errors.Is(err, io.EOF) && len(out) > 0 {
				return string(out), nil
			}
			return string(out), err
		}
		if b == 255 { // IAC
			if err := consumeIAC(c.reader); err != nil {
				return string(out), err
			}
			continue
		}
		out = append(out, b)
	}
}

func (c *TCPConn) Close() error {
	return c.conn.Close()
}

func consumeIAC(r *bufio.Reader) error {
	cmd, err := r.ReadByte()
	if err != nil {
		return err
	}
	switch cmd {
	case 251, 252, 253, 254: // WILL, WONT, DO, DONT
		_, err = r.ReadByte()
		return err
	case 250: // SB ... IAC SE
		for {
			b, readErr := r.ReadByte()
			if readErr != nil {
				return readErr
			}
			if b != 255 {
				continue
			}
			next, readErr := r.ReadByte()
			if readErr != nil {
				return readErr
			}
			if next == 240 {
				return nil
			}
		}
	default:
		return nil
	}
}
