package olc

import (
	"fmt"
	"strconv"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// The limits are the C OLC limits from src/olc.h. They are exported so every
// frontend can present the same effective values as the telnet editor.
const (
	MaxRoomName  = 75
	MaxMobName   = 50
	MaxObjName   = 50
	MaxRoomDesc  = 1024
	MaxExitDesc  = 256
	MaxExtraDesc = 1024
	MaxMobDesc   = 1024
	MaxMessage   = 4096
)

// OperationKind identifies one semantic OLC write. An operation is ordered:
// callers apply a list in order, so a cascade and a later explicit write have
// the same meaning in every frontend.
type OperationKind uint8

const (
	OpClampInt OperationKind = iota + 1

	OpSetRoomName
	OpSetRoomDescription
	OpSetRoomFlag
	OpSetRoomSector
	OpEnsureExit
	OpSetExitTarget
	OpSetExitDescription
	OpSetExitKeywords
	OpSetExitKey
	OpSetExitDoorFlags
	OpPurgeExit
	OpAddExtraDescription
	OpRemoveExtraDescription
	OpSetExtraKeywords
	OpSetExtraDescription
	OpCopyRoom

	OpSetMobKeywords
	OpSetMobShortDescription
	OpSetMobLongDescription
	OpSetMobDetailedDescription
	OpSetMobSex
	OpSetMobHitroll
	OpSetMobDamroll
	OpSetMobNumDamageDice
	OpSetMobSizeDamageDice
	OpSetMobNumHPDice
	OpSetMobSizeHPDice
	OpSetMobAddHP
	OpSetMobAC
	OpSetMobExp
	OpSetMobGold
	OpSetMobPosition
	OpSetMobDefaultPosition
	OpSetMobAttack
	OpSetMobLevel
	OpSetMobAlignment
	OpSetMobRace

	OpSetObjKeywords
	OpSetObjShortDescription
	OpSetObjLongDescription
	OpSetObjActionDescription
	OpSetObjExtraKeywords
	OpSetObjExtraDescription
	OpSetObjAffect
	OpAddObjAffect
	OpRemoveObjAffect
	OpAddObjExtraDescription
	OpRemoveObjExtraDescription
	OpSetObjValue1
	OpSetObjValue2
	OpSetObjValue3
	OpSetObjValue4
	OpToggleObjContainerFlag

	OpAddShopProduct
	OpRemoveShopProduct
	OpSetShopBuyProfit
	OpSetShopSellProfit
	OpSetShopKeeper
	OpSetShopFlags
	OpSetShopWithWho
	OpAddShopRoom
	OpRemoveShopRoom
	OpSetShopOpenHour1
	OpSetShopOpenHour2
	OpSetShopCloseHour1
	OpSetShopCloseHour2

	OpSetZoneTopRoom
	OpSetZoneName
	OpSetZoneLifespan
	OpSetZoneResetMode
	OpAddZoneCommand
	OpModifyZoneCommand
	OpRemoveZoneCommand
	OpReorderZoneCommand
)

// Operation is the transport-free typed input to Apply. Only the target
// appropriate for Kind is used. Result is used by OpClampInt for menu values
// that are not fields in the working copy (for example SEDIT's selected type).
type Operation struct {
	Kind      OperationKind
	Value     int
	Low       int
	High      int
	Text      string
	Result    *int
	Bit       int
	Index     int
	Direction string

	Room       *parser.Room
	Exit       *parser.Exit
	Extra      *parser.ExtraDesc
	Source     *parser.Room
	RoomExists func(int) bool
	Mob        *parser.Mob
	Obj        *parser.Obj
	Affect     *parser.ObjAffect
	Affects    *[]parser.ObjAffect
	Shop       *parser.ShopProto
	Zone       *parser.Zone
	Command    *parser.ZoneCommand
	ToIndex    int
}

// Apply performs one C-semantics OLC write. It intentionally mutates the
// caller's working copy: the ordered operation list is the shared core that
// both telnet and future web controllers will drive.
func Apply(op Operation) error {
	switch op.Kind {
	case OpClampInt:
		if op.Result == nil {
			return fmt.Errorf("olc clamp requires a result")
		}
		*op.Result = clampInt(op.Value, op.Low, op.High)
	case OpSetRoomName:
		if op.Room == nil {
			return fmt.Errorf("room name operation requires a room")
		}
		// REDIT's C path writes arg[MAX_ROOM_NAME-1] = '\0', so the
		// maximum stored name is one byte shorter than the macro value.
		op.Room.Name = truncateBytes(op.Text, MaxRoomName-1)
	case OpSetRoomDescription:
		if op.Room == nil {
			return fmt.Errorf("room description operation requires a room")
		}
		op.Room.Description = truncateBytes(op.Text, MaxRoomDesc)
	case OpSetRoomFlag:
		if op.Room == nil {
			return fmt.Errorf("room flag operation requires a room")
		}
		if op.Bit < 0 || op.Bit >= 28 {
			return fmt.Errorf("room flag operation has invalid bit %d", op.Bit)
		}
		for len(op.Room.Flags) < 4 {
			op.Room.Flags = append(op.Room.Flags, "0")
		}
		word := op.Bit / 32
		bit := uint(op.Bit % 32)
		value, err := parseFlagWord(op.Room.Flags[word])
		if err != nil {
			value = 0
		}
		if op.Value == 0 {
			value &^= 1 << bit
		} else {
			value |= 1 << bit
		}
		op.Room.Flags[word] = formatFlagWord(value)
	case OpSetRoomSector:
		if op.Room == nil {
			return fmt.Errorf("room sector operation requires a room")
		}
		if op.Value < 0 || op.Value >= 16 {
			return fmt.Errorf("room sector operation has invalid value %d", op.Value)
		}
		op.Room.Sector = op.Value
	case OpEnsureExit:
		if op.Room == nil {
			return fmt.Errorf("ensure exit operation requires a room")
		}
		if op.Direction == "" {
			return fmt.Errorf("ensure exit operation requires a direction")
		}
		if op.Room.Exits == nil {
			op.Room.Exits = make(map[string]parser.Exit)
		}
		if _, ok := op.Room.Exits[op.Direction]; !ok {
			op.Room.Exits[op.Direction] = parser.Exit{Direction: op.Direction, ToRoom: op.Value}
		}
	case OpSetExitTarget:
		if op.Room == nil {
			return fmt.Errorf("exit target operation requires a room")
		}
		if op.Direction == "" {
			return fmt.Errorf("exit target operation requires a direction")
		}
		if op.Value != -1 && op.RoomExists != nil && !op.RoomExists(op.Value) {
			return fmt.Errorf("exit target room %d does not exist", op.Value)
		}
		exit := ensureRoomExit(op.Room, op.Direction)
		exit.ToRoom = op.Value
		op.Room.Exits[op.Direction] = exit
	case OpSetExitDescription:
		if op.Exit == nil && op.Room == nil {
			return fmt.Errorf("exit description operation requires an exit")
		}
		if op.Room != nil && op.Direction == "" && op.Exit == nil {
			return fmt.Errorf("exit description operation requires a direction")
		}
		if op.Exit != nil {
			op.Exit.Description = truncateBytes(op.Text, MaxExitDesc)
		} else {
			exit := ensureRoomExit(op.Room, op.Direction)
			exit.Description = truncateBytes(op.Text, MaxExitDesc)
			op.Room.Exits[op.Direction] = exit
		}
	case OpSetExitKeywords:
		if op.Room == nil {
			return fmt.Errorf("exit keywords operation requires a room")
		}
		if op.Direction == "" {
			return fmt.Errorf("exit keywords operation requires a direction")
		}
		exit := ensureRoomExit(op.Room, op.Direction)
		exit.Keywords = op.Text
		op.Room.Exits[op.Direction] = exit
	case OpSetExitKey:
		if op.Room == nil {
			return fmt.Errorf("exit key operation requires a room")
		}
		if op.Direction == "" {
			return fmt.Errorf("exit key operation requires a direction")
		}
		exit := ensureRoomExit(op.Room, op.Direction)
		exit.Key = op.Value
		op.Room.Exits[op.Direction] = exit
	case OpSetExitDoorFlags:
		if op.Room == nil {
			return fmt.Errorf("exit door flags operation requires a room")
		}
		if op.Direction == "" {
			return fmt.Errorf("exit door flags operation requires a direction")
		}
		if op.Value < 0 || op.Value > 2 {
			return fmt.Errorf("exit door flags operation has invalid value %d", op.Value)
		}
		exit := ensureRoomExit(op.Room, op.Direction)
		switch op.Value {
		case 0:
			exit.ExitInfo = 0
			exit.DoorState = 0
		case 1:
			exit.ExitInfo = parser.ExitIsDoor
			exit.DoorState = 1
		case 2:
			exit.ExitInfo = parser.ExitIsDoor | parser.ExitPickproof
			exit.DoorState = 2
		}
		op.Room.Exits[op.Direction] = exit
	case OpPurgeExit:
		if op.Room == nil {
			return fmt.Errorf("purge exit operation requires a room")
		}
		if op.Direction == "" {
			return fmt.Errorf("purge exit operation requires a direction")
		}
		delete(op.Room.Exits, op.Direction)
	case OpAddExtraDescription:
		if op.Room == nil {
			return fmt.Errorf("add extra description operation requires a room")
		}
		extra := parser.ExtraDesc{}
		if op.Extra != nil {
			extra = *op.Extra
		}
		extra.Description = truncateBytes(extra.Description, MaxExtraDesc)
		if op.Index < 0 || op.Index >= len(op.Room.ExtraDescs) {
			op.Room.ExtraDescs = append(op.Room.ExtraDescs, extra)
		} else {
			op.Room.ExtraDescs = append(op.Room.ExtraDescs, parser.ExtraDesc{})
			copy(op.Room.ExtraDescs[op.Index+1:], op.Room.ExtraDescs[op.Index:])
			op.Room.ExtraDescs[op.Index] = extra
		}
	case OpRemoveExtraDescription:
		if op.Room == nil {
			return fmt.Errorf("remove extra description operation requires a room")
		}
		if op.Index < 0 || op.Index >= len(op.Room.ExtraDescs) {
			return fmt.Errorf("extra description index %d is out of range", op.Index)
		}
		copy(op.Room.ExtraDescs[op.Index:], op.Room.ExtraDescs[op.Index+1:])
		op.Room.ExtraDescs = op.Room.ExtraDescs[:len(op.Room.ExtraDescs)-1]
	case OpSetExtraKeywords:
		if op.Room == nil {
			return fmt.Errorf("extra keywords operation requires a room")
		}
		if op.Index < 0 || op.Index >= len(op.Room.ExtraDescs) {
			return fmt.Errorf("extra description index %d is out of range", op.Index)
		}
		op.Room.ExtraDescs[op.Index].Keywords = op.Text
	case OpSetExtraDescription:
		if op.Extra == nil && op.Room == nil {
			return fmt.Errorf("extra description operation requires an extra description")
		}
		if op.Extra != nil {
			op.Extra.Description = truncateBytes(op.Text, MaxExtraDesc)
		} else if op.Index >= 0 && op.Index < len(op.Room.ExtraDescs) {
			op.Room.ExtraDescs[op.Index].Description = truncateBytes(op.Text, MaxExtraDesc)
		} else {
			return fmt.Errorf("extra description index %d is out of range", op.Index)
		}
	case OpCopyRoom:
		if op.Room == nil || op.Source == nil {
			return fmt.Errorf("room copy operation requires source and destination rooms")
		}
		op.Room.Name = truncateBytes(op.Source.Name, MaxRoomName-1)
		op.Room.Description = truncateBytes(op.Source.Description, MaxRoomDesc)

	case OpSetMobKeywords:
		if op.Mob == nil {
			return fmt.Errorf("mob keywords operation requires a mob")
		}
		op.Mob.Keywords = truncateBytes(op.Text, MaxMobName)
	case OpSetMobShortDescription:
		if op.Mob == nil {
			return fmt.Errorf("mob short description operation requires a mob")
		}
		op.Mob.ShortDesc = truncateBytes(op.Text, MaxMobName)
	case OpSetMobLongDescription:
		if op.Mob == nil {
			return fmt.Errorf("mob long description operation requires a mob")
		}
		op.Mob.LongDesc = truncateBytes(op.Text, MaxMobName) + "\r\n"
	case OpSetMobDetailedDescription:
		if op.Mob == nil {
			return fmt.Errorf("mob detailed description operation requires a mob")
		}
		op.Mob.DetailedDesc = truncateBytes(op.Text, MaxMobDesc)
	case OpSetMobSex:
		if op.Mob == nil {
			return fmt.Errorf("mob sex operation requires a mob")
		}
		op.Mob.Sex = clampInt(op.Value, 0, 2)
	case OpSetMobHitroll:
		if op.Mob == nil {
			return fmt.Errorf("mob hitroll operation requires a mob")
		}
		op.Mob.THAC0 = 20 - clampInt(op.Value, 0, 127)
	case OpSetMobDamroll:
		if op.Mob == nil {
			return fmt.Errorf("mob damroll operation requires a mob")
		}
		op.Mob.Damage.Plus = clampInt(op.Value, 0, 127)
	case OpSetMobNumDamageDice:
		if op.Mob == nil {
			return fmt.Errorf("mob damage dice operation requires a mob")
		}
		op.Mob.Damage.Num = clampInt(op.Value, 0, 127)
	case OpSetMobSizeDamageDice:
		if op.Mob == nil {
			return fmt.Errorf("mob damage sides operation requires a mob")
		}
		op.Mob.Damage.Sides = clampInt(op.Value, 0, 127)
	case OpSetMobNumHPDice:
		if op.Mob == nil {
			return fmt.Errorf("mob hp dice operation requires a mob")
		}
		op.Mob.HP.Num = clampInt(op.Value, 0, 50)
	case OpSetMobSizeHPDice:
		if op.Mob == nil {
			return fmt.Errorf("mob hp sides operation requires a mob")
		}
		op.Mob.HP.Sides = clampInt(op.Value, 0, 3000)
	case OpSetMobAddHP:
		if op.Mob == nil {
			return fmt.Errorf("mob hp plus operation requires a mob")
		}
		op.Mob.HP.Plus = clampInt(op.Value, 0, 30000)
	case OpSetMobAC:
		if op.Mob == nil {
			return fmt.Errorf("mob ac operation requires a mob")
		}
		op.Mob.AC = clampInt(op.Value, -200, 200)
	case OpSetMobExp:
		if op.Mob == nil {
			return fmt.Errorf("mob exp operation requires a mob")
		}
		op.Mob.Exp = clampInt(op.Value, 0, int(^uint(0)>>1))
	case OpSetMobGold:
		if op.Mob == nil {
			return fmt.Errorf("mob gold operation requires a mob")
		}
		op.Mob.Gold = clampInt(op.Value, 0, int(^uint(0)>>1))
	case OpSetMobPosition:
		if op.Mob == nil {
			return fmt.Errorf("mob position operation requires a mob")
		}
		op.Mob.Position = clampInt(op.Value, 0, 14)
	case OpSetMobDefaultPosition:
		if op.Mob == nil {
			return fmt.Errorf("mob default position operation requires a mob")
		}
		op.Mob.DefaultPos = clampInt(op.Value, 0, 14)
	case OpSetMobAttack:
		if op.Mob == nil {
			return fmt.Errorf("mob attack operation requires a mob")
		}
		op.Mob.BareHandAttack = clampInt(op.Value, 0, 14)
	case OpSetMobLevel:
		if op.Mob == nil {
			return fmt.Errorf("mob level operation requires a mob")
		}
		applyMobLevel(op.Mob, op.Value)
	case OpSetMobAlignment:
		if op.Mob == nil {
			return fmt.Errorf("mob alignment operation requires a mob")
		}
		op.Mob.Alignment = clampInt(op.Value, -1000, 1000)
	case OpSetMobRace:
		if op.Mob == nil {
			return fmt.Errorf("mob race operation requires a mob")
		}
		op.Mob.Race = clampInt(op.Value, 0, 30)

	case OpSetObjKeywords:
		if op.Obj == nil {
			return fmt.Errorf("object keywords operation requires an object")
		}
		op.Obj.Keywords = truncateBytes(op.Text, MaxObjName)
	case OpSetObjShortDescription:
		if op.Obj == nil {
			return fmt.Errorf("object short description operation requires an object")
		}
		op.Obj.ShortDesc = truncateBytes(op.Text, MaxObjName)
	case OpSetObjLongDescription:
		if op.Obj == nil {
			return fmt.Errorf("object long description operation requires an object")
		}
		op.Obj.LongDesc = truncateBytes(op.Text, MaxObjName)
	case OpSetObjActionDescription:
		if op.Obj == nil {
			return fmt.Errorf("object action description operation requires an object")
		}
		op.Obj.ActionDesc = truncateBytes(op.Text, MaxMessage)
	case OpSetObjExtraKeywords:
		if op.Extra == nil && op.Obj == nil {
			return fmt.Errorf("object extra keywords operation requires an extra description")
		}
		if op.Extra != nil {
			op.Extra.Keywords = op.Text
		} else if op.Index >= 0 && op.Index < len(op.Obj.ExtraDescs) {
			op.Obj.ExtraDescs[op.Index].Keywords = op.Text
		} else {
			return fmt.Errorf("object extra description index %d is out of range", op.Index)
		}
	case OpSetObjExtraDescription:
		if op.Extra == nil && op.Obj == nil {
			return fmt.Errorf("object extra description operation requires an extra description")
		}
		if op.Extra != nil {
			op.Extra.Description = truncateBytes(op.Text, MaxExtraDesc)
		} else if op.Index >= 0 && op.Index < len(op.Obj.ExtraDescs) {
			op.Obj.ExtraDescs[op.Index].Description = truncateBytes(op.Text, MaxExtraDesc)
		} else {
			return fmt.Errorf("object extra description index %d is out of range", op.Index)
		}
	case OpAddObjAffect:
		if op.Obj == nil {
			return fmt.Errorf("add object affect operation requires an object")
		}
		if len(op.Obj.Affects) >= parser.MAX_OBJ_AFFECT {
			return fmt.Errorf("object affect limit %d exceeded", parser.MAX_OBJ_AFFECT)
		}
		affect := parser.ObjAffect{}
		if op.Affect != nil {
			affect = *op.Affect
		}
		if op.Index < 0 || op.Index >= len(op.Obj.Affects) {
			op.Obj.Affects = append(op.Obj.Affects, affect)
		} else {
			op.Obj.Affects = append(op.Obj.Affects, parser.ObjAffect{})
			copy(op.Obj.Affects[op.Index+1:], op.Obj.Affects[op.Index:])
			op.Obj.Affects[op.Index] = affect
		}
	case OpRemoveObjAffect:
		if op.Obj == nil {
			return fmt.Errorf("remove object affect operation requires an object")
		}
		if op.Index < 0 || op.Index >= len(op.Obj.Affects) {
			return fmt.Errorf("object affect index %d is out of range", op.Index)
		}
		copy(op.Obj.Affects[op.Index:], op.Obj.Affects[op.Index+1:])
		op.Obj.Affects = op.Obj.Affects[:len(op.Obj.Affects)-1]
	case OpSetObjAffect:
		if op.Affects == nil || op.Affect == nil {
			return fmt.Errorf("set object affect operation requires affect storage and value")
		}
		if op.Index < 0 || op.Index >= parser.MAX_OBJ_AFFECT {
			return fmt.Errorf("object affect index %d is out of range", op.Index)
		}
		for len(*op.Affects) <= op.Index {
			*op.Affects = append(*op.Affects, parser.ObjAffect{})
		}
		(*op.Affects)[op.Index] = *op.Affect
	case OpAddObjExtraDescription:
		if op.Obj == nil {
			return fmt.Errorf("add object extra description operation requires an object")
		}
		extra := parser.ExtraDesc{}
		if op.Extra != nil {
			extra = *op.Extra
		}
		extra.Description = truncateBytes(extra.Description, MaxExtraDesc)
		if op.Index < 0 || op.Index >= len(op.Obj.ExtraDescs) {
			op.Obj.ExtraDescs = append(op.Obj.ExtraDescs, extra)
		} else {
			op.Obj.ExtraDescs = append(op.Obj.ExtraDescs, parser.ExtraDesc{})
			copy(op.Obj.ExtraDescs[op.Index+1:], op.Obj.ExtraDescs[op.Index:])
			op.Obj.ExtraDescs[op.Index] = extra
		}
	case OpRemoveObjExtraDescription:
		if op.Obj == nil {
			return fmt.Errorf("remove object extra description operation requires an object")
		}
		if op.Index < 0 || op.Index >= len(op.Obj.ExtraDescs) {
			return fmt.Errorf("object extra description index %d is out of range", op.Index)
		}
		copy(op.Obj.ExtraDescs[op.Index:], op.Obj.ExtraDescs[op.Index+1:])
		op.Obj.ExtraDescs = op.Obj.ExtraDescs[:len(op.Obj.ExtraDescs)-1]
	case OpSetObjValue1, OpSetObjValue2, OpSetObjValue3, OpSetObjValue4:
		if op.Obj == nil {
			return fmt.Errorf("object value operation requires an object")
		}
		index := int(op.Kind - OpSetObjValue1)
		low, high := objectValueBounds(op.Obj.TypeFlag, index)
		if op.Low != 0 || op.High != 0 {
			low, high = op.Low, op.High
		}
		if low > high {
			return fmt.Errorf("object value operation has invalid bounds %d..%d", low, high)
		}
		op.Obj.Values[index] = clampInt(op.Value, low, high)
	case OpToggleObjContainerFlag:
		if op.Obj == nil {
			return fmt.Errorf("object container flag operation requires an object")
		}
		if op.Obj.TypeFlag != 15 || op.Value < 0 || op.Value > 3 {
			return fmt.Errorf("object container flag %d is invalid", op.Value)
		}
		op.Obj.Values[1] ^= 1 << uint(op.Value)

	case OpAddShopProduct:
		if op.Shop == nil {
			return fmt.Errorf("add shop product operation requires a shop")
		}
		if op.Value < 0 {
			return nil // -1 is the C list terminator, never a stored product.
		}
		if op.Index < 0 || op.Index >= len(op.Shop.Products) {
			op.Shop.Products = append(op.Shop.Products, op.Value)
		} else {
			op.Shop.Products = append(op.Shop.Products, 0)
			copy(op.Shop.Products[op.Index+1:], op.Shop.Products[op.Index:])
			op.Shop.Products[op.Index] = op.Value
		}
	case OpRemoveShopProduct:
		if op.Shop == nil {
			return fmt.Errorf("remove shop product operation requires a shop")
		}
		if op.Index < 0 || op.Index >= len(op.Shop.Products) {
			return fmt.Errorf("shop product index %d is out of range", op.Index)
		}
		copy(op.Shop.Products[op.Index:], op.Shop.Products[op.Index+1:])
		op.Shop.Products = op.Shop.Products[:len(op.Shop.Products)-1]
	case OpSetShopBuyProfit, OpSetShopSellProfit, OpSetShopKeeper, OpSetShopFlags, OpSetShopWithWho,
		OpAddShopRoom, OpRemoveShopRoom:
		if op.Shop == nil {
			return fmt.Errorf("shop operation requires a shop")
		}
		switch op.Kind {
		case OpSetShopBuyProfit:
			op.Shop.BuyProfit = float64(op.Value) / 100.0
		case OpSetShopSellProfit:
			op.Shop.SellProfit = float64(op.Value) / 100.0
		case OpSetShopKeeper:
			op.Shop.KeeperVNum = op.Value
		case OpSetShopFlags:
			op.Shop.Bitvector = op.Value
		case OpSetShopWithWho:
			op.Shop.WithWho = op.Value
		case OpAddShopRoom:
			if op.Value >= 0 {
				op.Shop.Rooms = append(op.Shop.Rooms, op.Value)
			}
		case OpRemoveShopRoom:
			if op.Index < 0 || op.Index >= len(op.Shop.Rooms) {
				return fmt.Errorf("shop room index %d is out of range", op.Index)
			}
			copy(op.Shop.Rooms[op.Index:], op.Shop.Rooms[op.Index+1:])
			op.Shop.Rooms = op.Shop.Rooms[:len(op.Shop.Rooms)-1]
		}
	case OpSetShopOpenHour1, OpSetShopOpenHour2, OpSetShopCloseHour1, OpSetShopCloseHour2:
		if op.Shop == nil {
			return fmt.Errorf("shop hour operation requires a shop")
		}
		value := clampInt(op.Value, 0, 28)
		switch op.Kind {
		case OpSetShopOpenHour1:
			op.Shop.OpenHour1 = value
		case OpSetShopOpenHour2:
			op.Shop.OpenHour2 = value
		case OpSetShopCloseHour1:
			op.Shop.CloseHour1 = value
		case OpSetShopCloseHour2:
			op.Shop.CloseHour2 = value
		}
	case OpSetZoneTopRoom:
		if op.Zone == nil {
			return fmt.Errorf("zone top operation requires a zone")
		}
		if op.Low > op.High {
			return fmt.Errorf("zone top operation has invalid bounds %d..%d", op.Low, op.High)
		}
		op.Zone.TopRoom = clampInt(op.Value, op.Low, op.High)
	case OpSetZoneName:
		if op.Zone == nil {
			return fmt.Errorf("zone name operation requires a zone")
		}
		op.Zone.Name = truncateBytes(op.Text, MaxRoomName-1)
	case OpSetZoneLifespan:
		if op.Zone == nil {
			return fmt.Errorf("zone lifespan operation requires a zone")
		}
		op.Zone.Lifespan = clampInt(op.Value, 0, 240)
	case OpSetZoneResetMode:
		if op.Zone == nil {
			return fmt.Errorf("zone reset mode operation requires a zone")
		}
		op.Zone.ResetMode = clampInt(op.Value, 0, 2)
	case OpAddZoneCommand:
		if op.Zone == nil || op.Command == nil {
			return fmt.Errorf("add zone command operation requires a zone and command")
		}
		command := *op.Command
		if op.Index < 0 || op.Index >= len(op.Zone.Commands) {
			op.Zone.Commands = append(op.Zone.Commands, command)
		} else {
			op.Zone.Commands = append(op.Zone.Commands, parser.ZoneCommand{})
			copy(op.Zone.Commands[op.Index+1:], op.Zone.Commands[op.Index:])
			op.Zone.Commands[op.Index] = command
		}
	case OpModifyZoneCommand:
		if op.Zone == nil || op.Command == nil {
			return fmt.Errorf("modify zone command operation requires a zone and command")
		}
		if op.Index < 0 || op.Index >= len(op.Zone.Commands) {
			return fmt.Errorf("zone command index %d is out of range", op.Index)
		}
		op.Zone.Commands[op.Index] = *op.Command
	case OpRemoveZoneCommand:
		if op.Zone == nil {
			return fmt.Errorf("remove zone command operation requires a zone")
		}
		if op.Index < 0 || op.Index >= len(op.Zone.Commands) {
			return fmt.Errorf("zone command index %d is out of range", op.Index)
		}
		copy(op.Zone.Commands[op.Index:], op.Zone.Commands[op.Index+1:])
		op.Zone.Commands = op.Zone.Commands[:len(op.Zone.Commands)-1]
	case OpReorderZoneCommand:
		if op.Zone == nil {
			return fmt.Errorf("reorder zone command operation requires a zone")
		}
		if op.Index < 0 || op.Index >= len(op.Zone.Commands) || op.ToIndex < 0 || op.ToIndex >= len(op.Zone.Commands) {
			return fmt.Errorf("zone command reorder %d -> %d is out of range", op.Index, op.ToIndex)
		}
		if op.Index != op.ToIndex {
			command := op.Zone.Commands[op.Index]
			if op.Index < op.ToIndex {
				copy(op.Zone.Commands[op.Index:op.ToIndex], op.Zone.Commands[op.Index+1:op.ToIndex+1])
			} else {
				copy(op.Zone.Commands[op.ToIndex+1:op.Index+1], op.Zone.Commands[op.ToIndex:op.Index])
			}
			op.Zone.Commands[op.ToIndex] = command
		}
	default:
		return fmt.Errorf("unknown OLC operation %d", op.Kind)
	}
	return nil
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func objectValueBounds(typeFlag, index int) (int, int) {
	const (
		itemScroll    = 2
		itemWand      = 3
		itemStaff     = 4
		itemWeapon    = 5
		itemPotion    = 10
		itemContainer = 15
		itemDrinkcon  = 17
		itemFountain  = 23
	)
	const numSpells, numLiqTypes, numAttackTypes = 104, 16, 15
	switch index {
	case 0:
		return -32000, 32000
	case 1:
		if typeFlag == itemScroll || typeFlag == itemPotion {
			return 0, numSpells - 1
		}
		if typeFlag == itemContainer {
			return 0, 15
		}
	case 2:
		switch typeFlag {
		case itemScroll, itemPotion:
			return 0, numSpells - 1
		case itemWeapon, itemWand, itemStaff:
			return 0, 20
		case itemDrinkcon, itemFountain:
			return 0, numLiqTypes - 1
		}
	case 3:
		switch typeFlag {
		case itemScroll, itemPotion:
			return 0, numSpells - 1
		case itemWand, itemStaff:
			return 1, numSpells - 1
		case itemWeapon:
			return 0, numAttackTypes - 1
		}
	}
	return -32000, 32000
}

func truncateBytes(value string, limit int) string {
	if limit < 0 {
		limit = 0
	}
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

func parseFlagWord(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

func formatFlagWord(value uint32) string {
	return strconv.FormatUint(uint64(value), 10)
}

func ensureRoomExit(room *parser.Room, direction string) parser.Exit {
	if room.Exits == nil {
		room.Exits = make(map[string]parser.Exit)
	}
	if exit, ok := room.Exits[direction]; ok {
		return exit
	}
	return parser.Exit{Direction: direction}
}

// This is the C EXP_LOOKUP table from medit.c, including the documented
// bounded lookup used by the Go port for levels above the table.
var meditExpLookup = []int{
	25,
	100, 200, 350, 600, 900,
	1500, 2500, 3500, 4500, 6000,
	7500, 9000, 10500, 12000, 14000,
	16000, 18000, 20000, 22500, 25000,
	27500, 30000, 32500, 35000, 37500,
	40000, 45000, 50000, 55000, 60000,
	90000, 120000, 180000, 270000, 360000,
	363000, 369000, 372000, 375000, 400000,
}

// applyMobLevel is the MEDIT_LEVEL ten-field cascade. The final level write
// intentionally has a zero floor, matching C's "set it again to handle 0s".
func applyMobLevel(mob *parser.Mob, raw int) {
	level := clampInt(raw, 1, 100)
	mob.Level = level

	if level <= len(meditExpLookup)-1 {
		mob.Exp = meditExpLookup[level]
	} else {
		// C indexes EXP_LOOKUP out of bounds for levels 41-100. The Go port
		// clamps the lookup instead of reproducing that undefined read.
		mob.Exp = meditExpLookup[len(meditExpLookup)-1]
	}

	if level > 10 {
		mob.Damage.Num = int(float64(level) / 1.50)
	} else {
		mob.Damage.Num = (level + 1) / 2
	}
	mob.Damage.Sides = 4
	if level > 10 {
		mob.Damage.Plus = int(float64(level+1) / 1.50)
	} else {
		mob.Damage.Plus = (level + 1) / 2
	}
	mob.HP.Num = level
	mob.HP.Sides = 5
	mob.HP.Plus = 10*level + 10
	if level > 22 {
		mob.HP.Plus += 13 * (level - 22)
	}
	if level > 30 {
		mob.HP.Plus += 560 * (level - 30)
	}
	mob.AC = 100 - (10 * level)
	mob.THAC0 = 20 - level
	mob.Level = clampInt(raw, 0, 100)
}
