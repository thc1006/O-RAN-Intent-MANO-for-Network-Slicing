# TN Manager and Agent Makefile

.PHONY: all manager agent test clean

MANAGER_IMG ?= tn-manager:latest
AGENT_IMG ?= tn-agent:latest

all: manager agent

# Build TN Manager
manager:
	@echo "Building TN Manager..."
	cd manager && go build -o bin/manager main.go

# Build TN Agent
agent:
	@echo "Building TN Agent..."
	cd agent && go build -o bin/agent main.go

# Docker images
docker-build-manager:
	docker build -t $(MANAGER_IMG) -f manager/Dockerfile manager/

docker-build-agent:
	docker build -t $(AGENT_IMG) -f agent/Dockerfile agent/

docker-build: docker-build-manager docker-build-agent

# Kind cluster setup
kind-setup:
	@echo "Creating Kind cluster..."
	cd tests/e2e && bash setup.sh

kind-load: docker-build
	@echo "Loading images into Kind..."
	kind load docker-image $(MANAGER_IMG) --name tn-test
	kind load docker-image $(AGENT_IMG) --name tn-test

# Deploy to Kind
deploy: kind-load
	@echo "Deploying TN components..."
	kubectl apply -f manager/config/crd/bases/
	kubectl apply -f manager/config/samples/

# Run tests
test-unit:
	@echo "Running unit tests..."
	cd manager && go test ./...
	cd agent && go test ./...

test-iperf: kind-setup deploy
	@echo "Running iPerf3 performance tests..."
	cd tests/iperf && go test -v -timeout 30m

test: test-unit test-iperf

# Performance validation targets
validate-embb:
	@echo "Validating eMBB profile (4.57 Mbps, 16.1ms)..."
	kubectl apply -f manager/config/samples/tnslice_embb.yaml
	@sleep 10
	cd tests/iperf && go test -run TestEMBB -v

validate-urllc:
	@echo "Validating uRLLC profile (0.93 Mbps, 6.3ms)..."
	kubectl apply -f manager/config/samples/tnslice_urllc.yaml
	@sleep 10
	cd tests/iperf && go test -run TestURLLC -v

validate-miot:
	@echo "Validating mIoT profile (2.77 Mbps, 15.7ms)..."
	kubectl apply -f manager/config/samples/tnslice_miot.yaml
	@sleep 10
	cd tests/iperf && go test -run TestMIOT -v

validate-all: validate-embb validate-urllc validate-miot
	@echo "All performance validations complete!"

# Cleanup
clean:
	@echo "Cleaning up..."
	kind delete cluster --name tn-test || true
	rm -f manager/bin/manager agent/bin/agent

clean-slices:
	kubectl delete tnslices --all -n default || true

# Debug helpers
logs-manager:
	kubectl logs -n tn-system deployment/tn-manager -f

logs-agent:
	kubectl logs -n tn-system ds/tn-agent -f

tc-status:
	@echo "Checking TC status on nodes..."
	kubectl exec -n tn-system ds/tn-agent -- tc qdisc show

vxlan-status:
	@echo "Checking VXLAN interfaces..."
	kubectl exec -n tn-system ds/tn-agent -- ip link show type vxlan

# Help
help:
	@echo "TN Manager and Agent Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make all              - Build manager and agent"
	@echo "  make docker-build     - Build Docker images"
	@echo "  make kind-setup       - Create Kind test cluster"
	@echo "  make deploy           - Deploy to Kind cluster"
	@echo "  make test             - Run all tests"
	@echo "  make validate-all     - Validate all performance profiles"
	@echo "  make clean            - Clean up everything"
	@echo ""
	@echo "Performance Profiles:"
	@echo "  eMBB:  4.57 Mbps, 16.1ms latency"
	@echo "  mIoT:  2.77 Mbps, 15.7ms latency"
	@echo "  uRLLC: 0.93 Mbps, 6.3ms latency"