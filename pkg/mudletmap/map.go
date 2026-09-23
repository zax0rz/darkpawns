// Package mudletmap renders the game world as a Mudlet XML map, the file
// Mudlet imports with loadMap("….xml") and fetches through GMCP Client.Map.
//
// The map is the whole world, laid out on Mudlet's grid: each zone is a Mudlet
// area, rooms are keyed by vnum (the same number GMCP Room.Info sends, so the
// live mapper and the downloaded map agree), and each zone is placed by
// walking its exits from its lowest-numbered room. Dark Pawns has no
// coordinates of its own, so the layout is a projection: where a zone's exits
// don't fit a grid (mazes, one-way loops) a room is pushed further along its
// direction rather than stacked on another.
package mudletmap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Directions in C's dirs[] order, with the grid step each one takes and the
// word Mudlet's XML importer expects.
var directions = []struct {
	key     string
	word    string
	x, y, z int
}{
	{"north", "north", 0, 1, 0},
	{"east", "east", 1, 0, 0},
	{"south", "south", 0, -1, 0},
	{"west", "west", -1, 0, 0},
	{"up", "up", 0, 0, 1},
	{"down", "down", 0, 0, -1},
}

// EnvironmentBase is the first Mudlet environment id the package colours.
// Sector n maps to EnvironmentBase+n; sectors without a name in
// EnvironmentNames map to the Unknown id. The package's mapper uses the same
// ids, so rooms from the download and rooms mapped live look the same.
const EnvironmentBase = 257

// EnvironmentNames are the sector names C's look_at_room prints with
// PRF_ROOMFLAGS (act.informative.c); GMCP Room.Info sends the same strings.
var EnvironmentNames = []string{
	"Inside", "City", "Field", "Forest", "Hills", "Mountain", "Water Swim",
	"Water Noswim", "Underwater", "Flying", "Desert", "Fire", "Earth", "Wind",
	"Water",
}

// Environment returns the sector's name and Mudlet environment id.
func Environment(sector int) (string, int) {
	if sector >= 0 && sector < len(EnvironmentNames) {
		return EnvironmentNames[sector], EnvironmentBase + sector
	}
	return "Unknown", EnvironmentBase + len(EnvironmentNames)
}

// World is what the map needs from the game.
type World interface {
	Rooms() []parser.Room
	GetZone(number int) (*parser.Zone, bool)
}

// AreaNames names every zone that has rooms for Mudlet. A zone keeps its own
// name; the rare zones that share a name get their number appended, since a
// Mudlet area is identified by name and two zones must not merge into one.
func AreaNames(world World) map[int]string {
	names := map[int]string{}
	count := map[string]int{}
	rooms := world.Rooms()
	for i := range rooms {
		room := &rooms[i]
		if _, done := names[room.Zone]; done {
			continue
		}
		name := fmt.Sprintf("Zone %d", room.Zone)
		if zone, ok := world.GetZone(room.Zone); ok && zone != nil && zone.Name != "" {
			name = zone.Name
		}
		names[room.Zone] = name
		count[name]++
	}
	for zone, name := range names {
		if count[name] > 1 {
			names[zone] = fmt.Sprintf("%s (zone %d)", name, zone)
		}
	}
	return names
}

type coord struct{ x, y, z int }

// layout places every room of one zone on the grid. Components that no exit
// connects are set side by side, left to right.
func layout(rooms []parser.Room, inZone map[int]*parser.Room) map[int]coord {
	placed := map[int]coord{}
	taken := map[coord]bool{}
	maxX := -3
	place := func(vnum int, at coord) {
		placed[vnum] = at
		taken[at] = true
		if at.x > maxX {
			maxX = at.x
		}
	}
	for i := range rooms {
		start := rooms[i].VNum
		if _, done := placed[start]; done {
			continue
		}
		place(start, coord{maxX + 3, 0, 0})
		queue := []int{start}
		for len(queue) > 0 {
			vnum := queue[0]
			queue = queue[1:]
			from := placed[vnum]
			for _, dir := range directions {
				exit, ok := inZone[vnum].Exits[dir.key]
				if !ok {
					continue
				}
				if _, same := inZone[exit.ToRoom]; !same {
					continue
				}
				if _, done := placed[exit.ToRoom]; done {
					continue
				}
				at := coord{from.x + dir.x, from.y + dir.y, from.z + dir.z}
				// Keep going the way the exit points until the square is free,
				// so the room still lies in its exit's direction.
				for step := 0; taken[at] && step < 64; step++ {
					at = coord{at.x + dir.x, at.y + dir.y, at.z + dir.z}
				}
				for step := 1; taken[at]; step++ {
					at = coord{from.x + dir.x + step, from.y + dir.y - step, from.z + dir.z}
				}
				place(exit.ToRoom, at)
				queue = append(queue, exit.ToRoom)
			}
		}
	}
	return placed
}

// XML map document, in the element and attribute names Mudlet's
// XMLimport::readMap reads.
type (
	xmlMap struct {
		XMLName xml.Name  `xml:"map"`
		Areas   []xmlArea `xml:"areas>area"`
		Rooms   []xmlRoom `xml:"rooms>room"`
	}
	xmlArea struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name,attr"`
	}
	xmlRoom struct {
		ID          int       `xml:"id,attr"`
		Area        int       `xml:"area,attr"`
		Title       string    `xml:"title,attr"`
		Environment int       `xml:"environment,attr"`
		Coord       xmlCoord  `xml:"coord"`
		Exits       []xmlExit `xml:"exit"`
	}
	xmlCoord struct {
		X int `xml:"x,attr"`
		Y int `xml:"y,attr"`
		Z int `xml:"z,attr"`
	}
	xmlExit struct {
		Direction string `xml:"direction,attr"`
		Target    int    `xml:"target,attr"`
		Door      int    `xml:"door,attr,omitempty"`
	}
)

// Generate renders the world. The output depends only on the world, so its
// Version changes exactly when a room, exit or zone name does.
func Generate(world World) ([]byte, error) {
	names := AreaNames(world)
	byZone := map[int][]parser.Room{}
	all := map[int]*parser.Room{}
	rooms := world.Rooms()
	for i := range rooms {
		room := rooms[i]
		// Mudlet room ids start at 1; vnum 0 is The Void, which no exit
		// reaches and nobody walks.
		if room.VNum < 1 {
			continue
		}
		byZone[room.Zone] = append(byZone[room.Zone], room)
	}
	zones := make([]int, 0, len(byZone))
	for zone, rooms := range byZone {
		zones = append(zones, zone)
		sort.Slice(rooms, func(i, j int) bool { return rooms[i].VNum < rooms[j].VNum })
		for i := range rooms {
			all[rooms[i].VNum] = &rooms[i]
		}
	}
	sort.Ints(zones)

	var doc xmlMap
	for _, zone := range zones {
		// Mudlet reserves area -1 and treats area ids as positive, so zone n
		// is area n+1.
		area := zone + 1
		doc.Areas = append(doc.Areas, xmlArea{ID: area, Name: names[zone]})
		inZone := map[int]*parser.Room{}
		for i := range byZone[zone] {
			inZone[byZone[zone][i].VNum] = &byZone[zone][i]
		}
		placed := layout(byZone[zone], inZone)
		for i := range byZone[zone] {
			room := &byZone[zone][i]
			_, env := Environment(room.Sector)
			at := placed[room.VNum]
			out := xmlRoom{
				ID: room.VNum, Area: area, Title: room.Name, Environment: env,
				Coord: xmlCoord{X: at.x, Y: at.y, Z: at.z},
			}
			for _, dir := range directions {
				exit, ok := room.Exits[dir.key]
				if !ok || all[exit.ToRoom] == nil {
					continue
				}
				// Door 1 marks "a door is here". Whether it is open, closed
				// or locked changes with zone resets and players, so the map
				// does not claim a state.
				door := 0
				if exit.DoorState > 0 {
					door = 1
				}
				out.Exits = append(out.Exits, xmlExit{Direction: dir.word, Target: exit.ToRoom, Door: door})
			}
			doc.Rooms = append(doc.Rooms, out)
		}
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", " ")
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// Version is a short content hash of a generated map, for clients to tell a
// new map from the one they have.
func Version(generated []byte) string {
	sum := sha256.Sum256(generated)
	return hex.EncodeToString(sum[:6])
}
