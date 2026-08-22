import { useEffect, useState, type FormEvent, type MouseEvent } from 'react'
import { Sparkles, X } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useUI, type AuthMode } from '../context/UIContext'
import { getErrorMessage } from '../lib/errors'

export function AuthDialog() {
  const { authMode, closeAuth, openAuth, notify } = useUI()
  const { login, register } = useAuth()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!authMode) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !submitting) closeAuth()
    }
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = ''
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [authMode, closeAuth, submitting])

  useEffect(() => {
    setError('')
  }, [authMode])

  if (!authMode) return null

  const switchMode = (mode: AuthMode) => {
    setError('')
    openAuth(mode)
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')

    const cleanUsername = username.trim()
    if (cleanUsername.length < 3) {
      setError('用户名至少需要 3 个字符')
      return
    }
    if (password.length < 6) {
      setError('密码至少需要 6 个字符')
      return
    }

    setSubmitting(true)
    try {
      const user =
        authMode === 'login'
          ? await login({ username: cleanUsername, password })
          : await register({
              username: cleanUsername,
              password,
              nickname: nickname.trim() || undefined,
            })
      closeAuth()
      setUsername('')
      setPassword('')
      setNickname('')
      notify('success', authMode === 'login' ? '欢迎回来' : '加入成功', `你好，${user.nickname || user.username}`)
    } catch (submitError) {
      setError(getErrorMessage(submitError))
    } finally {
      setSubmitting(false)
    }
  }

  const stopPropagation = (event: MouseEvent<HTMLDivElement>) => event.stopPropagation()

  return (
    <div className="dialog-backdrop" onMouseDown={() => !submitting && closeAuth()}>
      <div
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="auth-title"
        onMouseDown={stopPropagation}
      >
        <div className="dialog__header">
          <div>
            <span className="eyebrow"><Sparkles size={14} aria-hidden="true" />欢迎来到野集</span>
            <h2 id="auth-title">{authMode === 'login' ? '回来坐坐' : '第一次见面'}</h2>
          </div>
          <button className="icon-button" type="button" onClick={closeAuth} aria-label="关闭" disabled={submitting}>
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <div className="dialog__body">
          <div className="auth-tabs" role="tablist" aria-label="登录或注册">
            <button className={`auth-tab${authMode === 'login' ? ' active' : ''}`} type="button" onClick={() => switchMode('login')} role="tab" aria-selected={authMode === 'login'}>
              登录
            </button>
            <button className={`auth-tab${authMode === 'register' ? ' active' : ''}`} type="button" onClick={() => switchMode('register')} role="tab" aria-selected={authMode === 'register'}>
              注册
            </button>
          </div>

          <form onSubmit={onSubmit}>
            <div className="field">
              <label htmlFor="auth-username">用户名 <span>3–64 字符</span></label>
              <input id="auth-username" name="username" autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} placeholder="你的登录用户名" maxLength={64} required autoFocus />
            </div>
            {authMode === 'register' && (
              <div className="field">
                <label htmlFor="auth-nickname">昵称 <span>可选</span></label>
                <input id="auth-nickname" name="nickname" autoComplete="nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder="大家怎么称呼你" maxLength={64} />
              </div>
            )}
            <div className="field">
              <label htmlFor="auth-password">密码 <span>6–64 字符</span></label>
              <input id="auth-password" name="password" type="password" autoComplete={authMode === 'login' ? 'current-password' : 'new-password'} value={password} onChange={(event) => setPassword(event.target.value)} placeholder="输入密码" minLength={6} maxLength={64} required />
            </div>
            {error && <p className="form-error" role="alert">{error}</p>}
            <div className="form-actions">
              <button className="button button--dark button--wide" type="submit" disabled={submitting}>
                {submitting ? '请稍候…' : authMode === 'login' ? '进入野集' : '注册并进入'}
              </button>
            </div>
          </form>
          <p className="auth-note">继续即表示你愿意友善表达、尊重不同，并为自己发布的内容负责。</p>
        </div>
      </div>
    </div>
  )
}
