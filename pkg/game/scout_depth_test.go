package game

import "testing"

func TestDoScoutTerrainAndCrowdDescriptions(t *testing.T) {
	terrainTests := []struct {
		sector int
		want   string
	}{
		{sector: 0, want: "the inside of a structure"},
		{sector: 1, want: "the cobblestones of a city"},
		{sector: 2, want: "a wide swath of field"},
		{sector: 3, want: "the dense forest"},
		{sector: 4, want: "high hills"},
		{sector: 5, want: "jagged mountains"},
		{sector: 6, want: "a large stretch of water"},
		{sector: 7, want: "a large stretch of water"},
		{sector: 8, want: "the watery depths"},
		{sector: 9, want: "thin air"},
		{sector: 10, want: "a vast wasteland"},
		{sector: 11, want: "the endless elemental plane"},
		{sector: 14, want: "the endless elemental plane"},
		{sector: 15, want: "a murky swamp"},
	}
	for _, test := range terrainTests {
		if got := scoutTerrain(test.sector); got != test.want {
			t.Errorf("scoutTerrain(%d) = %q, want %q", test.sector, got, test.want)
		}
	}

	crowdTests := []struct {
		count int
		want  string
	}{
		{count: 1, want: "someone"},
		{count: 2, want: "a few people"},
		{count: 3, want: "a few people"},
		{count: 4, want: "a group of people"},
		{count: 5, want: "a group of people"},
		{count: 6, want: "a large group of people"},
		{count: 9, want: "a large group of people"},
		{count: 10, want: "a crowd of people"},
		{count: 12, want: "a crowd of people"},
		{count: 13, want: "a large crowd of people"},
		{count: 14, want: "a large crowd of people"},
		{count: 15, want: "a large mob"},
	}
	for _, test := range crowdTests {
		if got := scoutCrowdSize(test.count); got != test.want {
			t.Errorf("scoutCrowdSize(%d) = %q, want %q", test.count, got, test.want)
		}
	}
}
