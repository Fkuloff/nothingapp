// Tiny pub/sub for "open this chat next" — used to bridge FCM notifications into ChatsPage
// without going through the router (HashRouter doesn't reliably pass search params on native).
//
// FCM hook calls setPendingChat(id); ChatsPage subscribes and flips its activeChatId.
// The value is one-shot: reading it clears it, so the chat only opens once per notification tap.

type Listener = (chatId: number) => void

let pending: number | null = null
const listeners = new Set<Listener>()

export function setPendingChat(chatId: number) {
  pending = chatId
  for (const l of listeners) l(chatId)
}

/**
 * Subscribe to pending chat ids. If a chat was set before subscription, it is delivered
 * immediately and then cleared. Returns an unsubscribe function.
 */
export function subscribePendingChat(cb: Listener): () => void {
  listeners.add(cb)
  if (pending !== null) {
    const id = pending
    pending = null
    // Defer to avoid firing during component render.
    queueMicrotask(() => cb(id))
  }
  return () => {
    listeners.delete(cb)
  }
}
