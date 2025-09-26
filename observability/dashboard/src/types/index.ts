// Base types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'operator' | 'viewer'
  avatar?: string
  lastLogin?: string
}

export interface Intent {
  id: string
  name: string
  description: string
  type: 'network-slice' | 'resource-allocation' | 'service-deployment'
  status: 'pending' | 'processing' | 'active' | 'failed' | 'completed'
  priority: 'low' | 'medium' | 'high' | 'critical'
  createdAt: string
  updatedAt: string
  createdBy: string
  requirements: {
    bandwidth?: number
    latency?: number
    reliability?: number
    coverage?: string[]
    qos?: QoSProfile
  }
  slices?: NetworkSlice[]
  deployments?: Deployment[]
  metadata?: Record<string, any>
}

export interface NetworkSlice {
  id: string
  name: string
  status: 'active' | 'inactive' | 'degraded' | 'failed'
  type: 'eMBB' | 'URLLC' | 'mMTC'
  intentId?: string
  specs: {
    bandwidth: number
    latency: number
    reliability: number
    userCount: number
  }
  resources: {
    cpu: ResourceUsage
    memory: ResourceUsage
    network: ResourceUsage
  }
  sla: {
    uptime: number
    availability: number
    throughput: number
  }
  createdAt: string
  updatedAt: string
  endpoints: string[]
  qosProfile: QoSProfile
}

export interface QoSProfile {
  name: string
  priority: number
  maxBitrate: number
  guaranteedBitrate: number
  packetDelayBudget: number
  packetErrorRate: number
}

export interface ResourceUsage {
  used: number
  total: number
  percentage: number
  trend?: 'up' | 'down' | 'stable'
}

export interface Infrastructure {
  clusters: KubernetesCluster[]
  nodes: NodeInfo[]
  pods: PodInfo[]
  services: ServiceInfo[]
  metrics: InfrastructureMetrics
}

export interface KubernetesCluster {
  id: string
  name: string
  status: 'healthy' | 'warning' | 'critical'
  version: string
  nodeCount: number
  podCount: number
  resources: {
    cpu: ResourceUsage
    memory: ResourceUsage
    storage: ResourceUsage
  }
  region: string
  provider: string
}

export interface NodeInfo {
  id: string
  name: string
  status: 'ready' | 'not-ready' | 'unknown'
  role: 'master' | 'worker'
  cpu: ResourceUsage
  memory: ResourceUsage
  pods: number
  version: string
  createdAt: string
}

export interface PodInfo {
  id: string
  name: string
  namespace: string
  status: 'running' | 'pending' | 'succeeded' | 'failed' | 'unknown'
  node: string
  restarts: number
  cpu: number
  memory: number
  createdAt: string
}

export interface ServiceInfo {
  id: string
  name: string
  namespace: string
  type: 'ClusterIP' | 'NodePort' | 'LoadBalancer' | 'ExternalName'
  clusterIP: string
  externalIP?: string
  ports: Array<{
    name: string
    port: number
    targetPort: number
    protocol: string
  }>
  selector: Record<string, string>
}

export interface InfrastructureMetrics {
  overall: {
    cpuUsage: number
    memoryUsage: number
    storageUsage: number
    networkIO: {
      ingress: number
      egress: number
    }
  }
  byCluster: Array<{
    clusterId: string
    metrics: {
      cpuUsage: number
      memoryUsage: number
      podCount: number
      nodeCount: number
    }
  }>
}

export interface Experiment {
  id: string
  name: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  type: 'performance' | 'reliability' | 'scalability' | 'integration'
  parameters: Record<string, any>
  results?: ExperimentResults
  createdAt: string
  startedAt?: string
  completedAt?: string
  duration?: number
  createdBy: string
}

export interface ExperimentResults {
  summary: {
    success: boolean
    score: number
    metrics: Record<string, number>
  }
  details: {
    logs: string[]
    measurements: Array<{
      timestamp: string
      metric: string
      value: number
      unit: string
    }>
    artifacts: Array<{
      name: string
      type: string
      url: string
      size: number
    }>
  }
}

export interface SystemMetrics {
  timestamp: string
  cpu: {
    usage: number
    cores: number
    frequency: number
  }
  memory: {
    used: number
    total: number
    cached: number
    buffers: number
  }
  network: {
    ingress: number
    egress: number
    connections: number
    errors: number
  }
  storage: {
    used: number
    total: number
    iops: number
    throughput: number
  }
  custom?: Record<string, number>
}

export interface Alert {
  id: string
  title: string
  message: string
  severity: 'info' | 'warning' | 'error' | 'critical'
  status: 'open' | 'acknowledged' | 'resolved'
  source: string
  timestamp: string
  acknowledgedBy?: string
  acknowledgedAt?: string
  resolvedAt?: string
  metadata?: Record<string, any>
}

export interface Config {
  system: {
    name: string
    version: string
    environment: 'development' | 'staging' | 'production'
    debug: boolean
  }
  api: {
    baseUrl: string
    timeout: number
    retries: number
  }
  websocket: {
    url: string
    reconnectInterval: number
    maxReconnectAttempts: number
  }
  prometheus: {
    url: string
    scrapeInterval: number
  }
  auth: {
    provider: 'oauth2' | 'oidc' | 'local'
    clientId?: string
    issuer?: string
    redirectUri?: string
  }
  features: {
    experiments: boolean
    realTimeMetrics: boolean
    advancedCharts: boolean
    notifications: boolean
  }
}

export interface Deployment {
  id: string
  name: string
  intentId: string
  status: 'pending' | 'deploying' | 'active' | 'failed' | 'terminated'
  type: 'pod' | 'service' | 'configmap' | 'secret'
  namespace: string
  replicas: {
    desired: number
    current: number
    ready: number
  }
  resources: {
    requests: {
      cpu: string
      memory: string
    }
    limits: {
      cpu: string
      memory: string
    }
  }
  createdAt: string
  updatedAt: string
  events: Array<{
    type: string
    reason: string
    message: string
    timestamp: string
  }>
}

// API Response types
export interface ApiResponse<T> {
  data: T
  message?: string
  success: boolean
  timestamp: string
}

export interface PaginatedResponse<T> {
  data: T[]
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}

// WebSocket message types
export interface WebSocketMessage {
  type: string
  payload: any
  timestamp: string
}

export interface MetricsUpdate extends WebSocketMessage {
  type: 'metrics_update'
  payload: {
    metrics: SystemMetrics
    source: string
  }
}

export interface IntentUpdate extends WebSocketMessage {
  type: 'intent_update'
  payload: {
    intent: Intent
    action: 'created' | 'updated' | 'deleted'
  }
}

export interface SliceUpdate extends WebSocketMessage {
  type: 'slice_update'
  payload: {
    slice: NetworkSlice
    action: 'created' | 'updated' | 'deleted'
  }
}

export interface AlertUpdate extends WebSocketMessage {
  type: 'alert_update'
  payload: {
    alert: Alert
    action: 'created' | 'updated' | 'resolved'
  }
}