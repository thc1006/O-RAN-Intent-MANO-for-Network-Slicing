# CI/CD Operational Runbooks

## 🚨 Emergency Response Procedures

### Critical Pipeline Failure
**Symptoms**: Multiple workflow failures, critical alerts, system unavailable

**Immediate Response (< 5 minutes)**:
1. Check GitHub Actions status page: https://www.githubstatus.com/
2. Verify repository access and permissions
3. Check for recent infrastructure changes
4. Review last successful run artifacts

**Investigation Steps**:
```bash
# Check recent workflow runs
gh run list --limit 20 --json conclusion,status,name

# Download failure logs
gh run download <failed-run-id>

# Check system resources
kubectl top nodes  # If using self-hosted runners
```

**Resolution Actions**:
- **If GitHub Actions issue**: Wait for service restoration, communicate to team
- **If code issue**: Revert last commit, create hotfix PR
- **If infrastructure issue**: Scale resources, check cluster health
- **If dependency issue**: Pin versions, update lock files

**Communication Template**:
```
🚨 CI/CD System Alert - Production Impact

Issue: [Brief description]
Impact: [Affected services/deployments]
ETA: [Expected resolution time]
Actions: [Current resolution steps]

Updates will be provided every 30 minutes.
```

### Security Vulnerability Detection
**Symptoms**: Critical security alerts, vulnerability scanners failing

**Immediate Response (< 15 minutes)**:
1. Assess vulnerability severity and scope
2. Check if vulnerability is actively exploited
3. Verify current deployment status
4. Implement temporary mitigations if needed

**Assessment Commands**:
```bash
# Check security alerts
gh api repos/:owner/:repo/security-advisories

# Review vulnerability details
gh api repos/:owner/:repo/dependabot/alerts

# Check affected components
grep -r "vulnerable-package" . --include="*.mod" --include="*.txt"
```

**Mitigation Strategies**:
- **Critical (CVSS 9.0+)**: Immediate deployment halt, emergency patching
- **High (CVSS 7.0-8.9)**: Expedited patching within 24 hours
- **Medium (CVSS 4.0-6.9)**: Standard patching cycle within 1 week

### Performance Degradation
**Symptoms**: Tests timing out, deployment times > 8 minutes, throughput below targets

**Quick Diagnostics**:
```bash
# Check resource utilization
kubectl top pods -n oran-mano --sort-by=cpu
kubectl top pods -n oran-mano --sort-by=memory

# Review performance metrics
curl -s http://prometheus:9090/api/v1/query?query=rate(http_requests_total[5m])

# Check node health
kubectl describe nodes | grep -A 5 "Conditions:"
```

**Performance Baseline Verification**:
```yaml
Expected Targets:
  - URLLC Latency: ≤ 6.3ms
  - eMBB Latency: ≤ 15.7ms
  - mMTC Latency: ≤ 16.1ms
  - Deployment Time: ≤ 480s
  - Test Success Rate: ≥ 95%
```

## 🔄 Standard Operating Procedures

### Daily Health Check
**Schedule**: Every morning at 9 AM local time
**Duration**: 15 minutes

**Checklist**:
- [ ] Review overnight workflow runs
- [ ] Check security alert status
- [ ] Verify backup and artifact retention
- [ ] Review performance metrics trends
- [ ] Validate monitoring system health

**Commands**:
```bash
# Get overnight run summary
gh run list --created "yesterday" --json conclusion,name,status

# Check security status
gh api repos/:owner/:repo/dependabot/alerts | jq '[.[] | select(.state == "open")] | length'

# Verify artifact storage
gh api repos/:owner/:repo/actions/artifacts | jq '.total_count'

# Performance trend check
curl -s "http://grafana:3000/api/dashboards/uid/cicd-perf" | jq '.dashboard.title'
```

### Weekly Performance Review
**Schedule**: Every Monday at 10 AM
**Duration**: 30 minutes

**Review Items**:
1. **Success Rate Analysis**: Target ≥95%
2. **Performance Trends**: Week-over-week comparison
3. **Security Posture**: Vulnerability aging report
4. **Resource Utilization**: Cost and efficiency metrics
5. **Action Items**: Follow-up on previous week's issues

**Report Template**:
```markdown
# Weekly CI/CD Performance Report - Week of [Date]

## Key Metrics
- Workflow Success Rate: [X]% (Target: ≥95%)
- Average Build Time: [X] minutes (Target: ≤8 minutes)
- Security Alerts: [X] critical, [X] high (Target: 0 critical)
- Test Coverage: [X]% (Target: ≥85%)

## Trends
- [Trend 1]: [Description and impact]
- [Trend 2]: [Description and impact]

## Action Items
- [ ] [Action item 1 with owner and due date]
- [ ] [Action item 2 with owner and due date]

## Recommendations
- [Recommendation 1]
- [Recommendation 2]
```

### Monthly Security Review
**Schedule**: First Wednesday of each month
**Duration**: 60 minutes

**Review Areas**:
1. **Vulnerability Management**: Patching cadence and effectiveness
2. **Access Control**: User permissions and authentication methods
3. **Supply Chain**: Dependency security and provenance
4. **Compliance**: Industry standards and regulatory requirements
5. **Incident Response**: Review of security incidents and improvements

**Security Assessment**:
```bash
# Generate security report
./scripts/security-assessment.sh

# Review dependency freshness
go list -u -m all | grep -v "indirect"
pip list --outdated

# Check for secrets exposure
git log --grep="password\|secret\|token" --oneline

# Validate branch protection
gh api repos/:owner/:repo/branches/main/protection
```

## 🛠️ Maintenance Procedures

### Dependency Updates
**Frequency**: Weekly for minor updates, immediately for security patches

**Go Dependencies**:
```bash
# Check for updates
go list -u -m all

# Update dependencies
go get -u ./...
go mod tidy

# Test after updates
go test ./...
```

**Python Dependencies**:
```bash
# Check for updates
pip list --outdated

# Update dependencies
pip-review --auto

# Update requirements
pip freeze > requirements.txt

# Test after updates
pytest tests/
```

**Container Base Images**:
```bash
# Check for updates
docker pull nginx:latest
docker pull golang:1.24-alpine

# Update Dockerfiles
sed -i 's/golang:1.24.6/golang:1.24.7/' Dockerfile

# Rebuild and test
docker build -t test-image .
docker run test-image go version
```

### Cache Management
**Frequency**: Weekly maintenance, immediate when issues arise

**GitHub Actions Cache**:
```bash
# List cache entries
gh cache list

# Delete old caches
gh cache delete <cache-key>

# Clear all caches (emergency)
gh cache delete --all
```

**Docker Cache**:
```bash
# Check cache usage
docker system df

# Clean build cache
docker builder prune -f

# Clean all unused data
docker system prune -a -f
```

### Performance Optimization
**Frequency**: Monthly review, immediate for degradation

**Build Time Optimization**:
```yaml
# Parallel job analysis
jobs:
  analyze-parallel:
    strategy:
      matrix:
        component: [orchestrator, vnf-operator, o2-client]
      max-parallel: 3  # Optimize based on runner capacity
```

**Resource Optimization**:
```bash
# Analyze resource usage
kubectl top pods -n oran-mano --sort-by=memory

# Optimize resource requests/limits
kubectl patch deployment orchestrator -p '{"spec":{"template":{"spec":{"containers":[{"name":"orchestrator","resources":{"requests":{"memory":"256Mi","cpu":"250m"}}}]}}}}'

# Monitor impact
kubectl get pods -n oran-mano -o custom-columns="NAME:.metadata.name,CPU:.spec.containers[*].resources.requests.cpu,MEMORY:.spec.containers[*].resources.requests.memory"
```

## 🐛 Troubleshooting Guides

### Test Failures
**Unit Test Failures**:
```bash
# Run specific test with verbose output
go test -v -run TestSpecificFunction ./pkg/component

# Check test coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Debug race conditions
go test -race -count=100 ./...
```

**Integration Test Failures**:
```bash
# Check cluster status
kubectl cluster-info
kubectl get nodes

# Verify component deployments
kubectl get deployments -n oran-mano
kubectl describe pod <failing-pod> -n oran-mano

# Check logs
kubectl logs -f deployment/orchestrator -n oran-mano
```

**Performance Test Failures**:
```bash
# Check system resources during tests
top -p $(pgrep k6)
iostat -x 1

# Verify network connectivity
iperf3 -c <target-host> -t 30

# Review test configuration
cat performance-test-config.json | jq '.thresholds'
```

### Build Failures
**Docker Build Issues**:
```bash
# Debug multi-stage builds
docker build --target=<stage-name> .

# Check layer caching
docker history <image-name>

# Analyze build context
docker build --no-cache --progress=plain .
```

**Dependency Issues**:
```bash
# Go module issues
go mod download
go mod verify
go clean -modcache

# Python dependency conflicts
pip-compile --upgrade requirements.in
pip-sync requirements.txt
```

### Deployment Failures
**Kubernetes Deployment Issues**:
```bash
# Check deployment status
kubectl rollout status deployment/<name> -n <namespace>

# View events
kubectl get events -n <namespace> --sort-by='.lastTimestamp'

# Debug pod issues
kubectl describe pod <pod-name> -n <namespace>
kubectl logs <pod-name> -n <namespace> --previous
```

**Health Check Failures**:
```bash
# Test health endpoints manually
kubectl port-forward service/<service-name> 8080:8080 -n <namespace>
curl -v http://localhost:8080/health

# Check readiness/liveness probes
kubectl get pods -n <namespace> -o custom-columns="NAME:.metadata.name,READY:.status.conditions[?(@.type=='Ready')].status,RESTART:.status.containerStatuses[0].restartCount"
```

## 🔐 Security Response Procedures

### Vulnerability Response
**Critical Vulnerability (CVSS 9.0+)**:
1. **Immediate containment** (< 1 hour)
2. **Impact assessment** (< 2 hours)
3. **Emergency patching** (< 4 hours)
4. **Verification testing** (< 6 hours)
5. **Communication** to stakeholders

**Response Commands**:
```bash
# Quick vulnerability assessment
grype . --only-fixed

# Emergency patching
go get -u <vulnerable-package>@<fixed-version>
docker build --no-cache -t emergency-fix .

# Rapid deployment
kubectl set image deployment/<name> <container>=<new-image> -n <namespace>
kubectl rollout status deployment/<name> -n <namespace>
```

### Incident Documentation
**Template for Security Incidents**:
```markdown
# Security Incident Report - [Date]

## Summary
- **Incident ID**: SEC-YYYY-MM-DD-###
- **Severity**: [Critical/High/Medium/Low]
- **Component**: [Affected component]
- **Discovery Time**: [Timestamp]
- **Resolution Time**: [Timestamp]

## Description
[Detailed description of the vulnerability/incident]

## Impact Assessment
- **Systems Affected**: [List of affected systems]
- **Data Exposure**: [Any data exposure details]
- **Service Availability**: [Impact on service availability]

## Response Actions
1. [Action 1 with timestamp]
2. [Action 2 with timestamp]
3. [Action 3 with timestamp]

## Root Cause Analysis
[Analysis of how the vulnerability was introduced]

## Prevention Measures
- [Measure 1]
- [Measure 2]
- [Measure 3]

## Lessons Learned
[Key takeaways for future improvements]
```

## 📊 Monitoring and Alerting

### Alert Response Procedures
**Critical Alerts** (Response time: < 5 minutes):
- Workflow failures with system impact
- Security vulnerabilities actively exploited
- Performance degradation > 50%
- System availability < 95%

**Warning Alerts** (Response time: < 30 minutes):
- Success rate drops below 90%
- High severity vulnerabilities
- Performance degradation 20-50%
- Resource utilization > 80%

**Info Alerts** (Response time: < 2 hours):
- Successful deployments
- Weekly/monthly reports
- Maintenance notifications
- Performance within acceptable ranges

### Escalation Procedures
```yaml
Level 1 - On-call Engineer:
  - Initial response and triage
  - Basic troubleshooting
  - Communication to team

Level 2 - Senior Engineer:
  - Complex technical issues
  - Architecture decisions
  - Cross-team coordination

Level 3 - Engineering Manager:
  - Business impact decisions
  - Resource allocation
  - External communication

Level 4 - Executive:
  - Critical business decisions
  - Public communication
  - Vendor escalation
```

## 📝 Documentation Maintenance

### Runbook Updates
**Frequency**: After each incident, quarterly reviews

**Update Triggers**:
- New vulnerability types discovered
- Process improvements identified
- Tool or technology changes
- Regulatory requirement changes

**Review Process**:
1. Incident post-mortem findings
2. Team feedback and suggestions
3. Industry best practice updates
4. Regulatory compliance changes
5. Tool and technology evolution

---

**Document Version**: 2.0
**Last Updated**: 2025-09-26
**Next Review**: 2025-12-26
**Owner**: CI/CD Team
**Reviewers**: Security Team, Platform Team

For emergency support: Create a GitHub issue with `critical` and `ci-cd` labels, or contact the on-call engineer via PagerDuty.