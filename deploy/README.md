# O-RAN Intent MANO Deployment Guide

This directory contains deployment configurations for the O-RAN Intent-Based MANO system with natural language processing capabilities.

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                   │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌──────────────┐         ┌─────────────────┐           │
│  │ NLP Service  │◄────────┤  Orchestrator   │           │
│  │ (FastAPI)    │  HTTP   │  (Go)           │           │
│  │ Port: 8082   │         │  Port: 8080     │           │
│  └──────────────┘         └────────┬────────┘           │
│  Namespace:                        │                     │
│  oran-nlp                          │ Argo CD API        │
│                                    ▼                     │
│                        ┌────────────────────┐            │
│                        │    Argo CD         │            │
│                        │    Applications    │            │
│                        └─────────┬──────────┘            │
│                                  │                       │
│                                  ▼                       │
│                    ┌──────────────────────┐              │
│                    │  Network Slices      │              │
│                    │  (eMBB/URLLC/mMTC)   │              │
│                    └──────────────────────┘              │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

## 📦 Deployment Options

### Option 1: Docker Compose (Development)

**Prerequisites:**
- Docker 24.0+
- Docker Compose 2.0+

**Deploy:**
```bash
cd deploy
docker-compose up -d
```

**Access:**
- NLP Service: http://localhost:8082
- Orchestrator: http://localhost:8080
- Metrics: http://localhost:9090/metrics

**Test:**
```bash
# Check services
curl http://localhost:8082/health
curl http://localhost:8080/health

# Send natural language intent
curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy high-bandwidth video streaming for 100 users"}'
```

**Stop:**
```bash
docker-compose down
```

### Option 2: Kubernetes (Production)

**Prerequisites:**
- Kubernetes 1.28+
- kubectl configured
- Argo CD installed in `argocd` namespace

**Step 1: Deploy NLP Service**
```bash
kubectl apply -f k8s/nlp/deployment.yaml
```

**Step 2: Deploy Orchestrator**
```bash
kubectl apply -f k8s/orchestrator/deployment.yaml
```

**Step 3: Verify Deployments**
```bash
# Check pods
kubectl get pods -n oran-nlp
kubectl get pods -n oran-orchestrator

# Check services
kubectl get svc -n oran-nlp
kubectl get svc -n oran-orchestrator
```

**Step 4: Access Services**
```bash
# Port forward NLP service
kubectl port-forward -n oran-nlp svc/nlp-service 8082:8082

# Port forward Orchestrator
kubectl port-forward -n oran-orchestrator svc/orchestrator 8080:8080

# Or use NodePort
# NLP: http://<node-ip>:30082
# Orchestrator: http://<node-ip>:30080
```

**Step 5: Test End-to-End**
```bash
# Send natural language intent
curl -X POST http://localhost:8080/api/v1/intents/natural \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy ultra-low latency slice for autonomous vehicles",
    "session_id": "prod-session-001"
  }'

# Check Argo CD applications
kubectl get applications -n argocd -l managed-by=oran-orchestrator

# Verify deployed slices
kubectl get ns | grep oran-slice
```

## 🔧 Configuration

### NLP Service Configuration

**Environment Variables:**
- `NLP_LOG_LEVEL`: Log level (debug, info, warning, error) - Default: info
- `NLP_PORT`: Service port - Default: 8082

**ConfigMap:** `k8s/nlp/deployment.yaml`

### Orchestrator Configuration

**Environment Variables:**
- `NLP_SERVICE_URL`: NLP service URL - Default: http://nlp-service.oran-nlp.svc.cluster.local:8082
- `ARGOCD_NAMESPACE`: Argo CD namespace - Default: argocd
- `LOG_LEVEL`: Log level - Default: info

**ConfigMap:** `k8s/orchestrator/deployment.yaml`

## 🔐 Security

### RBAC Permissions

The orchestrator requires the following Kubernetes permissions:

- **Argo CD Applications**: CRUD operations
- **ConfigMaps**: CRUD operations
- **Namespaces**: Read and create
- **Deployments/Services**: Read access

See `k8s/orchestrator/deployment.yaml` for full RBAC configuration.

### Network Policies

For production deployments, consider adding NetworkPolicies:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nlp-service-policy
  namespace: oran-nlp
spec:
  podSelector:
    matchLabels:
      app: nlp-service
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: oran-orchestrator
    ports:
    - protocol: TCP
      port: 8082
```

## 📊 Monitoring

### Prometheus Metrics

Both services expose Prometheus metrics:

**NLP Service:**
- Endpoint: http://nlp-service:8082/metrics
- Metrics: Intent processing times, success rates

**Orchestrator:**
- Endpoint: http://orchestrator:9090/metrics
- Metrics: Intent processing duration, slice deployments, active slices

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-services
  namespace: monitoring
spec:
  selector:
    matchLabels:
      component: nlp
  endpoints:
  - port: http
    path: /metrics
```

## 🧪 Testing

### Integration Tests

```bash
# Run E2E test script
cd ../orchestrator/test
bash e2e_natural_language_test.sh
```

### Performance Testing

```bash
# Load test with Apache Bench
ab -n 1000 -c 10 -p intent.json -T application/json \
   http://localhost:8080/api/v1/intents/natural

# intent.json content:
# {"intent": "Deploy high-bandwidth video streaming"}
```

## 🔄 Scaling

### Horizontal Pod Autoscaler

Both services include HPA configurations:

**NLP Service:**
- Min replicas: 2
- Max replicas: 10
- Target CPU: 70%
- Target Memory: 80%

**Orchestrator:**
- Min replicas: 2
- Max replicas: 10
- Target CPU: 70%
- Target Memory: 80%

### Manual Scaling

```bash
# Scale NLP service
kubectl scale deployment nlp-service -n oran-nlp --replicas=5

# Scale orchestrator
kubectl scale deployment orchestrator -n oran-orchestrator --replicas=5
```

## 🐛 Troubleshooting

### Check Logs

```bash
# NLP service logs
kubectl logs -n oran-nlp -l app=nlp-service --tail=100 -f

# Orchestrator logs
kubectl logs -n oran-orchestrator -l app=orchestrator --tail=100 -f
```

### Common Issues

**Issue: NLP service unhealthy**
```bash
# Check pod status
kubectl describe pod -n oran-nlp <pod-name>

# Check health endpoint
kubectl exec -n oran-nlp <pod-name> -- curl localhost:8082/health
```

**Issue: Orchestrator can't reach NLP service**
```bash
# Test service DNS
kubectl exec -n oran-orchestrator <pod-name> -- \
  nslookup nlp-service.oran-nlp.svc.cluster.local

# Test connectivity
kubectl exec -n oran-orchestrator <pod-name> -- \
  curl http://nlp-service.oran-nlp.svc.cluster.local:8082/health
```

**Issue: Argo CD application not created**
```bash
# Check RBAC permissions
kubectl auth can-i create applications.argoproj.io \
  --as=system:serviceaccount:oran-orchestrator:orchestrator -n argocd

# Check Argo CD
kubectl get applications -n argocd
kubectl describe application <app-name> -n argocd
```

## 🔄 Updates

### Rolling Updates

```bash
# Update NLP service image
kubectl set image deployment/nlp-service \
  nlp-service=oran-nlp-service:v1.1.0 -n oran-nlp

# Update orchestrator image
kubectl set image deployment/orchestrator \
  orchestrator=oran-orchestrator:v0.2.0 -n oran-orchestrator

# Check rollout status
kubectl rollout status deployment/nlp-service -n oran-nlp
kubectl rollout status deployment/orchestrator -n oran-orchestrator
```

### Rollback

```bash
# Rollback NLP service
kubectl rollout undo deployment/nlp-service -n oran-nlp

# Rollback orchestrator
kubectl rollout undo deployment/orchestrator -n oran-orchestrator
```

## 📚 Additional Resources

- [Argo CD Documentation](https://argo-cd.readthedocs.io/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [Prometheus Monitoring](https://prometheus.io/docs/introduction/overview/)

## 🆘 Support

For issues and questions:
- GitHub Issues: https://github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing/issues
- Documentation: ../docs/

---

**Last Updated:** October 2025
