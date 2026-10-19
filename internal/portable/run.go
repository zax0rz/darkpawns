package portable

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	instance, err := filepath.Abs(instance)
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
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	lib := filepath.Join(instance, "lib")
	cmd := exec.Command(filepath.Join(bundle, "darkpawns-server"+suffix), "-world", filepath.Join(lib, "world"), "-scripts", filepath.Join(lib, "world", "scripts"), "-db", filepath.Join(lib, "data", "darkpawns.db"), "-web", filepath.Join(bundle, "release-assets", "web"), "-http-bind", "127.0.0.1", "-port", strconv.Itoa(cfg.HTTPPort), "-telnet-port", "0", "-whod-port", "0")
	cmd.Dir = lib
	cmd.Env = instanceEnv(os.Environ(), bundle, instance, cfg)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, errOut
	prepareProcess(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start bundled server: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		signalErr := interruptProcess(cmd.Process)
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case err := <-done:
			return errors.Join(signalErr, err)
		case <-timer.C:
			killErr := cmd.Process.Kill()
			waitErr := <-done
			return errors.Join(errors.New("server did not stop within 30 seconds"), signalErr, killErr, waitErr)
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
	data := filepath.Join(instance, "lib", "data")
	return append(env, "JWT_SECRET="+cfg.JWTSecret, "ENVIRONMENT=production", "ADMIN_STORE_PATH="+filepath.Join(data, "admin.json"), "ADMIN_UI_DIR="+filepath.Join(bundle, "release-assets", "admin-ui"), "DP_NOTIFICATION_CONFIG_PATH="+filepath.Join(data, "notifications.json"), "DP_NOTIFICATION_KEY_PATH="+filepath.Join(data, "notifications.json.key"))
}
