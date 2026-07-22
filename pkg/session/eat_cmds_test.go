package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestApplyAffect(t *testing.T) {
	tests := []struct {
		name     string
		location int
		modifier int
		setup    func(*game.Player)
		check    func(*testing.T, *game.Player)
	}{
		{
			name:     "apply_str",
			location: 1,
			modifier: 3,
			check: func(t *testing.T, p *game.Player) {
				if p.Stats.Str != 18 {
					t.Errorf("STR = %d, want 18", p.Stats.Str)
				}
			},
		},
		{
			name:     "apply_dex",
			location: 2,
			modifier: -2,
			check: func(t *testing.T, p *game.Player) {
				if p.Stats.Dex != 13 {
					t.Errorf("DEX = %d, want 13", p.Stats.Dex)
				}
			},
		},
		{
			name:     "apply_hit_heal",
			location: 12,
			modifier: 30,
			setup: func(p *game.Player) {
				p.Health = 50
				p.MaxHealth = 100
			},
			check: func(t *testing.T, p *game.Player) {
				if p.Health != 80 {
					t.Errorf("Health = %d, want 80 (capped at max)", p.Health)
				}
			},
		},
		{
			name:     "apply_hit_cap_at_max",
			location: 12,
			modifier: 200,
			setup: func(p *game.Player) {
				p.Health = 50
				p.MaxHealth = 100
			},
			check: func(t *testing.T, p *game.Player) {
				if p.Health != 100 {
					t.Errorf("Health = %d, want 100 (capped at max)", p.Health)
				}
			},
		},
		{
			name:     "apply_hit_min_zero",
			location: 12,
			modifier: -200,
			setup: func(p *game.Player) {
				p.Health = 50
				p.MaxHealth = 100
			},
			check: func(t *testing.T, p *game.Player) {
				if p.Health != 0 {
					t.Errorf("Health = %d, want 0 (min zero)", p.Health)
				}
			},
		},
		{
			name:     "apply_mana_heal",
			location: 13,
			modifier: 50,
			setup: func(p *game.Player) {
				p.Mana = 30
				p.MaxMana = 200
			},
			check: func(t *testing.T, p *game.Player) {
				if p.Mana != 80 {
					t.Errorf("Mana = %d, want 80", p.Mana)
				}
			},
		},
		{
			name:     "apply_move_heal",
			location: 14,
			modifier: 25,
			setup: func(p *game.Player) {
				p.Move = 10
				p.MaxMove = 100
			},
			check: func(t *testing.T, p *game.Player) {
				if p.Move != 35 {
					t.Errorf("Move = %d, want 35", p.Move)
				}
			},
		},
		{
			name:     "apply_hitroll",
			location: 17,
			modifier: 5,
			check: func(t *testing.T, p *game.Player) {
				if p.Hitroll != 5 {
					t.Errorf("Hitroll = %d, want 5", p.Hitroll)
				}
			},
		},
		{
			name:     "apply_damroll",
			location: 18,
			modifier: 3,
			check: func(t *testing.T, p *game.Player) {
				if p.Damroll != 3 {
					t.Errorf("Damroll = %d, want 3", p.Damroll)
				}
			},
		},
		{
			name:     "unknown_location",
			location: 99,
			modifier: 10,
			check: func(t *testing.T, p *game.Player) {
				// No stats should change
				if p.Stats.Str != 15 || p.Health != 10 {
					t.Error("unknown location should not modify player")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := game.NewPlayer(1, "Tester", 1001)
			p.Stats.Str = 15
			p.Stats.Dex = 15
			p.Health = 10
			p.MaxHealth = 100
			p.Mana = 10
			p.MaxMana = 100
			p.Move = 10
			p.MaxMove = 100
			p.Hitroll = 0
			p.Damroll = 0
			if tt.setup != nil {
				tt.setup(p)
			}
			applyAffect(p, tt.location, tt.modifier, "test potion")
			if tt.check != nil {
				tt.check(t, p)
			}
		})
	}
}
