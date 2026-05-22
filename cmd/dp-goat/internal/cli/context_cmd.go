package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newContextCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Show the full context packet (state + narrative summary + recent events)",
		Long: `Display the full context packet that the LLM mind receives when it
wakes up. Includes current state, a narrative summary of recent events,
and the last N raw events.`,
		Example: `  dp-goat context
  dp-goat context --json
  dp-goat --name Machine context`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			data, err := c.Get("/context", nil)
			if err != nil {
				return fmt.Errorf("get context: %w", err)
			}

			if flags.asJSON {
				fmt.Println(string(data))
				return nil
			}

			return printContext(data)
		},
	}

	return cmd
}

func printContext(data []byte) error {
	var ctx struct {
		State struct {
			Player struct {
				Name      string `json:"name"`
				Health    int    `json:"health"`
				MaxHealth int    `json:"max_health"`
				Mana      int    `json:"mana"`
				Level     int    `json:"level"`
				Exp       int    `json:"exp"`
				Gold      int    `json:"gold"`
			} `json:"player"`
			Room struct {
				Name  string   `json:"name"`
				Vnum  int      `json:"vnum"`
				Exits []string `json:"exits"`
				Mobs  []struct {
					Name     string `json:"name"`
					Fighting bool   `json:"fighting"`
				} `json:"mobs"`
			} `json:"room"`
			Fighting string `json:"fighting,omitempty"`
		} `json:"state"`
		Summary string `json:"summary"`
		Events  []struct {
			Seq  uint64 `json:"seq"`
			Type string `json:"type"`
		} `json:"events"`
	}

	if err := json.Unmarshal(data, &ctx); err != nil {
		return fmt.Errorf("parse context: %w", err)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// State
	fmt.Fprintf(tw, "Room:\t%s (vnum %d)\n", ctx.State.Room.Name, ctx.State.Room.Vnum)
	fmt.Fprintf(tw, "HP:\t%d/%d\n", ctx.State.Player.Health, ctx.State.Player.MaxHealth)
	fmt.Fprintf(tw, "Mana:\t%d\n", ctx.State.Player.Mana)
	fmt.Fprintf(tw, "Level:\t%d  Exp:%d  Gold:%d\n", ctx.State.Player.Level, ctx.State.Player.Exp, ctx.State.Player.Gold)

	if len(ctx.State.Room.Exits) > 0 {
		fmt.Fprintf(tw, "Exits:\t%v\n", ctx.State.Room.Exits)
	}

	if len(ctx.State.Room.Mobs) > 0 {
		for _, m := range ctx.State.Room.Mobs {
			status := ""
			if m.Fighting {
				status = " [fighting]"
			}
			fmt.Fprintf(tw, "  Mob:\t%s%s\n", m.Name, status)
		}
	}

	if ctx.State.Fighting != "" {
		fmt.Fprintf(tw, "Fighting:\t%s\n", ctx.State.Fighting)
	}

	// Narrative
	if ctx.Summary != "" {
		fmt.Fprintf(tw, "\nWhat happened:\n%s\n", ctx.Summary)
	}

	// Events
	if len(ctx.Events) > 0 {
		fmt.Fprintf(tw, "\nRecent events (%d):\n", len(ctx.Events))
		for _, e := range ctx.Events {
			fmt.Fprintf(tw, "  [%d] %s\n", e.Seq, e.Type)
		}
	}

	return tw.Flush()
}
