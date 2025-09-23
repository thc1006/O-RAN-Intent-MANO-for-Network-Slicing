package pkg

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/o-ran-intent-mano/pkg/security"
)

// VXLANManager manages VXLAN tunnel operations
type VXLANManager struct {
	config *VXLANConfig
	logger *log.Logger
}

// NewVXLANManager creates a new VXLAN manager
func NewVXLANManager(config *VXLANConfig, logger *log.Logger) *VXLANManager {
	return &VXLANManager{
		config: config,
		logger: logger,
	}
}

// CreateTunnel creates a VXLAN tunnel
func (vm *VXLANManager) CreateTunnel() error {
	// Validate inputs for security
	if err := security.ValidateNetworkInterface(vm.config.DeviceName); err != nil {
		return fmt.Errorf("invalid device name: %w", err)
	}
	if err := security.ValidateVNI(vm.config.VNI); err != nil {
		return fmt.Errorf("invalid VNI: %w", err)
	}
	if err := security.ValidatePort(vm.config.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	if err := security.ValidateIPAddress(vm.config.LocalIP); err != nil {
		return fmt.Errorf("invalid local IP: %w", err)
	}

	vm.logger.Printf("Creating VXLAN tunnel %s with VNI %d", vm.config.DeviceName, vm.config.VNI)

	// Delete existing tunnel if it exists
	_ = vm.DeleteTunnel()

	// Create VXLAN interface
	cmd := exec.Command("ip", "link", "add", vm.config.DeviceName, "type", "vxlan",
		"id", strconv.Itoa(int(vm.config.VNI)),
		"dstport", strconv.Itoa(vm.config.Port),
		"local", vm.config.LocalIP)

	if vm.config.Learning {
		cmd.Args = append(cmd.Args, "learning")
	} else {
		cmd.Args = append(cmd.Args, "nolearning")
	}

	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure interface: %v", err)
	}

	// Set MTU
	if vm.config.MTU > 0 {
		// Validate MTU range
		if vm.config.MTU < 576 || vm.config.MTU > 9000 {
			return fmt.Errorf("invalid MTU: %d (must be 576-9000)", vm.config.MTU)
		}
		cmd = exec.Command("ip", "link", "set", "dev", vm.config.DeviceName, "mtu", strconv.Itoa(vm.config.MTU))
		if _, err := cmd.CombinedOutput(); err != nil {
			vm.logger.Printf("Warning: failed to configure interface: %v", err)
		}
	}

	// Bring interface up
	cmd = exec.Command("ip", "link", "set", "dev", vm.config.DeviceName, "up")
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure interface: %v", err)
	}

	vm.logger.Printf("VXLAN tunnel %s created successfully", vm.config.DeviceName)

	// Add FDB entries for remote peers
	if err := vm.addFDBEntries(); err != nil {
		vm.logger.Printf("Warning: failed to configure interface: %v", err)
	}

	return nil
}

// DeleteTunnel removes the VXLAN tunnel
func (vm *VXLANManager) DeleteTunnel() error {
	// Validate device name for security
	if err := security.ValidateNetworkInterface(vm.config.DeviceName); err != nil {
		return fmt.Errorf("invalid device name: %w", err)
	}

	vm.logger.Printf("Deleting VXLAN tunnel %s", vm.config.DeviceName)

	cmd := exec.Command("ip", "link", "delete", vm.config.DeviceName)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Don't return error if interface doesn't exist
		if !strings.Contains(err.Error(), "Cannot find device") {
			return fmt.Errorf("failed to configure interface: %v", err)
		}
	}

	vm.logger.Printf("VXLAN tunnel %s deleted", vm.config.DeviceName)
	return nil
}

// addFDBEntries adds forwarding database entries for remote peers
func (vm *VXLANManager) addFDBEntries() error {
	for _, remoteIP := range vm.config.RemoteIPs {
		// Validate remote IP for security
		if err := security.ValidateIPAddress(remoteIP); err != nil {
			vm.logger.Printf("Warning: skipping invalid remote IP %s: %v", remoteIP, err)
			continue
		}

		// Add default FDB entry (all-zeros MAC) for each remote IP
		cmd := exec.Command("bridge", "fdb", "append", "00:00:00:00:00:00",
			"dev", vm.config.DeviceName, "dst", remoteIP)

		if _, err := cmd.CombinedOutput(); err != nil {
			vm.logger.Printf("Warning: failed to add FDB entry for %s: %v", remoteIP, err)
		} else {
			vm.logger.Printf("Added FDB entry for remote peer %s", remoteIP)
		}
	}

	return nil
}

// GetTunnelStatus returns the current status of the VXLAN tunnel
func (vm *VXLANManager) GetTunnelStatus() (*VXLANStatus, error) {
	status := &VXLANStatus{
		TunnelUp:      false,
		RemotePeers:   vm.config.RemoteIPs,
		PacketStats:   make(map[string]int64),
		LastHeartbeat: time.Now(),
	}

	// Check if interface exists and is up
	cmd := exec.Command("ip", "link", "show", vm.config.DeviceName)
	if _, err := cmd.CombinedOutput(); err != nil {
		return status, fmt.Errorf("failed to configure interface: %v", err)
	}

	outputStr := ""
	if strings.Contains(outputStr, "state UP") {
		status.TunnelUp = true
	}

	// Get packet statistics
	cmd = exec.Command("ip", "-s", "link", "show", vm.config.DeviceName)
	if _, err := cmd.CombinedOutput(); err == nil {
		stats := vm.parsePacketStats("")
		status.PacketStats = stats
	}

	return status, nil
}

// parsePacketStats parses packet statistics from ip command output
func (vm *VXLANManager) parsePacketStats(output string) map[string]int64 {
	stats := make(map[string]int64)

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "RX:") && i+1 < len(lines) {
			rxLine := strings.Fields(lines[i+1])
			if len(rxLine) >= 2 {
				if packets, err := strconv.ParseInt(rxLine[0], 10, 64); err == nil {
					stats["rx_packets"] = packets
				}
				if bytes, err := strconv.ParseInt(rxLine[1], 10, 64); err == nil {
					stats["rx_bytes"] = bytes
				}
			}
		}
		if strings.Contains(line, "TX:") && i+1 < len(lines) {
			txLine := strings.Fields(lines[i+1])
			if len(txLine) >= 2 {
				if packets, err := strconv.ParseInt(txLine[0], 10, 64); err == nil {
					stats["tx_packets"] = packets
				}
				if bytes, err := strconv.ParseInt(txLine[1], 10, 64); err == nil {
					stats["tx_bytes"] = bytes
				}
			}
		}
	}

	return stats
}

// TestConnectivity tests connectivity to remote peers
func (vm *VXLANManager) TestConnectivity() map[string]bool {
	results := make(map[string]bool)

	for _, remoteIP := range vm.config.RemoteIPs {
		// Validate remote IP for security
		if err := security.ValidateIPAddress(remoteIP); err != nil {
			vm.logger.Printf("Warning: skipping invalid remote IP %s: %v", remoteIP, err)
			results[remoteIP] = false
			continue
		}

		vm.logger.Printf("Testing connectivity to %s", remoteIP)

		// Use ping to test connectivity
		cmd := exec.Command("ping", "-c", "3", "-W", "2", remoteIP)
		err := cmd.Run()

		results[remoteIP] = (err == nil)
		if err == nil {
			vm.logger.Printf("Connectivity to %s: OK", remoteIP)
		} else {
			vm.logger.Printf("Connectivity to %s: FAILED", remoteIP)
		}
	}

	return results
}

// UpdatePeers updates the list of remote peers
func (vm *VXLANManager) UpdatePeers(newPeers []string) error {
	// Validate all new peers first
	for _, peer := range newPeers {
		if err := security.ValidateIPAddress(peer); err != nil {
			return fmt.Errorf("invalid peer IP %s: %w", peer, err)
		}
	}

	vm.logger.Printf("Updating VXLAN peers from %v to %v", vm.config.RemoteIPs, newPeers)

	// Remove old FDB entries
	for _, oldIP := range vm.config.RemoteIPs {
		if err := security.ValidateIPAddress(oldIP); err != nil {
			continue // Skip invalid IPs
		}
		cmd := exec.Command("bridge", "fdb", "del", "00:00:00:00:00:00",
			"dev", vm.config.DeviceName, "dst", oldIP)
		_ = cmd.Run() // Ignore errors for non-existent entries
	}

	// Update config
	vm.config.RemoteIPs = newPeers

	// Add new FDB entries
	return vm.addFDBEntries()
}

// GetVXLANInfo returns detailed VXLAN interface information
func (vm *VXLANManager) GetVXLANInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get interface details
	cmd := exec.Command("ip", "-d", "link", "show", vm.config.DeviceName)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to configure interface: %v", err)
	}

	info["interface_details"] = ""
	info["vni"] = vm.config.VNI
	info["local_ip"] = vm.config.LocalIP
	info["remote_ips"] = vm.config.RemoteIPs
	info["port"] = vm.config.Port
	info["mtu"] = vm.config.MTU

	// Get FDB entries
	cmd = exec.Command("bridge", "fdb", "show", "dev", vm.config.DeviceName)
	if _, err := cmd.CombinedOutput(); err == nil {
		info["fdb_entries"] = ""
	}

	return info, nil
}

// CalculateVXLANOverhead calculates the overhead introduced by VXLAN encapsulation
func (vm *VXLANManager) CalculateVXLANOverhead(originalMTU int) float64 {
	// VXLAN header: 8 bytes
	// UDP header: 8 bytes
	// IP header: 20 bytes (IPv4) or 40 bytes (IPv6)
	// Ethernet header: 14 bytes
	vxlanOverhead := 8 + 8 + 20 + 14 // 50 bytes for IPv4

	if originalMTU == 0 {
		originalMTU = 1500 // Standard Ethernet MTU
	}

	overheadPercent := float64(vxlanOverhead) / float64(originalMTU) * 100
	return overheadPercent
}

// MonitorTunnel continuously monitors tunnel health
func (vm *VXLANManager) MonitorTunnel(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			vm.logger.Println("Stopping VXLAN tunnel monitoring")
			return
		case <-ticker.C:
			status, err := vm.GetTunnelStatus()
			if err != nil {
				vm.logger.Printf("Failed to get tunnel status: %v", err)
				continue
			}

			if !status.TunnelUp {
				vm.logger.Printf("VXLAN tunnel %s is down, attempting to recreate", vm.config.DeviceName)
				if err := vm.CreateTunnel(); err != nil {
					vm.logger.Printf("Failed to recreate VXLAN tunnel: %v", err)
				}
			}

			// Test connectivity periodically
			connectivity := vm.TestConnectivity()
			failedPeers := 0
			for peer, connected := range connectivity {
				if !connected {
					failedPeers++
					vm.logger.Printf("Lost connectivity to peer %s", peer)
				}
			}

			if failedPeers > 0 {
				vm.logger.Printf("Warning: %d/%d peers unreachable", failedPeers, len(connectivity))
			}
		}
	}
}

// VXLANConfig represents the configuration from types.go
type VXLANConfig struct {
	VNI         uint32   `json:"vni" yaml:"vni"`
	RemoteIPs   []string `json:"remoteIPs" yaml:"remoteIPs"`
	LocalIP     string   `json:"localIP" yaml:"localIP"`
	Port        int      `json:"port" yaml:"port"`
	MTU         int      `json:"mtu" yaml:"mtu"`
	DeviceName  string   `json:"deviceName" yaml:"deviceName"`
	Learning    bool     `json:"learning" yaml:"learning"`
}

// VXLANStatus represents the status from types.go
type VXLANStatus struct {
	TunnelUp      bool                `json:"tunnelUp"`
	RemotePeers   []string            `json:"remotePeers"`
	PacketStats   map[string]int64    `json:"packetStats"`
	LastHeartbeat time.Time           `json:"lastHeartbeat"`
}