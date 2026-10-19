package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

type setupPlan struct {
	Instance string
	Port     string
	Browser  bool
}

func defaultInstance() (string, error) {
	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is unset; choose a folder with --instance")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support")
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base != "" && !filepath.IsAbs(base) {
			return "", errors.New("XDG_DATA_HOME must be absolute; choose a folder with --instance")
		}
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("default data folder must be absolute; choose a folder with --instance")
	}
	return filepath.Join(base, "darkpawns", "world"), nil
}

// resolveDestination validates without creating anything. Resolve the nearest
// existing ancestor to detect symlinks into a replaceable release, including
// when the selected parent directories have not been created yet.
func resolveDestination(bundle, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("choose a new instance folder")
	}
	if strings.ContainsFunc(value, unicode.IsControl) {
		return "", errors.New("folder contains control characters")
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(path); err == nil {
		return "", errors.New("folder already exists; choose a new folder, or use darkpawns run --instance to resume it")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	ancestor := filepath.Dir(path)
	tail := []string{filepath.Base(path)}
	for {
		info, err := os.Stat(ancestor)
		if err == nil {
			if !info.IsDir() {
				return "", errors.New("instance parent is not a folder")
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if filepath.Dir(ancestor) == ancestor {
			return "", err
		}
		tail = append(tail, filepath.Base(ancestor))
		ancestor = filepath.Dir(ancestor)
	}
	ancestor, err = filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", err
	}
	for i := len(tail) - 1; i >= 0; i-- {
		ancestor = filepath.Join(ancestor, tail[i])
	}
	realBundle, err := filepath.EvalSymlinks(bundle)
	if err != nil {
		return "", err
	}
	realBundle, err = filepath.Abs(realBundle)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(realBundle, ancestor)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("choose an instance folder outside the replaceable release folder")
	}
	return ancestor, nil
}

func availablePort(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("choose a port between 1 and 65535")
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("port %d is unavailable; choose another port: %w", port, err)
	}
	return listener.Close()
}

func (p setupPlan) summary() string {
	browser := "No (the address will be printed)"
	if p.Browser {
		browser = "Yes, when the server is ready"
	}
	return fmt.Sprintf("Instance: %s\nWeb client: http://127.0.0.1:%s/\nOpen browser: %s\n\nThe first character you create becomes the immortal.\nCreate it before allowing other people to connect.\nKeep this terminal open while playing; Ctrl+C saves and stops the server.", p.Instance, p.Port, browser)
}

func setupForm(ctx context.Context, bundle string, p *setupPlan, in io.Reader, out io.Writer) (bool, error) {
	confirmed := false
	form := huh.NewForm(
		huh.NewGroup(huh.NewInput().Title("1. Choose a home").Description("A new folder for your world, players and settings. Keep it outside the release folder.").Value(&p.Instance).Validate(func(value string) error { _, err := resolveDestination(bundle, value); return err })),
		huh.NewGroup(huh.NewInput().Title("2. Choose a port").Description("Local browser access only. The default is 4350.").Value(&p.Port).Validate(availablePort)),
		huh.NewGroup(
			huh.NewConfirm().Title("Open your browser after launch?").Value(&p.Browser),
			huh.NewConfirm().Title("3. Create instance and launch?").Description(p.summary()).DescriptionFunc(func() string { return p.summary() }, p).Affirmative("Create and launch").Negative("Cancel").Value(&confirmed),
		),
	).WithInput(in).WithOutput(out).WithTheme(huh.ThemeBase()).WithAccessible(false)
	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}
	return confirmed, nil
}

// Plain mode deliberately uses one shared scanner. EOF is cancellation, never
// an implicit approval; buffered input cannot be lost between prompts.
func setupPlain(ctx context.Context, bundle string, p *setupPlan, in io.Reader, out io.Writer) (bool, error) {
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	type line struct {
		value string
		err   error
	}
	lines := make(chan line)
	go func() {
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			select {
			case lines <- line{value: scanner.Text()}:
			case <-readCtx.Done():
				return
			}
		}
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		}
		select {
		case lines <- line{err: err}:
		case <-readCtx.Done():
		}
	}()
	ask := func(label, fallback string, validate func(string) error) (string, error) {
		for {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if _, err := fmt.Fprintf(out, "%s [%s]: ", label, fallback); err != nil {
				return "", err
			}
			var value string
			select {
			case next := <-lines:
				if next.err != nil {
					return "", next.err
				}
				value = next.value
			case <-ctx.Done():
				return "", ctx.Err()
			}
			if value == "" {
				value = fallback
			}
			if validate == nil {
				return value, nil
			}
			if err := validate(value); err == nil {
				return value, nil
			} else {
				if _, err := fmt.Fprintln(out, err); err != nil {
					return "", err
				}
			}
		}
	}
	var err error
	p.Instance, err = ask("1. New instance folder", p.Instance, func(value string) error { _, err := resolveDestination(bundle, value); return err })
	if err != nil {
		return false, err
	}
	p.Port, err = ask("2. Local web port", p.Port, availablePort)
	if err != nil {
		return false, err
	}
	boolean := func(value string) error {
		switch strings.ToLower(value) {
		case "y", "yes", "n", "no":
			return nil
		}
		return errors.New("enter yes or no")
	}
	fallback := "no"
	if p.Browser {
		fallback = "yes"
	}
	browser, err := ask("Open browser after launch?", fallback, boolean)
	if err != nil {
		return false, err
	}
	p.Browser = strings.EqualFold(browser, "yes") || strings.EqualFold(browser, "y")
	if _, err := fmt.Fprintln(out, "\n"+p.summary()); err != nil {
		return false, err
	}
	answer, err := ask("3. Create instance and launch?", "no", boolean)
	return strings.EqualFold(answer, "yes") || strings.EqualFold(answer, "y"), err
}

func setupCommand(instance *string, bundle func() (string, error)) *cobra.Command {
	var port int
	var plain, yes, noBrowser bool
	cmd := &cobra.Command{Use: "setup", Short: "Choose a home and port, then launch your new world", Args: cobra.NoArgs}
	cmd.Flags().IntVar(&port, "port", 4350, "Initial local web client port")
	cmd.Flags().BoolVar(&plain, "plain", false, "Use line prompts for screen readers or a basic terminal")
	cmd.Flags().BoolVar(&yes, "yes", false, "Create and launch using the selected values without prompts")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the address without opening a browser")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		base, err := bundle()
		if err != nil {
			return err
		}
		destination := *instance
		if destination == "" {
			destination, err = defaultInstance()
			if err != nil {
				return err
			}
		}
		p := setupPlan{Instance: destination, Port: strconv.Itoa(port), Browser: !noBrowser && browserAvailable()}
		out := cmd.OutOrStdout()
		if _, err := fmt.Fprint(out, "Dark Pawns setup\nA local world with its own players and settings.\n\n"); err != nil {
			return err
		}
		if !yes {
			var confirmed bool
			file, ok := cmd.InOrStdin().(*os.File)
			if !plain && (!ok || !term.IsTerminal(file.Fd())) {
				return errors.New("setup needs a terminal; use --plain for line prompts or --yes for explicit unattended setup")
			}
			compact := false
			if ok {
				width, height, sizeErr := term.GetSize(file.Fd())
				compact = sizeErr == nil && (width < 60 || height < 24)
			}
			if plain || compact || os.Getenv("TERM") == "dumb" || os.Getenv("NO_COLOR") != "" {
				confirmed, err = setupPlain(cmd.Context(), base, &p, cmd.InOrStdin(), out)
			} else {
				confirmed, err = setupForm(cmd.Context(), base, &p, cmd.InOrStdin(), out)
			}
			if errors.Is(err, io.EOF) || (err == nil && !confirmed) {
				_, err = fmt.Fprintln(out, "Setup cancelled. No instance was created.")
				return err
			}
			if err != nil {
				return err
			}
		} else if _, err := fmt.Fprintln(out, p.summary()); err != nil {
			return err
		}
		if err := cmd.Context().Err(); err != nil {
			return err
		}
		p.Instance, err = resolveDestination(base, p.Instance)
		if err != nil {
			return err
		}
		if err := availablePort(p.Port); err != nil {
			return err
		}
		m, err := portable.ReadManifest(base)
		if err != nil {
			return err
		}
		if err := portable.Verify(base, m); err != nil {
			return err
		}
		// Only after review and verification may setup create missing parent folders.
		if err := os.MkdirAll(filepath.Dir(p.Instance), 0o700); err != nil {
			return fmt.Errorf("create instance parent: %w", err)
		}
		port, err := strconv.Atoi(p.Port)
		if err != nil {
			return err
		}
		if err := portable.Init(base, p.Instance, port); err != nil {
			return fmt.Errorf("initialize instance: %w", err)
		}
		if _, err := fmt.Fprintf(out, "\nInstance created: %s\nRestart with: %s\nStarting server. Keep this terminal open; press Ctrl+C to save and stop.\n", p.Instance, restartCommand(p.Instance)); err != nil {
			return err
		}
		err = portable.RunWithReady(cmd.Context(), base, p.Instance, cmd.InOrStdin(), out, cmd.ErrOrStderr(), func(url string) error {
			if _, err := fmt.Fprintf(out, "\nReady: %s/\nCreate your first character in the web client to become the immortal.\n", url); err != nil {
				return err
			}
			if p.Browser {
				if err := openBrowser(cmd.Context(), url+"/"); err != nil {
					_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Could not open a browser: %v\nOpen %s/ manually. The server is still running.\n", err, url)
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("server launch failed; your instance is retained at %q. Resolve the error above, then use %s: %w", p.Instance, restartCommand(p.Instance), err)
		}
		return nil
	}
	return cmd
}

func browserAvailable() bool {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false
	}
	return runtime.GOOS != "linux" || os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func openBrowser(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32.exe", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	return exec.CommandContext(ctx, name, args...).Run()
}

// Commands are quoted for the operator's platform shell, not Go string syntax.
func restartCommand(instance string) string {
	launcher, err := os.Executable()
	if err != nil {
		launcher = "darkpawns"
	}
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	prefix := ""
	if runtime.GOOS == "windows" {
		quote = func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
		prefix = "& "
	}
	return prefix + quote(launcher) + " run --instance " + quote(instance)
}
