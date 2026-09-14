import { act, render } from '@testing-library/react'
import { useEffect } from 'react'
import { describe, expect, it, vi } from 'vitest'
import {
  AuthProvider,
  useAuth,
  type LogoutResult,
} from '../src/context/AuthContext'
import { api, TOKEN_STORAGE_KEY } from '../src/lib/api'
import { USER_STATUS, type User } from '../src/types'

const user: User = {
  id: 7,
  create_time: '2026-09-14T00:00:00Z',
  update_time: '2026-09-14T00:00:00Z',
  username: 'tester',
  nickname: 'Tester',
  status: USER_STATUS.NORMAL,
  introduction: '',
  avatar_url: '',
  is_admin: false,
}

type AuthValue = ReturnType<typeof useAuth>

function renderAuth(): { current: () => AuthValue } {
  let value: AuthValue | null = null

  function Probe() {
    const auth = useAuth()
    useEffect(() => {
      value = auth
    }, [auth])
    return null
  }

  render(<AuthProvider><Probe /></AuthProvider>)
  return {
    current() {
      if (!value) throw new Error('认证上下文尚未初始化')
      return value
    },
  }
}

describe('AuthProvider public session contract', () => {
  it('commits token and user together after the profile succeeds', async () => {
    vi.spyOn(api, 'login').mockResolvedValue({ token: 'new-token' })
    let resolveProfile!: (profile: { user: User }) => void
    const profilePromise = new Promise<{ user: User }>((resolve) => {
      resolveProfile = resolve
    })
    const getMe = vi.spyOn(api, 'getMe').mockReturnValue(profilePromise)
    const auth = renderAuth()

    const loginPromise = auth.current().login({ username: 'tester', password: 'secret' })
    await vi.waitFor(() => expect(getMe).toHaveBeenCalledTimes(1))

    expect(auth.current().token).toBeNull()
    expect(auth.current().user).toBeNull()
    expect(auth.current().isAuthenticated).toBe(false)
    expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull()

    await act(async () => {
      resolveProfile({ user })
      await loginPromise
    })

    expect(auth.current().token).toBe('new-token')
    expect(auth.current().user).toEqual(user)
    expect(auth.current().isAuthenticated).toBe(true)
    expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBe('new-token')
  })

  it('does not leave a partial login when loading the profile fails', async () => {
    const profileError = new Error('profile unavailable')
    vi.spyOn(api, 'login').mockResolvedValue({ token: 'candidate-token' })
    vi.spyOn(api, 'getMe').mockRejectedValue(profileError)
    const auth = renderAuth()

    await expect(auth.current().login({ username: 'tester', password: 'secret' }))
      .rejects.toBe(profileError)

    expect(auth.current().token).toBeNull()
    expect(auth.current().user).toBeNull()
    expect(auth.current().isAuthenticated).toBe(false)
    expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull()
  })

  it('clears the local session even when server-side revocation fails', async () => {
    vi.spyOn(api, 'login').mockResolvedValue({ token: 'active-token' })
    vi.spyOn(api, 'getMe').mockResolvedValue({ user })
    const revokeError = new Error('server unavailable')
    const auth = renderAuth()

    await act(async () => {
      await auth.current().login({ username: 'tester', password: 'secret' })
    })
    vi.spyOn(api, 'logout').mockImplementation(async () => {
      expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull()
      throw revokeError
    })

    let result: LogoutResult | undefined
    await act(async () => {
      result = await auth.current().logout()
    })

    expect(result).toEqual({
      localSessionCleared: true,
      serverRevocationConfirmed: false,
      error: revokeError,
    })
    expect(auth.current().token).toBeNull()
    expect(auth.current().user).toBeNull()
    expect(auth.current().isAuthenticated).toBe(false)
    expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull()
  })
})
