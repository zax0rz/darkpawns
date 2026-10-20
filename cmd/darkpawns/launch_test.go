package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

type consoleGate struct {
	reading chan struct{}
	release chan struct{}
}

func (g consoleGate) Read(p []byte) (int, error) {
	close(g.reading)
	<-g.release
	p[0] = '\n'
	return 1, nil
}

func TestExplorerSetupWaitsAfterSuccessAndError(t *testing.T) {
	previous := cobra.MousetrapHelpText
	t.Cleanup(func() { cobra.MousetrapHelpText = previous })
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[failure], func(t *testing.T) {
			gate := consoleGate{reading: make(chan struct{}), release: make(chan struct{})}
			defer close(gate.release)
			var out, errOut bytes.Buffer
			cobra.MousetrapHelpText = "This is a command line tool."
			done := make(chan int, 1)
			go func() {
				done <- runLauncher(nil, true, func(args []string) error {
					if cobra.MousetrapHelpText != "" {
						return errors.New("Cobra mousetrap was not disabled before setup")
					}
					if !reflect.DeepEqual(args, []string{"setup"}) {
						return errors.New("Explorer did not select setup")
					}
					if failure {
						return errors.New("setup failed")
					}
					return nil
				}, gate, &out, &errOut)
			}()
			select {
			case <-gate.reading:
			case <-time.After(3 * time.Second):
				t.Fatal("did not wait for console input")
			}
			if !strings.Contains(out.String(), "Press Enter to close this window") {
				t.Fatal(out.String())
			}
			if failure && !strings.Contains(errOut.String(), "darkpawns: setup failed") {
				t.Fatal("error was not visible before the pause")
			}
			select {
			case <-done:
				t.Fatal("console closed without Enter")
			default:
			}
			// Hand the console a newline without sleeps; Read cannot return before this.
			gate.release <- struct{}{}
			select {
			case code := <-done:
				want := 0
				if failure {
					want = 1
				}
				if code != want {
					t.Fatalf("exit=%d want=%d", code, want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Enter did not close the console")
			}
		})
	}
}

type unexpectedRead struct{}

func (unexpectedRead) Read([]byte) (int, error) { panic("terminal launch attempted a console pause") }

func TestTerminalLaunchKeepsArgumentsAndDoesNotPause(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"--version"}, {"status", "--instance", "a world"}, {"_serve", "--instance", "a world"}} {
		var got []string
		var out, errOut bytes.Buffer
		code := runLauncher(args, false, func(selected []string) error { got = selected; return nil }, unexpectedRead{}, &out, &errOut)
		if code != 0 || !reflect.DeepEqual(got, args) || out.Len() != 0 || errOut.Len() != 0 {
			t.Fatalf("args=%v selected=%v exit=%d output=%q/%q", args, got, code, out.String(), errOut.String())
		}
	}
}

func TestExplorerShortcutKeepsExplicitArguments(t *testing.T) {
	args := []string{"setup", "--instance", "another world", "--plain"}
	var got []string
	code := runLauncher(args, true, func(selected []string) error { got = selected; return nil }, strings.NewReader("\r\n"), io.Discard, io.Discard)
	if code != 0 || !reflect.DeepEqual(got, args) {
		t.Fatalf("selected=%v exit=%d", got, code)
	}
}

func TestExplorerActuallyReachesSetupAndPreservesExistingWorld(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new world cancelled", true: "existing world refused"}[existing], func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "world")
			marker := filepath.Join(destination, "keep.txt")
			if existing {
				if err := os.Mkdir(destination, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(marker, []byte("existing player state"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			code := runLauncher(nil, true, func(args []string) error {
				root := command()
				root.SetArgs(append(args, "--plain", "--instance", destination))
				root.SetIn(strings.NewReader("\n"))
				root.SetOut(&out)
				root.SetErr(&out)
				return root.ExecuteContext(context.Background())
			}, strings.NewReader("\n"), &out, &out)
			if code != 0 || !strings.Contains(out.String(), "Setup cancelled. No instance was created.") || !strings.Contains(out.String(), "Press Enter to close this window") {
				t.Fatalf("exit=%d output=%s", code, out.String())
			}
			if existing {
				data, err := os.ReadFile(marker)
				if err != nil || string(data) != "existing player state" {
					t.Fatal("existing world changed", err)
				}
				if !strings.Contains(out.String(), "already exists") {
					t.Fatal("existing destination not refused", out.String())
				}
			} else if _, err := os.Stat(destination); !os.IsNotExist(err) {
				t.Fatal("cancel created a world", err)
			}
		})
	}
}

type failedConsole struct{}

func (failedConsole) Read([]byte) (int, error) { return 0, errors.New("console unavailable") }
func TestExplorerClosedAndFailedInput(t *testing.T) {
	for _, tc := range []struct {
		in   io.Reader
		want int
	}{{strings.NewReader(""), 0}, {failedConsole{}, 1}} {
		var out bytes.Buffer
		code := runLauncher(nil, true, func([]string) error { return nil }, tc.in, &out, &out)
		if code != tc.want {
			t.Fatalf("exit=%d want=%d", code, tc.want)
		}
	}
}
