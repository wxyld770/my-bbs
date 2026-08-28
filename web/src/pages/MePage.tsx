import { ArrowRight, CalendarDays, KeyRound, LogIn, LogOut, PenLine, RefreshCw } from 'lucide-react'
import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Avatar } from '../components/Avatar'
import { InvitationDialog } from '../components/InvitationDialog'
import { PostCard } from '../components/PostCard'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatDateTime, getDisplayName } from '../lib/format'
import type { PostListItem } from '../types'

const PAGE_SIZE = 10

export function MePage() {
  const { token, user, isAuthenticated, isBootstrapping, refreshUser, logout, handleSessionError } = useAuth()
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

  useEffect(() => {
    setNickname(user?.nickname ?? '')
    setIntroduction(user?.introduction ?? '')
    setAvatarURL(user?.avatar_url ?? '')
  }, [user])

  const avatarAvailableAt = user?.avatar_updated_at
    ? new Date(user.avatar_updated_at).getTime() + 24 * 60 * 60 * 1000
    : Number.NaN
  const avatarCoolingDown = Number.isFinite(avatarAvailableAt) && avatarAvailableAt > avatarClock
  const avatarChanged = avatarURL.trim() !== (user?.avatar_url ?? '')

  useEffect(() => {
    if (!Number.isFinite(avatarAvailableAt) || avatarAvailableAt <= Date.now()) return
    const timeout = window.setTimeout(
      () => setAvatarClock(Date.now()),
      Math.min(avatarAvailableAt - Date.now() + 100, 2_147_483_647),
    )
    return () => window.clearTimeout(timeout)
  }, [avatarAvailableAt])

  const loadPosts = useCallback(async () => {
    if (!token) {
      setLoadingPosts(false)
      return
    }
    setLoadingPosts(true)
    setPostError('')
    try {
      const page = await api.listMyPosts(token, { pageNo: 1, pageSize: PAGE_SIZE })
      setPosts(page.list)
      setPageNo(1)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      if (handleSessionError(loadError)) openAuth('login')
      else setPostError(getErrorMessage(loadError))
    } finally {
      setLoadingPosts(false)
    }
  }, [handleSessionError, openAuth, token])

  useEffect(() => {
    void loadPosts()
  }, [contentVersion, loadPosts])

  const loadMore = async () => {
    if (!token) return
    try {
      const nextPage = pageNo + 1
      const page = await api.listMyPosts(token, { pageNo: nextPage, pageSize: PAGE_SIZE })
      setPosts((current) => [...current, ...page.list.filter((post) => !current.some((item) => item.id === post.id))])
      setPageNo(nextPage)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      setPostError(getErrorMessage(loadError))
    }
  }

  const saveProfile = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!token) return
    setSavingProfile(true)
    setProfileError('')
    try {
      await api.updateProfile(token, { nickname: nickname.trim(), introduction: introduction.trim() })
      await refreshUser()
      notify('success', '个人资料已更新')
    } catch (saveError) {
      if (handleSessionError(saveError)) openAuth('login')
      else setProfileError(getErrorMessage(saveError))
    } finally {
      setSavingProfile(false)
    }
  }

  const saveAvatar = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!token || savingAvatar || avatarCoolingDown || !avatarChanged) return
    setSavingAvatar(true)
    setAvatarError('')
    try {
      await api.updateAvatar(token, { avatar_url: avatarURL.trim() })
      await refreshUser()
      setAvatarClock(Date.now())
      notify('success', avatarURL.trim() ? '头像已更新' : '已经恢复默认头像')
    } catch (saveError) {
      if (handleSessionError(saveError)) openAuth('login')
      else setAvatarError(getErrorMessage(saveError))
    } finally {
      setSavingAvatar(false)
    }
  }

  const signOut = async () => {
    await logout()
    notify('success', '已经退出登录')
  }

  const generateInvitation = async () => {
    if (!token || generatingInvitation || invitationCode) return
    setGeneratingInvitation(true)
    try {
      const invitation = await api.createInvitation(token)
      setInvitationCode(invitation.code)
    } catch (generateError) {
      if (handleSessionError(generateError)) openAuth('login')
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
              disabled={generatingInvitation || Boolean(invitationCode)}
              aria-describedby="invitation-eligibility-help"
              aria-busy={generatingInvitation}
            >
              <KeyRound size={16} aria-hidden="true" />
              {generatingInvitation ? '生成中…' : '生成邀请码'}
            </button>
            <p className="profile-card__invitation-help" id="invitation-eligibility-help">
              注册满 7 天后可生成；成功发布一篇帖子后可立即生成。
            </p>
            <button className="button button--soft" type="button" onClick={() => void signOut()}><LogOut size={16} aria-hidden="true" />退出登录</button>
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
              <button className="button button--primary button--small" type="button" onClick={() => openComposer()}><PenLine size={14} aria-hidden="true" />新帖子</button>
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
                <button className="button button--primary" type="button" onClick={() => openComposer()}>开始写</button>
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
                  disabled={savingAvatar || avatarCoolingDown}
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
                <button className="button button--soft button--wide" type="submit" disabled={savingAvatar || avatarCoolingDown || !avatarChanged}>
                  {savingAvatar ? '更新中…' : '更新头像'}
                </button>
              </div>
            </form>
            <div className="form-card__divider" aria-hidden="true" />
            <form onSubmit={saveProfile}>
              <div className="field">
                <label htmlFor="profile-nickname">昵称 <span>{Array.from(nickname).length}/64</span></label>
                <input id="profile-nickname" value={nickname} onChange={(event) => setNickname(event.target.value)} maxLength={64} placeholder="大家怎么称呼你" />
              </div>
              <div className="field">
                <label htmlFor="profile-introduction">个人介绍 <span>{Array.from(introduction).length}/1024</span></label>
                <textarea id="profile-introduction" value={introduction} onChange={(event) => setIntroduction(event.target.value)} maxLength={1024} placeholder="写几句关于自己" />
              </div>
              {profileError && <p className="form-error" role="alert">{profileError}</p>}
              <div className="form-actions">
                <button className="button button--dark button--wide" type="submit" disabled={savingProfile}>{savingProfile ? '保存中…' : '保存资料'}</button>
              </div>
            </form>
          </aside>
        </div>
      </div>
      <InvitationDialog code={invitationCode} onClose={closeInvitation} />
    </>
  )
}
