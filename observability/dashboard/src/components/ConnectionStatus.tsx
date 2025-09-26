import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Wifi, WifiOff, AlertCircle } from 'lucide-react'
import { useWebSocket } from '@/contexts/WebSocketContext'

export const ConnectionStatus: React.FC = () => {
  const { isConnected } = useWebSocket()

  return (
    <AnimatePresence>
      {!isConnected && (
        <motion.div
          initial={{ opacity: 0, y: 50 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 50 }}
          className="fixed bottom-4 right-4 z-50"
        >
          <div className="flex items-center space-x-2 rounded-lg bg-danger-600 px-4 py-2 text-white shadow-lg">
            <WifiOff className="h-4 w-4" />
            <span className="text-sm font-medium">Connection Lost</span>
            <AlertCircle className="h-4 w-4" />
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}