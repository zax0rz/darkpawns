// Module db-migrate is the operator-only PostgreSQL -> SQLite conversion bridge.
//
// It is a separate Go module on purpose. The shipped game runtime is SQLite-only
// and must not compile or depend on lib/pq, but the conversion bridge still needs
// a PostgreSQL driver while production PostgreSQL remains the rollback authority.
// A nested module is what keeps both true at once: it may import the runtime
// (to build the destination with the production schema initialisation), the
// runtime cannot import it, and `go build ./...` in the repository root never
// sees it.
module github.com/zax0rz/darkpawns/tools/db-migrate

go 1.26.9

require (
	github.com/lib/pq v1.12.3
	github.com/zax0rz/darkpawns v0.0.0
	golang.org/x/crypto v0.54.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/yuin/gopher-lua v1.1.2 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/sys v0.47.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	modernc.org/libc v1.75.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.59.0 // indirect
)

// The tool reads the runtime's schema initialisation and character codec, so it
// builds against this checkout rather than a published version: the two must
// never drift.
replace github.com/zax0rz/darkpawns => ../..
