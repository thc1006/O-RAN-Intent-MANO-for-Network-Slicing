package vxlan

import (
	"sync"
	"sync/atomic"
)

// VXLANCommand represents a VXLAN operation command
type VXLANCommand struct {
	Action string
	VNI    uint32
	Device string
	Remote string
}

// PoolStats tracks pool performance metrics
type PoolStats struct {
	Allocations uint64
	Reuses      uint64
}

// VXLANCommandPool provides object pooling for VXLANCommand instances
type VXLANCommandPool struct {
	pool        *sync.Pool
	allocations uint64
	reuses      uint64
}

// NewVXLANCommandPool creates a new command pool
func NewVXLANCommandPool() *VXLANCommandPool {
	p := &VXLANCommandPool{}

	p.pool = &sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&p.allocations, 1)
			return &VXLANCommand{}
		},
	}

	return p
}

// Get retrieves a command from the pool
func (p *VXLANCommandPool) Get() *VXLANCommand {
	cmd := p.pool.Get().(*VXLANCommand)

	// Track reuses - a Get after initial allocation is a reuse
	// Since New() increments allocations, any Get beyond allocations is a reuse
	currentAllocs := atomic.LoadUint64(&p.allocations)
	if currentAllocs > 0 {
		atomic.AddUint64(&p.reuses, 1)
	}

	// Reset command for clean state
	cmd.Action = ""
	cmd.VNI = 0
	cmd.Device = ""
	cmd.Remote = ""

	return cmd
}

// Put returns a command to the pool
func (p *VXLANCommandPool) Put(cmd *VXLANCommand) {
	if cmd == nil {
		return
	}

	// Clear command before returning to pool
	cmd.Action = ""
	cmd.VNI = 0
	cmd.Device = ""
	cmd.Remote = ""

	p.pool.Put(cmd)
}

// Stats returns pool statistics
func (p *VXLANCommandPool) Stats() PoolStats {
	return PoolStats{
		Allocations: atomic.LoadUint64(&p.allocations),
		Reuses:      atomic.LoadUint64(&p.reuses),
	}
}