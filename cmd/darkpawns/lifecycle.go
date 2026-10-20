package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

func addLifecycle(root *cobra.Command, instance *string, bundle func() (string, error)) {
	choose := func() (string, error) {
		if *instance != "" {
			return *instance, nil
		}
		return defaultInstance()
	}
	for _, name := range []string{"start", "stop", "restart", "status", "doctor", "_serve", "supervise"} {
		var asJSON bool
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
		cmd.RunE = func(cmd *cobra.Command, _ []string) error {
			path, err := choose()
			if err != nil {
				return err
			}
			base, err := bundle()
			if err != nil {
				return err
			}
			switch name {
			case "supervise":
				return portable.Supervise(cmd.Context(), base, path, cmd.ErrOrStderr())
			case "_serve":
				return portable.ServeBackground(cmd.Context(), base, path)
			case "stop":
				if err := portable.Stop(cmd.Context(), path); err != nil {
					return err
				}
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Stopped.")
				return err
			case "doctor":
				checks := portable.Doctor(cmd.Context(), base, path)
				failed := false
				for _, check := range checks {
					if !check.OK {
						failed = true
					}
				}
				if asJSON {
					err = json.NewEncoder(cmd.OutOrStdout()).Encode(checks)
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
				status, err = portable.Start(cmd.Context(), base, path)
			case "restart":
				status, err = portable.Restart(cmd.Context(), base, path)
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
	service := &cobra.Command{Use: "service", Short: "Configure automatic startup for your current user"}
	for _, name := range []string{"install", "remove"} {
		cmd := &cobra.Command{Use: name, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := choose()
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
