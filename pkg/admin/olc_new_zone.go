package admin

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

type newZoneInput struct {
	Zone string `path:"zone" doc:"New zone number."`
}

type newZoneBody struct {
	Zone      int    `json:"zone"`
	Name      string `json:"name"`
	TopRoom   int    `json:"top_room"`
	Lifespan  int    `json:"lifespan"`
	ResetMode int    `json:"reset_mode"`
	Created   bool   `json:"created"`
}

type newZoneOutput struct {
	Body newZoneBody
}

func registerOLCZoneCreation(api huma.API, world *game.World, database *db.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "create-olc-zone",
		Method:      http.MethodPost,
		Path:        "/admin/olc/zones/{zone}",
		Summary:     "Create a new OLC zone",
		Description: "Creates the C-compatible zone, room, mobile, object, and shop files for a new zone. Requires HIGOD.",
		Middlewares: huma.Middlewares{olcNewZoneGate(world, database)},
	}, func(ctx context.Context, in *newZoneInput) (*newZoneOutput, error) {
		number, err := strconv.Atoi(in.Zone)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid zone")
		}
		if number < 0 {
			return nil, huma.NewError(http.StatusBadRequest, "invalid zone")
		}
		if number > olc.NewZoneMax {
			return nil, huma.NewError(http.StatusBadRequest, "326 is the highest zone allowed.")
		}
		start := number * 100
		for _, zone := range world.GetAllZones() {
			if zone.Number*100 <= start && zone.TopRoom >= start {
				return nil, huma.NewError(http.StatusConflict, "A zone already covers that area.")
			}
		}
		if err := olc.WriteNewZoneFiles(world.WorldPath, number); err != nil {
			slog.Error("web OLC new-zone file creation failed", "zone", number, "error", err)
			return nil, huma.NewError(http.StatusInternalServerError, "could not create zone files")
		}
		zone, created := world.CreateZone(number)
		if !created {
			return nil, huma.NewError(http.StatusConflict, "A zone already covers that area.")
		}
		for _, extension := range []string{"zon", "wld", "mob", "obj", "shp"} {
			if err := olc.UpdateNewZoneIndex(world.WorldPath, number, extension); err != nil {
				slog.Error("web OLC new-zone index update failed", "zone", number, "type", extension, "error", err)
			}
		}
		return &newZoneOutput{Body: newZoneBody{
			Zone:      zone.Number,
			Name:      zone.Name,
			TopRoom:   zone.TopRoom,
			Lifespan:  zone.Lifespan,
			ResetMode: zone.ResetMode,
			Created:   true,
		}}, nil
	})
}

func olcNewZoneGate(world *game.World, database *db.DB) func(huma.Context, func(huma.Context)) {
	base := olcVocabularyGate(world, database)
	return func(ctx huma.Context, next func(huma.Context)) {
		base(ctx, func(ctx huma.Context) {
			claims, ok := auth.GetClaimsFromContext(ctx.Context())
			if !ok || claims == nil || database == nil {
				writeOLCError(ctx, http.StatusUnauthorized, olcMiddlewareError{Error: "unauthorized"})
				return
			}
			record, err := database.GetPlayer(claims.PlayerName)
			if err != nil || record == nil {
				writeOLCError(ctx, http.StatusUnauthorized, olcMiddlewareError{Error: "unauthorized"})
				return
			}
			if record.Level < olc.NewZoneRequiredLevel {
				writeOLCError(ctx, http.StatusForbidden, olcMiddlewareError{
					Error:    "action requires a higher level",
					Required: fmt.Sprintf("level %d (%s)", olc.NewZoneRequiredLevel, olc.NewZoneRequiredLabel),
				})
				return
			}
			next(ctx)
		})
	}
}
