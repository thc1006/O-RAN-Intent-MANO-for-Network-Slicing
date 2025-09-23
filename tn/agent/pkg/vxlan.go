package pkg

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
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
	// Check if we should mock VXLAN operations in CI
	if ShouldMockVXLAN() {
		vm.logger.Println("Running in CI environment - mocking VXLAN tunnel creation")
		return nil
	}

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

	security.SafeLogf(vm.logger, "Creating VXLAN tunnel %s with VNI %d", security.SanitizeForLog(vm.config.DeviceName), vm.config.VNI)

	// Delete existing tunnel if it exists
	_ = vm.DeleteTunnel()

	// Create VXLAN interface
	vniStr := strconv.Itoa(int(vm.config.VNI))
	portStr := strconv.Itoa(vm.config.Port)
	ipArgs := []string{"link", "add", vm.config.DeviceName, "type", "vxlan", "id", vniStr, "dstport", portStr, "local", vm.config.LocalIP}

	// Add learning/nolearning parameter
	if vm.config.Learning {
		ipArgs = append(ipArgs, "learning")
	} else {
		ipArgs = append(ipArgs, "nolearning")
	}

	// Use secure ip command execution
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, ipArgs...); err != nil {
		return fmt.Errorf("failed to create VXLAN interface: %v", err)
	}

	// Set MTU
	if vm.config.MTU > 0 {
		// Validate MTU range
		if vm.config.MTU < 576 || vm.config.MTU > 9000 {
			return fmt.Errorf("invalid MTU: %d (must be 576-9000)", vm.config.MTU)
		}
		mtuStr := strconv.Itoa(vm.config.MTU)
		mtuArgs := []string{"link", "set", "dev", vm.config.DeviceName, "mtu", mtuStr}

		// Use secure ip command execution for MTU setting
		mtuCtx, mtuCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer mtuCancel()

		if _, err := security.SecureExecuteWithValidation(mtuCtx, "ip", security.ValidateIPArgs, mtuArgs...); err != nil {
			security.SafeLogError(vm.logger, "Warning: failed to set MTU", err)
		}
	}

	// Bring interface up
	upArgs := []string{"link", "set", "dev", vm.config.DeviceName, "up"}

	// Use secure ip command execution to bring interface up
	upCtx, upCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer upCancel()

	if _, err := security.SecureExecuteWithValidation(upCtx, "ip", security.ValidateIPArgs, upArgs...); err != nil {
		return fmt.Errorf("failed to bring interface up: %v", err)
	}

	security.SafeLogf(vm.logger, "VXLAN tunnel %s created successfully", security.SanitizeForLog(vm.config.DeviceName))

	// Add FDB entries for remote peers
	if err := vm.addFDBEntries(); err != nil {
		security.SafeLogError(vm.logger, "Warning: failed to configure interface", err)
	}

	return nil
}

// DeleteTunnel removes the VXLAN tunnel
func (vm *VXLANManager) DeleteTunnel() error {
	// Check if we should mock VXLAN operations in CI
	if ShouldMockVXLAN() {
		vm.logger.Println("Running in CI environment - mocking VXLAN tunnel deletion")
		return nil
	}

	// Validate device name for security
	if err := security.ValidateNetworkInterface(vm.config.DeviceName); err != nil {
		return fmt.Errorf("invalid device name: %w", err)
	}

	security.SafeLogf(vm.logger, "Deleting VXLAN tunnel %s", security.SanitizeForLog(vm.config.DeviceName))

	delArgs := []string{"link", "delete", vm.config.DeviceName}

	// Use secure ip command execution for interface deletion
	delCtx, delCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer delCancel()

	if _, err := security.SecureExecuteWithValidation(delCtx, "ip", security.ValidateIPArgs, delArgs...); err != nil {
		// Don't return error if interface doesn't exist
		if !strings.Contains(err.Error(), "Cannot find device") {
			return fmt.Errorf("failed to delete interface: %v", err)
		}
	}

	security.SafeLogf(vm.logger, "VXLAN tunnel %s deleted", security.SanitizeForLog(vm.config.DeviceName))
	return nil
}

// addFDBEntries adds forwarding database entries for remote peers
func (vm *VXLANManager) addFDBEntries() error {
	for _, remoteIP := range vm.config.RemoteIPs {
		// Validate remote IP for security
		if err := security.ValidateIPAddress(remoteIP); err != nil {
			security.SafeLogf(vm.logger, "Warning: skipping invalid remote IP %s: %s", security.SanitizeIPForLog(remoteIP), security.SanitizeErrorForLog(err))
			continue
		}

		// Add default FDB entry (all-zeros MAC) for each remote IP
		fdbArgs := []string{"fdb", "append", "00:00:00:00:00:00", "dev", vm.config.DeviceName, "dst", remoteIP}

		// Use secure bridge command execution for FDB entries
		fdbCtx, fdbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer fdbCancel()

		if _, err := security.SecureExecute(fdbCtx, "bridge", fdbArgs...); err != nil {
			security.SafeLogf(vm.logger, "Warning: failed to add FDB entry for %s: %s", security.SanitizeIPForLog(remoteIP), security.SanitizeErrorForLog(err))
		} else {
			security.SafeLogf(vm.logger, "Added FDB entry for remote peer %s", security.SanitizeIPForLog(remoteIP))
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

	// Mock status in CI environment
	if ShouldMockVXLAN() {
		status.TunnelUp = true
		status.PacketStats["rx_packets"] = 1000
		status.PacketStats["tx_packets"] = 1000
		status.PacketStats["rx_bytes"] = 100000
		status.PacketStats["tx_bytes"] = 100000
		return status, nil
	}

	// Check if interface exists and is up
	showArgs := []string{"link", "show", vm.config.DeviceName}

	// Use secure ip command execution to check interface status
	showCtx, showCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer showCancel()

	output, err := security.SecureExecuteWithValidation(showCtx, "ip", security.ValidateIPArgs, showArgs...)
	if err != nil {
		return status, fmt.Errorf("failed to get interface status: %v", err)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "state UP") {
		status.TunnelUp = true
	}

	// Get packet statistics
	statsArgs := []string{"-s", "link", "show", vm.config.DeviceName}

	// Use secure ip command execution for statistics
	statsCtx, statsCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer statsCancel()

	if statsOutput, err := security.SecureExecuteWithValidation(statsCtx, "ip", security.ValidateIPArgs, statsArgs...); err == nil {
		stats := vm.parsePacketStats(string(statsOutput))
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

	// Mock connectivity in CI environment
	if ShouldMockVXLAN() {
		for _, remoteIP := range vm.config.RemoteIPs {
			results[remoteIP] = true
		}
		return results
	}

	for _, remoteIP := range vm.config.RemoteIPs {
		// Validate remote IP for security
		if err := security.ValidateIPAddress(remoteIP); err != nil {
			security.SafeLogf(vm.logger, "Warning: skipping invalid remote IP %s: %s", security.SanitizeIPForLog(remoteIP), security.SanitizeErrorForLog(err))
			results[remoteIP] = false
			continue
		}

		security.SafeLogf(vm.logger, "Testing connectivity to %s", security.SanitizeIPForLog(remoteIP))

		// Use ping to test connectivity
		pingArgs := []string{"-c", "3", "-W", "2", remoteIP}

		// Use secure ping command execution
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer pingCancel()

		_, err := security.SecureExecute(pingCtx, "ping", pingArgs...)

		results[remoteIP] = (err == nil)
		if err == nil {
			security.SafeLogf(vm.logger, "Connectivity to %s: OK", security.SanitizeIPForLog(remoteIP))
		} else {
			security.SafeLogf(vm.logger, "Connectivity to %s: FAILED", security.SanitizeIPForLog(remoteIP))
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

	security.SafeLogf(vm.logger, "Updating VXLAN peers from %v to %v", vm.config.RemoteIPs, newPeers)

	// Remove old FDB entries
	for _, oldIP := range vm.config.RemoteIPs {
		if err := security.ValidateIPAddress(oldIP); err != nil {
			continue // Skip invalid IPs
		}
		delFdbArgs := []string{"fdb", "del", "00:00:00:00:00:00", "dev", vm.config.DeviceName, "dst", oldIP}

		// Use secure bridge command execution for FDB deletion
		delFdbCtx, delFdbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer delFdbCancel()

		_, _ = security.SecureExecute(delFdbCtx, "bridge", delFdbArgs...) // Ignore errors for non-existent entries
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
	detailArgs := []string{"-d", "link", "show", vm.config.DeviceName}

	// Use secure ip command execution for interface details
	detailCtx, detailCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer detailCancel()

	detailOutput, err := security.SecureExecuteWithValidation(detailCtx, "ip", security.ValidateIPArgs, detailArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface details: %v", err)
	}

	info["interface_details"] = string(detailOutput)
	info["vni"] = vm.config.VNI
	info["local_ip"] = vm.config.LocalIP
	info["remote_ips"] = vm.config.RemoteIPs
	info["port"] = vm.config.Port
	info["mtu"] = vm.config.MTU

	// Get FDB entries
	fdbShowArgs := []string{"fdb", "show", "dev", vm.config.DeviceName}

	// Use secure bridge command execution for FDB show
	fdbShowCtx, fdbShowCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer fdbShowCancel()

	if fdbOutput, err := security.SecureExecute(fdbShowCtx, "bridge", fdbShowArgs...); err == nil {
		info["fdb_entries"] = string(fdbOutput)
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
				security.SafeLogError(vm.logger, "Failed to get tunnel status", err)
				continue
			}

			if !status.TunnelUp {
				security.SafeLogf(vm.logger, "VXLAN tunnel %s is down, attempting to recreate", security.SanitizeForLog(vm.config.DeviceName))
				if err := vm.CreateTunnel(); err != nil {
					security.SafeLogError(vm.logger, "Failed to recreate VXLAN tunnel", err)
				}
			}

			// Test connectivity periodically
			connectivity := vm.TestConnectivity()
			failedPeers := 0
			for peer, connected := range connectivity {
				if !connected {
					failedPeers++
					security.SafeLogf(vm.logger, "Lost connectivity to peer %s", security.SanitizeIPForLog(peer))
				}
			}

			if failedPeers > 0 {
				security.SafeLogf(vm.logger, "Warning: %d/%d peers unreachable", failedPeers, len(connectivity))
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