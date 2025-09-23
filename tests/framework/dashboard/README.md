# O-RAN Intent-MANO Test Dashboard

A comprehensive test reporting and analytics dashboard for the O-RAN Intent-based MANO system.

## Features

### 📊 Comprehensive Test Analytics
- **Test Results**: Unit, integration, E2E, and performance test results
- **Code Coverage**: Multi-language coverage analysis (Go, Python)
- **Quality Gates**: Automated quality threshold validation
- **Security Scanning**: Vulnerability analysis and compliance reporting
- **Thesis Validation**: Performance target validation for research metrics

### 🎯 Quality Monitoring
- **Real-time Metrics**: Live dashboard with WebSocket updates
- **Historical Trends**: Test performance over time
- **Alert System**: Threshold-based notifications
- **Regression Detection**: Performance degradation alerts

### 🚀 CI/CD Integration
- **GitHub Actions**: Automated dashboard generation
- **GitHub Pages**: Hosted dashboard publishing
- **Pull Request Reports**: Automated PR commenting
- **Artifact Management**: Test report archiving

## Quick Start

### Running the Dashboard Locally

```bash
# Build the dashboard tool
cd tests/framework/dashboard/cmd/dashboard
go build -o dashboard-tool .

# Generate static dashboard
./dashboard-tool --output=dashboard.html

# Start dashboard server
./dashboard-tool --serve --port=8080
```

### Dashboard Server

```bash
# Start with custom configuration
./dashboard-tool --serve --config=config.yaml --port=8080

# Run metrics aggregation only
./dashboard-tool --aggregate-only --config=config.yaml

# Generate static report
./dashboard-tool --output=reports/dashboard.html
```

### Configuration

The dashboard is configured via `config.yaml`:

```yaml
# Basic dashboard settings
dashboard:
  title: "O-RAN Intent-MANO Test Dashboard"
  refresh_rate: 30
  port: 8080

# Quality thresholds
thresholds:
  coverage.overall: 90.0
  test.success_rate: 95.0
  security.critical_vulns: 0
  performance.deployment_time: 10.0

# Data sources
data_sources:
  - name: "unit-tests"
    type: "junit"
    path: "reports/unit-tests.xml"
```

## Features

### Test Results Dashboard
- **Test Suite Summary**: Pass/fail rates and execution times
- **Coverage Analysis**: Code coverage metrics with package breakdown
- **Trend Analysis**: Historical test performance
- **Failure Analysis**: Detailed failure reporting

### Performance Monitoring
- **Thesis Validation**: Automated validation of research performance targets
- **Deployment Time**: E2E deployment time tracking
- **Throughput Analysis**: Network slice throughput validation
- **Latency Monitoring**: Real-time latency measurements

### Security Dashboard
- **Vulnerability Scanning**: Multi-tool security analysis
- **License Compliance**: Open source license monitoring
- **Static Analysis**: Code quality and security issue detection
- **Dependency Scanning**: Third-party package vulnerability tracking

### Quality Gates
- **Automated Thresholds**: Configurable quality criteria
- **Pass/Fail Decisions**: Clear quality gate status
- **Emergency Override**: Manual quality gate bypass capability
- **Historical Tracking**: Quality score trends over time

## API Endpoints

### REST API
```
GET  /api/metrics      - Current test metrics
GET  /api/alerts       - Active alerts
GET  /api/history      - Historical metrics
POST /api/refresh      - Refresh metrics
GET  /api/config       - Dashboard configuration
```

### WebSocket API
```
/ws                    - Real-time metrics updates
```

### Export Endpoints
```
GET /export/json       - Export metrics as JSON
GET /export/csv        - Export metrics as CSV
GET /export/pdf        - Export dashboard as PDF (future)
```

## Thesis Integration

The dashboard includes specific validation for thesis research targets:

### Network Slice Performance Targets
- **URLLC**: Throughput ≥4.57 Mbps, Latency ≤6.3 ms
- **eMBB**: Throughput ≥2.77 Mbps, Latency ≤15.7 ms
- **mMTC**: Throughput ≥0.93 Mbps, Latency ≤16.1 ms

### Deployment Performance
- **Target**: E2E deployment time <10 minutes
- **Monitoring**: Real-time deployment tracking
- **Validation**: Automated pass/fail assessment

## CI/CD Integration

### GitHub Actions Workflow
The dashboard is automatically generated in the CI/CD pipeline:

1. **Test Execution**: All test suites run in parallel
2. **Report Collection**: Test results and coverage data collected
3. **Dashboard Generation**: Static dashboard created
4. **GitHub Pages**: Dashboard published to Pages
5. **PR Comments**: Dashboard links added to pull requests

### Deployment
- **Main Branch**: Dashboard deployed to GitHub Pages
- **Pull Requests**: Dashboard artifacts attached
- **Nightly Builds**: Comprehensive dashboard generation

## Development

### Building
```bash
cd tests/framework/dashboard
go mod tidy
go build ./cmd/dashboard
```

### Testing
```bash
go test ./...
```

### Local Development
```bash
# Start development server with hot reload
./dashboard-tool --serve --debug --config=config.yaml
```

## Architecture

### Components
- **Dashboard**: HTML/CSS/JS frontend with real-time updates
- **Metrics Aggregator**: Collects and processes test data
- **Reporter**: Generates various report formats
- **WebSocket Server**: Real-time data streaming
- **Quality Gates**: Threshold validation and alerting

### Data Flow
1. Test tools generate reports (JUnit XML, coverage, JSON)
2. Metrics aggregator collects and processes reports
3. Dashboard generates visualizations
4. WebSocket server streams real-time updates
5. Quality gates validate thresholds and generate alerts

## Configuration Reference

### Dashboard Settings
- `title`: Dashboard title
- `refresh_rate`: Auto-refresh interval (seconds)
- `port`: HTTP server port

### Quality Thresholds
- `coverage.overall`: Minimum overall code coverage
- `test.success_rate`: Minimum test success rate
- `security.critical_vulns`: Maximum critical vulnerabilities
- `performance.deployment_time`: Maximum deployment time

### Data Sources
- `name`: Source identifier
- `type`: Data type (junit, coverage, performance, security)
- `path`: File path to data source
- `format`: File format (xml, json, text)

## License

This dashboard is part of the O-RAN Intent-MANO project and follows the same licensing terms.