package olc

import "github.com/zax0rz/darkpawns/pkg/parser"

// Bounds is an inclusive numeric range enforced by the shared OLC operation.
type Bounds struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// SchemaField is one schema-driven editor control. Options are numbered by
// the values accepted by the corresponding OLC operation.
type SchemaField struct {
	Key     string            `json:"key"`
	Label   string            `json:"label"`
	Control string            `json:"control"`
	Bounds  *Bounds           `json:"bounds,omitempty"`
	Options []VocabularyEntry `json:"options,omitempty"`
}

// SchemaAction describes a gated action exposed by an editor surface. The
// browser renders the requirement from this descriptor; it never carries a
// copy of the level ladder.
type SchemaAction struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	RequiredLevel int    `json:"required_level"`
	RequiredLabel string `json:"required_label"`
	Allowed       bool   `json:"allowed"`
}

// ZoneCommandArgument describes one of the three type-dependent reset
// command arguments. A hidden argument is still present in the descriptor so
// the command editor can render from one stable three-slot shape.
type ZoneCommandArgument struct {
	Key     string            `json:"key"`
	Label   string            `json:"label"`
	Control string            `json:"control"`
	Bounds  *Bounds           `json:"bounds,omitempty"`
	Options []VocabularyEntry `json:"options,omitempty"`
	Visible bool              `json:"visible"`
}

type ZoneCommandDescriptor struct {
	Command   string                 `json:"command"`
	Label     string                 `json:"label"`
	Arguments [3]ZoneCommandArgument `json:"arguments"`
}

// ObjectValueField is a resolved row of the object value matrix. The browser
// receives a concrete row for every type and slot; it never evaluates a
// visibility expression.
type ObjectValueField struct {
	Label   string            `json:"label"`
	Control string            `json:"control"`
	Min     int               `json:"min"`
	Max     int               `json:"max"`
	Visible bool              `json:"visible"`
	Options []VocabularyEntry `json:"options,omitempty"`
}

type ObjectValueMatrixEntry struct {
	ItemType      int                 `json:"item_type"`
	ItemTypeLabel string              `json:"item_type_label"`
	Values        [4]ObjectValueField `json:"values"`
}

// AppliesDescriptor describes the bounded object-affect list and the semantic
// operations that mutate it. The browser does not need to duplicate the
// MAX_OBJ_AFFECT limit or invent operation names.
type AppliesDescriptor struct {
	Max             int               `json:"max"`
	Options         []VocabularyEntry `json:"options"`
	AddOperation    string            `json:"add_operation"`
	RemoveOperation string            `json:"remove_operation"`
}

// ExitDescriptor contains the room exit vocabulary shared by telnet and the
// bespoke web exit editor.
type ExitDescriptor struct {
	Directions  []string          `json:"directions"`
	DoorOptions []VocabularyEntry `json:"door_options"`
}

// Schema is the complete vocabulary projection for one OLC editor.
type Schema struct {
	Kind         string                   `json:"kind"`
	Fields       []SchemaField            `json:"fields"`
	Bespoke      []string                 `json:"bespoke,omitempty"`
	Actions      []SchemaAction           `json:"actions,omitempty"`
	ValueMatrix  []ObjectValueMatrixEntry `json:"value_matrix,omitempty"`
	Applies      *AppliesDescriptor       `json:"applies,omitempty"`
	Exits        *ExitDescriptor          `json:"exits,omitempty"`
	ZoneCommands []ZoneCommandDescriptor  `json:"zone_commands,omitempty"`
}

const (
	MobSexMin                      = 0
	MobSexMax                      = 2
	MobHitrollMin                  = 0
	MobHitrollMax                  = 127
	MobDamrollMin                  = 0
	MobDamrollMax                  = 127
	MobDamageDiceMin               = 0
	MobDamageDiceMax               = 127
	MobDamageSidesMin              = 0
	MobDamageSidesMax              = 127
	MobHPDiceMin                   = 0
	MobHPDiceMax                   = 50
	MobHPSidesMin                  = 0
	MobHPSidesMax                  = 3000
	MobHPPlusMin                   = 0
	MobHPPlusMax                   = 30000
	MobACMin                       = -200
	MobACMax                       = 200
	MobExpMin                      = 0
	MobGoldMin                     = 0
	MobPositionMin                 = 0
	MobPositionMax                 = 14
	MobAttackMin                   = 0
	MobAttackMax                   = 14
	MobLevelMin                    = 0
	MobLevelMax                    = 100
	MobAlignmentMin                = -1000
	MobAlignmentMax                = 1000
	MobRaceMin                     = 0
	MobRaceMax                     = 30
	ShopHourMin                    = 0
	ShopHourMax                    = 28
	ZoneLifespanMin                = 0
	ZoneLifespanMax                = 240
	ZoneResetModeMin               = 0
	ZoneResetModeConditionalOffset = 1
	ZoneResetModeMax               = 2
	ObjectLoadMin                  = 0
	ObjectLoadMax                  = 100
	ObjectValueMin                 = -32000
	ObjectValueMax                 = 32000
	ObjectContainerMin             = 0
	ObjectContainerMax             = 15
	ObjectWeaponMin                = 0
	ObjectWeaponMax                = 20
	ObjectWandMin                  = 1
	ObjectSpellMin                 = 0
)

const MaxIntValue = int(^uint(0) >> 1)

// NewEntityRequiredLevel mirrors LVL_BUILDER in
// src/olc.h:54 and the level gate shared by the five OLC editors.
// NewZoneRequiredLevel and NewZoneRequiredLabel mirror LVL_HIGOD in
// src/structs.h:614, the action gate in src/olc.c:133, and the Go telnet
// branch in pkg/session/zedit.go:83.
const (
	NewEntityRequiredLevel = 31
	NewEntityRequiredLabel = "BUILDER"
	NewZoneRequiredLevel   = 36
	NewZoneRequiredLabel   = "HIGOD"
)

const (
	ItemLight      = 1
	ItemScroll     = 2
	ItemWand       = 3
	ItemStaff      = 4
	ItemWeapon     = 5
	ItemFireweapon = 6
	ItemMissile    = 7
	ItemArmor      = 9
	ItemPotion     = 10
	ItemContainer  = 15
	ItemNote       = 16
	ItemDrinkcon   = 17
	ItemFood       = 19
	ItemMoney      = 20
	ItemFountain   = 23
)

var (
	objectSpellOptions     = VocabularyOptions(SpellNames)
	objectLiquidOptions    = VocabularyOptions(LiquidNames)
	objectAttackOptions    = VocabularyOptions(AttackNames)
	objectContainerOptions = VocabularyOptions(ContainerFlagNames)
	objectApplyOptions     = VocabularyOptions(ApplyTypeNames)
)

var objectApplies = &AppliesDescriptor{
	Max:             parser.MAX_OBJ_AFFECT,
	Options:         objectApplyOptions,
	AddOperation:    "add_affect",
	RemoveOperation: "remove_affect",
}

var roomExits = &ExitDescriptor{
	Directions:  []string{"north", "east", "south", "west", "up", "down"},
	DoorOptions: schemaOptions(ExitDoorFlagNames),
}

var zoneCommandDescriptors = []ZoneCommandDescriptor{
	zoneCommand("M", "Load mobile to room", zoneArgument("mob_vnum", "Mobile VNUM", "vnum"), zoneArgument("max", "Maximum in world", "number"), zoneArgument("room_vnum", "Room VNUM", "vnum")),
	zoneCommand("O", "Load object to room", zoneArgument("object_vnum", "Object VNUM", "vnum"), zoneArgument("max", "Maximum in world", "number"), zoneArgument("room_vnum", "Room VNUM", "vnum")),
	zoneCommand("E", "Equip mobile with object", zoneArgument("object_vnum", "Object VNUM", "vnum"), zoneArgument("max", "Maximum in world", "number"), zoneSelectArgument("equipment", "Equipment position", schemaOptions(ZoneEquipmentNames))),
	zoneCommand("G", "Give object to mobile", zoneArgument("object_vnum", "Object VNUM", "vnum"), zoneArgument("max", "Maximum in world", "number"), hiddenZoneArgument("unused")),
	zoneCommand("P", "Put object in another object", zoneArgument("object_vnum", "Object VNUM", "vnum"), zoneArgument("max", "Maximum in world", "number"), zoneArgument("container_vnum", "Container VNUM", "vnum")),
	zoneCommand("D", "Open, close, or lock a door", zoneArgument("room_vnum", "Room VNUM", "vnum"), zoneSelectArgument("direction", "Exit direction", schemaOptions(ZoneDirections[:len(ZoneDirections)-1])), zoneSelectArgument("door_state", "Door state", []VocabularyEntry{{Value: 0, Label: "Door open"}, {Value: 1, Label: "Door closed"}, {Value: 2, Label: "Door locked"}})),
	zoneCommand("R", "Remove a mobile or object", zoneArgument("room_vnum", "Room VNUM", "vnum"), zoneSelectArgument("target_kind", "Target kind", []VocabularyEntry{{Value: 0, Label: "Mobile"}, {Value: 1, Label: "Object"}}), zoneArgument("target_vnum", "Target VNUM", "vnum")),
	zoneCommand("L", "Begin or end looping", zoneArgument("room_vnum", "Room VNUM", "vnum"), zoneSelectArgument("loop_mode", "Loop mode", []VocabularyEntry{{Value: 0, Label: "Loop start"}, {Value: 1, Label: "Loop finish"}}), zoneArgument("repeat_count", "Repeat count", "number")),
}

func zoneCommand(command, label string, args ...ZoneCommandArgument) ZoneCommandDescriptor {
	descriptor := ZoneCommandDescriptor{Command: command, Label: label}
	copy(descriptor.Arguments[:], args)
	return descriptor
}

func zoneArgument(key, label, control string) ZoneCommandArgument {
	return ZoneCommandArgument{Key: key, Label: label, Control: control, Visible: true}
}

func zoneSelectArgument(key, label string, options []VocabularyEntry) ZoneCommandArgument {
	argument := zoneArgument(key, label, "select")
	argument.Options = options
	return argument
}

func hiddenZoneArgument(key string) ZoneCommandArgument {
	return ZoneCommandArgument{Key: key, Control: "number", Visible: false}
}

// ObjectValueBounds is the same matrix used by OpSetObjValue1..4. Invisible
// rows retain the C operation's generic bounds for a stable, inspectable row.
func ObjectValueBounds(typeFlag, index int) (int, int) {
	switch index {
	case 0:
		return ObjectValueMin, ObjectValueMax
	case 1:
		switch typeFlag {
		case ItemScroll, ItemPotion:
			return ObjectSpellMin, len(SpellNames) - 1
		case ItemContainer:
			return ObjectContainerMin, ObjectContainerMax
		}
	case 2:
		switch typeFlag {
		case ItemScroll, ItemPotion:
			return ObjectSpellMin, len(SpellNames) - 1
		case ItemWeapon, ItemWand, ItemStaff:
			return ObjectWeaponMin, ObjectWeaponMax
		case ItemDrinkcon, ItemFountain:
			return ObjectSpellMin, len(LiquidNames) - 1
		}
	case 3:
		switch typeFlag {
		case ItemScroll, ItemPotion:
			return ObjectSpellMin, len(SpellNames) - 1
		case ItemWand, ItemStaff:
			return ObjectWandMin, len(SpellNames) - 1
		case ItemWeapon:
			return ObjectSpellMin, len(AttackNames) - 1
		}
	}
	return ObjectValueMin, ObjectValueMax
}

func hiddenObjectValueField(typeFlag, index int) ObjectValueField {
	min, max := ObjectValueBounds(typeFlag, index)
	return ObjectValueField{Min: min, Max: max, Control: "number"}
}

func objectValueField(typeFlag, index int) ObjectValueField {
	field := hiddenObjectValueField(typeFlag, index)
	field.Visible = true
	field.Label = objectValueLabel(typeFlag, index)
	switch {
	case index == 1 && (typeFlag == ItemScroll || typeFlag == ItemPotion):
		field.Control = "spell_picker"
		field.Options = objectSpellOptions
	case index == 2 && (typeFlag == ItemScroll || typeFlag == ItemPotion):
		field.Control = "spell_picker"
		field.Options = objectSpellOptions
	case index == 3 && (typeFlag == ItemScroll || typeFlag == ItemPotion || typeFlag == ItemWand || typeFlag == ItemStaff):
		field.Control = "spell_picker"
		field.Options = objectSpellOptions
	case index == 1 && typeFlag == ItemContainer:
		field.Control = "checkbox_group"
		field.Options = objectContainerOptions
	case index == 2 && (typeFlag == ItemDrinkcon || typeFlag == ItemFountain):
		field.Control = "select"
		field.Options = objectLiquidOptions
	case index == 3 && typeFlag == ItemWeapon:
		field.Control = "select"
		field.Options = objectAttackOptions
	}
	return field
}

func objectValueLabel(typeFlag, index int) string {
	switch index {
	case 0:
		switch typeFlag {
		case ItemScroll, ItemWand, ItemStaff, ItemPotion:
			return "Spell level : "
		case ItemArmor:
			return "Apply to AC : "
		case ItemContainer:
			return "Max weight to contain : "
		case ItemDrinkcon, ItemFountain:
			return "Max drink units : "
		case ItemFood:
			return "Hours to fill stomach : "
		case ItemMoney:
			return "Number of gold coins : "
		}
	case 1:
		switch typeFlag {
		case ItemScroll, ItemPotion:
			return "Enter spell choice (0 for none) : "
		case ItemWand, ItemStaff:
			return "Max number of charges : "
		case ItemMissile, ItemFireweapon, ItemWeapon:
			return "Number of damage dice : "
		case ItemContainer:
			return "Container flags: "
		case ItemDrinkcon, ItemFountain:
			return "Initial drink units : "
		}
	case 2:
		switch typeFlag {
		case ItemLight:
			return "Number of hours (0 = burnt, -1 is infinite) : "
		case ItemWand, ItemStaff:
			return "Number of charges remaining : "
		case ItemScroll, ItemPotion:
			return "Enter spell choice (0 for none) : "
		case ItemMissile, ItemFireweapon, ItemWeapon:
			return "Size of damage dice : "
		case ItemContainer:
			return "Vnum of key to open container (-1 for no key) : "
		}
		if typeFlag == ItemDrinkcon || typeFlag == ItemFountain {
			return "Enter drink type : "
		}
	case 3:
		switch typeFlag {
		case ItemScroll, ItemPotion, ItemWand, ItemStaff:
			return "Enter spell choice (0 for none) : "
		case ItemWeapon:
			return "Enter weapon type : "
		case ItemDrinkcon, ItemFountain, ItemFood:
			return "Poisoned (0 = not poison) : "
		}
	}
	return ""
}

func objectValueVisible(typeFlag, index int) bool {
	switch index {
	case 0:
		return typeFlag == ItemScroll || typeFlag == ItemWand || typeFlag == ItemStaff || typeFlag == ItemPotion || typeFlag == ItemArmor || typeFlag == ItemContainer || typeFlag == ItemDrinkcon || typeFlag == ItemFountain || typeFlag == ItemFood || typeFlag == ItemMoney
	case 1:
		return typeFlag == ItemScroll || typeFlag == ItemPotion || typeFlag == ItemWand || typeFlag == ItemStaff || typeFlag == ItemMissile || typeFlag == ItemFireweapon || typeFlag == ItemWeapon || typeFlag == ItemContainer || typeFlag == ItemDrinkcon || typeFlag == ItemFountain
	case 2:
		return typeFlag == ItemLight || typeFlag == ItemScroll || typeFlag == ItemPotion || typeFlag == ItemWand || typeFlag == ItemStaff || typeFlag == ItemMissile || typeFlag == ItemFireweapon || typeFlag == ItemWeapon || typeFlag == ItemContainer || typeFlag == ItemDrinkcon || typeFlag == ItemFountain
	case 3:
		return typeFlag == ItemScroll || typeFlag == ItemPotion || typeFlag == ItemWand || typeFlag == ItemStaff || typeFlag == ItemWeapon || typeFlag == ItemDrinkcon || typeFlag == ItemFountain || typeFlag == ItemFood
	default:
		return false
	}
}

// ObjectValueMatrix contains one resolved four-slot row for every C item type.
var ObjectValueMatrix = buildObjectValueMatrix()

func buildObjectValueMatrix() []ObjectValueMatrixEntry {
	matrix := make([]ObjectValueMatrixEntry, len(ItemTypeNames))
	for typeFlag, label := range ItemTypeNames {
		row := ObjectValueMatrixEntry{ItemType: typeFlag, ItemTypeLabel: label}
		for index := range row.Values {
			row.Values[index] = hiddenObjectValueField(typeFlag, index)
			if objectValueVisible(typeFlag, index) {
				row.Values[index] = objectValueField(typeFlag, index)
			}
		}
		matrix[typeFlag] = row
	}
	return matrix
}

func schemaBounds(min, max int) *Bounds {
	return &Bounds{Min: min, Max: max}
}

func schemaOptions(names []string) []VocabularyEntry {
	return VocabularyOptions(names)
}

func schemaOptionsFrom(names []string, firstValue int) []VocabularyEntry {
	options := schemaOptions(names)
	for i := range options {
		options[i].Value += firstValue
	}
	return options
}

func schemaFlagOptions(flags []FlagVocabulary) []VocabularyEntry {
	return FlagOptions(flags)
}

func textField(key, label, control string) SchemaField {
	return SchemaField{Key: key, Label: label, Control: control}
}

func numberField(key, label string, min, max int) SchemaField {
	return SchemaField{Key: key, Label: label, Control: "number", Bounds: schemaBounds(min, max)}
}

func unboundedNumberField(key, label string) SchemaField {
	return SchemaField{Key: key, Label: label, Control: "number"}
}

func selectField(key, label string, options []VocabularyEntry) SchemaField {
	return SchemaField{Key: key, Label: label, Control: "select", Options: options}
}

func checkboxField(key, label string, options []VocabularyEntry) SchemaField {
	return SchemaField{Key: key, Label: label, Control: "checkbox_group", Options: options}
}

func objectValueMatrixField() SchemaField {
	return SchemaField{Key: "values", Label: "Values", Control: "number"}
}

// SchemaForKind returns the vocabulary consumed by the generic web editor.
// Object level and timer are intentionally absent: their P6 operations are
// faithful no-ops and exposing dead controls would invite false edits.
func SchemaForKind(kind string) (Schema, bool) {
	schema := Schema{Kind: kind}
	switch kind {
	case "room":
		schema.Actions = []SchemaAction{{Key: "new_room", Label: "New room", RequiredLevel: NewEntityRequiredLevel, RequiredLabel: NewEntityRequiredLabel}}
		schema.Bespoke = []string{"exits", "extra_descriptions"}
		schema.Fields = []SchemaField{
			textField("name", "Name", "text"),
			textField("description", "Description", "textarea"),
			checkboxField("flags", "Room flags", schemaOptions(RoomFlagNames)),
			selectField("sector", "Sector type", schemaOptions(SectorNames)),
			textField("script_name", "Script name", "text"),
			checkboxField("script_flags", "Script flags", schemaOptions(RoomScriptFlagNames)),
		}
		schema.Exits = roomExits
	case "mob":
		schema.Actions = []SchemaAction{{Key: "new_mob", Label: "New mob", RequiredLevel: NewEntityRequiredLevel, RequiredLabel: NewEntityRequiredLabel}}
		schema.Fields = []SchemaField{
			textField("keywords", "Alias", "text"),
			textField("short_description", "Short description", "text"),
			textField("long_description", "Long description", "text"),
			textField("detailed_description", "Detailed description", "textarea"),
			textField("noise", "Noise", "text"),
			selectField("sex", "Gender", schemaOptions(GenderNames)),
			numberField("hitroll", "Hitroll", MobHitrollMin, MobHitrollMax),
			numberField("damroll", "Damroll", MobDamrollMin, MobDamrollMax),
			numberField("damage_dice", "Damage dice", MobDamageDiceMin, MobDamageDiceMax),
			numberField("damage_sides", "Damage sides", MobDamageSidesMin, MobDamageSidesMax),
			numberField("hp_dice", "HP dice", MobHPDiceMin, MobHPDiceMax),
			numberField("hp_sides", "HP sides", MobHPSidesMin, MobHPSidesMax),
			numberField("hp_plus", "HP plus", MobHPPlusMin, MobHPPlusMax),
			numberField("ac", "AC", MobACMin, MobACMax),
			numberField("exp", "Experience", MobExpMin, MaxIntValue),
			numberField("gold", "Gold", MobGoldMin, MaxIntValue),
			selectField("position", "Position", schemaOptions(PositionNames)),
			selectField("default_position", "Default position", schemaOptions(PositionNames)),
			selectField("attack", "Attack type", schemaOptions(AttackNames)),
			numberField("level", "Level", MobLevelMin, MobLevelMax),
			numberField("alignment", "Alignment", MobAlignmentMin, MobAlignmentMax),
			selectField("race", "Race", schemaOptions(MobRaceNames)),
			checkboxField("action_flags", "Mob flags", schemaFlagOptions(MobActionFlags)),
			checkboxField("affect_flags", "Affect flags", schemaFlagOptions(MobAffectFlags)),
			textField("script_name", "Script name", "text"),
			checkboxField("script_flags", "Script flags", schemaOptions(MobScriptFlagNames)),
		}
	case "obj":
		schema.Actions = []SchemaAction{{Key: "new_obj", Label: "New object", RequiredLevel: NewEntityRequiredLevel, RequiredLabel: NewEntityRequiredLabel}}
		schema.Bespoke = []string{"values", "applies", "extra_descriptions"}
		schema.Fields = []SchemaField{
			textField("keywords", "Keywords", "text"),
			textField("short_description", "Short description", "text"),
			textField("long_description", "Long description", "text"),
			textField("action_description", "Action description", "textarea"),
			selectField("type", "Item type", schemaOptionsFrom(ItemTypeNames[1:], ItemLight)),
			checkboxField("extra_flags", "Object flags", schemaOptions(ItemExtraFlagNames)),
			checkboxField("wear_flags", "Wear flags", schemaOptions(ItemWearFlagNames)),
			unboundedNumberField("weight", "Weight"),
			unboundedNumberField("cost", "Cost"),
			numberField("cost_per_day", "Cost per day", ObjectLoadMin, ObjectLoadMax),
			objectValueMatrixField(),
			textField("script_name", "Script name", "text"),
			checkboxField("script_flags", "Script flags", schemaOptions(ObjectScriptFlagNames)),
		}
		schema.ValueMatrix = ObjectValueMatrix
		schema.Applies = objectApplies
	case "shop":
		schema.Actions = []SchemaAction{{Key: "new_shop", Label: "New shop", RequiredLevel: NewEntityRequiredLevel, RequiredLabel: NewEntityRequiredLabel}}
		schema.Bespoke = []string{"products", "rooms", "trade_namelist"}
		schema.Fields = []SchemaField{
			{Key: "products", Label: "Products", Control: "vnum_list"},
			{Key: "rooms", Label: "Shop rooms", Control: "vnum_list"},
			{Key: "trade_namelist", Label: "Trade namelist", Control: "trade_namelist"},
			unboundedNumberField("buy_profit", "Buy rate"),
			unboundedNumberField("sell_profit", "Sell rate"),
			{Key: "keeper", Label: "Keeper", Control: "vnum"},
			checkboxField("flags", "Shop flags", schemaOptions(ShopFlagNames)),
			checkboxField("with_who", "No Trade With", schemaOptions(ShopTradeNames)),
			numberField("open_hour_1", "Open 1", ShopHourMin, ShopHourMax),
			numberField("open_hour_2", "Open 2", ShopHourMin, ShopHourMax),
			numberField("close_hour_1", "Close 1", ShopHourMin, ShopHourMax),
			numberField("close_hour_2", "Close 2", ShopHourMin, ShopHourMax),
			textField("message_no_item_keeper", "Keeper no item", "text"),
			textField("message_no_item_player", "Player no item", "text"),
			textField("message_no_buy", "Keeper no buy", "text"),
			textField("message_no_cash_keeper", "Keeper no cash", "text"),
			textField("message_no_cash_player", "Player no cash", "text"),
			textField("message_buy", "Buy success", "text"),
			textField("message_sell", "Sell success", "text"),
		}
	case "zone":
		schema.Bespoke = []string{"commands"}
		schema.Actions = []SchemaAction{{Key: "create_zone", Label: "New zone", RequiredLevel: NewZoneRequiredLevel, RequiredLabel: NewZoneRequiredLabel}}
		schema.ZoneCommands = zoneCommandDescriptors
		schema.Fields = []SchemaField{
			textField("name", "Zone name", "text"),
			numberField("lifespan", "Lifespan", ZoneLifespanMin, ZoneLifespanMax),
			selectField("reset_mode", "Reset Mode", []VocabularyEntry{
				{Value: ZoneResetModeMin, Label: "Never reset"},
				{Value: ZoneResetModeMin + ZoneResetModeConditionalOffset, Label: "Reset when no players are in zone."},
				{Value: ZoneResetModeMax, Label: "Normal reset."},
			}),
			{Key: "top_room", Label: "Top of zone", Control: "vnum"},
		}
	default:
		return Schema{}, false
	}
	return schema, true
}
