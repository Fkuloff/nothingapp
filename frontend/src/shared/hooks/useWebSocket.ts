import { useEffect, useRef, useState } from 'react'
import { endpoints } from '../../shared/api/endpoints'
import { getAuthToken } from '../../shared/api/httpClient'
import type { WSMessageAction, WSEvent } from '../../shared/api/types'

type UseWebSocketProps = {
  chatId: number
  onMessage: (event: WSEvent) => void
  enabled?: boolean
}

export function useWebSocket({ chatId, onMessage, enabled = true }: UseWebSocketProps) {
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | undefined>(undefined)
  const reconnectAttemptsRef = useRef(0)
  const onMessageRef = useRef(onMessage)
  const chatIdRef = useRef(chatId)

  const maxReconnectAttempts = 5
  const reconnectDelay = 2000

  // Keep refs up to date
  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  useEffect(() => {
    chatIdRef.current = chatId
  }, [chatId])

  useEffect(() => {
    if (!enabled) {
      return
    }

    let isActive = true
    let connectionInProgress = false

    const connect = () => {
      // Prevent multiple connections
      if (!isActive) {
        console.log('Component unmounted, aborting connection')
        return
      }

      if (connectionInProgress) {
        console.log('Connection already in progress')
        return
      }

      if (wsRef.current?.readyState === WebSocket.OPEN) {
        console.log('WebSocket already connected')
        return
      }

      if (wsRef.current?.readyState === WebSocket.CONNECTING) {
        console.log('WebSocket already connecting')
        return
      }

      // Close any existing connection first
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }

      connectionInProgress = true

      const wsBaseUrl = import.meta.env.VITE_API_BASE_URL || window.location.origin
      const token = getAuthToken()
      const query = token ? `?token=${encodeURIComponent(token)}` : ''
      // Use global WebSocket endpoint instead of per-chat endpoint
      const wsUrl = `${wsBaseUrl.replace(/^http/, 'ws')}${endpoints.ws.global}${query}`

      console.log('Connecting to WebSocket:', wsUrl)
      const ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        connectionInProgress = false
        if (!isActive) {
          console.log('Component unmounted after connection, closing...')
          ws.close()
          return
        }
        console.log('WebSocket connected')
        setIsConnected(true)
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as WSEvent
          // Only process messages for the current chat
          if ('chat_id' in data && data.chat_id === chatIdRef.current) {
            onMessageRef.current(data)
          } else if ('error' in data) {
            // Always process error messages
            onMessageRef.current(data)
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      ws.onerror = (error) => {
        connectionInProgress = false
        console.error('WebSocket error:', error)
      }

      ws.onclose = (event) => {
        connectionInProgress = false
        console.log('WebSocket closed', event.code, event.reason)
        setIsConnected(false)

        const currentWs = wsRef.current
        if (currentWs === ws) {
          wsRef.current = null
        }

        // Only reconnect if still active and haven't exceeded attempts
        if (isActive && reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current++
          console.log(`Attempting to reconnect (${reconnectAttemptsRef.current}/${maxReconnectAttempts})...`)
          reconnectTimeoutRef.current = window.setTimeout(connect, reconnectDelay)
        } else if (reconnectAttemptsRef.current >= maxReconnectAttempts) {
          console.error('Max reconnection attempts reached')
        }
      }

      wsRef.current = ws
    }

    // Small delay to prevent multiple connects in StrictMode
    const initTimeout = window.setTimeout(connect, 100)

    return () => {
      isActive = false
      connectionInProgress = false

      window.clearTimeout(initTimeout)

      if (reconnectTimeoutRef.current) {
        window.clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = undefined
      }

      if (wsRef.current) {
        const ws = wsRef.current
        wsRef.current = null
        ws.close()
      }

      setIsConnected(false)
    }
  }, [enabled])

  const send = (data: WSMessageAction) => {
    const ws = wsRef.current
    if (!ws) {
      console.error('WebSocket is not initialized')
      return false
    }

    if (ws.readyState === WebSocket.CONNECTING) {
      console.log('WebSocket is connecting, queuing message...')
      // Wait for connection to open
      const sendWhenReady = () => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify(data))
        }
      }
      ws.addEventListener('open', sendWhenReady, { once: true })
      return true
    }

    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data))
      return true
    }

    console.error('WebSocket is not connected, state:', ws.readyState)
    return false
  }

  return { isConnected, send }
}
