import { createContext, useContext } from 'react'

import type { WSEvent, WSMessageAction } from '../../shared/api/types'

// 'connecting' = callee tapped a doorbell push and is waiting for the caller's
// fresh offer (the re-offer handshake) before the call goes active.
export type CallStatus = 'idle' | 'outgoing' | 'incoming' | 'active' | 'connecting'

export type CallState = {
  status: CallStatus
  callId: string | null
  chatId: number | null
  otherUserId: number | null
  otherUsername: string | null
  otherAvatar: string | null
  isMuted: boolean
  callDuration: number
}

type CallContextType = {
  callState: CallState
  initiateCall: (chatId: number, otherUserId: number, otherUsername: string, otherAvatar?: string | null) => void
  acceptCall: () => void
  rejectCall: () => void
  hangup: () => void
  toggleMute: () => void
  handleCallEvent: (event: WSEvent, callerInfo?: { username: string; avatar?: string | null }) => void
  registerSend: (fn: ((data: WSMessageAction) => boolean) | null) => void
}

export const CallContext = createContext<CallContextType | null>(null)

export function useCallContext(): CallContextType {
  const ctx = useContext(CallContext)
  if (!ctx) throw new Error('useCallContext must be used within CallProvider')
  return ctx
}
