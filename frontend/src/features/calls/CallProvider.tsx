import { useCallback, useEffect, useRef, useState } from 'react'

import type { WSEvent, WSMessageAction } from '../../shared/api/types'
import type { CallState } from './CallContext'
import { CallContext } from './CallContext'
import { createRingback, createRingtone } from './ringback'
import { useWebRTC } from './useWebRTC'

const initialState: CallState = {
  status: 'idle',
  callId: null,
  chatId: null,
  otherUserId: null,
  otherUsername: null,
  otherAvatar: null,
  isMuted: false,
  callDuration: 0,
}

export function CallProvider({ children }: { children: React.ReactNode }) {
  const [callState, setCallState] = useState<CallState>(initialState)
  const sendRef = useRef<((data: WSMessageAction) => boolean) | null>(null)
  const callStateRef = useRef(callState)
  const incomingSdpRef = useRef<string | null>(null)

  const webrtc = useWebRTC()

  // Keep ref in sync
  useEffect(() => {
    callStateRef.current = callState
  }, [callState])

  // Ringback tone while outgoing
  useEffect(() => {
    if (callState.status !== 'outgoing') return
    const ringback = createRingback()
    ringback.start()
    return () => ringback.stop()
  }, [callState.status])

  // Ringtone for incoming call
  useEffect(() => {
    if (callState.status !== 'incoming') return
    const ringtone = createRingtone()
    ringtone.start()
    return () => ringtone.stop()
  }, [callState.status])

  // Call duration timer
  useEffect(() => {
    if (callState.status !== 'active') return
    const interval = setInterval(() => {
      setCallState((prev) => ({ ...prev, callDuration: prev.callDuration + 1 }))
    }, 1000)
    return () => clearInterval(interval)
  }, [callState.status])

  // Send hangup when page is about to unload (refresh/close tab)
  useEffect(() => {
    const handleBeforeUnload = () => {
      const state = callStateRef.current
      if (state.status !== 'idle' && state.callId && state.chatId) {
        sendRef.current?.({ action: 'call_hangup', chat_id: state.chatId, call_id: state.callId })
      }
    }
    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [])

  // Auto-hangup on WebRTC connection failure
  useEffect(() => {
    if (webrtc.connectionState === 'failed' || webrtc.connectionState === 'disconnected') {
      if (callStateRef.current.status === 'active') {
        hangup()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [webrtc.connectionState])

  const send = useCallback((data: WSMessageAction): boolean => {
    return sendRef.current?.(data) ?? false
  }, [])

  const registerSend = useCallback((fn: ((data: WSMessageAction) => boolean) | null) => {
    sendRef.current = fn
  }, [])

  const resetState = useCallback(() => {
    setCallState(initialState)
    incomingSdpRef.current = null
  }, [])

  const hangup = useCallback(() => {
    const state = callStateRef.current
    if (state.status !== 'idle' && state.callId && state.chatId) {
      send({ action: 'call_hangup', chat_id: state.chatId, call_id: state.callId })
    }
    webrtc.hangup()
    resetState()
  }, [send, webrtc, resetState])

  const sendIceCandidate = useCallback((chatId: number, callId: string) => {
    return (candidate: string) => {
      send({ action: 'call_ice', chat_id: chatId, call_id: callId, candidate })
    }
  }, [send])

  const initiateCall = useCallback(async (
    chatId: number,
    otherUserId: number,
    otherUsername: string,
    otherAvatar?: string | null,
  ) => {
    if (callStateRef.current.status !== 'idle') return

    const callId = crypto.randomUUID()

    setCallState({
      status: 'outgoing',
      callId,
      chatId,
      otherUserId,
      otherUsername,
      otherAvatar: otherAvatar ?? null,
      isMuted: false,
      callDuration: 0,
    })

    try {
      const sdp = await webrtc.startCall(sendIceCandidate(chatId, callId))
      send({ action: 'call_offer', chat_id: chatId, call_id: callId, sdp, sdp_type: 'offer' })
    } catch {
      webrtc.hangup()
      resetState()
    }
  }, [webrtc, send, sendIceCandidate, resetState])

  const acceptCall = useCallback(async () => {
    const state = callStateRef.current
    if (state.status !== 'incoming' || !incomingSdpRef.current || !state.chatId || !state.callId) return

    try {
      const answerSdp = await webrtc.answerCall(
        incomingSdpRef.current,
        sendIceCandidate(state.chatId, state.callId),
      )
      send({ action: 'call_answer', chat_id: state.chatId, call_id: state.callId, sdp: answerSdp, sdp_type: 'answer' })
      incomingSdpRef.current = null
      setCallState((prev) => ({ ...prev, status: 'active' }))
    } catch {
      webrtc.hangup()
      resetState()
    }
  }, [webrtc, send, sendIceCandidate, resetState])

  const rejectCall = useCallback(() => {
    const state = callStateRef.current
    if (state.status !== 'incoming' || !state.chatId || !state.callId) return
    send({ action: 'call_reject', chat_id: state.chatId, call_id: state.callId })
    webrtc.hangup()
    resetState()
  }, [send, webrtc, resetState])

  const toggleMute = useCallback(() => {
    webrtc.toggleMute()
    setCallState((prev) => ({ ...prev, isMuted: !prev.isMuted }))
  }, [webrtc])

  const handleCallEvent = useCallback((event: WSEvent, callerInfo?: { username: string; avatar?: string | null }) => {
    if ('error' in event) return
    const state = callStateRef.current

    switch (event.action) {
      case 'call_offer': {
        if (state.status !== 'idle') {
          send({ action: 'call_reject', chat_id: event.chat_id, call_id: event.call_id })
          return
        }
        incomingSdpRef.current = event.sdp
        setCallState({
          status: 'incoming',
          callId: event.call_id,
          chatId: event.chat_id,
          otherUserId: event.user_id,
          otherUsername: callerInfo?.username ?? null,
          otherAvatar: callerInfo?.avatar ?? null,
          isMuted: false,
          callDuration: 0,
        })
        break
      }

      case 'call_answer': {
        if (state.status !== 'outgoing' || state.callId !== event.call_id) return
        webrtc.handleRemoteAnswer(event.sdp).then(() => {
          setCallState((prev) => ({ ...prev, status: 'active' }))
        }).catch(() => {
          hangup()
        })
        break
      }

      case 'call_ice': {
        if (!state.callId || state.callId !== event.call_id) return
        webrtc.handleRemoteIceCandidate(event.candidate).catch(console.error)
        break
      }

      case 'call_hangup':
      case 'call_reject': {
        if (state.callId !== event.call_id) return
        webrtc.hangup()
        resetState()
        break
      }
    }
  }, [webrtc, send, hangup, resetState])

  return (
    <CallContext.Provider value={{
      callState,
      initiateCall,
      acceptCall,
      rejectCall,
      hangup,
      toggleMute,
      handleCallEvent,
      registerSend,
    }}>
      {children}
    </CallContext.Provider>
  )
}
