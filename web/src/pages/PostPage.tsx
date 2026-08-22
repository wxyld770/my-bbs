import { ArrowLeft, Heart, LockKeyhole, MessageCircle, RefreshCw, Send, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Avatar } from '../components/Avatar'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatCount, formatDateTime, formatRelativeTime, getDisplayName } from '../lib/format'
import { POST_VISIBILITY, type Comment, type PostDetail } from '../types'

const COMMENT_PAGE_SIZE = 20

export function PostPage() {
  const { id } = useParams()
  const postId = Number(id)
  const navigate = useNavigate()
  const { token, user, isAuthenticated, handleSessionError } = useAuth()
  const { openAuth, openComposer, notify, contentVersion } = useUI()
  const [detail, setDetail] = useState<PostDetail | null>(null)
  const [comments, setComments] = useState<Comment[]>([])
  const [commentPage, setCommentPage] = useState(1)
  const [hasMoreComments, setHasMoreComments] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingComments, setLoadingComments] = useState(false)
  const [error, setError] = useState('')
  const [commentText, setCommentText] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)
  const [liking, setLiking] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Comment | null>(null)
  const [deletingComment, setDeletingComment] = useState(false)

  const validId = Number.isSafeInteger(postId) && postId > 0

  const loadPost = useCallback(async () => {
    if (!validId) {
      setError('这篇帖子不存在。')
      setLoading(false)
      return
    }
    setLoading(true)
    setError('')
    try {
      const postDetail = await api.getPost(postId, token)
      setDetail(postDetail)
      if (postDetail.post.visible === POST_VISIBILITY.PUBLIC) {
        const page = await api.listComments(postId, { pageNo: 1, pageSize: COMMENT_PAGE_SIZE })
        setComments(page.list)
        setCommentPage(1)
        setHasMoreComments(page.hasMore && page.list.length > 0)
      } else {
        setComments([])
        setHasMoreComments(false)
      }
    } catch (loadError) {
      if (handleSessionError(loadError)) {
        notify('info', '登录状态已失效', '已按访客身份重新加载。')
      }
      setError(getErrorMessage(loadError))
    } finally {
      setLoading(false)
    }
  }, [handleSessionError, notify, postId, token, validId])

  useEffect(() => {
    void loadPost()
  }, [contentVersion, loadPost])

  const isOwner = Boolean(user && detail && user.id === detail.post.user_id)
  const commentsEnabled = detail?.post.visible === POST_VISIBILITY.PUBLIC

  const authorLink = useMemo(() => {
    if (!detail?.post.user) return '/'
    return user?.id === detail.post.user.id ? '/me' : `/u/${detail.post.user.id}`
  }, [detail, user])

  const toggleLike = async () => {
    if (!isAuthenticated || !token) {
      openAuth('login')
      return
    }
    setLiking(true)
    try {
      const result = await api.toggleLike(token, postId)
      setDetail((current) => current ? { ...current, is_liked: result.liked, like_count: result.like_count } : current)
    } catch (likeError) {
      if (handleSessionError(likeError)) {
        openAuth('login')
        notify('info', '请重新登录后点赞')
      } else {
        notify('error', '点赞没有完成', getErrorMessage(likeError))
      }
    } finally {
      setLiking(false)
    }
  }

  const submitComment = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const content = commentText.trim()
    if (!content) return
    if (!token || !user) {
      openAuth('login')
      return
    }
    if (new Blob([content]).size > 65535) {
      notify('error', '评论太长了', '请将内容精简后再发布。')
      return
    }

    setSubmittingComment(true)
    try {
      await api.createComment(token, postId, { content })
      const now = new Date().toISOString()
      const optimisticComment: Comment = {
        id: -Date.now(),
        post_id: postId,
        user_id: user.id,
        content,
        user,
        create_time: now,
        update_time: now,
      }
      setComments((current) => [...current, optimisticComment])
      setDetail((current) => current ? { ...current, comment_count: current.comment_count + 1 } : current)
      setCommentText('')
      notify('success', '评论已发布')
    } catch (commentError) {
      if (handleSessionError(commentError)) {
        openAuth('login')
        notify('info', '登录状态已失效', '请重新登录后评论。')
      } else {
        notify('error', '评论没有发布', getErrorMessage(commentError))
      }
    } finally {
      setSubmittingComment(false)
    }
  }

  const loadMoreComments = async () => {
    setLoadingComments(true)
    try {
      const nextPage = commentPage + 1
      const page = await api.listComments(postId, { pageNo: nextPage, pageSize: COMMENT_PAGE_SIZE })
      setComments((current) => {
        const knownIds = new Set(current.map((comment) => comment.id))
        return [...current, ...page.list.filter((comment) => !knownIds.has(comment.id))]
      })
      setCommentPage(nextPage)
      setHasMoreComments(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      notify('error', '评论加载失败', getErrorMessage(loadError))
    } finally {
      setLoadingComments(false)
    }
  }

  const deleteComment = async () => {
    if (!deleteTarget || !token || deleteTarget.id < 1) return
    setDeletingComment(true)
    try {
      await api.deleteComment(token, deleteTarget.id)
      setComments((current) => current.filter((comment) => comment.id !== deleteTarget.id))
      setDetail((current) => current ? { ...current, comment_count: Math.max(0, current.comment_count - 1) } : current)
      setDeleteTarget(null)
      notify('success', '评论已删除')
    } catch (deleteError) {
      if (handleSessionError(deleteError)) openAuth('login')
      else notify('error', '删除失败', getErrorMessage(deleteError))
    } finally {
      setDeletingComment(false)
    }
  }

  if (loading) {
    return (
      <div className="page-wrap">
        <div className="skeleton-card" style={{ height: 540 }} />
      </div>
    )
  }

  if (error || !detail) {
    const missing = error && (error.includes('不存在') || error.includes('找不到'))
    return (
      <div className="page-wrap">
        <div className="error-state">
          <div className="error-state__icon">{missing ? <LockKeyhole size={24} aria-hidden="true" /> : <RefreshCw size={24} aria-hidden="true" />}</div>
          <h2>{missing ? '这篇帖子不在这里' : '暂时无法打开'}</h2>
          <p>{missing ? '它可能已被删除、设为私密，或从未存在过。' : error}</p>
          <button className="button button--dark" type="button" onClick={() => navigate('/')}>返回广场</button>
        </div>
      </div>
    )
  }

  return (
    <div className="page-wrap">
      <div className="page-header">
        <Link className="back-link" to="/"><ArrowLeft size={17} aria-hidden="true" />回到广场</Link>
        {isOwner && <button className="button button--soft button--small" type="button" onClick={() => openComposer(detail.post)}>编辑这篇</button>}
      </div>

      <div className="article-layout">
        <div>
          <article className="article-card">
            <header className="article-card__header">
              <Link className="article-author" to={authorLink}>
                <Avatar user={detail.post.user} size="md" />
                <span className="article-author__meta">
                  <strong>{getDisplayName(detail.post.user)}</strong>
                  <span>{formatDateTime(detail.post.create_time)}</span>
                </span>
              </Link>
              <h1>{detail.post.title}</h1>
              {detail.post.visible === POST_VISIBILITY.PRIVATE && <span className="post-card__tag private"><LockKeyhole size={12} aria-hidden="true" /> 仅自己可见</span>}
            </header>
            <div className="article-card__body">{detail.post.content}</div>
            <footer className="article-card__actions">
              <button className={`stat-pill${detail.is_liked ? ' liked' : ''}`} type="button" onClick={() => void toggleLike()} disabled={liking || !commentsEnabled}>
                <Heart size={17} fill={detail.is_liked ? 'currentColor' : 'none'} aria-hidden="true" />
                {detail.is_liked ? '已喜欢' : '喜欢'} · {formatCount(detail.like_count)}
              </button>
              <span className="stat-pill"><MessageCircle size={17} aria-hidden="true" />评论 · {formatCount(detail.comment_count)}</span>
            </footer>
          </article>

          <section className="comment-section" aria-labelledby="comment-heading">
            <div className="section-heading">
              <div>
                <span className="eyebrow">CONVERSATION</span>
                <h2 id="comment-heading">接着聊聊</h2>
              </div>
              <span className="section-count">{formatCount(detail.comment_count)} 条回应</span>
            </div>

            {!commentsEnabled ? (
              <div className="empty-state">
                <div className="empty-state__icon"><LockKeyhole size={23} aria-hidden="true" /></div>
                <h3>私密帖子不开放评论</h3>
                <p>将帖子公开后，其他人才能加入讨论。</p>
              </div>
            ) : (
              <>
                {isAuthenticated ? (
                  <form className="comment-composer" onSubmit={submitComment}>
                    <Avatar user={user} size="sm" />
                    <textarea value={commentText} onChange={(event) => setCommentText(event.target.value)} placeholder="写下你的回应…" aria-label="评论内容" required />
                    <button className="button button--dark button--small" type="submit" disabled={submittingComment || !commentText.trim()}>
                      <Send size={14} aria-hidden="true" />{submittingComment ? '发送中' : '发送'}
                    </button>
                  </form>
                ) : (
                  <button className="button button--soft button--wide" type="button" onClick={() => openAuth('login')}>登录后参与讨论</button>
                )}

                {comments.length === 0 ? (
                  <div className="empty-state">
                    <div className="empty-state__icon"><MessageCircle size={23} aria-hidden="true" /></div>
                    <h3>还没有回应</h3>
                    <p>读完有一点共鸣？留下第一条评论吧。</p>
                  </div>
                ) : (
                  <div className="comment-list">
                    {comments.map((comment) => (
                      <article className="comment" key={comment.id}>
                        <Avatar user={comment.user} size="sm" />
                        <div>
                          <div className="comment__top">
                            <strong>{getDisplayName(comment.user)}</strong>
                            <span>{formatRelativeTime(comment.create_time)}</span>
                            {user?.id === comment.user_id && comment.id > 0 && (
                              <button className="text-button comment__delete" type="button" onClick={() => setDeleteTarget(comment)} aria-label="删除评论">
                                <Trash2 size={14} aria-hidden="true" />
                              </button>
                            )}
                          </div>
                          <p>{comment.content}</p>
                        </div>
                      </article>
                    ))}
                  </div>
                )}
                {hasMoreComments && (
                  <div className="load-more">
                    <button className="button button--soft" type="button" onClick={() => void loadMoreComments()} disabled={loadingComments}>
                      {loadingComments ? '正在加载…' : '展开更多评论'}
                    </button>
                  </div>
                )}
              </>
            )}
          </section>
        </div>

        <aside className="side-rail" aria-label="帖子信息">
          <section className="rail-card rail-card--accent">
            <span className="rail-card__index">POST / {String(detail.post.id).padStart(4, '0')}</span>
            <h3>慢一点，读完它。</h3>
            <p>好的交流不是等着反驳，而是先听清对方真正想说什么。</p>
          </section>
          <section className="rail-card">
            <span className="rail-card__index">UPDATED</span>
            <p>{formatDateTime(detail.post.update_time)}</p>
          </section>
        </aside>
      </div>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除这条评论？"
        description="删除后无法恢复。"
        confirmLabel="确认删除"
        danger
        busy={deletingComment}
        onConfirm={() => void deleteComment()}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  )
}
