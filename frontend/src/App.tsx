import './App.css'

import { useCallback, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'

import { ActiveCallOverlay } from './features/calls/ActiveCallOverlay'
import { useCallContext } from './features/calls/CallContext'
import { IncomingCallModal } from './features/calls/IncomingCallModal'
import { SlideMenu } from './features/menu/SlideMenu'

export type OutletContextType = {
  menuOpen: boolean
  setMenuOpen: (open: boolean) => void
  onChatSelectedRef: React.MutableRefObject<((chatId: number) => void) | null>
}

export default function AppLayout() {
  const [menuOpen, setMenuOpen] = useState(false)
  const onChatSelectedRef = useRef<((chatId: number) => void) | null>(null)
  const { callState, acceptCall, rejectCall, hangup, toggleMute } = useCallContext()

  const handleChatSelected = useCallback((chatId: number) => {
    onChatSelectedRef.current?.(chatId)
  }, [])

  return (
    <div className="telegram-layout">
      <SlideMenu isOpen={menuOpen} onClose={() => setMenuOpen(false)} onChatSelected={handleChatSelected} />
      <div className="telegram-main">
        <Outlet context={{ menuOpen, setMenuOpen, onChatSelectedRef } satisfies OutletContextType} />
      </div>

      {callState.status === 'incoming' && (
        <IncomingCallModal
          callerName={callState.otherUsername || 'Неизвестный'}
          callerAvatar={callState.otherAvatar}
          onAccept={acceptCall}
          onReject={rejectCall}
        />
      )}

      {(callState.status === 'outgoing' || callState.status === 'active') && (
        <ActiveCallOverlay
          otherUsername={callState.otherUsername || ''}
          otherAvatar={callState.otherAvatar}
          duration={callState.callDuration}
          isMuted={callState.isMuted}
          status={callState.status}
          onToggleMute={toggleMute}
          onHangup={hangup}
        />
      )}
    </div>
  )
}
