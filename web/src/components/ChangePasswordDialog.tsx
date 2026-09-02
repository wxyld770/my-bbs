import { KeyRound, X } from 'lucide-react'
import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api, shouldClearToken } from '../lib/api'
import { getErrorMessage } from '../lib/errors'

const PASSWORD_MIN_RUNES = 6
const PASSWORD_MAX_RUNES = 64
const PASSWORD_MAX_BYTES = 72

type PasswordField = 'old' | 'new' | 'confirm' | 'form'

interface PasswordFormError {
  field: PasswordField
  message: string
}

interface PasswordRequest {
  controller: AbortController
  sessionIdentity: string
}

interface ChangePasswordDialogProps {
  open: boolean
  onClose: () => void
}

export function ChangePasswordDialog({ open, onClose }: ChangePasswordDialogProps) {
  const {
    token,
    user,
    isCurrentSession,
    clearSession,
    handleSessionError,
  } = useAuth()
  const { openAuth, notify } = useUI()
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordError, setPasswordError] = useState<PasswordFormError | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const dialogRef = useRef<HTMLDivElement>(null)
  const oldPasswordRef = useRef<HTMLInputElement>(null)
  const newPasswordRef = useRef<HTMLInputElement>(null)
  const confirmPasswordRef = useRef<HTMLInputElement>(null)
  const requestRef = useRef<PasswordRequest | null>(null)
  const openedSessionIdentityRef = useRef('')
  const sessionIdentity = token && user ? `${user.id}:${token}` : ''
  const latestSessionIdentityRef = useRef(sessionIdentity)
  latestSessionIdentityRef.current = sessionIdentity

  const resetFields = useCallback(() => {
    setOldPassword('')
    setNewPassword('')
    setConfirmPassword('')
    setPasswordError(null)
  }, [])

  const requestClose = useCallback(() => {
    if (requestRef.current) return
    resetFields()
    onClose()
  }, [onClose, resetFields])
  const requestCloseRef = useRef(requestClose)
  requestCloseRef.current = requestClose

  useEffect(() => {
    if (!open) {
      openedSessionIdentityRef.current = ''
      resetFields()
      setSubmitting(false)
      return
    }

    openedSessionIdentityRef.current = latestSessionIdentityRef.current
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    oldPasswordRef.current?.focus()

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (!requestRef.current) {
          event.preventDefault()
          requestCloseRef.current()
        }
        return
      }
      if (event.key !== 'Tab') return

      const focusable = Array.from(
        dialogRef.current?.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ) ?? [],
      )
      if (focusable.length === 0) {
        event.preventDefault()
        dialogRef.current?.focus()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (!dialogRef.current?.contains(document.activeElement)) {
        event.preventDefault()
        const target = event.shiftKey ? last : first
        target.focus()
      } else if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      requestRef.current?.controller.abort()
      requestRef.current = null
      previousFocus?.focus()
    }
  }, [open, resetFields])

  useEffect(() => {
    if (
      !open ||
      (sessionIdentity && openedSessionIdentityRef.current === sessionIdentity)
    ) return

    requestRef.current?.controller.abort()
    requestRef.current = null
    setSubmitting(false)
    resetFields()
    onClose()
  }, [onClose, open, resetFields, sessionIdentity])

  useEffect(() => () => {
    requestRef.current?.controller.abort()
    requestRef.current = null
  }, [])

  const reportPasswordError = (field: PasswordField, message: string) => {
    setPasswordError({ field, message })
    if (field === 'old') oldPasswordRef.current?.focus()
    if (field === 'new') newPasswordRef.current?.focus()
    if (field === 'confirm') confirmPasswordRef.current?.focus()
  }

  const changePassword = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!open || !token || !user || submitting || requestRef.current) return

    if (!oldPassword) {
      reportPasswordError('old', '请输入旧密码。')
      return
    }
    const newPasswordRunes = Array.from(newPassword).length
    if (newPasswordRunes < PASSWORD_MIN_RUNES || newPasswordRunes > PASSWORD_MAX_RUNES) {
      reportPasswordError('new', `新密码必须为 ${PASSWORD_MIN_RUNES}–${PASSWORD_MAX_RUNES} 个字符。`)
      return
    }
    if (new TextEncoder().encode(newPassword).length > PASSWORD_MAX_BYTES) {
      reportPasswordError('new', `新密码 UTF-8 编码后不能超过 ${PASSWORD_MAX_BYTES} 字节，请减少中文、表情等多字节字符。`)
      return
    }
    if (newPassword === oldPassword) {
      reportPasswordError('new', '新密码不能与旧密码相同。')
      return
    }
    if (confirmPassword !== newPassword) {
      reportPasswordError('confirm', '两次输入的新密码不一致。')
      return
    }

    const submittedToken = token
    const submittedSessionIdentity = sessionIdentity
    const controller = new AbortController()
    const request: PasswordRequest = {
      controller,
      sessionIdentity: submittedSessionIdentity,
    }
    requestRef.current = request
    setSubmitting(true)
    setPasswordError(null)

    const requestIsCurrent = () => (
      !controller.signal.aborted &&
      requestRef.current === request &&
      isCurrentSession(submittedToken) &&
      latestSessionIdentityRef.current === submittedSessionIdentity
    )

    try {
      await api.changePassword(
        submittedToken,
        { old_password: oldPassword, new_password: newPassword },
        controller.signal,
      )
      if (!requestIsCurrent() || !clearSession(submittedToken)) return

      requestRef.current = null
      setSubmitting(false)
      resetFields()
      onClose()
      openAuth('login')
      notify('success', '密码已修改', '请使用新密码重新登录。')
    } catch (changeError) {
      if (!requestIsCurrent()) return
      if (handleSessionError(changeError, submittedToken)) {
        requestRef.current = null
        setSubmitting(false)
        resetFields()
        onClose()
        openAuth('login')
      } else if (!shouldClearToken(changeError)) {
        setPasswordError({ field: 'form', message: getErrorMessage(changeError) })
      }
    } finally {
      if (requestRef.current === request) {
        requestRef.current = null
        if (latestSessionIdentityRef.current === submittedSessionIdentity) {
          setSubmitting(false)
        }
      }
    }
  }

  if (!open) return null

  return (
    <div className="dialog-backdrop" onMouseDown={() => requestCloseRef.current()}>
      <div
        ref={dialogRef}
        className="dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="change-password-title"
        aria-describedby="change-password-help"
        aria-busy={submitting}
        tabIndex={-1}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <div className="dialog__header">
          <div>
            <span className="eyebrow"><KeyRound size={14} aria-hidden="true" />账号安全</span>
            <h2 id="change-password-title">修改密码</h2>
          </div>
          <button className="icon-button" type="button" onClick={requestClose} disabled={submitting} aria-label="关闭修改密码弹窗">
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <div className="dialog__body">
          <form onSubmit={changePassword} noValidate>
            <div className="field">
              <label htmlFor="account-old-password">旧密码</label>
              <input
                ref={oldPasswordRef}
                id="account-old-password"
                name="old_password"
                type="password"
                autoComplete="current-password"
                value={oldPassword}
                onChange={(event) => {
                  setOldPassword(event.target.value)
                  setPasswordError(null)
                }}
                disabled={submitting}
                required
                aria-invalid={passwordError?.field === 'old'}
                aria-describedby={`change-password-help${passwordError?.field === 'old' ? ' change-password-error' : ''}`}
              />
            </div>
            <div className="field">
              <label htmlFor="account-new-password">新密码 <span>6–64 字符</span></label>
              <input
                ref={newPasswordRef}
                id="account-new-password"
                name="new_password"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(event) => {
                  setNewPassword(event.target.value)
                  setPasswordError(null)
                }}
                disabled={submitting}
                required
                aria-invalid={passwordError?.field === 'new'}
                aria-describedby={`change-password-help${passwordError?.field === 'new' ? ' change-password-error' : ''}`}
              />
            </div>
            <div className="field">
              <label htmlFor="account-confirm-password">确认新密码</label>
              <input
                ref={confirmPasswordRef}
                id="account-confirm-password"
                name="confirm_password"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(event) => {
                  setConfirmPassword(event.target.value)
                  setPasswordError(null)
                }}
                disabled={submitting}
                required
                aria-invalid={passwordError?.field === 'confirm'}
                aria-describedby={`change-password-help${passwordError?.field === 'confirm' ? ' change-password-error' : ''}`}
              />
            </div>
            <p className="field-help" id="change-password-help">
              密码不要与旧密码相同，且 UTF-8 编码后不能超过 72 字节。修改成功后，所有设备都需要重新登录。
            </p>
            {passwordError && (
              <p className="form-error" id="change-password-error" role="alert">
                {passwordError.message}
              </p>
            )}
            <div className="form-actions">
              <button className="button button--soft" type="button" onClick={requestClose} disabled={submitting}>取消</button>
              <button className="button button--dark" type="submit" disabled={submitting}>
                {submitting ? '修改中…' : '确认修改'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
