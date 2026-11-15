package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCharacterKujiKiriMobileGatesAndDrawParity(t *testing.T) {
	tests := []struct {
		name      string
		level     int
		setup     func(*MobInstance)
		want      string
		wantDraws int
	}{
		{
			name:  "class gate",
			level: 9,
			want:  "You know nothing of kuji-kiri!\r\n",
		},
		{
			name:  "ninja class bypasses class gate",
			level: 9,
			setup: func(mob *MobInstance) {
				class := ClassNinja
				mob.Runtime.ClassOverride = &class
			},
			want:      "You have not mastered this art yet!\r\n",
			wantDraws: 1,
		},
		{
			name:  "fighting gate",
			level: LVL_IMMORT,
			setup: func(mob *MobInstance) {
				mob.SetFightingBody(NewMob(&parser.Mob{VNum: 9002, Keywords: "target"}, 1002))
			},
			want: "You are too busy fighting to practice kuji-kiri!\r\n",
		},
		{
			name:  "affect gate",
			level: LVL_IMMORT,
			setup: func(mob *MobInstance) { mob.SetAffected(affKujiKiri) },
			want:  "You can not practice kuji-kiri again right now!\r\n",
		},
		{
			name:  "mounted gate",
			level: LVL_IMMORT,
			setup: func(mob *MobInstance) { mob.SetAffected(affMounted) },
			want:  "Dismount first!\r\n",
		},
		{
			name:      "dummy mob skill",
			level:     LVL_IMMORT,
			want:      "You have not mastered this art yet!\r\n",
			wantDraws: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mob := NewMob(&parser.Mob{VNum: 9001, Keywords: "body", Level: tt.level}, 1001)
			if tt.setup != nil {
				tt.setup(mob)
			}

			var output string
			ctx := CharacterCommandContext{
				Actor: mob,
				PrivateOutput: func(message string) {
					output += message
				},
			}
			const seed = 9407
			dprng.ResetStream(seed)
			wantStream := dprng.New(seed)
			for range tt.wantDraws {
				wantStream.Number(1, 101)
			}

			if got := CharacterKujiKiri(ctx, SkillKkRin); got != CharacterCommandHandled {
				t.Fatalf("result = %v, want handled", got)
			}
			if output != tt.want {
				t.Errorf("output = %q, want %q", output, tt.want)
			}
			if got, want := dprng.Next(), wantStream.Next(); got != want {
				t.Errorf("next RNG value = %d, want %d (draws=%d)", got, want, tt.wantDraws)
			}
		})
	}
}
