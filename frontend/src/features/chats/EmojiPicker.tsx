import { useEffect, useRef } from 'react'
import { EMOJIS } from '../../shared/constants'

type Props = {
  onSelect: (emoji: string) => void
  onClose: () => void
  toggleRef: React.RefObject<HTMLButtonElement | null>
}

export function EmojiPicker({ onSelect, onClose, toggleRef }: Props) {
  const popoverRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onClickOutside = (event: MouseEvent) => {
      const popover = popoverRef.current
      const toggle = toggleRef.current
      if (!popover || !toggle) return

      if (popover.contains(event.target as Node) || toggle.contains(event.target as Node)) {
        return
      }
      onClose()
    }

    document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [onClose, toggleRef])

  return (
    <div className="emoji-popover glassy" ref={popoverRef}>
      {EMOJIS.map((emoji) => (
        <button
          key={emoji}
          type="button"
          className="emoji-option"
          onClick={() => onSelect(emoji)}
        >
          {emoji}
        </button>
      ))}
    </div>
  )
}
