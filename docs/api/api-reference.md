# O-RAN Intent-MANO API Reference

## Overview

The O-RAN Intent-MANO API provides comprehensive orchestration capabilities for O-RAN-based network function virtualization and intent-based network management. This document provides detailed information about all available endpoints, request/response formats, and usage examples.

## Table of Contents

1. [Authentication](#authentication)
2. [Rate Limiting](#rate-limiting)
3. [Error Handling](#error-handling)
4. [Orchestrator API](#orchestrator-api)
5. [VNF Operator API](#vnf-operator-api)
6. [TN Manager API](#tn-manager-api)
7. [O2 Client API](#o2-client-api)
8. [Monitoring API](#monitoring-api)
9. [Webhooks](#webhooks)
10. [SDK Examples](#sdk-examples)

## Authentication

The API uses Bearer token authentication with JWT tokens.

### Obtaining Access Token

```http
POST /v1/auth/login
Content-Type: application/json

{
  "username": "admin@oran-mano.io",
  "password": "SecureP@ssw0rd"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "def50200e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

### Using the Token

Include the token in the Authorization header for all API requests:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Refreshing Tokens

```http
POST /v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "def50200e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

## Rate Limiting

The API implements rate limiting to ensure fair usage:

- **Standard endpoints**: 1000 requests per minute
- **Deployment operations**: 100 requests per minute
- **Monitoring endpoints**: 5000 requests per minute

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1609459200
```

## Error Handling

The API uses standard HTTP status codes and returns detailed error information in JSON format.

### Error Response Format

```json
{
  "status": 400,
  "title": "Bad Request",
  "detail": "The request could not be understood by the server",
  "type": "https://api.oran-mano.io/errors/bad-request",
  "instance": "/v1/orchestrator/intents/invalid-id"
}
```

### Validation Errors

For input validation failures (HTTP 422):

```json
{
  "status": 422,
  "title": "Validation Error",
  "detail": "Input validation failed",
  "errors": [
    {
      "field": "bandwidth",
      "message": "bandwidth must be between 1 and 10000",
      "code": "RANGE_ERROR"
    },
    {
      "field": "slice_type",
      "message": "slice_type must be one of: eMBB, uRLLC, mIoT, balanced",
      "code": "ENUM_ERROR"
    }
  ]
}
```

### Common Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `204 No Content` - Request successful, no content returned
- `400 Bad Request` - Invalid request format
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict
- `422 Unprocessable Entity` - Validation failed
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Orchestrator API

The Orchestrator API manages intent-based network slice orchestration.

### List QoS Intents

Retrieve all QoS intents with optional filtering.

```http
GET /v1/orchestrator/intents?limit=50&offset=0&slice_type=uRLLC&status=deployed
```

**Parameters:**
- `limit` (optional): Maximum items to return (1-1000, default: 50)
- `offset` (optional): Items to skip (default: 0)
- `slice_type` (optional): Filter by slice type (eMBB, uRLLC, mIoT, balanced)
- `status` (optional): Filter by status (planned, deployed, failed, deleting)

**Response:**
```json
{
  "intents": [
    {
      "id": "intent-001",
      "bandwidth": 100.0,
      "latency": 10.0,
      "slice_type": "uRLLC",
      "jitter": 2.0,
      "packet_loss": 0.001,
      "reliability": 0.9999,
      "status": "deployed",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:45:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

### Create QoS Intent

Submit a new QoS intent for orchestration.

```http
POST /v1/orchestrator/intents
Content-Type: application/json

{
  "bandwidth": 500.0,
  "latency": 5.0,
  "slice_type": "eMBB",
  "jitter": 1.0,
  "packet_loss": 0.0001,
  "reliability": 0.999
}
```

**Response (201 Created):**
```json
{
  "id": "intent-002",
  "bandwidth": 500.0,
  "latency": 5.0,
  "slice_type": "eMBB",
  "jitter": 1.0,
  "packet_loss": 0.0001,
  "reliability": 0.999,
  "status": "planned",
  "created_at": "2024-01-15T12:00:00Z",
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Generate Orchestration Plan

Create an orchestration plan without executing deployment.

```http
POST /v1/orchestrator/plan
Content-Type: application/json

{
  "intents": [
    {
      "bandwidth": 100.0,
      "latency": 10.0,
      "slice_type": "uRLLC"
    },
    {
      "bandwidth": 500.0,
      "latency": 50.0,
      "slice_type": "eMBB"
    }
  ],
  "dry_run": true
}
```

**Response:**
```json
{
  "id": "plan-001",
  "timestamp": "2024-01-15T12:15:00Z",
  "allocations": [
    {
      "slice_id": "slice-uRLLC-001",
      "qos": {
        "bandwidth": 100.0,
        "latency": 10.0,
        "slice_type": "uRLLC"
      },
      "placement": {
        "site_id": "edge01",
        "cloud_type": "edge",
        "region": "us-east",
        "zone": "us-east-1a",
        "score": 95.5,
        "constraints_met": true,
        "reasons": ["Low latency requirement met", "Sufficient resources available"]
      },
      "resources": {
        "ran_resources": {
          "cpu_cores": 30.0,
          "memory_mb": 12800,
          "antennas": 100,
          "frequency_mhz": 13500
        },
        "cn_resources": {
          "cpu_cores": 40.0,
          "memory_mb": 25600,
          "storage_gb": 200,
          "upf_capacity": 1000
        },
        "tn_resources": {
          "bandwidth_mbps": 100.0,
          "vlan_id": 2000,
          "qos_class": "uRLLC",
          "latency_budget_ms": 10.0
        }
      },
      "status": "planned"
    }
  ],
  "total_slices": 2,
  "estimated_cost": 1250.50,
  "estimated_time": 300
}
```

### Execute Deployment

Apply an orchestration plan and deploy network slices.

```http
POST /v1/orchestrator/deploy
Content-Type: application/json

{
  "plan_id": "plan-001",
  "force": false
}
```

**Response (202 Accepted):**
```json
{
  "deployment_id": "deploy-001",
  "status": "running",
  "progress_percent": 0,
  "message": "Deployment initiated",
  "started_at": "2024-01-15T12:30:00Z",
  "estimated_completion": "2024-01-15T12:35:00Z"
}
```

## VNF Operator API

The VNF Operator API manages Virtual Network Function lifecycle.

### List VNFs

Retrieve all VNFs with optional filtering.

```http
GET /v1/vnf-operator/vnfs?type=UPF&status=Running&limit=20
```

**Response:**
```json
{
  "vnfs": [
    {
      "id": "vnf-upf-001",
      "name": "upf-edge-001",
      "type": "UPF",
      "version": "v1.2.3",
      "status": "Running",
      "qos": {
        "bandwidth": 1000.0,
        "latency": 5.0,
        "slice_type": "eMBB"
      },
      "placement": {
        "cloud_type": "edge",
        "region": "us-east",
        "zone": "us-east-1a"
      },
      "target_clusters": ["edge-cluster-01"],
      "resources": {
        "cpu_cores": 8,
        "memory_gb": 16,
        "storage_gb": 100
      },
      "image": {
        "repository": "registry.oran.io/vnf/upf",
        "tag": "v1.2.3",
        "pull_policy": "IfNotPresent"
      },
      "instances": [
        {
          "cluster": "edge-cluster-01",
          "namespace": "oran-vnf",
          "name": "upf-edge-001-0",
          "status": "Running",
          "ip_address": "10.244.1.15",
          "endpoints": [
            {
              "name": "n3",
              "protocol": "UDP",
              "port": 2152,
              "external_ip": "203.0.113.10"
            }
          ]
        }
      ],
      "created_at": "2024-01-15T09:00:00Z",
      "updated_at": "2024-01-15T09:15:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### Create VNF

Deploy a new Virtual Network Function.

```http
POST /v1/vnf-operator/vnfs
Content-Type: application/json

{
  "name": "amf-regional-001",
  "type": "AMF",
  "version": "v2.1.0",
  "qos": {
    "bandwidth": 500.0,
    "latency": 20.0,
    "slice_type": "balanced"
  },
  "placement": {
    "cloud_type": "regional",
    "region": "us-east",
    "preferred_zones": ["us-east-1b", "us-east-1c"]
  },
  "target_clusters": ["regional-cluster-01"],
  "resources": {
    "cpu_cores": 4,
    "memory_gb": 8,
    "storage_gb": 50
  },
  "config": {
    "plmn_id": "001001",
    "tac": "000001"
  },
  "image": {
    "repository": "registry.oran.io/vnf/amf",
    "tag": "v2.1.0",
    "pull_policy": "Always"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "vnf-amf-001",
  "name": "amf-regional-001",
  "type": "AMF",
  "version": "v2.1.0",
  "status": "Creating",
  "qos": {
    "bandwidth": 500.0,
    "latency": 20.0,
    "slice_type": "balanced"
  },
  "placement": {
    "cloud_type": "regional",
    "region": "us-east",
    "preferred_zones": ["us-east-1b", "us-east-1c"]
  },
  "target_clusters": ["regional-cluster-01"],
  "resources": {
    "cpu_cores": 4,
    "memory_gb": 8,
    "storage_gb": 50
  },
  "config": {
    "plmn_id": "001001",
    "tac": "000001"
  },
  "image": {
    "repository": "registry.oran.io/vnf/amf",
    "tag": "v2.1.0",
    "pull_policy": "Always"
  },
  "instances": [],
  "created_at": "2024-01-15T13:00:00Z",
  "updated_at": "2024-01-15T13:00:00Z"
}
```

### Scale VNF

Scale VNF instances up or down.

```http
POST /v1/vnf-operator/vnfs/vnf-upf-001/scale
Content-Type: application/json

{
  "replicas": 3
}
```

**Response (202 Accepted):**
```json
{
  "operation_id": "scale-op-001",
  "status": "running",
  "message": "Scaling VNF to 3 replicas",
  "started_at": "2024-01-15T13:30:00Z"
}
```

## TN Manager API

The Transport Network Manager API handles network slice configuration and performance testing.

### List TN Agents

Retrieve all registered Transport Network agents.

```http
GET /v1/tn-manager/agents
```

**Response:**
```json
{
  "agents": [
    {
      "id": "agent-edge-01",
      "name": "Edge Site 01 Agent",
      "endpoint": "https://edge01.oran.io:8443",
      "status": "connected",
      "version": "v1.0.0",
      "capabilities": ["vxlan", "tc", "performance_test"],
      "last_seen": "2024-01-15T13:45:00Z",
      "created_at": "2024-01-10T08:00:00Z"
    },
    {
      "id": "agent-regional-01",
      "name": "Regional Site 01 Agent",
      "endpoint": "https://regional01.oran.io:8443",
      "status": "connected",
      "version": "v1.0.0",
      "capabilities": ["vxlan", "tc", "performance_test", "monitoring"],
      "last_seen": "2024-01-15T13:44:00Z",
      "created_at": "2024-01-10T08:30:00Z"
    }
  ]
}
```

### Configure Network Slice

Configure a network slice on a specific TN agent.

```http
POST /v1/tn-manager/agents/agent-edge-01/slices
Content-Type: application/json

{
  "slice_id": "slice-uRLLC-001",
  "bandwidth_mbps": 100.0,
  "vlan_id": 2001,
  "qos_class": "uRLLC",
  "priority": 7,
  "latency_budget_ms": 10.0
}
```

**Response:**
```json
{
  "slice_id": "slice-uRLLC-001",
  "status": "configured",
  "bandwidth_allocated": 100.0,
  "vlan_id": 2001,
  "message": "Network slice configured successfully",
  "configured_at": "2024-01-15T14:00:00Z"
}
```

### Run Performance Test

Execute a performance test on a TN agent.

```http
POST /v1/tn-manager/agents/agent-edge-01/performance
Content-Type: application/json

{
  "test_id": "perf-test-001",
  "duration_seconds": 60,
  "bandwidth_mbps": 100.0,
  "packet_size": 1500,
  "target_host": "192.168.1.100"
}
```

**Response:**
```json
{
  "test_id": "perf-test-001",
  "throughput": {
    "avg_mbps": 98.5,
    "max_mbps": 99.8,
    "min_mbps": 97.2
  },
  "latency": {
    "avg_rtt_ms": 8.5,
    "max_rtt_ms": 12.3,
    "min_rtt_ms": 7.1,
    "jitter_ms": 1.2
  },
  "packet_loss": {
    "rate": 0.0001,
    "packets_sent": 100000,
    "packets_received": 99990
  },
  "test_duration": 60,
  "timestamp": "2024-01-15T14:15:00Z"
}
```

### Get Agent Status

Retrieve the current status of a TN agent.

```http
GET /v1/tn-manager/agents/agent-edge-01/status
```

**Response:**
```json
{
  "agent_id": "agent-edge-01",
  "status": "healthy",
  "uptime_seconds": 432000,
  "active_slices": 3,
  "total_bandwidth_mbps": 1000.0,
  "available_bandwidth_mbps": 700.0,
  "cpu_usage_percent": 25.3,
  "memory_usage_percent": 45.7,
  "interfaces": [
    {
      "name": "eth0",
      "status": "up",
      "speed_mbps": 1000.0,
      "utilization_percent": 30.0
    },
    {
      "name": "eth1",
      "status": "up",
      "speed_mbps": 1000.0,
      "utilization_percent": 0.0
    }
  ],
  "last_updated": "2024-01-15T14:20:00Z"
}
```

## O2 Client API

The O2 Client API provides integration with O-RAN O2 DMS (Deployment Management Service).

### List Deployment Managers

Retrieve available O2 DMS deployment managers.

```http
GET /v1/o2-client/deployment-managers?limit=10
```

**Response:**
```json
{
  "deployment_managers": [
    {
      "id": "dm-kubernetes-01",
      "name": "Kubernetes Deployment Manager",
      "description": "Kubernetes-based deployment manager for containerized NFs",
      "endpoint": "https://k8s-dm.oran.io/o2dms/v1",
      "status": "available",
      "capabilities": ["containerized_nf", "helm_charts", "kustomize"],
      "supported_nf_types": ["UPF", "AMF", "SMF", "PCF"],
      "version": "v1.0.0"
    },
    {
      "id": "dm-openstack-01",
      "name": "OpenStack Deployment Manager",
      "description": "OpenStack-based deployment manager for VM-based NFs",
      "endpoint": "https://os-dm.oran.io/o2dms/v1",
      "status": "available",
      "capabilities": ["vm_based_nf", "heat_templates"],
      "supported_nf_types": ["gNB", "CU", "DU", "RU"],
      "version": "v1.1.0"
    }
  ]
}
```

### List NF Deployment Descriptors

Retrieve NF deployment descriptors for a specific deployment manager.

```http
GET /v1/o2-client/deployment-managers/dm-kubernetes-01/nf-descriptors?limit=5
```

**Response:**
```json
{
  "descriptors": [
    {
      "id": "nfd-upf-basic",
      "name": "Basic UPF Deployment",
      "version": "v1.2.0",
      "nf_type": "UPF",
      "vendor": "ORAN Vendor",
      "description": "Basic User Plane Function deployment descriptor",
      "input_parameters": {
        "type": "object",
        "properties": {
          "plmn_id": {"type": "string"},
          "dnn": {"type": "string"},
          "ip_pool": {"type": "string"}
        },
        "required": ["plmn_id", "dnn"]
      },
      "artifacts": [
        {
          "name": "helm-chart",
          "type": "application/x-helm-chart",
          "url": "https://registry.oran.io/charts/upf-v1.2.0.tgz"
        }
      ]
    }
  ]
}
```

### Create NF Deployment

Deploy a Network Function using O2 DMS.

```http
POST /v1/o2-client/deployment-managers/dm-kubernetes-01/deployments
Content-Type: application/json

{
  "name": "upf-deployment-001",
  "description": "UPF deployment for slice uRLLC-001",
  "nf_deployment_descriptor_id": "nfd-upf-basic",
  "input_params": {
    "plmn_id": "001001",
    "dnn": "internet",
    "ip_pool": "10.45.0.0/16"
  },
  "location_constraints": ["edge", "us-east-1a"],
  "extensions": {
    "oran.io/qos-bandwidth": 100.0,
    "oran.io/qos-latency": 10.0,
    "oran.io/slice-type": "uRLLC"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "nfd-001",
  "name": "upf-deployment-001",
  "description": "UPF deployment for slice uRLLC-001",
  "nf_deployment_descriptor_id": "nfd-upf-basic",
  "status": "instantiating",
  "input_params": {
    "plmn_id": "001001",
    "dnn": "internet",
    "ip_pool": "10.45.0.0/16"
  },
  "location_constraints": ["edge", "us-east-1a"],
  "created_at": "2024-01-15T15:00:00Z",
  "updated_at": "2024-01-15T15:00:00Z"
}
```

## Monitoring API

The Monitoring API provides system health checks and metrics collection.

### Health Check

Check overall system health (no authentication required).

```http
GET /v1/monitoring/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T15:30:00Z",
  "version": "1.0.0",
  "components": {
    "database": "healthy",
    "redis": "healthy",
    "message_queue": "healthy"
  }
}
```

### Get System Metrics

Retrieve system metrics in Prometheus format.

```http
GET /v1/monitoring/metrics
```

**Response (text/plain):**
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1024
http_requests_total{method="POST",status="201"} 245
http_requests_total{method="PUT",status="200"} 89
# HELP active_vnfs Number of active VNFs
# TYPE active_vnfs gauge
active_vnfs 15
# HELP network_slices_total Total number of network slices
# TYPE network_slices_total counter
network_slices_total 8
```

### Get Slice Metrics

Retrieve detailed metrics for a specific network slice.

```http
GET /v1/monitoring/slices/slice-uRLLC-001/metrics?timeframe=1h
```

**Response:**
```json
{
  "slice_id": "slice-uRLLC-001",
  "timeframe": "1h",
  "bandwidth_utilization": {
    "current_mbps": 87.5,
    "allocated_mbps": 100.0,
    "utilization_percent": 87.5,
    "peak_mbps": 98.2
  },
  "latency": {
    "current_ms": 8.2,
    "target_ms": 10.0,
    "percentile_95_ms": 9.1,
    "percentile_99_ms": 9.8
  },
  "throughput": {
    "rx_mbps": 87.5,
    "tx_mbps": 85.3,
    "rx_packets_per_sec": 7291,
    "tx_packets_per_sec": 7109
  },
  "quality": {
    "packet_loss_rate": 0.0001,
    "jitter_ms": 1.1,
    "error_rate": 0.00005
  },
  "availability": {
    "uptime_percent": 99.98,
    "downtime_seconds": 72,
    "sla_compliance": true
  },
  "timestamp": "2024-01-15T15:45:00Z"
}
```

## Webhooks

The system supports webhooks for real-time notifications of important events.

### Deployment Status Changed

Triggered when a deployment status changes.

**Event Types:**
- `deployment.started` - Deployment has started
- `deployment.completed` - Deployment completed successfully
- `deployment.failed` - Deployment failed

**Payload:**
```json
{
  "event": "deployment.completed",
  "deployment_id": "deploy-001",
  "status": "completed",
  "message": "All network slices deployed successfully",
  "timestamp": "2024-01-15T16:00:00Z"
}
```

### Slice Status Changed

Triggered when a network slice status changes.

**Event Types:**
- `slice.created` - New slice created
- `slice.deployed` - Slice deployed successfully
- `slice.failed` - Slice deployment failed
- `slice.deleted` - Slice deleted

**Payload:**
```json
{
  "event": "slice.deployed",
  "slice_id": "slice-uRLLC-001",
  "status": "active",
  "metrics": {
    "slice_id": "slice-uRLLC-001",
    "bandwidth_utilization": {
      "current_mbps": 0.0,
      "allocated_mbps": 100.0,
      "utilization_percent": 0.0
    },
    "latency": {
      "current_ms": 0.0,
      "target_ms": 10.0
    }
  },
  "timestamp": "2024-01-15T16:05:00Z"
}
```

## SDK Examples

### Python SDK

```python
import requests
from typing import Dict, List, Optional

class ORANMANOClient:
    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url
        self.session = requests.Session()
        self.authenticate(username, password)

    def authenticate(self, username: str, password: str):
        """Authenticate and obtain access token"""
        response = self.session.post(
            f"{self.base_url}/auth/login",
            json={"username": username, "password": password}
        )
        response.raise_for_status()

        token_data = response.json()
        self.session.headers.update({
            "Authorization": f"Bearer {token_data['access_token']}"
        })

    def create_qos_intent(self, bandwidth: float, latency: float,
                         slice_type: str, **kwargs) -> Dict:
        """Create a new QoS intent"""
        intent_data = {
            "bandwidth": bandwidth,
            "latency": latency,
            "slice_type": slice_type,
            **kwargs
        }

        response = self.session.post(
            f"{self.base_url}/orchestrator/intents",
            json=intent_data
        )
        response.raise_for_status()
        return response.json()

    def list_vnfs(self, vnf_type: Optional[str] = None,
                  status: Optional[str] = None) -> List[Dict]:
        """List VNFs with optional filtering"""
        params = {}
        if vnf_type:
            params["type"] = vnf_type
        if status:
            params["status"] = status

        response = self.session.get(
            f"{self.base_url}/vnf-operator/vnfs",
            params=params
        )
        response.raise_for_status()
        return response.json()["vnfs"]

    def deploy_vnf(self, vnf_spec: Dict) -> Dict:
        """Deploy a new VNF"""
        response = self.session.post(
            f"{self.base_url}/vnf-operator/vnfs",
            json=vnf_spec
        )
        response.raise_for_status()
        return response.json()

    def get_slice_metrics(self, slice_id: str, timeframe: str = "1h") -> Dict:
        """Get metrics for a network slice"""
        response = self.session.get(
            f"{self.base_url}/monitoring/slices/{slice_id}/metrics",
            params={"timeframe": timeframe}
        )
        response.raise_for_status()
        return response.json()

# Usage example
client = ORANMANOClient(
    base_url="https://api.oran-mano.io/v1",
    username="admin@oran-mano.io",
    password="SecureP@ssw0rd"
)

# Create a uRLLC intent
intent = client.create_qos_intent(
    bandwidth=100.0,
    latency=10.0,
    slice_type="uRLLC",
    reliability=0.9999
)
print(f"Created intent: {intent['id']}")

# List running UPF VNFs
upfs = client.list_vnfs(vnf_type="UPF", status="Running")
print(f"Found {len(upfs)} running UPF instances")

# Deploy a new AMF
amf_spec = {
    "name": "amf-001",
    "type": "AMF",
    "version": "v2.1.0",
    "qos": {"bandwidth": 500.0, "latency": 20.0, "slice_type": "balanced"},
    "placement": {"cloud_type": "regional", "region": "us-east"},
    "target_clusters": ["regional-cluster-01"],
    "resources": {"cpu_cores": 4, "memory_gb": 8, "storage_gb": 50},
    "image": {"repository": "registry.oran.io/vnf/amf", "tag": "v2.1.0"}
}

amf = client.deploy_vnf(amf_spec)
print(f"Deployed AMF: {amf['id']}")
```

### Go SDK

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

type ORANMANOClient struct {
    BaseURL    string
    HTTPClient *http.Client
    AuthToken  string
}

type QoSIntent struct {
    ID          string  `json:"id,omitempty"`
    Bandwidth   float64 `json:"bandwidth"`
    Latency     float64 `json:"latency"`
    SliceType   string  `json:"slice_type"`
    Jitter      *float64 `json:"jitter,omitempty"`
    PacketLoss  *float64 `json:"packet_loss,omitempty"`
    Reliability *float64 `json:"reliability,omitempty"`
    Status      string  `json:"status,omitempty"`
    CreatedAt   time.Time `json:"created_at,omitempty"`
    UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type AuthResponse struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token"`
}

func NewORANMANOClient(baseURL, username, password string) (*ORANMANOClient, error) {
    client := &ORANMANOClient{
        BaseURL: baseURL,
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }

    if err := client.Authenticate(username, password); err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }

    return client, nil
}

func (c *ORANMANOClient) Authenticate(username, password string) error {
    authData := map[string]string{
        "username": username,
        "password": password,
    }

    var authResponse AuthResponse
    if err := c.doRequest("POST", "/auth/login", authData, &authResponse); err != nil {
        return err
    }

    c.AuthToken = authResponse.AccessToken
    return nil
}

func (c *ORANMANOClient) CreateQoSIntent(intent *QoSIntent) (*QoSIntent, error) {
    var result QoSIntent
    if err := c.doRequest("POST", "/orchestrator/intents", intent, &result); err != nil {
        return nil, err
    }
    return &result, nil
}

func (c *ORANMANOClient) ListQoSIntents(sliceType, status string, limit, offset int) ([]QoSIntent, error) {
    params := url.Values{}
    if sliceType != "" {
        params.Add("slice_type", sliceType)
    }
    if status != "" {
        params.Add("status", status)
    }
    if limit > 0 {
        params.Add("limit", fmt.Sprintf("%d", limit))
    }
    if offset > 0 {
        params.Add("offset", fmt.Sprintf("%d", offset))
    }

    endpoint := "/orchestrator/intents"
    if len(params) > 0 {
        endpoint += "?" + params.Encode()
    }

    var response struct {
        Intents []QoSIntent `json:"intents"`
        Total   int         `json:"total"`
        Limit   int         `json:"limit"`
        Offset  int         `json:"offset"`
    }

    if err := c.doRequest("GET", endpoint, nil, &response); err != nil {
        return nil, err
    }

    return response.Intents, nil
}

func (c *ORANMANOClient) doRequest(method, endpoint string, body interface{}, result interface{}) error {
    var reqBody []byte
    var err error

    if body != nil {
        reqBody, err = json.Marshal(body)
        if err != nil {
            return fmt.Errorf("failed to marshal request body: %w", err)
        }
    }

    req, err := http.NewRequest(method, c.BaseURL+endpoint, bytes.NewBuffer(reqBody))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    if c.AuthToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.AuthToken)
    }

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    if result != nil {
        if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
            return fmt.Errorf("failed to decode response: %w", err)
        }
    }

    return nil
}

func main() {
    // Create client
    client, err := NewORANMANOClient(
        "https://api.oran-mano.io/v1",
        "admin@oran-mano.io",
        "SecureP@ssw0rd",
    )
    if err != nil {
        panic(err)
    }

    // Create a uRLLC intent
    reliability := 0.9999
    intent := &QoSIntent{
        Bandwidth:   100.0,
        Latency:     10.0,
        SliceType:   "uRLLC",
        Reliability: &reliability,
    }

    createdIntent, err := client.CreateQoSIntent(intent)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Created intent: %s\n", createdIntent.ID)

    // List all deployed intents
    intents, err := client.ListQoSIntents("", "deployed", 50, 0)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d deployed intents\n", len(intents))
}
```

### curl Examples

#### Create QoS Intent
```bash
# Authenticate first
TOKEN=$(curl -s -X POST https://api.oran-mano.io/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin@oran-mano.io","password":"SecureP@ssw0rd"}' \
  | jq -r '.access_token')

# Create uRLLC intent
curl -X POST https://api.oran-mano.io/v1/orchestrator/intents \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "bandwidth": 100.0,
    "latency": 10.0,
    "slice_type": "uRLLC",
    "reliability": 0.9999
  }'
```

#### Deploy VNF
```bash
curl -X POST https://api.oran-mano.io/v1/vnf-operator/vnfs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "upf-edge-001",
    "type": "UPF",
    "version": "v1.2.3",
    "qos": {
      "bandwidth": 1000.0,
      "latency": 5.0,
      "slice_type": "eMBB"
    },
    "placement": {
      "cloud_type": "edge",
      "region": "us-east"
    },
    "target_clusters": ["edge-cluster-01"],
    "resources": {
      "cpu_cores": 8,
      "memory_gb": 16,
      "storage_gb": 100
    },
    "image": {
      "repository": "registry.oran.io/vnf/upf",
      "tag": "v1.2.3"
    }
  }'
```

#### Get Slice Metrics
```bash
curl -X GET "https://api.oran-mano.io/v1/monitoring/slices/slice-uRLLC-001/metrics?timeframe=1h" \
  -H "Authorization: Bearer $TOKEN"
```

This completes the comprehensive API reference documentation for the O-RAN Intent-MANO system, providing detailed information about all endpoints, request/response formats, and practical usage examples in multiple programming languages.