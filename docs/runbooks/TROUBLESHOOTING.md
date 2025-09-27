# O-RAN Monitoring Stack Troubleshooting Guide

This guide provides comprehensive troubleshooting procedures for the O-RAN monitoring stack.

## Quick Diagnosis

### Health Check Script

```bash
#!/bin/bash
# quick-health-check.sh

echo "=== O-RAN Monitoring Stack Health Check ==="

# Check namespace
kubectl get namespace oran-monitoring || echo "❌ Namespace missing"

# Check pods
echo "=== Pod Status ==="
kubectl get pods -n oran-monitoring -o wide

# Check services
echo "=== Service Status ==="
kubectl get svc -n oran-monitoring

# Check persistent volumes
echo "=== Storage Status ==="
kubectl get pvc -n oran-monitoring

# Check recent events
echo "=== Recent Events ==="
kubectl get events -n oran-monitoring --sort-by='.lastTimestamp' | tail -10
```

### Common Commands

```bash
# Quick status check
kubectl get all -n oran-monitoring

# Check resource usage
kubectl top pods -n oran-monitoring
kubectl top nodes

# View logs
kubectl logs -f deployment/prometheus -n oran-monitoring
kubectl logs -f deployment/grafana -n oran-monitoring
kubectl logs -f deployment/alertmanager -n oran-monitoring
```

## Component-Specific Troubleshooting

### Prometheus Issues

#### Pod Won't Start

**Symptoms:**
- Prometheus pod stuck in `Pending`, `CrashLoopBackOff`, or `Error` state
- Events show resource or configuration issues

**Diagnosis:**
```bash
# Check pod status and events
kubectl describe pod -l app.kubernetes.io/name=prometheus -n oran-monitoring

# Check logs
kubectl logs deployment/prometheus -n oran-monitoring

# Check configuration
kubectl get configmap prometheus-config -n oran-monitoring -o yaml
```

**Common Causes & Solutions:**

1. **Resource Constraints**
   ```bash
   # Check available resources
   kubectl describe nodes | grep -A 5 "Allocatable"

   # Solution: Reduce resource requests or add nodes
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","resources":{"requests":{"cpu":"250m","memory":"1Gi"}}}]}}}}'
   ```

2. **Storage Issues**
   ```bash
   # Check PVC status
   kubectl get pvc -n oran-monitoring
   kubectl describe pvc prometheus-storage -n oran-monitoring

   # Solution: Check storage class and availability
   kubectl get storageclass
   ```

3. **Configuration Errors**
   ```bash
   # Validate Prometheus config
   kubectl exec deployment/prometheus -n oran-monitoring -- promtool check config /etc/prometheus/prometheus.yml

   # Solution: Fix configuration errors in ConfigMap
   kubectl edit configmap prometheus-config -n oran-monitoring
   ```

4. **RBAC Issues**
   ```bash
   # Check service account permissions
   kubectl auth can-i list nodes --as=system:serviceaccount:oran-monitoring:prometheus

   # Solution: Apply correct RBAC
   kubectl apply -f monitoring/prometheus/prometheus-rbac.yaml
   ```

#### No Data Being Scraped

**Symptoms:**
- Prometheus starts but shows no targets or all targets are down
- Queries return no results

**Diagnosis:**
```bash
# Check targets via port-forward
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/targets" | jq .

# Check service discovery
curl "http://localhost:9090/api/v1/targets" | jq '.data.activeTargets[] | select(.health != "up")'
```

**Solutions:**

1. **Network Connectivity**
   ```bash
   # Test connectivity from Prometheus pod
   kubectl exec deployment/prometheus -n oran-monitoring -- wget -O- --timeout=5 http://kubernetes.default.svc:443/api/v1/nodes

   # Check network policies
   kubectl get networkpolicy -n oran-monitoring
   ```

2. **Service Discovery Issues**
   ```bash
   # Check if services exist
   kubectl get svc --all-namespaces -l prometheus.io/scrape=true

   # Verify service annotations
   kubectl get svc -o yaml | grep -A 3 -B 3 prometheus.io
   ```

3. **Firewall/Security Context**
   ```bash
   # Check pod security context
   kubectl get pod -l app.kubernetes.io/name=prometheus -n oran-monitoring -o yaml | grep -A 10 securityContext

   # Test with relaxed security (temporary)
   kubectl patch deployment prometheus -n oran-monitoring --type='merge' -p='{"spec":{"template":{"spec":{"securityContext":{"runAsUser":0}}}}}'
   ```

#### High Memory Usage

**Symptoms:**
- Prometheus pod consuming excessive memory
- OOMKilled events
- Slow query performance

**Diagnosis:**
```bash
# Check memory usage
kubectl top pod -l app.kubernetes.io/name=prometheus -n oran-monitoring

# Check series count
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/query?query=prometheus_tsdb_head_series"

# Check highest cardinality metrics
curl "http://localhost:9090/api/v1/query?query=topk(10,count%20by%20(__name__)({__name__!=\"\"}))"
```

**Solutions:**

1. **Increase Memory Limits**
   ```bash
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","resources":{"limits":{"memory":"8Gi"}}}]}}}}'
   ```

2. **Reduce Retention**
   ```bash
   # Edit Prometheus args
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","args":["--storage.tsdb.retention.time=7d"]}]}}}}'
   ```

3. **Optimize Scrape Configuration**
   ```bash
   # Increase scrape intervals for non-critical targets
   kubectl edit configmap prometheus-config -n oran-monitoring
   # Change scrape_interval: 30s to 60s for high-volume targets
   ```

### Grafana Issues

#### Cannot Access Grafana UI

**Symptoms:**
- Grafana service unreachable
- Login page not loading
- Connection refused errors

**Diagnosis:**
```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=grafana -n oran-monitoring

# Check service
kubectl get svc grafana -n oran-monitoring

# Check ingress (if configured)
kubectl get ingress -n oran-monitoring

# Port forward for direct access
kubectl port-forward -n oran-monitoring svc/grafana 3000:3000
```

**Solutions:**

1. **Pod Issues**
   ```bash
   # Check logs
   kubectl logs deployment/grafana -n oran-monitoring

   # Restart deployment
   kubectl rollout restart deployment/grafana -n oran-monitoring
   ```

2. **Service Configuration**
   ```bash
   # Verify service endpoints
   kubectl get endpoints grafana -n oran-monitoring

   # Check service selector
   kubectl describe svc grafana -n oran-monitoring
   ```

3. **Ingress/LoadBalancer Issues**
   ```bash
   # Check ingress controller
   kubectl get pods -n ingress-nginx

   # Verify ingress rules
   kubectl describe ingress grafana -n oran-monitoring
   ```

#### Login Issues

**Symptoms:**
- Cannot login with admin credentials
- "Invalid username or password" errors
- Authentication failures

**Diagnosis:**
```bash
# Check admin credentials
kubectl get secret grafana-admin-credentials -n oran-monitoring -o yaml

# Decode password
kubectl get secret grafana-admin-credentials -n oran-monitoring -o jsonpath='{.data.password}' | base64 -d

# Check Grafana logs for auth errors
kubectl logs deployment/grafana -n oran-monitoring | grep -i auth
```

**Solutions:**

1. **Reset Admin Password**
   ```bash
   # Create new password secret
   kubectl create secret generic grafana-admin-credentials \
     --from-literal=password='new-secure-password' \
     -n oran-monitoring --dry-run=client -o yaml | kubectl apply -f -

   # Restart Grafana
   kubectl rollout restart deployment/grafana -n oran-monitoring
   ```

2. **Database Issues**
   ```bash
   # Check if database is corrupted
   kubectl exec deployment/grafana -n oran-monitoring -- sqlite3 /var/lib/grafana/grafana.db "SELECT * FROM user WHERE login='admin';"

   # Reset admin user (if needed)
   kubectl exec deployment/grafana -n oran-monitoring -- grafana-cli admin reset-admin-password new-password
   ```

#### Dashboards Not Loading

**Symptoms:**
- Empty dashboard list
- Dashboards show "No data" or loading errors
- Panels not rendering

**Diagnosis:**
```bash
# Check dashboard provisioning
kubectl get configmap -l app.kubernetes.io/name=grafana -n oran-monitoring

# Check data source configuration
kubectl port-forward -n oran-monitoring svc/grafana 3000:3000 &
curl -u admin:password "http://localhost:3000/api/datasources"

# Check dashboard files
kubectl exec deployment/grafana -n oran-monitoring -- ls -la /var/lib/grafana/dashboards/
```

**Solutions:**

1. **Data Source Issues**
   ```bash
   # Test Prometheus connection from Grafana
   kubectl exec deployment/grafana -n oran-monitoring -- wget -O- --timeout=5 http://prometheus:9090/api/v1/query?query=up

   # Recreate data source
   kubectl delete configmap grafana-datasources -n oran-monitoring
   kubectl apply -f monitoring/grafana/datasources.yaml
   ```

2. **Dashboard Provisioning**
   ```bash
   # Check dashboard provider config
   kubectl get configmap grafana-dashboard-provider -n oran-monitoring -o yaml

   # Restart Grafana to reload dashboards
   kubectl rollout restart deployment/grafana -n oran-monitoring
   ```

### AlertManager Issues

#### Alerts Not Firing

**Symptoms:**
- No alerts in AlertManager UI
- Expected alerts not triggering
- Prometheus shows firing alerts but AlertManager doesn't

**Diagnosis:**
```bash
# Check AlertManager status
kubectl port-forward -n oran-monitoring svc/alertmanager 9093:9093 &
curl "http://localhost:9093/api/v2/status"

# Check Prometheus alerting configuration
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/alertmanagers"

# Check alert rules
curl "http://localhost:9090/api/v1/rules"
```

**Solutions:**

1. **Prometheus-AlertManager Connection**
   ```bash
   # Verify AlertManager service
   kubectl get svc alertmanager -n oran-monitoring

   # Check network connectivity
   kubectl exec deployment/prometheus -n oran-monitoring -- wget -O- --timeout=5 http://alertmanager:9093/api/v2/status
   ```

2. **Alert Rules Configuration**
   ```bash
   # Validate alert rules
   kubectl exec deployment/prometheus -n oran-monitoring -- promtool check rules /etc/prometheus/rules/*.yml

   # Reload configuration
   kubectl exec deployment/prometheus -n oran-monitoring -- curl -X POST http://localhost:9090/-/reload
   ```

#### Notifications Not Sending

**Symptoms:**
- Alerts firing but no notifications received
- AlertManager shows alerts but notifications fail
- Error logs in AlertManager

**Diagnosis:**
```bash
# Check AlertManager logs
kubectl logs deployment/alertmanager -n oran-monitoring

# Check configuration
kubectl get secret alertmanager-config -n oran-monitoring -o yaml

# Test notification channels
kubectl port-forward -n oran-monitoring svc/alertmanager 9093:9093 &
curl "http://localhost:9093/api/v2/alerts"
```

**Solutions:**

1. **Email Configuration**
   ```bash
   # Test SMTP connectivity
   kubectl exec deployment/alertmanager -n oran-monitoring -- nc -zv smtp.company.com 587

   # Update SMTP settings
   kubectl edit secret alertmanager-config -n oran-monitoring
   ```

2. **Webhook/Slack Configuration**
   ```bash
   # Test webhook endpoint
   kubectl exec deployment/alertmanager -n oran-monitoring -- curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL -d '{"text":"test"}'

   # Verify configuration format
   kubectl exec deployment/alertmanager -n oran-monitoring -- amtool config check /etc/alertmanager/alertmanager.yml
   ```

## Storage Issues

### Persistent Volume Problems

**Symptoms:**
- Pods stuck in pending due to volume mounting issues
- Data loss after pod restarts
- Storage capacity warnings

**Diagnosis:**
```bash
# Check PVC status
kubectl get pvc -n oran-monitoring
kubectl describe pvc -n oran-monitoring

# Check storage class
kubectl get storageclass

# Check available storage
kubectl get pv
```

**Solutions:**

1. **Storage Class Issues**
   ```bash
   # List available storage classes
   kubectl get storageclass

   # Set default storage class
   kubectl patch storageclass <class-name> -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
   ```

2. **Capacity Issues**
   ```bash
   # Expand PVC (if supported)
   kubectl patch pvc prometheus-storage -n oran-monitoring -p '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'

   # Monitor expansion
   kubectl get pvc prometheus-storage -n oran-monitoring -w
   ```

3. **Mount Issues**
   ```bash
   # Check node storage
   kubectl describe nodes | grep -A 10 "Attached Volumes"

   # Force pod restart to retry mount
   kubectl delete pod -l app.kubernetes.io/name=prometheus -n oran-monitoring
   ```

## Network and Connectivity Issues

### Service Discovery Problems

**Symptoms:**
- Prometheus can't discover targets
- Intermittent connectivity between components
- DNS resolution failures

**Diagnosis:**
```bash
# Test DNS resolution
kubectl exec deployment/prometheus -n oran-monitoring -- nslookup grafana.oran-monitoring.svc.cluster.local

# Check service endpoints
kubectl get endpoints -n oran-monitoring

# Test network connectivity
kubectl exec deployment/prometheus -n oran-monitoring -- nc -zv grafana 3000
```

**Solutions:**

1. **DNS Issues**
   ```bash
   # Check CoreDNS
   kubectl get pods -n kube-system -l k8s-app=kube-dns
   kubectl logs -n kube-system deployment/coredns

   # Restart CoreDNS
   kubectl rollout restart deployment/coredns -n kube-system
   ```

2. **Network Policy Restrictions**
   ```bash
   # Check network policies
   kubectl get networkpolicy -n oran-monitoring

   # Temporary removal for testing
   kubectl delete networkpolicy --all -n oran-monitoring
   ```

3. **Service Mesh Issues** (if using Istio/Linkerd)
   ```bash
   # Check sidecar injection
   kubectl get pods -n oran-monitoring -o jsonpath='{.items[*].spec.containers[*].name}'

   # Disable service mesh temporarily
   kubectl label namespace oran-monitoring istio-injection-
   ```

## Performance Issues

### High CPU/Memory Usage

**Symptoms:**
- Pods consuming excessive resources
- Slow query responses
- Cluster node resource exhaustion

**Diagnosis:**
```bash
# Check resource usage
kubectl top pods -n oran-monitoring
kubectl top nodes

# Check resource requests/limits
kubectl describe pods -n oran-monitoring | grep -A 5 "Requests:\|Limits:"

# Monitor over time
watch kubectl top pods -n oran-monitoring
```

**Solutions:**

1. **Optimize Prometheus**
   ```bash
   # Reduce query concurrency
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","args":["--query.max-concurrency=20"]}]}}}}'

   # Optimize storage settings
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","args":["--storage.tsdb.wal-compression"]}]}}}}'
   ```

2. **Optimize Grafana**
   ```bash
   # Limit dashboard refresh rates
   # Edit dashboards to use longer refresh intervals

   # Reduce concurrent users
   kubectl scale deployment grafana --replicas=1 -n oran-monitoring
   ```

3. **Horizontal Scaling**
   ```bash
   # Scale Grafana (if stateless)
   kubectl scale deployment grafana --replicas=3 -n oran-monitoring

   # Add Prometheus federation for scaling
   # See scaling documentation
   ```

### Slow Query Performance

**Symptoms:**
- Dashboards loading slowly
- Query timeouts
- High latency in Prometheus

**Diagnosis:**
```bash
# Check query performance
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/query?query=prometheus_engine_query_duration_seconds"

# Check slow queries
curl "http://localhost:9090/api/v1/status/tsdb" | jq .
```

**Solutions:**

1. **Query Optimization**
   ```bash
   # Use recording rules for complex queries
   kubectl edit configmap prometheus-config -n oran-monitoring
   # Add recording rules for frequently used complex queries
   ```

2. **Index Optimization**
   ```bash
   # Restart Prometheus to rebuild indexes
   kubectl rollout restart deployment/prometheus -n oran-monitoring

   # Monitor TSDB stats
   curl "http://localhost:9090/api/v1/status/tsdb"
   ```

## Security Issues

### RBAC and Permissions

**Symptoms:**
- Permission denied errors
- Components unable to access Kubernetes API
- Authentication failures

**Diagnosis:**
```bash
# Check service accounts
kubectl get sa -n oran-monitoring

# Test permissions
kubectl auth can-i list pods --as=system:serviceaccount:oran-monitoring:prometheus

# Check role bindings
kubectl get rolebinding,clusterrolebinding -o wide | grep oran-monitoring
```

**Solutions:**

1. **Fix RBAC**
   ```bash
   # Apply correct RBAC
   kubectl apply -f monitoring/rbac/

   # Create service account
   kubectl create sa prometheus -n oran-monitoring

   # Bind cluster role
   kubectl create clusterrolebinding prometheus-binding \
     --clusterrole=prometheus \
     --serviceaccount=oran-monitoring:prometheus
   ```

2. **Security Context Issues**
   ```bash
   # Check pod security standards
   kubectl get namespace oran-monitoring -o yaml | grep pod-security

   # Update security context
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"securityContext":{"runAsNonRoot":true,"runAsUser":65534}}}}}'
   ```

### Certificate and TLS Issues

**Symptoms:**
- TLS handshake failures
- Certificate validation errors
- HTTPS endpoints not accessible

**Diagnosis:**
```bash
# Check certificates
kubectl get secrets -n oran-monitoring | grep tls

# Verify certificate validity
kubectl get secret prometheus-tls -n oran-monitoring -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

**Solutions:**

1. **Renew Certificates**
   ```bash
   # Generate new certificates
   openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
     -keyout prometheus.key -out prometheus.crt \
     -subj "/CN=prometheus.oran-monitoring.svc.cluster.local"

   # Update secret
   kubectl create secret tls prometheus-tls \
     --cert=prometheus.crt --key=prometheus.key \
     -n oran-monitoring --dry-run=client -o yaml | kubectl apply -f -
   ```

2. **Disable TLS for Troubleshooting**
   ```bash
   # Temporarily disable TLS
   kubectl patch deployment prometheus -n oran-monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","args":["--web.listen-address=0.0.0.0:9090"]}]}}}}'
   ```

## Data and Configuration Issues

### Configuration Validation

**Symptoms:**
- Components not starting due to config errors
- Invalid YAML or syntax errors
- Missing required configuration

**Diagnosis:**
```bash
# Validate Prometheus config
kubectl exec deployment/prometheus -n oran-monitoring -- promtool check config /etc/prometheus/prometheus.yml

# Validate AlertManager config
kubectl exec deployment/alertmanager -n oran-monitoring -- amtool config check /etc/alertmanager/alertmanager.yml

# Check ConfigMap syntax
kubectl get configmap prometheus-config -n oran-monitoring -o yaml | yq eval '.'
```

**Solutions:**

1. **Fix Configuration Syntax**
   ```bash
   # Edit configuration
   kubectl edit configmap prometheus-config -n oran-monitoring

   # Validate before applying
   kubectl get configmap prometheus-config -n oran-monitoring -o yaml > config.yaml
   promtool check config config.yaml
   ```

2. **Reload Configuration**
   ```bash
   # Reload Prometheus config
   kubectl exec deployment/prometheus -n oran-monitoring -- curl -X POST http://localhost:9090/-/reload

   # Restart if reload fails
   kubectl rollout restart deployment/prometheus -n oran-monitoring
   ```

### Data Loss and Recovery

**Symptoms:**
- Historical data missing
- Metrics gaps
- Dashboard showing no data for historical periods

**Diagnosis:**
```bash
# Check data retention
kubectl exec deployment/prometheus -n oran-monitoring -- find /prometheus -name "*.db" -exec ls -la {} \;

# Check TSDB status
kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
curl "http://localhost:9090/api/v1/status/tsdb"
```

**Solutions:**

1. **Restore from Backup**
   ```bash
   # Stop Prometheus
   kubectl scale deployment prometheus --replicas=0 -n oran-monitoring

   # Restore data
   kubectl exec deployment/prometheus -n oran-monitoring -- tar xzf /backup/prometheus-data.tar.gz -C /

   # Start Prometheus
   kubectl scale deployment prometheus --replicas=1 -n oran-monitoring
   ```

2. **Data Corruption Recovery**
   ```bash
   # Check for corruption
   kubectl exec deployment/prometheus -n oran-monitoring -- promtool tsdb analyze /prometheus

   # Compact data
   kubectl exec deployment/prometheus -n oran-monitoring -- promtool tsdb create-blocks-from tsdb /prometheus /prometheus-new
   ```

## Monitoring and Alerting for the Monitoring Stack

### Self-Monitoring Setup

```yaml
# self-monitoring-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: monitoring-stack-alerts
  namespace: oran-monitoring
spec:
  groups:
  - name: monitoring.rules
    rules:
    - alert: PrometheusDown
      expr: up{job="prometheus"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Prometheus is down"
        description: "Prometheus has been down for more than 5 minutes"

    - alert: GrafanaDown
      expr: up{job="grafana"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Grafana is down"
        description: "Grafana has been down for more than 5 minutes"

    - alert: AlertManagerDown
      expr: up{job="alertmanager"} == 0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "AlertManager is down"
        description: "AlertManager has been down for more than 5 minutes"
```

### Automated Recovery

```bash
# Create automated recovery job
cat > monitoring-recovery.yaml << 'EOF'
apiVersion: batch/v1
kind: CronJob
metadata:
  name: monitoring-recovery
  namespace: oran-monitoring
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: recovery
            image: bitnami/kubectl:latest
            command:
            - /bin/bash
            - -c
            - |
              # Check if prometheus is healthy
              if ! kubectl get deployment prometheus -n oran-monitoring -o jsonpath='{.status.readyReplicas}' | grep -q "1"; then
                echo "Prometheus unhealthy, restarting..."
                kubectl rollout restart deployment/prometheus -n oran-monitoring
              fi

              # Similar checks for Grafana and AlertManager
          restartPolicy: OnFailure
EOF

kubectl apply -f monitoring-recovery.yaml
```

## Emergency Procedures

### Complete Stack Recovery

```bash
#!/bin/bash
# emergency-recovery.sh

echo "=== Emergency O-RAN Monitoring Stack Recovery ==="

# 1. Stop all components
kubectl scale deployment prometheus --replicas=0 -n oran-monitoring
kubectl scale deployment grafana --replicas=0 -n oran-monitoring
kubectl scale deployment alertmanager --replicas=0 -n oran-monitoring

# 2. Check and fix storage issues
kubectl get pvc -n oran-monitoring
kubectl describe pvc -n oran-monitoring

# 3. Restore from backups (if needed)
# ./restore-monitoring-backup.sh

# 4. Restart components
kubectl scale deployment prometheus --replicas=1 -n oran-monitoring
kubectl scale deployment grafana --replicas=1 -n oran-monitoring
kubectl scale deployment alertmanager --replicas=1 -n oran-monitoring

# 5. Wait for readiness
kubectl wait --for=condition=available deployment/prometheus -n oran-monitoring --timeout=300s
kubectl wait --for=condition=available deployment/grafana -n oran-monitoring --timeout=300s
kubectl wait --for=condition=available deployment/alertmanager -n oran-monitoring --timeout=300s

# 6. Verify functionality
./deployment/kubernetes/health-checks/check-prometheus-targets.sh
./deployment/kubernetes/health-checks/check-grafana-dashboards.sh
./deployment/kubernetes/health-checks/check-alerts.sh

echo "=== Recovery Complete ==="
```

## Getting Help

### Log Collection

```bash
# Collect all relevant logs
mkdir -p troubleshooting-logs/$(date +%Y%m%d-%H%M%S)
cd troubleshooting-logs/$(date +%Y%m%d-%H%M%S)

# Pod logs
kubectl logs deployment/prometheus -n oran-monitoring > prometheus.log
kubectl logs deployment/grafana -n oran-monitoring > grafana.log
kubectl logs deployment/alertmanager -n oran-monitoring > alertmanager.log

# Configuration
kubectl get configmap -n oran-monitoring -o yaml > configmaps.yaml
kubectl get secret -n oran-monitoring -o yaml > secrets.yaml

# Status
kubectl get all -n oran-monitoring -o yaml > resources.yaml
kubectl describe pods -n oran-monitoring > pod-descriptions.txt
kubectl get events -n oran-monitoring > events.txt

# System info
kubectl version > cluster-info.txt
kubectl get nodes -o wide >> cluster-info.txt

echo "Logs collected in $(pwd)"
```

### Contact Information

- **Internal Support**: oran-monitoring-team@company.com
- **Emergency Contact**: +1-XXX-XXX-XXXX
- **Documentation**: [Internal Wiki](https://wiki.company.com/oran-monitoring)
- **Issue Tracker**: [JIRA Project](https://jira.company.com/projects/ORAN-MON)

### Escalation Matrix

1. **Level 1**: Self-service using this guide
2. **Level 2**: Team lead or senior engineer
3. **Level 3**: Platform engineering team
4. **Level 4**: Vendor support (if applicable)

Remember to include relevant logs, configurations, and error messages when escalating issues.