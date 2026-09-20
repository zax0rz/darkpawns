package olc

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestAuthorizedBuilderBoundary(t *testing.T) {
	if Authorized(34, 30, 31) {
		t.Fatal("level 34 builder was authorized outside assigned zone")
	}
	if !Authorized(34, 30, 30) {
		t.Fatal("level 34 builder was refused inside assigned zone")
	}
	if !Authorized(35, 30, 31) {
		t.Fatal("level 35 god was refused outside assigned zone")
	}
}

func TestZoneForVNum(t *testing.T) {
	zones := []*parser.Zone{
		{Number: 30, TopRoom: 3099},
		{Number: 31, TopRoom: 3199},
	}

	zone, ok := ZoneForVNum(zones, 3099)
	if !ok || zone != zones[0] {
		t.Fatalf("upper boundary lookup = (%v, %t), want zone 30", zone, ok)
	}
	if _, ok := ZoneForVNum(zones, 3100); !ok {
		t.Fatal("lower boundary of zone 31 was not found")
	}
	if _, ok := ZoneForVNum(zones, 3200); ok {
		t.Fatal("out-of-range VNUM unexpectedly matched a zone")
	}
}
