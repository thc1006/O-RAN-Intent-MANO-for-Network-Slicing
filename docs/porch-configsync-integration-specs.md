# Porch and ConfigSync Integration Specifications

## Executive Summary

This document provides comprehensive specifications for integrating Porch (Nephio package management) and ConfigSync (GitOps synchronization) within the O-RAN Intent-Based MANO system. The specifications enable intent-driven network slice orchestration with sub-10-minute deployment targets through automated package generation, multi-cluster distribution, and coordinated deployment workflows.

## 1. Porch API Integration Patterns

### 1.1 PackageRevision Lifecycle Management

The PackageRevision lifecycle follows the pattern: Draft → Proposed → Published → Deployed, with automated transitions based on validation and approval criteria.

#### 1.1.1 Draft State Operations

```yaml
# Draft Package Creation from Intent
apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: network-slice-embb-001-v1
  namespace: porch-system
  labels:
    intent.oran.mano/slice-type: "embb"
    intent.oran.mano/service-id: "slice-001"
    porch.kpt.dev/lifecycle: "draft"
spec:
  packageName: network-slice-embb-001
  revision: v1
  repository: nephio-packages
  workspaceName: ws-embb-001
  lifecycle: Draft
  resources:
    - name: "network-slice-blueprint"
      content: |
        apiVersion: workload.nephio.org/v1alpha1
        kind: NetworkSlice
        metadata:
          name: embb-slice-001
        spec:
          sliceType: eMBB
          qosProfile:
            name: "embb-high-throughput"
            maxBitRate: "1Gbps"
            latency: "20ms"
            reliability: "99.9%"
          coverage:
            areas:
              - geofence: "central-london"
                priority: 1
              - geofence: "canary-wharf"
                priority: 2
          endpoints:
            ran:
              - cluster: "edge-01"
                location: "site-a"
              - cluster: "edge-02"
                location: "site-b"
            core:
              cluster: "regional-01"
              location: "region-a"
```

#### 1.1.2 Package Validation Functions

```yaml
# Validation Function Pipeline
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Function
metadata:
  name: network-slice-validator
  namespace: porch-functions
spec:
  image: gcr.io/oran-mano/network-slice-validator:v1.0.0
  configPath: validation-config.yaml
  config:
    rules:
      - name: "qos-compliance"
        description: "Validate QoS parameters against O-RAN standards"
        enabled: true
      - name: "resource-limits"
        description: "Ensure resource requests are within cluster capacity"
        enabled: true
      - name: "security-policies"
        description: "Validate security policies and RBAC"
        enabled: true
    thresholds:
      cpu: "80%"
      memory: "85%"
      storage: "90%"
```

#### 1.1.3 Automated Promotion Workflow

```yaml
# Package Promotion Automation
apiVersion: v1
kind: ConfigMap
metadata:
  name: promotion-workflow
  namespace: porch-system
data:
  workflow.yaml: |
    promotion:
      draft_to_proposed:
        triggers:
          - validation_passed: true
          - security_scan_clean: true
          - resource_validation: true
        automation:
          enabled: true
          approvers:
            - "system:serviceaccount:porch-system:auto-promoter"
        conditions:
          - name: "all-validations-pass"
            expression: "validation.status == 'passed'"
          - name: "no-security-issues"
            expression: "security.findings.critical == 0"

      proposed_to_published:
        triggers:
          - manual_approval: true
          - change_review_complete: true
        approvers:
          - "group:network-architects"
          - "group:security-team"
        automation:
          enabled: false  # Requires manual approval

      published_to_deployed:
        triggers:
          - deployment_ready: true
          - cluster_capacity_available: true
        automation:
          enabled: true
          conditions:
            - name: "cluster-readiness"
              expression: "cluster.status == 'ready'"
            - name: "capacity-check"
              expression: "cluster.capacity.available >= package.requirements"
```

### 1.2 Repository Management for Multi-Vendor NF Catalogs

#### 1.2.1 Repository Structure Definition

```yaml
# Multi-Vendor Package Repository Configuration
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: nephio-nf-catalog
  namespace: porch-system
  labels:
    catalog.nephio.org/type: "network-functions"
spec:
  type: git
  content: Package
  git:
    repo: https://github.com/oran-mano/nephio-nf-catalog
    branch: main
    directory: /
    auth:
      secretRef:
        name: git-credentials
  deployment: true
  mutators:
    - image: gcr.io/oran-mano/nf-customizer:v1.0.0
    - image: gcr.io/oran-mano/qos-injector:v1.0.0
    - image: gcr.io/oran-mano/vendor-adapter:v1.0.0
---
# Vendor-Specific Repository for RAN Functions
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: vendor-a-ran-catalog
  namespace: porch-system
  labels:
    catalog.nephio.org/vendor: "vendor-a"
    catalog.nephio.org/domain: "ran"
spec:
  type: git
  content: Package
  git:
    repo: https://github.com/vendor-a/ran-functions
    branch: main
    directory: /packages
    auth:
      secretRef:
        name: vendor-a-credentials
  deployment: false  # Catalog only, not for direct deployment
  upstream:
    git:
      repo: https://github.com/oran-mano/nephio-nf-catalog
      branch: main
      directory: /upstream/vendor-a
```

#### 1.2.2 Package Catalog Organization

```
nephio-nf-catalog/
├── catalog/
│   ├── ran/
│   │   ├── gnb/
│   │   │   ├── vendor-a/
│   │   │   │   ├── v1.0.0/
│   │   │   │   │   ├── Kptfile
│   │   │   │   │   ├── package.yaml
│   │   │   │   │   ├── workload.yaml
│   │   │   │   │   └── customizations/
│   │   │   │   └── v1.1.0/
│   │   │   ├── vendor-b/
│   │   │   └── vendor-c/
│   │   ├── cu/
│   │   └── du/
│   ├── core/
│   │   ├── amf/
│   │   ├── smf/
│   │   ├── upf/
│   │   └── nrf/
│   └── edge/
│       ├── mec-platform/
│       └── cdn/
├── blueprints/
│   ├── network-slices/
│   │   ├── embb/
│   │   ├── urllc/
│   │   └── miot/
│   └── scenarios/
└── functions/
    ├── validators/
    ├── mutators/
    └── generators/
```

### 1.3 Function Evaluation and Package Mutation Workflows

#### 1.3.1 QoS-Driven Package Customization Function

```yaml
# QoS Mutation Function
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Function
metadata:
  name: qos-driven-mutator
  namespace: porch-functions
spec:
  image: gcr.io/oran-mano/qos-mutator:v1.2.0
  configPath: qos-config.yaml
  config:
    mutations:
      - target: "workload.nephio.org/v1alpha1/NetworkFunction"
        conditions:
          - path: "spec.qosProfile.sliceType"
            value: "eMBB"
        operations:
          - op: "replace"
            path: "/spec/resources/requests/cpu"
            value: "2000m"
          - op: "replace"
            path: "/spec/resources/requests/memory"
            value: "4Gi"
          - op: "add"
            path: "/spec/nodeAffinity"
            value:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms:
                - matchExpressions:
                  - key: "node-type"
                    operator: In
                    values: ["high-performance"]

      - target: "workload.nephio.org/v1alpha1/NetworkFunction"
        conditions:
          - path: "spec.qosProfile.sliceType"
            value: "uRLLC"
        operations:
          - op: "replace"
            path: "/spec/resources/requests/cpu"
            value: "4000m"
          - op: "replace"
            path: "/spec/resources/requests/memory"
            value: "8Gi"
          - op: "add"
            path: "/spec/tolerations"
            value:
            - key: "latency-critical"
              operator: "Equal"
              value: "true"
              effect: "NoSchedule"
```

#### 1.3.2 Multi-Site Deployment Function

```yaml
# Multi-Site Package Generator Function
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Function
metadata:
  name: multi-site-generator
  namespace: porch-functions
spec:
  image: gcr.io/oran-mano/multi-site-generator:v1.0.0
  configPath: site-config.yaml
  config:
    sites:
      - name: "edge-01"
        type: "edge"
        location: "london-east"
        clusters: ["kind-edge-01"]
        resources:
          cpu: "16"
          memory: "32Gi"
          storage: "1Ti"
        specializations:
          - ran-functions
          - edge-computing

      - name: "regional-01"
        type: "regional"
        location: "london-central"
        clusters: ["kind-regional-01"]
        resources:
          cpu: "64"
          memory: "128Gi"
          storage: "10Ti"
        specializations:
          - core-functions
          - data-analytics

      - name: "central-01"
        type: "central"
        location: "london-west"
        clusters: ["kind-central-01"]
        resources:
          cpu: "128"
          memory: "512Gi"
          storage: "100Ti"
        specializations:
          - management-functions
          - ml-workloads

    generation_rules:
      - name: "ran-placement"
        condition: "spec.workload.type == 'ran'"
        target_sites: ["edge-01", "edge-02"]
        distribution: "active-active"

      - name: "core-placement"
        condition: "spec.workload.type == 'core'"
        target_sites: ["regional-01"]
        distribution: "active-standby"

      - name: "management-placement"
        condition: "spec.workload.type == 'management'"
        target_sites: ["central-01"]
        distribution: "singleton"
```

## 2. ConfigSync GitOps Patterns

### 2.1 Multi-Cluster Package Distribution Strategies

#### 2.1.1 Hierarchical Distribution Pattern

```yaml
# Root Sync Configuration for Central Management
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: nephio-root-sync
  namespace: config-management-system
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-deployments
    branch: main
    dir: /clusters/central
    auth: ssh
    secretRef:
      name: git-creds
  override:
    resources:
    - group: ""
      kind: "Namespace"
      operations: ["*"]
    - group: "workload.nephio.org"
      kind: "*"
      operations: ["*"]
  renderingRequired: true
---
# Repo Sync for Edge Clusters
apiVersion: configsync.gke.io/v1beta1
kind: RepoSync
metadata:
  name: edge-cluster-sync
  namespace: nephio-workloads
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-deployments
    branch: main
    dir: /clusters/edge/edge-01
    auth: ssh
    secretRef:
      name: git-creds
  override:
    namespaceStrategy: explicit
    resources:
    - group: "workload.nephio.org"
      kind: "NetworkFunction"
      operations: ["create", "update", "patch"]
```

#### 2.1.2 Multi-Cluster Coordination

```yaml
# ClusterSelector for Targeted Deployments
apiVersion: configmanagement.gke.io/v1
kind: ClusterSelector
metadata:
  name: edge-clusters
spec:
  selector:
    matchLabels:
      cluster-type: "edge"
      deployment-zone: "london"
---
apiVersion: configmanagement.gke.io/v1
kind: ClusterSelector
metadata:
  name: regional-clusters
spec:
  selector:
    matchLabels:
      cluster-type: "regional"
      deployment-zone: "uk"
---
# Policy for Cross-Cluster Dependencies
apiVersion: configmanagement.gke.io/v1
kind: ResourceQuota
metadata:
  name: network-slice-quota
  namespace: nephio-workloads
  annotations:
    cluster-selector: "edge-clusters"
spec:
  hard:
    pods: "50"
    persistentvolumeclaims: "10"
    services: "20"
    secrets: "30"
    configmaps: "30"
    requests.cpu: "20"
    requests.memory: "40Gi"
    requests.storage: "1Ti"
```

### 2.2 Repository Synchronization Patterns

#### 2.2.1 Selective Synchronization

```yaml
# Namespace-Specific Sync Configuration
apiVersion: configsync.gke.io/v1beta1
kind: RepoSync
metadata:
  name: ran-functions-sync
  namespace: ran-workloads
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-packages
    branch: main
    dir: /deployments/ran
    auth: ssh
    secretRef:
      name: git-creds
  override:
    resources:
    - group: "workload.nephio.org"
      kind: "RANFunction"
      operations: ["*"]
    - group: "apps"
      kind: "Deployment"
      operations: ["create", "update", "patch"]
      namespaces: ["ran-workloads"]
  renderingRequired: true
  hydrationController:
    enabled: true
    source:
      path: /blueprints/ran
---
# Core Network Functions Sync
apiVersion: configsync.gke.io/v1beta1
kind: RepoSync
metadata:
  name: core-functions-sync
  namespace: core-workloads
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-packages
    branch: main
    dir: /deployments/core
    auth: ssh
    secretRef:
      name: git-creds
  override:
    resources:
    - group: "workload.nephio.org"
      kind: "CoreFunction"
      operations: ["*"]
    - group: "networking.k8s.io"
      kind: "*"
      operations: ["*"]
      namespaces: ["core-workloads"]
```

#### 2.2.2 Configuration Drift Detection

```yaml
# Drift Detection Policy
apiVersion: v1
kind: ConfigMap
metadata:
  name: drift-detection-config
  namespace: config-management-system
data:
  config.yaml: |
    driftDetection:
      enabled: true
      interval: "30s"

      # Critical resources that must not drift
      protectedResources:
        - group: "workload.nephio.org"
          kind: "NetworkSlice"
          action: "remediate"
        - group: "security.nephio.org"
          kind: "SecurityPolicy"
          action: "alert_and_remediate"
        - group: ""
          kind: "Secret"
          namespaces: ["nephio-system"]
          action: "alert_only"

      # Allowable drift for certain resources
      toleratedDrift:
        - group: "apps"
          kind: "Deployment"
          paths: ["/spec/replicas"]
          threshold: "10%"
        - group: ""
          kind: "ConfigMap"
          paths: ["/data/debug-level"]
          action: "log_only"

      # Remediation strategies
      remediation:
        automatic:
          enabled: true
          backoff: "exponential"
          maxRetries: 3

        manual:
          escalation: true
          notificationChannels:
            - slack: "#oran-mano-alerts"
            - email: "ops-team@company.com"
```

### 2.3 Namespace Management and Resource Isolation

#### 2.3.1 Network Slice Namespace Strategy

```yaml
# Namespace Template for Network Slices
apiVersion: v1
kind: Namespace
metadata:
  name: slice-embb-001
  labels:
    slice.oran.mano/type: "embb"
    slice.oran.mano/id: "slice-001"
    slice.oran.mano/tenant: "operator-a"
    config.kubernetes.io/local-config: "true"
  annotations:
    config.kubernetes.io/namespace-selector: "slice-type=embb"
    configsync.gke.io/managed: "enabled"
spec:
  finalizers:
  - kubernetes
---
# RBAC for Slice Isolation
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: slice-embb-001
  name: slice-operator
rules:
- apiGroups: ["workload.nephio.org"]
  resources: ["networkfunctions", "networkslices"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
# Network Policy for Slice Isolation
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: slice-isolation-policy
  namespace: slice-embb-001
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          slice.oran.mano/type: "embb"
    - namespaceSelector:
        matchLabels:
          name: "nephio-system"
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          slice.oran.mano/type: "embb"
    - namespaceSelector:
        matchLabels:
          name: "kube-system"
```

## 3. Integration Workflows

### 3.1 Intent-Driven Package Generation → Porch → ConfigSync → Deployment

#### 3.1.1 End-to-End Workflow Definition

```yaml
# Workflow Definition for Intent Processing
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: intent-to-deployment
  namespace: nephio-system
spec:
  entrypoint: intent-processing
  templates:
  - name: intent-processing
    dag:
      tasks:
      - name: parse-intent
        template: parse-natural-language
        arguments:
          parameters:
          - name: intent
            value: "{{workflow.parameters.user-intent}}"

      - name: generate-qos-config
        template: qos-generation
        dependencies: [parse-intent]
        arguments:
          parameters:
          - name: parsed-intent
            value: "{{tasks.parse-intent.outputs.parameters.result}}"

      - name: create-package-draft
        template: porch-package-creation
        dependencies: [generate-qos-config]
        arguments:
          parameters:
          - name: qos-config
            value: "{{tasks.generate-qos-config.outputs.parameters.result}}"

      - name: validate-package
        template: package-validation
        dependencies: [create-package-draft]
        arguments:
          parameters:
          - name: package-revision
            value: "{{tasks.create-package-draft.outputs.parameters.revision}}"

      - name: promote-to-proposed
        template: package-promotion
        dependencies: [validate-package]
        when: "{{tasks.validate-package.outputs.parameters.status}} == 'passed'"

      - name: approve-package
        template: manual-approval
        dependencies: [promote-to-proposed]

      - name: publish-package
        template: package-publishing
        dependencies: [approve-package]
        when: "{{tasks.approve-package.outputs.parameters.approved}} == 'true'"

      - name: sync-to-clusters
        template: configsync-distribution
        dependencies: [publish-package]

      - name: monitor-deployment
        template: deployment-monitoring
        dependencies: [sync-to-clusters]

  - name: parse-natural-language
    container:
      image: gcr.io/oran-mano/nlp-processor:v1.0.0
      command: [python]
      args: ["/app/parse_intent.py"]
      env:
      - name: INTENT_TEXT
        value: "{{inputs.parameters.intent}}"
    outputs:
      parameters:
      - name: result
        valueFrom:
          path: /tmp/parsed-intent.json

  - name: qos-generation
    container:
      image: gcr.io/oran-mano/qos-generator:v1.0.0
      command: [python]
      args: ["/app/generate_qos.py"]
      env:
      - name: PARSED_INTENT
        value: "{{inputs.parameters.parsed-intent}}"
    outputs:
      parameters:
      - name: result
        valueFrom:
          path: /tmp/qos-config.json

  - name: porch-package-creation
    container:
      image: gcr.io/oran-mano/porch-client:v1.0.0
      command: ["/app/create-package"]
      args: ["--qos-config", "{{inputs.parameters.qos-config}}"]
    outputs:
      parameters:
      - name: revision
        valueFrom:
          path: /tmp/package-revision.txt
```

#### 3.1.2 Package Lifecycle Automation

```yaml
# Package Lifecycle Controller
apiVersion: apps/v1
kind: Deployment
metadata:
  name: package-lifecycle-controller
  namespace: nephio-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: package-lifecycle-controller
  template:
    metadata:
      labels:
        app: package-lifecycle-controller
    spec:
      containers:
      - name: controller
        image: gcr.io/oran-mano/package-lifecycle:v1.0.0
        env:
        - name: PORCH_API_ENDPOINT
          value: "https://porch-api.nephio-system.svc.cluster.local"
        - name: CONFIGSYNC_REPO
          value: "https://github.com/oran-mano/nephio-deployments"
        - name: AUTOMATION_RULES
          valueFrom:
            configMapKeyRef:
              name: lifecycle-automation
              key: rules.yaml
        volumeMounts:
        - name: automation-config
          mountPath: /etc/automation
      volumes:
      - name: automation-config
        configMap:
          name: lifecycle-automation
---
# Lifecycle Automation Rules
apiVersion: v1
kind: ConfigMap
metadata:
  name: lifecycle-automation
  namespace: nephio-system
data:
  rules.yaml: |
    automation:
      triggers:
        - event: "PackageRevision.lifecycle.changed"
          from: "Draft"
          to: "Proposed"
          condition: "validation.status == 'passed'"
          action: "auto-promote"

        - event: "PackageRevision.lifecycle.changed"
          from: "Proposed"
          to: "Published"
          condition: "approval.status == 'approved'"
          action: "auto-publish"

        - event: "PackageRevision.lifecycle.changed"
          from: "Published"
          to: "Deployed"
          condition: "deployment.ready == true"
          action: "sync-to-clusters"

      policies:
        - name: "security-gate"
          stage: "proposed"
          required: true
          validations:
            - "security-scan-passed"
            - "no-critical-vulnerabilities"

        - name: "capacity-check"
          stage: "published"
          required: true
          validations:
            - "cluster-capacity-sufficient"
            - "resource-quotas-available"

        - name: "dependency-check"
          stage: "deployment"
          required: true
          validations:
            - "dependencies-available"
            - "version-compatibility"
```

### 3.2 Error Handling and Rollback Procedures

#### 3.2.1 Automated Rollback Configuration

```yaml
# Rollback Policy Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: rollback-policies
  namespace: nephio-system
data:
  policies.yaml: |
    rollback:
      triggers:
        - name: "deployment-failure"
          condition: "deployment.status == 'failed'"
          timeout: "10m"
          action: "auto-rollback"

        - name: "health-check-failure"
          condition: "health.check.failed_count >= 3"
          timeout: "5m"
          action: "auto-rollback"

        - name: "resource-exhaustion"
          condition: "cluster.resources.available < 10%"
          timeout: "2m"
          action: "scale-down-and-rollback"

      strategies:
        - name: "immediate-rollback"
          steps:
            - "stop-new-deployments"
            - "revert-package-revision"
            - "sync-previous-state"
            - "verify-rollback-success"

        - name: "canary-rollback"
          steps:
            - "reduce-canary-traffic"
            - "monitor-stability"
            - "full-rollback-if-needed"

        - name: "blue-green-rollback"
          steps:
            - "switch-traffic-to-green"
            - "drain-blue-environment"
            - "cleanup-failed-deployment"

---
# Rollback Execution Workflow
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: rollback-execution
  namespace: nephio-system
spec:
  entrypoint: execute-rollback
  templates:
  - name: execute-rollback
    inputs:
      parameters:
      - name: failed-revision
      - name: target-revision
      - name: rollback-strategy
    dag:
      tasks:
      - name: validate-rollback
        template: validate-rollback-target
        arguments:
          parameters:
          - name: target-revision
            value: "{{inputs.parameters.target-revision}}"

      - name: create-rollback-plan
        template: generate-rollback-plan
        dependencies: [validate-rollback]
        arguments:
          parameters:
          - name: strategy
            value: "{{inputs.parameters.rollback-strategy}}"

      - name: execute-plan
        template: execute-rollback-plan
        dependencies: [create-rollback-plan]
        arguments:
          parameters:
          - name: plan
            value: "{{tasks.create-rollback-plan.outputs.parameters.plan}}"

      - name: verify-rollback
        template: verify-rollback-success
        dependencies: [execute-plan]

      - name: cleanup-artifacts
        template: cleanup-failed-artifacts
        dependencies: [verify-rollback]
        arguments:
          parameters:
          - name: failed-revision
            value: "{{inputs.parameters.failed-revision}}"
```

## 4. Repository Structure Patterns

### 4.1 Package Catalog Repositories (Upstream Templates)

```
nephio-package-catalog/
├── README.md
├── catalog-info.yaml
├── packages/
│   ├── network-functions/
│   │   ├── ran/
│   │   │   ├── gnb/
│   │   │   │   ├── default/
│   │   │   │   │   ├── Kptfile
│   │   │   │   │   ├── package.yaml
│   │   │   │   │   ├── workload.yaml
│   │   │   │   │   ├── service.yaml
│   │   │   │   │   └── network-attachment.yaml
│   │   │   │   ├── vendor-a/
│   │   │   │   └── vendor-b/
│   │   │   ├── cu/
│   │   │   └── du/
│   │   ├── core/
│   │   │   ├── amf/
│   │   │   ├── smf/
│   │   │   ├── upf/
│   │   │   └── nrf/
│   │   └── edge/
│   ├── network-slices/
│   │   ├── embb/
│   │   │   ├── high-throughput/
│   │   │   └── standard/
│   │   ├── urllc/
│   │   │   ├── ultra-low-latency/
│   │   │   └── standard/
│   │   └── miot/
│   └── scenarios/
│       ├── smart-city/
│       ├── industrial-iot/
│       └── enhanced-mobile-broadband/
├── functions/
│   ├── validators/
│   │   ├── qos-validator/
│   │   ├── security-validator/
│   │   └── resource-validator/
│   ├── mutators/
│   │   ├── qos-injector/
│   │   ├── site-customizer/
│   │   └── vendor-adapter/
│   └── generators/
│       ├── slice-generator/
│       ├── multi-site-generator/
│       └── dependency-generator/
├── blueprints/
│   ├── deployment-patterns/
│   │   ├── single-cluster/
│   │   ├── multi-cluster/
│   │   └── edge-cloud/
│   └── integration-patterns/
│       ├── o2ims/
│       ├── o2dms/
│       └── configsync/
└── docs/
    ├── package-development.md
    ├── validation-rules.md
    └── deployment-guides/
```

### 4.2 Blueprint Repositories (Customized Packages)

```yaml
# Blueprint Repository Configuration
apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: nephio-blueprints
  namespace: porch-system
  labels:
    catalog.nephio.org/type: "blueprints"
spec:
  type: git
  content: Package
  git:
    repo: https://github.com/oran-mano/nephio-blueprints
    branch: main
    directory: /
    auth:
      secretRef:
        name: git-credentials
  deployment: false
  upstream:
    git:
      repo: https://github.com/oran-mano/nephio-package-catalog
      branch: main
      directory: /packages
  mutators:
    - image: gcr.io/oran-mano/blueprint-customizer:v1.0.0
    - image: gcr.io/oran-mano/site-injector:v1.0.0
```

### 4.3 Deployment Repositories (Cluster-Specific Configs)

```
nephio-deployments/
├── clusters/
│   ├── central/
│   │   ├── central-01/
│   │   │   ├── cluster-config/
│   │   │   │   ├── namespace.yaml
│   │   │   │   ├── rbac.yaml
│   │   │   │   └── network-policies.yaml
│   │   │   ├── workloads/
│   │   │   │   ├── management-plane/
│   │   │   │   ├── orchestration/
│   │   │   │   └── analytics/
│   │   │   └── configsync/
│   │   │       ├── root-sync.yaml
│   │   │       └── repo-sync.yaml
│   │   └── central-02/
│   ├── regional/
│   │   ├── regional-01/
│   │   │   ├── cluster-config/
│   │   │   ├── workloads/
│   │   │   │   ├── core-network/
│   │   │   │   ├── edge-orchestration/
│   │   │   │   └── data-plane/
│   │   │   └── configsync/
│   │   └── regional-02/
│   └── edge/
│       ├── edge-01/
│       │   ├── cluster-config/
│       │   ├── workloads/
│       │   │   ├── ran-functions/
│       │   │   ├── edge-apps/
│       │   │   └── local-cache/
│       │   └── configsync/
│       └── edge-02/
├── network-slices/
│   ├── active/
│   │   ├── slice-001-embb/
│   │   ├── slice-002-urllc/
│   │   └── slice-003-miot/
│   ├── staging/
│   └── templates/
├── policies/
│   ├── security/
│   ├── networking/
│   └── resource-management/
└── automation/
    ├── workflows/
    ├── functions/
    └── triggers/
```

## 5. Concrete Implementation Specifications

### 5.1 Network Slice Package Generation (eMBB, uRLLC, mIoT)

#### 5.1.1 eMBB Package Specification

```yaml
# eMBB Network Slice Package
apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: embb-slice-template-v1
  namespace: porch-system
spec:
  packageName: embb-slice-template
  revision: v1
  repository: nephio-blueprints
  lifecycle: Published
  resources:
    - name: "slice-definition"
      content: |
        apiVersion: workload.nephio.org/v1alpha1
        kind: NetworkSlice
        metadata:
          name: embb-slice
        spec:
          sliceType: eMBB
          qosProfile:
            name: "embb-profile"
            maxBitRateUL: "100Mbps"
            maxBitRateDL: "1Gbps"
            latency: "20ms"
            packetLossRate: "1e-5"
            reliability: "99.9%"

          # RAN Function Requirements
          ranFunctions:
            - type: "gNB"
              vendor: "vendor-a"
              version: "v1.2.0"
              placement:
                clusters: ["edge-01", "edge-02"]
                antiAffinity: true
              resources:
                cpu: "4000m"
                memory: "8Gi"
                hugepages: "2Gi"
              interfaces:
                - name: "n2"
                  type: "external"
                  network: "n2-network"
                - name: "n3"
                  type: "external"
                  network: "n3-network"

          # Core Network Function Requirements
          coreNetworkFunctions:
            - type: "AMF"
              placement:
                clusters: ["regional-01"]
              resources:
                cpu: "2000m"
                memory: "4Gi"
            - type: "SMF"
              placement:
                clusters: ["regional-01"]
              resources:
                cpu: "1000m"
                memory: "2Gi"
            - type: "UPF"
              placement:
                clusters: ["edge-01", "edge-02"]
              resources:
                cpu: "8000m"
                memory: "16Gi"
                hugepages: "4Gi"

          # Service Level Objectives
          slo:
            availability: "99.99%"
            throughput:
              uplink: "100Mbps"
              downlink: "1Gbps"
            latency:
              max: "20ms"
              p99: "15ms"
            coverage:
              areas: ["london-central", "london-east"]
              mobility: "high"
```

#### 5.1.2 uRLLC Package Specification

```yaml
# uRLLC Network Slice Package
apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: urllc-slice-template-v1
  namespace: porch-system
spec:
  packageName: urllc-slice-template
  revision: v1
  repository: nephio-blueprints
  lifecycle: Published
  resources:
    - name: "slice-definition"
      content: |
        apiVersion: workload.nephio.org/v1alpha1
        kind: NetworkSlice
        metadata:
          name: urllc-slice
        spec:
          sliceType: uRLLC
          qosProfile:
            name: "urllc-profile"
            maxBitRateUL: "10Mbps"
            maxBitRateDL: "10Mbps"
            latency: "1ms"
            packetLossRate: "1e-7"
            reliability: "99.999%"
            jitter: "0.1ms"

          # Ultra-Low Latency RAN Configuration
          ranFunctions:
            - type: "gNB"
              vendor: "vendor-specialized"
              version: "v2.0.0"
              placement:
                clusters: ["edge-01"]
                nodeSelector:
                  hardware: "high-performance"
                  latency-optimized: "true"
              resources:
                cpu: "8000m"
                memory: "16Gi"
                hugepages: "8Gi"
              scheduling:
                priority: "system-critical"
                preemption: "never"
              interfaces:
                - name: "n2"
                  type: "external"
                  network: "n2-network-urllc"
                  qos: "dedicated"
                - name: "n3"
                  type: "external"
                  network: "n3-network-urllc"
                  qos: "dedicated"

          # Dedicated Core Network Functions
          coreNetworkFunctions:
            - type: "AMF"
              mode: "dedicated"
              placement:
                clusters: ["regional-01"]
                nodeSelector:
                  latency-zone: "ultra-low"
              resources:
                cpu: "4000m"
                memory: "8Gi"
            - type: "UPF"
              mode: "dedicated"
              placement:
                clusters: ["edge-01"]
                nodeSelector:
                  dpdk-enabled: "true"
              resources:
                cpu: "16000m"
                memory: "32Gi"
                hugepages: "16Gi"

          # Critical SLOs
          slo:
            availability: "99.999%"
            latency:
              max: "1ms"
              p99: "0.5ms"
              jitter: "0.1ms"
            reliability: "99.999%"
            processing:
              guaranteed: true
              isolation: "complete"
```

#### 5.1.3 mIoT Package Specification

```yaml
# mIoT Network Slice Package
apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: miot-slice-template-v1
  namespace: porch-system
spec:
  packageName: miot-slice-template
  revision: v1
  repository: nephio-blueprints
  lifecycle: Published
  resources:
    - name: "slice-definition"
      content: |
        apiVersion: workload.nephio.org/v1alpha1
        kind: NetworkSlice
        metadata:
          name: miot-slice
        spec:
          sliceType: mIoT
          qosProfile:
            name: "miot-profile"
            maxBitRateUL: "1Mbps"
            maxBitRateDL: "1Mbps"
            latency: "100ms"
            packetLossRate: "1e-3"
            reliability: "99%"
            connectionDensity: "1000000/km2"

          # Massive Connectivity RAN Configuration
          ranFunctions:
            - type: "gNB"
              vendor: "vendor-iot"
              version: "v1.5.0"
              placement:
                clusters: ["edge-01", "edge-02"]
                distribution: "spread"
              resources:
                cpu: "2000m"
                memory: "4Gi"
              scaling:
                enabled: true
                minReplicas: 2
                maxReplicas: 10
                connectionBasedScaling: true
              interfaces:
                - name: "n2"
                  type: "shared"
                  network: "n2-network"
                - name: "n3"
                  type: "shared"
                  network: "n3-network"

          # Shared Core Network Functions
          coreNetworkFunctions:
            - type: "AMF"
              mode: "shared"
              placement:
                clusters: ["regional-01"]
              scaling:
                enabled: true
                metric: "connection-count"
              resources:
                cpu: "1000m"
                memory: "2Gi"
            - type: "SMF"
              mode: "shared"
              placement:
                clusters: ["regional-01"]
              resources:
                cpu: "500m"
                memory: "1Gi"

          # Optimized SLOs for mIoT
          slo:
            availability: "99%"
            connectionDensity: "1000000/km2"
            latency:
              max: "100ms"
              tolerance: "high"
            energyEfficiency: "optimized"
            batteryLife: "10years"
```

### 5.2 Multi-Site Deployment Coordination

#### 5.2.1 Site Coordination Controller

```yaml
# Multi-Site Coordination Controller
apiVersion: apps/v1
kind: Deployment
metadata:
  name: multi-site-coordinator
  namespace: nephio-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: multi-site-coordinator
  template:
    metadata:
      labels:
        app: multi-site-coordinator
    spec:
      containers:
      - name: coordinator
        image: gcr.io/oran-mano/multi-site-coordinator:v1.0.0
        env:
        - name: SITES_CONFIG
          valueFrom:
            configMapKeyRef:
              name: sites-configuration
              key: sites.yaml
        - name: COORDINATION_STRATEGY
          value: "hierarchical"
        ports:
        - containerPort: 8080
        - containerPort: 9090  # Metrics
        volumeMounts:
        - name: site-configs
          mountPath: /etc/sites
      volumes:
      - name: site-configs
        configMap:
          name: sites-configuration
---
# Sites Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: sites-configuration
  namespace: nephio-system
data:
  sites.yaml: |
    sites:
      central:
        - name: "central-01"
          type: "management"
          location:
            region: "london"
            zone: "west"
            coordinates: [51.5074, -0.1278]
          clusters:
            - name: "kind-central-01"
              kubeconfig: "/etc/kubeconfigs/central-01"
          capabilities:
            - "management-plane"
            - "orchestration"
            - "analytics"
            - "ml-workloads"
          resources:
            total:
              cpu: "128"
              memory: "512Gi"
              storage: "100Ti"
            available:
              cpu: "64"
              memory: "256Gi"
              storage: "50Ti"

      regional:
        - name: "regional-01"
          type: "aggregation"
          location:
            region: "london"
            zone: "central"
            coordinates: [51.5156, -0.0919]
          clusters:
            - name: "kind-regional-01"
              kubeconfig: "/etc/kubeconfigs/regional-01"
          capabilities:
            - "core-network"
            - "edge-orchestration"
            - "regional-analytics"
          resources:
            total:
              cpu: "64"
              memory: "128Gi"
              storage: "10Ti"

      edge:
        - name: "edge-01"
          type: "access"
          location:
            region: "london"
            zone: "east"
            coordinates: [51.5074, 0.0278]
          clusters:
            - name: "kind-edge-01"
              kubeconfig: "/etc/kubeconfigs/edge-01"
          capabilities:
            - "ran-functions"
            - "edge-computing"
            - "local-breakout"
          resources:
            total:
              cpu: "32"
              memory: "64Gi"
              storage: "2Ti"

        - name: "edge-02"
          type: "access"
          location:
            region: "london"
            zone: "south"
            coordinates: [51.4074, -0.1278]
          clusters:
            - name: "kind-edge-02"
              kubeconfig: "/etc/kubeconfigs/edge-02"
          capabilities:
            - "ran-functions"
            - "edge-computing"
            - "local-breakout"

    coordination:
      deployment_strategies:
        - name: "ran-placement"
          pattern: "edge-only"
          function_types: ["gNB", "CU", "DU"]
          constraints:
            latency: "<5ms"
            bandwidth: ">10Gbps"

        - name: "core-placement"
          pattern: "regional-primary"
          function_types: ["AMF", "SMF", "AUSF", "NRF"]
          constraints:
            availability: ">99.9%"
            redundancy: "active-standby"

        - name: "upf-placement"
          pattern: "distributed"
          function_types: ["UPF"]
          constraints:
            latency: "<10ms"
            throughput: ">1Gbps"

      synchronization:
        method: "gitops"
        interval: "30s"
        conflict_resolution: "central-authority"
        rollback_strategy: "cascade"
```

### 5.3 O2ims/O2dms Integration Patterns

#### 5.3.1 O2ims Integration for Inventory Management

```yaml
# O2ims Resource Pool Discovery
apiVersion: o2ims.o-ran.org/v1alpha1
kind: ResourcePool
metadata:
  name: london-edge-pool
  namespace: o2ims-system
spec:
  poolId: "rp-london-edge-001"
  description: "London Edge Computing Resource Pool"
  location:
    region: "london"
    availabilityZone: "east"
  resources:
    - resourceTypeId: "rt-compute-001"
      resourceTypeName: "EdgeComputeNode"
      capacity:
        cpu: "32"
        memory: "64Gi"
        storage: "2Ti"
      allocated:
        cpu: "16"
        memory: "32Gi"
        storage: "1Ti"
    - resourceTypeId: "rt-network-001"
      resourceTypeName: "NetworkInterface"
      capacity:
        bandwidth: "25Gbps"
        ports: 8
      allocated:
        bandwidth: "10Gbps"
        ports: 4
  deploymentManagers:
    - dmId: "dm-nephio-001"
      dmName: "nephio-edge-dm"
      endpointUri: "https://nephio-dm.edge-01.local"
  globalCloudId: "gc-london-001"
---
# O2ims Subscription for Resource Updates
apiVersion: o2ims.o-ran.org/v1alpha1
kind: Subscription
metadata:
  name: nephio-resource-updates
  namespace: o2ims-system
spec:
  subscriptionId: "sub-nephio-001"
  consumerSubscriptionId: "nephio-orchestrator-001"
  filter: |
    {
      "resourceTypes": ["EdgeComputeNode", "NetworkInterface"],
      "regions": ["london"],
      "events": ["ResourceCreated", "ResourceUpdated", "ResourceDeleted"]
    }
  callbackUri: "https://nephio-orchestrator.central-01.local/o2ims/callback"
  authentication:
    authType: "OAUTH2_CLIENT_CREDENTIALS"
    secretRef:
      name: "o2ims-auth-secret"
```

#### 5.3.2 O2dms Integration for Deployment Management

```yaml
# O2dms Deployment Manager for Nephio
apiVersion: o2dms.o-ran.org/v1alpha1
kind: DeploymentManager
metadata:
  name: nephio-deployment-manager
  namespace: o2dms-system
spec:
  deploymentManagerId: "dm-nephio-001"
  name: "Nephio Multi-Site DM"
  description: "Deployment manager for Nephio-based network functions"
  endpointUri: "https://nephio-dm.central-01.local"
  supportedResourceTypes:
    - "NetworkFunction"
    - "NetworkSlice"
    - "RANFunction"
    - "CoreFunction"
  capabilities:
    - "lifecycle-management"
    - "configuration-management"
    - "fault-management"
    - "performance-management"
  extensions:
    porch:
      enabled: true
      repositoryUrl: "https://github.com/oran-mano/nephio-packages"
    configSync:
      enabled: true
      rootSyncRepo: "https://github.com/oran-mano/nephio-deployments"
---
# NF Deployment Request via O2dms
apiVersion: o2dms.o-ran.org/v1alpha1
kind: NFDeploymentRequest
metadata:
  name: gnb-deployment-001
  namespace: o2dms-system
spec:
  nfDeploymentRequestId: "nfdr-gnb-001"
  deploymentManagerId: "dm-nephio-001"
  name: "gNB-EdgeSite-001"
  description: "Deploy gNB at edge site 001"
  nfPackageId: "pkg-gnb-vendor-a-v1.2.0"
  locationConstraints:
    - region: "london"
      zone: "east"
      site: "edge-01"
  resourceRequirements:
    cpu: "4000m"
    memory: "8Gi"
    storage: "100Gi"
    hugepages: "2Gi"
  connectivityRequirements:
    - interfaceName: "n2"
      networkName: "n2-network"
      ipVersion: "IPv4"
    - interfaceName: "n3"
      networkName: "n3-network"
      ipVersion: "IPv4"
  additionalParams:
    sliceId: "slice-001"
    qosProfile: "embb-profile"
    vendor: "vendor-a"
    customizations:
      cellId: "001"
      plmnId: "12345"
      frequency: "3.5GHz"
```

### 5.4 Production-Ready YAML Manifests

#### 5.4.1 Porch Server Configuration

```yaml
# Porch Server Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: porch-server
  namespace: porch-system
  labels:
    app: porch-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: porch-server
  template:
    metadata:
      labels:
        app: porch-server
    spec:
      serviceAccountName: porch-server
      containers:
      - name: porch-server
        image: gcr.io/kpt-dev/porch-server:v0.0.33
        args:
        - --secure-port=8443
        - --kubeconfig=/etc/kubeconfig/config
        - --authorization-kubeconfig=/etc/kubeconfig/config
        - --authentication-kubeconfig=/etc/kubeconfig/config
        - --audit-log-path=/var/log/audit.log
        - --audit-log-maxage=30
        - --audit-log-maxbackup=3
        - --audit-log-maxsize=100
        - --tls-cert-file=/etc/certs/tls.crt
        - --tls-private-key-file=/etc/certs/tls.key
        ports:
        - containerPort: 8443
          name: webhook-api
        resources:
          limits:
            cpu: 1000m
            memory: 1Gi
          requests:
            cpu: 100m
            memory: 128Mi
        volumeMounts:
        - name: kubeconfig
          mountPath: /etc/kubeconfig
          readOnly: true
        - name: certs
          mountPath: /etc/certs
          readOnly: true
        - name: audit-logs
          mountPath: /var/log
        env:
        - name: GIT_AUTHOR_NAME
          value: "Porch Server"
        - name: GIT_AUTHOR_EMAIL
          value: "porch@oran-mano.org"
        - name: GIT_COMMITTER_NAME
          value: "Porch Server"
        - name: GIT_COMMITTER_EMAIL
          value: "porch@oran-mano.org"
        livenessProbe:
          httpGet:
            path: /livez
            port: webhook-api
            scheme: HTTPS
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: webhook-api
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: kubeconfig
        secret:
          secretName: porch-server-kubeconfig
      - name: certs
        secret:
          secretName: porch-server-certs
      - name: audit-logs
        emptyDir: {}
---
# Porch Server Service
apiVersion: v1
kind: Service
metadata:
  name: porch-server
  namespace: porch-system
spec:
  type: ClusterIP
  ports:
  - port: 443
    targetPort: webhook-api
    protocol: TCP
    name: webhook-api
  selector:
    app: porch-server
---
# Porch Server RBAC
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: porch-server
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets", "events"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["config.porch.kpt.dev"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["porch.kpt.dev"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["gitops.kpt.dev"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: porch-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: porch-server
subjects:
- kind: ServiceAccount
  name: porch-server
  namespace: porch-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: porch-server
  namespace: porch-system
```

#### 5.4.2 ConfigSync Multi-Cluster Setup

```yaml
# Config Management Operator
apiVersion: v1
kind: Namespace
metadata:
  name: config-management-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: config-management-operator
  namespace: config-management-system
spec:
  replicas: 1
  selector:
    matchLabels:
      name: config-management-operator
  template:
    metadata:
      labels:
        name: config-management-operator
    spec:
      serviceAccountName: config-management-operator
      containers:
      - name: manager
        image: gcr.io/config-management-release/config-management-operator:1.15.1
        command:
        - /manager
        args:
        - --enable-leader-election
        env:
        - name: WATCH_NAMESPACE
          value: ""
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: OPERATOR_NAME
          value: "config-management-operator"
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 100m
            memory: 128Mi
        ports:
        - containerPort: 8080
          name: metrics
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
---
# Root Sync for Central Management
apiVersion: configsync.gke.io/v1beta1
kind: RootSync
metadata:
  name: root-sync
  namespace: config-management-system
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-deployments
    branch: main
    dir: /clusters/central/central-01
    auth: ssh
    secretRef:
      name: git-creds
    noSSLVerify: false
  override:
    resources:
    - group: ""
      kind: "Namespace"
      operations: ["*"]
    - group: "workload.nephio.org"
      kind: "*"
      operations: ["*"]
    - group: "o2ims.o-ran.org"
      kind: "*"
      operations: ["*"]
    - group: "o2dms.o-ran.org"
      kind: "*"
      operations: ["*"]
  renderingRequired: true
---
# Repo Sync for Edge Workloads
apiVersion: configsync.gke.io/v1beta1
kind: RepoSync
metadata:
  name: edge-workloads-sync
  namespace: nephio-workloads
spec:
  sourceFormat: unstructured
  git:
    repo: https://github.com/oran-mano/nephio-deployments
    branch: main
    dir: /clusters/edge/edge-01/workloads
    auth: ssh
    secretRef:
      name: git-creds
    noSSLVerify: false
  override:
    namespaceStrategy: explicit
    resources:
    - group: "workload.nephio.org"
      kind: "NetworkFunction"
      operations: ["create", "update", "patch"]
      namespaces: ["ran-workloads"]
    - group: "workload.nephio.org"
      kind: "RANFunction"
      operations: ["create", "update", "patch"]
      namespaces: ["ran-workloads"]
  renderingRequired: true
  hydrationController:
    enabled: true
    source:
      path: /blueprints/edge
```

#### 5.4.3 Monitoring and Observability

```yaml
# Prometheus Configuration for Porch/ConfigSync
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    rule_files:
      - "/etc/prometheus/rules/*.yml"

    scrape_configs:
    - job_name: 'porch-server'
      static_configs:
      - targets: ['porch-server.porch-system.svc.cluster.local:8080']
      metrics_path: /metrics
      scrape_interval: 30s

    - job_name: 'config-management-operator'
      static_configs:
      - targets: ['config-management-operator.config-management-system.svc.cluster.local:8080']
      metrics_path: /metrics
      scrape_interval: 30s

    - job_name: 'nephio-orchestrator'
      static_configs:
      - targets: ['nephio-orchestrator.nephio-system.svc.cluster.local:9090']
      metrics_path: /metrics
      scrape_interval: 15s

    - job_name: 'network-functions'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          - ran-workloads
          - core-workloads
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
---
# Grafana Dashboard for GitOps Metrics
apiVersion: v1
kind: ConfigMap
metadata:
  name: gitops-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  gitops-metrics.json: |
    {
      "dashboard": {
        "id": null,
        "title": "GitOps Metrics",
        "tags": ["gitops", "porch", "configsync"],
        "timezone": "browser",
        "panels": [
          {
            "id": 1,
            "title": "Package Revisions by Lifecycle",
            "type": "stat",
            "targets": [
              {
                "expr": "count by (lifecycle) (porch_package_revisions)",
                "legendFormat": "{{lifecycle}}"
              }
            ],
            "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
          },
          {
            "id": 2,
            "title": "ConfigSync Sync Status",
            "type": "stat",
            "targets": [
              {
                "expr": "config_sync_status",
                "legendFormat": "{{cluster}}"
              }
            ],
            "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
          },
          {
            "id": 3,
            "title": "Deployment Success Rate",
            "type": "graph",
            "targets": [
              {
                "expr": "rate(deployment_success_total[5m]) / rate(deployment_attempts_total[5m]) * 100",
                "legendFormat": "Success Rate %"
              }
            ],
            "gridPos": {"h": 8, "w": 24, "x": 0, "y": 8}
          },
          {
            "id": 4,
            "title": "Time to Deploy (P95)",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, deployment_duration_seconds_bucket)",
                "legendFormat": "P95 Deploy Time"
              }
            ],
            "gridPos": {"h": 8, "w": 24, "x": 0, "y": 16}
          }
        ],
        "time": {
          "from": "now-1h",
          "to": "now"
        },
        "refresh": "30s"
      }
    }
```

## 6. Performance Optimization for <10 Minute Deployment Target

### 6.1 Deployment Pipeline Optimization

```yaml
# Fast-Track Deployment Pipeline
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: fast-track-deployment
  namespace: nephio-system
spec:
  entrypoint: optimized-deployment
  templates:
  - name: optimized-deployment
    dag:
      tasks:
      # Parallel validation and preparation (0-2 minutes)
      - name: validate-intent
        template: fast-validation
        arguments:
          parameters:
          - name: intent
            value: "{{workflow.parameters.intent}}"

      - name: pre-cache-images
        template: image-precaching
        arguments:
          parameters:
          - name: required-images
            value: "{{workflow.parameters.required-images}}"

      - name: prepare-resources
        template: resource-preparation
        arguments:
          parameters:
          - name: target-clusters
            value: "{{workflow.parameters.target-clusters}}"

      # Package generation (2-4 minutes)
      - name: generate-packages
        template: optimized-package-generation
        dependencies: [validate-intent]
        arguments:
          parameters:
          - name: validated-intent
            value: "{{tasks.validate-intent.outputs.parameters.result}}"

      # Parallel distribution (4-7 minutes)
      - name: distribute-to-edge
        template: parallel-distribution
        dependencies: [generate-packages, prepare-resources]
        arguments:
          parameters:
          - name: packages
            value: "{{tasks.generate-packages.outputs.parameters.packages}}"
          - name: target-type
            value: "edge"

      - name: distribute-to-regional
        template: parallel-distribution
        dependencies: [generate-packages, prepare-resources]
        arguments:
          parameters:
          - name: packages
            value: "{{tasks.generate-packages.outputs.parameters.packages}}"
          - name: target-type
            value: "regional"

      # Coordinated deployment (7-10 minutes)
      - name: deploy-core-first
        template: coordinated-deployment
        dependencies: [distribute-to-regional]
        arguments:
          parameters:
          - name: deployment-order
            value: "core-first"

      - name: deploy-ran-parallel
        template: coordinated-deployment
        dependencies: [deploy-core-first, distribute-to-edge, pre-cache-images]
        arguments:
          parameters:
          - name: deployment-order
            value: "ran-parallel"

      # Validation and completion (9-10 minutes)
      - name: validate-deployment
        template: fast-validation-check
        dependencies: [deploy-ran-parallel]

      - name: enable-traffic
        template: traffic-enablement
        dependencies: [validate-deployment]

  - name: fast-validation
    container:
      image: gcr.io/oran-mano/fast-validator:v1.0.0
      command: ["/app/validate"]
      args: ["--mode", "fast", "--timeout", "30s"]
      resources:
        requests:
          cpu: "500m"
          memory: "512Mi"
        limits:
          cpu: "1000m"
          memory: "1Gi"

  - name: optimized-package-generation
    container:
      image: gcr.io/oran-mano/optimized-generator:v1.0.0
      command: ["/app/generate"]
      args: ["--parallel", "true", "--cache-enabled", "true"]
      resources:
        requests:
          cpu: "2000m"
          memory: "4Gi"
        limits:
          cpu: "4000m"
          memory: "8Gi"
```

### 6.2 Resource Pre-warming Strategy

```yaml
# Resource Pre-warming Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: pre-warming-config
  namespace: nephio-system
data:
  config.yaml: |
    preWarming:
      images:
        - name: "gnb-base"
          image: "gcr.io/oran-mano/gnb:v1.2.0"
          clusters: ["edge-01", "edge-02"]
          priority: "high"
        - name: "amf-base"
          image: "gcr.io/oran-mano/amf:v1.1.0"
          clusters: ["regional-01"]
          priority: "high"
        - name: "upf-base"
          image: "gcr.io/oran-mano/upf:v1.3.0"
          clusters: ["edge-01", "edge-02"]
          priority: "medium"

      resources:
        nodes:
          - cluster: "edge-01"
            prepare:
              - "hugepages-2Mi=2Gi"
              - "cpu-manager-policy=static"
              - "topology-manager-policy=best-effort"
          - cluster: "regional-01"
            prepare:
              - "network-bandwidth=25Gbps"
              - "storage-class=high-iops"

      networks:
        - name: "n2-network"
          type: "macvlan"
          clusters: ["edge-01", "edge-02", "regional-01"]
          prepare: true
        - name: "n3-network"
          type: "ipvlan"
          clusters: ["edge-01", "edge-02"]
          prepare: true

      scheduling:
        enabled: true
        trigger: "intent-received"
        parallel: true
        timeout: "60s"
---
# Pre-warming Job
apiVersion: batch/v1
kind: Job
metadata:
  name: resource-pre-warmer
  namespace: nephio-system
spec:
  template:
    spec:
      containers:
      - name: pre-warmer
        image: gcr.io/oran-mano/pre-warmer:v1.0.0
        command: ["/app/pre-warm"]
        args: ["--config", "/etc/config/config.yaml"]
        volumeMounts:
        - name: config
          mountPath: /etc/config
        env:
        - name: PARALLEL_JOBS
          value: "10"
        - name: TIMEOUT
          value: "60s"
      volumes:
      - name: config
        configMap:
          name: pre-warming-config
      restartPolicy: OnFailure
  backoffLimit: 3
```

## Conclusion

This comprehensive specification document provides the foundation for implementing Porch and ConfigSync integration within the O-RAN Intent-Based MANO system. The specifications include:

1. **Complete Porch API Integration** with lifecycle management, validation, and automated promotion
2. **Multi-cluster ConfigSync patterns** with hierarchical distribution and drift detection
3. **End-to-end workflows** from intent processing to deployment
4. **Production-ready configurations** for all components
5. **Performance optimizations** targeting sub-10-minute deployment times

The specifications are designed to be:
- **Production-ready** with comprehensive error handling and monitoring
- **Scalable** across multiple sites and clusters
- **Secure** with proper RBAC and network isolation
- **Observable** with detailed metrics and logging
- **Fast** with parallel processing and resource pre-warming

Implementation teams can use these specifications as blueprints for building a robust, automated network slice orchestration system that meets the demanding requirements of modern O-RAN deployments.