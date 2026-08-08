package agent

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
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

func TestConvertNarrativeMemoryToEvent_MarshalError(t *testing.T) {
	mem := &db.NarrativeMemory{
		ID:        42,
		AgentName: "tester",
		EventType: db.NarrEventMobKill,
		Summary:   "Killed a rat.",
	}

	unmarshalable := struct {
		Name string
		Ch   chan int
	}{Name: "with-channel"}

	event := ConvertNarrativeMemoryToEvent(mem, unmarshalable)
	if event.RawEventData != "" {
		t.Errorf("expected empty RawEventData on marshal failure, got %q", event.RawEventData)
	}
	if event.EventID != "mem_42" {
		t.Errorf("expected EventID mem_42, got %q", event.EventID)
	}
	if event.Summary != mem.Summary {
		t.Errorf("expected Summary %q, got %q", mem.Summary, event.Summary)
	}
}

func TestConvertNarrativeMemoryToEvent_ValidRawEvent(t *testing.T) {
	mem := &db.NarrativeMemory{
		ID:        43,
		AgentName: "tester",
		EventType: db.NarrEventMobKill,
	}

	event := ConvertNarrativeMemoryToEvent(mem, map[string]string{"victim": "rat"})
	if event.RawEventData == "" {
		t.Fatal("expected non-empty RawEventData for marshalable raw event")
	}
}
