import { ArrowLeft, Heart, LockKeyhole, MessageCircle, Pin, RefreshCw, Send, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Avatar } from '../components/Avatar'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { PostContent } from '../components/PostContent'
import { PinControl } from '../components/PinControl'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatCount, formatDateTime, formatRelativeTime, getDisplayName } from '../lib/format'
import { POST_VISIBILITY, type Comment, type PostDetail, type PostPinDuration } from '../types'

const COMMENT_PAGE_SIZE = 20

const READING_PROMPTS = [
  {
    title: '慢一点，也没关系。',
    body: '你不必在每一天都成为更好的人，能照顾好此刻的自己就很了不起。',
  },
  {
    title: '今天也值得期待。',
    body: '生活偶尔会把答案藏起来，但它从没有忘记为认真前行的人留一盏灯。',
  },
  {
    title: '允许自己休息。',
    body: '停下来不是后退，而是把丢在路上的勇气一点点捡回来。',
  },
  {
    title: '别急着否定自己。',
    body: '那些暂时没有回声的努力，也在悄悄地把你带往更远的地方。',
  },
  {
    title: '向前走一点点。',
    body: '你不需要一次走完所有的路，今天比昨天多走一步就很好。',
  },
  {
    title: '你已经做得很好。',
    body: '走到今天的你，曾经也穿过许多以为迈不过去的黑夜。',
  },
  {
    title: '保持自己的节奏。',
    body: '别人的花期不是你的时钟，你只需要在自己的季节里好好生长。',
  },
  {
    title: '好事正在发生。',
    body: '眼前的平静不代表没有变化，有些美好正在你看不见的地方生根。',
  },
  {
    title: '把今天过好。',
    body: '不必提前为明天的风雨担心，此刻的阳光也值得你认真收藏。',
  },
  {
    title: '愿你忠于自己。',
    body: '不用把每一次选择都做给别人看，内心安静就是最好的方向。',
  },
  {
    title: '夜晚总会过去。',
    body: '再漫长的黑夜也有尽头，你要做的只是再相信黎明一次。',
  },
  {
    title: '生活会慢慢回答。',
    body: '你此刻的迷茫并不是结局，它只是答案出现以前的一小段留白。',
  },
  {
    title: '给自己一点时间。',
    body: '不是所有问题都要在今天解决，有些答案会在你安静生活时慢慢浮现。',
  },
  {
    title: '你不必事事完美。',
    body: '真实的生活本就带着缺口，愿意继续往前走就已经值得肯定。',
  },
  {
    title: '风会带来新的方向。',
    body: '眼前的路暂时不清楚也没关系，当你继续出发，新的方向自会出现。',
  },
  {
    title: '小小的进步也算数。',
    body: '不用等到翻越高山才庆祝，每一次没有放弃都是你留下的里程碑。',
  },
  {
    title: '答案还在路上。',
    body: '一时的沉默不是拒绝，生活只是在用它的速度为你准备回音。',
  },
  {
    title: '先把自己照顾好。',
    body: '你可以关心很多人，也别忘了给自己留一份同样认真的温柔。',
  },
  {
    title: '别怕走得慢。',
    body: '只要你还在朝着想去的方向前进，慢一些也是属于自己的风景。',
  },
  {
    title: '重新开始也是勇敢。',
    body: '改变方向不代表之前的努力白费，那些经历已经成为你新的底气。',
  },
  {
    title: '你值得被温柔对待。',
    body: '不必用委屈换来理解，真正重要的关系会尊重你的感受和边界。',
  },
  {
    title: '没有白走的路。',
    body: '那些曾让你困惑的绕路，总会在未来的某一天变成理解世界的新角度。',
  },
  {
    title: '把烦恼交给明天。',
    body: '今天已经走了很远，先让自己好好休息，醒来时世界会有新的空气。',
  },
  {
    title: '相信时间的力量。',
    body: '一时放不下的事不用勉强，时间会帮你留下珍贵的，也带走沉重的。',
  },
  {
    title: '心里有光，脚下就有路。',
    body: '哪怕只剩一点小小的期待，它也能陪你走过眼前这段不太容易的路。',
  },
  {
    title: '不必讨好所有人。',
    body: '你的人生不是一张需要每个人都满意的答卷，坦然做自己就很好。',
  },
  {
    title: '你的感受很重要。',
    body: '难过不需要先得到别人的认可，你可以承认它，也可以慢慢安放它。',
  },
  {
    title: '等一等花开。',
    body: '暂时看不见成果的日子里，根也在努力生长，请再给自己一点耐心。',
  },
  {
    title: '勇敢不是毫不害怕。',
    body: '真正的勇敢是带着心里的忐忑，仍然愿意为想要的生活向前一步。',
  },
  {
    title: '明天会有新的风景。',
    body: '今天没能如愿的事并不代表结束，新的一天会带来重新选择的可能。',
  },
] as const

export function PostPage() {
  const { id } = useParams()
  const postId = Number(id)
  const navigate = useNavigate()
  const { token, user, isAuthenticated, canWrite, handleSessionError } = useAuth()
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
  const [confirmDeletePost, setConfirmDeletePost] = useState(false)
  const [postAction, setPostAction] = useState<'delete' | 'pin' | null>(null)

  const validId = Number.isSafeInteger(postId) && postId > 0
  const readingPrompt = useMemo(
    () => READING_PROMPTS[Math.floor(Math.random() * READING_PROMPTS.length)],
    [postId],
  )

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
  const isAdmin = user?.is_admin === true
  const commentsEnabled = detail?.post.visible === POST_VISIBILITY.PUBLIC
  const canPin = detail?.post.visible === POST_VISIBILITY.PUBLIC

  const authorLink = useMemo(() => {
    if (!detail?.post.user) return '/'
    return user?.id === detail.post.user.id ? '/me' : `/u/${detail.post.user.id}`
  }, [detail, user])

  const toggleLike = async () => {
    if (!isAuthenticated || !token) {
      openAuth('login')
      return
    }
    if (!canWrite) return
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

  const handlePostActionFailure = (action: string, actionError: unknown) => {
    if (handleSessionError(actionError)) {
      openAuth('login')
      notify('info', '登录状态已失效', '请重新登录后继续。')
      return
    }
    notify('error', `${action}没有完成`, getErrorMessage(actionError))
  }

  const togglePin = async (duration?: PostPinDuration) => {
    if (!token || !detail || !isAdmin || !canWrite) return
    if (!detail.post.is_pinned && !canPin) return
    setPostAction('pin')
    try {
      if (detail.post.is_pinned) {
        await api.unpinPost(token, postId)
        setDetail((current) => current ? {
          ...current,
          post: { ...current.post, is_pinned: false, is_permanent: false, pinned_until: null },
        } : current)
        notify('success', '已取消置顶')
      } else {
        if (!duration) return
        const result = await api.pinPost(token, postId, duration)
        setDetail((current) => current ? {
          ...current,
          post: { ...current.post, is_pinned: true, is_permanent: result.is_permanent, pinned_until: result.pinned_until },
        } : current)
        notify('success', result.is_permanent ? '帖子已永久置顶' : '帖子已置顶', result.is_permanent ? undefined : `将在 ${formatDateTime(result.pinned_until)} 自动取消置顶。`)
      }
    } catch (actionError) {
      handlePostActionFailure(detail.post.is_pinned ? '取消置顶' : '置顶', actionError)
    } finally {
      setPostAction(null)
    }
  }

  const deletePost = async () => {
    if (!token || !detail || !isAdmin || !canWrite) return
    setPostAction('delete')
    try {
      await api.deletePost(token, postId)
      setConfirmDeletePost(false)
      notify('success', '帖子已删除')
      navigate('/', { replace: true })
    } catch (actionError) {
      handlePostActionFailure('删除', actionError)
    } finally {
      setPostAction(null)
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
    if (!canWrite) return
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
    if (!deleteTarget || !token || !canWrite || deleteTarget.id < 1) return
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
        {canWrite && (isOwner || isAdmin) && (
          <div className="page-header__actions" aria-label="帖子操作">
            {isOwner && <button className="button button--soft button--small" type="button" onClick={() => openComposer(detail.post)}>编辑这篇</button>}
            {isAdmin && (
              <>
                <PinControl
                  isPinned={detail.post.is_pinned}
                  canPin={canPin}
                  busy={Boolean(postAction)}
                  postTitle={detail.post.title}
                  onPin={togglePin}
                  onUnpin={() => togglePin()}
                />
                <button className="button button--danger button--small" type="button" onClick={() => setConfirmDeletePost(true)} disabled={Boolean(postAction)}>
                  <Trash2 size={14} aria-hidden="true" />删除帖子
                </button>
              </>
            )}
          </div>
        )}
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
              <div className="article-card__badges">
                {detail.post.is_pinned && (
                  <span className="post-card__tag pinned" title={detail.post.is_permanent ? '永久置顶' : detail.post.pinned_until ? `置顶至 ${formatDateTime(detail.post.pinned_until)}` : '已置顶'}>
                    <Pin size={12} aria-hidden="true" /> {detail.post.is_permanent ? '永久置顶' : '置顶'}
                  </span>
                )}
                {detail.post.visible === POST_VISIBILITY.PRIVATE && <span className="post-card__tag private"><LockKeyhole size={12} aria-hidden="true" /> 仅自己可见</span>}
              </div>
            </header>
            <div className="article-card__body">
              <PostContent content={detail.post.content} />
            </div>
            <footer className="article-card__actions">
              <button className={`stat-pill${detail.is_liked ? ' liked' : ''}`} type="button" onClick={() => void toggleLike()} disabled={liking || !commentsEnabled || !canWrite} title={!canWrite && isAuthenticated ? '账号已被禁言，当前为只读模式' : undefined}>
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
                {isAuthenticated && !canWrite ? (
                  <div className="read-only-notice" role="status">
                    <LockKeyhole size={18} aria-hidden="true" />
                    <span>账号已被禁言，当前可以浏览内容，但不能评论或点赞。</span>
                  </div>
                ) : isAuthenticated ? (
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
                            {canWrite && user?.id === comment.user_id && comment.id > 0 && (
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
            <h3>{readingPrompt.title}</h3>
            <p>{readingPrompt.body}</p>
          </section>
          <section className="rail-card">
            <span className="rail-card__index">UPDATED</span>
            <p>{formatDateTime(detail.post.update_time)}</p>
          </section>
        </aside>
      </div>

      <ConfirmDialog
        open={confirmDeletePost}
        title="删除这篇帖子？"
        description="这是管理员操作。删除后无法恢复，与它相关的内容也不会再出现在广场中。"
        confirmLabel="确认删除"
        danger
        busy={postAction === 'delete'}
        onConfirm={() => void deletePost()}
        onCancel={() => setConfirmDeletePost(false)}
      />

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
