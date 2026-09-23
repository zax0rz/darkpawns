package game

import "testing"

// TestPRFAndPLRBitsDoNotOverlap: C keeps PLR and PRF flags in separate arrays
// (structs.h PLR_* 0..21, PRF_* 0..31). Player.Flags holds both, PRF from
// prfBase, so no PRF bit may land on a PLR bit. The old base of 20 put
// PRF_BRIEF on PLR_REMORT and PRF_COMPACT on PLR_EXTRACT.
func TestPRFAndPLRBitsDoNotOverlap(t *testing.T) {
	plr := map[string]int{
		"outlaw": PlrOutlaw, "open": PlrOpen, "frozen": PlrFrozen, "dontset": PlrDontset,
		"writing": PlrWriting, "mailing": PlrMailing, "crash": PlrCrash, "siteok": PlrSiteok,
		"noshout": PlrNoshout, "notitle": PlrNotitle, "deleted": PlrDeleted, "loadroom": PlrLoadroom,
		"nowizlist": PlrNowizlist, "nodelete": PlrNODELETE, "invstart": PlrInvstart, "cryo": PlrCRYO,
		"werewolf": PlrWerewolf, "vampire": PlrVampire, "it": PlrIt, "chosen": PlrChosen,
		"remort": PlrRemort, "extract": PlrExtract, "calibrate": plrFlagMap["calibrate"],
	}
	prf := []int{
		PrfBrief, PrfCompact, PrfDeaf, PrfNotell, PrfDisphp, PrfDispmmana, PrfDispmove, PrfAutoexit,
		PrfNohassle, PrfQuest, PrfSummonable, PrfNoRepeat, PrfHolyLight, PrfColor1, PrfColor2, PrfNowiz,
		PrfLog1, PrfLog2, PrfNoAuctions, PrfNoGossip, PrfNoGratz, PrfRoomFlags, PrfAFK, PrfAutoLoot,
		PrfAutoGold, PrfAutoSplit, PrfDispTank, PrfDispTarget, PrfNoNewbie, PrfInactive, PrfNoCTell, PrfNoBroad,
	}
	for i, bit := range prf {
		if bit != prfBase+i {
			t.Errorf("PRF bit %d is %d, want prfBase+%d (C's PRF numbering)", i, bit, i)
		}
	}
	for name, bit := range plr {
		if bit < 0 || bit >= prfBase {
			t.Errorf("PLR %s = %d, outside 0..%d", name, bit, prfBase-1)
		}
	}
}

// TestPLRAccessorsShareOneStore: both accessor families read and write
// Player.Flags, so a flag set one way is seen the other way. Before, the
// lycanthropy spell set PLR_WEREWOLF through SetPLRFlag (a second field)
// and transform read it through GetFlags, so an infected player could never
// transform.
func TestPLRAccessorsShareOneStore(t *testing.T) {
	p := &Player{}
	p.SetPLRFlag(PlrWerewolf)
	if p.GetFlags()&(1<<PlrWerewolf) == 0 {
		t.Fatal("SetPLRFlag is not visible through GetFlags")
	}
	p.SetPlrFlag(PlrChosen, true)
	if !p.HasPLRFlag(PlrChosen) {
		t.Fatal("SetPlrFlag is not visible through HasPLRFlag")
	}
	p.ClearPLRFlag(PlrWerewolf)
	if p.GetFlags()&(1<<PlrWerewolf) != 0 {
		t.Fatal("ClearPLRFlag did not clear Flags")
	}

	// Preferences no longer masquerade as PLR state.
	p.SetPlrFlag(PrfBrief, true)
	p.SetPlrFlag(PrfCompact, true)
	if p.HasPLRFlag(PlrRemort) || p.HasPLRFlag(PlrExtract) {
		t.Fatal("brief/compact read as PLR_REMORT/PLR_EXTRACT")
	}
}

// TestMigrateFlagsV1: an old save's preferences land on the new bits, and
// its PLR bits are untouched.
func TestMigrateFlagsV1(t *testing.T) {
	old := uint64(1)<<PlrFrozen | 1<<20 | 1<<31 | 1<<49 | 1<<29 // frozen, brief, color1, quest, holylight
	want := uint64(1)<<PlrFrozen | 1<<uint(PrfBrief) | 1<<uint(PrfColor1) | 1<<uint(PrfQuest) | 1<<uint(PrfHolyLight)
	if got := migrateFlagsV1(old); got != want {
		t.Fatalf("migrateFlagsV1(%#x) = %#x, want %#x", old, got, want)
	}
}

// TestDeserializeMigratesVersion1Flags: a version 1 save goes through the
// migration on load; a current save does not.
func TestDeserializeMigratesVersion1Flags(t *testing.T) {
	v1, err := DeserializePlayer(`{"save_version":1,"name":"Old","flags":1048576}`) // bit 20: brief in v1
	if err != nil {
		t.Fatal(err)
	}
	if v1.GetFlags() != 1<<uint(PrfBrief) {
		t.Fatalf("v1 flags = %#x, want brief at %d", v1.GetFlags(), PrfBrief)
	}
	current, err := SerializePlayer(v1)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DeserializePlayer(current)
	if err != nil {
		t.Fatal(err)
	}
	if again.GetFlags() != v1.GetFlags() {
		t.Fatalf("round trip flags = %#x, want %#x", again.GetFlags(), v1.GetFlags())
	}
}
