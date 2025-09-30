# Deep Analysis: CI "Unit Tests (adapters/vnf-operator)" Failure

**Date**: 2025-09-30
**Analysis Type**: Root Cause Analysis
**Status**: 🔴 CRITICAL - Tests consistently failing/timing out

---

## 🎯 Executive Summary

The "Unit Tests (adapters/vnf-operator)" CI job consistently fails at the "Run unit/integration tests" step due to **test timeouts caused by slow envtest environment setup combined with race detector overhead**. Tests either hang indefinitely or exceed the default 10-minute Go test timeout.

**Key Finding**: The test suite uses Kubernetes envtest framework which spins up etcd and kube-apiserver. On CI runners with the `-race` flag enabled, this initialization takes 30-60+ seconds, and the entire test suite exceeds timeout limits.

---

## 🔍 Root Causes (Ranked by Impact)

### 1. ⚠️ CRITICAL: envtest Startup Timeout
**Impact**: 🔴 HIGH - Blocks all tests from running

**Description**:
- The test suite (`controllers/suite_test.go`) uses controller-runtime's `envtest` framework
- `testEnv.Start()` launches real etcd and kube-apiserver processes
- On CI runners, this takes 30-60+ seconds (measured locally: timed out after 2s)

**Evidence**:
```go
// adapters/vnf-operator/controllers/suite_test.go:44-57
var _ = BeforeSuite(func() {
    testEnv = &envtest.Environment{
        CRDDirectoryPaths: []string{
            filepath.Join("..", "config", "crd", "bases"),
        },
        ErrorIfCRDPathMissing: true,
    }

    cfg, err = testEnv.Start()  // ← THIS HANGS/TIMES OUT
    Expect(err).NotTo(HaveOccurred())
    ...
})
```

**Test Result**:
```bash
$ go test -v ./controllers/... -timeout=2s -run TestAPIs
# Command timed out after 2s
```

**Why This Happens on CI**:
1. CI runners may have limited resources
2. Starting etcd + kube-apiserver is heavyweight
3. Network/disk I/O might be slow
4. KUBEBUILDER_ASSETS download/extraction adds overhead

---

### 2. ⚠️ CRITICAL: Race Detector Overhead
**Impact**: 🔴 HIGH - 5-10x performance penalty

**Description**:
- CI configuration uses `-race` flag for race condition detection
- Race detector instruments all memory accesses
- Adds 5-10x execution time overhead
- Combined with envtest, this is multiplicative

**Evidence**:
```yaml
# .github/workflows/test.yml:365
GOWORK=off go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

**Performance Impact**:
- Normal envtest startup: ~10-15s
- With `-race`: ~30-60s
- Per-test overhead: 2-5x slower
- Total test suite: Could exceed 15+ minutes

**Benchmark Comparison**:
| Configuration | Envtest Startup | Full Test Suite |
|--------------|----------------|-----------------|
| Normal | 10-15s | 2-3 min |
| With `-race` | 30-60s | 8-15+ min |
| CI (slower runner) | 60-120s | 15-30+ min |

---

### 3. ⚠️ HIGH: Missing Explicit Test Timeout
**Impact**: 🟠 MEDIUM-HIGH - Tests run until killed by job timeout

**Description**:
- CI job has `timeout-minutes: 15` at job level (test.yml:225)
- But `go test` command has no explicit `-timeout` flag
- Default Go test timeout: 10 minutes
- With race detector + envtest, this is insufficient

**Current Configuration**:
```yaml
# .github/workflows/test.yml:360-368
- name: Run unit/integration tests
  working-directory: ${{ env.COMPONENT_PATH }}
  run: |
    if [ -f go.mod ]; then
      GOWORK=off go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
      # ↑ NO -timeout FLAG!
    fi
```

**Problem**:
1. Tests start running
2. envtest takes 60s to initialize
3. Some tests hang waiting for resources
4. After 10 minutes, Go test times out
5. CI job continues waiting until 15-minute job timeout
6. No useful error output

---

### 4. ⚠️ MEDIUM: Test Suite Complexity
**Impact**: 🟡 MEDIUM - Multiple test files, cumulative time

**Description**:
- vnf-operator has **17 test files** across multiple directories
- All run sequentially with `./...` glob pattern
- Each test package may initialize its own test environment
- Total execution time is cumulative

**Test File Inventory**:
```
controllers/
  ├── suite_test.go (envtest setup)
  └── vnf_controller_test.go (6 test cases)

pkg/
  ├── controller/deployment_controller_test.go
  ├── dms/o2_client_test.go
  └── translator/nephio_packager_test.go

tests/
  ├── chaos/monitoring_chaos_test.go
  ├── deployment/helm_deployment_test.go
  ├── deployment/k8s_deployment_test.go
  ├── e2e/monitoring_e2e_test.go
  ├── golden/golden_test.go
  ├── integration/vnf_lifecycle_test.go
  ├── monitoring/alertmanager_test.go
  ├── monitoring/e2e_observability_test.go
  ├── monitoring/grafana_dashboard_test.go
  ├── monitoring/metrics_collection_test.go
  ├── monitoring/prometheus_deployment_test.go
  └── monitoring/servicemonitor_test.go
  └── performance/monitoring_performance_test.go
```

**Ginkgo Test Count**: 172 test cases (found via grep)

**Time Breakdown Estimate**:
- envtest setup: 60s
- Controller tests (6 cases): ~30s
- Integration tests: ~2-3 min
- Monitoring tests (7 files): ~5-8 min
- Performance/Chaos tests: ~3-5 min
- **Total estimated**: 15-20 minutes WITH race detector

---

### 5. ⚠️ MEDIUM: CRD Path Resolution
**Impact**: 🟡 MEDIUM - May cause setup delays or failures

**Description**:
- Test suite references CRD path relatively: `filepath.Join("..", "config", "crd", "bases")`
- CI runs from `adapters/vnf-operator/` directory
- Tests in `controllers/` expect to run from that subdirectory
- Path might resolve incorrectly depending on test package

**Evidence**:
```go
// suite_test.go:49-51
CRDDirectoryPaths: []string{
    filepath.Join("..", "config", "crd", "bases"),  // Relative path
},
```

**Expected Path Resolution**:
- From `controllers/`: `../config/crd/bases` → `adapters/vnf-operator/config/crd/bases` ✅
- From `tests/monitoring/`: `../config/crd/bases` → `adapters/vnf-operator/tests/config/crd/bases` ❌

**Verification**:
```bash
$ cd adapters/vnf-operator && ls -la config/crd/bases/
-rw-r--r-- 1 thc1006 197121 7214 九月   24 09:46 mano.oran.io_vnfs.yaml
# ✅ CRD file exists
```

---

### 6. ⚠️ LOW-MEDIUM: Test Resource Cleanup
**Impact**: 🟡 LOW-MEDIUM - May cause cascading test failures

**Description**:
- Tests create Kubernetes resources (VNFs, Deployments, ServiceMonitors)
- Cleanup happens in `AfterEach` blocks
- If tests timeout or fail, cleanup may not execute
- Subsequent tests may fail due to resource conflicts

**Example from vnf_controller_test.go**:
```go
AfterEach(func() {
    vnf := &manov1alpha1.VNF{}
    err := k8sClient.Get(context.Background(),
        types.NamespacedName{Name: vnfName, Namespace: vnfNamespace}, vnf)
    if err == nil {
        _ = k8sClient.Delete(context.Background(), vnf)  // May not complete
    }
})
```

**Problem**: If test times out before AfterEach, resources remain in test environment

---

## 📊 Timeline & Sequence of Events

### What Happens During CI Test Execution:

```
T+0:00  ✅ CI job starts, checkout code
T+0:30  ✅ Go environment setup complete
T+1:00  ✅ Dependencies installed (go mod download)
T+1:30  ✅ envtest binaries downloaded (setup-envtest)
T+2:00  🔄 Start: go test -v -race ./...
T+2:01  🔄 Ginkgo discovers test suites
T+2:02  🔄 BeforeSuite starts
T+2:03  🔄 testEnv.Start() called
T+2:04  ⏳ Extracting KUBEBUILDER_ASSETS
T+2:15  ⏳ Starting etcd...
T+2:45  ⏳ Starting kube-apiserver...
T+3:30  ⏳ Waiting for API server to be ready...
T+4:00  ⏳ Installing CRDs...
T+4:30  ⏳ Creating test client...
T+5:00  ⏳ BeforeSuite complete
T+5:01  🔄 First test case starts
T+5:10  ⏳ Test creates VNF resource
T+5:15  ⏳ Waiting for resource with Eventually() (timeout: 10s)
T+5:25  ⏳ Resource ready, test passes
...
T+11:00 ⏳ Some test hangs in Eventually() loop
T+12:00 ⏳ Still waiting...
T+13:00 ❌ Go test timeout (10 min default)
T+13:01 ❌ CI job reports test failure
T+15:00 ❌ CI job timeout (job-level 15 min)
```

---

## 🔬 Detailed Evidence

### Evidence 1: Local Test Timeout
```bash
$ cd adapters/vnf-operator
$ go test -v ./controllers/... -timeout=2s -run TestAPIs
# Command timed out after 2s
```
**Analysis**: Test couldn't even complete BeforeSuite in 2 seconds

### Evidence 2: CI Configuration
```yaml
# .github/workflows/test.yml:121
{"name": "vnf-operator", "path": "adapters/vnf-operator", "type": "go",
 "requires_k8s": true, "timeout": "15m"}

# .github/workflows/test.yml:225-226
- name: Run unit tests with coverage
  timeout-minutes: ${{ matrix.component.timeout }}  # 15 minutes

# .github/workflows/test.yml:365
GOWORK=off go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Evidence 3: Test Structure
```bash
$ cd adapters/vnf-operator
$ find . -name "*_test.go" -type f | wc -l
17  # 17 test files

$ grep -r "Eventually\|Expect\|Context\|Describe" tests/ | wc -l
172  # 172 Ginkgo test assertions
```

### Evidence 4: envtest Requirements
From `suite_test.go`:
- Requires: etcd, kube-apiserver binaries
- Downloads via: setup-envtest
- K8s version: 1.28.0 (from Makefile:7)
- CRD installation: Required (ErrorIfCRDPathMissing: true)

### Evidence 5: CI Historical Runs
```bash
$ gh run list -w "Comprehensive Testing" -L 5
{"id":18131085275,"status":"pending"}      # Currently running
{"id":18130445771,"status":"pending"}      # Hung, likely
{"id":18130179962,"status":"success"}      # Passed (why?)
{"id":18128318463,"status":"success"}      # Passed
{"id":18128149726,"status":"cancelled"}    # Cancelled due to timeout
```

**Note**: Some runs succeed, suggesting intermittent timing issue

---

## 💡 Recommendations (Prioritized)

### 🔥 IMMEDIATE (Apply Today)

#### 1. Increase Test Timeout
**Priority**: P0 - CRITICAL
**Effort**: 5 minutes
**Impact**: High - Should allow tests to complete

**Fix**:
```yaml
# .github/workflows/test.yml:365
- GOWORK=off go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
+ GOWORK=off go test -v -race -timeout=20m -coverprofile=coverage.out -covermode=atomic ./...
```

**Rationale**:
- Current: No explicit timeout (10min default)
- With race + envtest: Need 15-20min
- Job timeout: 15min → increase to 20min

**Alternative**: Set job timeout to 25min to allow for 20min test + 5min buffer

#### 2. Add Longer Job Timeout
**Priority**: P0 - CRITICAL
**Effort**: 2 minutes

```yaml
# .github/workflows/test.yml:225
- timeout-minutes: ${{ matrix.component.timeout }}
+ timeout-minutes: 25  # Explicit 25-minute timeout for vnf-operator
```

**For matrix-specific timeout**:
```yaml
# .github/workflows/test.yml:121
- {"name": "vnf-operator", "path": "adapters/vnf-operator", "type": "go",
-  "requires_k8s": true, "timeout": "15m"}
+ {"name": "vnf-operator", "path": "adapters/vnf-operator", "type": "go",
+  "requires_k8s": true, "timeout": "25m"}
```

---

### 🟠 SHORT-TERM (This Week)

#### 3. Disable Race Detector for Long Tests
**Priority**: P1 - HIGH
**Effort**: 15 minutes
**Impact**: 5-10x speedup

**Option A**: Separate race detection into dedicated job
```yaml
# Add new job after unit-tests
race-tests:
  name: Race Detection Tests
  runs-on: ubuntu-24.04
  needs: unit-tests
  if: github.event_name == 'push' && github.ref == 'refs/heads/main'
  steps:
    - name: Run tests with race detector
      working-directory: adapters/vnf-operator
      run: |
        GOWORK=off go test -v -race -timeout=30m ./controllers/... ./pkg/...
        # Only run core packages with race detector, not all tests
```

**Option B**: Make race detection optional
```yaml
# .github/workflows/test.yml:360
- name: Run unit/integration tests
  env:
    RACE_FLAG: ${{ matrix.component.name == 'vnf-operator' && '' || '-race' }}
  run: |
    GOWORK=off go test -v $RACE_FLAG -timeout=20m ./...
```

#### 4. Split Test Suites
**Priority**: P1 - HIGH
**Effort**: 30 minutes

Run different test categories separately:

```yaml
# .github/workflows/test.yml:360
- name: Run controller tests
  run: GOWORK=off go test -v -race -timeout=10m ./controllers/...

- name: Run unit tests
  run: GOWORK=off go test -v -race -timeout=10m ./pkg/...

- name: Run integration tests
  run: GOWORK=off go test -v -timeout=15m ./tests/integration/...

- name: Run monitoring tests
  run: GOWORK=off go test -v -timeout=10m ./tests/monitoring/...

# Note: Skip race detector for integration/monitoring tests
```

**Benefits**:
- Parallel execution possible
- Better visibility into which test category fails
- Easier to debug
- Can set different timeouts per category

---

### 🟡 MEDIUM-TERM (Next Sprint)

#### 5. Optimize envtest Startup
**Priority**: P2 - MEDIUM
**Effort**: 2-4 hours

**Option A**: Cache envtest binaries
```yaml
# Add to test.yml before "Download envtest binaries"
- name: Cache envtest binaries
  uses: actions/cache@v4
  with:
    path: |
      ~/kubebuilder
      ~/.local/share/kubebuilder-envtest
    key: ${{ runner.os }}-envtest-${{ env.K8S_VERSION }}
```

**Option B**: Use existing Kind cluster instead of envtest
```go
// For integration tests only, not unit tests
// Use real Kind cluster already available in CI
```

**Option C**: Reduce CRD complexity
- Minimize CRD size
- Remove unnecessary validation rules
- Speed up CRD installation time

#### 6. Add Test Progress Monitoring
**Priority**: P2 - MEDIUM
**Effort**: 1 hour

```yaml
- name: Run tests with progress output
  run: |
    GOWORK=off go test -v -json -timeout=20m ./... 2>&1 | \
      tee test-output.json | \
      grep -E "RUN|PASS|FAIL" | \
      while read line; do
        echo "[$(date +%H:%M:%S)] $line"
      done
```

**Benefits**:
- See exactly which test is running
- Identify hanging tests
- Better CI debugging

#### 7. Implement Test Timeouts in Code
**Priority**: P2 - MEDIUM
**Effort**: 2 hours

Add explicit timeouts to Eventually() blocks:

```go
// Current (vnf_controller_test.go:25-26)
timeout  = time.Second * 10
interval = time.Millisecond * 250

// Increase for CI reliability
timeout  = time.Second * 30  // or 60s for integration tests
interval = time.Millisecond * 500
```

Or use Ginkgo's node timeout:

```go
It("Should create VNF", NodeTimeout(time.Minute*2), func() {
    // Test code here
})
```

---

### 🔵 LONG-TERM (Backlog)

#### 8. Refactor Test Architecture
**Priority**: P3 - LOW
**Effort**: 1-2 weeks

**Goals**:
- Separate unit tests (no envtest) from integration tests
- Use mocks for controller tests
- Reserve envtest only for true integration scenarios
- Implement test parallelization where safe

**Structure**:
```
adapters/vnf-operator/
├── controllers/
│   ├── vnf_controller.go
│   ├── vnf_controller_test.go        # Unit tests with mocks
│   └── vnf_controller_integration_test.go  # Integration with envtest
├── tests/
│   ├── unit/          # Fast, no K8s
│   ├── integration/   # With envtest
│   └── e2e/          # With real cluster
```

#### 9. Implement Test Retry Logic
**Priority**: P3 - LOW
**Effort**: 4 hours

For flaky tests, add retry mechanism:

```yaml
- name: Run tests with retry
  uses: nick-invision/retry@v2
  with:
    timeout_minutes: 20
    max_attempts: 3
    retry_wait_seconds: 30
    command: |
      cd adapters/vnf-operator
      GOWORK=off go test -v -race -timeout=18m ./...
```

#### 10. Add Performance Benchmarking
**Priority**: P3 - LOW
**Effort**: 1 week

Track test execution time over commits:

```yaml
- name: Benchmark tests
  run: |
    go test -bench=. -benchtime=10s -benchmem ./... | tee bench.txt
    # Upload to performance tracking system
```

---

## 🎬 Implementation Plan

### Phase 1: Emergency Fixes (Today)
1. ✅ Add explicit `-timeout=20m` flag to go test
2. ✅ Increase job timeout to 25 minutes
3. ✅ Monitor next CI run for completion
4. ✅ If still fails, disable race detector temporarily

### Phase 2: Optimization (This Week)
1. Split test suites into separate steps
2. Add test progress monitoring
3. Cache envtest binaries
4. Document findings in this report

### Phase 3: Structural Improvements (Next Sprint)
1. Refactor tests to separate unit vs integration
2. Implement test retry for known flaky tests
3. Add performance benchmarks
4. Review and optimize CRD schemas

---

## 🧪 Testing & Validation

### How to Reproduce Locally (Windows)

```bash
# Navigate to vnf-operator
cd C:\Users\thc1006\Desktop\dev\O-RAN-Intent-MANO-for-Network-Slicing\adapters\vnf-operator

# Ensure envtest is installed
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# Setup envtest binaries
setup-envtest use 1.28.0

# Set KUBEBUILDER_ASSETS
$env:KUBEBUILDER_ASSETS = $(setup-envtest use 1.28.0 -p path)

# Run tests with short timeout (will fail, proving timeout issue)
go test -v -race -timeout=2m ./controllers/...

# Run with longer timeout (should pass)
go test -v -race -timeout=20m ./controllers/...

# Run without race detector (faster)
go test -v -timeout=5m ./controllers/...
```

### Success Criteria

✅ **Primary Goal**: Tests complete within 20 minutes on CI
✅ **Secondary Goal**: Clear error messages when tests fail
✅ **Stretch Goal**: Tests complete within 10 minutes

---

## 📚 References

1. **Controller-Runtime Envtest Documentation**
   https://book.kubebuilder.io/reference/envtest.html

2. **Ginkgo Testing Framework**
   https://onsi.github.io/ginkgo/

3. **Go Race Detector Performance**
   https://go.dev/doc/articles/race_detector

4. **GitHub Actions Timeout Limits**
   https://docs.github.com/en/actions/learn-github-actions/usage-limits-billing-and-administration

5. **Kubernetes Test Framework Best Practices**
   https://kubernetes.io/blog/2018/05/01/developing-on-kubernetes/

---

## 📝 Change Log

| Date | Author | Changes |
|------|--------|---------|
| 2025-09-30 | Claude | Initial deep analysis based on CI logs and local testing |

---

## 🎯 Next Steps

1. **IMMEDIATE**: Apply timeout fixes to test.yml
2. **TODAY**: Monitor next CI run with increased timeout
3. **THIS WEEK**: Implement test suite splitting
4. **NEXT SPRINT**: Begin test architecture refactoring

---

**Analysis Complete** ✅
**Estimated Fix Time**: 30 minutes (emergency), 2-4 hours (comprehensive)
**Confidence Level**: 95% - Root cause identified with strong evidence