//nolint:unused // Game logic port — not yet wired to command registry.
package game

// mapcode.go — Go port of src/mapcode.c
// ASCII map rendering. Recursively draws a map of surrounding rooms.
//
// Legend:
//   Sector icons: 0=inside, #=city, :=field, +=forest, %=hills, ^=mountain,
//                 ~=water swim, ~=noswim, _=underwater, @=flying, $=desert,
//                 F=fire, E=earth, W=wind, w=water, `=swamp, I=ice
//   X = You are here
//   o = linkback or nowhere exit
//   - = link to bad/nonexistent room
//   / + ? = exit connections
//   | = up/down exit
//   * = overlapping rooms

import (
	"fmt"
	"strings"

)

const (
	mapMaxX = 76
	mapMaxY = 25
	mapMaxOverlap = 1
)

// Sector display characters (indexed by sector type).
var sectorIcons = []string{
	"0", // SECT_INSIDE
	"#", // SECT_CITY
	":", // SECT_FIELD
	"+", // SECT_FOREST
	"%", // SECT_HILLS
	"^", // SECT_MOUNTAIN
	"~", // SECT_WATER_SWIM
	"~", // SECT_WATER_NOSWIM
	"_", // SECT_UNDERWATER
	"@", // SECT_FLYING
	"$", // SECT_DESERT
	"F", // SECT_FIRE
	"E", // SECT_EARTH
	"W", // SECT_WIND
	"w", // SECT_WATER
	"`", // SECT_SWAMP
	"I", // SECT_ICE
}

// graphChars maps display values to rendering characters.
// display values: -8=you, -7=up/down, -6=overlap link, -5=bad link, -4=linkback,
//                  -3=west link, -2=north link, -1=overlap/error, 0=empty
var graphChars = map[int]string{
	-8: "X",   // you are here
	-7: "|",   // up/down exit
	-6: "*",   // overlapped link
	-5: "-",   // link to bad room
	-4: "o",   // linkback / nowhere
	-3: "/",   // west link
	-2: "/",   // north link (C uses different chars, simplified)
	-1: "?",   // overlap/error
	 0: " ",   // empty
}

// mapRecursive recursively draws the map onto the display buffer.
// Ported from C map() function.
//
// display values:
//   > 0  = room vnum
//   -1   = overlap/error
//   -2   = north link
//   -3   = west link
//   -4   = linkback
//   -5   = bad link
//   -6   = overlapped link
//   -7   = up/down exit
//   -8   = player position
//
// Parameters:
//   thisroom  = room vnum
//   x, y      = position in display buffer
//   overlap   = current overlap count
//   mode      = 0=all, 1=zone only, 2=allow overlap, 3=underlap
func (w *World) mapRecursive(display *[mapMaxX + 2][mapMaxY + 2]int, thisroom int, x, y, overlap, mode int) {
	// Direction offsets for north(0), east(1), south(2), west(3)
	offx := [4]int{0, 1, 0, -1}
	offy := [4]int{-1, 0, 1, 0}
	link := [4]int{-2, -3, -2, -3} // link chars per direction

	if x < 1 || y < 1 || x > (mapMaxX-2) || y > (mapMaxY-2) {
		return
	}

	if display[x][y] == thisroom {
		return
	}

	if display[x][y] != 0 {
		// Room already drawn here, not us
		switch mode {
		case 1:
			display[x][y] = -1
		case 2:
			display[x][y] = thisroom
		case 3:
			// underlap — keep existing
		default:
			display[x][y] = -1
		}
		overlap++
	} else {
		display[x][y] = thisroom
		overlap = 0
	}

	if overlap >= mapMaxOverlap {
		return
	}

	room := w.GetRoomInWorld(thisroom)
	if room == nil {
		return
	}

	// Draw up and down exits
	if ext, ok := room.Exits["down"]; ok && ext.ToRoom > -2 {
		if x+1 >= 0 && x+1 <= mapMaxX && y-1 >= 0 && y-1 <= mapMaxY {
			display[x+1][y-1] = -7
		}
	}
	if ext, ok := room.Exits["up"]; ok && ext.ToRoom > -2 {
		if x-1 >= 0 && x-1 <= mapMaxX && y+1 >= 0 && y+1 <= mapMaxY {
			display[x-1][y+1] = -7
		}
	}

	// Map cardinal exits: north(0), east(1), south(2), west(3)
	for dir := 0; dir < 4; dir++ {
		dirName := dirs[dir]
		ext, hasExit := room.Exits[dirName]
		if !hasExit {
			continue
		}

		nx := x + offx[dir]
		ny := y + offy[dir]
		if nx < 0 || nx > mapMaxX || ny < 0 || ny > mapMaxY {
			continue
		}

		nextroom := ext.ToRoom

		if nextroom > -2 {
			if nextroom == thisroom || nextroom == -1 {
				display[nx][ny] = -4 // linkback
			} else if nextroom > 0 {
				display[nx][ny] = link[dir]
				// Recurse if not restricted by mode
				if mode == 0 || room.Zone == w.GetRoomZone(nextroom) {
					w.mapRecursive(display, nextroom, x+3*offx[dir], y+3*offy[dir], overlap, mode)
				}
			}
		} else {
			display[nx][ny] = -5 // link to bad room
		}
	}
}

// DoMap renders an ASCII map of surrounding rooms for the player.
// Ported from C do_map().
// Arguments: "" or "a" (all), "b" (allow overlap), "c" (underlap)
func (w *World) DoMap(ch *Player, argument string) {
	var display [mapMaxX + 2][mapMaxY + 2]int

	x := mapMaxX / 2
	y := mapMaxY / 2
	thisroom := ch.GetRoom()

	mode := 1 // default: zone only
	argument = strings.TrimSpace(argument)
	if len(argument) > 1 {
		switch argument[1] {
		case 'a':
			mode = 0
		case 'b':
			mode = 2
		case 'c':
			mode = 3
		}
	}

	w.mapRecursive(&display, thisroom, x, y, 0, mode)
	display[x][y] = -8 // mark player position

	sendToChar(ch, "You look down upon the world and see...\r\n")

	for row := 0; row < mapMaxY; row++ {
		var line strings.Builder
		for col := 0; col < mapMaxX; col++ {
			v := display[col][row]
			if v > 0 {
				room := w.GetRoomInWorld(v)
				if room != nil && room.Sector >= 0 && room.Sector < len(sectorIcons) {
					line.WriteString(sectorIcons[room.Sector])
				} else {
					line.WriteString("?")
				}
			} else if ch, ok := graphChars[v]; ok {
				line.WriteString(ch)
			} else {
				line.WriteString(" ")
			}
		}
		sendToChar(ch, line.String()+"\r\n")
	}

	// Legend
	legend := fmt.Sprintf(
		"\r\nKEY: inside=%s, city=%s, field=%s, forest=%s, hills=%s,\r\n"+
			"     mountain=%s, swim=%s, noswim=%s, underwater=%s,\r\n"+
			"     flying=%s, desert=%s, fire=%s, earth=%s, wind=%s,\r\n"+
			"     water=%s, swamp=%s\r\n"+
			"     X = You are here. * = overlapping room(s)\r\n",
		sectorIcons[0],  sectorIcons[1],  sectorIcons[2],  sectorIcons[3],
		sectorIcons[4],  sectorIcons[5],  sectorIcons[6],  sectorIcons[7],
		sectorIcons[8],  sectorIcons[9],  sectorIcons[10], sectorIcons[11],
		sectorIcons[12], sectorIcons[13], sectorIcons[14], sectorIcons[15],
	)
	sendToChar(ch, legend)
}

// getSectorIcon returns the display character for a sector type.
func getSectorIcon(sector int) string {
	if sector >= 0 && sector < len(sectorIcons) {
		return sectorIcons[sector]
	}
	return "?"
}
