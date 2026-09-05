package game

import "testing"

func TestHomeLoadRoomUsesExistingSaveRoomField(t *testing.T) {
	player := NewPlayer(1, "Homekeeper", 1001)
	player.SetLoadRoom(1002)
	player.SetPlrFlag(PlrLoadroom, true)

	data := playerToSaveData(player)
	if data.RoomVNum != 1002 {
		t.Fatalf("saved room_vnum = %d, want selected load room 1002", data.RoomVNum)
	}

	restored := saveDataToPlayer(data)
	if restored.GetLoadRoom() != 1002 {
		t.Fatalf("restored load room = %d, want 1002", restored.GetLoadRoom())
	}
}
