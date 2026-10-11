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
	// C's write path emits the string's own spelling verbatim
	// (send_to_char src/comm.c:2335 → write_to_output :1348 →
	// write_to_descriptor :1380): the LFCR pair renders as written.
	if got := renderedOutput(s); got != "owned body\n\r" {
		t.Fatalf("concrete mobile output = %q", got)
	}
	s.DetachTransport()
	m.world.MobileMessageSink(body, []byte("detached\n\r"))
	if got := renderedOutput(s); got != "" {
		t.Fatalf("detached mobile output = %q", got)
	}
}
