// Command dp-draw-tracer wraps the unchanged C reference oracle binary under
// Linux ptrace, setting software breakpoints on number(), dice(), and uniform()
// to record the ELF virtual return address of every top-level logged draw.
//
// To preserve direct PID ownership for dp-oracle-diff's listener-ownership
// check and signal/crash handling, the initial process spawns a detached
// "--attach <pid>" helper, grants it PR_SET_PTRACER, waits for PTRACE_ATTACH +
// PTRACE_O_TRACEEXEC, and then execs the unchanged reference binary in place.
// The reference binary therefore runs as the exact PID dp-oracle-diff spawned.
//
// When DP_C_DRAW_LOG_FILE is set and an evidence directory is configured
// (DP_DRAW_SITE_LOG_FILE, or DP_ORACLE_EVIDENCE_ROOT + DP_ORACLE_EVIDENCE),
// each top-level draw appends one line "<hex-ret-vaddr>\t<primitive>\n" to
// c-draw-sites.log in 1:1 order with c-draws.log.
package main

import (
	"bufio"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultOracleBin = "/home/zach/darkpawns-c-oracle/bin/circle"
	prSetPtracer     = 0x59616d61
)

type symRange struct {
	start uint64
	end   uint64
}

type breakpoint struct {
	name     string
	addr     uintptr
	orig     [1]byte
	retVaddr uint64 // non-zero for call-site breakpoints (e.g. external call uniform)
}

func main() {
	runtime.LockOSThread()
	if len(os.Args) == 3 && os.Args[1] == "--attach" {
		pid, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "dp-draw-tracer: invalid pid %q: %v\n", os.Args[2], err)
			os.Exit(1)
		}
		if err := runTracer(pid); err != nil {
			fmt.Fprintf(os.Stderr, "dp-draw-tracer: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if err := runParent(); err != nil {
		fmt.Fprintf(os.Stderr, "dp-draw-tracer: %v\n", err)
		os.Exit(1)
	}
}

func targetBinary() (string, error) {
	target := os.Getenv("DP_DRAW_TRACER_TARGET")
	if target == "" {
		target = defaultOracleBin
	}
	return filepath.Abs(target)
}

func runParent() error {
	target, err := targetBinary()
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	argv := append([]string{target}, os.Args[1:]...)

	if resolveLogPath() == "" {
		// #nosec G204 -- trusted local oracle tracer forwarding to reference binary
		return syscall.Exec(target, argv, os.Environ())
	}

	readyR, readyW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create ready pipe: %w", err)
	}
	defer func() { _ = readyR.Close() }()

	self, err := os.Executable()
	if err != nil {
		_ = readyW.Close()
		return fmt.Errorf("resolve self: %w", err)
	}

	// #nosec G204 -- re-execing self as ptrace helper
	helper := exec.Command(self, "--attach", strconv.Itoa(os.Getpid()))
	helper.Stderr = os.Stderr
	helper.Env = os.Environ()
	helper.ExtraFiles = []*os.File{readyW}
	if err := helper.Start(); err != nil {
		_ = readyW.Close()
		return fmt.Errorf("start tracer helper: %w", err)
	}
	_ = readyW.Close()

	// Allow the helper to PTRACE_ATTACH under Yama ptrace_scope=1.
	_, _, _ = syscall.Syscall6(syscall.SYS_PRCTL, prSetPtracer, uintptr(helper.Process.Pid), 0, 0, 0, 0)

	var ack [1]byte
	if _, err := readyR.Read(ack[:]); err != nil {
		_ = helper.Process.Kill()
		return fmt.Errorf("wait for tracer attach: %w", err)
	}
	_ = readyR.Close()

	// Exec the unchanged reference binary in place; helper catches PTRACE_EVENT_EXEC.
	// #nosec G204 -- trusted local oracle reference binary
	return syscall.Exec(target, argv, os.Environ())
}

func runTracer(pid int) error {
	readyW := os.NewFile(3, "readyW")
	defer func() { _ = readyW.Close() }()

	target, err := targetBinary()
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	logPath := resolveLogPath()
	if logPath == "" {
		return errors.New("no draw site log path configured")
	}

	syms, ptLoadVaddr, err := readDrawSymbols(target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o750); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	// #nosec G304 G703 -- logPath comes from trusted harness evidence env vars
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open draw site log: %w", err)
	}
	logWriter := bufio.NewWriterSize(logFile, 64*1024)
	defer func() {
		_ = logWriter.Flush()
		_ = logFile.Close()
	}()

	// Retry briefly in case we run before the parent calls prctl(PR_SET_PTRACER).
	var attachErr error
	for range 100 {
		attachErr = syscall.PtraceAttach(pid)
		if attachErr == nil {
			break
		}
		if !errors.Is(attachErr, syscall.EPERM) {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if attachErr != nil {
		return fmt.Errorf("ptrace attach %d: %w", pid, attachErr)
	}

	var ws syscall.WaitStatus
	if _, err := waitPID(pid, &ws); err != nil {
		return fmt.Errorf("wait after attach: %w", err)
	}
	if !ws.Stopped() {
		return nil
	}
	if err := syscall.PtraceSetOptions(pid, syscall.PTRACE_O_TRACEEXEC); err != nil {
		return fmt.Errorf("set PTRACE_O_TRACEEXEC: %w", err)
	}
	if err := syscall.PtraceCont(pid, 0); err != nil {
		return fmt.Errorf("cont before exec: %w", err)
	}

	// Signal parent to proceed to execve(circle).
	if _, err := readyW.Write([]byte{1}); err != nil {
		return fmt.Errorf("signal parent: %w", err)
	}
	_ = readyW.Close()

	// Wait until PTRACE_EVENT_EXEC fires on the new circle image.
	for {
		if _, err := waitPID(pid, &ws); err != nil {
			if errors.Is(err, syscall.ECHILD) {
				return nil
			}
			return fmt.Errorf("wait for exec: %w", err)
		}
		if ws.Exited() || ws.Signaled() {
			return nil
		}
		if ws.Stopped() {
			if ws.StopSignal() == syscall.SIGTRAP && ws.TrapCause() == syscall.PTRACE_EVENT_EXEC {
				break
			}
			sig := int(ws.StopSignal())
			if sig == int(syscall.SIGTRAP) {
				sig = 0
			}
			if err := syscall.PtraceCont(pid, sig); err != nil {
				return nil
			}
		}
	}

	pieBase, err := findPIEBase(pid, target, ptLoadVaddr)
	if err != nil {
		return err
	}

	uniformCallSites, err := findExternalCalls(target, syms["uniform"].start, syms["number"])
	if err != nil {
		return err
	}

	bps := []*breakpoint{
		{name: "number", addr: uintptr(pieBase + syms["number"].start)},
		{name: "dice", addr: uintptr(pieBase + syms["dice"].start)},
	}
	for _, callVaddr := range uniformCallSites {
		bps = append(bps, &breakpoint{
			name:     "uniform_call",
			addr:     uintptr(pieBase + callVaddr),
			retVaddr: callVaddr + 5,
		})
	}
	bpByAddr := make(map[uintptr]*breakpoint, len(bps))
	for _, bp := range bps {
		if err := installBreakpoint(pid, bp); err != nil {
			return fmt.Errorf("install breakpoint %s: %w", bp.name, err)
		}
		bpByAddr[bp.addr] = bp
	}

	diceRange := syms["dice"]

	if err := syscall.PtraceCont(pid, 0); err != nil {
		return fmt.Errorf("cont after exec: %w", err)
	}

	var retBuf [8]byte
	var regs syscall.PtraceRegs
	lineBuf := make([]byte, 0, 64)

	for {
		if _, err := waitPID(pid, &ws); err != nil {
			if errors.Is(err, syscall.ECHILD) {
				return nil
			}
			return fmt.Errorf("wait4: %w", err)
		}
		if ws.Exited() || ws.Signaled() {
			return nil
		}
		if !ws.Stopped() {
			continue
		}
		sig := ws.StopSignal()
		if sig != syscall.SIGTRAP {
			if err := syscall.PtraceCont(pid, int(sig)); err != nil {
				return nil
			}
			continue
		}

		if err := syscall.PtraceGetRegs(pid, &regs); err != nil {
			return nil
		}
		pc := uintptr(regs.Rip - 1)
		bp, ok := bpByAddr[pc]
		if !ok {
			if err := syscall.PtraceCont(pid, 0); err != nil {
				return nil
			}
			continue
		}

		shouldLog := false
		var retVaddr uint64
		primName := bp.name
		switch bp.name {
		case "uniform_call":
			retVaddr = bp.retVaddr
			primName = "uniform"
			shouldLog = true
		case "dice":
			// utils.c:104: if (size <= 0 || num <= 0) return 0; (no draw logged)
			num := int32(regs.Rdi)  // #nosec G115 -- C int argument in x86-64 ABI rdi
			size := int32(regs.Rsi) // #nosec G115 -- C int argument in x86-64 ABI rsi
			if num > 0 && size > 0 {
				if _, err := syscall.PtracePeekData(pid, uintptr(regs.Rsp), retBuf[:]); err != nil {
					return nil
				}
				retVaddr = binary.LittleEndian.Uint64(retBuf[:]) - pieBase
				shouldLog = true
			}
		case "number":
			if _, err := syscall.PtracePeekData(pid, uintptr(regs.Rsp), retBuf[:]); err != nil {
				return nil
			}
			retVaddr = binary.LittleEndian.Uint64(retBuf[:]) - pieBase
			shouldLog = retVaddr < diceRange.start || retVaddr >= diceRange.end
		}
		if shouldLog {
			lineBuf = lineBuf[:0]
			lineBuf = append(lineBuf, '0', 'x')
			lineBuf = strconv.AppendUint(lineBuf, retVaddr, 16)
			lineBuf = append(lineBuf, '\t')
			lineBuf = append(lineBuf, primName...)
			lineBuf = append(lineBuf, '\n')
			if _, err := logWriter.Write(lineBuf); err != nil {
				return fmt.Errorf("write draw site: %w", err)
			}
		}

		if _, err := syscall.PtracePokeData(pid, bp.addr, bp.orig[:]); err != nil {
			return nil
		}
		regs.Rip = uint64(bp.addr)
		if err := syscall.PtraceSetRegs(pid, &regs); err != nil {
			return nil
		}
		if err := syscall.PtraceSingleStep(pid); err != nil {
			return nil
		}

		pendingSig := 0
		for {
			if _, err := waitPID(pid, &ws); err != nil {
				return nil
			}
			if ws.Exited() || ws.Signaled() {
				return nil
			}
			if ws.Stopped() {
				stepSig := ws.StopSignal()
				if stepSig == syscall.SIGTRAP {
					break
				}
				pendingSig = int(stepSig)
				if err := syscall.PtraceSingleStep(pid); err != nil {
					return nil
				}
			}
		}

		if _, err := syscall.PtracePokeData(pid, bp.addr, []byte{0xcc}); err != nil {
			return nil
		}
		if err := syscall.PtraceCont(pid, pendingSig); err != nil {
			return nil
		}
	}
}

func resolveLogPath() string {
	if os.Getenv("DP_C_DRAW_LOG_FILE") == "" {
		return ""
	}
	if explicit := os.Getenv("DP_DRAW_SITE_LOG_FILE"); explicit != "" {
		return explicit
	}
	root := os.Getenv("DP_ORACLE_EVIDENCE_ROOT")
	evidence := os.Getenv("DP_ORACLE_EVIDENCE")
	if root == "" || evidence == "" {
		return ""
	}
	return filepath.Join(root, evidence, "c-draw-sites.log")
}

func readDrawSymbols(path string) (map[string]symRange, uint64, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open ELF %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	syms, err := f.Symbols()
	if err != nil {
		return nil, 0, fmt.Errorf("read ELF symbols: %w", err)
	}
	want := map[string]symRange{}
	for _, s := range syms {
		switch s.Name {
		case "number", "dice", "uniform":
			want[s.Name] = symRange{start: s.Value, end: s.Value + s.Size}
		}
	}
	for _, name := range []string{"number", "dice", "uniform"} {
		if _, ok := want[name]; !ok {
			return nil, 0, fmt.Errorf("symbol %q not found in %s", name, path)
		}
	}

	minVaddr := ^uint64(0)
	for _, p := range f.Progs {
		if p.Type == elf.PT_LOAD && p.Vaddr < minVaddr {
			minVaddr = p.Vaddr
		}
	}
	if minVaddr == ^uint64(0) {
		minVaddr = 0
	}
	return want, minVaddr, nil
}

func findPIEBase(pid int, target string, ptLoadVaddr uint64) (uint64, error) {
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		resolved = target
	}
	// #nosec G304 -- reading /proc/<pid>/maps for our own ptrace tracee
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return 0, fmt.Errorf("open /proc/%d/maps: %w", pid, err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 6 {
			continue
		}
		if fields[2] != "00000000" {
			continue
		}
		if fields[5] != resolved && fields[5] != target {
			continue
		}
		dash := strings.IndexByte(fields[0], '-')
		if dash <= 0 {
			continue
		}
		start, err := strconv.ParseUint(fields[0][:dash], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("parse map start %q: %w", fields[0], err)
		}
		return start - ptLoadVaddr, nil
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("could not find base mapping for %s in /proc/%d/maps", target, pid)
}

func installBreakpoint(pid int, bp *breakpoint) error {
	if _, err := syscall.PtracePeekData(pid, bp.addr, bp.orig[:]); err != nil {
		return err
	}
	if _, err := syscall.PtracePokeData(pid, bp.addr, []byte{0xcc}); err != nil {
		return err
	}
	return nil
}

func waitPID(pid int, ws *syscall.WaitStatus) (int, error) {
	for {
		wpid, err := syscall.Wait4(pid, ws, 0, nil)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		return wpid, err
	}
}

// findExternalCalls scans the ELF .text section for x86-64 direct near calls
// (opcode 0xE8 + int32 rel32) targeting targetVaddr outside excludeRange.
func findExternalCalls(path string, targetVaddr uint64, excludeRange symRange) ([]uint64, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ELF %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	sec := f.Section(".text")
	if sec == nil {
		return nil, fmt.Errorf(".text section not found in %s", path)
	}
	data, err := sec.Data()
	if err != nil {
		return nil, fmt.Errorf("read .text: %w", err)
	}
	base := sec.Addr
	var sites []uint64
	for i := 0; i+5 <= len(data); i++ {
		if data[i] != 0xe8 {
			continue
		}
		rel := int64(int32(binary.LittleEndian.Uint32(data[i+1 : i+5])))
		callVaddr := base + uint64(i)
		dest := uint64(int64(callVaddr+5) + rel)
		if dest != targetVaddr {
			continue
		}
		if callVaddr >= excludeRange.start && callVaddr < excludeRange.end {
			continue
		}
		sites = append(sites, callVaddr)
	}
	return sites, nil
}
