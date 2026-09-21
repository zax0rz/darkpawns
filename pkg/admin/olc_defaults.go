package admin

import "github.com/zax0rz/darkpawns/pkg/parser"

func newOLCRoom(vnum, zone int) parser.Room {
	return parser.Room{
		VNum:        vnum,
		Name:        "An unfinished room",
		Description: "You are in an unfinished room.\n",
		Zone:        zone,
		Flags:       []string{"0", "0", "0", "0"},
		Exits:       make(map[string]parser.Exit),
	}
}

func newOLCMob(vnum int) parser.Mob {
	return parser.Mob{
		VNum:         vnum,
		Keywords:     "mob unfinished",
		ShortDesc:    "the unfinished mob",
		LongDesc:     "An unfinished mob stands here.\r\n",
		DetailedDesc: "It looks, err, unfinished.\r\n",
		Race:         16,
		Position:     8,
		DefaultPos:   8,
		AC:           100,
		HP:           parser.DiceRoll{Num: 1, Sides: 1},
		Damage:       parser.DiceRoll{Num: 1, Sides: 1},
		Weight:       200,
		Height:       198,
		ActionFlags:  []string{"ISNPC"},
	}
}

func newOLCObject(vnum int) parser.Obj {
	return parser.Obj{
		VNum:      vnum,
		Keywords:  "unfinished object",
		ShortDesc: "an unfinished object",
		LongDesc:  "An unfinished object is lying here.",
		WearFlags: [4]int{1, 0, 0, 0},
	}
}

func newOLCShop(vnum int) parser.ShopProto {
	return parser.ShopProto{
		VNum:       vnum,
		KeeperVNum: -1,
		BuyProfit:  1.0,
		SellProfit: 1.0,
		Messages: [7]string{
			"%s Sorry, I don't stock that item.",
			"%s You don't seem to have that.",
			"%s I don't trade in such items.",
			"%s I can't afford that!",
			"%s You are too poor!",
			"%s That'll be %d coins, thanks.",
			"%s I'll give you %d coins for that.",
		},
		CloseHour1: 28,
	}
}
