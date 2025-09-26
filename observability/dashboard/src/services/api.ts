import axios, { AxiosInstance, AxiosResponse } from 'axios'
import toast from 'react-hot-toast'
import {
  Intent,
  NetworkSlice,
  Infrastructure,
  Experiment,
  SystemMetrics,
  Alert,
  Config,
  ApiResponse,
  PaginatedResponse,
  Deployment
} from '@/types'

class ApiService {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('auth_token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        return config
      },
      (error) => {
        return Promise.reject(error)
      }
    )

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      async (error) => {
        const { response } = error

        if (response?.status === 401) {
          // Unauthorized - redirect to login
          localStorage.removeItem('auth_token')
          window.location.href = '/login'
          return Promise.reject(error)
        }

        if (response?.status === 403) {
          toast.error('Access denied. Insufficient permissions.')
        } else if (response?.status >= 500) {
          toast.error('Server error. Please try again later.')
        } else if (!response) {
          toast.error('Network error. Please check your connection.')
        }

        return Promise.reject(error)
      }
    )
  }

  // Helper method to handle API responses
  private async handleResponse<T>(response: AxiosResponse<ApiResponse<T>>): Promise<T> {
    return response.data.data
  }

  private async handlePaginatedResponse<T>(
    response: AxiosResponse<PaginatedResponse<T>>
  ): Promise<PaginatedResponse<T>> {
    return response.data
  }

  // Intent Management
  async getIntents(params?: {
    page?: number
    limit?: number
    status?: string
    type?: string
    priority?: string
  }): Promise<PaginatedResponse<Intent>> {
    const response = await this.client.get('/intents', { params })
    return this.handlePaginatedResponse(response)
  }

  async getIntent(id: string): Promise<Intent> {
    const response = await this.client.get(`/intents/${id}`)
    return this.handleResponse(response)
  }

  async createIntent(intent: Partial<Intent>): Promise<Intent> {
    const response = await this.client.post('/intents', intent)
    return this.handleResponse(response)
  }

  async updateIntent(id: string, updates: Partial<Intent>): Promise<Intent> {
    const response = await this.client.put(`/intents/${id}`, updates)
    return this.handleResponse(response)
  }

  async deleteIntent(id: string): Promise<void> {
    await this.client.delete(`/intents/${id}`)
  }

  async executeIntent(id: string): Promise<Intent> {
    const response = await this.client.post(`/intents/${id}/execute`)
    return this.handleResponse(response)
  }

  // Network Slice Management
  async getNetworkSlices(params?: {
    page?: number
    limit?: number
    status?: string
    type?: string
  }): Promise<PaginatedResponse<NetworkSlice>> {
    const response = await this.client.get('/slices', { params })
    return this.handlePaginatedResponse(response)
  }

  async getNetworkSlice(id: string): Promise<NetworkSlice> {
    const response = await this.client.get(`/slices/${id}`)
    return this.handleResponse(response)
  }

  async createNetworkSlice(slice: Partial<NetworkSlice>): Promise<NetworkSlice> {
    const response = await this.client.post('/slices', slice)
    return this.handleResponse(response)
  }

  async updateNetworkSlice(id: string, updates: Partial<NetworkSlice>): Promise<NetworkSlice> {
    const response = await this.client.put(`/slices/${id}`, updates)
    return this.handleResponse(response)
  }

  async deleteNetworkSlice(id: string): Promise<void> {
    await this.client.delete(`/slices/${id}`)
  }

  async getSliceMetrics(id: string, timeRange?: string): Promise<SystemMetrics[]> {
    const response = await this.client.get(`/slices/${id}/metrics`, {
      params: { timeRange }
    })
    return this.handleResponse(response)
  }

  // Infrastructure Management
  async getInfrastructure(): Promise<Infrastructure> {
    const response = await this.client.get('/infrastructure')
    return this.handleResponse(response)
  }

  async getClusterStatus(clusterId?: string): Promise<any> {
    const endpoint = clusterId ? `/infrastructure/clusters/${clusterId}` : '/infrastructure/clusters'
    const response = await this.client.get(endpoint)
    return this.handleResponse(response)
  }

  async getNodeMetrics(nodeId?: string): Promise<SystemMetrics[]> {
    const endpoint = nodeId ? `/infrastructure/nodes/${nodeId}/metrics` : '/infrastructure/nodes/metrics'
    const response = await this.client.get(endpoint)
    return this.handleResponse(response)
  }

  async getPodLogs(namespace: string, podName: string, lines?: number): Promise<string[]> {
    const response = await this.client.get(`/infrastructure/pods/${namespace}/${podName}/logs`, {
      params: { lines }
    })
    return this.handleResponse(response)
  }

  // Experiment Management
  async getExperiments(params?: {
    page?: number
    limit?: number
    status?: string
    type?: string
  }): Promise<PaginatedResponse<Experiment>> {
    const response = await this.client.get('/experiments', { params })
    return this.handlePaginatedResponse(response)
  }

  async getExperiment(id: string): Promise<Experiment> {
    const response = await this.client.get(`/experiments/${id}`)
    return this.handleResponse(response)
  }

  async createExperiment(experiment: Partial<Experiment>): Promise<Experiment> {
    const response = await this.client.post('/experiments', experiment)
    return this.handleResponse(response)
  }

  async startExperiment(id: string): Promise<Experiment> {
    const response = await this.client.post(`/experiments/${id}/start`)
    return this.handleResponse(response)
  }

  async stopExperiment(id: string): Promise<Experiment> {
    const response = await this.client.post(`/experiments/${id}/stop`)
    return this.handleResponse(response)
  }

  async getExperimentResults(id: string): Promise<any> {
    const response = await this.client.get(`/experiments/${id}/results`)
    return this.handleResponse(response)
  }

  // Metrics and Monitoring
  async getSystemMetrics(timeRange?: string, resolution?: string): Promise<SystemMetrics[]> {
    const response = await this.client.get('/metrics/system', {
      params: { timeRange, resolution }
    })
    return this.handleResponse(response)
  }

  async getCustomMetrics(query: string, timeRange?: string): Promise<any> {
    const response = await this.client.get('/metrics/custom', {
      params: { query, timeRange }
    })
    return this.handleResponse(response)
  }

  async getPrometheusMetrics(query: string, time?: string): Promise<any> {
    const response = await this.client.get('/metrics/prometheus', {
      params: { query, time }
    })
    return this.handleResponse(response)
  }

  // Alerts Management
  async getAlerts(params?: {
    page?: number
    limit?: number
    severity?: string
    status?: string
  }): Promise<PaginatedResponse<Alert>> {
    const response = await this.client.get('/alerts', { params })
    return this.handlePaginatedResponse(response)
  }

  async acknowledgeAlert(id: string): Promise<Alert> {
    const response = await this.client.post(`/alerts/${id}/acknowledge`)
    return this.handleResponse(response)
  }

  async resolveAlert(id: string): Promise<Alert> {
    const response = await this.client.post(`/alerts/${id}/resolve`)
    return this.handleResponse(response)
  }

  // Configuration Management
  async getConfig(): Promise<Config> {
    const response = await this.client.get('/config')
    return this.handleResponse(response)
  }

  async updateConfig(updates: Partial<Config>): Promise<Config> {
    const response = await this.client.put('/config', updates)
    return this.handleResponse(response)
  }

  // Deployment Management
  async getDeployments(params?: {
    page?: number
    limit?: number
    namespace?: string
    status?: string
  }): Promise<PaginatedResponse<Deployment>> {
    const response = await this.client.get('/deployments', { params })
    return this.handlePaginatedResponse(response)
  }

  async getDeployment(id: string): Promise<Deployment> {
    const response = await this.client.get(`/deployments/${id}`)
    return this.handleResponse(response)
  }

  async scaleDeployment(id: string, replicas: number): Promise<Deployment> {
    const response = await this.client.post(`/deployments/${id}/scale`, { replicas })
    return this.handleResponse(response)
  }

  async restartDeployment(id: string): Promise<Deployment> {
    const response = await this.client.post(`/deployments/${id}/restart`)
    return this.handleResponse(response)
  }

  // Health and Status
  async getHealth(): Promise<{ status: string; checks: Record<string, boolean> }> {
    const response = await this.client.get('/health')
    return this.handleResponse(response)
  }

  async getVersion(): Promise<{ version: string; buildTime: string; gitCommit: string }> {
    const response = await this.client.get('/version')
    return this.handleResponse(response)
  }
}

export const apiService = new ApiService()
export default apiService