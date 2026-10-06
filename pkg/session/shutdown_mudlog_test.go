package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestShutdownForceMudlog(t *testing.T) {
	for _, level := range []int{39, 40} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			m, a, v, w := wizardLogFixture(t)
			a.player.SetLevel(level)
			file := captureMudlogFile(t)
			ready := false
			game.SetLogWriter(&flagProbe{buf: file, when: func() { ready = len(a.send) == 2 && len(v.send) == 1 }})
			if err := executeCommand(a, "shutdown", []string{"reboot"}, false); err != nil {
				t.Fatal(err)
			}
			payload := "(GC) Logactor forced all to save"
			if !ready || !strings.Contains(file.String(), payload) {
				t.Fatalf("shutdown force must log after ack before victims: %q", file.String())
			}
			victimOutput := strings.Join(drainSessionText(t, v), "")
			if !strings.Contains(victimOutput, "Logactor has forced you to 'save'.\r\n") {
				t.Fatalf("shutdown command remainder: %q", victimOutput)
			}
			got := strings.Join(drainSessionText(t, w), "")
			if strings.Contains(got, "[ "+payload+" ]\r\n") != (level == 39) {
				t.Fatalf("observer threshold at actor level %d: %q", level, got)
			}
			select {
			case request := <-m.ShutdownRequests():
				if request.Marker != ".fastboot" {
					t.Fatal("wrong marker")
				}
			default:
				t.Fatal("missing shutdown request")
			}
		})
	}
}
