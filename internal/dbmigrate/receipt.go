package dbmigrate

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Phases name the stage a failure happened in, so a receipt is actionable
// without reading the code.
const (
	PhaseValidate  = "validate"
	PhasePreflight = "preflight"
	PhaseSchema    = "schema"
	PhaseCopy      = "copy"
	PhaseIntegrity = "integrity"
	PhaseVerify    = "verify"
	PhaseFinalize  = "finalize"
	PhaseInstall   = "install"
	ModeConvert    = "convert"
	ModeVerifyOnly = "verify-only"
)

// Failure records what stopped a run.
type Failure struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

// ColumnInfo is one copied column as the receipt reports it.
type ColumnInfo struct {
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	SourceType      string `json:"source_type"`
	DestinationType string `json:"destination_type"`
	NotNull         bool   `json:"not_null"`
}

// TableSchemaInfo is the reconciled plan for one table.
type TableSchemaInfo struct {
	Table           string       `json:"table"`
	Columns         []ColumnInfo `json:"columns"`
	GeneratedID     string       `json:"generated_id,omitempty"`
	PrimaryKey      []string     `json:"primary_key"`
	MissingInSource []string     `json:"missing_in_source,omitempty"`
	ExtraInSource   []string     `json:"extra_in_source,omitempty"`
}

// TableCopy records what was copied for one table.
type TableCopy struct {
	Table        string `json:"table"`
	Rows         int64  `json:"rows"`
	Milliseconds int64  `json:"milliseconds"`
}

// IDSequence reports the post-migration id watermark for one AUTOINCREMENT
// table. SequenceAboveMax is the proof that the next application insert cannot
// reuse a migrated id.
type IDSequence struct {
	Table            string `json:"table"`
	Column           string `json:"column"`
	MaxID            int64  `json:"max_id"`
	Sequence         int64  `json:"sequence"`
	SequenceAboveMax bool   `json:"sequence_above_max"`
}

// SourceInfo describes the PostgreSQL side without credentials.
type SourceInfo struct {
	DSN           string   `json:"dsn"`
	Tables        []string `json:"tables"`
	UnknownTables []string `json:"unknown_tables,omitempty"`
}

// DestinationInfo describes the SQLite side.
type DestinationInfo struct {
	Path      string   `json:"path"`
	Existed   bool     `json:"existed"`
	Bytes     int64    `json:"bytes,omitempty"`
	Integrity string   `json:"integrity,omitempty"`
	Indexes   []string `json:"indexes,omitempty"`
}

// OptionsSummary is the effective configuration, recorded so a receipt explains
// itself.
type OptionsSummary struct {
	Verify           bool `json:"verify"`
	VerifyOnly       bool `json:"verify_only"`
	Replace          bool `json:"replace"`
	DropExtraTables  bool `json:"drop_extra_tables"`
	DropExtraColumns bool `json:"drop_extra_columns"`
	BatchSize        int  `json:"batch_size"`
}

// Receipt is the machine-readable and human-readable result of one run. It
// carries counts, digests, schema facts and timings: never a password hash, a
// DSN credential or a player's data.
type Receipt struct {
	Tool         string            `json:"tool"`
	GeneratedAt  time.Time         `json:"generated_at"`
	Mode         string            `json:"mode"`
	Source       SourceInfo        `json:"source"`
	Destination  DestinationInfo   `json:"destination"`
	Options      OptionsSummary    `json:"options"`
	Schema       []TableSchemaInfo `json:"schema,omitempty"`
	Copied       []TableCopy       `json:"copied,omitempty"`
	IDSequences  []IDSequence      `json:"id_sequences,omitempty"`
	Verification *Verification     `json:"verification,omitempty"`
	DurationMS   int64             `json:"duration_ms"`
	Failure      *Failure          `json:"failure,omitempty"`
	OK           bool              `json:"ok"`
}

// JSON renders the receipt for a machine to consume.
func (r *Receipt) JSON() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("no receipt")
	}
	return json.MarshalIndent(r, "", "  ")
}

// Render writes the concise terminal summary.
func (r *Receipt) Render() string {
	if r == nil {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, "%s %s\n", r.Tool, r.Mode)
	fmt.Fprintf(&out, "  source:      %s\n", r.Source.DSN)
	fmt.Fprintf(&out, "  destination: %s\n", r.Destination.Path)
	if len(r.Source.UnknownTables) > 0 {
		fmt.Fprintf(&out, "  source tables outside the migrated set: %s\n", strings.Join(r.Source.UnknownTables, ", "))
	}
	if len(r.Copied) > 0 {
		fmt.Fprintf(&out, "  copied:\n")
		for _, copy := range r.Copied {
			fmt.Fprintf(&out, "    %-18s %8d row(s)  %dms\n", copy.Table, copy.Rows, copy.Milliseconds)
		}
	}
	if r.Destination.Integrity != "" {
		fmt.Fprintf(&out, "  sqlite integrity_check: %s\n", r.Destination.Integrity)
	}
	if len(r.IDSequences) > 0 {
		fmt.Fprintf(&out, "  id watermarks:\n")
		for _, sequence := range r.IDSequences {
			fmt.Fprintf(&out, "    %-18s max=%d sequence=%d above_max=%t\n",
				sequence.Table, sequence.MaxID, sequence.Sequence, sequence.SequenceAboveMax)
		}
	}
	if r.Verification != nil {
		fmt.Fprintf(&out, "  verification:\n")
		for i := range r.Verification.Tables {
			table := &r.Verification.Tables[i]
			state := "ok"
			if !table.OK {
				state = "FAILED"
			}
			fmt.Fprintf(&out, "    %-18s source=%d destination=%d content=%s %s\n",
				table.Table, table.SourceRows, table.DestinationRows, short(table.SourceContentDigest), state)
			for _, column := range table.MismatchedColumns {
				fmt.Fprintf(&out, "      column %s differs\n", column)
			}
			for _, note := range table.Notes {
				fmt.Fprintf(&out, "      note: %s\n", note)
			}
		}
		fmt.Fprintf(&out, "    unique folded player names hold: %t\n", r.Verification.UniqueFoldedNamesHolds)
		fmt.Fprintf(&out, "    indexes: %s\n", strings.Join(r.Verification.Indexes, ", "))
	}
	if r.Failure != nil {
		fmt.Fprintf(&out, "  failure (%s): %s\n", r.Failure.Phase, r.Failure.Message)
	}
	fmt.Fprintf(&out, "  duration: %dms\n", r.DurationMS)
	if r.OK {
		out.WriteString("  result: OK\n")
	} else {
		out.WriteString("  result: FAILED\n")
	}
	return out.String()
}

func short(digest string) string {
	if len(digest) <= 12 {
		return digest
	}
	return digest[:12]
}
