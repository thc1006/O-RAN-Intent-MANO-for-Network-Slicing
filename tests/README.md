# O-RAN Intent-MANO Integration Test Suite

This directory contains comprehensive integration tests for the O-RAN Intent-MANO system, focusing on reproducing thesis performance targets and validating end-to-end deployment workflows.

## Test Structure

### Integration Tests (`integration/`)

#### End-to-End Workflow Tests (`e2e_workflow_test.go`)
- **Intent → QoS → VNF Deployment**: Complete workflow validation from natural language intent to deployed VNFs
- **Multi-cluster Deployment**: Validation across edge, regional, and central clusters
- **Performance Targets**: Validates thesis targets (6.3ms, 15.7ms, 16.1ms RTT; 0.93, 2.77, 4.57 Mbps)
- **Deployment Timing**: Ensures E2E deployment completes within 10 minutes

**Key Test Scenarios:**
- Ultra-low latency edge computing (autonomous vehicles)
- High-bandwidth streaming services (4K video)
- Balanced IoT services (smart city applications)

#### Multi-Cluster Deployment Tests (`multi_cluster_test.go`)
- **Cluster Validation**: Health checks and resource validation across clusters
- **Failover Testing**: Cluster failure and recovery scenarios
- **Cross-cluster Communication**: Network connectivity and performance validation
- **Load Distribution**: Optimal load balancing across cluster types

**Test Coverage:**
- Edge cluster deployments (low latency requirements)
- Regional cluster deployments (balanced performance)
- Central cluster deployments (high bandwidth requirements)
- Cascading failure scenarios and recovery

#### O2 Interface Integration Tests (`o2_interface_test.go`)
- **O2IMS Testing**: Infrastructure Management Service validation
- **O2DMS Testing**: Deployment Management Service validation
- **VNF Lifecycle**: Complete VNF deployment lifecycle via O2 interfaces
- **CNF Support**: Containerized Network Function deployment testing

**API Coverage:**
- Infrastructure discovery and inventory
- Resource pool management
- VNF/CNF deployment operations
- Event subscription and notification
- Performance and error handling

#### Nephio Integration Tests (`nephio_integration_test.go`)
- **Porch Package Generation**: GitOps package creation and validation
- **Config Sync**: Multi-cluster configuration synchronization
- **Package Versioning**: Version control and rollback scenarios
- **GitOps Workflow**: Complete GitOps deployment pipeline

**Features Tested:**
- Package generation from VNF specifications
- Multi-site package consistency
- Deployment via GitOps workflows
- Configuration drift detection and correction
- Performance at scale (concurrent deployments)

#### VNF Operator Lifecycle Tests (`vnf_lifecycle_test.go`)
- **Complete Lifecycle**: Creation, deployment, scaling, upgrade, termination
- **Operator Behavior**: Controller response times and reconciliation loops
- **Failure Recovery**: Pod, node, and network failure scenarios
- **Performance Optimization**: Resource efficiency and timing validation

**Lifecycle Phases:**
- VNF creation and validation
- Deployment and readiness monitoring
- Scaling (up/down and auto-scaling)
- Rolling upgrades and rollbacks
- Failure detection and recovery

### Performance Tests (`performance/`)

#### QoS Validation Tests (`qos_validation_test.go`)
- **Latency Validation**: Thesis targets (6.3ms, 15.7ms, 16.1ms RTT)
- **Throughput Validation**: Thesis targets (0.93, 2.77, 4.57 Mbps)
- **Network Slice Isolation**: QoS enforcement between slices
- **SLA Compliance**: Continuous monitoring and violation detection

**Test Profiles:**
- uRLLC: Ultra-reliable low-latency communication
- mIoT: Massive IoT with balanced requirements
- eMBB: Enhanced mobile broadband
- Edge Gaming: Real-time applications
- Industrial Automation: Stringent latency requirements

#### Network Performance Tests (`network_performance_test.go`)
- **Thesis Target Validation**: Comprehensive performance measurement
- **Multi-hop Path Testing**: End-to-end network path validation
- **Stress Testing**: Performance under network congestion
- **Slice Isolation**: Concurrent slice performance validation

**Measurement Types:**
- High-frequency latency testing (1000+ samples)
- Sustained throughput testing (extended duration)
- Burst capacity testing
- Performance consistency over time
- Resource utilization correlation

### End-to-End Tests (`e2e/`)

#### Deployment Timing Validation (`deployment_timing_test.go`)
- **10-Minute SLA**: Validates E2E deployment within thesis requirement
- **Phase Analysis**: Detailed timing breakdown of deployment phases
- **Bottleneck Identification**: Performance optimization recommendations
- **Stress Testing**: Timing under resource constraints and high load

**Timing Scenarios:**
- Single VNF edge deployment (< 5 minutes)
- Multi-VNF core network (< 8 minutes)
- Cross-cluster distributed deployment (< 10 minutes)
- High-bandwidth video streaming (< 7 minutes)
- Stress test with multiple parallel deployments

## Thesis Performance Targets

The test suite validates the following performance targets from the research:

### Network Latency Targets
- **uRLLC**: 6.3ms RTT (ultra-low latency)
- **mIoT**: 15.7ms RTT (balanced performance)
- **eMBB**: 16.1ms RTT (high bandwidth)

### Throughput Targets
- **uRLLC**: 0.93 Mbps (latency-optimized)
- **mIoT**: 2.77 Mbps (balanced)
- **eMBB**: 4.57 Mbps (bandwidth-optimized)

### Deployment Timing
- **E2E Deployment**: < 10 minutes (complete intent-to-deployment)
- **Individual VNF**: < 5 minutes (single VNF deployment)
- **SLA Compliance**: ≥ 90% of deployments meet timing requirements

## Running the Tests

### Prerequisites
- Go 1.21+
- Kubernetes clusters (kind recommended for testing)
- O2 interface implementations
- Nephio/Porch installation
- Network simulation tools

### Quick Start
```bash
# Run all integration tests
go test ./integration/... -v

# Run performance validation
go test ./performance/... -v

# Run end-to-end timing tests
go test ./e2e/... -v

# Run specific test suite
go test ./integration/e2e_workflow_test.go -v

# Run with extended timeout for full scenarios
go test ./integration/... -timeout 60m -v
```

### Test Configuration
Tests use environment variables and configuration files:

```bash
# Set test cluster contexts
export EDGE_CLUSTER_CONTEXT="kind-edge-01"
export REGIONAL_CLUSTER_CONTEXT="kind-regional-01"
export CENTRAL_CLUSTER_CONTEXT="kind-central-01"

# Configure O2 endpoints
export O2IMS_ENDPOINT="http://o2ims-service:8080"
export O2DMS_ENDPOINT="http://o2dms-service:8080"

# Set Nephio/Porch configuration
export PORCH_ENDPOINT="http://porch-server:7007"
export CONFIG_SYNC_REPO="https://github.com/org/config-repo"
```

### Test Reports
Tests generate comprehensive reports in JSON format:

- `testdata/results/e2e_results_*.json` - E2E workflow results
- `testdata/results/performance_report_*.json` - Performance metrics
- `testdata/timing_reports/deployment_timing_report_*.json` - Timing analysis

## Test Categories

### Functional Tests
- ✅ Intent processing and QoS translation
- ✅ VNF placement and cluster selection
- ✅ Porch package generation and validation
- ✅ GitOps workflow execution
- ✅ Multi-cluster deployment coordination
- ✅ O2 interface integration (O2IMS/O2DMS)
- ✅ VNF operator lifecycle management

### Performance Tests
- ✅ Network latency validation (thesis targets)
- ✅ Throughput measurement and validation
- ✅ QoS parameter enforcement
- ✅ Resource utilization efficiency
- ✅ Deployment timing optimization
- ✅ Scalability under load

### Reliability Tests
- ✅ Cluster failure and recovery
- ✅ Network partition handling
- ✅ VNF failure detection and recovery
- ✅ Configuration drift correction
- ✅ Upgrade and rollback scenarios
- ✅ Stress testing and edge cases

## Validation Criteria

### Performance Compliance
- **Latency**: Within ±15% of thesis targets
- **Throughput**: Within ±20% of thesis targets
- **Deployment Time**: ≤ 10 minutes for E2E deployment
- **Success Rate**: ≥ 95% for all operations

### Quality Gates
- **Unit Test Coverage**: ≥ 80%
- **Integration Test Coverage**: All major workflows
- **Performance Regression**: No degradation > 10%
- **Error Rate**: ≤ 5% for all operations

### Compliance Reporting
Tests generate compliance reports showing:
- SLA adherence rates
- Performance trend analysis
- Bottleneck identification
- Optimization recommendations
- Resource efficiency metrics

## Contributing

### Adding New Tests
1. Follow existing test structure and naming conventions
2. Include comprehensive validation and assertions
3. Add performance measurement where applicable
4. Update this README with new test coverage
5. Ensure tests are deterministic and isolated

### Test Guidelines
- Use table-driven tests for multiple scenarios
- Include both positive and negative test cases
- Validate timing and performance requirements
- Provide clear test descriptions and failure messages
- Clean up resources after test completion

### Performance Testing
- Always measure and validate against thesis targets
- Include confidence intervals and statistical analysis
- Test under various load conditions
- Document performance baselines and regressions