//go:build linux

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// oracleRuntime is shared by every worktree. XDG_RUNTIME_DIR and TMPDIR must
// not accidentally create independent boot locks or worker budgets.
func oracleRuntime() string {
	parent := fmt.Sprintf("/run/user/%d", os.Getuid())
	if st, err := os.Stat(parent); err == nil && st.IsDir() {
		return filepath.Join(parent, "dp-census")
	}
	return fmt.Sprintf("/tmp/dp-census-%d", os.Getuid())
}

func bootLease(timeout time.Duration) (func(), error) {
	dir := oracleRuntime()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(dir, "boot.lock"))
	// Admission can queue behind 72 boots; it is separate from the unchanged
	// per-engine readiness deadline. The worker's 240s watchdog still bounds it.
	ctx, cancel := context.WithTimeout(context.Background(), max(timeout, 2*time.Minute))
	defer cancel()
	ok, err := lock.TryLockContext(ctx, 10*time.Millisecond)
	if err != nil || !ok {
		return nil, fmt.Errorf("listener ownership: boot lease: %w", errors.Join(err, ctx.Err()))
	}
	return func() { _ = lock.Unlock() }, nil
}

// Keep server numbers outside the kernel's automatic client-port range. The
// boot lease coordinates harnesses; this also avoids an unrelated outgoing
// connection claiming a just-released reservation during engine boot.
func reserveOraclePort(port int) (net.Listener, error) {
	if port == 0 {
		raw, err := os.ReadFile("/proc/sys/net/ipv4/ip_local_port_range")
		if err != nil {
			return nil, err
		}
		var low, high int
		if _, err := fmt.Sscanf(string(raw), "%d %d", &low, &high); err != nil {
			return nil, err
		}
		first, last := 10000, low-2
		if last < first {
			first = 1024
		}
		if last < first {
			first, last = high+1, 65534
		}
		if last-first < 5 {
			return nil, errors.New("listener ownership: no non-ephemeral server port range")
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(last-first+1)))
		if err != nil {
			return nil, err
		}
		port = first + int(n.Int64())
	}
	return net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
}

// ownedListener consults kernel socket inodes, never a banner or the mere
// existence of a TCP endpoint. Both IPv4 and dual-stack Go listeners count.
func ownedListener(pid, port int) (owned, foreign bool, err error) {
	inodes := map[string]bool{}
	for _, table := range []string{"tcp", "tcp6"} {
		f, openErr := os.Open(filepath.Join("/proc/net", table))
		if openErr != nil {
			return false, false, openErr
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 10 || fields[3] != "0A" {
				continue
			}
			_, rawPort, found := strings.Cut(fields[1], ":")
			n, parseErr := strconv.ParseInt(rawPort, 16, 32)
			if found && parseErr == nil && int(n) == port {
				inodes[fields[9]] = true
			}
		}
		scanErr := scanner.Err()
		_ = f.Close()
		if scanErr != nil {
			return false, false, scanErr
		}
	}
	if len(inodes) == 0 {
		return false, false, nil
	}
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return false, false, err
	}
	for _, entry := range entries {
		target, readErr := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, entry.Name()))
		if readErr != nil {
			continue
		} // An unrelated descriptor can close while scanning.
		inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
		if _, found := inodes[inode]; found {
			owned = true
			delete(inodes, inode)
		}
	}
	return owned, len(inodes) > 0, nil
}

func waitOwnedListener(p *process, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-p.done:
			return fmt.Errorf("listener ownership: %s exited before readiness: %v\nserver log:\n%s", p.name, p.waitErr, p.log.String())
		default:
		}
		own, foreign, err := ownedListener(p.cmd.Process.Pid, port)
		if err != nil {
			return fmt.Errorf("listener ownership: inspect %s port %d: %w", p.name, port, err)
		}
		if foreign {
			return fmt.Errorf("listener ownership: %s port %d belongs to a foreign process\nserver log:\n%s", p.name, port, p.log.String())
		}
		if own {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("listener ownership: %s did not own port %d within %s\nserver log:\n%s", p.name, port, timeout, p.log.String())
}

// ownedStart keeps the boot lease until every listener belongs to the newly
// spawned engine. Restarts reserve the same ports before releasing them again.
func ownedStart(timeout time.Duration, reservations []net.Listener, ports []int, start func() (*process, error)) func() (*process, error) {
	first := true
	return func() (*process, error) {
		release, err := bootLease(timeout)
		if err != nil {
			return nil, err
		}
		defer release()
		if !first {
			reservations = nil
			for _, port := range ports {
				ln, listenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
				if listenErr != nil {
					for _, held := range reservations {
						_ = held.Close()
					}
					return nil, fmt.Errorf("listener ownership: reserve restart port %d: %w", port, listenErr)
				}
				reservations = append(reservations, ln)
			}
		}
		first = false
		for _, held := range reservations {
			_ = held.Close()
		}
		p, err := start()
		if err != nil {
			return nil, err
		}
		for _, port := range ports {
			if err := waitOwnedListener(p, port, timeout); err != nil {
				p.stop()
				return nil, err
			}
		}
		return p, nil
	}
}
