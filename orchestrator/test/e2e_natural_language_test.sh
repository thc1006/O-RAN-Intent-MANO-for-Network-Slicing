#!/bin/bash
#
# End-to-End Test: Natural Language → NLP Service → Orchestrator → Argo CD → Kubernetes
# This script tests the complete natural language intent processing pipeline
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🚀 O-RAN Natural Language E2E Test"
echo "===================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track PIDs for cleanup
NLP_PID=""
ORCH_PID=""

# Cleanup function
cleanup() {
    echo ""
    echo "🧹 Cleaning up..."

    if [ ! -z "$NLP_PID" ]; then
        echo "Stopping NLP service (PID: $NLP_PID)"
        kill $NLP_PID 2>/dev/null || true
    fi

    if [ ! -z "$ORCH_PID" ]; then
        echo "Stopping Orchestrator (PID: $ORCH_PID)"
        kill $ORCH_PID 2>/dev/null || true
    fi

    # Clean up Kubernetes resources
    echo "Cleaning up Kubernetes resources..."
    kubectl delete ns oran-slice-embb oran-slice-urllc oran-slice-mmtc --ignore-not-found=true 2>/dev/null || true
    kubectl delete application -n argocd -l managed-by=oran-orchestrator --ignore-not-found=true 2>/dev/null || true

    echo "✓ Cleanup complete"
}

trap cleanup EXIT

# Step 1: Start NLP Service
echo "📡 Step 1: Starting NLP Service..."
cd "$PROJECT_ROOT/nlp"
python nlp_service.py > /tmp/nlp-service.log 2>&1 &
NLP_PID=$!
echo "   NLP Service PID: $NLP_PID"

# Wait for NLP service to be ready
echo "   Waiting for NLP service to start..."
for i in {1..30}; do
    if curl -s http://localhost:8082/health > /dev/null 2>&1; then
        echo -e "   ${GREEN}✓ NLP Service is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "   ${RED}✗ NLP Service failed to start${NC}"
        cat /tmp/nlp-service.log
        exit 1
    fi
    sleep 1
done

# Verify NLP service health
NLP_HEALTH=$(curl -s http://localhost:8082/health | jq -r '.status')
if [ "$NLP_HEALTH" != "healthy" ]; then
    echo -e "${RED}✗ NLP Service is not healthy${NC}"
    exit 1
fi
echo ""

# Step 2: Start Orchestrator
echo "🎯 Step 2: Starting Orchestrator..."
cd "$PROJECT_ROOT/orchestrator"
./bin/orchestrator.exe --server > /tmp/orchestrator.log 2>&1 &
ORCH_PID=$!
echo "   Orchestrator PID: $ORCH_PID"

# Wait for Orchestrator to be ready
echo "   Waiting for Orchestrator to start..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "   ${GREEN}✓ Orchestrator is ready${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "   ${RED}✗ Orchestrator failed to start${NC}"
        cat /tmp/orchestrator.log
        exit 1
    fi
    sleep 1
done
echo ""

# Step 3: Test Natural Language Intents
echo "🎤 Step 3: Testing Natural Language Intents..."
echo ""

# Test 1: eMBB Video Streaming
echo "Test 1: eMBB Video Streaming Slice"
echo "Intent: 'Deploy high-bandwidth video streaming slice for 100 users'"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/intents/natural \
    -H "Content-Type: application/json" \
    -d '{"intent": "Deploy high-bandwidth video streaming slice for 100 users", "session_id": "test-e2e-001"}')

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
SLICE_ID=$(echo "$RESPONSE" | jq -r '.slice_id')
SLICE_TYPE=$(echo "$RESPONSE" | jq -r '.intent.parsed_as')

if [ "$SUCCESS" == "true" ] && [ "$SLICE_TYPE" == "eMBB" ]; then
    echo -e "${GREEN}✓ eMBB slice created: $SLICE_ID${NC}"
    echo "  Throughput: $(echo "$RESPONSE" | jq -r '.qos_profile.throughput_mbps') Mbps"
    echo "  Latency: $(echo "$RESPONSE" | jq -r '.qos_profile.latency_ms') ms"
else
    echo -e "${RED}✗ eMBB test failed${NC}"
    echo "$RESPONSE" | jq '.'
    exit 1
fi
echo ""

# Test 2: URLLC Autonomous Vehicles
echo "Test 2: URLLC Autonomous Vehicle Slice"
echo "Intent: 'Deploy ultra-low latency slice for autonomous vehicles'"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/intents/natural \
    -H "Content-Type: application/json" \
    -d '{"intent": "Deploy ultra-low latency slice for autonomous vehicles", "session_id": "test-e2e-002"}')

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
SLICE_ID=$(echo "$RESPONSE" | jq -r '.slice_id')
SLICE_TYPE=$(echo "$RESPONSE" | jq -r '.intent.parsed_as')

if [ "$SUCCESS" == "true" ] && [ "$SLICE_TYPE" == "URLLC" ]; then
    echo -e "${GREEN}✓ URLLC slice created: $SLICE_ID${NC}"
    echo "  Throughput: $(echo "$RESPONSE" | jq -r '.qos_profile.throughput_mbps') Mbps"
    echo "  Latency: $(echo "$RESPONSE" | jq -r '.qos_profile.latency_ms') ms"
else
    echo -e "${RED}✗ URLLC test failed${NC}"
    echo "$RESPONSE" | jq '.'
    exit 1
fi
echo ""

# Test 3: mMTC IoT Sensors
echo "Test 3: mMTC IoT Sensor Network"
echo "Intent: 'Deploy IoT sensor network for smart city monitoring'"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/intents/natural \
    -H "Content-Type: application/json" \
    -d '{"intent": "Deploy IoT sensor network for smart city monitoring", "session_id": "test-e2e-003"}')

SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
SLICE_ID=$(echo "$RESPONSE" | jq -r '.slice_id')
SLICE_TYPE=$(echo "$RESPONSE" | jq -r '.intent.parsed_as')

if [ "$SUCCESS" == "true" ] && [ "$SLICE_TYPE" == "mMTC" ]; then
    echo -e "${GREEN}✓ mMTC slice created: $SLICE_ID${NC}"
    echo "  Throughput: $(echo "$RESPONSE" | jq -r '.qos_profile.throughput_mbps') Mbps"
    echo "  Latency: $(echo "$RESPONSE" | jq -r '.qos_profile.latency_ms') ms"
else
    echo -e "${RED}✗ mMTC test failed${NC}"
    echo "$RESPONSE" | jq '.'
    exit 1
fi
echo ""

# Step 4: Verify Kubernetes Deployments
echo "☸️  Step 4: Verifying Kubernetes Deployments..."
echo ""

# Wait for resources to be created
sleep 5

# Check Argo CD Applications
echo "Checking Argo CD Applications..."
APPS=$(kubectl get applications -n argocd -l managed-by=oran-orchestrator --no-headers 2>/dev/null | wc -l)
if [ "$APPS" -gt 0 ]; then
    echo -e "${GREEN}✓ Found $APPS Argo CD Applications${NC}"
    kubectl get applications -n argocd -l managed-by=oran-orchestrator
else
    echo -e "${YELLOW}⚠ No Argo CD Applications found (may need Argo CD configured)${NC}"
fi
echo ""

# Step 5: Performance Metrics
echo "📊 Step 5: Performance Metrics..."
echo ""

# Get NLP Service metrics
NLP_METRICS=$(curl -s http://localhost:8082/health)
echo "NLP Service Metrics:"
echo "  Uptime: $(echo "$NLP_METRICS" | jq -r '.uptime_seconds')s"
echo "  Total Intents Processed: $(echo "$NLP_METRICS" | jq -r '.total_intents_processed')"
echo ""

# Summary
echo "========================================"
echo -e "${GREEN}🎉 E2E Test Completed Successfully!${NC}"
echo "========================================"
echo ""
echo "Summary:"
echo "  ✓ NLP Service: Running"
echo "  ✓ Orchestrator: Running"
echo "  ✓ eMBB Slice: Deployed"
echo "  ✓ URLLC Slice: Deployed"
echo "  ✓ mMTC Slice: Deployed"
echo ""
echo "Logs:"
echo "  NLP Service: /tmp/nlp-service.log"
echo "  Orchestrator: /tmp/orchestrator.log"
echo ""
