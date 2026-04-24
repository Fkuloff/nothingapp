import { createContext, type ReactNode, useCallback, useContext, useEffect, useRef } from 'react'

import { isNative } from '../platform'

/**
 * Handler returns true if the back press was consumed (stop here),
 * false to let the next handler down the stack try.
 */
type BackHandler = () => boolean

type BackStackApi = {
  push: (handler: BackHandler) => void
  remove: (handler: BackHandler) => void
}

const BackStackContext = createContext<BackStackApi | null>(null)

export function AndroidBackProvider({ children }: { children: ReactNode }) {
  const stackRef = useRef<BackHandler[]>([])

  const push = useCallback((handler: BackHandler) => {
    stackRef.current.push(handler)
  }, [])

  const remove = useCallback((handler: BackHandler) => {
    const stack = stackRef.current
    for (let i = stack.length - 1; i >= 0; i--) {
      if (stack[i] === handler) {
        stack.splice(i, 1)
        return
      }
    }
  }, [])

  useEffect(() => {
    if (!isNative()) return

    let removed = false
    let listener: { remove: () => void } | undefined

    const setup = async () => {
      const { App } = await import('@capacitor/app')
      const handle = await App.addListener('backButton', ({ canGoBack }) => {
        for (let i = stackRef.current.length - 1; i >= 0; i--) {
          if (stackRef.current[i]()) return
        }
        if (canGoBack) {
          window.history.back()
        } else {
          App.exitApp()
        }
      })
      if (removed) {
        handle.remove()
      } else {
        listener = handle
      }
    }

    void setup()

    return () => {
      removed = true
      listener?.remove()
    }
  }, [])

  return <BackStackContext.Provider value={{ push, remove }}>{children}</BackStackContext.Provider>
}

/**
 * Register a hardware back-button handler (Android). Noop on web and when enabled is false.
 *
 * Handlers are stacked LIFO: the most recently mounted (e.g. topmost modal) gets the press first.
 * Return true from the handler if you consumed the event, false to pass down the stack.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useAndroidBack(handler: BackHandler, enabled: boolean) {
  const ctx = useContext(BackStackContext)
  const handlerRef = useRef(handler)

  // Keep the ref up to date without accessing it during render.
  useEffect(() => {
    handlerRef.current = handler
  })

  useEffect(() => {
    if (!enabled || !ctx) return
    const wrapped: BackHandler = () => handlerRef.current()
    ctx.push(wrapped)
    return () => ctx.remove(wrapped)
  }, [enabled, ctx])
}
