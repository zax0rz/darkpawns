package portable

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func TestNativeServiceDefinitionsPreserveArguments(t *testing.T) {
	launcher := `/release with space/$HOME%/darkpawns`
	instance := `/world with <tags> & "quotes"/world`
	for _, platform := range []string{"linux", "darwin", "windows"} {
		data, err := renderService(platform, "darkpawns-test", launcher, instance, "operator")
		if err != nil {
			t.Fatal(err)
		}
		if platform == "linux" {
			text := string(data)
			if !strings.Contains(text, "$$HOME%%") || !strings.Contains(text, `\"quotes\"`) || !strings.Contains(text, "KillMode=mixed") {
				t.Fatal(text)
			}
			continue
		}
		decoder := xml.NewDecoder(bytes.NewReader(data))
		var text []string
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if chars, ok := token.(xml.CharData); ok {
				text = append(text, string(chars))
			}
		}
		joined := strings.Join(text, "\n")
		if platform == "windows" {
			var task struct {
				Actions struct {
					Exec struct {
						Command   string
						Arguments string
					}
				}
			}
			if err := xml.Unmarshal(data, &task); err != nil {
				t.Fatal(err)
			}
			encoded := strings.TrimPrefix(task.Actions.Exec.Arguments, "-NoProfile -NonInteractive -WindowStyle Hidden -EncodedCommand ")
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatal(err)
			}
			words := make([]uint16, len(data)/2)
			for i := range words {
				words[i] = binary.LittleEndian.Uint16(data[2*i:])
			}
			script := string(utf16.Decode(words))
			if !strings.Contains(script, launcher) || !strings.Contains(script, instance) {
				t.Fatal(script)
			}
			if strings.Contains(task.Actions.Exec.Arguments, "%") || !strings.HasSuffix(task.Actions.Exec.Command, `\powershell.exe`) {
				t.Fatal(task)
			}
		} else if !strings.Contains(joined, launcher) || !strings.Contains(joined, instance) {
			t.Fatal(joined)
		}
	}
}

func TestServiceOwnershipRejectsChangedDefinition(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux file-backed registration unit")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	instance, err := canonicalInstance(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	name := serviceName(instance)
	path, err := servicePath(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeExclusive(path, []byte("external edit")); err != nil {
		t.Fatal(err)
	}
	record := serviceRecord{Platform: runtime.GOOS, Name: name, Path: path, Launcher: filepath.Join(instance, "launcher"), SHA256: strings.Repeat("0", 64)}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, ".service.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ServiceInstalled(instance); err == nil {
		t.Fatal("modified definition treated as owned")
	}
}

func TestLaunchAgentWaitsForDeregistrationAndBootstrap(t *testing.T) {
	prints, bootstraps := 0, 0
	err := startLaunchAgent(context.Background(), "gui/1/owned", "gui/1", "owned.plist", func(_ context.Context, _ string, args ...string) error {
		switch args[0] {
		case "bootout":
			return nil
		case "print":
			prints++
			if prints == 1 {
				return nil
			}
			return errors.New("not registered")
		case "bootstrap":
			bootstraps++
			if bootstraps == 1 {
				return errors.New("teardown still pending")
			}
			return nil
		}
		t.Fatalf("unexpected native operation: %v", args)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if prints != 3 || bootstraps != 2 {
		t.Fatalf("did not observe the manager transition: prints=%d bootstrap=%d", prints, bootstraps)
	}
}

func TestLaunchAgentReloadFailureRemainsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := startLaunchAgent(ctx, "gui/1/owned", "gui/1", "owned.plist", func(_ context.Context, _ string, args ...string) error {
		if args[0] == "bootout" {
			return nil
		}
		return errors.New("manager unavailable")
	})
	if err == nil {
		t.Fatal("failed native manager transition reported success")
	}
}

func TestQueryStartupTaskActionRetriesTransientExecFailureOnce(t *testing.T) {
	defer func(prev func(string) ([]byte, error)) { runWindowsTaskQuery = prev }(runWindowsTaskQuery)
	calls := 0
	runWindowsTaskQuery = func(_ string) ([]byte, error) {
		calls++
		if calls == 1 {
			return nil, errors.New(`query owned startup task: exec: canceling Cmd: TerminateProcess: Access is denied.: {"Command":"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe","Arguments":"-NoProfile"}`)
		}
		return []byte("\xef\xbb\xbf" + `{"Command":"C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe","Arguments":"-NoProfile -NonInteractive"}`), nil
	}

	action, err := queryStartupTaskActionWithRetry("darkpawns-test")
	if err != nil {
		t.Fatalf("queryStartupTaskActionWithRetry = %v, want success on retry", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if !strings.HasSuffix(action.Command, `\powershell.exe`) || action.Arguments != "-NoProfile -NonInteractive" {
		t.Fatalf("unexpected action: %+v", action)
	}
}

func TestQueryStartupTaskActionFailsAfterTwoExecAttempts(t *testing.T) {
	defer func(prev func(string) ([]byte, error)) { runWindowsTaskQuery = prev }(runWindowsTaskQuery)
	calls := 0
	runWindowsTaskQuery = func(_ string) ([]byte, error) {
		calls++
		return nil, errors.New("query owned startup task: exit status 1:")
	}

	_, err := queryStartupTaskActionWithRetry("darkpawns-test")
	if err == nil {
		t.Fatal("queryStartupTaskActionWithRetry succeeded when both attempts failed")
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if !strings.Contains(err.Error(), "two attempts") {
		t.Fatalf("error = %v, want two-attempt diagnosis", err)
	}
}

func TestQueryStartupTaskActionDoesNotRetryMalformedJSON(t *testing.T) {
	defer func(prev func(string) ([]byte, error)) { runWindowsTaskQuery = prev }(runWindowsTaskQuery)
	calls := 0
	runWindowsTaskQuery = func(_ string) ([]byte, error) {
		calls++
		return []byte("not-json"), nil
	}

	_, err := queryStartupTaskActionWithRetry("darkpawns-test")
	if err == nil {
		t.Fatal("queryStartupTaskActionWithRetry succeeded on malformed JSON")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (malformed JSON must not retry)", calls)
	}
}
