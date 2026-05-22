package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newWatchCmd(flags *rootFlags) *cobra.Command {
	var eventTypes string

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Stream real-time events from the daemon",
		Long: `Poll the daemon for new events and print them as they arrive.
Press Ctrl+C to stop.`,
		Example: `  dp-goat watch
  dp-goat watch --events combat,tell
  dp-goat --name Machine watch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Parse event type filter
			var filter map[string]bool
			if eventTypes != "" {
				filter = make(map[string]bool)
				for _, t := range strings.Split(eventTypes, ",") {
					filter[strings.TrimSpace(t)] = true
				}
			}

			// Set up signal handler
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			fmt.Fprintf(os.Stderr, "Watching events (Ctrl+C to stop)...\n")

			var lastSeq uint64
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-sigCh:
					fmt.Fprintf(os.Stderr, "\nStopped watching.\n")
					return nil
				case <-ticker.C:
					// Poll for events since last seen
					path := fmt.Sprintf("/events?since=%d", lastSeq)
					data, err := c.Get(path, nil)
					if err != nil {
						// Daemon might be restarting
						continue
					}

					var events []struct {
						Seq       uint64          `json:"seq"`
						Type      string          `json:"type"`
						Timestamp string          `json:"ts"`
						Data      json.RawMessage `json:"data"`
					}
					if err := json.Unmarshal(data, &events); err != nil {
						continue
					}

					for _, ev := range events {
						// Apply filter
						if filter != nil && !filter[ev.Type] {
							continue
						}

						// Print event
						ts := ev.Timestamp
						if len(ts) > 19 {
							ts = ts[:19]
						}
						fmt.Printf("[%s] #%d %s\n", ts, ev.Seq, ev.Type)

						// Print data summary
						var summary string
						switch ev.Type {
						case "tell":
							var t struct {
								From    string `json:"from"`
								Message string `json:"message"`
							}
							if json.Unmarshal(ev.Data, &t) == nil {
								summary = fmt.Sprintf("  %s: %q", t.From, t.Message)
							}
						case "say":
							var s struct {
								From    string `json:"from"`
								Message string `json:"message"`
							}
							if json.Unmarshal(ev.Data, &s) == nil {
								summary = fmt.Sprintf("  %s: %q", s.From, s.Message)
							}
						case "vars":
							var v struct {
								RoomName string `json:"ROOM_NAME"`
								Health   int    `json:"HEALTH"`
							}
							if json.Unmarshal(ev.Data, &v) == nil {
								if v.RoomName != "" {
									summary = fmt.Sprintf("  room: %s", v.RoomName)
								}
							}
						}

						if summary != "" {
							fmt.Println(summary)
						}

						if ev.Seq > lastSeq {
							lastSeq = ev.Seq
						}
					}
				}
			}
		},
	}

	cmd.Flags().StringVar(&eventTypes, "events", "", "comma-separated event types to watch (e.g. combat,tell,say)")

	return cmd
}
