import React from 'react'
import { motion } from 'framer-motion'
import { AlertTriangle, Info, CheckCircle, X } from 'lucide-react'

interface AlertBannerProps {
  type: 'info' | 'success' | 'warning' | 'error'
  message: string
  action?: {
    label: string
    onClick: () => void
  }
  onClose?: () => void
  className?: string
}

export const AlertBanner: React.FC<AlertBannerProps> = ({
  type,
  message,
  action,
  onClose,
  className = '',
}) => {
  const config = {
    info: {
      bgColor: 'bg-primary-50',
      textColor: 'text-primary-800',
      iconColor: 'text-primary-600',
      icon: Info,
    },
    success: {
      bgColor: 'bg-success-50',
      textColor: 'text-success-800',
      iconColor: 'text-success-600',
      icon: CheckCircle,
    },
    warning: {
      bgColor: 'bg-warning-50',
      textColor: 'text-warning-800',
      iconColor: 'text-warning-600',
      icon: AlertTriangle,
    },
    error: {
      bgColor: 'bg-danger-50',
      textColor: 'text-danger-800',
      iconColor: 'text-danger-600',
      icon: AlertTriangle,
    },
  }

  const { bgColor, textColor, iconColor, icon: Icon } = config[type]

  return (
    <motion.div
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      className={`${bgColor} ${className}`}
    >
      <div className="mx-auto max-w-7xl px-3 py-3 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between flex-wrap">
          <div className="flex items-center flex-1 min-w-0">
            <span className={`flex p-2 rounded-lg ${bgColor}`}>
              <Icon className={`h-5 w-5 ${iconColor}`} />
            </span>
            <p className={`ml-3 text-sm font-medium ${textColor} truncate`}>
              {message}
            </p>
          </div>

          <div className="flex items-center space-x-4">
            {action && (
              <button
                onClick={action.onClick}
                className={`text-sm font-medium ${textColor} hover:underline`}
              >
                {action.label}
              </button>
            )}
            {onClose && (
              <button
                onClick={onClose}
                className={`${textColor} hover:opacity-75`}
              >
                <X className="h-5 w-5" />
              </button>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  )
}