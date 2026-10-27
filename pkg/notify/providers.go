package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	keyPattern       = regexp.MustCompile(`^[A-Za-z0-9]{30}$`)
	webhookPattern   = regexp.MustCompile(`^/api/webhooks/[0-9]+/[A-Za-z0-9_-]+$`)
	heartbeatPattern = regexp.MustCompile(`^/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	providerClient   = &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
)

func providerURL(raw, host string, path *regexp.Regexp) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != host || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || !path.MatchString(u.Path) {
		return "", false
	}
	return u.String(), true
}

func deliverWithCredentials(ctx context.Context, id, message string, credentials Credentials) error {
	_, err := deliverProvider(ctx, id, message, credentials)
	return err
}

// deliverProvider additionally reports Discord's response-driven cooldown.
// Neither response bodies nor credential-bearing URLs escape this boundary.
func deliverProvider(ctx context.Context, id, message string, credentials Credentials) (time.Duration, error) {
	if !validCredentials(id, credentials) {
		return 0, ErrNotReady
	}
	var target, contentType string
	var body io.Reader
	method := http.MethodPost
	switch id {
	case Pushover:
		target = "https://api.pushover.net/1/messages.json"
		form := url.Values{"token": {credentials.Token}, "user": {credentials.User}, "message": {message}, "priority": {"0"}}
		body = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	case Discord, Gossip:
		target, _ = providerURL(credentials.URL, "discord.com", webhookPattern)
		target += "?wait=true"
		data, err := json.Marshal(struct {
			Content         string `json:"content"`
			AllowedMentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}{Content: message, AllowedMentions: struct {
			Parse []string `json:"parse"`
		}{Parse: []string{}}})
		if err != nil {
			return 0, errors.New("could not encode notification")
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	case "heartbeat":
		target, _ = providerURL(credentials.URL, "hc-ping.com", heartbeatPattern)
		method = http.MethodGet
	default:
		return 0, ErrInvalid
	}
	// Addresses are fixed provider hosts with strict path validation. Never follow
	// redirects, expose upstream response bodies, or include credential-bearing URLs
	// in returned errors (including net/http's *url.Error).
	req, err := http.NewRequestWithContext(ctx, method, target, body) // #nosec G704 -- fixed Pushover URL or providerURL-validated HTTPS discord.com/hc-ping.com host and path; redirects refused
	if err != nil {
		return 0, errors.New("could not create provider request")
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := providerClient.Do(req) // #nosec G704 -- target passes providerURL host/path allowlist; client refuses redirects; covered by TestProviderURLRestrictions and TestProviderRejectionAndRedirect
	if err != nil {
		return 0, errors.New("provider request failed or timed out")
	}
	defer resp.Body.Close()
	cooldown := time.Duration(0)
	if id == Gossip {
		cooldown = discordCooldown(resp, time.Now())
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cooldown, fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	if id == Pushover {
		var result struct {
			Status int `json:"status"`
		}
		if err = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil || result.Status != 1 {
			return 0, errors.New("pushover did not accept the message")
		}
	}
	return cooldown, nil
}
