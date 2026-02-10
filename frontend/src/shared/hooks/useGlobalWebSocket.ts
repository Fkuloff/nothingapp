import { useEffect, useRef, useState, useCallback } from 'react'
import { endpoints } from '../api/endpoints'
import { getAuthToken } from '../api/httpClient'
import type { WSMessageAction, WSEvent } from '../api/types'

type UseGlobalWebSocketProps = {
  onMessage: (event: WSEvent) => void
  enabled?: boolean
}

export function useGlobalWebSocket({ onMessage, enabled = true }: UseGlobalWebSocketProps) {
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | undefined>(undefined)
  const reconnectAttemptsRef = useRef(0)
  const onMessageRef = useRef(onMessage)

  const maxReconnectAttempts = 10
  const reconnectDelay = 2000

  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  useEffect(() => {
    if (!enabled) {
      return
    }

    let isActive = true
    let connectionInProgress = false

    const connect = () => {
      if (!isActive || connectionInProgress) return

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
        console.log('Global WebSocket connected')
        setIsConnected(true)
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as WSEvent
          onMessageRef.current(data)
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

        if (wsRef.current === ws) {
          wsRef.current = null
        }

        if (isActive && reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current++
          reconnectTimeoutRef.current = window.setTimeout(connect, reconnectDelay)
        }
      }

      wsRef.current = ws
    }

    const initTimeout = window.setTimeout(connect, 100)

    return () => {
      isActive = false
      window.clearTimeout(initTimeout)
      if (reconnectTimeoutRef.current) {
        window.clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
      setIsConnected(false)
    }
  }, [enabled])

  const send = useCallback((data: WSMessageAction) => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return false
    }
    ws.send(JSON.stringify(data))
    return true
  }, [])

  return { isConnected, send }
}
