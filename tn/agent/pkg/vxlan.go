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
	// Validate all up arguments
	for _, arg := range upArgs {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return fmt.Errorf("invalid up command argument %s: %w", arg, err)
		}
	}
	cmd = exec.Command("ip", upArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure interface: %v", err)
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
	// Validate device name for security
	if err := security.ValidateNetworkInterface(vm.config.DeviceName); err != nil {
		return fmt.Errorf("invalid device name: %w", err)
	}

	security.SafeLogf(vm.logger, "Deleting VXLAN tunnel %s", security.SanitizeForLog(vm.config.DeviceName))

	delArgs := []string{"link", "delete", vm.config.DeviceName}
	// Validate all delete arguments
	for _, arg := range delArgs {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return fmt.Errorf("invalid delete command argument %s: %w", arg, err)
		}
	}
	cmd := exec.Command("ip", delArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		// Don't return error if interface doesn't exist
		if !strings.Contains(err.Error(), "Cannot find device") {
			return fmt.Errorf("failed to configure interface: %v", err)
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
		// Validate all fdb arguments
		for _, arg := range fdbArgs {
			if err := security.ValidateCommandArgument(arg); err != nil {
				security.SafeLogf(vm.logger, "Warning: skipping FDB entry due to invalid argument %s: %s", security.SanitizeForLog(arg), security.SanitizeErrorForLog(err))
				continue
			}
		}
		cmd := exec.Command("bridge", fdbArgs...)

		if _, err := cmd.CombinedOutput(); err != nil {
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

	// Check if interface exists and is up
	showArgs := []string{"link", "show", vm.config.DeviceName}
	// Validate all show arguments
	for _, arg := range showArgs {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return status, fmt.Errorf("invalid show command argument %s: %w", arg, err)
		}
	}
	cmd := exec.Command("ip", showArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		return status, fmt.Errorf("failed to configure interface: %v", err)
	}

	outputStr := ""
	if strings.Contains(outputStr, "state UP") {
		status.TunnelUp = true
	}

	// Get packet statistics
	statsArgs := []string{"-s", "link", "show", vm.config.DeviceName}
	// Validate all stats arguments
	for _, arg := range statsArgs {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return status, fmt.Errorf("invalid stats command argument %s: %w", arg, err)
		}
	}
	cmd = exec.Command("ip", statsArgs...)
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
			security.SafeLogf(vm.logger, "Warning: skipping invalid remote IP %s: %s", security.SanitizeIPForLog(remoteIP), security.SanitizeErrorForLog(err))
			results[remoteIP] = false
			continue
		}

		security.SafeLogf(vm.logger, "Testing connectivity to %s", security.SanitizeIPForLog(remoteIP))

		// Use ping to test connectivity
		pingArgs := []string{"-c", "3", "-W", "2", remoteIP}
		// Validate all ping arguments
		for _, arg := range pingArgs {
			if err := security.ValidateCommandArgument(arg); err != nil {
				security.SafeLogf(vm.logger, "Warning: skipping ping due to invalid argument %s: %s", security.SanitizeForLog(arg), security.SanitizeErrorForLog(err))
				results[remoteIP] = false
				continue
			}
		}
		cmd := exec.Command("ping", pingArgs...)
		err := cmd.Run()

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
		// Validate all delete fdb arguments
		for _, arg := range delFdbArgs {
			if err := security.ValidateCommandArgument(arg); err != nil {
				continue // Skip invalid arguments
			}
		}
		cmd := exec.Command("bridge", delFdbArgs...)
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
	detailArgs := []string{"-d", "link", "show", vm.config.DeviceName}
	// Validate all detail arguments
	for _, arg := range detailArgs {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return nil, fmt.Errorf("invalid detail command argument %s: %w", arg, err)
		}
	}
	cmd := exec.Command("ip", detailArgs...)
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
	fdbShowArgs := []string{"fdb", "show", "dev", vm.config.DeviceName}
	// Validate all fdb show arguments
	for _, arg := range fdbShowArgs {
		if err := security.ValidateCommandArgument(arg); err != nil {
			return info, fmt.Errorf("invalid fdb show command argument %s: %w", arg, err)
		}
	}
	cmd = exec.Command("bridge", fdbShowArgs...)
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