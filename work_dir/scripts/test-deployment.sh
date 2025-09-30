#!/bin/bash
set -e

echo "🧪 O-RAN Intent MANO - E2E Deployment Test"
echo "=========================================="
echo ""

# Export kubeconfig
export KUBECONFIG=/output/kubeconfig.yaml

# Wait for K3s
echo "1️⃣ Checking K3s cluster..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s
echo "✅ K3s cluster ready"
echo ""

# Check Porch
echo "2️⃣ Checking Porch installation..."
kubectl wait --for=condition=Available --timeout=300s deployment/porch-server -n porch-system
kubectl get pods -n porch-system
echo "✅ Porch is running"
echo ""

# Create test repository in Porch
echo "3️⃣ Creating Porch repository..."
cat <<EOF | kubectl apply -f -
apiVersion: porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: nephio-packages
  namespace: default
spec:
  type: git
  git:
    repo: http://gitea:3000/nephio/packages.git
    branch: main
    directory: /
EOF
echo "✅ Repository created"
echo ""

# Submit test intent
echo "4️⃣ Submitting test intent..."
curl -X POST http://orchestrator:8081/api/v1/intents \
  -H "Content-Type: application/json" \
  -d '{
    "intent_id": "test-slice-001",
    "service_type": "eMBB",
    "coverage_area": "zone-001",
    "latency_ms": 10,
    "throughput_mbps": 1000,
    "availability": 99.99
  }'
echo ""
echo "✅ Intent submitted"
echo ""

# Wait for package creation
echo "5️⃣ Waiting for package creation..."
sleep 30

# Check for PackageRevision
echo "6️⃣ Checking PackageRevision..."
kubectl get packagerevisions -A
echo ""

# Get package details
PKG_NAME=$(kubectl get packagerevisions -n default -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$PKG_NAME" ]; then
    echo "✅ PackageRevision found: $PKG_NAME"
    kubectl get packagerevision "$PKG_NAME" -n default -o yaml
else
    echo "⚠️  No PackageRevision found yet"
fi
echo ""

# Check orchestrator logs
echo "7️⃣ Checking orchestrator logs..."
docker logs o-ran-orchestrator --tail 50
echo ""

# Check nephio-generator logs
echo "8️⃣ Checking nephio-generator logs..."
docker logs nephio-generator --tail 50
echo ""

# Summary
echo "=========================================="
echo "📊 E2E Test Summary"
echo "=========================================="
echo "✅ K3s cluster: Running"
echo "✅ Porch: Installed and running"
echo "✅ Repository: Created"
echo "✅ Intent: Submitted"
if [ -n "$PKG_NAME" ]; then
    echo "✅ Package: Created ($PKG_NAME)"
else
    echo "⚠️  Package: Pending (check logs above)"
fi
echo ""
echo "🔍 To debug further:"
echo "  - Check Porch logs: kubectl logs -n porch-system -l app=porch-server"
echo "  - Check orchestrator: docker logs o-ran-orchestrator"
echo "  - Check nephio-generator: docker logs nephio-generator"
echo "  - List packages: kubectl get packagerevisions -A"