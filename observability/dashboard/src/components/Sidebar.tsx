import React from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import {
  LayoutDashboard,
  Target,
  Network,
  Server,
  FlaskConical,
  Settings,
  X,
  Activity,
  Users,
  Shield,
  BarChart3,
} from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useWebSocket } from '@/contexts/WebSocketContext'

interface SidebarProps {
  open: boolean
  onClose: () => void
}

const navigation = [
  {
    name: 'Dashboard',
    href: '/dashboard',
    icon: LayoutDashboard,
    description: 'System overview and key metrics',
  },
  {
    name: 'Intent Management',
    href: '/intents',
    icon: Target,
    description: 'Submit and track intents',
  },
  {
    name: 'Network Slices',
    href: '/slices',
    icon: Network,
    description: 'Monitor slice status and performance',
  },
  {
    name: 'Infrastructure',
    href: '/infrastructure',
    icon: Server,
    description: 'Cluster and resource monitoring',
  },
  {
    name: 'Experiments',
    href: '/experiments',
    icon: FlaskConical,
    description: 'Test execution and results',
  },
  {
    name: 'Settings',
    href: '/settings',
    icon: Settings,
    description: 'System configuration',
  },
]

const Sidebar: React.FC<SidebarProps> = ({ open, onClose }) => {
  const { user } = useAuth()
  const { isConnected } = useWebSocket()
  const location = useLocation()

  const sidebarVariants = {
    open: {
      x: 0,
      transition: {
        type: 'spring',
        stiffness: 300,
        damping: 30,
      },
    },
    closed: {
      x: '-100%',
      transition: {
        type: 'spring',
        stiffness: 300,
        damping: 30,
      },
    },
  }

  const overlayVariants = {
    open: { opacity: 1 },
    closed: { opacity: 0 },
  }

  return (
    <>
      {/* Mobile overlay */}
      <AnimatePresence>
        {open && (
          <motion.div
            initial="closed"
            animate="open"
            exit="closed"
            variants={overlayVariants}
            className="fixed inset-0 z-20 bg-gray-600 bg-opacity-75 lg:hidden"
            onClick={onClose}
          />
        )}
      </AnimatePresence>

      {/* Sidebar */}
      <motion.div
        initial="closed"
        animate={open ? 'open' : 'closed'}
        variants={sidebarVariants}
        className="fixed inset-y-0 left-0 z-30 flex w-64 flex-col bg-white shadow-xl lg:static lg:z-auto lg:shadow-none"
      >
        <div className="flex h-16 items-center justify-between px-6 shadow-sm">
          <div className="flex items-center space-x-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-600">
              <Activity className="h-5 w-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-semibold text-gray-900">O-RAN MANO</h1>
              <div className="flex items-center space-x-1">
                <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-success-500' : 'bg-danger-500'}`} />
                <span className="text-xs text-gray-500">
                  {isConnected ? 'Connected' : 'Disconnected'}
                </span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 lg:hidden"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-1 px-3 py-4">
          {navigation.map((item) => {
            const isActive = location.pathname.startsWith(item.href)
            return (
              <NavLink
                key={item.name}
                to={item.href}
                onClick={onClose}
                className={({ isActive: navIsActive }) =>
                  `group flex items-center rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                    navIsActive || isActive
                      ? 'bg-primary-50 text-primary-700'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`
                }
              >
                <item.icon
                  className={`mr-3 h-5 w-5 flex-shrink-0 ${
                    isActive ? 'text-primary-600' : 'text-gray-400 group-hover:text-gray-500'
                  }`}
                />
                <div className="flex-1">
                  <div>{item.name}</div>
                  <div className="text-xs text-gray-500">{item.description}</div>
                </div>
              </NavLink>
            )
          })}
        </nav>

        {/* User profile */}
        {user && (
          <div className="border-t border-gray-200 p-4">
            <div className="flex items-center space-x-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100">
                {user.avatar ? (
                  <img
                    src={user.avatar}
                    alt={user.name}
                    className="h-8 w-8 rounded-full"
                  />
                ) : (
                  <Users className="h-4 w-4 text-primary-600" />
                )}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{user.name}</p>
                <p className="text-xs text-gray-500 truncate">{user.email}</p>
              </div>
              <div className="flex items-center space-x-1">
                {user.role === 'admin' && (
                  <Shield className="h-4 w-4 text-warning-500" title="Administrator" />
                )}
                <div className={`h-2 w-2 rounded-full bg-success-500`} title="Online" />
              </div>
            </div>
          </div>
        )}

        {/* System status */}
        <div className="border-t border-gray-200 p-4">
          <div className="flex items-center justify-between text-xs text-gray-500">
            <span>System Status</span>
            <BarChart3 className="h-4 w-4" />
          </div>
          <div className="mt-2 space-y-1">
            <div className="flex justify-between">
              <span className="text-xs text-gray-600">API</span>
              <span className="text-xs text-success-600">Healthy</span>
            </div>
            <div className="flex justify-between">
              <span className="text-xs text-gray-600">Database</span>
              <span className="text-xs text-success-600">Connected</span>
            </div>
            <div className="flex justify-between">
              <span className="text-xs text-gray-600">WebSocket</span>
              <span className={`text-xs ${isConnected ? 'text-success-600' : 'text-danger-600'}`}>
                {isConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>
        </div>
      </motion.div>
    </>
  )
}

export default Sidebar