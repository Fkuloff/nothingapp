// Tiny pub/sub for "a call doorbell was tapped" — bridges an FCM call
// notification tap into CallProvider without going through the router. Mirrors
// pendingChat. The callee taps the "Входящий звонок" notification, the app cold-
// or warm-starts, and CallProvider replays this to emit `call_ready` once the
// WebSocket connects, prompting the caller to send a fresh offer.

export type PendingCall = {
  callId: string
  chatId: number
  callerId: number
}

type Listener = (call: PendingCall) => void

let pending: PendingCall | null = null
const listeners = new Set<Listener>()

export function setPendingCall(call: PendingCall) {
  pending = call
  for (const l of listeners) l(call)
}

/**
 * Subscribe to pending call doorbells. If one was set before subscription, it is
 * delivered immediately and then cleared (one-shot, so a tap rings once).
 * Returns an unsubscribe function.
 */
export function subscribePendingCall(cb: Listener): () => void {
  listeners.add(cb)
  if (pending !== null) {
    const call = pending
    pending = null
    // Defer to avoid firing during component render.
    queueMicrotask(() => cb(call))
  }
  return () => {
    listeners.delete(cb)
  }
}
