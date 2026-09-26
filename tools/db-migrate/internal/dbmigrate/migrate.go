package dbmigrate

import (
	"errors"
	"fmt"
	"strings"

	_ "github.com/lib/pq"

	"github.com/zax0rz/darkpawns/pkg/db"
)

// ToolName identifies this tool and its receipts.
const ToolName = "dp-db-migrate"

// DefaultBatchSize is how many rows one INSERT statement carries. Multi-row
// inserts are what keep a migration of a few thousand rows to a few thousand
// statements instead of one per row; the batch stays inside one transaction.
const DefaultBatchSize = 200

// Options is one migration request.
type Options struct {
	// Source is the PostgreSQL DSN to read. It may come from a flag, from
	// DP_MIGRATE_FROM, or from DATABASE_URL; the caller resolves that.
	Source string
	// Destination is the SQLite path or sqlite:// DSN to write.
	Destination string
	// Verify runs the independent comparison after the copy.
	Verify bool
	// VerifyOnly compares two existing databases and writes nothing.
	VerifyOnly bool
	// Replace allows an existing non-empty destination to be replaced, through a
	// temporary file and an atomic rename.
	Replace bool
	// DropExtraTables allows a source table outside the five migrated ones to be
	// left behind.
	DropExtraTables bool
	// DropExtraColumns allows source columns with no destination column to be
	// dropped.
	DropExtraColumns bool
	// ConversionReceipt is the path to a JSON receipt written by an earlier
	// conversion. It is only meaningful with VerifyOnly, where it is the sole
	// proof that the source tables and columns the destination cannot hold were
	// explicitly allowed when the conversion ran: verification accepts an
	// allowance from a conversion receipt and from nowhere else.
	ConversionReceipt string
	// BatchSize overrides DefaultBatchSize.
	BatchSize int
	// Logf receives progress lines. Nil discards them.
	Logf func(format string, args ...any)
}

func (o Options) logf(format string, args ...any) {
	if o.Logf != nil {
		o.Logf(format, args...)
	}
}

func (o Options) batchSize() int {
	if o.BatchSize > 0 {
		return o.BatchSize
	}
	return DefaultBatchSize
}

// DestinationPath resolves a destination string to a filesystem path.
//
// The runtime owns what a SQLite setting looks like (pkg/db.SQLitePath), and it
// refuses a PostgreSQL URL, an empty setting and an in-memory database. The
// refusal is wrapped rather than replaced so the caller sees which side of the
// migration was wrong.
func DestinationPath(destination string) (string, error) {
	path, err := db.SQLitePath(destination)
	if err != nil {
		return "", fmt.Errorf("destination must be a SQLite file: %w", err)
	}
	return path, nil
}

// IsPostgresDSN reports whether a setting names a PostgreSQL server.
//
// The check lives here rather than in the runtime, and deliberately: the runtime
// is SQLite-only and no longer knows what a PostgreSQL URL looks like, while
// this module is the only thing in the repository that speaks PostgreSQL. It is
// also the only thing that would ever need to say yes.
func IsPostgresDSN(setting string) bool {
	lower := strings.ToLower(strings.TrimSpace(setting))
	return strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://")
}

// ValidateOptions refuses ambiguous or reversed configurations before anything
// is opened. A migration that guesses which side is the source is a migration
// that can destroy the wrong database.
func ValidateOptions(options Options) error {
	if strings.TrimSpace(options.Source) == "" {
		return errors.New("no PostgreSQL source configured")
	}
	if strings.TrimSpace(options.Destination) == "" {
		return errors.New("no SQLite destination configured")
	}
	if !IsPostgresDSN(options.Source) {
		return fmt.Errorf("source must be a PostgreSQL DSN: %q is not a postgres:// or postgresql:// URL",
			RedactDSN(options.Source))
	}
	destinationPath, err := DestinationPath(options.Destination)
	if err != nil {
		return err
	}
	// A source that is not a PostgreSQL URL cannot be the destination's path, so
	// the reversed invocation is already refused above. This catch is for the
	// case a caller passes the same string to both flags.
	if strings.TrimSpace(options.Source) == options.Destination {
		return fmt.Errorf("source and destination are the same setting: %s", destinationPath)
	}
	if options.VerifyOnly && options.Replace {
		return errors.New("--verify-only writes nothing, so --replace has no meaning")
	}
	if options.VerifyOnly && (options.DropExtraTables || options.DropExtraColumns) {
		// The refusal is deliberate: verify-only must not be talked out of checking
		// something by an argument that exists to let a conversion leave it behind.
		// The only proof it accepts is the conversion receipt that records the
		// decision that was actually made.
		return errors.New(
			"--drop-extra-tables and --drop-extra-columns are conversion flags, and verify-only takes no allowances from this command line: pass the receipt of the conversion that made the decision instead (--conversion-receipt <path>)")
	}
	if !options.VerifyOnly && options.ConversionReceipt != "" {
		return errors.New(
			"--conversion-receipt is only meaningful with --verify-only: a conversion records the allowances this command line gives it rather than inheriting an earlier run's")
	}
	return nil
}

// RedactDSN removes credentials and query parameters so a DSN can appear in a
// receipt or a log line. PostgreSQL passwords live in the userinfo of the URL and
// in a password= parameter; both are dropped, and only the scheme, host, port and
// database name remain. The runtime has its own redactor for the setting it
// accepts; this module needs one that also understands a PostgreSQL URL, because
// that is the only DSN it is ever handed.
func RedactDSN(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return ""
	}
	if !IsPostgresDSN(trimmed) {
		if _, rest, found := strings.Cut(trimmed, "://"); found {
			return "sqlite://" + rest
		}
		return "sqlite://" + trimmed
	}
	scheme, rest, _ := strings.Cut(trimmed, "://")
	rest, _, _ = strings.Cut(rest, "?")
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		rest = rest[at+1:]
	}
	if rest == "" {
		return scheme + "://"
	}
	return scheme + "://" + rest
}
