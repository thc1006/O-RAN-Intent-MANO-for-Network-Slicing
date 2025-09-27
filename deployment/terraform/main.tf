# O-RAN MANO Infrastructure as Code
# Main Terraform configuration for Kubernetes cluster provisioning

terraform {
  required_version = ">= 1.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "~> 1.14"
    }
  }

  # Backend configuration for state management
  backend "local" {
    path = "terraform.tfstate"
  }

  # For production, use remote backend:
  # backend "s3" {
  #   bucket = "oran-mano-terraform-state"
  #   key    = "monitoring/terraform.tfstate"
  #   region = "us-west-2"
  # }
}

# Provider configurations
provider "kubernetes" {
  config_path    = var.kubeconfig_path
  config_context = var.kubeconfig_context
}

provider "helm" {
  kubernetes {
    config_path    = var.kubeconfig_path
    config_context = var.kubeconfig_context
  }
}

provider "kubectl" {
  config_path    = var.kubeconfig_path
  config_context = var.kubeconfig_context
}

# Local variables
locals {
  common_labels = {
    "app.kubernetes.io/managed-by" = "terraform"
    "oran.io/component"            = "infrastructure"
    "oran.io/environment"          = var.environment
  }

  monitoring_labels = merge(local.common_labels, {
    "app.kubernetes.io/component" = "monitoring"
  })
}

# Create monitoring namespace
resource "kubernetes_namespace" "monitoring" {
  metadata {
    name = var.monitoring_namespace
    labels = local.monitoring_labels
    annotations = {
      "oran.io/created-by"    = "terraform"
      "oran.io/creation-date" = timestamp()
    }
  }
}

# Create O-RAN namespace
resource "kubernetes_namespace" "oran" {
  metadata {
    name = var.oran_namespace
    labels = merge(local.common_labels, {
      "app.kubernetes.io/component" = "oran-mano"
    })
    annotations = {
      "oran.io/created-by"    = "terraform"
      "oran.io/creation-date" = timestamp()
    }
  }
}

# Install cert-manager (prerequisite for monitoring stack)
resource "helm_release" "cert_manager" {
  count = var.install_cert_manager ? 1 : 0

  name       = "cert-manager"
  repository = "https://charts.jetstack.io"
  chart      = "cert-manager"
  version    = var.cert_manager_version
  namespace  = "cert-manager"

  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "installCRDs"
    value = "true"
  }

  set {
    name  = "global.leaderElection.namespace"
    value = "cert-manager"
  }

  depends_on = [kubernetes_namespace.monitoring]
}

# Create storage class for monitoring persistence
resource "kubernetes_storage_class" "monitoring_storage" {
  count = var.create_storage_class ? 1 : 0

  metadata {
    name = "monitoring-storage"
    labels = local.monitoring_labels
  }

  storage_provisioner    = var.storage_provisioner
  reclaim_policy        = "Retain"
  volume_binding_mode   = "WaitForFirstConsumer"
  allow_volume_expansion = true

  parameters = var.storage_parameters
}

# Create network policies for monitoring namespace
resource "kubernetes_network_policy" "monitoring_ingress" {
  count = var.enable_network_policies ? 1 : 0

  metadata {
    name      = "monitoring-ingress"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
  }

  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/part-of" = "kube-prometheus-stack"
      }
    }

    policy_types = ["Ingress"]

    ingress {
      from {
        namespace_selector {
          match_labels = {
            name = kubernetes_namespace.monitoring.metadata[0].name
          }
        }
      }
    }

    ingress {
      from {
        namespace_selector {
          match_labels = {
            name = kubernetes_namespace.oran.metadata[0].name
          }
        }
      }
    }

    # Allow ingress controller access
    ingress {
      from {
        namespace_selector {
          match_labels = {
            name = "ingress-nginx"
          }
        }
      }
    }
  }
}

# Create RBAC for monitoring
resource "kubernetes_service_account" "monitoring_operator" {
  metadata {
    name      = "monitoring-operator"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
  }

  automount_service_account_token = true
}

resource "kubernetes_cluster_role" "monitoring_operator" {
  metadata {
    name   = "monitoring-operator"
    labels = local.monitoring_labels
  }

  rule {
    api_groups = [""]
    resources  = ["nodes", "nodes/proxy", "services", "endpoints", "pods"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = ["extensions"]
    resources  = ["ingresses"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = ["networking.k8s.io"]
    resources  = ["ingresses"]
    verbs      = ["get", "list", "watch"]
  }
}

resource "kubernetes_cluster_role_binding" "monitoring_operator" {
  metadata {
    name   = "monitoring-operator"
    labels = local.monitoring_labels
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role.monitoring_operator.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account.monitoring_operator.metadata[0].name
    namespace = kubernetes_namespace.monitoring.metadata[0].name
  }
}

# Create secrets for external integrations
resource "kubernetes_secret" "alertmanager_config" {
  count = var.alertmanager_external_config != "" ? 1 : 0

  metadata {
    name      = "alertmanager-external-config"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
  }

  type = "Opaque"

  data = {
    "alertmanager.yml" = var.alertmanager_external_config
  }
}

resource "kubernetes_secret" "grafana_datasources" {
  count = length(var.grafana_external_datasources) > 0 ? 1 : 0

  metadata {
    name      = "grafana-external-datasources"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
  }

  type = "Opaque"

  data = {
    for name, config in var.grafana_external_datasources :
    "${name}.yml" => config
  }
}

# Create ConfigMaps for custom configurations
resource "kubernetes_config_map" "prometheus_config" {
  count = var.prometheus_custom_config != "" ? 1 : 0

  metadata {
    name      = "prometheus-custom-config"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels    = local.monitoring_labels
  }

  data = {
    "prometheus.yml" = var.prometheus_custom_config
  }
}

resource "kubernetes_config_map" "grafana_dashboards" {
  count = length(var.grafana_custom_dashboards) > 0 ? 1 : 0

  metadata {
    name      = "grafana-custom-dashboards"
    namespace = kubernetes_namespace.monitoring.metadata[0].name
    labels = merge(local.monitoring_labels, {
      "grafana_dashboard" = "1"
    })
  }

  data = var.grafana_custom_dashboards
}

# Output important information
output "monitoring_namespace" {
  description = "The monitoring namespace name"
  value       = kubernetes_namespace.monitoring.metadata[0].name
}

output "oran_namespace" {
  description = "The O-RAN namespace name"
  value       = kubernetes_namespace.oran.metadata[0].name
}

output "storage_class" {
  description = "The storage class for monitoring persistence"
  value       = var.create_storage_class ? kubernetes_storage_class.monitoring_storage[0].metadata[0].name : null
}

output "service_account" {
  description = "The monitoring service account"
  value       = kubernetes_service_account.monitoring_operator.metadata[0].name
}