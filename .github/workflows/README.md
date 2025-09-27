# GitHub Workflows

This directory contains the essential CI/CD workflows for the O-RAN Intent-MANO project.

## Active Workflows

### 1. **ci.yml** - Continuous Integration
Primary CI pipeline that runs on every push and pull request:
- Go linting with golangci-lint
- Unit tests for all components (orchestrator, ran-dms, cn-dms, tn)
- Integration tests
- Code coverage reporting
- Triggers: push, pull_request

### 2. **docker-build.yml** - Docker Image Build
Builds and validates Docker images for all components:
- Multi-stage Docker builds
- Image optimization and security scanning
- Registry push on main branch
- Triggers: push to main, pull_request, manual dispatch

### 3. **test.yml** - Comprehensive Testing
Runs comprehensive test suite:
- Unit tests with coverage
- Integration tests
- E2E tests (when available)
- Test result artifacts
- Triggers: push, pull_request, schedule (nightly)

### 4. **trivy-scan.yml** - Security Scanning
Security vulnerability scanning:
- Docker image scanning with Trivy
- Dependency vulnerability checks
- SARIF report upload to GitHub Security
- Triggers: push to main, pull_request, schedule (daily)

## Archived Workflows

The following workflows have been archived to `.github/workflows-backup/`:
- `build.yml` - Replaced by docker-build.yml
- `ci-quickfix.yml` - Temporary fix, no longer needed
- `enhanced-ci.yml`, `enhanced-ci-v2.yml` - Overly complex, replaced by ci.yml
- `security*.yml` (multiple) - Consolidated into trivy-scan.yml
- `deployment*.yml` (multiple) - Deployment handled separately
- `monitoring-alerting.yml` - Monitoring setup completed
- `performance-testing.yml` - Not currently active
- `validate-metrics.yml` - Integrated into test.yml
- `workflow-validation.yml` - Not needed

## Workflow Maintenance

To restore archived workflows if needed:
```bash
cp .github/workflows-backup/<workflow-name>.yml .github/workflows/
```

To permanently delete archived workflows:
```bash
rm -rf .github/workflows-backup/
```