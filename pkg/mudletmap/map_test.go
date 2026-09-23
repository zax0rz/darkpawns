package mudletmap

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// fakeWorld is the slice of game.World the map reads.
type fakeWorld struct {
	rooms []parser.Room
	zones map[int]*parser.Zone
}

func (w *fakeWorld) Rooms() []parser.Room { return w.rooms }

func (w *fakeWorld) GetZone(number int) (*parser.Zone, bool) {
	zone, ok := w.zones[number]
	return zone, ok
}

func exits(pairs ...interface{}) map[string]parser.Exit {
	out := map[string]parser.Exit{}
	for i := 0; i < len(pairs); i += 2 {
		dir := pairs[i].(string)
		out[dir] = parser.Exit{Direction: dir, ToRoom: pairs[i+1].(int)}
	}
	return out
}

// keep is a small two-zone world: a hall with rooms in four directions, a
// loft above, a detached shrine, a room the layout must push aside, and an
// exit into the next zone. Zones 7 and 8 share a name.
func keep() *fakeWorld {
	door := exits("north", 102, "east", 103, "west", 201)
	d := door["north"]
	d.DoorState = 1
	door["north"] = d
	return &fakeWorld{
		zones: map[int]*parser.Zone{
			1: {Number: 1, Name: "The Keep"},
			2: {Number: 2, Name: "The Moor"},
			7: {Number: 7, Name: "New Zone"},
			8: {Number: 8, Name: "New Zone"},
		},
		rooms: []parser.Room{
			{VNum: 0, Name: "The Void", Zone: 1},
			{VNum: 101, Name: "Hall", Zone: 1, Sector: 0, Exits: door},
			{VNum: 102, Name: "North Hall", Zone: 1, Sector: 1, Exits: exits("south", 101, "east", 104)},
			{VNum: 103, Name: "East Hall", Zone: 1, Exits: exits("west", 101, "up", 105, "north", 106)},
			// 104 and 106 both want the square north-east of the hall.
			{VNum: 104, Name: "Corner A", Zone: 1, Exits: exits("west", 102)},
			{VNum: 106, Name: "Corner B", Zone: 1, Exits: exits("south", 103)},
			{VNum: 105, Name: "Loft", Zone: 1, Sector: 99, Exits: exits("down", 103)},
			{VNum: 110, Name: "Shrine", Zone: 1},
			{VNum: 201, Name: "Moor Edge", Zone: 2, Exits: exits("east", 101, "north", 999)},
			{VNum: 701, Name: "Stub A", Zone: 7},
			{VNum: 801, Name: "Stub B", Zone: 8},
		},
	}
}

type parsedMap struct {
	Areas []xmlArea `xml:"areas>area"`
	Rooms []xmlRoom `xml:"rooms>room"`
}

func generate(t *testing.T, world World) (parsedMap, []byte) {
	t.Helper()
	data, err := Generate(world)
	if err != nil {
		t.Fatal(err)
	}
	var doc parsedMap
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("map is not valid XML: %v", err)
	}
	return doc, data
}

func TestGenerateLaysOutAZone(t *testing.T) {
	doc, _ := generate(t, keep())
	rooms := map[int]xmlRoom{}
	for _, room := range doc.Rooms {
		rooms[room.ID] = room
	}
	if _, ok := rooms[0]; ok {
		t.Fatal("vnum 0 is not a valid Mudlet room id and must be left out")
	}
	at := func(id int) xmlCoord { return rooms[id].Coord }
	want := map[int]xmlCoord{
		101: {0, 0, 0},
		102: {0, 1, 0},
		103: {1, 0, 0},
		104: {1, 1, 0}, // placed first, from 102
		106: {1, 2, 0}, // wanted 104's square; pushed on north
		105: {1, 0, 1},
		110: {4, 0, 0}, // unconnected: set beside the rest
	}
	for id, c := range want {
		if at(id) != c {
			t.Errorf("room %d at %+v, want %+v", id, at(id), c)
		}
	}
	if rooms[101].Area != 2 || rooms[201].Area != 3 {
		t.Fatalf("zone n must be area n+1: 101 in %d, 201 in %d", rooms[101].Area, rooms[201].Area)
	}
	if rooms[101].Environment != EnvironmentBase || rooms[102].Environment != EnvironmentBase+1 || rooms[105].Environment != EnvironmentBase+len(EnvironmentNames) {
		t.Fatalf("environments: %d %d %d", rooms[101].Environment, rooms[102].Environment, rooms[105].Environment)
	}
	var sawDoor, sawCrossZone bool
	for _, exit := range rooms[101].Exits {
		if exit.Direction == "north" && exit.Door == 1 {
			sawDoor = true
		}
		if exit.Direction == "west" && exit.Target == 201 {
			sawCrossZone = true
		}
	}
	if !sawDoor || !sawCrossZone {
		t.Fatalf("hall exits = %+v", rooms[101].Exits)
	}
	for _, exit := range rooms[201].Exits {
		if exit.Target == 999 {
			t.Fatal("an exit to a room that doesn't exist was kept")
		}
	}
}

func TestAreaNamesKeepZonesApart(t *testing.T) {
	names := AreaNames(keep())
	if names[1] != "The Keep" || names[7] != "New Zone (zone 7)" || names[8] != "New Zone (zone 8)" {
		t.Fatalf("area names = %v", names)
	}
	doc, _ := generate(t, keep())
	seen := map[string]bool{}
	for _, area := range doc.Areas {
		if seen[area.Name] {
			t.Fatalf("two areas are both named %q", area.Name)
		}
		seen[area.Name] = true
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	_, first := generate(t, keep())
	_, second := generate(t, keep())
	if !bytes.Equal(first, second) || Version(first) != Version(second) {
		t.Fatal("the same world produced two different maps")
	}
	world := keep()
	world.rooms[1].Name = "Great Hall"
	_, renamed := generate(t, world)
	if Version(renamed) == Version(first) {
		t.Fatal("renaming a room did not change the map version")
	}
}

// TestGenerateWholeWorld maps the real world: every room is present, and no
// two rooms of one area share a square, so the map never draws one room on
// top of another.
func TestGenerateWholeWorld(t *testing.T) {
	dir, err := filepath.Abs("../../lib/world")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseWorld(dir)
	if err != nil {
		t.Fatal(err)
	}
	zones := map[int]*parser.Zone{}
	for i := range parsed.Zones {
		zones[parsed.Zones[i].Number] = &parsed.Zones[i]
	}
	world := &fakeWorld{rooms: parsed.Rooms, zones: zones}
	start := time.Now()
	doc, data := generate(t, world)
	elapsed := time.Since(start)

	wantRooms := 0
	for _, room := range parsed.Rooms {
		if room.VNum > 0 {
			wantRooms++
		}
	}
	if len(doc.Rooms) != wantRooms {
		t.Fatalf("map has %d rooms, world has %d", len(doc.Rooms), wantRooms)
	}
	squares := map[[4]int]int{}
	for _, room := range doc.Rooms {
		key := [4]int{room.Area, room.Coord.X, room.Coord.Y, room.Coord.Z}
		if other, taken := squares[key]; taken {
			t.Fatalf("rooms %d and %d share %v", other, room.ID, key)
		}
		squares[key] = room.ID
	}
	t.Logf("%d rooms in %d areas, %d KB, generated in %v", len(doc.Rooms), len(doc.Areas), len(data)/1024, elapsed)
}

func TestCacheServesAndRevalidates(t *testing.T) {
	world := keep()
	cache := NewCache(world, time.Hour)
	now := time.Unix(1000, 0)
	cache.now = func() time.Time { return now }

	get := func(etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/darkpawns-map.xml", nil)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		rec := httptest.NewRecorder()
		cache.ServeHTTP(rec, req)
		return rec
	}
	first := get("")
	if first.Code != http.StatusOK || first.Header().Get("Content-Type") != "application/xml; charset=utf-8" {
		t.Fatalf("GET = %d %q", first.Code, first.Header().Get("Content-Type"))
	}
	etag := first.Header().Get("ETag")
	if etag == "" || get(etag).Code != http.StatusNotModified {
		t.Fatal("a client with the current map should get 304")
	}

	// A world change is served once the cache's lifetime has passed.
	world.rooms[1].Name = "Great Hall"
	if get(etag).Code != http.StatusNotModified {
		t.Fatal("the map changed before its cache lifetime ran out")
	}
	now = now.Add(2 * time.Hour)
	if changed := get(etag); changed.Code != http.StatusOK || changed.Header().Get("ETag") == etag {
		t.Fatal("an edited world was not served after the cache lifetime")
	}

	post := httptest.NewRecorder()
	cache.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/darkpawns-map.xml", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d", post.Code)
	}
}
