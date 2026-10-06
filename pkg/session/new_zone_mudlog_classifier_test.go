package session

import (
	"errors"
	"os"
	"testing"
)

func TestNewZoneMudlogOpenClassifier(t *testing.T) {
	w := makeZeditTestWorld(t)
	_ = newTestManager(t, w, nil)
	file := captureMudlogFile(t)
	for _, err := range []error{
		errors.New("unclassified failure"),
		&os.PathError{Op: "write", Path: "31.zon", Err: os.ErrPermission},
		&os.PathError{Op: "close", Path: "31.zon", Err: os.ErrPermission},
		&os.PathError{Op: "stat", Path: "31.zon", Err: os.ErrPermission},
		&os.PathError{Op: "open", Path: "31.unknown", Err: os.ErrPermission},
	} {
		zeditLogNewZoneOpenFailure(err)
	}
	if file.Len() != 0 {
		t.Fatalf("non-fopen failures must not invent C broadcasts: %q", file.String())
	}
}
