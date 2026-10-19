package notify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func setProviderEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DP_NOTIFY_PUSHOVER_TOKEN", strings.Repeat("a", 30))
	t.Setenv("DP_NOTIFY_PUSHOVER_USER", strings.Repeat("b", 30))
	t.Setenv("DP_NOTIFY_DISCORD_WEBHOOK", "https://discord.com/api/webhooks/123/private-token")
	t.Setenv("DP_NOTIFY_HEARTBEAT_URL", "https://hc-ping.com/12345678-1234-1234-1234-123456789abc")
}

func TestProviderURLRestrictions(t *testing.T) {
	for _, raw := range []string{
		"http://discord.com/api/webhooks/123/token",
		"https://discord.com.evil.test/api/webhooks/123/token",
		"https://127.0.0.1/api/webhooks/123/token",
		"https://discord.com:443/api/webhooks/123/token",
		"https://user:pass@discord.com/api/webhooks/123/token",
		"https://discord.com/api/webhooks/123/token?thread_id=1",
		"https://discord.com/api/webhooks/123/token#fragment",
		"https://discord.com/api/webhooks/123/%74oken",
	} {
		if _, ok := providerURL(raw, "discord.com", webhookPattern); ok {
			t.Fatalf("accepted %s", raw)
		}
	}
	setProviderEnv(t)
	for _, id := range []string{Discord, Pushover, "heartbeat"} {
		if !credentialsReady(id) {
			t.Fatal("valid credential rejected", id)
		}
	}
	t.Setenv("DP_NOTIFY_HEARTBEAT_URL", "https://hc-ping.com/arbitrary/path")
	if credentialsReady("heartbeat") {
		t.Fatal("arbitrary heartbeat URL allowed")
	}
}

func TestDeliveryWireAndSecretRedaction(t *testing.T) {
	setProviderEnv(t)
	original := providerClient.Transport
	t.Cleanup(func() { providerClient.Transport = original })
	providerClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body = string(b)
		}
		responseBody := "{}"
		switch r.URL.Host {
		case "api.pushover.net":
			form, err := url.ParseQuery(body)
			if err != nil {
				t.Fatal(err)
			}
			if r.Method != "POST" || form.Get("token") != strings.Repeat("a", 30) || form.Get("user") != strings.Repeat("b", 30) || form.Get("message") != "Avery died." || form.Get("priority") != "0" {
				t.Fatalf("wrong Pushover request %v", r)
			}
			responseBody = `{"status":1}`
		case "discord.com":
			if r.Method != "POST" || r.URL.Query().Get("wait") != "true" || !strings.Contains(body, `"allowed_mentions":{"parse":[]}`) {
				t.Fatalf("unsafe Discord payload %s", body)
			}
		case "hc-ping.com":
			if r.Method != "GET" || body != "" {
				t.Fatal("heartbeat must contain no player data")
			}
		default:
			t.Fatal("unapproved destination")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(responseBody)), Header: http.Header{}}, nil
	})
	for _, id := range []string{Pushover, Discord, "heartbeat"} {
		if err := deliver(context.Background(), id, "Avery died."); err != nil {
			t.Fatal(err)
		}
	}
	providerClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("private-token " + r.URL.String())
	})
	err := deliver(context.Background(), Discord, "test")
	if err == nil || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "https:") {
		t.Fatal("credential URL leaked", err)
	}
}

func TestProviderRejectionAndRedirect(t *testing.T) {
	setProviderEnv(t)
	original := providerClient.Transport
	t.Cleanup(func() { providerClient.Transport = original })
	for _, tc := range []struct {
		status int
		body   string
	}{{200, `{"status":0}`}, {200, "not json"}, {429, "secret provider response"}, {500, "secret provider response"}} {
		providerClient.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: http.Header{}}, nil
		})
		err := deliver(context.Background(), Pushover, "test")
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("rejection or redaction failure", err)
		}
	}
	calls := 0
	providerClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Location": []string{"https://127.0.0.1/private"}}, Request: r}, nil
	})
	if deliver(context.Background(), Discord, "test") == nil || calls != 1 {
		t.Fatal("redirect followed")
	}
}

func credentialsReady(id string) bool { return validCredentials(id, envCredentials(id)) }
func deliver(ctx context.Context, id, message string) error {
	return deliverWithCredentials(ctx, id, message, envCredentials(id))
}
