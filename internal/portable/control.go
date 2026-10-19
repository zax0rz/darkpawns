package portable

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type RuntimeStatus struct {
	State    string `json:"state"`
	Instance string `json:"instance"`
	Bundle   string `json:"bundle"`
	PID      int    `json:"pid"`
	URL      string `json:"url"`
}

type controlRecord struct {
	Address string `json:"address"`
	Token   string `json:"token"`
}

// The local operator endpoint is separate from the game's HTTP/admin surface.
// Its per-run capability is written only inside the private instance directory.
func serveControl(instance, bundle string, port int, stop context.CancelFunc) (markReady func(), closeControl func() error, err error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, nil, errors.Join(err, listener.Close())
	}
	record := controlRecord{Address: listener.Addr().String(), Token: hex.EncodeToString(secret)}
	state := RuntimeStatus{State: "starting", Instance: instance, Bundle: bundle, PID: os.Getpid(), URL: "http://127.0.0.1:" + strconv.Itoa(port) + "/"}
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(state); err != nil {
			return
		}
	})
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		state.State = "stopping"
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
		stop()
	})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+record.Token)) != 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 3 * time.Second, MaxHeaderBytes: 4096}
	path := filepath.Join(instance, ".control.json")
	data, err := json.Marshal(record)
	if err == nil {
		err = writePrivate(path, data)
	}
	if err != nil {
		return nil, nil, errors.Join(err, listener.Close())
	}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			stop()
		}
	}()
	return func() { mu.Lock(); state.State = "running"; mu.Unlock() }, func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		closeErr := server.Shutdown(shutdownCtx)
		if closeErr != nil {
			closeErr = errors.Join(closeErr, server.Close())
		}
		removeErr := os.Remove(path)
		if os.IsNotExist(removeErr) {
			removeErr = nil
		}
		return errors.Join(closeErr, removeErr)
	}, nil
}

func writePrivate(path string, data []byte) (result error) {
	// A stale record is replaceable only after acquiring this instance's run lock.
	temp, err := os.CreateTemp(filepath.Dir(path), ".runtime-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer func() {
		err := os.Remove(name)
		if os.IsNotExist(err) {
			err = nil
		}
		result = errors.Join(result, err)
	}()
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	_, writeErr := temp.Write(data)
	err = errors.Join(writeErr, temp.Sync(), temp.Close())
	if err != nil {
		return err
	}
	// Windows cannot rename over an existing file. Remove only the named runtime
	// record while holding the run lock; never remove game data.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(name, path)
}

func controlRequest(ctx context.Context, instance, method, path string) (RuntimeStatus, error) {
	var status RuntimeStatus
	var record controlRecord
	if err := readJSON(filepath.Join(instance, ".control.json"), &record); err != nil {
		return status, err
	}
	host, port, err := net.SplitHostPort(record.Address)
	number, portErr := strconv.Atoi(port)
	if err != nil || host != "127.0.0.1" || portErr != nil || number < 1 || number > 65535 || len(record.Token) != 64 {
		return status, errors.New("invalid local control record")
	}
	if _, err := hex.DecodeString(record.Token); err != nil {
		return status, errors.New("invalid local control capability")
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://"+record.Address+path, nil)
	if err != nil {
		return status, err
	}
	req.Header.Set("Authorization", "Bearer "+record.Token)
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		return status, err
	}
	defer response.Body.Close()
	if method == http.MethodPost && response.StatusCode == http.StatusAccepted {
		return status, nil
	}
	if response.StatusCode != http.StatusOK {
		return status, fmt.Errorf("local control returned HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 8192)).Decode(&status); err != nil {
		return status, err
	}
	canonical, err := canonicalInstance(instance)
	if err != nil {
		return status, err
	}
	if status.Instance != canonical {
		return status, errors.New("control endpoint belongs to another instance")
	}
	return status, nil
}

func canonicalInstance(instance string) (string, error) {
	path, err := filepath.Abs(instance)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}
