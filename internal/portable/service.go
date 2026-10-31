package portable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

type serviceRecord struct {
	Platform string `json:"platform"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	Launcher string `json:"launcher"`
}

func serviceName(instance string) string {
	sum := sha256.Sum256([]byte(instance))
	return "darkpawns-" + hex.EncodeToString(sum[:12])
}

func servicePath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "linux":
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(home, ".config")
		}
		if !filepath.IsAbs(base) {
			return "", errors.New("XDG_CONFIG_HOME must be absolute")
		}
		return filepath.Join(base, "systemd", "user", name+".service"), nil
	case "darwin":
		return filepath.Join(home, "Library", "LaunchAgents", name+".plist"), nil
	case "windows":
		return "", nil // Task Scheduler owns the registered task.
	default:
		return "", errors.New("automatic startup is supported on Linux, macOS and Windows")
	}
}

type windowsTaskActionRecord struct {
	Command   string
	Arguments string
}

// The Windows startup-task query runs inside `start` before the 40-second
// readiness wait, and `scripts/portable_lifecycle_smoke.py` caps the whole
// `start` subprocess at 60 seconds. Both attempts therefore share one 15-second
// budget (10s first attempt + the remaining budget for the retry) so the first
// attempt has enough headroom to avoid the 5s PowerShell/COM teardown race
// while a double failure still exits and prints the two-attempt error well
// before the 60-second caller cap.
var (
	windowsTaskQueryBudget    = 15 * time.Second
	windowsTaskQueryFirstWait = 10 * time.Second
	runWindowsTaskQuery       = execWindowsTaskQuery
)

func execWindowsTaskQuery(ctx context.Context, name string) ([]byte, error) {
	script := "[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); $ErrorActionPreference='Stop'; $s=New-Object -ComObject Schedule.Service; $s.Connect(); $t=$s.GetFolder('\\').GetTask('" + name + "'); $a=$t.Definition.Actions.Item(1); [pscustomobject]@{Command=$a.Path; Arguments=$a.Arguments} | ConvertTo-Json -Compress"
	data, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("query owned startup task: %w: %s", err, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// queryStartupTaskActionWithRetry reads the owned scheduled task's action
// through the Schedule.Service COM API. Loaded Windows CI runners can stall
// PowerShell/COM teardown past a 5-second deadline (causing context
// cancellation to race process exit with `TerminateProcess: Access is denied.`)
// or fail a single COM query transiently (`exit status 1` with empty output).
// Both attempts share one 15-second budget so the two-attempt error can print
// within the smoke test's 60-second cap on `start`; JSON decode errors and
// action mismatches surface immediately without retry.
func queryStartupTaskActionWithRetry(name string) (windowsTaskActionRecord, error) {
	budgetCtx, cancelBudget := context.WithTimeout(context.Background(), windowsTaskQueryBudget)
	defer cancelBudget()

	firstWait := windowsTaskQueryFirstWait
	if firstWait > windowsTaskQueryBudget {
		firstWait = windowsTaskQueryBudget
	}
	firstCtx, cancelFirst := context.WithTimeout(budgetCtx, firstWait)
	data, err := runWindowsTaskQuery(firstCtx, name)
	cancelFirst()
	if err != nil {
		firstErr := err
		slog.Warn("startup task query failed; retrying once",
			"task", name,
			"error", firstErr.Error())
		data, err = runWindowsTaskQuery(budgetCtx, name)
		if err != nil {
			return windowsTaskActionRecord{}, fmt.Errorf("query owned startup task after two attempts (first: %v; second: %w)", firstErr, err)
		}
		slog.Warn("startup task query succeeded on the retry; the first attempt failed",
			"task", name,
			"first_error", firstErr.Error())
	}
	var action windowsTaskActionRecord
	if err := json.Unmarshal(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), &action); err != nil {
		return windowsTaskActionRecord{}, err
	}
	return action, nil
}

func readService(instance string) (serviceRecord, error) {
	var record serviceRecord
	if err := readJSON(filepath.Join(instance, ".service.json"), &record); err != nil {
		return record, err
	}
	if record.Platform != runtime.GOOS || record.Name != serviceName(instance) || !filepath.IsAbs(record.Launcher) {
		return record, errors.New("invalid automatic-start registration; inspect .service.json")
	}
	want, err := servicePath(record.Name)
	if err != nil {
		return record, err
	}
	if record.Path != want {
		return record, errors.New("automatic-start location has changed; restore its original environment before removing it")
	}
	if want != "" {
		info, err := os.Lstat(want)
		if err != nil {
			return record, err
		}
		if !info.Mode().IsRegular() {
			return record, errors.New("automatic-start definition is not a regular file")
		}
		data, err := os.ReadFile(want)
		if err != nil {
			return record, err
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != record.SHA256 {
			return record, errors.New("automatic-start definition was modified externally; restore it before removing it")
		}
	}
	if runtime.GOOS == "windows" {
		action, err := queryStartupTaskActionWithRetry(record.Name)
		if err != nil {
			return record, err
		}
		expectedCommand, expectedArgs := windowsTaskAction(record.Launcher, instance)
		if !strings.EqualFold(action.Command, expectedCommand) || action.Arguments != expectedArgs {
			return record, errors.New("startup task was modified externally; refusing to operate on it")
		}
	}
	return record, nil
}

func ServiceInstalled(instance string) (bool, error) {
	_, err := readService(instance)
	if os.IsNotExist(err) {
		// Only a missing record means uninstalled. A missing registered definition is
		// broken registration and must not silently fall back to another supervisor.
		if _, statErr := os.Stat(filepath.Join(instance, ".service.json")); os.IsNotExist(statErr) {
			return false, nil
		}
	}
	return err == nil, err
}

func serviceCommand(ctx context.Context, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func InstallService(ctx context.Context, instance string) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	if _, err := Load(instance); err != nil {
		return err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	if active, err := supervisionActive(instance); err != nil {
		return err
	} else if active {
		return errors.New("stop system supervision before registering sign-in startup")
	}
	if _, err := os.Lstat(filepath.Join(instance, ".service.json")); !os.IsNotExist(err) {
		return fmt.Errorf("automatic-start registration already exists; remove it with %s before reinstalling", OperatorCommand(instance, "service", "remove"))
	}
	launcher, err := os.Executable()
	if err != nil {
		return err
	}
	launcher, err = filepath.EvalSymlinks(launcher)
	if err != nil {
		return err
	}
	name := serviceName(instance)
	path, err := servicePath(name)
	if err != nil {
		return err
	}
	account, err := user.Current()
	if err != nil {
		return err
	}
	data, err := renderService(runtime.GOOS, name, launcher, instance, account.Uid)
	if err != nil {
		return err
	}
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		if err := writeExclusive(path, data); err != nil {
			return err
		}
	}
	sum := sha256.Sum256(data)
	record := serviceRecord{Platform: runtime.GOOS, Name: name, Path: path, SHA256: hex.EncodeToString(sum[:]), Launcher: launcher}
	// Persist ownership before registration. If the native command fails, retain
	// the record and definition so service remove can undo exactly this attempt.
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := writeExclusive(filepath.Join(instance, ".service.json"), encoded); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		if err := serviceCommand(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return err
		}
		return serviceCommand(ctx, "systemctl", "--user", "enable", name+".service")
	case "darwin":
		return serviceCommand(ctx, "launchctl", "bootstrap", "gui/"+account.Uid, path)
	case "windows":
		// /Create without /F refuses replacing an unrelated task with the same name.
		temp, err := os.CreateTemp(instance, ".task-*.xml")
		if err != nil {
			return err
		}
		defer func() {
			err := os.Remove(temp.Name())
			if os.IsNotExist(err) {
				err = nil
			}
			result = errors.Join(result, err)
		}()
		_, writeErr := temp.Write(windowsTaskFile(data))
		if err := errors.Join(writeErr, temp.Close()); err != nil {
			return err
		}
		err = serviceCommand(ctx, "schtasks.exe", "/Create", "/TN", name, "/XML", temp.Name())
		if err != nil {
			return errors.Join(err, os.Remove(filepath.Join(instance, ".service.json")))
		}
		return nil
	}
	return errors.New("unsupported service platform")
}

func startService(ctx context.Context, instance string) error {
	record, err := readService(instance)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		return serviceCommand(ctx, "systemctl", "--user", "restart", record.Name+".service")
	case "darwin":
		account, err := user.Current()
		if err != nil {
			return err
		}
		domain := "gui/" + account.Uid
		return startLaunchAgent(ctx, domain+"/"+record.Name, domain, record.Path, serviceCommand)
	case "windows":
		// The previous supervisor can release its run lock just before Task
		// Scheduler observes exit. Wait for that task to finish before launching it.
		script := "$ErrorActionPreference='Stop'; $s=New-Object -ComObject Schedule.Service; $s.Connect(); $t=$s.GetFolder('\\').GetTask('" + record.Name + "'); $deadline=(Get-Date).AddSeconds(5); while ($t.State -eq 4) { if ((Get-Date) -gt $deadline) { throw 'Previous startup task is still exiting' }; Start-Sleep -Milliseconds 100 }; $t.Run($null) | Out-Null"
		return serviceCommand(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	}
	return errors.New("unsupported service platform")
}

func RemoveService(ctx context.Context, instance string) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	record, err := readService(instance)
	if err != nil {
		return err
	}
	if err := stopInstance(ctx, instance); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "linux":
		// disable also succeeds for an already disabled, owned unit.
		if err := serviceCommand(ctx, "systemctl", "--user", "--no-reload", "disable", record.Name+".service"); err != nil {
			return err
		}
	case "darwin":
		account, err := user.Current()
		if err != nil {
			return err
		}
		// A stopped LaunchAgent remains loaded; bootout removes its registration.
		if err := serviceCommand(ctx, "launchctl", "bootout", "gui/"+account.Uid+"/"+record.Name); err != nil {
			// An install may have failed before bootstrap. Only tolerate a verified
			// absent job, not a failed bootout of a job that is still registered.
			if probe := serviceCommand(ctx, "launchctl", "print", "gui/"+account.Uid+"/"+record.Name); probe == nil {
				return err
			}
		}
	case "windows":
		if err := serviceCommand(ctx, "schtasks.exe", "/Delete", "/TN", record.Name, "/F"); err != nil {
			return err
		}
	}
	if record.Path != "" {
		if err := os.Remove(record.Path); err != nil {
			return err
		}
	}
	if runtime.GOOS == "linux" {
		if err := serviceCommand(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return err
		}
	}
	return os.Remove(filepath.Join(instance, ".service.json"))
}

func xmlText(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func systemdArg(value string) string {
	return strconv.Quote(strings.ReplaceAll(strings.ReplaceAll(value, "%", "%%"), "$", "$$"))
}

// Task Scheduler expands environment expressions even in Exec arguments.
// A UTF-16 encoded PowerShell command carries literal paths (including %) with
// single-quote escaping, without interpolating operator input into shell syntax.
// https://learn.microsoft.com/windows/win32/taskschd/execaction
func windowsTaskAction(launcher, instance string) (string, string) {
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
	script := "$ErrorActionPreference='Stop'; & " + quote(launcher) + " '_serve' '--instance' " + quote(instance) + "; exit $LASTEXITCODE"
	var encoded []byte
	for _, v := range utf16.Encode([]rune(script)) {
		encoded = append(encoded, byte(v&0xff), byte(v>>8))
	}
	system := os.Getenv("SystemRoot")
	if system == "" {
		system = `C:\Windows`
	}
	executable := strings.TrimRight(system, `\/`) + `\System32\WindowsPowerShell\v1.0\powershell.exe`
	return executable, "-NoProfile -NonInteractive -WindowStyle Hidden -EncodedCommand " + base64.StdEncoding.EncodeToString(encoded)
}

func renderService(platform, name, launcher, instance, account string) ([]byte, error) {
	switch platform {
	case "linux":
		return []byte("[Unit]\nDescription=Dark Pawns private world\n[Service]\nType=simple\nExecStart=" + systemdArg(launcher) + " _serve --instance " + systemdArg(instance) + "\nTimeoutStopSec=45\nKillMode=mixed\n[Install]\nWantedBy=default.target\n"), nil
	case "darwin":
		return []byte(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>` + xmlText(name) + `</string><key>ProgramArguments</key><array><string>` + xmlText(launcher) + `</string><string>_serve</string><string>--instance</string><string>` + xmlText(instance) + `</string></array><key>RunAtLoad</key><true/><key>ExitTimeOut</key><integer>45</integer><key>StandardOutPath</key><string>` + xmlText(filepath.Join(instance, "lib", "logs", "launcher.log")) + `</string><key>StandardErrorPath</key><string>` + xmlText(filepath.Join(instance, "lib", "logs", "launcher.log")) + `</string></dict></plist>`), nil
	case "windows":
		command, args := windowsTaskAction(launcher, instance)
		return []byte(`<?xml version="1.0" encoding="UTF-8"?><Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task"><Triggers><LogonTrigger><Enabled>true</Enabled><UserId>` + xmlText(account) + `</UserId></LogonTrigger></Triggers><Principals><Principal id="Operator"><UserId>` + xmlText(account) + `</UserId><LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals><Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries><StopIfGoingOnBatteries>false</StopIfGoingOnBatteries><ExecutionTimeLimit>PT0S</ExecutionTimeLimit><AllowHardTerminate>false</AllowHardTerminate><Enabled>true</Enabled></Settings><Actions Context="Operator"><Exec><Command>` + xmlText(command) + `</Command><Arguments>` + xmlText(args) + `</Arguments></Exec></Actions></Task>`), nil
	}
	return nil, errors.New("unsupported service platform")
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	return errors.Join(writeErr, f.Sync(), f.Close())
}

func windowsTaskFile(data []byte) []byte {
	text := strings.Replace(string(data), `encoding="UTF-8"`, `encoding="UTF-16"`, 1)
	result := []byte{0xff, 0xfe}
	for _, v := range utf16.Encode([]rune(text)) {
		result = append(result, byte(v&0xff), byte(v>>8))
	}
	return result
}

// bootout can return before launchd finishes deregistration. Observe absence
// before bootstrap, and bound the native manager's teardown/registration window.
// Retrying this transition never changes the owned definition or force-kills it.
func startLaunchAgent(ctx context.Context, target, domain, path string, command func(context.Context, string, ...string) error) error {
	if err := command(ctx, "launchctl", "bootout", target); err != nil {
		if probe := command(ctx, "launchctl", "print", target); probe == nil {
			return err
		}
	}
	wait, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		if probe := command(wait, "launchctl", "print", target); probe != nil {
			if err := command(wait, "launchctl", "bootstrap", domain, path); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		select {
		case <-wait.Done():
			return errors.Join(errors.New("launchd did not finish reloading the owned job"), lastErr, wait.Err())
		case <-ticker.C:
		}
	}
}
