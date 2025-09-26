# O-RAN Intent-MANO Developer Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [API Versioning Strategy](#api-versioning-strategy)
3. [Authentication & Security](#authentication--security)
4. [Rate Limiting](#rate-limiting)
5. [Error Handling Best Practices](#error-handling-best-practices)
6. [Webhook Integration](#webhook-integration)
7. [SDK Usage](#sdk-usage)
8. [Testing Strategies](#testing-strategies)
9. [Performance Optimization](#performance-optimization)
10. [Monitoring & Observability](#monitoring--observability)
11. [Common Integration Patterns](#common-integration-patterns)
12. [Troubleshooting](#troubleshooting)

## Getting Started

### Prerequisites

- **API Access**: Valid credentials for the O-RAN Intent-MANO system
- **Development Environment**: Support for HTTP clients and JSON processing
- **Network Access**: Connectivity to the API endpoints

### Quick Start

1. **Obtain API Credentials**
   ```bash
   # Contact your system administrator for:
   # - API endpoint URL
   # - Username and password
   # - Required permissions
   ```

2. **Test Connectivity**
   ```bash
   curl -X GET https://api.oran-mano.io/v1/monitoring/health
   ```

3. **Authenticate**
   ```bash
   curl -X POST https://api.oran-mano.io/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"your-username","password":"your-password"}'
   ```

4. **Make Your First API Call**
   ```bash
   TOKEN="your-access-token"
   curl -X GET https://api.oran-mano.io/v1/orchestrator/intents \
     -H "Authorization: Bearer $TOKEN"
   ```

## API Versioning Strategy

### Version Format

The API uses semantic versioning in the URL path: `/v{major}`

- **Current Version**: v1
- **Version Lifecycle**: Each major version is supported for minimum 18 months
- **Migration Path**: Comprehensive migration guides provided for major version changes

### Version Headers

```http
API-Version: 1.0.0
API-Deprecation-Date: 2025-12-31T23:59:59Z
API-Sunset-Date: 2026-06-30T23:59:59Z
```

### Backward Compatibility

- **Minor Changes**: Additive only (new fields, endpoints)
- **Breaking Changes**: Only in major version updates
- **Deprecation Notice**: 6 months minimum before removal

### Version Migration Example

```python
# Support multiple API versions
class ORANMANOClient:
    def __init__(self, base_url: str, api_version: str = "v1"):
        self.base_url = f"{base_url}/{api_version}"

    def get_intents_v1(self):
        # v1 implementation
        return self._request("GET", "/orchestrator/intents")

    def get_intents_v2(self):
        # v2 implementation with enhanced features
        return self._request("GET", "/orchestrator/intents",
                           headers={"API-Version": "2.0.0"})
```

## Authentication & Security

### JWT Token Management

#### Token Lifecycle
```python
import time
import jwt
from datetime import datetime, timedelta

class TokenManager:
    def __init__(self, username: str, password: str):
        self.username = username
        self.password = password
        self.access_token = None
        self.refresh_token = None
        self.token_expiry = None

    def authenticate(self):
        """Initial authentication"""
        response = requests.post(f"{API_BASE}/auth/login", json={
            "username": self.username,
            "password": self.password
        })

        if response.status_code == 200:
            data = response.json()
            self.access_token = data["access_token"]
            self.refresh_token = data["refresh_token"]

            # Decode JWT to get expiry
            decoded = jwt.decode(self.access_token,
                               options={"verify_signature": False})
            self.token_expiry = datetime.fromtimestamp(decoded["exp"])

    def get_valid_token(self):
        """Get valid access token, refreshing if necessary"""
        if not self.access_token or self.is_token_expired():
            if self.refresh_token:
                self.refresh_access_token()
            else:
                self.authenticate()
        return self.access_token

    def is_token_expired(self):
        """Check if token is expired (with 5 min buffer)"""
        if not self.token_expiry:
            return True
        return datetime.now() > (self.token_expiry - timedelta(minutes=5))

    def refresh_access_token(self):
        """Refresh access token using refresh token"""
        response = requests.post(f"{API_BASE}/auth/refresh", json={
            "refresh_token": self.refresh_token
        })

        if response.status_code == 200:
            data = response.json()
            self.access_token = data["access_token"]
            # Update expiry time
            decoded = jwt.decode(self.access_token,
                               options={"verify_signature": False})
            self.token_expiry = datetime.fromtimestamp(decoded["exp"])
```

#### Security Best Practices

1. **Token Storage**
   ```python
   # ✅ Store tokens securely
   import keyring

   keyring.set_password("oran-mano", "access_token", token)
   token = keyring.get_password("oran-mano", "access_token")

   # ❌ Don't store in plain text files or environment variables
   ```

2. **HTTPS Only**
   ```python
   # ✅ Always use HTTPS
   session = requests.Session()
   session.verify = True  # Verify SSL certificates

   # ❌ Never disable SSL verification in production
   session.verify = False  # Don't do this!
   ```

3. **Input Validation**
   ```python
   def validate_intent_data(intent_data: dict) -> bool:
       """Validate QoS intent data before sending"""
       required_fields = ["bandwidth", "latency", "slice_type"]

       for field in required_fields:
           if field not in intent_data:
               raise ValueError(f"Missing required field: {field}")

       # Validate ranges
       if not (1 <= intent_data["bandwidth"] <= 10000):
           raise ValueError("Bandwidth must be between 1 and 10000 Mbps")

       if not (1 <= intent_data["latency"] <= 1000):
           raise ValueError("Latency must be between 1 and 1000 ms")

       return True
   ```

## Rate Limiting

### Understanding Rate Limits

The API implements different rate limits based on operation type:

```python
RATE_LIMITS = {
    "standard": {"requests": 1000, "window": 60},      # 1000/min
    "deployment": {"requests": 100, "window": 60},     # 100/min
    "monitoring": {"requests": 5000, "window": 60},    # 5000/min
}
```

### Rate Limit Headers

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1609459200
X-RateLimit-Retry-After: 60
```

### Handling Rate Limits

```python
import time
from datetime import datetime

class RateLimitHandler:
    def __init__(self):
        self.rate_limit_reset = {}

    def handle_rate_limit(self, response):
        """Handle rate limit response"""
        if response.status_code == 429:
            retry_after = int(response.headers.get("X-RateLimit-Retry-After", 60))
            reset_time = int(response.headers.get("X-RateLimit-Reset", 0))

            print(f"Rate limit exceeded. Waiting {retry_after} seconds...")
            time.sleep(retry_after)
            return True
        return False

    def make_request_with_retry(self, method, url, **kwargs):
        """Make request with automatic retry on rate limit"""
        max_retries = 3
        retry_count = 0

        while retry_count < max_retries:
            response = requests.request(method, url, **kwargs)

            if response.status_code != 429:
                return response

            if self.handle_rate_limit(response):
                retry_count += 1
                continue
            else:
                break

        return response

# Usage
handler = RateLimitHandler()
response = handler.make_request_with_retry("GET", f"{API_BASE}/orchestrator/intents")
```

### Exponential Backoff Strategy

```python
import random

def exponential_backoff(attempt: int, base_delay: float = 1.0, max_delay: float = 60.0):
    """Calculate delay with exponential backoff and jitter"""
    delay = min(base_delay * (2 ** attempt), max_delay)
    jitter = random.uniform(0, delay * 0.1)  # Add 10% jitter
    return delay + jitter

def make_request_with_backoff(url, **kwargs):
    """Make request with exponential backoff"""
    max_attempts = 5

    for attempt in range(max_attempts):
        try:
            response = requests.get(url, **kwargs)

            if response.status_code == 200:
                return response
            elif response.status_code == 429:
                delay = exponential_backoff(attempt)
                time.sleep(delay)
                continue
            else:
                response.raise_for_status()

        except requests.exceptions.RequestException as e:
            if attempt == max_attempts - 1:
                raise e
            delay = exponential_backoff(attempt)
            time.sleep(delay)

    raise Exception("Max retry attempts exceeded")
```

## Error Handling Best Practices

### Comprehensive Error Handler

```python
import logging
from typing import Optional, Dict, Any

class ORANMANOError(Exception):
    """Base exception for ORAN MANO API errors"""
    def __init__(self, message: str, status_code: int = None,
                 error_type: str = None, details: Dict = None):
        self.message = message
        self.status_code = status_code
        self.error_type = error_type
        self.details = details or {}
        super().__init__(self.message)

class ValidationError(ORANMANOError):
    """Input validation error"""
    pass

class AuthenticationError(ORANMANOError):
    """Authentication failed"""
    pass

class RateLimitError(ORANMANOError):
    """Rate limit exceeded"""
    pass

class APIErrorHandler:
    def __init__(self):
        self.logger = logging.getLogger(__name__)

    def handle_response(self, response):
        """Handle API response and raise appropriate exceptions"""
        if response.status_code >= 200 and response.status_code < 300:
            return response.json() if response.content else None

        try:
            error_data = response.json()
        except ValueError:
            error_data = {"detail": response.text}

        status_code = response.status_code
        title = error_data.get("title", "API Error")
        detail = error_data.get("detail", "An error occurred")
        error_type = error_data.get("type", "unknown")

        # Log error details
        self.logger.error(f"API Error {status_code}: {title} - {detail}")

        # Raise specific exception based on status code
        if status_code == 400:
            raise ValidationError(detail, status_code, error_type, error_data)
        elif status_code == 401:
            raise AuthenticationError(detail, status_code, error_type, error_data)
        elif status_code == 403:
            raise AuthenticationError(f"Access forbidden: {detail}",
                                    status_code, error_type, error_data)
        elif status_code == 404:
            raise ORANMANOError(f"Resource not found: {detail}",
                              status_code, error_type, error_data)
        elif status_code == 409:
            raise ORANMANOError(f"Conflict: {detail}",
                              status_code, error_type, error_data)
        elif status_code == 422:
            # Handle validation errors with field details
            errors = error_data.get("errors", [])
            field_errors = {err.get("field"): err.get("message")
                          for err in errors if "field" in err}
            raise ValidationError(detail, status_code, error_type,
                                {"field_errors": field_errors})
        elif status_code == 429:
            retry_after = response.headers.get("X-RateLimit-Retry-After", "60")
            raise RateLimitError(f"Rate limit exceeded. Retry after {retry_after}s",
                               status_code, error_type, error_data)
        else:
            raise ORANMANOError(f"API Error: {title} - {detail}",
                              status_code, error_type, error_data)

# Usage example
error_handler = APIErrorHandler()

try:
    response = requests.post(f"{API_BASE}/orchestrator/intents",
                           json=intent_data, headers=headers)
    result = error_handler.handle_response(response)
    print("Intent created successfully:", result["id"])

except ValidationError as e:
    print("Validation failed:")
    for field, message in e.details.get("field_errors", {}).items():
        print(f"  {field}: {message}")

except AuthenticationError as e:
    print("Authentication error:", e.message)
    # Re-authenticate

except RateLimitError as e:
    print("Rate limited:", e.message)
    # Implement backoff strategy

except ORANMANOError as e:
    print(f"API error [{e.status_code}]: {e.message}")
```

### Retry Logic with Circuit Breaker

```python
import time
from enum import Enum
from datetime import datetime, timedelta

class CircuitState(Enum):
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"

class CircuitBreaker:
    def __init__(self, failure_threshold=5, timeout=60):
        self.failure_threshold = failure_threshold
        self.timeout = timeout
        self.failure_count = 0
        self.last_failure_time = None
        self.state = CircuitState.CLOSED

    def call(self, func, *args, **kwargs):
        """Execute function with circuit breaker protection"""
        if self.state == CircuitState.OPEN:
            if self._should_attempt_reset():
                self.state = CircuitState.HALF_OPEN
            else:
                raise Exception("Circuit breaker is OPEN")

        try:
            result = func(*args, **kwargs)
            self._on_success()
            return result
        except Exception as e:
            self._on_failure()
            raise e

    def _should_attempt_reset(self):
        """Check if enough time has passed to attempt reset"""
        if self.last_failure_time is None:
            return True
        return datetime.now() > (self.last_failure_time + timedelta(seconds=self.timeout))

    def _on_success(self):
        """Handle successful call"""
        self.failure_count = 0
        self.state = CircuitState.CLOSED

    def _on_failure(self):
        """Handle failed call"""
        self.failure_count += 1
        self.last_failure_time = datetime.now()

        if self.failure_count >= self.failure_threshold:
            self.state = CircuitState.OPEN
```

## Webhook Integration

### Setting Up Webhooks

```python
from flask import Flask, request, jsonify
import hmac
import hashlib

app = Flask(__name__)

WEBHOOK_SECRET = "your-webhook-secret"

def verify_webhook_signature(payload, signature, secret):
    """Verify webhook signature for security"""
    expected_signature = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected_signature}", signature)

@app.route("/webhooks/deployment-status", methods=["POST"])
def handle_deployment_webhook():
    """Handle deployment status change webhooks"""
    # Verify signature
    signature = request.headers.get("X-Webhook-Signature")
    if not verify_webhook_signature(request.data, signature, WEBHOOK_SECRET):
        return jsonify({"error": "Invalid signature"}), 401

    event_data = request.json
    event_type = event_data.get("event")
    deployment_id = event_data.get("deployment_id")
    status = event_data.get("status")

    print(f"Deployment {deployment_id} status changed to {status}")

    # Handle different event types
    if event_type == "deployment.started":
        handle_deployment_started(deployment_id)
    elif event_type == "deployment.completed":
        handle_deployment_completed(deployment_id)
    elif event_type == "deployment.failed":
        handle_deployment_failed(deployment_id, event_data.get("message"))

    return jsonify({"status": "received"}), 200

@app.route("/webhooks/slice-status", methods=["POST"])
def handle_slice_webhook():
    """Handle network slice status change webhooks"""
    signature = request.headers.get("X-Webhook-Signature")
    if not verify_webhook_signature(request.data, signature, WEBHOOK_SECRET):
        return jsonify({"error": "Invalid signature"}), 401

    event_data = request.json
    event_type = event_data.get("event")
    slice_id = event_data.get("slice_id")

    print(f"Slice {slice_id} event: {event_type}")

    if event_type == "slice.deployed":
        # Start monitoring the slice
        start_slice_monitoring(slice_id, event_data.get("metrics"))
    elif event_type == "slice.failed":
        # Alert operations team
        send_alert(f"Slice {slice_id} deployment failed")

    return jsonify({"status": "received"}), 200

def handle_deployment_started(deployment_id):
    """Handle deployment started event"""
    print(f"Starting monitoring for deployment {deployment_id}")
    # Update deployment tracking system
    # Send notifications to relevant teams

def handle_deployment_completed(deployment_id):
    """Handle deployment completed event"""
    print(f"Deployment {deployment_id} completed successfully")
    # Update status in internal systems
    # Trigger post-deployment validation

def handle_deployment_failed(deployment_id, error_message):
    """Handle deployment failed event"""
    print(f"Deployment {deployment_id} failed: {error_message}")
    # Alert operations team
    # Trigger rollback procedures if necessary

def start_slice_monitoring(slice_id, initial_metrics):
    """Start monitoring for deployed slice"""
    print(f"Starting monitoring for slice {slice_id}")
    # Set up monitoring dashboards
    # Configure alerts for SLA violations

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
```

### Webhook Registration

```python
def register_webhook(webhook_url: str, events: list, secret: str):
    """Register webhook endpoint"""
    webhook_data = {
        "url": webhook_url,
        "events": events,
        "secret": secret,
        "active": True
    }

    response = requests.post(
        f"{API_BASE}/webhooks/subscriptions",
        json=webhook_data,
        headers={"Authorization": f"Bearer {access_token}"}
    )

    if response.status_code == 201:
        webhook_id = response.json()["id"]
        print(f"Webhook registered successfully: {webhook_id}")
        return webhook_id
    else:
        print("Failed to register webhook:", response.text)
        return None

# Register webhooks for deployment and slice events
deployment_webhook_id = register_webhook(
    "https://your-domain.com/webhooks/deployment-status",
    ["deployment.started", "deployment.completed", "deployment.failed"],
    WEBHOOK_SECRET
)

slice_webhook_id = register_webhook(
    "https://your-domain.com/webhooks/slice-status",
    ["slice.created", "slice.deployed", "slice.failed", "slice.deleted"],
    WEBHOOK_SECRET
)
```

## SDK Usage

### Python SDK Advanced Usage

```python
import asyncio
import aiohttp
from contextlib import asynccontextmanager

class AsyncORANMANOClient:
    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.session = None
        self.token_manager = TokenManager(username, password)

    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        await self.authenticate()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()

    async def authenticate(self):
        """Async authentication"""
        async with self.session.post(
            f"{self.base_url}/auth/login",
            json={"username": self.username, "password": self.password}
        ) as response:
            if response.status == 200:
                data = await response.json()
                self.token_manager.access_token = data["access_token"]
            else:
                raise AuthenticationError("Failed to authenticate")

    async def create_intent_batch(self, intents: list):
        """Create multiple intents concurrently"""
        tasks = []
        async with asyncio.TaskGroup() as tg:
            for intent in intents:
                task = tg.create_task(self.create_intent(intent))
                tasks.append(task)

        return [task.result() for task in tasks]

    async def create_intent(self, intent_data: dict):
        """Create single intent asynchronously"""
        headers = {"Authorization": f"Bearer {self.token_manager.access_token}"}

        async with self.session.post(
            f"{self.base_url}/orchestrator/intents",
            json=intent_data,
            headers=headers
        ) as response:
            if response.status == 201:
                return await response.json()
            else:
                raise ORANMANOError(f"Failed to create intent: {response.status}")

# Usage
async def main():
    intents = [
        {"bandwidth": 100, "latency": 10, "slice_type": "uRLLC"},
        {"bandwidth": 500, "latency": 50, "slice_type": "eMBB"},
        {"bandwidth": 50, "latency": 100, "slice_type": "mIoT"}
    ]

    async with AsyncORANMANOClient(API_BASE, username, password) as client:
        results = await client.create_intent_batch(intents)
        print(f"Created {len(results)} intents successfully")

asyncio.run(main())
```

### Go SDK Advanced Features

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

type ORANMANOConfig struct {
    BaseURL     string
    Username    string
    Password    string
    Timeout     time.Duration
    MaxRetries  int
    RateLimit   int
}

type ORANMANOClient struct {
    config      ORANMANOConfig
    httpClient  *http.Client
    tokenMgr    *TokenManager
    rateLimiter *rate.Limiter
    circuit     *CircuitBreaker
}

// Connection pool for concurrent operations
type ConnectionPool struct {
    clients []*ORANMANOClient
    current int
    mutex   sync.Mutex
}

func NewConnectionPool(config ORANMANOConfig, poolSize int) (*ConnectionPool, error) {
    pool := &ConnectionPool{
        clients: make([]*ORANMANOClient, poolSize),
    }

    for i := 0; i < poolSize; i++ {
        client, err := NewORANMANOClient(config)
        if err != nil {
            return nil, fmt.Errorf("failed to create client %d: %w", i, err)
        }
        pool.clients[i] = client
    }

    return pool, nil
}

func (p *ConnectionPool) GetClient() *ORANMANOClient {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    client := p.clients[p.current]
    p.current = (p.current + 1) % len(p.clients)
    return client
}

// Concurrent intent creation
func (p *ConnectionPool) CreateIntentsBatch(ctx context.Context, intents []QoSIntent) ([]QoSIntent, error) {
    if len(intents) == 0 {
        return nil, nil
    }

    type result struct {
        intent QoSIntent
        err    error
        index  int
    }

    resultChan := make(chan result, len(intents))
    sem := make(chan struct{}, 10) // Limit concurrent requests

    var wg sync.WaitGroup

    for i, intent := range intents {
        wg.Add(1)
        go func(idx int, intentData QoSIntent) {
            defer wg.Done()

            // Acquire semaphore
            sem <- struct{}{}
            defer func() { <-sem }()

            client := p.GetClient()
            createdIntent, err := client.CreateQoSIntent(ctx, &intentData)

            resultChan <- result{
                intent: *createdIntent,
                err:    err,
                index:  idx,
            }
        }(i, intent)
    }

    // Wait for all goroutines to complete
    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // Collect results
    results := make([]QoSIntent, len(intents))
    errors := make([]error, 0)

    for res := range resultChan {
        if res.err != nil {
            errors = append(errors, fmt.Errorf("intent %d: %w", res.index, res.err))
        } else {
            results[res.index] = res.intent
        }
    }

    if len(errors) > 0 {
        return results, fmt.Errorf("batch creation had %d errors: %v", len(errors), errors)
    }

    return results, nil
}

// Progress tracking for long-running operations
type DeploymentProgress struct {
    ID          string
    Status      string
    Progress    int
    Message     string
    StartedAt   time.Time
    UpdatedAt   time.Time
}

func (c *ORANMANOClient) WaitForDeploymentCompletion(ctx context.Context,
    deploymentID string, progressCallback func(DeploymentProgress)) (*DeploymentStatus, error) {

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    timeout := time.After(30 * time.Minute)

    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-timeout:
            return nil, fmt.Errorf("deployment %s timed out after 30 minutes", deploymentID)
        case <-ticker.C:
            status, err := c.GetDeploymentStatus(ctx, deploymentID)
            if err != nil {
                continue // Retry on error
            }

            // Call progress callback
            if progressCallback != nil {
                progressCallback(DeploymentProgress{
                    ID:        deploymentID,
                    Status:    status.Status,
                    Progress:  status.ProgressPercent,
                    Message:   status.Message,
                    UpdatedAt: time.Now(),
                })
            }

            switch status.Status {
            case "completed":
                return status, nil
            case "failed", "cancelled":
                return status, fmt.Errorf("deployment failed: %s", status.Message)
            }
        }
    }
}

// Usage example
func main() {
    config := ORANMANOConfig{
        BaseURL:    "https://api.oran-mano.io/v1",
        Username:   "admin@oran-mano.io",
        Password:   "SecureP@ssw0rd",
        Timeout:    30 * time.Second,
        MaxRetries: 3,
        RateLimit:  100,
    }

    pool, err := NewConnectionPool(config, 5)
    if err != nil {
        panic(err)
    }

    // Create multiple intents concurrently
    intents := []QoSIntent{
        {Bandwidth: 100, Latency: 10, SliceType: "uRLLC"},
        {Bandwidth: 500, Latency: 50, SliceType: "eMBB"},
        {Bandwidth: 50, Latency: 100, SliceType: "mIoT"},
    }

    ctx := context.Background()
    results, err := pool.CreateIntentsBatch(ctx, intents)
    if err != nil {
        fmt.Printf("Batch creation error: %v\n", err)
    } else {
        fmt.Printf("Created %d intents successfully\n", len(results))
    }

    // Wait for deployment with progress tracking
    client := pool.GetClient()
    deploymentStatus, err := client.WaitForDeploymentCompletion(ctx, "deploy-001",
        func(progress DeploymentProgress) {
            fmt.Printf("Deployment %s: %s (%d%%) - %s\n",
                progress.ID, progress.Status, progress.Progress, progress.Message)
        })

    if err != nil {
        fmt.Printf("Deployment failed: %v\n", err)
    } else {
        fmt.Printf("Deployment completed: %s\n", deploymentStatus.Status)
    }
}
```

## Testing Strategies

### Unit Testing API Clients

```python
import unittest
from unittest.mock import Mock, patch, MagicMock
import requests

class TestORANMANOClient(unittest.TestCase):
    def setUp(self):
        self.client = ORANMANOClient(
            base_url="https://api.test.com/v1",
            username="test@example.com",
            password="test-password"
        )

    @patch('requests.post')
    def test_authentication_success(self, mock_post):
        """Test successful authentication"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "access_token": "test-token",
            "token_type": "Bearer",
            "expires_in": 3600,
            "refresh_token": "refresh-token"
        }
        mock_post.return_value = mock_response

        result = self.client.authenticate("test@example.com", "test-password")

        self.assertTrue(result)
        self.assertEqual(self.client.access_token, "test-token")
        mock_post.assert_called_once()

    @patch('requests.post')
    def test_authentication_failure(self, mock_post):
        """Test authentication failure"""
        mock_response = Mock()
        mock_response.status_code = 401
        mock_response.json.return_value = {
            "status": 401,
            "title": "Unauthorized",
            "detail": "Invalid credentials"
        }
        mock_post.return_value = mock_response

        with self.assertRaises(AuthenticationError):
            self.client.authenticate("test@example.com", "wrong-password")

    @patch('requests.post')
    def test_create_intent_success(self, mock_post):
        """Test successful intent creation"""
        self.client.access_token = "test-token"

        mock_response = Mock()
        mock_response.status_code = 201
        mock_response.json.return_value = {
            "id": "intent-001",
            "bandwidth": 100.0,
            "latency": 10.0,
            "slice_type": "uRLLC",
            "status": "planned"
        }
        mock_post.return_value = mock_response

        intent_data = {
            "bandwidth": 100.0,
            "latency": 10.0,
            "slice_type": "uRLLC"
        }

        result = self.client.create_qos_intent(**intent_data)

        self.assertEqual(result["id"], "intent-001")
        self.assertEqual(result["slice_type"], "uRLLC")

    @patch('requests.post')
    def test_create_intent_validation_error(self, mock_post):
        """Test intent creation with validation error"""
        self.client.access_token = "test-token"

        mock_response = Mock()
        mock_response.status_code = 422
        mock_response.json.return_value = {
            "status": 422,
            "title": "Validation Error",
            "detail": "Input validation failed",
            "errors": [
                {"field": "bandwidth", "message": "bandwidth must be between 1 and 10000", "code": "RANGE_ERROR"}
            ]
        }
        mock_post.return_value = mock_response

        intent_data = {
            "bandwidth": 50000.0,  # Invalid value
            "latency": 10.0,
            "slice_type": "uRLLC"
        }

        with self.assertRaises(ValidationError) as context:
            self.client.create_qos_intent(**intent_data)

        self.assertIn("bandwidth", str(context.exception))

class TestRateLimitHandler(unittest.TestCase):
    def setUp(self):
        self.handler = RateLimitHandler()

    def test_rate_limit_detection(self):
        """Test rate limit detection"""
        mock_response = Mock()
        mock_response.status_code = 429
        mock_response.headers = {
            "X-RateLimit-Retry-After": "60",
            "X-RateLimit-Reset": "1609459200"
        }

        with patch('time.sleep') as mock_sleep:
            result = self.handler.handle_rate_limit(mock_response)

        self.assertTrue(result)
        mock_sleep.assert_called_once_with(60)

    def test_no_rate_limit(self):
        """Test normal response handling"""
        mock_response = Mock()
        mock_response.status_code = 200

        result = self.handler.handle_rate_limit(mock_response)
        self.assertFalse(result)

if __name__ == '__main__':
    unittest.main()
```

### Integration Testing

```python
import pytest
import time
from contextlib import contextmanager

class TestORANMANOIntegration:
    @pytest.fixture(scope="class")
    def client(self):
        """Setup client for integration tests"""
        client = ORANMANOClient(
            base_url=os.getenv("ORAN_MANO_API_URL", "https://api.staging.oran-mano.io/v1"),
            username=os.getenv("ORAN_MANO_USERNAME"),
            password=os.getenv("ORAN_MANO_PASSWORD")
        )
        client.authenticate()
        return client

    @contextmanager
    def temporary_intent(self, client, intent_data):
        """Context manager for creating and cleaning up test intents"""
        intent = client.create_qos_intent(**intent_data)
        try:
            yield intent
        finally:
            try:
                client.delete_qos_intent(intent["id"])
            except Exception as e:
                print(f"Failed to cleanup intent {intent['id']}: {e}")

    def test_intent_lifecycle(self, client):
        """Test complete intent lifecycle"""
        intent_data = {
            "bandwidth": 100.0,
            "latency": 10.0,
            "slice_type": "uRLLC",
            "reliability": 0.9999
        }

        with self.temporary_intent(client, intent_data) as intent:
            # Verify intent was created
            assert intent["id"] is not None
            assert intent["slice_type"] == "uRLLC"
            assert intent["status"] == "planned"

            # Retrieve intent
            retrieved = client.get_qos_intent(intent["id"])
            assert retrieved["id"] == intent["id"]

            # Update intent
            updated_data = intent_data.copy()
            updated_data["bandwidth"] = 150.0
            updated = client.update_qos_intent(intent["id"], updated_data)
            assert updated["bandwidth"] == 150.0

    def test_orchestration_workflow(self, client):
        """Test complete orchestration workflow"""
        # Create multiple intents
        intents_data = [
            {"bandwidth": 100, "latency": 10, "slice_type": "uRLLC"},
            {"bandwidth": 500, "latency": 50, "slice_type": "eMBB"}
        ]

        created_intents = []
        try:
            for intent_data in intents_data:
                intent = client.create_qos_intent(**intent_data)
                created_intents.append(intent)

            # Generate orchestration plan
            plan = client.generate_orchestration_plan({
                "intents": intents_data,
                "dry_run": True
            })

            assert plan["total_slices"] == 2
            assert len(plan["allocations"]) == 2

            # Verify plan contains valid allocations
            for allocation in plan["allocations"]:
                assert allocation["slice_id"] is not None
                assert allocation["placement"]["site_id"] is not None
                assert allocation["resources"]["ran_resources"] is not None

        finally:
            # Cleanup
            for intent in created_intents:
                try:
                    client.delete_qos_intent(intent["id"])
                except Exception as e:
                    print(f"Failed to cleanup intent {intent['id']}: {e}")

    def test_performance_under_load(self, client):
        """Test API performance under load"""
        import concurrent.futures
        import statistics

        def create_and_delete_intent():
            start_time = time.time()
            intent_data = {
                "bandwidth": 100.0,
                "latency": 10.0,
                "slice_type": "uRLLC"
            }

            intent = client.create_qos_intent(**intent_data)
            client.delete_qos_intent(intent["id"])

            return time.time() - start_time

        # Run 50 concurrent operations
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(create_and_delete_intent) for _ in range(50)]
            durations = [future.result() for future in concurrent.futures.as_completed(futures)]

        # Verify performance metrics
        avg_duration = statistics.mean(durations)
        max_duration = max(durations)

        assert avg_duration < 2.0, f"Average duration too high: {avg_duration}s"
        assert max_duration < 5.0, f"Max duration too high: {max_duration}s"

        print(f"Performance test completed:")
        print(f"  Average duration: {avg_duration:.2f}s")
        print(f"  Max duration: {max_duration:.2f}s")
        print(f"  95th percentile: {statistics.quantiles(durations, n=20)[18]:.2f}s")
```

### Load Testing with Locust

```python
from locust import HttpUser, task, between
import random
import json

class ORANMANOUser(HttpUser):
    wait_time = between(1, 3)

    def on_start(self):
        """Login when user starts"""
        response = self.client.post("/auth/login", json={
            "username": "load-test@oran-mano.io",
            "password": "load-test-password"
        })

        if response.status_code == 200:
            token_data = response.json()
            self.access_token = token_data["access_token"]
            self.client.headers.update({
                "Authorization": f"Bearer {self.access_token}"
            })
        else:
            raise Exception("Failed to authenticate")

    @task(10)
    def list_intents(self):
        """List QoS intents (most common operation)"""
        self.client.get("/orchestrator/intents")

    @task(5)
    def get_specific_intent(self):
        """Get specific intent"""
        # Use a pre-existing intent ID or create one
        intent_id = "intent-001"  # Replace with actual ID
        self.client.get(f"/orchestrator/intents/{intent_id}")

    @task(3)
    def create_intent(self):
        """Create new QoS intent"""
        slice_types = ["uRLLC", "eMBB", "mIoT", "balanced"]
        intent_data = {
            "bandwidth": random.uniform(10, 1000),
            "latency": random.uniform(1, 100),
            "slice_type": random.choice(slice_types),
            "reliability": random.uniform(0.95, 0.99999)
        }

        response = self.client.post("/orchestrator/intents", json=intent_data)

        if response.status_code == 201:
            intent = response.json()
            # Store intent ID for potential deletion
            if not hasattr(self, 'created_intents'):
                self.created_intents = []
            self.created_intents.append(intent["id"])

    @task(1)
    def delete_intent(self):
        """Delete created intent"""
        if hasattr(self, 'created_intents') and self.created_intents:
            intent_id = self.created_intents.pop()
            self.client.delete(f"/orchestrator/intents/{intent_id}")

    @task(2)
    def list_vnfs(self):
        """List VNFs"""
        self.client.get("/vnf-operator/vnfs")

    @task(1)
    def check_health(self):
        """Check system health"""
        self.client.get("/monitoring/health")

    @task(2)
    def get_metrics(self):
        """Get system metrics"""
        self.client.get("/monitoring/metrics")

# Run with: locust -f load_test.py --host=https://api.oran-mano.io/v1
```

## Performance Optimization

### Caching Strategies

```python
import redis
import pickle
import hashlib
from functools import wraps
from datetime import timedelta

class APICache:
    def __init__(self, redis_url="redis://localhost:6379"):
        self.redis_client = redis.from_url(redis_url)

    def cache_key(self, prefix: str, *args, **kwargs) -> str:
        """Generate cache key from function arguments"""
        key_data = f"{prefix}:{args}:{sorted(kwargs.items())}"
        return hashlib.md5(key_data.encode()).hexdigest()

    def cached_request(self, ttl_seconds=300):
        """Decorator for caching API responses"""
        def decorator(func):
            @wraps(func)
            def wrapper(self, *args, **kwargs):
                cache_key = self.cache_key(func.__name__, *args, **kwargs)

                # Try to get from cache
                cached_result = self.redis_client.get(cache_key)
                if cached_result:
                    return pickle.loads(cached_result)

                # Execute function and cache result
                result = func(self, *args, **kwargs)
                self.redis_client.setex(
                    cache_key,
                    ttl_seconds,
                    pickle.dumps(result)
                )

                return result
            return wrapper
        return decorator

class CachedORANMANOClient(ORANMANOClient):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.cache = APICache()

    @APICache().cached_request(ttl_seconds=60)  # Cache for 1 minute
    def list_qos_intents(self, slice_type=None, status=None, limit=50, offset=0):
        """Cached version of list intents"""
        return super().list_qos_intents(slice_type, status, limit, offset)

    @APICache().cached_request(ttl_seconds=300)  # Cache for 5 minutes
    def list_vnfs(self, vnf_type=None, status=None):
        """Cached version of list VNFs"""
        return super().list_vnfs(vnf_type, status)

    def invalidate_cache(self, pattern: str):
        """Invalidate cache entries matching pattern"""
        keys = self.cache.redis_client.keys(f"*{pattern}*")
        if keys:
            self.cache.redis_client.delete(*keys)
```

### Connection Pooling

```python
import requests
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

class OptimizedORANMANOClient:
    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url
        self.session = self._create_optimized_session()
        self.token_manager = TokenManager(username, password)

    def _create_optimized_session(self):
        """Create session with connection pooling and retry strategy"""
        session = requests.Session()

        # Configure retry strategy
        retry_strategy = Retry(
            total=3,
            status_forcelist=[429, 500, 502, 503, 504],
            method_whitelist=["HEAD", "GET", "OPTIONS", "POST", "PUT", "DELETE"],
            backoff_factor=1
        )

        # Configure HTTP adapter with connection pooling
        adapter = HTTPAdapter(
            pool_connections=20,    # Number of connection pools
            pool_maxsize=20,        # Max connections per pool
            max_retries=retry_strategy,
            pool_block=False
        )

        session.mount("http://", adapter)
        session.mount("https://", adapter)

        # Set default timeouts
        session.timeout = (10, 30)  # (connect, read) timeout

        return session

    def close(self):
        """Close session and connections"""
        if self.session:
            self.session.close()

# Usage with context manager
with OptimizedORANMANOClient(API_BASE, username, password) as client:
    intents = client.list_qos_intents()
    print(f"Found {len(intents)} intents")
```

### Batch Operations

```python
class BatchORANMANOClient(ORANMANOClient):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.batch_size = 50

    def create_intents_batch(self, intents_data: list, batch_size: int = None):
        """Create multiple intents in batches"""
        if batch_size is None:
            batch_size = self.batch_size

        results = []
        errors = []

        for i in range(0, len(intents_data), batch_size):
            batch = intents_data[i:i + batch_size]

            try:
                batch_results = self._create_intent_batch(batch)
                results.extend(batch_results)
            except Exception as e:
                errors.append(f"Batch {i//batch_size + 1}: {e}")

        return {
            "created": results,
            "errors": errors,
            "total_attempted": len(intents_data),
            "success_count": len(results),
            "error_count": len(errors)
        }

    def _create_intent_batch(self, batch: list):
        """Create a single batch of intents"""
        # Use concurrent.futures for parallel creation
        import concurrent.futures

        results = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            future_to_intent = {
                executor.submit(self.create_qos_intent, **intent_data): intent_data
                for intent_data in batch
            }

            for future in concurrent.futures.as_completed(future_to_intent):
                try:
                    result = future.result()
                    results.append(result)
                except Exception as e:
                    intent_data = future_to_intent[future]
                    print(f"Failed to create intent {intent_data}: {e}")

        return results
```

## Monitoring & Observability

### Comprehensive Monitoring Setup

```python
import logging
import time
from prometheus_client import Counter, Histogram, Gauge, start_http_server
from opentelemetry import trace
from opentelemetry.exporter.jaeger.thrift import JaegerExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

# Prometheus metrics
API_REQUESTS_TOTAL = Counter('oran_mano_api_requests_total',
                            'Total API requests', ['method', 'endpoint', 'status_code'])
API_REQUEST_DURATION = Histogram('oran_mano_api_request_duration_seconds',
                                'API request duration', ['method', 'endpoint'])
ACTIVE_CONNECTIONS = Gauge('oran_mano_active_connections',
                          'Number of active connections')

# Setup tracing
trace.set_tracer_provider(TracerProvider())
jaeger_exporter = JaegerExporter(
    agent_host_name="localhost",
    agent_port=6831,
)
span_processor = BatchSpanProcessor(jaeger_exporter)
trace.get_tracer_provider().add_span_processor(span_processor)
tracer = trace.get_tracer(__name__)

class MonitoredORANMANOClient(ORANMANOClient):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.logger = self._setup_logging()

    def _setup_logging(self):
        """Setup structured logging"""
        logger = logging.getLogger("oran_mano_client")
        logger.setLevel(logging.INFO)

        handler = logging.StreamHandler()
        formatter = logging.Formatter(
            '{"timestamp": "%(asctime)s", "level": "%(levelname)s", '
            '"component": "oran_mano_client", "message": "%(message)s", '
            '"extra": %(extra)s}'
        )
        handler.setFormatter(formatter)
        logger.addHandler(handler)

        return logger

    def _make_request(self, method: str, endpoint: str, **kwargs):
        """Monitored request method"""
        with tracer.start_as_current_span(f"{method} {endpoint}") as span:
            start_time = time.time()

            try:
                ACTIVE_CONNECTIONS.inc()

                # Add span attributes
                span.set_attribute("http.method", method)
                span.set_attribute("http.url", f"{self.base_url}{endpoint}")

                response = super()._make_request(method, endpoint, **kwargs)

                # Record metrics
                duration = time.time() - start_time
                API_REQUEST_DURATION.labels(method=method, endpoint=endpoint).observe(duration)
                API_REQUESTS_TOTAL.labels(
                    method=method,
                    endpoint=endpoint,
                    status_code=response.status_code
                ).inc()

                # Log request
                self.logger.info(
                    f"API request completed",
                    extra={
                        "method": method,
                        "endpoint": endpoint,
                        "status_code": response.status_code,
                        "duration_ms": duration * 1000,
                        "response_size": len(response.content) if response.content else 0
                    }
                )

                span.set_attribute("http.status_code", response.status_code)
                span.set_attribute("http.response_size", len(response.content) if response.content else 0)

                return response

            except Exception as e:
                # Record error metrics
                API_REQUESTS_TOTAL.labels(
                    method=method,
                    endpoint=endpoint,
                    status_code="error"
                ).inc()

                # Log error
                self.logger.error(
                    f"API request failed",
                    extra={
                        "method": method,
                        "endpoint": endpoint,
                        "error": str(e),
                        "duration_ms": (time.time() - start_time) * 1000
                    }
                )

                span.record_exception(e)
                span.set_status(trace.Status(trace.StatusCode.ERROR, str(e)))

                raise
            finally:
                ACTIVE_CONNECTIONS.dec()

# Start Prometheus metrics server
start_http_server(8000)

# Usage
client = MonitoredORANMANOClient(API_BASE, username, password)
```

### Health Check Implementation

```python
class HealthChecker:
    def __init__(self, client: ORANMANOClient):
        self.client = client
        self.last_check = None
        self.last_status = None

    def check_api_health(self) -> dict:
        """Comprehensive API health check"""
        checks = {
            "timestamp": time.time(),
            "overall_status": "healthy",
            "checks": {}
        }

        # Test authentication
        try:
            self.client.authenticate()
            checks["checks"]["authentication"] = {
                "status": "healthy",
                "message": "Authentication successful"
            }
        except Exception as e:
            checks["checks"]["authentication"] = {
                "status": "unhealthy",
                "message": f"Authentication failed: {e}"
            }
            checks["overall_status"] = "unhealthy"

        # Test basic API endpoints
        endpoints_to_test = [
            ("/monitoring/health", "Health endpoint"),
            ("/orchestrator/intents", "Orchestrator API"),
            ("/vnf-operator/vnfs", "VNF Operator API"),
            ("/tn-manager/agents", "TN Manager API")
        ]

        for endpoint, description in endpoints_to_test:
            try:
                start_time = time.time()
                response = self.client.session.get(f"{self.client.base_url}{endpoint}")
                duration = time.time() - start_time

                if response.status_code == 200:
                    checks["checks"][endpoint] = {
                        "status": "healthy",
                        "message": f"{description} responding",
                        "response_time_ms": duration * 1000
                    }
                else:
                    checks["checks"][endpoint] = {
                        "status": "unhealthy",
                        "message": f"{description} returned {response.status_code}",
                        "response_time_ms": duration * 1000
                    }
                    checks["overall_status"] = "unhealthy"

            except Exception as e:
                checks["checks"][endpoint] = {
                    "status": "unhealthy",
                    "message": f"{description} error: {e}"
                }
                checks["overall_status"] = "unhealthy"

        self.last_check = checks
        self.last_status = checks["overall_status"]

        return checks

    def continuous_health_monitoring(self, interval_seconds=30):
        """Run continuous health monitoring"""
        import threading

        def monitor():
            while True:
                try:
                    health_status = self.check_api_health()

                    if health_status["overall_status"] != "healthy":
                        # Send alert
                        self.send_health_alert(health_status)

                    # Log health status
                    print(f"Health check: {health_status['overall_status']}")

                except Exception as e:
                    print(f"Health check failed: {e}")

                time.sleep(interval_seconds)

        monitor_thread = threading.Thread(target=monitor, daemon=True)
        monitor_thread.start()

    def send_health_alert(self, health_status: dict):
        """Send health alert (implement your preferred alerting method)"""
        print(f"🚨 HEALTH ALERT: API status is {health_status['overall_status']}")
        for check_name, check_result in health_status["checks"].items():
            if check_result["status"] != "healthy":
                print(f"  - {check_name}: {check_result['message']}")
```

## Common Integration Patterns

### Event-Driven Architecture

```python
import asyncio
import aiohttp
from typing import Callable, Dict, List

class EventDrivenORANMANOClient:
    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.event_handlers: Dict[str, List[Callable]] = {}
        self.polling_tasks = []

    def on_event(self, event_type: str):
        """Decorator for registering event handlers"""
        def decorator(func):
            if event_type not in self.event_handlers:
                self.event_handlers[event_type] = []
            self.event_handlers[event_type].append(func)
            return func
        return decorator

    async def emit_event(self, event_type: str, event_data: dict):
        """Emit event to all registered handlers"""
        if event_type in self.event_handlers:
            for handler in self.event_handlers[event_type]:
                try:
                    if asyncio.iscoroutinefunction(handler):
                        await handler(event_data)
                    else:
                        handler(event_data)
                except Exception as e:
                    print(f"Event handler error for {event_type}: {e}")

    async def start_deployment_monitoring(self, deployment_id: str):
        """Monitor deployment status and emit events"""
        async def monitor():
            previous_status = None

            while True:
                try:
                    status = await self.get_deployment_status(deployment_id)

                    if status["status"] != previous_status:
                        await self.emit_event("deployment_status_changed", {
                            "deployment_id": deployment_id,
                            "previous_status": previous_status,
                            "current_status": status["status"],
                            "progress": status.get("progress_percent", 0),
                            "message": status.get("message", "")
                        })

                        previous_status = status["status"]

                        # Stop monitoring if deployment is complete
                        if status["status"] in ["completed", "failed", "cancelled"]:
                            break

                except Exception as e:
                    await self.emit_event("monitoring_error", {
                        "deployment_id": deployment_id,
                        "error": str(e)
                    })

                await asyncio.sleep(5)  # Poll every 5 seconds

        task = asyncio.create_task(monitor())
        self.polling_tasks.append(task)
        return task

# Usage example
client = EventDrivenORANMANOClient(API_BASE, username, password)

@client.on_event("deployment_status_changed")
async def handle_deployment_status(event_data):
    deployment_id = event_data["deployment_id"]
    status = event_data["current_status"]

    print(f"Deployment {deployment_id} status: {status}")

    if status == "completed":
        # Trigger post-deployment actions
        await post_deployment_validation(deployment_id)
    elif status == "failed":
        # Trigger failure handling
        await handle_deployment_failure(deployment_id, event_data["message"])

@client.on_event("monitoring_error")
def handle_monitoring_error(event_data):
    print(f"Monitoring error for {event_data['deployment_id']}: {event_data['error']}")
    # Implement error handling logic

async def post_deployment_validation(deployment_id: str):
    """Run validation after successful deployment"""
    print(f"Running post-deployment validation for {deployment_id}")
    # Implement validation logic

async def handle_deployment_failure(deployment_id: str, error_message: str):
    """Handle deployment failure"""
    print(f"Deployment {deployment_id} failed: {error_message}")
    # Implement failure handling logic
```

### Workflow Orchestration

```python
from enum import Enum
from dataclasses import dataclass
from typing import List, Optional, Callable
import asyncio

class WorkflowStatus(Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"

@dataclass
class WorkflowStep:
    name: str
    action: Callable
    dependencies: List[str] = None
    retry_count: int = 3
    timeout_seconds: int = 300

class WorkflowOrchestrator:
    def __init__(self, client: ORANMANOClient):
        self.client = client
        self.workflows = {}

    async def execute_workflow(self, workflow_id: str, steps: List[WorkflowStep],
                             context: dict = None) -> dict:
        """Execute workflow with dependency management"""
        context = context or {}

        workflow_state = {
            "id": workflow_id,
            "status": WorkflowStatus.RUNNING,
            "steps": {step.name: {"status": "pending", "result": None} for step in steps},
            "context": context,
            "start_time": time.time()
        }

        self.workflows[workflow_id] = workflow_state

        try:
            # Build dependency graph
            dependency_graph = self._build_dependency_graph(steps)

            # Execute steps in topological order
            completed_steps = set()

            while len(completed_steps) < len(steps):
                # Find steps ready to execute
                ready_steps = [
                    step for step in steps
                    if (step.name not in completed_steps and
                        all(dep in completed_steps for dep in (step.dependencies or [])))
                ]

                if not ready_steps:
                    raise Exception("Circular dependency detected or no steps ready")

                # Execute ready steps in parallel
                tasks = []
                for step in ready_steps:
                    task = asyncio.create_task(
                        self._execute_step(workflow_id, step, context)
                    )
                    tasks.append((step.name, task))

                # Wait for completion
                for step_name, task in tasks:
                    try:
                        result = await task
                        workflow_state["steps"][step_name] = {
                            "status": "completed",
                            "result": result
                        }
                        completed_steps.add(step_name)
                        context[f"step_{step_name}_result"] = result

                    except Exception as e:
                        workflow_state["steps"][step_name] = {
                            "status": "failed",
                            "error": str(e)
                        }
                        workflow_state["status"] = WorkflowStatus.FAILED
                        raise e

            workflow_state["status"] = WorkflowStatus.COMPLETED
            workflow_state["end_time"] = time.time()

        except Exception as e:
            workflow_state["status"] = WorkflowStatus.FAILED
            workflow_state["error"] = str(e)
            workflow_state["end_time"] = time.time()
            raise e

        return workflow_state

    async def _execute_step(self, workflow_id: str, step: WorkflowStep, context: dict):
        """Execute individual workflow step with retry logic"""
        for attempt in range(step.retry_count):
            try:
                # Execute step with timeout
                result = await asyncio.wait_for(
                    step.action(self.client, context),
                    timeout=step.timeout_seconds
                )
                return result

            except Exception as e:
                if attempt == step.retry_count - 1:
                    raise e
                await asyncio.sleep(2 ** attempt)  # Exponential backoff

    def _build_dependency_graph(self, steps: List[WorkflowStep]) -> dict:
        """Build dependency graph for workflow steps"""
        graph = {}
        for step in steps:
            graph[step.name] = step.dependencies or []
        return graph

# Define workflow steps
async def create_intent_step(client: ORANMANOClient, context: dict):
    """Step: Create QoS intent"""
    intent_data = context["intent_data"]
    intent = await client.create_qos_intent(**intent_data)
    return intent

async def generate_plan_step(client: ORANMANOClient, context: dict):
    """Step: Generate orchestration plan"""
    intent = context["step_create_intent_result"]
    plan = await client.generate_orchestration_plan({
        "intents": [intent],
        "dry_run": False
    })
    return plan

async def deploy_vnfs_step(client: ORANMANOClient, context: dict):
    """Step: Deploy VNFs"""
    plan = context["step_generate_plan_result"]

    deployments = []
    for allocation in plan["allocations"]:
        # Deploy VNFs based on allocation
        vnf_data = {
            "name": f"vnf-{allocation['slice_id']}",
            "type": "UPF",  # Determine from allocation
            "placement": allocation["placement"],
            "resources": allocation["resources"]
        }
        vnf = await client.deploy_vnf(vnf_data)
        deployments.append(vnf)

    return deployments

async def configure_tn_step(client: ORANMANOClient, context: dict):
    """Step: Configure transport network"""
    plan = context["step_generate_plan_result"]

    configurations = []
    for allocation in plan["allocations"]:
        tn_config = {
            "slice_id": allocation["slice_id"],
            "bandwidth_mbps": allocation["resources"]["tn_resources"]["bandwidth_mbps"],
            "qos_class": allocation["qos"]["slice_type"]
        }
        config_result = await client.configure_tn_slice("agent-01", tn_config)
        configurations.append(config_result)

    return configurations

# Usage
async def deploy_complete_slice():
    client = ORANMANOClient(API_BASE, username, password)
    orchestrator = WorkflowOrchestrator(client)

    # Define workflow
    workflow_steps = [
        WorkflowStep(
            name="create_intent",
            action=create_intent_step,
            dependencies=[]
        ),
        WorkflowStep(
            name="generate_plan",
            action=generate_plan_step,
            dependencies=["create_intent"]
        ),
        WorkflowStep(
            name="deploy_vnfs",
            action=deploy_vnfs_step,
            dependencies=["generate_plan"]
        ),
        WorkflowStep(
            name="configure_tn",
            action=configure_tn_step,
            dependencies=["generate_plan"]
        )
    ]

    # Execute workflow
    context = {
        "intent_data": {
            "bandwidth": 100.0,
            "latency": 10.0,
            "slice_type": "uRLLC"
        }
    }

    try:
        result = await orchestrator.execute_workflow(
            "slice-deployment-001",
            workflow_steps,
            context
        )
        print(f"Workflow completed: {result['status']}")

    except Exception as e:
        print(f"Workflow failed: {e}")

# Run the workflow
asyncio.run(deploy_complete_slice())
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Authentication Problems

```python
def diagnose_auth_issues(client: ORANMANOClient):
    """Diagnose common authentication issues"""
    print("🔍 Diagnosing authentication issues...")

    # Check network connectivity
    try:
        response = requests.get(f"{client.base_url}/monitoring/health", timeout=10)
        print(f"✅ Network connectivity: OK (status: {response.status_code})")
    except requests.exceptions.ConnectTimeout:
        print("❌ Network connectivity: Connection timeout")
        print("   Solution: Check network connection and firewall settings")
        return
    except requests.exceptions.ConnectionError:
        print("❌ Network connectivity: Connection error")
        print("   Solution: Verify API URL and network configuration")
        return

    # Test authentication endpoint
    try:
        auth_response = requests.post(
            f"{client.base_url}/auth/login",
            json={"username": client.username, "password": client.password},
            timeout=10
        )

        if auth_response.status_code == 200:
            print("✅ Authentication: Credentials are valid")

            # Check token format
            token_data = auth_response.json()
            if "access_token" in token_data:
                print("✅ Token format: Valid")

                # Test token usage
                test_response = requests.get(
                    f"{client.base_url}/orchestrator/intents",
                    headers={"Authorization": f"Bearer {token_data['access_token']}"},
                    timeout=10
                )

                if test_response.status_code == 200:
                    print("✅ Token usage: Working correctly")
                elif test_response.status_code == 401:
                    print("❌ Token usage: Token rejected")
                    print("   Solution: Check token format and ensure proper Bearer prefix")
                else:
                    print(f"⚠️  Token usage: Unexpected status {test_response.status_code}")

            else:
                print("❌ Token format: Missing access_token in response")

        elif auth_response.status_code == 401:
            print("❌ Authentication: Invalid credentials")
            print("   Solution: Verify username and password")
        elif auth_response.status_code == 429:
            print("❌ Authentication: Rate limited")
            print("   Solution: Wait before retrying or check rate limits")
        else:
            print(f"❌ Authentication: Unexpected status {auth_response.status_code}")
            print(f"   Response: {auth_response.text}")

    except Exception as e:
        print(f"❌ Authentication test failed: {e}")
```

#### 2. Rate Limiting Issues

```python
def diagnose_rate_limit_issues(client: ORANMANOClient):
    """Diagnose and handle rate limiting"""
    print("🔍 Diagnosing rate limit issues...")

    # Check current rate limit status
    try:
        response = client.session.get(f"{client.base_url}/orchestrator/intents?limit=1")

        rate_limit_info = {
            "limit": response.headers.get("X-RateLimit-Limit"),
            "remaining": response.headers.get("X-RateLimit-Remaining"),
            "reset": response.headers.get("X-RateLimit-Reset")
        }

        print(f"Current rate limit status:")
        print(f"  Limit: {rate_limit_info['limit']} requests")
        print(f"  Remaining: {rate_limit_info['remaining']} requests")

        if rate_limit_info['reset']:
            reset_time = datetime.fromtimestamp(int(rate_limit_info['reset']))
            print(f"  Reset time: {reset_time}")

        remaining = int(rate_limit_info['remaining'] or 0)
        if remaining < 10:
            print("⚠️  Warning: Low remaining requests")
            print("   Solution: Implement request throttling or wait for reset")

    except Exception as e:
        print(f"Failed to check rate limit status: {e}")

def implement_adaptive_rate_limiting(client: ORANMANOClient):
    """Implement adaptive rate limiting"""
    class AdaptiveRateLimiter:
        def __init__(self):
            self.current_delay = 0
            self.success_count = 0
            self.consecutive_failures = 0

        def adjust_delay(self, response):
            """Adjust delay based on response"""
            if response.status_code == 429:
                self.consecutive_failures += 1
                retry_after = int(response.headers.get("X-RateLimit-Retry-After", 60))
                self.current_delay = max(self.current_delay * 2, retry_after)
            else:
                self.success_count += 1
                self.consecutive_failures = 0

                # Gradually reduce delay on success
                if self.success_count > 5:
                    self.current_delay = max(0, self.current_delay * 0.9)
                    self.success_count = 0

        def get_delay(self):
            """Get current delay"""
            return self.current_delay

    return AdaptiveRateLimiter()
```

#### 3. Validation Errors

```python
def diagnose_validation_errors(error_response: dict):
    """Diagnose and explain validation errors"""
    print("🔍 Diagnosing validation errors...")

    if "errors" in error_response:
        print("Validation errors found:")

        for error in error_response["errors"]:
            field = error.get("field", "unknown")
            message = error.get("message", "No message")
            code = error.get("code", "unknown")

            print(f"  ❌ Field: {field}")
            print(f"     Error: {message}")
            print(f"     Code: {code}")

            # Provide specific solutions
            if code == "RANGE_ERROR":
                if "bandwidth" in field.lower():
                    print("     💡 Solution: Bandwidth must be between 1 and 10000 Mbps")
                elif "latency" in field.lower():
                    print("     💡 Solution: Latency must be between 1 and 1000 ms")
            elif code == "ENUM_ERROR":
                if "slice_type" in field.lower():
                    print("     💡 Solution: slice_type must be one of: eMBB, uRLLC, mIoT, balanced")
            elif code == "REQUIRED_ERROR":
                print(f"     💡 Solution: Field '{field}' is required and cannot be empty")

def validate_intent_data_locally(intent_data: dict) -> list:
    """Local validation to catch errors before API call"""
    errors = []

    # Required fields
    required_fields = ["bandwidth", "latency", "slice_type"]
    for field in required_fields:
        if field not in intent_data:
            errors.append(f"Missing required field: {field}")

    # Range validation
    if "bandwidth" in intent_data:
        if not (1 <= intent_data["bandwidth"] <= 10000):
            errors.append("Bandwidth must be between 1 and 10000 Mbps")

    if "latency" in intent_data:
        if not (1 <= intent_data["latency"] <= 1000):
            errors.append("Latency must be between 1 and 1000 ms")

    # Enum validation
    if "slice_type" in intent_data:
        valid_types = ["eMBB", "uRLLC", "mIoT", "balanced"]
        if intent_data["slice_type"] not in valid_types:
            errors.append(f"slice_type must be one of: {', '.join(valid_types)}")

    return errors

# Usage
intent_data = {
    "bandwidth": 15000,  # Invalid - too high
    "latency": 5,
    "slice_type": "invalid_type"  # Invalid enum
}

validation_errors = validate_intent_data_locally(intent_data)
if validation_errors:
    print("❌ Local validation failed:")
    for error in validation_errors:
        print(f"  - {error}")
else:
    print("✅ Local validation passed")
```

#### 4. Performance Issues

```python
def diagnose_performance_issues(client: ORANMANOClient):
    """Diagnose API performance issues"""
    print("🔍 Diagnosing performance issues...")

    # Test response times for different endpoints
    endpoints = [
        "/monitoring/health",
        "/orchestrator/intents",
        "/vnf-operator/vnfs",
        "/tn-manager/agents"
    ]

    for endpoint in endpoints:
        try:
            start_time = time.time()
            response = client.session.get(f"{client.base_url}{endpoint}")
            duration = time.time() - start_time

            status = "✅" if duration < 2.0 else "⚠️" if duration < 5.0 else "❌"
            print(f"{status} {endpoint}: {duration:.2f}s")

            if duration > 5.0:
                print(f"   💡 Solution: Endpoint is slow, consider:")
                print(f"      - Using pagination for list endpoints")
                print(f"      - Implementing client-side caching")
                print(f"      - Checking network connectivity")

        except Exception as e:
            print(f"❌ {endpoint}: Failed ({e})")

    # Check connection pooling
    if hasattr(client.session, 'adapters'):
        print("\n🔍 Connection pool status:")
        for prefix, adapter in client.session.adapters.items():
            if hasattr(adapter, 'poolmanager'):
                pool_manager = adapter.poolmanager
                print(f"  {prefix}: {pool_manager.num_pools} pools")
            else:
                print(f"  {prefix}: Basic adapter (no pooling)")
                print("     💡 Solution: Use HTTPAdapter with connection pooling")

def optimize_client_performance(base_url: str) -> ORANMANOClient:
    """Create optimized client for better performance"""
    session = requests.Session()

    # Configure optimized adapter
    adapter = HTTPAdapter(
        pool_connections=20,
        pool_maxsize=20,
        pool_block=False
    )

    session.mount("http://", adapter)
    session.mount("https://", adapter)

    # Set reasonable timeouts
    session.timeout = (5, 30)  # (connect, read)

    # Configure keep-alive
    session.headers.update({
        "Connection": "keep-alive",
        "Keep-Alive": "timeout=30, max=100"
    })

    client = ORANMANOClient(base_url, username, password)
    client.session = session

    return client
```

This comprehensive developer guide provides practical examples and best practices for integrating with the O-RAN Intent-MANO API system. It covers everything from basic authentication to advanced monitoring and troubleshooting techniques, helping developers build robust and efficient applications.