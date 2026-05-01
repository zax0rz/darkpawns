package optimization

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketOptimizer provides WebSocket performance optimizations.
type WebSocketOptimizer struct {
	mu                sync.RWMutex
	compressionLevel  int
	batchWindow       time.Duration
	maxBatchSize      int
	enableCompression bool
}

// NewWebSocketOptimizer creates a new WebSocket optimizer.
func NewWebSocketOptimizer() *WebSocketOptimizer {
	return &WebSocketOptimizer{
		compressionLevel:  gzip.DefaultCompression,
		batchWindow:       50 * time.Millisecond, // 50ms batching window
		maxBatchSize:      100,                   // Max messages per batch
		enableCompression: true,
	}
}

// MessageBatch represents a batch of WebSocket messages.
type MessageBatch struct {
	Messages []json.RawMessage
	SentAt   time.Time
}

// CompressedWebSocket extends the standard WebSocket with compression.
type CompressedWebSocket struct {
	conn             *websocket.Conn
	compressionLevel int
	readBuffer       bytes.Buffer
	writeBuffer      bytes.Buffer
	gzipReader       *gzip.Reader
	gzipWriter       *gzip.Writer
	mu               sync.Mutex
}

// NewCompressedWebSocket creates a new compressed WebSocket wrapper.
func NewCompressedWebSocket(conn *websocket.Conn, compressionLevel int) *CompressedWebSocket {
	return &CompressedWebSocket{
		conn:             conn,
		compressionLevel: compressionLevel,
	}
}

// WriteMessage writes a compressed message to the WebSocket.
func (cws *CompressedWebSocket) WriteMessage(messageType int, data []byte) error {
	cws.mu.Lock()
	defer cws.mu.Unlock()

	if cws.compressionLevel == gzip.NoCompression {
		return cws.conn.WriteMessage(messageType, data)
	}

	// Compress the data
	cws.writeBuffer.Reset()
	gzipWriter, err := gzip.NewWriterLevel(&cws.writeBuffer, cws.compressionLevel)
	if err != nil {
		return err
	}

	if _, err := gzipWriter.Write(data); err != nil {
		return err
	}

	if err := gzipWriter.Close(); err != nil {
		return err
	}

	return cws.conn.WriteMessage(messageType, cws.writeBuffer.Bytes())
}

// ReadMessage reads and decompresses a message from the WebSocket.
func (cws *CompressedWebSocket) ReadMessage() (int, []byte, error) {
	messageType, data, err := cws.conn.ReadMessage()
	if err != nil {
		return messageType, nil, err
	}

	if cws.compressionLevel == gzip.NoCompression {
		return messageType, data, nil
	}

	// Decompress the data
	cws.readBuffer.Reset()
	cws.readBuffer.Write(data)

	if cws.gzipReader == nil {
		cws.gzipReader, err = gzip.NewReader(&cws.readBuffer)
		if err != nil {
			return messageType, nil, err
		}
	} else {
		if err := cws.gzipReader.Reset(&cws.readBuffer); err != nil {
			return messageType, nil, err
		}
	}

	decompressed, err := io.ReadAll(cws.gzipReader)
	if err != nil {
		return messageType, nil, err
	}

	return messageType, decompressed, nil
}

// Close closes the WebSocket and compression resources.
func (cws *CompressedWebSocket) Close() error {
	cws.mu.Lock()
	defer cws.mu.Unlock()

	if cws.gzipReader != nil {
		_ = cws.gzipReader.Close()
	}
	if cws.gzipWriter != nil {
		_ = cws.gzipWriter.Close()
	}

	return cws.conn.Close()
}

// BatchedSender manages batched WebSocket message sending.
type BatchedSender struct {
	mu           sync.Mutex
	batch        *MessageBatch
	batchWindow  time.Duration
	maxBatchSize int
	sendFunc     func([]json.RawMessage) error
	timer        *time.Timer
	flushChan    chan struct{}
	closed       bool
}

// NewBatchedSender creates a new batched sender.
func NewBatchedSender(batchWindow time.Duration, maxBatchSize int, sendFunc func([]json.RawMessage) error) *BatchedSender {
	bs := &BatchedSender{
		batchWindow:  batchWindow,
		maxBatchSize: maxBatchSize,
		sendFunc:     sendFunc,
		flushChan:    make(chan struct{}, 1),
	}

	bs.timer = time.AfterFunc(batchWindow, bs.flushTimer)
	return bs
}

// Send adds a message to the batch.
func (bs *BatchedSender) Send(message json.RawMessage) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.closed {
		return ErrPoolClosed
	}

	if bs.batch == nil {
		bs.batch = &MessageBatch{
			Messages: make([]json.RawMessage, 0, bs.maxBatchSize),
			SentAt:   time.Now(),
		}
	}

	bs.batch.Messages = append(bs.batch.Messages, message)

	// Flush if batch is full
	if len(bs.batch.Messages) >= bs.maxBatchSize {
		return bs.flushLocked()
	}

	// Reset timer
	bs.timer.Stop()
	bs.timer.Reset(bs.batchWindow)

	return nil
}

// Flush immediately sends the current batch.
func (bs *BatchedSender) Flush() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.flushLocked()
}

// flushLocked sends the current batch (caller must hold the lock).
func (bs *BatchedSender) flushLocked() error {
	if bs.batch == nil || len(bs.batch.Messages) == 0 {
		return nil
	}

	batch := bs.batch
	bs.batch = nil

	// Send batch asynchronously
	go func() {
		if err := bs.sendFunc(batch.Messages); err != nil {
			slog.Warn("batched send failed",
				"batch_size", len(batch.Messages),
				"error", err)
		}
	}()

	return nil
}

// flushTimer handles timer-based flushing.
func (bs *BatchedSender) flushTimer() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.batch != nil && len(bs.batch.Messages) > 0 {
		_ = bs.flushLocked()
	}
}

// Close gracefully shuts down the batched sender.
func (bs *BatchedSender) Close() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.closed = true
	bs.timer.Stop()

	// Flush any remaining messages
	return bs.flushLocked()
}

// BackpressureMonitor monitors WebSocket backpressure.
type BackpressureMonitor struct {
	mu             sync.RWMutex
	sendChanSizes  map[string]int
	maxBufferSize  int
	alertThreshold float64 // 0.0 to 1.0
}

// NewBackpressureMonitor creates a new backpressure monitor.
func NewBackpressureMonitor(maxBufferSize int, alertThreshold float64) *BackpressureMonitor {
	return &BackpressureMonitor{
		sendChanSizes:  make(map[string]int),
		maxBufferSize:  maxBufferSize,
		alertThreshold: alertThreshold,
	}
}

// Update updates the buffer size for a session.
func (bm *BackpressureMonitor) Update(sessionID string, bufferSize int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.sendChanSizes[sessionID] = bufferSize
}

// Remove removes a session from monitoring.
func (bm *BackpressureMonitor) Remove(sessionID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	delete(bm.sendChanSizes, sessionID)
}

// CheckBackpressure checks if any sessions are experiencing backpressure.
func (bm *BackpressureMonitor) CheckBackpressure() []string {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	var problematic []string
	threshold := int(float64(bm.maxBufferSize) * bm.alertThreshold)

	for sessionID, size := range bm.sendChanSizes {
		if size > threshold {
			problematic = append(problematic, sessionID)
		}
	}

	return problematic
}

// GetStats returns backpressure statistics.
func (bm *BackpressureMonitor) GetStats() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_sessions"] = len(bm.sendChanSizes)

	var totalBuffer int
	for _, size := range bm.sendChanSizes {
		totalBuffer += size
	}

	if len(bm.sendChanSizes) > 0 {
		stats["avg_buffer_size"] = totalBuffer / len(bm.sendChanSizes)
		stats["max_buffer_size"] = bm.maxBufferSize
	}

	return stats
}
