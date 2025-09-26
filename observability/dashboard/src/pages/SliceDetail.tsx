import React from 'react'
import { useParams } from 'react-router-dom'
import { Helmet } from 'react-helmet-async'
import { Network } from 'lucide-react'

const SliceDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>()

  return (
    <>
      <Helmet>
        <title>Slice Detail - O-RAN Intent-MANO</title>
      </Helmet>
      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <h1 className="text-2xl font-bold text-gray-900">Network Slice: {id}</h1>
          <p className="text-gray-600 mt-2">Detailed monitoring and management for this network slice</p>
        </div>
      </div>
    </>
  )
}

export default SliceDetail