package portable

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// Run holds the instance lock for the entire child lifetime, including shutdown.
// It forwards console interruption, then waits for the game's normal saves.
func Run(ctx context.Context, bundle, instance string, in io.Reader, out, errOut io.Writer) (result error) {
	return run(ctx, bundle, instance, in, out, errOut, nil, "")
}

// RunWithReady calls ready only after the child's local health endpoint responds.
// Startup failure or timeout stops and reaps the child before releasing its lock.
func RunWithReady(ctx context.Context, bundle, instance string, in io.Reader, out, errOut io.Writer, ready func(string) error) error {
	return run(ctx, bundle, instance, in, out, errOut, ready, "")
}

func run(ctx context.Context, bundle, instance string, in io.Reader, out, errOut io.Writer, ready func(string) error, updateID string) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	bundle, err = ActiveBundle(bundle, instance)
	if err != nil {
		return err
	}
	bundle, err = filepath.Abs(bundle)
	if err != nil {
		return err
	}
	cfg, err := Load(instance)
	if err != nil {
		return err
	}
	m, err := ReadManifest(bundle)
	if err != nil {
		return err
	}
	if err := Verify(bundle, m); err != nil {
		return err
	}
	lock := flock.New(filepath.Join(instance, ".run.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return errors.New("this instance is already running")
	}
	defer func() { result = errors.Join(result, lock.Close()) }()
	if err := updateRunAllowed(instance, bundle, updateID); err != nil {
		return err
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	markReady, closeControl, err := serveControl(instance, bundle, cfg.HTTPPort, stop)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, closeControl(result)) }()

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	lib := filepath.Join(instance, "lib")
	cmd := exec.Command(filepath.Join(bundle, "darkpawns-server"+suffix), "-world", filepath.Join(lib, "world"), "-scripts", filepath.Join(lib, "world", "scripts"), "-db", filepath.Join(lib, "data", "darkpawns.db"), "-web", filepath.Join(bundle, "release-assets", "web"), "-http-bind", "127.0.0.1", "-port", strconv.Itoa(cfg.HTTPPort), "-telnet-bind", cfg.TelnetBind, "-telnet-tls-bind", cfg.TelnetTLSBind, "-whod-bind", cfg.WhodBind, "-telnet-port", strconv.Itoa(cfg.TelnetPort), "-telnet-tls-port", strconv.Itoa(cfg.TelnetTLSPort), "-whod-port", strconv.Itoa(cfg.WhodPort))
	if cfg.GreetingOverlay {
		cmd.Args = append(cmd.Args, "-greeting-overlay")
	}
	if err := preflightHosting(cfg); err != nil {
		return err
	}
	if err := publishMudlet(bundle, cfg); err != nil {
		return err
	}
	cmd.Dir = lib
	cmd.Env = instanceEnv(os.Environ(), bundle, instance, cfg)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, errOut
	prepareProcess(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start bundled server: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var readiness <-chan error
	url := "http://127.0.0.1:" + strconv.Itoa(cfg.HTTPPort)
	if ready == nil {
		ready = func(string) error { return nil }
	}
	{
		ch := make(chan error, 1)
		readiness = ch
		go func() { ch <- waitForHTTP(monitorCtx, url+"/health", 30*time.Second) }()
	}
	var startupErr error
	for {
		select {
		case err := <-done:
			if ready != nil {
				return errors.Join(errors.New("server exited before becoming ready"), err)
			}
			return err
		case err := <-readiness:
			readiness = nil
			// An early process exit takes precedence over an unrelated HTTP response.
			select {
			case exitErr := <-done:
				return errors.Join(errors.New("server exited before becoming ready"), exitErr)
			default:
			}
			if err == nil {
				markReady()
				err = ready(url)
			}
			ready = nil
			if err != nil {
				startupErr = err
			} else {
				continue
			}
		case <-ctx.Done():
		}
		signalErr := interruptProcess(cmd.Process)
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case err := <-done:
			return errors.Join(startupErr, signalErr, err)
		case <-timer.C:
			killErr := cmd.Process.Kill()
			waitErr := <-done
			return errors.Join(startupErr, errors.New("server did not stop within 30 seconds"), signalErr, killErr, waitErr)
		}
	}
}

func waitForHTTP(ctx context.Context, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		response, err := client.Do(req)
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("local web client did not become ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func instanceEnv(inherited []string, bundle, instance string, cfg Config) []string {
	env := make([]string, 0, len(inherited))
	for _, entry := range inherited {
		key, _, _ := strings.Cut(entry, "=")
		key = strings.ToUpper(key)
		// A portable instance must not inherit another installation's database,
		// secrets, notification destinations, test controls, or TLS configuration.
		if strings.HasPrefix(key, "DP_") || strings.HasPrefix(key, "ADMIN_") || strings.HasPrefix(key, "DARKPAWNS_") || strings.HasPrefix(key, "TLS_") || strings.HasPrefix(key, "TELNET_TLS_") {
			continue
		}
		switch key {
		case "JWT_SECRET", "ENVIRONMENT", "DATABASE_URL", "USE_TLS", "DISCORD_WEBHOOK_URL":
			continue
		}
		env = append(env, entry)
	}
	for key, value := range cfg.ServerEnv {
		env = append(env, key+"="+value)
	}
	data := filepath.Join(instance, "lib", "data")
	return append(env, "JWT_SECRET="+cfg.JWTSecret, "ENVIRONMENT=production", "ADMIN_STORE_PATH="+filepath.Join(data, "admin.json"), "ADMIN_UI_DIR="+filepath.Join(bundle, "release-assets", "admin-ui"), "DP_NOTIFICATION_CONFIG_PATH="+filepath.Join(data, "notifications.json"), "DP_NOTIFICATION_KEY_PATH="+filepath.Join(data, "notifications.json.key"))
}
