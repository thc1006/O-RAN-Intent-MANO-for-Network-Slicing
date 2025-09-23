#!/bin/bash
# Traffic Control (TC) Delay Configuration for Inter-Site Links

set -euo pipefail

# Configuration
CLUSTER_NAME=${CLUSTER_NAME:-central}

# Delay matrix (in milliseconds)
declare -A DELAY_MATRIX=(
    ["central-regional"]=7
    ["central-edge01"]=7
    ["central-edge02"]=7
    ["regional-central"]=7
    ["regional-edge01"]=5
    ["regional-edge02"]=5
    ["edge01-central"]=7
    ["edge01-regional"]=5
    ["edge01-edge02"]=5
    ["edge02-central"]=7
    ["edge02-regional"]=5
    ["edge02-edge01"]=5
)

# Additional TC parameters
JITTER=${JITTER:-0.5}        # Jitter in ms
LOSS=${LOSS:-0}              # Packet loss percentage
CORRELATION=${CORRELATION:-25} # Correlation percentage
DISTRIBUTION=${DISTRIBUTION:-normal} # Delay distribution

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Check if tc is available
check_tc() {
    if ! command -v tc &> /dev/null; then
        log_error "tc (traffic control) is not installed"
        echo "Install with: apt-get install iproute2 (Debian/Ubuntu) or yum install iproute-tc (RHEL/CentOS)"
        exit 1
    fi
}

# Get delay for a link
get_delay() {
    local src=$1
    local dst=$2
    local key="${src}-${dst}"

    echo ${DELAY_MATRIX[${key}]:-0}
}

# Apply delay to interface
apply_delay() {
    local interface=$1
    local delay=$2
    local jitter=${3:-${JITTER}}
    local loss=${4:-${LOSS}}

    log_info "Configuring TC on ${interface}: delay=${delay}ms jitter=${jitter}ms loss=${loss}%"

    # Check if interface exists
    if ! ip link show ${interface} &> /dev/null; then
        log_error "Interface ${interface} does not exist"
        return 1
    fi

    # Remove existing qdisc
    tc qdisc del dev ${interface} root 2>/dev/null || true

    if [[ "${delay}" == "0" ]] && [[ "${loss}" == "0" ]]; then
        log_info "No delay or loss configured for ${interface}"
        return 0
    fi

    # Build netem command
    local netem_cmd="tc qdisc add dev ${interface} root netem"

    # Add delay if specified
    if [[ "${delay}" != "0" ]]; then
        netem_cmd="${netem_cmd} delay ${delay}ms"

        # Add jitter if specified
        if [[ "${jitter}" != "0" ]]; then
            netem_cmd="${netem_cmd} ${jitter}ms ${CORRELATION}%"
        fi

        # Add distribution
        netem_cmd="${netem_cmd} distribution ${DISTRIBUTION}"
    fi

    # Add packet loss if specified
    if [[ "${loss}" != "0" ]]; then
        netem_cmd="${netem_cmd} loss ${loss}%"
    fi

    # Apply the configuration
    eval ${netem_cmd}

    if [[ $? -eq 0 ]]; then
        log_info "TC configuration applied successfully to ${interface}"
    else
        log_error "Failed to apply TC configuration to ${interface}"
        return 1
    fi
}

# Configure delays for all VXLAN interfaces
configure_all_delays() {
    log_info "Configuring delays for cluster: ${CLUSTER_NAME}"

    # Get all VXLAN interfaces
    local vxlan_interfaces=$(ip link show | grep -o 'vxlan-[a-z0-9]*' || true)

    if [[ -z "${vxlan_interfaces}" ]]; then
        log_warn "No VXLAN interfaces found. Run setup-vxlan.sh first."
        return 1
    fi

    # Apply delays to each interface
    for interface in ${vxlan_interfaces}; do
        # Extract remote cluster name from interface name
        local remote_cluster=${interface#vxlan-}

        # Get delay for this link
        local delay=$(get_delay ${CLUSTER_NAME} ${remote_cluster})

        if [[ "${delay}" != "0" ]]; then
            apply_delay ${interface} ${delay}
        else
            log_warn "No delay configured for ${CLUSTER_NAME} to ${remote_cluster}"
        fi
    done

    log_info "Delay configuration complete"
}

# Show current TC configuration
show_tc_status() {
    echo "========================================="
    echo "TC Configuration for ${CLUSTER_NAME}"
    echo "========================================="

    local vxlan_interfaces=$(ip link show | grep -o 'vxlan-[a-z0-9]*' || true)

    if [[ -z "${vxlan_interfaces}" ]]; then
        echo "No VXLAN interfaces found"
        return
    fi

    for interface in ${vxlan_interfaces}; do
        echo -e "\n${BLUE}Interface: ${interface}${NC}"

        # Show qdisc configuration
        local qdisc_info=$(tc qdisc show dev ${interface} 2>/dev/null || echo "No qdisc configured")
        echo "Qdisc: ${qdisc_info}"

        # Show statistics
        if tc qdisc show dev ${interface} | grep -q netem; then
            echo -e "\nStatistics:"
            tc -s qdisc show dev ${interface} | grep -A5 "qdisc netem"
        fi
    done
}

# Test latency to verify configuration
test_latency() {
    local target_ip=$1
    local expected_delay=$2
    local tolerance=${3:-1}

    log_info "Testing latency to ${target_ip} (expected: ${expected_delay}±${tolerance}ms)"

    # Run ping test
    local ping_result=$(ping -c 10 -i 0.2 -q ${target_ip} 2>/dev/null || echo "FAILED")

    if [[ "${ping_result}" == "FAILED" ]]; then
        log_error "Cannot reach ${target_ip}"
        return 1
    fi

    # Extract average RTT
    local avg_rtt=$(echo "${ping_result}" | grep "rtt min/avg/max" | awk -F'/' '{print $5}')

    if [[ -z "${avg_rtt}" ]]; then
        log_error "Could not parse ping results"
        return 1
    fi

    # Check if RTT is within expected range
    local min_expected=$(awk "BEGIN {print ${expected_delay} - ${tolerance}}")
    local max_expected=$(awk "BEGIN {print ${expected_delay} + ${tolerance}}")

    if awk "BEGIN {exit !(${avg_rtt} >= ${min_expected} && ${avg_rtt} <= ${max_expected})}"; then
        log_info "✓ RTT: ${avg_rtt}ms (within expected range)"
        return 0
    else
        log_warn "✗ RTT: ${avg_rtt}ms (expected: ${expected_delay}±${tolerance}ms)"
        return 1
    fi
}

# Remove all TC configurations
cleanup_tc() {
    log_info "Removing all TC configurations"

    local vxlan_interfaces=$(ip link show | grep -o 'vxlan-[a-z0-9]*' || true)

    if [[ -z "${vxlan_interfaces}" ]]; then
        log_warn "No VXLAN interfaces found"
        return
    fi

    for interface in ${vxlan_interfaces}; do
        tc qdisc del dev ${interface} root 2>/dev/null || true
        log_info "Cleared TC configuration on ${interface}"
    done

    log_info "TC cleanup complete"
}

# Advanced TC configuration with bandwidth limits
configure_advanced() {
    local interface=$1
    local delay=$2
    local bandwidth=${3:-1000}  # Default 1Gbps
    local burst=${4:-32}        # Burst size in KB

    log_info "Applying advanced TC configuration to ${interface}"

    # Remove existing configuration
    tc qdisc del dev ${interface} root 2>/dev/null || true

    # Create HTB root qdisc
    tc qdisc add dev ${interface} root handle 1: htb default 10

    # Create class with bandwidth limit
    tc class add dev ${interface} parent 1: classid 1:10 htb rate ${bandwidth}mbit burst ${burst}k

    # Add netem for delay
    tc qdisc add dev ${interface} parent 1:10 handle 10: netem \
        delay ${delay}ms ${JITTER}ms ${CORRELATION}% \
        distribution ${DISTRIBUTION} \
        loss ${LOSS}%

    log_info "Advanced TC configuration applied: bandwidth=${bandwidth}Mbit delay=${delay}ms"
}

# Monitor TC statistics
monitor_tc() {
    local interface=$1
    local interval=${2:-1}

    log_info "Monitoring TC statistics for ${interface} (press Ctrl+C to stop)"

    while true; do
        clear
        echo "========================================="
        echo "TC Statistics for ${interface}"
        echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "========================================="

        # Show qdisc statistics
        tc -s qdisc show dev ${interface}

        # Show class statistics if HTB is configured
        if tc class show dev ${interface} | grep -q htb; then
            echo -e "\nHTB Classes:"
            tc -s class show dev ${interface}
        fi

        sleep ${interval}
    done
}

# Main function
main() {
    case "${1:-configure}" in
        configure)
            check_root
            check_tc
            configure_all_delays
            ;;
        status)
            show_tc_status
            ;;
        cleanup)
            check_root
            cleanup_tc
            ;;
        test)
            if [[ $# -lt 3 ]]; then
                echo "Usage: $0 test <target_ip> <expected_delay_ms>"
                exit 1
            fi
            test_latency "$2" "$3"
            ;;
        advanced)
            if [[ $# -lt 3 ]]; then
                echo "Usage: $0 advanced <interface> <delay_ms> [bandwidth_mbit]"
                exit 1
            fi
            check_root
            check_tc
            configure_advanced "$2" "$3" "${4:-1000}"
            ;;
        monitor)
            if [[ $# -lt 2 ]]; then
                echo "Usage: $0 monitor <interface> [interval_seconds]"
                exit 1
            fi
            monitor_tc "$2" "${3:-1}"
            ;;
        *)
            echo "Usage: $0 [configure|status|cleanup|test|advanced|monitor]"
            echo ""
            echo "Commands:"
            echo "  configure           - Apply delays to all VXLAN interfaces"
            echo "  status             - Show current TC configuration"
            echo "  cleanup            - Remove all TC configurations"
            echo "  test <ip> <delay>  - Test latency to IP"
            echo "  advanced <if> <ms> - Configure advanced TC with bandwidth"
            echo "  monitor <if>       - Monitor TC statistics"
            echo ""
            echo "Environment variables:"
            echo "  CLUSTER_NAME - Current cluster name (default: central)"
            echo "  JITTER      - Jitter in ms (default: 0.5)"
            echo "  LOSS        - Packet loss % (default: 0)"
            echo ""
            echo "Configured delays:"
            for key in "${!DELAY_MATRIX[@]}"; do
                echo "  ${key}: ${DELAY_MATRIX[${key}]}ms"
            done
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"