import './EmojiPicker.css'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { CloseIcon } from '../../shared/components/Icons'
import type { EmojiCategory } from './emojiData'
import { EMOJI_CATEGORIES, EMOJI_KEYWORDS } from './emojiData'
import { useRecentEmojis } from './useRecentEmojis'

type Props = {
  onSelect: (emoji: string) => void
  onClose: () => void
}

export function EmojiPicker({ onSelect, onClose }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const sectionRefs = useRef<Map<string, HTMLDivElement>>(new Map())
  const [activeCategory, setActiveCategory] = useState('smileys')
  const [searchQuery, setSearchQuery] = useState('')
  const { recentEmojis, addRecentEmoji } = useRecentEmojis()

  // Escape to close
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  // Scroll-spy (throttled via rAF)
  useEffect(() => {
    const container = scrollRef.current
    if (!container) return

    let rafId = 0
    const handleScroll = () => {
      if (rafId) return
      rafId = requestAnimationFrame(() => {
        rafId = 0
        const top = container.scrollTop + 10
        let active = 'smileys'
        for (const [id, el] of sectionRefs.current.entries()) {
          if (el.offsetTop <= top) {
            active = id
          }
        }
        setActiveCategory((prev) => (prev === active ? prev : active))
      })
    }

    container.addEventListener('scroll', handleScroll, { passive: true })
    return () => {
      container.removeEventListener('scroll', handleScroll)
      if (rafId) cancelAnimationFrame(rafId)
    }
  }, [])

  // Search filter
  const filteredEmojis = useMemo(() => {
    const q = searchQuery.toLowerCase().trim()
    if (!q) return []
    const results: string[] = []
    for (const cat of EMOJI_CATEGORIES) {
      for (const emoji of cat.emojis) {
        const kw = EMOJI_KEYWORDS[emoji]
        if (kw?.some((k) => k.includes(q))) {
          results.push(emoji)
        }
      }
    }
    return results
  }, [searchQuery])

  const handleCategoryClick = useCallback((id: string) => {
    const el = sectionRefs.current.get(id)
    const container = scrollRef.current
    if (el && container) {
      container.scrollTop = el.offsetTop
      setActiveCategory(id)
    }
  }, [])

  const handleEmojiClick = useCallback(
    (emoji: string) => {
      addRecentEmoji(emoji)
      onSelect(emoji)
    },
    [addRecentEmoji, onSelect],
  )

  const setSectionRef = useCallback((id: string, el: HTMLDivElement | null) => {
    if (el) {
      sectionRefs.current.set(id, el)
    }
  }, [])

  const isSearching = searchQuery.trim().length > 0

  // Pre-build recent category object
  const recentCategory = useMemo<EmojiCategory | null>(
    () =>
      recentEmojis.length > 0
        ? { id: 'recent', name: 'Недавние', icon: '🕐', emojis: recentEmojis }
        : null,
    [recentEmojis],
  )

  // All categories to render (recent + standard)
  const allCategories = useMemo(() => {
    const cats: EmojiCategory[] = []
    if (recentCategory) cats.push(recentCategory)
    cats.push(...EMOJI_CATEGORIES)
    return cats
  }, [recentCategory])

  return (
    <div className="emoji-picker">
      {/* Category tabs row */}
      <div className="emoji-picker__toolbar">
        <div className="emoji-picker__tabs">
          {recentCategory && (
            <button
              type="button"
              className={`emoji-picker__tab${activeCategory === 'recent' ? ' active' : ''}`}
              onClick={() => handleCategoryClick('recent')}
              title="Недавние"
            >
              🕐
            </button>
          )}
          {EMOJI_CATEGORIES.map((cat) => (
            <button
              key={cat.id}
              type="button"
              className={`emoji-picker__tab${activeCategory === cat.id ? ' active' : ''}`}
              onClick={() => handleCategoryClick(cat.id)}
              title={cat.name}
            >
              {cat.icon}
            </button>
          ))}
        </div>
        <button type="button" className="emoji-picker__close-btn" onClick={onClose} title="Закрыть">
          <CloseIcon size={16} />
        </button>
      </div>

      {/* Search bar */}
      <div className="emoji-picker__search-bar">
        <svg className="emoji-picker__search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="14" height="14" aria-hidden="true">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.35-4.35" />
        </svg>
        <input
          type="text"
          className="emoji-picker__search-input"
          placeholder="Поиск"
          aria-label="Поиск эмодзи"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        {searchQuery && (
          <button
            type="button"
            className="emoji-picker__search-clear"
            aria-label="Очистить поиск"
            onClick={() => setSearchQuery('')}
          >
            <CloseIcon size={12} />
          </button>
        )}
      </div>

      {/* Scrollable content */}
      <div className="emoji-picker__scroll" ref={scrollRef}>
        {isSearching ? (
          <>
            {filteredEmojis.length > 0 ? (
              <div className="emoji-picker__section">
                <div className="emoji-picker__grid">
                  {filteredEmojis.map((emoji) => (
                    <button
                      key={emoji}
                      type="button"
                      className="emoji-btn"
                      onClick={() => handleEmojiClick(emoji)}
                    >
                      {emoji}
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="emoji-picker__empty">Ничего не найдено</div>
            )}
          </>
        ) : (
          <>
            {allCategories.map((cat) => (
              <div
                key={cat.id}
                className="emoji-picker__section"
                ref={(el) => setSectionRef(cat.id, el)}
              >
                <div className="emoji-picker__section-title">{cat.name}</div>
                <div className="emoji-picker__grid">
                  {cat.emojis.map((emoji) => (
                    <button
                      key={emoji}
                      type="button"
                      className="emoji-btn"
                      onClick={() => handleEmojiClick(emoji)}
                    >
                      {emoji}
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  )
}
