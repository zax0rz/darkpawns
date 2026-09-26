package dbmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/moderation"
)

// Run performs one migration request. It returns a receipt describing what
// happened even when it fails, so a failed run is diagnosable; the error is what
// the caller turns into a nonzero exit.
//
// Failure contract: on any error the destination is untouched. The database is
// built at a temporary sibling path, so a run that dies during the copy, the
// integrity check or the verification leaves no file where a successful result
// would be.
func Run(ctx context.Context, options Options) (*Receipt, error) {
	started := time.Now()
	mode := ModeConvert
	if options.VerifyOnly {
		mode = ModeVerifyOnly
	}
	receipt := &Receipt{
		Tool:        ToolName,
		GeneratedAt: started.UTC(),
		Mode:        mode,
		Options: OptionsSummary{
			Verify:           options.Verify,
			VerifyOnly:       options.VerifyOnly,
			Replace:          options.Replace,
			DropExtraTables:  options.DropExtraTables,
			DropExtraColumns: options.DropExtraColumns,
			BatchSize:        options.batchSize(),
		},
	}
	finish := func(err error) (*Receipt, error) {
		receipt.DurationMS = time.Since(started).Milliseconds()
		if err != nil {
			receipt.OK = false
			return receipt, err
		}
		receipt.OK = true
		return receipt, nil
	}
	fail := func(phase string, err error) (*Receipt, error) {
		receipt.Failure = &Failure{Phase: phase, Message: err.Error()}
		return finish(err)
	}

	if err := ValidateOptions(options); err != nil {
		return fail(PhaseValidate, err)
	}
	receipt.Source.DSN = RedactDSN(options.Source)
	destinationPath, err := DestinationPath(options.Destination)
	if err != nil {
		return fail(PhaseValidate, err)
	}
	receipt.Destination.Path = destinationPath

	sourceConn, err := openSource(options.Source)
	if err != nil {
		return fail(PhasePreflight, err)
	}
	defer func() { _ = sourceConn.Close() }()

	// One read-only repeatable-read transaction is the source for everything:
	// the catalog, the copy and the verification all see the same snapshot, and
	// the read-only flag is what makes "never mutate the source" a property of
	// the connection rather than of this code's self-discipline.
	sourceTx, err := sourceConn.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return fail(PhasePreflight, fmt.Errorf("open read-only source snapshot: %w", err))
	}
	defer func() { _ = sourceTx.Rollback() }()

	if options.VerifyOnly {
		return runVerifyOnly(ctx, options, receipt, sourceTx, destinationPath, finish, fail)
	}
	return runConvert(ctx, options, receipt, sourceTx, destinationPath, finish, fail)
}

func openSource(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	// Deliberately not pkg/db.New: that runs schema initialisation, and a
	// migration must not run DDL against the database it is reading. The source
	// keeps its pool defaults from database/sql.
	conn.SetMaxOpenConns(4)
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connect to source: %w", err)
	}
	return conn, nil
}

// openDestinationReadOnly opens an existing SQLite file without creating it and
// without touching its journal mode, for --verify-only.
func openDestinationReadOnly(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("destination: %w", err)
	}
	conn, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open destination: %w", err)
	}
	conn.SetMaxOpenConns(2)
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connect to destination: %w", err)
	}
	return conn, nil
}

func runVerifyOnly(
	ctx context.Context, options Options, receipt *Receipt, sourceTx *sql.Tx, destinationPath string,
	finish func(error) (*Receipt, error), fail func(string, error) (*Receipt, error),
) (*Receipt, error) {
	destinationConn, err := openDestinationReadOnly(destinationPath)
	if err != nil {
		return fail(PhasePreflight, err)
	}
	defer func() { _ = destinationConn.Close() }()

	plans, err := reconcileAll(ctx, sourceTx, destinationConn)
	if err != nil {
		return fail(PhaseSchema, err)
	}
	receipt.Schema = describeSchemas(plans)

	verification, err := Verify(ctx, sourceTx, destinationConn, plans)
	receipt.Verification = verification
	if err != nil {
		return fail(PhaseVerify, err)
	}
	if !verification.OK {
		return fail(PhaseVerify, fmt.Errorf("verification failed: %s", FailureSummary(verification)))
	}
	return finish(nil)
}

func runConvert(
	ctx context.Context, options Options, receipt *Receipt, sourceTx *sql.Tx, destinationPath string,
	finish func(error) (*Receipt, error), fail func(string, error) (*Receipt, error),
) (result *Receipt, returnErr error) {
	if info, err := os.Stat(destinationPath); err == nil {
		receipt.Destination.Existed = true
		if info.Size() > 0 && !options.Replace {
			return fail(PhasePreflight, fmt.Errorf(
				"destination %s exists and is not empty (%d bytes); pass --replace to build a verified replacement",
				destinationPath, info.Size()))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fail(PhasePreflight, fmt.Errorf("inspect destination: %w", err))
	}

	tables, err := SourceTables(ctx, sourceTx)
	if err != nil {
		return fail(PhasePreflight, err)
	}
	receipt.Source.Tables = tables
	unknown := UnknownTables(tables)
	receipt.Source.UnknownTables = unknown
	if len(unknown) > 0 && !options.DropExtraTables {
		return fail(PhasePreflight, fmt.Errorf(
			"source has %d table(s) outside the migrated set: %v; they have no SQLite destination, so this run stops rather than leaving them behind (pass --drop-extra-tables only if you have decided they are obsolete)",
			len(unknown), unknown))
	}

	temporary := temporaryPath(destinationPath)
	removeDatabase(temporary)
	defer func() {
		if returnErr != nil {
			removeDatabase(temporary)
		}
	}()

	options.logf("building schema at %s", temporary)
	destination, err := initDestinationSchema(ctx, "sqlite://"+temporary)
	if err != nil {
		return fail(PhaseSchema, err)
	}
	certifyClose := func(err error) error {
		if closeErr := destination.Close(); closeErr != nil && err == nil {
			return fmt.Errorf("close destination: %w", closeErr)
		}
		return err
	}
	defer func() { returnErr = certifyClose(returnErr) }()
	if err := os.Chmod(temporary, 0o600); err != nil {
		return fail(PhaseSchema, fmt.Errorf("set destination permissions: %w", err))
	}
	destinationConn := destination.conn()

	plans, err := reconcileAll(ctx, sourceTx, destinationConn)
	if err != nil {
		return fail(PhaseSchema, err)
	}
	receipt.Schema = describeSchemas(plans)
	for _, plan := range plans {
		if len(plan.ExtraInSource) > 0 && !options.DropExtraColumns {
			return fail(PhaseSchema, fmt.Errorf(
				"source table %s has column(s) with no destination column: %v (pass --drop-extra-columns to drop them, or extend the runtime schema)",
				plan.Name, plan.ExtraInSource))
		}
		if len(plan.MissingInSource) > 0 {
			options.logf("note: %s is missing %d source column(s); the schema default applies: %v",
				plan.Name, len(plan.MissingInSource), plan.MissingInSource)
		}
	}

	tx, err := destinationConn.BeginTx(ctx, nil)
	if err != nil {
		return fail(PhaseCopy, fmt.Errorf("begin destination transaction: %w", err))
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, plan := range plans {
		started := time.Now()
		options.logf("copying %s", plan.Name)
		rows, err := copyTable(ctx, sourceTx, tx, plan, options.batchSize())
		if err != nil {
			return fail(PhaseCopy, err)
		}
		receipt.Copied = append(receipt.Copied, TableCopy{
			Table:        plan.Name,
			Rows:         rows,
			Milliseconds: time.Since(started).Milliseconds(),
		})
	}

	maxIDs, err := syncSequences(ctx, tx, plans)
	if err != nil {
		return fail(PhaseCopy, err)
	}
	if err := tx.Commit(); err != nil {
		return fail(PhaseCopy, fmt.Errorf("commit destination: %w", err))
	}
	committed = true

	integrity, err := integrityCheck(ctx, destinationConn)
	receipt.Destination.Integrity = integrity
	if err != nil {
		return fail(PhaseIntegrity, err)
	}

	if options.Verify {
		verification, err := Verify(ctx, sourceTx, destinationConn, plans)
		receipt.Verification = verification
		if err != nil {
			return fail(PhaseVerify, err)
		}
		if !verification.OK {
			return fail(PhaseVerify, fmt.Errorf(
				"verification failed (%s); the destination was not installed and no partial file was left behind",
				FailureSummary(verification)))
		}
	}

	sequences, err := readIDSequences(ctx, destinationConn, plans, maxIDs)
	if err != nil {
		return fail(PhaseFinalize, err)
	}
	receipt.IDSequences = sequences
	for _, sequence := range sequences {
		if !sequence.SequenceAboveMax {
			return fail(PhaseFinalize, fmt.Errorf(
				"%s.%s sequence %d is not above the highest migrated id %d: new rows could reuse an id",
				sequence.Table, sequence.Column, sequence.Sequence, sequence.MaxID))
		}
	}

	if err := finalizeDatabase(ctx, destinationConn); err != nil {
		return fail(PhaseFinalize, err)
	}
	indexes, err := DestinationIndexes(ctx, destinationConn)
	if err != nil {
		return fail(PhaseFinalize, err)
	}
	receipt.Destination.Indexes = indexes

	returnErr = certifyClose(nil)
	if returnErr != nil {
		return fail(PhaseFinalize, returnErr)
	}

	if err := installFile(temporary, destinationPath); err != nil {
		return fail(PhaseInstall, err)
	}
	if info, err := os.Stat(destinationPath); err == nil {
		receipt.Destination.Bytes = info.Size()
	}
	return finish(nil)
}

// destinationSchema owns the two handles that together are the production
// schema: the game store and the moderation store. One Close covers both, so a
// failed run cannot leave the moderation cleanup goroutine behind.
type destinationSchema struct {
	database   *db.DB
	moderation *moderation.Manager
}

func (d *destinationSchema) conn() *sql.DB { return d.database.SQLDB() }

func (d *destinationSchema) Close() error {
	if d.moderation != nil {
		d.moderation.Close()
	}
	return d.database.Close()
}

// initDestinationSchema creates the destination through the same constructors the
// server runs at boot: pkg/db.New creates the game-store tables and their
// migrations, and moderation.NewManager creates the four moderation tables. A
// hand-written schema here would drift the first time either package changed its
// DDL, which is the whole reason the tool does not carry one.
func initDestinationSchema(ctx context.Context, dsn string) (*destinationSchema, error) {
	database, err := db.New(dsn)
	if err != nil {
		return nil, fmt.Errorf("create destination game-store schema: %w", err)
	}
	manager := moderation.NewManager(database.SQLDB(), database.Dialect())
	schema := &destinationSchema{database: database, moderation: manager}
	if err := schema.confirmTables(ctx); err != nil {
		_ = schema.Close()
		return nil, err
	}
	return schema, nil
}

// confirmTables fails loudly when a table the runtime expects is missing.
// moderation.NewManager only logs its schema failure, and a destination missing
// a moderation table would otherwise be discovered as a confusing column error
// much later in the run.
func (d *destinationSchema) confirmTables(ctx context.Context) error {
	for _, table := range Tables {
		var name string
		err := d.conn().QueryRowContext(ctx,
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(
				"destination schema is missing table %q after production schema initialisation (pkg/db.New + moderation.NewManager)",
				table)
		}
		if err != nil {
			return fmt.Errorf("confirm destination table %s: %w", table, err)
		}
	}
	return nil
}

func reconcileAll(ctx context.Context, source, destination queryer) ([]*Table, error) {
	plans := make([]*Table, 0, len(Tables))
	for _, table := range Tables {
		plan, err := Reconcile(ctx, source, destination, table)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

func describeSchemas(plans []*Table) []TableSchemaInfo {
	infos := make([]TableSchemaInfo, 0, len(plans))
	for _, plan := range plans {
		info := TableSchemaInfo{
			Table:           plan.Name,
			GeneratedID:     plan.GeneratedID,
			PrimaryKey:      plan.PrimaryKey,
			MissingInSource: plan.MissingInSource,
			ExtraInSource:   plan.ExtraInSource,
		}
		for _, column := range plan.Columns {
			info.Columns = append(info.Columns, ColumnInfo{
				Name:            column.Name,
				Kind:            string(column.Kind),
				SourceType:      column.SourceType,
				DestinationType: column.DestinationType,
				NotNull:         column.NotNull,
			})
		}
		infos = append(infos, info)
	}
	return infos
}

// readIDSequences reports each AUTOINCREMENT table's stored sequence beside the
// highest id that was copied. The sequence is what SQLite hands the next insert,
// so this is the id-reuse guarantee in one line.
func readIDSequences(ctx context.Context, conn queryer, plans []*Table, maxIDs map[string]int64) ([]IDSequence, error) {
	var sequences []IDSequence
	for _, plan := range plans {
		if plan.GeneratedID == "" {
			continue
		}
		entry := IDSequence{
			Table:  plan.Name,
			Column: plan.GeneratedID,
			MaxID:  maxIDs[plan.Name],
		}
		var seq sql.NullInt64
		err := conn.QueryRowContext(ctx, `SELECT seq FROM sqlite_sequence WHERE name = ?`, plan.Name).Scan(&seq)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("read sqlite_sequence for %s: %w", plan.Name, err)
		}
		if seq.Valid {
			entry.Sequence = seq.Int64
		}
		// SQLite increments the sequence on the next insert, so a sequence equal
		// to the maximum is already safe: the next id is max+1.
		entry.SequenceAboveMax = entry.Sequence >= entry.MaxID
		sequences = append(sequences, entry)
	}
	return sequences, nil
}

func temporaryPath(destinationPath string) string {
	return filepath.Join(filepath.Dir(destinationPath),
		"."+filepath.Base(destinationPath)+".migrating-"+strconv.Itoa(os.Getpid()))
}
