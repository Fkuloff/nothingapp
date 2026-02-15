import { useCallback,useEffect, useState } from 'react'

/**
 * Hook for inline confirmation pattern (e.g. "Delete contact?" with auto-reset).
 * Returns confirming state and toggle functions.
 *
 * @param timeout - Auto-reset delay in ms (default 5000)
 */
export function useConfirmAction(timeout = 5000) {
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    if (!confirming) return
    const timer = setTimeout(() => setConfirming(false), timeout)
    return () => clearTimeout(timer)
  }, [confirming, timeout])

  const startConfirm = useCallback(() => setConfirming(true), [])
  const cancelConfirm = useCallback(() => setConfirming(false), [])

  return { confirming, startConfirm, cancelConfirm } as const
}
