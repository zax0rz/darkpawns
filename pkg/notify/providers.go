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
	"os"
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

func credentialsReady(id string) bool {
	switch id {
	case Pushover:
		return keyPattern.MatchString(os.Getenv("DP_NOTIFY_PUSHOVER_TOKEN")) && keyPattern.MatchString(os.Getenv("DP_NOTIFY_PUSHOVER_USER"))
	case Discord:
		_, ok := providerURL(os.Getenv("DP_NOTIFY_DISCORD_WEBHOOK"), "discord.com", webhookPattern)
		return ok
	case "heartbeat":
		_, ok := providerURL(os.Getenv("DP_NOTIFY_HEARTBEAT_URL"), "hc-ping.com", heartbeatPattern)
		return ok
	}
	return false
}

func deliver(ctx context.Context, id, message string) error {
	if !credentialsReady(id) {
		return ErrNotReady
	}
	var target, contentType string
	var body io.Reader
	method := http.MethodPost
	switch id {
	case Pushover:
		target = "https://api.pushover.net/1/messages.json"
		form := url.Values{"token": {os.Getenv("DP_NOTIFY_PUSHOVER_TOKEN")}, "user": {os.Getenv("DP_NOTIFY_PUSHOVER_USER")}, "message": {message}, "priority": {"0"}}
		body = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	case Discord:
		target, _ = providerURL(os.Getenv("DP_NOTIFY_DISCORD_WEBHOOK"), "discord.com", webhookPattern)
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
			return errors.New("could not encode notification")
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	case "heartbeat":
		target, _ = providerURL(os.Getenv("DP_NOTIFY_HEARTBEAT_URL"), "hc-ping.com", heartbeatPattern)
		method = http.MethodGet
	default:
		return ErrInvalid
	}
	// Addresses are fixed provider hosts with strict path validation. Never follow
	// redirects, expose upstream response bodies, or include credential-bearing URLs
	// in returned errors (including net/http's *url.Error).
	req, err := http.NewRequestWithContext(ctx, method, target, body) // #nosec G704 -- fixed Pushover URL or providerURL-validated HTTPS discord.com/hc-ping.com host and path; redirects refused
	if err != nil {
		return errors.New("could not create provider request")
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := providerClient.Do(req) // #nosec G704 -- target passes providerURL host/path allowlist; client refuses redirects; covered by TestProviderURLRestrictions and TestProviderRejectionAndRedirect
	if err != nil {
		return errors.New("provider request failed or timed out")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	if id == Pushover {
		var result struct {
			Status int `json:"status"`
		}
		if err = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil || result.Status != 1 {
			return errors.New("pushover did not accept the message")
		}
	}
	return nil
}
