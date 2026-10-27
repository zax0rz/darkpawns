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

// mudlogCallerLevel resolves the immortal level whose mudlog visibility the
// current request is entitled to (L2-022/028: the caller's actual level, not
// a client-chosen query value). The router supplies a claims+store-backed
// resolver; direct handler tests inject their own.
type mudlogCallerLevel func(ctx context.Context) int

// registerMudlog registers the read endpoints for the mudlog feed on api.
func registerMudlog(api huma.API, feed *mudlog.Feed, callerLevel mudlogCallerLevel) {
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
		// The delivery predicate is MudLog's own: an immortal of Level with
		// syslog Type sees level <= Level and type <= Type. The level ceiling
		// is the caller's actual level; a query level may only narrow it
		// further, never lift it above what the caller is entitled to read.
		ceiling := callerLevel(ctx)
		if in.Level >= 0 && in.Level < ceiling {
			ceiling = in.Level
		}
		filtered := make([]mudlogEvent, 0, len(events))
		for _, e := range events {
			if in.Key != "" && (e.Key == "" || e.Key != in.Key) {
				continue
			}
			if e.Level < 0 || ceiling < e.Level {
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
		out.Body.Keys = []mudlog.KeyStat{}
		byKey := map[string]int{}
		ceiling := callerLevel(ctx)
		for _, e := range feed.Since(0, 0) {
			if e.Level < 0 || e.Level > ceiling {
				continue
			}
			key := e.Key
			if key == "" {
				key = "unkeyed"
			}
			idx, ok := byKey[key]
			if !ok {
				idx = len(out.Body.Keys)
				byKey[key] = idx
				out.Body.Keys = append(out.Body.Keys, mudlog.KeyStat{Key: key})
			}
			st := &out.Body.Keys[idx]
			st.Count++
			st.Level = e.Level
			st.Type = e.Typ
		}
		out.Body.Note = "live keys from the ring; static catalog lands with the key sweep"
		return out, nil
	})
}

// mudlogStream serves GET /admin/mudlog/stream as server-sent events. It is
// a plain handler (not a Huma operation) because SSE needs http.Flusher.
// Clients resume with the Last-Event-ID header or ?since=; the id of every
// event is its ring seq. Every delivered event respects the caller's
// immortal level (L2-028): the stream was previously unfiltered.
func mudlogStream(feed *mudlog.Feed, callerLevel mudlogCallerLevel) http.HandlerFunc {
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

		ceiling := callerLevel(r.Context())
		visible := func(e mudlogEvent) bool { return e.Level >= 0 && e.Level <= ceiling }
		// Subscribe before reading history so an event tapped during replay is
		// either in the ring snapshot, the live channel, or both. `last` drops
		// that overlap and starts at the resume point so history is not
		// re-delivered.
		last := after
		events, kicked, cancel := feed.Subscribe(last)
		defer func() { cancel() }()
		process := func(e mudlogEvent) (bool, error) {
			if e.Seq <= last {
				return false, nil
			}
			last = e.Seq
			if !visible(e) {
				return false, nil
			}
			return true, write(e)
		}
		for _, e := range feed.Since(after, 0) {
			if _, err := process(e); err != nil {
				return
			}
		}
		if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
			return
		}
		flusher.Flush()

		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-kicked:
				// Too slow for the live channel: replay the gap from the
				// ring. Register the replacement first so taps during the
				// replay cannot fall between subscriptions.
				nextEvents, nextKicked, nextCancel := feed.Subscribe(last)
				oldCancel := cancel
				events, kicked, cancel = nextEvents, nextKicked, nextCancel
				oldCancel()
				for _, e := range feed.Since(last, 0) {
					if _, err := process(e); err != nil {
						return
					}
				}
				flusher.Flush()
			case e, ok := <-events:
				if !ok {
					return
				}
				delivered, err := process(e)
				if err != nil {
					return
				}
				if delivered {
					flusher.Flush()
				}
			case <-heartbeat.C:
				// Re-resolve the ceiling so a demotion takes effect on an
				// open stream without waiting for a reconnect (the same
				// every-request semantics as the role guard).
				ceiling = callerLevel(r.Context())
				if err := ping(); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
