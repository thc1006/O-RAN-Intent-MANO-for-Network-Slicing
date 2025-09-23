// Copyright 2024 O-RAN Intent MANO Project
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"testing"
)

// TestValidateVXLANIPSecurity tests the enhanced security validation for VXLAN IP addresses
func TestValidateVXLANIPSecurity(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{
			name:    "valid private IP - 10.x.x.x",
			ip:      "10.1.1.1",
			wantErr: false,
		},
		{
			name:    "valid private IP - 172.16.x.x",
			ip:      "172.16.1.1",
			wantErr: false,
		},
		{
			name:    "valid private IP - 192.168.x.x",
			ip:      "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "invalid public IP - Google DNS",
			ip:      "8.8.8.8",
			wantErr: true,
		},
		{
			name:    "invalid public IP - Cloudflare DNS",
			ip:      "1.1.1.1",
			wantErr: true,
		},
		{
			name:    "invalid IP format",
			ip:      "not.an.ip",
			wantErr: true,
		},
		{
			name:    "empty IP",
			ip:      "",
			wantErr: true,
		},
		{
			name:    "malicious injection attempt",
			ip:      "10.1.1.1; rm -rf /",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateVXLANIPSecurity(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVXLANIPSecurity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateIPWithCIDR tests the CIDR validation functionality
func TestValidateIPWithCIDR(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "valid CIDR /24",
			cidr:    "10.1.1.1/24",
			wantErr: false,
		},
		{
			name:    "valid CIDR /16",
			cidr:    "192.168.1.1/16",
			wantErr: false,
		},
		{
			name:    "valid CIDR /8",
			cidr:    "10.0.0.1/8",
			wantErr: false,
		},
		{
			name:    "invalid CIDR - too large subnet",
			cidr:    "10.1.1.1/4",
			wantErr: true,
		},
		{
			name:    "invalid CIDR - too small subnet",
			cidr:    "10.1.1.1/31",
			wantErr: true,
		},
		{
			name:    "invalid CIDR format",
			cidr:    "10.1.1.1/",
			wantErr: true,
		},
		{
			name:    "network address (should fail)",
			cidr:    "10.1.1.0/24",
			wantErr: true,
		},
		{
			name:    "broadcast address (should fail)",
			cidr:    "10.1.1.255/24",
			wantErr: true,
		},
		{
			name:    "empty CIDR",
			cidr:    "",
			wantErr: true,
		},
		{
			name:    "IPv6 not supported",
			cidr:    "2001:db8::1/64",
			wantErr: true,
		},
		{
			name:    "malicious injection in CIDR",
			cidr:    "10.1.1.1/24; cat /etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateIPWithCIDR(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPWithCIDR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGenerateVXLANIPSecurity tests that the IP generation produces secure addresses
func TestGenerateVXLANIPSecurity(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name    string
		vxlanID int32
		nodeIP  string
		wantErr bool
	}{
		{
			name:    "valid inputs",
			vxlanID: 100,
			nodeIP:  "192.168.1.10",
			wantErr: false,
		},
		{
			name:    "edge case - very large VXLAN ID",
			vxlanID: 16777000,
			nodeIP:  "10.0.0.1",
			wantErr: false,
		},
		{
			name:    "malicious node IP",
			vxlanID: 100,
			nodeIP:  "127.0.0.1; rm -rf /",
			wantErr: false, // Should still generate valid IP despite malicious input
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vxlanIP := manager.generateVXLANIP(tt.vxlanID, tt.nodeIP)

			// Generated IP should always be valid for security validation
			if err := manager.validateVXLANIPSecurity(vxlanIP); err != nil {
				if !tt.wantErr {
					t.Errorf("generateVXLANIP() produced insecure IP %s: %v", vxlanIP, err)
				}
			}

			// Generated IP should always be in 10.x.x.x range
			if vxlanIP[0:3] != "10." {
				t.Errorf("generateVXLANIP() should produce 10.x.x.x addresses, got %s", vxlanIP)
			}
		})
	}
}