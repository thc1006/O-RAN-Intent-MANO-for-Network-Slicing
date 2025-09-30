#!/bin/bash
#
# Performance Benchmark: Natural Language E2E Processing
# Tests NLP service and orchestrator performance under load
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🚀 O-RAN NLP E2E Performance Benchmark"
echo "========================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
NLP_URL="${NLP_URL:-http://localhost:8082}"
ORCH_URL="${ORCH_URL:-http://localhost:8080}"
RESULTS_DIR="${SCRIPT_DIR}/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}Configuration:${NC}"
echo "  NLP Service: $NLP_URL"
echo "  Orchestrator: $ORCH_URL"
echo "  Results: $RESULTS_DIR"
echo ""

# Test intents for different slice types
cat > /tmp/intent_embb.json << 'EOF'
{"intent": "Deploy high-bandwidth video streaming for 100 users", "session_id": "bench-embb"}
EOF

cat > /tmp/intent_urllc.json << 'EOF'
{"intent": "Deploy ultra-low latency slice for autonomous vehicles", "session_id": "bench-urllc"}
EOF

cat > /tmp/intent_mmtc.json << 'EOF'
{"intent": "Deploy IoT sensor network for smart city monitoring", "session_id": "bench-mmtc"}
EOF

# Function to run benchmark
run_benchmark() {
    local name=$1
    local url=$2
    local endpoint=$3
    local data_file=$4
    local requests=$5
    local concurrency=$6
    local output_file="${RESULTS_DIR}/${name}_${TIMESTAMP}.txt"

    echo -e "${YELLOW}Running: $name${NC}"
    echo "  Requests: $requests"
    echo "  Concurrency: $concurrency"
    echo "  Endpoint: $endpoint"
    echo ""

    # Run Apache Bench
    if command -v ab &> /dev/null; then
        ab -n "$requests" -c "$concurrency" \
           -p "$data_file" \
           -T "application/json" \
           -g "${output_file}.tsv" \
           "$url$endpoint" > "$output_file" 2>&1

        # Extract key metrics
        echo -e "${GREEN}Results:${NC}"
        grep "Requests per second:" "$output_file" || echo "  N/A"
        grep "Time per request:" "$output_file" | head -1 || echo "  N/A"
        grep "Transfer rate:" "$output_file" || echo "  N/A"
        echo ""
    else
        echo -e "${YELLOW}⚠ Apache Bench (ab) not found, using curl for basic test${NC}"

        local total_time=0
        local success=0

        for i in $(seq 1 "$requests"); do
            start=$(date +%s%N)
            response=$(curl -s -w "%{http_code}" -o /dev/null \
                -X POST "$url$endpoint" \
                -H "Content-Type: application/json" \
                -d @"$data_file" 2>&1)
            end=$(date +%s%N)

            time_ms=$(( (end - start) / 1000000 ))
            total_time=$(( total_time + time_ms ))

            if [ "$response" = "200" ] || [ "$response" = "201" ]; then
                success=$(( success + 1 ))
            fi

            if [ $(( i % 10 )) -eq 0 ]; then
                echo -n "."
            fi
        done
        echo ""

        avg_time=$(( total_time / requests ))
        success_rate=$(awk "BEGIN {printf \"%.2f\", ($success / $requests) * 100}")

        echo -e "${GREEN}Results:${NC}"
        echo "  Completed: $success/$requests"
        echo "  Success rate: $success_rate%"
        echo "  Average time: ${avg_time}ms"
        echo "  Total time: ${total_time}ms"
        echo ""

        # Save results
        cat > "$output_file" << EOF_RESULTS
Benchmark: $name
Timestamp: $TIMESTAMP
Requests: $requests
Concurrency: $concurrency (simulated)
Completed: $success/$requests
Success rate: $success_rate%
Average response time: ${avg_time}ms
Total time: ${total_time}ms
EOF_RESULTS
    fi
}

# Test 1: NLP Service Direct
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 1: NLP Service - Direct Parsing"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
run_benchmark "nlp_embb" "$NLP_URL" "/api/v1/parse" "/tmp/intent_embb.json" 100 10
run_benchmark "nlp_urllc" "$NLP_URL" "/api/v1/parse" "/tmp/intent_urllc.json" 100 10
run_benchmark "nlp_mmtc" "$NLP_URL" "/api/v1/parse" "/tmp/intent_mmtc.json" 100 10

# Test 2: Orchestrator E2E
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 2: Orchestrator - E2E Natural Language"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
run_benchmark "orch_embb" "$ORCH_URL" "/api/v1/intents/natural" "/tmp/intent_embb.json" 50 5
run_benchmark "orch_urllc" "$ORCH_URL" "/api/v1/intents/natural" "/tmp/intent_urllc.json" 50 5
run_benchmark "orch_mmtc" "$ORCH_URL" "/api/v1/intents/natural" "/tmp/intent_mmtc.json" 50 5

# Test 3: Health Endpoints
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 3: Health Check Performance"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

for i in {1..20}; do
    nlp_health=$(curl -s -w "%{time_total}" -o /dev/null "$NLP_URL/health" 2>&1)
    orch_health=$(curl -s -w "%{time_total}" -o /dev/null "$ORCH_URL/health" 2>&1)
    echo "Iteration $i - NLP: ${nlp_health}s, Orchestrator: ${orch_health}s"
done

# Cleanup
rm -f /tmp/intent_*.json

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Benchmark Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Results saved to: $RESULTS_DIR"
echo ""
echo "Summary files:"
ls -lh "$RESULTS_DIR"/*"$TIMESTAMP"* 2>/dev/null || echo "  No files generated"
echo ""
echo "To analyze results:"
echo "  cat $RESULTS_DIR/*_${TIMESTAMP}.txt"
echo ""
