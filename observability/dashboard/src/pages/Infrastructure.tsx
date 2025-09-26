import React from 'react'
import { Helmet } from 'react-helmet-async'
import { Server } from 'lucide-react'

const Infrastructure: React.FC = () => {
  return (
    <>
      <Helmet>
        <title>Infrastructure - O-RAN Intent-MANO</title>
      </Helmet>
      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center space-x-3 mb-6">
            <Server className="h-8 w-8 text-primary-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Infrastructure</h1>
              <p className="text-gray-600">Monitor clusters, nodes, and resource utilization</p>
            </div>
          </div>
          <div className="card text-center py-12">
            <Server className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600">Infrastructure monitoring interface will be implemented here</p>
          </div>
        </div>
      </div>
    </>
  )
}

export default Infrastructure