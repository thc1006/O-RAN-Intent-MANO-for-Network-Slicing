import React from 'react'
import { Helmet } from 'react-helmet-async'
import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import {
  Activity,
  Target,
  Network,
  Server,
  AlertTriangle,
  CheckCircle,
  Clock,
  TrendingUp,
  TrendingDown,
  Zap,
  Users,
  Database,
} from 'lucide-react'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import apiService from '@/services/api'
import LoadingSpinner from '@/components/LoadingSpinner'
import { useWebSocket } from '@/contexts/WebSocketContext'

interface MetricCardProps {
  title: string
  value: string | number
  change?: {
    value: number
    type: 'increase' | 'decrease'
  }
  icon: React.ComponentType<any>
  color: 'primary' | 'success' | 'warning' | 'danger'
}

const MetricCard: React.FC<MetricCardProps> = ({ title, value, change, icon: Icon, color }) => {
  const colorClasses = {
    primary: 'bg-primary-500 text-primary-50',
    success: 'bg-success-500 text-success-50',
    warning: 'bg-warning-500 text-warning-50',
    danger: 'bg-danger-500 text-danger-50',
  }

  return (
    <motion.div
      whileHover={{ scale: 1.02 }}
      className="card"
    >
      <div className="flex items-center">
        <div className={`rounded-lg p-3 ${colorClasses[color]}`}>
          <Icon className="h-6 w-6" />
        </div>
        <div className="ml-4 flex-1">
          <p className="text-sm font-medium text-gray-600">{title}</p>
          <div className="flex items-center space-x-2">
            <p className="text-2xl font-bold text-gray-900">{value}</p>
            {change && (
              <div className={`flex items-center text-sm ${
                change.type === 'increase' ? 'text-success-600' : 'text-danger-600'
              }`}>
                {change.type === 'increase' ? (
                  <TrendingUp className="h-4 w-4 mr-1" />
                ) : (
                  <TrendingDown className="h-4 w-4 mr-1" />
                )}
                {Math.abs(change.value)}%
              </div>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  )
}

const Dashboard: React.FC = () => {
  const { lastMessage } = useWebSocket()

  // Fetch dashboard data
  const { data: systemMetrics, isLoading: metricsLoading } = useQuery({
    queryKey: ['system-metrics'],
    queryFn: () => apiService.getSystemMetrics('24h'),
    refetchInterval: 30000,
  })

  const { data: intents, isLoading: intentsLoading } = useQuery({
    queryKey: ['intents', { limit: 10 }],
    queryFn: () => apiService.getIntents({ limit: 10 }),
    refetchInterval: 60000,
  })

  const { data: slices, isLoading: slicesLoading } = useQuery({
    queryKey: ['slices', { limit: 10 }],
    queryFn: () => apiService.getNetworkSlices({ limit: 10 }),
    refetchInterval: 60000,
  })

  const { data: infrastructure, isLoading: infraLoading } = useQuery({
    queryKey: ['infrastructure'],
    queryFn: () => apiService.getInfrastructure(),
    refetchInterval: 60000,
  })

  const { data: alerts, isLoading: alertsLoading } = useQuery({
    queryKey: ['alerts', { limit: 5, status: 'open' }],
    queryFn: () => apiService.getAlerts({ limit: 5, status: 'open' }),
    refetchInterval: 30000,
  })

  if (metricsLoading || intentsLoading || slicesLoading || infraLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  // Sample data for charts
  const performanceData = systemMetrics?.slice(-24) || []
  const pieData = [
    { name: 'Active', value: slices?.data?.filter(s => s.status === 'active').length || 0, color: '#22c55e' },
    { name: 'Degraded', value: slices?.data?.filter(s => s.status === 'degraded').length || 0, color: '#f59e0b' },
    { name: 'Failed', value: slices?.data?.filter(s => s.status === 'failed').length || 0, color: '#ef4444' },
    { name: 'Inactive', value: slices?.data?.filter(s => s.status === 'inactive').length || 0, color: '#6b7280' },
  ]

  const intentStatusData = [
    { name: 'Active', value: intents?.data?.filter(i => i.status === 'active').length || 0 },
    { name: 'Processing', value: intents?.data?.filter(i => i.status === 'processing').length || 0 },
    { name: 'Failed', value: intents?.data?.filter(i => i.status === 'failed').length || 0 },
    { name: 'Pending', value: intents?.data?.filter(i => i.status === 'pending').length || 0 },
  ]

  const totalIntents = intents?.pagination?.total || 0
  const activeSlices = slices?.data?.filter(s => s.status === 'active').length || 0
  const totalClusters = infrastructure?.clusters?.length || 0
  const healthyClusters = infrastructure?.clusters?.filter(c => c.status === 'healthy').length || 0
  const openAlerts = alerts?.data?.length || 0

  return (
    <>
      <Helmet>
        <title>Dashboard - O-RAN Intent-MANO</title>
      </Helmet>

      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">System Overview</h1>
              <p className="text-gray-600">Real-time monitoring of your O-RAN Intent-MANO system</p>
            </div>
            <div className="flex items-center space-x-2 text-sm text-gray-500">
              <Activity className="h-4 w-4" />
              <span>Last updated: {new Date().toLocaleTimeString()}</span>
            </div>
          </div>

          {/* Key Metrics */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <MetricCard
              title="Total Intents"
              value={totalIntents}
              change={{ value: 12, type: 'increase' }}
              icon={Target}
              color="primary"
            />
            <MetricCard
              title="Active Slices"
              value={activeSlices}
              change={{ value: 5, type: 'increase' }}
              icon={Network}
              color="success"
            />
            <MetricCard
              title="Healthy Clusters"
              value={`${healthyClusters}/${totalClusters}`}
              icon={Server}
              color={healthyClusters === totalClusters ? 'success' : 'warning'}
            />
            <MetricCard
              title="Open Alerts"
              value={openAlerts}
              change={openAlerts > 0 ? { value: 25, type: 'increase' } : undefined}
              icon={AlertTriangle}
              color={openAlerts === 0 ? 'success' : 'danger'}
            />
          </div>

          {/* Charts Section */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* System Performance */}
            <div className="card">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">System Performance (24h)</h3>
              <ResponsiveContainer width="100%" height={300}>
                <AreaChart data={performanceData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis
                    dataKey="timestamp"
                    tickFormatter={(value) => new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  />
                  <YAxis />
                  <Tooltip
                    labelFormatter={(value) => new Date(value).toLocaleString()}
                    formatter={(value: any, name: string) => [`${value}%`, name]}
                  />
                  <Legend />
                  <Area
                    type="monotone"
                    dataKey="cpu.usage"
                    stackId="1"
                    stroke="#3b82f6"
                    fill="#3b82f6"
                    fillOpacity={0.6}
                    name="CPU Usage"
                  />
                  <Area
                    type="monotone"
                    dataKey="memory.used"
                    stackId="2"
                    stroke="#10b981"
                    fill="#10b981"
                    fillOpacity={0.6}
                    name="Memory Usage"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>

            {/* Network Slice Distribution */}
            <div className="card">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Network Slice Status</h3>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={pieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={100}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {pieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>

          {/* Intent Status Chart */}
          <div className="card">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Intent Status Distribution</h3>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={intentStatusData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="value" fill="#3b82f6" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Recent Activity and Alerts */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Recent Intents */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900">Recent Intents</h3>
                <Target className="h-5 w-5 text-gray-400" />
              </div>
              <div className="space-y-3">
                {intents?.data?.slice(0, 5).map((intent) => (
                  <div key={intent.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div className="flex items-center space-x-3">
                      <div className={`h-2 w-2 rounded-full ${
                        intent.status === 'active' ? 'bg-success-500' :
                        intent.status === 'processing' ? 'bg-warning-500' :
                        intent.status === 'failed' ? 'bg-danger-500' :
                        'bg-gray-400'
                      }`} />
                      <div>
                        <p className="text-sm font-medium text-gray-900">{intent.name}</p>
                        <p className="text-xs text-gray-500">{intent.type}</p>
                      </div>
                    </div>
                    <span className={`badge ${
                      intent.status === 'active' ? 'badge-success' :
                      intent.status === 'processing' ? 'badge-warning' :
                      intent.status === 'failed' ? 'badge-danger' :
                      'bg-gray-100 text-gray-800'
                    }`}>
                      {intent.status}
                    </span>
                  </div>
                )) || (
                  <p className="text-sm text-gray-500 text-center py-4">No recent intents</p>
                )}
              </div>
            </div>

            {/* Recent Alerts */}
            <div className="card">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900">Active Alerts</h3>
                <AlertTriangle className="h-5 w-5 text-gray-400" />
              </div>
              <div className="space-y-3">
                {alerts?.data?.map((alert) => (
                  <div key={alert.id} className="flex items-start space-x-3 p-3 bg-gray-50 rounded-lg">
                    <div className={`mt-0.5 h-2 w-2 rounded-full ${
                      alert.severity === 'critical' ? 'bg-danger-500' :
                      alert.severity === 'error' ? 'bg-danger-400' :
                      alert.severity === 'warning' ? 'bg-warning-500' :
                      'bg-primary-500'
                    }`} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-900 truncate">{alert.title}</p>
                      <p className="text-xs text-gray-500 mt-1">{alert.message}</p>
                      <p className="text-xs text-gray-400 mt-1">
                        {new Date(alert.timestamp).toLocaleString()}
                      </p>
                    </div>
                    <span className={`badge ${
                      alert.severity === 'critical' ? 'badge-danger' :
                      alert.severity === 'error' ? 'badge-danger' :
                      alert.severity === 'warning' ? 'badge-warning' :
                      'badge-info'
                    }`}>
                      {alert.severity}
                    </span>
                  </div>
                )) || (
                  <div className="text-center py-4">
                    <CheckCircle className="h-8 w-8 text-success-500 mx-auto mb-2" />
                    <p className="text-sm text-gray-500">No active alerts</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

export default Dashboard