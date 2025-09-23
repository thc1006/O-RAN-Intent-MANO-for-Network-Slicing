# Performance Optimization Guide for Sub-10-Minute Deployments

## Overview

This guide provides detailed strategies and configurations to achieve the target deployment time of less than 10 minutes for network slice orchestration in the O-RAN Intent-Based MANO system.

## Performance Timeline Breakdown

```
Target: < 10 minutes end-to-end deployment

0-2 min:  Intent Processing & Validation
2-4 min:  Package Generation & Customization
4-7 min:  Multi-Cluster Distribution
7-9 min:  Coordinated Deployment
9-10 min: Validation & Traffic Enablement
```

## 1. Intent Processing Optimization (0-2 minutes)

### 1.1 Fast NLP Processing

```yaml
# High-Performance NLP Service
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fast-nlp-processor
  namespace: nephio-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: fast-nlp-processor
  template:
    metadata:
      labels:
        app: fast-nlp-processor
    spec:
      nodeSelector:
        workload: "cpu-intensive"
      containers:
      - name: nlp-processor
        image: gcr.io/oran-mano/fast-nlp:v2.0.0
        resources:
          requests:
            cpu: "2000m"
            memory: "4Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"
        env:
        - name: MODEL_CACHE
          value: "enabled"
        - name: BATCH_SIZE
          value: "32"
        - name: MAX_SEQUENCE_LENGTH
          value: "512"
        - name: INFERENCE_TIMEOUT
          value: "30s"
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

### 1.2 Parallel Validation Pipeline

```yaml
# Parallel Validation Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: parallel-validation-config
  namespace: nephio-system
data:
  config.yaml: |
    validation:
      parallelism: 5
      timeout: "30s"

      validators:
        - name: "syntax-validator"
          image: "gcr.io/oran-mano/syntax-validator:v1.0.0"
          priority: 1
          timeout: "10s"

        - name: "qos-validator"
          image: "gcr.io/oran-mano/qos-validator:v1.0.0"
          priority: 2
          timeout: "15s"

        - name: "resource-validator"
          image: "gcr.io/oran-mano/resource-validator:v1.0.0"
          priority: 3
          timeout: "20s"

        - name: "security-validator"
          image: "gcr.io/oran-mano/security-validator:v1.0.0"
          priority: 2
          timeout: "25s"

        - name: "compliance-validator"
          image: "gcr.io/oran-mano/compliance-validator:v1.0.0"
          priority: 4
          timeout: "30s"

      optimization:
        cache_results: true
        skip_redundant: true
        early_termination: true
```

### 1.3 Intent Caching Strategy

```yaml
# Redis Cache for Intent Processing
apiVersion: apps/v1
kind: Deployment
metadata:
  name: intent-cache
  namespace: nephio-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: intent-cache
  template:
    metadata:
      labels:
        app: intent-cache
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        args:
        - redis-server
        - --maxmemory
        - 2gb
        - --maxmemory-policy
        - allkeys-lru
        - --save
        - ""
        - --appendonly
        - "no"
        ports:
        - containerPort: 6379
        resources:
          requests:
            cpu: "500m"
            memory: "2Gi"
          limits:
            cpu: "1000m"
            memory: "4Gi"
---
# Cache Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: intent-cache-config
  namespace: nephio-system
data:
  config.yaml: |
    cache:
      ttl: "1h"
      max_size: "1000"

      strategies:
        - type: "intent-templates"
          pattern: "intent:template:*"
          ttl: "24h"

        - type: "qos-profiles"
          pattern: "qos:profile:*"
          ttl: "12h"

        - type: "validation-results"
          pattern: "validation:*"
          ttl: "30m"

        - type: "resource-calculations"
          pattern: "resources:*"
          ttl: "2h"
```

## 2. Package Generation Optimization (2-4 minutes)

### 2.1 Parallel Package Generation

```yaml
# High-Performance Package Generator
apiVersion: apps/v1
kind: Deployment
metadata:
  name: parallel-package-generator
  namespace: nephio-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: parallel-package-generator
  template:
    metadata:
      labels:
        app: parallel-package-generator
    spec:
      nodeSelector:
        workload: "compute-intensive"
      containers:
      - name: generator
        image: gcr.io/oran-mano/parallel-generator:v1.0.0
        resources:
          requests:
            cpu: "4000m"
            memory: "8Gi"
          limits:
            cpu: "8000m"
            memory: "16Gi"
        env:
        - name: WORKER_THREADS
          value: "8"
        - name: BATCH_SIZE
          value: "5"
        - name: TEMPLATE_CACHE
          value: "enabled"
        - name: PARALLEL_FUNCTIONS
          value: "true"
        volumeMounts:
        - name: template-cache
          mountPath: /cache/templates
      volumes:
      - name: template-cache
        emptyDir:
          sizeLimit: "10Gi"
```

### 2.2 Template Pre-loading

```yaml
# Template Pre-loading Job
apiVersion: batch/v1
kind: Job
metadata:
  name: template-preloader
  namespace: nephio-system
spec:
  template:
    spec:
      containers:
      - name: preloader
        image: gcr.io/oran-mano/template-preloader:v1.0.0
        command:
        - /app/preload-templates
        args:
        - --source=https://github.com/oran-mano/nephio-package-catalog
        - --cache-dir=/cache/templates
        - --parallel=10
        - --compress=true
        env:
        - name: TEMPLATES_TO_PRELOAD
          value: "embb,urllc,miot,gnb,amf,smf,upf"
        volumeMounts:
        - name: template-cache
          mountPath: /cache/templates
        resources:
          requests:
            cpu: "2000m"
            memory: "4Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"
      volumes:
      - name: template-cache
        persistentVolumeClaim:
          claimName: template-cache-pvc
      restartPolicy: OnFailure
```

### 2.3 Function Pipeline Optimization

```yaml
# Optimized Function Pipeline
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-pipeline-config
  namespace: nephio-system
data:
  pipeline.yaml: |
    pipeline:
      execution:
        mode: "parallel"
        timeout: "2m"
        workers: 4

      functions:
        - name: "qos-injector"
          image: "gcr.io/oran-mano/qos-injector:v1.2.0"
          parallel: true
          timeout: "30s"
          resources:
            cpu: "500m"
            memory: "1Gi"

        - name: "site-customizer"
          image: "gcr.io/oran-mano/site-customizer:v1.0.0"
          parallel: true
          timeout: "45s"
          resources:
            cpu: "1000m"
            memory: "2Gi"

        - name: "resource-calculator"
          image: "gcr.io/oran-mano/resource-calculator:v1.1.0"
          depends_on: ["site-customizer"]
          timeout: "30s"
          resources:
            cpu: "500m"
            memory: "1Gi"

        - name: "validator"
          image: "gcr.io/oran-mano/validator:v1.3.0"
          depends_on: ["qos-injector", "resource-calculator"]
          timeout: "15s"
          resources:
            cpu: "500m"
            memory: "1Gi"

      optimization:
        function_cache: true
        result_memoization: true
        early_validation: true
```

## 3. Distribution Optimization (4-7 minutes)

### 3.1 Pre-warmed Git Repositories

```yaml
# Git Repository Pre-warming
apiVersion: batch/v1
kind: CronJob
metadata:
  name: git-repo-prewarmer
  namespace: config-management-system
spec:
  schedule: "*/10 * * * *"  # Every 10 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: git-prewarmer
            image: gcr.io/oran-mano/git-prewarmer:v1.0.0
            command:
            - /app/prewarm-repos
            args:
            - --repos=nephio-deployments,nephio-packages,nephio-blueprints
            - --parallel=5
            - --cache-dir=/cache/git
            env:
            - name: GIT_CLONE_DEPTH
              value: "1"
            - name: GIT_PARALLEL_JOBS
              value: "5"
            volumeMounts:
            - name: git-cache
              mountPath: /cache/git
            resources:
              requests:
                cpu: "1000m"
                memory: "2Gi"
              limits:
                cpu: "2000m"
                memory: "4Gi"
          volumes:
          - name: git-cache
            persistentVolumeClaim:
              claimName: git-cache-pvc
          restartPolicy: OnFailure
```

### 3.2 Optimized ConfigSync Configuration

```yaml
# High-Performance ConfigSync
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: optimized-root-sync
  namespace: config-management-system
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-deployments
    branch: main
    dir: /clusters/central/central-01
    auth: ssh
    secretRef:
      name: git-creds
    noSSLVerify: false

  # Performance optimizations
  override:
    gitSyncDepth: 1
    gitSyncWait: 5  # 5 seconds instead of default 15
    reconcilerPollingPeriod: 15s  # Faster polling

    # Resource optimization
    resources:
      gitSync:
        requests:
          cpu: "200m"
          memory: "512Mi"
        limits:
          cpu: "1000m"
          memory: "2Gi"
      reconciler:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "2000m"
          memory: "4Gi"
      admission-webhook:
        requests:
          cpu: "100m"
          memory: "256Mi"
        limits:
          cpu: "500m"
          memory: "1Gi"

  renderingRequired: true
  hydrationController:
    enabled: true
    resources:
      requests:
        cpu: "1000m"
        memory: "2Gi"
      limits:
        cpu: "4000m"
        memory: "8Gi"
```

### 3.3 Multi-Cluster Distribution Pipeline

```yaml
# Parallel Distribution Pipeline
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: parallel-distribution
  namespace: nephio-system
spec:
  entrypoint: distribute-packages
  templates:
  - name: distribute-packages
    dag:
      tasks:
      # Parallel preparation
      - name: prepare-central
        template: prepare-cluster
        arguments:
          parameters:
          - name: cluster-type
            value: "central"
          - name: packages
            value: "{{workflow.parameters.packages}}"

      - name: prepare-regional
        template: prepare-cluster
        arguments:
          parameters:
          - name: cluster-type
            value: "regional"
          - name: packages
            value: "{{workflow.parameters.packages}}"

      - name: prepare-edge
        template: prepare-cluster
        arguments:
          parameters:
          - name: cluster-type
            value: "edge"
          - name: packages
            value: "{{workflow.parameters.packages}}"

      # Coordinated distribution
      - name: distribute-to-central
        template: fast-distribution
        dependencies: [prepare-central]
        arguments:
          parameters:
          - name: target-clusters
            value: "central-01"

      - name: distribute-to-regional
        template: fast-distribution
        dependencies: [prepare-regional, distribute-to-central]
        arguments:
          parameters:
          - name: target-clusters
            value: "regional-01"

      - name: distribute-to-edge
        template: fast-distribution
        dependencies: [prepare-edge, distribute-to-regional]
        arguments:
          parameters:
          - name: target-clusters
            value: "edge-01,edge-02"

  - name: fast-distribution
    container:
      image: gcr.io/oran-mano/fast-distributor:v1.0.0
      command: ["/app/distribute"]
      args: ["--parallel", "true", "--timeout", "60s"]
      resources:
        requests:
          cpu: "1000m"
          memory: "2Gi"
        limits:
          cpu: "2000m"
          memory: "4Gi"
```

## 4. Deployment Coordination Optimization (7-9 minutes)

### 4.1 Resource Pre-warming

```yaml
# Resource Pre-warming DaemonSet
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: resource-prewarmer
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: resource-prewarmer
  template:
    metadata:
      labels:
        app: resource-prewarmer
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: prewarmer
        image: gcr.io/oran-mano/resource-prewarmer:v1.0.0
        securityContext:
          privileged: true
        command:
        - /app/prewarm-resources
        args:
        - --hugepages=4Gi
        - --cpu-isolation=true
        - --network-setup=true
        - --storage-prep=true
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        resources:
          requests:
            cpu: "100m"
            memory: "256Mi"
          limits:
            cpu: "500m"
            memory: "1Gi"
        volumeMounts:
        - name: proc
          mountPath: /host/proc
          readOnly: true
        - name: sys
          mountPath: /host/sys
          readOnly: true
      volumes:
      - name: proc
        hostPath:
          path: /proc
      - name: sys
        hostPath:
          path: /sys
```

### 4.2 Image Pre-pulling Strategy

```yaml
# Image Pre-puller Job
apiVersion: batch/v1
kind: Job
metadata:
  name: image-prepuller
  namespace: kube-system
spec:
  parallelism: 10
  template:
    spec:
      containers:
      - name: prepuller
        image: gcr.io/oran-mano/image-prepuller:v1.0.0
        command:
        - /app/prepull-images
        args:
        - --images-file=/etc/config/images.yaml
        - --parallel=5
        - --timeout=300s
        env:
        - name: DOCKER_CONFIG
          value: "/etc/docker"
        volumeMounts:
        - name: images-config
          mountPath: /etc/config
        - name: docker-config
          mountPath: /etc/docker
        resources:
          requests:
            cpu: "500m"
            memory: "1Gi"
          limits:
            cpu: "1000m"
            memory: "2Gi"
      volumes:
      - name: images-config
        configMap:
          name: prepull-images-config
      - name: docker-config
        secret:
          secretName: docker-registry-config
      restartPolicy: OnFailure
---
# Images to Pre-pull
apiVersion: v1
kind: ConfigMap
metadata:
  name: prepull-images-config
  namespace: kube-system
data:
  images.yaml: |
    images:
      ran_functions:
        - "gcr.io/oran-mano/gnb:v1.2.0"
        - "gcr.io/oran-mano/cu:v1.1.0"
        - "gcr.io/oran-mano/du:v1.1.0"

      core_functions:
        - "gcr.io/oran-mano/amf:v1.1.0"
        - "gcr.io/oran-mano/smf:v1.1.0"
        - "gcr.io/oran-mano/upf:v1.3.0"
        - "gcr.io/oran-mano/nrf:v1.0.0"

      management:
        - "gcr.io/oran-mano/orchestrator:v1.0.0"
        - "gcr.io/oran-mano/controller:v1.0.0"

      networking:
        - "quay.io/k8scni/multus:v3.8"
        - "ghcr.io/k8snetworkplumbingwg/sriov-network-operator:v1.2.0"
```

### 4.3 Fast Deployment Coordination

```yaml
# Fast Deployment Controller
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fast-deployment-controller
  namespace: nephio-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: fast-deployment-controller
  template:
    metadata:
      labels:
        app: fast-deployment-controller
    spec:
      containers:
      - name: controller
        image: gcr.io/oran-mano/fast-deployment-controller:v1.0.0
        args:
        - --concurrent-reconciles=10
        - --max-concurrent-deployments=20
        - --deployment-timeout=300s
        - --health-check-interval=5s
        env:
        - name: ENABLE_FAST_PROVISIONING
          value: "true"
        - name: SKIP_SLOW_VALIDATIONS
          value: "true"
        - name: PARALLEL_DEPLOYMENT
          value: "true"
        resources:
          requests:
            cpu: "1000m"
            memory: "2Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 9443
          name: webhook-server
          protocol: TCP
```

## 5. Validation and Traffic Enablement (9-10 minutes)

### 5.1 Fast Health Checks

```yaml
# Fast Health Check Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: fast-health-check-config
  namespace: nephio-system
data:
  config.yaml: |
    healthChecks:
      parallel: true
      timeout: "30s"

      checks:
        - name: "pod-readiness"
          type: "kubernetes"
          interval: "2s"
          timeout: "30s"

        - name: "service-connectivity"
          type: "network"
          interval: "5s"
          timeout: "10s"

        - name: "interface-status"
          type: "custom"
          command: "/app/check-interfaces"
          interval: "3s"
          timeout: "15s"

        - name: "qos-verification"
          type: "custom"
          command: "/app/verify-qos"
          interval: "5s"
          timeout: "20s"

      optimization:
        early_success: true
        fail_fast: true
        cache_results: true
```

### 5.2 Automated Traffic Enablement

```yaml
# Traffic Enablement Controller
apiVersion: apps/v1
kind: Deployment
metadata:
  name: traffic-enablement-controller
  namespace: nephio-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: traffic-enablement-controller
  template:
    metadata:
      labels:
        app: traffic-enablement-controller
    spec:
      containers:
      - name: controller
        image: gcr.io/oran-mano/traffic-controller:v1.0.0
        args:
        - --enable-fast-mode=true
        - --validation-timeout=30s
        - --traffic-ramp-duration=30s
        env:
        - name: AUTO_TRAFFIC_ENABLE
          value: "true"
        - name: CANARY_PERCENTAGE
          value: "10"
        - name: RAMP_UP_DURATION
          value: "30s"
        resources:
          requests:
            cpu: "500m"
            memory: "1Gi"
          limits:
            cpu: "1000m"
            memory: "2Gi"
```

## 6. System-Wide Performance Tuning

### 6.1 Kubernetes API Server Optimization

```yaml
# API Server Performance Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: apiserver-performance-config
  namespace: kube-system
data:
  config.yaml: |
    apiServer:
      # Increase API server throughput
      max-requests-inflight: 800
      max-mutating-requests-inflight: 400

      # Optimize watch cache
      watch-cache-sizes: "deployments.apps#1000,services#1000,networkfunctions.workload.nephio.org#500"

      # Reduce latency
      request-timeout: "60s"
      shutdown-delay-duration: "10s"

      # Optimize etcd
      etcd-compaction-interval: "5m"
      etcd-servers-overrides: "/networkfunctions=https://etcd-nf.kube-system.svc.cluster.local:2379"
```

### 6.2 Container Runtime Optimization

```yaml
# ContainerD Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: containerd-performance-config
  namespace: kube-system
data:
  config.toml: |
    [plugins."io.containerd.grpc.v1.cri"]
      max_concurrent_downloads = 10
      max_container_log_line_size = 16384

    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "overlayfs"
      default_runtime_name = "runc"

    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
      runtime_type = "io.containerd.runc.v2"

    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
      SystemdCgroup = true

    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."gcr.io"]
          endpoint = ["https://gcr.io", "https://mirror.gcr.io"]
```

### 6.3 Network Performance Tuning

```yaml
# Network Performance Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: network-performance-config
  namespace: kube-system
data:
  sysctl.conf: |
    # Network performance tuning
    net.core.rmem_max = 134217728
    net.core.wmem_max = 134217728
    net.ipv4.tcp_rmem = 4096 87380 134217728
    net.ipv4.tcp_wmem = 4096 65536 134217728
    net.core.netdev_max_backlog = 30000
    net.ipv4.tcp_congestion_control = bbr
    net.ipv4.tcp_slow_start_after_idle = 0

    # Reduce context switches
    net.core.busy_poll = 50
    net.core.busy_read = 50

    # Optimize for latency
    net.ipv4.tcp_low_latency = 1
    net.ipv4.tcp_timestamps = 0
```

## 7. Monitoring and Observability for Performance

### 7.1 Performance Metrics Collection

```yaml
# Performance Metrics ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: performance-metrics-config
  namespace: monitoring
data:
  metrics.yaml: |
    metrics:
      deployment_pipeline:
        - name: "intent_processing_duration"
          description: "Time spent processing intents"
          type: "histogram"
          buckets: [0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0]

        - name: "package_generation_duration"
          description: "Time spent generating packages"
          type: "histogram"
          buckets: [1.0, 5.0, 10.0, 30.0, 60.0, 120.0]

        - name: "distribution_duration"
          description: "Time spent distributing packages"
          type: "histogram"
          buckets: [10.0, 30.0, 60.0, 120.0, 180.0]

        - name: "deployment_duration"
          description: "Time spent on deployment"
          type: "histogram"
          buckets: [30.0, 60.0, 120.0, 300.0, 600.0]

        - name: "end_to_end_duration"
          description: "Total deployment time"
          type: "histogram"
          buckets: [60.0, 120.0, 300.0, 480.0, 600.0, 900.0]

      performance_targets:
        intent_processing: "120s"
        package_generation: "120s"
        distribution: "180s"
        deployment: "120s"
        validation: "60s"
        total: "600s"  # 10 minutes
```

### 7.2 Performance Dashboard

```yaml
# Grafana Performance Dashboard
apiVersion: v1
kind: ConfigMap
metadata:
  name: performance-dashboard
  namespace: monitoring
data:
  dashboard.json: |
    {
      "dashboard": {
        "title": "O-RAN MANO Performance Dashboard",
        "panels": [
          {
            "title": "Deployment Time Breakdown",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, intent_processing_duration_bucket)",
                "legendFormat": "Intent Processing (P95)"
              },
              {
                "expr": "histogram_quantile(0.95, package_generation_duration_bucket)",
                "legendFormat": "Package Generation (P95)"
              },
              {
                "expr": "histogram_quantile(0.95, distribution_duration_bucket)",
                "legendFormat": "Distribution (P95)"
              },
              {
                "expr": "histogram_quantile(0.95, deployment_duration_bucket)",
                "legendFormat": "Deployment (P95)"
              }
            ]
          },
          {
            "title": "Performance vs Target",
            "type": "stat",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, end_to_end_duration_bucket)",
                "legendFormat": "Current P95"
              }
            ],
            "thresholds": [
              {"color": "green", "value": 0},
              {"color": "yellow", "value": 480},
              {"color": "red", "value": 600}
            ]
          }
        ]
      }
    }
```

## 8. Troubleshooting Performance Issues

### 8.1 Common Performance Bottlenecks

1. **Intent Processing Delays**
   - Check NLP model loading time
   - Verify template cache hit rate
   - Monitor validation parallelism

2. **Package Generation Slowdowns**
   - Examine function pipeline execution
   - Check resource calculations efficiency
   - Verify template pre-loading

3. **Distribution Latencies**
   - Monitor git repository performance
   - Check ConfigSync reconciliation times
   - Verify network bandwidth

4. **Deployment Coordination Issues**
   - Check resource availability
   - Monitor image pull times
   - Verify cluster readiness

### 8.2 Performance Testing Tools

```yaml
# Performance Test Suite
apiVersion: batch/v1
kind: Job
metadata:
  name: performance-test-suite
  namespace: nephio-system
spec:
  template:
    spec:
      containers:
      - name: performance-tester
        image: gcr.io/oran-mano/performance-tester:v1.0.0
        command:
        - /app/run-performance-tests
        args:
        - --target-time=600s
        - --parallel-deployments=5
        - --slice-types=embb,urllc,miot
        - --iterations=10
        env:
        - name: TEST_MODE
          value: "performance"
        - name: METRICS_ENDPOINT
          value: "http://prometheus.monitoring.svc.cluster.local:9090"
        resources:
          requests:
            cpu: "1000m"
            memory: "2Gi"
          limits:
            cpu: "2000m"
            memory: "4Gi"
      restartPolicy: OnFailure
```

## 9. Continuous Performance Optimization

### 9.1 Automated Performance Tuning

```yaml
# Performance Optimization Controller
apiVersion: apps/v1
kind: Deployment
metadata:
  name: performance-optimizer
  namespace: nephio-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: performance-optimizer
  template:
    metadata:
      labels:
        app: performance-optimizer
    spec:
      containers:
      - name: optimizer
        image: gcr.io/oran-mano/performance-optimizer:v1.0.0
        args:
        - --optimization-interval=300s
        - --target-deployment-time=600s
        - --auto-tune=true
        env:
        - name: ENABLE_AUTO_SCALING
          value: "true"
        - name: ENABLE_RESOURCE_OPTIMIZATION
          value: "true"
        - name: ENABLE_CACHE_TUNING
          value: "true"
        resources:
          requests:
            cpu: "500m"
            memory: "1Gi"
          limits:
            cpu: "1000m"
            memory: "2Gi"
```

This comprehensive performance optimization guide provides the foundation for achieving sub-10-minute deployment times in the O-RAN Intent-Based MANO system through systematic optimization of each deployment phase, infrastructure tuning, and continuous performance monitoring.