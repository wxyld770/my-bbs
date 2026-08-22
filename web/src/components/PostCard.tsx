import { ArrowUpRight, Eye, EyeOff, Pencil, Trash2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatRelativeTime, getDisplayName } from '../lib/format'
import { getFirstPostImage, getPostTextSummary } from '../lib/postContent'
import { POST_VISIBILITY, type Post } from '../types'
import { Avatar } from './Avatar'
import { ConfirmDialog } from './ConfirmDialog'

interface PostCardProps {
  post: Post
  manageable?: boolean
  onChanged?: () => void
}

export function PostCard({ post, manageable = false, onChanged }: PostCardProps) {
  const { token, handleSessionError } = useAuth()
  const { openAuth, openComposer, notify, refreshContent } = useUI()
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [busyAction, setBusyAction] = useState<'delete' | 'visibility' | null>(null)
  const [previewFailed, setPreviewFailed] = useState(false)
  const previewImage = getFirstPostImage(post.content)

  useEffect(() => {
    setPreviewFailed(false)
  }, [previewImage])

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

  return (
    <article className="post-card">
      <Link className="post-card__link" to={`/post/${post.id}`}>
        <div className="post-card__meta">
          <Avatar user={post.user} size="sm" />
          <div className="post-card__author">
            <strong>{getDisplayName(post.user)}</strong>
            <span>{formatRelativeTime(post.create_time)}</span>
          </div>
          <span className={`post-card__tag${post.visible === POST_VISIBILITY.PRIVATE ? ' private' : ''}`}>
            {post.visible === POST_VISIBILITY.PRIVATE ? '仅自己' : '公开'}
          </span>
        </div>
        <h3>{post.title}</h3>
        <p className="post-card__excerpt">{getPostTextSummary(post.content, 150)}</p>
        {previewImage && !previewFailed && (
          <span className="post-card__image" aria-hidden="true">
            <img
              src={previewImage}
              alt=""
              loading="lazy"
              decoding="async"
              fetchPriority="low"
              referrerPolicy="no-referrer"
              onError={() => setPreviewFailed(true)}
            />
          </span>
        )}
        <div className="post-card__footer">
          <span>{post.update_time === post.create_time ? '初次发布' : `更新于 ${formatRelativeTime(post.update_time)}`}</span>
          <span className="post-card__read">读一读 <ArrowUpRight size={15} aria-hidden="true" /></span>
        </div>
      </Link>

      {manageable && (
        <div className="post-card__actions" aria-label="帖子管理操作">
          <button className="button button--soft button--small" type="button" onClick={() => openComposer(post)}>
            <Pencil size={14} aria-hidden="true" />编辑
          </button>
          <button className="button button--soft button--small" type="button" onClick={toggleVisibility} disabled={Boolean(busyAction)}>
            {post.visible === POST_VISIBILITY.PUBLIC ? <EyeOff size={14} aria-hidden="true" /> : <Eye size={14} aria-hidden="true" />}
            {busyAction === 'visibility' ? '处理中…' : post.visible === POST_VISIBILITY.PUBLIC ? '转为私密' : '公开'}
          </button>
          <button className="button button--danger button--small" type="button" onClick={() => setConfirmDelete(true)} disabled={Boolean(busyAction)}>
            <Trash2 size={14} aria-hidden="true" />删除
          </button>
        </div>
      )}

      <ConfirmDialog
        open={confirmDelete}
        title="删除这篇帖子？"
        description="删除后无法恢复，与它相关的内容也不会再出现在广场中。"
        confirmLabel="确认删除"
        danger
        busy={busyAction === 'delete'}
        onConfirm={deletePost}
        onCancel={() => setConfirmDelete(false)}
      />
    </article>
  )
}
