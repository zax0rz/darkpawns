package telnet

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/pkg/session"
)

// tlsHandshakeTimeout bounds how long an accepted TLS connection may take to
// finish its handshake, so a client that connects and stalls cannot hold a
// connection slot. A variable so tests can shorten it.
var tlsHandshakeTimeout = 10 * time.Second

// tlsAdvertisement is what MSSP tells clients about the TLS port.
type tlsAdvertisement struct {
	port     int
	hostname string
}

// tlsAdvert is set while a TLS listener runs; sendMSSP reads it.
var tlsAdvert atomic.Pointer[tlsAdvertisement]

// ListenTLS starts a TLS telnet listener on port, serving the same game as
// Listen: the telnet protocol runs unchanged inside the encrypted stream.
// certFile and keyFile are PEM files; they are re-read when they change on
// disk, so a certificate renewal needs no restart. Returns immediately.
func ListenTLS(port int, manager *session.Manager, certFile, keyFile string) error {
	certs, err := newCertReloader(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("telnet TLS certificate: %w", err)
	}
	addr := fmt.Sprintf(":%d", port)
	raw, err := listenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("telnet TLS listen: %w", err)
	}
	ln := tls.NewListener(raw, &tls.Config{
		MinVersion:     tls.VersionTLS12,
		GetCertificate: certs.getCertificate,
	})
	boundPort := port
	if tcp, ok := raw.Addr().(*net.TCPAddr); ok {
		boundPort = tcp.Port
	}
	slog.Info("Telnet TLS listening", "address", addr, "hostname", certs.hostname())

	connMu.Lock()
	tlsListener = ln
	connMu.Unlock()
	tlsAdvert.Store(&tlsAdvertisement{port: boundPort, hostname: certs.hostname()})

	serve(ln, manager)
	return nil
}

// completeTLSHandshake finishes the handshake of a TLS connection within
// tlsHandshakeTimeout and reports whether the connection is usable. Plain
// connections have no handshake and are always usable.
func completeTLSHandshake(conn net.Conn) bool {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), tlsHandshakeTimeout)
	defer cancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		slog.Info("Telnet TLS handshake failed", "remote_addr", conn.RemoteAddr(), "error", err)
		return false
	}
	return true
}

// certReloader serves a certificate from PEM files and reloads it when either
// file's modification time changes. A reload that fails keeps the certificate
// already loaded, so a half-written renewal never takes the port down.
type certReloader struct {
	certFile, keyFile string

	mu       sync.Mutex
	cert     *tls.Certificate
	certMod  time.Time
	keyMod   time.Time
	checked  time.Time
	hostName string
}

// certCheckInterval rate-limits the stat calls a busy listener makes.
var certCheckInterval = 30 * time.Second

func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	r := &certReloader{certFile: filepath.Clean(certFile), keyFile: filepath.Clean(keyFile)}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *certReloader) load() error {
	certInfo, err := os.Stat(r.certFile)
	if err != nil {
		return err
	}
	keyInfo, err := os.Stat(r.keyFile)
	if err != nil {
		return err
	}
	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return err
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return err
	}
	cert.Leaf = leaf
	r.cert = &cert
	r.certMod, r.keyMod = certInfo.ModTime(), keyInfo.ModTime()
	r.hostName = certHostname(leaf)
	return nil
}

func (r *certReloader) getCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if now := time.Now(); now.Sub(r.checked) >= certCheckInterval {
		r.checked = now
		if r.changedOnDisk() {
			if err := r.load(); err != nil {
				slog.Error("Telnet TLS certificate reload failed; keeping the current one", "cert", r.certFile, "error", err)
			} else {
				slog.Info("Telnet TLS certificate reloaded", "cert", r.certFile, "not_after", r.cert.Leaf.NotAfter)
			}
		}
	}
	return r.cert, nil
}

func (r *certReloader) changedOnDisk() bool {
	certInfo, err := os.Stat(r.certFile)
	if err != nil {
		return false
	}
	keyInfo, err := os.Stat(r.keyFile)
	if err != nil {
		return false
	}
	return !certInfo.ModTime().Equal(r.certMod) || !keyInfo.ModTime().Equal(r.keyMod)
}

func (r *certReloader) hostname() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hostName
}

// certHostname is the name MSSP advertises: the certificate's first DNS
// name that is not a wildcard. Mudlet only offers the TLS port to players
// who connected by that name, the one the certificate will verify.
func certHostname(leaf *x509.Certificate) string {
	for _, name := range leaf.DNSNames {
		if !strings.HasPrefix(name, "*.") {
			return name
		}
	}
	return ""
}
