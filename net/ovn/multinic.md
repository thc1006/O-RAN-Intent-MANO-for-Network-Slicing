# Kube-OVN Multi-NIC Multi-Cluster Configuration

## Overview

This guide configures Kube-OVN for multi-cluster connectivity with multiple network interfaces (NICs) per pod, enabling inter-site communication with controlled latency for O-RAN network slicing.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Central Cloud                         │
│  ┌──────────────┐                      ┌──────────────┐    │
│  │   Kube-OVN   │──────VXLAN──────────│   Services   │    │
│  │  10.0.0.0/16 │                      │              │    │
│  └──────────────┘                      └──────────────┘    │
└─────────────────────────────────────────────────────────────┘
         │                                        │
         │ 7ms delay                             │ 7ms delay
         │                                        │
┌────────▼────────────────┐          ┌───────────▼────────────┐
│    Regional Cloud       │          │      Edge-01           │
│  ┌──────────────┐      │          │  ┌──────────────┐     │
│  │   Kube-OVN   │◄─────┼──5ms─────┼─►│   Kube-OVN   │     │
│  │  10.1.0.0/16 │      │          │  │  10.2.0.0/16 │     │
│  └──────────────┘      │          │  └──────────────┘     │
└─────────────────────────┘          └──────────────────────┘
         │                                        │
         │ 5ms delay                             │ 5ms delay
         │                                        │
         └────────────────┬──────────────────────┘
                          │
                 ┌────────▼────────────┐
                 │      Edge-02        │
                 │  ┌──────────────┐  │
                 │  │   Kube-OVN   │  │
                 │  │  10.3.0.0/16 │  │
                 │  └──────────────┘  │
                 └─────────────────────┘
```

## Prerequisites

- Kubernetes clusters (1.24+) at each site
- Kube-OVN v1.12.0 or later
- Network connectivity between clusters
- Root/sudo access for TC configuration

## Installation

### 1. Install Kube-OVN on Each Cluster

```bash
# Download Kube-OVN installer
wget https://raw.githubusercontent.com/kubeovn/kube-ovn/release-1.12/dist/images/install.sh

# Customize for each cluster
# Central cluster
MASTER_NODES="central-master" \
POD_CIDR="10.0.0.0/16" \
SVC_CIDR="10.96.0.0/16" \
JOIN_CIDR="100.64.0.0/16" \
bash install.sh

# Regional cluster
MASTER_NODES="regional-master" \
POD_CIDR="10.1.0.0/16" \
SVC_CIDR="10.97.0.0/16" \
JOIN_CIDR="100.65.0.0/16" \
bash install.sh

# Edge-01 cluster
MASTER_NODES="edge01-master" \
POD_CIDR="10.2.0.0/16" \
SVC_CIDR="10.98.0.0/16" \
JOIN_CIDR="100.66.0.0/16" \
bash install.sh

# Edge-02 cluster
MASTER_NODES="edge02-master" \
POD_CIDR="10.3.0.0/16" \
SVC_CIDR="10.99.0.0/16" \
JOIN_CIDR="100.67.0.0/16" \
bash install.sh
```

### 2. Configure Multi-NIC Support

Create additional subnets for multi-NIC pods:

```yaml
# additional-subnet.yaml
apiVersion: kubeovn.io/v1
kind: Subnet
metadata:
  name: data-subnet
spec:
  protocol: IPv4
  cidrBlock: 172.16.0.0/24
  gateway: 172.16.0.1
  excludeIps:
  - 172.16.0.1..172.16.0.10
  namespaces:
  - default
---
apiVersion: kubeovn.io/v1
kind: Subnet
metadata:
  name: mgmt-subnet
spec:
  protocol: IPv4
  cidrBlock: 192.168.0.0/24
  gateway: 192.168.0.1
  excludeIps:
  - 192.168.0.1..192.168.0.10
  namespaces:
  - default
```

Apply to each cluster with appropriate CIDR adjustments:

```bash
kubectl apply -f additional-subnet.yaml
```

### 3. Configure Network Attachment Definitions

Create network attachment definitions for Multus CNI:

```yaml
# net-attach-def.yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: data-network
  namespace: default
spec:
  config: '{
    "cniVersion": "0.3.1",
    "type": "kube-ovn",
    "server_socket": "/run/openvswitch/kube-ovn-daemon.sock",
    "provider": "data-subnet"
  }'
---
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: mgmt-network
  namespace: default
spec:
  config: '{
    "cniVersion": "0.3.1",
    "type": "kube-ovn",
    "server_socket": "/run/openvswitch/kube-ovn-daemon.sock",
    "provider": "mgmt-subnet"
  }'
```

## Inter-Cluster VXLAN Tunnels

### 1. Create VXLAN Interfaces

On each cluster's nodes, create VXLAN tunnels to other sites:

```bash
#!/bin/bash
# setup-vxlan.sh - Run on each node

# Configuration
LOCAL_IP=$(hostname -I | awk '{print $1}')
VXLAN_ID=100

# Function to create VXLAN tunnel
create_tunnel() {
  local remote_ip=$1
  local remote_name=$2
  local vxlan_name="vxlan-${remote_name}"
  local vxlan_id=$3

  # Delete if exists
  ip link delete ${vxlan_name} 2>/dev/null || true

  # Create VXLAN interface
  ip link add ${vxlan_name} type vxlan \
    id ${vxlan_id} \
    remote ${remote_ip} \
    local ${LOCAL_IP} \
    dstport 4789 \
    dev eth0

  # Bring up interface
  ip link set ${vxlan_name} up

  # Add to OVS bridge
  ovs-vsctl add-port br-int ${vxlan_name} || true

  echo "Created tunnel ${vxlan_name} to ${remote_ip}"
}

# Central to Regional (VXLAN ID 101)
create_tunnel "REGIONAL_NODE_IP" "regional" 101

# Central to Edge-01 (VXLAN ID 102)
create_tunnel "EDGE01_NODE_IP" "edge01" 102

# Central to Edge-02 (VXLAN ID 103)
create_tunnel "EDGE02_NODE_IP" "edge02" 103

# Add more tunnels as needed based on current node
```

### 2. Configure OVS Flows

Add OpenFlow rules for inter-cluster routing:

```bash
# On Central cluster OVS
ovs-ofctl add-flow br-int "table=0,priority=100,ip,nw_dst=10.1.0.0/16,actions=output:vxlan-regional"
ovs-ofctl add-flow br-int "table=0,priority=100,ip,nw_dst=10.2.0.0/16,actions=output:vxlan-edge01"
ovs-ofctl add-flow br-int "table=0,priority=100,ip,nw_dst=10.3.0.0/16,actions=output:vxlan-edge02"

# On Regional cluster OVS
ovs-ofctl add-flow br-int "table=0,priority=100,ip,nw_dst=10.0.0.0/16,actions=output:vxlan-central"
ovs-ofctl add-flow br-int "table=0,priority=100,ip,nw_dst=10.2.0.0/16,actions=output:vxlan-edge01"
ovs-ofctl add-flow br-int "table=0,priority=100,ip,nw_dst=10.3.0.0/16,actions=output:vxlan-edge02"

# Similar for Edge clusters
```

## Traffic Control (TC) Delay Configuration

### 1. Apply Delays on VXLAN Interfaces

Use TC to inject delays on inter-site links:

```bash
#!/bin/bash
# configure-delays.sh

# Function to add delay to interface
add_delay() {
  local interface=$1
  local delay=$2

  # Clear existing qdisc
  tc qdisc del dev ${interface} root 2>/dev/null || true

  # Add delay using netem
  tc qdisc add dev ${interface} root netem delay ${delay}ms

  echo "Added ${delay}ms delay to ${interface}"
}

# Configure delays based on topology
# On Central node
add_delay "vxlan-regional" 7
add_delay "vxlan-edge01" 7
add_delay "vxlan-edge02" 7

# On Regional node
add_delay "vxlan-central" 7
add_delay "vxlan-edge01" 5
add_delay "vxlan-edge02" 5

# On Edge-01 node
add_delay "vxlan-central" 7
add_delay "vxlan-regional" 5
add_delay "vxlan-edge02" 5

# On Edge-02 node
add_delay "vxlan-central" 7
add_delay "vxlan-regional" 5
add_delay "vxlan-edge01" 5
```

### 2. Verify Delay Configuration

```bash
# Check qdisc configuration
tc qdisc show dev vxlan-regional

# Expected output:
# qdisc netem 8001: root refcnt 2 limit 1000 delay 7.0ms
```

## Multi-NIC Pod Configuration

### 1. Create Multi-NIC Pods

```yaml
# multi-nic-pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: multi-nic-test
  annotations:
    k8s.v1.cni.cncf.io/networks: |
      [
        {
          "name": "data-network",
          "interface": "eth1"
        },
        {
          "name": "mgmt-network",
          "interface": "eth2"
        }
      ]
spec:
  containers:
  - name: network-tools
    image: nicolaka/netshoot:latest
    command: ["/bin/sleep", "3650d"]
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - NET_RAW
```

### 2. Verify Multi-NIC Configuration

```bash
# Check interfaces in pod
kubectl exec multi-nic-test -- ip addr

# Expected output:
# 1: lo: <LOOPBACK,UP,LOWER_UP>
# 2: eth0@if5: <BROADCAST,MULTICAST,UP,LOWER_UP>  # Default network
#     inet 10.0.1.2/24
# 3: eth1@if6: <BROADCAST,MULTICAST,UP,LOWER_UP>  # Data network
#     inet 172.16.0.2/24
# 4: eth2@if7: <BROADCAST,MULTICAST,UP,LOWER_UP>  # Mgmt network
#     inet 192.168.0.2/24
```

## Service Mesh Configuration

### 1. Create Cross-Cluster Services

```yaml
# cross-cluster-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: cross-cluster-app
  annotations:
    kubeovn.io/logical-switch: "data-subnet"
spec:
  selector:
    app: cross-cluster
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
---
apiVersion: v1
kind: Endpoints
metadata:
  name: cross-cluster-app
subsets:
- addresses:
  - ip: 10.1.2.10  # Pod IP in Regional cluster
  - ip: 10.2.3.20  # Pod IP in Edge-01 cluster
  ports:
  - port: 8080
```

### 2. Configure Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-cross-cluster
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - ipBlock:
        cidr: 10.0.0.0/8  # Allow all cluster networks
  egress:
  - to:
    - ipBlock:
        cidr: 10.0.0.0/8
```

## Monitoring and Troubleshooting

### 1. Check Connectivity

```bash
# Test pod-to-pod connectivity
kubectl exec -it pod-in-central -- ping -c 5 10.1.2.10

# Check latency
kubectl exec -it pod-in-central -- ping -c 100 -i 0.1 10.2.3.20 | tail -1

# Trace route
kubectl exec -it pod-in-central -- traceroute 10.3.4.30
```

### 2. OVS Debugging

```bash
# Show OVS bridges
ovs-vsctl show

# Show flows
ovs-ofctl dump-flows br-int

# Monitor traffic
ovs-tcpdump -i vxlan-regional

# Check tunnel status
ovs-appctl ofproto/list-tunnels
```

### 3. TC Statistics

```bash
# Show qdisc statistics
tc -s qdisc show dev vxlan-regional

# Monitor dropped packets
watch -n 1 'tc -s qdisc show dev vxlan-regional'
```

## Performance Tuning

### 1. MTU Configuration

```bash
# Set MTU for VXLAN (account for overhead)
ip link set dev vxlan-regional mtu 1450

# Configure in Kube-OVN
kubectl edit configmap ovn-config -n kube-system
# Set: MTU=1450
```

### 2. CPU Affinity

```bash
# Pin OVS threads to specific CPUs
ovs-vsctl set Open_vSwitch . other_config:pmd-cpu-mask=0x6
```

### 3. Buffer Tuning

```bash
# Increase network buffers
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_rmem = 4096 87380 134217728' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_wmem = 4096 65536 134217728' >> /etc/sysctl.conf
sysctl -p
```

## Automation Scripts

### Complete Setup Script

```bash
#!/bin/bash
# setup-multicluster.sh

set -e

CLUSTER_NAME=${1:-central}
CLUSTER_CIDR=${2:-10.0.0.0/16}
SERVICE_CIDR=${3:-10.96.0.0/16}

echo "Setting up Kube-OVN for cluster: ${CLUSTER_NAME}"

# Install Kube-OVN
kubectl apply -f https://raw.githubusercontent.com/kubeovn/kube-ovn/release-1.12/dist/images/kube-ovn.yaml

# Wait for installation
kubectl wait --for=condition=ready pod -l app=kube-ovn-controller -n kube-system --timeout=300s

# Create additional subnets
kubectl apply -f - <<EOF
apiVersion: kubeovn.io/v1
kind: Subnet
metadata:
  name: data-subnet-${CLUSTER_NAME}
spec:
  protocol: IPv4
  cidrBlock: 172.16.${CLUSTER_NAME#*-}.0/24
  gateway: 172.16.${CLUSTER_NAME#*-}.1
EOF

# Configure VXLAN tunnels
./setup-vxlan.sh

# Apply TC delays
./configure-delays.sh

echo "Multi-cluster setup complete for ${CLUSTER_NAME}"
```

## Validation Checklist

- [ ] Kube-OVN installed on all clusters
- [ ] Multi-NIC subnets configured
- [ ] VXLAN tunnels established
- [ ] TC delays applied correctly
- [ ] Pod-to-pod connectivity works
- [ ] Services reachable across sites
- [ ] RTT matches configured delays (±1ms)
- [ ] Network policies enforced
- [ ] Monitoring configured