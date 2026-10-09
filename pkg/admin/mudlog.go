package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/zax0rz/darkpawns/pkg/mudlog"
)

// The mudlog feed endpoints (mudlog-push design §5, slice 1): replayable
// history, a live SSE tail, and the live key catalog. All of it sits behind
// the router's existing admin auth like every other /admin operation.

type mudlogEvent = mudlog.Event

type mudlogListInput struct {
	Since uint64 `query:"since" doc:"Replay events with seq greater than this."`
	// Type and Level are virtual-immortal filters (MudLog's own delivery
	// predicate); -1 leaves a filter unset so the valid 0 (OFF) value still
	// filters.
	Type  int    `query:"type" default:"-1" doc:"-1 = unset; else only events whose syslog type (OFF/BRF/NRM/CMP, 0-3) is at most this, like an immortal's syslog setting."`
	Level int    `query:"level" default:"-1" doc:"-1 = unset; else only events visible at this immortal level."`
	Key   string `query:"key" doc:"Only events with this C-site key (slice 2 stamps keys; most are unkeyed until then)."`
	Limit int    `query:"limit" doc:"Maximum events returned (default 200, cap 1000)."`
}

type mudlogListOutput struct {
	Body struct {
		Events  []mudlogEvent `json:"events"`
		LastSeq uint64        `json:"last_seq"`
	}
}

// registerMudlog registers the read endpoints for the mudlog feed on api.
func registerMudlog(api huma.API, feed *mudlog.Feed) {
	huma.Register(api, huma.Operation{
		OperationID: "list-mudlog-events",
		Method:      http.MethodGet,
		Path:        "/admin/mudlog",
		Summary:     "Mudlog events",
		Description: "Replayable, filterable tail of the mudlog observer's ring buffer (the same lines immortals see through syslog).",
	}, func(ctx context.Context, in *mudlogListInput) (*mudlogListOutput, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 200
		}
		if limit > mudlog.DefaultRingCap {
			limit = mudlog.DefaultRingCap
		}
		events := feed.Since(in.Since, limit)
		filtered := make([]mudlogEvent, 0, len(events))
		for _, e := range events {
			if in.Key != "" && (e.Key == "" || e.Key != in.Key) {
				continue
			}
			// The filter is MudLog's own delivery predicate: an immortal of
			// Level with syslog Type sees level <= Level and type <= Type.
			if in.Level >= 0 && in.Level < e.Level {
				continue
			}
			if in.Type >= 0 && in.Type < e.Typ {
				continue
			}
			filtered = append(filtered, e)
		}
		out := &mudlogListOutput{}
		out.Body.Events = filtered
		out.Body.LastSeq = feed.LastSeq()
		return out, nil
	})

	type mudlogCatalogOutput struct {
		Body struct {
			Keys []mudlog.KeyStat `json:"keys"`
			Note string           `json:"note"`
		}
	}
	huma.Register(api, huma.Operation{
		OperationID: "mudlog-catalog",
		Method:      http.MethodGet,
		Path:        "/admin/mudlog/catalog",
		Summary:     "Mudlog key catalog",
		Description: "Keys observed in the live ring. The static C-site catalog (labels, payload templates, privacy flags) arrives with the slice-2 key sweep.",
	}, func(ctx context.Context, in *struct{}) (*mudlogCatalogOutput, error) {
		out := &mudlogCatalogOutput{}
		out.Body.Keys = feed.Catalog()
		out.Body.Note = "live keys from the ring; static catalog lands with the key sweep"
		return out, nil
	})
}

// mudlogStream serves GET /admin/mudlog/stream as server-sent events. It is
// a plain handler (not a Huma operation) because SSE needs http.Flusher.
// Clients resume with the Last-Event-ID header or ?since=; the id of every
// event is its ring seq.
func mudlogStream(feed *mudlog.Feed) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Accel-Buffering", "no")

		after := feed.LastSeq()
		if id := r.Header.Get("Last-Event-ID"); id != "" {
			if n, err := strconv.ParseUint(id, 10, 64); err == nil {
				after = n
			}
		}
		if s := r.URL.Query().Get("since"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				after = n
			}
		}

		// A write error means the client is gone: stop streaming.
		write := func(e mudlogEvent) error {
			data, err := json.Marshal(e)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, "id: %d\nevent: mudlog\ndata: %s\n\n", e.Seq, data)
			return err
		}
		ping := func() error {
			_, err := fmt.Fprint(w, ": ping\n\n")
			return err
		}

		var last uint64
		for _, e := range feed.Since(after, 0) {
			if err := write(e); err != nil {
				return
			}
			last = e.Seq
		}
		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}
		flusher.Flush()

		events, kicked, cancel := feed.Subscribe(feed.LastSeq())
		defer cancel()
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-kicked:
				// Too slow for the live channel: replay the gap from the
				// ring and re-subscribe from where we got to.
				for _, e := range feed.Since(last, 0) {
					if err := write(e); err != nil {
						return
					}
					last = e.Seq
				}
				flusher.Flush()
				cancel()
				events, kicked, cancel = feed.Subscribe(last)
			case e, ok := <-events:
				if !ok {
					return
				}
				if err := write(e); err != nil {
					return
				}
				last = e.Seq
				flusher.Flush()
			case <-heartbeat.C:
				if err := ping(); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
