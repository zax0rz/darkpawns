package portable

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestReadinessWaitsForHealthyStatus(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	if err := waitForHTTP(context.Background(), server.URL, time.Second); err != nil {
		t.Fatal(err)
	}
	if count.Load() < 3 {
		t.Fatal("readiness accepted unhealthy status")
	}
}

func TestReadinessDoesNotFollowRedirectsAndTimesOut(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("followed readiness redirect") }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	if err := waitForHTTP(context.Background(), source.URL, 150*time.Millisecond); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestReadinessCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForHTTP(ctx, "http://127.0.0.1:1/health", time.Second); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
