package game

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

type channelRecord struct {
	to, channel, talker, line string
}

type recordingObserver struct {
	rooms    []int
	lines    []channelRecord
	ticks    int
	roomsFor []string
}

func (o *recordingObserver) RoomShown(p *Player, roomVNum int) {
	o.rooms = append(o.rooms, roomVNum)
	o.roomsFor = append(o.roomsFor, p.Name)
}

func (o *recordingObserver) ChannelLine(p *Player, channel, talker, line string) {
	o.lines = append(o.lines, channelRecord{to: p.Name, channel: channel, talker: talker, line: line})
}

func (o *recordingObserver) PointUpdated() { o.ticks++ }

// communicationScript exercises every Act-delivered channel line: say to the
// room and its echo, a tell and its echo, and a gossip broadcast.
func communicationScript(w *World, actor *Player) {
	actor.SetLevel(levelCanShout)
	w.DoSay(actor, "hello there")
	w.DoTell(actor, "Target psst")
	w.DoChannel(actor, "anyone around", "gossip")
}

// TestChannelMirrorLeavesTextUnchanged is the text-path guarantee: attaching
// the out-of-band observer changes no byte any player receives.
func TestChannelMirrorLeavesTextUnchanged(t *testing.T) {
	plainWorld, plainActor, _, plainOutput := newDirectedSpeechWorld(t)
	communicationScript(plainWorld, plainActor)

	observedWorld, observedActor, _, observedOutput := newDirectedSpeechWorld(t)
	observedWorld.OutOfBand = &recordingObserver{}
	communicationScript(observedWorld, observedActor)

	for _, name := range []string{"Actor", "Target"} {
		if got, want := directedOutput(observedOutput, name), directedOutput(plainOutput, name); got != want {
			t.Fatalf("%s text changed with an observer attached:\n got %q\nwant %q", name, got, want)
		}
	}
}

// TestChannelMirrorReportsDeliveredLines checks each mirrored line is the
// exact line its recipient was sent, tagged with the channel and the speaker
// as that recipient saw them.
func TestChannelMirrorReportsDeliveredLines(t *testing.T) {
	w, actor, _, output := newDirectedSpeechWorld(t)
	observer := &recordingObserver{}
	w.OutOfBand = observer
	communicationScript(w, actor)

	want := []channelRecord{
		{to: "Target", channel: "say", talker: "Actor", line: "Actor says, 'hello there'\r\n"},
		{to: "Actor", channel: "say", talker: "Actor", line: "You say 'hello there'\r\n"},
		{to: "Target", channel: "tell", talker: "Actor", line: "Actor tells you, 'psst'\r\n"},
		{to: "Actor", channel: "tell", talker: "Actor", line: "You tell Target, 'psst'\r\n"},
		{to: "Actor", channel: "gossip", talker: "Actor", line: "You gossip, 'anyone around'\r\n"},
		{to: "Target", channel: "gossip", talker: "Actor", line: "Actor gossips, 'anyone around'\r\n"},
	}
	if !reflect.DeepEqual(observer.lines, want) {
		t.Fatalf("mirrored lines:\n got %+v\nwant %+v", observer.lines, want)
	}
	for _, record := range observer.lines {
		if !strings.Contains(directedOutput(output, record.to), record.line) {
			t.Fatalf("mirrored line %q was never delivered to %s", record.line, record.to)
		}
	}
}

// TestChannelMirrorKeepsSpeakerVisibility proves the mirror names a speaker
// no better than the text did: an unseen gossiper is "Someone" in both.
func TestChannelMirrorKeepsSpeakerVisibility(t *testing.T) {
	w, actor, target, output := newDirectedSpeechWorld(t)
	observer := &recordingObserver{}
	w.OutOfBand = observer
	actor.SetLevel(levelCanShout)
	actor.SetAffect(affInvisible, true)
	w.DoChannel(actor, "who said that", "gossip")

	if got := directedOutput(output, target.Name); got != "Someone gossips, 'who said that'\r\n" {
		t.Fatalf("target text = %q", got)
	}
	var heard *channelRecord
	for i := range observer.lines {
		if observer.lines[i].to == target.Name {
			heard = &observer.lines[i]
		}
	}
	if heard == nil || heard.talker != "Someone" || heard.line != "Someone gossips, 'who said that'\r\n" {
		t.Fatalf("mirror for unseen speaker = %+v", heard)
	}
}

// TestChannelMirrorSkipsNoRepeatAcknowledgement: PRF_NOREPEAT replaces the
// echo with "Okay.", which is not a channel line.
func TestChannelMirrorSkipsNoRepeatAcknowledgement(t *testing.T) {
	w, actor, _, _ := newDirectedSpeechWorld(t)
	observer := &recordingObserver{}
	w.OutOfBand = observer
	actor.SetLevel(levelCanShout)
	actor.SetPlrFlag(PrfNoRepeat, true)
	w.DoChannel(actor, "quietly", "gossip")
	for _, record := range observer.lines {
		if record.to == actor.Name {
			t.Fatalf("norepeat acknowledgement was mirrored: %+v", record)
		}
	}
}

func newRoomObserverWorld(t *testing.T) (*World, *Player, *recordingObserver) {
	t.Helper()
	lit := []string{"0", "0", "0", "0"}
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 2001, Name: "Lit Hall", Flags: lit, Exits: map[string]parser.Exit{
			"north": {Direction: "north", ToRoom: 2002},
		}},
		{VNum: 2002, Name: "Dark Closet", Flags: []string{"1", "0", "0", "0"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	player := directedSpeechPlayer(t, w, 1, "Viewer", 2001)
	w.MessageSink = func(string, []byte) {}
	observer := &recordingObserver{}
	w.OutOfBand = observer
	return w, player, observer
}

// TestRoomShownFiresOnlyWhenARoomIsRendered: a room render reports the room
// once; peeking through an exit, darkness, and blindness report nothing,
// because none of them print the room the player is in.
func TestRoomShownFiresOnlyWhenARoomIsRendered(t *testing.T) {
	w, player, observer := newRoomObserverWorld(t)

	w.RenderObservationMessages(w.DoLook(player, "look", ""))
	if !reflect.DeepEqual(observer.rooms, []int{2001}) || observer.roomsFor[0] != "Viewer" {
		t.Fatalf("room render reported %v for %v", observer.rooms, observer.roomsFor)
	}

	w.RenderObservationMessages(w.DoLook(player, "look", "north"))
	if len(observer.rooms) != 1 {
		t.Fatalf("peeking north reported a room: %v", observer.rooms)
	}

	player.SetRoom(2002)
	w.RenderObservationMessages(w.DoLookRoom(player, true))
	if len(observer.rooms) != 1 {
		t.Fatalf("dark room reported a room: %v", observer.rooms)
	}

	player.SetRoom(2001)
	player.SetAffect(affBlind, true)
	w.RenderObservationMessages(w.DoLookRoom(player, true))
	if len(observer.rooms) != 1 {
		t.Fatalf("blind look reported a room: %v", observer.rooms)
	}
}

func TestPointUpdateNotifiesObserverOnce(t *testing.T) {
	w, _, observer := newRoomObserverWorld(t)
	w.PointUpdate()
	if observer.ticks != 1 {
		t.Fatalf("PointUpdated fired %d times, want 1", observer.ticks)
	}
}

// TestPCRaceAndClassTablesMatchC pins the two name tables to C's
// constants.c races[] and class.c pc_class_types[], in constant order.
func TestPCRaceAndClassTablesMatchC(t *testing.T) {
	races := map[int]string{
		RaceHuman: "Human", RaceElf: "Elven", RaceDwarf: "Dwarven", RaceKender: "Kenderkin",
		RaceMinotaur: "Minotauran", RaceRakshasa: "Rakshasan", RaceSsaur: "Ssauran",
	}
	for index, name := range races {
		if PCRaceTypes[index] != name {
			t.Fatalf("PCRaceTypes[%d] = %q, want %q", index, PCRaceTypes[index], name)
		}
	}
	if PCClassTypes[ClassMageUser] != "Magic User" || PCClassTypes[ClassMystic] != "Mystic" || len(PCClassTypes) != 12 {
		t.Fatalf("PCClassTypes = %q", PCClassTypes)
	}
}
