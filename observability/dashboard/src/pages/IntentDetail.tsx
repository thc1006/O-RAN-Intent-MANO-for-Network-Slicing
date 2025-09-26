import React from 'react'
import { useParams } from 'react-router-dom'
import { Helmet } from 'react-helmet-async'
import { useQuery } from '@tanstack/react-query'
import apiService from '@/services/api'
import LoadingSpinner from '@/components/LoadingSpinner'

const IntentDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>()

  const { data: intent, isLoading } = useQuery({
    queryKey: ['intent', id],
    queryFn: () => apiService.getIntent(id!),
    enabled: !!id,
  })

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    )
  }

  return (
    <>
      <Helmet>
        <title>{intent?.name} - Intent Detail</title>
      </Helmet>
      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <h1 className="text-2xl font-bold text-gray-900">Intent Detail: {intent?.name}</h1>
          <p className="text-gray-600 mt-2">Detailed view and management for this intent</p>
          {/* Implementation continues... */}
        </div>
      </div>
    </>
  )
}

export default IntentDetail