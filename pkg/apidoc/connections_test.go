package apidoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectionInfoReportsObservedListenersAndClearsStoppedOnes(t *testing.T) {
	mux := http.NewServeMux()
	doc := New()
	current := ConnectionInfo{}
	RegisterConnections(doc.NewAPI(mux), func() ConnectionInfo { return current })
	read := func() map[string]json.RawMessage {
		t.Helper()
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, httptest.NewRequest("GET", "/connection-info", nil))
		if r.Code != 200 {
			t.Fatalf("status=%d %s", r.Code, r.Body.String())
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	if len(read()) != 0 {
		t.Fatal("advertised disabled listeners")
	}
	current.Telnet = ConnectionAt("127.0.0.1:8123")
	current.TelnetTLS = ConnectionAt("[::1]:8124")
	body := read()
	if len(body) != 2 {
		t.Fatal(body)
	}
	var plain ConnectionListener
	if err := json.Unmarshal(body["telnet"], &plain); err != nil {
		t.Fatal(err)
	}
	if plain.Bind != "127.0.0.1" || plain.Port != 8123 {
		t.Fatal(plain)
	}
	current = ConnectionInfo{}
	if len(read()) != 0 {
		t.Fatal("advertised closed listeners")
	}
	for _, invalid := range []string{"", ":0", "not-an-address", "127.0.0.1:65536"} {
		if ConnectionAt(invalid) != nil {
			t.Fatal(invalid)
		}
	}
	if doc.OpenAPI().Paths["/connection-info"].Get == nil {
		t.Fatal("missing OpenAPI contract")
	}
}
