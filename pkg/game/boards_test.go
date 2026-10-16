package game

import (
	"bytes"
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

func TestGenBoard_ReadAndWriteMatchBoardEditorEntry(t *testing.T) {
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
	world.Boards.SetWorld(world)
	world.AddItemToRoom(NewObjectInstance(&parsed.Objs[0], 8008), 8008)
	player := NewPlayer(1, "Reader", 8008)
	if err := world.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	world.MessageSink = func(_ string, msg []byte) { output.Write(msg) }

	if !genBoard(world, player, nil, "read", "board") {
		t.Fatal("read board was not handled by gen_board")
	}
	if !bytes.Contains(output.Bytes(), []byte("This is a bulletin board.")) {
		t.Fatalf("read board output = %q", output.String())
	}
	output.Reset()
	if !genBoard(world, player, nil, "write", "headline") {
		t.Fatal("write board was not handled by gen_board")
	}
	if player.WriteMagic == 0 || player.GetFlags()&(1<<PlrWriting) == 0 {
		t.Fatalf("board write state = magic %d flags %d", player.WriteMagic, player.GetFlags())
	}
	if !bytes.Contains(output.Bytes(), []byte("Instructions: /s or @ to save")) {
		t.Fatalf("write board output = %q", output.String())
	}
}

func newBoardSpecialOutputTest(t *testing.T) (*World, *Player, *bytes.Buffer) {
	t.Helper()
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
	world.Boards.SetWorld(world)
	world.AddItemToRoom(NewObjectInstance(&parsed.Objs[0], 8008), 8008)
	player := NewPlayer(1, "Writer", 8008)
	if err := world.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	world.MessageSink = func(_ string, msg []byte) { output.Write(msg) }
	return world, player, &output
}

func TestGenBoardFullBoardWriteRefusalOutput(t *testing.T) {
	world, player, output := newBoardSpecialOutputTest(t)
	for i := 0; i < boards.MaxBoardMessages; i++ {
		magic := world.Boards.WriteMessage(0, player, "filled")
		if magic <= 0 {
			t.Fatalf("board post %d failed to allocate", i+1)
		}
		world.Boards.FinalizeBoardWrite(magic, player)
	}
	output.Reset()
	if !genBoard(world, player, nil, "write", "overflow") {
		t.Fatal("full-board write was not handled by gen_board")
	}
	if got, want := output.String(), "The board is full.\r\n"; got != want {
		t.Fatalf("full-board refusal = %q, want %q", got, want)
	}
}

func TestGenBoardGlobalSlotExhaustionRefusalOutput(t *testing.T) {
	world, player, output := newBoardSpecialOutputTest(t)
	capacity := boards.NumBoards*boards.MaxBoardMessages + 5
	for i := 0; i < capacity; i++ {
		if !genBoard(world, player, nil, "write", "") {
			t.Fatalf("empty-headline attempt %d was not handled by gen_board", i+1)
		}
		if got, want := output.String(), "We must have a headline!\r\n"; got != want {
			t.Fatalf("empty-headline attempt %d = %q, want %q", i+1, got, want)
		}
		output.Reset()
	}
	if !genBoard(world, player, nil, "write", "") {
		t.Fatal("slot-exhausted write was not handled by gen_board")
	}
	if got, want := output.String(), "The board is malfunctioning - sorry.\r\n"; got != want {
		t.Fatalf("global-slot refusal = %q, want %q", got, want)
	}
}
