// agentkeygen generates a Dark Pawns agent API key for a character.
//
// Usage:
//
//	go run ./cmd/agentkeygen -name "brenda69" -db "postgres://..."
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
	dsn := flag.String("db", "", "postgres connection string")
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: -name is required")
		flag.Usage()
		os.Exit(1)
	}
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "error: -db is required")
		flag.Usage()
		os.Exit(1)
	}

	database, err := db.New(*dsn)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	rawKey, id, err := database.CreateAgentKey(*name)
	if err != nil {
		slog.Error("create agent key", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Character: %s\n", *name)
	fmt.Printf("Key (id=%d): %s\n", id, rawKey)
	fmt.Println("(shown once — store in Vaultwarden)")
}
