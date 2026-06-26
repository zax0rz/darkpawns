package agent

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendMemoryEvent_2xxSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewREMSynthesisClient(PythonSystemConfig{
		BaseURL: server.URL,
		Enabled: true,
		Timeout: 5 * time.Second,
	})

	event := &MemoryEvent{AgentName: "tester", EventType: "mob_kill"}
	if err := client.SendMemoryEvent(event); err != nil {
		t.Fatalf("SendMemoryEvent returned error for 204: %v", err)
	}
}

func TestSendMemoryEvent_Repeated5xxReturnsError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewREMSynthesisClient(PythonSystemConfig{
		BaseURL: server.URL,
		Enabled: true,
		Timeout: 5 * time.Second,
	})

	event := &MemoryEvent{AgentName: "tester", EventType: "mob_kill"}
	err := client.SendMemoryEvent(event)
	if err == nil {
		t.Fatal("SendMemoryEvent returned nil error after repeated 500s")
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", calls.Load())
	}
}

func TestSendMemoryEvent_4xxReturnsErrorImmediately(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewREMSynthesisClient(PythonSystemConfig{
		BaseURL: server.URL,
		Enabled: true,
		Timeout: 5 * time.Second,
	})

	event := &MemoryEvent{AgentName: "tester", EventType: "mob_kill"}
	err := client.SendMemoryEvent(event)
	if err == nil {
		t.Fatal("SendMemoryEvent returned nil error for 400")
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 attempt for permanent 4xx, got %d", calls.Load())
	}
}

func TestSendMemoryEvent_429Retries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if calls.Load() == 3 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewREMSynthesisClient(PythonSystemConfig{
		BaseURL: server.URL,
		Enabled: true,
		Timeout: 5 * time.Second,
	})

	event := &MemoryEvent{AgentName: "tester", EventType: "mob_kill"}
	if err := client.SendMemoryEvent(event); err != nil {
		t.Fatalf("SendMemoryEvent returned error after 429 retries succeeded: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", calls.Load())
	}
}
