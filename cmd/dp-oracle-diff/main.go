// Command dp-oracle-diff launches the C oracle and Go port, drives both over
// telnet with one scenario, and reports their normalized transcript divergence.
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

const (
	// fixedTime pins the game calendar on both engines (DP_FIXED_TIME seam,
	// orthogonal to DP_CLOCK's pulse freeze). 1770838461 => 2pm/daytime,
	// {hours:14,day:17,month:8,year:1245}; both derive an identical calendar.
	fixedTime    = "1770838461"
	settlePulses = 40                                                                  // C PULSE_MOBILE: births a newbie exactly once.
	deadDBURL    = "postgres://x:x@127.0.0.1:1/nope?sslmode=disable&connect_timeout=1" // #nosec G101 -- deliberately unreachable placeholder DSN (dev oracle-diff harness), not a real credential
)

//go:embed scenarios/*.txt
var scenarioFiles embed.FS

type process struct {
	name    string
	cmd     *exec.Cmd
	done    chan struct{}
	waitErr error
	log     *safeBuffer
}

type safeBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func main() {
	os.Exit(run())
}

func run() int {
	var (
		scenarioName = flag.String("scenario", "look-start-room", "scenario name from scenarios/<name>.txt")
		seed         = flag.String("seed", "1", "shared deterministic DP_SEED value")
		showOracle   = flag.Bool("show-oracle", false, "print normalized C blocks even when both implementations match")
		showGoLog    = flag.Bool("show-go-log", false, "print the Go port server log after the report (debugging aid)")
		quiescence   = flag.Duration("quiescence", 300*time.Millisecond, "silence interval that marks the end of an output burst")
		bootTimeout  = flag.Duration("boot-timeout", 30*time.Second, "maximum wait for each telnet listener")
	)
	flag.Parse()
	if _, err := strconv.ParseUint(*seed, 10, 64); err != nil {
		fmt.Fprintln(os.Stderr, "dp-oracle-diff: seed must be an unsigned integer")
		return 1
	}

	oracleBin := os.Getenv("DP_ORACLE_BIN")
	if oracleBin == "" {
		fmt.Println("SKIP: DP_ORACLE_BIN is unset; C oracle differential run not available")
		return 0
	}
	if err := execute(*scenarioName, *quiescence, *bootTimeout, oracleBin, *seed, *showOracle, *showGoLog); err != nil {
		fmt.Fprintln(os.Stderr, "dp-oracle-diff:", err)
		return 1
	}
	return 0
}

func execute(scenarioName string, quiescence, bootTimeout time.Duration, oracleBin, seed string, showOracle, showGoLog bool) error {
	if quiescence <= 0 {
		return errors.New("quiescence must be positive")
	}
	scenarioFile := filepath.ToSlash(filepath.Join("scenarios", scenarioName+".txt"))
	f, err := scenarioFiles.Open(scenarioFile)
	if err != nil {
		return fmt.Errorf("open scenario %q: %w", scenarioName, err)
	}
	defer func() { _ = f.Close() }()
	scenario, err := oraclediff.ParseScenario(scenarioName, f)
	if err != nil {
		return err
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "dp-oracle-diff-")
	if err != nil {
		return fmt.Errorf("create work directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	goBin := filepath.Join(tmp, "dp-server")
	build := exec.Command("go", "build", "-o", goBin, "./cmd/server")
	build.Dir = repoRoot
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		return fmt.Errorf("build Go server: %w\n%s", buildErr, output)
	}

	oracleRoot, err := findOracleRoot(oracleBin)
	if err != nil {
		return err
	}
	oracleData := filepath.Join(tmp, "oracle-lib")
	if err := prepareOracleData(filepath.Join(oracleRoot, "lib"), oracleData, scenario.EmptyPlayers); err != nil {
		return err
	}
	// The copied data directory is disposable, so character creation never
	// mutates the oracle clone. Unless empty-players is requested, keep its
	// baseline player file so existing mortal scenarios retain today's boot.
	goWorld := filepath.Join(repoRoot, "lib", "world")
	if len(scenario.Fixtures) > 0 || len(scenario.ObjectSpawns) > 0 || len(scenario.MobFixtures) > 0 || len(scenario.MobAffFixtures) > 0 || len(scenario.QuietZones) > 0 || scenario.QuietAllMobs || len(scenario.ScriptlessMobIDs) > 0 || len(scenario.RoomExitFixtures) > 0 || len(scenario.RoomFlagFixtures) > 0 || len(scenario.RoomSectors) > 0 || len(scenario.ForceLoadVNums) > 0 {
		goWorld = filepath.Join(tmp, "go-world")
		if err := os.CopyFS(goWorld, os.DirFS(filepath.Join(repoRoot, "lib", "world"))); err != nil {
			return fmt.Errorf("copy Go world to throwaway directory: %w", err)
		}
		// Sibling lib/text rides along: the server derives help (and future
		// static text) from the -world dir's parent, so the throwaway layout
		// must mirror lib/{world,text}.
		if err := os.CopyFS(filepath.Join(tmp, "text"), os.DirFS(filepath.Join(repoRoot, "lib", "text"))); err != nil {
			return fmt.Errorf("copy lib/text to throwaway directory: %w", err)
		}
		if err := applyObjectFixtures(filepath.Join(oracleData, "world"), scenario.Fixtures); err != nil {
			return fmt.Errorf("apply C oracle fixtures: %w", err)
		}
		if err := applyObjectFixtures(goWorld, scenario.Fixtures); err != nil {
			return fmt.Errorf("apply Go port fixtures: %w", err)
		}
		if err := applyObjectSpawnFixtures(filepath.Join(oracleData, "world"), scenario.ObjectSpawns); err != nil {
			return fmt.Errorf("apply C oracle object spawn fixtures: %w", err)
		}
		if err := applyObjectSpawnFixtures(goWorld, scenario.ObjectSpawns); err != nil {
			return fmt.Errorf("apply Go port object spawn fixtures: %w", err)
		}
		if err := applyForceLoadFixtures(filepath.Join(oracleData, "world"), scenario.ForceLoadVNums); err != nil {
			return fmt.Errorf("apply C oracle force-load fixtures: %w", err)
		}
		if err := applyForceLoadFixtures(goWorld, scenario.ForceLoadVNums); err != nil {
			return fmt.Errorf("apply Go port force-load fixtures: %w", err)
		}
		if err := applyQuietZoneFixtures(filepath.Join(oracleData, "world"), scenario.QuietZones); err != nil {
			return fmt.Errorf("apply C oracle quiet-zone fixtures: %w", err)
		}
		if err := applyQuietZoneFixtures(goWorld, scenario.QuietZones); err != nil {
			return fmt.Errorf("apply Go port quiet-zone fixtures: %w", err)
		}
		if scenario.QuietAllMobs {
			if err := applyQuietMobFixtures(filepath.Join(oracleData, "world")); err != nil {
				return fmt.Errorf("apply C oracle quiet-mob fixture: %w", err)
			}
			if err := applyQuietMobFixtures(goWorld); err != nil {
				return fmt.Errorf("apply Go port quiet-mob fixture: %w", err)
			}
		}
		if err := applyMobFixtures(filepath.Join(oracleData, "world"), scenario.MobFixtures); err != nil {
			return fmt.Errorf("apply C oracle mob fixtures: %w", err)
		}
		if err := applyMobFixtures(goWorld, scenario.MobFixtures); err != nil {
			return fmt.Errorf("apply Go port mob fixtures: %w", err)
		}
		if err := applyMobAffFixtures(filepath.Join(oracleData, "world"), scenario.MobAffFixtures); err != nil {
			return fmt.Errorf("apply C oracle mob aff fixtures: %w", err)
		}
		if err := applyMobAffFixtures(goWorld, scenario.MobAffFixtures); err != nil {
			return fmt.Errorf("apply Go port mob aff fixtures: %w", err)
		}
		if err := applyScriptlessMobFixtures(filepath.Join(oracleData, "world"), scenario.ScriptlessMobIDs); err != nil {
			return fmt.Errorf("apply C oracle mob script fixtures: %w", err)
		}
		if err := applyScriptlessMobFixtures(goWorld, scenario.ScriptlessMobIDs); err != nil {
			return fmt.Errorf("apply Go port mob script fixtures: %w", err)
		}
		if err := applyRoomFixtures(filepath.Join(oracleData, "world"), scenario.RoomExitFixtures, scenario.RoomFlagFixtures, scenario.RoomSectors); err != nil {
			return fmt.Errorf("apply C oracle room fixtures: %w", err)
		}
		if err := applyRoomFixtures(goWorld, scenario.RoomExitFixtures, scenario.RoomFlagFixtures, scenario.RoomSectors); err != nil {
			return fmt.Errorf("apply Go port room fixtures: %w", err)
		}
	}

	goWork := filepath.Join(tmp, "go-work")
	if err := os.MkdirAll(filepath.Join(goWork, "data"), 0o750); err != nil {
		return fmt.Errorf("create Go runtime directory: %w", err)
	}
	oraclePort, goTelnetPort, goHTTPPort, err := allocatePorts()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	oracleProc, err := startProcess(ctx, "C oracle", oracleRoot,
		append(os.Environ(), "DP_SEED="+seed, "DP_CLOCK=1", "DP_FIXED_TIME="+fixedTime), oracleBin, "-d", oracleData, fmt.Sprint(oraclePort))
	if err != nil {
		return err
	}
	defer oracleProc.stop()

	goEnv := append(
		os.Environ(),
		"DP_SEED="+seed,
		"DP_CLOCK=1",
		"DP_FIXED_TIME="+fixedTime,
		"JWT_SECRET=oracle-diff-secret-at-least-32-characters-long",
		"ENVIRONMENT=development",
	)
	goEnv = withFreshMUDEnv(goEnv, scenario.EmptyPlayers)
	goProc, err := startProcess(
		ctx, "Go port", goWork,
		goEnv,
		goBin,
		"-world", goWorld,
		"-port", fmt.Sprint(goHTTPPort),
		"-telnet-port", fmt.Sprint(goTelnetPort),
		"-db", deadDBURL,
	)
	if err != nil {
		return err
	}
	defer goProc.stop()

	oracleAddr := fmt.Sprintf("127.0.0.1:%d", oraclePort)
	goAddr := fmt.Sprintf("127.0.0.1:%d", goTelnetPort)
	oracleNetConn, err := dialWhenReady(oracleProc, oracleAddr, bootTimeout)
	if err != nil {
		return err
	}
	oracleConn := oraclediff.NewTCPConn(oracleNetConn)
	defer func() { _ = oracleConn.Close() }()
	goNetConn, err := dialWhenReady(goProc, goAddr, bootTimeout)
	if err != nil {
		return err
	}
	if err := waitForLog(goProc, "World state restored", bootTimeout); err != nil {
		return err
	}
	goConn := oraclediff.NewTCPConn(goNetConn)
	defer func() { _ = goConn.Close() }()

	oracleSetup, err := oraclediff.RunSetupAndSettle(oracleConn, scenario.SetupOracle, settlePulses, quiescence)
	if err != nil {
		return fmt.Errorf("run C oracle setup: %w\nserver log:\n%s", err, oracleProc.log.String())
	}
	goSetup, err := oraclediff.RunSetupAndSettle(goConn, scenario.SetupPort, settlePulses, quiescence)
	if err != nil {
		return fmt.Errorf("run Go port setup: %w\nserver log:\n%s", err, goProc.log.String())
	}

	oraclePeers := make(map[string]oraclediff.Conn, len(scenario.Peers))
	goPeers := make(map[string]oraclediff.Conn, len(scenario.Peers))
	peerNames := make([]string, 0, len(scenario.Peers))
	for name := range scenario.Peers {
		peerNames = append(peerNames, name)
	}
	sort.Strings(peerNames)
	for _, name := range peerNames {
		peer := scenario.Peers[name]
		oraclePeerNet, dialErr := dialWhenReady(oracleProc, oracleAddr, bootTimeout)
		if dialErr != nil {
			return fmt.Errorf("dial C oracle %s: %w", name, dialErr)
		}
		oraclePeer := oraclediff.NewTCPConn(oraclePeerNet)
		defer func() { _ = oraclePeer.Close() }()
		if _, setupErr := oraclediff.RunSetupAndSettle(oraclePeer, peer.SetupOracle, settlePulses, quiescence); setupErr != nil {
			return fmt.Errorf("run C oracle %s setup: %w\nserver log:\n%s", name, setupErr, oracleProc.log.String())
		}
		oraclePeers[name] = oraclePeer

		goPeerNet, dialErr := dialWhenReady(goProc, goAddr, bootTimeout)
		if dialErr != nil {
			return fmt.Errorf("dial Go port %s: %w", name, dialErr)
		}
		goPeer := oraclediff.NewTCPConn(goPeerNet)
		defer func() { _ = goPeer.Close() }()
		if _, setupErr := oraclediff.RunSetupAndSettle(goPeer, peer.SetupPort, settlePulses, quiescence); setupErr != nil {
			return fmt.Errorf("run Go port %s setup: %w\nserver log:\n%s", name, setupErr, goProc.log.String())
		}
		goPeers[name] = goPeer
	}

	// Peer arrivals can emit room text to clients that completed setup earlier.
	// Clear that non-probe output so each compared block starts at its command.
	if err := drainClients(quiescence, oracleConn, oraclePeers); err != nil {
		return fmt.Errorf("drain C oracle setup output: %w", err)
	}
	if err := drainClients(quiescence, goConn, goPeers); err != nil {
		return fmt.Errorf("drain Go port setup output: %w", err)
	}
	if err := oraclediff.RunWarmup(oracleConn, oraclePeers, scenario.Warmup, quiescence); err != nil {
		return fmt.Errorf("run C oracle warmup: %w", err)
	}
	if err := oraclediff.RunWarmup(goConn, goPeers, scenario.Warmup, quiescence); err != nil {
		return fmt.Errorf("run Go port warmup: %w", err)
	}

	oracleActor, oracleAudience := probeClients(oracleConn, oraclePeers, scenario.ProbeActor)
	goActor, goAudience := probeClients(goConn, goPeers, scenario.ProbeActor)
	oracleBlocks, err := oraclediff.RunAudienceProbe(oracleActor, oracleAudience, scenario.Probe, quiescence)
	if err != nil {
		return fmt.Errorf("run C oracle probe: %w\nserver log:\n%s", err, oracleProc.log.String())
	}
	goBlocks, err := oraclediff.RunAudienceProbe(goActor, goAudience, scenario.Probe, quiescence)
	if err != nil {
		return fmt.Errorf("run Go port probe: %w\nserver log:\n%s", err, goProc.log.String())
	}
	diffs := make([]oraclediff.BlockDiff, 0, len(oracleBlocks)+1)
	// Character-creation coverage: diff the whole normalized setup transcript
	// (the nanny dialogue) as one block when the scenario opts in via
	// [creation:oracle]/[creation:port].
	if scenario.DiffSetup {
		oracleCreation := oraclediff.Normalize(oracleSetup)
		goCreation := oraclediff.Normalize(goSetup)
		diffs = append(diffs, oraclediff.BlockDiff{
			Command: "creation",
			Oracle:  oracleCreation,
			Go:      goCreation,
			Diff:    oraclediff.UnifiedDiff("c-oracle", "go-port", oracleCreation, goCreation),
		})
	}
	for i, oracleResult := range oracleBlocks {
		var oracleBlock, goBlock string
		oracleBlock = oraclediff.Normalize(oracleResult.Output)
		if i < len(goBlocks) {
			goBlock = oraclediff.Normalize(goBlocks[i].Output)
		}
		label := oracleResult.Command
		if len(scenario.Peers) > 0 {
			label = fmt.Sprintf("%s [%s]", oracleResult.Command, oracleResult.Audience)
		}
		diffs = append(diffs, oraclediff.BlockDiff{
			Command: label,
			Oracle:  oracleBlock,
			Go:      goBlock,
			Diff:    oraclediff.UnifiedDiff("c-oracle", "go-port", oracleBlock, goBlock),
		})
	}

	fmt.Print(oraclediff.Report(oraclediff.ReportMeta{
		Scenario:   scenario.Name,
		OracleAddr: oracleAddr,
		GoAddr:     goAddr,
		Seed:       seed,
	}, diffs))
	if showOracle {
		fmt.Println("normalized C oracle blocks:")
		for _, diff := range diffs {
			fmt.Printf("--- [%s]\n%s", diff.Command, diff.Oracle)
		}
	}
	if showGoLog {
		fmt.Println("go port server log:")
		fmt.Print(goProc.log.String())
	}
	return nil
}

func prepareOracleData(source, destination string, emptyPlayers bool) error {
	if err := os.CopyFS(destination, os.DirFS(source)); err != nil {
		return fmt.Errorf("copy C oracle lib to throwaway directory: %w", err)
	}
	if !emptyPlayers {
		return nil
	}
	playersPath := filepath.Join(destination, "etc", "players")
	if err := os.WriteFile(playersPath, nil, 0o600); err != nil {
		return fmt.Errorf("empty disposable C oracle player file: %w", err)
	}
	return nil
}

func withFreshMUDEnv(env []string, enabled bool) []string {
	filtered := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, "DP_FRESH_MUD=") {
			filtered = append(filtered, entry)
		}
	}
	if enabled {
		filtered = append(filtered, "DP_FRESH_MUD=1")
	}
	return filtered
}

func probeClients(primary oraclediff.Conn, peers map[string]oraclediff.Conn, actorName string) (oraclediff.Conn, map[string]oraclediff.Conn) {
	if actorName == "" {
		return primary, peers
	}
	audience := make(map[string]oraclediff.Conn, len(peers))
	audience["primary"] = primary
	for name, conn := range peers {
		if name != actorName {
			audience[name] = conn
		}
	}
	return peers[actorName], audience
}

func drainClients(quiescence time.Duration, primary oraclediff.Conn, peers map[string]oraclediff.Conn) error {
	if _, err := primary.ReadUntilQuiescent(quiescence); err != nil {
		return fmt.Errorf("actor: %w", err)
	}
	for name, conn := range peers {
		if _, err := conn.ReadUntilQuiescent(quiescence); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func applyObjectFixtures(worldDir string, fixtures []oraclediff.ObjectFixture) error {
	for _, fixture := range fixtures {
		if err := makeScrollFixtureInert(worldDir, fixture.ObjectVNum); err != nil {
			return err
		}
	}
	return nil
}

func applyObjectSpawnFixtures(worldDir string, fixtures []oraclediff.ObjectSpawnFixture) error {
	for _, fixture := range fixtures {
		path := filepath.Join(worldDir, "zon", fmt.Sprintf("%d.zon", fixture.ZoneNumber))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read zone %d: %w", fixture.ZoneNumber, err)
		}
		marker := []byte("\nS\n$")
		index := bytes.LastIndex(data, marker)
		if index < 0 {
			return fmt.Errorf("zone %d has no terminal reset marker", fixture.ZoneNumber)
		}
		command := fmt.Sprintf("\nO 0 %d %d %d", fixture.ObjectVNum, fixture.MaxExisting, fixture.RoomVNum)
		updated := make([]byte, 0, len(data)+len(command))
		updated = append(updated, data[:index]...)
		updated = append(updated, command...)
		updated = append(updated, data[index:]...)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat zone %d: %w", fixture.ZoneNumber, err)
		}
		if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer vnum, not request-derived
			return fmt.Errorf("write zone %d: %w", fixture.ZoneNumber, err)
		}
	}
	return nil
}

func applyMobFixtures(worldDir string, fixtures []oraclediff.MobFixture) error {
	for _, fixture := range fixtures {
		path := filepath.Join(worldDir, "zon", fmt.Sprintf("%d.zon", fixture.ZoneNumber))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read zone %d: %w", fixture.ZoneNumber, err)
		}
		marker := []byte("\nS\n$")
		index := bytes.LastIndex(data, marker)
		if index < 0 {
			return fmt.Errorf("zone %d has no terminal reset marker", fixture.ZoneNumber)
		}
		command := fmt.Sprintf("\nM 0 %d %d %d", fixture.MobVNum, fixture.MaxExisting, fixture.RoomVNum)
		updated := make([]byte, 0, len(data)+len(command))
		updated = append(updated, data[:index]...)
		updated = append(updated, command...)
		updated = append(updated, data[index:]...)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat zone %d: %w", fixture.ZoneNumber, err)
		}
		if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer vnum, not request-derived
			return fmt.Errorf("write zone %d: %w", fixture.ZoneNumber, err)
		}
	}
	return nil
}

// applyMobAffFixtures patches a mob prototype's innate affected-by bitmask in
// a disposable world copy: the flag line after the vnum header's four
// ~-terminated text blocks carries act, then affected, as its first two
// fields. C's read_mobile copies those bits onto every instance, which is
// what mag_affects' mob-affection gate tests.
func applyMobAffFixtures(worldDir string, fixtures []oraclediff.MobAffFixture) error {
	for _, fixture := range fixtures {
		path := filepath.Join(worldDir, "mob", fmt.Sprintf("%d.mob", fixture.MobVNum/100))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read mob file for vnum %d: %w", fixture.MobVNum, err)
		}
		lines := strings.Split(string(data), "\n")
		header := fmt.Sprintf("#%d", fixture.MobVNum)
		inTarget := false
		blocksClosed := 0
		patched := false
		for index, line := range lines {
			if strings.HasPrefix(line, "#") {
				inTarget = line == header
				blocksClosed = 0
				continue
			}
			if !inTarget || patched {
				continue
			}
			// Four Diku text blocks (alias, short, long, detail) precede the
			// flag line; each closes on a line ENDING with '~' — the marker may
			// be inline ("trainee guard~") or standalone ("~").
			trimmed := strings.TrimSpace(line)
			if blocksClosed < 4 {
				if strings.HasSuffix(trimmed, "~") {
					blocksClosed++
				}
				continue
			}
			// Flag-line layout (db.c parse_mobile): eight flag words — act
			// words 1-4, AFFECTED words 5-8 — then alignment, then the type
			// letter. The first affected word (fields[4]) carries AFF bits
			// 0-31, which is what the mob-affection gate tests.
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return fmt.Errorf("mob %d flag line %q has no affected word", fixture.MobVNum, line)
			}
			fields[4] = fmt.Sprintf("%d", fixture.AffMask)
			lines[index] = strings.Join(fields, " ")
			patched = true
		}
		if !patched {
			return fmt.Errorf("mob %d not found or flag line not reached in %s", fixture.MobVNum, path)
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat mob file for vnum %d: %w", fixture.MobVNum, err)
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer-derived file name
			return fmt.Errorf("write mob file for vnum %d: %w", fixture.MobVNum, err)
		}
	}
	return nil
}

func applyScriptlessMobFixtures(worldDir string, mobVNums []int) error {
	for _, mobVNum := range mobVNums {
		path := filepath.Join(worldDir, "mob", fmt.Sprintf("%d.mob", mobVNum/100))
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read mob zone %d: %w", mobVNum/100, err)
		}
		lines := strings.Split(string(data), "\n")
		inTarget := false
		found := false
		for index, line := range lines {
			if strings.HasPrefix(line, "#") {
				inTarget = line == fmt.Sprintf("#%d", mobVNum)
				continue
			}
			if inTarget && strings.HasPrefix(strings.TrimSpace(line), "Script:") {
				lines[index] = "* oracle fixture: " + line
				found = true
			}
		}
		if !found {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat mob zone %d: %w", mobVNum/100, err)
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer vnum, not request-derived
			return fmt.Errorf("write mob zone %d: %w", mobVNum/100, err)
		}
	}
	return nil
}

func applyQuietZoneFixtures(worldDir string, zoneNumbers []int) error {
	for _, zoneNumber := range zoneNumbers {
		path := filepath.Join(worldDir, "zon", fmt.Sprintf("%d.zon", zoneNumber))
		if err := quietMobResets(path); err != nil {
			return fmt.Errorf("quiet zone %d: %w", zoneNumber, err)
		}
	}
	return nil
}

func applyQuietMobFixtures(worldDir string) error {
	paths, err := filepath.Glob(filepath.Join(worldDir, "zon", "*.zon"))
	if err != nil {
		return fmt.Errorf("list zone files: %w", err)
	}
	for _, path := range paths {
		if err := quietMobResets(path); err != nil {
			return err
		}
	}
	return nil
}

func quietMobResets(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "M ") || strings.HasPrefix(trimmed, "G ") || strings.HasPrefix(trimmed, "E ") {
			lines[index] = "* oracle fixture: " + line
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer vnum, not request-derived
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func makeScrollFixtureInert(worldDir string, objectVNum int) error {
	path := filepath.Join(worldDir, "obj", fmt.Sprintf("%d.obj", objectVNum/100))
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read object file for %d: %w", objectVNum, err)
	}
	startMarker := fmt.Sprintf("#%d\n", objectVNum)
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return fmt.Errorf("object %d not found in %s", objectVNum, path)
	}
	end := bytes.Index(data[start+len(startMarker):], []byte("\n#"))
	if end < 0 {
		end = len(data) - start - len(startMarker)
	}
	end += start + len(startMarker)
	record := string(data[start:end])
	lines := strings.Split(record, "\n")
	changed := false
	for i := 0; i+1 < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 9 {
			continue
		}
		validFlags := true
		for _, field := range fields[:9] {
			if _, parseErr := strconv.Atoi(field); parseErr != nil {
				validFlags = false
				break
			}
		}
		if !validFlags {
			continue
		}
		values := strings.Fields(lines[i+1])
		if len(values) != 4 {
			return fmt.Errorf("object %d has invalid scroll values %q", objectVNum, lines[i+1])
		}
		fields[0] = "2"
		lines[i] = strings.Join(fields, " ")
		lines[i+1] = values[0] + " -1 -1 -1"
		changed = true
		break
	}
	if !changed {
		return fmt.Errorf("object %d is not a scroll fixture", objectVNum)
	}
	updated := append([]byte{}, data[:start]...)
	updated = append(updated, strings.Join(lines, "\n")...)
	updated = append(updated, data[end:]...)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat object file for %d: %w", objectVNum, err)
	}
	if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer vnum, not request-derived
		return fmt.Errorf("write object file for %d: %w", objectVNum, err)
	}
	return nil
}

// applyForceLoadFixtures rewrites one object prototype's load percent to a
// always-true 500% in each server's disposable world copy. Circle gates every
// zone-reset object load on percent_load (GET_OBJ_LOAD > uniform()*100), so a
// low-percent prototype would load — or not — on a seeded RNG draw and make
// spawn-obj fixtures probabilistic. The source world trees are never modified.
func applyForceLoadFixtures(worldDir string, objectVNums []int) error {
	for _, objectVNum := range objectVNums {
		if err := forceObjectLoad(worldDir, objectVNum); err != nil {
			return err
		}
	}
	return nil
}

func forceObjectLoad(worldDir string, objectVNum int) error {
	path := filepath.Join(worldDir, "obj", fmt.Sprintf("%d.obj", objectVNum/100))
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read object file for %d: %w", objectVNum, err)
	}
	startMarker := fmt.Sprintf("#%d\n", objectVNum)
	start := bytes.Index(data, []byte(startMarker))
	if start < 0 {
		return fmt.Errorf("object %d not found in %s", objectVNum, path)
	}
	end := bytes.Index(data[start+len(startMarker):], []byte("\n#"))
	if end < 0 {
		end = len(data) - start - len(startMarker)
	}
	end += start + len(startMarker)
	record := string(data[start:end])
	lines := strings.Split(record, "\n")
	changed := false
	for i := 0; i+2 < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 9 {
			continue
		}
		validFlags := true
		for _, field := range fields[:9] {
			if _, parseErr := strconv.Atoi(field); parseErr != nil {
				validFlags = false
				break
			}
		}
		if !validFlags {
			continue
		}
		wcl := strings.Fields(lines[i+2])
		if len(wcl) < 3 {
			return fmt.Errorf("object %d has invalid weight/cost/load line %q", objectVNum, lines[i+2])
		}
		if _, parseErr := strconv.ParseFloat(wcl[2], 64); parseErr != nil {
			return fmt.Errorf("object %d has invalid load percent %q", objectVNum, wcl[2])
		}
		wcl[2] = "500.00"
		lines[i+2] = strings.Join(wcl, " ")
		changed = true
		break
	}
	if !changed {
		return fmt.Errorf("object %d is not a force-load fixture candidate", objectVNum)
	}
	updated := append([]byte{}, data[:start]...)
	updated = append(updated, strings.Join(lines, "\n")...)
	updated = append(updated, data[end:]...)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat object file for %d: %w", objectVNum, err)
	}
	if err := os.WriteFile(path, updated, info.Mode().Perm()); err != nil { // #nosec G703 -- dev oracle-diff harness; path is a filepath.Join of a trusted world dir and an integer vnum, not request-derived
		return fmt.Errorf("write object file for %d: %w", objectVNum, err)
	}
	return nil
}

func startProcess(ctx context.Context, name, dir string, env []string, command string, args ...string) (*process, error) {
	log := &safeBuffer{}
	cmd := exec.CommandContext(ctx, command, args...) // #nosec G702 -- dev oracle-diff harness; command/args are hardcoded engine binaries, not request-derived
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	p := &process{name: name, cmd: cmd, done: make(chan struct{}), log: log}
	go func() {
		p.waitErr = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

func (p *process) stop() {
	if p == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Signal(os.Interrupt)
	select {
	case <-p.done:
	case <-time.After(3 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}

func dialWhenReady(p *process, addr string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-p.done:
			return nil, fmt.Errorf("%s exited before readiness: %w\nserver log:\n%s", p.name, p.waitErr, p.log.String())
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("%s did not accept connections on %s within %s: %w\nserver log:\n%s",
		p.name, addr, timeout, lastErr, p.log.String())
}

func waitForLog(p *process, marker string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(p.log.String(), marker) {
			return nil
		}
		select {
		case <-p.done:
			return fmt.Errorf("%s exited before logging %q: %w\nserver log:\n%s", p.name, marker, p.waitErr, p.log.String())
		default:
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("%s did not log %q within %s\nserver log:\n%s", p.name, marker, timeout, p.log.String())
}

// allocatePorts accounts for CircleMUD's undocumented second listener: WHOD
// always binds oraclePort+1. Holding all four sockets at once avoids handing a
// duplicate/free-looking port to another server before process startup.
func allocatePorts() (oracle, goTelnet, goHTTP int, err error) {
	for attempt := 0; attempt < 20; attempt++ {
		oracleListener, listenErr := net.Listen("tcp", "127.0.0.1:0")
		if listenErr != nil {
			return 0, 0, 0, fmt.Errorf("reserve C oracle port: %w", listenErr)
		}
		oracle = oracleListener.Addr().(*net.TCPAddr).Port
		whodListener, whodErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", oracle+1))
		if whodErr != nil {
			_ = oracleListener.Close()
			continue
		}
		goTelnetListener, telnetErr := net.Listen("tcp", "127.0.0.1:0")
		if telnetErr != nil {
			_ = whodListener.Close()
			_ = oracleListener.Close()
			return 0, 0, 0, fmt.Errorf("reserve Go telnet port: %w", telnetErr)
		}
		goHTTPListener, httpErr := net.Listen("tcp", "127.0.0.1:0")
		if httpErr != nil {
			_ = goTelnetListener.Close()
			_ = whodListener.Close()
			_ = oracleListener.Close()
			return 0, 0, 0, fmt.Errorf("reserve Go HTTP port: %w", httpErr)
		}
		goTelnet = goTelnetListener.Addr().(*net.TCPAddr).Port
		goHTTP = goHTTPListener.Addr().(*net.TCPAddr).Port
		_ = goHTTPListener.Close()
		_ = goTelnetListener.Close()
		_ = whodListener.Close()
		_ = oracleListener.Close()
		return oracle, goTelnet, goHTTP, nil
	}
	return 0, 0, 0, errors.New("could not reserve adjacent C oracle and WHOD ports")
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate repository root (go.mod)")
		}
		dir = parent
	}
}

func findOracleRoot(bin string) (string, error) {
	absBin, err := filepath.Abs(bin)
	if err != nil {
		return "", fmt.Errorf("resolve DP_ORACLE_BIN: %w", err)
	}
	info, err := os.Stat(absBin) // #nosec G703 -- dev oracle-diff harness; absBin derives from the developer-supplied DP_ORACLE_BIN, not request input
	if err != nil {
		return "", fmt.Errorf("DP_ORACLE_BIN %q: %w", absBin, err)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("DP_ORACLE_BIN %q is not an executable file", absBin)
	}
	for dir := filepath.Dir(absBin); ; dir = filepath.Dir(dir) {
		world := filepath.Join(dir, "lib", "world")
		if stat, statErr := os.Stat(world); statErr == nil && stat.IsDir() { // #nosec G703 -- dev harness walking up from the trusted oracle binary
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("could not find C oracle lib/world above DP_ORACLE_BIN %q", absBin)
}
