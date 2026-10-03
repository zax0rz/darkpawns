package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMobileEquipmentOutputConcreteDescriptor(t *testing.T) {
	m, s, _ := switchGateFixture(t)
	body := game.NewMob(&parser.Mob{VNum: 7, ShortDesc: "a duplicate"}, 1001)
	other := game.NewMob(&parser.Mob{VNum: 7, ShortDesc: "a duplicate"}, 1001)
	renderedOutput(s)
	m.world.MobileMessageSink(body, []byte("ordinary\n\r"))
	if got := renderedOutput(s); got != "" {
		t.Fatalf("descriptorless mobile output = %q", got)
	}
	m.mu.Lock()
	s.isSwitched = true
	s.switchedMob = body
	m.mu.Unlock()
	m.playerLifecycleMu.Lock()
	m.world.MobileMessageSink(other, []byte("wrong body\n\r"))
	m.world.MobileMessageSink(body, []byte("owned body\n\r"))
	m.playerLifecycleMu.Unlock()
	if got := renderedOutput(s); got != "owned body\r\n" {
		t.Fatalf("concrete mobile output = %q", got)
	}
	s.DetachTransport()
	m.world.MobileMessageSink(body, []byte("detached\n\r"))
	if got := renderedOutput(s); got != "" {
		t.Fatalf("detached mobile output = %q", got)
	}
}
