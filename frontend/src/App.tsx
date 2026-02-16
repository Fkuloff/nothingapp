import './App.css'

import { useCallback, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'

import { SlideMenu } from './features/menu/SlideMenu'

export type OutletContextType = {
  menuOpen: boolean
  setMenuOpen: (open: boolean) => void
  onChatSelectedRef: React.MutableRefObject<((chatId: number) => void) | null>
}

export default function AppLayout() {
  const [menuOpen, setMenuOpen] = useState(false)
  const onChatSelectedRef = useRef<((chatId: number) => void) | null>(null)

  const handleChatSelected = useCallback((chatId: number) => {
    onChatSelectedRef.current?.(chatId)
  }, [])

  return (
    <div className="telegram-layout">
      <SlideMenu isOpen={menuOpen} onClose={() => setMenuOpen(false)} onChatSelected={handleChatSelected} />
      <div className="telegram-main">
        <Outlet context={{ menuOpen, setMenuOpen, onChatSelectedRef } satisfies OutletContextType} />
      </div>
    </div>
  )
}
