# O-RAN Monitoring Stack Scaling Guide

This guide provides comprehensive instructions for scaling the O-RAN monitoring stack based on workload demands and growth patterns.

## Overview

Scaling the monitoring stack involves both vertical scaling (increasing resources) and horizontal scaling (adding replicas or federation) to handle increased metrics volume, query load, and user demands.

## Capacity Planning

### Metrics Volume Estimation

| Component | Metrics/Second | Storage/Day | CPU/Core | Memory/GB |
|-----------|---------------|-------------|----------|-----------|
| Kubernetes | 1000-5000 | 2-10GB | 0.5-1.0 | 2-4 |
| O-RAN NLP | 500-2000 | 1-5GB | 0.2-0.5 | 1-2 |
| O-RAN Orchestrator | 1000-3000 | 2-8GB | 0.3-0.8 | 1-3 |
| O-RAN RAN | 2000-8000 | 4-20GB | 0.5-2.0 | 2-6 |
| O-RAN CN | 1500-6000 | 3-15GB | 0.4-1.5 | 2-5 |
| O-RAN TN | 1000-4000 | 2-10GB | 0.3-1.0 | 1-4 |

### Growth Projection Formula

```bash
# Calculate required resources
metrics_per_second = base_metrics * (1 + growth_rate)^years
storage_per_day_gb = metrics_per_second * 86400 * bytes_per_metric / 1024^3
prometheus_memory_gb = active_series * 2KB / 1024^3
prometheus_cpu_cores = query_rate * 0.01
```

### Resource Monitoring

```bash
# Monitor current resource usage
kubectl top pods -n oran-monitoring
kubectl top nodes

# Get detailed metrics
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &

# Query resource usage
curl "http://localhost:9090/api/v1/query?query=rate(prometheus_tsdb_head_samples_appended_total[5m])"
curl "http://localhost:9090/api/v1/query?query=prometheus_tsdb_head_series"
curl "http://localhost:9090/api/v1/query?query=rate(prometheus_http_requests_total[5m])"
```

## Vertical Scaling

### Prometheus Scaling

#### CPU Scaling

```yaml
# prometheus-cpu-scaling.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: oran-monitoring
spec:
  template:
    spec:
      containers:
      - name: prometheus
        resources:
          requests:
            cpu: 1000m      # Increased from 500m
            memory: 4Gi
          limits:
            cpu: 4000m      # Increased from 2000m
            memory: 8Gi
        args:
        - '--query.max-concurrency=100'  # Increased from 50
        - '--query.max-samples=100000000'  # Increased from 50M
```

#### Memory Scaling

```yaml
# prometheus-memory-scaling.yaml
spec:
  template:
    spec:
      containers:
      - name: prometheus
        resources:
          requests:
            memory: 8Gi     # Increased from 4Gi
          limits:
            memory: 16Gi    # Increased from 8Gi
        args:
        - '--storage.tsdb.head-chunks-write-queue-size=20000'  # Increased
```

#### Storage Scaling

```bash
# Expand Prometheus PVC (if storage class supports it)
kubectl patch pvc prometheus-storage -n oran-monitoring -p '{"spec":{"resources":{"requests":{"storage":"500Gi"}}}}'

# Monitor expansion
kubectl get pvc prometheus-storage -n oran-monitoring -w

# Alternative: Create new larger PVC and migrate
kubectl apply -f - << EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: prometheus-storage-large
  namespace: oran-monitoring
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
  storageClassName: fast-ssd
EOF
```

### Grafana Scaling

#### Resource Scaling

```yaml
# grafana-scaling.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: oran-monitoring
spec:
  replicas: 3  # Horizontal scaling
  template:
    spec:
      containers:
      - name: grafana
        resources:
          requests:
            cpu: 500m       # Increased from 250m
            memory: 2Gi     # Increased from 1Gi
          limits:
            cpu: 2000m      # Increased from 1000m
            memory: 4Gi     # Increased from 2Gi
        env:
        - name: GF_DATABASE_MAX_OPEN_CONN
          value: "20"
        - name: GF_DATABASE_MAX_IDLE_CONN
          value: "10"
```

### AlertManager Scaling

```yaml
# alertmanager-scaling.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager
  namespace: oran-monitoring
spec:
  replicas: 3  # High availability
  template:
    spec:
      containers:
      - name: alertmanager
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 2Gi
        args:
        - '--cluster.listen-address=0.0.0.0:9094'
        - '--cluster.peer=alertmanager-0.alertmanager:9094'
        - '--cluster.peer=alertmanager-1.alertmanager:9094'
        - '--cluster.peer=alertmanager-2.alertmanager:9094'
```

## Horizontal Scaling

### Prometheus Federation

#### High-Level Federation Setup

```yaml
# prometheus-federation.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-global
  namespace: oran-monitoring
spec:
  template:
    spec:
      containers:
      - name: prometheus
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.retention.time=90d'  # Longer retention for global
        - '--storage.tsdb.retention.size=200GB'
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-global-config
  namespace: oran-monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 30s
      evaluation_interval: 30s

    rule_files:
    - "/etc/prometheus/rules/*.yml"

    scrape_configs:
    # Federate from regional Prometheus instances
    - job_name: 'federate-region-1'
      scrape_interval: 15s
      honor_labels: true
      metrics_path: '/federate'
      params:
        'match[]':
        - '{job=~"oran-.*"}'
        - '{__name__=~"oran:.*"}'  # Recording rules
      static_configs:
      - targets:
        - 'prometheus-region-1:9090'

    - job_name: 'federate-region-2'
      scrape_interval: 15s
      honor_labels: true
      metrics_path: '/federate'
      params:
        'match[]':
        - '{job=~"oran-.*"}'
        - '{__name__=~"oran:.*"}'
      static_configs:
      - targets:
        - 'prometheus-region-2:9090'
```

#### Regional Prometheus Configuration

```yaml
# prometheus-regional-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-region-1-config
  namespace: oran-monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
      external_labels:
        region: 'region-1'
        cluster: 'oran-intent-mano'

    rule_files:
    - "/etc/prometheus/rules/*.yml"

    scrape_configs:
    # Region-specific O-RAN components
    - job_name: 'oran-nlp-region-1'
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
          - oran-nlp-region-1
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: nlp-service

    # Aggregate recording rules for federation
    - job_name: 'prometheus'
      static_configs:
      - targets: ['localhost:9090']
```

### Grafana Load Balancing

#### Multiple Grafana Instances

```yaml
# grafana-ha.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: oran-monitoring
spec:
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: grafana
  template:
    metadata:
      labels:
        app.kubernetes.io/name: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:10.2.0
        env:
        - name: GF_DATABASE_TYPE
          value: postgres
        - name: GF_DATABASE_HOST
          value: postgresql:5432
        - name: GF_DATABASE_NAME
          value: grafana
        - name: GF_DATABASE_USER
          value: grafana
        - name: GF_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-db-credentials
              key: password
        - name: GF_SESSION_PROVIDER
          value: postgres
        - name: GF_SESSION_PROVIDER_CONFIG
          value: "user=grafana password=$(GF_DATABASE_PASSWORD) host=postgresql port=5432 dbname=grafana sslmode=disable"
---
apiVersion: v1
kind: Service
metadata:
  name: grafana-ha
  namespace: oran-monitoring
spec:
  type: LoadBalancer
  ports:
  - port: 3000
    targetPort: 3000
  selector:
    app.kubernetes.io/name: grafana
```

#### PostgreSQL for Grafana HA

```yaml
# postgresql-for-grafana.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgresql
  namespace: oran-monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgresql
  template:
    metadata:
      labels:
        app: postgresql
    spec:
      containers:
      - name: postgresql
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: grafana
        - name: POSTGRES_USER
          value: grafana
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-db-credentials
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgresql-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgresql-storage
        persistentVolumeClaim:
          claimName: postgresql-storage
```

### AlertManager Clustering

#### AlertManager Cluster Configuration

```yaml
# alertmanager-cluster.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: alertmanager
  namespace: oran-monitoring
spec:
  serviceName: alertmanager
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: alertmanager
  template:
    metadata:
      labels:
        app.kubernetes.io/name: alertmanager
    spec:
      containers:
      - name: alertmanager
        image: prom/alertmanager:v0.26.0
        args:
        - '--config.file=/etc/alertmanager/alertmanager.yml'
        - '--storage.path=/alertmanager'
        - '--data.retention=120h'
        - '--cluster.listen-address=0.0.0.0:9094'
        - '--cluster.advertise-address=$(POD_IP):9094'
        - '--cluster.peer=alertmanager-0.alertmanager.oran-monitoring.svc.cluster.local:9094'
        - '--cluster.peer=alertmanager-1.alertmanager.oran-monitoring.svc.cluster.local:9094'
        - '--cluster.peer=alertmanager-2.alertmanager.oran-monitoring.svc.cluster.local:9094'
        - '--web.listen-address=0.0.0.0:9093'
        env:
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        ports:
        - containerPort: 9093
          name: web
        - containerPort: 9094
          name: cluster
        volumeMounts:
        - name: alertmanager-storage
          mountPath: /alertmanager
        - name: config-volume
          mountPath: /etc/alertmanager
      volumes:
      - name: config-volume
        configMap:
          name: alertmanager-config
  volumeClaimTemplates:
  - metadata:
      name: alertmanager-storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

## Auto-Scaling

### Horizontal Pod Autoscaler (HPA)

#### Grafana HPA

```yaml
# grafana-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: grafana-hpa
  namespace: oran-monitoring
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: grafana
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

#### Custom Metrics HPA

```yaml
# prometheus-custom-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: prometheus-query-hpa
  namespace: oran-monitoring
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: prometheus-query
  minReplicas: 2
  maxReplicas: 8
  metrics:
  - type: Pods
    pods:
      metric:
        name: prometheus_http_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"
  - type: Pods
    pods:
      metric:
        name: prometheus_query_duration_seconds
      target:
        type: AverageValue
        averageValue: "0.5"
```

### Vertical Pod Autoscaler (VPA)

```yaml
# prometheus-vpa.yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: prometheus-vpa
  namespace: oran-monitoring
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: prometheus
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: prometheus
      minAllowed:
        cpu: 500m
        memory: 2Gi
      maxAllowed:
        cpu: 8000m
        memory: 32Gi
      controlledResources: ["cpu", "memory"]
```

## Performance Optimization

### Prometheus Query Optimization

#### Query Splitting and Caching

```yaml
# prometheus-query-frontend.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-query-frontend
  namespace: oran-monitoring
spec:
  replicas: 3
  selector:
    matchLabels:
      app: prometheus-query-frontend
  template:
    metadata:
      labels:
        app: prometheus-query-frontend
    spec:
      containers:
      - name: query-frontend
        image: thanosio/thanos:v0.32.0
        args:
        - query-frontend
        - --http-address=0.0.0.0:9090
        - --query-frontend.downstream-url=http://prometheus:9090
        - --query-frontend.split-queries-by-interval=24h
        - --query-frontend.align-queries-with-step
        - --query-range.request-downsampled
        - --cache-compression-type=snappy
        ports:
        - containerPort: 9090
          name: http
```

#### Recording Rules for Performance

```yaml
# performance-recording-rules.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-performance-rules
  namespace: oran-monitoring
data:
  performance.yml: |
    groups:
    - name: oran.performance.rules
      interval: 30s
      rules:
      # Pre-aggregate common queries
      - record: oran:cpu_utilization_by_component
        expr: |
          avg_over_time(
            (1 - rate(node_cpu_seconds_total{mode="idle"}[5m]))
            [1h:1m]
          ) by (component, instance)

      - record: oran:memory_utilization_by_component
        expr: |
          avg_over_time(
            (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))
            [1h:1m]
          ) by (component, instance)

      - record: oran:request_rate_by_service
        expr: |
          sum(rate(http_requests_total[5m])) by (service, method, status)

      - record: oran:error_rate_by_service
        expr: |
          sum(rate(http_requests_total{status=~"4..|5.."}[5m])) by (service)
          /
          sum(rate(http_requests_total[5m])) by (service)

      # Downsampled data for long-term queries
      - record: oran:daily_metrics_summary
        expr: |
          avg_over_time(oran:cpu_utilization_by_component[24h])
```

### Storage Optimization

#### TSDB Configuration

```yaml
# prometheus-optimized-storage.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: oran-monitoring
spec:
  template:
    spec:
      containers:
      - name: prometheus
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus/'
        - '--storage.tsdb.retention.time=15d'
        - '--storage.tsdb.retention.size=50GB'
        - '--storage.tsdb.wal-compression'
        - '--storage.tsdb.allow-overlapping-blocks'
        - '--storage.tsdb.head-chunks-write-queue-size=10000'
        - '--storage.tsdb.wal-segment-size=32MB'
        volumeMounts:
        - name: prometheus-storage
          mountPath: /prometheus
          # Use faster storage class
        - name: prometheus-wal
          mountPath: /prometheus/wal
      volumes:
      - name: prometheus-storage
        persistentVolumeClaim:
          claimName: prometheus-storage-ssd
      - name: prometheus-wal
        persistentVolumeClaim:
          claimName: prometheus-wal-nvme
```

#### Compaction Optimization

```bash
# Create compaction job
cat > prometheus-compaction.yaml << 'EOF'
apiVersion: batch/v1
kind: CronJob
metadata:
  name: prometheus-compaction
  namespace: oran-monitoring
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: compaction
            image: prom/prometheus:v2.48.0
            command:
            - promtool
            - tsdb
            - create-blocks-from
            - openmetrics
            - /prometheus
            - /prometheus-compacted
            volumeMounts:
            - name: prometheus-storage
              mountPath: /prometheus
            - name: prometheus-compacted
              mountPath: /prometheus-compacted
          volumes:
          - name: prometheus-storage
            persistentVolumeClaim:
              claimName: prometheus-storage
          - name: prometheus-compacted
            persistentVolumeClaim:
              claimName: prometheus-compacted
          restartPolicy: OnFailure
EOF

kubectl apply -f prometheus-compaction.yaml
```

## Network Scaling

### Service Mesh Integration

#### Istio Integration for Monitoring

```yaml
# istio-monitoring-gateway.yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: monitoring-gateway
  namespace: oran-monitoring
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - prometheus.oran.company.com
    - grafana.oran.company.com
    - alertmanager.oran.company.com
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: SIMPLE
      credentialName: monitoring-tls
    hosts:
    - prometheus.oran.company.com
    - grafana.oran.company.com
    - alertmanager.oran.company.com
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: monitoring-vs
  namespace: oran-monitoring
spec:
  hosts:
  - prometheus.oran.company.com
  - grafana.oran.company.com
  - alertmanager.oran.company.com
  gateways:
  - monitoring-gateway
  http:
  - match:
    - headers:
        host:
          exact: prometheus.oran.company.com
    route:
    - destination:
        host: prometheus
        port:
          number: 9090
  - match:
    - headers:
        host:
          exact: grafana.oran.company.com
    route:
    - destination:
        host: grafana
        port:
          number: 3000
  - match:
    - headers:
        host:
          exact: alertmanager.oran.company.com
    route:
    - destination:
        host: alertmanager
        port:
          number: 9093
```

### CDN Integration for Grafana

```yaml
# grafana-cdn-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-cdn-config
  namespace: oran-monitoring
data:
  grafana.ini: |
    [server]
    serve_from_sub_path = true
    root_url = https://grafana.oran.company.com/

    [security]
    cookie_secure = true
    cookie_samesite = strict

    [auth]
    disable_login_form = false
    oauth_auto_login = true

    [unified_alerting]
    enabled = true

    [feature_toggles]
    enable = "dashboardPreviews,publicDashboards"

    [external_image_storage]
    provider = s3
    bucket_url = https://s3.amazonaws.com/oran-monitoring-images
    access_key = ${AWS_ACCESS_KEY_ID}
    secret_key = ${AWS_SECRET_ACCESS_KEY}
```

## Multi-Cluster Scaling

### Cross-Cluster Federation

```yaml
# prometheus-cross-cluster.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-cross-cluster-config
  namespace: oran-monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 30s
      evaluation_interval: 30s
      external_labels:
        cluster: 'main-cluster'
        region: 'us-west-2'

    scrape_configs:
    # Local cluster monitoring
    - job_name: 'kubernetes-apiservers'
      kubernetes_sd_configs:
      - role: endpoints
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token

    # Federation from other clusters
    - job_name: 'federate-cluster-east'
      scrape_interval: 30s
      honor_labels: true
      metrics_path: '/federate'
      params:
        'match[]':
        - '{__name__=~"oran:.*"}'
        - '{job=~"oran-.*"}'
      static_configs:
      - targets:
        - 'prometheus.cluster-east.company.com:9090'

    - job_name: 'federate-cluster-eu'
      scrape_interval: 30s
      honor_labels: true
      metrics_path: '/federate'
      params:
        'match[]':
        - '{__name__=~"oran:.*"}'
        - '{job=~"oran-.*"}'
      static_configs:
      - targets:
        - 'prometheus.cluster-eu.company.com:9090'

    # Remote write to central storage
    remote_write:
    - url: "https://prometheus-central.company.com/api/v1/write"
      basic_auth:
        username: cluster-main
        password_file: /etc/remote-write/password
```

### Thanos for Long-term Storage

```yaml
# thanos-sidecar.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus-with-thanos
  namespace: oran-monitoring
spec:
  template:
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:v2.48.0
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus'
        - '--storage.tsdb.retention.time=6h'  # Reduced for Thanos
        - '--storage.tsdb.min-block-duration=2h'
        - '--storage.tsdb.max-block-duration=2h'
        - '--web.enable-lifecycle'

      - name: thanos-sidecar
        image: thanosio/thanos:v0.32.0
        args:
        - sidecar
        - --tsdb.path=/prometheus
        - --prometheus.url=http://localhost:9090
        - --objstore.config-file=/etc/thanos/objstore.yml
        - --http-address=0.0.0.0:19191
        - --grpc-address=0.0.0.0:19090
        ports:
        - containerPort: 19191
          name: sidecar-http
        - containerPort: 19090
          name: sidecar-grpc
        volumeMounts:
        - name: prometheus-storage
          mountPath: /prometheus
        - name: thanos-objstore-config
          mountPath: /etc/thanos
      volumes:
      - name: thanos-objstore-config
        secret:
          secretName: thanos-objstore-config
---
apiVersion: v1
kind: Secret
metadata:
  name: thanos-objstore-config
  namespace: oran-monitoring
stringData:
  objstore.yml: |
    type: S3
    config:
      bucket: "oran-monitoring-thanos"
      endpoint: "s3.amazonaws.com"
      access_key: "${AWS_ACCESS_KEY_ID}"
      secret_key: "${AWS_SECRET_ACCESS_KEY}"
      insecure: false
```

## Monitoring Scaling Operations

### Scaling Metrics

```yaml
# scaling-monitoring.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: scaling-monitoring
  namespace: oran-monitoring
spec:
  groups:
  - name: scaling.rules
    rules:
    - alert: PrometheusHighQueryLoad
      expr: rate(prometheus_http_requests_total{handler="/api/v1/query"}[5m]) > 100
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Prometheus query load is high"
        description: "Prometheus is receiving {{ $value }} queries per second"

    - alert: GrafanaHighCPU
      expr: rate(container_cpu_usage_seconds_total{pod=~"grafana-.*"}[5m]) > 0.8
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "Grafana CPU usage is high"
        description: "Grafana pod {{ $labels.pod }} CPU usage is {{ $value }}"

    - alert: PrometheusStorageGrowth
      expr: |
        predict_linear(
          prometheus_tsdb_head_series[6h], 7*24*3600
        ) > 10000000
      for: 1h
      labels:
        severity: warning
      annotations:
        summary: "Prometheus storage growth is high"
        description: "Prometheus series count will exceed 10M in 7 days"
```

### Automated Scaling Scripts

```bash
#!/bin/bash
# auto-scaling-monitor.sh

# Configuration
NAMESPACE="oran-monitoring"
PROMETHEUS_CPU_THRESHOLD=80
GRAFANA_MEMORY_THRESHOLD=85
STORAGE_THRESHOLD=80

# Get current resource usage
PROMETHEUS_CPU=$(kubectl top pod -n $NAMESPACE -l app.kubernetes.io/name=prometheus --no-headers | awk '{print $3}' | sed 's/%//')
GRAFANA_MEMORY=$(kubectl top pod -n $NAMESPACE -l app.kubernetes.io/name=grafana --no-headers | awk '{print $4}' | sed 's/%//')
STORAGE_USAGE=$(kubectl exec -n $NAMESPACE deployment/prometheus -- df /prometheus | tail -1 | awk '{print $5}' | sed 's/%//')

echo "Current usage: Prometheus CPU: ${PROMETHEUS_CPU}%, Grafana Memory: ${GRAFANA_MEMORY}%, Storage: ${STORAGE_USAGE}%"

# Scale Prometheus if needed
if [[ $PROMETHEUS_CPU -gt $PROMETHEUS_CPU_THRESHOLD ]]; then
    echo "Scaling Prometheus CPU..."
    kubectl patch deployment prometheus -n $NAMESPACE -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","resources":{"limits":{"cpu":"4000m"},"requests":{"cpu":"1000m"}}}]}}}}'
fi

# Scale Grafana if needed
if [[ $GRAFANA_MEMORY -gt $GRAFANA_MEMORY_THRESHOLD ]]; then
    echo "Scaling Grafana replicas..."
    CURRENT_REPLICAS=$(kubectl get deployment grafana -n $NAMESPACE -o jsonpath='{.spec.replicas}')
    NEW_REPLICAS=$((CURRENT_REPLICAS + 1))
    kubectl scale deployment grafana --replicas=$NEW_REPLICAS -n $NAMESPACE
fi

# Alert on storage if needed
if [[ $STORAGE_USAGE -gt $STORAGE_THRESHOLD ]]; then
    echo "Storage usage is high: ${STORAGE_USAGE}%"
    # Trigger storage expansion or cleanup
    kubectl annotate pvc prometheus-storage -n $NAMESPACE scaling.oran.com/expand=true
fi
```

## Cost Optimization

### Resource Right-Sizing

```bash
# resource-analysis.sh
#!/bin/bash

echo "=== Resource Usage Analysis ==="

# Analyze actual vs requested resources
kubectl get pods -n oran-monitoring -o custom-columns="NAME:.metadata.name,CPU_REQ:.spec.containers[*].resources.requests.cpu,MEM_REQ:.spec.containers[*].resources.requests.memory"

# Get actual usage
kubectl top pods -n oran-monitoring

# Calculate waste
echo "=== Resource Efficiency ==="
for pod in $(kubectl get pods -n oran-monitoring -o name); do
    echo "Analyzing $pod..."
    # This would require metrics collection over time
done
```

### Storage Optimization

```yaml
# storage-optimization.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-retention-config
  namespace: oran-monitoring
data:
  retention-policy.yml: |
    # Different retention for different metric types
    global:
      retention: 15d

    # Override for specific metrics
    metric_retention:
      # High-frequency metrics - shorter retention
      - metric_regex: "container_.*"
        retention: 7d

      # Business metrics - longer retention
      - metric_regex: "oran_.*"
        retention: 30d

      # Debug metrics - very short retention
      - metric_regex: "debug_.*"
        retention: 1d
```

## Disaster Recovery for Scaled Environment

### Backup Strategy

```bash
# scaled-backup.sh
#!/bin/bash

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/oran-monitoring/$BACKUP_DATE"

# Backup all instances
for instance in prometheus-main prometheus-region-1 prometheus-region-2; do
    echo "Backing up $instance..."
    kubectl exec -n oran-monitoring deployment/$instance -- \
        tar czf - /prometheus | gzip > "$BACKUP_DIR/${instance}-data.tar.gz"
done

# Backup Grafana cluster
kubectl exec -n oran-monitoring deployment/grafana -- \
    pg_dump grafana | gzip > "$BACKUP_DIR/grafana-db.sql.gz"

echo "Backup completed: $BACKUP_DIR"
```

## Performance Testing for Scaled Environment

```bash
# load-test-scaled.sh
#!/bin/bash

# Test query performance under load
echo "Testing Prometheus query performance..."
for i in {1..100}; do
    curl -s "http://prometheus.oran-monitoring.svc.cluster.local:9090/api/v1/query?query=up" &
done

wait

# Test Grafana dashboard load
echo "Testing Grafana dashboard performance..."
for i in {1..50}; do
    curl -s -u admin:password "http://grafana.oran-monitoring.svc.cluster.local:3000/api/dashboards/home" &
done

wait

echo "Load testing completed"
```

This comprehensive scaling guide provides the foundation for growing your O-RAN monitoring stack from a small deployment to an enterprise-scale, multi-region setup while maintaining performance and reliability.