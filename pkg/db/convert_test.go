package db

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRecordToPlayerReconstructsPersistedMail(t *testing.T) {
	world := newConversionWorld(t, nil)
	const mailText = " * * * * Dark Pawns Mail System * * * *\r\n" +
		"Date: Mon Jan 1 00:00:00 1990\r\n" +
		"  To: Recipient\r\n" +
		"From: Sender\r\n\r\n" +
		"exact persisted body"
	record := conversionRecord(`[{"vnum":-1,"count":1,"locate":0,"state":{"mail_text":"` +
		" * * * * Dark Pawns Mail System * * * *\\r\\n" +
		"Date: Mon Jan 1 00:00:00 1990\\r\\n" +
		"  To: Recipient\\r\\n" +
		"From: Sender\\r\\n\\r\\nexact persisted body" +
		`"}}]`)

	restored, err := RecordToPlayer(record, world)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}
	items := restored.Inventory.FindItems("")
	if len(items) != 1 {
		t.Fatalf("restored inventory items = %d, want 1", len(items))
	}
	mail := items[0]
	if mail.VNum != -1 || mail.Prototype != nil {
		t.Fatalf("restored mail identity = vnum %d prototype %v, want synthetic vnum -1", mail.VNum, mail.Prototype)
	}
	if mail.Runtime.MailText != mailText {
		t.Fatalf("restored mail text = %q, want exact sender/body text %q", mail.Runtime.MailText, mailText)
	}
	if !strings.Contains(mail.Runtime.MailText, "From: Sender\r\n\r\nexact persisted body") {
		t.Fatalf("restored mail lost sender/body: %q", mail.Runtime.MailText)
	}
	if mail.GetKeywords() != "mail paper letter" || mail.GetShortDesc() != "a piece of mail" {
		t.Fatalf("restored mail identity fields = keywords %q short %q", mail.GetKeywords(), mail.GetShortDesc())
	}
	if mail.GetTypeFlag() != game.ITEM_NOTE || !mail.CanPickUp {
		t.Fatalf("restored mail type fields = type %d can_pick_up %t, want note/take", mail.GetTypeFlag(), mail.CanPickUp)
	}
	if got, want := mail.Location, game.LocInventoryPlayer(restored.Name); got != want {
		t.Fatalf("restored mail location = %#v, want %#v", got, want)
	}
	if err := mail.Location.Validate(); err != nil {
		t.Fatalf("restored mail location invalid: %v", err)
	}
}

func TestRecordToPlayerDoesNotTreatUnrelatedSyntheticAsMail(t *testing.T) {
	world := newConversionWorld(t, nil)
	record := conversionRecord(`[{"vnum":-1,"count":1,"locate":0,"state":{"name":"the corpse of someone","short_desc":"a corpse"}}]`)

	restored, err := RecordToPlayer(record, world)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}
	if got := len(restored.Inventory.FindItems("")); got != 0 {
		t.Fatalf("unrelated synthetic inventory items = %d, want 0", got)
	}
}

func TestRecordToPlayerSkipsMalformedPersistedMailState(t *testing.T) {
	world := newConversionWorld(t, nil)
	tests := []struct {
		name string
		json string
	}{
		{name: "missing_state", json: `[{"vnum":-1,"count":1,"locate":0}]`},
		{name: "missing_mail_text", json: `[{"vnum":-1,"count":1,"locate":0,"state":{}}]`},
		{name: "null_mail_text", json: `[{"vnum":-1,"count":1,"locate":0,"state":{"mail_text":null}}]`},
		{name: "non_string_mail_text", json: `[{"vnum":-1,"count":1,"locate":0,"state":{"mail_text":7}}]`},
		{name: "empty_mail_text", json: `[{"vnum":-1,"count":1,"locate":0,"state":{"mail_text":""}}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restored, err := RecordToPlayer(conversionRecord(tt.json), world)
			if err != nil {
				t.Fatalf("RecordToPlayer: %v", err)
			}
			if got := len(restored.Inventory.FindItems("")); got != 0 {
				t.Fatalf("malformed mail inventory items = %d, want 0", got)
			}
		})
	}
}

func TestRecordToPlayerKeepsOrdinaryPrototypeInventoryPath(t *testing.T) {
	proto := parser.Obj{VNum: 42, ShortDesc: "a plain token", Keywords: "token", TypeFlag: game.ITEM_OTHER}
	world := newConversionWorld(t, []parser.Obj{proto})
	record := conversionRecord(`[{"vnum":42,"count":1,"locate":0,"state":{"short_desc_override":"a marked token"}}]`)

	restored, err := RecordToPlayer(record, world)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}
	items := restored.Inventory.FindItems("")
	if len(items) != 1 {
		t.Fatalf("ordinary inventory items = %d, want 1", len(items))
	}
	item := items[0]
	if item.Prototype == nil || item.Prototype.VNum != proto.VNum || item.VNum != proto.VNum {
		t.Fatalf("ordinary item identity = vnum %d prototype %#v", item.VNum, item.Prototype)
	}
	if item.GetShortDesc() != "a marked token" || item.GetTypeFlag() != game.ITEM_OTHER {
		t.Fatalf("ordinary item state/type = short %q type %d", item.GetShortDesc(), item.GetTypeFlag())
	}
}

func newConversionWorld(t *testing.T, objects []parser.Obj) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Conversion room", Zone: 1}},
		Objs:  objects,
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	return world
}

func conversionRecord(inventory string) *PlayerRecord {
	return &PlayerRecord{
		ID:        777,
		Name:      "Recipient",
		RoomVNum:  1001,
		Level:     1,
		Class:     3,
		Race:      0,
		StatStr:   10,
		StatInt:   10,
		StatWis:   10,
		StatDex:   10,
		StatCon:   10,
		StatCha:   10,
		Inventory: []byte(inventory),
		Equipment: []byte("{}"),
	}
}

func TestPlayerToRecordAndBack(t *testing.T) {
	// Create mock world with object prototypes
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Spawn Room", Zone: 1},
		},
		Objs: []parser.Obj{
			{
				VNum:      10,
				ShortDesc: "a shiny gold coin",
				Keywords:  "coin gold shiny",
			},
			{
				VNum:      20,
				ShortDesc: "a broad sword",
				Keywords:  "sword broad",
			},
		},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })

	// Set up a player
	p := game.NewPlayer(123, "ConvertTestPlayer", 1001)
	p.SetLevel(15)
	p.SetExp(25000)
	p.SetHP(85)
	p.SetMaxHP(100)
	p.SetMove(90)
	p.SetMaxMove(100)
	p.Gold = 500
	p.BankGold = 1000
	p.Stats.Str = 18
	p.Stats.StrAdd = 50
	p.Stats.Int = 14
	p.Stats.Wis = 12
	p.Stats.Dex = 15
	p.Stats.Con = 13
	p.Stats.Cha = 11
	p.Hunger = 20
	p.Thirst = 24
	p.Title = "the Champion of Light"

	// Add inventory item
	proto10, ok := world.GetObjPrototype(10)
	if !ok {
		t.Fatal("expected prototype 10")
	}
	item10 := game.NewObjectInstance(proto10, -1)
	_ = p.Inventory.AddItem(item10)

	// Equip item
	proto20, ok := world.GetObjPrototype(20)
	if !ok {
		t.Fatal("expected prototype 20")
	}
	item20 := game.NewObjectInstance(proto20, -1)
	// Equip in slot Wield
	slot := game.SlotWield
	p.Equipment.Slots[slot] = item20

	// Convert Player to Record
	rec, err := PlayerToRecord(p, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord failed: %v", err)
	}

	// Verify record values
	if rec.ID != p.ID {
		t.Errorf("Record ID = %d, want %d", rec.ID, p.ID)
	}
	if rec.Name != p.Name {
		t.Errorf("Record Name = %q, want %q", rec.Name, p.Name)
	}
	if rec.Title != p.Title {
		t.Errorf("Record Title = %q, want %q", rec.Title, p.Title)
	}
	if rec.Level != p.Level {
		t.Errorf("Record Level = %d, want %d", rec.Level, p.Level)
	}
	if rec.Exp != p.Exp {
		t.Errorf("Record Exp = %d, want %d", rec.Exp, p.Exp)
	}
	if rec.StatStr != p.Stats.Str {
		t.Errorf("Record Str = %d, want %d", rec.StatStr, p.Stats.Str)
	}

	// Restore Player from Record
	restored, err := RecordToPlayer(rec, world)
	if err != nil {
		t.Fatalf("RecordToPlayer failed: %v", err)
	}

	// Verify restored values
	if restored.ID != p.ID {
		t.Errorf("Restored ID = %d, want %d", restored.ID, p.ID)
	}
	if restored.Name != p.Name {
		t.Errorf("Restored Name = %q, want %q", restored.Name, p.Name)
	}
	if restored.Title != p.Title {
		t.Errorf("Restored Title = %q, want %q", restored.Title, p.Title)
	}
	if restored.Level != p.Level {
		t.Errorf("Restored Level = %d, want %d", restored.Level, p.Level)
	}
	if restored.GetRoom() != p.GetRoom() {
		t.Errorf("Restored Room = %d, want %d", restored.GetRoom(), p.GetRoom())
	}
	if restored.Stats.Str != p.Stats.Str {
		t.Errorf("Restored Stats.Str = %d, want %d", restored.Stats.Str, p.Stats.Str)
	}
	if restored.Inventory.MaxWeight != 280 {
		t.Errorf("Restored MaxWeight = %d, want 280", restored.Inventory.MaxWeight)
	}
	if restored.Inventory.Capacity != 19 {
		t.Errorf("Restored Capacity = %d, want 19", restored.Inventory.Capacity)
	}

	// Verify inventory restored
	invItems := restored.Inventory.FindItems("")
	if len(invItems) != 1 {
		t.Fatalf("expected 1 restored inventory item, got %d", len(invItems))
	}
	if invItems[0].GetVNum() != 10 {
		t.Errorf("restored item VNum = %d, want 10", invItems[0].GetVNum())
	}

	// Verify equipment restored
	eqItem := restored.Equipment.Slots[slot]
	if eqItem == nil {
		t.Fatal("expected equipped item in slot Wield")
	}
	if eqItem.GetVNum() != 20 {
		t.Errorf("restored equipped VNum = %d, want 20", eqItem.GetVNum())
	}
}

func TestTakeNameOverrideDoesNotPersistAcrossRecordReload(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Spawn Room", Zone: 1}},
		Objs: []parser.Obj{
			{
				VNum:       8019,
				ShortDesc:  "a frayed tunic",
				Keywords:   "tunic",
				WearFlags:  [4]int{1<<0 | 1<<3},
				ExtraFlags: [4]int{1 << 17}, // ITEM_TAKE_NAME
			},
			{VNum: 8020, ShortDesc: "a plain token", Keywords: "token"},
		},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	p := game.NewPlayer(456, "Rebooter", 1001)
	tunicProto, ok := world.GetObjPrototype(8019)
	if !ok {
		t.Fatal("expected tunic prototype")
	}
	tunic := game.NewObjectInstance(tunicProto, -1)
	tunic.Runtime.ShortDescOverride = "Rebooter's tunic"
	tunic.Location = game.LocEquippedPlayer(p.Name, game.SlotBody)
	if err := p.Equipment.SetSlot(game.SlotBody, tunic); err != nil {
		t.Fatalf("SetSlot tunic: %v", err)
	}

	// A non-take-name override remains persistent; the exclusion must be
	// specific to C's runtime-only ITEM_TAKE_NAME rename.
	tokenProto, ok := world.GetObjPrototype(8020)
	if !ok {
		t.Fatal("expected token prototype")
	}
	token := game.NewObjectInstance(tokenProto, -1)
	token.Runtime.ShortDescOverride = "a personalized token"
	token.Location = game.LocInventoryPlayer(p.Name)
	if err := p.Inventory.AddItem(token); err != nil {
		t.Fatalf("AddItem token: %v", err)
	}

	rec, err := PlayerToRecord(p, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord: %v", err)
	}
	restored, err := RecordToPlayer(rec, world)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}

	restoredTunic, found := restored.Equipment.GetItemInSlot(game.SlotBody)
	if !found {
		t.Fatal("restored take-name tunic is not equipped")
	}
	if got, want := restoredTunic.GetShortDesc(), "a frayed tunic"; got != want {
		t.Fatalf("take-name description after reload = %q, want prototype %q", got, want)
	}
	restoredToken, found := restored.Inventory.FindItem("token")
	if !found {
		t.Fatal("restored ordinary override item is missing")
	}
	if got, want := restoredToken.GetShortDesc(), "a personalized token"; got != want {
		t.Fatalf("ordinary description override after reload = %q, want %q", got, want)
	}
}

// TestRecordToPlayer_OverCapacityInventoryNotDropped guards against silent item
// loss on load: a saved inventory larger than the current capacity must be fully
// restored, not truncated. Before the fix, RecordToPlayer discarded the
// ErrInventoryFull from AddItem and dropped the overflow.
func TestRecordToPlayer_OverCapacityInventoryNotDropped(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Spawn Room", Zone: 1}},
		Objs:  []parser.Obj{{VNum: 10, ShortDesc: "a gold coin", Keywords: "coin gold"}},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })

	proto, ok := world.GetObjPrototype(10)
	if !ok {
		t.Fatal("expected prototype 10")
	}

	p := game.NewPlayer(321, "PackRat", 1001)
	const itemCount = 30 // exceeds the default capacity of 20
	p.Inventory.Capacity = 5
	for i := 0; i < itemCount; i++ {
		p.Inventory.RestoreItem(game.NewObjectInstance(proto, -1))
	}
	if got := len(p.Inventory.FindItems("")); got != itemCount {
		t.Fatalf("setup: inventory has %d items, want %d", got, itemCount)
	}

	rec, err := PlayerToRecord(p, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord failed: %v", err)
	}
	restored, err := RecordToPlayer(rec, world)
	if err != nil {
		t.Fatalf("RecordToPlayer failed: %v", err)
	}

	if got := len(restored.Inventory.FindItems("")); got != itemCount {
		t.Errorf("restored inventory has %d items, want %d (items dropped on load)", got, itemCount)
	}
}

// A rented container keeps what is inside it, nested containers included,
// whether it is carried or worn (DP-1324; C Crash_save/Crash_load, objsave.c).
func TestContainerContentsSurviveRecordRoundTrip(t *testing.T) {
	world := newConversionWorld(t, []parser.Obj{
		{VNum: 8032, ShortDesc: "a backpack", Keywords: "backpack", TypeFlag: game.ITEM_CONTAINER, Weight: 1},
		{VNum: 8033, ShortDesc: "a small sack", Keywords: "sack", TypeFlag: game.ITEM_CONTAINER, Weight: 1},
		{VNum: 8010, ShortDesc: "a loaf of bread", Keywords: "bread", TypeFlag: game.ITEM_FOOD, Weight: 2},
		{VNum: 8063, ShortDesc: "a water skin", Keywords: "skin", TypeFlag: game.ITEM_OTHER, Weight: 10},
		{VNum: 8040, ShortDesc: "a belt pouch", Keywords: "pouch", TypeFlag: game.ITEM_CONTAINER, Weight: 1},
		{VNum: 8041, ShortDesc: "a copper ring", Keywords: "ring", TypeFlag: game.ITEM_OTHER, Weight: 1},
	})
	spawn := func(vnum int) *game.ObjectInstance {
		t.Helper()
		obj, err := world.SpawnObject(vnum, -1)
		if err != nil {
			t.Fatal(err)
		}
		return obj
	}
	p, err := RecordToPlayer(conversionRecord("[]"), world)
	if err != nil {
		t.Fatal(err)
	}
	backpack, sack := spawn(8032), spawn(8033)
	sack.Contains = []*game.ObjectInstance{spawn(8010)}
	backpack.Contains = []*game.ObjectInstance{sack, spawn(8063)}
	p.Inventory.RestoreItem(backpack)
	pouch := spawn(8040)
	pouch.Contains = []*game.ObjectInstance{spawn(8041)}
	p.Equipment.Slots[game.SlotAbout] = pouch

	rec, err := PlayerToRecord(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	back, err := RecordToPlayer(rec, world)
	if err != nil {
		t.Fatal(err)
	}

	carried := back.Inventory.FindItems("")
	if len(carried) != 1 || carried[0].VNum != 8032 {
		t.Fatalf("carried = %v, want just the backpack", vnums(carried))
	}
	gotPack := carried[0]
	if got := vnums(gotPack.Contains); len(got) != 2 || got[0] != 8033 || got[1] != 8063 {
		t.Fatalf("backpack holds %v, want [8033 8063] in saved order", got)
	}
	gotSack := gotPack.Contains[0]
	if got := vnums(gotSack.Contains); len(got) != 1 || got[0] != 8010 {
		t.Fatalf("sack holds %v, want [8010]", got)
	}
	if gotPack.GetTotalWeight() != 14 {
		t.Fatalf("backpack total weight = %d, want 14", gotPack.GetTotalWeight())
	}
	worn := back.Equipment.Slots[game.SlotAbout]
	if worn == nil || worn.VNum != 8040 || len(worn.Contains) != 1 || worn.Contains[0].VNum != 8041 {
		t.Fatalf("worn pouch = %+v, want pouch holding the ring", worn)
	}

	// Restored objects are world-registered, so a container can be found by
	// the ID its contents point at.
	for _, c := range []*game.ObjectInstance{gotPack, gotSack, worn} {
		if c.ID == 0 {
			t.Fatalf("restored container %d has no world ID", c.VNum)
		}
		for _, inside := range c.Contains {
			if inside.Location != game.LocContainer(c.ID) {
				t.Fatalf("object %d location = %+v, want inside container %d", inside.VNum, inside.Location, c.ID)
			}
		}
	}
}

func vnums(objs []*game.ObjectInstance) []int {
	out := make([]int, 0, len(objs))
	for _, o := range objs {
		out = append(out, o.VNum)
	}
	return out
}
