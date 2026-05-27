package cli

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newInitCmd(flags *rootFlags) *cobra.Command {
	var (
		newChar  bool
		password string
		logLevel string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Start the dp-goat daemon and optionally create a character",
		Long: `Start the dp-goat-body daemon for a character. The daemon maintains
a persistent WebSocket connection to the MUD server and exposes a
Unix socket for CLI commands.

Use --new to create a new character through the creation wizard.`,
		Example: `  dp-goat init --name Machine
  dp-goat init --name Machine --new
  dp-goat init --name Machine --new --password mysecret`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			sockPath := filepath.Join(home, ".dp-goat", "sock", flags.playerName+".sock")

			// Check if daemon is already running
			if _, err := net.Dial("unix", sockPath); err == nil {
				fmt.Printf("Daemon already running for %s (socket: %s)\n", flags.playerName, sockPath)
				return nil
			}

			// Find the daemon binary
			daemonBin, err := findDaemonBinary()
			if err != nil {
				return fmt.Errorf("daemon binary not found: %w\nhint: build with: go build ./cmd/dp-goatd/", err)
			}

			// Start daemon as background process
			args_ := []string{"--name", flags.playerName}
			if logLevel != "" {
				args_ = append(args_, "--log-level", logLevel)
			}

			// #nosec G204 -- daemon binary is trusted
			execCmd := exec.Command(daemonBin, args_...)
			execCmd.Stdout = nil // daemon logs to stderr
			execCmd.Stderr = nil
			execCmd.SysProcAttr = detachedProcAttr()

			if err := execCmd.Start(); err != nil {
				return fmt.Errorf("start daemon: %w", err)
			}

			fmt.Printf("Starting daemon for %s (pid %d)...\n", flags.playerName, execCmd.Process.Pid)

			// Wait for socket to appear
			deadline := time.After(10 * time.Second)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-deadline:
					return fmt.Errorf("timeout waiting for daemon socket: %s", sockPath)
				case <-ticker.C:
					if _, err := net.Dial("unix", sockPath); err == nil {
						fmt.Printf("Daemon connected (socket: %s)\n", sockPath)
						return nil
					}
				}
			}
		},
	}

	cmd.Flags().BoolVar(&newChar, "new", false, "create a new character")
	cmd.Flags().StringVar(&password, "password", "", "character password")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "daemon log level")

	return cmd
}

// findDaemonBinary looks for the dp-goatd binary in standard locations.
func findDaemonBinary() (string, error) {
	// Check same directory as CLI binary
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "dp-goatd")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Check GOPATH/bin
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		candidate := filepath.Join(gopath, "bin", "dp-goatd")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Check ~/go/bin
	home, _ := os.UserHomeDir()
	candidate := filepath.Join(home, "go", "bin", "dp-goatd")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}

	return "", fmt.Errorf("dp-goatd not found in PATH, GOPATH/bin, or ~/go/bin")
}
