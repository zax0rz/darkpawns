package admin

import (
	"context"
	"net/http"
	"sort"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
)

type olcVNumRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type olcVNumKindMap struct {
	Kind      string         `json:"kind"`
	Start     int            `json:"start"`
	End       int            `json:"end"`
	Used      []int          `json:"used"`
	Free      []olcVNumRange `json:"free"`
	Suggested int            `json:"suggested"`
}

type olcVNumMap struct {
	Zone  int              `json:"zone"`
	Start int              `json:"start"`
	End   int              `json:"end"`
	Kinds []olcVNumKindMap `json:"kinds"`
}

type olcVNumMapInput struct {
	Zone string `path:"zone" doc:"Zone number."`
}

type olcVNumMapOutput struct {
	Body olcVNumMap
}

type olcVNumLookupInput struct {
	Kind string `path:"kind" doc:"OLC kind: room, mob, obj, or shop."`
	VNum string `path:"vnum" doc:"VNUM to check."`
}

type olcVNumLookup struct {
	Kind       string `json:"kind"`
	VNum       int    `json:"vnum"`
	Exists     bool   `json:"exists"`
	Name       string `json:"name"`
	ZoneNumber int    `json:"zone_number"`
}

type olcVNumLookupOutput struct {
	Body olcVNumLookup
}

// registerOLCReadGaps adds the two read-only workshop lookups. Neither
// operation opens a draft, renews a lease, claims a resource, or marks a
// zone dirty; they are deliberately behind the builder-level vocabulary gate.
func registerOLCReadGaps(api huma.API, world *game.World, database *db.DB) {
	vocabularyGate := olcVocabularyGate(world, database)

	huma.Register(api, huma.Operation{
		OperationID: "get-olc-vnum-map",
		Method:      http.MethodGet,
		Path:        "/admin/olc/zones/{zone}/vnums",
		Summary:     "Get a zone VNUM map",
		Description: "Returns used and free VNUM ranges for rooms, mobiles, objects, and shops without claiming any resource.",
		Middlewares: huma.Middlewares{olcGate(world, database)},
	}, func(ctx context.Context, in *olcVNumMapInput) (*olcVNumMapOutput, error) {
		zoneNumber, err := strconv.Atoi(in.Zone)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid zone")
		}
		zone, ok := world.SnapshotZone(zoneNumber)
		if !ok {
			return nil, huma.NewError(http.StatusNotFound, "zone not found")
		}
		start, end := zone.Number*100, zone.TopRoom
		if end < start {
			end = start
		}
		kinds := make([]olcVNumKindMap, 0, 4)
		for _, kind := range []string{"room", "mob", "obj", "shop"} {
			used := usedVNums(world, kind, start, end)
			free := freeVNumRanges(start, end, used)
			suggested := 0
			if len(free) > 0 {
				suggested = free[0].Start
			}
			kinds = append(kinds, olcVNumKindMap{
				Kind:      kind,
				Start:     start,
				End:       end,
				Used:      used,
				Free:      free,
				Suggested: suggested,
			})
		}
		return &olcVNumMapOutput{Body: olcVNumMap{Zone: zone.Number, Start: start, End: end, Kinds: kinds}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "lookup-olc-vnum",
		Method:      http.MethodGet,
		Path:        "/admin/olc/lookup/{kind}/{vnum}/name",
		Summary:     "Look up an OLC VNUM",
		Description: "Returns existence and a display name for one VNUM without opening a draft or taking a claim.",
		Middlewares: huma.Middlewares{vocabularyGate},
	}, func(ctx context.Context, in *olcVNumLookupInput) (*olcVNumLookupOutput, error) {
		vnum, err := strconv.Atoi(in.VNum)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid vnum")
		}
		kind, ok := olcKindForPath(in.Kind)
		if !ok || kind == olc.KindZone {
			return nil, huma.NewError(http.StatusBadRequest, "unknown OLC lookup kind")
		}
		lookup := olcVNumLookup{Kind: in.Kind, VNum: vnum}
		if zone, ok := entityZone(world, kind, vnum); ok {
			lookup.ZoneNumber = zone.Number
		}
		switch kind {
		case olc.KindRoom:
			if room, exists := world.SnapshotRoom(vnum); exists {
				lookup.Exists = true
				lookup.Name = room.Name
			}
		case olc.KindMob:
			if mob, exists := world.SnapshotMob(vnum); exists {
				lookup.Exists = true
				lookup.Name = mob.ShortDesc
			}
		case olc.KindObject:
			if object, exists := world.SnapshotObj(vnum); exists {
				lookup.Exists = true
				lookup.Name = object.ShortDesc
			}
		case olc.KindShop:
			if _, exists := world.SnapshotShop(vnum); exists {
				lookup.Exists = true
				lookup.Name = "Shop #" + strconv.Itoa(vnum)
			}
		}
		return &olcVNumLookupOutput{Body: lookup}, nil
	})
}

func usedVNums(world *game.World, kind string, start, end int) []int {
	used := make([]int, 0)
	switch kind {
	case "room":
		rooms := world.SnapshotRooms()
		for index := range rooms {
			room := &rooms[index]
			if room.VNum >= start && room.VNum <= end {
				used = append(used, room.VNum)
			}
		}
	case "mob":
		mobs := world.SnapshotMobs()
		for index := range mobs {
			mob := &mobs[index]
			if mob.VNum >= start && mob.VNum <= end {
				used = append(used, mob.VNum)
			}
		}
	case "obj":
		objects := world.SnapshotObjs()
		for index := range objects {
			object := &objects[index]
			if object.VNum >= start && object.VNum <= end {
				used = append(used, object.VNum)
			}
		}
	case "shop":
		shops := world.SnapshotShops()
		for index := range shops {
			shop := &shops[index]
			if shop.VNum >= start && shop.VNum <= end {
				used = append(used, shop.VNum)
			}
		}
	}
	sort.Ints(used)
	return uniqueInts(used)
}

func uniqueInts(values []int) []int {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func freeVNumRanges(start, end int, used []int) []olcVNumRange {
	free := make([]olcVNumRange, 0)
	cursor := start
	for _, value := range used {
		if value > cursor {
			free = append(free, olcVNumRange{Start: cursor, End: value - 1})
		}
		if value >= cursor {
			cursor = value + 1
		}
	}
	if cursor <= end {
		free = append(free, olcVNumRange{Start: cursor, End: end})
	}
	return free
}
