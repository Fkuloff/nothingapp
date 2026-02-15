import { useEffect, useRef, useState } from 'react'

import { endpoints } from '../../shared/api/endpoints'
import { getAuthToken } from '../../shared/api/httpClient'
import type { WSEvent,WSMessageAction } from '../../shared/api/types'

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
      if (!isActive || connectionInProgress) {
        return
      }

      if (wsRef.current?.readyState === WebSocket.OPEN ||
          wsRef.current?.readyState === WebSocket.CONNECTING) {
        return
      }

      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }

      connectionInProgress = true

      const wsBaseUrl = import.meta.env.VITE_API_BASE_URL || window.location.origin
      const token = getAuthToken()
      const query = token ? `?token=${encodeURIComponent(token)}` : ''
      const wsUrl = `${wsBaseUrl.replace(/^http/, 'ws')}${endpoints.ws.global}${query}`

      const ws = new WebSocket(wsUrl)

      ws.onopen = () => {
        connectionInProgress = false
        if (!isActive) {
          ws.close()
          return
        }
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

      ws.onerror = () => {
        connectionInProgress = false
      }

      ws.onclose = () => {
        connectionInProgress = false
        setIsConnected(false)

        const currentWs = wsRef.current
        if (currentWs === ws) {
          wsRef.current = null
        }

        if (isActive && reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current++
          reconnectTimeoutRef.current = window.setTimeout(connect, reconnectDelay)
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
      return false
    }

    if (ws.readyState === WebSocket.CONNECTING) {
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

    return false
  }

  return { isConnected, send }
}
