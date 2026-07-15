package oraclediff

import (
	"fmt"
	"strings"
)

type ReportMeta struct {
	Scenario   string
	OracleAddr string
	GoAddr     string
	Seed       string
}

// BlockDiff is the normalized comparison for one probe command.
type BlockDiff struct {
	Command string
	Oracle  string // normalized oracle block
	Go      string // normalized go block
	Diff    string // unified diff; empty if blocks match
}

// Report emits a block-aligned differential report.
func Report(meta ReportMeta, diffs []BlockDiff) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Dark Pawns Tier-1 differential report\n")
	fmt.Fprintf(&out, "scenario: %s\n", meta.Scenario)
	fmt.Fprintf(&out, "c-oracle: %s (DP_SEED=%s)\n", meta.OracleAddr, meta.Seed)
	fmt.Fprintf(&out, "go-port: %s (DP_SEED=%s)\n", meta.GoAddr, meta.Seed)

	var anyDiff bool
	for _, d := range diffs {
		if d.Diff != "" {
			anyDiff = true
			break
		}
	}
	if !anyDiff {
		out.WriteString("result: no normalized divergence\n")
		return out.String()
	}

	out.WriteString("result: normalized divergence detected\n\n")
	for _, d := range diffs {
		if d.Diff == "" {
			fmt.Fprintf(&out, "--- [%s] c-oracle\n+++ [%s] go-port\n(no normalized divergence)\n\n", d.Command, d.Command)
			continue
		}
		// Prefix the diff header with the command label.
		lines := strings.Split(d.Diff, "\n")
		for i, line := range lines {
			if i == 0 && strings.HasPrefix(line, "--- ") {
				fmt.Fprintf(&out, "--- [%s] %s\n", d.Command, strings.TrimPrefix(line, "--- "))
				continue
			}
			if i == 1 && strings.HasPrefix(line, "+++ ") {
				fmt.Fprintf(&out, "+++ [%s] %s\n", d.Command, strings.TrimPrefix(line, "+++ "))
				continue
			}
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	return out.String()
}
