package olc

import "testing"

func TestCollapsedMobFlagVocabularyLengthsAgree(t *testing.T) {
	tests := []struct {
		name         string
		flags        []FlagVocabulary
		storage      []string
		display      []string
		wantEditable int
		wantStorage  int
	}{
		{name: "action", flags: MobActionFlags, storage: MobActionStorageNames, display: MobActionDisplayNames, wantEditable: 25, wantStorage: 26},
		{name: "affect", flags: MobAffectFlags, storage: MobAffectStorageNames, display: MobAffectDisplayNames, wantEditable: 37, wantStorage: 39},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if len(test.flags) != test.wantEditable {
				t.Fatalf("editable flag count = %d, want %d", len(test.flags), test.wantEditable)
			}
			if len(test.storage) != test.wantStorage {
				t.Fatalf("storage flag count = %d, want %d", len(test.storage), test.wantStorage)
			}
			if len(test.flags) != len(test.display) {
				t.Fatalf("collapsed storage/display lengths disagree: flags=%d display=%d", len(test.flags), len(test.display))
			}
			for i, flag := range test.flags {
				if flag.Bit != i {
					t.Errorf("flag %d has bit %d", i, flag.Bit)
				}
				if test.storage[i] != flag.Storage {
					t.Errorf("storage[%d] = %q, want %q", i, test.storage[i], flag.Storage)
				}
				if test.display[i] != flag.Label {
					t.Errorf("display[%d] = %q, want %q", i, test.display[i], flag.Label)
				}
			}
		})
	}
}

func TestObjectValueMatrixCoversEveryItemTypeAndSlot(t *testing.T) {
	if got, want := len(ObjectValueMatrix), len(ItemTypeNames); got != want {
		t.Fatalf("matrix rows = %d, want %d item types", got, want)
	}
	for itemType, row := range ObjectValueMatrix {
		if row.ItemType != itemType {
			t.Errorf("matrix row %d has item type %d", itemType, row.ItemType)
		}
		if len(row.Values) != 4 {
			t.Errorf("item type %d has %d value slots", itemType, len(row.Values))
		}
	}
}
