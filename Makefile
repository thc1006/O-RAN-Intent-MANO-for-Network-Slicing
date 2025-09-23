# O-RAN Intent MANO Makefile
# Reproducible development environment setup

SHELL := /bin/bash
.PHONY: help setup check kind k3s clean test lint format license spell install-tools verify-env

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

# Configuration
PYTHON := python3
GO := go
KIND_CLUSTER := oran-mano-local
K3S_CLUSTER := oran-mano-k3s
PROJECT_ROOT := $(shell pwd)

# Version pins for reproducibility
KIND_VERSION := v0.20.0
K3S_VERSION := v1.28.5+k3s1
KPT_VERSION := v1.0.0-beta.49
KUBECTL_VERSION := v1.28.5

## help: Display this help message
help:
	@echo "O-RAN Intent MANO Development Environment"
	@echo "=========================================="
	@echo ""
	@echo "Available targets:"
	@echo ""
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## /  /'

## setup: Bootstrap the development environment
setup: verify-env install-tools
	@echo -e "$(GREEN)Setting up development environment...$(NC)"
	@bash scripts/bootstrap.sh
	@echo -e "$(GREEN)Environment setup complete!$(NC)"

## check: Run all quality checks (lint, format, license, spell)
check: verify-env lint format license spell
	@echo -e "$(GREEN)All checks passed!$(NC)"

## lint: Run code linting for Python, Go, and YAML
lint: verify-env
	@echo -e "$(YELLOW)Running linting checks...$(NC)"
	@bash scripts/check-lint.sh

## format: Check code formatting
format: verify-env
	@echo -e "$(YELLOW)Checking code formatting...$(NC)"
	@bash scripts/check-format.sh

## license: Verify license headers
license: verify-env
	@echo -e "$(YELLOW)Checking license headers...$(NC)"
	@bash scripts/check-license.sh

## spell: Run spell checking
spell: verify-env
	@echo -e "$(YELLOW)Running spell check...$(NC)"
	@bash scripts/check-spell.sh

## kind: Create local Kind Kubernetes cluster
kind: verify-env
	@echo -e "$(YELLOW)Creating Kind cluster '$(KIND_CLUSTER)'...$(NC)"
	@if kind get clusters 2>/dev/null | grep -q "^$(KIND_CLUSTER)$$"; then \
		echo -e "$(YELLOW)Cluster '$(KIND_CLUSTER)' already exists$(NC)"; \
	else \
		kind create cluster --name $(KIND_CLUSTER) --config=clusters/kind-config.yaml 2>/dev/null || \
		kind create cluster --name $(KIND_CLUSTER); \
		echo -e "$(GREEN)Kind cluster created successfully$(NC)"; \
	fi
	@kubectl cluster-info --context kind-$(KIND_CLUSTER)

## k3s: Create local K3s Kubernetes cluster
k3s: verify-env
	@echo -e "$(YELLOW)Creating K3s cluster...$(NC)"
	@if [[ "$$(uname)" == "Linux" ]] || [[ "$$(uname)" == "Darwin" ]]; then \
		if ! command -v k3s &> /dev/null; then \
			echo -e "$(YELLOW)Installing K3s...$(NC)"; \
			curl -sfL https://get.k3s.io | K3S_VERSION=$(K3S_VERSION) sh -s - --write-kubeconfig-mode 644; \
		fi; \
		if ! systemctl is-active --quiet k3s 2>/dev/null && ! pgrep k3s > /dev/null; then \
			if [[ "$$(uname)" == "Linux" ]]; then \
				sudo systemctl start k3s 2>/dev/null || k3s server --write-kubeconfig-mode 644 & \
			else \
				k3s server --write-kubeconfig-mode 644 & \
			fi; \
			sleep 10; \
		fi; \
		echo -e "$(GREEN)K3s cluster is running$(NC)"; \
		export KUBECONFIG=/etc/rancher/k3s/k3s.yaml; \
		kubectl get nodes; \
	else \
		echo -e "$(RED)K3s is only supported on Linux and macOS$(NC)"; \
		exit 1; \
	fi

## kind-delete: Delete Kind cluster
kind-delete:
	@echo -e "$(YELLOW)Deleting Kind cluster '$(KIND_CLUSTER)'...$(NC)"
	@kind delete cluster --name $(KIND_CLUSTER)
	@echo -e "$(GREEN)Kind cluster deleted$(NC)"

## k3s-stop: Stop K3s cluster
k3s-stop:
	@echo -e "$(YELLOW)Stopping K3s cluster...$(NC)"
	@if [[ "$$(uname)" == "Linux" ]]; then \
		sudo systemctl stop k3s 2>/dev/null || pkill k3s; \
	else \
		pkill k3s 2>/dev/null || true; \
	fi
	@echo -e "$(GREEN)K3s cluster stopped$(NC)"

## install-tools: Install required development tools
install-tools:
	@echo -e "$(YELLOW)Installing development tools...$(NC)"
	@bash scripts/install-tools.sh
	@echo -e "$(GREEN)Tools installation complete$(NC)"

## verify-env: Verify environment variables and configuration
verify-env:
	@echo -e "$(YELLOW)Verifying environment...$(NC)"
	@if [ ! -f .env ] && [ -f .env.sample ]; then \
		echo -e "$(YELLOW)Creating .env from .env.sample...$(NC)"; \
		cp .env.sample .env; \
		echo -e "$(YELLOW)Please update .env with your configuration$(NC)"; \
	fi
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) > /dev/null 2>&1 || true; \
	fi
	@echo -e "$(GREEN)Environment verified$(NC)"

## test: Run all tests
test: verify-env
	@echo -e "$(YELLOW)Running tests...$(NC)"
	@if [ -d "nlp" ] && [ -f "nlp/tests/test_*.py" ] 2>/dev/null; then \
		cd nlp && $(PYTHON) -m pytest tests/ -v; \
	fi
	@if [ -d "orchestrator" ] && [ -f "orchestrator/go.mod" ] 2>/dev/null; then \
		cd orchestrator && $(GO) test ./... -v; \
	fi
	@if [ -d "adapters/vnf-operator" ] && [ -f "adapters/vnf-operator/go.mod" ] 2>/dev/null; then \
		cd adapters/vnf-operator && $(GO) test ./... -v; \
	fi
	@echo -e "$(GREEN)All tests passed$(NC)"

## build: Build all components
build: verify-env
	@echo -e "$(YELLOW)Building components...$(NC)"
	@if [ -d "orchestrator" ] && [ -f "orchestrator/go.mod" ] 2>/dev/null; then \
		cd orchestrator && $(GO) build -o bin/orchestrator ./cmd/orchestrator; \
	fi
	@if [ -d "adapters/vnf-operator" ] && [ -f "adapters/vnf-operator/Makefile" ] 2>/dev/null; then \
		cd adapters/vnf-operator && make build; \
	fi
	@echo -e "$(GREEN)Build complete$(NC)"

## deploy-local: Deploy to local cluster
deploy-local: verify-env
	@echo -e "$(YELLOW)Deploying to local cluster...$(NC)"
	@if kubectl config current-context | grep -q kind; then \
		echo -e "$(GREEN)Deploying to Kind cluster...$(NC)"; \
	elif kubectl config current-context | grep -q k3s; then \
		echo -e "$(GREEN)Deploying to K3s cluster...$(NC)"; \
	else \
		echo -e "$(RED)No local cluster found. Run 'make kind' or 'make k3s' first.$(NC)"; \
		exit 1; \
	fi
	@kubectl apply -k clusters/local/
	@echo -e "$(GREEN)Local deployment complete$(NC)"

## experiments: Run performance experiments
experiments: verify-env
	@echo -e "$(YELLOW)Running experiments...$(NC)"
	@if [ -f "experiments/run_suite.sh" ]; then \
		bash experiments/run_suite.sh; \
	else \
		echo -e "$(YELLOW)No experiments found$(NC)"; \
	fi

## clean: Clean generated files and build artifacts
clean:
	@echo -e "$(YELLOW)Cleaning build artifacts...$(NC)"
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@find . -type f -name "*.pyc" -delete 2>/dev/null || true
	@find . -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name "bin" -exec rm -rf {} + 2>/dev/null || true
	@find . -type f -name "*.log" -delete 2>/dev/null || true
	@echo -e "$(GREEN)Clean complete$(NC)"

## info: Display environment information
info:
	@echo "O-RAN Intent MANO Environment Info"
	@echo "===================================="
	@echo "Project Root: $(PROJECT_ROOT)"
	@echo "Python: $$(which $(PYTHON)) ($$($(PYTHON) --version 2>&1))"
	@echo "Go: $$(which $(GO)) ($$($(GO) version 2>&1))"
	@echo "Docker: $$(which docker) ($$(docker --version 2>&1))"
	@echo "Kubectl: $$(which kubectl) ($$(kubectl version --client --short 2>&1))"
	@echo "Kind: $$(which kind) ($$(kind --version 2>&1))"
	@echo ""
	@echo "Active Kubernetes Context: $$(kubectl config current-context 2>&1 || echo 'None')"

# Default target
.DEFAULT_GOAL := help