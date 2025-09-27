# Prometheus Monitoring Stack Configuration
# This file contains the Terraform configuration for the Prometheus monitoring stack

# Install Prometheus Operator using Helm
resource "helm_release" "prometheus_operator" {
  name       = "prometheus-operator"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = var.prometheus_operator_version
  namespace  = kubernetes_namespace.monitoring.metadata[0].name

  create_namespace = false
  wait             = true
  timeout          = 1200

  values = [
    templatefile("${path.module}/values/prometheus-values.yaml", {
      environment             = var.environment
      prometheus_retention    = var.prometheus_retention
      prometheus_storage_size = var.prometheus_storage_size
      grafana_admin_password  = var.grafana_admin_password
      alertmanager_enabled    = var.alertmanager_enabled
      storage_class          = var.create_storage_class ? kubernetes_storage_class.monitoring_storage[0].metadata[0].name : var.storage_class_name
      enable_ingress         = var.enable_ingress
      ingress_domain         = var.ingress_domain
    })
  ]

  set {
    name  = "prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.ruleSelectorNilUsesHelmValues"
    value = "false"
  }

  # Storage configuration
  set {
    name  = "prometheus.prometheusSpec.retention"
    value = var.prometheus_retention
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage"
    value = var.prometheus_storage_size
  }

  # Grafana configuration
  set {
    name  = "grafana.adminPassword"
    value = var.grafana_admin_password
    type  = "string"
  }

  set {
    name  = "grafana.service.type"
    value = var.grafana_service_type
  }

  set {
    name  = "grafana.persistence.enabled"
    value = var.grafana_persistence_enabled
  }

  set {
    name  = "grafana.persistence.size"
    value = var.grafana_storage_size
  }

  # AlertManager configuration
  set {
    name  = "alertmanager.enabled"
    value = var.alertmanager_enabled
  }

  set {
    name  = "alertmanager.alertmanagerSpec.retention"
    value = var.alertmanager_retention
  }

  set {
    name  = "alertmanager.alertmanagerSpec.storage.volumeClaimTemplate.spec.resources.requests.storage"
    value = var.alertmanager_storage_size
  }

  # Node Exporter configuration
  set {
    name  = "nodeExporter.enabled"
    value = var.node_exporter_enabled
  }

  # kube-state-metrics configuration
  set {
    name  = "kubeStateMetrics.enabled"
    value = var.kube_state_metrics_enabled
  }

  # Prometheus Node Exporter configuration for O-RAN specific metrics
  set {
    name  = "prometheus-node-exporter.extraArgs"
    value = "{--collector.systemd,--collector.processes,--collector.tcpstat}"
  }

  depends_on = [
    kubernetes_namespace.monitoring,
    helm_release.cert_manager
  ]
}

# Create ServiceMonitors for O-RAN components
resource "kubectl_manifest" "vnf_operator_servicemonitor" {
  yaml_body = <<YAML
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: vnf-operator
  namespace: ${kubernetes_namespace.monitoring.metadata[0].name}
  labels:
    app: vnf-operator
    prometheus: kube-prometheus
    oran.io/component: vnf-operator
spec:
  selector:
    matchLabels:
      app: vnf-operator
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    scheme: http
  - port: http-metrics
    interval: 30s
    path: /metrics
    scheme: http
  namespaceSelector:
    matchNames:
    - ${var.oran_namespace}
    - vnf-operator-system
YAML

  depends_on = [helm_release.prometheus_operator]
}

resource "kubectl_manifest" "intent_management_servicemonitor" {
  yaml_body = <<YAML
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: intent-management
  namespace: ${kubernetes_namespace.monitoring.metadata[0].name}
  labels:
    app: intent-management
    prometheus: kube-prometheus
    oran.io/component: intent-management
spec:
  selector:
    matchLabels:
      app: intent-management
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    scheme: http
  - port: http-metrics
    interval: 30s
    path: /metrics
    scheme: http
  namespaceSelector:
    matchNames:
    - ${var.oran_namespace}
    - intent-management-system
YAML

  depends_on = [helm_release.prometheus_operator]
}

resource "kubectl_manifest" "oran_components_servicemonitor" {
  yaml_body = <<YAML
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-components
  namespace: ${kubernetes_namespace.monitoring.metadata[0].name}
  labels:
    app: oran-components
    prometheus: kube-prometheus
    oran.io/component: monitoring
spec:
  selector:
    matchLabels:
      oran.io/monitoring: "enabled"
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
    scheme: http
  - port: http-metrics
    interval: 30s
    path: /metrics
    scheme: http
  namespaceSelector: {}
YAML

  depends_on = [helm_release.prometheus_operator]
}

# Create PrometheusRules for O-RAN specific alerting
resource "kubectl_manifest" "oran_alerting_rules" {
  yaml_body = <<YAML
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: oran-mano-rules
  namespace: ${kubernetes_namespace.monitoring.metadata[0].name}
  labels:
    app: oran-mano
    prometheus: kube-prometheus
    role: alert-rules
    oran.io/component: alerting
spec:
  groups:
  - name: oran-mano.rules
    interval: 30s
    rules:
    - alert: ORANComponentDown
      expr: up{job=~"vnf-operator|intent-management"} == 0
      for: 5m
      labels:
        severity: critical
        component: oran-mano
      annotations:
        summary: "O-RAN component {{ $labels.job }} is down"
        description: "O-RAN component {{ $labels.job }} on instance {{ $labels.instance }} has been down for more than 5 minutes."

    - alert: VNFOperatorHighErrorRate
      expr: rate(vnf_operator_errors_total[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
        component: vnf-operator
      annotations:
        summary: "VNF Operator high error rate"
        description: "VNF Operator is experiencing high error rate: {{ $value }} errors/second"

    - alert: IntentManagementHighLatency
      expr: histogram_quantile(0.95, rate(intent_management_request_duration_seconds_bucket[5m])) > 0.5
      for: 5m
      labels:
        severity: warning
        component: intent-management
      annotations:
        summary: "Intent Management high latency"
        description: "Intent Management 95th percentile latency is {{ $value }}s"

    - alert: PrometheusConfigReloadFailed
      expr: prometheus_config_last_reload_successful == 0
      for: 10m
      labels:
        severity: critical
        component: prometheus
      annotations:
        summary: "Prometheus configuration reload failed"
        description: "Prometheus configuration reload has been failing for {{ $labels.instance }}"

    - alert: AlertmanagerConfigInconsistent
      expr: count by (service) (alertmanager_config_hash{service="alertmanager-operated"}) > 1
      for: 5m
      labels:
        severity: critical
        component: alertmanager
      annotations:
        summary: "Alertmanager configuration inconsistent"
        description: "Alertmanager configuration is inconsistent across replicas"

  - name: oran-performance.rules
    interval: 30s
    rules:
    - alert: HighMemoryUsage
      expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100 > 80
      for: 5m
      labels:
        severity: warning
        component: infrastructure
      annotations:
        summary: "High memory usage on node {{ $labels.instance }}"
        description: "Memory usage is above 80% on node {{ $labels.instance }}: {{ $value }}%"

    - alert: HighCPUUsage
      expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
      for: 5m
      labels:
        severity: warning
        component: infrastructure
      annotations:
        summary: "High CPU usage on node {{ $labels.instance }}"
        description: "CPU usage is above 80% on node {{ $labels.instance }}: {{ $value }}%"

    - alert: PodCrashLooping
      expr: rate(kube_pod_container_status_restarts_total[15m]) > 0
      for: 5m
      labels:
        severity: critical
        component: "{{ $labels.container }}"
      annotations:
        summary: "Pod {{ $labels.pod }} is crash looping"
        description: "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} is restarting frequently"
YAML

  depends_on = [helm_release.prometheus_operator]
}

# Create Grafana dashboards for O-RAN components
resource "kubernetes_config_map" "oran_grafana_dashboards" {
  metadata {
    name      = "oran-grafana-dashboards"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels = merge(local.monitoring_labels, {
      "grafana_dashboard" = "1"
    })
  }

  data = {
    "oran-overview.json" = file("${path.module}/dashboards/oran-overview.json")
    "vnf-operator.json"  = file("${path.module}/dashboards/vnf-operator.json")
    "intent-management.json" = file("${path.module}/dashboards/intent-management.json")
    "infrastructure.json" = file("${path.module}/dashboards/infrastructure.json")
  }

  depends_on = [helm_release.prometheus_operator]
}

# Create ingress for monitoring services (if enabled)
resource "kubernetes_ingress_v1" "prometheus_ingress" {
  count = var.enable_ingress ? 1 : 0

  metadata {
    name      = "prometheus-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
    annotations = {
      "kubernetes.io/ingress.class"                = "nginx"
      "nginx.ingress.kubernetes.io/rewrite-target" = "/"
      "cert-manager.io/cluster-issuer"             = "letsencrypt-prod"
    }
  }

  spec {
    tls {
      hosts       = ["prometheus.${var.ingress_domain}"]
      secret_name = "prometheus-tls"
    }

    rule {
      host = "prometheus.${var.ingress_domain}"
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = "prometheus-operator-kube-p-prometheus"
              port {
                number = 9090
              }
            }
          }
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_operator]
}

resource "kubernetes_ingress_v1" "grafana_ingress" {
  count = var.enable_ingress ? 1 : 0

  metadata {
    name      = "grafana-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
    annotations = {
      "kubernetes.io/ingress.class"                = "nginx"
      "nginx.ingress.kubernetes.io/rewrite-target" = "/"
      "cert-manager.io/cluster-issuer"             = "letsencrypt-prod"
    }
  }

  spec {
    tls {
      hosts       = ["grafana.${var.ingress_domain}"]
      secret_name = "grafana-tls"
    }

    rule {
      host = "grafana.${var.ingress_domain}"
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = "prometheus-operator-grafana"
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_operator]
}

resource "kubernetes_ingress_v1" "alertmanager_ingress" {
  count = var.enable_ingress && var.alertmanager_enabled ? 1 : 0

  metadata {
    name      = "alertmanager-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
    annotations = {
      "kubernetes.io/ingress.class"                = "nginx"
      "nginx.ingress.kubernetes.io/rewrite-target" = "/"
      "cert-manager.io/cluster-issuer"             = "letsencrypt-prod"
    }
  }

  spec {
    tls {
      hosts       = ["alertmanager.${var.ingress_domain}"]
      secret_name = "alertmanager-tls"
    }

    rule {
      host = "alertmanager.${var.ingress_domain}"
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = "prometheus-operator-kube-p-alertmanager"
              port {
                number = 9093
              }
            }
          }
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_operator]
}

# Output monitoring service information
output "prometheus_service" {
  description = "Prometheus service information"
  value = {
    name      = "prometheus-operator-kube-p-prometheus"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    port      = 9090
    url       = var.enable_ingress ? "https://prometheus.${var.ingress_domain}" : "http://localhost:9090"
  }
}

output "grafana_service" {
  description = "Grafana service information"
  value = {
    name      = "prometheus-operator-grafana"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    port      = 80
    url       = var.enable_ingress ? "https://grafana.${var.ingress_domain}" : "http://localhost:3000"
    username  = "admin"
    password  = var.grafana_admin_password
  }
}

output "alertmanager_service" {
  description = "AlertManager service information"
  value = var.alertmanager_enabled ? {
    name      = "prometheus-operator-kube-p-alertmanager"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    port      = 9093
    url       = var.enable_ingress ? "https://alertmanager.${var.ingress_domain}" : "http://localhost:9093"
  } : null
}