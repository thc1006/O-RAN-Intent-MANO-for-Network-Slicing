import React, { useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { useWebSocket } from '@/contexts/WebSocketContext'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'
import { AlertBanner } from '@/components/AlertBanner'
import { ConnectionStatus } from '@/components/ConnectionStatus'

const Layout: React.FC = () => {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { isConnected } = useWebSocket()
  const location = useLocation()

  // Get page title from route
  const getPageTitle = () => {
    const path = location.pathname
    if (path === '/dashboard') return 'Dashboard'
    if (path.startsWith('/intents')) return 'Intent Management'
    if (path.startsWith('/slices')) return 'Network Slices'
    if (path.startsWith('/infrastructure')) return 'Infrastructure'
    if (path.startsWith('/experiments')) return 'Experiments'
    if (path.startsWith('/settings')) return 'Settings'
    return 'O-RAN Intent-MANO'
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Header */}
        <Header
          title={getPageTitle()}
          onMenuClick={() => setSidebarOpen(true)}
        />

        {/* Alert Banner for disconnected state */}
        {!isConnected && (
          <AlertBanner
            type="warning"
            message="Real-time connection lost. Some features may not work correctly."
            action={{
              label: 'Retry',
              onClick: () => window.location.reload(),
            }}
          />
        )}

        {/* Main content area */}
        <main className="flex-1 overflow-hidden">
          <div className="h-full">
            <Outlet />
          </div>
        </main>

        {/* Connection status indicator */}
        <ConnectionStatus />
      </div>
    </div>
  )
}

export default Layout