package session

import "testing"

// C's mob_race_name (src/act.wizard.c:692-697) returns "None" for a negative
// slot and otherwise names the race; do_stat prints all five slots
// (:891-898). wizardMobRaceNames is the port's copy of C's mob_races table
// (src/constants.c:138), so index 18 is Giant and index 20 is Ogre.
func TestStatRaceHateName(t *testing.T) {
	cases := []struct {
		race int
		want string
	}{
		{-1, "None"},
		{0, "Human"},
		{18, "Giant"},
		{20, "Ogre"},
		{len(wizardMobRaceNames), "None"},
	}
	for _, c := range cases {
		if got := statRaceHateName(c.race); got != c.want {
			t.Errorf("statRaceHateName(%d) = %q, want %q", c.race, got, c.want)
		}
	}
}
