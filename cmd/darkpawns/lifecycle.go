package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

func addLifecycle(root *cobra.Command, instance *string, bundle func() (string, error)) {
	choose := func(cmd *cobra.Command) (string, error) { return chooseInstance(cmd, *instance) }
	for _, name := range []string{"start", "stop", "restart", "status", "doctor", "_serve", "supervise"} {
		var asJSON, watch, yes bool
		cmd := &cobra.Command{Use: name, Args: cobra.NoArgs, Hidden: name == "_serve"}
		switch name {
		case "supervise":
			cmd.Short = "Keep a hosted world running under an external service manager"
		case "start":
			cmd.Short = "Start your world in the background and wait until ready"
		case "stop":
			cmd.Short = "Save and stop your world gracefully"
		case "restart":
			cmd.Short = "Save, stop and start your world"
		case "status":
			cmd.Short = "Show the state of your instance"
		case "doctor":
			cmd.Short = "Check your release, instance, database and startup registration"
		}
		if name == "doctor" || name == "status" {
			cmd.Flags().BoolVar(&asJSON, "json", false, "Print structured output")
		}
		if name == "status" {
			cmd.Flags().BoolVar(&watch, "watch", false, "Watch live status, players and the launcher log (TTY only)")
		}
		if name == "restart" {
			cmd.Flags().BoolVar(&yes, "yes", false, "Restart without interactive confirmation")
		}
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			path, err := choose(cmd)
			if errors.Is(err, errCancelled) {
				return nil
			}
			if err != nil {
				return err
			}
			base, err := bundle()
			if err != nil {
				return err
			}
			if name == "status" && watch && !asJSON && interactiveOutput(cmd) {
				return watchStatus(cmd, path, base)
			}
			if name == "restart" && !yes && interactiveOutput(cmd) {
				status, statusErr := portable.Status(cmd.Context(), path)
				if statusErr != nil {
					return statusErr
				}
				reader, loginErr := signInAdmin(cmd, path, status)
				if loginErr != nil {
					return loginErr
				}
				if reader != nil {
					defer reader.client.CloseIdleConnections()
				}
				snapshot := readCLISnapshot(cmd.Context(), path, base, reader)
				approved, confirmErr := confirmAction(cmd, "Restart this instance?", onlineImpact(snapshot)+"\nYour world will stop and start again.")
				if confirmErr != nil {
					return confirmErr
				}
				if !approved {
					_, err := fmt.Fprintln(cmd.OutOrStdout(), "Restart cancelled. Your world was not stopped.")
					return err
				}
			}
			switch name {
			case "supervise":
				return portable.Supervise(cmd.Context(), base, path, cmd.ErrOrStderr())
			case "_serve":
				return portable.ServeBackground(cmd.Context(), base, path)
			case "stop":
				stop := func(ctx context.Context) (struct{}, error) { return struct{}{}, portable.Stop(ctx, path) }
				if interactiveOutput(cmd) {
					_, err = runWait(cmd, "Saving and stopping the world", nil, stop)
				} else {
					_, err = stop(cmd.Context())
				}
				if err != nil {
					return err
				}
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Stopped.")
				return err
			case "doctor":
				var checks []portable.Check
				if interactiveOutput(cmd) && !asJSON {
					checks, err = runWait(cmd, "Checking this instance", nil, func(ctx context.Context) ([]portable.Check, error) { return portable.Doctor(ctx, base, path), nil })
					if err != nil {
						return err
					}
				} else {
					checks = portable.Doctor(cmd.Context(), base, path)
				}
				failed := false
				for _, check := range checks {
					if !check.OK {
						failed = true
					}
				}
				if asJSON {
					err = json.NewEncoder(cmd.OutOrStdout()).Encode(checks)
				} else if richOutput(cmd) {
					err = checklist(cmd.OutOrStdout(), "Dark Pawns · diagnostics", checks, cliWidth(cmd))
				} else {
					for _, check := range checks {
						label := "OK"
						if !check.OK {
							label = "CHECK"
						}
						if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s: %s\n", label, check.Name, check.Detail); err != nil {
							break
						}
					}
				}
				if err != nil {
					return err
				}
				if failed {
					return errors.New("doctor found issues; follow the suggested steps above")
				}
				return nil
			}
			var status portable.RuntimeStatus
			switch name {
			case "start":
				if interactiveOutput(cmd) {
					status, err = runWait(cmd, "Starting the world", nil, func(ctx context.Context) (portable.RuntimeStatus, error) { return portable.Start(ctx, base, path) })
				} else {
					status, err = portable.Start(cmd.Context(), base, path)
				}
			case "restart":
				if interactiveOutput(cmd) {
					status, err = runWait(cmd, "Restarting the world", nil, func(ctx context.Context) (portable.RuntimeStatus, error) { return portable.Restart(ctx, base, path) })
				} else {
					status, err = portable.Restart(cmd.Context(), base, path)
				}
			default:
				status, err = portable.Status(cmd.Context(), path)
			}
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\nInstance: %s\n", status.State, status.Instance)
			if err == nil && status.URL != "" {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Web client: %s\n", status.URL)
			}
			return err
		}
		root.AddCommand(cmd)
	}
	// The network apply transaction spawned by the control server: the
	// console saves a configuration draft, and this command restarts the
	// world on it, verifies every requested listener bound, and restores
	// the previous configuration when one did not. It carries the revision
	// the operator reviewed and refuses to run if the configuration changed
	// underneath it.
	var applyRevision int64
	applyConfig := &cobra.Command{
		Use:    "_apply-config",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !cmd.Flags().Changed("revision") {
				return errors.New("--revision is required: the revision of the configuration the operator reviewed")
			}
			path, err := choose(cmd)
			if err != nil {
				return err
			}
			return portable.ApplyNetworkConfig(cmd.Context(), path, applyRevision)
		},
	}
	applyConfig.Flags().Int64Var(&applyRevision, "revision", 0, "the configuration revision the operator reviewed")
	root.AddCommand(applyConfig)

	service := &cobra.Command{Use: "service", Short: "Configure automatic startup for your current user"}
	for _, name := range []string{"install", "remove"} {
		cmd := &cobra.Command{Use: name, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := choose(cmd)
			if err != nil {
				return err
			}
			if name == "install" {
				err = portable.InstallService(cmd.Context(), path)
			} else {
				err = portable.RemoveService(cmd.Context(), path)
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Automatic startup %sed.\n", map[string]string{"install": "install", "remove": "remov"}[name])
			return err
		}}
		service.AddCommand(cmd)
	}
	root.AddCommand(service)
}
