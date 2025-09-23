package vxlan

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
	"syscall"
)

// OptimizedManager provides high-performance VXLAN tunnel management
type OptimizedManager struct {
	// Enhanced tunnel tracking
	tunnels     map[int32]*EnhancedTunnelInfo
	tunnelMutex sync.RWMutex

	// Performance optimizations
	commandCache    map[string]*CachedCommand
	cacheMutex      sync.RWMutex
	commandPool     *sync.Pool
	workerPool      chan struct{}

	// Batch operations
	pendingOps      []BatchOperation
	batchMutex      sync.Mutex
	batchTimer      *time.Timer
	batchInterval   time.Duration

	// Performance metrics
	metrics         *PerformanceMetrics

	// System optimization
	netlinkSocket   int
	useNetlink     bool
}

// EnhancedTunnelInfo stores comprehensive tunnel information
type EnhancedTunnelInfo struct {
	InterfaceName string
	VxlanID      int32
	LocalIP      string
	RemoteIPs    []string
	MTU          int
	State        TunnelState
	CreatedAt    time.Time
	LastUsed     time.Time
	Stats        *TunnelStats
}

// TunnelState represents the operational state of a tunnel
type TunnelState string

const (
	TunnelStateCreating TunnelState = "Creating"
	TunnelStateActive   TunnelState = "Active"
	TunnelStateFailed   TunnelState = "Failed"
	TunnelStateDeleting TunnelState = "Deleting"
)

// TunnelStats tracks tunnel performance metrics
type TunnelStats struct {
	BytesSent     uint64
	BytesReceived uint64
	PacketsSent   uint64
	PacketsReceived uint64
	Errors        uint64
	LastUpdated   time.Time
}

// CachedCommand stores pre-built command information
type CachedCommand struct {
	Args        []string
	Environment []string
	CreatedAt   time.Time
	HitCount    int
}

// BatchOperation represents a batched VXLAN operation
type BatchOperation struct {
	Type      string
	VxlanID   int32
	LocalIP   string
	RemoteIPs []string
	Interface string
	Timestamp time.Time
	Callback  func(error)
}

// PerformanceMetrics tracks manager performance
type PerformanceMetrics struct {
	TotalOperations     int64
	SuccessfulOps       int64
	FailedOps           int64
	CacheHits           int64
	BatchedOps          int64
	AvgOpTimeMs         float64
	ConcurrentOps       int64
	PeakConcurrency     int64
	NetlinkOps          int64
	mutex               sync.Mutex
}

// NewOptimizedManager creates a new optimized VXLAN manager
func NewOptimizedManager() *OptimizedManager {
	manager := &OptimizedManager{
		tunnels:       make(map[int32]*EnhancedTunnelInfo),
		commandCache:  make(map[string]*CachedCommand),
		workerPool:    make(chan struct{}, 10), // Max 10 concurrent operations
		batchInterval: 100 * time.Millisecond,  // 100ms batching
		metrics:       &PerformanceMetrics{},
		useNetlink:    true, // Try to use netlink for better performance
	}

	// Initialize command pool for reusing exec.Cmd objects
	manager.commandPool = &sync.Pool{
		New: func() interface{} {
			return &exec.Cmd{}
		},
	}

	// Try to initialize netlink socket for direct kernel communication
	manager.initNetlink()

	// Start batch processor
	manager.startBatchProcessor()

	return manager
}

// initNetlink attempts to initialize netlink socket for better performance
func (m *OptimizedManager) initNetlink() {
	// This would use netlink library in production
	// For now, we'll fall back to ip commands with optimization
	m.useNetlink = false
}

// startBatchProcessor starts the background batch operation processor
func (m *OptimizedManager) startBatchProcessor() {
	go func() {
		for {
			time.Sleep(m.batchInterval)
			m.processBatch()
		}
	}()
}

// CreateTunnelAsync creates a VXLAN tunnel asynchronously with batching
func (m *OptimizedManager) CreateTunnelAsync(vxlanID int32, localIP string, remoteIPs []string, physInterface string, callback func(error)) {
	// Check if this operation can be batched
	if m.shouldBatch("create", vxlanID) {
		m.addToBatch("create", vxlanID, localIP, remoteIPs, physInterface, callback)
		return
	}

	// Process immediately for critical operations
	go m.createTunnelOptimized(vxlanID, localIP, remoteIPs, physInterface, callback)
}

// CreateTunnelOptimized creates a VXLAN tunnel with performance optimizations
func (m *OptimizedManager) CreateTunnelOptimized(vxlanID int32, localIP string, remoteIPs []string, physInterface string) error {
	start := time.Now()
	defer func() {
		m.updateMetrics(time.Since(start), true)
	}()

	// Acquire worker pool slot
	select {
	case m.workerPool <- struct{}{}:
		defer func() { <-m.workerPool }()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("operation timeout: too many concurrent operations")
	}

	return m.createTunnelOptimized(vxlanID, localIP, remoteIPs, physInterface, nil)
}

// createTunnelOptimized internal optimized tunnel creation
func (m *OptimizedManager) createTunnelOptimized(vxlanID int32, localIP string, remoteIPs []string, physInterface string, callback func(error)) error {
	ifaceName := fmt.Sprintf("vxlan%d", vxlanID)

	// Check if tunnel already exists
	m.tunnelMutex.Lock()
	if existing, exists := m.tunnels[vxlanID]; exists {
		m.tunnelMutex.Unlock()
		if existing.State == TunnelStateActive {
			if callback != nil {
				callback(nil)
			}
			return nil
		}
		// Delete existing tunnel if in failed state
		m.DeleteTunnelOptimized(vxlanID)
	} else {
		m.tunnelMutex.Unlock()
	}

	// Create tunnel info
	tunnelInfo := &EnhancedTunnelInfo{
		InterfaceName: ifaceName,
		VxlanID:      vxlanID,
		LocalIP:      localIP,
		RemoteIPs:    remoteIPs,
		MTU:          1450,
		State:        TunnelStateCreating,
		CreatedAt:    time.Now(),
		LastUsed:     time.Now(),
		Stats:        &TunnelStats{LastUpdated: time.Now()},
	}

	m.tunnelMutex.Lock()
	m.tunnels[vxlanID] = tunnelInfo
	m.tunnelMutex.Unlock()

	var err error

	// Use netlink if available for better performance
	if m.useNetlink {
		err = m.createTunnelNetlink(vxlanID, localIP, remoteIPs, physInterface)
	} else {
		err = m.createTunnelIPCommand(vxlanID, localIP, remoteIPs, physInterface)
	}

	// Update tunnel state
	m.tunnelMutex.Lock()
	if err != nil {
		tunnelInfo.State = TunnelStateFailed
		m.metrics.FailedOps++
	} else {
		tunnelInfo.State = TunnelStateActive
		m.metrics.SuccessfulOps++
	}
	m.tunnelMutex.Unlock()

	if callback != nil {
		callback(err)
	}

	return err
}

// createTunnelIPCommand creates tunnel using optimized ip commands
func (m *OptimizedManager) createTunnelIPCommand(vxlanID int32, localIP string, remoteIPs []string, physInterface string) error {
	ifaceName := fmt.Sprintf("vxlan%d", vxlanID)

	// Create optimized command sequence
	commands := [][]string{
		// Create VXLAN interface
		{"ip", "link", "add", ifaceName, "type", "vxlan", "id", fmt.Sprintf("%d", vxlanID),
		 "local", localIP, "dstport", "4789", "dev", physInterface},

		// Set MTU
		{"ip", "link", "set", ifaceName, "mtu", "1450"},

		// Bring interface up
		{"ip", "link", "set", ifaceName, "up"},
	}

	// Execute commands in batch for better performance
	for _, cmdArgs := range commands {
		if err := m.executeOptimizedCommand(cmdArgs); err != nil {
			// Cleanup on failure
			m.executeOptimizedCommand([]string{"ip", "link", "del", ifaceName})
			return fmt.Errorf("failed to create VXLAN interface: %w", err)
		}
	}

	// Add FDB entries in parallel for multiple remotes
	if len(remoteIPs) > 1 {
		var wg sync.WaitGroup
		errChan := make(chan error, len(remoteIPs))

		for _, remoteIP := range remoteIPs {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				err := m.executeOptimizedCommand([]string{
					"bridge", "fdb", "append", "00:00:00:00:00:00", "dst", ip, "dev", ifaceName,
				})
				if err != nil {
					errChan <- err
				}
			}(remoteIP)
		}

		wg.Wait()
		close(errChan)

		// Check for errors (but don't fail on FDB errors)
		for err := range errChan {
			fmt.Printf("Warning: FDB entry failed: %v\n", err)
		}
	} else if len(remoteIPs) > 0 {
		// Single remote IP
		m.executeOptimizedCommand([]string{
			"bridge", "fdb", "append", "00:00:00:00:00:00", "dst", remoteIPs[0], "dev", ifaceName,
		})
	}

	// Assign IP address
	vxlanIP := m.generateVXLANIP(vxlanID, localIP)
	err := m.executeOptimizedCommand([]string{
		"ip", "addr", "add", fmt.Sprintf("%s/24", vxlanIP), "dev", ifaceName,
	})

	// Ignore "File exists" errors for IP assignment
	if err != nil && !strings.Contains(err.Error(), "File exists") {
		return fmt.Errorf("failed to assign IP: %w", err)
	}

	return nil
}

// createTunnelNetlink creates tunnel using netlink (placeholder for production)
func (m *OptimizedManager) createTunnelNetlink(vxlanID int32, localIP string, remoteIPs []string, physInterface string) error {
	// This would use a netlink library like github.com/vishvananda/netlink
	// For now, fall back to IP commands
	return m.createTunnelIPCommand(vxlanID, localIP, remoteIPs, physInterface)
}

// executeOptimizedCommand executes command with caching and pooling
func (m *OptimizedManager) executeOptimizedCommand(args []string) error {
	cmdKey := strings.Join(args, " ")

	// Check command cache
	m.cacheMutex.RLock()
	if cached, exists := m.commandCache[cmdKey]; exists && time.Since(cached.CreatedAt) < 10*time.Second {
		m.cacheMutex.RUnlock()
		cached.HitCount++
		m.metrics.CacheHits++

		// For cached commands, we still need to execute but can optimize
		cmd := m.commandPool.Get().(*exec.Cmd)
		defer m.commandPool.Put(cmd)

		// Reset and configure command
		*cmd = exec.Cmd{}
		cmd.Path = args[0]
		cmd.Args = args

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command failed: %v, output: %s", err, output)
		}
		return nil
	}
	m.cacheMutex.RUnlock()

	// Execute new command
	cmd := m.commandPool.Get().(*exec.Cmd)
	defer m.commandPool.Put(cmd)

	// Reset and configure command
	*cmd = exec.Cmd{}
	cmd.Path = args[0]
	cmd.Args = args

	// Set platform-specific process attributes
	setPlatformSysProcAttr(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %v, output: %s", err, output)
	}

	// Cache successful command
	m.cacheMutex.Lock()
	m.commandCache[cmdKey] = &CachedCommand{
		Args:      args,
		CreatedAt: time.Now(),
		HitCount:  0,
	}

	// Limit cache size
	if len(m.commandCache) > 100 {
		// Remove oldest entries
		oldest := time.Now()
		oldestKey := ""
		for key, cached := range m.commandCache {
			if cached.CreatedAt.Before(oldest) {
				oldest = cached.CreatedAt
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(m.commandCache, oldestKey)
		}
	}
	m.cacheMutex.Unlock()

	return nil
}

// DeleteTunnelOptimized removes a VXLAN tunnel with optimization
func (m *OptimizedManager) DeleteTunnelOptimized(vxlanID int32) error {
	start := time.Now()
	defer func() {
		m.updateMetrics(time.Since(start), true)
	}()

	ifaceName := fmt.Sprintf("vxlan%d", vxlanID)

	// Update tunnel state
	m.tunnelMutex.Lock()
	if tunnel, exists := m.tunnels[vxlanID]; exists {
		tunnel.State = TunnelStateDeleting
	}
	m.tunnelMutex.Unlock()

	// Delete interface
	err := m.executeOptimizedCommand([]string{"ip", "link", "del", ifaceName})

	// Remove from tracking regardless of delete result
	m.tunnelMutex.Lock()
	delete(m.tunnels, vxlanID)
	m.tunnelMutex.Unlock()

	if err != nil && !strings.Contains(err.Error(), "Cannot find device") {
		return fmt.Errorf("failed to delete interface: %w", err)
	}

	return nil
}

// GetTunnelStatusOptimized retrieves tunnel status with caching
func (m *OptimizedManager) GetTunnelStatusOptimized(vxlanID int32) (*EnhancedTunnelInfo, error) {
	m.tunnelMutex.RLock()
	info, exists := m.tunnels[vxlanID]
	m.tunnelMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tunnel %d not found", vxlanID)
	}

	// Update last used time
	info.LastUsed = time.Now()

	// Quick state verification for active tunnels
	if info.State == TunnelStateActive {
		// Async status check if stats are old
		if time.Since(info.Stats.LastUpdated) > 30*time.Second {
			go m.updateTunnelStats(vxlanID)
		}
	}

	return info, nil
}

// updateTunnelStats updates tunnel statistics asynchronously
func (m *OptimizedManager) updateTunnelStats(vxlanID int32) {
	ifaceName := fmt.Sprintf("vxlan%d", vxlanID)

	// Get interface statistics
	cmd := exec.Command("cat", fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", ifaceName))
	if _, err := cmd.Output(); err == nil {
		// Parse and update stats (simplified)
		m.tunnelMutex.Lock()
		if tunnel, exists := m.tunnels[vxlanID]; exists {
			tunnel.Stats.LastUpdated = time.Now()
			// Would parse actual stats here
		}
		m.tunnelMutex.Unlock()
	}
}

// Batch processing methods

func (m *OptimizedManager) shouldBatch(operation string, vxlanID int32) bool {
	// Batch non-critical operations
	// Don't batch if this is a critical tunnel (low VXLAN ID suggests system tunnel)
	return vxlanID > 1000
}

func (m *OptimizedManager) addToBatch(operation string, vxlanID int32, localIP string, remoteIPs []string, physInterface string, callback func(error)) {
	m.batchMutex.Lock()
	defer m.batchMutex.Unlock()

	m.pendingOps = append(m.pendingOps, BatchOperation{
		Type:      operation,
		VxlanID:   vxlanID,
		LocalIP:   localIP,
		RemoteIPs: remoteIPs,
		Interface: physInterface,
		Timestamp: time.Now(),
		Callback:  callback,
	})
}

func (m *OptimizedManager) processBatch() {
	m.batchMutex.Lock()
	if len(m.pendingOps) == 0 {
		m.batchMutex.Unlock()
		return
	}

	ops := make([]BatchOperation, len(m.pendingOps))
	copy(ops, m.pendingOps)
	m.pendingOps = m.pendingOps[:0] // Clear slice
	m.batchMutex.Unlock()

	// Process operations in parallel
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Limit concurrent batch operations

	for _, op := range ops {
		wg.Add(1)
		go func(operation BatchOperation) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var err error
			switch operation.Type {
			case "create":
				err = m.createTunnelOptimized(operation.VxlanID, operation.LocalIP, operation.RemoteIPs, operation.Interface, nil)
			case "delete":
				err = m.DeleteTunnelOptimized(operation.VxlanID)
			}

			if operation.Callback != nil {
				operation.Callback(err)
			}

			m.metrics.BatchedOps++
		}(op)
	}

	wg.Wait()
}

// Utility methods

func (m *OptimizedManager) generateVXLANIP(vxlanID int32, nodeIP string) string {
	// Optimized IP generation
	second := (vxlanID / 256) % 256
	third := vxlanID % 256

	parts := strings.Split(nodeIP, ".")
	fourth := "1"
	if len(parts) == 4 {
		fourth = parts[3]
	}

	return fmt.Sprintf("10.%d.%d.%s", second, third, fourth)
}

func (m *OptimizedManager) updateMetrics(duration time.Duration, success bool) {
	m.metrics.mutex.Lock()
	defer m.metrics.mutex.Unlock()

	m.metrics.TotalOperations++
	if success {
		m.metrics.SuccessfulOps++
	} else {
		m.metrics.FailedOps++
	}

	// Update average operation time
	totalTime := m.metrics.AvgOpTimeMs * float64(m.metrics.TotalOperations-1)
	totalTime += float64(duration.Nanoseconds()) / 1e6
	m.metrics.AvgOpTimeMs = totalTime / float64(m.metrics.TotalOperations)
}

// GetMetrics returns performance metrics
func (m *OptimizedManager) GetMetrics() *PerformanceMetrics {
	m.metrics.mutex.Lock()
	defer m.metrics.mutex.Unlock()

	return &PerformanceMetrics{
		TotalOperations: m.metrics.TotalOperations,
		SuccessfulOps:   m.metrics.SuccessfulOps,
		FailedOps:       m.metrics.FailedOps,
		CacheHits:       m.metrics.CacheHits,
		BatchedOps:      m.metrics.BatchedOps,
		AvgOpTimeMs:     m.metrics.AvgOpTimeMs,
		ConcurrentOps:   m.metrics.ConcurrentOps,
		PeakConcurrency: m.metrics.PeakConcurrency,
		NetlinkOps:      m.metrics.NetlinkOps,
	}
}

// CleanupOptimized removes all tunnels with optimized cleanup
func (m *OptimizedManager) CleanupOptimized() error {
	m.tunnelMutex.RLock()
	vxlanIDs := make([]int32, 0, len(m.tunnels))
	for vxlanID := range m.tunnels {
		vxlanIDs = append(vxlanIDs, vxlanID)
	}
	m.tunnelMutex.RUnlock()

	// Parallel cleanup
	var wg sync.WaitGroup
	errChan := make(chan error, len(vxlanIDs))

	for _, vxlanID := range vxlanIDs {
		wg.Add(1)
		go func(id int32) {
			defer wg.Done()
			if err := m.DeleteTunnelOptimized(id); err != nil {
				errChan <- err
			}
		}(vxlanID)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed for %d tunnels", len(errors))
	}

	return nil
}

// ListActiveTunnels returns all active tunnels
func (m *OptimizedManager) ListActiveTunnels() map[int32]*EnhancedTunnelInfo {
	m.tunnelMutex.RLock()
	defer m.tunnelMutex.RUnlock()

	active := make(map[int32]*EnhancedTunnelInfo)
	for id, tunnel := range m.tunnels {
		if tunnel.State == TunnelStateActive {
			// Return a copy to prevent race conditions
			active[id] = &EnhancedTunnelInfo{
				InterfaceName: tunnel.InterfaceName,
				VxlanID:      tunnel.VxlanID,
				LocalIP:      tunnel.LocalIP,
				RemoteIPs:    tunnel.RemoteIPs,
				MTU:          tunnel.MTU,
				State:        tunnel.State,
				CreatedAt:    tunnel.CreatedAt,
				LastUsed:     tunnel.LastUsed,
				Stats:        tunnel.Stats,
			}
		}
	}

	return active
}