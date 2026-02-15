import './EmojiPicker.css'

import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { CloseIcon } from '../../shared/components/Icons'
import type { EmojiCategory } from './emojiData'
import { EMOJI_CATEGORIES, EMOJI_KEYWORDS } from './emojiData'
import { useRecentEmojis } from './useRecentEmojis'

// Height per row (36px button + 1px gap)
const ROW_H = 37
const COLS = 8
const TITLE_H = 30

function estimateSectionHeight(count: number) {
  return TITLE_H + Math.ceil(count / COLS) * ROW_H
}

// --- Lazy section: renders placeholder until visible ---
const LazySection = memo(function LazySection({
  id,
  category,
  onEmojiClick,
  onSectionRef,
}: {
  id: string
  category: { name: string; emojis: string[] }
  onEmojiClick: (emoji: string) => void
  onSectionRef: (id: string, el: HTMLDivElement | null) => void
}) {
  const ref = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) setVisible(true)
      },
      { rootMargin: '200px 0px' },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  const setRef = useCallback(
    (el: HTMLDivElement | null) => {
      (ref as React.MutableRefObject<HTMLDivElement | null>).current = el
      onSectionRef(id, el)
    },
    [id, onSectionRef],
  )

  const placeholderH = estimateSectionHeight(category.emojis.length)

  return (
    <div className="emoji-picker__section" ref={setRef}>
      <div className="emoji-picker__section-title">{category.name}</div>
      {visible ? (
        <div className="emoji-picker__grid">
          {category.emojis.map((emoji) => (
            <button
              key={emoji}
              type="button"
              className="emoji-btn"
              onClick={() => onEmojiClick(emoji)}
            >
              {emoji}
            </button>
          ))}
        </div>
      ) : (
        <div style={{ height: placeholderH - TITLE_H }} />
      )}
    </div>
  )
})

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
  }, [searchQuery])

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

  // Pre-build recent category object for LazySection
  const recentCategory = useMemo<EmojiCategory | null>(
    () =>
      recentEmojis.length > 0
        ? { id: 'recent', name: 'Недавние', icon: '🕐', emojis: recentEmojis }
        : null,
    [recentEmojis],
  )

  return (
    <div className="emoji-picker">
      {/* Header with search */}
      <div className="emoji-picker__header">
        <div className="emoji-picker__search">
          <svg className="emoji-picker__search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" width="16" height="16">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.35-4.35" />
          </svg>
          <input
            type="text"
            className="emoji-picker__search-input"
            placeholder="Поиск эмодзи..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button
              type="button"
              className="emoji-picker__search-clear"
              onClick={() => setSearchQuery('')}
            >
              <CloseIcon size={14} />
            </button>
          )}
        </div>
        <button type="button" className="emoji-picker__close" onClick={onClose} title="Закрыть">
          <CloseIcon size={18} />
        </button>
      </div>

      {/* Category tabs */}
      {!isSearching && (
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
      )}

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
            {recentCategory && (
              <LazySection
                id="recent"
                category={recentCategory}
                onEmojiClick={handleEmojiClick}
                onSectionRef={setSectionRef}
              />
            )}
            {EMOJI_CATEGORIES.map((cat) => (
              <LazySection
                key={cat.id}
                id={cat.id}
                category={cat}
                onEmojiClick={handleEmojiClick}
                onSectionRef={setSectionRef}
              />
            ))}
          </>
        )}
      </div>
    </div>
  )
}
