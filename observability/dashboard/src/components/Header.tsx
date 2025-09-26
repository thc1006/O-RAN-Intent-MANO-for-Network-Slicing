import React, { useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
  Menu,
  Bell,
  Search,
  Settings,
  LogOut,
  User,
  Sun,
  Moon,
  RefreshCw,
  Wifi,
  WifiOff,
} from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useWebSocket } from '@/contexts/WebSocketContext'
import { useQuery } from '@tanstack/react-query'
import apiService from '@/services/api'
import toast from 'react-hot-toast'

interface HeaderProps {
  title: string
  onMenuClick: () => void
}

const Header: React.FC<HeaderProps> = ({ title, onMenuClick }) => {
  const { user, logout } = useAuth()
  const { isConnected } = useWebSocket()
  const [searchQuery, setSearchQuery] = useState('')
  const [showNotifications, setShowNotifications] = useState(false)
  const [showUserMenu, setShowUserMenu] = useState(false)

  // Fetch alerts for notifications
  const { data: alerts, refetch: refetchAlerts } = useQuery({
    queryKey: ['alerts', { limit: 10, status: 'open' }],
    queryFn: () => apiService.getAlerts({ limit: 10, status: 'open' }),
    refetchInterval: 30000, // Refetch every 30 seconds
  })

  const unreadAlerts = alerts?.data?.filter(alert => alert.status === 'open') || []

  const handleLogout = async () => {
    try {
      await logout()
      toast.success('Logged out successfully')
    } catch (error) {
      toast.error('Logout failed')
    }
  }

  const handleRefresh = () => {
    window.location.reload()
  }

  return (
    <header className="flex h-16 items-center justify-between bg-white px-6 shadow-sm">
      {/* Left side */}
      <div className="flex items-center space-x-4">
        <button
          onClick={onMenuClick}
          className="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 lg:hidden"
        >
          <Menu className="h-5 w-5" />
        </button>

        <div>
          <h1 className="text-xl font-semibold text-gray-900">{title}</h1>
          <div className="flex items-center space-x-2 text-sm text-gray-500">
            <span>{new Date().toLocaleDateString()}</span>
            <span>•</span>
            <div className="flex items-center space-x-1">
              {isConnected ? (
                <Wifi className="h-3 w-3 text-success-500" />
              ) : (
                <WifiOff className="h-3 w-3 text-danger-500" />
              )}
              <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Center - Search */}
      <div className="hidden md:flex flex-1 max-w-md mx-8">
        <div className="relative w-full">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search intents, slices, infrastructure..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-md border border-gray-300 bg-white py-2 pl-10 pr-4 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>
      </div>

      {/* Right side */}
      <div className="flex items-center space-x-2">
        {/* Refresh button */}
        <button
          onClick={handleRefresh}
          className="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          title="Refresh"
        >
          <RefreshCw className="h-5 w-5" />
        </button>

        {/* Notifications */}
        <div className="relative">
          <button
            onClick={() => setShowNotifications(!showNotifications)}
            className="relative rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          >
            <Bell className="h-5 w-5" />
            {unreadAlerts.length > 0 && (
              <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-danger-500 text-xs font-medium text-white">
                {unreadAlerts.length > 9 ? '9+' : unreadAlerts.length}
              </span>
            )}
          </button>

          {/* Notifications dropdown */}
          {showNotifications && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="absolute right-0 mt-2 w-80 rounded-md bg-white py-2 shadow-lg ring-1 ring-black ring-opacity-5 z-50"
            >
              <div className="px-4 py-2 border-b border-gray-200">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-medium text-gray-900">Notifications</h3>
                  <button
                    onClick={() => refetchAlerts()}
                    className="text-xs text-primary-600 hover:text-primary-700"
                  >
                    Refresh
                  </button>
                </div>
              </div>

              <div className="max-h-64 overflow-y-auto">
                {unreadAlerts.length === 0 ? (
                  <div className="px-4 py-6 text-center text-sm text-gray-500">
                    No new notifications
                  </div>
                ) : (
                  unreadAlerts.map((alert) => (
                    <Link
                      key={alert.id}
                      to={`/alerts/${alert.id}`}
                      className="block px-4 py-3 hover:bg-gray-50"
                      onClick={() => setShowNotifications(false)}
                    >
                      <div className="flex items-start space-x-3">
                        <div className={`mt-1 h-2 w-2 rounded-full ${
                          alert.severity === 'critical' ? 'bg-danger-500' :
                          alert.severity === 'error' ? 'bg-danger-400' :
                          alert.severity === 'warning' ? 'bg-warning-500' :
                          'bg-primary-500'
                        }`} />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-gray-900 truncate">
                            {alert.title}
                          </p>
                          <p className="text-xs text-gray-500 truncate">
                            {alert.message}
                          </p>
                          <p className="text-xs text-gray-400 mt-1">
                            {new Date(alert.timestamp).toLocaleTimeString()}
                          </p>
                        </div>
                      </div>
                    </Link>
                  ))
                )}
              </div>

              {unreadAlerts.length > 0 && (
                <div className="border-t border-gray-200 px-4 py-2">
                  <Link
                    to="/alerts"
                    className="text-xs text-primary-600 hover:text-primary-700"
                    onClick={() => setShowNotifications(false)}
                  >
                    View all alerts
                  </Link>
                </div>
              )}
            </motion.div>
          )}
        </div>

        {/* User menu */}
        <div className="relative">
          <button
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="flex items-center space-x-2 rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          >
            {user?.avatar ? (
              <img
                src={user.avatar}
                alt={user.name}
                className="h-6 w-6 rounded-full"
              />
            ) : (
              <User className="h-5 w-5" />
            )}
            <span className="hidden sm:block text-sm font-medium text-gray-700">
              {user?.name}
            </span>
          </button>

          {/* User dropdown */}
          {showUserMenu && (
            <motion.div
              initial={{ opacity: 0, y: -10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              className="absolute right-0 mt-2 w-48 rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 z-50"
            >
              <div className="px-4 py-2 border-b border-gray-200">
                <p className="text-sm font-medium text-gray-900">{user?.name}</p>
                <p className="text-xs text-gray-500">{user?.email}</p>
                <span className={`inline-flex mt-1 items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                  user?.role === 'admin' ? 'bg-warning-100 text-warning-800' :
                  user?.role === 'operator' ? 'bg-primary-100 text-primary-800' :
                  'bg-gray-100 text-gray-800'
                }`}>
                  {user?.role}
                </span>
              </div>

              <Link
                to="/settings"
                className="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                onClick={() => setShowUserMenu(false)}
              >
                <Settings className="mr-3 h-4 w-4" />
                Settings
              </Link>

              <button
                onClick={handleLogout}
                className="flex w-full items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
              >
                <LogOut className="mr-3 h-4 w-4" />
                Sign out
              </button>
            </motion.div>
          )}
        </div>
      </div>
    </header>
  )
}

export default Header