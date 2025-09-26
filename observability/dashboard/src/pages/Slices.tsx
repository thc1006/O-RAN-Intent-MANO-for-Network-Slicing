import React from 'react'
import { Helmet } from 'react-helmet-async'
import { Network } from 'lucide-react'

const Slices: React.FC = () => {
  return (
    <>
      <Helmet>
        <title>Network Slices - O-RAN Intent-MANO</title>
      </Helmet>
      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center space-x-3 mb-6">
            <Network className="h-8 w-8 text-primary-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Network Slices</h1>
              <p className="text-gray-600">Monitor and manage network slice instances</p>
            </div>
          </div>
          <div className="card text-center py-12">
            <Network className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600">Network Slices monitoring interface will be implemented here</p>
          </div>
        </div>
      </div>
    </>
  )
}

export default Slices