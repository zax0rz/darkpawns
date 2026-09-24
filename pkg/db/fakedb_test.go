package db

// A database/sql driver that accepts every statement, for tests that only need
// a query to run (or, with fail set, to fail) without a real database.

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
)

var (
	fakeDriverMu sync.Mutex
	fakeFail     bool
)

func init() {
	sql.Register("fakedb", fakeDBDriver{})
}

type fakeDBDriver struct{}

func setFakeFail(fail bool) {
	fakeDriverMu.Lock()
	defer fakeDriverMu.Unlock()
	fakeFail = fail
}

func isFakeFail() bool {
	fakeDriverMu.Lock()
	defer fakeDriverMu.Unlock()
	return fakeFail
}

func (fakeDBDriver) Open(name string) (driver.Conn, error) {
	return fakeDBConn{}, nil
}

type fakeDBConn struct{}

func (fakeDBConn) Prepare(query string) (driver.Stmt, error) { return nil, nil }
func (fakeDBConn) Close() error                              { return nil }
func (fakeDBConn) Begin() (driver.Tx, error)                 { return nil, nil }

func (fakeDBConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if isFakeFail() {
		return nil, errors.New("simulated DB error")
	}
	return driver.ResultNoRows, nil
}

// CheckNamedValue accepts any Go value so the fake driver does not reject
// PostgreSQL arrays or other argument types during tests.
func (fakeDBConn) CheckNamedValue(nv *driver.NamedValue) error {
	return nil
}

func newFakeDB(t *testing.T, fail bool) *DB {
	t.Helper()
	setFakeFail(fail)
	conn, err := sql.Open("fakedb", "")
	if err != nil {
		t.Fatalf("open fake db: %v", err)
	}
	return &DB{conn: conn}
}
