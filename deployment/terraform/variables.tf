# Terraform Variables for O-RAN MANO Infrastructure
# This file defines all the configurable variables for the infrastructure

# Environment and basic configuration
variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}

variable "kubeconfig_context" {
  description = "Kubeconfig context to use"
  type        = string
  default     = ""
}

# Namespace configuration
variable "monitoring_namespace" {
  description = "Namespace for monitoring components"
  type        = string
  default     = "monitoring"
}

variable "oran_namespace" {
  description = "Namespace for O-RAN components"
  type        = string
  default     = "oran-mano"
}

# Prometheus configuration
variable "prometheus_operator_version" {
  description = "Version of the Prometheus Operator Helm chart"
  type        = string
  default     = "51.2.0"
}

variable "prometheus_retention" {
  description = "Data retention period for Prometheus"
  type        = string
  default     = "15d"
}

variable "prometheus_storage_size" {
  description = "Storage size for Prometheus data"
  type        = string
  default     = "50Gi"
}

variable "prometheus_custom_config" {
  description = "Custom Prometheus configuration (YAML)"
  type        = string
  default     = ""
}

# Grafana configuration
variable "grafana_admin_password" {
  description = "Admin password for Grafana"
  type        = string
  default     = "admin123"
  sensitive   = true
}

variable "grafana_service_type" {
  description = "Kubernetes service type for Grafana"
  type        = string
  default     = "ClusterIP"
  validation {
    condition     = contains(["ClusterIP", "NodePort", "LoadBalancer"], var.grafana_service_type)
    error_message = "Grafana service type must be one of: ClusterIP, NodePort, LoadBalancer."
  }
}

variable "grafana_persistence_enabled" {
  description = "Enable persistent storage for Grafana"
  type        = bool
  default     = true
}

variable "grafana_storage_size" {
  description = "Storage size for Grafana data"
  type        = string
  default     = "10Gi"
}

variable "grafana_external_datasources" {
  description = "External datasources configuration for Grafana"
  type        = map(string)
  default     = {}
}

variable "grafana_custom_dashboards" {
  description = "Custom Grafana dashboards (JSON)"
  type        = map(string)
  default     = {}
}

# AlertManager configuration
variable "alertmanager_enabled" {
  description = "Enable AlertManager"
  type        = bool
  default     = true
}

variable "alertmanager_retention" {
  description = "Data retention period for AlertManager"
  type        = string
  default     = "120h"
}

variable "alertmanager_storage_size" {
  description = "Storage size for AlertManager data"
  type        = string
  default     = "10Gi"
}

variable "alertmanager_external_config" {
  description = "External AlertManager configuration (YAML)"
  type        = string
  default     = ""
}

# Component enablement flags
variable "node_exporter_enabled" {
  description = "Enable Node Exporter"
  type        = bool
  default     = true
}

variable "kube_state_metrics_enabled" {
  description = "Enable kube-state-metrics"
  type        = bool
  default     = true
}

variable "install_cert_manager" {
  description = "Install cert-manager as part of the deployment"
  type        = bool
  default     = true
}

variable "cert_manager_version" {
  description = "Version of cert-manager to install"
  type        = string
  default     = "v1.13.0"
}

# Storage configuration
variable "create_storage_class" {
  description = "Create a storage class for monitoring persistence"
  type        = bool
  default     = true
}

variable "storage_class_name" {
  description = "Name of existing storage class to use (if not creating)"
  type        = string
  default     = "standard"
}

variable "storage_provisioner" {
  description = "Storage provisioner for the storage class"
  type        = string
  default     = "kubernetes.io/host-path"
}

variable "storage_parameters" {
  description = "Parameters for the storage class"
  type        = map(string)
  default     = {}
}

# Network configuration
variable "enable_network_policies" {
  description = "Enable network policies for security"
  type        = bool
  default     = false
}

variable "enable_ingress" {
  description = "Enable ingress for external access"
  type        = bool
  default     = false
}

variable "ingress_domain" {
  description = "Domain for ingress resources"
  type        = string
  default     = "example.com"
}

# Resource limits and requests
variable "prometheus_resources" {
  description = "Resource limits and requests for Prometheus"
  type = object({
    limits = object({
      cpu    = string
      memory = string
    })
    requests = object({
      cpu    = string
      memory = string
    })
  })
  default = {
    limits = {
      cpu    = "2000m"
      memory = "4Gi"
    }
    requests = {
      cpu    = "500m"
      memory = "2Gi"
    }
  }
}

variable "grafana_resources" {
  description = "Resource limits and requests for Grafana"
  type = object({
    limits = object({
      cpu    = string
      memory = string
    })
    requests = object({
      cpu    = string
      memory = string
    })
  })
  default = {
    limits = {
      cpu    = "200m"
      memory = "512Mi"
    }
    requests = {
      cpu    = "100m"
      memory = "256Mi"
    }
  }
}

variable "alertmanager_resources" {
  description = "Resource limits and requests for AlertManager"
  type = object({
    limits = object({
      cpu    = string
      memory = string
    })
    requests = object({
      cpu    = string
      memory = string
    })
  })
  default = {
    limits = {
      cpu    = "100m"
      memory = "256Mi"
    }
    requests = {
      cpu    = "50m"
      memory = "128Mi"
    }
  }
}

# High availability configuration
variable "prometheus_replicas" {
  description = "Number of Prometheus replicas for HA"
  type        = number
  default     = 1
  validation {
    condition     = var.prometheus_replicas >= 1 && var.prometheus_replicas <= 3
    error_message = "Prometheus replicas must be between 1 and 3."
  }
}

variable "alertmanager_replicas" {
  description = "Number of AlertManager replicas for HA"
  type        = number
  default     = 1
  validation {
    condition     = var.alertmanager_replicas >= 1 && var.alertmanager_replicas <= 3
    error_message = "AlertManager replicas must be between 1 and 3."
  }
}

# Monitoring configuration
variable "scrape_interval" {
  description = "Global scrape interval for Prometheus"
  type        = string
  default     = "30s"
}

variable "evaluation_interval" {
  description = "Global evaluation interval for Prometheus rules"
  type        = string
  default     = "30s"
}

# Security configuration
variable "enable_rbac" {
  description = "Enable RBAC for monitoring components"
  type        = bool
  default     = true
}

variable "enable_psp" {
  description = "Enable Pod Security Policies"
  type        = bool
  default     = false
}

variable "enable_security_context" {
  description = "Enable security contexts for pods"
  type        = bool
  default     = true
}

# Backup and disaster recovery
variable "enable_backup" {
  description = "Enable backup for monitoring data"
  type        = bool
  default     = false
}

variable "backup_schedule" {
  description = "Cron schedule for backups"
  type        = string
  default     = "0 2 * * *"
}

variable "backup_retention" {
  description = "Backup retention period"
  type        = string
  default     = "30d"
}

# External integrations
variable "slack_webhook_url" {
  description = "Slack webhook URL for alerts"
  type        = string
  default     = ""
  sensitive   = true
}

variable "pagerduty_integration_key" {
  description = "PagerDuty integration key for alerts"
  type        = string
  default     = ""
  sensitive   = true
}

variable "email_smtp_host" {
  description = "SMTP host for email alerts"
  type        = string
  default     = ""
}

variable "email_smtp_port" {
  description = "SMTP port for email alerts"
  type        = number
  default     = 587
}

variable "email_from" {
  description = "From email address for alerts"
  type        = string
  default     = ""
}

# Performance tuning
variable "prometheus_wal_compression" {
  description = "Enable WAL compression for Prometheus"
  type        = bool
  default     = true
}

variable "prometheus_query_timeout" {
  description = "Query timeout for Prometheus"
  type        = string
  default     = "2m"
}

variable "prometheus_query_max_concurrency" {
  description = "Maximum query concurrency for Prometheus"
  type        = number
  default     = 20
}

# Feature flags
variable "enable_thanos" {
  description = "Enable Thanos for long-term storage"
  type        = bool
  default     = false
}

variable "enable_jaeger" {
  description = "Enable Jaeger for distributed tracing"
  type        = bool
  default     = false
}

variable "enable_loki" {
  description = "Enable Loki for log aggregation"
  type        = bool
  default     = false
}

# Tags and labels
variable "additional_tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "additional_labels" {
  description = "Additional labels to apply to all Kubernetes resources"
  type        = map(string)
  default     = {}
}