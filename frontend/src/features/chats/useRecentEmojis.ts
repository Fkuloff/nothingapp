import { useCallback, useState } from 'react'

const STORAGE_KEY = 'recent_emojis'
const MAX_RECENT = 32

export function useRecentEmojis() {
  const [recentEmojis, setRecentEmojis] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      return stored ? JSON.parse(stored) : []
    } catch {
      return []
    }
  })

  const addRecentEmoji = useCallback((emoji: string) => {
    setRecentEmojis((prev) => {
      const filtered = prev.filter((e) => e !== emoji)
      const updated = [emoji, ...filtered].slice(0, MAX_RECENT)
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
      } catch {
        // localStorage may be full or unavailable (private browsing)
      }
      return updated
    })
  }, [])

  return { recentEmojis, addRecentEmoji }
}
