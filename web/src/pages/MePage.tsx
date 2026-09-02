import { ArrowRight, CalendarDays, KeyRound, LockKeyhole, LogIn, LogOut, PenLine, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import { Avatar } from '../components/Avatar'
import { ChangePasswordDialog } from '../components/ChangePasswordDialog'
import { InvitationDialog } from '../components/InvitationDialog'
import { PostCard } from '../components/PostCard'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api, shouldClearToken } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatDateTime, getDisplayName } from '../lib/format'
import type { PostListItem } from '../types'

const PAGE_SIZE = 10

interface PostsRequest {
  controller: AbortController
  generation: number
  sessionIdentity: string
}

export function MePage() {
  const { token, user, isAuthenticated, isBootstrapping, canWrite, refreshUser, logout, isCurrentSession, handleSessionError } = useAuth()
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
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false)
  const firstPageRequestRef = useRef<PostsRequest | null>(null)
  const loadMoreRequestRef = useRef<PostsRequest | null>(null)
  const postsGenerationRef = useRef(0)
  const sessionIdentity = token && user ? `${user.id}:${token}` : ''
  const latestSessionIdentityRef = useRef(sessionIdentity)
  latestSessionIdentityRef.current = sessionIdentity

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
  }, [sessionIdentity])

  useEffect(() => () => {
    postsGenerationRef.current += 1
    firstPageRequestRef.current?.controller.abort()
    firstPageRequestRef.current = null
    loadMoreRequestRef.current?.controller.abort()
    loadMoreRequestRef.current = null
  }, [])

  const avatarAvailableAt = user?.avatar_updated_at
    ? new Date(user.avatar_updated_at).getTime() + 24 * 60 * 60 * 1000
    : Number.NaN
  const avatarCoolingDown = Number.isFinite(avatarAvailableAt) && avatarAvailableAt > avatarClock
  const avatarChanged = avatarURL.trim() !== (user?.avatar_url ?? '')
  const isReadOnly = Boolean(user) && !canWrite

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
    if (!token || !canWrite || savingProfile) return
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
    if (!token || !canWrite || savingAvatar || avatarCoolingDown || !avatarChanged) return
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

  const signOut = async () => {
    if (!token || signingOut) return
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
    if (!token || !canWrite || generatingInvitation || invitationCode) return
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

  const closePasswordDialog = useCallback(() => {
    setPasswordDialogOpen(false)
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
              disabled={!canWrite || generatingInvitation || Boolean(invitationCode)}
              aria-busy={generatingInvitation}
            >
              <KeyRound size={16} aria-hidden="true" />
              {generatingInvitation ? '生成中…' : '生成邀请码'}
            </button>
            <button
              className="button button--soft"
              type="button"
              onClick={() => void signOut()}
              disabled={signingOut}
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
                  disabled={!canWrite || savingAvatar || avatarCoolingDown}
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
                <button className="button button--soft button--wide" type="submit" disabled={!canWrite || savingAvatar || avatarCoolingDown || !avatarChanged}>
                  {savingAvatar ? '更新中…' : '更新头像'}
                </button>
              </div>
            </form>
            <div className="form-card__divider" aria-hidden="true" />
            <form onSubmit={saveProfile}>
              <div className="field">
                <label htmlFor="profile-nickname">昵称 <span>{Array.from(nickname).length}/64</span></label>
                <input id="profile-nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} maxLength={64} placeholder="大家怎么称呼你" disabled={!canWrite} />
              </div>
              <div className="field">
                <label htmlFor="profile-introduction">个人介绍 <span>{Array.from(introduction).length}/1024</span></label>
                <textarea id="profile-introduction" value={introduction} onChange={(event) => setIntroduction(event.target.value)} maxLength={1024} placeholder="写几句关于自己" disabled={!canWrite} />
              </div>
              {profileError && <p className="form-error" role="alert">{profileError}</p>}
              <div className="form-actions">
                <button className="button button--dark button--wide" type="submit" disabled={!canWrite || savingProfile}>{savingProfile ? '保存中…' : '保存资料'}</button>
              </div>
            </form>
            <div className="form-card__divider" aria-hidden="true" />
            <div className="form-actions">
              <button
                className="button button--soft button--wide"
                type="button"
                onClick={() => setPasswordDialogOpen(true)}
                disabled={signingOut}
              >
                <KeyRound size={16} aria-hidden="true" />
                修改密码
              </button>
            </div>
          </aside>
        </div>
      </div>
      <InvitationDialog code={invitationCode} onClose={closeInvitation} />
      <ChangePasswordDialog open={passwordDialogOpen} onClose={closePasswordDialog} />
    </>
  )
}
