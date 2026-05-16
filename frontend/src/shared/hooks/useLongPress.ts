import { useCallback, useRef } from 'react'

type LongPressHandlers = {
  onMouseDown: (e: React.MouseEvent) => void
  onMouseUp: () => void
  onMouseLeave: () => void
  onTouchStart: () => void
  onTouchEnd: () => void
  onTouchMove: () => void
  onContextMenu: (e: React.MouseEvent) => void
  onClickCapture: (e: React.MouseEvent) => void
}

type Options = {
  threshold?: number
}

/**
 * Distinguish a long-press from a regular tap.
 *
 * Pattern: long-press fires after `threshold` ms of holding. After firing, the
 * following click event is swallowed so the row's plain onClick doesn't also run.
 *
 * Returns spread-into-element handlers. The element should also have its own onClick
 * for the short-tap primary action — it will be skipped automatically after a long-press.
 */
export function useLongPress(onLongPress: () => void, options: Options = {}): LongPressHandlers {
  const threshold = options.threshold ?? 500
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const firedRef = useRef(false)

  const start = useCallback(() => {
    firedRef.current = false
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => {
      firedRef.current = true
      onLongPress()
    }, threshold)
  }, [onLongPress, threshold])

  const cancel = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }, [])

  return {
    onMouseDown: (e) => { if (e.button === 0) start() },
    onMouseUp: cancel,
    onMouseLeave: cancel,
    onTouchStart: start,
    onTouchEnd: cancel,
    onTouchMove: cancel,
    onContextMenu: (e) => {
      e.preventDefault()
      cancel()
      firedRef.current = true
      onLongPress()
    },
    onClickCapture: (e) => {
      if (firedRef.current) {
        e.stopPropagation()
        e.preventDefault()
        firedRef.current = false
      }
    },
  }
}
