# GitHub Actions Best Practices 2025-09

This document outlines the latest GitHub Actions best practices as of September 2025, focusing on security, runner images, outputs, caching, and performance.

## Security Best Practices

### 1. Permission Management

**Principle of Least Privilege**
```yaml
# Set workflow permissions to read-only by default
permissions: {}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read      # Only grant specific permissions needed
      actions: read       # Be explicit about each permission
      security-events: write  # For security scanning
    steps:
      - uses: actions/checkout@v4
```

**Key Security Rules:**
- Always set `permissions: {}` at workflow level to force job-level specification
- Never grant `write-all` or leave permissions unspecified
- Use `contents: read` for most jobs, `write` only when pushing changes
- Limit `security-events: write` to security scanning jobs only

### 2. Secrets and Credential Management

**Short-lived Credentials with OIDC**
```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      id-token: write  # Required for OIDC
      contents: read
    steps:
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
          aws-region: us-east-1
          role-session-name: GitHubActions
```

**Secret Masking**
```yaml
- name: Mask sensitive values
  run: |
    echo "::add-mask::$SENSITIVE_VALUE"
    # Now $SENSITIVE_VALUE will be redacted from logs
```

**Best Practices:**
- Use OpenID Connect (OIDC) for cloud provider authentication
- Never store long-lived credentials as secrets
- Access secrets individually by name, not all at once
- Use `::add-mask::` for non-GitHub secrets that appear in logs
- Secrets are not passed to workflows triggered from forks by default

### 3. Input Validation and Injection Prevention

```yaml
- name: Validate inputs safely
  run: |
    # Use parameter substitution instead of direct interpolation
    input_value="${{ github.event.inputs.user_input }}"
    # Validate before use
    if [[ ! "$input_value" =~ ^[a-zA-Z0-9_-]+$ ]]; then
      echo "Invalid input format"
      exit 1
    fi
```

### 4. Runtime Security with Harden-Runner

```yaml
- name: Harden Runner
  uses: step-security/harden-runner@v2
  with:
    egress-policy: strict
    allowed-endpoints: >
      api.github.com:443
      github.com:443
      registry.npmjs.org:443
```

## Runner Images and Environment

### 1. Latest OS Images (2025)

**Recommended Runner Images:**
```yaml
strategy:
  matrix:
    os: [ubuntu-24.04, ubuntu-22.04]  # Use latest Ubuntu LTS
    include:
      - os: windows-2025  # Latest Windows Server
      - os: macos-15      # Latest macOS
```

**Self-hosted Runner Security:**
```yaml
runs-on: [self-hosted, linux, x64, isolated]  # Use labels for security boundaries
```

### 2. Container Security

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: node:20-alpine  # Use specific versions, prefer Alpine for security
      options: --read-only --tmpfs /tmp:exec  # Security hardening
```

## Output and Artifact Management

### 1. Multiline Outputs with JSON Serialization

**Best Practice: Use jq for reliable multiline outputs**
```yaml
- name: Set multiline output safely
  id: multiline
  run: |
    # Serialize to JSON using jq
    content=$(cat <<'EOF'
    Line 1 with special chars: ${{ github.token }}
    Line 2 with "quotes" and 'apostrophes'
    Line 3 with Unicode: 🚀
    EOF
    )
    # Use jq to safely serialize
    echo "content=$(printf '%s' "$content" | jq -Rs '.')" >> "$GITHUB_OUTPUT"

- name: Use multiline output
  run: |
    # Deserialize using fromJSON
    echo '${{ fromJSON(steps.multiline.outputs.content) }}'
```

**Alternative: Heredoc with Random Delimiter**
```yaml
- name: Set multiline output with heredoc
  id: heredoc
  run: |
    delimiter="$(openssl rand -hex 8)"
    echo "content<<${delimiter}" >> "$GITHUB_OUTPUT"
    cat your_multiline_file.txt >> "$GITHUB_OUTPUT"
    echo "${delimiter}" >> "$GITHUB_OUTPUT"
```

### 2. Matrix Strategy with JSON

```yaml
- name: Generate matrix
  id: matrix
  run: |
    # Use jq -c for compact JSON output
    matrix=$(echo '["item1", "item2", "item3"]' | jq -c '.')
    echo "items=$matrix" >> "$GITHUB_OUTPUT"

jobs:
  test:
    strategy:
      matrix:
        item: ${{ fromJSON(needs.setup.outputs.items) }}
```

### 3. Artifact Security

```yaml
- name: Upload artifacts securely
  uses: actions/upload-artifact@v4
  with:
    name: build-artifacts
    path: |
      dist/
      !dist/**/*.log      # Exclude log files
      !dist/**/*.env      # Exclude environment files
    retention-days: 7     # Minimize retention period
```

**Security Rules:**
- Never include sensitive information in artifacts
- Use explicit inclusion patterns, exclude sensitive file types
- Set minimal retention periods (7-30 days max)
- Artifacts are accessible to anyone with repository access

## Caching Best Practices

### 1. Effective Cache Configuration

**Node.js Example:**
```yaml
- name: Setup Node.js with caching
  uses: actions/setup-node@v4
  with:
    node-version: '20'
    cache: 'npm'
    cache-dependency-path: |
      package-lock.json
      packages/*/package-lock.json

- name: Cache node_modules
  uses: actions/cache@v4
  with:
    path: |
      ~/.npm
      node_modules
      */*/node_modules
    key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
    restore-keys: |
      ${{ runner.os }}-node-
```

### 2. Docker Layer Caching

```yaml
- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Build and push with cache
  uses: docker/build-push-action@v5
  with:
    context: .
    push: true
    tags: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}
    cache-from: type=gha
    cache-to: type=gha,mode=max
    platforms: linux/amd64,linux/arm64
```

### 3. Cache Security

```yaml
- name: Cache with secure key
  uses: actions/cache@v4
  with:
    path: ~/.cache/go-build
    # Use file hash for key, avoid sensitive values
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    # Never use secrets in cache keys
    # ❌ key: ${{ runner.os }}-${{ secrets.TOKEN }}-${{ github.actor }}
    # ✅ key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
```

### 4. Cross-platform Caching

```yaml
- name: Cache with cross-platform support
  uses: actions/cache@v4
  with:
    path: |
      ~/.cache/pip
      ~/AppData/Local/pip/Cache  # Windows
    key: ${{ runner.os }}-pip-${{ hashFiles('**/requirements.txt') }}
    enableCrossOsArchive: true  # Enable cross-platform cache sharing
```

## Performance Optimizations

### 1. Parallel Job Execution

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
        node-version: [18, 20, 22]
      fail-fast: false  # Continue other jobs if one fails
    runs-on: ${{ matrix.os }}
```

### 2. Conditional Job Execution

```yaml
jobs:
  test:
    if: github.event_name == 'push' || github.event.pull_request.draft == false
    runs-on: ubuntu-latest

  deploy:
    needs: test
    if: github.ref == 'refs/heads/main' && success()
    runs-on: ubuntu-latest
```

### 3. Efficient Dependency Installation

```yaml
- name: Install dependencies efficiently
  run: |
    # Use ci command for faster, reliable installs
    npm ci --prefer-offline --no-audit --no-fund
    # For Python
    pip install --no-deps -r requirements.txt
```

## Action Pinning and Updates

### 1. Pin to Stable Major Versions

```yaml
steps:
  # ✅ Pin to stable major versions
  - uses: actions/checkout@v4
  - uses: actions/setup-node@v4
  - uses: actions/setup-python@v5
  - uses: actions/cache@v4

  # ❌ Avoid floating tags
  # - uses: actions/checkout@main
  # - uses: actions/setup-node@latest
```

### 2. Third-party Action Security

```yaml
steps:
  # ✅ Pin third-party actions to specific SHA
  - uses: docker/build-push-action@4a13e500e55cf31b7a5d59a38ab2040ab0f42f56  # v5.1.0

  # Or use a tool like Dependabot to manage updates
  - uses: step-security/harden-runner@v2  # Maintained security action
```

## Workflow Organization

### 1. Reusable Workflows

```yaml
# .github/workflows/reusable-test.yml
name: Reusable Test Workflow

on:
  workflow_call:
    inputs:
      node-version:
        required: true
        type: string
    secrets:
      NPM_TOKEN:
        required: true

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ inputs.node-version }}
```

### 2. Composite Actions

```yaml
# .github/actions/setup-environment/action.yml
name: 'Setup Environment'
description: 'Setup Node.js and install dependencies'
inputs:
  node-version:
    description: 'Node.js version'
    required: true
    default: '20'

runs:
  using: 'composite'
  steps:
    - uses: actions/setup-node@v4
      with:
        node-version: ${{ inputs.node-version }}
        cache: 'npm'
    - run: npm ci
      shell: bash
```

## Monitoring and Debugging

### 1. Debug Logging

```yaml
- name: Debug information
  run: |
    echo "Runner OS: ${{ runner.os }}"
    echo "GitHub context:"
    echo '${{ toJSON(github) }}' | jq .
  env:
    ACTIONS_STEP_DEBUG: true  # Enable debug logging
```

### 2. Job Summaries

```yaml
- name: Test Results
  run: |
    echo "## Test Results 🧪" >> $GITHUB_STEP_SUMMARY
    echo "| Test Suite | Status | Duration |" >> $GITHUB_STEP_SUMMARY
    echo "|------------|--------|----------|" >> $GITHUB_STEP_SUMMARY
    echo "| Unit Tests | ✅ Pass | 2m 30s |" >> $GITHUB_STEP_SUMMARY
```

## Common Pitfalls to Avoid

### 1. Security Anti-patterns

```yaml
# ❌ DON'T: Log sensitive information
- run: echo "Token: ${{ secrets.GITHUB_TOKEN }}"

# ❌ DON'T: Use secrets in conditions
- if: ${{ secrets.DEPLOY_KEY != '' }}

# ❌ DON'T: Grant excessive permissions
permissions: write-all

# ❌ DON'T: Trust user input without validation
- run: echo "Hello ${{ github.event.inputs.name }}"
```

### 2. Performance Anti-patterns

```yaml
# ❌ DON'T: Install unnecessary dependencies
- run: npm install  # Use npm ci instead

# ❌ DON'T: Run jobs unnecessarily
# Missing: if: conditions on expensive jobs

# ❌ DON'T: Use inefficient caching
# Missing: proper cache keys and paths
```

### 3. Reliability Anti-patterns

```yaml
# ❌ DON'T: Use unstable action versions
- uses: some-action@main

# ❌ DON'T: Ignore error handling
- run: potentially-failing-command
  # Missing: continue-on-error or proper error handling

# ❌ DON'T: Create non-deterministic workflows
- run: sleep $((RANDOM % 10))  # Non-deterministic behavior
```

## Summary

These best practices emphasize:
- **Security-first approach** with least privilege and OIDC
- **Reliable multiline outputs** using JSON serialization
- **Efficient caching** with proper key management
- **Performance optimization** through parallelization and caching
- **Proper action pinning** for security and reliability
- **Comprehensive monitoring** and debugging capabilities

Following these practices ensures robust, secure, and efficient CI/CD pipelines for 2025 and beyond.