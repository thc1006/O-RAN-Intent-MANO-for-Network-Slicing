# O-RAN Intent-MANO Dashboard Installation Guide

This guide provides step-by-step instructions for installing and configuring the O-RAN Intent-MANO web monitoring dashboard.

## 🚀 Quick Start

### Local Development

```bash
cd observability/dashboard
npm install
cp .env.example .env.local
npm run dev
```

### Docker Deployment

```bash
cd observability/dashboard
docker-compose up -d
```

### Kubernetes Deployment

```bash
cd observability/dashboard
./scripts/deploy.sh deploy
```

## 📋 Prerequisites

### System Requirements

- **Operating System**: Linux, macOS, or Windows with WSL2
- **Node.js**: Version 18.0.0 or higher
- **npm**: Version 9.0.0 or higher
- **Docker**: Version 20.10 or higher (for containerized deployment)
- **Kubernetes**: Version 1.25 or higher (for K8s deployment)

### Dependencies

- **O-RAN Intent-MANO Orchestrator**: Running API server
- **Prometheus**: For metrics collection (optional)
- **OAuth2 Provider**: For authentication (optional)

## 🔧 Installation Methods

### Method 1: Local Development

1. **Navigate to dashboard directory:**
   ```bash
   cd observability/dashboard
   ```

2. **Install Node.js dependencies:**
   ```bash
   npm install
   ```

3. **Configure environment:**
   ```bash
   cp .env.example .env.local
   ```

4. **Edit configuration (required):**
   ```bash
   nano .env.local
   ```

   Update the following variables:
   ```env
   VITE_API_BASE_URL=http://localhost:8080/api
   VITE_WS_URL=ws://localhost:8080
   VITE_PROMETHEUS_URL=http://localhost:9090
   ```

5. **Start development server:**
   ```bash
   npm run dev
   ```

6. **Access dashboard:**
   Open http://localhost:3000 in your browser

### Method 2: Production Build

1. **Follow steps 1-4 from Method 1**

2. **Build for production:**
   ```bash
   npm run build
   ```

3. **Serve with a web server:**
   ```bash
   # Using serve
   npx serve -s dist -l 3000

   # Using nginx
   sudo cp -r dist/* /var/www/html/
   sudo systemctl restart nginx
   ```

### Method 3: Docker Deployment

1. **Navigate to dashboard directory:**
   ```bash
   cd observability/dashboard
   ```

2. **Create environment file:**
   ```bash
   cp .env.example .env.production
   ```

3. **Configure for your environment:**
   ```bash
   nano .env.production
   ```

4. **Build and run with Docker:**
   ```bash
   # Build image
   docker build -t oran-dashboard:latest .

   # Run container
   docker run -d \
     --name oran-dashboard \
     --env-file .env.production \
     -p 3000:3000 \
     oran-dashboard:latest
   ```

5. **Or use Docker Compose:**
   ```bash
   docker-compose up -d
   ```

### Method 4: Kubernetes Deployment

1. **Prepare the environment:**
   ```bash
   cd observability/dashboard
   ```

2. **Review Kubernetes manifests:**
   ```bash
   ls kubernetes/
   # deployment.yaml contains all necessary resources
   ```

3. **Update configuration:**
   Edit `kubernetes/deployment.yaml` and update:
   - API endpoints
   - OAuth configuration
   - Domain names
   - Resource limits

4. **Deploy using the script:**
   ```bash
   ./scripts/deploy.sh deploy
   ```

5. **Or deploy manually:**
   ```bash
   # Create namespace
   kubectl create namespace oran-system

   # Apply manifests
   kubectl apply -f kubernetes/ -n oran-system

   # Check deployment status
   kubectl get pods -n oran-system
   ```

## ⚙️ Configuration

### Environment Variables

#### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `VITE_API_BASE_URL` | Orchestrator API endpoint | `http://localhost:8080/api` |
| `VITE_WS_URL` | WebSocket endpoint | `ws://localhost:8080` |

#### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_PROMETHEUS_URL` | Prometheus server URL | `http://localhost:9090` |
| `VITE_OAUTH_CLIENT_ID` | OAuth2 client ID | - |
| `VITE_OAUTH_REDIRECT_URI` | OAuth2 redirect URI | - |
| `VITE_OAUTH_ISSUER` | OAuth2 issuer URL | - |
| `VITE_GRAFANA_URL` | Grafana dashboard URL | - |

#### Feature Flags

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_ENABLE_EXPERIMENTS` | Enable experiments module | `true` |
| `VITE_ENABLE_REAL_TIME_METRICS` | Enable real-time metrics | `true` |
| `VITE_ENABLE_ADVANCED_CHARTS` | Enable advanced charts | `true` |
| `VITE_ENABLE_NOTIFICATIONS` | Enable notifications | `true` |

### Backend Integration

The dashboard requires the following API endpoints to be available:

#### Authentication
- `POST /api/auth/login` - User login
- `POST /api/auth/logout` - User logout
- `POST /api/auth/refresh` - Token refresh
- `GET /api/auth/me` - Get user info

#### Intent Management
- `GET /api/intents` - List intents
- `POST /api/intents` - Create intent
- `GET /api/intents/:id` - Get intent details
- `PUT /api/intents/:id` - Update intent
- `DELETE /api/intents/:id` - Delete intent
- `POST /api/intents/:id/execute` - Execute intent

#### Network Slices
- `GET /api/slices` - List network slices
- `GET /api/slices/:id` - Get slice details
- `GET /api/slices/:id/metrics` - Get slice metrics

#### Infrastructure
- `GET /api/infrastructure` - Infrastructure overview
- `GET /api/infrastructure/clusters` - Cluster list
- `GET /api/infrastructure/nodes` - Node list

#### System
- `GET /api/health` - Health check
- `GET /api/metrics/system` - System metrics
- `GET /api/alerts` - System alerts

### WebSocket Events

Configure the backend to emit these events:

- `metrics_update` - Real-time metrics
- `intent_update` - Intent status changes
- `slice_update` - Network slice updates
- `alert_update` - System alerts

## 🔐 Security Configuration

### HTTPS Setup

1. **For production, use HTTPS:**
   ```nginx
   server {
       listen 443 ssl http2;
       server_name dashboard.oran.local;

       ssl_certificate /path/to/certificate.crt;
       ssl_certificate_key /path/to/private.key;

       location / {
           proxy_pass http://localhost:3000;
       }
   }
   ```

### OAuth2 Setup

1. **Configure OAuth2 provider**
2. **Update environment variables:**
   ```env
   VITE_OAUTH_CLIENT_ID=your-client-id
   VITE_OAUTH_REDIRECT_URI=https://dashboard.oran.local/auth/callback
   VITE_OAUTH_ISSUER=https://your-provider.com
   ```

### CORS Configuration

Ensure your backend allows requests from the dashboard origin:
```javascript
// Express.js example
app.use(cors({
  origin: ['http://localhost:3000', 'https://dashboard.oran.local'],
  credentials: true
}))
```

## 🧪 Testing the Installation

### Health Checks

1. **Application health:**
   ```bash
   curl http://localhost:3000/health
   ```

2. **API connectivity:**
   ```bash
   curl http://localhost:8080/api/health
   ```

3. **WebSocket connection:**
   Check browser dev tools for WebSocket connection

### Functional Testing

1. **Access the dashboard**
2. **Test authentication**
3. **Verify real-time updates**
4. **Check all navigation links**
5. **Test intent creation/management**

## 🐛 Troubleshooting

### Common Issues

#### 1. Build Failures

**Problem:** npm install or build fails
```bash
# Solution: Clear cache and reinstall
rm -rf node_modules package-lock.json
npm cache clean --force
npm install
```

#### 2. API Connection Issues

**Problem:** Dashboard can't connect to backend
```bash
# Check API endpoint
curl -v http://localhost:8080/api/health

# Check CORS configuration
# Check firewall settings
# Verify environment variables
```

#### 3. WebSocket Connection Drops

**Problem:** Real-time updates don't work
```bash
# Check WebSocket endpoint
# Verify proxy configuration supports WebSocket upgrades
# Check network timeouts
```

#### 4. Authentication Issues

**Problem:** Login doesn't work
```bash
# Verify OAuth2 configuration
# Check redirect URIs
# Verify client credentials
```

### Debug Mode

Enable debug logging:
```bash
VITE_DEBUG=true npm run dev
```

### Log Analysis

Check application logs:
```bash
# Docker logs
docker logs oran-dashboard

# Kubernetes logs
kubectl logs -f deployment/oran-dashboard -n oran-system

# Browser console logs
# Check Network tab in browser dev tools
```

## 📊 Performance Optimization

### Production Optimizations

1. **Enable gzip compression**
2. **Configure CDN for static assets**
3. **Set up proper caching headers**
4. **Use HTTP/2**
5. **Optimize images and bundle size**

### Monitoring

1. **Set up application monitoring**
2. **Configure error tracking**
3. **Monitor WebSocket connections**
4. **Track API response times**

## 🔄 Updates and Maintenance

### Updating the Dashboard

1. **Pull latest changes:**
   ```bash
   git pull origin main
   ```

2. **Update dependencies:**
   ```bash
   npm update
   ```

3. **Rebuild and redeploy:**
   ```bash
   npm run build
   ./scripts/deploy.sh deploy
   ```

### Backup and Recovery

1. **Backup configuration files**
2. **Document custom modifications**
3. **Test restore procedures**

## 🆘 Support

### Getting Help

1. **Check this documentation**
2. **Review troubleshooting section**
3. **Check application logs**
4. **Open GitHub issue with details**

### Information to Include

- Dashboard version
- Operating system
- Node.js version
- Error messages
- Configuration (sanitized)
- Steps to reproduce

---

**For additional support, please refer to the main README.md file or open an issue on GitHub.**