package vxlan

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// BatchMetrics tracks batcher performance metrics
type BatchMetrics struct {
	TotalCommands    uint64
	FlushCount       uint64
	AvgBatchSize     float64
	AvgFlushDuration time.Duration
}

// VXLANBatcher batches VXLAN commands for efficient processing
type VXLANBatcher struct {
	batchSize      int
	flushInterval  time.Duration
	commands       []VXLANCommand
	mu             sync.Mutex
	flushCallback  func([]VXLANCommand)
	timer          *time.Timer
	ctx            context.Context
	cancel         context.CancelFunc
	started        bool

	// Metrics
	totalCommands    uint64
	flushCount       uint64
	totalBatchSize   uint64
	totalFlushTime   int64 // nanoseconds
}

// NewVXLANBatcher creates a new command batcher
func NewVXLANBatcher(batchSize int, flushInterval time.Duration) *VXLANBatcher {
	return &VXLANBatcher{
		batchSize:     batchSize,
		flushInterval: flushInterval,
		commands:      make([]VXLANCommand, 0, batchSize),
	}
}

// Start initializes the batcher and returns a context for lifecycle management
func (b *VXLANBatcher) Start() (context.Context, context.CancelFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return b.ctx, b.cancel
	}

	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.started = true

	// Start timer-based flush goroutine
	go b.timerLoop()

	return b.ctx, b.cancel
}

// Add adds a command to the batch
func (b *VXLANBatcher) Add(cmd VXLANCommand) error {
	b.mu.Lock()

	if !b.started {
		b.mu.Unlock()
		return errors.New("batcher not started")
	}

	if b.ctx.Err() != nil {
		b.mu.Unlock()
		return errors.New("context cancelled")
	}

	// Validate command
	if cmd.Action == "" && cmd.VNI == 0 {
		b.mu.Unlock()
		return errors.New("invalid command")
	}

	b.commands = append(b.commands, cmd)
	atomic.AddUint64(&b.totalCommands, 1)

	shouldFlush := len(b.commands) >= b.batchSize
	b.mu.Unlock()

	// Flush if batch size reached
	if shouldFlush {
		b.Flush()
	}

	return nil
}

// Flush immediately flushes pending commands
func (b *VXLANBatcher) Flush() {
	b.mu.Lock()

	if len(b.commands) == 0 {
		b.mu.Unlock()
		return
	}

	// Copy commands for processing
	batch := make([]VXLANCommand, len(b.commands))
	copy(batch, b.commands)
	b.commands = b.commands[:0] // Clear slice

	// Update metrics
	atomic.AddUint64(&b.flushCount, 1)
	atomic.AddUint64(&b.totalBatchSize, uint64(len(batch)))

	callback := b.flushCallback
	b.mu.Unlock()

	// Execute callback outside lock and always measure time
	start := time.Now()
	if callback != nil {
		callback(batch)
	}
	duration := time.Since(start)

	// Update flush duration metrics (even if callback is nil, measure overhead)
	atomic.AddInt64(&b.totalFlushTime, int64(duration))
}

// OnFlush sets the callback function for batch processing
func (b *VXLANBatcher) OnFlush(callback func([]VXLANCommand)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushCallback = callback
}

// Metrics returns current performance metrics
func (b *VXLANBatcher) Metrics() *BatchMetrics {
	totalCommands := atomic.LoadUint64(&b.totalCommands)
	flushCount := atomic.LoadUint64(&b.flushCount)
	totalBatchSize := atomic.LoadUint64(&b.totalBatchSize)
	totalFlushTime := atomic.LoadInt64(&b.totalFlushTime)

	avgBatchSize := float64(0)
	if flushCount > 0 {
		avgBatchSize = float64(totalBatchSize) / float64(flushCount)
	}

	avgFlushDuration := time.Duration(0)
	if flushCount > 0 {
		avgFlushDuration = time.Duration(totalFlushTime / int64(flushCount))
	}

	return &BatchMetrics{
		TotalCommands:    totalCommands,
		FlushCount:       flushCount,
		AvgBatchSize:     avgBatchSize,
		AvgFlushDuration: avgFlushDuration,
	}
}

// timerLoop runs the timer-based flush mechanism
func (b *VXLANBatcher) timerLoop() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			// Final flush before exit
			b.Flush()
			return

		case <-ticker.C:
			b.Flush()
		}
	}
}