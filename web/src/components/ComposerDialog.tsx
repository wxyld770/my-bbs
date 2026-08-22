import { useEffect, useState, type FormEvent, type MouseEvent } from 'react'
import { Feather, X } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'

export function ComposerDialog() {
  const { token, handleSessionError } = useAuth()
  const { composer, closeComposer, openAuth, notify, refreshContent } = useUI()
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!composer) return
    setTitle(composer.mode === 'edit' ? composer.post.title : '')
    setContent(composer.mode === 'edit' ? composer.post.content : '')
    setError('')
  }, [composer])

  useEffect(() => {
    if (!composer) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !submitting) closeComposer()
    }
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = ''
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [closeComposer, composer, submitting])

  if (!composer) return null

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    const cleanTitle = title.trim()
    const cleanContent = content.trim()
    if (!cleanTitle) {
      setError('给这段话取一个标题吧')
      return
    }
    if (!cleanContent) {
      setError('正文还没有内容')
      return
    }
    if (!token) {
      closeComposer()
      openAuth('login')
      return
    }

    setSubmitting(true)
    try {
      if (composer.mode === 'edit') {
        await api.updatePost(token, composer.post.id, { title: cleanTitle, content: cleanContent })
      } else {
        await api.createPost(token, { title: cleanTitle, content: cleanContent })
      }
      closeComposer()
      refreshContent()
      notify('success', composer.mode === 'edit' ? '内容已更新' : '发布成功', composer.mode === 'edit' ? '修改已经保存。' : '你的话已经出现在广场上。')
    } catch (submitError) {
      if (handleSessionError(submitError)) {
        closeComposer()
        openAuth('login')
        notify('info', '登录状态已失效', '请重新登录后继续。')
      } else {
        setError(getErrorMessage(submitError))
      }
    } finally {
      setSubmitting(false)
    }
  }

  const stopPropagation = (event: MouseEvent<HTMLDivElement>) => event.stopPropagation()

  return (
    <div className="dialog-backdrop" onMouseDown={() => !submitting && closeComposer()}>
      <div className="dialog dialog--wide" role="dialog" aria-modal="true" aria-labelledby="composer-title" onMouseDown={stopPropagation}>
        <div className="dialog__header">
          <div>
            <span className="eyebrow"><Feather size={14} aria-hidden="true" />{composer.mode === 'edit' ? '继续打磨' : '写给正在看的人'}</span>
            <h2 id="composer-title">{composer.mode === 'edit' ? '编辑帖子' : '说点什么'}</h2>
          </div>
          <button className="icon-button" type="button" onClick={closeComposer} aria-label="关闭" disabled={submitting}>
            <X size={18} aria-hidden="true" />
          </button>
        </div>
        <div className="dialog__body">
          <form onSubmit={onSubmit}>
            <div className="field">
              <label htmlFor="post-title">标题 <span>{Array.from(title).length}/255</span></label>
              <input id="post-title" value={title} onChange={(event) => setTitle(event.target.value)} placeholder="一句话说清你想聊什么" maxLength={255} required autoFocus />
            </div>
            <div className="field">
              <label htmlFor="post-content">正文 <span>{new Blob([content]).size}/65,535 bytes</span></label>
              <textarea id="post-content" style={{ minHeight: 260 }} value={content} onChange={(event) => setContent(event.target.value)} placeholder="慢慢写，认真说。这里支持换行。" required />
            </div>
            {error && <p className="form-error" role="alert">{error}</p>}
            <div className="form-actions">
              <button className="button button--soft" type="button" onClick={closeComposer} disabled={submitting}>先不写了</button>
              <button className="button button--primary" type="submit" disabled={submitting || new Blob([content]).size > 65535}>
                {submitting ? '正在保存…' : composer.mode === 'edit' ? '保存修改' : '发布到广场'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
