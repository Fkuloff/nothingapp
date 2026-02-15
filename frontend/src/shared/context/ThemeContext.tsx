import { type ReactNode,useEffect, useState } from 'react'

import { type Theme,ThemeContext } from './themeTypes'

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => {
    const saved = localStorage.getItem('theme') as Theme
    return saved || 'dark'
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }, [theme])

  const toggleTheme = () => setThemeState(prev => prev === 'dark' ? 'light' : 'dark')

  const setTheme = (newTheme: Theme) => setThemeState(newTheme)

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}
