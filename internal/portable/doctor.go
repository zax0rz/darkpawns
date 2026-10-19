package portable

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	manifest, err := ReadManifest(bundle)
	if err == nil {
		err = Verify(bundle, manifest)
	}
	add("Release assets", err, "Checksums match", "Extract a complete fresh bundle; keep your instance folder.")
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
