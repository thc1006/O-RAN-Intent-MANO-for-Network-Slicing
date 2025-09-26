import axios, { AxiosInstance } from 'axios'

interface PrometheusQueryResult {
  metric: Record<string, string>
  value?: [number, string]
  values?: [number, string][]
}

interface PrometheusResponse {
  status: 'success' | 'error'
  data: {
    resultType: 'matrix' | 'vector' | 'scalar' | 'string'
    result: PrometheusQueryResult[]
  }
  error?: string
  warnings?: string[]
}

interface PrometheusRangeParams {
  query: string
  start: string | number
  end: string | number
  step: string | number
}

interface PrometheusInstantParams {
  query: string
  time?: string | number
}

class PrometheusService {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: import.meta.env.VITE_PROMETHEUS_URL || 'http://localhost:9090',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
    })
  }

  // Execute instant query
  async query(params: PrometheusInstantParams): Promise<PrometheusResponse> {
    const response = await this.client.get('/api/v1/query', { params })
    return response.data
  }

  // Execute range query
  async queryRange(params: PrometheusRangeParams): Promise<PrometheusResponse> {
    const response = await this.client.get('/api/v1/query_range', { params })
    return response.data
  }

  // Get available metrics
  async getMetrics(): Promise<string[]> {
    const response = await this.client.get('/api/v1/label/__name__/values')
    return response.data.data
  }

  // Get metric metadata
  async getMetricMetadata(metric?: string): Promise<Record<string, any>> {
    const params = metric ? { metric } : {}
    const response = await this.client.get('/api/v1/metadata', { params })
    return response.data.data
  }

  // Get targets
  async getTargets(): Promise<any> {
    const response = await this.client.get('/api/v1/targets')
    return response.data.data
  }

  // Get alerts
  async getAlerts(): Promise<any> {
    const response = await this.client.get('/api/v1/alerts')
    return response.data.data
  }

  // Get rules
  async getRules(): Promise<any> {
    const response = await this.client.get('/api/v1/rules')
    return response.data.data
  }

  // Helper methods for common queries

  // Get CPU usage for a specific service
  async getCPUUsage(service: string, timeRange: string = '1h'): Promise<PrometheusResponse> {
    const query = `avg(rate(container_cpu_usage_seconds_total{container="${service}"}[5m])) * 100`
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    return this.queryRange({
      query,
      start,
      end,
      step: '1m',
    })
  }

  // Get memory usage for a specific service
  async getMemoryUsage(service: string, timeRange: string = '1h'): Promise<PrometheusResponse> {
    const query = `avg(container_memory_usage_bytes{container="${service}"}) / 1024 / 1024`
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    return this.queryRange({
      query,
      start,
      end,
      step: '1m',
    })
  }

  // Get network I/O for a specific service
  async getNetworkIO(service: string, timeRange: string = '1h'): Promise<{
    ingress: PrometheusResponse
    egress: PrometheusResponse
  }> {
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    const [ingress, egress] = await Promise.all([
      this.queryRange({
        query: `sum(rate(container_network_receive_bytes_total{container="${service}"}[5m]))`,
        start,
        end,
        step: '1m',
      }),
      this.queryRange({
        query: `sum(rate(container_network_transmit_bytes_total{container="${service}"}[5m]))`,
        start,
        end,
        step: '1m',
      }),
    ])

    return { ingress, egress }
  }

  // Get pod count for a specific deployment
  async getPodCount(deployment: string): Promise<PrometheusResponse> {
    const query = `kube_deployment_status_replicas{deployment="${deployment}"}`
    return this.query({ query })
  }

  // Get service availability
  async getServiceAvailability(service: string, timeRange: string = '24h'): Promise<PrometheusResponse> {
    const query = `avg_over_time(up{job="${service}"}[${timeRange}]) * 100`
    return this.query({ query })
  }

  // Get HTTP request rate
  async getHTTPRequestRate(service: string, timeRange: string = '1h'): Promise<PrometheusResponse> {
    const query = `sum(rate(http_requests_total{service="${service}"}[5m]))`
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    return this.queryRange({
      query,
      start,
      end,
      step: '1m',
    })
  }

  // Get HTTP error rate
  async getHTTPErrorRate(service: string, timeRange: string = '1h'): Promise<PrometheusResponse> {
    const query = `sum(rate(http_requests_total{service="${service}",status=~"5.."}[5m])) / sum(rate(http_requests_total{service="${service}"}[5m])) * 100`
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    return this.queryRange({
      query,
      start,
      end,
      step: '1m',
    })
  }

  // Get custom metrics for O-RAN Intent MANO
  async getIntentMetrics(timeRange: string = '1h'): Promise<{
    activeIntents: PrometheusResponse
    intentSuccessRate: PrometheusResponse
    intentExecutionTime: PrometheusResponse
  }> {
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    const [activeIntents, intentSuccessRate, intentExecutionTime] = await Promise.all([
      this.query({
        query: 'oran_intents_active_total',
      }),
      this.queryRange({
        query: 'rate(oran_intents_success_total[5m]) / rate(oran_intents_total[5m]) * 100',
        start,
        end,
        step: '5m',
      }),
      this.queryRange({
        query: 'histogram_quantile(0.95, rate(oran_intent_execution_duration_seconds_bucket[5m]))',
        start,
        end,
        step: '5m',
      }),
    ])

    return { activeIntents, intentSuccessRate, intentExecutionTime }
  }

  // Get network slice metrics
  async getSliceMetrics(timeRange: string = '1h'): Promise<{
    activeSlices: PrometheusResponse
    sliceThroughput: PrometheusResponse
    sliceLatency: PrometheusResponse
  }> {
    const end = Math.floor(Date.now() / 1000)
    const start = end - this.parseTimeRange(timeRange)

    const [activeSlices, sliceThroughput, sliceLatency] = await Promise.all([
      this.query({
        query: 'oran_network_slices_active_total',
      }),
      this.queryRange({
        query: 'sum by (slice_id) (rate(oran_slice_throughput_bytes_total[5m]))',
        start,
        end,
        step: '1m',
      }),
      this.queryRange({
        query: 'histogram_quantile(0.95, rate(oran_slice_latency_seconds_bucket[5m]))',
        start,
        end,
        step: '1m',
      }),
    ])

    return { activeSlices, sliceThroughput, sliceLatency }
  }

  // Get infrastructure health metrics
  async getInfrastructureMetrics(): Promise<{
    kubernetesNodes: PrometheusResponse
    podStatus: PrometheusResponse
    clusterCPU: PrometheusResponse
    clusterMemory: PrometheusResponse
  }> {
    const [kubernetesNodes, podStatus, clusterCPU, clusterMemory] = await Promise.all([
      this.query({
        query: 'kube_node_status_condition{condition="Ready",status="true"}',
      }),
      this.query({
        query: 'kube_pod_status_phase',
      }),
      this.query({
        query: 'avg(100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100))',
      }),
      this.query({
        query: 'avg((1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100)',
      }),
    ])

    return { kubernetesNodes, podStatus, clusterCPU, clusterMemory }
  }

  // Utility method to parse time ranges
  private parseTimeRange(timeRange: string): number {
    const unit = timeRange.slice(-1)
    const value = parseInt(timeRange.slice(0, -1))

    switch (unit) {
      case 'm':
        return value * 60
      case 'h':
        return value * 60 * 60
      case 'd':
        return value * 24 * 60 * 60
      case 'w':
        return value * 7 * 24 * 60 * 60
      default:
        return value
    }
  }

  // Format values for display
  formatValue(value: string, unit: string = ''): string {
    const num = parseFloat(value)
    if (isNaN(num)) return value

    if (unit === 'bytes') {
      const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
      const i = Math.floor(Math.log(num) / Math.log(1024))
      return `${(num / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`
    }

    if (unit === 'percent') {
      return `${num.toFixed(2)}%`
    }

    if (unit === 'duration') {
      if (num < 1) return `${(num * 1000).toFixed(0)}ms`
      if (num < 60) return `${num.toFixed(2)}s`
      return `${(num / 60).toFixed(2)}m`
    }

    return num.toLocaleString()
  }
}

export const prometheusService = new PrometheusService()
export default prometheusService