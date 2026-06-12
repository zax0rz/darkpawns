// agentkeygen generates a Dark Pawns agent API key for a character.
//
// Usage:
//
//	DATABASE_URL="postgres://..." go run ./cmd/agentkeygen -name "brenda69"
//
// Output:
//
//	Character: brenda69
//	Key: dp_<32hex>
//	(shown once — store in Vaultwarden)
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/zax0rz/darkpawns/pkg/db"
)

func main() {
	name := flag.String("name", "", "character name to associate the key with")
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: -name is required")
		flag.Usage()
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "error: DATABASE_URL environment variable is required")
		fmt.Fprintln(os.Stderr, "example: export DATABASE_URL='postgres://user:pass@localhost/darkpawns'")
		os.Exit(1)
	}

	if err := run(*name, dsn); err != nil {
		slog.Error("agentkeygen failed", "error", err)
		os.Exit(1)
	}
}

func run(name, dsn string) error {
	database, err := db.New(dsn)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() { _ = database.Close() }()

	return runWithDB(name, database)
}

func runWithDB(name string, database db.Database) error {
	if _, err := database.GetPlayer(name); err != nil {
		return fmt.Errorf("get player %q: %w", name, err)
	}

	rawKey, id, err := database.CreateAgentKey(name)
	if err != nil {
		return fmt.Errorf("create agent key: %w", err)
	}

	fmt.Printf("Character: %s\n", name)
	fmt.Printf("Key (id=%d): %s\n", id, rawKey)
	fmt.Println("(shown once — store in Vaultwarden)")
	return nil
}
