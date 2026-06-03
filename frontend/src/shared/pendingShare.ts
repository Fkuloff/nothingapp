// One-shot pub/sub bridging an Android share into ChatsPage's chat-picker.
// Mirrors pendingChat: reading clears it, so a cold-start share set before
// ChatsPage subscribes is still delivered exactly once.

type Listener = (text: string) => void

let pending: string | null = null
const listeners = new Set<Listener>()

export function setPendingShare(text: string) {
  pending = text
  for (const l of listeners) l(text)
}

/** A share set before subscription is replayed once, then cleared. */
export function subscribePendingShare(cb: Listener): () => void {
  listeners.add(cb)
  if (pending !== null) {
    const text = pending
    pending = null
    // Defer to avoid firing during component render.
    queueMicrotask(() => cb(text))
  }
  return () => {
    listeners.delete(cb)
  }
}
