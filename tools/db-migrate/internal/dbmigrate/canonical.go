package dbmigrate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// nullSentinel is the canonical form of SQL NULL. It cannot collide with a real
// value: no column in this schema stores this byte.
const nullSentinel = "\x00null"

// timestampLayouts are the spellings a timestamp can arrive in. PostgreSQL hands
// back time.Time for its timestamp types; SQLite's driver hands back time.Time
// when it recognises the stored text and a string when it does not.
var timestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	// SQLite stores text verbatim, so a value written as text comes back in the
	// spelling it was written with. "-07" is the offset without a colon and
	// with nothing after it ("+00", "-04"), which RFC3339 never produces and
	// modernc hands over exactly as stored.
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999-07",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05-07",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999Z07",
	"2006-01-02T15:04:05.999999999-07",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Canonicalize renders one scanned value in the comparison form for its kind.
//
// The rules, and why each exists:
//
//   - NULL is one sentinel, so a null is never confused with the empty string.
//   - integers are decimal text: PostgreSQL int4 arrives as int64 and SQLite
//     INTEGER as int64, so this normalises the box and never the value. A
//     negative room_vnum sentinel survives unchanged.
//   - booleans are "1"/"0": PostgreSQL BOOLEAN scans to bool, SQLite's declared
//     BOOLEAN scans to int64, and the app writes either through one statement.
//   - timestamps are UTC RFC3339Nano instants. PostgreSQL timestamptz and SQLite
//     TIMESTAMP both carry an instant, and SQLite preserves the offset it was
//     written with, so comparing local wall clocks would be wrong and comparing
//     instants is right. A literal with no zone is read as UTC, matching SQLite's
//     CURRENT_TIMESTAMP, and that assumption is reported rather than hidden.
//   - JSON is compared semantically: decoded with json.Number (so an integer
//     literal is not rounded through float64) and re-encoded compactly with keys
//     sorted. PostgreSQL jsonb re-orders and re-spaces, so a byte comparison
//     would report a difference where the value is identical; the raw bytes are
//     compared separately and reported as reformatting.
//   - intervals are text. PostgreSQL renders '1h30m' as '01:30:00' and the
//     runtime never reads this column, so text is both the honest and the exact
//     comparison.
func Canonicalize(kind ValueKind, value any) (string, error) {
	if value == nil {
		return nullSentinel, nil
	}
	switch kind {
	case KindInteger:
		n, err := asInt64(value)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(n, 10), nil
	case KindBoolean:
		b, err := asBool(value)
		if err != nil {
			return "", err
		}
		if b {
			return "1", nil
		}
		return "0", nil
	case KindTimestamp:
		t, err := asTime(value)
		if err != nil {
			return "", err
		}
		return t.UTC().Format(time.RFC3339Nano), nil
	case KindJSON:
		text, err := asText(value)
		if err != nil {
			return "", err
		}
		return canonicalJSON(text)
	case KindInterval, KindText:
		return asText(value)
	default:
		return "", fmt.Errorf("unknown value kind %q", kind)
	}
}

// RawText renders value exactly as the driver handed it over, for the
// reformatting note. It is only meaningful for JSON columns.
func RawText(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	return asText(value)
}

func asInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d does not fit in a signed 64-bit integer", v)
		}
		return int64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("cannot read %T as an integer", value)
	}
}

func asBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int64:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int:
		return v != 0, nil
	case []byte:
		return parseBoolText(string(v))
	case string:
		return parseBoolText(v)
	default:
		return false, fmt.Errorf("cannot read %T as a boolean", value)
	}
}

func parseBoolText(text string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "1", "t", "true":
		return true, nil
	case "0", "f", "false":
		return false, nil
	}
	return false, fmt.Errorf("cannot read %q as a boolean", text)
}

func asTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case []byte:
		return parseTimeText(string(v))
	case string:
		return parseTimeText(v)
	default:
		return time.Time{}, fmt.Errorf("cannot read %T as a timestamp", value)
	}
}

func parseTimeText(text string) (time.Time, error) {
	trimmed := strings.TrimSpace(text)
	for _, layout := range timestampLayouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot read %q as a timestamp", text)
}

func asText(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int:
		return strconv.Itoa(v), nil
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	default:
		return "", fmt.Errorf("cannot read %T as text", value)
	}
}

// canonicalJSON re-encodes a JSON document with sorted keys and no insignificant
// whitespace. json.Number keeps numeric literals exactly as written, so a change
// to a stored number is still reported.
func canonicalJSON(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", nil
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("column holds %d bytes that are not JSON: %w", len(text), err)
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("re-encode JSON: %w", err)
	}
	return strings.TrimRight(buffer.String(), "\n"), nil
}

// JSONReformatOnly reports whether two raw JSON texts decode to the same value.
// It is the difference between "the copy lost something" and "one dialect
// re-spaced the same document", and the receipt states which one happened.
func JSONReformatOnly(left, right string) bool {
	leftCanonical, leftErr := canonicalJSON(left)
	rightCanonical, rightErr := canonicalJSON(right)
	return leftErr == nil && rightErr == nil && leftCanonical == rightCanonical
}

// RowDigest is the per-row content digest: one canonical value per copied column,
// in column order, so two rows are equal exactly when every column is.
func RowDigest(columns, values []string) string {
	hash := sha256.New()
	for i, column := range columns {
		hash.Write([]byte(column))
		hash.Write([]byte{'='})
		if i < len(values) {
			hash.Write([]byte(values[i]))
		}
		hash.Write([]byte{0x1f})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// TableDigest is the whole-table digest. It is computed over the sorted row
// digests, so it is independent of scan order: a migration that preserved every
// row but changed the physical order of the copy must not look like a change.
func TableDigest(rowDigests []string) string {
	sorted := append([]string(nil), rowDigests...)
	sort.Strings(sorted)
	hash := sha256.New()
	for _, digest := range sorted {
		hash.Write([]byte(digest))
		hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// SetDigest digests a set of key strings, order-independently.
func SetDigest(keys []string) string { return TableDigest(keys) }
