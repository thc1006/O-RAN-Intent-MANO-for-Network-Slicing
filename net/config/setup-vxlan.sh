#!/bin/bash
# VXLAN Tunnel Setup Script for Multi-Cluster Connectivity

set -euo pipefail

# Configuration from environment or defaults
LOCAL_IP=${LOCAL_IP:-$(hostname -I | awk '{print $1}')}
CLUSTER_NAME=${CLUSTER_NAME:-central}
OVS_BRIDGE=${OVS_BRIDGE:-br-int}
VXLAN_PORT=${VXLAN_PORT:-4789}
MTU_SIZE=${MTU_SIZE:-1450}

# Cluster endpoint mappings
declare -A CLUSTER_ENDPOINTS=(
    ["central"]="${CENTRAL_IP:-10.100.0.10}"
    ["regional"]="${REGIONAL_IP:-10.100.1.10}"
    ["edge01"]="${EDGE01_IP:-10.100.2.10}"
    ["edge02"]="${EDGE02_IP:-10.100.3.10}"
)

# VXLAN ID mappings
declare -A VXLAN_IDS=(
    ["central-regional"]=101
    ["central-edge01"]=102
    ["central-edge02"]=103
    ["regional-edge01"]=104
    ["regional-edge02"]=105
    ["edge01-edge02"]=106
)

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check for required commands
    local required_commands=("ip" "ovs-vsctl" "ovs-ofctl")
    for cmd in "${required_commands[@]}"; do
        if ! command -v ${cmd} &> /dev/null; then
            log_error "${cmd} is not installed"
            exit 1
        fi
    done

    # Check if OVS is running
    if ! ovs-vsctl show &> /dev/null; then
        log_error "Open vSwitch is not running"
        exit 1
    fi

    # Check if OVS bridge exists
    if ! ovs-vsctl br-exists ${OVS_BRIDGE}; then
        log_error "OVS bridge ${OVS_BRIDGE} does not exist"
        exit 1
    fi

    log_info "Prerequisites check passed"
}

# Get VXLAN ID for a tunnel
get_vxlan_id() {
    local src=$1
    local dst=$2

    # Create a sorted key for bidirectional mapping
    if [[ "${src}" < "${dst}" ]]; then
        local key="${src}-${dst}"
    else
        local key="${dst}-${src}"
    fi

    echo ${VXLAN_IDS[${key}]:-100}
}

# Create VXLAN tunnel
create_vxlan_tunnel() {
    local remote_cluster=$1
    local remote_ip=$2

    if [[ "${remote_cluster}" == "${CLUSTER_NAME}" ]]; then
        return 0  # Skip self
    fi

    local vxlan_name="vxlan-${remote_cluster}"
    local vxlan_id=$(get_vxlan_id ${CLUSTER_NAME} ${remote_cluster})

    log_info "Creating VXLAN tunnel to ${remote_cluster} (${remote_ip}) with ID ${vxlan_id}"

    # Delete existing interface if it exists
    ip link delete ${vxlan_name} 2>/dev/null || true

    # Create VXLAN interface
    ip link add ${vxlan_name} type vxlan \
        id ${vxlan_id} \
        remote ${remote_ip} \
        local ${LOCAL_IP} \
        dstport ${VXLAN_PORT} \
        ttl 64 \
        dev $(ip route | grep default | awk '{print $5}' | head -1)

    # Configure interface
    ip link set ${vxlan_name} mtu ${MTU_SIZE}
    ip link set ${vxlan_name} up

    # Add to OVS bridge
    ovs-vsctl --may-exist add-port ${OVS_BRIDGE} ${vxlan_name}

    # Set OVS interface options
    ovs-vsctl set interface ${vxlan_name} type=system

    log_info "VXLAN tunnel ${vxlan_name} created successfully"
}

# Configure OVS flows for routing
configure_ovs_flows() {
    log_info "Configuring OVS flows for inter-cluster routing"

    # Define cluster CIDR blocks
    declare -A CLUSTER_CIDRS=(
        ["central"]="10.0.0.0/16"
        ["regional"]="10.1.0.0/16"
        ["edge01"]="10.2.0.0/16"
        ["edge02"]="10.3.0.0/16"
    )

    # Clear existing flows (be careful in production!)
    # ovs-ofctl del-flows ${OVS_BRIDGE} "table=0,priority=100"

    # Add flows for each remote cluster
    for cluster in "${!CLUSTER_CIDRS[@]}"; do
        if [[ "${cluster}" != "${CLUSTER_NAME}" ]]; then
            local cidr=${CLUSTER_CIDRS[${cluster}]}
            local vxlan_port=$(ovs-vsctl get interface vxlan-${cluster} ofport 2>/dev/null || echo "")

            if [[ -n "${vxlan_port}" ]] && [[ "${vxlan_port}" != "[]" ]]; then
                # Add flow for outgoing traffic to remote cluster
                ovs-ofctl add-flow ${OVS_BRIDGE} \
                    "table=0,priority=100,ip,nw_dst=${cidr},actions=output:${vxlan_port}"

                # Add flow for ARP traffic
                ovs-ofctl add-flow ${OVS_BRIDGE} \
                    "table=0,priority=100,arp,nw_dst=${cidr},actions=output:${vxlan_port}"

                log_info "Added flow for ${cluster} (${cidr}) via port ${vxlan_port}"
            else
                log_warn "Could not find OVS port for vxlan-${cluster}"
            fi
        fi
    done

    # Add default flows for local traffic
    ovs-ofctl add-flow ${OVS_BRIDGE} \
        "table=0,priority=50,actions=normal"
}

# Verify tunnel connectivity
verify_tunnel() {
    local remote_cluster=$1
    local remote_ip=$2

    if [[ "${remote_cluster}" == "${CLUSTER_NAME}" ]]; then
        return 0
    fi

    log_info "Verifying tunnel to ${remote_cluster}"

    # Check if interface exists
    if ! ip link show vxlan-${remote_cluster} &> /dev/null; then
        log_error "VXLAN interface vxlan-${remote_cluster} does not exist"
        return 1
    fi

    # Check if interface is up
    if ! ip link show vxlan-${remote_cluster} | grep -q "state UP"; then
        log_warn "VXLAN interface vxlan-${remote_cluster} is not UP"
    fi

    # Check OVS port
    if ! ovs-vsctl port-to-br vxlan-${remote_cluster} &> /dev/null; then
        log_error "VXLAN interface vxlan-${remote_cluster} not attached to OVS bridge"
        return 1
    fi

    # Ping remote endpoint (if reachable)
    if ping -c 1 -W 2 ${remote_ip} &> /dev/null; then
        log_info "Remote endpoint ${remote_ip} is reachable"
    else
        log_warn "Cannot ping remote endpoint ${remote_ip} (may be blocked by firewall)"
    fi

    return 0
}

# Setup all tunnels
setup_all_tunnels() {
    log_info "Setting up VXLAN tunnels for cluster: ${CLUSTER_NAME}"

    # Create tunnels to all other clusters
    for cluster in "${!CLUSTER_ENDPOINTS[@]}"; do
        if [[ "${cluster}" != "${CLUSTER_NAME}" ]]; then
            create_vxlan_tunnel ${cluster} ${CLUSTER_ENDPOINTS[${cluster}]}
        fi
    done

    # Configure OVS flows
    configure_ovs_flows

    # Verify all tunnels
    local all_good=true
    for cluster in "${!CLUSTER_ENDPOINTS[@]}"; do
        if [[ "${cluster}" != "${CLUSTER_NAME}" ]]; then
            if ! verify_tunnel ${cluster} ${CLUSTER_ENDPOINTS[${cluster}]}; then
                all_good=false
            fi
        fi
    done

    if ${all_good}; then
        log_info "All VXLAN tunnels configured successfully"
    else
        log_warn "Some tunnels may not be fully operational"
    fi
}

# Show tunnel status
show_status() {
    echo "========================================="
    echo "VXLAN Tunnel Status for ${CLUSTER_NAME}"
    echo "========================================="

    # Show interfaces
    echo -e "\nVXLAN Interfaces:"
    ip -br link show | grep vxlan || echo "No VXLAN interfaces found"

    # Show OVS ports
    echo -e "\nOVS Ports:"
    ovs-vsctl list-ports ${OVS_BRIDGE} | grep vxlan || echo "No VXLAN ports in OVS"

    # Show flows
    echo -e "\nOVS Flows (VXLAN related):"
    ovs-ofctl dump-flows ${OVS_BRIDGE} | grep -E "nw_dst=10\.[0-3]\.0\.0/16" || echo "No flows configured"

    # Show tunnel details
    echo -e "\nTunnel Details:"
    for intf in $(ip link show | grep vxlan | awk -F: '{print $2}'); do
        echo -e "\n${intf}:"
        ip -d link show ${intf} | grep -A3 vxlan
    done
}

# Remove all VXLAN tunnels
cleanup_tunnels() {
    log_info "Removing all VXLAN tunnels"

    # Remove from OVS
    for port in $(ovs-vsctl list-ports ${OVS_BRIDGE} | grep vxlan); do
        ovs-vsctl del-port ${OVS_BRIDGE} ${port}
        log_info "Removed ${port} from OVS bridge"
    done

    # Delete interfaces
    for intf in $(ip link show | grep vxlan | awk -F: '{print $2}'); do
        ip link delete ${intf}
        log_info "Deleted interface ${intf}"
    done

    log_info "Cleanup complete"
}

# Main function
main() {
    case "${1:-setup}" in
        setup)
            check_root
            check_prerequisites
            setup_all_tunnels
            ;;
        status)
            show_status
            ;;
        cleanup)
            check_root
            cleanup_tunnels
            ;;
        verify)
            check_prerequisites
            for cluster in "${!CLUSTER_ENDPOINTS[@]}"; do
                if [[ "${cluster}" != "${CLUSTER_NAME}" ]]; then
                    verify_tunnel ${cluster} ${CLUSTER_ENDPOINTS[${cluster}]}
                fi
            done
            ;;
        *)
            echo "Usage: $0 [setup|status|cleanup|verify]"
            echo ""
            echo "Commands:"
            echo "  setup    - Create VXLAN tunnels and configure OVS"
            echo "  status   - Show current tunnel status"
            echo "  cleanup  - Remove all VXLAN tunnels"
            echo "  verify   - Verify tunnel connectivity"
            echo ""
            echo "Environment variables:"
            echo "  CLUSTER_NAME - Current cluster name (default: central)"
            echo "  LOCAL_IP     - Local IP address for VXLAN"
            echo "  CENTRAL_IP   - Central cluster endpoint IP"
            echo "  REGIONAL_IP  - Regional cluster endpoint IP"
            echo "  EDGE01_IP    - Edge-01 cluster endpoint IP"
            echo "  EDGE02_IP    - Edge-02 cluster endpoint IP"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"