# Transport Network (TN) Agent for O-RAN Intent-MANO

## Overview

The Transport Network (TN) Agent provides comprehensive bandwidth shaping, VXLAN network management, and performance testing capabilities for the O-RAN Intent-MANO system. It implements thesis validation targets for throughput ({0.93, 2.77, 4.57} Mbps), RTT ({6.3, 15.7, 16.1} ms), and deployment time (<10 minutes).

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TN Manager    │────│   TN Agent 1    │────│   TN Agent 2    │
│   (Central)     │    │  (Edge/Regional)│    │  (Edge/Regional)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
        │                       │                       │
        │              ┌─────────────────┐              │
        └──────────────│   Performance   │──────────────┘
                       │    Testing &    │
                       │   Validation    │
                       └─────────────────┘
```

### Components

1. **TN Manager**: Centralized orchestrator for multi-cluster network management
2. **TN Agent**: Distributed agents for VXLAN, TC, and iperf3 management
3. **VXLAN Manager**: Handles tunnel creation and connectivity
4. **TC Manager**: Implements traffic shaping and QoS policies
5. **Iperf Manager**: Provides performance testing capabilities
6. **Bandwidth Monitor**: Real-time network performance monitoring

## Features

### Core Capabilities
- **VXLAN Tunnel Management**: Multi-site connectivity with automatic peer discovery
- **Traffic Shaping**: HTB-based bandwidth control with QoS classification
- **Performance Testing**: Integrated iperf3 for throughput and latency measurement
- **Real-time Monitoring**: Continuous bandwidth and performance metrics
- **Thesis Validation**: Automated validation against research targets

### Network Slice Support
- **eMBB (Enhanced Mobile Broadband)**: High throughput slices
- **URLLC (Ultra-Reliable Low Latency)**: Low latency communication
- **mMTC (Massive Machine Type Communications)**: Efficient IoT connectivity

### Performance Targets
- **Throughput**: 0.93, 2.77, 4.57 Mbps (thesis targets)
- **RTT**: 6.3, 15.7, 16.1 ms (ping latency targets)
- **Deployment**: <10 minutes end-to-end deployment time

## Quick Start

### Prerequisites

```bash
# Install dependencies
sudo apt-get update
sudo apt-get install -y iperf3 iproute2 bridge-utils

# Load kernel modules
sudo modprobe vxlan sch_htb sch_netem
```

### Building

```bash
# Build all components
make build

# Cross-compile for all platforms
make build-all

# Build with development environment
make dev
```

### Configuration

```bash
# Generate default configurations
make config

# Edit configurations
vim config/manager.yaml  # Manager configuration
vim config/agent.yaml    # Agent configuration
```

### Running

```bash
# Start TN Manager
./build/tn-manager -config config/manager.yaml

# Start TN Agent
./build/tn-agent -config config/agent.yaml

# Start demo environment
make demo
```

## Configuration

### Manager Configuration (config/manager.yaml)

```yaml
manager:
  clusterName: "tn-manager"
  networkCIDR: "10.244.0.0/16"
  monitoringPort: 9090

agents:
  - name: "edge-cluster-01"
    endpoint: "http://192.168.100.10:8080"
    enabled: true
  - name: "edge-cluster-02"
    endpoint: "http://192.168.100.11:8081"
    enabled: true

monitoring:
  metricsInterval: 30s
  retentionPeriod: 24h
  enableContinuous: true
```

### Agent Configuration (config/agent.yaml)

```yaml
agent:
  clusterName: "edge-cluster-01"
  monitoringPort: 8080

vxlan:
  vni: 1001
  localIP: "192.168.100.10"
  remoteIPs:
    - "192.168.100.11"
    - "192.168.100.12"
  port: 4789
  mtu: 1450

bandwidth:
  downlinkMbps: 10.0
  uplinkMbps: 10.0
  latencyMs: 10.0
  jitterMs: 2.0
  lossPercent: 0.1
```

## Testing

### Unit Tests

```bash
# Run unit tests
make test-unit

# Generate coverage report
make coverage
```

### Integration Tests

```bash
# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e
```

### Thesis Validation

```bash
# Complete thesis validation workflow
make thesis-validation

# Run thesis validation tests only
make test-thesis

# Collect and analyze metrics
make collect-thesis-metrics
```

### Benchmarking

```bash
# Run performance benchmarks
make bench
```

## API Reference

### TN Manager API

#### Register Agent
```http
POST /agents
Content-Type: application/json

{
  "name": "edge-cluster-01",
  "endpoint": "http://192.168.100.10:8080"
}
```

#### Run Performance Test
```http
POST /tests
Content-Type: application/json

{
  "testId": "thesis-validation-1",
  "sliceType": "eMBB",
  "duration": "60s",
  "protocol": "tcp"
}
```

### TN Agent API

#### Get Status
```http
GET /status
```

#### Configure Slice
```http
POST /slices/{sliceId}
Content-Type: application/json

{
  "qosClass": "high_throughput",
  "bandwidthPolicy": {
    "downlinkMbps": 5.0,
    "uplinkMbps": 5.0
  }
}
```

#### VXLAN Management
```http
GET /vxlan/status
PUT /vxlan/peers
POST /vxlan/connectivity
```

#### Traffic Control
```http
GET /tc/status
POST /tc/rules
DELETE /tc/rules
```

#### Bandwidth Monitoring
```http
GET /bandwidth
GET /bandwidth/stream  # Server-Sent Events
```

## Performance Metrics

### Throughput Metrics
- **Downlink/Uplink Mbps**: Measured throughput in each direction
- **Peak/Average/Min**: Statistical analysis of throughput performance
- **Bidirectional**: Simultaneous upload/download testing

### Latency Metrics
- **RTT (Round Trip Time)**: Ping-based latency measurement
- **Percentiles**: P50, P95, P99 latency distribution
- **Jitter**: Latency variation analysis

### Overhead Analysis
- **VXLAN Overhead**: Encapsulation impact on performance
- **TC Overhead**: Traffic control processing impact
- **Bandwidth Utilization**: Network efficiency metrics

## Network Slice Templates

### eMBB (Enhanced Mobile Broadband)
```yaml
sliceType: "eMBB"
qosClass: "high_throughput"
bandwidth:
  downlinkMbps: 5.0
  uplinkMbps: 2.0
  priority: 2
expectedPerformance:
  throughput: 4.57  # Mbps
  rtt: 16.1         # ms
```

### URLLC (Ultra-Reliable Low Latency)
```yaml
sliceType: "URLLC"
qosClass: "ultra_low_latency"
bandwidth:
  downlinkMbps: 3.0
  uplinkMbps: 3.0
  latencyMs: 5.0
  priority: 1
expectedPerformance:
  throughput: 2.77  # Mbps
  rtt: 15.7         # ms
```

### mMTC (Massive Machine Type Communications)
```yaml
sliceType: "mMTC"
qosClass: "efficient"
bandwidth:
  downlinkMbps: 1.0
  uplinkMbps: 0.5
  priority: 3
expectedPerformance:
  throughput: 0.93  # Mbps
  rtt: 6.3          # ms
```

## Deployment Scenarios

### Multi-Cluster Setup

```bash
# Edge cluster 1
./build/tn-agent -local-ip 192.168.100.10 -port 8080

# Edge cluster 2
./build/tn-agent -local-ip 192.168.100.11 -port 8081

# Regional cluster
./build/tn-agent -local-ip 192.168.100.12 -port 8082

# Central manager
./build/tn-manager -config config/manager.yaml
```

### Docker Deployment

```bash
# Build Docker images
make docker

# Run with Docker Compose
docker-compose up -d
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tn-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tn-agent
  template:
    metadata:
      labels:
        app: tn-agent
    spec:
      hostNetwork: true
      containers:
      - name: tn-agent
        image: tn-agent:latest
        securityContext:
          privileged: true
        volumeMounts:
        - name: config
          mountPath: /etc/tn
```

## Monitoring and Observability

### Metrics Collection
- **Real-time bandwidth monitoring** with configurable intervals
- **Performance metrics export** in JSON format
- **Continuous health monitoring** with alerting
- **Historical data retention** with configurable policies

### Logging
- **Structured logging** with configurable levels
- **Log rotation** with size and age-based policies
- **Centralized log aggregation** support

### Dashboards
- **Grafana integration** for visualization
- **Prometheus metrics** export
- **Real-time performance dashboards**

## Troubleshooting

### Common Issues

#### VXLAN Tunnel Not Connecting
```bash
# Check interface status
ip link show vxlan1001

# Verify remote peer connectivity
ping 192.168.100.11

# Check FDB entries
bridge fdb show dev vxlan1001
```

#### Traffic Shaping Not Working
```bash
# Check TC rules
tc qdisc show dev vxlan1001
tc class show dev vxlan1001

# Verify bandwidth usage
cat /proc/net/dev | grep vxlan1001
```

#### Performance Test Failures
```bash
# Check iperf3 server status
./build/tn-agent -config config/agent.yaml

# Test direct connectivity
iperf3 -c 192.168.100.11 -t 10

# Verify bandwidth allocation
curl http://localhost:8080/bandwidth
```

### Debug Mode

```bash
# Enable debug logging
./build/tn-agent -log-level debug

# Export detailed metrics
curl http://localhost:8080/metrics/export
```

## Development

### Building from Source

```bash
# Setup development environment
make dev

# Run tests
make test

# Generate configurations and scripts
make config scripts

# Lint and format code
make lint fmt
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `make test lint`
5. Submit a pull request

### Code Structure

```
tn/
├── agent/               # TN Agent implementation
│   ├── pkg/            # Core agent packages
│   │   ├── agent.go    # Main agent logic
│   │   ├── vxlan.go    # VXLAN management
│   │   ├── tc.go       # Traffic control
│   │   ├── iperf.go    # Performance testing
│   │   ├── monitor.go  # Bandwidth monitoring
│   │   └── http.go     # HTTP API handlers
│   └── cmd/            # Agent binary
├── manager/            # TN Manager implementation
│   ├── pkg/            # Core manager packages
│   │   ├── manager.go  # Main manager logic
│   │   ├── metrics.go  # Metrics collection
│   │   └── types.go    # Common types
│   └── cmd/            # Manager binary
├── tests/              # Test suites
│   ├── unit/           # Unit tests
│   ├── integration/    # Integration tests
│   └── e2e/            # End-to-end tests
└── Makefile           # Build automation
```

## Performance Optimization

### Network Tuning

```bash
# Optimize network stack
echo 'net.core.rmem_max = 268435456' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 268435456' >> /etc/sysctl.conf
sysctl -p

# Tune interface queues
ethtool -G eth0 rx 4096 tx 4096
```

### VXLAN Optimization

```bash
# Optimize VXLAN settings
ip link set vxlan1001 mtu 1450
ip link set vxlan1001 txqueuelen 1000
```

### TC Optimization

```bash
# Use appropriate queue disciplines
tc qdisc add dev vxlan1001 root handle 1: htb default 30
tc class add dev vxlan1001 parent 1: classid 1:1 htb rate 10mbit
```

## Security Considerations

### Network Security
- **IPSec encryption** for VXLAN tunnels (optional)
- **Access control lists** for API endpoints
- **Certificate-based authentication** between components

### Operational Security
- **Non-root execution** where possible
- **Capability-based permissions** for network operations
- **Secure configuration management**

## License

This project is part of the O-RAN Intent-MANO for Network Slicing research.

## Support

For issues and questions:
- Review the troubleshooting section
- Check existing issues in the repository
- Enable debug logging for detailed diagnostics

## Changelog

### v1.0.0
- Initial implementation
- VXLAN tunnel management
- Traffic control with HTB
- iperf3 integration
- Real-time bandwidth monitoring
- Thesis validation framework
- Multi-cluster support
- Comprehensive test suite