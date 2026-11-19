package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestLegacySwitchedCommandRefusalsUseBodyOutput(t *testing.T) {
	w := &World{}
	owner := NewPlayer(1, "Controller", 1001)
	owner.worldRef = w
	mob := NewMob(&parser.Mob{VNum: 9001, Keywords: "horse", ShortDesc: "a horse"}, 1001)
	mob.world = w

	var ownerOutput, bodyOutput strings.Builder
	w.MessageSink = func(name string, message []byte) {
		if name == owner.Name {
			ownerOutput.Write(message)
		}
	}
	w.MobileMessageSink = func(body *MobInstance, message []byte) {
		if body == mob {
			bodyOutput.Write(message)
		}
	}
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == mob {
			return owner
		}
		return nil
	}

	w.doTransform(owner, mob, "transform", "")
	w.doDisplay(owner, mob, "display", "")
	w.doRecall(owner, mob, "recall", "")
	w.doGenWrite(owner, mob, "bug", "report")

	const want = "You aren't transformable!\n\r" +
		"Monsters don't need displays.  Go away.\r\n" +
		"This command is not available for someone of your experience!\r\n" +
		"Monsters can't have ideas - Go away.\r\n"
	if got := bodyOutput.String(); got != want {
		t.Fatalf("switched command body output = %q, want %q", got, want)
	}
	if got := ownerOutput.String(); got != "" {
		t.Fatalf("descriptorless original received output %q", got)
	}
}
