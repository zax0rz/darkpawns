package e2e

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

type serverPorts struct {
	http, telnet, whod int
}

// Reserve the whole set together. Closing each ephemeral listener before
// allocating the next lets the OS return the same port for HTTP and telnet
// (PR #1926 CI). WHOD also needs its own allocation instead of an unchecked
// telnet+1. Release immediately before starting the child: this prevents
// overlap within the set, but cannot prevent an unrelated process claiming a
// port between release and the child's bind.
func reserveServerPorts(t *testing.T) (serverPorts, func()) {
	t.Helper()
	var listeners []net.Listener
	var once sync.Once
	release := func() {
		once.Do(func() {
			for _, listener := range listeners {
				if err := listener.Close(); err != nil {
					t.Errorf("release server port: %v", err)
				}
			}
		})
	}
	t.Cleanup(release)
	for range 3 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("reserve server port: %v", err)
		}
		listeners = append(listeners, listener)
	}
	return serverPorts{
		http:   listeners[0].Addr().(*net.TCPAddr).Port,
		telnet: listeners[1].Addr().(*net.TCPAddr).Port,
		whod:   listeners[2].Addr().(*net.TCPAddr).Port,
	}, release
}

func TestServerPortsAreDistinctAndReserved(t *testing.T) {
	ports, release := reserveServerPorts(t)
	if ports.http == ports.telnet || ports.http == ports.whod || ports.telnet == ports.whod {
		t.Fatalf("server ports overlap: %+v", ports)
	}
	for _, port := range []int{ports.http, ports.telnet, ports.whod} {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			if err := listener.Close(); err != nil {
				t.Errorf("close unexpected listener: %v", err)
			}
			t.Errorf("port %d was released before startup", port)
		}
	}
	release()
	release() // Startup release and t.Cleanup must be safe together.
	for _, port := range []int{ports.http, ports.telnet, ports.whod} {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Errorf("port %d remains reserved after release: %v", port, err)
			continue
		}
		if err := listener.Close(); err != nil {
			t.Errorf("close released-port probe: %v", err)
		}
	}
}
