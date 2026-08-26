import { ArrowRight, Feather, MessageCircle, RefreshCw, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { formatCount, getDisplayName } from '../lib/format'
import type { HotPostItem, PostListItem } from '../types'
import { PostCard } from '../components/PostCard'

const PAGE_SIZE = 10

export function HomePage() {
  const { isAuthenticated } = useAuth()
  const { openAuth, openComposer, contentVersion } = useUI()
  const [posts, setPosts] = useState<PostListItem[]>([])
  const [pageNo, setPageNo] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')
  const [hotPosts, setHotPosts] = useState<HotPostItem[]>([])
  const [hotLoading, setHotLoading] = useState(true)
  const [hotError, setHotError] = useState('')

  const loadFirstPage = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const page = await api.listPosts({ pageNo: 1, pageSize: PAGE_SIZE })
      setPosts(page.list)
      setPageNo(1)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      setError(getErrorMessage(loadError))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadHotPosts = useCallback(async () => {
    setHotLoading(true)
    setHotError('')
    try {
      const ranking = await api.listHotPosts()
      setHotPosts(ranking.list)
    } catch (loadError) {
      setHotError(getErrorMessage(loadError))
    } finally {
      setHotLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadFirstPage()
    void loadHotPosts()
  }, [contentVersion, loadFirstPage, loadHotPosts])

  const loadMore = async () => {
    setLoadingMore(true)
    try {
      const nextPage = pageNo + 1
      const page = await api.listPosts({ pageNo: nextPage, pageSize: PAGE_SIZE })
      setPosts((current) => {
        const knownIds = new Set(current.map((post) => post.id))
        return [...current, ...page.list.filter((post) => !knownIds.has(post.id))]
      })
      setPageNo(nextPage)
      setHasMore(page.hasMore && page.list.length > 0)
    } catch (loadError) {
      setError(getErrorMessage(loadError))
    } finally {
      setLoadingMore(false)
    }
  }

  const startWriting = () => {
    if (isAuthenticated) openComposer()
    else openAuth('register')
  }

  return (
    <div className="page-wrap page-wrap--home">
      <section className="home-intro" aria-labelledby="home-title">
        <div className="home-intro__copy">
          <span className="eyebrow"><Sparkles size={14} aria-hidden="true" />YEJI / REAL WORDS</span>
          <h1 id="home-title">把没说完的话，留在这里。</h1>
          <p className="home-intro__description">
            认真写下此刻所想，也耐心读完另一个人的故事。这里记录最新的表达，也看见今天正在发生的讨论。
          </p>
          <ul className="home-intro__values" aria-label="社区更在意的事">
            <li>真实，比漂亮更重要</li>
            <li>回应观点，也照顾说话的人</li>
            <li>热度只是入口，不是答案</li>
          </ul>
        </div>
        <div className="home-intro__action">
          <button className="button button--soft" type="button" onClick={startWriting}>
            <Feather size={17} aria-hidden="true" />
            {isAuthenticated ? '写一篇' : '开始表达'}
          </button>
        </div>
      </section>

      <div className="home-layout">
        <section aria-labelledby="feed-heading" aria-busy={loading || loadingMore}>
          <div className="section-heading home-feed__heading">
            <div>
              <span className="eyebrow">PUBLIC SQUARE</span>
              <h2 id="feed-heading">广场上的新鲜话</h2>
              <p>按发布时间向下展开，每一篇公开表达都会来到这里。</p>
            </div>
          </div>

          {loading ? (
            <div className="post-list" aria-label="正在加载帖子">
              <div className="skeleton-card" />
              <div className="skeleton-card" />
              <div className="skeleton-card" />
            </div>
          ) : error && posts.length === 0 ? (
            <div className="error-state">
              <div className="error-state__icon"><RefreshCw size={23} aria-hidden="true" /></div>
              <h2>广场暂时安静了</h2>
              <p>{error}</p>
              <button className="button button--dark" type="button" onClick={() => void loadFirstPage()}>重新加载</button>
            </div>
          ) : posts.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state__icon"><MessageCircle size={24} aria-hidden="true" /></div>
              <h3>还没有人开口</h3>
              <p>成为第一个在广场留下文字的人吧。</p>
              <button className="button button--primary" type="button" onClick={startWriting}>写下第一篇</button>
            </div>
          ) : (
            <>
              <div className="post-list">
                {posts.map((post) => <PostCard key={post.id} post={post} />)}
              </div>
              {error && <p className="form-error" role="alert">{error}</p>}
              {hasMore && (
                <div className="load-more">
                  <button className="button button--soft" type="button" onClick={() => void loadMore()} disabled={loadingMore}>
                    {loadingMore ? '正在翻页…' : '再读一些'} <ArrowRight size={16} aria-hidden="true" />
                  </button>
                </div>
              )}
            </>
          )}
        </section>

        <aside className="side-rail" aria-label="发现与社区信息">
          <section className="hot-list-card" aria-labelledby="hot-list-heading" aria-busy={hotLoading}>
            <div className="hot-list-card__head">
              <div className="hot-list-card__kicker">
                <span>TODAY / TOP 10</span>
                <span className="hot-list-card__badge">昨日 + 今日</span>
              </div>
              <h3 id="hot-list-heading">今日最热</h3>
              <p>按评论 60% 与点赞 40% 计算热度，看看这两天正在发生的讨论。</p>
            </div>

            {hotLoading ? (
              <>
                <div className="hot-list-card__loading" aria-hidden="true">
                  {Array.from({ length: 5 }, (_, index) => (
                    <span key={index} />
                  ))}
                </div>
                <span className="sr-only" role="status">正在加载今日最热榜单</span>
              </>
            ) : hotError ? (
              <div className="hot-list-card__empty" role="alert">
                <span>榜单暂时没有加载成功：{hotError}</span>
                <button className="button button--soft button--small" type="button" onClick={() => void loadHotPosts()}>
                  重新加载榜单
                </button>
              </div>
            ) : hotPosts.length > 0 ? (
              <ol className="hot-list" role="list" aria-label="今日最热榜单">
                {hotPosts.map((post, index) => (
                  <li key={post.id}>
                    <Link className="hot-list__link" to={`/post/${post.id}`}>
                      <span className="hot-list__rank">{String(index + 1).padStart(2, '0')}</span>
                      <span className="hot-list__body">
                        <strong>{post.title}</strong>
                        <small>{post.user ? getDisplayName(post.user) : '一位朋友'}</small>
                        <span className="hot-list__metrics">
                          <span>{formatCount(post.comment_count)} 条评论</span>
                          <span>{formatCount(post.like_count)} 次点赞</span>
                        </span>
                      </span>
                    </Link>
                  </li>
                ))}
              </ol>
            ) : (
              <div className="hot-list-card__empty">
                <span>空榜单，快去发布吧。</span>
                <button className="button button--primary button--small" type="button" onClick={startWriting}>
                  写一篇
                </button>
              </div>
            )}
          </section>
        </aside>
      </div>
    </div>
  )
}
