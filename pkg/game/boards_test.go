package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/boards"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestGenBoard_DoesNotInterceptBareLook(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 8008, Name: "Temple"}},
		Objs:  []parser.Obj{{VNum: 8099, Keywords: "board bulletin"}},
	}
	world, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)
	world.Boards = boards.InitBoards(t.TempDir())
	world.AddItemToRoom(NewObjectInstance(&parsed.Objs[0], 8008), 8008)
	player := NewPlayer(1, "Reader", 8008)

	if genBoard(world, player, nil, "look", "") {
		t.Fatal("bare look was intercepted by gen_board; C falls through to room look")
	}
	if !genBoard(world, player, nil, "look", "board") {
		t.Fatal("look board was not handled by gen_board")
	}
	if !genBoard(world, player, nil, "look", "BULLETIN") {
		t.Fatal("look BULLETIN was not handled case-insensitively like C isname")
	}
}
