package game

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

const (
	mailLifecycleHelperEnv   = "DP_MAIL_LIFECYCLE_HELPER"
	mailLifecyclePhaseEnv    = "DP_MAIL_LIFECYCLE_PHASE"
	mailLifecyclePreserveEnv = "DP_MAIL_LIFECYCLE_PRESERVE_DIR"
	mailLifecycleSenderID    = 101
	mailLifecycleRecipientID = 202
)

const mailLifecycleBody = "proof-mail-body"

// TestMailLookupFallbackWithoutInitialization records the deliberate no-DB
// boundary. InitMailSystem is the only writer for these hooks; the production
// no-DB configuration does not call it because it has no persistent identity
// authority.
func TestMailLookupFallbackWithoutInitialization(t *testing.T) {
	oldNameFunc, oldIDFunc := worldNameFunc, worldIDFunc
	worldNameFunc, worldIDFunc = nil, nil
	t.Cleanup(func() {
		worldNameFunc, worldIDFunc = oldNameFunc, oldIDFunc
	})

	if got := GetIDByName("Recipient"); got != -1 {
		t.Fatalf("uninitialized GetIDByName = %d, want -1", got)
	}
	if got := GetNameByID(mailLifecycleRecipientID); got != "Player(202)" {
		t.Fatalf("uninitialized GetNameByID = %q, want fallback", got)
	}
}

// TestMailLifecycleAcrossProcessRestart_HelperLevel runs the smallest Go-only
// mail vehicle with explicit InitMailSystem calls. Each phase is a distinct
// test process, so the restart phase cannot reuse package globals from send.
// This is the Go-native control for the production restart vehicle.
func TestMailLifecycleAcrossProcessRestart_HelperLevel(t *testing.T) {
	fixtureRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(fixtureRoot, "data"), 0o700); err != nil {
		t.Fatalf("create isolated data directory: %v", err)
	}

	sendOutput := runMailLifecycleHelper(t, fixtureRoot, "send")
	for _, want := range []string{
		"phase=send completed=true",
		"postmaster_assigned=true",
		"mail_file_bytes=512",
	} {
		if !strings.Contains(sendOutput, want) {
			t.Fatalf("send phase missing %q\n%s", want, sendOutput)
		}
	}

	restartOutput := runMailLifecycleHelper(t, fixtureRoot, "restart")
	for _, want := range []string{
		"phase=restart init_scan=true",
		"check=$n tells you, 'You have mail waiting.'",
		"receive=$n gives you a piece of mail.",
		"received_body=true",
		"second_check=$n tells you, 'Sorry, you don't have any mail waiting.'",
		"second_receive=$n tells you, 'Sorry, you don't have any mail waiting.'",
		"receive_inventory_items=1",
		"mail_remaining=false",
	} {
		if !strings.Contains(restartOutput, want) {
			t.Fatalf("restart phase missing %q\n%s", want, restartOutput)
		}
	}

	if preserveDir := os.Getenv(mailLifecyclePreserveEnv); preserveDir != "" {
		if err := preserveMailFixture(fixtureRoot, preserveDir); err != nil {
			t.Fatalf("preserve Go fixture: %v", err)
		}
		t.Logf("preserved Go native fixture under %s", preserveDir)
	}
}

// TestMailHelperLifecycleSameProcess is intentionally separate from the
// restart test. It proves only the explicit-initialization helper boundary:
// the assigned postmaster can complete a short message and the same process
// can check, receive, and consume it once.
func TestMailHelperLifecycleSameProcess(t *testing.T) {
	output := runMailLifecycleHelper(t, t.TempDir(), "same-process")
	for _, want := range []string{
		"phase=same-process completed=true",
		"postmaster_assigned=true",
		"check=$n tells you, 'You have mail waiting.'",
		"receive=$n gives you a piece of mail.",
		"received_body=true",
		"consumed_once=true",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("same-process helper missing %q\n%s", want, output)
		}
	}
}

func runMailLifecycleHelper(t *testing.T, fixtureRoot, phase string) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run", "^TestMailLifecycleHelperProcess$", "-test.v")
	cmd.Dir = fixtureRoot
	cmd.Env = append(os.Environ(),
		mailLifecycleHelperEnv+"=1",
		mailLifecyclePhaseEnv+"="+phase,
		"DP_FIXED_TIME=650337471",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mail helper phase %s: %v\n%s", phase, err, output)
	}
	if os.Getenv("DP_MAIL_LIFECYCLE_TRACE") == "1" {
		t.Logf("helper phase %s output:\n%s", phase, output)
	}
	return string(output)
}

// TestMailLifecycleHelperProcess is the child-process vehicle for the parent
// tests above. It is never a production boot proof: every phase explicitly
// calls InitMailSystem and uses a disposable CWD-relative data/mail path.
func TestMailLifecycleHelperProcess(t *testing.T) {
	if os.Getenv(mailLifecycleHelperEnv) != "1" {
		t.Skip("child-process helper")
	}

	phase := os.Getenv(mailLifecyclePhaseEnv)
	switch phase {
	case "send":
		runMailSendPhase(t)
	case "restart":
		runMailRestartPhase(t)
	case "same-process":
		runMailSameProcessPhase(t)
	default:
		t.Fatalf("unknown mail helper phase %q", phase)
	}
}

func runMailSendPhase(t *testing.T) {
	t.Helper()
	w, sender, postmasterFn, postmasterMob, _ := newMailLifecycleWorld(t)
	if !InitMailSystem(mailLifecycleNameByID, mailLifecycleIDByName) {
		t.Fatal("mail initialization failed")
	}

	postmasterFn(w, sender, postmasterMob, "mail", "Recipient body")
	if !HandleMailInput(sender, mailLifecycleBody) {
		// Expected: the first body line keeps the editor open.
	} else {
		t.Fatal("body line unexpectedly completed mail composition")
	}
	if !HandleMailInput(sender, "@") {
		t.Fatal("@ did not complete mail composition")
	}

	stat, err := os.Stat(MailFile)
	if err != nil {
		t.Fatalf("stat isolated mail file: %v", err)
	}
	fmt.Printf("phase=send completed=true postmaster_assigned=true mail_file_bytes=%d\n", stat.Size())
}

func runMailRestartPhase(t *testing.T) {
	w, _, postmasterFn, postmasterMob, messages := newMailLifecycleWorld(t)
	if !InitMailSystem(mailLifecycleNameByID, mailLifecycleIDByName) {
		t.Fatal("mail initialization failed")
	}

	recipient := NewPlayer(mailLifecycleRecipientID, "Recipient", 1001)
	if err := w.AddPlayer(recipient); err != nil {
		t.Fatalf("add recipient after restart: %v", err)
	}
	postmasterFn(w, recipient, postmasterMob, "check", "")
	checkOutput := mailLifecycleLastMessage(messages, recipient.Name)
	postmasterFn(w, recipient, postmasterMob, "receive", "")
	receiveOutput := mailLifecycleLastMessage(messages, recipient.Name)
	if len(recipient.Inventory.Items) != 1 {
		t.Fatalf("restart receive inventory items = %d, want 1", len(recipient.Inventory.Items))
	}
	receivedBody := strings.Contains(recipient.Inventory.Items[0].Runtime.MailText, mailLifecycleBody)
	postmasterFn(w, recipient, postmasterMob, "check", "")
	secondCheckOutput := mailLifecycleLastMessage(messages, recipient.Name)
	postmasterFn(w, recipient, postmasterMob, "receive", "")
	secondReceiveOutput := mailLifecycleLastMessage(messages, recipient.Name)
	stat, err := os.Stat(MailFile)
	if err != nil {
		t.Fatalf("stat mail file after restart: %v", err)
	}
	fmt.Printf("phase=restart init_scan=true mail_file_bytes=%d check=%s receive=%s received_body=%t second_check=%s second_receive=%s receive_inventory_items=%d mail_remaining=%t\n",
		stat.Size(), checkOutput, receiveOutput, receivedBody, secondCheckOutput, secondReceiveOutput, len(recipient.Inventory.Items), hasMail(mailLifecycleRecipientID))
}

func runMailSameProcessPhase(t *testing.T) {
	if err := os.Mkdir("data", 0o700); err != nil {
		t.Fatalf("create same-process data directory: %v", err)
	}
	w, sender, postmasterFn, postmasterMob, messages := newMailLifecycleWorld(t)
	if !InitMailSystem(mailLifecycleNameByID, mailLifecycleIDByName) {
		t.Fatal("mail initialization failed")
	}

	postmasterFn(w, sender, postmasterMob, "mail", "Recipient body")
	_ = HandleMailInput(sender, mailLifecycleBody)
	if !HandleMailInput(sender, "@") {
		t.Fatal("@ did not complete same-process mail composition")
	}

	recipient := NewPlayer(mailLifecycleRecipientID, "Recipient", 1001)
	if err := w.AddPlayer(recipient); err != nil {
		t.Fatalf("add recipient: %v", err)
	}
	postmasterFn(w, recipient, postmasterMob, "check", "")
	checkOutput := mailLifecycleLastMessage(messages, recipient.Name)
	postmasterFn(w, recipient, postmasterMob, "receive", "")
	receiveOutput := mailLifecycleLastMessage(messages, recipient.Name)

	if len(recipient.Inventory.Items) != 1 {
		t.Fatalf("same-process receive inventory items = %d, want 1", len(recipient.Inventory.Items))
	}
	receivedBody := strings.Contains(recipient.Inventory.Items[0].Runtime.MailText, mailLifecycleBody)
	consumedOnce := !hasMail(mailLifecycleRecipientID)
	fmt.Printf("phase=same-process completed=true postmaster_assigned=true check=%s receive=%s received_body=%t consumed_once=%t\n",
		checkOutput, receiveOutput, receivedBody, consumedOnce)
}

func newMailLifecycleWorld(t *testing.T) (*World, *Player, SpecFunc, *MobInstance, map[string][]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Mail proof room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 3010, ShortDesc: "the postman", Level: 30}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	messages := make(map[string][]string)
	w.MessageSink = func(playerName string, msg []byte) {
		messages[playerName] = append(messages[playerName], string(msg))
	}

	mobFn := GetMobSpec(3010)
	if mobFn == nil {
		t.Fatal("VNum 3010 has no assigned postmaster procedure")
	}
	postmasterMob := &MobInstance{VNum: 3010, Status: "standing"}
	sender := NewPlayer(mailLifecycleSenderID, "Sender", 1001)
	sender.Level = MailMinLevel
	sender.Gold = MailStampPrice + 1
	if err := w.AddPlayer(sender); err != nil {
		t.Fatalf("add sender: %v", err)
	}
	return w, sender, mobFn, postmasterMob, messages
}

func mailLifecycleNameByID(id int) string {
	switch id {
	case mailLifecycleSenderID:
		return "Sender"
	case mailLifecycleRecipientID:
		return "Recipient"
	default:
		return fmt.Sprintf("Player(%d)", id)
	}
}

func mailLifecycleIDByName(name string) int {
	if strings.EqualFold(name, "Recipient") {
		return mailLifecycleRecipientID
	}
	return -1
}

func mailLifecycleLastMessage(messages map[string][]string, playerName string) string {
	entries := messages[playerName]
	if len(entries) == 0 {
		return ""
	}
	return strings.TrimSpace(entries[len(entries)-1])
}

func preserveMailFixture(fixtureRoot, preserveRoot string) error {
	mailBytes, err := os.ReadFile(filepath.Join(fixtureRoot, MailFile))
	if err != nil {
		return err
	}
	mailDir := filepath.Join(preserveRoot, "go-native", "data")
	if err := os.MkdirAll(mailDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(mailDir, "mail"), mailBytes, 0o600)
}
