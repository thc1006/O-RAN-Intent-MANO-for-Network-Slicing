import React, { useState } from 'react'
import { Link } from 'react-router-dom'
import { Helmet } from 'react-helmet-async'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import {
  Plus,
  Search,
  Filter,
  MoreVertical,
  Play,
  Pause,
  Trash2,
  Edit,
  Eye,
  Target,
  Clock,
  CheckCircle,
  AlertCircle,
  XCircle,
} from 'lucide-react'
import apiService from '@/services/api'
import LoadingSpinner from '@/components/LoadingSpinner'
import { Intent } from '@/types'
import toast from 'react-hot-toast'

interface IntentCardProps {
  intent: Intent
  onExecute: (id: string) => void
  onDelete: (id: string) => void
}

const IntentCard: React.FC<IntentCardProps> = ({ intent, onExecute, onDelete }) => {
  const [showActions, setShowActions] = useState(false)

  const statusConfig = {
    pending: { icon: Clock, color: 'text-gray-500', bg: 'bg-gray-100' },
    processing: { icon: Target, color: 'text-warning-600', bg: 'bg-warning-100' },
    active: { icon: CheckCircle, color: 'text-success-600', bg: 'bg-success-100' },
    failed: { icon: XCircle, color: 'text-danger-600', bg: 'bg-danger-100' },
    completed: { icon: CheckCircle, color: 'text-success-600', bg: 'bg-success-100' },
  }

  const priorityColors = {
    low: 'bg-gray-100 text-gray-800',
    medium: 'bg-primary-100 text-primary-800',
    high: 'bg-warning-100 text-warning-800',
    critical: 'bg-danger-100 text-danger-800',
  }

  const { icon: StatusIcon, color, bg } = statusConfig[intent.status]

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={{ scale: 1.02 }}
      className="card"
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start space-x-4 flex-1">
          <div className={`rounded-lg p-2 ${bg}`}>
            <StatusIcon className={`h-5 w-5 ${color}`} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center space-x-2">
              <h3 className="text-lg font-semibold text-gray-900 truncate">{intent.name}</h3>
              <span className={`badge ${priorityColors[intent.priority]}`}>
                {intent.priority}
              </span>
            </div>
            <p className="text-sm text-gray-600 mt-1 line-clamp-2">{intent.description}</p>
            <div className="flex items-center space-x-4 mt-3 text-sm text-gray-500">
              <span>Type: {intent.type.replace('-', ' ')}</span>
              <span>•</span>
              <span>Created: {new Date(intent.createdAt).toLocaleDateString()}</span>
              {intent.slices && (
                <>
                  <span>•</span>
                  <span>{intent.slices.length} slice(s)</span>
                </>
              )}
            </div>

            {/* Requirements */}
            {intent.requirements && (
              <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
                {intent.requirements.bandwidth && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Bandwidth:</span>
                    <span className="font-medium">{intent.requirements.bandwidth} Mbps</span>
                  </div>
                )}
                {intent.requirements.latency && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Latency:</span>
                    <span className="font-medium">{intent.requirements.latency} ms</span>
                  </div>
                )}
                {intent.requirements.reliability && (
                  <div className="flex justify-between">
                    <span className="text-gray-500">Reliability:</span>
                    <span className="font-medium">{intent.requirements.reliability}%</span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="relative">
          <button
            onClick={() => setShowActions(!showActions)}
            className="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          >
            <MoreVertical className="h-4 w-4" />
          </button>

          {showActions && (
            <div className="absolute right-0 mt-2 w-48 rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 z-10">
              <Link
                to={`/intents/${intent.id}`}
                className="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                onClick={() => setShowActions(false)}
              >
                <Eye className="mr-3 h-4 w-4" />
                View Details
              </Link>
              <button
                onClick={() => {
                  onExecute(intent.id)
                  setShowActions(false)
                }}
                disabled={intent.status === 'processing'}
                className="flex w-full items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 disabled:opacity-50"
              >
                <Play className="mr-3 h-4 w-4" />
                Execute
              </button>
              <button
                onClick={() => setShowActions(false)}
                className="flex w-full items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
              >
                <Edit className="mr-3 h-4 w-4" />
                Edit
              </button>
              <button
                onClick={() => {
                  onDelete(intent.id)
                  setShowActions(false)
                }}
                className="flex w-full items-center px-4 py-2 text-sm text-danger-600 hover:bg-gray-100"
              >
                <Trash2 className="mr-3 h-4 w-4" />
                Delete
              </button>
            </div>
          )}
        </div>
      </div>
    </motion.div>
  )
}

const Intents: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [priorityFilter, setPriorityFilter] = useState<string>('all')
  const [page, setPage] = useState(1)
  const pageSize = 12

  const queryClient = useQueryClient()

  // Fetch intents with filters
  const { data: intentsData, isLoading, error } = useQuery({
    queryKey: ['intents', {
      page,
      limit: pageSize,
      status: statusFilter !== 'all' ? statusFilter : undefined,
      type: typeFilter !== 'all' ? typeFilter : undefined,
      priority: priorityFilter !== 'all' ? priorityFilter : undefined,
    }],
    queryFn: () => apiService.getIntents({
      page,
      limit: pageSize,
      status: statusFilter !== 'all' ? statusFilter : undefined,
      type: typeFilter !== 'all' ? typeFilter : undefined,
      priority: priorityFilter !== 'all' ? priorityFilter : undefined,
    }),
    keepPreviousData: true,
  })

  // Execute intent mutation
  const executeIntentMutation = useMutation({
    mutationFn: (id: string) => apiService.executeIntent(id),
    onSuccess: () => {
      toast.success('Intent execution started')
      queryClient.invalidateQueries({ queryKey: ['intents'] })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to execute intent')
    },
  })

  // Delete intent mutation
  const deleteIntentMutation = useMutation({
    mutationFn: (id: string) => apiService.deleteIntent(id),
    onSuccess: () => {
      toast.success('Intent deleted successfully')
      queryClient.invalidateQueries({ queryKey: ['intents'] })
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to delete intent')
    },
  })

  const handleExecute = (id: string) => {
    executeIntentMutation.mutate(id)
  }

  const handleDelete = (id: string) => {
    if (window.confirm('Are you sure you want to delete this intent?')) {
      deleteIntentMutation.mutate(id)
    }
  }

  // Filter intents based on search query
  const filteredIntents = intentsData?.data?.filter(intent =>
    intent.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    intent.description.toLowerCase().includes(searchQuery.toLowerCase())
  ) || []

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <AlertCircle className="h-12 w-12 text-danger-500 mx-auto mb-4" />
          <p className="text-gray-600">Failed to load intents</p>
        </div>
      </div>
    )
  }

  return (
    <>
      <Helmet>
        <title>Intent Management - O-RAN Intent-MANO</title>
      </Helmet>

      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Intent Management</h1>
              <p className="text-gray-600">Create, monitor, and manage your network intents</p>
            </div>
            <Link
              to="/intents/new"
              className="btn btn-primary inline-flex items-center"
            >
              <Plus className="mr-2 h-4 w-4" />
              Create Intent
            </Link>
          </div>

          {/* Filters */}
          <div className="card">
            <div className="flex flex-col md:flex-row gap-4">
              {/* Search */}
              <div className="flex-1">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                  <input
                    type="text"
                    placeholder="Search intents..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="input pl-10"
                  />
                </div>
              </div>

              {/* Filters */}
              <div className="flex gap-4">
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="input"
                >
                  <option value="all">All Status</option>
                  <option value="pending">Pending</option>
                  <option value="processing">Processing</option>
                  <option value="active">Active</option>
                  <option value="failed">Failed</option>
                  <option value="completed">Completed</option>
                </select>

                <select
                  value={typeFilter}
                  onChange={(e) => setTypeFilter(e.target.value)}
                  className="input"
                >
                  <option value="all">All Types</option>
                  <option value="network-slice">Network Slice</option>
                  <option value="resource-allocation">Resource Allocation</option>
                  <option value="service-deployment">Service Deployment</option>
                </select>

                <select
                  value={priorityFilter}
                  onChange={(e) => setPriorityFilter(e.target.value)}
                  className="input"
                >
                  <option value="all">All Priorities</option>
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </div>
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {['pending', 'processing', 'active', 'failed'].map((status) => {
              const count = intentsData?.data?.filter(i => i.status === status).length || 0
              const color = status === 'active' ? 'success' :
                          status === 'failed' ? 'danger' :
                          status === 'processing' ? 'warning' : 'gray'
              return (
                <div key={status} className="card text-center">
                  <p className="text-2xl font-bold text-gray-900">{count}</p>
                  <p className="text-sm text-gray-600 capitalize">{status}</p>
                </div>
              )
            })}
          </div>

          {/* Intents List */}
          {isLoading ? (
            <div className="flex justify-center py-12">
              <LoadingSpinner size="lg" />
            </div>
          ) : filteredIntents.length === 0 ? (
            <div className="text-center py-12">
              <Target className="h-12 w-12 text-gray-400 mx-auto mb-4" />
              <p className="text-gray-600">No intents found</p>
              <Link
                to="/intents/new"
                className="btn btn-primary mt-4 inline-flex items-center"
              >
                <Plus className="mr-2 h-4 w-4" />
                Create Your First Intent
              </Link>
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {filteredIntents.map((intent) => (
                <IntentCard
                  key={intent.id}
                  intent={intent}
                  onExecute={handleExecute}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          )}

          {/* Pagination */}
          {intentsData && intentsData.pagination.totalPages > 1 && (
            <div className="flex justify-center">
              <div className="flex space-x-2">
                <button
                  onClick={() => setPage(page - 1)}
                  disabled={page === 1}
                  className="btn btn-outline disabled:opacity-50"
                >
                  Previous
                </button>
                <span className="flex items-center px-4 py-2 text-sm text-gray-700">
                  Page {page} of {intentsData.pagination.totalPages}
                </span>
                <button
                  onClick={() => setPage(page + 1)}
                  disabled={page === intentsData.pagination.totalPages}
                  className="btn btn-outline disabled:opacity-50"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

export default Intents