package vxlan

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager manages VXLAN tunnel interfaces
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

	// Create VXLAN interface
	cmd := exec.Command("ip", "link", "add", ifaceName,
		"type", "vxlan",
		"id", fmt.Sprintf("%d", vxlanID),
		"local", localIP,
		"dstport", "4789",
		"dev", physInterface)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create vxlan interface: %v, output: %s", err, output)
	}

	// Set MTU (accounting for VXLAN overhead)
	cmd = exec.Command("ip", "link", "set", ifaceName, "mtu", "1450")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set MTU: %v, output: %s", err, output)
	}

	// Bring interface up
	cmd = exec.Command("ip", "link", "set", ifaceName, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bring interface up: %v, output: %s", err, output)
	}

	// Add FDB entries for each remote IP
	for _, remoteIP := range remoteIPs {
		cmd = exec.Command("bridge", "fdb", "append",
			"00:00:00:00:00:00",
			"dst", remoteIP,
			"dev", ifaceName)

		if output, err := cmd.CombinedOutput(); err != nil {
			// Log but don't fail on FDB errors
			fmt.Printf("Warning: failed to add FDB entry for %s: %v, output: %s\n",
				remoteIP, err, output)
		}
	}

	// Assign IP address to VXLAN interface
	vxlanIP := m.generateVXLANIP(vxlanID, localIP)
	cmd = exec.Command("ip", "addr", "add", fmt.Sprintf("%s/24", vxlanIP), "dev", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore if address already exists
		if !strings.Contains(string(output), "File exists") {
			return fmt.Errorf("failed to assign IP: %v, output: %s", err, output)
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

	cmd := exec.Command("ip", "link", "del", ifaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Ignore if interface doesn't exist
		if !strings.Contains(string(output), "Cannot find device") {
			return fmt.Errorf("failed to delete interface: %v, output: %s", err, output)
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
	cmd := exec.Command("ip", "link", "show", info.InterfaceName)
	output, err := cmd.CombinedOutput()
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

// Cleanup removes all managed VXLAN tunnels
func (m *Manager) Cleanup() error {
	for vxlanID := range m.tunnels {
		if err := m.DeleteTunnel(vxlanID); err != nil {
			return err
		}
	}
	return nil
}