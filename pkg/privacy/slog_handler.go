package privacy

import (
	"context"
	"log/slog"
	"os"
)

// PIIHandler wraps a slog.Handler and filters all log output through the
// privacy filter Client to strip PII before it reaches the output handler.
type PIIHandler struct {
	next      slog.Handler
	client    *Client
	filterErr error
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
// If the filter service fails, the original message and attr values are kept
// and the record is annotated with a pii_filter_error attr so operators can
// detect filter-service degradation instead of silently losing log content.
func (h *PIIHandler) Handle(ctx context.Context, r slog.Record) error {
	// Filter the record message.
	filteredMsg, _, msgErr := h.client.FilterText(r.Message)

	// Collect filtered attrs. Only filter string values — other kinds pass through.
	var attrs []slog.Attr
	var attrErr error
	r.Attrs(func(a slog.Attr) bool {
		fa, err := h.filterAttr(a)
		if err != nil && attrErr == nil {
			attrErr = err
		}
		attrs = append(attrs, fa)
		return true
	})

	// On filter failure, keep the original content so it is not silently
	// replaced with the fallback sentinel, and surface the error on the record.
	switch {
	case msgErr != nil:
		filteredMsg = r.Message
		attrs = append(attrs, slog.String("pii_filter_error", msgErr.Error()))
	case attrErr != nil:
		attrs = append(attrs, slog.String("pii_filter_error", attrErr.Error()))
	case h.filterErr != nil:
		attrs = append(attrs, slog.String("pii_filter_error", h.filterErr.Error()))
	}

	// Build a new record with the filtered message and attrs.
	newR := slog.NewRecord(r.Time, r.Level, filteredMsg, r.PC)
	newR.AddAttrs(attrs...)

	return h.next.Handle(ctx, newR) // #nosec G706
}

// filterAttr recursively filters string values inside an Attr. On filter
// failure the original Attr is returned unchanged so the value is not lost.
func (h *PIIHandler) filterAttr(a slog.Attr) (slog.Attr, error) {
	switch a.Value.Kind() {
	case slog.KindString:
		filtered, _, err := h.client.FilterText(a.Value.String())
		if err != nil {
			return a, err
		}
		return slog.String(a.Key, filtered), nil
	case slog.KindGroup:
		groupAttrs := a.Value.Group()
		filtered := make([]any, len(groupAttrs))
		var groupErr error
		for i, ga := range groupAttrs {
			fa, err := h.filterAttr(ga)
			if err != nil && groupErr == nil {
				groupErr = err
			}
			filtered[i] = fa
		}
		return slog.Group(a.Key, filtered...), groupErr
	default:
		return a, nil
	}
}

// WithAttrs filters the attrs through the privacy filter before handing them to
// the inner handler. Handler-level attrs are prepended by the inner handler at
// Handle time, after PIIHandler.Handle has already run, so they would otherwise
// bypass filtering entirely (Record.Attrs only iterates per-call attrs).
func (h *PIIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	filtered := make([]slog.Attr, len(attrs))
	var attrErr error
	for i, a := range attrs {
		fa, err := h.filterAttr(a)
		if err != nil && attrErr == nil {
			attrErr = err
		}
		filtered[i] = fa
	}
	return &PIIHandler{
		next:      h.next.WithAttrs(filtered),
		client:    h.client,
		filterErr: attrErr,
	}
}

// WithGroup delegates to the inner handler and wraps the result.
func (h *PIIHandler) WithGroup(name string) slog.Handler {
	return &PIIHandler{
		next:      h.next.WithGroup(name),
		client:    h.client,
		filterErr: h.filterErr,
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
