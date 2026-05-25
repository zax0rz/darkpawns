package session

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// sendDiscordNotification dispatches a content payload to a Discord Webhook.
// Executes asynchronously in a non-blocking background goroutine with a 5-second timeout.
func sendDiscordNotification(content string) {
	url := os.Getenv("DISCORD_WEBHOOK_URL")
	if url == "" {
		return
	}

	go func() {
		payload := map[string]string{
			"content": content,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			slog.Error("Discord: failed to marshal payload", "error", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
		if err != nil {
			slog.Error("Discord: failed to create request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("Discord: webhook delivery failed", "error", err)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			slog.Error("Discord: webhook returned non-success code", "status", resp.Status)
		}
	}()
}
