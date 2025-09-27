# O-RAN Monitoring Stack Alert Response Playbook

This playbook provides step-by-step response procedures for alerts in the O-RAN monitoring stack.

## Alert Classification

### Severity Levels

| Severity | Response Time | Description | Examples |
|----------|---------------|-------------|----------|
| **Critical** | 5 minutes | Service completely down | Prometheus down, Complete network failure |
| **High** | 15 minutes | Major functionality impacted | High error rate, Memory exhaustion |
| **Medium** | 1 hour | Degraded performance | Slow queries, High latency |
| **Low** | 4 hours | Warning conditions | Disk usage, Certificate expiry |

### Alert Categories

1. **Infrastructure Alerts**: Node, storage, network issues
2. **Application Alerts**: O-RAN component failures
3. **Performance Alerts**: Latency, throughput issues
4. **Security Alerts**: Unauthorized access, vulnerabilities
5. **Monitoring Alerts**: Monitoring stack issues

## General Response Workflow

### 1. Alert Acknowledgment (2 minutes)

```bash
# Acknowledge alert in AlertManager
curl -X POST http://alertmanager:9093/api/v2/silences \
  -H "Content-Type: application/json" \
  -d '{
    "matchers": [
      {
        "name": "alertname",
        "value": "ALERT_NAME"
      }
    ],
    "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'",
    "endsAt": "'$(date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%S.%3NZ)'",
    "createdBy": "oncall-engineer",
    "comment": "Investigating alert"
  }'

# Update incident tracking system
# Send initial acknowledgment to stakeholders
```

### 2. Initial Assessment (3 minutes)

```bash
# Quick health check
kubectl get pods -n oran-monitoring
kubectl get nodes
kubectl top nodes

# Check recent events
kubectl get events --sort-by='.lastTimestamp' | tail -20

# Check service status
./deployment/kubernetes/health-checks/check-prometheus-targets.sh
./deployment/kubernetes/health-checks/check-grafana-dashboards.sh
```

### 3. Information Gathering (5 minutes)

```bash
# Collect logs
mkdir -p incident-$(date +%Y%m%d_%H%M%S)
cd incident-$(date +%Y%m%d_%H%M%S)

# Component logs
kubectl logs -n oran-monitoring deployment/prometheus --tail=100 > prometheus.log
kubectl logs -n oran-monitoring deployment/grafana --tail=100 > grafana.log
kubectl logs -n oran-monitoring deployment/alertmanager --tail=100 > alertmanager.log

# System status
kubectl get all -n oran-monitoring > cluster-status.txt
kubectl describe pods -n oran-monitoring > pod-descriptions.txt
```

## Critical Alerts Response

### PrometheusDown

**Alert**: Prometheus server is down or unreachable

**Impact**: Complete loss of metrics collection and alerting

**Response Steps**:

1. **Immediate Actions** (0-5 minutes):
   ```bash
   # Check if pod is running
   kubectl get pods -n oran-monitoring -l app.kubernetes.io/name=prometheus

   # Check pod status
   kubectl describe pod -l app.kubernetes.io/name=prometheus -n oran-monitoring

   # Check recent events
   kubectl get events -n oran-monitoring --field-selector involvedObject.name=prometheus
   ```

2. **Diagnosis** (5-10 minutes):
   ```bash
   # Check logs
   kubectl logs deployment/prometheus -n oran-monitoring --tail=50

   # Check resource usage
   kubectl top pod -l app.kubernetes.io/name=prometheus -n oran-monitoring

   # Check PVC status
   kubectl get pvc -n oran-monitoring -l app.kubernetes.io/name=prometheus
   ```

3. **Recovery Actions**:
   ```bash
   # Option 1: Restart deployment
   kubectl rollout restart deployment/prometheus -n oran-monitoring

   # Option 2: Scale down and up
   kubectl scale deployment prometheus --replicas=0 -n oran-monitoring
   sleep 30
   kubectl scale deployment prometheus --replicas=1 -n oran-monitoring

   # Option 3: Force recreate pod
   kubectl delete pod -l app.kubernetes.io/name=prometheus -n oran-monitoring
   ```

4. **Verification**:
   ```bash
   # Wait for readiness
   kubectl wait --for=condition=available deployment/prometheus -n oran-monitoring --timeout=300s

   # Verify targets
   ./deployment/kubernetes/health-checks/check-prometheus-targets.sh
   ```

5. **Follow-up**:
   - Check data integrity after restart
   - Review and address root cause
   - Update incident documentation

### GrafanaDown

**Alert**: Grafana dashboard server is unreachable

**Impact**: Loss of visualization and dashboard access

**Response Steps**:

1. **Immediate Actions**:
   ```bash
   # Check pod status
   kubectl get pods -n oran-monitoring -l app.kubernetes.io/name=grafana
   kubectl describe pod -l app.kubernetes.io/name=grafana -n oran-monitoring
   ```

2. **Common Issues & Solutions**:

   **Database Connection Issues**:
   ```bash
   # Check database connectivity
   kubectl exec deployment/grafana -n oran-monitoring -- \
     nc -zv postgresql 5432

   # Restart Grafana
   kubectl rollout restart deployment/grafana -n oran-monitoring
   ```

   **Configuration Issues**:
   ```bash
   # Check configuration
   kubectl get configmap grafana-config -n oran-monitoring -o yaml

   # Validate configuration
   kubectl exec deployment/grafana -n oran-monitoring -- \
     grafana-cli admin reset-admin-password newpassword
   ```

   **Resource Issues**:
   ```bash
   # Check resource usage
   kubectl top pod -l app.kubernetes.io/name=grafana -n oran-monitoring

   # Increase resources
   kubectl patch deployment grafana -n oran-monitoring -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"grafana","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
   ```

### AlertManagerDown

**Alert**: AlertManager is not responsive

**Impact**: No alert notifications being sent

**Response Steps**:

1. **Immediate Actions**:
   ```bash
   # Check cluster status
   kubectl get pods -n oran-monitoring -l app.kubernetes.io/name=alertmanager

   # Check cluster communication
   kubectl logs -l app.kubernetes.io/name=alertmanager -n oran-monitoring | grep cluster
   ```

2. **Recovery**:
   ```bash
   # Restart StatefulSet
   kubectl rollout restart statefulset/alertmanager -n oran-monitoring

   # If clustering issues, restart one by one
   for i in {0..2}; do
     kubectl delete pod alertmanager-$i -n oran-monitoring
     kubectl wait --for=condition=ready pod/alertmanager-$i -n oran-monitoring --timeout=300s
   done
   ```

## High Severity Alerts Response

### HighErrorRate

**Alert**: Error rate is above acceptable threshold (>5%)

**Impact**: User experience degradation

**Response Steps**:

1. **Investigation**:
   ```bash
   # Identify error sources
   kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &

   # Query error rates by service
   curl "http://localhost:9090/api/v1/query?query=rate(http_requests_total{status=~\"4..|5..\"}[5m])/rate(http_requests_total[5m])"

   # Check application logs
   kubectl logs -l app.kubernetes.io/component=oran-nlp --tail=100
   kubectl logs -l app.kubernetes.io/component=oran-orchestrator --tail=100
   ```

2. **Common Causes**:
   - Database connectivity issues
   - Resource exhaustion
   - Configuration errors
   - External service dependencies

3. **Mitigation**:
   ```bash
   # Scale up if resource issue
   kubectl scale deployment oran-nlp --replicas=3

   # Check database connections
   kubectl exec deployment/oran-orchestrator -- \
     pg_isready -h postgresql -p 5432

   # Restart problematic services
   kubectl rollout restart deployment/oran-nlp
   ```

### MemoryUsageHigh

**Alert**: Memory usage above 85%

**Impact**: Potential OOMKill and service disruption

**Response Steps**:

1. **Immediate Assessment**:
   ```bash
   # Check current usage
   kubectl top pods -n oran-monitoring
   kubectl top nodes

   # Identify memory consumers
   kubectl get pods -n oran-monitoring --sort-by='.status.containerStatuses[0].usage.memory'
   ```

2. **Short-term Mitigation**:
   ```bash
   # Increase memory limits
   kubectl patch deployment prometheus -n oran-monitoring -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","resources":{"limits":{"memory":"8Gi"}}}]}}}}'

   # Scale horizontally if possible
   kubectl scale deployment grafana --replicas=2 -n oran-monitoring
   ```

3. **Long-term Solutions**:
   - Review resource allocation
   - Optimize queries and dashboards
   - Implement horizontal scaling
   - Consider node upgrades

### StorageSpaceHigh

**Alert**: Storage usage above 80%

**Impact**: Potential data loss and service failure

**Response Steps**:

1. **Immediate Actions**:
   ```bash
   # Check current usage
   kubectl exec deployment/prometheus -n oran-monitoring -- df -h /prometheus

   # Check retention settings
   kubectl describe deployment prometheus -n oran-monitoring | grep retention
   ```

2. **Emergency Cleanup**:
   ```bash
   # Reduce retention temporarily
   kubectl patch deployment prometheus -n oran-monitoring -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","args":["--storage.tsdb.retention.time=7d"]}]}}}}'

   # Manually clean old data
   kubectl exec deployment/prometheus -n oran-monitoring -- \
     find /prometheus -name "*.db" -mtime +7 -delete
   ```

3. **Permanent Solution**:
   ```bash
   # Expand PVC
   kubectl patch pvc prometheus-storage -n oran-monitoring -p \
     '{"spec":{"resources":{"requests":{"storage":"200Gi"}}}}'
   ```

## Performance Alerts Response

### SlowQueries

**Alert**: Query response time above 500ms

**Impact**: Dashboard and API performance degradation

**Response Steps**:

1. **Analysis**:
   ```bash
   # Identify slow queries
   kubectl port-forward -n oran-monitoring svc/prometheus 9090:9090 &
   curl "http://localhost:9090/api/v1/query?query=prometheus_http_request_duration_seconds{quantile=\"0.95\"}"

   # Check current query load
   curl "http://localhost:9090/api/v1/query?query=rate(prometheus_http_requests_total[5m])"
   ```

2. **Optimization**:
   ```bash
   # Increase query concurrency
   kubectl patch deployment prometheus -n oran-monitoring -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","args":["--query.max-concurrency=100"]}]}}}}'

   # Review and optimize dashboards
   # Implement recording rules for common queries
   ```

### HighLatency

**Alert**: Network or service latency above threshold

**Response Steps**:

1. **Network Check**:
   ```bash
   # Test connectivity
   kubectl exec deployment/prometheus -n oran-monitoring -- \
     ping -c 3 grafana.oran-monitoring.svc.cluster.local

   # Check DNS resolution
   kubectl exec deployment/prometheus -n oran-monitoring -- \
     nslookup grafana.oran-monitoring.svc.cluster.local
   ```

2. **Service Check**:
   ```bash
   # Check service endpoints
   kubectl get endpoints -n oran-monitoring

   # Test service connectivity
   kubectl exec deployment/prometheus -n oran-monitoring -- \
     curl -o /dev/null -s -w "%{time_total}" http://grafana:3000/api/health
   ```

## Security Alerts Response

### UnauthorizedAccess

**Alert**: Unusual access patterns detected

**Impact**: Potential security breach

**Response Steps**:

1. **Immediate Actions**:
   ```bash
   # Check recent access logs
   kubectl logs -n oran-monitoring deployment/grafana | grep -i "failed\|invalid\|unauthorized"

   # Review active sessions
   kubectl port-forward -n oran-monitoring svc/grafana 3000:3000 &
   curl -u admin:password "http://localhost:3000/api/user/auth-tokens"
   ```

2. **Security Measures**:
   ```bash
   # Change admin password
   kubectl exec deployment/grafana -n oran-monitoring -- \
     grafana-cli admin reset-admin-password new-secure-password

   # Revoke active sessions
   curl -X DELETE -u admin:new-password "http://localhost:3000/api/admin/user/{userid}/auth-tokens"

   # Enable additional logging
   kubectl patch deployment grafana -n oran-monitoring -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"grafana","env":[{"name":"GF_LOG_LEVEL","value":"debug"}]}]}}}}'
   ```

### CertificateExpiry

**Alert**: TLS certificates expiring soon

**Impact**: Service interruption when certificates expire

**Response Steps**:

1. **Check Certificate Status**:
   ```bash
   # Check certificate expiry
   kubectl get secret -n oran-monitoring prometheus-tls -o jsonpath='{.data.tls\.crt}' | \
     base64 -d | openssl x509 -text -noout | grep -A 2 "Validity"
   ```

2. **Renew Certificates**:
   ```bash
   # Generate new certificate
   openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
     -keyout prometheus.key -out prometheus.crt \
     -subj "/CN=prometheus.oran-monitoring.svc.cluster.local"

   # Update secret
   kubectl create secret tls prometheus-tls \
     --cert=prometheus.crt --key=prometheus.key \
     -n oran-monitoring --dry-run=client -o yaml | kubectl apply -f -

   # Restart services to pick up new certificates
   kubectl rollout restart deployment/prometheus -n oran-monitoring
   ```

## Communication Templates

### Initial Alert Notification

```
Subject: [CRITICAL] O-RAN Monitoring Alert - {AlertName}

Alert: {AlertName}
Severity: {Severity}
Status: Investigating
Time: {Timestamp}

Description: {AlertDescription}

Impact: {ImpactDescription}

Response Team: {ResponderName}
Estimated Resolution: {EstimatedTime}

Next Update: In 15 minutes

Dashboard: https://grafana.oran.company.com/d/alerts
Incident: #{IncidentNumber}
```

### Resolution Notification

```
Subject: [RESOLVED] O-RAN Monitoring Alert - {AlertName}

Alert: {AlertName}
Status: RESOLVED
Resolution Time: {ResolutionTime}
Duration: {AlertDuration}

Root Cause: {RootCause}

Actions Taken:
- {Action1}
- {Action2}
- {Action3}

Prevention Measures:
- {PreventionMeasure1}
- {PreventionMeasure2}

Post-Mortem: {PostMortemLink}
```

## Escalation Matrix

### Level 1 - On-Call Engineer (0-15 minutes)
- Initial response and basic troubleshooting
- Follow standard playbooks
- Gather initial information

### Level 2 - Senior Engineer (15-30 minutes)
- Complex troubleshooting
- Architecture decisions
- Coordinate with other teams

### Level 3 - Team Lead/Architect (30-60 minutes)
- Strategic decisions
- Resource allocation
- External vendor coordination

### Level 4 - Management (1+ hours)
- Business impact decisions
- Customer communication
- Post-incident review

## Post-Incident Activities

### Immediate (0-2 hours)
1. Verify full system recovery
2. Remove temporary fixes
3. Document timeline
4. Initial lessons learned

### Short-term (2-24 hours)
1. Detailed root cause analysis
2. Update monitoring and alerting
3. Implement immediate preventive measures
4. Team debrief

### Long-term (1-7 days)
1. Post-mortem report
2. Process improvements
3. Training updates
4. Infrastructure improvements

## Alert Tuning

### Regular Review Process
1. **Weekly**: Review alert frequency and accuracy
2. **Monthly**: Update thresholds based on baseline changes
3. **Quarterly**: Review and update playbooks

### Common Tuning Actions
```bash
# Adjust alert thresholds
kubectl patch prometheusrule monitoring-rules -n oran-monitoring --type='merge' -p \
  '{"spec":{"groups":[{"name":"oran.rules","rules":[{"alert":"HighMemoryUsage","expr":"memory_usage > 0.90"}]}]}}'

# Add alert inhibition rules
kubectl patch alertmanager-config -n oran-monitoring --type='merge' -p \
  '{"data":{"alertmanager.yml":"inhibit_rules:\n- source_match:\n    alertname: PrometheusDown\n  target_match:\n    alertname: TargetDown"}}'
```

### Metrics for Alert Quality
- **Precision**: % of alerts that required action
- **Recall**: % of actual issues that generated alerts
- **MTTD**: Mean Time To Detection
- **MTTR**: Mean Time To Resolution

This playbook should be regularly updated based on incident learnings and system changes.