package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/portable"
)

func desktopTestInstance(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("HOME", base)
	instance, err := defaultInstance()
	if err != nil {
		t.Fatal(err)
	}
	return instance
}

func writeDesktopInstance(t *testing.T, instance string) []byte {
	t.Helper()
	if err := os.MkdirAll(instance, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(portable.Config{Format: portable.Format, HTTPPort: 54321, JWTSecret: strings.Repeat("ab", 32)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, "instance.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDesktopLaunchCreatesNothingWithoutSetupReview(t *testing.T) {
	instance := desktopTestInstance(t)
	root := command()
	root.SetIn(strings.NewReader("")) // No terminal: actual setup must refuse before writing.
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	var errOut bytes.Buffer
	code := runLauncher(nil, true, func(args []string) error {
		root.SetArgs(args)
		return root.ExecuteContext(context.Background())
	}, strings.NewReader("\n"), io.Discard, &errOut)
	if code != 1 || !strings.Contains(errOut.String(), "setup needs a terminal") {
		t.Fatalf("exit=%d error=%s", code, errOut.String())
	}
	if _, err := os.Stat(instance); !os.IsNotExist(err) {
		t.Fatalf("created world without review: %v", err)
	}
}

func TestDesktopExistingWorldOpensReadyAddressAndWaits(t *testing.T) {
	instance := desktopTestInstance(t)
	original := writeDesktopInstance(t, instance)
	var order []string
	var out bytes.Buffer
	cmd := desktopCommand(func() (string, error) { return "bundle", nil }, func(_ context.Context, base, path string) (portable.RuntimeStatus, error) {
		if base != "bundle" || path != instance {
			t.Fatalf("wrong target: %s %s", base, path)
		}
		order = append(order, "start")
		// Start handles both initial states and returns only after readiness.
		return portable.RuntimeStatus{State: "running", URL: "http://127.0.0.1:54321/"}, nil
	}, func(_ context.Context, url string) error {
		if url != "http://127.0.0.1:54321/" {
			t.Fatalf("opened %q instead of configured home page", url)
		}
		order = append(order, "browse")
		return nil
	})
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	code := runLauncher(nil, true, func([]string) error { return cmd.ExecuteContext(context.Background()) }, strings.NewReader("\n"), &out, &out)
	if code != 0 || strings.Join(order, ",") != "start,browse" || !strings.Contains(out.String(), "Press Enter to close") || strings.Contains(strings.ToLower(out.String()), "setup") {
		t.Fatalf("exit=%d calls=%v output=%s", code, order, out.String())
	}
	data, err := os.ReadFile(filepath.Join(instance, "instance.json"))
	if err != nil || !bytes.Equal(original, data) {
		t.Fatalf("configuration changed: %s %v", data, err)
	}
}

func TestDesktopExistingWorldFailuresStayVisibleAndNeverReinstall(t *testing.T) {
	for _, failure := range []string{"invalid world", "start failed", "browser failed", "not ready"} {
		t.Run(failure, func(t *testing.T) {
			instance := desktopTestInstance(t)
			original := writeDesktopInstance(t, instance)
			if failure == "invalid world" {
				original = []byte("invalid configuration")
				if err := os.WriteFile(filepath.Join(instance, "instance.json"), original, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var starts, browsers int
			var out bytes.Buffer
			cmd := desktopCommand(func() (string, error) { return "bundle", nil }, func(context.Context, string, string) (portable.RuntimeStatus, error) {
				starts++
				if failure == "start failed" {
					return portable.RuntimeStatus{}, errors.New("occupied port")
				}
				if failure == "not ready" {
					return portable.RuntimeStatus{State: "starting"}, nil
				}
				return portable.RuntimeStatus{State: "running", URL: "http://127.0.0.1:54321/"}, nil
			}, func(context.Context, string) error { browsers++; return errors.New("no browser") })
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{})
			code := runLauncher(nil, true, func([]string) error { return cmd.ExecuteContext(context.Background()) }, strings.NewReader("\n"), &out, &out)
			if code != 1 || !strings.Contains(out.String(), "Press Enter to close") {
				t.Fatalf("exit=%d output=%s", code, out.String())
			}
			if failure == "invalid world" && starts != 0 || failure != "browser failed" && browsers != 0 {
				t.Fatalf("starts=%d browsers=%d", starts, browsers)
			}
			if failure == "browser failed" && (!strings.Contains(out.String(), "Open http://127.0.0.1:54321/ manually") || browsers != 1) {
				t.Fatal(out.String())
			}
			data, err := os.ReadFile(filepath.Join(instance, "instance.json"))
			if err != nil || !bytes.Equal(data, original) {
				t.Fatalf("existing world changed: %v", err)
			}
		})
	}
}

func TestDesktopDispatchStaysInAnchor(t *testing.T) {
	instance := desktopTestInstance(t)
	if err := os.MkdirAll(filepath.Join(instance, ".updates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, ".updates", "active.json"), []byte("bad active release"), 0o600); err != nil {
		t.Fatal(err)
	}
	// _open is new in this anchor. An older active release cannot dispatch it.
	// Start will resolve/validate that active release before starting a world.
	forwarded, err := forwardActive(command(), []string{"_open"})
	if forwarded || err != nil {
		t.Fatalf("forwarded=%v err=%v", forwarded, err)
	}
	if _, err := forwardActive(command(), []string{"status"}); err == nil {
		t.Fatal("negative control: status did not inspect the damaged active release")
	}
}
