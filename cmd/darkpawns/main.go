// darkpawns is the portable operator launcher. The legacy cmd/server flags and
// checkout startup remain unchanged; the wizard will build on this boundary.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

var (
	version = "dev"
	commit  = "unknown"
)

func command() *cobra.Command {
	root := &cobra.Command{Use: "darkpawns", Short: "Run a private Dark Pawns instance from a portable release", SilenceUsage: true, SilenceErrors: true, Version: version + " (" + commit + ")"}
	var instance string
	root.PersistentFlags().StringVar(&instance, "instance", "", "Instance directory outside the release (required for init/run)")
	bundle := func() (string, error) {
		path, err := os.Executable()
		if err != nil {
			return "", err
		}
		path, err = filepath.EvalSymlinks(path)
		return filepath.Dir(path), err
	}
	validate := func() error {
		if instance == "" {
			return fmt.Errorf("choose an instance directory with --instance <path>")
		}
		return nil
	}
	initCmd := &cobra.Command{Use: "init", Short: "Create a new instance; never overwrite an existing directory", Args: cobra.NoArgs}
	port := initCmd.Flags().Int("port", 4350, "Local web client port")
	initCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if err := validate(); err != nil {
			return err
		}
		base, err := bundle()
		if err != nil {
			return err
		}
		if err := portable.Init(base, instance, *port); err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Instance created. Run darkpawns run --instance <this directory> to start it.")
		return err
	}
	runCmd := &cobra.Command{Use: "run", Short: "Run an existing instance in the foreground on localhost", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := validate(); err != nil {
			return err
		}
		base, err := bundle()
		if err != nil {
			return err
		}
		return portable.Run(cmd.Context(), base, instance, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	}}
	root.AddCommand(initCmd, runCmd)
	return root
}

func execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return command().ExecuteContext(ctx)
}

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, "darkpawns:", err)
		os.Exit(1)
	}
}
