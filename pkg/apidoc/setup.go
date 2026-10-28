package apidoc

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// SetupStatus exposes only whether the world still needs its first character.
// StartedEmpty describes this process's startup, not a change to its listeners.
type SetupStatus struct {
	FirstCharacterPending bool `json:"first_character_pending"`
	StartedEmpty          bool `json:"started_empty"`
}

type setupStatusOutput struct {
	Body         SetupStatus
	CacheControl string `header:"Cache-Control"`
}

func RegisterSetupStatus(api huma.API, countPlayers func() (int, error), startedEmpty bool) {
	huma.Register(api, huma.Operation{OperationID: "get-setup-status", Method: http.MethodGet, Path: "/setup-status", Summary: "First-character setup status"}, func(_ context.Context, _ *struct{}) (*setupStatusOutput, error) {
		count, err := countPlayers()
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("Could not check setup status. Try again shortly.")
		}
		return &setupStatusOutput{Body: SetupStatus{FirstCharacterPending: count == 0, StartedEmpty: startedEmpty}, CacheControl: "no-store"}, nil
	})
}
