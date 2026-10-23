package session

import (
	"errors"
	"testing"
)

// L1-005 (option B, web half): on an empty player store a proxied client
// must not create the first character — C crowns it Implementor
// (src/db.c:3016). Local sessions still create it; once the world has an
// owner, proxied creation is ordinary entry again.
func TestPersistAcceptedCharacterProxiedEmptyWorldRefused(t *testing.T) {
	database := entryDatabase(t)
	m := entryWorldManager(t, database)

	newChar := func(name string, proxied bool) *Session {
		t.Helper()
		s := makeCharSession(t, m)
		s.charName = name
		s.charPassword = "hash"
		s.charClass = 0
		s.charRace = 0
		s.proxied = proxied
		return s
	}

	if err := newChar("Proxiedfirst", true).persistAcceptedCharacter(); !errors.Is(err, ErrFirstCharacterMustBeLocal) {
		t.Fatalf("proxied empty-world creation = %v, want ErrFirstCharacterMustBeLocal", err)
	}
	if err := newChar("Localowner", false).persistAcceptedCharacter(); err != nil {
		t.Fatalf("local empty-world creation = %v, want nil", err)
	}
	// The world now has its immortal; a proxied session creating the second
	// character is normal entry and must not be refused.
	if err := newChar("Secondmortal", true).persistAcceptedCharacter(); err != nil {
		t.Fatalf("proxied creation on a populated world = %v, want nil", err)
	}
}
