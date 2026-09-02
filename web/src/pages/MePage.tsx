import { ArrowRight, CalendarDays, KeyRound, LockKeyhole, LogIn, LogOut, PenLine, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import { Avatar } from '../components/Avatar'
import { InvitationDialog } from '../components/InvitationDialog'
import { PostCard } from '../components/PostCard'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api, shouldClearToken } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatDateTime, getDisplayName } from '../lib/format'
import type { PostListItem } from '../types'

const PAGE_SIZE = 10
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

interface PostsRequest {
  controller: AbortController
  generation: number
  sessionIdentity: string
}

export function MePage() {
  const { token, user, isAuthenticated, isBootstrapping, canWrite, refreshUser, logout, isCurrentSession, clearSession, handleSessionError } = useAuth()
  const { openAuth, openComposer, notify, contentVersion } = useUI()
  const [posts, setPosts] = useState<PostListItem[]>([])
  const [pageNo, setPageNo] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loadingPosts, setLoadingPosts] = useState(true)
  const [postError, setPostError] = useState('')
  const [nickname, setNickname] = useState('')
  const [introduction, setIntroduction] = useState('')
  const [profileError, setProfileError] = useState('')
  const [savingProfile, setSavingProfile] = useState(false)
  const [avatarURL, setAvatarURL] = useState('')
  const [avatarError, setAvatarError] = useState('')
  const [savingAvatar, setSavingAvatar] = useState(false)
  const [avatarClock, setAvatarClock] = useState(() => Date.now())
  const [invitationCode, setInvitationCode] = useState<string | null>(null)
  const [generatingInvitation, setGeneratingInvitation] = useState(false)
  const [signingOut, setSigningOut] = useState(false)
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordError, setPasswordError] = useState<PasswordFormError | null>(null)
  const [changingPassword, setChangingPassword] = useState(false)
  const oldPasswordRef = useRef<HTMLInputElement>(null)
  const newPasswordRef = useRef<HTMLInputElement>(null)
  const confirmPasswordRef = useRef<HTMLInputElement>(null)
  const passwordRequestRef = useRef<PasswordRequest | null>(null)
  const firstPageRequestRef = useRef<PostsRequest | null>(null)
  const loadMoreRequestRef = useRef<PostsRequest | null>(null)
  const postsGenerationRef = useRef(0)
  const sessionIdentity = token && user ? `${user.id}:${token}` : ''
  const latestSessionIdentityRef = useRef(sessionIdentity)
  latestSessionIdentityRef.current = sessionIdentity

  const resetPasswordFields = useCallback(() => {
    setOldPassword('')
    setNewPassword('')
    setConfirmPassword('')
    setPasswordError(null)
  }, [])

  useEffect(() => {
    setNickname(user?.nickname ?? '')
    setIntroduction(user?.introduction ?? '')
    setAvatarURL(user?.avatar_url ?? '')
  }, [user])

  useLayoutEffect(() => {
    postsGenerationRef.current += 1
    firstPageRequestRef.current?.controller.abort()
    firstPageRequestRef.current = null
    loadMoreRequestRef.current?.controller.abort()
    loadMoreRequestRef.current = null
    setPosts([])
    setPageNo(1)
    setHasMore(false)
    setPostError('')
    setLoadingPosts(Boolean(sessionIdentity))

    const request = passwordRequestRef.current
    if (request && request.sessionIdentity !== sessionIdentity) {
      request.controller.abort()
      passwordRequestRef.current = null
    }
    setChangingPassword(false)
    resetPasswordFields()
  }, [resetPasswordFields, sessionIdentity])

  useEffect(() => () => {
    postsGenerationRef.current += 1
    firstPageRequestRef.current?.controller.abort()
    firstPageRequestRef.current = null
    loadMoreRequestRef.current?.controller.abort()
    loadMoreRequestRef.current = null
    passwordRequestRef.current?.controller.abort()
    passwordRequestRef.current = null
  }, [])

  const avatarAvailableAt = user?.avatar_updated_at
    ? new Date(user.avatar_updated_at).getTime() + 24 * 60 * 60 * 1000
    : Number.NaN
  const avatarCoolingDown = Number.isFinite(avatarAvailableAt) && avatarAvailableAt > avatarClock
  const avatarChanged = avatarURL.trim() !== (user?.avatar_url ?? '')
  const isReadOnly = Boolean(user) && !canWrite
  const passwordFormDisabled = changingPassword || signingOut || savingProfile || savingAvatar || generatingInvitation

  useEffect(() => {
    if (!Number.isFinite(avatarAvailableAt) || avatarAvailableAt <= Date.now()) return
    const timeout = window.setTimeout(
      () => setAvatarClock(Date.now()),
      Math.min(avatarAvailableAt - Date.now() + 100, 2_147_483_647),
    )
    return () => window.clearTimeout(timeout)
  }, [avatarAvailableAt])

  const loadPosts = useCallback(async () => {
    postsGenerationRef.current += 1
    const generation = postsGenerationRef.current
    firstPageRequestRef.current?.controller.abort()
    firstPageRequestRef.current = null
    loadMoreRequestRef.current?.controller.abort()
    loadMoreRequestRef.current = null

    const submittedToken = token
    const submittedSessionIdentity = sessionIdentity
    if (!submittedToken || !submittedSessionIdentity) {
      setPosts([])
      setPageNo(1)
      setHasMore(false)
      setPostError('')
      setLoadingPosts(false)
      return
    }

    const controller = new AbortController()
    const request: PostsRequest = {
      controller,
      generation,
      sessionIdentity: submittedSessionIdentity,
    }
    firstPageRequestRef.current = request
    const requestIsCurrent = () => (
      !controller.signal.aborted &&
      firstPageRequestRef.current === request &&
      postsGenerationRef.current === generation &&
      isCurrentSession(submittedToken) &&
      latestSessionIdentityRef.current === submittedSessionIdentity
    )

    setLoadingPosts(true)
    setPostError('')
    try {
      const page = await api.listMyPosts(
        submittedToken,
        { pageNo: 1, pageSize: PAGE_SIZE },
        controller.signal,
      )
      if (!requestIsCurrent()) return
      setPosts(page.list)
      setPageNo(1)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      if (!requestIsCurrent()) return
      if (shouldClearToken(loadError)) {
        if (handleSessionError(loadError, submittedToken)) openAuth('login')
        return
      }
      setPostError(getErrorMessage(loadError))
    } finally {
      const requestWasCurrent = requestIsCurrent()
      if (firstPageRequestRef.current === request) firstPageRequestRef.current = null
      if (requestWasCurrent) setLoadingPosts(false)
    }
  }, [handleSessionError, openAuth, sessionIdentity, token])

  useEffect(() => {
    void loadPosts()
  }, [contentVersion, loadPosts])

  const loadMore = async () => {
    if (
      !token ||
      !sessionIdentity ||
      loadingPosts ||
      firstPageRequestRef.current ||
      loadMoreRequestRef.current
    ) return

    const submittedToken = token
    const submittedSessionIdentity = sessionIdentity
    const generation = postsGenerationRef.current
    const nextPage = pageNo + 1
    const controller = new AbortController()
    const request: PostsRequest = {
      controller,
      generation,
      sessionIdentity: submittedSessionIdentity,
    }
    loadMoreRequestRef.current = request
    const requestIsCurrent = () => (
      !controller.signal.aborted &&
      loadMoreRequestRef.current === request &&
      postsGenerationRef.current === generation &&
      isCurrentSession(submittedToken) &&
      latestSessionIdentityRef.current === submittedSessionIdentity
    )

    setPostError('')
    try {
      const page = await api.listMyPosts(
        submittedToken,
        { pageNo: nextPage, pageSize: PAGE_SIZE },
        controller.signal,
      )
      if (!requestIsCurrent()) return
      setPosts((current) => [...current, ...page.list.filter((post) => !current.some((item) => item.id === post.id))])
      setPageNo(nextPage)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      if (!requestIsCurrent()) return
      if (shouldClearToken(loadError)) {
        if (handleSessionError(loadError, submittedToken)) openAuth('login')
        return
      }
      setPostError(getErrorMessage(loadError))
    } finally {
      if (loadMoreRequestRef.current === request) loadMoreRequestRef.current = null
    }
  }

  const saveProfile = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!token || !canWrite || savingProfile || changingPassword) return
    const submittedToken = token
    setSavingProfile(true)
    setProfileError('')
    try {
      await api.updateProfile(submittedToken, { nickname: nickname.trim(), introduction: introduction.trim() })
      if (!isCurrentSession(submittedToken)) return
      await refreshUser()
      if (!isCurrentSession(submittedToken)) return
      notify('success', '个人资料已更新')
    } catch (saveError) {
      if (!isCurrentSession(submittedToken)) return
      if (handleSessionError(saveError, submittedToken)) openAuth('login')
      else setProfileError(getErrorMessage(saveError))
    } finally {
      setSavingProfile(false)
    }
  }

  const saveAvatar = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!token || !canWrite || savingAvatar || changingPassword || avatarCoolingDown || !avatarChanged) return
    const submittedToken = token
    setSavingAvatar(true)
    setAvatarError('')
    try {
      await api.updateAvatar(submittedToken, { avatar_url: avatarURL.trim() })
      if (!isCurrentSession(submittedToken)) return
      await refreshUser()
      if (!isCurrentSession(submittedToken)) return
      setAvatarClock(Date.now())
      notify('success', avatarURL.trim() ? '头像已更新' : '已经恢复默认头像')
    } catch (saveError) {
      if (!isCurrentSession(submittedToken)) return
      if (handleSessionError(saveError, submittedToken)) openAuth('login')
      else setAvatarError(getErrorMessage(saveError))
    } finally {
      setSavingAvatar(false)
    }
  }

  const reportPasswordError = (field: PasswordField, message: string) => {
    setPasswordError({ field, message })
    if (field === 'old') oldPasswordRef.current?.focus()
    if (field === 'new') newPasswordRef.current?.focus()
    if (field === 'confirm') confirmPasswordRef.current?.focus()
  }

  const changePassword = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (
      !token ||
      !user ||
      changingPassword ||
      passwordRequestRef.current ||
      signingOut ||
      savingProfile ||
      savingAvatar ||
      generatingInvitation
    ) return

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
    passwordRequestRef.current = request
    setChangingPassword(true)
    setPasswordError(null)

    const requestIsCurrent = () => (
      !controller.signal.aborted &&
      passwordRequestRef.current === request &&
      isCurrentSession(submittedToken) &&
      latestSessionIdentityRef.current === submittedSessionIdentity
    )

    try {
      await api.changePassword(
        submittedToken,
        { old_password: oldPassword, new_password: newPassword },
        controller.signal,
      )
      if (!requestIsCurrent()) return

      // The server has already invalidated every old session. Clear this browser
      // only if it still owns the token that initiated the password change.
      if (!clearSession(submittedToken)) return

      passwordRequestRef.current = null
      setChangingPassword(false)
      resetPasswordFields()
      openAuth('login')
      notify('success', '密码已修改', '请使用新密码重新登录。')
    } catch (changeError) {
      if (!requestIsCurrent()) return
      if (handleSessionError(changeError, submittedToken)) {
        resetPasswordFields()
        openAuth('login')
      } else if (!shouldClearToken(changeError)) {
        setPasswordError({ field: 'form', message: getErrorMessage(changeError) })
      }
    } finally {
      if (passwordRequestRef.current === request) {
        passwordRequestRef.current = null
        if (latestSessionIdentityRef.current === submittedSessionIdentity) {
          setChangingPassword(false)
        }
      }
    }
  }

  const signOut = async () => {
    if (!token || signingOut || changingPassword) return
    const submittedToken = token
    setSigningOut(true)
    try {
      const sessionCleared = await logout()
      if (!sessionCleared) return
      notify('success', '已经退出登录')
    } catch (logoutError) {
      if (!isCurrentSession(submittedToken)) return
      if (handleSessionError(logoutError, submittedToken)) {
        notify('success', '登录状态已失效，本地会话已清除')
      } else {
        notify('error', '退出失败', getErrorMessage(logoutError))
      }
    } finally {
      setSigningOut(false)
    }
  }

  const generateInvitation = async () => {
    if (!token || !canWrite || generatingInvitation || changingPassword || invitationCode) return
    const submittedToken = token
    setGeneratingInvitation(true)
    try {
      const invitation = await api.createInvitation(submittedToken)
      if (!isCurrentSession(submittedToken)) return
      setInvitationCode(invitation.code)
    } catch (generateError) {
      if (!isCurrentSession(submittedToken)) return
      if (handleSessionError(generateError, submittedToken)) openAuth('login')
      else notify('error', '邀请码生成失败', getErrorMessage(generateError))
    } finally {
      setGeneratingInvitation(false)
    }
  }

  const closeInvitation = useCallback(() => {
    setInvitationCode(null)
  }, [])

  if (isBootstrapping) {
    return <div className="page-wrap"><div className="skeleton-card" style={{ height: 380 }} /></div>
  }

  if (!isAuthenticated || !user) {
    return (
      <div className="page-wrap">
        <div className="empty-state">
          <div className="empty-state__icon"><LogIn size={23} aria-hidden="true" /></div>
          <h3>登录后查看自己的空间</h3>
          <p>管理资料、公开或隐藏帖子，也可以继续编辑还没说完的话。</p>
          <button className="button button--dark" type="button" onClick={() => openAuth('login')}>去登录</button>
        </div>
      </div>
    )
  }

  return (
    <>
      <div className="page-wrap">
        {isReadOnly && (
          <div className="read-only-notice read-only-notice--page" role="status">
            <LockKeyhole size={19} aria-hidden="true" />
            <span><strong>当前为只读模式。</strong> 账号已被禁言，仍可登录、浏览、查询内容、修改密码和提交留言申诉，但不能发布、编辑、评论、点赞、修改资料或生成邀请码。</span>
          </div>
        )}
        <section className="profile-card" aria-labelledby="profile-name">
          <Avatar user={user} size="lg" />
          <div>
            <h1 id="profile-name">{getDisplayName(user)}</h1>
            <div className="profile-card__handle">@{user.username}</div>
            <p className="profile-card__intro">{user.introduction || '这个人还没有留下自我介绍。'}</p>
            <div className="profile-meta"><CalendarDays size={14} aria-hidden="true" /> 加入于 {formatDateTime(user.create_time, { year: 'numeric', month: 'long', day: 'numeric', hour: undefined, minute: undefined })}</div>
          </div>
          <div className="profile-card__actions">
            <button
              className="button button--dark"
              type="button"
              onClick={() => void generateInvitation()}
              disabled={!canWrite || generatingInvitation || changingPassword || Boolean(invitationCode)}
              aria-busy={generatingInvitation}
            >
              <KeyRound size={16} aria-hidden="true" />
              {generatingInvitation ? '生成中…' : '生成邀请码'}
            </button>
            <button
              className="button button--soft"
              type="button"
              onClick={() => void signOut()}
              disabled={signingOut || changingPassword}
              aria-busy={signingOut}
            >
              <LogOut size={16} aria-hidden="true" />
              {signingOut ? '退出中…' : '退出登录'}
            </button>
          </div>
        </section>

        <div className="profile-layout">
          <section aria-labelledby="my-posts-heading">
            <div className="section-heading">
              <div>
                <span className="eyebrow">MY WRITING</span>
                <h2 id="my-posts-heading">我写下的</h2>
                <p>公开与私密内容都会显示在这里。</p>
              </div>
              <button className="button button--primary button--small" type="button" onClick={() => openComposer()} disabled={!canWrite} title={!canWrite ? '账号已被禁言，当前为只读模式' : undefined}><PenLine size={14} aria-hidden="true" />新帖子</button>
            </div>

            {loadingPosts ? (
              <div className="post-list"><div className="skeleton-card" /><div className="skeleton-card" /></div>
            ) : postError && posts.length === 0 ? (
              <div className="error-state">
                <div className="error-state__icon"><RefreshCw size={23} aria-hidden="true" /></div>
                <h2>暂时没能读到帖子</h2>
                <p>{postError}</p>
                <button className="button button--dark" type="button" onClick={() => void loadPosts()}>重新加载</button>
              </div>
            ) : posts.length === 0 ? (
              <div className="empty-state">
                <div className="empty-state__icon"><PenLine size={23} aria-hidden="true" /></div>
                <h3>还没有写过帖子</h3>
                <p>第一篇不必完美，只要是真实的。</p>
                <button className="button button--primary" type="button" onClick={() => openComposer()} disabled={!canWrite}>开始写</button>
              </div>
            ) : (
              <>
                <div className="post-list">
                  {posts.map((post) => <PostCard key={post.id} post={post} manageable onChanged={() => void loadPosts()} />)}
                </div>
                {postError && <p className="form-error" role="alert">{postError}</p>}
                {hasMore && (
                  <div className="load-more">
                    <button className="button button--soft" type="button" onClick={() => void loadMore()}>继续加载 <ArrowRight size={15} aria-hidden="true" /></button>
                  </div>
                )}
              </>
            )}
          </section>

          <aside className="form-card">
            <h2>编辑资料</h2>
            <form onSubmit={saveAvatar}>
              <div className="field">
                <label htmlFor="profile-avatar">头像链接 <span>{avatarURL.length}/2048</span></label>
                <input
                  id="profile-avatar"
                  type="url"
                  inputMode="url"
                  autoComplete="url"
                  value={avatarURL}
                  onChange={(event) => setAvatarURL(event.target.value)}
                  maxLength={2048}
                  placeholder="https://example.com/avatar.jpg"
                  disabled={!canWrite || savingAvatar || changingPassword || avatarCoolingDown}
                  aria-describedby="profile-avatar-help"
                />
              </div>
              <p className="field-help" id="profile-avatar-help">
                {avatarCoolingDown
                  ? `头像每 24 小时只能修改一次，下次可修改：${formatDateTime(new Date(avatarAvailableAt))}`
                  : '仅支持 HTTPS 图片链接；留空可恢复默认头像。修改成功后 24 小时内不能再次更换。'}
              </p>
              {avatarError && <p className="form-error" role="alert">{avatarError}</p>}
              <div className="form-actions">
                <button className="button button--soft button--wide" type="submit" disabled={!canWrite || savingAvatar || changingPassword || avatarCoolingDown || !avatarChanged}>
                  {savingAvatar ? '更新中…' : '更新头像'}
                </button>
              </div>
            </form>
            <div className="form-card__divider" aria-hidden="true" />
            <form onSubmit={saveProfile}>
              <div className="field">
                <label htmlFor="profile-nickname">昵称 <span>{Array.from(nickname).length}/64</span></label>
                <input id="profile-nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} maxLength={64} placeholder="大家怎么称呼你" disabled={!canWrite || changingPassword} />
              </div>
              <div className="field">
                <label htmlFor="profile-introduction">个人介绍 <span>{Array.from(introduction).length}/1024</span></label>
                <textarea id="profile-introduction" value={introduction} onChange={(event) => setIntroduction(event.target.value)} maxLength={1024} placeholder="写几句关于自己" disabled={!canWrite || changingPassword} />
              </div>
              {profileError && <p className="form-error" role="alert">{profileError}</p>}
              <div className="form-actions">
                <button className="button button--dark button--wide" type="submit" disabled={!canWrite || savingProfile || changingPassword}>{savingProfile ? '保存中…' : '保存资料'}</button>
              </div>
            </form>
            <div className="form-card__divider" aria-hidden="true" />
            <h3 id="change-password-heading">修改密码</h3>
            <form
              onSubmit={changePassword}
              noValidate
              aria-labelledby="change-password-heading"
              aria-busy={changingPassword}
            >
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
                  disabled={passwordFormDisabled}
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
                  disabled={passwordFormDisabled}
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
                  disabled={passwordFormDisabled}
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
                <button className="button button--soft button--wide" type="submit" disabled={passwordFormDisabled}>
                  {changingPassword ? '修改中…' : '修改密码'}
                </button>
              </div>
            </form>
          </aside>
        </div>
      </div>
      <InvitationDialog code={invitationCode} onClose={closeInvitation} />
    </>
  )
}
