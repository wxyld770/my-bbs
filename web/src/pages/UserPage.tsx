import { ArrowLeft, ArrowRight, CalendarDays, RefreshCw, ShieldCheck, UserRoundCheck, UserRoundX } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { Avatar } from '../components/Avatar'
import { PostCard } from '../components/PostCard'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatDateTime, getDisplayName } from '../lib/format'
import { USER_STATUS, type PostListItem, type User } from '../types'

const PAGE_SIZE = 10

export function UserPage() {
  const { id } = useParams()
  const userId = Number(id)
  const { token, user: currentUser, canWrite, handleSessionError } = useAuth()
  const { notify } = useUI()
  const [profile, setProfile] = useState<User | null>(null)
  const [posts, setPosts] = useState<PostListItem[]>([])
  const [pageNo, setPageNo] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [managingUser, setManagingUser] = useState(false)

  const validId = Number.isSafeInteger(userId) && userId > 0

  const loadProfile = useCallback(async () => {
    if (!validId) {
      setError('这个用户不存在。')
      setLoading(false)
      return
    }
    setLoading(true)
    setError('')
    try {
      const [profileData, postPage] = await Promise.all([
        api.getUser(userId),
        api.listUserPosts(userId, { pageNo: 1, pageSize: PAGE_SIZE }),
      ])
      setProfile(profileData.user)
      setPosts(postPage.list)
      setPageNo(1)
      setHasMore(postPage.hasMore && postPage.list.length > 0)
    } catch (loadError) {
      setError(getErrorMessage(loadError))
    } finally {
      setLoading(false)
    }
  }, [userId, validId])

  useEffect(() => {
    void loadProfile()
  }, [loadProfile])

  const loadMore = async () => {
    const nextPage = pageNo + 1
    const page = await api.listUserPosts(userId, { pageNo: nextPage, pageSize: PAGE_SIZE })
    setPosts((current) => [...current, ...page.list.filter((post) => !current.some((item) => item.id === post.id))])
    setPageNo(nextPage)
    setHasMore(page.hasMore && page.list.length > 0)
  }

  const toggleMuted = async () => {
    if (!token || !profile || currentUser?.is_admin !== true || profile.is_admin) return
    const shouldMute = profile.status === USER_STATUS.NORMAL
    setManagingUser(true)
    try {
      if (shouldMute) {
        await api.muteUser(token, profile.id)
      } else {
        await api.unmuteUser(token, profile.id)
      }
      setProfile((current) => current ? {
        ...current,
        status: shouldMute ? USER_STATUS.MUTED : USER_STATUS.NORMAL,
      } : current)
      notify('success', shouldMute ? '用户已禁言' : '用户已解除禁言')
    } catch (manageError) {
      handleSessionError(manageError)
      notify('error', shouldMute ? '禁言失败' : '解除禁言失败', getErrorMessage(manageError))
    } finally {
      setManagingUser(false)
    }
  }

  if (currentUser?.id === userId) return <Navigate to="/me" replace />

  if (loading) return <div className="page-wrap"><div className="skeleton-card" style={{ height: 430 }} /></div>

  if (error || !profile) {
    return (
      <div className="page-wrap">
        <div className="error-state">
          <div className="error-state__icon"><RefreshCw size={23} aria-hidden="true" /></div>
          <h2>没有找到这个人</h2>
          <p>{error || '用户资料不存在。'}</p>
          <Link className="button button--dark" to="/">返回广场</Link>
        </div>
      </div>
    )
  }

  return (
    <div className="page-wrap">
      <div className="page-header"><Link className="back-link" to="/"><ArrowLeft size={17} aria-hidden="true" />回到广场</Link></div>
      <section className="profile-card">
        <Avatar user={profile} size="lg" />
        <div>
          <h1>{getDisplayName(profile)}</h1>
          <div className="profile-card__handle">@{profile.username}</div>
          <p className="profile-card__intro">{profile.introduction || '这个人还没有留下自我介绍。'}</p>
          <div className="profile-meta"><CalendarDays size={14} aria-hidden="true" /> 加入于 {formatDateTime(profile.create_time, { year: 'numeric', month: 'long', day: 'numeric', hour: undefined, minute: undefined })}</div>
        </div>
        <div className="profile-card__admin-actions">
          {profile.is_admin && <span className="post-card__tag pinned"><ShieldCheck size={13} aria-hidden="true" />管理员</span>}
          {profile.status === USER_STATUS.MUTED && <span className="post-card__tag private">已禁言</span>}
          {canWrite && currentUser?.is_admin === true && !profile.is_admin && (
            <button className={`button button--small ${profile.status === USER_STATUS.NORMAL ? 'button--danger' : 'button--soft'}`} type="button" onClick={() => void toggleMuted()} disabled={managingUser}>
              {profile.status === USER_STATUS.NORMAL ? <UserRoundX size={14} aria-hidden="true" /> : <UserRoundCheck size={14} aria-hidden="true" />}
              {managingUser ? '处理中…' : profile.status === USER_STATUS.NORMAL ? '禁言用户' : '解除禁言'}
            </button>
          )}
        </div>
      </section>

      <section style={{ marginTop: '3rem' }} aria-labelledby="user-posts-heading">
        <div className="section-heading">
          <div>
            <span className="eyebrow">PUBLIC WRITING</span>
            <h2 id="user-posts-heading">公开写下的</h2>
          </div>
          <span className="section-count">LOADED / {String(posts.length).padStart(2, '0')}</span>
        </div>
        {posts.length === 0 ? (
          <div className="empty-state"><h3>暂时没有公开帖子</h3><p>也许文字正在路上。</p></div>
        ) : (
          <>
            <div className="post-list">{posts.map((post) => <PostCard key={post.id} post={post} onChanged={() => void loadProfile()} />)}</div>
            {hasMore && <div className="load-more"><button className="button button--soft" type="button" onClick={() => void loadMore()}>继续加载 <ArrowRight size={15} aria-hidden="true" /></button></div>}
          </>
        )}
      </section>
    </div>
  )
}
