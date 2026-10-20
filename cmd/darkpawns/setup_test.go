package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSetupCancellationDoesNotCreateFolders(t *testing.T) {
	for _, input := range []string{"", "\n\nno\nno\nno\nno\n", "\n\nno\n", "\n\nno\nno\nno\nyesterday\nno\n"} {
		t.Run(strconv.Quote(input), func(t *testing.T) {
			bundle := t.TempDir()
			destination := filepath.Join(t.TempDir(), "missing parent", "world")
			p := setupPlan{Instance: destination, Port: freePort(t)}
			var out bytes.Buffer
			confirmed, err := setupPlain(context.Background(), bundle, &p, strings.NewReader(input), &out)
			if confirmed || (err != nil && !errors.Is(err, io.EOF)) {
				t.Fatalf("confirmed=%v err=%v", confirmed, err)
			}
			if _, err := os.Stat(filepath.Dir(destination)); !os.IsNotExist(err) {
				t.Fatal("cancellation created a folder")
			}
		})
	}
}

func TestSetupValidationAndReview(t *testing.T) {
	bundle := t.TempDir()
	root := t.TempDir()
	destination := filepath.Join(root, "my world")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	occupied := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	port := freePort(t)
	p := setupPlan{Instance: destination, Port: occupied}
	var out bytes.Buffer
	// Existing destination, corrected path, occupied port, corrected port, browser
	// choice and explicit review approval all pass through the actual line prompts.
	input := strings.Join([]string{root, destination, occupied, port, "0", "no", "no", "no", "yes"}, "\n") + "\n"
	confirmed, err := setupPlain(context.Background(), bundle, &p, strings.NewReader(input), &out)
	if err != nil || !confirmed {
		t.Fatalf("%v %v\n%s", confirmed, err, out.String())
	}
	for _, want := range []string{"already exists", "unavailable", "first character", destination, ":" + port + "/"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %s", want, out.String())
		}
	}
	if p.Port != port || p.Browser {
		t.Fatalf("incorrect plan: %+v", p)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("review wrote instance before final validation")
	}
}

func TestResolveDestination(t *testing.T) {
	bundle := t.TempDir()
	for _, path := range []string{bundle, filepath.Join(bundle, "missing", "world"), "", "bad\npath"} {
		if _, err := resolveDestination(bundle, path); err == nil {
			t.Fatalf("accepted %q", path)
		}
	}
	outsideParent := t.TempDir()
	outside := filepath.Join(outsideParent, "new parent", "world")
	canonicalParent, err := filepath.EvalSymlinks(outsideParent)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonicalParent, "new parent", "world")
	if got, err := resolveDestination(bundle, outside); err != nil || got != want {
		t.Fatalf("%q %v", got, err)
	}
	if runtime.GOOS != "windows" {
		outsideLink := filepath.Join(t.TempDir(), "outside-link")
		if err := os.Symlink(outsideParent, outsideLink); err != nil {
			t.Fatal(err)
		}
		if got, err := resolveDestination(bundle, filepath.Join(outsideLink, "new parent", "world")); err != nil || got != want {
			t.Fatalf("resolved outside link: %q %v", got, err)
		}
		link := filepath.Join(t.TempDir(), "release-link")
		if err := os.Symlink(bundle, link); err != nil {
			t.Fatal(err)
		}
		if _, err := resolveDestination(bundle, filepath.Join(link, "new", "world")); err == nil {
			t.Fatal("symlink into release accepted")
		}
	}
}

func TestPlainCancellationWhileWaiting(t *testing.T) {
	input, output := io.Pipe()
	defer input.Close()
	defer output.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := setupPlain(ctx, ".", &setupPlan{}, input, io.Discard); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled setup still waiting for input")
	}
}

func TestSetupRefusesNonTerminalAndOccupiedPortBeforeWrites(t *testing.T) {
	instance := filepath.Join(t.TempDir(), "parent", "world")
	bundle := t.TempDir()
	c := setupCommand(&instance, func() (string, error) { return bundle, nil })
	c.SetIn(strings.NewReader(""))
	c.SetOut(io.Discard)
	c.SetArgs(nil)
	if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "terminal") {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	c = setupCommand(&instance, func() (string, error) { return bundle, nil })
	c.SetOut(io.Discard)
	c.SetArgs([]string{"--yes", "--port", strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)})
	if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(instance)); !os.IsNotExist(err) {
		t.Fatal("failed preflight created parent")
	}
}

func TestDefaultInstanceAndHeadless(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LOCALAPPDATA", t.TempDir())
	case "linux":
		t.Setenv("XDG_DATA_HOME", t.TempDir())
	}
	path, err := defaultInstance()
	if err != nil || !filepath.IsAbs(path) || filepath.Base(path) != "world" || filepath.Base(filepath.Dir(path)) != "darkpawns" {
		t.Fatalf("%s %v", path, err)
	}
	t.Setenv("SSH_CONNECTION", "headless")
	if browserAvailable() {
		t.Fatal("SSH session should print a URL")
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	value := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestRestartCommandIncludesLauncherOutsidePath(t *testing.T) {
	launcher, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := restartCommand("world with spaces and 'quotes'")
	if !strings.Contains(command, launcher) || !strings.Contains(command, " run --instance ") {
		t.Fatal(command)
	}
	if runtime.GOOS == "windows" {
		if !strings.HasPrefix(command, "& '") || !strings.Contains(command, "''quotes''") {
			t.Fatal(command)
		}
	} else if !strings.Contains(command, "'\"'\"'") {
		t.Fatal(command)
	}
}

// The installer must use the actual login art, including its original alignment,
// rather than a separately drawn approximation. The same C fixture governs login.
func TestSetupUsesOriginalLoginArt(t *testing.T) {
	raw, err := os.ReadFile("../../pkg/session/testdata/c_greetings.txt")
	if err != nil {
		t.Fatal(err)
	}
	greeting := strings.TrimLeft(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	art, _, found := strings.Cut(greeting, "\n\n")
	if !found || setupLoginArt != art {
		t.Fatalf("installer splash differs from the original login artwork\ngot: %q\nwant: %q", setupLoginArt, art)
	}
}

func TestSetupTelnetValidationRefusesCollisionsAndAllowsDisable(t *testing.T) {
	p := setupPlan{Port: "4350"}
	for _, value := range []string{"4350", "04350", "-1", "65536", "no"} {
		if p.validateTelnet(value) == nil {
			t.Fatal(value)
		}
	}
	if err := p.validateTelnet("0"); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if p.validateTelnet(strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)) == nil {
		t.Fatal("occupied telnet port accepted")
	}
}
