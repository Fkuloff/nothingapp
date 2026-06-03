// Tiny one-shot pub/sub for "the user shared this text into the app" — bridges
// an Android ACTION_SEND (system share-sheet) into ChatsPage, which opens a
// chat-picker. Mirrors pendingChat / pendingCall.
//
// useShareTarget (driven by the native ShareTarget plugin) calls
// setPendingShare(text); ChatsPage subscribes and opens the "Поделиться в…"
// modal. One-shot: reading it clears it, so a single share is offered exactly
// once even if ChatsPage subscribes after the share already arrived (cold start).

type Listener = (text: string) => void

let pending: string | null = null
const listeners = new Set<Listener>()

export function setPendingShare(text: string) {
  pending = text
  for (const l of listeners) l(text)
}

/**
 * Subscribe to pending shared text. If a share was set before subscription, it
 * is delivered immediately and then cleared. Returns an unsubscribe function.
 */
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
