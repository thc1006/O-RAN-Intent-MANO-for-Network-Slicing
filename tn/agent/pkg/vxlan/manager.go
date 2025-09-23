package vxlan

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/pkg/security"
)

// Manager manages VXLAN tunnel interfaces
// Security Note: All command execution uses security.SecureExecute* functions
// which validate inputs and prevent command injection attacks.
type Manager struct {
	// Map of VXLAN ID to interface configuration
	tunnels map[int32]*TunnelInfo
}

// TunnelInfo stores information about a VXLAN tunnel
type TunnelInfo struct {
	InterfaceName string
	VxlanID      int32
	LocalIP      string
	RemoteIPs    []string
	MTU          int
}

// NewManager creates a new VXLAN manager
func NewManager() *Manager {
	return &Manager{
		tunnels: make(map[int32]*TunnelInfo),
	}
}

// CreateTunnel creates a new VXLAN tunnel interface
func (m *Manager) CreateTunnel(vxlanID int32, localIP string, remoteIPs []string, physInterface string) error {
	ifaceName := fmt.Sprintf("vxlan%d", vxlanID)

	// Check if tunnel already exists
	if _, exists := m.tunnels[vxlanID]; exists {
		// Delete existing tunnel
		if err := m.DeleteTunnel(vxlanID); err != nil {
			return fmt.Errorf("failed to delete existing tunnel: %w", err)
		}
	}

	// Create VXLAN interface using secure execution
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ipArgs := []string{"link", "add", ifaceName,
		"type", "vxlan",
		"id", fmt.Sprintf("%d", vxlanID),
		"local", localIP,
		"dstport", "4789",
		"dev", physInterface}

	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, ipArgs...); err != nil {
		return fmt.Errorf("failed to create vxlan interface: %v", err)
	}

	// Set MTU (accounting for VXLAN overhead)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mtuArgs := []string{"link", "set", ifaceName, "mtu", "1450"}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, mtuArgs...); err != nil {
		return fmt.Errorf("failed to set MTU: %v", err)
	}

	// Bring interface up
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	upArgs := []string{"link", "set", ifaceName, "up"}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, upArgs...); err != nil {
		return fmt.Errorf("failed to bring interface up: %v", err)
	}

	// Add FDB entries for each remote IP
	for _, remoteIP := range remoteIPs {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		fdbArgs := []string{"fdb", "append",
			"00:00:00:00:00:00",
			"dst", remoteIP,
			"dev", ifaceName}

		// #nosec G204 - Using security.SecureExecute with validated arguments to prevent command injection
		if _, err := security.SecureExecute(ctx, "bridge", fdbArgs...); err != nil {
			// Log but don't fail on FDB errors
			fmt.Printf("Warning: failed to add FDB entry for %s: %v\n",
				security.SanitizeForLog(remoteIP), err)
		}
		cancel()
	}

	// Assign IP address to VXLAN interface
	vxlanIP := m.generateVXLANIP(vxlanID, localIP)

	// Security validation: Validate generated VXLAN IP before use
	if err := security.ValidateIPAddress(vxlanIP); err != nil {
		return fmt.Errorf("invalid generated VXLAN IP: %w", err)
	}

	// Additional security: Ensure IP is in private range for VXLAN
	if err := m.validateVXLANIPSecurity(vxlanIP); err != nil {
		return fmt.Errorf("VXLAN IP security validation failed: %w", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pre-validate IP address format before constructing command arguments
	vxlanCIDR := fmt.Sprintf("%s/24", vxlanIP)
	if err := m.validateIPWithCIDR(vxlanCIDR); err != nil {
		return fmt.Errorf("invalid VXLAN CIDR format: %w", err)
	}

	// Security validation: Validate interface name before use
	if err := security.ValidateNetworkInterface(ifaceName); err != nil {
		return fmt.Errorf("invalid interface name: %w", err)
	}

	// Construct arguments with validated inputs only
	addrArgs := []string{"addr", "add", vxlanCIDR, "dev", ifaceName}

	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, addrArgs...); err != nil {
		// Ignore if address already exists
		if !strings.Contains(err.Error(), "exists") {
			return fmt.Errorf("failed to assign IP: %v", err)
		}
	}

	// Store tunnel information
	m.tunnels[vxlanID] = &TunnelInfo{
		InterfaceName: ifaceName,
		VxlanID:      vxlanID,
		LocalIP:      localIP,
		RemoteIPs:    remoteIPs,
		MTU:          1450,
	}

	return nil
}

// DeleteTunnel removes a VXLAN tunnel interface
func (m *Manager) DeleteTunnel(vxlanID int32) error {
	ifaceName := fmt.Sprintf("vxlan%d", vxlanID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	delArgs := []string{"link", "del", ifaceName}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	if _, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, delArgs...); err != nil {
		// Ignore if interface doesn't exist
		if !strings.Contains(err.Error(), "Cannot find device") && !strings.Contains(err.Error(), "does not exist") {
			return fmt.Errorf("failed to delete interface: %v", err)
		}
	}

	delete(m.tunnels, vxlanID)
	return nil
}

// GetTunnelStatus retrieves the status of a tunnel
func (m *Manager) GetTunnelStatus(vxlanID int32) (*TunnelInfo, error) {
	info, exists := m.tunnels[vxlanID]
	if !exists {
		return nil, fmt.Errorf("tunnel %d not found", vxlanID)
	}

	// Check if interface is up
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	showArgs := []string{"link", "show", info.InterfaceName}
	// #nosec G204 - Using security.SecureExecuteWithValidation with argument validation to prevent command injection
	output, err := security.SecureExecuteWithValidation(ctx, "ip", security.ValidateIPArgs, showArgs...)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %v", info.InterfaceName, err)
	}

	// Parse output to check state
	if strings.Contains(string(output), "state UP") {
		return info, nil
	}

	return info, fmt.Errorf("interface %s is down", info.InterfaceName)
}

// generateVXLANIP generates an IP address for the VXLAN interface
func (m *Manager) generateVXLANIP(vxlanID int32, nodeIP string) string {
	// Simple IP generation based on VXLAN ID
	// Uses 10.x.y.z where x.y is derived from VXLAN ID
	second := (vxlanID / 256) % 256
	third := vxlanID % 256

	// Extract last octet from node IP
	parts := strings.Split(nodeIP, ".")
	fourth := "1"
	if len(parts) == 4 {
		fourth = parts[3]
	}

	return fmt.Sprintf("10.%d.%d.%s", second, third, fourth)
}

// validateVXLANIPSecurity performs additional security validation for VXLAN IP addresses
func (m *Manager) validateVXLANIPSecurity(ip string) error {
	// Parse IP to ensure it's valid
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Ensure VXLAN IPs are in private address ranges for security
	privateRanges := []string{
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
	}

	for _, cidr := range privateRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(parsedIP) {
			return nil // Valid private IP
		}
	}

	return fmt.Errorf("VXLAN IP must be in private address range: %s", ip)
}

// validateIPWithCIDR validates IP address with CIDR notation
func (m *Manager) validateIPWithCIDR(cidr string) error {
	if cidr == "" {
		return fmt.Errorf("CIDR cannot be empty")
	}

	// Parse CIDR to ensure it's valid
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %s (%w)", cidr, err)
	}

	// Ensure IP is valid IPv4
	if ip.To4() == nil {
		return fmt.Errorf("only IPv4 addresses supported for VXLAN: %s", cidr)
	}

	// Validate subnet mask is appropriate for VXLAN (typically /24)
	ones, bits := network.Mask.Size()
	if bits != 32 { // IPv4
		return fmt.Errorf("invalid address family for VXLAN: %s", cidr)
	}
	if ones < 8 || ones > 30 {
		return fmt.Errorf("invalid subnet mask for VXLAN: /%d (must be /8 to /30)", ones)
	}

	// Additional security: prevent broadcast and network addresses
	if ip.Equal(network.IP) {
		return fmt.Errorf("cannot use network address: %s", cidr)
	}

	broadcast := make(net.IP, len(network.IP))
	copy(broadcast, network.IP)
	for i := range broadcast {
		broadcast[i] |= ^network.Mask[i]
	}
	if ip.Equal(broadcast) {
		return fmt.Errorf("cannot use broadcast address: %s", cidr)
	}

	return nil
}

// Cleanup removes all managed VXLAN tunnels
func (m *Manager) Cleanup() error {
	for vxlanID := range m.tunnels {
		if err := m.DeleteTunnel(vxlanID); err != nil {
			return err
		}
	}
	return nil
}