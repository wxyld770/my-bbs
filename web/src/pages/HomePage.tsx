import { ArrowRight, Feather, MessageCircle, RefreshCw, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { useUI } from '../context/UIContext'
import { api } from '../lib/api'
import { getErrorMessage } from '../lib/errors'
import { getDisplayName } from '../lib/format'
import type { Post } from '../types'
import { PostCard } from '../components/PostCard'

const PAGE_SIZE = 10

export function HomePage() {
  const { user, isAuthenticated } = useAuth()
  const { openAuth, openComposer, contentVersion } = useUI()
  const [posts, setPosts] = useState<Post[]>([])
  const [pageNo, setPageNo] = useState(1)
  const [hasMore, setHasMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')

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

  useEffect(() => {
    void loadFirstPage()
  }, [contentVersion, loadFirstPage])

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
    <div className="page-wrap">
      <section className="hero" aria-labelledby="home-title">
        <div className="hero__content">
          <span className="eyebrow"><Sparkles size={14} aria-hidden="true" />A PLACE FOR REAL WORDS</span>
          <h1 id="home-title">把没说完的话，留在这里。</h1>
          <p>
            没有热榜，也不催你追赶。认真写下此刻所想，也耐心读完另一个人的故事。
          </p>
          <div className="hero__actions">
            <button className="button button--dark" type="button" onClick={startWriting}>
              <Feather size={17} aria-hidden="true" />
              {isAuthenticated ? '写一篇新帖子' : '加入并开始表达'}
            </button>
            {!isAuthenticated && (
              <button className="button button--soft" type="button" onClick={() => openAuth('login')}>
                已经来过？登录 <ArrowRight size={16} aria-hidden="true" />
              </button>
            )}
          </div>
        </div>
      </section>

      <div className="home-layout">
        <section aria-labelledby="feed-heading">
          <div className="section-heading">
            <div>
              <span className="eyebrow">PUBLIC SQUARE</span>
              <h2 id="feed-heading">广场上的新鲜话</h2>
              <p>按发布时间排列，只展示公开内容。</p>
            </div>
            {!loading && <span className="section-count">LOADED / {String(posts.length).padStart(2, '0')}</span>}
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

        <aside className="side-rail" aria-label="社区信息">
          {isAuthenticated && user ? (
            <section className="rail-card rail-card--accent">
              <span className="rail-card__index">HELLO / 01</span>
              <h3>{getDisplayName(user)}，今天好。</h3>
              <p>{user.introduction || '还没有写个人介绍。或许今天可以从一句话开始。'}</p>
            </section>
          ) : (
            <section className="rail-card rail-card--accent">
              <span className="rail-card__index">WELCOME / 01</span>
              <h3>这里是野集。</h3>
              <p>一处轻量、克制的文字广场。注册后就能发帖、评论和为喜欢的表达点亮红心。</p>
            </section>
          )}
          <section className="rail-card">
            <span className="rail-card__index">COMMUNITY / 02</span>
            <h3>我们的小约定</h3>
            <ul className="rail-list">
              <li>把屏幕另一边当作真实的人。</li>
              <li>不同意，也可以认真听完。</li>
              <li>发布前，再读一遍自己的话。</li>
            </ul>
          </section>
        </aside>
      </div>
    </div>
  )
}
