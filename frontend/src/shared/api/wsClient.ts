type WebSocketFactory = (url: string) => WebSocket

const createSocket: WebSocketFactory = (url) => new WebSocket(url)

let globalSocket: WebSocket | null = null

export function connectToGlobalWs(factory: WebSocketFactory = createSocket): WebSocket {
  if (globalSocket && globalSocket.readyState === WebSocket.OPEN) {
    return globalSocket
  }

  const wsBaseUrl = import.meta.env.VITE_API_BASE_URL ?? location.origin
  const wsUrl = `${wsBaseUrl.replace(/^http/, 'ws')}/ws`
  globalSocket = factory(wsUrl)
  return globalSocket
}

export function getGlobalSocket(): WebSocket | null {
  return globalSocket
}

export function closeGlobalSocket() {
  if (globalSocket) {
    globalSocket.close()
    globalSocket = null
  }
}
