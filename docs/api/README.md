# O-RAN Intent-MANO API Documentation

This directory contains comprehensive API documentation for the O-RAN Intent-based Management and Network Orchestration (MANO) system.

## 📚 Documentation Contents

### 1. OpenAPI Specification
- **[openapi.yaml](./openapi.yaml)** - Complete OpenAPI 3.0 specification
  - Covers all four main APIs: Orchestrator, VNF Operator, TN Manager, and O2 Client
  - Includes detailed schemas, examples, and error responses
  - Supports authentication, rate limiting, and webhook documentation

### 2. API Reference Guide
- **[api-reference.md](./api-reference.md)** - Comprehensive API reference
  - Detailed endpoint documentation with examples
  - Authentication and error handling guides
  - SDK examples in Python, Go, and curl
  - Rate limiting and best practices

### 3. Developer Guide
- **[developer-guide.md](./developer-guide.md)** - Complete developer integration guide
  - API versioning strategy and migration paths
  - Security best practices and authentication patterns
  - Performance optimization techniques
  - Monitoring and observability setup
  - Common integration patterns and troubleshooting

### 4. Testing Resources
- **[postman-collection.json](./postman-collection.json)** - Postman collection for API testing
  - Pre-configured requests for all endpoints
  - Environment variables and authentication setup
  - Test scenarios and workflows
  - Automated tests with assertions

### 5. Interactive Documentation
- **[swagger-ui-setup.md](./swagger-ui-setup.md)** - Swagger UI deployment guide
  - Docker, Nginx, Node.js, and Kubernetes setups
  - Security configuration and customization
  - Branding and theme customization

## 🚀 Quick Start

### 1. View Interactive Documentation

#### Using Docker (Recommended)
```bash
# From the project root directory
docker run -p 8080:8080 -v $(pwd)/docs/api/openapi.yaml:/app/openapi.yaml swaggerapi/swagger-ui

# Access at: http://localhost:8080
```

#### Using Node.js
```bash
npm install -g swagger-ui-serve
swagger-ui-serve docs/api/openapi.yaml -p 8080
```

### 2. Import Postman Collection

1. Open Postman
2. Click "Import" → "File"
3. Select `docs/api/postman-collection.json`
4. Set environment variables:
   - `base_url`: Your API endpoint (e.g., `https://api.oran-mano.io/v1`)
   - Authenticate to get `access_token`

### 3. Test API Connectivity

```bash
# Health check (no authentication required)
curl -X GET https://api.oran-mano.io/v1/monitoring/health

# Authenticate
curl -X POST https://api.oran-mano.io/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"your-username","password":"your-password"}'

# Test authenticated endpoint
curl -X GET https://api.oran-mano.io/v1/orchestrator/intents \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## 🏗️ API Architecture

The O-RAN Intent-MANO system consists of four main API components:

### 1. Orchestrator API (`/orchestrator/*`)
- **Purpose**: Intent-based network slice orchestration
- **Key Features**:
  - QoS intent management (CRUD operations)
  - Orchestration planning and execution
  - Resource allocation and placement decisions
- **Main Endpoints**:
  - `GET/POST /intents` - Manage QoS intents
  - `POST /plan` - Generate orchestration plans
  - `POST /deploy` - Execute deployments

### 2. VNF Operator API (`/vnf-operator/*`)
- **Purpose**: Virtual Network Function lifecycle management
- **Key Features**:
  - VNF deployment and management
  - Scaling operations
  - Multi-cluster orchestration
- **Main Endpoints**:
  - `GET/POST /vnfs` - VNF management
  - `POST /vnfs/{id}/scale` - Scaling operations
  - `PUT/DELETE /vnfs/{id}` - VNF lifecycle

### 3. TN Manager API (`/tn-manager/*`)
- **Purpose**: Transport Network configuration and monitoring
- **Key Features**:
  - Network slice configuration
  - Performance testing
  - Agent management
- **Main Endpoints**:
  - `GET/POST /agents` - TN agent management
  - `POST /agents/{id}/slices` - Slice configuration
  - `POST /agents/{id}/performance` - Performance testing

### 4. O2 Client API (`/o2-client/*`)
- **Purpose**: O-RAN O2 DMS/IMS integration
- **Key Features**:
  - Deployment manager operations
  - NF descriptor management
  - Standards-compliant O-RAN integration
- **Main Endpoints**:
  - `GET /deployment-managers` - DM discovery
  - `GET/POST /deployments` - NF deployment management
  - `GET /nf-descriptors` - NF descriptor operations

## 🔐 Authentication

The API uses **Bearer token authentication** with JWT tokens:

```bash
# 1. Obtain access token
POST /auth/login
{
  "username": "your-username",
  "password": "your-password"
}

# Response
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "refresh-token-string"
}

# 2. Use token in requests
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# 3. Refresh when needed
POST /auth/refresh
{
  "refresh_token": "refresh-token-string"
}
```

## 📊 Rate Limiting

Different rate limits apply based on operation type:

| Endpoint Type | Rate Limit | Window |
|---------------|------------|--------|
| Standard Operations | 1000 requests | per minute |
| Deployment Operations | 100 requests | per minute |
| Monitoring Endpoints | 5000 requests | per minute |

Rate limit headers are included in responses:
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1609459200
```

## 🛡️ Error Handling

The API uses standard HTTP status codes with detailed error responses:

```json
{
  "status": 422,
  "title": "Validation Error",
  "detail": "Input validation failed",
  "type": "https://api.oran-mano.io/errors/validation",
  "instance": "/v1/orchestrator/intents",
  "errors": [
    {
      "field": "bandwidth",
      "message": "bandwidth must be between 1 and 10000",
      "code": "RANGE_ERROR"
    }
  ]
}
```

## 📈 Monitoring

### Health Checks
```bash
# System health (no auth required)
GET /monitoring/health

# Prometheus metrics (no auth required)
GET /monitoring/metrics

# Slice-specific metrics
GET /monitoring/slices/{sliceId}/metrics?timeframe=1h
```

### Webhooks

The system supports webhooks for real-time event notifications:

- **Deployment Events**: `deployment.started`, `deployment.completed`, `deployment.failed`
- **Slice Events**: `slice.created`, `slice.deployed`, `slice.failed`, `slice.deleted`

Register webhooks:
```bash
POST /webhooks/subscriptions
{
  "url": "https://your-webhook-endpoint.com/webhook",
  "events": ["deployment.completed", "slice.deployed"],
  "secret": "your-webhook-secret"
}
```

## 🔧 SDK Examples

### Python
```python
from oran_mano_client import ORANMANOClient

client = ORANMANOClient(
    base_url="https://api.oran-mano.io/v1",
    username="your-username",
    password="your-password"
)

# Create intent
intent = client.create_qos_intent(
    bandwidth=100.0,
    latency=10.0,
    slice_type="uRLLC"
)
print(f"Created intent: {intent['id']}")
```

### Go
```go
client, err := NewORANMANOClient("https://api.oran-mano.io/v1", "username", "password")
if err != nil {
    log.Fatal(err)
}

intent := &QoSIntent{
    Bandwidth: 100.0,
    Latency:   10.0,
    SliceType: "uRLLC",
}

created, err := client.CreateQoSIntent(context.Background(), intent)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created intent: %s\n", created.ID)
```

### JavaScript/Node.js
```javascript
const { ORANMANOClient } = require('oran-mano-client');

const client = new ORANMANOClient({
    baseURL: 'https://api.oran-mano.io/v1',
    username: 'your-username',
    password: 'your-password'
});

const intent = await client.createQoSIntent({
    bandwidth: 100.0,
    latency: 10.0,
    slice_type: 'uRLLC'
});
console.log(`Created intent: ${intent.id}`);
```

## 📋 Common Workflows

### 1. Complete Slice Deployment
```bash
# 1. Create QoS intent
POST /orchestrator/intents
{
  "bandwidth": 100.0,
  "latency": 10.0,
  "slice_type": "uRLLC"
}

# 2. Generate orchestration plan
POST /orchestrator/plan
{
  "intents": [{ "bandwidth": 100.0, "latency": 10.0, "slice_type": "uRLLC" }],
  "dry_run": true
}

# 3. Deploy VNFs
POST /vnf-operator/vnfs
{
  "name": "upf-edge-001",
  "type": "UPF",
  "placement": { "cloud_type": "edge" },
  "qos": { "bandwidth": 100.0, "latency": 10.0 }
}

# 4. Configure transport network
POST /tn-manager/agents/{agentId}/slices
{
  "slice_id": "slice-001",
  "bandwidth_mbps": 100.0,
  "qos_class": "uRLLC"
}
```

### 2. Performance Testing
```bash
# 1. Check agent status
GET /tn-manager/agents/{agentId}/status

# 2. Run performance test
POST /tn-manager/agents/{agentId}/performance
{
  "test_id": "perf-test-001",
  "duration_seconds": 60,
  "bandwidth_mbps": 100.0
}

# 3. Monitor slice metrics
GET /monitoring/slices/{sliceId}/metrics?timeframe=1h
```

## 🐛 Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify credentials and check token expiry
   - Ensure proper Bearer token format
   - Check network connectivity

2. **Rate Limiting**
   - Implement exponential backoff
   - Monitor rate limit headers
   - Consider request batching

3. **Validation Errors**
   - Check field requirements and ranges
   - Validate enum values
   - Use local validation before API calls

### Debug Tools

```bash
# Enable verbose logging
export DEBUG=oran-mano-client:*

# Test connectivity
curl -v https://api.oran-mano.io/v1/monitoring/health

# Validate OpenAPI spec
swagger-codegen validate -i docs/api/openapi.yaml
```

## 📞 Support

- **Documentation Issues**: Create an issue in the project repository
- **API Questions**: Refer to the [Developer Guide](./developer-guide.md)
- **Bug Reports**: Include request/response details and error messages

## 📝 Contributing

To contribute to the API documentation:

1. Edit the OpenAPI specification in `docs/api/openapi.yaml`
2. Update relevant markdown files
3. Test changes with Swagger UI
4. Submit a pull request with description of changes

## 📚 Additional Resources

- **[OpenAPI 3.0 Specification](https://swagger.io/specification/)**
- **[O-RAN Alliance Specifications](https://www.o-ran.org/specifications)**
- **[Swagger UI Documentation](https://swagger.io/tools/swagger-ui/)**
- **[Postman Learning Center](https://learning.postman.com/)**

---

*Last updated: 2024-01-15*
*API Version: v1.0.0*