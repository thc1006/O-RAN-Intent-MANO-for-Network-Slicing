# Terraform Outputs for O-RAN MANO Infrastructure
# This file defines outputs for the deployed infrastructure

# Namespace outputs
output "namespaces" {
  description = "Created namespaces"
  value = {
    monitoring = kubernetes_namespace.monitoring.metadata[0].name
    oran       = kubernetes_namespace.oran.metadata[0].name
  }
}

# Service endpoints
output "service_endpoints" {
  description = "Service endpoints for monitoring components"
  value = {
    prometheus = {
      name         = "prometheus-operator-kube-p-prometheus"
      namespace    = kubernetes_namespace.monitoring.metadata[0].name
      port         = 9090
      internal_url = "http://prometheus-operator-kube-p-prometheus.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9090"
      external_url = var.enable_ingress ? "https://prometheus.${var.ingress_domain}" : null
    }
    grafana = {
      name         = "prometheus-operator-grafana"
      namespace    = kubernetes_namespace.monitoring.metadata[0].name
      port         = 80
      internal_url = "http://prometheus-operator-grafana.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:80"
      external_url = var.enable_ingress ? "https://grafana.${var.ingress_domain}" : null
    }
    alertmanager = var.alertmanager_enabled ? {
      name         = "prometheus-operator-kube-p-alertmanager"
      namespace    = kubernetes_namespace.monitoring.metadata[0].name
      port         = 9093
      internal_url = "http://prometheus-operator-kube-p-alertmanager.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9093"
      external_url = var.enable_ingress ? "https://alertmanager.${var.ingress_domain}" : null
    } : null
  }
}

# Access credentials
output "credentials" {
  description = "Access credentials for monitoring services"
  value = {
    grafana = {
      username = "admin"
      password = var.grafana_admin_password
    }
  }
  sensitive = true
}

# Port forward commands
output "port_forward_commands" {
  description = "Commands to port-forward to monitoring services"
  value = {
    prometheus   = "kubectl port-forward -n ${kubernetes_namespace.monitoring.metadata[0].name} svc/prometheus-operator-kube-p-prometheus 9090:9090"
    grafana      = "kubectl port-forward -n ${kubernetes_namespace.monitoring.metadata[0].name} svc/prometheus-operator-grafana 3000:80"
    alertmanager = var.alertmanager_enabled ? "kubectl port-forward -n ${kubernetes_namespace.monitoring.metadata[0].name} svc/prometheus-operator-kube-p-alertmanager 9093:9093" : null
  }
}

# Configuration information
output "configuration" {
  description = "Configuration details for the deployed infrastructure"
  value = {
    environment            = var.environment
    prometheus_retention   = var.prometheus_retention
    prometheus_storage     = var.prometheus_storage_size
    grafana_persistence    = var.grafana_persistence_enabled
    alertmanager_enabled   = var.alertmanager_enabled
    ingress_enabled        = var.enable_ingress
    network_policies       = var.enable_network_policies
    storage_class          = var.create_storage_class ? kubernetes_storage_class.monitoring_storage[0].metadata[0].name : var.storage_class_name
  }
}

# Storage information
output "storage" {
  description = "Storage configuration and details"
  value = {
    storage_class = var.create_storage_class ? {
      name                = kubernetes_storage_class.monitoring_storage[0].metadata[0].name
      provisioner         = var.storage_provisioner
      parameters          = var.storage_parameters
      reclaim_policy      = "Retain"
      volume_binding_mode = "WaitForFirstConsumer"
    } : null
    prometheus_storage_size   = var.prometheus_storage_size
    grafana_storage_size      = var.grafana_storage_size
    alertmanager_storage_size = var.alertmanager_storage_size
  }
}

# ServiceMonitors
output "service_monitors" {
  description = "Created ServiceMonitors for O-RAN components"
  value = [
    "vnf-operator",
    "intent-management",
    "oran-components"
  ]
}

# RBAC information
output "rbac" {
  description = "RBAC resources created"
  value = {
    service_account = kubernetes_service_account.monitoring_operator.metadata[0].name
    cluster_role    = kubernetes_cluster_role.monitoring_operator.metadata[0].name
    cluster_role_binding = kubernetes_cluster_role_binding.monitoring_operator.metadata[0].name
  }
}

# Helm release information
output "helm_releases" {
  description = "Deployed Helm releases"
  value = {
    prometheus_operator = {
      name      = helm_release.prometheus_operator.name
      namespace = helm_release.prometheus_operator.namespace
      version   = helm_release.prometheus_operator.version
      chart     = helm_release.prometheus_operator.chart
    }
    cert_manager = var.install_cert_manager ? {
      name      = helm_release.cert_manager[0].name
      namespace = helm_release.cert_manager[0].namespace
      version   = helm_release.cert_manager[0].version
      chart     = helm_release.cert_manager[0].chart
    } : null
  }
}

# Network policies
output "network_policies" {
  description = "Created network policies"
  value = var.enable_network_policies ? [
    kubernetes_network_policy.monitoring_ingress[0].metadata[0].name
  ] : []
}

# Ingress information
output "ingress_resources" {
  description = "Created ingress resources"
  value = var.enable_ingress ? {
    prometheus = {
      name  = kubernetes_ingress_v1.prometheus_ingress[0].metadata[0].name
      hosts = kubernetes_ingress_v1.prometheus_ingress[0].spec[0].rule[0].host
      url   = "https://${kubernetes_ingress_v1.prometheus_ingress[0].spec[0].rule[0].host}"
    }
    grafana = {
      name  = kubernetes_ingress_v1.grafana_ingress[0].metadata[0].name
      hosts = kubernetes_ingress_v1.grafana_ingress[0].spec[0].rule[0].host
      url   = "https://${kubernetes_ingress_v1.grafana_ingress[0].spec[0].rule[0].host}"
    }
    alertmanager = var.alertmanager_enabled ? {
      name  = kubernetes_ingress_v1.alertmanager_ingress[0].metadata[0].name
      hosts = kubernetes_ingress_v1.alertmanager_ingress[0].spec[0].rule[0].host
      url   = "https://${kubernetes_ingress_v1.alertmanager_ingress[0].spec[0].rule[0].host}"
    } : null
  } : null
}

# Monitoring targets
output "monitoring_targets" {
  description = "Expected monitoring targets"
  value = {
    kubernetes_components = [
      "kubernetes-apiservers",
      "kubernetes-nodes",
      "kubernetes-pods",
      "kube-state-metrics"
    ]
    monitoring_stack = [
      "prometheus-operator-prometheus",
      "prometheus-operator-alertmanager",
      "prometheus-operator-grafana"
    ]
    oran_components = [
      "vnf-operator",
      "intent-management",
      "oran-components"
    ]
  }
}

# Health check URLs
output "health_check_urls" {
  description = "Health check URLs for monitoring components"
  value = {
    prometheus = {
      internal = "http://prometheus-operator-kube-p-prometheus.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9090/-/healthy"
      ready    = "http://prometheus-operator-kube-p-prometheus.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9090/-/ready"
    }
    grafana = {
      internal = "http://prometheus-operator-grafana.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:80/api/health"
    }
    alertmanager = var.alertmanager_enabled ? {
      internal = "http://prometheus-operator-kube-p-alertmanager.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9093/-/healthy"
      ready    = "http://prometheus-operator-kube-p-alertmanager.${kubernetes_namespace.monitoring.metadata[0].name}.svc.cluster.local:9093/-/ready"
    } : null
  }
}

# Resource quotas (if enabled)
output "resource_quotas" {
  description = "Resource quotas for monitoring namespace"
  value = {
    prometheus_resources   = var.prometheus_resources
    grafana_resources      = var.grafana_resources
    alertmanager_resources = var.alertmanager_resources
  }
}

# Validation commands
output "validation_commands" {
  description = "Commands to validate the deployment"
  value = {
    check_pods         = "kubectl get pods -n ${kubernetes_namespace.monitoring.metadata[0].name}"
    check_services     = "kubectl get services -n ${kubernetes_namespace.monitoring.metadata[0].name}"
    check_servicemonitors = "kubectl get servicemonitors -n ${kubernetes_namespace.monitoring.metadata[0].name}"
    check_prometheusrules = "kubectl get prometheusrules -n ${kubernetes_namespace.monitoring.metadata[0].name}"
    check_ingress      = var.enable_ingress ? "kubectl get ingress -n ${kubernetes_namespace.monitoring.metadata[0].name}" : null
    prometheus_targets = "curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, instance: .labels.instance, health: .health}'"
    grafana_health     = "curl -s http://localhost:3000/api/health"
  }
}

# Backup information
output "backup_configuration" {
  description = "Backup configuration (if enabled)"
  value = var.enable_backup ? {
    enabled   = var.enable_backup
    schedule  = var.backup_schedule
    retention = var.backup_retention
  } : null
}

# Security configuration
output "security_configuration" {
  description = "Security configuration summary"
  value = {
    rbac_enabled           = var.enable_rbac
    network_policies       = var.enable_network_policies
    pod_security_policies  = var.enable_psp
    security_contexts      = var.enable_security_context
    tls_enabled           = var.enable_ingress
  }
}

# Performance configuration
output "performance_configuration" {
  description = "Performance tuning configuration"
  value = {
    scrape_interval           = var.scrape_interval
    evaluation_interval       = var.evaluation_interval
    prometheus_replicas       = var.prometheus_replicas
    alertmanager_replicas     = var.alertmanager_replicas
    wal_compression          = var.prometheus_wal_compression
    query_timeout            = var.prometheus_query_timeout
    query_max_concurrency    = var.prometheus_query_max_concurrency
  }
}

# Integration status
output "integration_status" {
  description = "Status of optional integrations"
  value = {
    thanos_enabled = var.enable_thanos
    jaeger_enabled = var.enable_jaeger
    loki_enabled   = var.enable_loki
    cert_manager   = var.install_cert_manager
  }
}