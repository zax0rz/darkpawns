package optimization

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestBatchProcessor_FlushReturnsError(t *testing.T) {
	wantErr := errors.New("flush failed")
	bp := NewBatchProcessor(2, time.Hour, func(ops []BatchOperation) error {
		return wantErr
	})
	defer bp.Close()

	// Add two operations to trigger a full-batch flush.
	_ = bp.Add(BatchOperation{Type: "insert"})
	if err := bp.Add(BatchOperation{Type: "insert"}); err == nil {
		t.Error("Add expected flush error, got nil")
	}
}

func TestBatchProcessor_CloseReturnsPendingError(t *testing.T) {
	wantErr := errors.New("close flush failed")
	bp := NewBatchProcessor(10, time.Hour, func(ops []BatchOperation) error {
		return wantErr
	})

	_ = bp.Add(BatchOperation{Type: "insert"})
	if err := bp.Close(); !errors.Is(err, wantErr) {
		t.Errorf("Close err = %v, want %v", err, wantErr)
	}
}

func TestBatchedSender_FlushReturnsError(t *testing.T) {
	wantErr := errors.New("send failed")
	bs := NewBatchedSender(time.Hour, 2, func(msgs []json.RawMessage) error {
		return wantErr
	})
	defer bs.Close()

	_ = bs.Send(json.RawMessage(`"hello"`))
	if err := bs.Send(json.RawMessage(`"world"`)); err == nil {
		t.Error("Send expected flush error, got nil")
	}
}

func TestBatchedSender_CloseReturnsPendingError(t *testing.T) {
	wantErr := errors.New("close send failed")
	bs := NewBatchedSender(time.Hour, 10, func(msgs []json.RawMessage) error {
		return wantErr
	})

	_ = bs.Send(json.RawMessage(`"hello"`))
	if err := bs.Close(); !errors.Is(err, wantErr) {
		t.Errorf("Close err = %v, want %v", err, wantErr)
	}
}
