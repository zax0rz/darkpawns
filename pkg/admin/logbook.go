package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/mudlog"
)

type LogbookInput struct {
	Source   string `query:"source" default:"server" enum:"server,mudlog"`
	Severity string `query:"severity" enum:",INFO,WARN,ERROR,DEBUG"`
	Type     int    `query:"type" default:"-1" minimum:"-1" maximum:"3"`
	Level    int    `query:"level" default:"-1" minimum:"-1" maximum:"40"`
	Key      string `query:"key"`
	Search   string `query:"search" maxLength:"500"`
	After    string `query:"after" doc:"Inclusive RFC3339 timestamp."`
	Before   string `query:"before" doc:"Inclusive RFC3339 timestamp."`
}
type logEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Raw     string `json:"raw"`
	Seq     uint64 `json:"seq,omitempty"`
}
type logbookOutput struct {
	Body struct {
		Entries  []logEntry `json:"entries"`
		Retained int        `json:"retained"`
		LastSeq  uint64     `json:"last_seq"`
		Dropped  int64      `json:"dropped"`
	}
}

func logbookEntries(ctx context.Context, in *LogbookInput, buf *LogBuffer, feed *mudlog.Feed, caller mudlogCallerLevel) (*logbookOutput, error) {
	var after, before time.Time
	var err error
	if in.After != "" {
		after, err = time.Parse(time.RFC3339, in.After)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid after timestamp")
		}
	}
	if in.Before != "" {
		before, err = time.Parse(time.RFC3339, in.Before)
		if err != nil {
			return nil, huma.Error400BadRequest("Invalid before timestamp")
		}
	}
	if !after.IsZero() && !before.IsZero() && before.Before(after) {
		return nil, huma.Error400BadRequest("before must follow after")
	}
	out := &logbookOutput{}
	out.Body.Entries = []logEntry{}
	entries := []logEntry{}
	if in.Source == "mudlog" {
		if feed == nil {
			return nil, huma.Error503ServiceUnavailable("Mudlog observer unavailable")
		}
		ceiling := caller(ctx)
		if in.Level >= 0 && in.Level < ceiling {
			ceiling = in.Level
		}
		for _, e := range feed.Since(0, 0) {
			// Always apply the server-resolved ceiling before exposing or exporting data.
			if e.Level < 0 || e.Level > ceiling {
				continue
			}
			out.Body.Retained++
			if (in.Type >= 0 && e.Typ > in.Type) || (in.Key != "" && in.Key != e.Key) {
				continue
			}
			entries = append(entries, logEntry{Time: e.Time.UTC().Format(time.RFC3339Nano), Level: fmt.Sprint(e.Level), Type: fmt.Sprint(e.Typ), Message: e.Text, Raw: e.Text, Seq: e.Seq})
		}
		out.Body.LastSeq = feed.LastSeq()
		out.Body.Dropped = feed.Stats().Dropped
	} else if buf != nil {
		for _, raw := range buf.GetRecent(0) {
			parts := strings.SplitN(raw, " ", 3)
			e := logEntry{Type: "server", Message: raw, Raw: raw}
			if len(parts) == 3 {
				if _, err := time.Parse(time.RFC3339, parts[1]); err == nil {
					e.Level = parts[0]
					e.Time = parts[1]
					e.Message = parts[2]
				}
			}
			entries = append(entries, e)
		}
		out.Body.Retained = len(entries)
	}
	for _, e := range entries {
		if in.Severity != "" && e.Level != in.Severity {
			continue
		}
		if in.Search != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(in.Search)) {
			continue
		}
		if !after.IsZero() || !before.IsZero() {
			t, err := time.Parse(time.RFC3339Nano, e.Time)
			if err != nil || (!after.IsZero() && t.Before(after)) || (!before.IsZero() && t.After(before)) {
				continue
			}
		}
		out.Body.Entries = append(out.Body.Entries, e)
	}
	return out, nil
}

type logExportInput struct {
	LogbookInput
	Format string `query:"format" default:"log" enum:"log,csv"`
}
type logExportOutput struct {
	ContentType string `header:"Content-Type"`
	Disposition string `header:"Content-Disposition"`
	Cache       string `header:"Cache-Control"`
	Body        []byte
}

func csvCell(s string) string {
	if strings.HasPrefix(strings.TrimLeft(s, " \t\r\n"), "=") || strings.HasPrefix(strings.TrimLeft(s, " \t\r\n"), "+") || strings.HasPrefix(strings.TrimLeft(s, " \t\r\n"), "-") || strings.HasPrefix(strings.TrimLeft(s, " \t\r\n"), "@") {
		return "'" + s
	}
	return s
}

func registerLogbook(api huma.API, buf *LogBuffer, feed *mudlog.Feed, caller mudlogCallerLevel) {
	huma.Register(api, huma.Operation{OperationID: "get-logbook", Method: http.MethodGet, Path: "/admin/logbook", Summary: "Filtered retained server log or visible mudlog"}, func(ctx context.Context, in *LogbookInput) (*logbookOutput, error) {
		return logbookEntries(ctx, in, buf, feed, caller)
	})
	huma.Register(api, huma.Operation{OperationID: "export-logbook", Method: http.MethodGet, Path: "/admin/logbook/export", Summary: "Download currently filtered retained logs"}, func(ctx context.Context, in *logExportInput) (*logExportOutput, error) {
		result, err := logbookEntries(ctx, &in.LogbookInput, buf, feed, caller)
		if err != nil {
			return nil, err
		}
		var b bytes.Buffer
		typ := "text/plain; charset=utf-8"
		if in.Format == "csv" {
			typ = "text/csv; charset=utf-8"
			w := csv.NewWriter(&b)
			rows := [][]string{{"time", "level", "type", "message"}}
			for _, e := range result.Body.Entries {
				rows = append(rows, []string{csvCell(e.Time), csvCell(e.Level), csvCell(e.Type), csvCell(e.Message)})
			}
			if err := w.WriteAll(rows); err != nil {
				return nil, huma.Error500InternalServerError("CSV export failed", err)
			}
		} else {
			for _, e := range result.Body.Entries {
				b.WriteString(e.Raw)
				b.WriteByte('\n')
			}
		}
		return &logExportOutput{ContentType: typ, Disposition: fmt.Sprintf(`attachment; filename="darkpawns-%s.%s"`, in.Source, in.Format), Cache: "no-store", Body: b.Bytes()}, nil
	})
}
