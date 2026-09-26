package admin

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/game"
)

type worldSearchHit struct {
	Kind  string `json:"kind"`
	VNum  int    `json:"vnum"`
	Name  string `json:"name"`
	Zone  int    `json:"zone"`
	Score int    `json:"-"`
}

type worldSearchResponse struct {
	Total   int              `json:"total"`
	Results []worldSearchHit `json:"results"`
}

type worldSearchInput struct {
	Query string `query:"q" doc:"Search words or a VNUM across zones, rooms, mobs, objects and shops."`
}

type worldSearchOutput struct {
	Body worldSearchResponse
}

func registerWorldSearch(api huma.API, world *game.World) {
	huma.Register(api, huma.Operation{
		OperationID: "search-world",
		Method:      "GET",
		Path:        "/admin/search",
		Summary:     "Search world content",
		Description: "Search loaded zone, room, mob, object and shop definitions. Results reflect committed live world edits. Returns at most 100 matches, ranked by name and VNUM, with total match count.",
	}, func(ctx context.Context, in *worldSearchInput) (*worldSearchOutput, error) {
		return &worldSearchOutput{Body: searchWorld(world, in.Query)}, nil
	})
}

func searchWorld(world *game.World, rawQuery string) worldSearchResponse {
	query := strings.ToLower(strings.TrimSpace(rawQuery))
	response := worldSearchResponse{Results: []worldSearchHit{}}
	if len(query) < 2 {
		return response
	}
	terms := strings.Fields(query)
	add := func(kind string, vnum int, name string, zone int, fields ...string) {
		primary := strings.ToLower(name)
		text := strings.ToLower(strings.Join(append([]string{strconv.Itoa(vnum), name}, fields...), " "))
		for _, term := range terms {
			if !strings.Contains(text, term) {
				return
			}
		}
		score := 0
		if strconv.Itoa(vnum) == query || primary == query {
			score = 3
		} else if strings.Contains(primary, query) {
			score = 2
		} else if strings.Contains(text, query) {
			score = 1
		}
		response.Results = append(response.Results, worldSearchHit{Kind: kind, VNum: vnum, Name: name, Zone: zone, Score: score})
	}
	for _, zone := range world.GetAllZones() {
		add("zone", zone.Number, zone.Name, zone.Number)
	}
	rooms := world.SnapshotRooms()
	for index := range rooms {
		room := &rooms[index]
		add("room", room.VNum, room.Name, room.Zone, room.Description, room.ScriptName)
	}
	for _, mob := range world.GetAllMobPrototypes() {
		add("mob", mob.VNum, mob.ShortDesc, 0, mob.Keywords, mob.LongDesc, mob.DetailedDesc, mob.ScriptName)
	}
	for _, object := range world.GetAllObjPrototypes() {
		add("object", object.VNum, object.ShortDesc, 0, object.Keywords, object.LongDesc, object.ActionDesc, object.ScriptName)
	}
	for _, shop := range world.GetAllShops() {
		add("shop", shop.VNum, shop.KeeperName, 0, strconv.Itoa(shop.KeeperVNum), strconv.Itoa(shop.RoomVNum))
	}
	sort.Slice(response.Results, func(i, j int) bool {
		a, b := response.Results[i], response.Results[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.VNum < b.VNum
	})
	response.Total = len(response.Results)
	if len(response.Results) > 100 {
		response.Results = response.Results[:100]
	}
	return response
}
