# O-RAN Intent-MANO Web Dashboard

A comprehensive React-based monitoring dashboard for the O-RAN Intent-Based MANO (Management and Orchestration) system. This dashboard provides real-time monitoring, intent management, network slice visualization, and infrastructure oversight capabilities.

## 🚀 Features

### Core Functionality
- **Real-time Monitoring**: WebSocket-based live updates for system metrics and status
- **Intent Management**: Create, track, and manage network intents with full lifecycle support
- **Network Slice Monitoring**: Visualize and monitor network slice performance and status
- **Infrastructure Oversight**: Monitor Kubernetes clusters, nodes, and resource utilization
- **Experiment Management**: Design, execute, and analyze system experiments
- **OAuth2 Authentication**: Secure login with multiple provider support

### Technical Features
- **Modern React 18** with TypeScript for type safety
- **Responsive Design** with Tailwind CSS
- **Real-time Data** via WebSocket connections
- **Chart Visualizations** with Recharts library
- **State Management** with React Query for server state
- **Performance Optimized** with code splitting and lazy loading
- **Progressive Web App** capabilities
- **Docker Containerized** for easy deployment

## 📋 Requirements

- Node.js 18.0.0 or higher
- npm 9.0.0 or higher
- Docker (for containerized deployment)
- Access to O-RAN Intent-MANO orchestrator API
- Prometheus metrics endpoint (optional)

## 🛠️ Installation

### Local Development

1. **Clone and navigate to the dashboard directory:**
   ```bash
   cd observability/dashboard
   ```

2. **Install dependencies:**
   ```bash
   npm install
   ```

3. **Configure environment variables:**
   ```bash
   cp .env.example .env.local
   # Edit .env.local with your configuration
   ```

4. **Start development server:**
   ```bash
   npm run dev
   ```

5. **Open your browser:**
   ```
   http://localhost:3000
   ```

### Docker Deployment

1. **Build the Docker image:**
   ```bash
   docker build -t oran-dashboard:latest .
   ```

2. **Run with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

3. **Access the dashboard:**
   ```
   http://localhost:3000
   ```

## ⚙️ Configuration

### Environment Variables

Create a `.env.local` file with the following variables:

```bash
# API Configuration
VITE_API_BASE_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080
VITE_PROMETHEUS_URL=http://localhost:9090

# OAuth2 Configuration
VITE_OAUTH_CLIENT_ID=your-oauth-client-id
VITE_OAUTH_REDIRECT_URI=http://localhost:3000/auth/callback
VITE_OAUTH_ISSUER=https://your-oauth-provider.com

# Environment
VITE_ENVIRONMENT=development
VITE_APP_VERSION=1.0.0

# Feature Flags
VITE_ENABLE_EXPERIMENTS=true
VITE_ENABLE_REAL_TIME_METRICS=true
VITE_ENABLE_ADVANCED_CHARTS=true
VITE_ENABLE_NOTIFICATIONS=true
```

### Backend Integration

The dashboard expects the following API endpoints from the orchestrator:

#### Intent Management
- `GET /api/intents` - List intents with pagination
- `POST /api/intents` - Create new intent
- `GET /api/intents/:id` - Get intent details
- `PUT /api/intents/:id` - Update intent
- `DELETE /api/intents/:id` - Delete intent
- `POST /api/intents/:id/execute` - Execute intent

#### Network Slices
- `GET /api/slices` - List network slices
- `GET /api/slices/:id` - Get slice details
- `GET /api/slices/:id/metrics` - Get slice metrics

#### Infrastructure
- `GET /api/infrastructure` - Get infrastructure overview
- `GET /api/infrastructure/clusters` - List clusters
- `GET /api/infrastructure/nodes` - List nodes
- `GET /api/infrastructure/pods/:namespace/:pod/logs` - Get pod logs

#### Experiments
- `GET /api/experiments` - List experiments
- `POST /api/experiments` - Create experiment
- `POST /api/experiments/:id/start` - Start experiment
- `POST /api/experiments/:id/stop` - Stop experiment

#### System
- `GET /api/health` - Health check
- `GET /api/metrics/system` - System metrics
- `GET /api/alerts` - System alerts

### WebSocket Events

The dashboard subscribes to the following WebSocket events:

- `metrics_update` - Real-time system metrics
- `intent_update` - Intent status changes
- `slice_update` - Network slice updates
- `alert_update` - System alerts

## 🎨 User Interface

### Dashboard Overview
- System-wide metrics and KPIs
- Recent activity timeline
- Alert notifications
- Quick access to key functions

### Intent Management
- Intent creation wizard
- Lifecycle tracking
- Status monitoring
- Bulk operations

### Network Slices
- Slice topology visualization
- Performance metrics
- SLA monitoring
- Resource allocation

### Infrastructure
- Cluster health status
- Node resource utilization
- Pod management
- Log aggregation

### Experiments
- Experiment designer
- Execution monitoring
- Results analysis
- Historical data

## 🔧 Development

### Available Scripts

```bash
# Development
npm run dev              # Start development server
npm run type-check       # TypeScript type checking

# Building
npm run build           # Build for production
npm run preview         # Preview production build

# Testing
npm run test            # Run tests
npm run test:ui         # Run tests with UI
npm run test:coverage   # Run tests with coverage

# Linting and Formatting
npm run lint            # ESLint checking
npm run lint:fix        # Fix ESLint issues
npm run format          # Format with Prettier
npm run format:check    # Check Prettier formatting

# Docker
npm run docker:build   # Build Docker image
npm run docker:run     # Run Docker container
```

### Project Structure

```
src/
├── components/          # Reusable UI components
│   ├── ui/             # Basic UI components
│   ├── charts/         # Chart components
│   └── forms/          # Form components
├── pages/              # Page components
├── contexts/           # React contexts
├── hooks/              # Custom React hooks
├── services/           # API and external services
├── types/              # TypeScript type definitions
├── utils/              # Utility functions
└── assets/            # Static assets
```

### Adding New Features

1. **Create component in appropriate directory**
2. **Add TypeScript types in `/types`**
3. **Implement API service in `/services`**
4. **Add routing in `App.tsx`**
5. **Update navigation in `Sidebar.tsx`**
6. **Write tests**

### Styling Guidelines

- Use Tailwind CSS utility classes
- Follow the existing design system
- Ensure responsive design (mobile-first)
- Maintain accessibility standards
- Use consistent spacing and typography

## 📊 Monitoring Integration

### Prometheus Integration

The dashboard integrates with Prometheus for advanced metrics:

```typescript
import { prometheusService } from '@/services/prometheus'

// Get CPU usage
const cpuData = await prometheusService.getCPUUsage('orchestrator', '1h')

// Get custom metrics
const intentMetrics = await prometheusService.getIntentMetrics('24h')
```

### Custom Metrics

Define custom metrics for O-RAN specific monitoring:

- `oran_intents_active_total` - Number of active intents
- `oran_intents_success_rate` - Intent success rate
- `oran_network_slices_active_total` - Active network slices
- `oran_slice_throughput_bytes_total` - Slice throughput
- `oran_slice_latency_seconds` - Slice latency histogram

### Grafana Integration

The dashboard can embed Grafana panels:

```bash
VITE_GRAFANA_URL=http://localhost:3001
```

## 🔒 Security

### Authentication

- OAuth2/OIDC support
- JWT token management
- Automatic token refresh
- Role-based access control

### Network Security

- HTTPS in production
- CORS configuration
- Rate limiting
- Input validation
- XSS protection

### Configuration

- Secure environment variable handling
- Secret management
- Docker security best practices

## 🚀 Deployment

### Production Deployment

1. **Build the application:**
   ```bash
   npm run build
   ```

2. **Use the production Docker image:**
   ```bash
   docker build -t oran-dashboard:prod .
   docker run -p 3000:3000 oran-dashboard:prod
   ```

3. **With reverse proxy (recommended):**
   ```nginx
   server {
       listen 80;
       server_name dashboard.oran.local;

       location / {
           proxy_pass http://localhost:3000;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-dashboard
spec:
  replicas: 2
  selector:
    matchLabels:
      app: oran-dashboard
  template:
    metadata:
      labels:
        app: oran-dashboard
    spec:
      containers:
      - name: dashboard
        image: oran-dashboard:latest
        ports:
        - containerPort: 3000
        env:
        - name: VITE_API_BASE_URL
          value: "http://oran-orchestrator:8080/api"
---
apiVersion: v1
kind: Service
metadata:
  name: oran-dashboard-service
spec:
  selector:
    app: oran-dashboard
  ports:
  - port: 80
    targetPort: 3000
  type: LoadBalancer
```

## 🔧 Troubleshooting

### Common Issues

1. **Connection Failed**
   - Verify API_BASE_URL configuration
   - Check network connectivity
   - Validate CORS settings

2. **WebSocket Connection Drops**
   - Check WS_URL configuration
   - Verify proxy WebSocket support
   - Review network timeouts

3. **Build Failures**
   - Clear node_modules and reinstall
   - Check Node.js version compatibility
   - Verify environment variables

4. **Performance Issues**
   - Enable production build optimizations
   - Check bundle size analysis
   - Optimize image assets

### Debug Mode

Enable debug logging:

```bash
VITE_DEBUG=true npm run dev
```

### Health Checks

The application provides health endpoints:

- `/health` - Basic health check
- `/api/health` - Backend health check
- `/metrics` - Prometheus metrics

## 📝 Contributing

1. **Fork the repository**
2. **Create a feature branch**
3. **Make your changes**
4. **Add tests**
5. **Update documentation**
6. **Submit a pull request**

### Code Style

- Follow TypeScript best practices
- Use ESLint and Prettier configurations
- Write meaningful commit messages
- Include tests for new features

## 📄 License

This project is licensed under the Apache License 2.0. See the LICENSE file for details.

## 🤝 Support

For support and questions:

- Open an issue on GitHub
- Check the troubleshooting guide
- Review the API documentation
- Consult the deployment guides

## 🎯 Roadmap

### Version 1.1
- [ ] Advanced alerting rules
- [ ] Custom dashboard widgets
- [ ] Multi-tenancy support
- [ ] Enhanced experiment workflows

### Version 1.2
- [ ] AI-powered insights
- [ ] Predictive analytics
- [ ] Advanced automation rules
- [ ] Mobile application

---

**Built with ❤️ for the O-RAN community**