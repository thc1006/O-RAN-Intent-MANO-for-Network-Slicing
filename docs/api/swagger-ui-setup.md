# Swagger UI Setup for O-RAN Intent-MANO API

This guide provides instructions for setting up Swagger UI to provide interactive API documentation for the O-RAN Intent-MANO system.

## Table of Contents

1. [Docker Setup](#docker-setup)
2. [Nginx Setup](#nginx-setup)
3. [Node.js Setup](#nodejs-setup)
4. [Kubernetes Deployment](#kubernetes-deployment)
5. [Configuration Options](#configuration-options)
6. [Security Considerations](#security-considerations)
7. [Customization](#customization)

## Docker Setup

### Simple Docker Deployment

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  swagger-ui:
    image: swaggerapi/swagger-ui:latest
    ports:
      - "8080:8080"
    environment:
      - SWAGGER_JSON=/app/openapi.yaml
      - BASE_URL=/docs
    volumes:
      - ./docs/api/openapi.yaml:/app/openapi.yaml:ro
    restart: unless-stopped

  swagger-editor:
    image: swaggerapi/swagger-editor:latest
    ports:
      - "8081:8080"
    environment:
      - SWAGGER_FILE=/app/openapi.yaml
    volumes:
      - ./docs/api/openapi.yaml:/app/openapi.yaml:rw
    restart: unless-stopped
```

Start the services:

```bash
# From the project root directory
docker-compose up -d

# Access Swagger UI at: http://localhost:8080
# Access Swagger Editor at: http://localhost:8081
```

### Advanced Docker Setup with Authentication

Create `docker-compose.production.yml`:

```yaml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - ./docs/api:/usr/share/nginx/html/docs:ro
    depends_on:
      - swagger-ui
    restart: unless-stopped

  swagger-ui:
    image: swaggerapi/swagger-ui:latest
    expose:
      - "8080"
    environment:
      - SWAGGER_JSON=/app/openapi.yaml
      - BASE_URL=/docs/api
      - SUPPORTED_SUBMIT_METHODS=["get", "post", "put", "delete", "patch"]
      - OAUTH_CLIENT_ID=your-oauth-client-id
      - OAUTH_CLIENT_SECRET=your-oauth-client-secret
      - OAUTH_REALM=oran-mano
      - OAUTH_APP_NAME=O-RAN Intent-MANO API
      - OAUTH_SCOPES=read write
    volumes:
      - ./docs/api/openapi.yaml:/app/openapi.yaml:ro
      - ./swagger-ui-config.js:/usr/share/nginx/html/swagger-ui-config.js:ro
    restart: unless-stopped

  swagger-editor:
    image: swaggerapi/swagger-editor:latest
    expose:
      - "8080"
    environment:
      - SWAGGER_FILE=/app/openapi.yaml
    volumes:
      - ./docs/api/openapi.yaml:/app/openapi.yaml:rw
    restart: unless-stopped
```

Create `nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' https://api.oran-mano.io;" always;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_docs:10m rate=10r/s;

    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name docs.oran-mano.io;
        return 301 https://$server_name$request_uri;
    }

    # HTTPS server
    server {
        listen 443 ssl http2;
        server_name docs.oran-mano.io;

        # SSL configuration
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        # Basic authentication for editor
        location /docs/editor/ {
            auth_basic "O-RAN MANO API Editor";
            auth_basic_user_file /etc/nginx/.htpasswd;

            proxy_pass http://swagger-editor:8080/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Public API documentation
        location /docs/api/ {
            limit_req zone=api_docs burst=20 nodelay;

            proxy_pass http://swagger-ui:8080/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # Cache static assets
            location ~* \.(css|js|png|jpg|jpeg|gif|ico|svg)$ {
                expires 1y;
                add_header Cache-Control "public, immutable";
            }
        }

        # Serve static documentation files
        location /docs/ {
            alias /usr/share/nginx/html/docs/;
            index index.html;
            try_files $uri $uri/ =404;
        }

        # Health check
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

Create `.htpasswd` for editor authentication:

```bash
# Install htpasswd utility
sudo apt-get install apache2-utils

# Create password file
htpasswd -c .htpasswd admin

# Enter password when prompted
```

## Nginx Setup

For standalone Nginx deployment without Docker:

Create `/etc/nginx/sites-available/oran-mano-docs`:

```nginx
server {
    listen 80;
    server_name docs.oran-mano.io;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name docs.oran-mano.io;

    # SSL configuration
    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;

    # Document root
    root /var/www/oran-mano-docs;
    index index.html;

    # Serve Swagger UI
    location /docs/api/ {
        alias /usr/share/swagger-ui/;
        try_files $uri $uri/ @swagger-ui;
    }

    location @swagger-ui {
        # Fallback to index.html for SPA routing
        try_files /index.html =404;
    }

    # Serve OpenAPI spec
    location /docs/api/openapi.yaml {
        alias /var/www/oran-mano-docs/openapi.yaml;
        add_header Content-Type application/x-yaml;
        add_header Access-Control-Allow-Origin *;
    }

    # Serve other documentation
    location /docs/ {
        try_files $uri $uri/ =404;
    }
}
```

Install and configure Swagger UI:

```bash
# Download Swagger UI
wget https://github.com/swagger-api/swagger-ui/archive/v4.15.5.tar.gz
tar -xzf v4.15.5.tar.gz
sudo cp -r swagger-ui-4.15.5/dist/* /usr/share/swagger-ui/

# Create custom index.html
sudo tee /usr/share/swagger-ui/index.html > /dev/null << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>O-RAN Intent-MANO API Documentation</title>
    <link rel="stylesheet" type="text/css" href="./swagger-ui-bundle.css" />
    <link rel="stylesheet" type="text/css" href="./swagger-ui-standalone-preset.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            background-color: #1e3a8a;
        }
        .swagger-ui .topbar .download-url-wrapper .select-label {
            color: white;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="./swagger-ui-bundle.js"></script>
    <script src="./swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/docs/api/openapi.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log('Swagger UI loaded successfully');
                },
                requestInterceptor: function(request) {
                    // Add custom headers or modify requests
                    request.headers['X-API-Client'] = 'SwaggerUI';
                    return request;
                },
                responseInterceptor: function(response) {
                    // Handle responses
                    return response;
                }
            });
        };
    </script>
</body>
</html>
EOF

# Enable site
sudo ln -s /etc/nginx/sites-available/oran-mano-docs /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

## Node.js Setup

For a Node.js-based setup with customization:

Create `package.json`:

```json
{
  "name": "oran-mano-api-docs",
  "version": "1.0.0",
  "description": "O-RAN Intent-MANO API Documentation Server",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js",
    "build": "webpack --mode=production",
    "serve": "npm run build && npm start"
  },
  "dependencies": {
    "express": "^4.18.2",
    "swagger-ui-express": "^4.6.3",
    "yamljs": "^0.3.0",
    "helmet": "^7.0.0",
    "express-rate-limit": "^6.8.1",
    "cors": "^2.8.5",
    "compression": "^1.7.4"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}
```

Create `server.js`:

```javascript
const express = require('express');
const swaggerUi = require('swagger-ui-express');
const YAML = require('yamljs');
const helmet = require('helmet');
const rateLimit = require('express-rate-limit');
const cors = require('cors');
const compression = require('compression');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3000;

// Load OpenAPI specification
const swaggerDocument = YAML.load('./docs/api/openapi.yaml');

// Security middleware
app.use(helmet({
    contentSecurityPolicy: {
        directives: {
            defaultSrc: ["'self'"],
            scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
            styleSrc: ["'self'", "'unsafe-inline'"],
            imgSrc: ["'self'", "data:", "https:"],
            connectSrc: ["'self'", "https://api.oran-mano.io"]
        }
    }
}));

// Compression
app.use(compression());

// CORS
app.use(cors({
    origin: process.env.NODE_ENV === 'production'
        ? ['https://docs.oran-mano.io', 'https://api.oran-mano.io']
        : true
}));

// Rate limiting
const limiter = rateLimit({
    windowMs: 15 * 60 * 1000, // 15 minutes
    max: 100, // limit each IP to 100 requests per windowMs
    message: 'Too many requests from this IP, please try again later.'
});
app.use('/docs/', limiter);

// Custom Swagger UI options
const swaggerOptions = {
    explorer: true,
    swaggerOptions: {
        supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
        tryItOutEnabled: true,
        filter: true,
        displayRequestDuration: true,
        docExpansion: 'none',
        defaultModelsExpandDepth: 2,
        defaultModelExpandDepth: 2,
        showExtensions: true,
        showCommonExtensions: true,
        requestInterceptor: function(request) {
            // Add API key if available
            const apiKey = localStorage.getItem('oran-mano-api-key');
            if (apiKey) {
                request.headers['Authorization'] = `Bearer ${apiKey}`;
            }
            return request;
        }
    },
    customCss: `
        .swagger-ui .topbar {
            background-color: #1e3a8a;
        }
        .swagger-ui .info .title {
            color: #1e3a8a;
        }
        .swagger-ui .scheme-container {
            background: #f8f9fa;
            border: 1px solid #dee2e6;
        }
        .swagger-ui .btn.authorize {
            background-color: #28a745;
            border-color: #28a745;
        }
        .swagger-ui .btn.authorize:hover {
            background-color: #218838;
            border-color: #1e7e34;
        }
    `,
    customSiteTitle: "O-RAN Intent-MANO API Documentation",
    customfavIcon: "/favicon.ico"
};

// Custom authentication handler
app.get('/docs/auth-callback', (req, res) => {
    const { code, state } = req.query;

    // Handle OAuth callback
    res.send(`
        <script>
            if (window.opener) {
                window.opener.postMessage({
                    type: 'oauth-callback',
                    code: '${code}',
                    state: '${state}'
                }, '*');
                window.close();
            }
        </script>
    `);
});

// Serve static files
app.use('/docs/static', express.static(path.join(__dirname, 'docs')));

// API documentation endpoint
app.use('/docs/api', swaggerUi.serve, swaggerUi.setup(swaggerDocument, swaggerOptions));

// Health check
app.get('/health', (req, res) => {
    res.status(200).json({
        status: 'healthy',
        timestamp: new Date().toISOString(),
        version: process.env.npm_package_version || '1.0.0'
    });
});

// Serve additional documentation
app.get('/docs', (req, res) => {
    res.send(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>O-RAN Intent-MANO Documentation</title>
            <style>
                body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
                .nav { background: #1e3a8a; color: white; padding: 20px; margin: -20px -20px 20px -20px; }
                .nav h1 { margin: 0; }
                .card { border: 1px solid #ddd; border-radius: 8px; padding: 20px; margin: 20px 0; }
                .card h2 { color: #1e3a8a; margin-top: 0; }
                .btn { display: inline-block; background: #1e3a8a; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px; margin: 5px; }
                .btn:hover { background: #153e75; }
            </style>
        </head>
        <body>
            <div class="nav">
                <h1>O-RAN Intent-MANO Documentation</h1>
                <p>Comprehensive API documentation and developer resources</p>
            </div>

            <div class="card">
                <h2>🚀 API Documentation</h2>
                <p>Interactive API documentation with Swagger UI</p>
                <a href="/docs/api" class="btn">Open API Docs</a>
            </div>

            <div class="card">
                <h2>📚 Developer Guide</h2>
                <p>Complete guide for developers integrating with the O-RAN Intent-MANO API</p>
                <a href="/docs/static/developer-guide.md" class="btn">View Guide</a>
            </div>

            <div class="card">
                <h2>📋 API Reference</h2>
                <p>Detailed API reference documentation</p>
                <a href="/docs/static/api-reference.md" class="btn">View Reference</a>
            </div>

            <div class="card">
                <h2>📦 Postman Collection</h2>
                <p>Import our Postman collection for easy API testing</p>
                <a href="/docs/static/postman-collection.json" class="btn">Download Collection</a>
            </div>

            <div class="card">
                <h2>🔧 OpenAPI Specification</h2>
                <p>Raw OpenAPI 3.0 specification file</p>
                <a href="/docs/api/openapi.yaml" class="btn">Download YAML</a>
                <a href="/docs/api/openapi.json" class="btn">Download JSON</a>
            </div>
        </body>
        </html>
    `);
});

// Error handling
app.use((err, req, res, next) => {
    console.error(err.stack);
    res.status(500).json({
        error: 'Internal Server Error',
        message: process.env.NODE_ENV === 'development' ? err.message : 'Something went wrong!'
    });
});

// 404 handler
app.use((req, res) => {
    res.status(404).json({
        error: 'Not Found',
        message: 'The requested resource was not found'
    });
});

app.listen(PORT, () => {
    console.log(`📚 O-RAN Intent-MANO API Documentation server running on port ${PORT}`);
    console.log(`🌐 API Docs: http://localhost:${PORT}/docs/api`);
    console.log(`📖 Documentation Hub: http://localhost:${PORT}/docs`);
});

module.exports = app;
```

Install and run:

```bash
npm install
npm start

# For development with auto-reload
npm run dev
```

## Kubernetes Deployment

Create Kubernetes manifests for production deployment:

`k8s/namespace.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: oran-mano-docs
  labels:
    name: oran-mano-docs
```

`k8s/configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: swagger-ui-config
  namespace: oran-mano-docs
data:
  swagger-config.js: |
    window.ui = SwaggerUIBundle({
      url: '/api/openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset
      ],
      plugins: [
        SwaggerUIBundle.plugins.DownloadUrl
      ],
      layout: "StandaloneLayout",
      supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
      oauth2RedirectUrl: window.location.origin + '/docs/oauth2-redirect.html',
      requestInterceptor: function(request) {
        request.headers['X-API-Client'] = 'SwaggerUI-K8s';
        return request;
      }
    });

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: openapi-spec
  namespace: oran-mano-docs
data:
  openapi.yaml: |
    # Include your OpenAPI specification here
    # You can also mount this from a volume or fetch from a URL
```

`k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: swagger-ui
  namespace: oran-mano-docs
  labels:
    app: swagger-ui
spec:
  replicas: 2
  selector:
    matchLabels:
      app: swagger-ui
  template:
    metadata:
      labels:
        app: swagger-ui
    spec:
      containers:
      - name: swagger-ui
        image: swaggerapi/swagger-ui:v4.15.5
        ports:
        - containerPort: 8080
        env:
        - name: SWAGGER_JSON
          value: "/app/openapi.yaml"
        - name: BASE_URL
          value: "/docs"
        - name: SUPPORTED_SUBMIT_METHODS
          value: '["get", "post", "put", "delete", "patch"]'
        volumeMounts:
        - name: openapi-spec
          mountPath: /app/openapi.yaml
          subPath: openapi.yaml
        - name: swagger-config
          mountPath: /usr/share/nginx/html/swagger-config.js
          subPath: swagger-config.js
        resources:
          requests:
            memory: "64Mi"
            cpu: "100m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /docs
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /docs
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: openapi-spec
        configMap:
          name: openapi-spec
      - name: swagger-config
        configMap:
          name: swagger-ui-config
```

`k8s/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: swagger-ui-service
  namespace: oran-mano-docs
  labels:
    app: swagger-ui
spec:
  selector:
    app: swagger-ui
  ports:
  - name: http
    port: 80
    targetPort: 8080
    protocol: TCP
  type: ClusterIP
```

`k8s/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: swagger-ui-ingress
  namespace: oran-mano-docs
  annotations:
    kubernetes.io/ingress.class: "nginx"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
  - hosts:
    - docs.oran-mano.io
    secretName: swagger-ui-tls
  rules:
  - host: docs.oran-mano.io
    http:
      paths:
      - path: /docs
        pathType: Prefix
        backend:
          service:
            name: swagger-ui-service
            port:
              number: 80
```

Deploy to Kubernetes:

```bash
# Apply all manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n oran-mano-docs
kubectl get ingress -n oran-mano-docs

# View logs
kubectl logs -f deployment/swagger-ui -n oran-mano-docs
```

## Configuration Options

### Environment Variables

```bash
# Swagger UI Configuration
SWAGGER_JSON=/app/openapi.yaml          # Path to OpenAPI spec
BASE_URL=/docs                          # Base URL for Swagger UI
SUPPORTED_SUBMIT_METHODS=["get","post"] # Allowed HTTP methods
OAUTH_CLIENT_ID=your-client-id          # OAuth client ID
OAUTH_CLIENT_SECRET=your-secret         # OAuth client secret
OAUTH_REALM=oran-mano                   # OAuth realm
OAUTH_APP_NAME=O-RAN Intent-MANO API    # OAuth app name

# Server Configuration
PORT=3000                               # Server port
NODE_ENV=production                     # Environment
LOG_LEVEL=info                          # Logging level

# Security
HELMET_CSP_ENABLED=true                 # Enable CSP
RATE_LIMIT_MAX=100                      # Rate limit max requests
RATE_LIMIT_WINDOW=900000               # Rate limit window (15 min)
```

### Custom Swagger UI Configuration

Create `swagger-ui-config.js`:

```javascript
// Custom Swagger UI configuration
window.onload = function() {
    const ui = SwaggerUIBundle({
        url: '/docs/api/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIStandalonePreset
        ],
        plugins: [
            SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout",

        // Customize behavior
        supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
        tryItOutEnabled: true,
        filter: true,
        displayRequestDuration: true,
        showExtensions: true,
        showCommonExtensions: true,

        // UI customization
        docExpansion: 'list',
        defaultModelsExpandDepth: 1,
        defaultModelExpandDepth: 1,

        // Request/response interceptors
        requestInterceptor: function(request) {
            // Add authentication token from localStorage
            const token = localStorage.getItem('oran-mano-access-token');
            if (token) {
                request.headers['Authorization'] = `Bearer ${token}`;
            }

            // Add custom headers
            request.headers['X-API-Client'] = 'SwaggerUI';
            request.headers['X-Requested-With'] = 'SwaggerUI';

            console.log('Request:', request);
            return request;
        },

        responseInterceptor: function(response) {
            console.log('Response:', response);

            // Handle authentication errors
            if (response.status === 401) {
                localStorage.removeItem('oran-mano-access-token');
                alert('Authentication expired. Please log in again.');
            }

            return response;
        },

        // OAuth configuration
        oauth2RedirectUrl: window.location.origin + '/docs/oauth2-redirect.html',

        // Custom validators
        validatorUrl: null, // Disable online validator

        // Plugin configuration
        pluginOptions: {
            downloadUrl: {
                timeout: 30000
            }
        }
    });

    // Add custom authentication UI
    setTimeout(function() {
        addCustomAuthUI();
    }, 1000);
};

function addCustomAuthUI() {
    // Add login form to the page
    const authContainer = document.createElement('div');
    authContainer.id = 'custom-auth';
    authContainer.style.cssText = `
        position: fixed;
        top: 10px;
        right: 10px;
        background: white;
        border: 1px solid #ccc;
        border-radius: 4px;
        padding: 10px;
        box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        z-index: 1000;
    `;

    const token = localStorage.getItem('oran-mano-access-token');

    if (token) {
        authContainer.innerHTML = `
            <div>
                <span style="color: green;">✓ Authenticated</span>
                <button onclick="logout()" style="margin-left: 10px;">Logout</button>
            </div>
        `;
    } else {
        authContainer.innerHTML = `
            <div>
                <input type="text" id="username" placeholder="Username" style="margin: 2px;">
                <input type="password" id="password" placeholder="Password" style="margin: 2px;">
                <button onclick="login()" style="margin: 2px;">Login</button>
            </div>
        `;
    }

    document.body.appendChild(authContainer);
}

async function login() {
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;

    if (!username || !password) {
        alert('Please enter username and password');
        return;
    }

    try {
        const response = await fetch('/v1/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('oran-mano-access-token', data.access_token);
            location.reload(); // Reload to update UI
        } else {
            alert('Login failed');
        }
    } catch (error) {
        console.error('Login error:', error);
        alert('Login error: ' + error.message);
    }
}

function logout() {
    localStorage.removeItem('oran-mano-access-token');
    location.reload();
}
```

## Security Considerations

### Authentication Integration

1. **OAuth 2.0 / OpenID Connect**:
   ```javascript
   // Configure OAuth in Swagger UI
   const ui = SwaggerUIBundle({
       // ... other config
       oauth2RedirectUrl: window.location.origin + '/docs/oauth2-redirect.html',
       initOAuth: {
           clientId: 'your-client-id',
           realm: 'oran-mano',
           appName: 'O-RAN Intent-MANO API Docs',
           scopeSeparator: ' ',
           scopes: 'read write',
           additionalQueryStringParams: {},
           useBasicAuthenticationWithAccessCodeGrant: false,
           usePkceWithAuthorizationCodeGrant: true
       }
   });
   ```

2. **API Key Authentication**:
   ```javascript
   // Add API key to all requests
   requestInterceptor: function(request) {
       const apiKey = prompt('Enter your API key:');
       if (apiKey) {
           request.headers['X-API-Key'] = apiKey;
       }
       return request;
   }
   ```

### Content Security Policy

```javascript
// Express.js CSP configuration
app.use(helmet({
    contentSecurityPolicy: {
        directives: {
            defaultSrc: ["'self'"],
            scriptSrc: [
                "'self'",
                "'unsafe-inline'", // Required for Swagger UI
                "'unsafe-eval'"    // Required for Swagger UI
            ],
            styleSrc: ["'self'", "'unsafe-inline'"],
            imgSrc: ["'self'", "data:", "https:"],
            connectSrc: ["'self'", "https://api.oran-mano.io"],
            fontSrc: ["'self'"],
            objectSrc: ["'none'"],
            mediaSrc: ["'self'"],
            frameSrc: ["'none'"]
        }
    }
}));
```

### Rate Limiting

```javascript
// Advanced rate limiting
const rateLimit = require('express-rate-limit');

const createRateLimit = (windowMs, max, message) => rateLimit({
    windowMs,
    max,
    message: { error: message },
    standardHeaders: true,
    legacyHeaders: false,
    keyGenerator: (req) => {
        // Use IP + User-Agent for more accurate limiting
        return req.ip + req.get('User-Agent');
    }
});

// Different limits for different endpoints
app.use('/docs/api', createRateLimit(15 * 60 * 1000, 100, 'Too many API doc requests'));
app.use('/docs', createRateLimit(15 * 60 * 1000, 50, 'Too many doc requests'));
```

## Customization

### Custom Theme

Create `custom-theme.css`:

```css
/* O-RAN Intent-MANO Custom Theme */
:root {
    --oran-primary: #1e3a8a;
    --oran-secondary: #3b82f6;
    --oran-accent: #10b981;
    --oran-background: #f8fafc;
    --oran-surface: #ffffff;
    --oran-text: #1f2937;
    --oran-text-secondary: #6b7280;
}

/* Header customization */
.swagger-ui .topbar {
    background: linear-gradient(135deg, var(--oran-primary), var(--oran-secondary));
    border-bottom: 3px solid var(--oran-accent);
}

.swagger-ui .topbar .download-url-wrapper {
    display: none; /* Hide URL input */
}

/* Info section */
.swagger-ui .info {
    margin: 30px 0;
}

.swagger-ui .info .title {
    color: var(--oran-primary);
    font-size: 36px;
    font-weight: 700;
    margin-bottom: 10px;
}

.swagger-ui .info .description {
    color: var(--oran-text-secondary);
    font-size: 16px;
    line-height: 1.6;
}

/* Operation customization */
.swagger-ui .opblock {
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    margin-bottom: 16px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.swagger-ui .opblock.opblock-get .opblock-summary {
    background: rgba(16, 185, 129, 0.1);
    border-color: var(--oran-accent);
}

.swagger-ui .opblock.opblock-post .opblock-summary {
    background: rgba(59, 130, 246, 0.1);
    border-color: var(--oran-secondary);
}

.swagger-ui .opblock.opblock-put .opblock-summary {
    background: rgba(245, 158, 11, 0.1);
    border-color: #f59e0b;
}

.swagger-ui .opblock.opblock-delete .opblock-summary {
    background: rgba(239, 68, 68, 0.1);
    border-color: #ef4444;
}

/* Button customization */
.swagger-ui .btn {
    border-radius: 6px;
    font-weight: 500;
    transition: all 0.2s ease;
}

.swagger-ui .btn.authorize {
    background-color: var(--oran-accent);
    border-color: var(--oran-accent);
}

.swagger-ui .btn.authorize:hover {
    background-color: #059669;
    border-color: #059669;
    transform: translateY(-1px);
}

.swagger-ui .btn.execute {
    background-color: var(--oran-primary);
    border-color: var(--oran-primary);
}

.swagger-ui .btn.execute:hover {
    background-color: #1e40af;
    border-color: #1e40af;
    transform: translateY(-1px);
}

/* Response customization */
.swagger-ui .responses-inner h4,
.swagger-ui .responses-inner h5 {
    color: var(--oran-primary);
}

.swagger-ui .response .response-col_status {
    color: var(--oran-accent);
    font-weight: 600;
}

/* Schema customization */
.swagger-ui .model-box {
    background: var(--oran-background);
    border: 1px solid #e5e7eb;
    border-radius: 6px;
}

.swagger-ui .model .model-title {
    color: var(--oran-primary);
    font-weight: 600;
}

/* Custom footer */
.swagger-ui::after {
    content: '';
    display: block;
    height: 60px;
    background: linear-gradient(135deg, var(--oran-primary), var(--oran-secondary));
    margin-top: 40px;
    position: relative;
}

.swagger-ui::after::before {
    content: 'O-RAN Intent-MANO API Documentation';
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: white;
    font-weight: 500;
}

/* Loading animation */
.swagger-ui .loading-container {
    background: var(--oran-background);
}

.swagger-ui .loading-container::before {
    content: '⚡ Loading O-RAN Intent-MANO API Documentation...';
    color: var(--oran-primary);
    font-weight: 500;
}

/* Mobile responsiveness */
@media (max-width: 768px) {
    .swagger-ui .info .title {
        font-size: 28px;
    }

    .swagger-ui .topbar {
        padding: 10px 20px;
    }

    .swagger-ui .wrapper {
        padding: 0 10px;
    }
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
    :root {
        --oran-background: #111827;
        --oran-surface: #1f2937;
        --oran-text: #f9fafb;
        --oran-text-secondary: #d1d5db;
    }

    .swagger-ui .scheme-container {
        background: var(--oran-surface);
        border-color: #374151;
    }

    .swagger-ui .opblock {
        background: var(--oran-surface);
        border-color: #374151;
    }
}
```

### Custom Logo and Branding

Add logo and branding elements:

```html
<!-- Add to index.html -->
<div class="custom-header" style="
    background: linear-gradient(135deg, #1e3a8a, #3b82f6);
    color: white;
    padding: 20px;
    text-align: center;
    margin-bottom: 20px;
">
    <img src="/docs/static/logo.png" alt="O-RAN Logo" style="height: 60px; vertical-align: middle; margin-right: 20px;">
    <h1 style="display: inline-block; margin: 0; vertical-align: middle;">
        O-RAN Intent-MANO API Documentation
    </h1>
    <p style="margin: 10px 0 0 0; opacity: 0.9;">
        Comprehensive API for Intent-based Management and Network Orchestration
    </p>
</div>
```

This comprehensive setup guide provides multiple deployment options for Swagger UI with the O-RAN Intent-MANO API documentation, from simple Docker deployments to production-ready Kubernetes clusters with security, customization, and monitoring capabilities.