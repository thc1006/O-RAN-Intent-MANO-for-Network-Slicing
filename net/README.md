# Multi-Cluster Network Configuration

This directory contains configurations and scripts for multi-cluster Kube-OVN connectivity with controlled inter-site latency emulation.

## Directory Structure

```
net/
├── ovn/
│   └── multinic.md          # Comprehensive Kube-OVN multi-NIC configuration guide
├── tests/
│   └── latency.sh           # Automated latency testing script
└── config/
    ├── setup-vxlan.sh       # VXLAN tunnel setup script
    ├── configure-delays.sh  # TC delay configuration script
    └── test-pods.yaml       # Kubernetes test pod deployments
```

## Network Topology

```
                    ┌────────────┐
                    │  Central   │
                    │  (0ms)     │
                    └─────┬──────┘
                      7ms │ 7ms
                ┌─────────┼─────────┐
                ▼         │         ▼
          ┌──────────┐    │   ┌──────────┐
          │ Regional │    │   │  Edge-01 │
          │  (0ms)   │    │   │  (0ms)   │
          └─────┬────┘    │   └──────────┘
              5ms │       │        5ms
                  └───────┼────────┘
                          ▼
                    ┌──────────┐
                    │  Edge-02 │
                    │  (0ms)   │
                    └──────────┘
```

## Quick Start

### 1. Prerequisites

- Kubernetes clusters (1.24+) at each site
- Kube-OVN v1.12.0+ installed
- Root access for TC configuration
- Network connectivity between cluster nodes

### 2. Setup Process

```bash
# On each cluster node (as root)

# 1. Set cluster identity
export CLUSTER_NAME=central  # or regional, edge01, edge02
export LOCAL_IP=$(hostname -I | awk '{print $1}')

# 2. Configure cluster endpoints
export CENTRAL_IP=10.100.0.10
export REGIONAL_IP=10.100.1.10
export EDGE01_IP=10.100.2.10
export EDGE02_IP=10.100.3.10

# 3. Setup VXLAN tunnels
./net/config/setup-vxlan.sh setup

# 4. Configure TC delays
./net/config/configure-delays.sh configure

# 5. Verify configuration
./net/config/setup-vxlan.sh status
./net/config/configure-delays.sh status
```

### 3. Deploy Test Pods

```bash
# Deploy test infrastructure
kubectl apply -f net/config/test-pods.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app=latency-test -n network-testing --timeout=60s
```

### 4. Run Tests

```bash
# Run comprehensive latency tests
./net/tests/latency.sh --namespace network-testing

# Check specific link
kubectl exec -it test-pod-central -n network-testing -- ping -c 10 10.1.1.10
```

## Configuration Details

### VXLAN Configuration

| Link | VXLAN ID | Delay | Purpose |
|------|----------|-------|---------|
| Central ↔ Regional | 101 | 7ms | Core to aggregation |
| Central ↔ Edge-01 | 102 | 7ms | Core to edge |
| Central ↔ Edge-02 | 103 | 7ms | Core to edge |
| Regional ↔ Edge-01 | 104 | 5ms | Aggregation to edge |
| Regional ↔ Edge-02 | 105 | 5ms | Aggregation to edge |
| Edge-01 ↔ Edge-02 | 106 | 5ms | Edge to edge |

### Network Subnets

| Cluster | Pod CIDR | Service CIDR | Data Subnet | Mgmt Subnet |
|---------|----------|--------------|-------------|-------------|
| Central | 10.0.0.0/16 | 10.96.0.0/16 | 172.16.0.0/24 | 192.168.0.0/24 |
| Regional | 10.1.0.0/16 | 10.97.0.0/16 | 172.16.1.0/24 | 192.168.1.0/24 |
| Edge-01 | 10.2.0.0/16 | 10.98.0.0/16 | 172.16.2.0/24 | 192.168.2.0/24 |
| Edge-02 | 10.3.0.0/16 | 10.99.0.0/16 | 172.16.3.0/24 | 192.168.3.0/24 |

## Testing and Validation

### Expected Results

The latency test script validates:

1. **Pod-to-Pod RTT**: Should match configured delays (±1ms tolerance)
   - Central ↔ Regional: 7ms
   - Central ↔ Edge: 7ms
   - Regional ↔ Edge: 5ms
   - Edge ↔ Edge: 5ms
   - Same cluster: <1ms

2. **Service Connectivity**: All cross-cluster services should be reachable

3. **Packet Loss**: Should be minimal (<0.1%)

### Test Output

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "total_tests": 12,
  "passed_tests": 12,
  "failed_tests": 0,
  "success_rate": "100.00%",
  "test_results": {
    "central_to_regional": {
      "avg_rtt": "7.2ms",
      "status": "PASSED"
    },
    "regional_to_edge01": {
      "avg_rtt": "5.1ms",
      "status": "PASSED"
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **VXLAN tunnel not established**
   ```bash
   # Check interface exists
   ip link show | grep vxlan

   # Check OVS configuration
   ovs-vsctl show
   ```

2. **TC delays not applied**
   ```bash
   # Check qdisc configuration
   tc qdisc show dev vxlan-regional

   # Monitor statistics
   tc -s qdisc show dev vxlan-regional
   ```

3. **High packet loss**
   ```bash
   # Check MTU settings
   ip link show vxlan-regional

   # Adjust MTU if needed
   ip link set dev vxlan-regional mtu 1400
   ```

### Debug Commands

```bash
# Check OVS flows
ovs-ofctl dump-flows br-int

# Monitor VXLAN traffic
tcpdump -i vxlan-regional -n

# Test specific delay
./net/config/configure-delays.sh test 10.1.1.10 7

# Clear all configurations
./net/config/setup-vxlan.sh cleanup
./net/config/configure-delays.sh cleanup
```

## Performance Tuning

### Network Optimization

```bash
# Increase network buffers
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728

# CPU affinity for OVS
ovs-vsctl set Open_vSwitch . other_config:pmd-cpu-mask=0x6

# Disable offloading for VXLAN
ethtool -K eth0 gro off
ethtool -K eth0 tso off
```

### Advanced TC Configuration

```bash
# Add bandwidth limit with delay
./net/config/configure-delays.sh advanced vxlan-regional 7 1000

# Monitor in real-time
./net/config/configure-delays.sh monitor vxlan-regional
```

## Integration with O-RAN MANO

This network configuration supports the O-RAN Intent-based MANO framework by:

1. **Realistic Latency**: Emulates real-world inter-site delays
2. **Multi-NIC Support**: Separates data and management traffic
3. **QoS Validation**: Ensures SLA compliance for network slices
4. **Scalability Testing**: Validates multi-cluster deployments

## References

- [Kube-OVN Documentation](https://github.com/kubeovn/kube-ovn)
- [Linux TC Manual](https://man7.org/linux/man-pages/man8/tc.8.html)
- [VXLAN RFC 7348](https://tools.ietf.org/html/rfc7348)
- [O-RAN Architecture](https://www.o-ran.org/specifications)