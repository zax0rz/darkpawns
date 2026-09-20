package admin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

type olcZoneSaveInput struct {
	Zone int `path:"zone" doc:"Zone number to serialize."`
}

type olcZoneSaveBody struct {
	Zone  int  `json:"zone"`
	Saved bool `json:"saved"`
}

type olcZoneSaveOutput struct {
	Body olcZoneSaveBody
}

func registerOLCZoneSave(api huma.API, world *game.World, database *db.DB, saver OLCZoneSaveProvider) {
	gate := olcGate(world, database)
	huma.Register(api, huma.Operation{
		OperationID: "save-olc-zone",
		Method:      http.MethodPost,
		Path:        "/admin/olc/zones/{zone}/save",
		Summary:     "Save an OLC zone",
		Description: "Serializes the five zone-scoped OLC files under one save lock and clears the dirty entries written.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *olcZoneSaveInput) (*olcZoneSaveOutput, error) {
		if saver == nil {
			return nil, huma.NewError(http.StatusServiceUnavailable, "OLC save unavailable")
		}
		if err := saver.SaveOLCZone(in.Zone); err != nil {
			var conflict *olc.ZoneSaveConflict
			if errors.As(err, &conflict) {
				idle := time.Since(conflict.Entry.ClaimedAt)
				if idle < 0 || conflict.Entry.ClaimedAt.IsZero() {
					idle = 0
				}
				return nil, &olcConflictError{Message: "zone save refused: an editor holds a zone member", Holder: conflict.Entry.OwnerDisplayName, Frontend: string(conflict.Entry.OwnerFrontend), Idle: idle.Round(time.Second).String(), IdleSecs: int64(idle / time.Second)}
			}
			return nil, huma.NewError(http.StatusInternalServerError, err.Error())
		}
		return &olcZoneSaveOutput{Body: olcZoneSaveBody{Zone: in.Zone, Saved: true}}, nil
	})
}
