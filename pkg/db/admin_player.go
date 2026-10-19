package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
)

// AdminSavePlayer atomically edits a saved character, optional credentials and
// the independent crash/rent snapshot. Callers must reserve the offline name.
// It preserves the existing save format and rejects intervening game saves.
func (d *DB) AdminSavePlayer(updated, original *PlayerRecord, password *string, objects bool) (err error) {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			slog.Error("admin player rollback failed", "error", rollbackErr)
		}
	}()
	query, args := playerCompareUpdate(updated, original)
	query += " AND password_hash IS ?"
	args = append(args, original.Password)
	result, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrPlayerRecordChanged
	}
	if password != nil {
		if _, err = tx.Exec("UPDATE players SET password_hash=?, failed_login_attempts=0, locked_until=NULL WHERE id=?", *password, original.ID); err != nil {
			return err
		}
	}
	if objects {
		items, e := objectSaveRecords(updated.Inventory, updated.Equipment)
		if e != nil {
			return e
		}
		raw, e := json.Marshal(items)
		if e != nil {
			return e
		}
		// Preserve an existing rent/cryo kind; a new grant gets a rented snapshot.
		if _, err = tx.Exec("INSERT INTO object_saves(identity,kind,objects) VALUES(?,2,?) ON CONFLICT(identity) DO UPDATE SET objects=excluded.objects", ObjectSaveIdentity(original.Name), raw); err != nil {
			return err
		}
	}
	return tx.Commit()
}
