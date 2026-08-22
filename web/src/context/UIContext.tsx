import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import type { Post } from '../types'

export type AuthMode = 'login' | 'register'
export type ToastTone = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  tone: ToastTone
  title: string
  message?: string
}

export type ComposerState =
  | { mode: 'create' }
  | { mode: 'edit'; post: Post }
  | null

interface UIContextValue {
  authMode: AuthMode | null
  composer: ComposerState
  toasts: ToastItem[]
  contentVersion: number
  openAuth: (mode?: AuthMode) => void
  closeAuth: () => void
  openComposer: (post?: Post) => void
  closeComposer: () => void
  notify: (tone: ToastTone, title: string, message?: string) => void
  dismissToast: (id: number) => void
  refreshContent: () => void
}

const UIContext = createContext<UIContextValue | null>(null)

export function UIProvider({ children }: { children: ReactNode }) {
  const [authMode, setAuthMode] = useState<AuthMode | null>(null)
  const [composer, setComposer] = useState<ComposerState>(null)
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const [contentVersion, setContentVersion] = useState(0)
  const nextToastId = useRef(1)

  const openAuth = useCallback((mode: AuthMode = 'login') => setAuthMode(mode), [])
  const closeAuth = useCallback(() => setAuthMode(null), [])
  const openComposer = useCallback((post?: Post) => {
    setComposer(post ? { mode: 'edit', post } : { mode: 'create' })
  }, [])
  const closeComposer = useCallback(() => setComposer(null), [])

  const dismissToast = useCallback((id: number) => {
    setToasts((items) => items.filter((item) => item.id !== id))
  }, [])

  const notify = useCallback(
    (tone: ToastTone, title: string, message?: string) => {
      const id = nextToastId.current++
      setToasts((items) => [...items.slice(-2), { id, tone, title, message }])
      window.setTimeout(() => dismissToast(id), 4200)
    },
    [dismissToast],
  )

  const refreshContent = useCallback(() => {
    setContentVersion((value) => value + 1)
  }, [])

  const value = useMemo<UIContextValue>(
    () => ({
      authMode,
      composer,
      toasts,
      contentVersion,
      openAuth,
      closeAuth,
      openComposer,
      closeComposer,
      notify,
      dismissToast,
      refreshContent,
    }),
    [
      authMode,
      closeAuth,
      closeComposer,
      composer,
      contentVersion,
      dismissToast,
      notify,
      openAuth,
      openComposer,
      refreshContent,
      toasts,
    ],
  )

  return <UIContext.Provider value={value}>{children}</UIContext.Provider>
}

export function useUI(): UIContextValue {
  const value = useContext(UIContext)
  if (!value) throw new Error('useUI 必须在 UIProvider 内使用')
  return value
}
