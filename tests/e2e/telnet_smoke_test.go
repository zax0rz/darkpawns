// Package e2e contains end-to-end tests that exercise the fully assembled
// Dark Pawns server binary over the wire — not just individual packages.
//
// The first test here, TestTelnetSmoke_GuestEntersWorld, exists because the
// server once shipped two bugs that every unit test passed straight through:
//
//   - main.go wired a nil *db.DB into the db.Database interface, so the
//     advertised "run without a database" path panicked on boot (DP-589).
//   - the telnet layer double-encoded the login envelope, so no telnet client
//     could ever log in.
//
// Both were invisible to in-process unit tests. The only thing that catches
// them is launching the real binary and connecting to it like a player would.
// That is what this test does: build ./cmd/server, run it with NO database,
// telnet in as a guest, and assert we actually land in the world.
package e2e

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// deadDBURL points at a closed port so db.New fails fast, forcing the
// no-database boot path — the exact configuration that used to panic.
const deadDBURL = "postgres://x:x@127.0.0.1:1/nope?sslmode=disable&connect_timeout=1"

func TestTelnetSmoke_GuestEntersWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}

	conn, r := launchAndDial(t)
	defer conn.Close()

	// Handshake: the server greets and asks for a name.
	if got := readUntil(t, conn, r, "By what name", 10*time.Second); got == "" {
		t.Fatal("never received the name prompt")
	}

	// A name beginning with "guest" bypasses passwords and character creation
	// and drops straight into the world as a level-1 Warrior.
	mustWrite(t, conn, "guest\r\n")

	// Read through to the MOTD, which the server sends *after* the
	// look-on-entry room block. By the time "Welcome to Dark Pawns" arrives,
	// the room name, vnum and exits have all already been streamed.
	entered := readUntil(t, conn, r, "Welcome to Dark Pawns", 10*time.Second)
	if entered == "" {
		t.Fatal("guest never entered the world (no MOTD received after login)")
	}
	for _, want := range []string{"Temple Altar", "[8004]", "Exits:"} {
		if !strings.Contains(entered, want) {
			t.Errorf("entry output missing %q\n---\n%s", want, entered)
		}
	}

	// Move east into the adjacent room, then back west. This exercises the
	// full command pipeline end to end. Under the mob-AI deadlock (DP-590)
	// these commands hung forever; readUntil's timeout converts that into a
	// failing test instead of a hang. Room vnums are fixed world data:
	// Temple Altar (8004) ←→ 8160.
	mustWrite(t, conn, "east\r\n")
	if moved := readUntil(t, conn, r, "[8160]", 10*time.Second); moved == "" {
		t.Fatal("`east` did not move guest to room 8160 — command pipeline stalled (deadlock regression?)")
	}
	mustWrite(t, conn, "west\r\n")
	if back := readUntil(t, conn, r, "[8004]", 10*time.Second); back == "" {
		t.Error("`west` did not return guest to room 8004")
	}

	mustWrite(t, conn, "quit\r\n")
}

// TestTelnetSmoke_CharacterCreation drives the full new-character creation
// state machine over telnet with no database, then enters the world. It guards
// two fixes: the input loop must forward a blank line (Enter) as char input so
// the final "PRESS RETURN" step works instead of disconnecting (DP-589), and
// the mob-AI deadlock must stay fixed so the look-on-entry doesn't hang.
func TestTelnetSmoke_CharacterCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}

	conn, r := launchAndDial(t)
	defer conn.Close()

	entered := createWarrior(t, conn, r, "Smoke_Newbie", "secretpw")
	if !strings.Contains(entered, "Smoke_Newbie") {
		t.Errorf("entry output does not name the new character\n---\n%s", entered)
	}

	mustWrite(t, conn, "quit\r\n")
}

// TestTelnetSmoke_Combat creates a Warrior, walks to the busy Temple Square,
// engages an NPC, and verifies the combat engine produces rounds. Mobs wander
// (the AI runs after DP-590), so it looks, targets whatever NPC is present, and
// retries if the target moved on. This is the regression guard that combat does
// not deadlock — the original mob-AI deadlock lived right next to it.
func TestTelnetSmoke_Combat(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}

	conn, r := launchAndDial(t)
	defer conn.Close()

	name := fmt.Sprintf("Brawl%d", time.Now().UnixNano()%100000)
	createWarrior(t, conn, r, name, "brawlpw")

	// Walk to Temple Square [8021] (8004 →south→ 8008 →south→ 8021), which
	// always has NPCs passing through.
	mustWrite(t, conn, "south\r\n")
	if readUntil(t, conn, r, "[8008]", 10*time.Second) == "" {
		t.Fatal("did not reach Temple of the Cross [8008]")
	}
	mustWrite(t, conn, "south\r\n")
	if readUntil(t, conn, r, "[8021]", 10*time.Second) == "" {
		t.Fatal("did not reach Temple Square [8021]")
	}

	// Engage: look, target a present NPC, hit it; retry if it wandered off.
	engaged := false
	for i := 0; i < 8 && !engaged; i++ {
		mustWrite(t, conn, "look\r\n")
		look := readUntil(t, conn, r, "Exits:", 4*time.Second)
		kw := firstMobKeyword(look)
		if kw == "" {
			continue // no NPC visible this tick; look again
		}
		mustWrite(t, conn, "hit "+kw+"\r\n")
		if readUntil(t, conn, r, "You attack", 3*time.Second) != "" {
			engaged = true
		}
	}
	if !engaged {
		t.Fatal("could not engage any NPC in Temple Square after several attempts")
	}

	// The engine fires a round every ~2s. Misses are common, so accept any
	// round outcome. If combat deadlocked we would see nothing and time out.
	rounds := readUntilAny(t, conn, r,
		[]string{"You miss", "You hit", " damage", "hits you", "misses you", "dies", "is dead"},
		12*time.Second)
	if rounds == "" {
		t.Error("engaged a target but observed no combat rounds — engine stalled?")
	}

	mustWrite(t, conn, "quit\r\n")
}

// TestTelnetSmoke_SkillKick creates a Warrior, walks to the busy Temple Square,
// and uses the `kick` skill on an NPC. Warriors are granted kick at level 1 in
// NewCharacter, and the skill pipeline — CanUseSkill (class/level/position gate)
// → DoKick → sendSkillResult — must produce a hit-or-miss message over the wire.
// Note kick needs no active fight: PosStanding (8) already satisfies the
// PosFighting (7) minimum, so a standing player may kick a present target. Mobs
// wander, so it looks, targets whatever NPC is present, and retries.
func TestTelnetSmoke_SkillKick(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}

	conn, r := launchAndDial(t)
	defer conn.Close()

	name := fmt.Sprintf("Kick%d", time.Now().UnixNano()%100000)
	createWarrior(t, conn, r, name, "kickpw")
	walkToTempleSquare(t, conn, r)

	// Look, target a present NPC, kick it; retry if it wandered off. Both
	// outcomes name the skill: "You kick <x> square in the chest!" (hit) or
	// "You try to kick <x>, but miss!" (miss) — match the lowercase "kick" that
	// only the skill emits ("Kick who?" for a missing target is capitalized).
	kicked := false
	for i := 0; i < 8 && !kicked; i++ {
		mustWrite(t, conn, "look\r\n")
		look := readUntil(t, conn, r, "Exits:", 4*time.Second)
		kw := firstMobKeyword(look)
		if kw == "" {
			continue // no NPC visible this tick; look again
		}
		mustWrite(t, conn, "kick "+kw+"\r\n")
		if readUntil(t, conn, r, "kick", 3*time.Second) != "" {
			kicked = true
		}
	}
	if !kicked {
		t.Fatal("could not land a `kick` on any NPC in Temple Square — skill pipeline broken?")
	}

	mustWrite(t, conn, "quit\r\n")
}

// TestTelnetSmoke_SpellCasting creates a level-1 Mage and exercises the spell
// pipeline that was 100% broken until grantClassSpells populated SpellMap (the
// data existed but was never wired in, so `cast` rejected every spell). It
// asserts three things end to end: a known low-level spell casts, a too-high
// spell stays unknown (SpellMap is scoped by level, not blanket-granted), and an
// offensive spell can be thrown at an NPC.
func TestTelnetSmoke_SpellCasting(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}

	conn, r := launchAndDial(t)
	defer conn.Close()

	name := fmt.Sprintf("Mage%d", time.Now().UnixNano()%100000)
	createChar(t, conn, r, name, "magepw", "M")

	// A level-1 Mage knows infravision; the self-cast must succeed. This is the
	// core regression guard: before grantClassSpells, SpellMap was empty and
	// every cast came back "You don't know ...".
	mustWrite(t, conn, "cast infravision\r\n")
	if readUntil(t, conn, r, "You cast 'infravision'", 5*time.Second) == "" {
		t.Fatal("level-1 Mage could not cast infravision — SpellMap not populated (grantClassSpells regression?)")
	}

	// fireball is a level-15 Mage spell; a level-1 Mage must NOT know it yet.
	// This guards that spells are granted by level rather than all at once.
	mustWrite(t, conn, "cast fireball\r\n")
	if readUntil(t, conn, r, "You don't know", 5*time.Second) == "" {
		t.Error("level-1 Mage was allowed to cast the level-15 fireball — spell levels not enforced")
	}

	// Offensive cast at an NPC: walk to Temple Square and throw magic missile.
	// Mobs wander, so look, target a present NPC, and retry. A failed target is
	// refunded, so only a real cast spends mana — the Mage's 100 mana is ample.
	walkToTempleSquare(t, conn, r)
	cast := false
	for i := 0; i < 8 && !cast; i++ {
		mustWrite(t, conn, "look\r\n")
		look := readUntil(t, conn, r, "Exits:", 4*time.Second)
		kw := firstMobKeyword(look)
		if kw == "" {
			continue
		}
		mustWrite(t, conn, "cast 'magic missile' "+kw+"\r\n")
		if readUntil(t, conn, r, "You cast 'magic missile'", 3*time.Second) != "" {
			cast = true
		}
	}
	if !cast {
		t.Fatal("could not cast magic missile at any NPC in Temple Square")
	}

	mustWrite(t, conn, "quit\r\n")
}

// TestTelnetSmoke_PersistenceRoundTrip exercises the database-backed paths that
// have no equivalent without persistence: a new character is saved to the DB,
// then on a fresh connection the returning-player login loads it back. It also
// guards DP-591 (a wrong password used to panic and crash the whole server).
//
// Gated on DP_TEST_DB_URL so the default `go test` (and CI without a database)
// stays green. Point it at a throwaway/local database — the test creates and
// then deletes a uniquely-named character.
func TestTelnetSmoke_PersistenceRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}
	dbURL := os.Getenv("DP_TEST_DB_URL")
	if dbURL == "" {
		t.Skip("set DP_TEST_DB_URL to a test database to run the persistence round-trip")
	}

	// game.ValidName caps names at 20 chars, so keep this short and unique.
	name := fmt.Sprintf("Rt%d", time.Now().UnixNano()%100000000)
	const password = "roundtrippw"
	t.Cleanup(func() { deleteTestPlayer(t, dbURL, name) })

	// --- Connection 1: create the character, which writes it to the DB. ---
	c1, r1 := launchAndDialDB(t, dbURL)
	if got := readUntil(t, c1, r1, "By what name", 10*time.Second); got == "" {
		t.Fatal("conn1: never got name prompt")
	}
	mustWrite(t, c1, name+"\r\n")
	creation := []struct{ awaitPrompt, send string }{
		{"new character?", "y\r\n"}, // DB: "...create a new character?"; no-DB: "Create new character?"
		{"Choose a password", password + "\r\n"},
		{"Confirm password", password + "\r\n"},
		{"Did I get that right", "Y\r\n"},
		// DP-909: the nanny no longer re-prompts for the password — the telnet
		// auth layer already collected it (Choose/Confirm above), so confirming
		// the name jumps straight to the ANSI color stage (single collection,
		// matching C interpreter.c CON_NEWPASSWD/CON_CNFPASSWD).
		{"ANSI color", "N\r\n"},
		{"sex", "M\r\n"},
		{"Race:", "H\r\n"},
		{"Class:", "W\r\n"},
		{"home town", "K\r\n"},
		{"keep these stats", "Y\r\n"},
		{"PRESS RETURN", "\r\n"},
	}
	for _, st := range creation {
		if got := readUntil(t, c1, r1, st.awaitPrompt, 10*time.Second); got == "" {
			t.Fatalf("conn1: creation stalled at %q", st.awaitPrompt)
		}
		mustWrite(t, c1, st.send)
	}
	if got := readUntil(t, c1, r1, "[8004]", 10*time.Second); got == "" {
		t.Fatal("conn1: new character did not enter the world")
	}
	mustWrite(t, c1, "quit\r\n")
	_ = c1.Close()

	// --- Connection 2: wrong password must be rejected, NOT crash (DP-591). ---
	c2, r2 := launchAndDialDB(t, dbURL)
	if got := readUntil(t, c2, r2, "By what name", 10*time.Second); got == "" {
		t.Fatal("conn2: never got name prompt")
	}
	mustWrite(t, c2, name+"\r\n")
	if got := readUntil(t, c2, r2, "Password", 10*time.Second); got == "" {
		t.Fatal("conn2: returning player was not prompted for a password (not loaded from DB?)")
	}
	mustWrite(t, c2, "definitely-wrong\r\n")
	if got := readUntil(t, c2, r2, "Invalid password", 10*time.Second); got == "" {
		t.Error("conn2: wrong password did not produce an 'Invalid password' rejection")
	}
	_ = c2.Close()

	// --- Connection 3: correct password loads the persisted character. ---
	c3, r3 := launchAndDialDB(t, dbURL)
	if got := readUntil(t, c3, r3, "By what name", 10*time.Second); got == "" {
		t.Fatal("conn3: never got name prompt")
	}
	mustWrite(t, c3, name+"\r\n")
	if got := readUntil(t, c3, r3, "Password", 10*time.Second); got == "" {
		t.Fatal("conn3: returning player was not prompted for a password")
	}
	mustWrite(t, c3, password+"\r\n")
	loaded := readUntil(t, c3, r3, "[8004]", 10*time.Second)
	if loaded == "" {
		t.Fatal("conn3: persisted character did not load back into the world")
	}
	if !strings.Contains(loaded, name) {
		t.Errorf("conn3: loaded room output does not name the character\n---\n%s", loaded)
	}
	mustWrite(t, c3, "quit\r\n")
	_ = c3.Close()
}

// --- helpers ---

// deleteTestPlayer removes a character created by the persistence test so the
// (possibly shared) database is left as it was found.
func deleteTestPlayer(t *testing.T, dbURL, name string) {
	t.Helper()
	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Logf("cleanup: open db: %v", err)
		return
	}
	defer conn.Close()
	if _, err := conn.Exec("DELETE FROM players WHERE name = $1", name); err != nil {
		t.Logf("cleanup: delete %s: %v", name, err)
	}
}

// launchAndDial builds the server once (cached across tests), starts a fresh
// instance on free ports with no database, waits for the telnet port, and
// returns a connected client. Cleanup (process kill, log dump on failure) is
// registered on t.
func launchAndDial(t *testing.T) (net.Conn, *bufio.Reader) {
	return launchAndDialDB(t, deadDBURL)
}

// launchAndDialDB is launchAndDial with an explicit database URL, used by the
// persistence test to point at a real database.
func launchAndDialDB(t *testing.T, dbURL string) (net.Conn, *bufio.Reader) {
	t.Helper()
	root := repoRoot(t)
	bin := serverBinary(t, root)
	httpPort := freePort(t)
	telnetPort := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(
		ctx, bin,
		"-world", filepath.Join(root, "lib", "world"),
		"-port", fmt.Sprintf("%d", httpPort),
		"-telnet-port", fmt.Sprintf("%d", telnetPort),
		"-db", dbURL,
	)
	cmd.Dir = root // world loading reads some paths relative to cwd
	// JWT_SECRET is required for token issuance; production sets it via env.
	// Must be >=32 chars so boot validation passes and CI exercises the real
	// issuance path (DP-910), not the silent-failure path.
	cmd.Env = append(os.Environ(), "JWT_SECRET=e2e-smoke-test-secret-at-least-32-chars-long", "ENVIRONMENT=development")
	var logBuf strings.Builder
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("server log:\n%s", logBuf.String())
		}
	})

	conn := dialWhenReady(t, fmt.Sprintf("127.0.0.1:%d", telnetPort), 30*time.Second)
	return conn, bufio.NewReader(conn)
}

// createWarrior drives new-character creation for a Warrior. See createChar.
func createWarrior(t *testing.T, conn net.Conn, r *bufio.Reader, name, password string) string {
	t.Helper()
	return createChar(t, conn, r, name, password, "W")
}

// createChar drives the new-character creation flow to completion as a Human of
// the given class (the letter shown on the class menu: "W" Warrior, "M" Mage,
// …) and returns the room text shown on entry. The name must be <=20 chars to
// satisfy game.ValidName. Human is used because every class is available to it.
// Works against both the DB and no-DB paths (the confirm prompt differs
// slightly, so it keys on the shared substring "new character?").
func createChar(t *testing.T, conn net.Conn, r *bufio.Reader, name, password, class string) string {
	t.Helper()
	if got := readUntil(t, conn, r, "By what name", 10*time.Second); got == "" {
		t.Fatal("never received the name prompt")
	}
	mustWrite(t, conn, name+"\r\n")
	for _, st := range []struct{ awaitPrompt, send string }{
		{"new character?", "y\r\n"}, // create-new confirmation
		{"Choose a password", password + "\r\n"},
		{"Confirm password", password + "\r\n"},
		{"Did I get that right", "Y\r\n"}, // confirm_name
		// DP-909: nanny no longer re-prompts for password (telnet layer
		// already collected it above) — straight to ANSI color.
		{"ANSI color", "N\r\n"},
		{"sex", "M\r\n"},
		{"Race:", "H\r\n"}, // Human
		{"Class:", class + "\r\n"},
		{"home town", "K\r\n"}, // Kir Drax'in
		{"keep these stats", "Y\r\n"},
		{"PRESS RETURN", "\r\n"}, // blank line finalizes (DP-589)
	} {
		if got := readUntil(t, conn, r, st.awaitPrompt, 10*time.Second); got == "" {
			t.Fatalf("character creation stalled at prompt %q", st.awaitPrompt)
		}
		mustWrite(t, conn, st.send)
	}
	entered := readUntil(t, conn, r, "[8004]", 10*time.Second)
	if entered == "" {
		t.Fatal("new character never entered the world")
	}
	return entered
}

// walkToTempleSquare moves a freshly-created character from the Temple Altar
// [8004] down to the busy Temple Square [8021] (8004 →south→ 8008 →south→
// 8021), which always has NPCs passing through to target.
func walkToTempleSquare(t *testing.T, conn net.Conn, r *bufio.Reader) {
	t.Helper()
	mustWrite(t, conn, "south\r\n")
	if readUntil(t, conn, r, "[8008]", 10*time.Second) == "" {
		t.Fatal("did not reach Temple of the Cross [8008]")
	}
	mustWrite(t, conn, "south\r\n")
	if readUntil(t, conn, r, "[8021]", 10*time.Second) == "" {
		t.Fatal("did not reach Temple Square [8021]")
	}
}

// firstMobKeyword extracts a hittable keyword from look output by taking the
// last word of the first "<something> is here." line. Mobs render before
// players in the room block, so the first such line is an NPC.
func firstMobKeyword(look string) string {
	for _, line := range strings.Split(look, "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), "\r")
		idx := strings.Index(line, " is here")
		if idx <= 0 {
			continue
		}
		words := strings.Fields(line[:idx]) // e.g. "a large training monster"
		if len(words) == 0 {
			continue
		}
		kw := strings.Trim(strings.ToLower(words[len(words)-1]), ".,!")
		if kw != "" {
			return kw
		}
	}
	return ""
}

// findRepoRoot walks up from the working directory until it finds go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate repo root (go.mod)")
		}
		dir = parent
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := findRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func mustRepoRoot() string {
	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e:", err)
		os.Exit(1)
	}
	return root
}

// serverBin is the path to the server binary, built once by TestMain and
// shared by every e2e test in this package.
var serverBin string

// TestMain builds ./cmd/server a single time for the whole package (unless
// running with -short, where the e2e tests are skipped anyway).
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	tmp, err := os.MkdirTemp("", "dp-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: temp dir:", err)
		os.Exit(1)
	}
	bin := filepath.Join(tmp, "dp-server")
	build := exec.Command("go", "build", "-o", bin, "./cmd/server")
	build.Dir = mustRepoRoot()
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "e2e: build server: %v\n%s", err, out)
		_ = os.RemoveAll(tmp)
		os.Exit(1)
	}
	serverBin = bin

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

func serverBinary(t *testing.T, _ string) string {
	t.Helper()
	if serverBin == "" {
		t.Fatal("server binary not built (TestMain did not run?)")
	}
	return serverBin
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func dialWhenReady(t *testing.T, addr string, timeout time.Duration) net.Conn {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			return conn
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server never accepted connections on %s within %s", addr, timeout)
	return nil
}

// readUntil reads until the marker appears or the timeout elapses. See
// readUntilAny. Returns "" if the marker never arrives.
func readUntil(t *testing.T, conn net.Conn, r *bufio.Reader, marker string, timeout time.Duration) string {
	return readUntilAny(t, conn, r, []string{marker}, timeout)
}

// readUntilAny reads from the connection, stripping telnet IAC negotiation,
// until any of the markers appears or the timeout elapses. Returns the
// accumulated text once a marker is found, or "" if none arrive in time.
func readUntilAny(t *testing.T, conn net.Conn, r *bufio.Reader, markers []string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var out strings.Builder
	matched := func() bool {
		s := out.String()
		for _, m := range markers {
			if strings.Contains(s, m) {
				return true
			}
		}
		return false
	}
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		b, err := r.ReadByte()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if matched() {
					return out.String()
				}
				continue
			}
			break // EOF or hard error
		}
		if b == 255 { // IAC — consume the negotiation sequence
			consumeIAC(r)
			continue
		}
		out.WriteByte(b)
		if matched() {
			return out.String()
		}
	}
	if matched() {
		return out.String()
	}
	return ""
}

// consumeIAC reads the bytes following an IAC (0xFF) byte: a 3-byte
// WILL/WONT/DO/DONT option, or a subnegotiation terminated by IAC SE.
func consumeIAC(r *bufio.Reader) {
	cmd, err := r.ReadByte()
	if err != nil {
		return
	}
	switch cmd {
	case 251, 252, 253, 254: // WILL, WONT, DO, DONT — followed by one option byte
		_, _ = r.ReadByte()
	case 250: // SB ... IAC SE
		for {
			x, err := r.ReadByte()
			if err != nil {
				return
			}
			if x == 255 {
				if y, err := r.ReadByte(); err != nil || y == 240 {
					return
				}
			}
		}
	}
}

func mustWrite(t *testing.T, conn net.Conn, s string) {
	t.Helper()
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(s)); err != nil {
		t.Fatalf("write %q: %v", strings.TrimSpace(s), err)
	}
}
