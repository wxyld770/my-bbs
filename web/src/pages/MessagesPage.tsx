import {
  ArrowRight,
  Inbox,
  LogIn,
  MessageSquareText,
  RefreshCw,
  Send,
  ShieldCheck,
} from 'lucide-react'
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from 'react'
import { Link } from 'react-router-dom'
import { Avatar } from '../components/Avatar'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatDateTime, formatRelativeTime, getDisplayName } from '../lib/format'
import type { SiteMessage } from '../types'

const PAGE_SIZE = 12
const MAX_CONTENT_LENGTH = 2_000

interface MessageSubmitRequest {
  controller: AbortController
  generation: number
  ownerKey: string
  token: string
}

export function MessagesPage() {
  const {
    token,
    user,
    isAuthenticated,
    isBootstrapping,
    isCurrentSession,
    handleSessionError,
  } = useAuth()
  const { openAuth, notify } = useUI()
  const [messages, setMessages] = useState<SiteMessage[]>([])
  const [pageNo, setPageNo] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [listError, setListError] = useState('')
  const [content, setContent] = useState('')
  const [formError, setFormError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [messagesOwnerKey, setMessagesOwnerKey] = useState('')
  const loadGeneration = useRef(0)

  const isAdmin = user?.is_admin === true
  const ownerKey = token && user ? `${user.id}:${isAdmin ? 'admin' : 'member'}` : ''
  const submitRequestRef = useRef<MessageSubmitRequest | null>(null)
  const submitGenerationRef = useRef(0)
  const latestOwnerKeyRef = useRef(ownerKey)
  const latestTokenRef = useRef(token)
  latestOwnerKeyRef.current = ownerKey
  latestTokenRef.current = token
  const ownerChanging = messagesOwnerKey !== ownerKey
  const visibleMessages = ownerChanging ? [] : messages
  const listLoading = loading || ownerChanging
  const contentLength = Array.from(content).length

  const loadFirstPage = useCallback(async (signal?: AbortSignal) => {
    const generation = ++loadGeneration.current
    setPageNo(1)
    setHasMore(false)
    setLoadingMore(false)

    if (!token || !user) {
      setMessages([])
      setMessagesOwnerKey('')
      setListError('')
      setLoading(false)
      return
    }
    const submittedToken = token

    setLoading(true)
    setListError('')
    try {
      const page = isAdmin
        ? await api.listAdminMessages(submittedToken, { pageNo: 1, pageSize: PAGE_SIZE }, signal)
        : await api.listMyMessages(submittedToken, { pageNo: 1, pageSize: PAGE_SIZE }, signal)
      if (signal?.aborted || generation !== loadGeneration.current || !isCurrentSession(submittedToken)) return
      setMessages(page.list)
      setPageNo(1)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      if (signal?.aborted || generation !== loadGeneration.current || !isCurrentSession(submittedToken)) return
      if (handleSessionError(loadError, submittedToken)) openAuth('login')
      else setListError(getErrorMessage(loadError))
    } finally {
      if (!signal?.aborted && generation === loadGeneration.current && isCurrentSession(submittedToken)) setLoading(false)
    }
  }, [handleSessionError, isAdmin, isCurrentSession, openAuth, token, user])

  useEffect(() => {
    const controller = new AbortController()
    // 留言属于当前认证主体的私密数据。主体或管理员模式变化时必须先
    // 清空旧列表，不能在新请求失败后继续展示上一主体的数据。
    loadGeneration.current += 1
    setMessages([])
    setMessagesOwnerKey(ownerKey)
    setPageNo(1)
    setHasMore(false)
    setLoadingMore(false)
    setListError('')
    void loadFirstPage(controller.signal)
    return () => {
      loadGeneration.current += 1
      controller.abort()
    }
  }, [loadFirstPage, ownerKey])

  useEffect(() => {
    setSubmitting(false)
    setContent('')
    setFormError('')
    return () => {
      submitGenerationRef.current += 1
      submitRequestRef.current?.controller.abort()
      submitRequestRef.current = null
    }
  }, [ownerKey, token])

  const loadMore = async () => {
    if (!token || !user || loadingMore || !hasMore) return
    const submittedToken = token
    const generation = loadGeneration.current
    const nextPage = pageNo + 1
    setLoadingMore(true)
    setListError('')
    try {
      const page = isAdmin
        ? await api.listAdminMessages(submittedToken, { pageNo: nextPage, pageSize: PAGE_SIZE })
        : await api.listMyMessages(submittedToken, { pageNo: nextPage, pageSize: PAGE_SIZE })
      if (generation !== loadGeneration.current || !isCurrentSession(submittedToken)) return
      setMessages((current) => {
        const knownIds = new Set(current.map((message) => message.id))
        return [...current, ...page.list.filter((message) => !knownIds.has(message.id))]
      })
      setPageNo(nextPage)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      if (generation !== loadGeneration.current || !isCurrentSession(submittedToken)) return
      if (handleSessionError(loadError, submittedToken)) openAuth('login')
      else setListError(getErrorMessage(loadError))
    } finally {
      if (generation === loadGeneration.current && isCurrentSession(submittedToken)) setLoadingMore(false)
    }
  }

  const submitMessage = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!token || !user || isAdmin || submitting || submitRequestRef.current) return
    const submittedToken = token
    const submittedOwnerKey = ownerKey

    const normalizedContent = content.trim()
    const normalizedLength = Array.from(normalizedContent).length
    if (normalizedLength === 0) {
      setFormError('请先写下需要管理员协助处理的问题。')
      return
    }
    if (normalizedLength > MAX_CONTENT_LENGTH) {
      setFormError(`留言不能超过 ${MAX_CONTENT_LENGTH} 个字符。`)
      return
    }

    const controller = new AbortController()
    const generation = submitGenerationRef.current + 1
    submitGenerationRef.current = generation
    const request: MessageSubmitRequest = {
      controller,
      generation,
      ownerKey: submittedOwnerKey,
      token: submittedToken,
    }
    submitRequestRef.current = request
    const requestIsCurrent = () => (
      !controller.signal.aborted &&
      submitRequestRef.current === request &&
      submitGenerationRef.current === generation &&
      isCurrentSession(submittedToken) &&
      latestOwnerKeyRef.current === submittedOwnerKey &&
      latestTokenRef.current === submittedToken
    )

    setSubmitting(true)
    setFormError('')
    try {
      await api.createMessage(
        submittedToken,
        { content: normalizedContent },
        controller.signal,
      )
      if (!requestIsCurrent()) return
      setContent('')
      notify('success', '留言已送达', '管理员可以在留言列表中看到它。')
      await loadFirstPage(controller.signal)
    } catch (submitError) {
      if (!requestIsCurrent()) return
      if (handleSessionError(submitError, submittedToken)) openAuth('login')
      else setFormError(getErrorMessage(submitError))
    } finally {
      const requestWasCurrent = requestIsCurrent()
      if (submitRequestRef.current === request) submitRequestRef.current = null
      if (requestWasCurrent) setSubmitting(false)
    }
  }

  const updateContent = (value: string) => {
    setContent(Array.from(value).slice(0, MAX_CONTENT_LENGTH).join(''))
    if (formError) setFormError('')
  }

  if (isBootstrapping) {
    return <div className="page-wrap"><div className="skeleton-card" style={{ height: 430 }} /></div>
  }

  if (!isAuthenticated || !token || !user) {
    return (
      <div className="page-wrap">
        <div className="empty-state">
          <div className="empty-state__icon"><LogIn size={23} aria-hidden="true" /></div>
          <h3>登录后使用留言</h3>
          <p>你可以向管理员说明遇到的问题，并在这里查看自己发起过的留言。</p>
          <button className="button button--dark" type="button" onClick={() => openAuth('login')}>去登录</button>
        </div>
      </div>
    )
  }

  return (
    <div className="page-wrap messages-page">
      <section className={`messages-hero${isAdmin ? ' messages-hero--admin' : ''}`} aria-labelledby="messages-title">
        <div>
          <span className="eyebrow">
            {isAdmin
              ? <><ShieldCheck size={14} aria-hidden="true" />ADMIN INBOX</>
              : <><MessageSquareText size={14} aria-hidden="true" />MESSAGE TO ADMIN</>}
          </span>
          <h1 id="messages-title">{isAdmin ? '用户留言' : '给管理员留句话'}</h1>
          <p>
            {isAdmin
              ? '查看全部用户留言，并进入用户主页处理禁言申诉、密码重置等问题。'
              : '遇到账号、申诉或使用问题时，可以在这里告诉管理员；只有你和管理员能看到。'}
          </p>
        </div>
        <div className="messages-hero__mark" aria-hidden="true">
          <Inbox size={30} />
        </div>
      </section>

      <div className={`messages-layout${isAdmin ? ' messages-layout--admin' : ''}`}>
        {!isAdmin && (
          <aside className="form-card message-composer" aria-labelledby="message-compose-heading">
            <h2 id="message-compose-heading">发起留言</h2>
            <form onSubmit={submitMessage}>
              <div className="field">
                <label htmlFor="message-content">
                  问题说明
                  <span>{contentLength}/{MAX_CONTENT_LENGTH}</span>
                </label>
                <textarea
                  id="message-content"
                  value={content}
                  onChange={(event) => updateContent(event.target.value)}
                  placeholder="请说明遇到的问题，以及希望管理员协助处理的事项。"
                  rows={9}
                  disabled={submitting}
                  aria-describedby={`message-content-help${formError ? ' message-content-error' : ''}`}
                  aria-invalid={Boolean(formError)}
                />
              </div>
              <p className="field-help" id="message-content-help">请勿填写密码、邀请码等敏感信息。即使账号被禁言，也可以在这里提交申诉。</p>
              {formError && <p className="form-error" id="message-content-error" role="alert">{formError}</p>}
              <div className="form-actions">
                <button className="button button--dark button--wide" type="submit" disabled={submitting || content.trim().length === 0} aria-busy={submitting}>
                  <Send size={15} aria-hidden="true" />
                  {submitting ? '发送中…' : '发送给管理员'}
                </button>
              </div>
            </form>
          </aside>
        )}

        <section className="message-history" aria-labelledby="message-history-heading" aria-busy={listLoading || loadingMore}>
          <div className="section-heading">
            <div>
              <span className="eyebrow">{isAdmin ? 'ALL MESSAGES' : 'MY HISTORY'}</span>
              <h2 id="message-history-heading">{isAdmin ? '全部留言' : '历史留言'}</h2>
              <p>{isAdmin ? '按最新留言优先排列。' : '这里只会显示你自己发起的内容。'}</p>
            </div>
            {!listLoading && <span className="section-count">LOADED / {String(visibleMessages.length).padStart(2, '0')}</span>}
          </div>

          {listLoading ? (
            <div className="message-list" aria-label="正在加载留言">
              <div className="skeleton-card" />
              <div className="skeleton-card" />
              <div className="skeleton-card" />
            </div>
          ) : listError && visibleMessages.length === 0 ? (
            <div className="error-state">
              <div className="error-state__icon"><RefreshCw size={23} aria-hidden="true" /></div>
              <h2>暂时没能读到留言</h2>
              <p>{listError}</p>
              <button className="button button--dark" type="button" onClick={() => void loadFirstPage()}>重新加载</button>
            </div>
          ) : visibleMessages.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state__icon"><MessageSquareText size={23} aria-hidden="true" /></div>
              <h3>{isAdmin ? '暂时没有用户留言' : '还没有留下过消息'}</h3>
              <p>{isAdmin ? '有用户发起问题时，留言会出现在这里。' : '需要协助时，在左侧写下第一条留言吧。'}</p>
            </div>
          ) : (
            <>
              <div className="message-list">
                {visibleMessages.map((message) => (
                  <MessageCard key={message.id} message={message} showAuthor={isAdmin} />
                ))}
              </div>
              {listError && <p className="form-error message-list__error" role="alert">{listError}</p>}
              {hasMore && (
                <div className="load-more">
                  <button className="button button--soft" type="button" onClick={() => void loadMore()} disabled={loadingMore} aria-busy={loadingMore}>
                    {loadingMore ? '加载中…' : <><span>继续加载</span><ArrowRight size={15} aria-hidden="true" /></>}
                  </button>
                </div>
              )}
            </>
          )}
        </section>
      </div>
    </div>
  )
}

function MessageCard({ message, showAuthor }: { message: SiteMessage; showAuthor: boolean }) {
  const author = message.user
  return (
    <article className="message-card">
      <header className="message-card__header">
        {showAuthor ? (
          author ? (
            <Link className="message-card__author" to={`/u/${author.id}`}>
              <Avatar user={author} size="sm" />
              <span>
                <strong>{getDisplayName(author)}</strong>
                <small>@{author.username}</small>
              </span>
            </Link>
          ) : (
            <span className="message-card__author-fallback">用户 #{message.user_id}</span>
          )
        ) : (
          <span className="message-card__mine"><MessageSquareText size={13} aria-hidden="true" />我的留言</span>
        )}
        <time dateTime={message.create_time} title={formatDateTime(message.create_time)}>
          {formatRelativeTime(message.create_time)}
        </time>
      </header>
      <p className="message-card__content">{message.content}</p>
      {showAuthor && author && (
        <Link className="message-card__profile-link" to={`/u/${author.id}`}>
          进入用户主页处理 <ArrowRight size={14} aria-hidden="true" />
        </Link>
      )}
    </article>
  )
}
