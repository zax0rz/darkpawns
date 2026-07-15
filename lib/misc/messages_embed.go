// Package miscdata embeds the canonical runtime data stored in lib/misc.
package miscdata

import _ "embed"

// FightMessages is the byte-identical CircleMUD fight-message corpus.
//
//go:embed messages
var FightMessages []byte
