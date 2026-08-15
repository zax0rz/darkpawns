// Package contact provides the public website contact endpoint.
package contact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxBodyBytes = 16 << 10

type submission struct {
	Character string `json:"character"`
	Years     string `json:"years"`
	Email     string `json:"email"`
	Message   string `json:"message"`
	Website   string `json:"website"`
	Turnstile string `json:"turnstile"`
}

type verifier interface {
	Verify(context.Context, string, string) error
}

type sender interface {
	Send(submission) error
}

// Handler validates and delivers public contact submissions.
type Handler struct {
	verify verifier
	send   sender
	limit  *rateLimiter
}

// NewFromEnvironment constructs a contact handler without exposing its
// destination or credentials to the static website.
func NewFromEnvironment() (*Handler, error) {
	to := os.Getenv("CONTACT_TO")
	host := os.Getenv("CONTACT_SMTP_HOST")
	port := os.Getenv("CONTACT_SMTP_PORT")
	user := os.Getenv("CONTACT_SMTP_USER")
	password := os.Getenv("CONTACT_SMTP_PASSWORD")
	turnstileSecret := os.Getenv("CONTACT_TURNSTILE_SECRET")
	if to == "" || host == "" || port == "" || user == "" || password == "" || turnstileSecret == "" {
		return nil, errors.New("contact delivery environment is incomplete")
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return nil, fmt.Errorf("invalid CONTACT_TO: %w", err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid CONTACT_SMTP_PORT: %w", err)
	}

	return &Handler{
		verify: &turnstileVerifier{secret: turnstileSecret, client: &http.Client{Timeout: 8 * time.Second}},
		send: &smtpSender{
			address: net.JoinHostPort(host, port), host: host, username: user,
			password: password, from: user, to: to,
		},
		limit: newRateLimiter(5, time.Hour),
	}, nil
}

// UnavailableHandler returns a safe response when private delivery settings
// have not been installed on the server.
func UnavailableHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, "Contact is temporarily unavailable.")
	})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, "Use the contact form to send a message.")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, "The contact form could not be read.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var form submission
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&form); err != nil {
		writeJSON(w, http.StatusBadRequest, "The contact form could not be read.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, "The contact form could not be read.")
		return
	}

	if form.Website != "" {
		writeJSON(w, http.StatusOK, "Your message has been sent.")
		return
	}
	if message := validate(form); message != "" {
		writeJSON(w, http.StatusBadRequest, message)
		return
	}

	clientIP := remoteIP(r)
	if !h.limit.Allow(clientIP, time.Now()) {
		writeJSON(w, http.StatusTooManyRequests, "Too many messages have been sent from this connection. Try again later.")
		return
	}
	if err := h.verify.Verify(r.Context(), form.Turnstile, clientIP); err != nil {
		writeJSON(w, http.StatusBadRequest, "The spam check did not pass. Please try again.")
		return
	}
	if err := h.send.Send(form); err != nil {
		slog.Error("contact delivery failed", "error", err)
		writeJSON(w, http.StatusBadGateway, "Your message could not be sent. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, "Your message has been sent.")
}

func validate(form submission) string {
	form.Email = strings.TrimSpace(form.Email)
	form.Message = strings.TrimSpace(form.Message)
	if len(form.Character) > 80 || len(form.Years) > 80 || len(form.Email) > 254 || len(form.Message) > 4000 {
		return "One or more fields are too long."
	}
	if form.Email == "" || form.Message == "" || form.Turnstile == "" {
		return "A reply address, message, and completed spam check are required."
	}
	address, err := mail.ParseAddress(form.Email)
	if err != nil || address.Address != form.Email {
		return "Enter a valid reply address."
	}
	if len(form.Message) < 20 {
		return "The message needs a little more detail."
	}
	return ""
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	parsed := net.ParseIP(host)
	if parsed != nil && parsed.IsLoopback() {
		forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
		if net.ParseIP(forwarded) != nil {
			return forwarded
		}
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": message}); err != nil {
		slog.Warn("contact response write failed", "error", err)
	}
}

type turnstileVerifier struct {
	secret string
	client *http.Client
}

func (v *turnstileVerifier) Verify(ctx context.Context, token, remoteIP string) error {
	values := url.Values{"secret": {v.secret}, "response": {token}, "remoteip": {remoteIP}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<10)).Decode(&result); err != nil {
		return err
	}
	if !result.Success {
		return errors.New("turnstile rejected submission")
	}
	return nil
}

type smtpSender struct {
	address, host, username, password, from, to string
}

func (s *smtpSender) Send(form submission) error {
	subjectName := strings.NewReplacer("\r", " ", "\n", " ").Replace(strings.TrimSpace(form.Character))
	if subjectName == "" {
		subjectName = "website visitor"
	}
	body := fmt.Sprintf(
		"To: %s\r\nFrom: %s\r\nReply-To: %s\r\nSubject: Dark Pawns contact from %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nCharacter: %s\r\nYears played: %s\r\nReply address: %s\r\n\r\n%s\r\n",
		s.to, s.from, form.Email, subjectName, form.Character, form.Years, form.Email, form.Message,
	)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)
	return smtp.SendMail(s.address, auth, s.from, []string{s.to}, []byte(body))
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	maxKeys int
	entries map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, maxKeys: 10_000, entries: make(map[string][]time.Time)}
}

func (l *rateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.entries[key]; !exists && len(l.entries) >= l.maxKeys {
		return false
	}
	cutoff := now.Add(-l.window)
	kept := l.entries[key][:0]
	for _, stamp := range l.entries[key] {
		if stamp.After(cutoff) {
			kept = append(kept, stamp)
		}
	}
	if len(kept) >= l.limit {
		l.entries[key] = kept
		return false
	}
	l.entries[key] = append(kept, now)
	return true
}
