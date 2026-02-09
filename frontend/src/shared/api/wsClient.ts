export type WebSocketFactory = (url: string) => WebSocket

const createSocket: WebSocketFactory = (url) => new WebSocket(url)

export function connectToChat(chatId: number, factory: WebSocketFactory = createSocket) {
  const wsBaseUrl = import.meta.env.VITE_API_BASE_URL ?? location.origin
  const wsUrl = `${wsBaseUrl.replace(/^http/, 'ws')}/ws/chat/${chatId}`
  return factory(wsUrl)
}
