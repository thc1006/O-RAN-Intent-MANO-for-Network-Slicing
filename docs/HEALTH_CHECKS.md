# O-RAN Intent-MANO Health Check Procedures

## Overview

This document provides comprehensive health check procedures for the O-RAN Intent-MANO system, including service health monitoring, system diagnostics, performance monitoring, and automated health validation.

## Health Check Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Application    │    │   System        │    │   Network       │
│  Health Checks  │    │   Health        │    │   Health        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Prometheus    │◄──►│   Health        │◄──►│    Grafana      │
│   Metrics       │    │   Monitor       │    │   Dashboards    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Service Health Checks

### HTTP Health Endpoints

Each service exposes standardized health endpoints:

| Service | Health Endpoint | Readiness Endpoint | Metrics Endpoint |
|---------|-----------------|-------------------|------------------|
| Orchestrator | `/health` | `/ready` | `/metrics` |
| VNF Operator | `/healthz` | `/readyz` | `/metrics` |
| O2 Client | `/health` | `/ready` | `/metrics` |
| TN Manager | `/health` | `/ready` | `/metrics` |
| TN Agent E01 | `/health` | `/ready` | `/metrics` |
| TN Agent E02 | `/health` | `/ready` | `/metrics` |
| RAN DMS | `/health` | `/ready` | `/metrics` |
| CN DMS | `/health` | `/ready` | `/metrics` |

### Quick Health Check Script

```bash
#!/bin/bash
# Quick health check for all services

SERVICES=(
    "orchestrator:8080:/health"
    "vnf-operator:8081:/healthz"
    "o2-client:8083:/health"
    "tn-manager:8084:/health"
    "tn-agent-edge01:8085:/health"
    "tn-agent-edge02:8086:/health"
    "ran-dms:8087:/health"
    "cn-dms:8088:/health"
    "prometheus:9090:/-/healthy"
    "grafana:3000:/api/health"
)

echo "O-RAN MANO Health Check Report"
echo "=============================="
echo "Timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo ""

healthy=0
total=${#SERVICES[@]}

for service_info in "${SERVICES[@]}"; do
    IFS=':' read -r service port path <<< "$service_info"

    if curl -f -s --max-time 5 "http://localhost:${port}${path}" > /dev/null 2>&1; then
        echo "✓ $service - HEALTHY"
        ((healthy++))
    else
        echo "✗ $service - UNHEALTHY"
    fi
done

echo ""
echo "Summary: $healthy/$total services healthy"

if [ $healthy -eq $total ]; then
    echo "Overall Status: HEALTHY"
    exit 0
else
    echo "Overall Status: UNHEALTHY"
    exit 1
fi
```

### Individual Service Health Checks

#### Orchestrator Health Check

```bash
# Basic health check
curl -f http://localhost:8080/health

# Expected response:
{
    "status": "healthy",
    "timestamp": "2024-09-25T10:30:00Z",
    "version": "v1.0.0",
    "checks": {
        "database": "healthy",
        "dms_connectivity": "healthy",
        "memory_usage": "healthy",
        "cpu_usage": "healthy"
    }
}

# Detailed readiness check
curl -f http://localhost:8080/ready

# Performance metrics
curl http://localhost:9090/metrics | grep orchestrator_
```

#### VNF Operator Health Check

```bash
# Health check
curl -f http://localhost:8081/healthz

# Readiness check
curl -f http://localhost:8082/readyz

# Leader election status (Kubernetes only)
kubectl get lease -n oran-mano oran-vnf-operator-leader-election
```

#### DMS Health Checks

```bash
# RAN DMS
curl -f http://localhost:8087/health
curl -f -k https://localhost:8443/health  # HTTPS

# CN DMS
curl -f http://localhost:8088/health
curl -f -k https://localhost:8444/health  # HTTPS

# Database connectivity
curl -f http://localhost:8087/api/v1/status
curl -f http://localhost:8088/api/v1/status
```

#### Transport Network Health Checks

```bash
# TN Manager
curl -f http://localhost:8084/health

# Check TN topology
curl -f http://localhost:8084/api/v1/topology

# TN Agents
curl -f http://localhost:8085/health  # Edge01
curl -f http://localhost:8086/health  # Edge02

# Check iPerf3 servers
nc -zv localhost 5201  # Edge01 iPerf3
nc -zv localhost 5202  # Edge02 iPerf3
```

## System Health Monitoring

### Automated Health Monitor

The system includes an automated health monitor that continuously checks all services:

```bash
# Start health monitor
docker-compose --profile monitoring up -d health-monitor

# View health monitor logs
docker logs -f oran-health-monitor

# Check generated reports
ls deploy/docker/test-results/health-*.json
```

### Health Monitor Configuration

```bash
# Environment variables for health monitor
export MONITOR_INTERVAL=30          # Check interval in seconds
export RESULTS_DIR=/results         # Results directory
export LOG_LEVEL=INFO              # Logging level

# Service endpoints (automatically discovered from Docker Compose)
export ORCHESTRATOR_URL=http://orchestrator:8080
export RAN_DMS_URL=http://ran-dms:8080
export CN_DMS_URL=http://cn-dms:8080
```

### Health Check Reports

The health monitor generates JSON reports:

```json
{
  "timestamp": "2024-09-25T10:30:00Z",
  "monitor_version": "1.0.0",
  "services": {
    "orchestrator": {
      "status": "HEALTHY",
      "endpoint": "http://orchestrator:8080/health",
      "response_time_ms": 45,
      "error": "",
      "last_check": "2024-09-25T10:30:00Z"
    },
    "vnf-operator": {
      "status": "HEALTHY",
      "endpoint": "http://vnf-operator:8081/healthz",
      "response_time_ms": 23,
      "error": "",
      "last_check": "2024-09-25T10:30:00Z"
    }
  },
  "summary": {
    "total_services": 10,
    "healthy_services": 10,
    "unhealthy_services": 0,
    "overall_status": "HEALTHY"
  }
}
```

## Performance Health Monitoring

### Key Performance Metrics

#### Response Time Monitoring

```bash
# Monitor API response times
curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:8080/health"

# curl-format.txt content:
#      time_namelookup:  %{time_namelookup}\n
#         time_connect:  %{time_connect}\n
#      time_appconnect:  %{time_appconnect}\n
#     time_pretransfer:  %{time_pretransfer}\n
#        time_redirect:  %{time_redirect}\n
#   time_starttransfer:  %{time_starttransfer}\n
#                     ----------\n
#           time_total:  %{time_total}\n
```

#### Resource Utilization Monitoring

```bash
# CPU and memory usage per service
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"

# Expected output:
# CONTAINER           CPU %     MEM USAGE / LIMIT     MEM %
# oran-orchestrator   2.34%     245.7MiB / 512MiB     48.0%
# oran-vnf-operator   1.12%     128.3MiB / 512MiB     25.1%
# oran-ran-dms        0.67%     156.2MiB / 512MiB     30.5%

# Disk usage
df -h
docker system df

# Network statistics
docker exec oran-orchestrator cat /proc/net/dev
```

#### Database Health Monitoring

```bash
# RAN DMS database
curl -s http://localhost:8087/api/v1/metrics | grep db_
# db_connections_active 5
# db_connections_idle 2
# db_query_duration_seconds 0.023

# CN DMS database
curl -s http://localhost:8088/api/v1/metrics | grep db_
```

### Performance Thresholds

| Metric | Warning Threshold | Critical Threshold | Action |
|--------|------------------|-------------------|---------|
| API Response Time | >500ms | >2s | Scale service |
| CPU Usage | >70% | >90% | Add resources |
| Memory Usage | >80% | >95% | Scale/restart |
| Disk Usage | >80% | >90% | Clean logs |
| Database Connections | >80% pool | >95% pool | Tune pool |
| Error Rate | >5% | >10% | Investigate |

## Network Health Checks

### Service Connectivity Matrix

Test connectivity between all service pairs:

```bash
#!/bin/bash
# Service connectivity matrix test

SERVICES=("orchestrator" "vnf-operator" "o2-client" "tn-manager" "ran-dms" "cn-dms")
CONNECTIVITY_TESTS=(
    "orchestrator:ran-dms:8080"
    "orchestrator:cn-dms:8080"
    "vnf-operator:ran-dms:8080"
    "o2-client:ran-dms:8080"
    "o2-client:cn-dms:8080"
    "tn-manager:orchestrator:8080"
)

echo "Service Connectivity Matrix"
echo "=========================="

for test in "${CONNECTIVITY_TESTS[@]}"; do
    IFS=':' read -r from to port <<< "$test"

    if docker exec oran-$from nc -zv $to $port 2>/dev/null; then
        echo "✓ $from -> $to:$port"
    else
        echo "✗ $from -> $to:$port"
    fi
done
```

### Network Performance Tests

```bash
# Latency between TN agents
docker exec oran-tn-agent-edge01 ping -c 10 tn-agent-edge02

# Throughput between agents
docker exec oran-tn-agent-edge01 iperf3 -s -D  # Start server
docker exec oran-tn-agent-edge02 iperf3 -c tn-agent-edge01 -t 10

# DNS resolution
docker exec oran-orchestrator nslookup ran-dms
docker exec oran-orchestrator nslookup cn-dms
```

### Network Policy Validation

```bash
# Verify network policies are working (Kubernetes)
# Should succeed - allowed communication
kubectl exec -n oran-mano deployment/oran-orchestrator -- nc -zv oran-ran-dms 8080

# Should fail - blocked by network policy
kubectl exec -n oran-edge deployment/oran-tn-agent-edge01 -- nc -zv oran-orchestrator 8080 2>&1 | grep -q "Connection refused" && echo "✓ Network policy working" || echo "✗ Network policy not enforced"
```

## Database Health Monitoring

### Database Connection Health

```bash
# Check database connections
curl -s http://localhost:8087/api/v1/status | jq '.database'
curl -s http://localhost:8088/api/v1/status | jq '.database'

# Expected response:
{
  "status": "healthy",
  "connections": {
    "active": 3,
    "idle": 7,
    "total": 10
  },
  "last_query": "2024-09-25T10:30:00Z",
  "query_duration_ms": 23
}
```

### Database Performance Metrics

```bash
# Query performance metrics from DMS services
curl -s http://localhost:8087/metrics | grep -E "(db_query_duration|db_connections)"
curl -s http://localhost:8088/metrics | grep -E "(db_query_duration|db_connections)"

# Slow query detection
curl -s http://localhost:8087/api/v1/slow-queries
```

## Security Health Checks

### Certificate Health

```bash
# Check certificate expiration
openssl x509 -in deploy/docker/certs/server.crt -text -noout | grep "Not After"

# TLS connectivity tests
openssl s_client -connect localhost:8443 -servername ran-dms < /dev/null
openssl s_client -connect localhost:8444 -servername cn-dms < /dev/null
```

### Security Compliance Checks

```bash
# Container security
docker scan oran-orchestrator:latest
docker scan oran-vnf-operator:latest

# Pod security (Kubernetes)
kubectl get pod -n oran-mano -o jsonpath='{.items[*].spec.securityContext}'

# Network policy enforcement
kubectl get networkpolicy -n oran-mano
```

## Alerting and Notifications

### Prometheus Alert Rules

The system includes pre-configured alert rules in `deploy/docker/configs/prometheus/alert_rules.yml`:

- Service Down
- High Memory Usage (>80%)
- High CPU Usage (>80%)
- Disk Space Low (>80%)
- High API Latency (>500ms)
- High Error Rate (>10%)

### Alert Status Check

```bash
# Check active alerts
curl -s http://localhost:9090/api/v1/alerts | jq '.data[].alerts[]'

# Check alert rules
curl -s http://localhost:9090/api/v1/rules | jq '.data.groups[].rules[]'
```

## Automated Health Validation

### Continuous Health Monitoring

```bash
# Deploy health monitor with alerting
docker-compose --profile monitoring up -d health-monitor

# Health monitor will:
# 1. Check service health every 30 seconds
# 2. Generate health reports
# 3. Send alerts for failures
# 4. Create daily summary reports
```

### Health Check API

The health monitor exposes an API for external monitoring:

```bash
# Get overall health status
curl http://localhost:8089/health

# Get detailed service status
curl http://localhost:8089/services

# Get historical health data
curl http://localhost:8089/history?hours=24
```

## Troubleshooting Health Issues

### Common Health Check Failures

#### Service Unavailable

```bash
# Check if service is running
docker-compose ps orchestrator

# Check service logs
docker-compose logs orchestrator

# Check port binding
netstat -tulpn | grep :8080

# Restart service
docker-compose restart orchestrator
```

#### Database Connection Issues

```bash
# Check database health
curl -s http://localhost:8087/api/v1/status

# Check database logs
docker-compose logs ran-dms | grep -i error

# Reset database connection pool
curl -X POST http://localhost:8087/api/v1/reset-connections
```

#### High Resource Usage

```bash
# Identify resource-hungry containers
docker stats --no-stream | sort -k3 -hr

# Scale down if needed
docker-compose up -d --scale orchestrator=1

# Clean up resources
docker system prune -f
```

#### Network Connectivity Issues

```bash
# Check Docker networks
docker network ls
docker network inspect deploy_mano-net

# Test DNS resolution
docker exec oran-orchestrator nslookup ran-dms

# Check firewall rules
sudo iptables -L
```

### Health Check Debugging

```bash
# Enable debug logging for health checks
export LOG_LEVEL=DEBUG

# Restart health monitor with debug mode
docker-compose restart health-monitor

# View detailed logs
docker logs -f oran-health-monitor
```

### Recovery Procedures

#### Automated Recovery

```bash
# Health monitor includes automatic recovery for:
# - Service restarts on health check failures
# - Database connection pool resets
# - Memory cleanup for high usage
# - Log rotation for disk space

# Manual recovery trigger
curl -X POST http://localhost:8089/recover
```

#### Manual Recovery Steps

```bash
# Step 1: Stop all services
docker-compose down

# Step 2: Clean up resources
docker system prune -f
docker volume prune -f

# Step 3: Restart infrastructure
docker-compose up -d ran-dms cn-dms prometheus grafana

# Step 4: Wait for health
sleep 30

# Step 5: Start core services
docker-compose up -d orchestrator vnf-operator o2-client tn-manager

# Step 6: Start edge services
docker-compose up -d tn-agent-edge01 tn-agent-edge02

# Step 7: Verify health
./deploy/scripts/deploy-local.sh health
```

## Health Metrics Dashboard

### Grafana Health Dashboards

Access pre-configured health dashboards:

1. **System Overview**: http://localhost:3000/d/system-overview
   - Service health status
   - Response time trends
   - Error rate monitoring

2. **Resource Utilization**: http://localhost:3000/d/resources
   - CPU and memory usage
   - Disk space utilization
   - Network I/O statistics

3. **Service Details**: http://localhost:3000/d/service-details
   - Per-service metrics
   - Database performance
   - API request patterns

### Custom Health Queries

```promql
# Service availability over time
up{job=~"oran-.*"}

# Average response time by service
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# Error rate percentage
100 * (rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]))

# Memory usage percentage
100 * (container_memory_working_set_bytes / container_spec_memory_limit_bytes)

# Database connection utilization
db_connections_active / db_connections_max * 100
```

## Health Check Best Practices

### Design Principles

1. **Fast Responses**: Health checks should complete within 5 seconds
2. **Lightweight**: Minimal resource consumption
3. **Comprehensive**: Cover all critical dependencies
4. **Actionable**: Provide clear status and error information
5. **Consistent**: Standardized across all services

### Implementation Guidelines

1. **Graceful Degradation**: Services should handle partial failures
2. **Circuit Breakers**: Prevent cascading failures
3. **Retry Logic**: Handle transient failures
4. **Timeout Handling**: Prevent hanging health checks
5. **Caching**: Cache health status for performance

### Monitoring Strategy

1. **Multi-Level Monitoring**: Application, system, and network levels
2. **Proactive Alerting**: Alert on trends, not just failures
3. **Historical Analysis**: Track health trends over time
4. **Automated Recovery**: Self-healing where possible
5. **Regular Reviews**: Adjust thresholds based on operational data

This comprehensive health check framework ensures the O-RAN Intent-MANO system maintains high availability and reliability while providing operators with clear visibility into system health and performance.