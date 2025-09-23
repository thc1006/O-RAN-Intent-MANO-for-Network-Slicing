# Security Test Suite

This directory contains comprehensive security tests for the O-RAN Intent MANO project, validating the security fixes and ensuring robust protection against various attack vectors.

## Test Structure

### Core Security Tests

1. **Command Injection Prevention Tests** (`command_injection_test.go`)
   - Tests secure subprocess execution against various injection attacks
   - Validates command allowlist enforcement
   - Tests argument validation and sanitization
   - Includes timeout enforcement and output size limits

2. **HTTP Server Security Tests** (`http_security_test.go`)
   - Tests HTTP security configuration and middleware
   - Validates request size limits and timeouts
   - Tests security headers injection
   - Includes CORS validation and rate limiting

3. **Error Handling Security Tests** (`error_handling_test.go`)
   - Tests secure error handling and logging
   - Validates sensitive data sanitization
   - Tests error rate limiting and truncation
   - Includes panic recovery and concurrent safety

4. **Negative Security Tests** (`negative_security_test.go`)
   - Comprehensive security violation detection tests
   - Tests various attack vectors and bypass attempts
   - Includes fuzzing and edge case testing
   - Validates security boundary enforcement

5. **Integration Security Tests** (`integration_security_test.go`)
   - End-to-end security workflow testing
   - Tests security component integration
   - Validates concurrent security operations
   - Includes compliance testing (OWASP, CWE)

6. **Performance Benchmarks** (`benchmark_security_test.go`)
   - Security performance benchmarks and regression testing
   - Memory usage analysis under attack scenarios
   - Concurrent operation performance testing
   - Resource consumption validation

7. **Fuzzing Tests** (`fuzzing_security_test.go`)
   - Advanced fuzzing tests for edge cases
   - Unicode and encoding attack testing
   - ReDoS (Regular Expression DoS) protection
   - Property-based testing for security invariants

8. **Coverage Analysis** (`coverage_security_test.go`)
   - Security test coverage validation
   - Code quality analysis for security functions
   - Test maintainability assessment
   - Compliance reporting

## Security Test Categories

### Input Validation Tests
- Command argument validation
- IP address validation
- File path validation
- Network interface validation
- Environment variable validation

### Attack Prevention Tests
- Command injection attacks
- Path traversal attacks
- SQL injection attempts
- XSS prevention
- CSRF protection
- Buffer overflow attempts
- Unicode bypass attempts
- Encoding bypass attempts

### Security Boundary Tests
- Authentication boundaries
- Authorization boundaries
- Input size limits
- Rate limiting
- Timeout enforcement
- Resource consumption limits

### Error Handling Tests
- Sensitive data sanitization
- Error message standardization
- Logging security
- Panic recovery
- Information disclosure prevention

## Test Execution

### Running All Security Tests
```bash
go test -v ./tests/security/...
```

### Running Specific Test Categories
```bash
# Command injection tests
go test -v ./tests/security/ -run TestCommandInjectionPrevention

# HTTP security tests
go test -v ./tests/security/ -run TestHTTPServerSecurityConfiguration

# Fuzzing tests (extended)
go test -v ./tests/security/ -run TestFuzzing

# Performance benchmarks
go test -v ./tests/security/ -run Benchmark -bench=.
```

### Running with Coverage
```bash
go test -v -coverprofile=security_coverage.out ./tests/security/...
go tool cover -html=security_coverage.out -o security_coverage.html
```

## Security Test Requirements

### Coverage Requirements
- **Critical Security Functions**: >90% line coverage
- **High Security Functions**: >80% line coverage
- **Medium Security Functions**: >70% line coverage
- **Overall Security Coverage**: >85%

### Test Quality Requirements
- **Negative Test Coverage**: >50% for security functions
- **Edge Case Coverage**: >70% of identified edge cases
- **Boundary Test Coverage**: >75% of security boundaries
- **Error Handling Coverage**: >60% of error scenarios

## Security Compliance

### Standards Compliance
- **OWASP Top 10 2021**: All relevant risks addressed
- **CWE Top 25**: Most dangerous weaknesses covered
- **NIST Cybersecurity Framework**: Core security functions tested
- **ISO 27001**: Security control testing

### Attack Vectors Tested
- **Injection Attacks**: Command, SQL, NoSQL, LDAP, XPath
- **Broken Authentication**: Session management, credential validation
- **Sensitive Data Exposure**: Error messages, logs, responses
- **XML External Entities (XXE)**: XML parsing security
- **Broken Access Control**: Authorization bypass attempts
- **Security Misconfiguration**: Default settings, headers
- **Cross-Site Scripting (XSS)**: Stored, reflected, DOM-based
- **Insecure Deserialization**: Object injection attacks
- **Using Components with Known Vulnerabilities**: Dependency scanning
- **Insufficient Logging & Monitoring**: Security event detection

## Test Data and Fixtures

### Malicious Payloads
- Command injection payloads
- Path traversal sequences
- XSS vectors
- SQL injection attempts
- Unicode normalization attacks
- Binary exploitation attempts

### Test Environments
- Isolated test containers
- Mock external services
- Controlled network environments
- Temporary file systems

## Security Test Best Practices

### Test Design
1. **Defense in Depth**: Test multiple security layers
2. **Least Privilege**: Validate minimal permission enforcement
3. **Fail Secure**: Ensure secure failure modes
4. **Complete Mediation**: Test all access control points
5. **Open Design**: Security through proper design, not obscurity

### Test Implementation
1. **Deterministic Tests**: Reproducible results
2. **Isolated Tests**: No dependencies between tests
3. **Fast Execution**: Efficient test runtime
4. **Clear Assertions**: Explicit security requirements
5. **Comprehensive Coverage**: All attack vectors and edge cases

### Test Maintenance
1. **Regular Updates**: Keep pace with threat landscape
2. **Automated Execution**: Continuous security testing
3. **Performance Monitoring**: Track test execution metrics
4. **Documentation Updates**: Maintain current test documentation
5. **Compliance Tracking**: Monitor regulatory requirement changes

## Security Test Automation

### Continuous Integration
- Automated security test execution on every commit
- Security regression detection
- Performance benchmark validation
- Coverage requirement enforcement

### Security Pipelines
- Static application security testing (SAST)
- Dynamic application security testing (DAST)
- Interactive application security testing (IAST)
- Software composition analysis (SCA)

## Incident Response Testing

### Security Incident Simulation
- Breach detection testing
- Response time measurement
- Recovery procedure validation
- Communication protocol testing

### Forensic Capability Testing
- Log integrity verification
- Evidence collection procedures
- Chain of custody validation
- Timeline reconstruction testing

## Security Metrics and Reporting

### Key Security Metrics
- Test coverage percentages
- Vulnerability detection rates
- False positive/negative rates
- Mean time to detection (MTTD)
- Mean time to response (MTTR)

### Security Dashboards
- Real-time security test status
- Trend analysis and reporting
- Risk assessment summaries
- Compliance status tracking

## Contributing to Security Tests

### Adding New Security Tests
1. Identify security requirements
2. Design comprehensive test cases
3. Implement with proper coverage
4. Document test purpose and methodology
5. Update this README with new test information

### Security Test Review Process
1. Peer review for test completeness
2. Security expert validation
3. Performance impact assessment
4. Integration with existing test suite
5. Documentation review and approval

## Security Resources

### External Security Testing Tools
- **OWASP ZAP**: Web application security testing
- **Burp Suite**: Security testing platform
- **Nmap**: Network discovery and security auditing
- **Wireshark**: Network protocol analyzer

### Security Documentation
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [NIST SP 800-53](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [CWE/SANS Top 25](https://cwe.mitre.org/top25/)
- [ASVS - Application Security Verification Standard](https://owasp.org/www-project-application-security-verification-standard/)

## Contact and Support

For questions about security testing or to report security issues:
- Security Team: [security@example.com]
- Documentation: [security-docs-link]
- Issue Tracking: [security-issues-link]

---

**Note**: This security test suite is designed to validate the robustness of security fixes and ensure comprehensive protection against known attack vectors. Regular updates and maintenance are essential to keep pace with the evolving threat landscape.