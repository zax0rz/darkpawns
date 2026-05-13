package dreaming

import "testing"

func TestKillValence(t *testing.T) {
	cases := []struct {
		agentLevel, mobLevel int
		want                 int
	}{
		{20, 5, 0},  // trivial rat
		{20, 13, 1}, // easy
		{20, 20, 2}, // equal level — challenging
		{20, 25, 3}, // mob outlevels agent — epic
		{0, 10, 1},  // unknown agent level → default
		{10, 0, 1},  // unknown mob level → default
	}
	for _, c := range cases {
		got := KillValence(c.agentLevel, c.mobLevel)
		if got != c.want {
			t.Errorf("KillValence(%d, %d) = %d, want %d", c.agentLevel, c.mobLevel, got, c.want)
		}
	}
}

func TestFleeValence(t *testing.T) {
	cases := []struct {
		hpPct int
		want  int
	}{
		{100, -3}, // full health, cowardly
		{80, -3},  // boundary: exactly 80 → -3
		{79, -2},  // just below 80
		{40, -2},  // boundary: exactly 40 → -2
		{39, -1},  // just below 40
		{20, -1},  // boundary: exactly 20 → -1
		{19, 0},   // near-death, survival instinct
		{1, 0},
	}
	for _, c := range cases {
		got := fleeValence(c.hpPct)
		if got != c.want {
			t.Errorf("fleeValence(%d) = %d, want %d", c.hpPct, got, c.want)
		}
	}
}

func TestPercentHP(t *testing.T) {
	cases := []struct {
		hp, maxHP int
		want      int
	}{
		{50, 100, 50},
		{0, 100, 0},
		{100, 100, 100},
		{12, 50, 24},
		{0, 0, 100}, // zero maxHP → treat as full
	}
	for _, c := range cases {
		got := percentHP(c.hp, c.maxHP)
		if got != c.want {
			t.Errorf("percentHP(%d, %d) = %d, want %d", c.hp, c.maxHP, got, c.want)
		}
	}
}

func TestSpeechValence(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"thank you friend", 1},
		{"I hate you, you fool", -1},
		{"", 0},
		{"the weather is fine today", 0},
		{"please help me", 1},
		{"die traitor", -1},
	}
	for _, c := range cases {
		got := speechValence(c.text)
		if got != c.want {
			t.Errorf("speechValence(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}

func TestDamageValence(t *testing.T) {
	cases := []struct {
		hp, maxHP int
		want      int
	}{
		{2, 100, -3},  // near-death <5%
		{10, 100, -2}, // badly hurt 5-15%
		{25, 100, -1}, // moderate 15-30%
		{50, 100, 0},  // not significant
		{0, 0, -1},    // zero maxHP fallback
	}
	for _, c := range cases {
		got := damageValence(c.hp, c.maxHP)
		if got != c.want {
			t.Errorf("damageValence(%d, %d) = %d, want %d", c.hp, c.maxHP, got, c.want)
		}
	}
}
