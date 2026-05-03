package main

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	nethttppprof "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	runtimepprof "runtime/pprof"
	"sync"
	"time"
)

// Profiler manages performance profiling
type Profiler struct {
	mu           sync.Mutex
	cpuProfile   *os.File
	memProfile   *os.File
	blockProfile *os.File
	mutexProfile *os.File
	profileDir   string
	enabled      bool
}

// NewProfiler creates a new profiler
func NewProfiler(profileDir string) *Profiler {
	return &Profiler{
		profileDir: profileDir,
		enabled:    false,
	}
}

// StartCPUProfile starts CPU profiling
func (p *Profiler) StartCPUProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.enabled {
		return fmt.Errorf("profiler already running")
	}

	// Create profile directory
// #nosec G301
// #nosec G703
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Start CPU profiling
	cpuFile := fmt.Sprintf("%s/cpu-%d.prof", p.profileDir, time.Now().Unix())
// #nosec G302
// #nosec G304
// #nosec G703
	f, err := os.OpenFile(cpuFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create cpu profile: %w", err)
	}

	if err := runtimepprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("start cpu profile: %w", err)
	}

	p.cpuProfile = f
	p.enabled = true

// #nosec G706
	slog.Info("CPU profiling started", "file", cpuFile)
	return nil
}

// StopCPUProfile stops CPU profiling
func (p *Profiler) StopCPUProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled || p.cpuProfile == nil {
		return fmt.Errorf("cpu profiling not running")
	}

	runtimepprof.StopCPUProfile()
	_ = p.cpuProfile.Close()
	p.cpuProfile = nil

	slog.Info("CPU profiling stopped")
	return nil
}

// WriteHeapProfile writes heap profile
func (p *Profiler) WriteHeapProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create profile directory
// #nosec G301
// #nosec G703
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Write heap profile
	heapFile := fmt.Sprintf("%s/heap-%d.prof", p.profileDir, time.Now().Unix())
// #nosec G302
// #nosec G304
// #nosec G703
	f, err := os.OpenFile(heapFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create heap profile: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := runtimepprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("write heap profile: %w", err)
	}

	p.memProfile = f
// #nosec G706
	slog.Info("Heap profile written", "file", heapFile)
	return nil
}

// StartBlockProfile starts block profiling
func (p *Profiler) StartBlockProfile(rate int) {
	runtime.SetBlockProfileRate(rate)
	slog.Info("Block profiling started", "rate", rate)
}

// StopBlockProfile stops block profiling and writes profile
func (p *Profiler) StopBlockProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create profile directory
// #nosec G301
// #nosec G703
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Write block profile
	blockFile := fmt.Sprintf("%s/block-%d.prof", p.profileDir, time.Now().Unix())
// #nosec G302
// #nosec G304
// #nosec G703
	f, err := os.OpenFile(blockFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create block profile: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := runtimepprof.Lookup("block").WriteTo(f, 0); err != nil {
		return fmt.Errorf("write block profile: %w", err)
	}

	p.blockProfile = f
// #nosec G706
	slog.Info("Block profile written", "file", blockFile)

	// Reset block profile rate
	runtime.SetBlockProfileRate(0)
	return nil
}

// StartMutexProfile starts mutex profiling
func (p *Profiler) StartMutexProfile(rate int) {
	runtime.SetMutexProfileFraction(rate)
	slog.Info("Mutex profiling started", "fraction", rate)
}

// StopMutexProfile stops mutex profiling and writes profile
func (p *Profiler) StopMutexProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create profile directory
// #nosec G301
// #nosec G703
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Write mutex profile
	mutexFile := fmt.Sprintf("%s/mutex-%d.prof", p.profileDir, time.Now().Unix())
// #nosec G302
// #nosec G304
// #nosec G703
	f, err := os.OpenFile(mutexFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create mutex profile: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := runtimepprof.Lookup("mutex").WriteTo(f, 0); err != nil {
		return fmt.Errorf("write mutex profile: %w", err)
	}

	p.mutexProfile = f
// #nosec G706
	slog.Info("Mutex profile written", "file", mutexFile)

	// Reset mutex profile fraction
	runtime.SetMutexProfileFraction(0)
	return nil
}

// GoroutineDump dumps current goroutines
func (p *Profiler) GoroutineDump() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create profile directory
// #nosec G301
// #nosec G703
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}

	// Write goroutine dump
	goroutineFile := fmt.Sprintf("%s/goroutine-%d.txt", p.profileDir, time.Now().Unix())
// #nosec G302
// #nosec G304
// #nosec G703
	f, err := os.OpenFile(goroutineFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create goroutine dump: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := runtimepprof.Lookup("goroutine").WriteTo(f, 2); err != nil {
		return fmt.Errorf("write goroutine dump: %w", err)
	}

// #nosec G706
	slog.Info("Goroutine dump written", "file", goroutineFile)
	return nil
}

// MemoryStats returns current memory statistics
func (p *Profiler) MemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":           m.Alloc,
		"total_alloc":     m.TotalAlloc,
		"sys":             m.Sys,
		"heap_alloc":      m.HeapAlloc,
		"heap_sys":        m.HeapSys,
		"heap_idle":       m.HeapIdle,
		"heap_in_use":     m.HeapInuse,
		"heap_released":   m.HeapReleased,
		"heap_objects":    m.HeapObjects,
		"num_gc":          m.NumGC,
		"next_gc":         m.NextGC,
		"last_gc":         m.LastGC,
		"gc_cpu_fraction": m.GCCPUFraction,
		"num_goroutines":  runtime.NumGoroutine(),
		"num_cpus":        runtime.NumCPU(),
	}
}

// PrintMemoryStats prints memory statistics
func (p *Profiler) PrintMemoryStats() {
	stats := p.MemoryStats()

	slog.Info("=== Memory Statistics ===")
// #nosec G706
	slog.Info("Allocated", "bytes", stats["alloc"])
// #nosec G706
	slog.Info("Total Allocated", "bytes", stats["total_alloc"])
// #nosec G706
	slog.Info("System", "bytes", stats["sys"])
// #nosec G706
	slog.Info("Heap Allocated", "bytes", stats["heap_alloc"])
// #nosec G706
	slog.Info("Heap System", "bytes", stats["heap_sys"])
// #nosec G706
	slog.Info("Heap In Use", "bytes", stats["heap_in_use"])
// #nosec G706
	slog.Info("Heap Objects", "count", stats["heap_objects"])
// #nosec G706
	slog.Info("Goroutines", "count", stats["num_goroutines"])
// #nosec G706
	slog.Info("GC Cycles", "count", stats["num_gc"])
// #nosec G706
	slog.Info("GC CPU Fraction", "fraction", stats["gc_cpu_fraction"])
}

// PerformanceMonitor continuously monitors performance
type PerformanceMonitor struct {
	mu       sync.Mutex
	profiler *Profiler
	interval time.Duration
	stopChan chan struct{}
	metrics  []PerformanceMetric
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Timestamp  time.Time
	Memory     map[string]interface{}
	Goroutines int
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(profiler *Profiler, interval time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		profiler: profiler,
		interval: interval,
		stopChan: make(chan struct{}),
		metrics:  make([]PerformanceMetric, 0, 1000),
	}
}

// Start starts the performance monitor
func (pm *PerformanceMonitor) Start() {
	go pm.monitor()
// #nosec G706
	slog.Info("Performance monitor started", "interval", pm.interval)
}

// monitor periodically collects performance metrics
func (pm *PerformanceMonitor) monitor() {
	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.collectMetrics()
		case <-pm.stopChan:
			return
		}
	}
}

// collectMetrics collects current performance metrics
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metric := PerformanceMetric{
		Timestamp:  time.Now(),
		Memory:     pm.profiler.MemoryStats(),
		Goroutines: runtime.NumGoroutine(),
	}

	pm.metrics = append(pm.metrics, metric)

	// Keep only last 1000 metrics
	if len(pm.metrics) > 1000 {
		pm.metrics = pm.metrics[1:]
	}
}

// Stop stops the performance monitor
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopChan)
}

// GetMetrics returns collected metrics
func (pm *PerformanceMonitor) GetMetrics() []PerformanceMetric {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metrics := make([]PerformanceMetric, len(pm.metrics))
	copy(metrics, pm.metrics)

	return metrics
}

// AnalyzeMetrics analyzes collected metrics
func (pm *PerformanceMonitor) AnalyzeMetrics() map[string]interface{} {
	metrics := pm.GetMetrics()
	if len(metrics) == 0 {
		return nil
	}

	analysis := make(map[string]interface{})

	// Calculate averages
	var totalAlloc, totalGoroutines int64
	var maxAlloc, maxGoroutines int64
	minAlloc := int64(^uint64(0) >> 1)
	minGoroutines := int64(^uint64(0) >> 1)

	for _, metric := range metrics {
		alloc := metric.Memory["alloc"].(uint64)
		goroutines := int64(metric.Goroutines)

// #nosec G115
		totalAlloc += int64(alloc)
		totalGoroutines += goroutines

// #nosec G115
		if int64(alloc) > maxAlloc {
// #nosec G115
			maxAlloc = int64(alloc)
		}
// #nosec G115
		if int64(alloc) < minAlloc {
// #nosec G115
			minAlloc = int64(alloc)
		}
		if goroutines > maxGoroutines {
			maxGoroutines = goroutines
		}
		if goroutines < minGoroutines {
			minGoroutines = goroutines
		}
	}

	analysis["avg_memory"] = totalAlloc / int64(len(metrics))
	analysis["avg_goroutines"] = totalGoroutines / int64(len(metrics))
	analysis["max_memory"] = maxAlloc
	analysis["min_memory"] = minAlloc
	analysis["max_goroutines"] = maxGoroutines
	analysis["min_goroutines"] = minGoroutines
	analysis["sample_count"] = len(metrics)
	analysis["duration"] = metrics[len(metrics)-1].Timestamp.Sub(metrics[0].Timestamp)

	return analysis
}

// StartPProfServer starts the pprof HTTP server with basic auth
func StartPProfServer(addr string) {
	user := os.Getenv("PPROF_USER")
	pass := os.Getenv("PPROF_PASS")
	if user == "" || pass == "" {
		slog.Info("PPROF_USER/PPROF_PASS not set, pprof server not started")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
		if username, password, ok := r.BasicAuth(); !ok ||
			subtle.ConstantTimeCompare([]byte(username), []byte(user)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="pprof"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		nethttppprof.Index(w, r)
	})
	mux.HandleFunc("/debug/pprof/cmdline", nethttppprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", nethttppprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", nethttppprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", nethttppprof.Trace)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
// #nosec G706
		slog.Info("Starting pprof server", "address", addr)
		if err := server.ListenAndServe(); err != nil {
			slog.Error("pprof server error", "error", err)
		}
	}()
}

// RunProfilingSession runs a complete profiling session
func RunProfilingSession(profileDir string, duration time.Duration) error {
	profiler := NewProfiler(profileDir)

	// Start CPU profiling
	if err := profiler.StartCPUProfile(); err != nil {
		return fmt.Errorf("start cpu profile: %w", err)
	}

	// Start block profiling
	profiler.StartBlockProfile(1)

	// Start mutex profiling
	profiler.StartMutexProfile(1)

	// Start performance monitor
	monitor := NewPerformanceMonitor(profiler, 1*time.Second)
	monitor.Start()

// #nosec G706
	slog.Info("Profiling session started", "duration", duration)

	// Wait for duration
	time.Sleep(duration)

	// Stop profiling
	_ = profiler.StopCPUProfile()
	_ = profiler.StopBlockProfile()
	_ = profiler.StopMutexProfile()
	_ = profiler.WriteHeapProfile()
	_ = profiler.GoroutineDump()

	monitor.Stop()

	// Print analysis
	analysis := monitor.AnalyzeMetrics()
	if analysis != nil {
		slog.Info("=== Performance Analysis ===")
// #nosec G706
		slog.Info("Duration", "value", analysis["duration"])
// #nosec G706
		slog.Info("Samples", "count", analysis["sample_count"])
// #nosec G706
		slog.Info("Avg Memory", "bytes", analysis["avg_memory"])
// #nosec G706
		slog.Info("Min Memory", "bytes", analysis["min_memory"])
// #nosec G706
		slog.Info("Max Memory", "bytes", analysis["max_memory"])
// #nosec G706
		slog.Info("Avg Goroutines", "count", analysis["avg_goroutines"])
// #nosec G706
		slog.Info("Min Goroutines", "count", analysis["min_goroutines"])
// #nosec G706
		slog.Info("Max Goroutines", "count", analysis["max_goroutines"])
	}

	profiler.PrintMemoryStats()

// #nosec G706
	slog.Info("Profiling session completed", "dir", profileDir)
	return nil
}

func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: profiler <command>")
		fmt.Println("Commands:")
		fmt.Println("  monitor <dir> <duration> - Run profiling session")
		fmt.Println("  pprof <addr> - Start pprof server")
		fmt.Println("  stats - Print current memory stats")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "monitor":
		if len(os.Args) < 4 {
			fmt.Println("Usage: profiler monitor <dir> <duration>")
			os.Exit(1)
		}

		profileDir := os.Args[2]
		duration, err := time.ParseDuration(os.Args[3])
		if err != nil {
			slog.Error("Invalid duration", "error", err)
			os.Exit(1)
		}

		if err := RunProfilingSession(profileDir, duration); err != nil {
			slog.Error("profiling failed", "error", err)
			os.Exit(1)
		}

	case "pprof":
		addr := ":6060"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}

		StartPProfServer(addr)

		// Wait for interrupt
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan

	case "stats":
		profiler := NewProfiler("/tmp")
		profiler.PrintMemoryStats()

	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}
