import { Eye, EyeOff, Heart, MessageCircle, Pencil, Pin, Trash2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatCount, formatDateTime, formatRelativeTime, getDisplayName } from '../lib/format'
import { POST_VISIBILITY, type PostListItem, type PostPinDuration } from '../types'
import { Avatar } from './Avatar'
import { ConfirmDialog } from './ConfirmDialog'
import { PinControl } from './PinControl'

interface PostCardProps {
  post: PostListItem
  manageable?: boolean
  onChanged?: () => void
}

export function PostCard({ post, manageable = false, onChanged }: PostCardProps) {
  const { token, user, canWrite, handleSessionError } = useAuth()
  const { openAuth, openComposer, notify, refreshContent } = useUI()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busyAction, setBusyAction] = useState<'delete' | 'edit' | 'pin' | 'visibility' | null>(null)
  const isAdmin = user?.is_admin === true
  const hasActions = canWrite && (manageable || isAdmin)
  const hasTags = manageable || post.is_pinned
  const canPin = post.visible === POST_VISIBILITY.PUBLIC
  const hasLikeCount = Number.isFinite(post.like_count)
  const hasCommentCount = Number.isFinite(post.comment_count)
  const likeCount = hasLikeCount ? formatCount(post.like_count) : '—'
  const commentCount = hasCommentCount ? formatCount(post.comment_count) : '—'

  const handleFailure = (error: unknown) => {
    if (handleSessionError(error)) {
      openAuth('login')
      notify('info', '登录状态已失效', '请重新登录后继续。')
      return
    }
    notify('error', '操作没有完成', getErrorMessage(error))
  }

  const toggleVisibility = async () => {
    if (!token) {
      openAuth('login')
      return
    }
    setBusyAction('visibility')
    try {
      const next = post.visible === POST_VISIBILITY.PUBLIC ? POST_VISIBILITY.PRIVATE : POST_VISIBILITY.PUBLIC
      await api.setPostVisibility(token, post.id, next)
      notify('success', next === POST_VISIBILITY.PUBLIC ? '已公开' : '已设为仅自己可见')
      refreshContent()
      onChanged?.()
    } catch (error) {
      handleFailure(error)
    } finally {
      setBusyAction(null)
    }
  }

  const editPost = async () => {
    if (!token) {
      openAuth('login')
      return
    }
    setBusyAction('edit')
    try {
      const detail = await api.getPost(post.id, token)
      openComposer(detail.post)
    } catch (error) {
      handleFailure(error)
    } finally {
      setBusyAction(null)
    }
  }

  const deletePost = async () => {
    if (!token) {
      setConfirmDelete(false)
      openAuth('login')
      return
    }
    setBusyAction('delete')
    try {
      await api.deletePost(token, post.id)
      setConfirmDelete(false)
      notify('success', '帖子已删除')
      refreshContent()
      onChanged?.()
    } catch (error) {
      handleFailure(error)
    } finally {
      setBusyAction(null)
    }
  }

  const togglePin = async (duration?: PostPinDuration) => {
    if (!token) {
      openAuth('login')
      return
    }
    if (!isAdmin) return
    if (!post.is_pinned && !canPin) return
    setBusyAction('pin')
    try {
      if (post.is_pinned) {
        await api.unpinPost(token, post.id)
        notify('success', '已取消置顶')
      } else {
        if (!duration) return
        const result = await api.pinPost(token, post.id, duration)
        notify('success', result.is_permanent ? '帖子已永久置顶' : '帖子已置顶', result.is_permanent ? undefined : `将在 ${formatDateTime(result.pinned_until)} 自动取消置顶。`)
      }
      refreshContent()
      onChanged?.()
    } catch (error) {
      handleFailure(error)
    } finally {
      setBusyAction(null)
    }
  }

  return (
    <>
      <article className={`post-card${hasActions ? ' post-card--manageable' : ''}${post.is_pinned ? ' post-card--pinned' : ''}`}>
        <Link className={`post-card__link${hasTags ? ' post-card__link--manageable' : ''}`} to={`/post/${post.id}`}>
          <div className="post-card__meta">
            <Avatar user={post.user} size="sm" />
            <div className="post-card__author">
              <strong>{getDisplayName(post.user)}</strong>
              <span>{formatRelativeTime(post.create_time)}</span>
            </div>
          </div>
          <h3 title={post.title}>{post.title}</h3>
          <span className="post-card__stats">
            <span aria-label={hasLikeCount ? `${likeCount} 次点赞` : '点赞数将在接口更新后显示'}><Heart size={15} aria-hidden="true" />{likeCount}</span>
            <span aria-label={hasCommentCount ? `${commentCount} 条评论` : '评论数将在接口更新后显示'}><MessageCircle size={15} aria-hidden="true" />{commentCount}</span>
          </span>
          {hasTags && (
            <span className="post-card__tags">
              {post.is_pinned && (
                <span className="post-card__tag pinned" title={post.is_permanent ? '永久置顶' : post.pinned_until ? `置顶至 ${formatDateTime(post.pinned_until)}` : '已置顶'}>
                  <Pin size={12} aria-hidden="true" />{post.is_permanent ? '永久置顶' : '置顶'}
                </span>
              )}
              {manageable && (
                <span className={`post-card__tag${post.visible === POST_VISIBILITY.PRIVATE ? ' private' : ''}`}>
                  {post.visible === POST_VISIBILITY.PRIVATE ? '仅自己' : '公开'}
                </span>
              )}
            </span>
          )}
        </Link>

        {hasActions && (
          <div className="post-card__actions" aria-label="帖子管理操作">
            {manageable && (
              <>
                <button className="button button--soft button--small" type="button" onClick={() => void editPost()} disabled={Boolean(busyAction)} aria-label={`编辑《${post.title}》`}>
                  <Pencil size={14} aria-hidden="true" />{busyAction === 'edit' ? '读取中…' : '编辑'}
                </button>
                <button className="button button--soft button--small" type="button" onClick={toggleVisibility} disabled={Boolean(busyAction)} aria-label={`${post.visible === POST_VISIBILITY.PUBLIC ? '转为私密' : '公开'}《${post.title}》`}>
                  {post.visible === POST_VISIBILITY.PUBLIC ? <EyeOff size={14} aria-hidden="true" /> : <Eye size={14} aria-hidden="true" />}
                  {busyAction === 'visibility' ? '处理中…' : post.visible === POST_VISIBILITY.PUBLIC ? '转为私密' : '公开'}
                </button>
              </>
            )}
            {isAdmin && (
              <PinControl
                isPinned={post.is_pinned}
                canPin={canPin}
                busy={Boolean(busyAction)}
                postTitle={post.title}
                onPin={togglePin}
                onUnpin={() => togglePin()}
              />
            )}
            {(manageable || isAdmin) && (
              <button className="button button--danger button--small" type="button" onClick={() => setConfirmDelete(true)} disabled={Boolean(busyAction)} aria-label={`删除《${post.title}》`}>
                <Trash2 size={14} aria-hidden="true" />删除
              </button>
            )}
          </div>
        )}
      </article>

      <ConfirmDialog
        open={confirmDelete}
        title="删除这篇帖子？"
        description={isAdmin && !manageable
          ? '这是管理员操作。删除后无法恢复，与它相关的内容也不会再出现在广场中。'
          : '删除后无法恢复，与它相关的内容也不会再出现在广场中。'}
        confirmLabel="确认删除"
        danger
        busy={busyAction === 'delete'}
        onConfirm={deletePost}
        onCancel={() => setConfirmDelete(false)}
      />
    </>
  )
}
