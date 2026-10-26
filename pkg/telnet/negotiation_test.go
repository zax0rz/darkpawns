package telnet

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/session"
)

func countTelnetCommand(data []byte, command, option byte) int {
	count := 0
	for i := 0; i+1 < len(data); {
		if data[i] != IAC {
			i++
			continue
		}
		switch data[i+1] {
		case WILL, WONT, DO, DONT:
			if i+2 >= len(data) {
				return count
			}
			if data[i+1] == command && data[i+2] == option {
				count++
			}
			i += 3
		case SB:
			end := bytes.Index(data[i+2:], []byte{IAC, SE})
			if end < 0 {
				return count
			}
			i += 2 + end + 2
		default:
			i += 2
		}
	}
	return count
}

// TestTelnetNegotiationArmsSettleWithoutEchoingDuplicateRequests exercises
// all four negotiation verbs through readLineBody. In particular, unsolicited
// WONT and DONT frames must not be answered, and repeated refusals get one
// response until the peer settles the option state.
func TestTelnetNegotiationArmsSettleWithoutEchoingDuplicateRequests(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = server.Close() })
	tc := &telnetConn{
		Conn: server,
		br:   bufio.NewReader(server),
		wmu:  make(chan struct{}, 1),
	}

	output := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(client)
		output <- data
	}()
	lineDone := make(chan struct{})
	go func() {
		_, _ = tc.readLine()
		close(lineDone)
	}()

	input := []byte{
		IAC, WONT, OPT_ECHO,
		IAC, WONT, OPT_ECHO,
		IAC, DONT, OPT_EOR,
		IAC, DONT, OPT_EOR,
		IAC, DO, OPT_SGA,
		IAC, DO, OPT_SGA,
		IAC, DONT, OPT_SGA,
		IAC, WILL, 222,
		IAC, WILL, 222,
		IAC, WILL, OPT_ECHO,
		IAC, WILL, OPT_ECHO,
		IAC, WONT, OPT_ECHO,
		'\n',
	}
	if err := client.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Write(input); err != nil {
		t.Fatalf("write negotiation script: %v", err)
	}
	select {
	case <-lineDone:
	case <-time.After(2 * time.Second):
		t.Fatal("readLine did not finish after the input line")
	}
	_ = server.Close()
	got := <-output
	want := []byte{
		IAC, WONT, OPT_SGA,
		IAC, DONT, 222,
		IAC, DO, OPT_ECHO,
		IAC, DONT, OPT_ECHO,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("negotiation replies = %v, want %v", got, want)
	}
}

// TestTelnetAcknowledgingClientGetsBoundedFrames reproduces a client that
// accepts each server WILL (except MCCP2, which this client cannot decode).
// The initial MSSP offer must produce one payload, not a new WILL/payload for
// every DO acknowledgement.
func TestTelnetAcknowledgingClientGetsBoundedFrames(t *testing.T) {
	manager, world := newTestManager(t)
	defer world.StopAITicker()

	client, server := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(server, manager, session.ConnectionIdentity{})
	}()
	t.Cleanup(func() { _ = client.Close() })

	responses := make(chan []byte, 128)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for response := range responses {
			if _, err := client.Write(response); err != nil {
				return
			}
		}
	}()

	readResult := make(chan struct {
		frames map[[2]byte]int
		mssp   int
		err    string
	}, 1)
	go func() {
		frames := make(map[[2]byte]int)
		var all, pending bytes.Buffer
		mssp := 0
		promptSeen := false
		readBuf := make([]byte, 4096)
		for {
			deadline := time.Now().Add(2 * time.Second)
			if promptSeen {
				deadline = time.Now().Add(100 * time.Millisecond)
			}
			_ = client.SetReadDeadline(deadline)
			n, err := client.Read(readBuf)
			if n > 0 {
				chunk := readBuf[:n]
				all.Write(chunk)
				pending.Write(chunk)
				data := pending.Bytes()
				consumed := 0
				for consumed < len(data) {
					if data[consumed] != IAC {
						consumed++
						continue
					}
					if consumed+1 >= len(data) {
						break
					}
					command := data[consumed+1]
					switch command {
					case WILL, WONT, DO, DONT:
						if consumed+2 >= len(data) {
							goto parsed
						}
						option := data[consumed+2]
						frames[[2]byte{command, option}]++
						switch command {
						case WILL:
							answer := DO
							if option == OPT_COMPRESS2 {
								answer = DONT
							}
							responses <- []byte{IAC, answer, option}
						case DO:
							responses <- []byte{IAC, WILL, option}
						}
						consumed += 3
					case SB:
						if consumed+2 >= len(data) {
							goto parsed
						}
						end := bytes.Index(data[consumed+3:], []byte{IAC, SE})
						if end < 0 {
							goto parsed
						}
						if data[consumed+2] == OPT_MSSP {
							mssp++
						}
						consumed += 3 + end + 2
					default:
						consumed += 2
					}
				}
			parsed:
				remaining := append([]byte(nil), data[consumed:]...)
				pending.Reset()
				pending.Write(remaining)
				if bytes.Contains(all.Bytes(), []byte("By what name do you wish to be known? ")) {
					promptSeen = true
				}
				frameTotal := 0
				for _, count := range frames {
					frameTotal += count
				}
				if frameTotal > 32 {
					readResult <- struct {
						frames map[[2]byte]int
						mssp   int
						err    string
					}{frames, mssp, "server exceeded 32 negotiation frames"}
					return
				}
			}
			if err != nil {
				if timeout, ok := err.(net.Error); ok && timeout.Timeout() && promptSeen {
					readResult <- struct {
						frames map[[2]byte]int
						mssp   int
						err    string
					}{frames, mssp, ""}
					return
				}
				readResult <- struct {
					frames map[[2]byte]int
					mssp   int
					err    string
				}{frames, mssp, err.Error()}
				return
			}
		}
	}()

	result := <-readResult
	close(responses)
	_ = client.Close()
	<-writerDone
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("telnet handler did not stop after the client closed")
	}
	if result.err != "" {
		t.Fatal(result.err)
	}
	if got := result.frames[[2]byte{WILL, OPT_MSSP}]; got != 1 {
		t.Errorf("server WILL MSSP frames = %d, want 1", got)
	}
	if got := result.frames[[2]byte{WILL, OPT_GMCP}]; got != 1 {
		t.Errorf("server WILL GMCP frames = %d, want 1", got)
	}
	if got := result.frames[[2]byte{WILL, OPT_EOR}]; got != 1 {
		t.Errorf("server WILL EOR frames = %d, want 1", got)
	}
	if got := result.frames[[2]byte{WILL, OPT_COMPRESS2}]; got != 1 {
		t.Errorf("server WILL COMPRESS2 frames = %d, want 1", got)
	}
	if result.mssp != 1 {
		t.Errorf("MSSP payloads = %d, want 1", result.mssp)
	}
}
