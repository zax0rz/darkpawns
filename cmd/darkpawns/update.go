package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zax0rz/darkpawns/internal/portable"
)

func addUpdate(root *cobra.Command, instance *string, bundle func() (string, error)) {
	choose := func() (string, error) {
		if *instance != "" {
			return *instance, nil
		}
		return defaultInstance()
	}
	var archive, digest string
	var yes, check bool
	cmd := &cobra.Command{Use: "update", Short: "Verify, back up and update your portable instance", Args: cobra.NoArgs}
	cmd.Flags().StringVar(&archive, "archive", "", "Use a local release archive instead of the latest stable GitHub release")
	cmd.Flags().StringVar(&digest, "sha256", "", "Expected SHA-256 for a local archive (required with --archive)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Approve the update and graceful restart without prompting")
	cmd.Flags().BoolVar(&check, "check", false, "Verify the available release without stopping or updating your world")
	cmd.MarkFlagsRequiredTogether("archive", "sha256")
	cmd.RunE = func(cmd *cobra.Command, _ []string) (result error) {
		path, err := choose()
		if err != nil {
			return err
		}
		base, err := bundle()
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Checking and staging the release. Your world keeps running during verification."); err != nil {
			return err
		}
		plan, err := portable.PrepareUpdate(cmd.Context(), base, path, portable.UpdateOptions{Archive: archive, SHA256: digest})
		if err != nil {
			return err
		}
		defer func() { result = errors.Join(result, plan.Close()) }()
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Release: %s -> %s\nArchive SHA-256: %s\n", plan.PreviousVersion, plan.Version, plan.SHA256); err != nil {
			return err
		}
		if plan.UpToDate {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Already up to date. Your world was not stopped.")
			return err
		}
		if check {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Release verified. No update applied.")
			return err
		}
		if !yes {
			if _, err := fmt.Fprint(cmd.OutOrStdout(), "This stops the world and disconnects players. A private backup is retained.\nIf startup fails, the previous snapshot is restored; attempted state is retained for inspection.\nApply update? [y/N]: "); err != nil {
				return err
			}
			answer, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
			if err != nil || strings.ToLower(strings.TrimSpace(answer)) != "y" {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Update cancelled. Your world was not stopped.")
				return err
			}
		}
		if err := plan.Apply(cmd.Context()); err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Update complete. Release and backup are retained in your instance's .updates folder.")
		return err
	}
	recoverCmd := &cobra.Command{Use: "recover", Short: "Restore the previous snapshot after an interrupted update", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := choose()
		if err != nil {
			return err
		}
		if err := portable.RecoverUpdate(cmd.Context(), path); err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Previous release restored. Backup and attempted state retained.")
		return err
	}}
	cmd.AddCommand(recoverCmd)
	root.AddCommand(cmd)
	var transaction string
	serve := &cobra.Command{Use: "_serve-update", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		path, err := choose()
		if err != nil {
			return err
		}
		base, err := bundle()
		if err != nil {
			return err
		}
		return portable.ServeUpdateBackground(cmd.Context(), base, path, transaction)
	}}
	serve.Flags().StringVar(&transaction, "transaction", "", "Pending update transaction")
	root.AddCommand(serve)
}

// Dispatch before Cobra resolves commands, so updated help, --version and newly
// added commands are available through the original PATH/native-job anchor.
func forwardActive(root *cobra.Command, args []string) (bool, error) {
	flags := pflag.NewFlagSet("entry", pflag.ContinueOnError)
	flags.SetOutput(io.Discard) // Cobra owns help and usage; this pass only finds dispatch flags.
	flags.ParseErrorsAllowlist.UnknownFlags = true
	selected := flags.String("instance", "", "Instance directory")
	if err := flags.Parse(args); err != nil {
		return false, nil
	} // Cobra owns usage errors.
	operations := flags.Args()
	if len(operations) > 0 {
		first := operations[0]
		// The anchor owns Explorer dispatch, including for older active releases
		// which do not know _open. Start still resolves the world's active bundle.
		if first == "init" || first == "setup" || first == "_open" || first == "_serve-update" || (first == "update" && len(operations) > 1 && operations[1] == "recover") {
			return false, nil
		}
	}
	instance := *selected
	if instance == "" {
		var err error
		instance, err = defaultInstance()
		if err != nil {
			return false, err
		}
	}
	if portable.PendingUpdate(instance) != nil {
		return false, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return false, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return false, err
	}
	base := filepath.Dir(executable)
	active, err := portable.ActiveBundle(base, instance)
	if err != nil {
		return false, err
	}
	active, err = filepath.EvalSymlinks(active)
	if err != nil {
		return false, err
	}
	if active == base {
		return false, nil
	}
	executable = filepath.Join(active, "darkpawns")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	return true, portable.ForwardOperator(root.Context(), executable, args, root.InOrStdin(), root.OutOrStdout(), root.ErrOrStderr())
}
