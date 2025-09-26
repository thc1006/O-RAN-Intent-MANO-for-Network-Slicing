import React from 'react'
import { Helmet } from 'react-helmet-async'
import { Settings as SettingsIcon } from 'lucide-react'

const Settings: React.FC = () => {
  return (
    <>
      <Helmet>
        <title>Settings - O-RAN Intent-MANO</title>
      </Helmet>
      <div className="h-full overflow-auto bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center space-x-3 mb-6">
            <SettingsIcon className="h-8 w-8 text-primary-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
              <p className="text-gray-600">Configure system preferences and integrations</p>
            </div>
          </div>
          <div className="card text-center py-12">
            <SettingsIcon className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600">Settings management interface will be implemented here</p>
          </div>
        </div>
      </div>
    </>
  )
}

export default Settings