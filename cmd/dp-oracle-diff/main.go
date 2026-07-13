// Command dp-oracle-diff launches the C oracle and Go port, drives both over
// telnet with one scenario, and reports their normalized transcript divergence.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

const (
	fixedSeed = "1"
	deadDBURL = "postgres://x:x@127.0.0.1:1/nope?sslmode=disable&connect_timeout=1"
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
		quiescence   = flag.Duration("quiescence", 300*time.Millisecond, "silence interval that marks the end of an output burst")
		bootTimeout  = flag.Duration("boot-timeout", 30*time.Second, "maximum wait for each telnet listener")
	)
	flag.Parse()

	oracleBin := os.Getenv("DP_ORACLE_BIN")
	if oracleBin == "" {
		fmt.Println("SKIP: DP_ORACLE_BIN is unset; C oracle differential run not available")
		return 0
	}
	if err := execute(*scenarioName, *quiescence, *bootTimeout, oracleBin); err != nil {
		fmt.Fprintln(os.Stderr, "dp-oracle-diff:", err)
		return 1
	}
	return 0
}

func execute(scenarioName string, quiescence, bootTimeout time.Duration, oracleBin string) error {
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
	if err := os.CopyFS(oracleData, os.DirFS(filepath.Join(oracleRoot, "lib"))); err != nil {
		return fmt.Errorf("copy C oracle lib to throwaway directory: %w", err)
	}
	// The copied data directory is disposable, so character creation never
	// mutates the oracle clone. Keep its baseline player file: Circle promotes
	// the first record in an empty file to Implementor, which would invalidate
	// this mortal-room scenario.

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
		append(os.Environ(), "DP_SEED="+fixedSeed), oracleBin, "-d", oracleData, fmt.Sprint(oraclePort))
	if err != nil {
		return err
	}
	defer oracleProc.stop()

	goProc, err := startProcess(
		ctx, "Go port", goWork,
		append(
			os.Environ(),
			"JWT_SECRET=oracle-diff-secret-at-least-32-characters-long",
			"ENVIRONMENT=development",
		),
		goBin,
		"-world", filepath.Join(repoRoot, "lib", "world"),
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
	goConn := oraclediff.NewTCPConn(goNetConn)
	defer func() { _ = goConn.Close() }()

	oracleRaw, err := oraclediff.RunScenario(oracleConn, scenario, quiescence)
	if err != nil {
		return fmt.Errorf("run C oracle scenario: %w\nserver log:\n%s", err, oracleProc.log.String())
	}
	goRaw, err := oraclediff.RunScenario(goConn, scenario, quiescence)
	if err != nil {
		return fmt.Errorf("run Go port scenario: %w\nserver log:\n%s", err, goProc.log.String())
	}

	fmt.Print(oraclediff.Report(oraclediff.ReportMeta{
		Scenario:   scenario.Name,
		OracleAddr: oracleAddr,
		GoAddr:     goAddr,
		Seed:       fixedSeed,
	}, oraclediff.Normalize(oracleRaw), oraclediff.Normalize(goRaw)))
	return nil
}

func startProcess(ctx context.Context, name, dir string, env []string, command string, args ...string) (*process, error) {
	log := &safeBuffer{}
	cmd := exec.CommandContext(ctx, command, args...)
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
	info, err := os.Stat(absBin)
	if err != nil {
		return "", fmt.Errorf("DP_ORACLE_BIN %q: %w", absBin, err)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("DP_ORACLE_BIN %q is not an executable file", absBin)
	}
	for dir := filepath.Dir(absBin); ; dir = filepath.Dir(dir) {
		world := filepath.Join(dir, "lib", "world")
		if stat, statErr := os.Stat(world); statErr == nil && stat.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("could not find C oracle lib/world above DP_ORACLE_BIN %q", absBin)
}
