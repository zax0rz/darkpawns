package game

import (
	"encoding/json"
	"testing"
)

func TestCharacterDataAdminEditsPreserveIdentityAndUnknownFields(t *testing.T) {
	raw := []byte(`{"save_version":1,"name":"Oldname","flags":1048576,"clan_id":8,"clan_rank":2,"future_field":{"kept":true}}`)
	flags, err := CharacterDataFlags(raw)
	if err != nil || flags&(1<<uint(PrfBrief)) == 0 {
		t.Fatalf("migrated flags = %#x, err=%v", flags, err)
	}
	flags |= 1 << uint(PlrNODELETE)
	updated, err := SetCharacterDataFlags(raw, flags)
	if err != nil {
		t.Fatal(err)
	}
	updated, err = RenameCharacterData(updated, "Newname")
	if err != nil {
		t.Fatal(err)
	}
	updated, err = SetCharacterDataClan(updated, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(updated, &fields); err != nil {
		t.Fatal(err)
	}
	var version int
	var name string
	var clanID, clanRank int
	var future map[string]bool
	if err := json.Unmarshal(fields["save_version"], &version); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fields["name"], &name); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fields["clan_id"], &clanID); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fields["clan_rank"], &clanRank); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fields["future_field"], &future); err != nil {
		t.Fatal(err)
	}
	flags, err = CharacterDataFlags(updated)
	if err != nil {
		t.Fatal(err)
	}
	if version != CurrentSaveVersion || name != "Newname" || clanID != 0 || clanRank != 0 || !future["kept"] || flags&(1<<uint(PlrNODELETE)) == 0 || flags&(1<<uint(PrfBrief)) == 0 {
		t.Fatalf("edited character data version=%d name=%q clan=%d/%d future=%v flags=%#x", version, name, clanID, clanRank, future, flags)
	}
}
