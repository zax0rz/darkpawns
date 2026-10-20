package telnet

import (
	"crypto/tls"
	"net"
	"testing"
	"time"
)

func nonLoopbackIPv4(t *testing.T) string {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err == nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			return ip.String()
		}
	}
	t.Skip("no non-loopback IPv4 interface available")
	return ""
}

func TestExplicitTelnetBindingConfinesPlainAndTLS(t *testing.T) {
	external := nonLoopbackIPv4(t)
	manager, world := newTestManager(t)
	t.Cleanup(world.StopAITicker)
	cert, key, pool := writeTestCert(t, t.TempDir(), "bind-", "localhost")
	for _, encrypted := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "TLS"}[encrypted], func(t *testing.T) {
			defer Stop()
			var err error
			if encrypted {
				err = ListenTLSOn("127.0.0.1", 0, manager, cert, key)
			} else {
				err = ListenOn("127.0.0.1", 0, manager)
			}
			if err != nil {
				t.Fatal(err)
			}
			plain, secure := ListenerAddresses()
			addr := plain
			if encrypted {
				addr = secure
			}
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				t.Fatal(err)
			}
			remote, err := net.DialTimeout("tcp", net.JoinHostPort(external, port), time.Second)
			if err == nil {
				remote.Close()
				t.Fatal("loopback listener accepted a non-loopback connection")
			}
			conn, err := net.DialTimeout("tcp", addr, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if encrypted {
				tlsConn := tls.Client(conn, &tls.Config{RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12})
				if err := tlsConn.Handshake(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if err := Listen(0, manager); err != nil {
		t.Fatal(err)
	}
	defer Stop()
	plain, _ := ListenerAddresses()
	host, port, err := net.SplitHostPort(plain)
	if err != nil {
		t.Fatal(err)
	}
	if !net.ParseIP(host).IsUnspecified() {
		t.Fatalf("default no longer wildcard: %s", plain)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(external, port), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	Stop()
	plain, secure := ListenerAddresses()
	if plain != "" || secure != "" {
		t.Fatal("closed listener advertised")
	}
}
