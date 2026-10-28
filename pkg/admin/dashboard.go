package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

type (
	dashboardInput struct {
		Range string `query:"range" default:"24h" enum:"24h,7d"`
	}
	dashboardPoint struct {
		Time    time.Time `json:"time"`
		Online  *int      `json:"online"`
		Peak    *int      `json:"peak"`
		Samples int       `json:"samples"`
		telemetry.Counts
	}
)

type dashboardResponse struct {
	Now              time.Time         `json:"now"`
	StartedAt        time.Time         `json:"started_at"`
	UptimeSeconds    int64             `json:"uptime_seconds"`
	Online           int               `json:"online"`
	RecordingSince   *time.Time        `json:"recording_since,omitempty"`
	Recording        bool              `json:"recording"`
	PersistenceError string            `json:"persistence_error,omitempty"`
	Release          telemetry.Release `json:"release"`
	Points           []dashboardPoint  `json:"points"`
}
type dashboardOutput struct{ Body dashboardResponse }

func dashboardSnapshot(world *game.World, h *telemetry.History, span string, now time.Time) dashboardResponse {
	out := dashboardResponse{Now: now, StartedAt: processStartTime, UptimeSeconds: int64(now.Sub(processStartTime).Seconds()), Online: world.GetPlayerCount(), Points: []dashboardPoint{}}
	if h == nil {
		return out
	}
	s := h.Snapshot()
	out.Recording = true
	out.RecordingSince = &s.StartedAt
	out.Release = s.Release
	out.PersistenceError = s.PersistenceError
	width := time.Hour
	bins := 24
	if span == "7d" {
		width = 24 * time.Hour
		bins = 7
	}
	start := now.UTC().Truncate(width).Add(-time.Duration(bins-1) * width)
	for i := 0; i < bins; i++ {
		out.Points = append(out.Points, dashboardPoint{Time: start.Add(time.Duration(i) * width)})
	}
	for _, p := range s.Points {
		i := int(p.Time.Sub(start) / width)
		if p.Time.Before(start) || i >= bins {
			continue
		}
		b := &out.Points[i]
		v := p.Online
		b.Online = &v
		if b.Peak == nil || p.Peak > *b.Peak {
			peak := p.Peak
			b.Peak = &peak
		}
		b.Samples += p.Samples
		b.Logins += p.Logins
		b.Characters += p.Characters
		b.Deaths += p.Deaths
		b.Warnings += p.Warnings
		b.Errors += p.Errors
	}
	return out
}

func registerDashboard(api huma.API, world *game.World, h *telemetry.History) {
	huma.Register(api, huma.Operation{OperationID: "get-dashboard", Method: http.MethodGet, Path: "/admin/dashboard", Summary: "Retained dashboard observations"}, func(ctx context.Context, in *dashboardInput) (*dashboardOutput, error) {
		return &dashboardOutput{Body: dashboardSnapshot(world, h, in.Range, time.Now().UTC())}, nil
	})
}
