package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

// Explorer opens the default world. Explicit terminal/shortcut commands retain
// their own behavior. Start is idempotent and waits for readiness; it never
// restarts a running world just to open its web page.
func desktopCommand(bundle func() (string, error), start func(context.Context, string, string) (portable.RuntimeStatus, error), browse func(context.Context, string) error) *cobra.Command {
	cmd := &cobra.Command{Use: "_open", Hidden: true, Args: cobra.NoArgs}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		instance, err := defaultInstance()
		if err != nil {
			return err
		}
		if _, err := os.Lstat(instance); os.IsNotExist(err) {
			setup := setupCommand(&instance, bundle)
			setup.SetIn(cmd.InOrStdin())
			setup.SetOut(cmd.OutOrStdout())
			setup.SetErr(cmd.ErrOrStderr())
			setup.SetArgs([]string{})
			return setup.ExecuteContext(cmd.Context())
		} else if err != nil {
			return err
		}
		// An existing but incomplete/invalid directory is never a fresh install.
		// Report its error and leave it intact, instead of offering to overwrite it.
		if _, err := portable.Load(instance); err != nil {
			return fmt.Errorf("open world at %q: %w", instance, err)
		}
		base, err := bundle()
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Dark Pawns\n\nOpening your world. Waiting for the web client..."); err != nil {
			return err
		}
		status, err := start(cmd.Context(), base, instance)
		if err != nil {
			return err
		}
		if status.State != "running" || status.URL == "" {
			return fmt.Errorf("world is %s without a ready web address; run %s", status.State, operatorCommand(instance, "doctor"))
		}
		url := strings.TrimSuffix(status.URL, "/") + "/"
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\nReady: %s\nYour world is running in the background.\nStop with: %s\n", url, operatorCommand(instance, "stop")); err != nil {
			return err
		}
		if err := browse(cmd.Context(), url); err != nil {
			return fmt.Errorf("could not open a browser: %w\nOpen %s manually. Your world is still running", err, url)
		}
		return nil
	}
	return cmd
}
