package game

import (
	"testing"
)

// conAppGolden is the con_app[] table transcribed verbatim from src/constants.c:1124-1150
var conAppGolden = [26][2]int{
	{-4, 20}, // con = 0
	{-3, 25}, // con = 1
	{-2, 30},
	{-2, 35},
	{-1, 40},
	{-1, 45}, // con = 5
	{-1, 50},
	{0, 55},
	{0, 60},
	{0, 65},
	{0, 70}, // con = 10
	{0, 75},
	{0, 80},
	{0, 85},
	{0, 88},
	{1, 90}, // con = 15
	{2, 95},
	{2, 97},
	{3, 99}, // con = 18
	{3, 99},
	{4, 99}, // con = 20
	{5, 99},
	{5, 99},
	{5, 99},
	{6, 99},
	{6, 99}, // con = 25
}

// intAppGolden is the int_app[] table transcribed verbatim from src/constants.c:1156-1183
var intAppGolden = [26]int{
	3,  // int = 0
	5,  // int = 1
	7,
	8,
	9,
	10, // int = 5
	11,
	12,
	13,
	15,
	17, // int = 10
	19,
	22,
	25,
	30,
	35, // int = 15
	40,
	45,
	50, // int = 18
	53,
	55, // int = 20
	56,
	57,
	58,
	59,
	60, // int = 25
}

// wisAppGolden is the wis_app[] table transcribed verbatim from src/constants.c:1187-1214
var wisAppGolden = [26]int{
	0, // wis = 0
	0, // wis = 1
	0,
	0,
	0,
	0, // wis = 5
	0,
	0,
	0,
	0,
	0, // wis = 10
	0,
	2,
	2,
	3,
	3, // wis = 15
	3,
	4,
	5, // wis = 18
	6,
	6, // wis = 20
	6,
	6,
	7,
	7,
	7, // wis = 25
}

// TestAttributeApp_GoldenAgainstCSource asserts that Go conApp, intApp, and wisApp tables match C constants exactly.
func TestAttributeApp_GoldenAgainstCSource(t *testing.T) {
	// 1. conApp
	if len(conApp) != len(conAppGolden) {
		t.Fatalf("conApp length = %d; want %d", len(conApp), len(conAppGolden))
	}
	for i, want := range conAppGolden {
		if conApp[i].Hitp != want[0] || conApp[i].Shock != want[1] {
			t.Errorf("conApp[%d] = {Hitp:%d, Shock:%d}; want {Hitp:%d, Shock:%d}",
				i, conApp[i].Hitp, conApp[i].Shock, want[0], want[1])
		}
	}

	// 2. intApp
	if len(intApp) != len(intAppGolden) {
		t.Fatalf("intApp length = %d; want %d", len(intApp), len(intAppGolden))
	}
	for i, want := range intAppGolden {
		if intApp[i].Learn != want {
			t.Errorf("intApp[%d] = {Learn:%d}; want {Learn:%d}",
				i, intApp[i].Learn, want)
		}
	}

	// 3. wisApp
	if len(wisApp) != len(wisAppGolden) {
		t.Fatalf("wisApp length = %d; want %d", len(wisApp), len(wisAppGolden))
	}
	for i, want := range wisAppGolden {
		if wisApp[i].Bonus != want {
			t.Errorf("wisApp[%d] = {Bonus:%d}; want {Bonus:%d}",
				i, wisApp[i].Bonus, want)
		}
	}
}
