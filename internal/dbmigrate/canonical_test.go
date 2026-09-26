package dbmigrate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSQLiteKindDoesNotReadIntervalAsInteger(t *testing.T) {
	// "INT" is a prefix of "INTERVAL": matching the shorter one first read
	// admin_log.duration as an integer and made the reconciliation refuse a table
	// that is identical on both sides.
	cases := map[string]ValueKind{
		"INTERVAL":    KindInterval,
		"interval":    KindInterval,
		"INTEGER":     KindInteger,
		"INT":         KindInteger,
		"INT(11)":     KindInteger,
		"BOOLEAN":     KindBoolean,
		"bool":        KindBoolean,
		"TIMESTAMP":   KindTimestamp,
		"DATETIME":    KindTimestamp,
		"DATE":        KindTimestamp,
		"JSON":        KindJSON,
		"VARCHAR(32)": KindText,
		"TEXT":        KindText,
		"":            KindText,
	}
	for declared, want := range cases {
		got, err := sqliteKind(declared)
		if err != nil {
			t.Fatalf("sqliteKind(%q): %v", declared, err)
		}
		if got != want {
			t.Errorf("sqliteKind(%q) = %s, want %s", declared, got, want)
		}
	}
}

func TestPostgresKindMapping(t *testing.T) {
	cases := map[string]ValueKind{
		"boolean":                     KindBoolean,
		"integer":                     KindInteger,
		"bigint":                      KindInteger,
		"smallint":                    KindInteger,
		"timestamp with time zone":    KindTimestamp,
		"timestamp without time zone": KindTimestamp,
		"json":                        KindJSON,
		"jsonb":                       KindJSON,
		"interval":                    KindInterval,
		"character varying":           KindText,
		"text":                        KindText,
	}
	for dataType, want := range cases {
		got, err := postgresKind(dataType)
		if err != nil {
			t.Fatalf("postgresKind(%q): %v", dataType, err)
		}
		if got != want {
			t.Errorf("postgresKind(%q) = %s, want %s", dataType, got, want)
		}
	}
	if _, err := postgresKind("money"); err == nil {
		t.Error("postgresKind should refuse a type it cannot canonicalize")
	}
}

func TestCanonicalizeScalars(t *testing.T) {
	cases := []struct {
		name  string
		kind  ValueKind
		value any
		want  string
	}{
		{"null", KindInteger, nil, nullSentinel},
		{"int64", KindInteger, int64(-1), "-1"},
		{"negative sentinel", KindInteger, int64(-1), "-1"},
		{"int32", KindInteger, int32(7), "7"},
		{"int text", KindInteger, "42", "42"},
		{"bool true", KindBoolean, true, "1"},
		{"bool false", KindBoolean, false, "0"},
		{"sqlite bool as int", KindBoolean, int64(1), "1"},
		{"text", KindText, "Zax", "Zax"},
		{"text from bytes", KindText, []byte("Zax"), "Zax"},
		{"interval", KindInterval, []byte("01:30:00"), "01:30:00"},
		{"empty text is not null", KindText, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Canonicalize(tc.kind, tc.value)
			if err != nil {
				t.Fatalf("Canonicalize: %v", err)
			}
			if got != tc.want {
				t.Errorf("Canonicalize = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCanonicalizeTimestampIsAnInstant(t *testing.T) {
	// The same instant written in three zones canonicalizes identically, and the
	// wall-clock text does not leak into the comparison.
	est := time.Date(2026, 9, 26, 20, 5, 0, 500000000, time.FixedZone("EDT", -4*3600))
	utc := est.UTC()
	cairo := est.In(time.FixedZone("EET", 2*3600))

	fromTime, err := Canonicalize(KindTimestamp, est)
	if err != nil {
		t.Fatal(err)
	}
	fromUTC, err := Canonicalize(KindTimestamp, utc)
	if err != nil {
		t.Fatal(err)
	}
	fromCairo, err := Canonicalize(KindTimestamp, cairo)
	if err != nil {
		t.Fatal(err)
	}
	if fromTime != fromUTC || fromTime != fromCairo {
		t.Fatalf("zone changed the canonical instant: %q / %q / %q", fromTime, fromUTC, fromCairo)
	}
	if !strings.HasSuffix(fromTime, "Z") || !strings.HasPrefix(fromTime, "2026-09-27T00:05:00.5") {
		t.Errorf("canonical timestamp = %q, want the 2026-09-27T00:05:00.5Z instant", fromTime)
	}

	// A zone-less literal (SQLite's CURRENT_TIMESTAMP) is read as UTC, and one
	// with an offset keeps its meaning.
	zoneLess, err := Canonicalize(KindTimestamp, "2026-09-26 20:05:00")
	if err != nil {
		t.Fatal(err)
	}
	if zoneLess != "2026-09-26T20:05:00Z" {
		t.Errorf("zone-less literal = %q, want it read as UTC", zoneLess)
	}
	offset, err := Canonicalize(KindTimestamp, "2026-09-26 20:05:00.5-04:00")
	if err != nil {
		t.Fatal(err)
	}
	if offset != fromTime {
		t.Errorf("offset literal = %q, want %q", offset, fromTime)
	}
}

func TestCanonicalizeJSONIsSemanticWithoutLosingNumbers(t *testing.T) {
	left := `{"b": 2, "a": {"nested": [1, 2, 3]}}`
	right := `{"a":{"nested":[1,2,3]},"b":2}`
	leftCanonical, err := Canonicalize(KindJSON, []byte(left))
	if err != nil {
		t.Fatal(err)
	}
	rightCanonical, err := Canonicalize(KindJSON, right)
	if err != nil {
		t.Fatal(err)
	}
	if leftCanonical != rightCanonical {
		t.Fatalf("whitespace/key order changed the value: %q vs %q", leftCanonical, rightCanonical)
	}

	// A numeric literal is content, not formatting: 1 and 1.0 must not collapse,
	// or a migration could change a stored number and still verify.
	one, _ := Canonicalize(KindJSON, `{"n":1}`)
	onePoint, _ := Canonicalize(KindJSON, `{"n":1.0}`)
	if one == onePoint {
		t.Error("1 and 1.0 canonicalized equally; a changed number would pass verification")
	}

	// Unicode and escaped quotes survive: no HTML escaping, no mangling. The
	// document is built by the encoder rather than hand-escaped in the test, so
	// the fixture cannot be wrong about JSON's own escaping rules.
	original := map[string]any{"t": `a "quoted" 日本語 title`}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	unicode, err := Canonicalize(KindJSON, encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unicode, "日本語") {
		t.Errorf("unicode was escaped or lost: %q", unicode)
	}
	if strings.Contains(unicode, `\u65e5`) {
		t.Errorf("unicode arrived as an escape sequence: %q", unicode)
	}
	if !strings.Contains(unicode, `\"quoted\"`) {
		t.Errorf("escaped quotes were lost: %q", unicode)
	}
	if unicode != string(encoded) {
		t.Errorf("canonical form of encoder output changed it: %q vs %q", unicode, encoded)
	}

	if _, err := Canonicalize(KindJSON, "not json"); err == nil {
		t.Error("a JSON column holding non-JSON text should be an error, not a silent pass")
	}
	if !JSONReformatOnly(left, right) {
		t.Error("JSONReformatOnly should treat re-spacing as reformatting")
	}
	if JSONReformatOnly(`{"a":1}`, `{"a":2}`) {
		t.Error("JSONReformatOnly should not call a value change reformatting")
	}
}

func TestDigestsAreOrderIndependentAndSensitive(t *testing.T) {
	columns := []string{"id", "name"}
	rowA := []string{"1", "Zax"}
	rowB := []string{"2", "roamer"}
	if RowDigest(columns, rowA) == RowDigest(columns, rowB) {
		t.Error("different rows have the same digest")
	}
	if RowDigest(columns, rowA) != RowDigest(columns, []string{"1", "Zax"}) {
		t.Error("the same row digest is not stable")
	}
	// A column/value boundary must be unambiguous: moving a character across the
	// separator must change the digest.
	if RowDigest(columns, []string{"1Zax", ""}) == RowDigest(columns, []string{"1", "Zax"}) {
		t.Error("digest ignored the column boundary")
	}

	first := TableDigest([]string{"a", "b", "c"})
	shuffled := TableDigest([]string{"c", "a", "b"})
	if first != shuffled {
		t.Error("table digest depends on scan order")
	}
	if first == TableDigest([]string{"a", "b"}) {
		t.Error("table digest ignored a missing row")
	}
}

func TestRedactDSN(t *testing.T) {
	cases := map[string]string{
		"postgres://user:secret@db.example:5432/darkpawns?sslmode=require": "postgres://db.example:5432/darkpawns",
		"postgres://user@localhost/darkpawns":                              "postgres://localhost/darkpawns",
		"postgres:///darkpawns?host=/var/run/postgresql":                   "postgres:///darkpawns",
		"sqlite:///srv/data/darkpawns.db":                                  "sqlite:///srv/data/darkpawns.db",
	}
	for dsn, want := range cases {
		got := RedactDSN(dsn)
		if got != want {
			t.Errorf("RedactDSN(%q) = %q, want %q", dsn, got, want)
		}
		if strings.Contains(got, "secret") || strings.Contains(got, "user:") {
			t.Errorf("RedactDSN(%q) leaked credentials: %q", dsn, got)
		}
	}
}

func TestValidateOptionsRefusesDangerousShapes(t *testing.T) {
	pg := "postgres://user:pw@localhost:5432/darkpawns"
	sqlite := "sqlite:///tmp/darkpawns.db"
	valid := Options{Source: pg, Destination: sqlite, Verify: true}
	if err := ValidateOptions(valid); err != nil {
		t.Fatalf("a valid request was refused: %v", err)
	}

	cases := map[string]Options{
		"no source":                     {Destination: sqlite},
		"no destination":                {Source: pg},
		"reversed":                      {Source: sqlite, Destination: pg},
		"sqlite source":                 {Source: "sqlite:///tmp/a.db", Destination: sqlite},
		"postgres destination":          {Source: pg, Destination: pg},
		"in-memory destination":         {Source: pg, Destination: ":memory:"},
		"same file":                     {Source: sqlite, Destination: sqlite},
		"verify-only with replace":      {Source: pg, Destination: sqlite, VerifyOnly: true, Replace: true},
		"verify-only with drop tables":  {Source: pg, Destination: sqlite, VerifyOnly: true, DropExtraTables: true},
		"verify-only with drop columns": {Source: pg, Destination: sqlite, VerifyOnly: true, DropExtraColumns: true},
		"conversion with a receipt":     {Source: pg, Destination: sqlite, ConversionReceipt: "/tmp/receipt.json"},
	}
	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateOptions(options); err == nil {
				t.Fatalf("%s was accepted: %+v", name, options)
			}
		})
	}
}

func TestDestinationPathShapes(t *testing.T) {
	cases := map[string]string{
		"/tmp/darkpawns.db":      "/tmp/darkpawns.db",
		"sqlite:///tmp/dark.db":  "/tmp/dark.db",
		"sqlite://./relative.db": "relative.db",
	}
	for input, want := range cases {
		got, err := DestinationPath(input)
		if err != nil {
			t.Fatalf("DestinationPath(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("DestinationPath(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := DestinationPath("postgres://localhost/darkpawns"); err == nil {
		t.Error("a PostgreSQL destination should be refused")
	}
	if _, err := DestinationPath("file::memory:"); err == nil {
		t.Error("an in-memory destination should be refused")
	}
}

func TestUnknownTables(t *testing.T) {
	got := UnknownTables([]string{"admin_log", "players", "chat_logs", "word_filters", "admin_users"})
	want := []string{"chat_logs", "admin_users"}
	if len(got) != len(want) {
		t.Fatalf("UnknownTables = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("UnknownTables[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestVerifyOnlyNeverInheritsTheConversionFlags pins the contract the follow-up
// asks for: the escape hatches that let a conversion leave data behind are not
// accepted in verify-only at all, so a verification cannot be talked out of
// checking something by an argument on its own command line. Its only proof is
// the receipt of the conversion that made the decision.
func TestVerifyOnlyNeverInheritsTheConversionFlags(t *testing.T) {
	pg := "postgres://user:pw@localhost:5432/darkpawns"
	sqlite := "sqlite:///tmp/darkpawns.db"

	for _, options := range []Options{
		{Source: pg, Destination: sqlite, VerifyOnly: true, DropExtraTables: true},
		{Source: pg, Destination: sqlite, VerifyOnly: true, DropExtraColumns: true},
		{Source: pg, Destination: sqlite, VerifyOnly: true, DropExtraTables: true, DropExtraColumns: true},
	} {
		err := ValidateOptions(options)
		if err == nil {
			t.Fatalf("verify-only accepted a conversion allowance flag: %+v", options)
		}
		if !strings.Contains(err.Error(), "--conversion-receipt") {
			t.Errorf("refusal does not point at the only proof verify-only accepts: %v", err)
		}
	}

	// The same flags on a conversion are exactly what they look like, and the
	// receipt of that conversion is accepted by a later verification.
	conversion := Options{Source: pg, Destination: sqlite, DropExtraTables: true, DropExtraColumns: true}
	if err := ValidateOptions(conversion); err != nil {
		t.Fatalf("a conversion was refused its own flags: %v", err)
	}
	verification := Options{Source: pg, Destination: sqlite, VerifyOnly: true, ConversionReceipt: "/tmp/receipt.json"}
	if err := ValidateOptions(verification); err != nil {
		t.Fatalf("a verification was refused a receipt: %v", err)
	}

	// A conversion has nothing to inherit, so a receipt on one is refused rather
	// than silently ignored.
	err := ValidateOptions(Options{Source: pg, Destination: sqlite, ConversionReceipt: "/tmp/receipt.json"})
	if err == nil || !strings.Contains(err.Error(), "--verify-only") {
		t.Fatalf("a conversion accepted a receipt: %v", err)
	}
}
