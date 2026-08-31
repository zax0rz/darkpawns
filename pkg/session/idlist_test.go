package session

import (
	"bytes"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestWriteObjectIDListMatchesCReportShape(t *testing.T) {
	objects := []parser.Obj{
		{
			ShortDesc: "a bright scroll",
			TypeFlag:  2,
			ExtraFlags: [4]int{
				1<<0 | 1<<7,
			},
			Values:  [4]int{0, 3, 0, 38},
			Weight:  1,
			Cost:    25,
			Affects: []parser.ObjAffect{{Location: 17, Modifier: -1}},
		},
		{
			ShortDesc: "a practice wand",
			TypeFlag:  3,
			Values:    [4]int{0, 3, 2, 38},
		},
		{
			ShortDesc: "a short sword",
			TypeFlag:  5,
			Values:    [4]int{0, 1, 6, 11},
		},
		{
			ShortDesc: "a steel vest",
			TypeFlag:  9,
			Values:    [4]int{7, 0, 0, 0},
		},
	}

	var got bytes.Buffer
	count, err := writeObjectIDList(&got, objects)
	if err != nil {
		t.Fatalf("writeObjectIDList returned error: %v", err)
	}
	if count != len(objects) {
		t.Fatalf("writeObjectIDList count = %d, want %d", count, len(objects))
	}
	want := "Object: 'a bright scroll', Item type: SCROLL\n" +
		"Item will give the following abilities:  NOBITS \n" +
		"Item is: GLOW !DROP \n" +
		"Encumbrance: 1, Value: 25\n" +
		"This SCROLL casts: blesssleep\n" +
		"Can affect you as :\n" +
		"   Affects: ARMOR By -1\n" +
		"\n" +
		"Object: 'a practice wand', Item type: CASTING_EQ\n" +
		"Item will give the following abilities:  NOBITS \n" +
		"Item is: NOBITS \n" +
		"Encumbrance: 0, Value: 0\n" +
		"This WAND casts: sleepIt has 3 maximum charges and 2 remaining.\n" +
		"\n" +
		"\n" +
		"Object: 'a short sword', Item type: WEAPON\n" +
		"Item will give the following abilities:  NOBITS \n" +
		"Item is: NOBITS \n" +
		"Encumbrance: 0, Value: 0\n" +
		"Damage Dice is '1D6' for an average per-round damage of 3.5.\n" +
		"\n" +
		"Object: 'a steel vest', Item type: ARMOR\n" +
		"Item will give the following abilities:  NOBITS \n" +
		"Item is: NOBITS \n" +
		"Encumbrance: 0, Value: 0\n" +
		"AC-apply is 7\n" +
		"\n"
	if got.String() != want {
		t.Fatalf("report mismatch\n got: %q\nwant: %q", got.String(), want)
	}
}

func TestIDListBitArrayUsesUndefinedForOutOfTableBits(t *testing.T) {
	if got := idlistBitArray([4]int{1 << 31, 0, 0, 0}, idlistExtraBits); got != "UNDEFINED " {
		t.Fatalf("idlistBitArray = %q, want %q", got, "UNDEFINED ")
	}
}
