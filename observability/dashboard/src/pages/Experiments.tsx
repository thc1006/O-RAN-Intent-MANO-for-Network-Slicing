import React from 'react'
import { Helmet } from 'react-helmet-async'
import { FlaskConical } from 'lucide-react'

const Experiments: React.FC = () => {
  return (
    <>
      <Helmet>
        <title>Experiments - O-RAN Intent-MANO</title>
      </Helmet>
      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center space-x-3 mb-6">
            <FlaskConical className="h-8 w-8 text-primary-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Experiments</h1>
              <p className="text-gray-600">Design, execute, and analyze system experiments</p>
            </div>
          </div>
          <div className="card text-center py-12">
            <FlaskConical className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600">Experiments management interface will be implemented here</p>
          </div>
        </div>
      </div>
    </>
  )
}

export default Experiments