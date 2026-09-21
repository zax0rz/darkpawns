package admin

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

// TestOLCSchemaCoversTelnetFields is the noise-gap pin for the five editor
// families. A field is either a schema field or an explicitly named bespoke
// component; adding a telnet control without adding one of those declarations
// fails here instead of silently leaving the web editor behind.
func TestOLCSchemaCoversTelnetFields(t *testing.T) {
	required := map[string]struct {
		fields  []string
		bespoke []string
	}{
		"room": {
			fields:  []string{"name", "description", "flags", "sector", "script_name", "script_flags"},
			bespoke: []string{"exits", "extra_descriptions"},
		},
		"mob": {
			fields: []string{"keywords", "short_description", "long_description", "detailed_description", "noise", "sex", "hitroll", "damroll", "damage_dice", "damage_sides", "hp_dice", "hp_sides", "hp_plus", "ac", "exp", "gold", "position", "default_position", "attack", "level", "alignment", "race", "action_flags", "affect_flags", "script_name", "script_flags"},
		},
		"obj": {
			fields:  []string{"keywords", "short_description", "long_description", "action_description", "type", "extra_flags", "wear_flags", "weight", "cost", "cost_per_day", "script_name", "script_flags"},
			bespoke: []string{"values", "applies", "extra_descriptions"},
		},
		"shop": {
			fields: []string{"products", "buy_profit", "sell_profit", "keeper", "rooms", "open_hour_1", "open_hour_2", "close_hour_1", "close_hour_2", "flags", "with_who", "message_no_item_keeper", "message_no_item_player", "message_no_buy", "message_no_cash_keeper", "message_no_cash_player", "message_buy", "message_sell", "trade_namelist"},
		},
		"zone": {
			fields:  []string{"name", "lifespan", "reset_mode", "top_room"},
			bespoke: []string{"commands"},
		},
	}

	for kind, want := range required {
		t.Run(kind, func(t *testing.T) {
			schema, ok := olc.SchemaForKind(kind)
			if !ok {
				t.Fatalf("schema kind %q is not registered", kind)
			}
			fields := make(map[string]bool, len(schema.Fields))
			for _, field := range schema.Fields {
				fields[field.Key] = true
			}
			bespoke := make(map[string]bool, len(schema.Bespoke))
			for _, component := range schema.Bespoke {
				bespoke[component] = true
			}
			for _, field := range want.fields {
				if !fields[field] && !bespoke[field] {
					t.Errorf("telnet field %q has no schema field or bespoke component", field)
				}
			}
			for _, component := range want.bespoke {
				if !bespoke[component] {
					t.Errorf("bespoke component %q is not declared by schema", component)
				}
			}
		})
	}
}
