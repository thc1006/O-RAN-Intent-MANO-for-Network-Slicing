# Kube-OVN Multi-Site Connectivity Configuration

This directory contains the complete Kube-OVN configuration for multi-site O-RAN network slicing deployment. The configuration enables GENEVE tunnels between sites, QoS-aware network policies, and comprehensive monitoring.

## Overview

The configuration supports a hub-and-spoke topology:
- **Central Site**: Acts as the OVN control plane and hub
- **Edge Sites**: Connect to central via GENEVE tunnels
- **QoS Classes**: Three priority levels (high/medium/low) with bandwidth guarantees
- **Network Isolation**: Per-slice network policies with controlled inter-slice communication

## Files Description

### Core Configuration Files

- **`ovn-central.yaml`**: Central cluster OVN configuration with control plane components
- **`ovn-edge.yaml`**: Edge site configuration connecting to central OVN databases
- **`interconnect.yaml`**: Multi-site interconnection with GENEVE tunnel setup
- **`kube-ovn-deployment.yaml`**: Complete OVN-Kubernetes deployment manifests
- **`gateway-config.yaml`**: Inter-cluster gateway configuration for cross-site connectivity

### Network Policies and QoS

- **`network-policies.yaml`**: QoS-aware network policies with three priority classes:
  - High Priority: 100Mbps bandwidth, 10ms latency target, DSCP 46
  - Medium Priority: 50Mbps bandwidth, 20ms latency target, DSCP 26
  - Low Priority: 10Mbps bandwidth, 100ms latency target, DSCP 10

### Monitoring and Topology

- **`topology-mapping.yaml`**: Network topology definitions and service discovery
- **`health-monitoring.yaml`**: Comprehensive health monitoring with Prometheus integration
- **`tests/connectivity_test.sh`**: Multi-site connectivity test suite

## Expected Performance Metrics

Based on the thesis requirements, the configuration targets:

| QoS Class | DL Throughput | RTT | Use Case |
|-----------|---------------|-----|----------|
| High      | 4.57 Mbps    | 16.1 ms | eMBB (Enhanced Mobile Broadband) |
| Medium    | 2.77 Mbps    | 15.7 ms | URLLC (Ultra-Reliable Low-Latency) |
| Low       | 0.93 Mbps    | 6.3 ms  | mMTC (Massive Machine-Type Communications) |

## Deployment Instructions

### 1. Central Cluster Setup

```bash
# Deploy OVN central components
kubectl apply -f ovn-central.yaml

# Deploy complete OVN-Kubernetes stack
kubectl apply -f kube-ovn-deployment.yaml

# Configure network policies
kubectl apply -f network-policies.yaml

# Setup interconnection
kubectl apply -f interconnect.yaml
```

### 2. Edge Cluster Setup

```bash
# Set environment variables for central cluster connection
export CENTRAL_OVN_NB_HOSTS="central-cluster-ip:30641"
export CENTRAL_OVN_SB_HOSTS="central-cluster-ip:30642"
export EDGE_SITE_NAME="edge01"  # or edge02
export EDGE_GATEWAY_NODES="edge01-gw-01"

# Deploy edge configuration
envsubst < ovn-edge.yaml | kubectl apply -f -

# Deploy gateway configuration
kubectl apply -f gateway-config.yaml
```

### 3. Monitoring Setup

```bash
# Deploy monitoring stack
kubectl apply -f health-monitoring.yaml

# Deploy topology discovery
kubectl apply -f topology-mapping.yaml
```

## Testing

### Run Connectivity Tests

```bash
# Set kubeconfig paths for all clusters
export CENTRAL_KUBECONFIG="~/.kube/central-config"
export EDGE01_KUBECONFIG="~/.kube/edge01-config"
export EDGE02_KUBECONFIG="~/.kube/edge02-config"

# Run comprehensive connectivity tests
./tests/connectivity_test.sh
```

### Test Results

The test suite validates:
- OVN central and edge connectivity
- GENEVE tunnel establishment
- QoS subnet configuration
- Network policy enforcement
- Cross-cluster pod connectivity
- Bandwidth and latency measurements
- Network slice isolation

## QoS Configuration

### Pod Annotations for QoS Classes

**High Priority Pods:**
```yaml
metadata:
  annotations:
    ovn.kubernetes.io/logical_switch: "high-priority-slice"
    ovn.kubernetes.io/ingress_rate: "100"
    ovn.kubernetes.io/egress_rate: "100"
  labels:
    qos-class: "high"
    network-slice: "embb"
```

**Medium Priority Pods:**
```yaml
metadata:
  annotations:
    ovn.kubernetes.io/logical_switch: "medium-priority-slice"
    ovn.kubernetes.io/ingress_rate: "50"
    ovn.kubernetes.io/egress_rate: "50"
  labels:
    qos-class: "medium"
    network-slice: "urllc"
```

**Low Priority Pods:**
```yaml
metadata:
  annotations:
    ovn.kubernetes.io/logical_switch: "low-priority-slice"
    ovn.kubernetes.io/ingress_rate: "10"
    ovn.kubernetes.io/egress_rate: "10"
  labels:
    qos-class: "low"
    network-slice: "mmtc"
```

## Network Architecture

### Subnet Allocation

- **Central Cluster**: 10.16.0.0/12
  - High Priority: 10.16.10.0/24
  - Medium Priority: 10.16.20.0/24
  - Low Priority: 10.16.30.0/24

- **Edge01 Cluster**: 10.244.0.0/16
- **Edge02 Cluster**: 10.32.0.0/16
- **Transit Network**: 172.20.0.0/16

### Gateway Configuration

- **Central Gateway**: 192.168.1.100
- **Edge01 Gateway**: 192.168.2.100
- **Edge02 Gateway**: 192.168.3.100

## Monitoring Endpoints

- **OVN Health Monitor**: :8080/metrics
- **OVN Exporter**: :9310/metrics
- **Performance Monitor**: :8081/metrics
- **Gateway Metrics**: :10662/metrics

## Troubleshooting

### Common Issues

1. **Tunnel Connectivity Issues**
   ```bash
   # Check tunnel interfaces
   ovs-vsctl show | grep genev

   # Verify tunnel IPs
   kubectl get configmap ovn-interconnect-config -n kube-ovn -o yaml
   ```

2. **QoS Policy Not Applied**
   ```bash
   # Check subnet configuration
   kubectl get subnet -n kube-ovn

   # Verify QoS policies
   kubectl get qospolicy -n kube-ovn
   ```

3. **Cross-Cluster Connectivity Failed**
   ```bash
   # Check gateway status
   kubectl get pods -n kube-ovn -l app=ovn-gateway

   # Verify routing
   kubectl exec -n kube-ovn ds/ovs-ovn -- ip route show
   ```

### Log Locations

- **OVN Logs**: `/var/log/ovn/`
- **OVS Logs**: `/var/log/openvswitch/`
- **Kube-OVN Logs**: `/var/log/kube-ovn/`

## Integration with O-RAN Components

This configuration supports O-RAN network functions deployment:

- **O-RAN Central-DU**: Deployed on central cluster with high QoS
- **O-RAN Distributed-DU**: Deployed on edge sites with medium QoS
- **O-RAN Radio Units**: Connected via fronthaul interface
- **Near-RT RIC**: Central deployment with low-latency connectivity
- **SMO**: Management and orchestration across all sites

## Performance Optimization

### Bandwidth Allocation

The configuration implements hierarchical QoS with guaranteed minimums and burst capabilities:

- **High Priority**: 100Mbps guaranteed, 125Mbps burst
- **Medium Priority**: 50Mbps guaranteed, 62.5Mbps burst
- **Low Priority**: 10Mbps guaranteed, 12.5Mbps burst

### Latency Optimization

- GENEVE tunnel optimization for minimal overhead
- DSCP marking for traffic prioritization
- Direct routing paths for critical flows
- Optimized MTU settings (1400 bytes) to avoid fragmentation

This configuration provides a production-ready foundation for multi-site O-RAN deployments with comprehensive QoS guarantees and monitoring capabilities.