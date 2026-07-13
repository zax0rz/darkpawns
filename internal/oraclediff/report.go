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

func Report(meta ReportMeta, oracleTranscript, goTranscript string) string {
	diff := UnifiedDiff("c-oracle", "go-port", oracleTranscript, goTranscript)
	var out strings.Builder
	fmt.Fprintf(&out, "Dark Pawns Tier-1 differential report\n")
	fmt.Fprintf(&out, "scenario: %s\n", meta.Scenario)
	fmt.Fprintf(&out, "c-oracle: %s (DP_SEED=%s)\n", meta.OracleAddr, meta.Seed)
	fmt.Fprintf(&out, "go-port: %s (seed unmatched; Tier 1 masks RNG values)\n", meta.GoAddr)
	if diff == "" {
		out.WriteString("result: no normalized divergence\n")
		return out.String()
	}
	out.WriteString("result: normalized divergence detected\n\n")
	out.WriteString(diff)
	return out.String()
}
