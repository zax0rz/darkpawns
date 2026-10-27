// Command admin-openapi exports the admin contract without booting a game server.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zax0rz/darkpawns/pkg/admin"
	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/mudlog"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	dir, err := os.MkdirTemp("", "darkpawns-admin-openapi-")
	if err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, os.RemoveAll(dir)) }()
	if err := os.Setenv("ADMIN_STORE_PATH", filepath.Join(dir, "store.json")); err != nil {
		return err
	}
	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		return err
	}
	defer world.StopAITicker()
	doc := apidoc.New()
	apidoc.RegisterHealth(doc.NewAPI(http.NewServeMux()))
	feed := mudlog.NewFeed(mudlog.DefaultQueue, mudlog.DefaultRingCap)
	defer feed.Stop()
	if _, err := admin.NewRouter(world, nil, admin.NewLogBuffer(1), nil, nil, admin.WithSharedSpec(doc), admin.WithMudlogFeed(feed)); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(doc.OpenAPI())
}
