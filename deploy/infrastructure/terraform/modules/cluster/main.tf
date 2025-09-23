# Cluster module for O-RAN Intent-MANO deployment

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.20"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.10"
    }
  }
}

# Variables
variable "cluster_name" {
  description = "Name of the cluster"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

variable "common_labels" {
  description = "Common labels to apply to all resources"
  type        = map(string)
}

variable "container_registry" {
  description = "Container registry URL"
  type        = string
}

variable "image_version" {
  description = "Application image version"
  type        = string
}

variable "enable_monitoring" {
  description = "Enable monitoring stack"
  type        = bool
  default     = true
}

variable "enable_service_mesh" {
  description = "Enable service mesh"
  type        = bool
  default     = true
}

variable "service_mesh_type" {
  description = "Service mesh type"
  type        = string
  default     = "istio"
}

variable "dns_zone" {
  description = "DNS zone for external access"
  type        = string
}

variable "enable_encryption" {
  description = "Enable encryption at rest"
  type        = bool
  default     = true
}

variable "is_central_cluster" {
  description = "Whether this is the central cluster"
  type        = bool
  default     = false
}

variable "database_password" {
  description = "Database password"
  type        = string
  default     = ""
  sensitive   = true
}

variable "central_endpoint" {
  description = "Central cluster endpoint"
  type        = string
  default     = ""
}

variable "regional_endpoint" {
  description = "Regional cluster endpoint"
  type        = string
  default     = ""
}

variable "performance_targets" {
  description = "Performance targets configuration"
  type = object({
    deployment_time_seconds = number
    throughput_mbps = object({
      high = number
      mid  = number
      low  = number
    })
    rtt_ms = object({
      high = number
      mid  = number
      low  = number
    })
  })
}

variable "components" {
  description = "Components to deploy in this cluster"
  type = map(object({
    enabled  = bool
    replicas = number
    resources = object({
      requests = object({
        cpu    = string
        memory = string
      })
      limits = object({
        cpu    = string
        memory = string
      })
    })
  }))
  default = {}
}

# Local values
locals {
  cluster_labels = merge(var.common_labels, {
    "cluster" = var.cluster_name
  })
}

# Namespace
resource "kubernetes_namespace" "main" {
  metadata {
    name   = var.namespace
    labels = local.cluster_labels
    
    annotations = {
      "cluster.o-ran-mano.io/name" = var.cluster_name
      "cluster.o-ran-mano.io/type" = var.is_central_cluster ? "central" : "edge"
    }
  }
}

# Service Account with RBAC
resource "kubernetes_service_account" "mano_service_account" {
  metadata {
    name      = "o-ran-mano"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  automount_service_account_token = true
}

resource "kubernetes_cluster_role" "mano_cluster_role" {
  metadata {
    name   = "o-ran-mano-${var.cluster_name}"
    labels = local.cluster_labels
  }
  
  rule {
    api_groups = [""]
    resources  = ["pods", "services", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"]
    verbs      = ["get", "list", "watch", "create", "update", "patch", "delete"]
  }
  
  rule {
    api_groups = ["apps"]
    resources  = ["deployments", "daemonsets", "replicasets", "statefulsets"]
    verbs      = ["get", "list", "watch", "create", "update", "patch", "delete"]
  }
  
  rule {
    api_groups = ["networking.k8s.io"]
    resources  = ["networkpolicies", "ingresses"]
    verbs      = ["get", "list", "watch", "create", "update", "patch", "delete"]
  }
  
  rule {
    api_groups = ["o2.o-ran.org"]
    resources  = ["*"]
    verbs      = ["*"]
  }
  
  rule {
    api_groups = ["nephio.org"]
    resources  = ["*"]
    verbs      = ["*"]
  }
}

resource "kubernetes_cluster_role_binding" "mano_cluster_role_binding" {
  metadata {
    name   = "o-ran-mano-${var.cluster_name}"
    labels = local.cluster_labels
  }
  
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role.mano_cluster_role.metadata[0].name
  }
  
  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.mano_service_account.metadata[0].name
    namespace = kubernetes_namespace.main.metadata[0].name
  }
}

# Secrets management
resource "kubernetes_secret" "cluster_config" {
  metadata {
    name      = "cluster-config"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  data = {
    cluster_name     = var.cluster_name
    environment      = var.environment
    central_endpoint = var.central_endpoint
    regional_endpoint = var.regional_endpoint
    dns_zone         = var.dns_zone
  }
  
  type = "Opaque"
}

resource "kubernetes_secret" "database_credentials" {
  count = var.is_central_cluster ? 1 : 0
  
  metadata {
    name      = "database-credentials"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  data = {
    password = var.database_password
    username = "o-ran-mano"
    database = "mano_db"
  }
  
  type = "Opaque"
}

# ConfigMap for performance targets
resource "kubernetes_config_map" "performance_config" {
  metadata {
    name      = "performance-config"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  data = {
    "performance-targets.json" = jsonencode(var.performance_targets)
  }
}

# Storage Classes for persistent storage
resource "kubernetes_storage_class" "fast_ssd" {
  count = var.is_central_cluster ? 1 : 0
  
  metadata {
    name   = "fast-ssd"
    labels = local.cluster_labels
  }
  
  storage_provisioner    = "kubernetes.io/gce-pd"
  reclaim_policy         = "Retain"
  allow_volume_expansion = true
  
  parameters = {
    type = "pd-ssd"
    replication-type = "regional-pd"
  }
}

# Persistent Volumes for central cluster database
resource "kubernetes_persistent_volume_claim" "database_storage" {
  count = var.is_central_cluster ? 1 : 0
  
  metadata {
    name      = "database-storage"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  spec {
    access_modes = ["ReadWriteOnce"]
    
    resources {
      requests = {
        storage = "100Gi"
      }
    }
    
    storage_class_name = var.is_central_cluster ? kubernetes_storage_class.fast_ssd[0].metadata[0].name : null
  }
}

# Network Policies
resource "kubernetes_network_policy" "default_deny" {
  metadata {
    name      = "default-deny"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  spec {
    pod_selector {}
    policy_types = ["Ingress", "Egress"]
  }
}

resource "kubernetes_network_policy" "allow_mano_communication" {
  metadata {
    name      = "allow-mano-communication"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
  }
  
  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/part-of" = "o-ran-mano"
      }
    }
    
    policy_types = ["Ingress", "Egress"]
    
    ingress {
      from {
        pod_selector {
          match_labels = {
            "app.kubernetes.io/part-of" = "o-ran-mano"
          }
        }
      }
      
      from {
        namespace_selector {
          match_labels = {
            name = "monitoring"
          }
        }
      }
    }
    
    egress {
      to {
        pod_selector {
          match_labels = {
            "app.kubernetes.io/part-of" = "o-ran-mano"
          }
        }
      }
    }
    
    egress {
      to {}
      ports {
        protocol = "TCP"
        port     = "53"
      }
      ports {
        protocol = "UDP"
        port     = "53"
      }
    }
    
    egress {
      to {}
      ports {
        protocol = "TCP"
        port     = "443"
      }
    }
  }
}

# Service Mesh Configuration
resource "kubernetes_namespace" "service_mesh" {
  count = var.enable_service_mesh ? 1 : 0
  
  metadata {
    name = var.service_mesh_type == "istio" ? "istio-system" : "linkerd"
    labels = merge(local.cluster_labels, {
      "name" = var.service_mesh_type == "istio" ? "istio-system" : "linkerd"
      var.service_mesh_type == "istio" ? "istio-injection" : "linkerd.io/control-plane-ns" = var.service_mesh_type == "istio" ? "enabled" : "linkerd"
    })
  }
}

# Istio Service Mesh
resource "helm_release" "istio_base" {
  count = var.enable_service_mesh && var.service_mesh_type == "istio" ? 1 : 0
  
  name       = "istio-base"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "base"
  version    = "1.18.0"
  namespace  = kubernetes_namespace.service_mesh[0].metadata[0].name
  
  create_namespace = false
  
  values = [
    yamlencode({
      defaultRevision = "default"
    })
  ]
}

resource "helm_release" "istiod" {
  count = var.enable_service_mesh && var.service_mesh_type == "istio" ? 1 : 0
  
  name       = "istiod"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "istiod"
  version    = "1.18.0"
  namespace  = kubernetes_namespace.service_mesh[0].metadata[0].name
  
  create_namespace = false
  
  depends_on = [helm_release.istio_base]
  
  values = [
    yamlencode({
      telemetry = {
        v2 = {
          enabled = true
        }
      }
      pilot = {
        traceSampling = 1.0
      }
    })
  ]
}

# Monitoring Stack
resource "kubernetes_namespace" "monitoring" {
  count = var.enable_monitoring ? 1 : 0
  
  metadata {
    name   = "monitoring"
    labels = local.cluster_labels
  }
}

resource "helm_release" "prometheus_stack" {
  count = var.enable_monitoring ? 1 : 0
  
  name       = "kube-prometheus-stack"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = "48.0.0"
  namespace  = kubernetes_namespace.monitoring[0].metadata[0].name
  
  create_namespace = false
  timeout          = 600
  
  values = [
    yamlencode({
      prometheus = {
        prometheusSpec = {
          retention = "30d"
          storageSpec = {
            volumeClaimTemplate = {
              spec = {
                storageClassName = var.is_central_cluster ? kubernetes_storage_class.fast_ssd[0].metadata[0].name : "standard"
                accessModes = ["ReadWriteOnce"]
                resources = {
                  requests = {
                    storage = "50Gi"
                  }
                }
              }
            }
          }
          additionalScrapeConfigs = [
            {
              job_name = "o-ran-mano-metrics"
              kubernetes_sd_configs = [{
                role = "pod"
                namespaces = {
                  names = [var.namespace]
                }
              }]
              relabel_configs = [{
                source_labels = ["__meta_kubernetes_pod_annotation_prometheus_io_scrape"]
                action = "keep"
                regex = "true"
              }]
            }
          ]
        }
      }
      grafana = {
        adminPassword = "admin"
        persistence = {
          enabled = true
          size = "10Gi"
          storageClassName = var.is_central_cluster ? kubernetes_storage_class.fast_ssd[0].metadata[0].name : "standard"
        }
        dashboardProviders = {
          "dashboardproviders.yaml" = {
            apiVersion = 1
            providers = [{
              name = "o-ran-mano"
              orgId = 1
              folder = "O-RAN MANO"
              type = "file"
              disableDeletion = false
              editable = true
              options = {
                path = "/var/lib/grafana/dashboards/o-ran-mano"
              }
            }]
          }
        }
        dashboards = {
          "o-ran-mano" = {
            "o-ran-mano-overview" = {
              url = "https://raw.githubusercontent.com/your-repo/o-ran-mano/main/deploy/monitoring/dashboards/overview.json"
            }
            "o-ran-mano-performance" = {
              url = "https://raw.githubusercontent.com/your-repo/o-ran-mano/main/deploy/monitoring/dashboards/performance.json"
            }
          }
        }
      }
      alertmanager = {
        alertmanagerSpec = {
          storage = {
            volumeClaimTemplate = {
              spec = {
                storageClassName = var.is_central_cluster ? kubernetes_storage_class.fast_ssd[0].metadata[0].name : "standard"
                accessModes = ["ReadWriteOnce"]
                resources = {
                  requests = {
                    storage = "10Gi"
                  }
                }
              }
            }
          }
        }
      }
    })
  ]
}

# Application deployments based on cluster type and component configuration
resource "kubernetes_deployment" "orchestrator" {
  count = lookup(var.components, "orchestrator", { enabled = false }).enabled ? 1 : 0
  
  metadata {
    name      = "orchestrator"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels = merge(local.cluster_labels, {
      "app.kubernetes.io/name"    = "orchestrator"
      "app.kubernetes.io/part-of" = "o-ran-mano"
    })
  }
  
  spec {
    replicas = var.components.orchestrator.replicas
    
    selector {
      match_labels = {
        "app.kubernetes.io/name" = "orchestrator"
      }
    }
    
    template {
      metadata {
        labels = merge(local.cluster_labels, {
          "app.kubernetes.io/name"    = "orchestrator"
          "app.kubernetes.io/part-of" = "o-ran-mano"
        })
        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/port"   = "8080"
          "prometheus.io/path"   = "/metrics"
        }
      }
      
      spec {
        service_account_name = kubernetes_service_account.mano_service_account.metadata[0].name
        
        container {
          name  = "orchestrator"
          image = "${var.container_registry}/orchestrator:${var.image_version}"
          
          port {
            name           = "http"
            container_port = 8080
            protocol       = "TCP"
          }
          
          port {
            name           = "grpc"
            container_port = 9090
            protocol       = "TCP"
          }
          
          resources {
            requests = {
              cpu    = var.components.orchestrator.resources.requests.cpu
              memory = var.components.orchestrator.resources.requests.memory
            }
            limits = {
              cpu    = var.components.orchestrator.resources.limits.cpu
              memory = var.components.orchestrator.resources.limits.memory
            }
          }
          
          env {
            name  = "CLUSTER_NAME"
            value = var.cluster_name
          }
          
          env {
            name  = "ENVIRONMENT"
            value = var.environment
          }
          
          env {
            name = "DB_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.database_credentials[0].metadata[0].name
                key  = "password"
              }
            }
          }
          
          volume_mount {
            name       = "config"
            mount_path = "/etc/orchestrator"
            read_only  = true
          }
          
          liveness_probe {
            http_get {
              path = "/health"
              port = 8080
            }
            initial_delay_seconds = 30
            period_seconds        = 10
          }
          
          readiness_probe {
            http_get {
              path = "/ready"
              port = 8080
            }
            initial_delay_seconds = 5
            period_seconds        = 5
          }
        }
        
        volume {
          name = "config"
          config_map {
            name = kubernetes_config_map.performance_config.metadata[0].name
          }
        }
      }
    }
  }
}

# Service for orchestrator
resource "kubernetes_service" "orchestrator" {
  count = lookup(var.components, "orchestrator", { enabled = false }).enabled ? 1 : 0
  
  metadata {
    name      = "orchestrator"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels = merge(local.cluster_labels, {
      "app.kubernetes.io/name"    = "orchestrator"
      "app.kubernetes.io/part-of" = "o-ran-mano"
    })
  }
  
  spec {
    selector = {
      "app.kubernetes.io/name" = "orchestrator"
    }
    
    port {
      name        = "http"
      port        = 80
      target_port = 8080
      protocol    = "TCP"
    }
    
    port {
      name        = "grpc"
      port        = 9090
      target_port = 9090
      protocol    = "TCP"
    }
    
    type = "ClusterIP"
  }
}

# Ingress for external access (central cluster only)
resource "kubernetes_ingress_v1" "orchestrator" {
  count = var.is_central_cluster && lookup(var.components, "orchestrator", { enabled = false }).enabled ? 1 : 0
  
  metadata {
    name      = "orchestrator"
    namespace = kubernetes_namespace.main.metadata[0].name
    labels    = local.cluster_labels
    
    annotations = {
      "kubernetes.io/ingress.class"                = "nginx"
      "cert-manager.io/cluster-issuer"             = "letsencrypt-prod"
      "nginx.ingress.kubernetes.io/ssl-redirect"   = "true"
      "nginx.ingress.kubernetes.io/force-ssl-redirect" = "true"
    }
  }
  
  spec {
    tls {
      hosts       = ["orchestrator.${var.dns_zone}"]
      secret_name = "orchestrator-tls"
    }
    
    rule {
      host = "orchestrator.${var.dns_zone}"
      
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          
          backend {
            service {
              name = kubernetes_service.orchestrator[0].metadata[0].name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }
}

# Similar deployments for other components would follow the same pattern...
# For brevity, I'm showing the pattern with orchestrator

# Output values
output "namespace" {
  description = "Kubernetes namespace"
  value       = kubernetes_namespace.main.metadata[0].name
}

output "cluster_endpoint" {
  description = "Cluster endpoint"
  value       = var.is_central_cluster ? "https://orchestrator.${var.dns_zone}" : "https://${var.cluster_name}.${var.dns_zone}"
}

output "orchestrator_endpoint" {
  description = "Orchestrator endpoint"
  value       = var.is_central_cluster && lookup(var.components, "orchestrator", { enabled = false }).enabled ? "https://orchestrator.${var.dns_zone}" : null
}

output "ran_dms_endpoint" {
  description = "RAN DMS endpoint"
  value       = lookup(var.components, "ran_dms", { enabled = false }).enabled ? "https://ran-dms.${var.dns_zone}" : null
}

output "prometheus_endpoint" {
  description = "Prometheus endpoint"
  value       = var.enable_monitoring ? "https://prometheus.${var.dns_zone}" : null
}

output "grafana_endpoint" {
  description = "Grafana endpoint"
  value       = var.enable_monitoring ? "https://grafana.${var.dns_zone}" : null
}

output "alertmanager_endpoint" {
  description = "Alertmanager endpoint"
  value       = var.enable_monitoring ? "https://alertmanager.${var.dns_zone}" : null
}
