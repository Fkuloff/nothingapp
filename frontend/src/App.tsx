import './App.css'

import { useState } from 'react'
import { Outlet } from 'react-router-dom'

import { SlideMenu } from './features/menu/SlideMenu'

export type OutletContextType = {
  menuOpen: boolean
  setMenuOpen: (open: boolean) => void
}

export default function AppLayout() {
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div className="telegram-layout">
      <SlideMenu isOpen={menuOpen} onClose={() => setMenuOpen(false)} />
      <div className="telegram-main">
        <Outlet context={{ menuOpen, setMenuOpen } satisfies OutletContextType} />
      </div>
    </div>
  )
}
