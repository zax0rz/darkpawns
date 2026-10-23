package portable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/internal/releaseauth"

	_ "modernc.org/sqlite"
)

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Doctor reports facts and recovery steps; it never rewrites configuration,
// resets credentials, repairs SQLite or kills processes.
func Doctor(ctx context.Context, bundle, instance string) []Check {
	checks := []Check{}
	originalBundle := bundle
	resolved, resolveErr := ActiveBundle(bundle, instance)
	if resolveErr == nil {
		bundle = resolved
	}
	add := func(name string, err error, success, fix string) {
		detail := success
		if err != nil {
			detail = err.Error() + ". " + fix
		}
		checks = append(checks, Check{Name: name, OK: err == nil, Detail: detail})
	}
	add("Active release", resolveErr, bundle, "Inspect .updates/active.json; retain the original launcher at "+originalBundle+" for recovery.")
	add("Update transaction", pendingUpdate(instance), "No interrupted update", "Use "+OperatorCommand(instance, "update", "recover")+" to recover an interrupted update.")
	// Which publisher key this build trusts: the pinned key, or the snapshot
	// override a CI/snapshot launcher was linked with. Published builds must
	// show the pinned fingerprint.
	effective := releaseauth.TrustedKeyBytes()
	fingerprint := sha256.Sum256(effective)
	if bytes.Equal(effective, releaseauth.TrustedKey) {
		add("Publisher key", nil, "pinned, sha256:"+hex.EncodeToString(fingerprint[:16]), "")
	} else {
		add("Publisher key", nil, "linked override key, sha256:"+hex.EncodeToString(fingerprint[:16]), "Published builds must trust the pinned key, not a linked override.")
	}
	// Strict bundle authentication (L1-006/007/021): the publisher signature
	// and every checksum the signed manifest lists, executables included.
	add("Release signature", PublisherVerify(bundle), "Publisher-signed, checksums match", "Restore this release from its signed archive; strict mode refuses to run an unsigned or modified bundle.")
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	info, err := os.Stat(filepath.Join(bundle, "darkpawns-server"+suffix))
	if err == nil && (!info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0)) {
		err = errors.New("bundled server is not an executable regular file")
	}
	add("Server executable", err, "Present", "Restore the server from this release archive.")
	instance, err = canonicalInstance(instance)
	if err != nil {
		add("Instance", err, "", "Choose an existing instance with --instance.")
		return checks
	}
	cfg, err := Load(instance)
	add("Instance", err, "Configuration is valid", "Restore the original instance.json from backup; do not regenerate its secret.")
	if err != nil {
		return checks
	}
	if runtime.GOOS != "windows" {
		for _, name := range []string{".", "instance.json"} {
			info, err := os.Stat(filepath.Join(instance, name))
			if err == nil && info.Mode().Perm()&0o077 != 0 {
				err = errors.New("instance or signing configuration is readable by other users")
			}
			add("Private storage "+name, err, "Private to your user", "Restrict the instance folder to 0700 and instance.json to 0600.")
		}
	}
	status, err := Status(ctx, instance)
	add("Launcher", err, status.State, "Inspect lib/logs/launcher.log; do not delete a held lock or kill a PID from an old file.")
	if err == nil && status.State == "stopped" {
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", cfg.HTTPPort))
		if err == nil {
			err = listener.Close()
		}
		add("Web port", err, "Available", "Identify the process using the configured port before starting this instance.")
	} else if err == nil {
		healthCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := waitForHTTP(healthCtx, strings.TrimSuffix(status.URL, "/")+"/health", 2*time.Second)
		cancel()
		add("Web client", err, status.URL, "Wait for startup or inspect the launcher log.")
	}
	if status.State == "stopped" {
		var last exitRecord
		if exitErr := readJSON(filepath.Join(instance, ".last-exit.json"), &last); !os.IsNotExist(exitErr) {
			if exitErr == nil && last.Error != "" {
				exitErr = errors.New(last.Error)
			}
			add("Last shutdown", exitErr, "Exited normally", "Inspect lib/logs/launcher.log and verify the saved data before restarting.")
		}
	}
	database := filepath.Join(instance, "lib", "data", "darkpawns.db")
	if _, err := os.Stat(database); os.IsNotExist(err) {
		add("SQLite", nil, "Not created yet; the first successful start creates it", "")
	} else if err != nil {
		add("SQLite", err, "", "Check the file and folder permissions.")
	} else if status.State != "stopped" {
		add("SQLite", nil, "Live instance; stop it to run the offline integrity check", "")
	} else {
		add("SQLite", checkDatabase(ctx, database), "Integrity check passed (read-only)", "Keep the original files and restore a consistent backup; doctor performs no repairs.")
	}
	installed, err := ServiceInstalled(instance)
	detail := "Not installed; configure sign-in startup with " + OperatorCommand(instance, "service", "install")
	if installed {
		detail = "Registered for your account"
	}
	add("Automatic startup", err, detail, "Restore any externally changed definition first. To clean up owned registration, use "+OperatorCommand(instance, "service", "remove")+", then install again in a supported user session.")
	var supervision supervisionState
	if stateErr := readJSON(filepath.Join(instance, ".supervision.json"), &supervision); !os.IsNotExist(stateErr) {
		active, activeErr := supervisionActive(instance)
		detail := fmt.Sprintf("Monitor active: %t; world requested: %t", active, supervision.Wanted)
		add("System supervision", errors.Join(stateErr, activeErr), detail, "Inspect the external service manager and .supervision.json; keep the original launcher.")
	}
	return checks
}

func checkDatabase(ctx context.Context, path string) (result error) {
	path = filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		path = "/" + path
	}
	uri := url.URL{Scheme: "file", Path: path}
	q := uri.Query()
	q.Set("mode", "ro")
	uri.RawQuery = q.Encode()
	database, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, database.Close()) }()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := database.QueryContext(ctx, "PRAGMA quick_check")
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, rows.Close()) }()
	count := 0
	for rows.Next() {
		var verdict string
		if err := rows.Scan(&verdict); err != nil {
			return err
		}
		if verdict != "ok" {
			return errors.New("SQLite integrity check failed")
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != 1 {
		return errors.New("SQLite integrity check did not complete")
	}
	return nil
}
