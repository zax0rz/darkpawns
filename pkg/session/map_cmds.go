package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

const (
	jjMapMaxX = 76
	jjMapMaxY = 25
	jjMapMaxO = 1
)

// directionName maps the C direction index (0=N,1=E,2=S,3=W) to the Go exit key.
var dirName = [4]string{"north", "east", "south", "west"}

// offX maps N/E/S/W direction index to x offset (same as C: {0,1,0,-1}).
var offX = [4]int{0, 1, 0, -1}

// offY maps N/E/S/W direction index to y offset (same as C: {-1,0,1,0}).
var offY = [4]int{-1, 0, 1, 0}

// link maps N/E/S/W direction index to link display value (same as C: {-2,-3,-2,-3}).
var link = [4]int{-2, -3, -2, -3}

// hasExit checks whether a room has an exit with to_room > -2 (matching the C pattern
// "if (world[thisroom].dir_option[DIR])" which was truthy when the exit pointer was non-null).
func hasExit(room *Room, dirName string) bool {
	if room == nil {
		return false
	}
	exit, ok := room.Exits[dirName]
	if !ok {
		return false
	}
	return exit.ToRoom > -2
}

// mapRecurse is a faithful port of the C map() function.
// display is a local [][]int passed by pointer for goroutine safety.
// rooms is the world's room map.
func mapRecurse(thisroom int, x, y, overlap, dontleavezone int, display *[][]int, rooms map[int]*Room) {
	// Bounds check — same as C: if ((x<1)||(y<1)||(x>(JJ_MAPMAXX-2))||(y>(JJ_MAPMAXY-2))) return;
	if x < 1 || y < 1 || x > (jjMapMaxX-2) || y > (jjMapMaxY-2) {
		return
	}

	// Already visited — same as C: if (display[x][y] == thisroom) return;
	if (*display)[x][y] == thisroom {
		return
	}

	// Collision handling — same as C:
	// if(display[x][y] != 0) { switch(dontleavezone) { ... } overlap++; }
	// else { display[x][y] = thisroom; overlap = 0; }
	if (*display)[x][y] != 0 {
		switch dontleavezone {
		case 1:
			(*display)[x][y] = -1
		case 2:
			(*display)[x][y] = thisroom
		case 3:
			// case 3: display[x][y] = display[x][y]; — no-op, keep current value
		default:
			(*display)[x][y] = -1
		}
		overlap++
	} else {
		(*display)[x][y] = thisroom
		overlap = 0
	}

	// Overlap limit — same as C: if (overlap >= JJ_MAPMAXO) return;
	if overlap >= jjMapMaxO {
		return
	}

	// Get current room
	thisRoomPtr := rooms[thisroom]
	if thisRoomPtr == nil {
		return
	}

	// Down exit marker — same as C: if (world[thisroom].dir_option[DOWN]) display[x+1][y-1] = -7;
	if hasExit(thisRoomPtr, "down") {
		(*display)[x+1][y-1] = -7
	}

	// Up exit marker — same as C: if (world[thisroom].dir_option[UP]) display[x-1][y+1] = -7;
	if hasExit(thisRoomPtr, "up") {
		(*display)[x-1][y+1] = -7
	}

	// Iterate N/E/S/W (dir=0..3) — same as C
	for dir := 0; dir < 4; dir++ {
		exit, ok := thisRoomPtr.Exits[dirName[dir]]
		if !ok {
			continue
		}

		nextroom := exit.ToRoom

		// Same as C: if (nextroom > -2)
		if nextroom > -2 {
			// Same as C: if((nextroom == thisroom)||(nextroom == -1))
			if nextroom == thisroom || nextroom == -1 {
				(*display)[x+offX[dir]][y+offY[dir]] = -4
			} else if nextroom > 0 {
				(*display)[x+offX[dir]][y+offY[dir]] = link[dir]

				// Same as C: if(!dontleavezone || (world[thisroom].zone == world[nextroom].zone))
				nextRoomPtr := rooms[nextroom]
				if !(dontleavezone != 0) || (thisRoomPtr.Zone == nextRoomPtr.Zone) {
					mapRecurse(nextroom, x+3*offX[dir], y+3*offY[dir], overlap, dontleavezone, display, rooms)
				}
			}
		} else {
			// Same as C: display[x+offx[dir]][y+offy[dir]] = -5;
			(*display)[x+offX[dir]][y+offY[dir]] = -5
		}
	}

	// Same as C: if((overlap)&&(display[x+offx[dir]][y+offy[dir]] != 0))
	// NOTE: In the C code, 'dir' here refers to the final value of dir after the loop (dir==4).
	if overlap != 0 && (*display)[x+offX[3]][y+offY[3]] != 0 {
		(*display)[x+offX[3]][y+offY[3]] = -6
	}
}

// CmdMap implements the do_map() command (faithful C-to-Go port).
// Usage: map [a|b|c]
func CmdMap(s *Session, args []string) error {
	// Level check — same as C: if (GET_LEVEL(ch)<LVL_IMMORT) { stc(...); return; }
	if s.player == nil || s.player.Level < LVL_IMMORT {
		s.Send("Type HELP MAP to see a map of town.\r\n")
		return nil
	}

	// Initialize display array — same as C: for(y=0;...) for(x=0;...) display[x][y] = 0;
	// Local slice for goroutine safety (no global).
	display := make([][]int, jjMapMaxX+2)
	for x := range display {
		display[x] = make([]int, jjMapMaxY+2)
	}

	// Same as C: x = JJ_MAPMAXX / 2; y = JJ_MAPMAXY / 2;
	x := jjMapMaxX / 2
	y := jjMapMaxY / 2

	// Same as C: thisroom = ch->in_room;
	thisroom := s.player.GetRoomVNum()

	// Determine mapping mode — same as C:
	// if (!*argument) i = 1;
	// else if (argument[1] == 'a') i = 0;
	// else if (argument[1] == 'b') i = 2;
	// else if (argument[1] == 'c') i = 3;
	// else i = 1;
	var i int
	arg := strings.Join(args, " ")
	if arg == "" {
		i = 1
	} else if len(arg) >= 2 && arg[1] == 'a' {
		i = 0
	} else if len(arg) >= 2 && arg[1] == 'b' {
		i = 2
	} else if len(arg) >= 2 && arg[1] == 'c' {
		i = 3
	} else {
		i = 1
	}

	// Build rooms map from world via lock-free snapshot
	snap := s.manager.world.GetSnapshotManager().Snapshot()
	if snap == nil {
		s.Send("World snapshot unavailable.\r\n")
		return nil
	}
	rooms := snap.Rooms

	// Same as C: map(thisroom,x,y,0,i, ch);
	mapRecurse(thisroom, x, y, 0, i, &display, rooms)

	// Same as C: display[x][y] = -8;
	display[x][y] = -8

	// Same as C: send_to_char("You look down upon the world and see...\r\n", ch);
	s.Send("You look down upon the world and see...\r\n")

	// Graphics for special display values (indexed as value+8, same as C)
	// graph[0..8] maps to display values -8..0:
	//   -8: &RX&n (player), -7: / (up/down), -6: + (overlap?), -5: ? (blocked)
	//   -4: &Do&n (no door), -3: &D-&n, -2: &D|&n, -1: &D*&n, 0: " "
	graph := []string{"&RX&n", "/", "+", "?", "&Do&n", "&D-&n", "&D|&n", "&D*&n", " "}

	// Sector icons (indexed by sector type 0..16, same as C)
	sectIcons := []string{
		"&g0&n", "&m#&n", "&Y:&n", "&G+&n", "&w%&n", "&y^&n", "&B~&n", "&b~&n",
		"&b_&n", "&C@&n", "&r$&n", "&RF&n", "&YE&n", "&CW&n", "&cw&n", "&D`&n", "&WI&n",
	}

	// Render map — same as C:
	// for(y=0;y<JJ_MAPMAXY;y++) { line[0]='\0'; for(x=0;...){...} strcat(line,"\r\n"); send_to_char(line,ch); }
	for y := 0; y < jjMapMaxY; y++ {
		var line strings.Builder
		for x := 0; x < jjMapMaxX; x++ {
			val := display[x][y]
			if val > 0 {
				// Same as C: strcat(line, sect_icons[world[display[x][y]].sector_type]);
				if room, ok := rooms[val]; ok && room.Sector >= 0 && room.Sector < len(sectIcons) {
					line.WriteString(sectIcons[room.Sector])
				} else {
					line.WriteString(" ")
				}
			} else {
				// Same as C: strcat(line, graph[(display[x][y])+8]);
				idx := val + 8
				if idx >= 0 && idx < len(graph) {
					line.WriteString(graph[idx])
				} else {
					line.WriteString(" ")
				}
			}
		}
		line.WriteString("\r\n")
		s.Send(line.String())
	}

	// Key legend output — ported from the C original's key legend
	s.Send("\r\nMap Key:\r\n")
	s.Send("&RX&n = You          &D*&n = Blocked exit\r\n")
	s.Send("&D|&n = N/S passage  &D-&n = E/W passage\r\n")
	s.Send("&Do&n = Closed door  / = Up/Down exit\r\n")
	s.Send("&g0&n = Inside       &m#&n = City       &Y:&n = Field\r\n")
	s.Send("&G+&n = Forest       &w%&n = Hills      &y^&n = Mountains\r\n")
	s.Send("&B~&n = Water(Swim)  &b~&n = Water      &b_&n = Rapids\r\n")
	s.Send("&C@&n = Unused       &r$&n = Road       &RF&n = Clan Hall\r\n")
	s.Send("&YE&n = Rune Essence &CW&n = Water(Walk)&cw&n = Water(Noswim)\r\n")
	s.Send("&D`&n = Death Trap   &WI&n = Rune Nexus\r\n")

	return nil
}

// init registers the map command.
func init() {
	cmdRegistry.Register("map", wrapArgs(CmdMap), "Display a map of the world.", LVL_IMMORT, 0)
}

// Room type alias for brevity — maps to parser.Room
type Room = parser.Room

/*
IMPROVEMENTS over the C original:

1. Global display array converted to local (goroutine safe):
   The C version used a package-level global `int display[78][27]` which would cause
   data races if multiple players invoked the map command concurrently. The Go version
   allocates the display as a local `[][]int` slice passed by pointer to mapRecurse().

2. Recursion uses pointer-to-slice instead of global state:
   mapRecurse() takes `*[][]int` as a parameter rather than relying on a global,
   making it safe for concurrent goroutines.

3. String building uses strings.Builder instead of strcat:
   The C version used strcat for string concatenation, which is O(n²) per row.
   The Go version uses `strings.Builder` for O(n) string assembly.

4. Exits use string-keyed map instead of int-indexed array:
   The C code accessed exits via numeric indices (0-5 for N/E/S/W/U/D).
   The Go parser stores exits in a `map[string]Exit`, so direction names
   ("north", "east", etc.) are used instead.

5. Room access uses lock-free snapshot instead of direct global array:
   The C code did direct array indexing on the world array. The Go version
   uses the World's lock-free snapshot for safe concurrent reads.

6. No unused `buf` variable:
   The C do_map() declared `char buf[MAX_STRING_LENGTH]` but never used it.
   The Go version omits this unused declaration.
*/
