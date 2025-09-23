# O-RAN Intent-MANO Infrastructure as Code
# Terraform configuration for multi-cluster deployment

terraform {
  required_version = ">= 1.0"
  
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.20"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.10"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.4"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }
  
  backend "s3" {
    # Configure remote state backend
    bucket = var.terraform_state_bucket
    key    = "o-ran-mano/terraform.tfstate"
    region = var.aws_region
  }
}

# Variables
variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "namespace" {
  description = "Kubernetes namespace"
  type        = string
  default     = "o-ran-mano"
}

variable "cluster_configs" {
  description = "Cluster configuration map"
  type = map(object({
    context    = string
    region     = string
    node_count = number
    node_type  = string
  }))
  default = {
    central = {
      context    = "central-cluster"
      region     = "us-central1"
      node_count = 5
      node_type  = "standard-4"
    }
    regional = {
      context    = "regional-cluster"
      region     = "us-west1"
      node_count = 3
      node_type  = "standard-2"
    }
    edge01 = {
      context    = "edge01-cluster"
      region     = "us-west2"
      node_count = 3
      node_type  = "standard-2"
    }
    edge02 = {
      context    = "edge02-cluster"
      region     = "us-east1"
      node_count = 3
      node_type  = "standard-2"
    }
  }
}

variable "enable_monitoring" {
  description = "Enable monitoring stack deployment"
  type        = bool
  default     = true
}

variable "enable_service_mesh" {
  description = "Enable service mesh deployment"
  type        = bool
  default     = true
}

variable "service_mesh_type" {
  description = "Service mesh type (istio or linkerd)"
  type        = string
  default     = "istio"
}

variable "container_registry" {
  description = "Container registry URL"
  type        = string
  default     = "docker.io/thc1006"
}

variable "image_version" {
  description = "Application image version"
  type        = string
  default     = "latest"
}

variable "terraform_state_bucket" {
  description = "S3 bucket for Terraform state"
  type        = string
}

variable "aws_region" {
  description = "AWS region for state bucket"
  type        = string
  default     = "us-west-2"
}

variable "dns_zone" {
  description = "DNS zone for external access"
  type        = string
  default     = "o-ran-mano.local"
}

variable "enable_encryption" {
  description = "Enable encryption at rest"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Backup retention period in days"
  type        = number
  default     = 30
}

# Local values
locals {
  common_labels = {
    "app.kubernetes.io/name"       = "o-ran-mano"
    "app.kubernetes.io/instance"   = var.environment
    "app.kubernetes.io/version"    = var.image_version
    "app.kubernetes.io/managed-by" = "terraform"
    "environment"                  = var.environment
  }
  
  # Performance targets from thesis
  performance_targets = {
    deployment_time_seconds = 600
    throughput_mbps = {
      high = 4.57
      mid  = 2.77
      low  = 0.93
    }
    rtt_ms = {
      high = 16.1
      mid  = 15.7
      low  = 6.3
    }
  }
}

# Random password for database
resource "random_password" "database_password" {
  length  = 32
  special = true
}

# Generate TLS certificates
resource "local_file" "tls_cert_config" {
  content = templatefile("${path.module}/templates/cert-config.yaml.tpl", {
    dns_zone = var.dns_zone
  })
  filename = "${path.module}/generated/cert-config.yaml"
}

# Kubernetes provider configuration for each cluster
provider "kubernetes" {
  alias          = "central"
  config_context = var.cluster_configs.central.context
}

provider "kubernetes" {
  alias          = "regional"
  config_context = var.cluster_configs.regional.context
}

provider "kubernetes" {
  alias          = "edge01"
  config_context = var.cluster_configs.edge01.context
}

provider "kubernetes" {
  alias          = "edge02"
  config_context = var.cluster_configs.edge02.context
}

provider "helm" {
  alias = "central"
  kubernetes {
    config_context = var.cluster_configs.central.context
  }
}

provider "helm" {
  alias = "regional"
  kubernetes {
    config_context = var.cluster_configs.regional.context
  }
}

provider "helm" {
  alias = "edge01"
  kubernetes {
    config_context = var.cluster_configs.edge01.context
  }
}

provider "helm" {
  alias = "edge02"
  kubernetes {
    config_context = var.cluster_configs.edge02.context
  }
}

# Central cluster resources
module "central_cluster" {
  source = "./modules/cluster"
  
  providers = {
    kubernetes = kubernetes.central
    helm       = helm.central
  }
  
  cluster_name         = "central"
  namespace           = var.namespace
  environment         = var.environment
  common_labels       = local.common_labels
  container_registry  = var.container_registry
  image_version       = var.image_version
  enable_monitoring   = var.enable_monitoring
  enable_service_mesh = var.enable_service_mesh
  service_mesh_type   = var.service_mesh_type
  dns_zone           = var.dns_zone
  enable_encryption   = var.enable_encryption
  
  # Central cluster specific configuration
  is_central_cluster = true
  database_password  = random_password.database_password.result
  
  # Performance targets
  performance_targets = local.performance_targets
  
  # Component configuration
  components = {
    orchestrator = {
      enabled  = true
      replicas = 3
      resources = {
        requests = {
          cpu    = "500m"
          memory = "1Gi"
        }
        limits = {
          cpu    = "2"
          memory = "4Gi"
        }
      }
    }
    o2_client = {
      enabled  = true
      replicas = 2
      resources = {
        requests = {
          cpu    = "250m"
          memory = "512Mi"
        }
        limits = {
          cpu    = "1"
          memory = "2Gi"
        }
      }
    }
    vnf_operator = {
      enabled  = true
      replicas = 2
      resources = {
        requests = {
          cpu    = "250m"
          memory = "512Mi"
        }
        limits = {
          cpu    = "1"
          memory = "2Gi"
        }
      }
    }
    tn_manager = {
      enabled  = true
      replicas = 2
      resources = {
        requests = {
          cpu    = "250m"
          memory = "512Mi"
        }
        limits = {
          cpu    = "1"
          memory = "2Gi"
        }
      }
    }
  }
}

# Regional cluster resources
module "regional_cluster" {
  source = "./modules/cluster"
  
  providers = {
    kubernetes = kubernetes.regional
    helm       = helm.regional
  }
  
  cluster_name         = "regional"
  namespace           = var.namespace
  environment         = var.environment
  common_labels       = local.common_labels
  container_registry  = var.container_registry
  image_version       = var.image_version
  enable_monitoring   = var.enable_monitoring
  enable_service_mesh = var.enable_service_mesh
  service_mesh_type   = var.service_mesh_type
  dns_zone           = var.dns_zone
  enable_encryption   = var.enable_encryption
  
  # Regional cluster specific configuration
  is_central_cluster = false
  central_endpoint   = module.central_cluster.orchestrator_endpoint
  
  # Performance targets
  performance_targets = local.performance_targets
  
  # Component configuration for regional cluster
  components = {
    ran_dms = {
      enabled  = true
      replicas = 2
      resources = {
        requests = {
          cpu    = "250m"
          memory = "512Mi"
        }
        limits = {
          cpu    = "1"
          memory = "2Gi"
        }
      }
    }
    cn_dms = {
      enabled  = true
      replicas = 2
      resources = {
        requests = {
          cpu    = "250m"
          memory = "512Mi"
        }
        limits = {
          cpu    = "1"
          memory = "2Gi"
        }
      }
    }
    tn_agent = {
      enabled  = true
      replicas = 1
      resources = {
        requests = {
          cpu    = "100m"
          memory = "256Mi"
        }
        limits = {
          cpu    = "500m"
          memory = "1Gi"
        }
      }
    }
  }
}

# Edge cluster 01 resources
module "edge01_cluster" {
  source = "./modules/cluster"
  
  providers = {
    kubernetes = kubernetes.edge01
    helm       = helm.edge01
  }
  
  cluster_name         = "edge01"
  namespace           = var.namespace
  environment         = var.environment
  common_labels       = local.common_labels
  container_registry  = var.container_registry
  image_version       = var.image_version
  enable_monitoring   = var.enable_monitoring
  enable_service_mesh = var.enable_service_mesh
  service_mesh_type   = var.service_mesh_type
  dns_zone           = var.dns_zone
  enable_encryption   = var.enable_encryption
  
  # Edge cluster specific configuration
  is_central_cluster = false
  central_endpoint   = module.central_cluster.orchestrator_endpoint
  regional_endpoint  = module.regional_cluster.ran_dms_endpoint
  
  # Performance targets
  performance_targets = local.performance_targets
  
  # Component configuration for edge cluster
  components = {
    tn_agent = {
      enabled  = true
      replicas = 1
      resources = {
        requests = {
          cpu    = "100m"
          memory = "256Mi"
        }
        limits = {
          cpu    = "500m"
          memory = "1Gi"
        }
      }
    }
  }
}

# Edge cluster 02 resources
module "edge02_cluster" {
  source = "./modules/cluster"
  
  providers = {
    kubernetes = kubernetes.edge02
    helm       = helm.edge02
  }
  
  cluster_name         = "edge02"
  namespace           = var.namespace
  environment         = var.environment
  common_labels       = local.common_labels
  container_registry  = var.container_registry
  image_version       = var.image_version
  enable_monitoring   = var.enable_monitoring
  enable_service_mesh = var.enable_service_mesh
  service_mesh_type   = var.service_mesh_type
  dns_zone           = var.dns_zone
  enable_encryption   = var.enable_encryption
  
  # Edge cluster specific configuration
  is_central_cluster = false
  central_endpoint   = module.central_cluster.orchestrator_endpoint
  regional_endpoint  = module.regional_cluster.ran_dms_endpoint
  
  # Performance targets
  performance_targets = local.performance_targets
  
  # Component configuration for edge cluster
  components = {
    tn_agent = {
      enabled  = true
      replicas = 1
      resources = {
        requests = {
          cpu    = "100m"
          memory = "256Mi"
        }
        limits = {
          cpu    = "500m"
          memory = "1Gi"
        }
      }
    }
  }
}

# Cross-cluster networking
resource "kubernetes_config_map" "cluster_topology" {
  for_each = var.cluster_configs
  
  provider = kubernetes.central
  
  metadata {
    name      = "cluster-topology"
    namespace = var.namespace
    labels    = local.common_labels
  }
  
  data = {
    "clusters.yaml" = yamlencode({
      clusters = {
        for name, config in var.cluster_configs : name => {
          region     = config.region
          endpoint   = "https://${name}.${var.dns_zone}"
          node_count = config.node_count
        }
      }
    })
  }
}

# Performance monitoring configuration
resource "kubernetes_config_map" "performance_targets" {
  for_each = var.cluster_configs
  
  provider = kubernetes.central
  
  metadata {
    name      = "performance-targets"
    namespace = var.namespace
    labels    = local.common_labels
  }
  
  data = {
    "targets.json" = jsonencode(local.performance_targets)
  }
}

# Output values
output "central_cluster_endpoint" {
  description = "Central cluster orchestrator endpoint"
  value       = module.central_cluster.orchestrator_endpoint
}

output "regional_cluster_endpoint" {
  description = "Regional cluster RAN DMS endpoint"
  value       = module.regional_cluster.ran_dms_endpoint
}

output "edge_clusters" {
  description = "Edge cluster endpoints"
  value = {
    edge01 = module.edge01_cluster.cluster_endpoint
    edge02 = module.edge02_cluster.cluster_endpoint
  }
}

output "monitoring_endpoints" {
  description = "Monitoring dashboard endpoints"
  value = {
    prometheus = module.central_cluster.prometheus_endpoint
    grafana    = module.central_cluster.grafana_endpoint
    alertmanager = module.central_cluster.alertmanager_endpoint
  }
}

output "performance_targets" {
  description = "Performance targets from thesis"
  value       = local.performance_targets
}

output "deployment_summary" {
  description = "Deployment summary information"
  value = {
    environment        = var.environment
    namespace         = var.namespace
    image_version     = var.image_version
    clusters_deployed = length(var.cluster_configs)
    monitoring_enabled = var.enable_monitoring
    service_mesh_enabled = var.enable_service_mesh
    encryption_enabled = var.enable_encryption
  }
}
