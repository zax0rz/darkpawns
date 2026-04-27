package privacy

import (
	"context"
	"log/slog"
	"os"
)

// PIIHandler wraps a slog.Handler and filters all log output through the
// privacy filter Client to strip PII before it reaches the output handler.
type PIIHandler struct {
	next   slog.Handler
	client *Client
}

// NewPIIHandler creates a PIIHandler that wraps next and filters all
// log records through the provided privacy Client.
func NewPIIHandler(next slog.Handler, client *Client) *PIIHandler {
	return &PIIHandler{next: next, client: client}
}

// Enabled delegates to the wrapped handler.
func (h *PIIHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle filters the record's message and all string-typed attrs through the
// privacy filter, then passes the filtered record to the wrapped handler.
func (h *PIIHandler) Handle(ctx context.Context, r slog.Record) error {
	// Filter the record message.
	filteredMsg, _, _ := h.client.FilterText(r.Message)

	// Collect filtered attrs. Only filter string values — other kinds pass through.
	var attrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, h.filterAttr(a))
		return true
	})

	// Build a new record with the filtered message and attrs.
	newR := slog.NewRecord(r.Time, r.Level, filteredMsg, r.PC)
	newR.AddAttrs(attrs...)

	return h.next.Handle(ctx, newR) // #nosec G706
}

// filterAttr recursively filters string values inside an Attr.
func (h *PIIHandler) filterAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindString:
		filtered, _, _ := h.client.FilterText(a.Value.String())
		return slog.String(a.Key, filtered)
	case slog.KindGroup:
		groupAttrs := a.Value.Group()
		filtered := make([]any, len(groupAttrs))
		for i, ga := range groupAttrs {
			filtered[i] = h.filterAttr(ga)
		}
		return slog.Group(a.Key, filtered...)
	default:
		return a
	}
}

// WithAttrs delegates to the inner handler and wraps the result.
func (h *PIIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PIIHandler{
		next:   h.next.WithAttrs(attrs),
		client: h.client,
	}
}

// WithGroup delegates to the inner handler and wraps the result.
func (h *PIIHandler) WithGroup(name string) slog.Handler {
	return &PIIHandler{
		next:   h.next.WithGroup(name),
		client: h.client,
	}
}

// InitSlogPII replaces the global slog logger with a PII-filtered handler.
// baseURL is the privacy filter service URL (empty = default).
//
// The default config filters: email, person, phone, address, secret, account_number.
// It intentionally does NOT filter date or url since game logs commonly
// include valid timestamps and WebSocket/game URLs.
func InitSlogPII(baseURL string) {
	config := DefaultFilterConfig()
	config.Categories = []string{
		CategoryEmail,
		CategoryPerson,
		CategoryPhone,
		CategoryAddress,
		CategorySecret,
		CategoryAccountNumber,
	}
	client := NewClient(baseURL, config)

	inner := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ // #nosec G706
		Level: slog.LevelInfo,
	})

	piiHandler := NewPIIHandler(inner, client)
	slog.SetDefault(slog.New(piiHandler))
}
