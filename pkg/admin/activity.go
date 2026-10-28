package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/telemetry"
)

type activityInput struct {
	Range  string `query:"range" default:"24h" enum:"24h,7d"`
	Kind   string `query:"kind" default:"login" enum:"login,character,death"`
	Search string `query:"search" maxLength:"32"`
	Before uint64 `query:"before"`
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"100"`
}
type activityResponse struct {
	Entries    []telemetry.Activity `json:"entries"`
	Recording  bool                 `json:"recording"`
	Since      *time.Time           `json:"since,omitempty"`
	Dropped    uint64               `json:"dropped"`
	Trimmed    uint64               `json:"trimmed"`
	Total      int                  `json:"total"`
	NextBefore uint64               `json:"next_before"`
}
type activityOutput struct {
	Body         activityResponse
	CacheControl string `header:"Cache-Control"`
}

func activitySnapshot(h *telemetry.History, in *activityInput, level int, now time.Time) activityResponse {
	out := activityResponse{Entries: []telemetry.Activity{}}
	if h == nil {
		return out
	}
	s := h.Snapshot()
	out.Recording, out.Since, out.Dropped, out.Trimmed = true, &s.ActivitySince, s.ActivityDropped, s.ActivityTrimmed
	width, bins := time.Hour, 24
	if in.Range == "7d" {
		width, bins = 24*time.Hour, 7
	}
	start := now.UTC().Truncate(width).Add(-time.Duration(bins-1) * width)
	for i := len(s.Activity) - 1; i >= 0; i-- {
		e := s.Activity[i]
		if e.Kind != in.Kind || e.MinLevel > level || e.MinLevel < 0 || e.Time.Before(start) || e.Time.After(now) || !strings.Contains(strings.ToLower(e.Player), strings.ToLower(in.Search)) {
			continue
		}
		out.Total++
		if in.Before != 0 && e.ID >= in.Before {
			continue
		}
		if len(out.Entries) < in.Limit {
			out.Entries = append(out.Entries, e)
		} else if out.NextBefore == 0 {
			out.NextBefore = out.Entries[len(out.Entries)-1].ID
		}
	}
	return out
}

func registerActivity(api huma.API, h *telemetry.History, callerLevel func(context.Context) int) {
	huma.Register(api, huma.Operation{OperationID: "get-activity", Method: http.MethodGet, Path: "/admin/activity/events", Summary: "Recorded character activity"}, func(ctx context.Context, in *activityInput) (*activityOutput, error) {
		return &activityOutput{Body: activitySnapshot(h, in, callerLevel(ctx), time.Now().UTC()), CacheControl: "no-store"}, nil
	})
}
