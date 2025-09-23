# E2E Deployment Experiments

This directory contains the automated experiment suite for measuring E2E deployment times and performance metrics in the O-RAN Intent-Based MANO system.

## Overview

The experiment suite reproduces the deployment timing results from the thesis research, targeting two deployment series:
- **Fast Series**: {407, 353, 257} seconds for eMBB, uRLLC, mIoT
- **Slow Series**: {532, 292, 220} seconds for eMBB, uRLLC, mIoT

## Structure

```
experiments/
├── run_suite.sh              # Master orchestration script
├── collect_metrics.py        # Metrics collection and analysis
├── scenarios/                # Intent scenario definitions
│   ├── embb.yaml            # eMBB scenario config
│   ├── urllc.yaml           # uRLLC scenario config
│   └── miot.yaml            # mIoT scenario config
├── lib/                      # Deployment modules
│   ├── common.sh            # Common functions
│   ├── deploy_ran.sh        # RAN deployment
│   ├── deploy_tn.sh         # Transport Network deployment
│   └── deploy_cn.sh         # Core Network deployment
├── config/                   # Configuration files
│   ├── thresholds.yaml      # Performance thresholds
│   └── monitoring.yaml      # Monitoring configuration
├── results/                  # Experiment results
└── logs/                    # Execution logs
```

## Quick Start

1. **Setup Environment**:
   ```bash
   # Ensure cluster is ready
   kubectl cluster-info

   # Install dependencies
   sudo apt install bc yq
   pip3 install pyyaml jsonschema
   ```

2. **Validate Setup**:
   ```bash
   # Run smoke test
   ./validate_suite.sh smoke

   # Check prerequisites
   ./validate_suite.sh prereqs
   ```

3. **Run Test Harness**:
   ```bash
   # Complete test with fast series
   python3 test_harness.py --series fast

   # Complete test with slow series
   python3 test_harness.py --series slow
   ```

4. **Run Individual Components**:
   ```bash
   # Fast series deployment
   ./run_suite.sh fast

   # Slow series deployment
   ./run_suite.sh slow
   ```

5. **View Results**:
   ```bash
   # JSON report
   cat results/report_<timestamp>.json

   # Test harness report
   cat results/test_report_<series>_<timestamp>.json

   # HTML visualization
   open results/report_<timestamp>.html
   ```

## Deployment Scenarios

### eMBB (Enhanced Mobile Broadband)
- **Target**: 407s (fast) / 532s (slow)
- **Profile**: High bandwidth (4.57 Mbps), relaxed latency (16.1 ms)
- **Use Case**: 4K video streaming, content delivery
- **Bottleneck**: SMF initialization (120s expected)

### uRLLC (Ultra-Reliable Low Latency)
- **Target**: 353s (fast) / 292s (slow)
- **Profile**: Low bandwidth (0.93 Mbps), ultra-low latency (6.3 ms)
- **Use Case**: Industrial automation, robotic control
- **Bottleneck**: Minimal SMF delay (30s expected)

### mIoT (Massive IoT)
- **Target**: 257s (fast) / 220s (slow)
- **Profile**: Balanced bandwidth (2.77 Mbps), moderate latency (15.7 ms)
- **Use Case**: IoT deployments, sensor networks
- **Bottleneck**: Moderate SMF delay (60s expected)

## Metrics Collected

### Timing Metrics
- Total E2E deployment time
- Per-domain breakdown (RAN/TN/CN)
- Intent processing time
- GitOps synchronization time

### Resource Metrics
- SMO CPU utilization (peak/average)
- SMO memory consumption (peak/average)
- O-Cloud memory usage per node
- Pod count and lifecycle events

### Bottleneck Detection
- SMF initialization delay timeline
- CPU spikes during deployment
- Memory pressure events
- Porch package sync times
- ConfigSync latencies

## Validation Criteria

### Performance Targets
- **Deployment Time**: Within ±tolerance of target values
- **Throughput**: Measured via iPerf3 (±10% tolerance)
- **Latency**: Measured via ping (±2ms tolerance)

### Resource Limits
- **SMO CPU**: < 2.0 cores peak
- **SMO Memory**: < 4GB peak
- **O-Cloud Memory**: < 16GB per node

### Bottleneck Requirements
- **SMF Delay Detection**: Must observe >60s initialization
- **Timeline Events**: Pod lifecycle progression
- **Resource Spikes**: CPU >80% during bottleneck

## Usage Examples

### Run Single Scenario
```bash
# Test only eMBB scenario
./lib/deploy_ran.sh embb fast
./lib/deploy_tn.sh embb fast
./lib/deploy_cn.sh embb fast
```

### Metrics Collection Only
```bash
# Collect system metrics
python3 collect_metrics.py collect_system \
  --output metrics.json \
  --smo-namespace oran-system \
  --ocloud-nodes kind-worker,kind-worker2

# Monitor SMF bottleneck
python3 collect_metrics.py monitor_smf --scenario embb
```

### Custom Validation
```bash
# Generate report with custom thresholds
python3 collect_metrics.py generate_report \
  --metrics results/metrics_*.json \
  --output custom_report.json \
  --html custom_report.html

# Run comprehensive test harness
python3 test_harness.py \
  --series fast \
  --verbose \
  --output custom_test_report.json

# Validate suite configuration
./validate_suite.sh all --verbose
```

## Configuration

### Thresholds (config/thresholds.yaml)
Customize deployment time targets, resource limits, and validation criteria.

### Monitoring (config/monitoring.yaml)
Configure metrics collection intervals and data sources.

### Scenarios (scenarios/*.yaml)
Modify intent requirements, resource specifications, and deployment sequences.

## Troubleshooting

### Common Issues

1. **Deployment Timeout**:
   ```bash
   # Check pod status
   kubectl get pods -A --field-selector status.phase!=Running

   # Check resource constraints
   kubectl describe nodes
   ```

2. **Metrics Collection Failure**:
   ```bash
   # Verify metrics server
   kubectl top nodes

   # Check Python dependencies
   python3 -c "import json, subprocess, time"
   ```

3. **SMF Bottleneck Not Detected**:
   ```bash
   # Check SMF pod logs
   kubectl logs -n oran-system -l component=smf

   # Verify monitoring configuration
   cat config/thresholds.yaml
   ```

### Debug Mode
```bash
# Enable debug logging
DEBUG=1 ./run_suite.sh fast

# Verbose metrics collection
python3 collect_metrics.py continuous --interval 1

# Comprehensive validation
./validate_suite.sh all --verbose

# Test harness with verbose output
python3 test_harness.py --series fast --verbose
```

## Results Analysis

### Success Criteria
- All deployment times within tolerance
- SMF bottleneck observable in timeline
- Resource usage below thresholds
- Performance targets achieved

### Report Format
```json
{
  "validation": {
    "passed": true,
    "series_type": "fast",
    "details": {
      "embb": {"actual": 410, "target": 407, "passed": true},
      "urllc": {"actual": 350, "target": 353, "passed": true},
      "miot": {"actual": 260, "target": 257, "passed": true}
    }
  },
  "bottlenecks": {
    "smf_init_delay": 118,
    "smf_timeline": [...]
  },
  "resources": {
    "smo_cpu_peak": 1.8,
    "smo_memory_peak": 3584
  }
}
```

## Integration

This experiment suite integrates with:
- **VNF Operator**: Deploys RAN/CN functions
- **TN Manager**: Creates transport slices
- **Nephio**: GitOps package management
- **Kubernetes**: Resource orchestration
- **Prometheus**: Metrics collection (optional)

For questions or issues, see the main project documentation.