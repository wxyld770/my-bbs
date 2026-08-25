import {
  ArrowRight,
  FileText,
  Heart,
  MessageCircle,
  RefreshCw,
  Search as SearchIcon,
  Users,
} from 'lucide-react'
import { type FormEvent, useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Avatar } from '../components/Avatar'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import {
  formatCount,
  formatRelativeTime,
  getDisplayName,
} from '../lib/format'
import {
  SEARCH_SCOPE,
  type SearchPost,
  type SearchScope,
  type SearchUser,
} from '../types'

const PAGE_SIZE = 10

const SCOPE_OPTIONS: ReadonlyArray<{
  value: SearchScope
  label: string
}> = [
  { value: SEARCH_SCOPE.ALL, label: '全部' },
  { value: SEARCH_SCOPE.USERS, label: '用户' },
  { value: SEARCH_SCOPE.POSTS, label: '帖子' },
]

export function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const rawQuery = searchParams.get('q') ?? ''
  const query = normalizeSearchTerm(rawQuery)
  const queryValidationMessage = query ? validateSearchTerm(query) : ''
  const scope = parseSearchScope(searchParams.get('scope'))

  const [input, setInput] = useState(rawQuery)
  const [users, setUsers] = useState<SearchUser[]>([])
  const [posts, setPosts] = useState<SearchPost[]>([])
  const [pageNo, setPageNo] = useState(1)
  const [usersHasMore, setUsersHasMore] = useState(false)
  const [postsHasMore, setPostsHasMore] = useState(false)
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')
  const [formError, setFormError] = useState('')
  const [reloadVersion, setReloadVersion] = useState(0)
  const loadMoreController = useRef<AbortController | null>(null)

  useEffect(() => {
    setInput(rawQuery)
    setFormError('')
  }, [rawQuery])

  useEffect(() => {
    const firstPageController = new AbortController()
    loadMoreController.current?.abort()
    loadMoreController.current = null

    setUsers([])
    setPosts([])
    setPageNo(1)
    setUsersHasMore(false)
    setPostsHasMore(false)
    setError('')
    setLoadingMore(false)

    if (!query) {
      setLoading(false)
      return () => firstPageController.abort()
    }

    if (queryValidationMessage) {
      setLoading(false)
      setFormError(queryValidationMessage)
      return () => firstPageController.abort()
    }

    setLoading(true)
    void api
      .search(
        { q: query, scope, pageNo: 1, pageSize: PAGE_SIZE },
        firstPageController.signal,
      )
      .then((result) => {
        setUsers(result.users.list)
        setPosts(result.posts.list)
        setUsersHasMore(result.users.hasMore)
        setPostsHasMore(result.posts.hasMore)
      })
      .catch((searchError: unknown) => {
        if (!firstPageController.signal.aborted) {
          setError(getErrorMessage(searchError))
        }
      })
      .finally(() => {
        if (!firstPageController.signal.aborted) setLoading(false)
      })

    return () => {
      firstPageController.abort()
      loadMoreController.current?.abort()
    }
  }, [query, queryValidationMessage, rawQuery, reloadVersion, scope])

  const submitSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const normalized = normalizeSearchTerm(input)
    const validationMessage = validateSearchTerm(normalized)
    if (validationMessage) {
      setFormError(validationMessage)
      return
    }

    setFormError('')
    if (normalized === query) {
      setReloadVersion((value) => value + 1)
      return
    }
    setSearchParams({ q: normalized, scope })
  }

  const changeScope = (nextScope: SearchScope) => {
    if (nextScope === scope) return
    if (query) setInput(query)
    setFormError('')
    const nextParams: Record<string, string> = { scope: nextScope }
    if (query) nextParams.q = query
    setSearchParams(nextParams)
  }

  const loadMore = async () => {
    if (loadingMore || !query) return

    const controller = new AbortController()
    loadMoreController.current?.abort()
    loadMoreController.current = controller
    const nextPage = pageNo + 1

    setLoadingMore(true)
    setError('')
    try {
      const result = await api.search(
        { q: query, scope, pageNo: nextPage, pageSize: PAGE_SIZE },
        controller.signal,
      )
      if (controller.signal.aborted) return

      setUsers((current) => mergeByID(current, result.users.list))
      setPosts((current) => mergeByID(current, result.posts.list))
      setUsersHasMore(result.users.hasMore)
      setPostsHasMore(result.posts.hasMore)
      setPageNo(nextPage)
    } catch (searchError) {
      if (!controller.signal.aborted) setError(getErrorMessage(searchError))
    } finally {
      if (!controller.signal.aborted) setLoadingMore(false)
      if (loadMoreController.current === controller) {
        loadMoreController.current = null
      }
    }
  }

  const showUsers = scope !== SEARCH_SCOPE.POSTS
  const showPosts = scope !== SEARCH_SCOPE.USERS
  const hasResults = users.length > 0 || posts.length > 0
  const hasMore =
    (showUsers && usersHasMore) || (showPosts && postsHasMore)

  return (
    <div className="page-wrap page-wrap--search" aria-busy={loading || loadingMore}>
      <header className="search-hero">
        <span className="eyebrow">DISCOVER / SEARCH</span>
        <h1>找到想继续读的人与话题。</h1>
        <p>搜索用户名、昵称、帖子标题与正文中的关键词。</p>

        <form className="search-form" role="search" onSubmit={submitSearch}>
          <label className="sr-only" htmlFor="global-search-input">搜索用户和帖子</label>
          <SearchIcon size={21} aria-hidden="true" />
          <input
            id="global-search-input"
            type="search"
            value={input}
            onChange={(event) => {
              setInput(event.target.value)
              if (formError) setFormError('')
            }}
            placeholder="输入至少两个字符"
            autoComplete="off"
            enterKeyHint="search"
            aria-describedby={formError ? 'search-form-error' : undefined}
            aria-invalid={Boolean(formError)}
          />
          <button className="button button--dark" type="submit">搜索</button>
        </form>
        {formError && (
          <p className="form-error search-form__error" id="search-form-error" role="alert">
            {formError}
          </p>
        )}

        <div className="search-scopes" role="group" aria-label="搜索范围">
          {SCOPE_OPTIONS.map((option) => (
            <button
              className={`search-scope${scope === option.value ? ' active' : ''}`}
              type="button"
              key={option.value}
              onClick={() => changeScope(option.value)}
              aria-pressed={scope === option.value}
            >
              {option.label}
            </button>
          ))}
        </div>
      </header>

      {!query || queryValidationMessage ? (
        <SearchPrompt />
      ) : loading ? (
        <SearchLoading />
      ) : error && !hasResults ? (
        <div className="error-state search-state" role="alert">
          <div className="error-state__icon"><RefreshCw size={24} aria-hidden="true" /></div>
          <h2>搜索暂时没有完成</h2>
          <p>{error}</p>
          <button className="button button--dark" type="button" onClick={() => setReloadVersion((value) => value + 1)}>
            重新搜索
          </button>
        </div>
      ) : !hasResults ? (
        <div className="empty-state search-state" role="status" aria-live="polite">
          <div className="empty-state__icon"><SearchIcon size={24} aria-hidden="true" /></div>
          <h3>没有找到“{query}”</h3>
          <p>试试更短的关键词，或者切换搜索范围。</p>
        </div>
      ) : (
        <>
          <div className="search-result-summary" role="status" aria-live="polite">
            <span>SEARCH RESULT</span>
            <strong>“{query}”</strong>
            <small>结果按相关性排列</small>
          </div>

          <div className={`search-results${scope !== SEARCH_SCOPE.ALL ? ' search-results--single' : ''}`}>
            {showUsers && (
              <SearchUsersSection users={users} />
            )}
            {showPosts && (
              <SearchPostsSection posts={posts} />
            )}
          </div>

          <span className="sr-only" role="status" aria-live="polite">
            已加载 {users.length} 位用户、{posts.length} 篇帖子
          </span>

          {error && <p className="form-error search-pagination-error" role="alert">{error}</p>}
          {hasMore && (
            <div className="load-more search-load-more">
              <button className="button button--soft" type="button" onClick={() => void loadMore()} disabled={loadingMore}>
                {loadingMore ? '继续寻找…' : '查看更多结果'}
                {!loadingMore && <ArrowRight size={16} aria-hidden="true" />}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}

function SearchPrompt() {
  return (
    <div className="empty-state search-state search-state--prompt">
      <div className="empty-state__icon"><SearchIcon size={24} aria-hidden="true" /></div>
      <h3>从一个名字或关键词开始</h3>
      <p>你可以找到社区成员，也可以从公开帖子里找到正在讨论的话题。</p>
    </div>
  )
}

function SearchLoading() {
  return (
    <div className="search-loading" role="status" aria-label="正在搜索">
      <div className="skeleton-card" />
      <div className="skeleton-card" />
      <div className="skeleton-card" />
    </div>
  )
}

function SearchUsersSection({ users }: { users: SearchUser[] }) {
  return (
    <section className="search-section" aria-labelledby="search-users-heading">
      <div className="search-section__header">
        <div>
          <span className="search-section__icon"><Users size={17} aria-hidden="true" /></span>
          <h2 id="search-users-heading">用户</h2>
        </div>
        <small>已加载 {users.length}</small>
      </div>

      {users.length > 0 ? (
        <div className="search-user-list">
          {users.map((user) => (
            <Link className="search-user-card" to={`/u/${user.id}`} key={user.id}>
              <Avatar user={user} size="md" />
              <span className="search-user-card__copy">
                <strong>{getDisplayName(user)}</strong>
                <small>@{user.username}</small>
                <p>{user.introduction || '这位朋友还没有写下个人介绍。'}</p>
              </span>
              <ArrowRight size={17} aria-hidden="true" />
            </Link>
          ))}
        </div>
      ) : (
        <p className="search-section__empty">没有匹配的用户。</p>
      )}
    </section>
  )
}

function SearchPostsSection({ posts }: { posts: SearchPost[] }) {
  return (
    <section className="search-section search-section--posts" aria-labelledby="search-posts-heading">
      <div className="search-section__header">
        <div>
          <span className="search-section__icon search-section__icon--post"><FileText size={17} aria-hidden="true" /></span>
          <h2 id="search-posts-heading">帖子</h2>
        </div>
        <small>已加载 {posts.length}</small>
      </div>

      {posts.length > 0 ? (
        <div className="search-post-list">
          {posts.map((post) => (
            <Link className="search-post-card" to={`/post/${post.id}`} key={post.id}>
              <span className="search-post-card__meta">
                <span>{post.user ? getDisplayName(post.user) : '一位朋友'}</span>
                <span aria-hidden="true">·</span>
                <time dateTime={post.create_time}>{formatRelativeTime(post.create_time)}</time>
              </span>
              <strong>{post.title}</strong>
              {post.excerpt && <p>{post.excerpt}</p>}
              <span className="search-post-card__stats">
                <span aria-label={`${formatCount(post.like_count)} 次点赞`}>
                  <Heart size={14} aria-hidden="true" />{formatCount(post.like_count)}
                </span>
                <span aria-label={`${formatCount(post.comment_count)} 条评论`}>
                  <MessageCircle size={14} aria-hidden="true" />{formatCount(post.comment_count)}
                </span>
              </span>
            </Link>
          ))}
        </div>
      ) : (
        <p className="search-section__empty">没有匹配的公开帖子。</p>
      )}
    </section>
  )
}

function normalizeSearchTerm(value: string): string {
  return value.trim().replace(/\s+/gu, ' ')
}

function validateSearchTerm(value: string): string {
  const length = Array.from(value).length
  if (length < 2) return '搜索关键词至少需要两个字符'
  if (length > 50) return '搜索关键词不能超过 50 个字符'
  if (new TextEncoder().encode(value).length > 200) {
    return '搜索关键词不能超过 200 字节'
  }
  return ''
}

function parseSearchScope(value: string | null): SearchScope {
  switch (value) {
    case SEARCH_SCOPE.USERS:
    case SEARCH_SCOPE.POSTS:
      return value
    default:
      return SEARCH_SCOPE.ALL
  }
}

function mergeByID<T extends { id: number }>(current: T[], incoming: T[]): T[] {
  const knownIDs = new Set(current.map((item) => item.id))
  return [...current, ...incoming.filter((item) => !knownIDs.has(item.id))]
}
