package vxlan

import (
	"fmt"
	"net"

	tnv1alpha1 "github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/tn/manager/api/v1alpha1"
)

// TunnelConfig represents VXLAN tunnel configuration
type TunnelConfig struct {
	InterfaceName string   `json:"interfaceName"`
	VxlanID      int32    `json:"vxlanId"`
	LocalIP      string   `json:"localIp"`
	RemoteIPs    []string `json:"remoteIps"`
	MTU          int      `json:"mtu"`
	Port         int      `json:"port"`
	Commands     []string `json:"commands"`
}

// Orchestrator manages VXLAN tunnel configurations
type Orchestrator struct {
	DefaultMTU  int
	DefaultPort int
}

// NewOrchestrator creates a new VXLAN orchestrator
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		DefaultMTU:  1450, // Account for VXLAN overhead (50 bytes)
		DefaultPort: 4789, // VXLAN standard port
	}
}

// GenerateTunnelConfigs creates tunnel configurations for the given endpoints
func (o *Orchestrator) GenerateTunnelConfigs(vxlanID int32, endpoints []tnv1alpha1.Endpoint) []TunnelConfig {
	var configs []TunnelConfig

	// Group endpoints by node
	nodeEndpoints := make(map[string][]tnv1alpha1.Endpoint)
	for _, ep := range endpoints {
		nodeEndpoints[ep.NodeName] = append(nodeEndpoints[ep.NodeName], ep)
	}

	// Generate configuration for each node
	for nodeName, nodeEps := range nodeEndpoints {
		if len(nodeEps) == 0 {
			continue
		}

		localEp := nodeEps[0] // Primary endpoint for this node
		config := TunnelConfig{
			InterfaceName: fmt.Sprintf("vxlan%d", vxlanID),
			VxlanID:      vxlanID,
			LocalIP:      localEp.IP,
			RemoteIPs:    []string{},
			MTU:          o.DefaultMTU,
			Port:         o.DefaultPort,
		}

		// Add remote IPs from other nodes
		for otherNode, otherEps := range nodeEndpoints {
			if otherNode != nodeName && len(otherEps) > 0 {
				config.RemoteIPs = append(config.RemoteIPs, otherEps[0].IP)
			}
		}

		// Generate setup commands
		config.Commands = o.generateSetupCommands(config, localEp.Interface)
		configs = append(configs, config)
	}

	return configs
}

func (o *Orchestrator) generateSetupCommands(config TunnelConfig, physInterface string) []string {
	var commands []string

	// Delete existing interface if it exists
	commands = append(commands,
		fmt.Sprintf("ip link del %s 2>/dev/null || true", config.InterfaceName))

	// Create VXLAN interface
	if len(config.RemoteIPs) > 0 {
		// Point-to-multipoint with FDB entries
		commands = append(commands,
			fmt.Sprintf("ip link add %s type vxlan id %d local %s dstport %d dev %s",
				config.InterfaceName, config.VxlanID, config.LocalIP, config.Port, physInterface))

		// Add FDB entries for each remote
		for _, remoteIP := range config.RemoteIPs {
			commands = append(commands,
				fmt.Sprintf("bridge fdb append to 00:00:00:00:00:00 dst %s dev %s",
					remoteIP, config.InterfaceName))
		}
	} else {
		// No remotes yet, create interface without remote
		commands = append(commands,
			fmt.Sprintf("ip link add %s type vxlan id %d local %s dstport %d dev %s nolearning",
				config.InterfaceName, config.VxlanID, config.LocalIP, config.Port, physInterface))
	}

	// Set MTU
	commands = append(commands,
		fmt.Sprintf("ip link set %s mtu %d", config.InterfaceName, config.MTU))

	// Bring interface up
	commands = append(commands,
		fmt.Sprintf("ip link set %s up", config.InterfaceName))

	// Assign IP address to VXLAN interface (using subnet based on VXLAN ID)
	vxlanIP := o.generateVXLANIP(config.VxlanID, config.LocalIP)
	commands = append(commands,
		fmt.Sprintf("ip addr add %s/24 dev %s 2>/dev/null || true", vxlanIP, config.InterfaceName))

	// Enable proxy ARP for better connectivity
	commands = append(commands,
		fmt.Sprintf("echo 1 > /proc/sys/net/ipv4/conf/%s/proxy_arp", config.InterfaceName))

	// Disable rp_filter for VXLAN interface
	commands = append(commands,
		fmt.Sprintf("echo 0 > /proc/sys/net/ipv4/conf/%s/rp_filter", config.InterfaceName))

	return commands
}

// generateVXLANIP generates an IP address for the VXLAN interface
// Uses 10.x.y.z where x.y is derived from VXLAN ID and z from node IP
func (o *Orchestrator) generateVXLANIP(vxlanID int32, nodeIP string) string {
	// Parse node IP to get last octet
	ip := net.ParseIP(nodeIP)
	if ip == nil {
		return fmt.Sprintf("10.%d.0.1", vxlanID%256)
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return fmt.Sprintf("10.%d.0.1", vxlanID%256)
	}

	// Use VXLAN ID for second and third octets, node IP for last
	second := (vxlanID / 256) % 256
	third := vxlanID % 256
	fourth := ipv4[3]

	return fmt.Sprintf("10.%d.%d.%d", second, third, fourth)
}

// GenerateCleanupCommands creates commands to remove VXLAN configuration
func (o *Orchestrator) GenerateCleanupCommands(vxlanID int32) []string {
	interfaceName := fmt.Sprintf("vxlan%d", vxlanID)
	return []string{
		fmt.Sprintf("ip link del %s 2>/dev/null || true", interfaceName),
	}
}

// ValidateEndpoints checks if endpoints are valid for VXLAN configuration
func (o *Orchestrator) ValidateEndpoints(endpoints []tnv1alpha1.Endpoint) error {
	if len(endpoints) < 2 {
		return fmt.Errorf("at least 2 endpoints required for VXLAN tunnel")
	}

	// Check for valid IPs
	for _, ep := range endpoints {
		if net.ParseIP(ep.IP) == nil {
			return fmt.Errorf("invalid IP address: %s", ep.IP)
		}

		if ep.Interface == "" {
			return fmt.Errorf("interface not specified for endpoint %s", ep.NodeName)
		}
	}

	// Check for duplicate IPs
	seen := make(map[string]bool)
	for _, ep := range endpoints {
		if seen[ep.IP] {
			return fmt.Errorf("duplicate IP address: %s", ep.IP)
		}
		seen[ep.IP] = true
	}

	return nil
}