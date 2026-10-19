package portable

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControlOwnershipAuthenticationAndStop(t *testing.T) {
	instance, err := canonicalInstance(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mark, closeControl, err := serveControl(instance, "release", 4350, cancel)
	if err != nil {
		t.Fatal(err)
	}
	defer closeControl(nil)
	var record controlRecord
	if err := readJSON(filepath.Join(instance, ".control.json"), &record); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ token, origin string }{{"", ""}, {"wrong", ""}, {record.Token, "https://example.com"}} {
		req, err := http.NewRequest(http.MethodPost, "http://"+record.Address+"/stop", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+test.token)
		req.Header.Set("Origin", test.origin)
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusForbidden {
			t.Fatal(response.StatusCode)
		}
		if ctx.Err() != nil {
			t.Fatal("unauthorized request stopped instance")
		}
	}
	mark()
	status, err := controlRequest(ctx, instance, http.MethodGet, "/status")
	if err != nil || status.State != "running" || status.Instance != instance {
		t.Fatalf("%+v %v", status, err)
	}
	other := t.TempDir()
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, ".control.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := controlRequest(context.Background(), other, http.MethodGet, "/status"); err == nil {
		t.Fatal("copied capability reported another instance as ours")
	}
	if _, err := controlRequest(context.Background(), instance, http.MethodPost, "/stop"); err != nil {
		t.Fatal(err)
	}
	if ctx.Err() == nil {
		t.Fatal("authenticated stop did not cancel owner")
	}
}

func TestControlRefusesRemoteOrCorruptAddress(t *testing.T) {
	for _, address := range []string{"example.com:80", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:80/path", "[::1]:80"} {
		instance := t.TempDir()
		data, _ := json.Marshal(controlRecord{Address: address, Token: strings.Repeat("a", 64)})
		if err := os.WriteFile(filepath.Join(instance, ".control.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := controlRequest(context.Background(), instance, http.MethodGet, "/status"); err == nil {
			t.Fatal(address)
		}
	}
}

func TestStatusDoesNotGuessFromHeldLock(t *testing.T) {
	instance := filepath.Join(t.TempDir(), "world")
	if err := Init(fixture(t), instance, 4350); err != nil {
		t.Fatal(err)
	}
	status, err := Status(context.Background(), instance)
	if err != nil || status.State != "stopped" {
		t.Fatalf("%+v %v", status, err)
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycleLock(instance); err == nil {
		t.Fatal("simultaneous lifecycle operations accepted")
	}
	if err := unlock(); err != nil {
		t.Fatal(err)
	}
}

func TestExitResultMustMatchAndPropagateFailure(t *testing.T) {
	instance := t.TempDir()
	for _, test := range []struct {
		record exitRecord
		ok     bool
	}{{exitRecord{RunID: "ours"}, true}, {exitRecord{RunID: "other"}, false}, {exitRecord{RunID: "ours", Error: "server did not stop within 30 seconds"}, false}} {
		data, err := json.Marshal(test.record)
		if err != nil {
			t.Fatal(err)
		}
		if err := writePrivate(filepath.Join(instance, ".last-exit.json"), data); err != nil {
			t.Fatal(err)
		}
		err = verifyExit(instance, "ours")
		if (err == nil) != test.ok {
			t.Fatalf("%+v: %v", test, err)
		}
	}
}
