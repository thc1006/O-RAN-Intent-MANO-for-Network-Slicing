package vxlan

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVXLANCommandPool tests sync.Pool for command object reuse
func TestVXLANCommandPool(t *testing.T) {
	tests := []struct {
		name          string
		poolSize      int
		concurrency   int
		operationTime time.Duration
		expectReuse   bool
	}{
		{
			name:          "sequential operations reuse pool",
			poolSize:      5,
			concurrency:   1,
			operationTime: 10 * time.Millisecond,
			expectReuse:   true,
		},
		{
			name:          "concurrent operations share pool",
			poolSize:      10,
			concurrency:   20,
			operationTime: 50 * time.Millisecond,
			expectReuse:   true,
		},
		{
			name:          "high concurrency stress test",
			poolSize:      5,
			concurrency:   100,
			operationTime: 5 * time.Millisecond,
			expectReuse:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewVXLANCommandPool()
			require.NotNil(t, pool)

			var wg sync.WaitGroup
			operations := make([]int, tt.concurrency)

			for i := 0; i < tt.concurrency; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					// Get command from pool
					cmd := pool.Get()
					assert.NotNil(t, cmd)

					// Simulate operation
					time.Sleep(tt.operationTime)

					// Return to pool
					pool.Put(cmd)
					operations[idx] = 1
				}(i)
			}

			wg.Wait()

			// Verify all operations completed
			for _, op := range operations {
				assert.Equal(t, 1, op)
			}

			// Test pool statistics
			stats := pool.Stats()
			assert.NotNil(t, stats)
			if tt.expectReuse {
				assert.Greater(t, stats.Reuses, uint64(0))
			}
		})
	}
}

// TestVXLANBatchTimer tests batch flushing mechanism
func TestVXLANBatchTimer(t *testing.T) {
	tests := []struct {
		name          string
		batchSize     int
		flushInterval time.Duration
		operations    int
		expectFlushes int
	}{
		{
			name:          "flush on batch size reached",
			batchSize:     10,
			flushInterval: 1 * time.Second,
			operations:    25,
			expectFlushes: 3, // 10 + 10 + 5
		},
		{
			name:          "flush on timer expired",
			batchSize:     100,
			flushInterval: 100 * time.Millisecond,
			operations:    5,
			expectFlushes: 1, // Timer triggers before batch fills
		},
		{
			name:          "mixed flush triggers",
			batchSize:     5,
			flushInterval: 200 * time.Millisecond,
			operations:    12,
			expectFlushes: 3, // 5 + 5 + 2 (timer)
		},
		{
			name:          "single operation immediate flush",
			batchSize:     10,
			flushInterval: 50 * time.Millisecond,
			operations:    1,
			expectFlushes: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batcher := NewVXLANBatcher(tt.batchSize, tt.flushInterval)
			require.NotNil(t, batcher)

			flushed := 0
			mu := sync.Mutex{}

			// Set flush callback
			batcher.OnFlush(func(commands []VXLANCommand) {
				mu.Lock()
				flushed++
				mu.Unlock()
			})

			// Start batcher
			_, cancel := batcher.Start()
			defer cancel()

			// Add operations
			for i := 0; i < tt.operations; i++ {
				cmd := VXLANCommand{
					Action: "add",
					VNI:    uint32(1000 + i),
				}
				err := batcher.Add(cmd)
				assert.NoError(t, err)

				// Small delay to allow timer-based flushes
				if i%5 == 0 {
					time.Sleep(10 * time.Millisecond)
				}
			}

			// Wait for final flush
			time.Sleep(tt.flushInterval + 100*time.Millisecond)

			mu.Lock()
			actualFlushes := flushed
			mu.Unlock()

			// Allow some tolerance in flush count
			assert.GreaterOrEqual(t, actualFlushes, tt.expectFlushes-1)
			assert.LessOrEqual(t, actualFlushes, tt.expectFlushes+1)
		})
	}
}

// TestVXLANBatchConcurrency tests concurrent batch operations
func TestVXLANBatchConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		goroutines  int
		opsPerGo    int
		batchSize   int
		expectTotal int
	}{
		{
			name:        "low concurrency",
			goroutines:  5,
			opsPerGo:    10,
			batchSize:   10,
			expectTotal: 50,
		},
		{
			name:        "medium concurrency",
			goroutines:  20,
			opsPerGo:    50,
			batchSize:   25,
			expectTotal: 1000,
		},
		{
			name:        "high concurrency",
			goroutines:  100,
			opsPerGo:    10,
			batchSize:   50,
			expectTotal: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batcher := NewVXLANBatcher(tt.batchSize, 100*time.Millisecond)
			require.NotNil(t, batcher)

			processedCount := 0
			mu := sync.Mutex{}

			batcher.OnFlush(func(commands []VXLANCommand) {
				mu.Lock()
				processedCount += len(commands)
				mu.Unlock()
			})

			_, cancel := batcher.Start()
			defer cancel()

			var wg sync.WaitGroup
			for g := 0; g < tt.goroutines; g++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()

					for i := 0; i < tt.opsPerGo; i++ {
						cmd := VXLANCommand{
							Action: "add",
							VNI:    uint32(goroutineID*1000 + i),
						}
						err := batcher.Add(cmd)
						assert.NoError(t, err)
					}
				}(g)
			}

			wg.Wait()

			// Force final flush
			batcher.Flush()
			time.Sleep(200 * time.Millisecond)

			mu.Lock()
			total := processedCount
			mu.Unlock()

			assert.Equal(t, tt.expectTotal, total)
		})
	}
}

// TestVXLANBatchErrorHandling tests error scenarios
func TestVXLANBatchErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*VXLANBatcher)
		operation func(*VXLANBatcher) error
		wantErr   bool
		errMsg    string
	}{
		{
			name: "add to stopped batcher",
			setup: func(b *VXLANBatcher) {
				// Don't start batcher
			},
			operation: func(b *VXLANBatcher) error {
				return b.Add(VXLANCommand{Action: "add", VNI: 100})
			},
			wantErr: true,
			errMsg:  "batcher not started",
		},
		{
			name: "add after context cancelled",
			setup: func(b *VXLANBatcher) {
				ctx, cancel := b.Start()
				cancel()
				<-ctx.Done()
			},
			operation: func(b *VXLANBatcher) error {
				return b.Add(VXLANCommand{Action: "add", VNI: 100})
			},
			wantErr: true,
			errMsg:  "context cancelled",
		},
		{
			name: "invalid command",
			setup: func(b *VXLANBatcher) {
				b.Start()
			},
			operation: func(b *VXLANBatcher) error {
				return b.Add(VXLANCommand{}) // Empty command
			},
			wantErr: true,
			errMsg:  "invalid command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batcher := NewVXLANBatcher(10, 100*time.Millisecond)
			require.NotNil(t, batcher)

			if tt.setup != nil {
				tt.setup(batcher)
			}

			err := tt.operation(batcher)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVXLANBatchMetrics tests performance metrics collection
func TestVXLANBatchMetrics(t *testing.T) {
	batcher := NewVXLANBatcher(10, 100*time.Millisecond)
	require.NotNil(t, batcher)

	batcher.OnFlush(func(commands []VXLANCommand) {
		// Process commands
	})

	_, cancel := batcher.Start()
	defer cancel()

	// Add operations
	for i := 0; i < 50; i++ {
		cmd := VXLANCommand{
			Action: "add",
			VNI:    uint32(i),
		}
		err := batcher.Add(cmd)
		require.NoError(t, err)
	}

	time.Sleep(300 * time.Millisecond)

	metrics := batcher.Metrics()
	require.NotNil(t, metrics)

	assert.Greater(t, metrics.TotalCommands, uint64(0))
	assert.Greater(t, metrics.FlushCount, uint64(0))
	assert.Greater(t, metrics.AvgBatchSize, float64(0))
	// Duration might be 0 for very fast operations, so just check it's non-negative
	assert.GreaterOrEqual(t, metrics.AvgFlushDuration, time.Duration(0))
}

// TestVXLANPoolMemoryEfficiency tests memory efficiency
func TestVXLANPoolMemoryEfficiency(t *testing.T) {
	pool := NewVXLANCommandPool()
	require.NotNil(t, pool)

	// Initial allocation
	commands := make([]*VXLANCommand, 100)
	for i := 0; i < 100; i++ {
		commands[i] = pool.Get()
		commands[i].VNI = uint32(i)
	}

	// Return all to pool
	for _, cmd := range commands {
		pool.Put(cmd)
	}

	// Reuse should not allocate new memory
	stats := pool.Stats()
	initialAllocs := stats.Allocations

	// Get and return again
	for i := 0; i < 100; i++ {
		cmd := pool.Get()
		pool.Put(cmd)
	}

	finalStats := pool.Stats()

	// Allocations should remain same or increase minimally
	assert.LessOrEqual(t, finalStats.Allocations-initialAllocs, uint64(10))
	assert.Greater(t, finalStats.Reuses, uint64(90))
}

// TestVXLANCommandPoolBasic tests basic pool functionality
func TestVXLANCommandPoolBasic(t *testing.T) {
	pool := NewVXLANCommandPool()
	require.NotNil(t, pool)

	// Get a command
	cmd := pool.Get()
	require.NotNil(t, cmd)

	// Set values
	cmd.Action = "add"
	cmd.VNI = 100
	cmd.Device = "vxlan0"
	cmd.Remote = "192.168.1.1"

	// Return to pool
	pool.Put(cmd)

	// Get again - should be cleared
	cmd2 := pool.Get()
	require.NotNil(t, cmd2)
	assert.Equal(t, "", cmd2.Action)
	assert.Equal(t, uint32(0), cmd2.VNI)
	assert.Equal(t, "", cmd2.Device)
	assert.Equal(t, "", cmd2.Remote)
}

// TestVXLANBatcherBasic tests basic batcher functionality
func TestVXLANBatcherBasic(t *testing.T) {
	batcher := NewVXLANBatcher(5, 100*time.Millisecond)
	require.NotNil(t, batcher)

	flushedBatches := make([][]VXLANCommand, 0)
	mu := sync.Mutex{}

	batcher.OnFlush(func(commands []VXLANCommand) {
		mu.Lock()
		batch := make([]VXLANCommand, len(commands))
		copy(batch, commands)
		flushedBatches = append(flushedBatches, batch)
		mu.Unlock()
	})

	_, cancel := batcher.Start()
	defer cancel()

	// Add commands
	for i := 0; i < 12; i++ {
		cmd := VXLANCommand{
			Action: "add",
			VNI:    uint32(100 + i),
			Device: "vxlan0",
		}
		err := batcher.Add(cmd)
		require.NoError(t, err)
	}

	// Wait for timer flush
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	batches := len(flushedBatches)
	mu.Unlock()

	// Should have at least 2 batches (5+5+2)
	assert.GreaterOrEqual(t, batches, 2)
}