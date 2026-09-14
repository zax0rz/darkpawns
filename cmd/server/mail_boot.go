package main

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// initializePersistentMail prepares the optional mail subsystem after the
// database is available. The C boot path sets no_mail and continues when its
// mail scan fails; the Go server keeps that availability boundary explicit by
// returning an error for the caller to log while leaving the rest of boot
// independent of mail.
func initializePersistentMail(database db.Database) error {
	game.DisableMailSystem()

	identity, err := newMailIdentity(database)
	if err != nil {
		return fmt.Errorf("mail identity initialization: %w", err)
	}
	if !game.InitMailSystem(identity.nameByID, identity.idByName) {
		return fmt.Errorf("mail storage initialization failed")
	}
	return nil
}
