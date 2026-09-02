import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { api, shouldClearToken, tokenStore } from '../lib/api'
import { USER_STATUS, type LoginRequest, type RegisterRequest, type User } from '../types'

interface AuthContextValue {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  canWrite: boolean
  isBootstrapping: boolean
  login: (input: LoginRequest) => Promise<User>
  register: (input: RegisterRequest) => Promise<User | null>
  logout: () => Promise<boolean>
  refreshUser: () => Promise<User | null>
  isCurrentSession: (expectedToken: string) => boolean
  clearSession: (expectedToken?: string) => boolean
  handleSessionError: (error: unknown, expectedToken: string) => boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => tokenStore.get())
  const [user, setUser] = useState<User | null>(null)
  const [isBootstrapping, setIsBootstrapping] = useState(Boolean(token))
  const tokenRef = useRef(token)

  const isCurrentSession = useCallback(
    (expectedToken: string) => tokenRef.current === expectedToken,
    [],
  )

  const clearSession = useCallback((expectedToken?: string) => {
    if (expectedToken !== undefined && !isCurrentSession(expectedToken)) {
      return false
    }
    tokenStore.clear(expectedToken)
    tokenRef.current = null
    setToken(null)
    setUser(null)
    setIsBootstrapping(false)
    return true
  }, [isCurrentSession])

  const handleSessionError = useCallback(
    (error: unknown, expectedToken: string) => {
      if (!shouldClearToken(error)) return false
      return clearSession(expectedToken)
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
      if (!isCurrentSession(token)) return null
      setUser(data.user)
      return data.user
    } catch (error) {
      handleSessionError(error, token)
      throw error
    }
  }, [handleSessionError, isCurrentSession, token])

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
        if (!cancelled && isCurrentSession(token)) setUser(data.user)
      })
      .catch((error: unknown) => {
        if (!cancelled && shouldClearToken(error)) clearSession(token)
      })
      .finally(() => {
        if (!cancelled && isCurrentSession(token)) setIsBootstrapping(false)
      })

    return () => {
      cancelled = true
    }
  }, [clearSession, isCurrentSession, token])

  const login = useCallback(async (input: LoginRequest) => {
    const data = await api.login(input)
    tokenStore.set(data.token)
    tokenRef.current = data.token
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
    if (!currentToken) return false
    await api.logout(currentToken)
    return clearSession(currentToken)
  }, [clearSession, token])

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      user,
      isAuthenticated: Boolean(token && user),
      canWrite: user?.status === USER_STATUS.NORMAL,
      isBootstrapping,
      login,
      register,
      logout,
      refreshUser,
      isCurrentSession,
      clearSession,
      handleSessionError,
    }),
    [
      clearSession,
      handleSessionError,
      isBootstrapping,
      isCurrentSession,
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
