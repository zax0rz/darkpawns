// Command dp-db-migrate converts one PostgreSQL Dark Pawns database into one
// SQLite database, verifies it, and installs it atomically.
//
// It is an operator tool, not part of the server: nothing in the running game
// imports it, and it never connects anywhere the operator did not name.
//
// A DSN passed in argv is visible to any local process that can read the process
// list. Prefer the environment (DP_MIGRATE_FROM / DP_MIGRATE_TO, or
// DATABASE_URL for the source), a systemd EnvironmentFile, or a shell that does
// not record the command line.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/zax0rz/darkpawns/internal/dbmigrate"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("dp-db-migrate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var (
		from              = flags.String("from", "", "PostgreSQL source DSN (env DP_MIGRATE_FROM, then DATABASE_URL)")
		to                = flags.String("to", "", "SQLite destination path or sqlite:// DSN (env DP_MIGRATE_TO)")
		verify            = flags.Bool("verify", true, "run the independent comparison after copying (use --verify=false to skip; not recommended)")
		verifyOnly        = flags.Bool("verify-only", false, "compare an existing pair and write nothing")
		replace           = flags.Bool("replace", false, "replace a non-empty destination through a temporary file and an atomic rename")
		dropExtraTables   = flags.Bool("drop-extra-tables", false, "leave source tables outside the migrated five behind instead of refusing")
		dropExtraColumns  = flags.Bool("drop-extra-columns", false, "drop source columns with no destination column instead of refusing")
		conversionReceipt = flags.String("conversion-receipt", "", "verify-only: the JSON receipt of the conversion that produced this destination; the only proof verify-only accepts that source tables or columns were deliberately left behind")
		batch             = flags.Int("batch", dbmigrate.DefaultBatchSize, "rows per INSERT statement")
		reportPath        = flags.String("report", "", "write the JSON receipt to this path")
		asJSON            = flags.Bool("json", false, "print the JSON receipt to stdout")
		quiet             = flags.Bool("quiet", false, "suppress progress lines")
		timeout           = flags.Duration("timeout", 30*time.Minute, "overall deadline")
	)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s --from <postgres-dsn> --to <sqlite-path> [--verify]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  convert:     %s --from postgres://... --to /path/darkpawns.db --verify\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  verify only: %s --from postgres://... --to /path/darkpawns.db --verify-only\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "               %s --from postgres://... --to /path/darkpawns.db --verify-only \\\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "                 --conversion-receipt /path/migration-receipt.json   # only if that conversion was told to leave a table or column behind\n\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}

	options := dbmigrate.Options{
		Source:            firstNonEmpty(*from, os.Getenv("DP_MIGRATE_FROM"), os.Getenv("DATABASE_URL")),
		Destination:       firstNonEmpty(*to, os.Getenv("DP_MIGRATE_TO")),
		Verify:            *verify,
		VerifyOnly:        *verifyOnly,
		Replace:           *replace,
		DropExtraTables:   *dropExtraTables,
		DropExtraColumns:  *dropExtraColumns,
		ConversionReceipt: *conversionReceipt,
		BatchSize:         *batch,
	}
	if !*quiet {
		options.Logf = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "dp-db-migrate: "+format+"\n", args...)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	// A migration killed half way must still leave the destination directory in a
	// state an operator can reason about; the engine removes its temporary file on
	// any error, so a signal is turned into an ordinary error.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	receipt, err := dbmigrate.Run(ctx, options)
	if receipt != nil {
		if *reportPath != "" {
			if writeErr := writeReport(*reportPath, receipt); writeErr != nil {
				fmt.Fprintf(os.Stderr, "dp-db-migrate: %v\n", writeErr)
				if err == nil {
					err = writeErr
				}
			}
		}
		if *asJSON {
			payload, jsonErr := receipt.JSON()
			if jsonErr != nil {
				fmt.Fprintf(os.Stderr, "dp-db-migrate: %v\n", jsonErr)
			} else {
				fmt.Println(string(payload))
			}
		} else if !*quiet {
			fmt.Print(receipt.Render())
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "dp-db-migrate: %v\n", err)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return exitFailure
		}
		if receipt != nil && receipt.Failure != nil && receipt.Failure.Phase == dbmigrate.PhaseValidate {
			return exitUsage
		}
		return exitFailure
	}
	return exitOK
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// writeReport writes the JSON receipt with owner-only permissions: it names
// tables, columns and row counts, which is operator information rather than
// public information.
func writeReport(path string, receipt *dbmigrate.Receipt) error {
	payload, err := receipt.JSON()
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(filepath.Clean(path), payload, 0o600); err != nil {
		return fmt.Errorf("write report %s: %w", path, err)
	}
	return nil
}
