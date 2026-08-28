import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { api, shouldClearToken, tokenStore } from '../lib/api'
import type { LoginRequest, RegisterRequest, User } from '../types'

interface AuthContextValue {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  isBootstrapping: boolean
  login: (input: LoginRequest) => Promise<User>
  register: (input: RegisterRequest) => Promise<User | null>
  logout: () => Promise<void>
  refreshUser: () => Promise<User | null>
  clearSession: () => void
  handleSessionError: (error: unknown) => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => tokenStore.get())
  const [user, setUser] = useState<User | null>(null)
  const [isBootstrapping, setIsBootstrapping] = useState(Boolean(token))

  const clearSession = useCallback(() => {
    tokenStore.clear()
    setToken(null)
    setUser(null)
    setIsBootstrapping(false)
  }, [])

  const handleSessionError = useCallback(
    (error: unknown) => {
      if (!shouldClearToken(error)) return false
      clearSession()
      return true
    },
    [clearSession],
  )

  const refreshUser = useCallback(async () => {
    if (!token) {
      setUser(null)
      return null
    }

    try {
      const data = await api.getMe(token)
      setUser(data.user)
      return data.user
    } catch (error) {
      handleSessionError(error)
      throw error
    }
  }, [handleSessionError, token])

  useEffect(() => {
    if (!token) {
      setIsBootstrapping(false)
      return
    }

    let cancelled = false
    setIsBootstrapping(true)
    api
      .getMe(token)
      .then((data) => {
        if (!cancelled) setUser(data.user)
      })
      .catch((error: unknown) => {
        if (!cancelled && shouldClearToken(error)) clearSession()
      })
      .finally(() => {
        if (!cancelled) setIsBootstrapping(false)
      })

    return () => {
      cancelled = true
    }
  }, [clearSession, token])

  const login = useCallback(async (input: LoginRequest) => {
    const data = await api.login(input)
    tokenStore.set(data.token)
    setToken(data.token)
    const profile = await api.getMe(data.token)
    setUser(profile.user)
    if (!profile.user) throw new Error('登录成功，但未能读取用户资料')
    return profile.user
  }, [])

  const register = useCallback(
    async (input: RegisterRequest) => {
      await api.register(input)
      try {
        return await login({ username: input.username, password: input.password })
      } catch {
        // 注册已经成功且邀请码已经消费，不能让自动登录失败看起来像注册失败。
        clearSession()
        return null
      }
    },
    [clearSession, login],
  )

  const logout = useCallback(async () => {
    const currentToken = token
    clearSession()
    if (!currentToken) return
    try {
      await api.logout(currentToken)
    } catch {
      // The local session is already cleared; a failed remote revoke must not
      // leave the browser looking signed in.
    }
  }, [clearSession, token])

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      user,
      isAuthenticated: Boolean(token && user),
      isBootstrapping,
      login,
      register,
      logout,
      refreshUser,
      clearSession,
      handleSessionError,
    }),
    [
      clearSession,
      handleSessionError,
      isBootstrapping,
      login,
      logout,
      refreshUser,
      register,
      token,
      user,
    ],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth 必须在 AuthProvider 内使用')
  return value
}
