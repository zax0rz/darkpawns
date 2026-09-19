package admin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// This file holds the tranche-1 Huma operations: the routes migrated off the
// plain admin mux onto generated, spec-documented operations. The router
// mounts the Huma API's mux behind the same rate-limit/CORS/role chain the
// hand-rolled handlers used, so the middleware composition is unchanged.

// liveAgentSessionsOutput is the JSON array returned by GET
// /admin/sessions/agents. It reuses LiveAgentSession so the wire shape is
// identical to the handler it replaces; a nil slice marshals to `null`, as
// before.
type liveAgentSessionsOutput struct {
	Body []LiveAgentSession
}

// registerLiveAgentSessions registers GET /admin/sessions/agents on api.
func registerLiveAgentSessions(api huma.API, provider LiveSessionProvider) {
	huma.Register(api, huma.Operation{
		OperationID: "list-live-agent-sessions",
		Method:      http.MethodGet,
		Path:        "/admin/sessions/agents",
		Summary:     "List connected game agent sessions",
		Description: "Live view of the AI agents currently connected to the game server, for the admin console.",
	}, func(ctx context.Context, in *struct{}) (*liveAgentSessionsOutput, error) {
		return &liveAgentSessionsOutput{Body: provider.GetLiveAgentSessions()}, nil
	})
}

// researchCaptureInput is the POST body for /admin/research/capture. Enabled
// is a pointer so that "field absent" is distinguishable from false; the
// handler rejects absent explicitly, matching the old handler's contract
// (a body that does not say which way is a 400, not a silent false). The
// omitempty keeps the property optional in the generated schema — required is
// enforced by the handler so the failure stays a 400, as it was on the plain
// mux, rather than a schema-validation 422.
type researchCaptureInput struct {
	Body struct {
		Enabled *bool `json:"enabled,omitempty" description:"true starts recording decision capture, false stops and flushes it"`
	}
}

// researchCaptureOutput reuses researchCaptureResponse so the wire shape —
// including records' omitempty — is identical to the old handler.
type researchCaptureOutput struct {
	Body researchCaptureResponse
}

// registerResearchCapture registers GET and POST /admin/research/capture on
// api. GET reports the capture state; POST toggles it. The audit logging and
// the response wording ("tells and says") carry over from the old handler
// unchanged.
func registerResearchCapture(api huma.API, provider LiveSessionProvider, auditLogger *audit.AuditLogger) {
	current := func() researchCaptureOutput {
		out := researchCaptureOutput{}
		out.Body.Enabled = provider.DecisionCaptureEnabled()
		out.Body.Available = provider.DecisionCaptureAvailable()
		if out.Body.Available {
			out.Body.Records = researchCaptureRecords
		}
		return out
	}

	huma.Register(api, huma.Operation{
		OperationID: "get-decision-capture",
		Method:      http.MethodGet,
		Path:        "/admin/research/capture",
		Summary:     "Get decision capture state",
		Description: "Reports whether decision capture is enabled and whether a research store is configured.",
	}, func(ctx context.Context, in *struct{}) (*researchCaptureOutput, error) {
		out := current()
		return &out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "set-decision-capture",
		Method:      http.MethodPost,
		Path:        "/admin/research/capture",
		Summary:     "Enable or disable decision capture",
		Description: "Starts or stops recording of command text including tells and says. Enabling without a configured research store fails with 409.",
	}, func(ctx context.Context, in *researchCaptureInput) (*researchCaptureOutput, error) {
		if in.Body.Enabled == nil {
			return nil, huma.Error400BadRequest(`body must be {"enabled": true|false}`)
		}
		if *in.Body.Enabled {
			if !provider.EnableDecisionCapture() {
				return nil, huma.NewError(http.StatusConflict,
					"no research store configured; set DP_RESEARCH_URL and restart")
			}
		} else {
			// Flushes what is buffered; see Manager.DisableDecisionCapture.
			provider.DisableDecisionCapture()
		}
		if auditLogger != nil {
			auditLogger.Log(audit.AuditEvent{
				EventType: "research",
				Action:    "decision_capture",
				Success:   true,
				Details:   fmt.Sprintf("enabled=%t records=%q", *in.Body.Enabled, researchCaptureRecords),
			})
		}
		slog.Info("decision capture toggled", "enabled", *in.Body.Enabled, "records", researchCaptureRecords)
		out := current()
		return &out, nil
	})
}

// ---------------------------------------------------------------------------
// Tranche 2: the builder world reads — eleven GET-only operations migrated
// off the plain admin mux onto typed Huma operations.
//
// Every success body is built by the same builder the legacy handler uses —
// the legacy constructors in handlers.go remain as the behavioral referees
// the existing tests exercise, and they share these builders with the
// operations. apidoc's JSON format pins the encoder to json.NewEncoder's
// defaults, so success bodies are byte-identical to the plain-mux handlers,
// escaping and trailing newline included. Error paths return
// *apidoc.PlainError, which reproduces http.Error's exact wire shape
// (text/plain, nosniff, raw body + newline) in place of Huma's problem+json,
// so the 400/404 bodies the prefix-stripping handlers emitted stay
// byte-identical as well.
//
// The {vnum} path params are typed as strings on purpose: the handlers they
// replace answered a non-numeric vnum with a 400 "invalid vnum" body, and
// Huma's own int-param validation would instead reject the request before the
// handler ran, as a 422. Parsing in the handler keeps the 400 bytes.
// ---------------------------------------------------------------------------

// zoneList builds the response for GET /admin/zones.
func zoneList(world *game.World) []zoneResponse {
	zones := world.GetAllZones()
	result := make([]zoneResponse, 0, len(zones))
	for _, z := range zones {
		result = append(result, zoneResponse{
			Number:    z.Number,
			Name:      z.Name,
			TopRoom:   z.TopRoom,
			Lifespan:  z.Lifespan,
			ResetMode: z.ResetMode,
		})
	}
	return result
}

type zonesOutput struct {
	Body []zoneResponse
}

// registerZones registers GET /admin/zones on api.
func registerZones(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "list-zones",
		Method:      http.MethodGet,
		Path:        "/admin/zones",
		Summary:     "List world zones",
		Description: "All zones in the loaded world: number, name, top room, lifespan and reset mode.",
	}, func(ctx context.Context, in *struct{}) (*zonesOutput, error) {
		return &zonesOutput{Body: zoneList(world)}, nil
	})
}

// serverInfo builds the response for GET /admin/server.
func serverInfo(world *game.World) serverInfoResponse {
	return serverInfoResponse{
		Uptime:      time.Since(processStartTime).Round(time.Second).String(),
		RoomCount:   world.GetRoomCount(),
		PlayerCount: world.GetPlayerCount(),
		ZoneCount:   len(world.GetAllZones()),
	}
}

type serverInfoOutput struct {
	Body serverInfoResponse
}

// clientIPKey carries the client IP from the withClientIP mount wrapper (the
// op handler has no *http.Request to run auth.GetIPFromRequest against) into
// the audit log, which the handler this operation replaces always populated.
type clientIPKey struct{}

// withClientIP records auth.GetIPFromRequest(r) on the request context so the
// server-info operation can audit it exactly as the old handler did.
func withClientIP(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), clientIPKey{}, auth.GetIPFromRequest(r)))
		next(w, r)
	}
}

// clientIPFrom returns the IP withClientIP stashed, or "" outside that mount.
func clientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

// registerServerInfo registers GET /admin/server on api. The admin-access
// audit event carries over from the old handler, client IP included.
func registerServerInfo(api huma.API, world *game.World, auditLogger *audit.AuditLogger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-server-info",
		Method:      http.MethodGet,
		Path:        "/admin/server",
		Summary:     "Server status",
		Description: "Uptime and world size: room, player and zone counts. Each request is written to the admin audit log.",
	}, func(ctx context.Context, in *struct{}) (*serverInfoOutput, error) {
		if auditLogger != nil {
			playerName := ""
			if claims, ok := auth.GetClaimsFromContext(ctx); ok {
				playerName = claims.PlayerName
			}
			auditLogger.Log(audit.AuditEvent{
				IPAddress: clientIPFrom(ctx),
				EventType: "administration",
				User:      playerName,
				Action:    "admin_server_info",
				Details:   "viewed server info",
				Success:   true,
			})
		}
		return &serverInfoOutput{Body: serverInfo(world)}, nil
	})
}

// logLineCount applies the ?lines= semantics the plain-mux handler had: a
// missing, unparsable or non-positive value falls back to the default of
// 100. (That is a silent fallback, not a 400 — the pre-migration behavior the
// existing tests pin.)
func logLineCount(q string) int {
	n := 100
	if q != "" {
		if parsed, err := strconv.Atoi(q); err == nil && parsed > 0 {
			n = parsed
		}
	}
	return n
}

type logsInput struct {
	Lines string `query:"lines" doc:"Number of recent log lines to return. Values that are not positive integers are ignored and the default of 100 is used."`
}

type logsOutput struct {
	Body []string
}

// registerLogs registers GET /admin/logs on api.
func registerLogs(api huma.API, logBuffer *LogBuffer) {
	huma.Register(api, huma.Operation{
		OperationID: "list-log-lines",
		Method:      http.MethodGet,
		Path:        "/admin/logs",
		Summary:     "Recent server log lines",
		Description: "The tail of the in-memory log ring buffer. Defaults to the last 100 lines.",
	}, func(ctx context.Context, in *logsInput) (*logsOutput, error) {
		return &logsOutput{Body: logBuffer.GetRecent(logLineCount(in.Lines))}, nil
	})
}

// playerList builds the response for GET /admin/players.
func playerList(world *game.World) []playerResponse {
	players := world.GetAllPlayers()
	result := make([]playerResponse, 0, len(players))
	for _, p := range players {
		result = append(result, playerResponse{
			Name:  p.Name,
			Level: p.Level,
			Room:  p.RoomVNum,
		})
	}
	return result
}

type playersOutput struct {
	Body []playerResponse
}

// registerPlayers registers GET /admin/players on api.
func registerPlayers(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "list-online-players",
		Method:      http.MethodGet,
		Path:        "/admin/players",
		Summary:     "List online players",
		Description: "Every connected player with level and current room.",
	}, func(ctx context.Context, in *struct{}) (*playersOutput, error) {
		return &playersOutput{Body: playerList(world)}, nil
	})
}

// mobView builds one mob response; the list and by-vnum operations share it.
func mobView(m *parser.Mob) mobResponse {
	return mobResponse{
		VNum:        m.VNum,
		Keywords:    m.Keywords,
		ShortDesc:   m.ShortDesc,
		LongDesc:    m.LongDesc,
		Level:       m.Level,
		Alignment:   m.Alignment,
		AC:          m.AC,
		HP:          m.HP.String(),
		Gold:        m.Gold,
		Exp:         m.Exp,
		Position:    m.Position,
		DefaultPos:  m.DefaultPos,
		Sex:         m.Sex,
		Race:        m.Race,
		ActionFlags: m.ActionFlags,
		AffectFlags: m.AffectFlags,
		ScriptName:  m.ScriptName,
		Str:         m.Str,
		Int:         m.Int,
		Wis:         m.Wis,
		Dex:         m.Dex,
		Con:         m.Con,
		Cha:         m.Cha,
	}
}

func mobList(world *game.World) []mobResponse {
	mobs := world.GetAllMobPrototypes()
	result := make([]mobResponse, 0, len(mobs))
	for _, m := range mobs {
		result = append(result, mobView(m))
	}
	return result
}

type mobsOutput struct {
	Body []mobResponse
}

type mobVnumInput struct {
	VNum string `path:"vnum" doc:"Mob prototype vnum. A non-numeric vnum returns 400; an unknown vnum returns 404."`
}

type mobOutput struct {
	Body mobResponse
}

// registerMobs registers GET /admin/mobs and GET /admin/mobs/{vnum} on api.
func registerMobs(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "list-mob-prototypes",
		Method:      http.MethodGet,
		Path:        "/admin/mobs",
		Summary:     "List mob prototypes",
		Description: "All mob prototypes in the loaded world.",
	}, func(ctx context.Context, in *struct{}) (*mobsOutput, error) {
		return &mobsOutput{Body: mobList(world)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-mob-prototype",
		Method:      http.MethodGet,
		Path:        "/admin/mobs/{vnum}",
		Summary:     "Get one mob prototype",
		Description: "A single mob prototype by vnum.",
	}, func(ctx context.Context, in *mobVnumInput) (*mobOutput, error) {
		vnum, err := strconv.Atoi(in.VNum)
		if err != nil {
			return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid vnum"}`)
		}
		mob, ok := world.GetMobPrototype(vnum)
		if !ok {
			return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"mob not found"}`)
		}
		return &mobOutput{Body: mobView(mob)}, nil
	})
}

// objView builds one object response; the list and by-vnum operations share
// it.
func objView(o *parser.Obj) objResponse {
	return objResponse{
		VNum:       o.VNum,
		Keywords:   o.Keywords,
		ShortDesc:  o.ShortDesc,
		LongDesc:   o.LongDesc,
		TypeFlag:   o.TypeFlag,
		Weight:     o.Weight,
		Cost:       o.Cost,
		ExtraFlags: o.ExtraFlags,
		WearFlags:  o.WearFlags,
		Values:     o.Values,
		ScriptName: o.ScriptName,
	}
}

func objList(world *game.World) []objResponse {
	objs := world.GetAllObjPrototypes()
	result := make([]objResponse, 0, len(objs))
	for _, o := range objs {
		result = append(result, objView(o))
	}
	return result
}

type objsOutput struct {
	Body []objResponse
}

type objVnumInput struct {
	VNum string `path:"vnum" doc:"Object prototype vnum. A non-numeric vnum returns 400; an unknown vnum returns 404."`
}

type objOutput struct {
	Body objResponse
}

// registerObjects registers GET /admin/objects and GET /admin/objects/{vnum}
// on api.
func registerObjects(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "list-object-prototypes",
		Method:      http.MethodGet,
		Path:        "/admin/objects",
		Summary:     "List object prototypes",
		Description: "All object prototypes in the loaded world.",
	}, func(ctx context.Context, in *struct{}) (*objsOutput, error) {
		return &objsOutput{Body: objList(world)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-object-prototype",
		Method:      http.MethodGet,
		Path:        "/admin/objects/{vnum}",
		Summary:     "Get one object prototype",
		Description: "A single object prototype by vnum.",
	}, func(ctx context.Context, in *objVnumInput) (*objOutput, error) {
		vnum, err := strconv.Atoi(in.VNum)
		if err != nil {
			return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid vnum"}`)
		}
		obj, ok := world.GetObjPrototype(vnum)
		if !ok {
			return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"object not found"}`)
		}
		return &objOutput{Body: objView(obj)}, nil
	})
}

func shopList(world *game.World) []shopResponse {
	shops := world.GetAllShops()
	result := make([]shopResponse, 0, len(shops))
	for _, s := range shops {
		result = append(result, shopResponse{
			KeeperVNum: s.KeeperVNum,
			BuyTypes:   s.BuyTypes,
			SellTypes:  s.SellTypes,
			ProfitBuy:  s.ProfitBuy,
			ProfitSell: s.ProfitSell,
			KeeperName: s.KeeperName,
			RoomVNum:   s.RoomVNum,
		})
	}
	return result
}

type shopsOutput struct {
	Body []shopResponse
}

// registerShops registers GET /admin/shops on api. The shop-by-keeper GET
// remains on the plain mux while SEDIT owns shop writes through the game
// command path.
func registerShops(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "list-shops",
		Method:      http.MethodGet,
		Path:        "/admin/shops",
		Summary:     "List shops",
		Description: "All shops in the loaded world with keeper, buy/sell types and profit margins.",
	}, func(ctx context.Context, in *struct{}) (*shopsOutput, error) {
		return &shopsOutput{Body: shopList(world)}, nil
	})
}

type roomVnumInput struct {
	VNum string `path:"vnum" doc:"Room vnum. A non-numeric vnum returns 400; an unknown vnum returns 404."`
}

type roomOutput struct {
	Body roomResponse
}

// registerRooms registers GET /admin/rooms/{vnum} on api.
func registerRooms(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "get-room",
		Method:      http.MethodGet,
		Path:        "/admin/rooms/{vnum}",
		Summary:     "Get one room",
		Description: "A single room by vnum: name, description, zone, sector and flags.",
	}, func(ctx context.Context, in *roomVnumInput) (*roomOutput, error) {
		vnum, err := strconv.Atoi(in.VNum)
		if err != nil {
			return nil, apidoc.NewPlainError(http.StatusBadRequest, `{"error":"invalid vnum"}`)
		}
		room := world.GetRoomInWorld(vnum)
		if room == nil {
			return nil, apidoc.NewPlainError(http.StatusNotFound, `{"error":"room not found"}`)
		}
		return &roomOutput{Body: roomResponse{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Zone:        room.Zone,
			Sector:      room.Sector,
			Flags:       room.Flags,
		}}, nil
	})
}

// metricsSnapshot builds the response for GET /admin/metrics.
func metricsSnapshot(world *game.World) metricsResponse {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return metricsResponse{
		MemoryAlloc:  memStats.Alloc,
		MemorySys:    memStats.Sys,
		MemoryHeap:   memStats.HeapInuse,
		Goroutines:   runtime.NumGoroutine(),
		GCCycles:     memStats.NumGC,
		LastGC:       time.Unix(0, int64(memStats.LastGC)).Format(time.RFC3339Nano), // #nosec G115 -- runtime GC nanosecond timestamp cannot exceed int64 before year 2262
		PauseTotalNs: memStats.PauseTotalNs,
		Uptime:       time.Since(processStartTime).Round(time.Second).String(),
		PlayerCount:  world.GetPlayerCount(),
		RoomCount:    world.GetRoomCount(),
		ZoneCount:    len(world.GetAllZones()),
	}
}

type metricsOutput struct {
	Body metricsResponse
}

// registerMetrics registers GET /admin/metrics on api.
func registerMetrics(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "get-server-metrics",
		Method:      http.MethodGet,
		Path:        "/admin/metrics",
		Summary:     "Runtime metrics",
		Description: "Go runtime memory and GC statistics plus world size and uptime.",
	}, func(ctx context.Context, in *struct{}) (*metricsOutput, error) {
		return &metricsOutput{Body: metricsSnapshot(world)}, nil
	})
}
