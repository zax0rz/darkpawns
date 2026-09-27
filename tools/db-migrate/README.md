# tools/db-migrate — the PostgreSQL → SQLite conversion bridge

An **operator-only** Go module that copies one PostgreSQL Dark Pawns database
into one SQLite database, verifies the copy independently, and installs it
atomically. It is the tool that performed the production conversion on
2026-09-26.

It is a separate module (`tools/db-migrate/go.mod`) for one reason: it needs a
PostgreSQL driver (`lib/pq`), and the shipped game runtime must not. Production
PostgreSQL is still the rollback authority, so the bridge has to keep working;
the runtime has to stop speaking PostgreSQL entirely. A nested module keeps both
true at once:

- `go build ./...` and `go test ./...` **in the repository root never see this
  module**, so the runtime and its releases carry no PostgreSQL driver;
- the runtime cannot import it, because a nested module is not on the parent
  module's import path (and `internal/` inside it narrows that further);
- this module imports the runtime (for the production schema initialisation and
  the character codec), so the two cannot drift.

## Build and run

```bash
cd tools/db-migrate
go build -o dp-db-migrate ./cmd/dp-db-migrate
./dp-db-migrate --help
```

Or from the repository root: `make db-migrate-build`.

The invocation moved when this module moved: it used to be
`go build -o dp-db-migrate ./cmd/dp-db-migrate` from the repository root. The
CLI itself is unchanged. The operator procedure — preflight, conversion, service
configuration, post-cutover checks, rollback — is
[`docs/operational/SQLITE-CUTOVER.md`](../../docs/operational/SQLITE-CUTOVER.md).

## Tests

The suite needs a **disposable PostgreSQL database of its own**, not the shared
`DATABASE_URL`:

```bash
createdb darkpawns_migrate_test
cd tools/db-migrate
DP_MIGRATE_TEST_DB_URL="postgres:///darkpawns_migrate_test?host=/var/run/postgresql" \
  go test ./... -count=1
```

Or from the repository root: `make db-migrate-test`.

Why its own database: the tests create the five production table names in
private schemas, and `information_schema` is per database, not per schema, so a
shared database would make those tables visible to other packages' tests.
Without `DP_MIGRATE_TEST_DB_URL` the PostgreSQL tests skip — a skip is not proof,
which is why this module has its own GitHub Actions workflow
(`.github/workflows/db-migrate.yml`) that creates an isolated database, proves
the whole suite, and runs a conversion through the built binary.

## The frozen source schema

`internal/dbmigrate/testdata/pgschema_*.sql` is the **PostgreSQL schema as it
stood when production was converted**, not what the current code would create.
That is deliberate: the runtime is SQLite-only now, the migration being verified
is a historical one, and a fixture that tracked live code could not prove
anything about it. If the fixture is ever updated, it must be updated to a
schema that production actually had.
