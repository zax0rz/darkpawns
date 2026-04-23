package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"


)

// Profiler manages performance profiling
type Profiler struct {
	mu          sync.Mutex
	cpuProfile  *os.File
	memProfile  *os.File
	blockProfile *os.File
	mutexProfile *os.File
	profileDir  string
	enabled     bool
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
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	
	// Start CPU profiling
	cpuFile := fmt.Sprintf("%s/cpu-%d.prof", p.profileDir, time.Now().Unix())
	f, err := os.Create(cpuFile)
	if err != nil {
		return fmt.Errorf("create cpu profile: %w", err)
	}
	
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("start cpu profile: %w", err)
	}
	
	p.cpuProfile = f
	p.enabled = true
	
	log.Printf("CPU profiling started: %s", cpuFile)
	return nil
}

// StopCPUProfile stops CPU profiling
func (p *Profiler) StopCPUProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.enabled || p.cpuProfile == nil {
		return fmt.Errorf("cpu profiling not running")
	}
	
	pprof.StopCPUProfile()
	p.cpuProfile.Close()
	p.cpuProfile = nil
	
	log.Println("CPU profiling stopped")
	return nil
}

// WriteHeapProfile writes heap profile
func (p *Profiler) WriteHeapProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create profile directory
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	
	// Write heap profile
	heapFile := fmt.Sprintf("%s/heap-%d.prof", p.profileDir, time.Now().Unix())
	f, err := os.Create(heapFile)
	if err != nil {
		return fmt.Errorf("create heap profile: %w", err)
	}
	defer f.Close()
	
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("write heap profile: %w", err)
	}
	
	p.memProfile = f
	log.Printf("Heap profile written: %s", heapFile)
	return nil
}

// StartBlockProfile starts block profiling
func (p *Profiler) StartBlockProfile(rate int) {
	runtime.SetBlockProfileRate(rate)
	log.Printf("Block profiling started with rate %d", rate)
}

// StopBlockProfile stops block profiling and writes profile
func (p *Profiler) StopBlockProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create profile directory
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	
	// Write block profile
	blockFile := fmt.Sprintf("%s/block-%d.prof", p.profileDir, time.Now().Unix())
	f, err := os.Create(blockFile)
	if err != nil {
		return fmt.Errorf("create block profile: %w", err)
	}
	defer f.Close()
	
	if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
		return fmt.Errorf("write block profile: %w", err)
	}
	
	p.blockProfile = f
	log.Printf("Block profile written: %s", blockFile)
	
	// Reset block profile rate
	runtime.SetBlockProfileRate(0)
	return nil
}

// StartMutexProfile starts mutex profiling
func (p *Profiler) StartMutexProfile(rate int) {
	runtime.SetMutexProfileFraction(rate)
	log.Printf("Mutex profiling started with fraction %d", rate)
}

// StopMutexProfile stops mutex profiling and writes profile
func (p *Profiler) StopMutexProfile() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create profile directory
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	
	// Write mutex profile
	mutexFile := fmt.Sprintf("%s/mutex-%d.prof", p.profileDir, time.Now().Unix())
	f, err := os.Create(mutexFile)
	if err != nil {
		return fmt.Errorf("create mutex profile: %w", err)
	}
	defer f.Close()
	
	if err := pprof.Lookup("mutex").WriteTo(f, 0); err != nil {
		return fmt.Errorf("write mutex profile: %w", err)
	}
	
	p.mutexProfile = f
	log.Printf("Mutex profile written: %s", mutexFile)
	
	// Reset mutex profile fraction
	runtime.SetMutexProfileFraction(0)
	return nil
}

// GoroutineDump dumps current goroutines
func (p *Profiler) GoroutineDump() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create profile directory
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	
	// Write goroutine dump
	goroutineFile := fmt.Sprintf("%s/goroutine-%d.txt", p.profileDir, time.Now().Unix())
	f, err := os.Create(goroutineFile)
	if err != nil {
		return fmt.Errorf("create goroutine dump: %w", err)
	}
	defer f.Close()
	
	if err := pprof.Lookup("goroutine").WriteTo(f, 2); err != nil {
		return fmt.Errorf("write goroutine dump: %w", err)
	}
	
	log.Printf("Goroutine dump written: %s", goroutineFile)
	return nil
}

// MemoryStats returns current memory statistics
func (p *Profiler) MemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return map[string]interface{}{
		"alloc":          m.Alloc,
		"total_alloc":    m.TotalAlloc,
		"sys":            m.Sys,
		"heap_alloc":     m.HeapAlloc,
		"heap_sys":       m.HeapSys,
		"heap_idle":      m.HeapIdle,
		"heap_in_use":    m.HeapInuse,
		"heap_released":  m.HeapReleased,
		"heap_objects":   m.HeapObjects,
		"num_gc":         m.NumGC,
		"next_gc":        m.NextGC,
		"last_gc":        m.LastGC,
		"gc_cpu_fraction": m.GCCPUFraction,
		"num_goroutines": runtime.NumGoroutine(),
		"num_cpus":       runtime.NumCPU(),
	}
}

// PrintMemoryStats prints memory statistics
func (p *Profiler) PrintMemoryStats() {
	stats := p.MemoryStats()
	
	fmt.Println("\n=== Memory Statistics ===")
	fmt.Printf("Allocated:          %d bytes\n", stats["alloc"])
	fmt.Printf("Total Allocated:    %d bytes\n", stats["total_alloc"])
	fmt.Printf("System:             %d bytes\n", stats["sys"])
	fmt.Printf("Heap Allocated:     %d bytes\n", stats["heap_alloc"])
	fmt.Printf("Heap System:        %d bytes\n", stats["heap_sys"])
	fmt.Printf("Heap In Use:        %d bytes\n", stats["heap_in_use"])
	fmt.Printf("Heap Objects:       %d\n", stats["heap_objects"])
	fmt.Printf("Goroutines:         %d\n", stats["num_goroutines"])
	fmt.Printf("GC Cycles:          %d\n", stats["num_gc"])
	fmt.Printf("GC CPU Fraction:    %.4f\n", stats["gc_cpu_fraction"])
}

// PerformanceMonitor continuously monitors performance
type PerformanceMonitor struct {
	mu          sync.Mutex
	profiler    *Profiler
	interval    time.Duration
	stopChan    chan struct{}
	metrics     []PerformanceMetric
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Timestamp time.Time
	Memory    map[string]interface{}
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
	log.Printf("Performance monitor started with interval %v", pm.interval)
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
		
		totalAlloc += int64(alloc)
		totalGoroutines += goroutines
		
		if int64(alloc) > maxAlloc {
			maxAlloc = int64(alloc)
		}
		if int64(alloc) < minAlloc {
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

// StartPProfServer starts the pprof HTTP server
func StartPProfServer(addr string) {
	go func() {
		log.Printf("Starting pprof server on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("pprof server error: %v", err)
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
	
	log.Printf("Profiling session started for %v", duration)
	
	// Wait for duration
	time.Sleep(duration)
	
	// Stop profiling
	profiler.StopCPUProfile()
	profiler.StopBlockProfile()
	profiler.StopMutexProfile()
	profiler.WriteHeapProfile()
	profiler.GoroutineDump()
	
	monitor.Stop()
	
	// Print analysis
	analysis := monitor.AnalyzeMetrics()
	if analysis != nil {
		fmt.Println("\n=== Performance Analysis ===")
		fmt.Printf("Duration:           %v\n", analysis["duration"])
		fmt.Printf("Samples:            %d\n", analysis["sample_count"])
		fmt.Printf("Avg Memory:         %d bytes\n", analysis["avg_memory"])
		fmt.Printf("Min Memory:         %d bytes\n", analysis["min_memory"])
		fmt.Printf("Max Memory:         %d bytes\n", analysis["max_memory"])
		fmt.Printf("Avg Goroutines:     %d\n", analysis["avg_goroutines"])
		fmt.Printf("Min Goroutines:     %d\n", analysis["min_goroutines"])
		fmt.Printf("Max Goroutines:     %d\n", analysis["max_goroutines"])
	}
	
	profiler.PrintMemoryStats()
	
	log.Printf("Profiling session completed. Profiles saved to %s", profileDir)
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
			log.Fatalf("Invalid duration: %v", err)
		}
		
		if err := RunProfilingSession(profileDir, duration); err != nil {
			log.Fatal(err)
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