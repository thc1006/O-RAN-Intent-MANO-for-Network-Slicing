import React, { createContext, useContext, useEffect, useRef, useState, ReactNode } from 'react'
import { io, Socket } from 'socket.io-client'
import toast from 'react-hot-toast'
import { WebSocketMessage, MetricsUpdate, IntentUpdate, SliceUpdate, AlertUpdate } from '@/types'

interface WebSocketContextType {
  socket: Socket | null
  isConnected: boolean
  lastMessage: WebSocketMessage | null
  sendMessage: (type: string, payload: any) => void
  subscribe: (event: string, callback: (data: any) => void) => () => void
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined)

export const useWebSocket = () => {
  const context = useContext(WebSocketContext)
  if (context === undefined) {
    throw new Error('useWebSocket must be used within a WebSocketProvider')
  }
  return context
}

interface WebSocketProviderProps {
  children: ReactNode
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const [socket, setSocket] = useState<Socket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
  const reconnectAttempts = useRef(0)
  const maxReconnectAttempts = 5
  const reconnectInterval = useRef<NodeJS.Timeout>()

  useEffect(() => {
    const connectWebSocket = () => {
      const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080'
      const token = localStorage.getItem('auth_token')

      const socketInstance = io(wsUrl, {
        auth: {
          token: token,
        },
        transports: ['websocket'],
        upgrade: false,
        rememberUpgrade: false,
      })

      socketInstance.on('connect', () => {
        console.log('WebSocket connected')
        setIsConnected(true)
        reconnectAttempts.current = 0

        // Clear any existing reconnect timer
        if (reconnectInterval.current) {
          clearTimeout(reconnectInterval.current)
        }

        toast.success('Real-time connection established')
      })

      socketInstance.on('disconnect', (reason) => {
        console.log('WebSocket disconnected:', reason)
        setIsConnected(false)

        // Auto-reconnect if disconnected unexpectedly
        if (reason === 'io server disconnect') {
          // Server-side disconnect, don't reconnect automatically
          toast.error('Connection terminated by server')
        } else {
          // Client-side disconnect, attempt to reconnect
          handleReconnect()
        }
      })

      socketInstance.on('connect_error', (error) => {
        console.error('WebSocket connection error:', error)
        setIsConnected(false)
        handleReconnect()
      })

      // Handle different message types
      socketInstance.on('metrics_update', (data: MetricsUpdate['payload']) => {
        setLastMessage({
          type: 'metrics_update',
          payload: data,
          timestamp: new Date().toISOString(),
        })
      })

      socketInstance.on('intent_update', (data: IntentUpdate['payload']) => {
        setLastMessage({
          type: 'intent_update',
          payload: data,
          timestamp: new Date().toISOString(),
        })

        // Show notification for intent updates
        const { intent, action } = data
        if (action === 'created') {
          toast.success(`New intent created: ${intent.name}`)
        } else if (action === 'updated' && intent.status === 'failed') {
          toast.error(`Intent failed: ${intent.name}`)
        }
      })

      socketInstance.on('slice_update', (data: SliceUpdate['payload']) => {
        setLastMessage({
          type: 'slice_update',
          payload: data,
          timestamp: new Date().toISOString(),
        })

        // Show notification for slice status changes
        const { slice, action } = data
        if (action === 'updated' && slice.status === 'failed') {
          toast.error(`Network slice failed: ${slice.name}`)
        }
      })

      socketInstance.on('alert_update', (data: AlertUpdate['payload']) => {
        setLastMessage({
          type: 'alert_update',
          payload: data,
          timestamp: new Date().toISOString(),
        })

        // Show notification for critical alerts
        const { alert } = data
        if (alert.severity === 'critical') {
          toast.error(`Critical Alert: ${alert.title}`)
        } else if (alert.severity === 'error') {
          toast.error(`Error: ${alert.title}`)
        } else if (alert.severity === 'warning') {
          toast.success(`Warning: ${alert.title}`)
        }
      })

      setSocket(socketInstance)
    }

    const handleReconnect = () => {
      if (reconnectAttempts.current < maxReconnectAttempts) {
        const timeout = Math.pow(2, reconnectAttempts.current) * 1000 // Exponential backoff

        reconnectInterval.current = setTimeout(() => {
          console.log(`Attempting to reconnect... (${reconnectAttempts.current + 1}/${maxReconnectAttempts})`)
          reconnectAttempts.current += 1
          connectWebSocket()
        }, timeout)
      } else {
        toast.error('Failed to reconnect. Please refresh the page.')
      }
    }

    // Initialize connection
    connectWebSocket()

    // Cleanup on unmount
    return () => {
      if (socket) {
        socket.disconnect()
      }
      if (reconnectInterval.current) {
        clearTimeout(reconnectInterval.current)
      }
    }
  }, [])

  const sendMessage = (type: string, payload: any) => {
    if (socket && isConnected) {
      socket.emit(type, payload)
    } else {
      console.warn('WebSocket not connected, cannot send message')
      toast.error('Real-time connection unavailable')
    }
  }

  const subscribe = (event: string, callback: (data: any) => void) => {
    if (socket) {
      socket.on(event, callback)
      return () => socket.off(event, callback)
    }
    return () => {}
  }

  const value: WebSocketContextType = {
    socket,
    isConnected,
    lastMessage,
    sendMessage,
    subscribe,
  }

  return <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>
}