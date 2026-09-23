// Package mudlet holds the Dark Pawns Mudlet client package: the Lua sources
// in src/, the importable darkpawns.xml generated from them by
// cmd/mudlet-package, and the version both carry.
package mudlet

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is the package version. The server announces it in Client.GUI, and
// Mudlet reinstalls the package whenever it changes, so it must be bumped
// (with a CHANGELOG.md entry) for every change to src/.
var Version = strings.TrimSpace(versionFile)
